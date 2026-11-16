package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/helpers"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"github.com/Caknoooo/go-gin-clean-starter/modules/arsip/dto"
	"github.com/Caknoooo/go-gin-clean-starter/modules/arsip/repository"
	dokumenService "github.com/Caknoooo/go-gin-clean-starter/modules/dokumen/service"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/constants"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/utils"
	"gorm.io/gorm"
)

const maxBerkasArsip = 5 << 20 // R3: maks. 5 MB per file

type ArsipService interface {
	Daftar(ctx context.Context, actor entities.User, f dto.Filter) ([]dto.ArsipResponse, int64, error)
	Ringkasan(ctx context.Context, actor entities.User, pendidikID string) (dto.RingkasanResponse, error)
	Detail(ctx context.Context, actor entities.User, id string) (dto.ArsipResponse, error)
	Unggah(ctx context.Context, actor entities.User, req dto.UnggahRequest) (dto.ArsipResponse, error)
	VersiBaru(ctx context.Context, actor entities.User, id string, req dto.VersiRequest) (dto.ArsipResponse, error)
	// Buka mengembalikan isi berkas satu versi (0 = terkini) dan mencatat log melihat/mengunduh.
	Buka(ctx context.Context, actor entities.User, id string, versi int, inline bool) (entities.Dokumen, io.ReadCloser, int64, error)
	Log(ctx context.Context, actor entities.User, f dto.LogFilter) ([]dto.LogResponse, int64, error)
	LogCSV(ctx context.Context, actor entities.User, f dto.LogFilter) ([]byte, error)
}

type arsipService struct {
	repo    repository.ArsipRepository
	dokumen dokumenService.DokumenService
	db      *gorm.DB
}

func NewArsipService(repo repository.ArsipRepository, dokumen dokumenService.DokumenService, db *gorm.DB) ArsipService {
	return &arsipService{repo: repo, dokumen: dokumen, db: db}
}

// cekBerkas: R3 menerima PDF atau JPG, maks. 5 MB per file.
func cekBerkas(fh *multipart.FileHeader) error {
	if fh == nil {
		return nil
	}
	if fh.Size > maxBerkasArsip {
		return utils.ErrBadRequest("ukuran berkas maksimal 5 MB")
	}
	f, err := fh.Open()
	if err != nil {
		return err
	}
	defer f.Close()
	head := make([]byte, 512)
	n, _ := io.ReadFull(f, head)
	if m := http.DetectContentType(head[:n]); m != "application/pdf" && m != "image/jpeg" {
		return utils.ErrBadRequest("berkas arsip harus PDF atau JPG")
	}
	return nil
}

// simpanVersi menyimpan berkas (+lampiran) lalu menulis versi dan log, di dalam transaksi pemanggil.
func (s *arsipService) simpanVersi(ctx context.Context, tx *gorm.DB, actor entities.User, a entities.Arsip, versi int,
	berkas, lampiran *multipart.FileHeader, catatan, aksi string, simpan func(entities.Dokumen)) error {
	doc, err := s.dokumen.Simpan(ctx, tx, berkas, &a.SekolahID, actor.ID)
	if err != nil {
		return err
	}
	simpan(doc)
	v := entities.ArsipVersi{ArsipID: a.ID, Versi: versi, DokumenID: doc.ID, DiunggahOleh: actor.ID, Catatan: catatan}
	if lampiran != nil {
		lamp, err := s.dokumen.Simpan(ctx, tx, lampiran, &a.SekolahID, actor.ID)
		if err != nil {
			return err
		}
		simpan(lamp)
		v.LampiranDokumenID = &lamp.ID
	}
	if err := tx.WithContext(ctx).Omit("Dokumen", "LampiranDokumen", "Pengunggah").Create(&v).Error; err != nil {
		return err
	}
	return s.repo.CatatLog(ctx, tx, &entities.ArsipLog{ArsipID: a.ID, Versi: versi, Aksi: aksi, UserID: actor.ID, Keterangan: catatan})
}

func (s *arsipService) transaksi(ctx context.Context, fn func(tx *gorm.DB, simpan func(entities.Dokumen)) error) error {
	var tersimpan []entities.Dokumen
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(tx, func(d entities.Dokumen) { tersimpan = append(tersimpan, d) })
	})
	if err != nil {
		s.dokumen.Hapus(tersimpan...)
	}
	return err
}

func (s *arsipService) Unggah(ctx context.Context, actor entities.User, req dto.UnggahRequest) (dto.ArsipResponse, error) {
	if err := cekBerkas(req.Berkas); err != nil {
		return dto.ArsipResponse{}, err
	}
	if err := cekBerkas(req.Lampiran); err != nil {
		return dto.ArsipResponse{}, err
	}
	p, err := s.repo.GetPendidik(ctx, actor, req.PendidikID)
	if err != nil {
		return dto.ArsipResponse{}, utils.ErrBadRequest("pendidik tidak ditemukan di wilayah Anda")
	}
	a := entities.Arsip{
		PendidikID: p.ID, SekolahID: p.SekolahID, Kategori: req.Kategori, JenisDokumen: req.JenisDokumen,
		Nomor: req.Nomor, Keterangan: req.Keterangan, VersiTerkini: 1,
	}
	if t, err := time.Parse("2006-01-02", req.TanggalDokumen); err == nil {
		a.TanggalDokumen = &t
	}
	err = s.transaksi(ctx, func(tx *gorm.DB, simpan func(entities.Dokumen)) error {
		if err := tx.WithContext(ctx).Omit("Pendidik", "Sekolah", "Versi").Create(&a).Error; err != nil {
			return err
		}
		return s.simpanVersi(ctx, tx, actor, a, 1, req.Berkas, req.Lampiran, "Unggahan awal", entities.ArsipAksiMengunggah, simpan)
	})
	if err != nil {
		return dto.ArsipResponse{}, err
	}
	return s.Detail(ctx, actor, a.ID.String())
}

