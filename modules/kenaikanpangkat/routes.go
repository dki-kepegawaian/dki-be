package kenaikanpangkat

import (
	"github.com/Caknoooo/go-gin-clean-starter/middlewares"
	"github.com/Caknoooo/go-gin-clean-starter/modules/auth/service"
	"github.com/Caknoooo/go-gin-clean-starter/modules/kenaikanpangkat/controller"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/constants"
	"github.com/gin-gonic/gin"
	"github.com/samber/do"
	"gorm.io/gorm"
)

func RegisterRoutes(server *gin.Engine, injector *do.Injector) {
	c := do.MustInvoke[controller.KenaikanPangkatController](injector)
	jwt := do.MustInvokeNamed[service.JWTService](injector, constants.JWTService)
	db := do.MustInvokeNamed[*gorm.DB](injector, constants.DB)

	semua := middlewares.Authorize(db) // scope sekolah/Sudin diterapkan di query
	operator := middlewares.Authorize(db, constants.ENUM_ROLE_OPERATOR_SEKOLAH)
	staf := middlewares.Authorize(db, constants.ENUM_ROLE_STAF_SUDIN)

	r := server.Group("/api/kenaikan-pangkat", middlewares.Authenticate(jwt))
	r.GET("", semua, c.Daftar)
	r.GET("/ringkasan", semua, c.Ringkasan)
	r.GET("/persyaratan", semua, c.Persyaratan)
	r.GET("/perkiraan", semua, c.Perkiraan)
	r.GET("/memenuhi-syarat", semua, c.MemenuhiSyarat)
	r.GET("/:id", semua, c.Detail)

	r.POST("/berkas", operator, c.UnggahBerkas)
	r.POST("", operator, c.Ajukan)
	r.POST("/:id/perbaikan", operator, c.Perbaiki)

	r.POST("/:id/verifikasi", staf, c.Verifikasi)
	r.POST("/kirim-bkd", staf, c.KirimBKD)
	r.POST("/:id/sk", staf, c.UnggahSK)
}
