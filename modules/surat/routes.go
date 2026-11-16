package surat

import (
	"github.com/Caknoooo/go-gin-clean-starter/middlewares"
	"github.com/Caknoooo/go-gin-clean-starter/modules/auth/service"
	"github.com/Caknoooo/go-gin-clean-starter/modules/surat/controller"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/constants"
	"github.com/gin-gonic/gin"
	"github.com/samber/do"
	"gorm.io/gorm"
)

func RegisterRoutes(server *gin.Engine, injector *do.Injector) {
	c := do.MustInvoke[controller.SuratController](injector)
	jwt := do.MustInvokeNamed[service.JWTService](injector, constants.JWTService)
	db := do.MustInvokeNamed[*gorm.DB](injector, constants.DB)

	// Generator Surat milik staf Sudin (admin juga boleh); pimpinan baca saja; operator sekolah tidak punya akses.
	baca := middlewares.Authorize(db, constants.ENUM_ROLE_STAF_SUDIN, constants.ENUM_ROLE_ADMIN, constants.ENUM_ROLE_PIMPINAN)
	kelola := middlewares.Authorize(db, constants.ENUM_ROLE_STAF_SUDIN, constants.ENUM_ROLE_ADMIN)

	r := server.Group("/api/surat", middlewares.Authenticate(jwt))
	r.GET("/jenis", baca, c.Jenis)
	r.GET("", baca, c.Daftar)
	r.GET("/:id", baca, c.Detail)
	r.GET("/:id/pratinjau", baca, c.Pratinjau)

	r.POST("", kelola, c.Buat)
	r.PUT("/:id", kelola, c.Ubah)
	r.POST("/:id/buat-dokumen", kelola, c.BuatDokumen)
	r.POST("/:id/final", kelola, c.TetapkanFinal)
	r.POST("/:id/batal", kelola, c.Batalkan)
	r.POST("/:id/serahkan", kelola, c.Serahkan)
}
