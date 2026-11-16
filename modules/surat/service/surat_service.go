package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/arsipkan"
	"strings"
	"time"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	dokumenService "github.com/Caknoooo/go-gin-clean-starter/modules/dokumen/service"
	notifikasiService "github.com/Caknoooo/go-gin-clean-starter/modules/notifikasi/service"
	"github.com/Caknoooo/go-gin-clean-starter/modules/surat/dto"
	"github.com/Caknoooo/go-gin-clean-starter/modules/surat/repository"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/aktivitas"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/helpers"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// KodeKlasifikasi dipakai semua jenis surat, mengikuti contoh nomor di Figma (0188/PK.01.02/2027).
const KodeKlasifikasi = "PK.01.02"

// MaksUkuranPindai mengikuti S4: "PDF, maks. 5 MB".
const MaksUkuranPindai = 5 << 20

type SuratService interface {
	Jenis() []dto.JenisResponse
	Buat(ctx context.Context, actor entities.User, req dto.BuatRequest) (dto.SuratResponse, error)
	Ubah(ctx context.Context, actor entities.User, id string, req dto.UbahRequest) (dto.SuratResponse, error)
	// Pratinjau merender PDF dari isian saat ini tanpa menyimpannya.
	Pratinjau(ctx context.Context, actor entities.User, id string) (string, []byte, error)
	BuatDokumen(ctx context.Context, actor entities.User, id string) (dto.SuratResponse, error)
	TetapkanFinal(ctx context.Context, actor entities.User, id string, req dto.FinalRequest) (dto.SuratResponse, error)
	Batalkan(ctx context.Context, actor entities.User, id string, req dto.BatalRequest) (dto.SuratResponse, error)
	Serahkan(ctx context.Context, actor entities.User, id string) (dto.SuratResponse, error)
	Detail(ctx context.Context, actor entities.User, id string) (dto.SuratResponse, error)
	Daftar(ctx context.Context, actor entities.User, f dto.Filter) ([]dto.SuratResponse, int64, error)
}

type suratService struct {
	repo       repository.SuratRepository
	dokumen    dokumenService.DokumenService
	notifikasi notifikasiService.NotifikasiService
	db         *gorm.DB
}

func NewSuratService(
	repo repository.SuratRepository,
	dokumen dokumenService.DokumenService,
	notifikasi notifikasiService.NotifikasiService,
	db *gorm.DB,
) SuratService {
	return &suratService{repo: repo, dokumen: dokumen, notifikasi: notifikasi, db: db}
}

// transaksi menjalankan fn dalam satu transaksi. Berkas yang sudah tersimpan
// lewat simpan() dihapus dari storage kalau transaksi gagal.
func (s *suratService) transaksi(ctx context.Context, fn func(tx *gorm.DB, simpan func(*entities.Dokumen)) error) error {
	var tersimpan []entities.Dokumen
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(tx, func(d *entities.Dokumen) { tersimpan = append(tersimpan, *d) })
	})
	if err != nil {
		s.dokumen.Hapus(tersimpan...)
	}
	return err
}

func (s *suratService) catatTransisi(ctx context.Context, tx *gorm.DB, actor entities.User, sr *entities.Surat, ke, aksi, catatan string) error {
	dari := sr.Status
	sr.Status = ke
	if err := s.repo.Update(ctx, tx, sr); err != nil {
		return err
	}
	if err := s.repo.AddRiwayat(ctx, tx, &entities.SuratRiwayat{
		SuratID: sr.ID, DariStatus: dari, KeStatus: ke, Oleh: actor.ID, Catatan: catatan,
	}); err != nil {
		return err
	}
	return aktivitas.Catat(tx, actor.ID, aksi, "surat", sr.ID, fmt.Sprintf("%s: %s -> %s", sr.Nomor, dari, ke))
}

