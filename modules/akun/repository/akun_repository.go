package repository

import (
	"context"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"github.com/Caknoooo/go-gin-clean-starter/modules/akun/dto"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/constants"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AkunRepository interface {
	List(ctx context.Context, actor entities.User, f dto.Filter) ([]entities.User, int64, error)
	GetByID(ctx context.Context, actor entities.User, id string) (entities.User, error)
	EmailDipakai(ctx context.Context, email string, kecuali uuid.UUID) (bool, error)
	Create(ctx context.Context, u *entities.User) error
	Save(ctx context.Context, tx *gorm.DB, u *entities.User) error
	UpdatePassword(ctx context.Context, tx *gorm.DB, id uuid.UUID, hash string) error
	CabutSesi(ctx context.Context, tx *gorm.DB, id uuid.UUID) error
	Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error
}

type akunRepository struct {
	db *gorm.DB
}

func NewAkunRepository(db *gorm.DB) AkunRepository {
	return &akunRepository{db: db}
}

// scope: admin melihat semua akun. Staf Sudin melihat operator di sekolah wilayahnya
// serta staf/pimpinan di Sudin-nya.
func scope(actor entities.User) func(*gorm.DB) *gorm.DB {
	return func(q *gorm.DB) *gorm.DB {
		if actor.Role == constants.ENUM_ROLE_ADMIN {
			return q
		}
		if actor.SudinID == nil {
			return q.Where("1 = 0")
		}
		sekolah := q.Session(&gorm.Session{NewDB: true}).Model(&entities.Sekolah{}).Select("id").Where("sudin_id = ?", *actor.SudinID)
		return q.Where("((users.role = ? AND users.sekolah_id IN (?)) OR (users.role IN ? AND users.sudin_id = ?))",
			constants.ENUM_ROLE_OPERATOR_SEKOLAH, sekolah,
			[]string{constants.ENUM_ROLE_STAF_SUDIN, constants.ENUM_ROLE_PIMPINAN}, *actor.SudinID)
	}
}

func (r *akunRepository) List(ctx context.Context, actor entities.User, f dto.Filter) ([]entities.User, int64, error) {
	q := r.db.WithContext(ctx).Model(&entities.User{}).Scopes(scope(actor))
	if f.Search != "" {
		q = q.Where("(users.name ILIKE ? OR users.email ILIKE ?)", "%"+f.Search+"%", "%"+f.Search+"%")
	}
	if f.Role != "" {
		q = q.Where("users.role = ?", f.Role)
	}
	if f.SekolahID != "" {
		if _, err := uuid.Parse(f.SekolahID); err != nil {
			return nil, 0, nil
		}
		q = q.Where("users.sekolah_id = ?", f.SekolahID)
	}
	if f.Aktif != nil {
		q = q.Where("users.is_aktif = ?", *f.Aktif)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []entities.User
	err := q.Preload("Sudin").Preload("Sekolah").Order("users.name ASC").Offset(f.Offset).Limit(f.Limit).Find(&items).Error
	return items, total, err
}

func (r *akunRepository) GetByID(ctx context.Context, actor entities.User, id string) (entities.User, error) {
	var u entities.User
	if _, err := uuid.Parse(id); err != nil {
		return u, gorm.ErrRecordNotFound
	}
	err := r.db.WithContext(ctx).Scopes(scope(actor)).Preload("Sudin").Preload("Sekolah").Take(&u, "users.id = ?", id).Error
	return u, err
}

func (r *akunRepository) EmailDipakai(ctx context.Context, email string, kecuali uuid.UUID) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&entities.User{}).Where("LOWER(email) = LOWER(?) AND id <> ?", email, kecuali).Count(&n).Error
	return n > 0, err
}

func (r *akunRepository) Create(ctx context.Context, u *entities.User) error {
	return r.db.WithContext(ctx).Omit(clause.Associations).Create(u).Error
}

func (r *akunRepository) Save(ctx context.Context, tx *gorm.DB, u *entities.User) error {
	return tx.WithContext(ctx).Omit(clause.Associations, "password").Save(u).Error
}

func (r *akunRepository) UpdatePassword(ctx context.Context, tx *gorm.DB, id uuid.UUID, hash string) error {
	return tx.WithContext(ctx).Model(&entities.User{}).Where("id = ?", id).Update("password", hash).Error
}

func (r *akunRepository) CabutSesi(ctx context.Context, tx *gorm.DB, id uuid.UUID) error {
	return tx.WithContext(ctx).Where("user_id = ?", id).Delete(&entities.RefreshToken{}).Error
}

func (r *akunRepository) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}
