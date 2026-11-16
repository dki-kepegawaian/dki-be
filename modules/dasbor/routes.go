// Package dasbor menyajikan tampilan lintas modul: Tugas Saya (T1), beranda per peran
// (D02, H1, B1), Daftar Pengajuan (D03), dan Rekap Kepegawaian (B2). Hanya membaca.
package dasbor

import (
	"github.com/Caknoooo/go-gin-clean-starter/middlewares"
	"github.com/Caknoooo/go-gin-clean-starter/modules/auth/service"
	"github.com/Caknoooo/go-gin-clean-starter/modules/dasbor/controller"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/constants"
	"github.com/gin-gonic/gin"
	"github.com/samber/do"
	"gorm.io/gorm"
)

func RegisterRoutes(server *gin.Engine, injector *do.Injector) {
	c := do.MustInvoke[controller.DasborController](injector)
	jwt := do.MustInvokeNamed[service.JWTService](injector, constants.JWTService)
	db := do.MustInvokeNamed[*gorm.DB](injector, constants.DB)

	r := server.Group("/api/dasbor", middlewares.Authenticate(jwt), middlewares.Authorize(db))
	r.GET("/tugas", c.Tugas)
	r.GET("/pengajuan", c.Pengajuan)
	r.GET("/pengajuan/ekspor", c.PengajuanCSV)
	r.GET("/ringkasan", c.Ringkasan)
	r.GET("/ringkasan/pdf", c.RekapPDF)
	r.POST("/pengingat-tenggat", middlewares.Authorize(db, constants.ENUM_ROLE_ADMIN), c.KirimPengingat)
}
