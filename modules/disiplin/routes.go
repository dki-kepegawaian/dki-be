package disiplin

import (
	"github.com/Caknoooo/go-gin-clean-starter/middlewares"
	"github.com/Caknoooo/go-gin-clean-starter/modules/auth/service"
	"github.com/Caknoooo/go-gin-clean-starter/modules/disiplin/controller"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/constants"
	"github.com/gin-gonic/gin"
	"github.com/samber/do"
	"gorm.io/gorm"
)

func RegisterRoutes(server *gin.Engine, injector *do.Injector) {
	c := do.MustInvoke[controller.DisiplinController](injector)
	jwt := do.MustInvokeNamed[service.JWTService](injector, constants.JWTService)
	db := do.MustInvokeNamed[*gorm.DB](injector, constants.DB)

	semua := middlewares.Authorize(db) // scope sekolah/Sudin + bagian internal diatur di service
	operator := middlewares.Authorize(db, constants.ENUM_ROLE_OPERATOR_SEKOLAH)
	staf := middlewares.Authorize(db, constants.ENUM_ROLE_STAF_SUDIN)
	pimpinan := middlewares.Authorize(db, constants.ENUM_ROLE_PIMPINAN)

	r := server.Group("/api/disiplin", middlewares.Authenticate(jwt))
	r.GET("", semua, c.Daftar)
	r.GET("/ringkasan", semua, c.Ringkasan)
	r.GET("/:id", semua, c.Detail)
	r.GET("/:id/dokumen/:dokumen_id", semua, c.UnduhBerkas)

	r.POST("", operator, c.Lapor)
	r.POST("/:id/kelengkapan", operator, c.Lengkapi)
	r.POST("/:id/laporan-pelaksanaan", operator, c.LaporanPelaksanaan)

	r.POST("/:id/verifikasi", staf, c.Verifikasi)
	r.POST("/:id/pemeriksaan", staf, c.Pemeriksaan)
	r.POST("/:id/rekomendasi", staf, c.Rekomendasi)
	r.POST("/:id/catat-keputusan", staf, c.CatatKeputusan)
	r.POST("/:id/tindak-lanjut", staf, c.TindakLanjut)
	r.POST("/:id/tutup", staf, c.Tutup)

	// DS6: satu-satunya langkah milik pimpinan.
	r.POST("/:id/keputusan", pimpinan, c.Putuskan)
}
