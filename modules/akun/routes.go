package akun

import (
	"github.com/Caknoooo/go-gin-clean-starter/middlewares"
	"github.com/Caknoooo/go-gin-clean-starter/modules/akun/controller"
	"github.com/Caknoooo/go-gin-clean-starter/modules/auth/service"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/constants"
	"github.com/gin-gonic/gin"
	"github.com/samber/do"
	"gorm.io/gorm"
)

// Manajemen akun oleh Sudin (dan admin). Tidak ada pendaftaran mandiri.
func RegisterRoutes(server *gin.Engine, injector *do.Injector) {
	c := do.MustInvoke[controller.AkunController](injector)
	jwt := do.MustInvokeNamed[service.JWTService](injector, constants.JWTService)
	db := do.MustInvokeNamed[*gorm.DB](injector, constants.DB)

	r := server.Group("/api/akun",
		middlewares.Authenticate(jwt),
		middlewares.Authorize(db, constants.ENUM_ROLE_STAF_SUDIN, constants.ENUM_ROLE_ADMIN))
	r.GET("", c.Daftar)
	r.GET("/:id", c.Detail)
	r.POST("", c.Tambah)
	r.PUT("/:id", c.Ubah)
	r.PUT("/:id/password", c.ResetPassword)
	r.DELETE("/:id", c.Nonaktifkan)
}
