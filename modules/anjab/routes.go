package anjab

import (
	"github.com/Caknoooo/go-gin-clean-starter/middlewares"
	"github.com/Caknoooo/go-gin-clean-starter/modules/anjab/controller"
	"github.com/Caknoooo/go-gin-clean-starter/modules/auth/service"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/constants"
	"github.com/gin-gonic/gin"
	"github.com/samber/do"
	"gorm.io/gorm"
)

func RegisterRoutes(server *gin.Engine, injector *do.Injector) {
	c := do.MustInvoke[controller.AnjabController](injector)
	jwt := do.MustInvokeNamed[service.JWTService](injector, constants.JWTService)
	db := do.MustInvokeNamed[*gorm.DB](injector, constants.DB)

	semua := middlewares.Authorize(db) // scope Sudin/sekolah diterapkan di query
	operator := middlewares.Authorize(db, constants.ENUM_ROLE_OPERATOR_SEKOLAH)
	staf := middlewares.Authorize(db, constants.ENUM_ROLE_STAF_SUDIN)
	pemantau := middlewares.Authorize(db, constants.ENUM_ROLE_STAF_SUDIN, constants.ENUM_ROLE_PIMPINAN, constants.ENUM_ROLE_ADMIN)

	r := server.Group("/api/anjab", middlewares.Authenticate(jwt))

	// Periode & parameter (A1–A3, A6)
	r.GET("/periode", semua, c.DaftarPeriode)
	r.GET("/periode/:id", semua, c.DetailPeriode)
	r.POST("/periode", staf, c.BuatPeriode)
	r.PUT("/periode/:id", staf, c.UbahPeriode)
	r.GET("/periode/:id/templat", staf, c.Templat)
	r.POST("/periode/:id/impor", staf, c.Impor)
	r.POST("/periode/:id/buka", staf, c.BukaPembaruan)
	r.PUT("/periode/:id/tenggat", staf, c.PerpanjangTenggat)
	r.POST("/periode/:id/pengingat", staf, c.KirimPengingat)
	r.GET("/periode/:id/monitoring", pemantau, c.Monitoring)
	r.GET("/periode/:id/monitoring/excel", pemantau, c.MonitoringExcel)
	r.GET("/periode/:id/monitoring/pdf", pemantau, c.LaporanPDF)

	// Data Anjab per sekolah (A3–A5)
	r.GET("/sekolah", semua, c.Daftar)
	r.GET("/sekolah/:id", semua, c.Detail)
	r.PUT("/sekolah/:id/rombel", operator, c.UbahRombel)
	r.POST("/sekolah/:id/guru", operator, c.TambahGuru)
	r.PUT("/sekolah/:id/guru/:guruId", operator, c.UbahGuru)
	r.DELETE("/sekolah/:id/guru/:guruId", operator, c.HapusGuru)
	r.POST("/sekolah/:id/kirim", operator, c.Kirim)
	r.POST("/sekolah/:id/tinjau", staf, c.Tinjau)
}
