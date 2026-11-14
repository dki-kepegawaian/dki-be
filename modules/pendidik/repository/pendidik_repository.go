package repository

import (
	"context"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/akses"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Filter struct {
	Search            string
	SekolahID         string
	StatusKepegawaian string
	Offset            int
	Limit             int
}

type PendidikRepository interface {
	List(ctx context.Context, actor entities.User, f Filter) ([]entities.Pendidik, int64, error)
	GetByID(ctx context.Context, actor entities.User, id string) (entities.Pendidik, error)
	Save(ctx context.Context, p *entities.Pendidik) error
}

type pendidikRepository struct {
	db *gorm.DB
}

func NewPendidikRepository(db *gorm.DB) PendidikRepository {
	return &pendidikRepository{db: db}
}

func (r *pendidikRepository) List(ctx context.Context, actor entities.User, f Filter) ([]entities.Pendidik, int64, error) {
	q := r.db.WithContext(ctx).Model(&entities.Pendidik{}).Scopes(akses.Scope(actor, "pendidik.sekolah_id"))
	if f.Search != "" {
		q = q.Where("(pendidik.nama ILIKE ? OR pendidik.nip LIKE ?)", "%"+f.Search+"%", f.Search+"%")
	}
	if f.SekolahID != "" {
		q = q.Where("pendidik.sekolah_id = ?", f.SekolahID)
	}
	if f.StatusKepegawaian != "" {
		q = q.Where("pendidik.status_kepegawaian = ?", f.StatusKepegawaian)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []entities.Pendidik
	err := q.Preload("Sekolah").Order("pendidik.nama ASC").Offset(f.Offset).Limit(f.Limit).Find(&items).Error
	return items, total, err
}

func (r *pendidikRepository) GetByID(ctx context.Context, actor entities.User, id string) (entities.Pendidik, error) {
	var p entities.Pendidik
	if _, err := uuid.Parse(id); err != nil {
		return p, gorm.ErrRecordNotFound
	}
	err := r.db.WithContext(ctx).Scopes(akses.Scope(actor, "pendidik.sekolah_id")).
		Preload("Sekolah").Take(&p, "pendidik.id = ?", id).Error
	return p, err
}

func (r *pendidikRepository) Save(ctx context.Context, p *entities.Pendidik) error {
	return r.db.WithContext(ctx).Omit("Sekolah").Save(p).Error
}