func (s *arsipService) VersiBaru(ctx context.Context, actor entities.User, id string, req dto.VersiRequest) (dto.ArsipResponse, error) {
	if err := cekBerkas(req.Berkas); err != nil {
		return dto.ArsipResponse{}, err
	}
	if err := cekBerkas(req.Lampiran); err != nil {
		return dto.ArsipResponse{}, err
	}
	err := s.transaksi(ctx, func(tx *gorm.DB, simpan func(entities.Dokumen)) error {
		a, err := s.repo.Get(ctx, tx, actor, id, true)
		if err != nil {
			return err
		}
		// Dokumen hasil modul (mis. surat izin cuti) hanya boleh diganti versinya oleh Sudin.
		if a.SumberModul != "" && actor.Role == constants.ENUM_ROLE_OPERATOR_SEKOLAH {
			return utils.ErrForbidden("dokumen dari modul " + a.SumberModul + " hanya bisa diperbarui oleh Sudin")
		}
		next := a.VersiTerkini + 1
		if err := tx.WithContext(ctx).Model(&entities.Arsip{}).Where("id = ?", a.ID).
			Updates(map[string]any{"versi_terkini": next, "updated_at": time.Now()}).Error; err != nil {
			return err
		}
		return s.simpanVersi(ctx, tx, actor, a, next, req.Berkas, req.Lampiran, req.Catatan, entities.ArsipAksiMenggantiVers, simpan)
	})
	if err != nil {
		return dto.ArsipResponse{}, err
	}
	return s.Detail(ctx, actor, id)
}

func (s *arsipService) Detail(ctx context.Context, actor entities.User, id string) (dto.ArsipResponse, error) {
	a, err := s.repo.Get(ctx, nil, actor, id, false)
	if err != nil {
		return dto.ArsipResponse{}, err
	}
	return dto.FromEntity(a, true), nil
}

func (s *arsipService) Daftar(ctx context.Context, actor entities.User, f dto.Filter) ([]dto.ArsipResponse, int64, error) {
	items, total, err := s.repo.List(ctx, actor, f)
	if err != nil {
		return nil, 0, err
	}
	out := make([]dto.ArsipResponse, 0, len(items))
	for _, a := range items {
		out = append(out, dto.FromEntity(a, false))
	}
	return out, total, nil
}

func (s *arsipService) Ringkasan(ctx context.Context, actor entities.User, pendidikID string) (dto.RingkasanResponse, error) {
	if _, err := s.repo.GetPendidik(ctx, actor, pendidikID); err != nil {
		return dto.RingkasanResponse{}, err
	}
	per, err := s.repo.Ringkasan(ctx, actor, pendidikID)
	if err != nil {
		return dto.RingkasanResponse{}, err
	}
	var total int64
	for _, n := range per {
		total += n
	}
	return dto.RingkasanResponse{Total: total, PerKategori: per}, nil
}

func (s *arsipService) Buka(ctx context.Context, actor entities.User, id string, versi int, inline bool) (entities.Dokumen, io.ReadCloser, int64, error) {
	a, err := s.repo.Get(ctx, nil, actor, id, false)
	if err != nil {
		return entities.Dokumen{}, nil, 0, err
	}
	if versi == 0 {
		versi = a.VersiTerkini
	}
	var v *entities.ArsipVersi
	for i := range a.Versi {
		if a.Versi[i].Versi == versi {
			v = &a.Versi[i]
		}
	}
	if v == nil {
		return entities.Dokumen{}, nil, 0, utils.ErrNotFound("versi tidak ditemukan")
	}
	doc, r, size, err := s.dokumen.Buka(ctx, actor, v.DokumenID.String())
	if err != nil {
		return entities.Dokumen{}, nil, 0, err
	}
	aksi := entities.ArsipAksiMengunduh
	if inline {
		aksi = entities.ArsipAksiMelihat
	}
	if err := s.repo.CatatLog(ctx, nil, &entities.ArsipLog{ArsipID: a.ID, Versi: versi, Aksi: aksi, UserID: actor.ID}); err != nil {
		r.Close()
		return entities.Dokumen{}, nil, 0, err
	}
	return doc, r, size, nil
}

func (s *arsipService) Log(ctx context.Context, actor entities.User, f dto.LogFilter) ([]dto.LogResponse, int64, error) {
	return s.repo.Log(ctx, actor, f)
}

func (s *arsipService) LogCSV(ctx context.Context, actor entities.User, f dto.LogFilter) ([]byte, error) {
	f.Offset, f.Limit = 0, 10000
	rows, _, err := s.repo.Log(ctx, actor, f)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"waktu", "aksi", "versi", "pengguna", "peran", "jenis_dokumen", "nomor", "guru", "keterangan"})
	for _, r := range rows {
		_ = w.Write([]string{r.Waktu.Format(time.RFC3339), r.Aksi, strconv.Itoa(r.Versi), aman(r.Pengguna), r.Peran,
			aman(r.JenisDokumen), aman(r.Nomor), aman(r.PendidikNama), aman(r.Keterangan)})
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}

// aman mencegah CSV/formula injection saat file dibuka di spreadsheet.
func aman(s string) string { return helpers.AmanCSV(s) }
