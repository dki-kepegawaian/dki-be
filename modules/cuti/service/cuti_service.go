package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"github.com/Caknoooo/go-gin-clean-starter/modules/cuti/dto"
	"github.com/Caknoooo/go-gin-clean-starter/modules/cuti/repository"
	dokumenService "github.com/Caknoooo/go-gin-clean-starter/modules/dokumen/service"
	notifikasiService "github.com/Caknoooo/go-gin-clean-starter/modules/notifikasi/service"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/aktivitas"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/helpers"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/utils"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const tglFormat = "2006-01-02"

var labelJenis = map[string]string{
	entities.CutiJenisTahunan:                "Cuti tahunan",
	entities.CutiJenisBesar:                  "Cuti besar",
	entities.CutiJenisSakit:                  "Cuti sakit",
	entities.CutiJenisMelahirkan:             "Cuti melahirkan",
	entities.CutiJenisAlasanPenting:          "Cuti alasan penting",
	entities.CutiJenisDiLuarTanggunganNegara: "Cuti di luar tanggungan negara",
}

type CutiService interface {
	Ajukan(ctx context.Context, actor entities.User, req dto.AjukanRequest) (dto.CutiResponse, error)
	Perbaiki(ctx context.Context, actor entities.User, id string, req dto.PerbaikiRequest) (dto.CutiResponse, error)
	Periksa(ctx context.Context, actor entities.User, id string, req dto.PeriksaRequest) (dto.CutiResponse, error)
	TerimaSuratFisik(ctx context.Context, actor entities.User, req dto.TerimaSuratFisikRequest) (dto.TerimaSuratFisikResponse, error)
	Putuskan(ctx context.Context, actor entities.User, id string, req dto.KeputusanRequest) (dto.CutiResponse, error)
	Detail(ctx context.Context, actor entities.User, id string) (dto.CutiResponse, error)
	Daftar(ctx context.Context, actor entities.User, f dto.Filter) ([]dto.CutiResponse, int64, error)
	Ringkasan(ctx context.Context, actor entities.User) (map[string]int64, error)
	SisaTahunan(ctx context.Context, actor entities.User, pendidikID string, tahun int) (dto.SisaTahunanResponse, error)
}

type cutiService struct {
	repo       repository.CutiRepository
	dokumen    dokumenService.DokumenService
	notifikasi notifikasiService.NotifikasiService
	db         *gorm.DB
}

func NewCutiService(
	repo repository.CutiRepository,
	dokumen dokumenService.DokumenService,
	notifikasi notifikasiService.NotifikasiService,
	db *gorm.DB,
) CutiService {
	return &cutiService{repo: repo, dokumen: dokumen, notifikasi: notifikasi, db: db}
}

// transaksi menjalankan fn dalam satu transaksi. Berkas yang sudah tersimpan
// lewat simpan() dihapus dari disk kalau transaksi gagal.
func (s *cutiService) transaksi(ctx context.Context, fn func(tx *gorm.DB, simpan func(*entities.Dokumen)) error) error {
	var tersimpan []entities.Dokumen
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(tx, func(d *entities.Dokumen) { tersimpan = append(tersimpan, *d) })
	})
	if err != nil {
		s.dokumen.Hapus(tersimpan...)
	}
	return err
}

func (s *cutiService) catatTransisi(ctx context.Context, tx *gorm.DB, actor entities.User, c *entities.Cuti, ke, aksi, catatan string) error {
	dari := c.Status
	c.Status = ke
	if err := s.repo.Update(ctx, tx, c); err != nil {
		return err
	}
	if err := s.repo.AddRiwayat(ctx, tx, &entities.CutiRiwayat{
		CutiID: c.ID, DariStatus: dari, KeStatus: ke, Oleh: actor.ID, Catatan: catatan,
	}); err != nil {
		return err
	}
	return aktivitas.Catat(tx, actor.ID, aksi, "cuti", c.ID, fmt.Sprintf("%s: %s -> %s", c.Nomor, dari, ke))
}