// nomorSurat memakai nomor isian bila ada (dicek unik), atau nomor urut berikutnya
// untuk tahun tanggal surat: 0188/PK.01.02/2027.
func (s *suratService) nomorSurat(ctx context.Context, tx *gorm.DB, isian string, tahun int, kecuali uuid.UUID) (string, error) {
	if nomor := strings.TrimSpace(isian); nomor != "" {
		dipakai, err := s.repo.NomorDipakai(ctx, tx, nomor, kecuali)
		if err != nil {
			return "", err
		}
		if dipakai {
			return "", utils.ErrConflict("nomor surat " + nomor + " sudah dipakai")
		}
		return nomor, nil
	}
	// Nomor isian manual bisa sudah memakai nomor urut berikutnya; lompati yang terpakai.
	for range 100 {
		reg, err := helpers.NomorRegistrasi(tx, "SRT", tahun)
		if err != nil {
			return "", err
		}
		urut := strings.TrimPrefix(reg, fmt.Sprintf("SRT-%d-", tahun))
		nomor := fmt.Sprintf("%s/%s/%d", urut, KodeKlasifikasi, tahun)
		dipakai, err := s.repo.NomorDipakai(ctx, tx, nomor, kecuali)
		if err != nil {
			return "", err
		}
		if !dipakai {
			return nomor, nil
		}
	}
	return "", utils.ErrConflict("gagal memberi nomor surat otomatis; isi nomor secara manual")
}

func nomorBentrok(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return utils.ErrConflict("nomor surat sudah dipakai")
	}
	return err
}

func (s *suratService) Jenis() []dto.JenisResponse {
	out := make([]dto.JenisResponse, 0, len(urutanJenis))
	for _, k := range urutanJenis {
		t := templates[k]
		out = append(out, dto.JenisResponse{Kode: k, Label: t.Label, Deskripsi: t.Deskripsi})
	}
	return out
}

func (s *suratService) Buat(ctx context.Context, actor entities.User, req dto.BuatRequest) (dto.SuratResponse, error) {
	tanggal, _ := time.Parse(dto.TglFormat, req.TanggalSurat) // format sudah divalidasi binding
	perihal := strings.TrimSpace(req.Perihal)
	if perihal == "" {
		return dto.SuratResponse{}, utils.ErrBadRequest("perihal wajib diisi")
	}

	var id uuid.UUID
	err := s.transaksi(ctx, func(tx *gorm.DB, _ func(*entities.Dokumen)) error {
		p, err := s.repo.GetPendidik(ctx, tx, actor, req.PendidikID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return utils.ErrNotFound("pendidik tidak ditemukan")
			}
			return err
		}
		nomor, err := s.nomorSurat(ctx, tx, req.Nomor, tanggal.Year(), uuid.Nil)
		if err != nil {
			return err
		}
		sr := entities.Surat{
			Nomor: nomor, Jenis: req.Jenis, PendidikID: p.ID, SekolahID: p.SekolahID, DibuatOleh: actor.ID,
			Perihal: perihal, TanggalSurat: tanggal, Keterangan: strings.TrimSpace(req.Keterangan),
			Status: entities.SuratStatusDraf,
		}
		if err := s.repo.Create(ctx, tx, &sr); err != nil {
			return nomorBentrok(err)
		}
		if err := s.repo.AddRiwayat(ctx, tx, &entities.SuratRiwayat{SuratID: sr.ID, KeStatus: sr.Status, Oleh: actor.ID}); err != nil {
			return err
		}
		id = sr.ID
		return aktivitas.Catat(tx, actor.ID, "surat.buat", "surat", sr.ID, nomor)
	})
	if err != nil {
		return dto.SuratResponse{}, err
	}
	return s.Detail(ctx, actor, id.String())
}

func (s *suratService) Ubah(ctx context.Context, actor entities.User, id string, req dto.UbahRequest) (dto.SuratResponse, error) {
	tanggal, _ := time.Parse(dto.TglFormat, req.TanggalSurat)
	perihal := strings.TrimSpace(req.Perihal)
	if perihal == "" {
		return dto.SuratResponse{}, utils.ErrBadRequest("perihal wajib diisi")
	}

	err := s.transaksi(ctx, func(tx *gorm.DB, _ func(*entities.Dokumen)) error {
		sr, err := s.repo.GetForUpdate(ctx, tx, actor, id)
		if err != nil {
			return err
		}
		if sr.Status != entities.SuratStatusDraf {
			return utils.ErrConflict("hanya surat berstatus draf yang bisa diubah")
		}
		if n := strings.TrimSpace(req.Nomor); n != "" && n != sr.Nomor {
			if sr.Nomor, err = s.nomorSurat(ctx, tx, n, tanggal.Year(), sr.ID); err != nil {
				return err
			}
		}
		sr.Perihal, sr.TanggalSurat, sr.Keterangan = perihal, tanggal, strings.TrimSpace(req.Keterangan)
		if err := s.repo.Update(ctx, tx, &sr); err != nil {
			return nomorBentrok(err)
		}
		return aktivitas.Catat(tx, actor.ID, "surat.ubah", "surat", sr.ID, sr.Nomor)
	})
	if err != nil {
		return dto.SuratResponse{}, err
	}
	return s.Detail(ctx, actor, id)
}

