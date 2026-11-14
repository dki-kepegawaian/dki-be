package repository

import (
	"context"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DokumenRepository interface {
	Create(ctx context.Context, tx *gorm.DB, doc *entities.Dokumen) error
	GetByID(ctx context.Context, tx *gorm.DB, id string) (entities.Dokumen, error)
}

type dokumenRepository struct {
	db *gorm.DB
}

func NewDokumenRepository(db *gorm.DB) DokumenRepository {
	return &dokumenRepository{db: db}
}

func (r *dokumenRepository) Create(ctx context.Context, tx *gorm.DB, doc *entities.Dokumen) error {
	if tx == nil {
		tx = r.db
	}
	return tx.WithContext(ctx).Create(doc).Error
}

func (r *dokumenRepository) GetByID(ctx context.Context, tx *gorm.DB, id string) (entities.Dokumen, error) {
	if tx == nil {
		tx = r.db
	}
	var doc entities.Dokumen
	if _, err := uuid.Parse(id); err != nil {
		return doc, gorm.ErrRecordNotFound
	}
	err := tx.WithContext(ctx).Take(&doc, "id = ?", id).Error
	return doc, err
}