func (s *cutiService) Ajukan(ctx context.Context, actor entities.User, req dto.AjukanRequest) (dto.CutiResponse, error) {
	if !req.DisetujuiKepsek {
		return dto.CutiResponse{}, utils.ErrBadRequest("permohonan harus sudah disetujui dan ditandatangani Kepala Sekolah")
	}
	mulai, _ := time.Parse(tglFormat, req.TanggalMulai) // format sudah divalidasi binding
	selesai, _ := time.Parse(tglFormat, req.TanggalSelesai)
	if selesai.Before(mulai) {
		return dto.CutiResponse{}, utils.ErrBadRequest("tanggal selesai tidak boleh sebelum tanggal mulai")
	}
	if selesai.After(mulai.AddDate(3, 0, 0)) {
		return dto.CutiResponse{}, utils.ErrBadRequest("rentang cuti maksimal 3 tahun")
	}
	hari := helpers.HariKerja(mulai, selesai)
	if hari == 0 {
		return dto.CutiResponse{}, utils.ErrBadRequest("rentang tanggal tidak mencakup hari kerja")
	}

	var id uuid.UUID
	err := s.transaksi(ctx, func(tx *gorm.DB, simpan func(*entities.Dokumen)) error {
		p, err := s.repo.GetPendidik(ctx, tx, actor, req.PendidikID, true)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return utils.ErrBadRequest("pendidik tidak ditemukan di sekolah Anda")
			}
			return err
		}

		if req.Jenis == entities.CutiJenisTahunan {
			if mulai.Year() != selesai.Year() {
				return utils.ErrBadRequest("cuti tahunan tidak boleh melewati pergantian tahun; ajukan terpisah per tahun")
			}
			terpakai, err := s.repo.HariTahunanTerpakai(ctx, tx, p.ID, mulai.Year())
			if err != nil {
				return err
			}
			if sisa := entities.CutiJatahTahunan - terpakai; hari > sisa {
				return utils.ErrBadRequest(fmt.Sprintf("sisa cuti tahunan %d hanya %d hari kerja", mulai.Year(), max(sisa, 0)))
			}
		}
		tumpang, err := s.repo.AdaTumpangTindih(ctx, tx, p.ID, mulai, selesai)
		if err != nil {
			return err
		}
		if tumpang {
			return utils.ErrConflict("pendidik sudah punya pengajuan cuti pada rentang tanggal tersebut")
		}

		doc, err := s.dokumen.Simpan(ctx, tx, req.Berkas, &p.SekolahID, actor.ID)
		if err != nil {
			return err
		}
		simpan(&doc)

		nomor, err := helpers.NomorRegistrasi(tx, "CUT", time.Now().Year())
		if err != nil {
			return err
		}
		c := entities.Cuti{
			Nomor: nomor, PendidikID: p.ID, SekolahID: p.SekolahID, DiajukanOleh: actor.ID,
			Jenis: req.Jenis, TanggalMulai: mulai, TanggalSelesai: selesai, HariKerja: hari,
			Alasan: strings.TrimSpace(req.Alasan), DisetujuiKepsek: true, BerkasPermohonanID: doc.ID,
			Status: entities.CutiStatusDiajukan,
		}
		if err := s.repo.Create(ctx, tx, &c); err != nil {
			return err
		}
		if err := s.repo.AddRiwayat(ctx, tx, &entities.CutiRiwayat{CutiID: c.ID, KeStatus: c.Status, Oleh: actor.ID}); err != nil {
			return err
		}
		if err := aktivitas.Catat(tx, actor.ID, "cuti.ajukan", "cuti", c.ID, nomor); err != nil {
			return err
		}

		staf, err := s.repo.StafSudinSekolah(ctx, tx, p.SekolahID)
		if err != nil {
			return err
		}
		id = c.ID
		return s.notifikasi.Kirim(ctx, tx, staf, "Pengajuan cuti baru",
			fmt.Sprintf("%s · %s · %s, %s", nomor, p.Nama, labelJenis[req.Jenis], p.Sekolah.Nama), tautan(c.ID))
	})
	if err != nil {
		return dto.CutiResponse{}, err
	}
	return s.Detail(ctx, actor, id.String())
}

func (s *cutiService) Perbaiki(ctx context.Context, actor entities.User, id string, req dto.PerbaikiRequest) (dto.CutiResponse, error) {
	err := s.transaksi(ctx, func(tx *gorm.DB, simpan func(*entities.Dokumen)) error {
		c, err := s.repo.GetForUpdate(ctx, tx, actor, id)
		if err != nil {
			return err
		}
		if c.Status != entities.CutiStatusPerluPerbaikan {
			return utils.ErrConflict("hanya pengajuan berstatus perlu perbaikan yang bisa diperbaiki")
		}
		doc, err := s.dokumen.Simpan(ctx, tx, req.Berkas, &c.SekolahID, actor.ID)
		if err != nil {
			return err
		}
		simpan(&doc)

		c.BerkasPermohonanID = doc.ID
		if a := strings.TrimSpace(req.Alasan); a != "" {
			c.Alasan = a
		}
		c.BerkasLengkap, c.BerkasTerbaca = nil, nil
		if err := s.catatTransisi(ctx, tx, actor, &c, entities.CutiStatusDiajukan, "cuti.perbaiki", "Berkas diunggah ulang"); err != nil {
			return err
		}
		staf, err := s.repo.StafSudinSekolah(ctx, tx, c.SekolahID)
		if err != nil {
			return err
		}
		return s.notifikasi.Kirim(ctx, tx, staf, "Berkas cuti diperbaiki",
			fmt.Sprintf("%s · %s mengunggah ulang berkas", c.Nomor, c.Sekolah.Nama), tautan(c.ID))
	})
	if err != nil {
		return dto.CutiResponse{}, err
	}
	return s.Detail(ctx, actor, id)
}