func (s *suratService) Pratinjau(ctx context.Context, actor entities.User, id string) (string, []byte, error) {
	sr, err := s.repo.GetDetail(ctx, actor, id)
	if err != nil {
		return "", nil, err
	}
	data, err := renderPDF(sr)
	if err != nil {
		return "", nil, err
	}
	return namaBerkas(sr), data, nil
}

func (s *suratService) BuatDokumen(ctx context.Context, actor entities.User, id string) (dto.SuratResponse, error) {
	err := s.transaksi(ctx, func(tx *gorm.DB, simpan func(*entities.Dokumen)) error {
		sr, err := s.repo.GetForUpdate(ctx, tx, actor, id)
		if err != nil {
			return err
		}
		if sr.Status != entities.SuratStatusDraf {
			return utils.ErrConflict("dokumen hanya bisa dibuat dari surat berstatus draf")
		}
		data, err := renderPDF(sr)
		if err != nil {
			return err
		}
		doc, err := s.dokumen.SimpanBytes(ctx, tx, namaBerkas(sr), data, &sr.SekolahID, actor.ID)
		if err != nil {
			return err
		}
		simpan(&doc)

		sr.DokumenID = &doc.ID
		if err := s.catatTransisi(ctx, tx, actor, &sr, entities.SuratStatusMenungguTTD, "surat.buat_dokumen", ""); err != nil {
			return err
		}
		staf, err := s.repo.StafSudinSekolah(ctx, tx, sr.SekolahID)
		if err != nil {
			return err
		}
		return s.notifikasi.Kirim(ctx, tx, staf, "Surat menunggu tanda tangan",
			fmt.Sprintf("Surat %s menunggu tanda tangan basah Kepala Sudin. Unggah pindaian setelah ditandatangani.", sr.Nomor),
			tautan(sr.ID))
	})
	if err != nil {
		return dto.SuratResponse{}, err
	}
	return s.Detail(ctx, actor, id)
}

func (s *suratService) TetapkanFinal(ctx context.Context, actor entities.User, id string, req dto.FinalRequest) (dto.SuratResponse, error) {
	if req.Berkas.Size > MaksUkuranPindai {
		return dto.SuratResponse{}, utils.ErrBadRequest("ukuran hasil pindai maksimal 5 MB")
	}

	err := s.transaksi(ctx, func(tx *gorm.DB, simpan func(*entities.Dokumen)) error {
		sr, err := s.repo.GetForUpdate(ctx, tx, actor, id)
		if err != nil {
			return err
		}
		if sr.Status != entities.SuratStatusMenungguTTD {
			return utils.ErrConflict("hanya surat yang menunggu tanda tangan yang bisa ditetapkan final")
		}
		doc, err := s.dokumen.Simpan(ctx, tx, req.Berkas, &sr.SekolahID, actor.ID)
		if err != nil {
			return err
		}
		simpan(&doc)
		if doc.MimeType != "application/pdf" {
			return utils.ErrBadRequest("hasil pindai harus PDF")
		}

		now := time.Now()
		sr.DokumenTTDID = &doc.ID
		sr.FinalOleh, sr.FinalAt = &actor.ID, &now
		if err := s.catatTransisi(ctx, tx, actor, &sr, entities.SuratStatusFinal, "surat.final", ""); err != nil {
			return err
		}
		kategori := entities.ArsipKategoriSurat
		if sr.Jenis == entities.SuratJenisTunjangan {
			kategori = entities.ArsipKategoriTunjangan
		}
		tgl := sr.TanggalSurat
		if err := arsipkan.Otomatis(tx, arsipkan.Data{
			PendidikID: sr.PendidikID, SekolahID: sr.SekolahID, Kategori: kategori,
			JenisDokumen: templates[sr.Jenis].Label, Nomor: sr.Nomor, TanggalDokumen: &tgl,
			DokumenID: doc.ID, SumberModul: "surat", SumberID: sr.ID, Oleh: actor.ID,
		}); err != nil {
			return err
		}
		operator, err := s.repo.OperatorSekolah(ctx, tx, sr.SekolahID)
		if err != nil {
			return err
		}
		return s.notifikasi.Kirim(ctx, tx, operator, "Surat final",
			fmt.Sprintf("%s · %s %s sudah final. Surat asli akan diserahkan Sudin.", sr.Nomor, templates[sr.Jenis].Label, sr.Pendidik.Nama),
			"/api/dokumen/"+doc.ID.String())
	})
	if err != nil {
		return dto.SuratResponse{}, err
	}
	return s.Detail(ctx, actor, id)
}

