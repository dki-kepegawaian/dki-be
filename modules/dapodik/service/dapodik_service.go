package service

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"regexp"
	"strings"
	"time"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"github.com/Caknoooo/go-gin-clean-starter/modules/dapodik/dto"
	"github.com/Caknoooo/go-gin-clean-starter/modules/dapodik/repository"
	dokumenService "github.com/Caknoooo/go-gin-clean-starter/modules/dokumen/service"
	notifikasiService "github.com/Caknoooo/go-gin-clean-starter/modules/notifikasi/service"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/aktivitas"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/helpers"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/utils"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var labelJenis = map[string]string{
	entities.DapodikJenisRevisiAkunOperator: "Pergantian/revisi akun operator",
	entities.DapodikJenisAutentikasiPTK:     "Autentikasi PTK/Admin Dapodik",
	entities.DapodikJenisKodeReferral:       "Kode referral operator yayasan",
	entities.DapodikJenisPerbaikanDataPTK:   "Perbaikan data PTK",
	entities.DapodikJenisLainnya:            "Lainnya",
}

// isianWajib: field formulir yang wajib per jenis (DP2 untuk revisi akun, DP5 untuk perbaikan PTK).
var isianWajib = map[string][]string{
	entities.DapodikJenisRevisiAkunOperator: {"nama_operator_lama", "nama_operator_baru", "nik_operator_baru", "email_operator_baru"},
	entities.DapodikJenisPerbaikanDataPTK:   {"data_diperbaiki"},
}

// lampiranMin: revisi akun operator butuh SK penunjukan operator dan surat permohonan (DP2).
var lampiranMin = map[string]int{entities.DapodikJenisRevisiAkunOperator: 2}

var (
	polaKunci = regexp.MustCompile(`^[a-z][a-z0-9_]{0,49}$`)
	polaNIK   = regexp.MustCompile(`^[0-9]{16}$`)
	polaEmail = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
)

type DapodikService interface {
	Ajukan(ctx context.Context, actor entities.User, req dto.AjukanRequest) (dto.TiketResponse, error)
	Lengkapi(ctx context.Context, actor entities.User, id string, req dto.LengkapiRequest) (dto.TiketResponse, error)
	Tinjau(ctx context.Context, actor entities.User, id string, req dto.TinjauRequest) (dto.TiketResponse, error)
	Detail(ctx context.Context, actor entities.User, id string) (dto.TiketResponse, error)
	Daftar(ctx context.Context, actor entities.User, f dto.Filter) ([]dto.TiketResponse, int64, error)
	Ringkasan(ctx context.Context, actor entities.User) (map[string]int64, error)
}

type dapodikService struct {
	repo       repository.DapodikRepository
	dokumen    dokumenService.DokumenService
	notifikasi notifikasiService.NotifikasiService
	db         *gorm.DB
}

func NewDapodikService(
	repo repository.DapodikRepository,
	dokumen dokumenService.DokumenService,
	notifikasi notifikasiService.NotifikasiService,
	db *gorm.DB,
) DapodikService {
	return &dapodikService{repo: repo, dokumen: dokumen, notifikasi: notifikasi, db: db}
}

// transaksi menjalankan fn dalam satu transaksi. Berkas yang sudah tersimpan
// lewat simpan() dihapus dari storage kalau transaksi gagal.
func (s *dapodikService) transaksi(ctx context.Context, fn func(tx *gorm.DB, simpan func(*entities.Dokumen)) error) error {
	var tersimpan []entities.Dokumen
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(tx, func(d *entities.Dokumen) { tersimpan = append(tersimpan, *d) })
	})
	if err != nil {
		s.dokumen.Hapus(tersimpan...)
	}
	return err
}

func (s *dapodikService) catatTransisi(ctx context.Context, tx *gorm.DB, actor entities.User, t *entities.DapodikTiket, ke, aksi, catatan string) error {
	dari := t.Status
	t.Status = ke
	if err := s.repo.Update(ctx, tx, t); err != nil {
		return err
	}
	if err := s.repo.AddRiwayat(ctx, tx, &entities.DapodikRiwayat{
		TiketID: t.ID, DariStatus: dari, KeStatus: ke, Oleh: actor.ID, Catatan: catatan,
	}); err != nil {
		return err
	}
	return aktivitas.Catat(tx, actor.ID, aksi, "dapodik_tiket", t.ID, fmt.Sprintf("%s: %s -> %s", t.Nomor, dari, ke))
}