func (s *cutiService) Periksa(ctx context.Context, actor entities.User, id string, req dto.PeriksaRequest) (dto.CutiResponse, error) {
	ok := *req.BerkasLengkap && *req.BerkasTerbaca
	catatan := strings.TrimSpace(req.Catatan)
	if !ok && catatan == "" {
		return dto.CutiResponse{}, utils.ErrBadRequest("catatan untuk sekolah wajib diisi jika berkas tidak lengkap atau tidak terbaca")
	}

	err := s.transaksi(ctx, func(tx *gorm.DB, _ func(*entities.Dokumen)) error {
		c, err := s.repo.GetForUpdate(ctx, tx, actor, id)
		if err != nil {
			return err
		}
		if c.Status != entities.CutiStatusDiajukan {
			return utils.ErrConflict("hanya pengajuan berstatus diajukan yang bisa diperiksa")
		}
		now := time.Now()
		c.BerkasLengkap, c.BerkasTerbaca = req.BerkasLengkap, req.BerkasTerbaca
		c.DiperiksaOleh, c.DiperiksaAt = &actor.ID, &now

		ke, judul, pesan := entities.CutiStatusMenungguSuratFisik, "Berkas cuti lengkap",
			fmt.Sprintf("%s · Antar surat permohonan asli dan nota dinas ke Sudin dalam 3 hari kerja.", c.Nomor)
		if !ok {
			ke, judul, pesan = entities.CutiStatusPerluPerbaikan, "Berkas cuti perlu perbaikan",
				fmt.Sprintf("%s · %s", c.Nomor, catatan)
			c.CatatanPerbaikan = catatan
		}
		if err := s.catatTransisi(ctx, tx, actor, &c, ke, "cuti.periksa", catatan); err != nil {
			return err
		}
		penerima, err := s.repo.OperatorSekolah(ctx, tx, c.SekolahID)
		if err != nil {
			return err
		}
		return s.notifikasi.Kirim(ctx, tx, penerima, judul, pesan, tautan(c.ID))
	})
	if err != nil {
		return dto.CutiResponse{}, err
	}
	return s.Detail(ctx, actor, id)
}

func (s *cutiService) TerimaSuratFisik(ctx context.Context, actor entities.User, req dto.TerimaSuratFisikRequest) (dto.TerimaSuratFisikResponse, error) {
	if !*req.NotaDinasAda {
		return dto.TerimaSuratFisikResponse{}, utils.ErrBadRequest("pengajuan tidak dapat ditandai diterima tanpa nota dinas kecamatan")
	}
	tanggal, _ := time.Parse(tglFormat, req.TanggalDiterima)
	if tanggal.After(time.Now()) {
		return dto.TerimaSuratFisikResponse{}, utils.ErrBadRequest("tanggal diterima tidak boleh di masa depan")
	}

	penerima := actor.ID
	if req.PenerimaID != "" {
		penerima = uuid.MustParse(req.PenerimaID)
	}

	var diterima []string
	err := s.transaksi(ctx, func(tx *gorm.DB, _ func(*entities.Dokumen)) error {
		if penerima != actor.ID {
			if actor.SudinID == nil {
				return utils.ErrBadRequest("penerima tidak valid")
			}
			valid, err := s.repo.StafSudinValid(ctx, tx, penerima, *actor.SudinID)
			if err != nil {
				return err
			}
			if !valid {
				return utils.ErrBadRequest("penerima harus staf Sudin di wilayah yang sama")
			}
		}

		for _, id := range req.CutiIDs {
			c, err := s.repo.GetForUpdate(ctx, tx, actor, id)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return utils.ErrNotFound("pengajuan " + id + " tidak ditemukan")
				}
				return err
			}
			if c.Status != entities.CutiStatusMenungguSuratFisik {
				return utils.ErrConflict(c.Nomor + " tidak sedang menunggu surat fisik")
			}
			c.SuratFisikDiterimaTanggal = &tanggal
			c.SuratFisikPenerimaID = &penerima
			c.NotaDinasAda = req.NotaDinasAda
			c.CatatanPenerimaan = strings.TrimSpace(req.Catatan)
			if err := s.catatTransisi(ctx, tx, actor, &c, entities.CutiStatusDiprosesSudin, "cuti.terima_surat_fisik", c.CatatanPenerimaan); err != nil {
				return err
			}
			operator, err := s.repo.OperatorSekolah(ctx, tx, c.SekolahID)
			if err != nil {
				return err
			}
			if err := s.notifikasi.Kirim(ctx, tx, operator, "Surat fisik cuti diterima",
				fmt.Sprintf("%s · Surat asli dan nota dinas diterima Sudin, menunggu tanda tangan pejabat.", c.Nomor), tautan(c.ID)); err != nil {
				return err
			}
			diterima = append(diterima, c.Nomor)
		}
		return nil
	})
	return dto.TerimaSuratFisikResponse{Diterima: diterima}, err
}

