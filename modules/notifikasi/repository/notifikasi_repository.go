package repository

import (
	"context"
	"time"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type NotifikasiRepository interface {
	CreateBatch(ctx context.Context, tx *gorm.DB, items []entities.Notifikasi) error
	ListByUser(ctx context.Context, userID uuid.UUID, hanyaBelumDibaca bool, offset, limit int) ([]entities.Notifikasi, int64, error)
	CountBelumDibaca(ctx context.Context, userID uuid.UUID) (int64, error)
	TandaiDibaca(ctx context.Context, userID uuid.UUID, id string) (int64, error)
	TandaiSemuaDibaca(ctx context.Context, userID uuid.UUID) error
}

type notifikasiRepository struct {
	db *gorm.DB
}

func NewNotifikasiRepository(db *gorm.DB) NotifikasiRepository {
	return &notifikasiRepository{db: db}
}

func (r *notifikasiRepository) CreateBatch(ctx context.Context, tx *gorm.DB, items []entities.Notifikasi) error {
	if len(items) == 0 {
		return nil
	}
	if tx == nil {
		tx = r.db
	}
	return tx.WithContext(ctx).Create(&items).Error
}

func (r *notifikasiRepository) ListByUser(ctx context.Context, userID uuid.UUID, hanyaBelumDibaca bool, offset, limit int) ([]entities.Notifikasi, int64, error) {
	q := r.db.WithContext(ctx).Model(&entities.Notifikasi{}).Where("user_id = ?", userID)
	if hanyaBelumDibaca {
		q = q.Where("dibaca_at IS NULL")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []entities.Notifikasi
	err := q.Order("created_at DESC").Offset(offset).Limit(limit).Find(&items).Error
	return items, total, err
}

func (r *notifikasiRepository) CountBelumDibaca(ctx context.Context, userID uuid.UUID) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&entities.Notifikasi{}).Where("user_id = ? AND dibaca_at IS NULL", userID).Count(&n).Error
	return n, err
}

func (r *notifikasiRepository) TandaiDibaca(ctx context.Context, userID uuid.UUID, id string) (int64, error) {
	if _, err := uuid.Parse(id); err != nil {
		return 0, nil
	}
	res := r.db.WithContext(ctx).Model(&entities.Notifikasi{}).
		Where("id = ? AND user_id = ? AND dibaca_at IS NULL", id, userID).
		Update("dibaca_at", time.Now())
	if res.Error != nil {
		return 0, res.Error
	}
	if res.RowsAffected == 0 {
		// Sudah dibaca sebelumnya juga dianggap berhasil; cek kepemilikan saja.
		var n int64
		err := r.db.WithContext(ctx).Model(&entities.Notifikasi{}).Where("id = ? AND user_id = ?", id, userID).Count(&n).Error
		return n, err
	}
	return res.RowsAffected, nil
}

func (r *notifikasiRepository) TandaiSemuaDibaca(ctx context.Context, userID uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&entities.Notifikasi{}).
		Where("user_id = ? AND dibaca_at IS NULL", userID).
		Update("dibaca_at", time.Now()).Error
}
