package service

import (
	"context"
	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"strings"

	"github.com/Caknoooo/go-gin-clean-starter/modules/user/dto"
	"github.com/Caknoooo/go-gin-clean-starter/modules/user/repository"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/helpers"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/utils"
	"gorm.io/gorm"
)

type UserService interface {
	GetUserById(ctx context.Context, userId string) (dto.UserResponse, error)
	GantiPassword(ctx context.Context, userId string, req dto.GantiPasswordRequest) error
	RiwayatMasuk(ctx context.Context, userId string) ([]dto.RiwayatMasukResponse, error)
	KeluarSesiLain(ctx context.Context, userId string, req dto.KeluarSesiLainRequest) error
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
	// Cabut semua sesi (refresh token) supaya sesi yang mungkin dicuri tidak bertahan,
	// sama seperti reset password oleh Sudin di modul akun.
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.userRepository.UpdatePassword(ctx, tx, userId, hash); err != nil {
			return err
		}
		return tx.Where("user_id = ?", userId).Delete(&entities.RefreshToken{}).Error
	})
}

func (s *userService) RiwayatMasuk(ctx context.Context, userId string) ([]dto.RiwayatMasukResponse, error) {
	rows, err := s.userRepository.RiwayatMasuk(ctx, userId, 10)
	if err != nil {
		return nil, err
	}
	out := make([]dto.RiwayatMasukResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, dto.RiwayatMasukResponse{Waktu: r.CreatedAt, Perangkat: perangkat(r.Keterangan)})
	}
	return out, nil
}

func (s *userService) KeluarSesiLain(ctx context.Context, userId string, req dto.KeluarSesiLainRequest) error {
	return s.userRepository.CabutSesiLain(ctx, userId, req.RefreshToken)
}

// perangkat meringkas User-Agent menjadi "Browser · OS" (mis. "Chrome · Windows").
func perangkat(ua string) string {
	browser, os := "Peramban lain", "OS lain"
	for _, b := range [][2]string{{"Edg/", "Edge"}, {"OPR/", "Opera"}, {"Firefox/", "Firefox"}, {"Chrome/", "Chrome"}, {"Safari/", "Safari"}, {"curl/", "curl"}} {
		if strings.Contains(ua, b[0]) {
			browser = b[1]
			break
		}
	}
	for _, o := range [][2]string{{"Windows", "Windows"}, {"Android", "Android"}, {"iPhone", "iPhone"}, {"iPad", "iPad"}, {"Mac OS", "macOS"}, {"Linux", "Linux"}} {
		if strings.Contains(ua, o[0]) {
			os = o[1]
			break
		}
	}
	return browser + " · " + os
}