func (s *cutiService) Putuskan(ctx context.Context, actor entities.User, id string, req dto.KeputusanRequest) (dto.CutiResponse, error) {
	tanggal, _ := time.Parse(tglFormat, req.TanggalSuratIzin)

	err := s.transaksi(ctx, func(tx *gorm.DB, simpan func(*entities.Dokumen)) error {
		c, err := s.repo.GetForUpdate(ctx, tx, actor, id)
		if err != nil {
			return err
		}
		if c.Status != entities.CutiStatusDiprosesSudin {
			return utils.ErrConflict("keputusan hanya bisa dicatat setelah surat fisik diterima")
		}
		doc, err := s.dokumen.Simpan(ctx, tx, req.SuratIzin, &c.SekolahID, actor.ID)
		if err != nil {
			return err
		}
		simpan(&doc)

		now := time.Now()
		c.NomorSuratIzin = strings.TrimSpace(req.NomorSuratIzin)
		c.TanggalSuratIzin = &tanggal
		c.SuratIzinID = &doc.ID
		c.CatatanKeputusan = strings.TrimSpace(req.Catatan)
		c.DiputuskanOleh, c.DiputuskanAt = &actor.ID, &now
		if err := s.catatTransisi(ctx, tx, actor, &c, req.Keputusan, "cuti.putuskan", c.CatatanKeputusan); err != nil {
			return err
		}

		judul := "Cuti disetujui"
		if req.Keputusan == entities.CutiStatusDitolak {
			judul = "Cuti ditolak"
		}
		operator, err := s.repo.OperatorSekolah(ctx, tx, c.SekolahID)
		if err != nil {
			return err
		}
		return s.notifikasi.Kirim(ctx, tx, operator, judul,
			fmt.Sprintf("%s · %s · Surat %s", c.Nomor, c.Pendidik.Nama, c.NomorSuratIzin), tautan(c.ID))
	})
	if err != nil {
		return dto.CutiResponse{}, err
	}
	return s.Detail(ctx, actor, id)
}

func (s *cutiService) Detail(ctx context.Context, actor entities.User, id string) (dto.CutiResponse, error) {
	c, err := s.repo.GetDetail(ctx, actor, id)
	if err != nil {
		return dto.CutiResponse{}, err
	}
	return dto.FromEntity(c), nil
}

func (s *cutiService) Daftar(ctx context.Context, actor entities.User, f dto.Filter) ([]dto.CutiResponse, int64, error) {
	items, total, err := s.repo.List(ctx, actor, f)
	if err != nil {
		return nil, 0, err
	}
	out := make([]dto.CutiResponse, 0, len(items))
	for _, c := range items {
		out = append(out, dto.FromEntity(c))
	}
	return out, total, nil
}

func (s *cutiService) Ringkasan(ctx context.Context, actor entities.User) (map[string]int64, error) {
	return s.repo.Ringkasan(ctx, actor)
}

func (s *cutiService) SisaTahunan(ctx context.Context, actor entities.User, pendidikID string, tahun int) (dto.SisaTahunanResponse, error) {
	p, err := s.repo.GetPendidik(ctx, nil, actor, pendidikID, false)
	if err != nil {
		return dto.SisaTahunanResponse{}, err
	}
	terpakai, err := s.repo.HariTahunanTerpakai(ctx, nil, p.ID, tahun)
	if err != nil {
		return dto.SisaTahunanResponse{}, err
	}
	return dto.SisaTahunanResponse{
		PendidikID: p.ID.String(), Tahun: tahun, Jatah: entities.CutiJatahTahunan,
		Terpakai: terpakai, Sisa: max(entities.CutiJatahTahunan-terpakai, 0),
	}, nil
}

func tautan(id uuid.UUID) string { return "/cuti/" + id.String() }