func (s *suratService) Batalkan(ctx context.Context, actor entities.User, id string, req dto.BatalRequest) (dto.SuratResponse, error) {
	alasan := strings.TrimSpace(req.Alasan)
	if alasan == "" {
		return dto.SuratResponse{}, utils.ErrBadRequest("alasan pembatalan wajib diisi")
	}
	err := s.transaksi(ctx, func(tx *gorm.DB, _ func(*entities.Dokumen)) error {
		sr, err := s.repo.GetForUpdate(ctx, tx, actor, id)
		if err != nil {
			return err
		}
		if sr.Status != entities.SuratStatusDraf && sr.Status != entities.SuratStatusMenungguTTD {
			return utils.ErrConflict("surat final atau sudah dibatalkan tidak bisa dibatalkan")
		}
		sr.AlasanBatal = alasan
		return s.catatTransisi(ctx, tx, actor, &sr, entities.SuratStatusDibatalkan, "surat.batal", alasan)
	})
	if err != nil {
		return dto.SuratResponse{}, err
	}
	return s.Detail(ctx, actor, id)
}

// Serahkan menandai surat asli sudah diserahkan ke guru/sekolah (S5). Status tetap final.
func (s *suratService) Serahkan(ctx context.Context, actor entities.User, id string) (dto.SuratResponse, error) {
	err := s.transaksi(ctx, func(tx *gorm.DB, _ func(*entities.Dokumen)) error {
		sr, err := s.repo.GetForUpdate(ctx, tx, actor, id)
		if err != nil {
			return err
		}
		if sr.Status != entities.SuratStatusFinal {
			return utils.ErrConflict("hanya surat final yang bisa diserahkan")
		}
		if sr.DiserahkanAt != nil {
			return utils.ErrConflict("surat sudah ditandai diserahkan")
		}
		now := time.Now()
		sr.DiserahkanOleh, sr.DiserahkanAt = &actor.ID, &now
		if err := s.repo.Update(ctx, tx, &sr); err != nil {
			return err
		}
		return aktivitas.Catat(tx, actor.ID, "surat.serahkan", "surat", sr.ID, sr.Nomor)
	})
	if err != nil {
		return dto.SuratResponse{}, err
	}
	return s.Detail(ctx, actor, id)
}

func (s *suratService) Detail(ctx context.Context, actor entities.User, id string) (dto.SuratResponse, error) {
	sr, err := s.repo.GetDetail(ctx, actor, id)
	if err != nil {
		return dto.SuratResponse{}, err
	}
	return respons(sr), nil
}

func (s *suratService) Daftar(ctx context.Context, actor entities.User, f dto.Filter) ([]dto.SuratResponse, int64, error) {
	items, total, err := s.repo.List(ctx, actor, f)
	if err != nil {
		return nil, 0, err
	}
	out := make([]dto.SuratResponse, 0, len(items))
	for _, sr := range items {
		out = append(out, respons(sr))
	}
	return out, total, nil
}

func respons(sr entities.Surat) dto.SuratResponse {
	var sudin *entities.Sudin
	if sr.Sekolah != nil {
		sudin = sr.Sekolah.Sudin
	}
	return dto.FromEntity(sr, templates[sr.Jenis].Label, kop(sudin), penandaTangan(sudin))
}

func tautan(id uuid.UUID) string { return "/surat/" + id.String() }
