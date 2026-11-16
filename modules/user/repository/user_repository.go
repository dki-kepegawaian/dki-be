package repository

import (
	"context"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"gorm.io/gorm"
)

type UserRepository interface {
	GetUserById(ctx context.Context, tx *gorm.DB, userId string) (entities.User, error)
	GetUserByEmail(ctx context.Context, tx *gorm.DB, email string) (entities.User, error)
	GetUserByUsername(ctx context.Context, tx *gorm.DB, username string) (entities.User, error)
	RiwayatMasuk(ctx context.Context, userID string, limit int) ([]entities.LogAktivitas, error)
	CabutSesiLain(ctx context.Context, userID string, kecualiToken string) error
	UpdatePassword(ctx context.Context, tx *gorm.DB, userId string, hash string) error
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) tx(tx *gorm.DB) *gorm.DB {
	if tx == nil {
		return r.db
	}
	return tx
}

func (r *userRepository) GetUserById(ctx context.Context, tx *gorm.DB, userId string) (entities.User, error) {
	var user entities.User
	err := r.tx(tx).WithContext(ctx).Preload("Sudin").Preload("Sekolah").Take(&user, "id = ?", userId).Error
	return user, err
}

func (r *userRepository) GetUserByEmail(ctx context.Context, tx *gorm.DB, email string) (entities.User, error) {
	var user entities.User
	err := r.tx(tx).WithContext(ctx).Take(&user, "LOWER(email) = LOWER(?)", email).Error
	return user, err
}

func (r *userRepository) GetUserByUsername(ctx context.Context, tx *gorm.DB, username string) (entities.User, error) {
	var user entities.User
	err := r.tx(tx).WithContext(ctx).Take(&user, "LOWER(username) = LOWER(?)", username).Error
	return user, err
}

func (r *userRepository) RiwayatMasuk(ctx context.Context, userID string, limit int) ([]entities.LogAktivitas, error) {
	var rows []entities.LogAktivitas
	err := r.db.WithContext(ctx).Where("user_id = ? AND aksi = ?", userID, "auth.masuk").
		Order("created_at DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *userRepository) CabutSesiLain(ctx context.Context, userID string, kecualiToken string) error {
	return r.db.WithContext(ctx).Where("user_id = ? AND token <> ?", userID, kecualiToken).
		Delete(&entities.RefreshToken{}).Error
}

func (r *userRepository) UpdatePassword(ctx context.Context, tx *gorm.DB, userId string, hash string) error {
	return r.tx(tx).WithContext(ctx).Model(&entities.User{}).Where("id = ?", userId).Update("password", hash).Error
}
