package service

import (
	"context"
	"regexp"
	"strings"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"github.com/Caknoooo/go-gin-clean-starter/modules/akun/dto"
	"github.com/Caknoooo/go-gin-clean-starter/modules/akun/repository"
	userDto "github.com/Caknoooo/go-gin-clean-starter/modules/user/dto"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/akses"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/aktivitas"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/constants"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/helpers"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/utils"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AkunService interface {
	Daftar(ctx context.Context, actor entities.User, f dto.Filter) ([]userDto.UserResponse, int64, error)
	Detail(ctx context.Context, actor entities.User, id string) (userDto.UserResponse, error)
	Tambah(ctx context.Context, actor entities.User, req dto.TambahRequest) (userDto.UserResponse, error)
	Ubah(ctx context.Context, actor entities.User, id string, req dto.UbahRequest) (userDto.UserResponse, error)
	ResetPassword(ctx context.Context, actor entities.User, id string, req dto.ResetPasswordRequest) error
	Nonaktifkan(ctx context.Context, actor entities.User, id string) error
}

type akunService struct {
	repo repository.AkunRepository
	db   *gorm.DB
}

func NewAkunService(repo repository.AkunRepository, db *gorm.DB) AkunService {
	return &akunService{repo: repo, db: db}
}

// bolehKelolaPeran: admin mengelola semua peran. Staf Sudin mengelola operator sekolah
// dan sesama staf di wilayahnya; akun pimpinan dan admin hanya dikelola admin.
func bolehKelolaPeran(actor entities.User, role string) bool {
	if actor.Role == constants.ENUM_ROLE_ADMIN {
		return true
	}
	return actor.Role == constants.ENUM_ROLE_STAF_SUDIN &&
		(role == constants.ENUM_ROLE_OPERATOR_SEKOLAH || role == constants.ENUM_ROLE_STAF_SUDIN)
}

// penempatan memvalidasi pasangan peran + sekolah/Sudin dan mengembalikan nilai yang disimpan.
func (s *akunService) penempatan(ctx context.Context, actor entities.User, role, sekolahID, sudinID string) (*uuid.UUID, *uuid.UUID, error) {
	if !bolehKelolaPeran(actor, role) {
		return nil, nil, utils.ErrForbidden("Anda tidak boleh mengelola akun dengan peran " + role)
	}
	switch role {
	case constants.ENUM_ROLE_OPERATOR_SEKOLAH:
		if sekolahID == "" {
			return nil, nil, utils.ErrBadRequest("sekolah_id wajib untuk operator sekolah")
		}
		id := uuid.MustParse(sekolahID)
		ok, err := akses.BolehSekolah(s.db.WithContext(ctx), actor, id)
		if err != nil {
			return nil, nil, err
		}
		if !ok {
			return nil, nil, utils.ErrBadRequest("sekolah tidak ditemukan di wilayah Anda")
		}
		return &id, nil, nil
	case constants.ENUM_ROLE_STAF_SUDIN, constants.ENUM_ROLE_PIMPINAN:
		if actor.Role != constants.ENUM_ROLE_ADMIN {
			return nil, actor.SudinID, nil // staf Sudin hanya bisa menempatkan di Sudin-nya
		}
		if sudinID == "" {
			return nil, nil, utils.ErrBadRequest("sudin_id wajib untuk staf Sudin dan pimpinan")
		}
		id := uuid.MustParse(sudinID)
		var n int64
		if err := s.db.WithContext(ctx).Model(&entities.Sudin{}).Where("id = ?", id).Count(&n).Error; err != nil {
			return nil, nil, err
		}
		if n == 0 {
			return nil, nil, utils.ErrBadRequest("Sudin tidak ditemukan")
		}
		return nil, &id, nil
	default: // admin
		return nil, nil, nil
	}
}

var polaUsername = regexp.MustCompile(`^[a-z0-9._]{3,50}$`)

// identitas menormalkan username dan NIP; username kosong/NIP kosong menjadi nil.
func (s *akunService) identitas(ctx context.Context, username, nip string, kecuali uuid.UUID) (*string, *string, error) {
	var u, n *string
	if username = strings.ToLower(strings.TrimSpace(username)); username != "" {
		if !polaUsername.MatchString(username) {
			return nil, nil, utils.ErrBadRequest("username hanya boleh huruf kecil, angka, titik, dan garis bawah (3-50 karakter)")
		}
		dipakai, err := s.repo.UsernameDipakai(ctx, username, kecuali)
		if err != nil {
			return nil, nil, err
		}
		if dipakai {
			return nil, nil, utils.ErrConflict("username sudah dipakai akun lain")
		}
		u = &username
	}
	if nip != "" {
		n = &nip
	}
	return u, n, nil
}

func (s *akunService) cekEmail(ctx context.Context, email string, kecuali uuid.UUID) error {
	dipakai, err := s.repo.EmailDipakai(ctx, email, kecuali)
	if err != nil {
		return err
	}
	if dipakai {
		return utils.ErrConflict("email sudah dipakai akun lain")
	}
	return nil
}

func (s *akunService) Daftar(ctx context.Context, actor entities.User, f dto.Filter) ([]userDto.UserResponse, int64, error) {
	items, total, err := s.repo.List(ctx, actor, f)
	if err != nil {
		return nil, 0, err
	}
	out := make([]userDto.UserResponse, 0, len(items))
	for _, u := range items {
		out = append(out, userDto.FromEntity(u))
	}
	return out, total, nil
}

