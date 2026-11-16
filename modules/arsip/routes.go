package arsip

import (
	"github.com/Caknoooo/go-gin-clean-starter/middlewares"
	"github.com/Caknoooo/go-gin-clean-starter/modules/arsip/controller"
	"github.com/Caknoooo/go-gin-clean-starter/modules/auth/service"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/constants"
	"github.com/gin-gonic/gin"
	"github.com/samber/do"
	"gorm.io/gorm"
)

// Arsip kepegawaian guru (R1–R4, H4). Semua peran membaca dalam scope-nya;
// operator, staf Sudin, dan admin bisa mengarsipkan; log hanya untuk Sudin, pimpinan, admin.
func RegisterRoutes(server *gin.Engine, injector *do.Injector) {
	c := do.MustInvoke[controller.ArsipController](injector)
	jwt := do.MustInvokeNamed[service.JWTService](injector, constants.JWTService)
	db := do.MustInvokeNamed[*gorm.DB](injector, constants.DB)

	semua := middlewares.Authorize(db)
	unggah := middlewares.Authorize(db, constants.ENUM_ROLE_STAF_SUDIN, constants.ENUM_ROLE_ADMIN, constants.ENUM_ROLE_OPERATOR_SEKOLAH)
	log := middlewares.Authorize(db, constants.ENUM_ROLE_STAF_SUDIN, constants.ENUM_ROLE_ADMIN, constants.ENUM_ROLE_PIMPINAN)

	r := server.Group("/api/arsip", middlewares.Authenticate(jwt))
	r.GET("", semua, c.Daftar)
	r.GET("/ringkasan", semua, c.Ringkasan)
	r.GET("/log", log, c.Log)
	r.GET("/log/ekspor", log, c.LogCSV)
	r.GET("/:id", semua, c.Detail)
	r.GET("/:id/unduh", semua, c.Unduh)
	r.POST("", unggah, c.Unggah)
	r.POST("/:id/versi", unggah, c.VersiBaru)
}
