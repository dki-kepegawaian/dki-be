package repository

import (
	"context"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/akses"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Filter struct {
	Search  string
	Jenjang string
	Offset  int
	Limit   int
}

type SekolahRepository interface {
	List(ctx context.Context, actor entities.User, f Filter) ([]entities.Sekolah, int64, error)
	GetByID(ctx context.Context, actor entities.User, id string) (entities.Sekolah, error)
}

type sekolahRepository struct {
	db *gorm.DB
}

func NewSekolahRepository(db *gorm.DB) SekolahRepository {
	return &sekolahRepository{db: db}
}

func (r *sekolahRepository) List(ctx context.Context, actor entities.User, f Filter) ([]entities.Sekolah, int64, error) {
	q := r.db.WithContext(ctx).Model(&entities.Sekolah{}).Scopes(akses.Scope(actor, "sekolah.id"))
	if f.Search != "" {
		q = q.Where("(nama ILIKE ? OR npsn = ?)", "%"+f.Search+"%", f.Search)
	}
	if f.Jenjang != "" {
		q = q.Where("jenjang = ?", f.Jenjang)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []entities.Sekolah
	err := q.Preload("Sudin").Order("nama ASC").Offset(f.Offset).Limit(f.Limit).Find(&items).Error
	return items, total, err
}

func (r *sekolahRepository) GetByID(ctx context.Context, actor entities.User, id string) (entities.Sekolah, error) {
	var s entities.Sekolah
	if _, err := uuid.Parse(id); err != nil {
		return s, gorm.ErrRecordNotFound
	}
	err := r.db.WithContext(ctx).Scopes(akses.Scope(actor, "sekolah.id")).Preload("Sudin").Take(&s, "sekolah.id = ?", id).Error
	return s, err
}