func (s *akunService) Detail(ctx context.Context, actor entities.User, id string) (userDto.UserResponse, error) {
	u, err := s.repo.GetByID(ctx, actor, id)
	if err != nil {
		return userDto.UserResponse{}, err
	}
	return userDto.FromEntity(u), nil
}

func (s *akunService) Tambah(ctx context.Context, actor entities.User, req dto.TambahRequest) (userDto.UserResponse, error) {
	sekolahID, sudinID, err := s.penempatan(ctx, actor, req.Role, req.SekolahID, req.SudinID)
	if err != nil {
		return userDto.UserResponse{}, err
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if err := s.cekEmail(ctx, email, uuid.Nil); err != nil {
		return userDto.UserResponse{}, err
	}
	username, nip, err := s.identitas(ctx, req.Username, req.NIP, uuid.Nil)
	if err != nil {
		return userDto.UserResponse{}, err
	}
	u := entities.User{
		Name: strings.TrimSpace(req.Name), Email: email, TelpNumber: req.TelpNumber, Username: username, NIP: nip,
		Password: req.Password, // di-hash oleh hook BeforeCreate
		Role:     req.Role, SekolahID: sekolahID, SudinID: sudinID, IsVerified: true, IsAktif: true,
	}
	if err := s.repo.Create(ctx, &u); err != nil {
		return userDto.UserResponse{}, err
	}
	if err := aktivitas.Catat(s.db.WithContext(ctx), actor.ID, "akun.tambah", "akun", u.ID, email+" ("+u.Role+")"); err != nil {
		return userDto.UserResponse{}, err
	}
	return s.Detail(ctx, actor, u.ID.String())
}

func (s *akunService) Ubah(ctx context.Context, actor entities.User, id string, req dto.UbahRequest) (userDto.UserResponse, error) {
	u, err := s.repo.GetByID(ctx, actor, id)
	if err != nil {
		return userDto.UserResponse{}, err
	}
	if !bolehKelolaPeran(actor, u.Role) {
		return userDto.UserResponse{}, utils.ErrForbidden("Anda tidak boleh mengubah akun dengan peran " + u.Role)
	}
	if u.ID == actor.ID && (req.Role != u.Role || !*req.IsAktif) {
		return userDto.UserResponse{}, utils.ErrBadRequest("tidak bisa mengubah peran atau menonaktifkan akun sendiri")
	}
	sekolahID, sudinID, err := s.penempatan(ctx, actor, req.Role, req.SekolahID, req.SudinID)
	if err != nil {
		return userDto.UserResponse{}, err
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if err := s.cekEmail(ctx, email, u.ID); err != nil {
		return userDto.UserResponse{}, err
	}
	username, nip, err := s.identitas(ctx, req.Username, req.NIP, u.ID)
	if err != nil {
		return userDto.UserResponse{}, err
	}

	err = s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		u.Name, u.Email, u.TelpNumber = strings.TrimSpace(req.Name), email, req.TelpNumber
		u.Username, u.NIP = username, nip
		u.Role, u.SekolahID, u.SudinID, u.IsAktif = req.Role, sekolahID, sudinID, *req.IsAktif
		u.Sudin, u.Sekolah = nil, nil
		if err := s.repo.Save(ctx, tx, &u); err != nil {
			return err
		}
		if !u.IsAktif {
			if err := s.repo.CabutSesi(ctx, tx, u.ID); err != nil {
				return err
			}
		}
		return aktivitas.Catat(tx, actor.ID, "akun.ubah", "akun", u.ID, email)
	})
	if err != nil {
		return userDto.UserResponse{}, err
	}
	return s.Detail(ctx, actor, id)
}

func (s *akunService) ResetPassword(ctx context.Context, actor entities.User, id string, req dto.ResetPasswordRequest) error {
	u, err := s.repo.GetByID(ctx, actor, id)
	if err != nil {
		return err
	}
	if !bolehKelolaPeran(actor, u.Role) {
		return utils.ErrForbidden("Anda tidak boleh mengubah akun dengan peran " + u.Role)
	}
	hash, err := helpers.HashPassword(req.PasswordBaru)
	if err != nil {
		return err
	}
	return s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		if err := s.repo.UpdatePassword(ctx, tx, u.ID, hash); err != nil {
			return err
		}
		// Sesi lama dicabut supaya password lama tidak bisa dipakai lewat refresh token.
		if err := s.repo.CabutSesi(ctx, tx, u.ID); err != nil {
			return err
		}
		return aktivitas.Catat(tx, actor.ID, "akun.reset_password", "akun", u.ID, u.Email)
	})
}

func (s *akunService) Nonaktifkan(ctx context.Context, actor entities.User, id string) error {
	u, err := s.repo.GetByID(ctx, actor, id)
	if err != nil {
		return err
	}
	if u.ID == actor.ID {
		return utils.ErrBadRequest("tidak bisa menonaktifkan akun sendiri")
	}
	if !bolehKelolaPeran(actor, u.Role) {
		return utils.ErrForbidden("Anda tidak boleh mengubah akun dengan peran " + u.Role)
	}
	return s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		if err := tx.Model(&entities.User{}).Where("id = ?", u.ID).Update("is_aktif", false).Error; err != nil {
			return err
		}
		if err := s.repo.CabutSesi(ctx, tx, u.ID); err != nil {
			return err
		}
		return aktivitas.Catat(tx, actor.ID, "akun.nonaktifkan", "akun", u.ID, u.Email)
	})
}
