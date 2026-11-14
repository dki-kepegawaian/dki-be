package service

import (
	"context"
	"time"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"github.com/Caknoooo/go-gin-clean-starter/modules/auth/dto"
	authRepo "github.com/Caknoooo/go-gin-clean-starter/modules/auth/repository"
	"github.com/Caknoooo/go-gin-clean-starter/modules/user/repository"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/helpers"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/utils"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Akun dibuat oleh Sudin lewat modul akun; tidak ada pendaftaran mandiri.
type AuthService interface {
	Login(ctx context.Context, req dto.LoginRequest) (dto.TokenResponse, error)
	RefreshToken(ctx context.Context, req dto.RefreshTokenRequest) (dto.TokenResponse, error)
	Logout(ctx context.Context, userId string) error
}

type authService struct {
	userRepository         repository.UserRepository
	refreshTokenRepository authRepo.RefreshTokenRepository
	jwtService             JWTService
	db                     *gorm.DB
}

func NewAuthService(
	userRepo repository.UserRepository,
	refreshTokenRepo authRepo.RefreshTokenRepository,
	jwtService JWTService,
	db *gorm.DB,
) AuthService {
	return &authService{
		userRepository:         userRepo,
		refreshTokenRepository: refreshTokenRepo,
		jwtService:             jwtService,
		db:                     db,
	}
}

var errKredensial = &utils.HTTPError{Status: 401, Message: "email atau password salah"}

func (s *authService) Login(ctx context.Context, req dto.LoginRequest) (dto.TokenResponse, error) {
	user, err := s.userRepository.GetUserByEmail(ctx, s.db, req.Email)
	if err != nil {
		// Pesan sama dengan password salah supaya keberadaan email tidak terungkap.
		return dto.TokenResponse{}, errKredensial
	}
	if ok, _ := helpers.CheckPassword(user.Password, []byte(req.Password)); !ok {
		return dto.TokenResponse{}, errKredensial
	}
	if !user.IsAktif {
		return dto.TokenResponse{}, utils.ErrForbidden("akun dinonaktifkan, hubungi Sudin")
	}
	return s.issue(ctx, user)
}

func (s *authService) RefreshToken(ctx context.Context, req dto.RefreshTokenRequest) (dto.TokenResponse, error) {
	rt, err := s.refreshTokenRepository.FindByToken(ctx, s.db, req.RefreshToken)
	if err != nil || rt.ExpiresAt.Before(time.Now()) {
		return dto.TokenResponse{}, &utils.HTTPError{Status: 401, Message: "refresh token tidak berlaku"}
	}
	if err := s.refreshTokenRepository.DeleteByToken(ctx, s.db, req.RefreshToken); err != nil {
		return dto.TokenResponse{}, err
	}
	if !rt.User.IsAktif {
		return dto.TokenResponse{}, utils.ErrForbidden("akun dinonaktifkan, hubungi Sudin")
	}
	return s.issue(ctx, rt.User)
}

func (s *authService) Logout(ctx context.Context, userId string) error {
	return s.refreshTokenRepository.DeleteByUserID(ctx, s.db, userId)
}

func (s *authService) issue(ctx context.Context, user entities.User) (dto.TokenResponse, error) {
	access := s.jwtService.GenerateAccessToken(user.ID.String(), user.Role)
	refresh, expiresAt := s.jwtService.GenerateRefreshToken()
	if _, err := s.refreshTokenRepository.Create(ctx, s.db, entities.RefreshToken{
		ID: uuid.New(), UserID: user.ID, Token: refresh, ExpiresAt: expiresAt,
	}); err != nil {
		return dto.TokenResponse{}, err
	}
	return dto.TokenResponse{AccessToken: access, RefreshToken: refresh, Role: user.Role}, nil
}
