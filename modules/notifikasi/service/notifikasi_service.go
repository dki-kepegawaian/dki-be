package service

import (
	"context"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"github.com/Caknoooo/go-gin-clean-starter/modules/notifikasi/repository"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/utils"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type NotifikasiService interface {
	// Kirim membuat notifikasi untuk tiap penerima di transaksi pemanggil.
	Kirim(ctx context.Context, tx *gorm.DB, penerima []uuid.UUID, judul, pesan, tautan string) error
	Daftar(ctx context.Context, userID uuid.UUID, hanyaBelumDibaca bool, offset, limit int) ([]entities.Notifikasi, int64, int64, error)
	TandaiDibaca(ctx context.Context, userID uuid.UUID, id string) error
	TandaiSemuaDibaca(ctx context.Context, userID uuid.UUID) error
}

type notifikasiService struct {
	repo repository.NotifikasiRepository
}

func NewNotifikasiService(repo repository.NotifikasiRepository) NotifikasiService {
	return &notifikasiService{repo: repo}
}

func (s *notifikasiService) Kirim(ctx context.Context, tx *gorm.DB, penerima []uuid.UUID, judul, pesan, tautan string) error {
	items := make([]entities.Notifikasi, 0, len(penerima))
	seen := map[uuid.UUID]bool{}
	for _, id := range penerima {
		if id == uuid.Nil || seen[id] {
			continue
		}
		seen[id] = true
		items = append(items, entities.Notifikasi{UserID: id, Judul: judul, Pesan: pesan, Tautan: tautan})
	}
	return s.repo.CreateBatch(ctx, tx, items)
}

func (s *notifikasiService) Daftar(ctx context.Context, userID uuid.UUID, hanyaBelumDibaca bool, offset, limit int) ([]entities.Notifikasi, int64, int64, error) {
	items, total, err := s.repo.ListByUser(ctx, userID, hanyaBelumDibaca, offset, limit)
	if err != nil {
		return nil, 0, 0, err
	}
	belum, err := s.repo.CountBelumDibaca(ctx, userID)
	return items, total, belum, err
}

func (s *notifikasiService) TandaiDibaca(ctx context.Context, userID uuid.UUID, id string) error {
	n, err := s.repo.TandaiDibaca(ctx, userID, id)
	if err != nil {
		return err
	}
	if n == 0 {
		return utils.ErrNotFound("notifikasi tidak ditemukan")
	}
	return nil
}

func (s *notifikasiService) TandaiSemuaDibaca(ctx context.Context, userID uuid.UUID) error {
	return s.repo.TandaiSemuaDibaca(ctx, userID)
}
