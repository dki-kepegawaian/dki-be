package middlewares

import (
	"net/http"
	"slices"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/constants"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const actorKey = "actor"

// Authorize dipasang setelah Authenticate. Ia memuat akun dari DB (supaya perubahan
// peran atau penempatan langsung berlaku) dan menolak peran di luar daftar.
// Tanpa daftar peran, keempat peran domain diizinkan (akun "user" bawaan starter tidak).
func Authorize(db *gorm.DB, roles ...string) gin.HandlerFunc {
	if len(roles) == 0 {
		roles = []string{
			constants.ENUM_ROLE_ADMIN, constants.ENUM_ROLE_STAF_SUDIN,
			constants.ENUM_ROLE_OPERATOR_SEKOLAH, constants.ENUM_ROLE_PIMPINAN,
		}
	}
	return func(ctx *gin.Context) {
		var user entities.User
		if err := db.WithContext(ctx.Request.Context()).Take(&user, "id = ?", ctx.GetString("user_id")).Error; err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, utils.BuildResponseFailed("akses ditolak", "akun tidak ditemukan", nil))
			return
		}
		if !user.IsAktif {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, utils.BuildResponseFailed("akses ditolak", "akun dinonaktifkan, hubungi Sudin", nil))
			return
		}
		if !slices.Contains(roles, user.Role) {
			ctx.AbortWithStatusJSON(http.StatusForbidden, utils.BuildResponseFailed("akses ditolak", "peran Anda tidak diizinkan untuk aksi ini", nil))
			return
		}
		ctx.Set(actorKey, user)
		ctx.Next()
	}
}

// Actor mengembalikan akun yang dimuat oleh Authorize.
func Actor(ctx *gin.Context) entities.User {
	return ctx.MustGet(actorKey).(entities.User)
}
