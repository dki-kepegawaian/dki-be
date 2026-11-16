// Package penerima menentukan siapa yang menerima notifikasi atau tugas berdasarkan
// penempatan akun saat ini (bukan identitas yang tersimpan di record lama).
// Semua fungsi hanya mengembalikan akun aktif.
package penerima

import (
	"context"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/constants"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func ids(ctx context.Context, tx *gorm.DB, where string, args ...any) ([]uuid.UUID, error) {
	var out []uuid.UUID
	err := tx.WithContext(ctx).Model(&entities.User{}).Where("is_aktif = true").Where(where, args...).Pluck("id", &out).Error
	return out, err
}

// OperatorSekolah: operator aktif yang saat ini ditempatkan di sekolah tersebut.
func OperatorSekolah(ctx context.Context, tx *gorm.DB, sekolahID uuid.UUID) ([]uuid.UUID, error) {
	return ids(ctx, tx, "role = ? AND sekolah_id = ?", constants.ENUM_ROLE_OPERATOR_SEKOLAH, sekolahID)
}

// StafSudinSekolah: staf Sudin aktif yang membawahi sekolah tersebut.
func StafSudinSekolah(ctx context.Context, tx *gorm.DB, sekolahID uuid.UUID) ([]uuid.UUID, error) {
	return ids(ctx, tx, "role = ? AND sudin_id = (SELECT sudin_id FROM sekolah WHERE id = ?)", constants.ENUM_ROLE_STAF_SUDIN, sekolahID)
}

// PimpinanSekolah: pimpinan aktif Sudin yang membawahi sekolah tersebut.
func PimpinanSekolah(ctx context.Context, tx *gorm.DB, sekolahID uuid.UUID) ([]uuid.UUID, error) {
	return ids(ctx, tx, "role = ? AND sudin_id = (SELECT sudin_id FROM sekolah WHERE id = ?)", constants.ENUM_ROLE_PIMPINAN, sekolahID)
}

// StafSudin: semua staf Sudin aktif di satu Sudin.
func StafSudin(ctx context.Context, tx *gorm.DB, sudinID uuid.UUID) ([]uuid.UUID, error) {
	return ids(ctx, tx, "role = ? AND sudin_id = ?", constants.ENUM_ROLE_STAF_SUDIN, sudinID)
}

// StafSudinValid: true bila userID adalah staf Sudin aktif di sudinID.
func StafSudinValid(ctx context.Context, tx *gorm.DB, userID, sudinID uuid.UUID) (bool, error) {
	got, err := ids(ctx, tx, "id = ? AND role = ? AND sudin_id = ?", userID, constants.ENUM_ROLE_STAF_SUDIN, sudinID)
	return len(got) > 0, err
}
