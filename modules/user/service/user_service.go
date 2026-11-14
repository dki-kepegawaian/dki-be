package service

import (
	"context"

	"github.com/Caknoooo/go-gin-clean-starter/modules/user/dto"
	"github.com/Caknoooo/go-gin-clean-starter/modules/user/repository"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/helpers"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/utils"
	"gorm.io/gorm"
)

type UserService interface {
	GetUserById(ctx context.Context, userId string) (dto.UserResponse, error)
	GantiPassword(ctx context.Context, userId string, req dto.GantiPasswordRequest) error
}

type userService struct {
	userRepository repository.UserRepository
	db             *gorm.DB
}

func NewUserService(userRepo repository.UserRepository, db *gorm.DB) UserService {
	return &userService{userRepository: userRepo, db: db}
}

func (s *userService) GetUserById(ctx context.Context, userId string) (dto.UserResponse, error) {
	user, err := s.userRepository.GetUserById(ctx, s.db, userId)
	if err != nil {
		return dto.UserResponse{}, err
	}
	return dto.FromEntity(user), nil
}

func (s *userService) GantiPassword(ctx context.Context, userId string, req dto.GantiPasswordRequest) error {
	user, err := s.userRepository.GetUserById(ctx, s.db, userId)
	if err != nil {
		return err
	}
	if ok, _ := helpers.CheckPassword(user.Password, []byte(req.PasswordLama)); !ok {
		return utils.ErrBadRequest("password lama salah")
	}
	hash, err := helpers.HashPassword(req.PasswordBaru)
	if err != nil {
		return err
	}
	return s.userRepository.UpdatePassword(ctx, s.db, userId, hash)
}