// simpanLampiran mengunggah berkas lalu menautkannya ke tiket, di transaksi pemanggil.
func (s *dapodikService) simpanLampiran(ctx context.Context, tx *gorm.DB, simpan func(*entities.Dokumen), actor entities.User, t *entities.DapodikTiket, files []*multipart.FileHeader, tahap string) error {
	for _, fh := range files {
		doc, err := s.dokumen.Simpan(ctx, tx, fh, &t.SekolahID, actor.ID)
		if err != nil {
			return err
		}
		simpan(&doc)
		if err := s.repo.AddLampiran(ctx, tx, &entities.DapodikLampiran{TiketID: t.ID, DokumenID: doc.ID, Tahap: tahap}); err != nil {
			return err
		}
	}
	return nil
}

func bersihkanIsian(jenis string, in map[string]string) (map[string]string, error) {
	out := map[string]string{}
	for k, v := range in {
		if !polaKunci.MatchString(k) {
			return nil, utils.ErrBadRequest("nama field isian tidak valid: " + k)
		}
		if v = strings.TrimSpace(v); v != "" {
			out[k] = v
		}
	}
	for _, k := range isianWajib[jenis] {
		if out[k] == "" {
			return nil, utils.ErrBadRequest("isian " + k + " wajib diisi")
		}
	}
	if nik, ok := out["nik_operator_baru"]; ok && !polaNIK.MatchString(nik) {
		return nil, utils.ErrBadRequest("NIK operator baru harus 16 digit")
	}
	if email, ok := out["email_operator_baru"]; ok && !polaEmail.MatchString(email) {
		return nil, utils.ErrBadRequest("email operator baru tidak valid")
	}
	return out, nil
}

// tambahHariKerja mengembalikan tanggal setelah n hari kerja (Senin–Jumat) dari t.
func tambahHariKerja(t time.Time, n int) time.Time {
	d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	for n > 0 {
		d = d.AddDate(0, 0, 1)
		if wd := d.Weekday(); wd != time.Saturday && wd != time.Sunday {
			n--
		}
	}
	return d
}

func (s *dapodikService) Ajukan(ctx context.Context, actor entities.User, req dto.AjukanRequest) (dto.TiketResponse, error) {
	if actor.SekolahID == nil {
		return dto.TiketResponse{}, utils.ErrBadRequest("akun operator belum ditempatkan di sekolah")
	}
	isian, err := bersihkanIsian(req.Jenis, req.Isian)
	if err != nil {
		return dto.TiketResponse{}, err
	}
	if req.Jenis == entities.DapodikJenisPerbaikanDataPTK && req.PendidikID == "" {
		return dto.TiketResponse{}, utils.ErrBadRequest("PTK yang datanya diperbaiki wajib dipilih")
	}
	if n := lampiranMin[req.Jenis]; len(req.Lampiran) < n {
		return dto.TiketResponse{}, utils.ErrBadRequest(fmt.Sprintf("layanan ini butuh minimal %d lampiran wajib", n))
	}
	uraian := strings.TrimSpace(req.Uraian)
	if uraian == "" {
		return dto.TiketResponse{}, utils.ErrBadRequest("uraian wajib diisi")
	}

	var id uuid.UUID
	err = s.transaksi(ctx, func(tx *gorm.DB, simpan func(*entities.Dokumen)) error {
		t := entities.DapodikTiket{
			SekolahID: *actor.SekolahID, DiajukanOleh: actor.ID, Jenis: req.Jenis,
			Isian: isian, Uraian: uraian, Status: entities.DapodikStatusBaru,
			Tenggat: tambahHariKerja(time.Now(), entities.DapodikTenggatHariKerja),
		}
		if req.PendidikID != "" {
			p, err := s.repo.GetPendidik(ctx, tx, actor, req.PendidikID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return utils.ErrBadRequest("PTK tidak ditemukan di sekolah Anda")
				}
				return err
			}
			t.PendidikID = &p.ID
		}
		nomor, err := helpers.NomorRegistrasi(tx, "DPK", time.Now().Year())
		if err != nil {
			return err
		}
		t.Nomor = nomor
		if err := s.repo.Create(ctx, tx, &t); err != nil {
			return err
		}
		if err := s.simpanLampiran(ctx, tx, simpan, actor, &t, req.Lampiran, entities.DapodikLampiranPengajuan); err != nil {
			return err
		}
		if err := s.repo.AddRiwayat(ctx, tx, &entities.DapodikRiwayat{TiketID: t.ID, KeStatus: t.Status, Oleh: actor.ID}); err != nil {
			return err
		}
		if err := aktivitas.Catat(tx, actor.ID, "dapodik.ajukan", "dapodik_tiket", t.ID, nomor); err != nil {
			return err
		}
		staf, err := s.repo.StafSudinSekolah(ctx, tx, t.SekolahID)
		if err != nil {
			return err
		}
		id = t.ID
		return s.notifikasi.Kirim(ctx, tx, staf, "Tiket Dapodik baru",
			fmt.Sprintf("%s · %s", nomor, labelJenis[req.Jenis]), tautan(t.ID))
	})
	if err != nil {
		return dto.TiketResponse{}, err
	}
	return s.Detail(ctx, actor, id.String())
}

