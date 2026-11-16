package service

import (
	"bytes"
	"context"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"github.com/Caknoooo/go-gin-clean-starter/modules/dokumen/repository"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/akses"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/constants"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/storage"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/utils"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const MaxUkuran = 10 << 20 // 10 MB

var tipeDiizinkan = map[string]string{
	"application/pdf": ".pdf",
	"image/jpeg":      ".jpg",
	"image/png":       ".png",
}

type DokumenService interface {
	// Simpan mengunggah berkas ke object storage lalu mencatat metadatanya lewat tx.
	// Jika transaksi pemanggil gagal, panggil Hapus untuk membersihkan objeknya.
	Simpan(ctx context.Context, tx *gorm.DB, fh *multipart.FileHeader, sekolahID *uuid.UUID, userID uuid.UUID) (entities.Dokumen, error)
	// SimpanBytes sama dengan Simpan untuk berkas yang dibuat server (mis. PDF hasil generate).
	SimpanBytes(ctx context.Context, tx *gorm.DB, nama string, data []byte, sekolahID *uuid.UUID, userID uuid.UUID) (entities.Dokumen, error)
	Hapus(docs ...entities.Dokumen)
	// Buka mengembalikan dokumen dan isinya jika actor berhak. Pemanggil wajib menutup reader.
	Buka(ctx context.Context, actor entities.User, id string) (entities.Dokumen, io.ReadCloser, int64, error)
}

type dokumenService struct {
	repo    repository.DokumenRepository
	db      *gorm.DB
	storage storage.Storage
}

func NewDokumenService(repo repository.DokumenRepository, db *gorm.DB, st storage.Storage) DokumenService {
	return &dokumenService{repo: repo, db: db, storage: st}
}

func (s *dokumenService) Simpan(ctx context.Context, tx *gorm.DB, fh *multipart.FileHeader, sekolahID *uuid.UUID, userID uuid.UUID) (entities.Dokumen, error) {
	if fh == nil {
		return entities.Dokumen{}, utils.ErrBadRequest("berkas wajib diunggah")
	}
	if fh.Size > MaxUkuran {
		return entities.Dokumen{}, utils.ErrBadRequest("ukuran berkas maksimal 10 MB")
	}

	src, err := fh.Open()
	if err != nil {
		return entities.Dokumen{}, err
	}
	defer src.Close()

	head := make([]byte, 512)
	n, _ := io.ReadFull(src, head)
	mime := http.DetectContentType(head[:n])
	ext, ok := tipeDiizinkan[mime]
	if !ok {
		return entities.Dokumen{}, utils.ErrBadRequest("berkas harus PDF, JPG, atau PNG")
	}
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return entities.Dokumen{}, err
	}
	return s.simpan(ctx, tx, src, fh.Size, mime, ext, fh.Filename, sekolahID, userID)
}

func (s *dokumenService) SimpanBytes(ctx context.Context, tx *gorm.DB, nama string, data []byte, sekolahID *uuid.UUID, userID uuid.UUID) (entities.Dokumen, error) {
	if len(data) == 0 || len(data) > MaxUkuran {
		return entities.Dokumen{}, utils.ErrBadRequest("ukuran berkas tidak valid")
	}
	mime := http.DetectContentType(data)
	ext, ok := tipeDiizinkan[mime]
	if !ok {
		return entities.Dokumen{}, utils.ErrBadRequest("berkas harus PDF, JPG, atau PNG")
	}
	return s.simpan(ctx, tx, bytes.NewReader(data), int64(len(data)), mime, ext, nama, sekolahID, userID)
}

func (s *dokumenService) simpan(ctx context.Context, tx *gorm.DB, r io.Reader, size int64, mime, ext, nama string, sekolahID *uuid.UUID, userID uuid.UUID) (entities.Dokumen, error) {
	id := uuid.New()
	key := path.Join("dokumen", time.Now().Format("2006/01"), id.String()+ext)
	if err := s.storage.Put(ctx, key, r, size, mime); err != nil {
		return entities.Dokumen{}, err
	}

	nama = path.Base(strings.ReplaceAll(nama, "\\", "/"))
	if r := []rune(nama); len(r) > 200 {
		nama = string(r[:200]) // kolom nama_asli varchar(255)
	}
	doc := entities.Dokumen{
		ID:           id,
		SekolahID:    sekolahID,
		DiunggahOleh: userID,
		NamaAsli:     nama,
		Path:         key,
		MimeType:     mime,
		Ukuran:       size,
	}
	if err := s.repo.Create(ctx, tx, &doc); err != nil {
		s.Hapus(doc)
		return entities.Dokumen{}, err
	}
	return doc, nil
}

func (s *dokumenService) Hapus(docs ...entities.Dokumen) {
	for _, d := range docs {
		if d.Path == "" {
			continue
		}
		// Context terpisah: request asal bisa sudah dibatalkan saat pembersihan.
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := s.storage.Delete(ctx, d.Path); err != nil {
			log.Printf("gagal menghapus objek %s: %v", d.Path, err)
		}
		cancel()
	}
}

func (s *dokumenService) Buka(ctx context.Context, actor entities.User, id string) (entities.Dokumen, io.ReadCloser, int64, error) {
	doc, err := s.repo.GetByID(ctx, nil, id)
	if err != nil {
		return entities.Dokumen{}, nil, 0, err
	}

	// Akses mengikuti penempatan sekarang, bukan siapa pengunggahnya: akun yang dipindah
	// tidak boleh tetap membuka berkas lama. Berkas tanpa sekolah_id hanya untuk admin
	// di sini; modul pemiliknya menyediakan endpoint unduh sendiri.
	boleh := actor.Role == constants.ENUM_ROLE_ADMIN
	if !boleh && doc.SekolahID != nil {
		boleh, err = akses.BolehSekolah(s.db.WithContext(ctx), actor, *doc.SekolahID)
		if err != nil {
			return entities.Dokumen{}, nil, 0, err
		}
	}
	if !boleh {
		// 404, bukan 403, supaya keberadaan dokumen sekolah lain tidak terungkap.
		return entities.Dokumen{}, nil, 0, utils.ErrNotFound("dokumen tidak ditemukan")
	}

	r, size, err := s.storage.Get(ctx, doc.Path)
	if err != nil {
		return entities.Dokumen{}, nil, 0, err
	}
	return doc, r, size, nil
}
