package cuti

import (
	"github.com/Caknoooo/go-gin-clean-starter/middlewares"
	"github.com/Caknoooo/go-gin-clean-starter/modules/auth/service"
	"github.com/Caknoooo/go-gin-clean-starter/modules/cuti/controller"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/constants"
	"github.com/gin-gonic/gin"
	"github.com/samber/do"
	"gorm.io/gorm"
)

func RegisterRoutes(server *gin.Engine, injector *do.Injector) {
	c := do.MustInvoke[controller.CutiController](injector)
	jwt := do.MustInvokeNamed[service.JWTService](injector, constants.JWTService)
	db := do.MustInvokeNamed[*gorm.DB](injector, constants.DB)

	semua := middlewares.Authorize(db) // scope sekolah/Sudin diterapkan di query
	operator := middlewares.Authorize(db, constants.ENUM_ROLE_OPERATOR_SEKOLAH)
	staf := middlewares.Authorize(db, constants.ENUM_ROLE_STAF_SUDIN)

	r := server.Group("/api/cuti", middlewares.Authenticate(jwt))
	r.GET("", semua, c.Daftar)
	r.GET("/ringkasan", semua, c.Ringkasan)
	r.GET("/sisa-tahunan", semua, c.SisaTahunan)
	r.GET("/:id", semua, c.Detail)

	r.POST("", operator, c.Ajukan)
	r.POST("/:id/perbaikan", operator, c.Perbaiki)

	r.POST("/:id/periksa", staf, c.Periksa)
	r.POST("/terima-surat-fisik", staf, c.TerimaSuratFisik)
	r.POST("/:id/keputusan", staf, c.Putuskan)
}
