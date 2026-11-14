// Package akses berisi aturan siapa boleh melihat data sekolah mana.
//
//   - admin: semua sekolah
//   - staf_sudin, pimpinan: sekolah di Sudin-nya
//   - operator_sekolah: sekolahnya sendiri
package akses

import (
	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/constants"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Scope membatasi query ke sekolah yang boleh dilihat actor.
// column adalah kolom sekolah_id pada tabel yang di-query (mis. "cuti.sekolah_id").
func Scope(actor entities.User, column string) func(*gorm.DB) *gorm.DB {
	return func(q *gorm.DB) *gorm.DB {
		switch actor.Role {
		case constants.ENUM_ROLE_ADMIN:
			return q
		case constants.ENUM_ROLE_STAF_SUDIN, constants.ENUM_ROLE_PIMPINAN:
			if actor.SudinID == nil {
				return q.Where("1 = 0")
			}
			return q.Where(column+" IN (?)", q.Session(&gorm.Session{NewDB: true}).
				Model(&entities.Sekolah{}).Select("id").Where("sudin_id = ?", *actor.SudinID))
		case constants.ENUM_ROLE_OPERATOR_SEKOLAH:
			if actor.SekolahID == nil {
				return q.Where("1 = 0")
			}
			return q.Where(column+" = ?", *actor.SekolahID)
		default:
			return q.Where("1 = 0")
		}
	}
}

// BolehSekolah mengecek apakah actor boleh mengakses data milik satu sekolah.
func BolehSekolah(db *gorm.DB, actor entities.User, sekolahID uuid.UUID) (bool, error) {
	var n int64
	err := db.Model(&entities.Sekolah{}).Scopes(Scope(actor, "sekolah.id")).Where("sekolah.id = ?", sekolahID).Count(&n).Error
	return n > 0, err
}