func (s *dapodikService) Lengkapi(ctx context.Context, actor entities.User, id string, req dto.LengkapiRequest) (dto.TiketResponse, error) {
	balasan := strings.TrimSpace(req.Balasan)
	if balasan == "" {
		return dto.TiketResponse{}, utils.ErrBadRequest("balasan untuk admin wajib diisi")
	}
	err := s.transaksi(ctx, func(tx *gorm.DB, simpan func(*entities.Dokumen)) error {
		t, err := s.repo.GetForUpdate(ctx, tx, actor, id)
		if err != nil {
			return err
		}
		if t.Status != entities.DapodikStatusPerluInfoTambahan {
			return utils.ErrConflict("hanya tiket berstatus perlu info tambahan yang bisa dilengkapi")
		}
		if err := s.simpanLampiran(ctx, tx, simpan, actor, &t, req.Lampiran, entities.DapodikLampiranInfoTambahan); err != nil {
			return err
		}
		t.BalasanInfo = balasan
		t.Tenggat = tambahHariKerja(time.Now(), entities.DapodikTenggatHariKerja)
		if err := s.catatTransisi(ctx, tx, actor, &t, entities.DapodikStatusBaru, "dapodik.lengkapi", balasan); err != nil {
			return err
		}
		staf, err := s.repo.StafSudinSekolah(ctx, tx, t.SekolahID)
		if err != nil {
			return err
		}
		return s.notifikasi.Kirim(ctx, tx, staf, "Tiket Dapodik dilengkapi",
			fmt.Sprintf("%s · %s mengirim info tambahan", t.Nomor, t.Sekolah.Nama), tautan(t.ID))
	})
	if err != nil {
		return dto.TiketResponse{}, err
	}
	return s.Detail(ctx, actor, id)
}

func (s *dapodikService) Tinjau(ctx context.Context, actor entities.User, id string, req dto.TinjauRequest) (dto.TiketResponse, error) {
	catatan := strings.TrimSpace(req.Catatan)
	if catatan == "" {
		return dto.TiketResponse{}, utils.ErrBadRequest("catatan wajib diisi")
	}
	err := s.transaksi(ctx, func(tx *gorm.DB, _ func(*entities.Dokumen)) error {
		t, err := s.repo.GetForUpdate(ctx, tx, actor, id)
		if err != nil {
			return err
		}
		if t.Status != entities.DapodikStatusBaru {
			return utils.ErrConflict("hanya tiket berstatus baru yang bisa ditinjau")
		}

		judul := "Tiket Dapodik butuh info tambahan"
		if req.Keputusan == entities.DapodikStatusSelesai {
			now := time.Now()
			t.CatatanPenyelesaian = catatan
			t.DiselesaikanOleh, t.DiselesaikanAt = &actor.ID, &now
			judul = "Tiket Dapodik selesai"
		} else {
			t.PermintaanInfo = catatan
		}
		if err := s.catatTransisi(ctx, tx, actor, &t, req.Keputusan, "dapodik.tinjau", catatan); err != nil {
			return err
		}
		operator, err := s.repo.OperatorSekolah(ctx, tx, t.SekolahID)
		if err != nil {
			return err
		}
		return s.notifikasi.Kirim(ctx, tx, operator, judul, fmt.Sprintf("%s · %s", t.Nomor, catatan), tautan(t.ID))
	})
	if err != nil {
		return dto.TiketResponse{}, err
	}
	return s.Detail(ctx, actor, id)
}

func (s *dapodikService) Detail(ctx context.Context, actor entities.User, id string) (dto.TiketResponse, error) {
	t, err := s.repo.GetDetail(ctx, actor, id)
	if err != nil {
		return dto.TiketResponse{}, err
	}
	return dto.FromEntity(t), nil
}

func (s *dapodikService) Daftar(ctx context.Context, actor entities.User, f dto.Filter) ([]dto.TiketResponse, int64, error) {
	items, total, err := s.repo.List(ctx, actor, f)
	if err != nil {
		return nil, 0, err
	}
	out := make([]dto.TiketResponse, 0, len(items))
	for _, t := range items {
		out = append(out, dto.FromEntity(t))
	}
	return out, total, nil
}

func (s *dapodikService) Ringkasan(ctx context.Context, actor entities.User) (map[string]int64, error) {
	return s.repo.Ringkasan(ctx, actor)
}

func tautan(id uuid.UUID) string { return "/dapodik/" + id.String() }
