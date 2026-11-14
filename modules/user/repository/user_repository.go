package repository

import (
	"context"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"gorm.io/gorm"
)

type UserRepository interface {
	GetUserById(ctx context.Context, tx *gorm.DB, userId string) (entities.User, error)
	GetUserByEmail(ctx context.Context, tx *gorm.DB, email string) (entities.User, error)
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

func (r *userRepository) UpdatePassword(ctx context.Context, tx *gorm.DB, userId string, hash string) error {
	return r.tx(tx).WithContext(ctx).Model(&entities.User{}).Where("id = ?", userId).Update("password", hash).Error
}
