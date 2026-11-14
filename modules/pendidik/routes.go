package pendidik

import (
	"github.com/Caknoooo/go-gin-clean-starter/middlewares"
	"github.com/Caknoooo/go-gin-clean-starter/modules/auth/service"
	"github.com/Caknoooo/go-gin-clean-starter/modules/pendidik/controller"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/constants"
	"github.com/gin-gonic/gin"
	"github.com/samber/do"
	"gorm.io/gorm"
)

func RegisterRoutes(server *gin.Engine, injector *do.Injector) {
	c := do.MustInvoke[controller.PendidikController](injector)
	jwt := do.MustInvokeNamed[service.JWTService](injector, constants.JWTService)
	db := do.MustInvokeNamed[*gorm.DB](injector, constants.DB)

	semua := middlewares.Authorize(db)
	pengelola := middlewares.Authorize(db, constants.ENUM_ROLE_STAF_SUDIN, constants.ENUM_ROLE_ADMIN)

	r := server.Group("/api/pendidik", middlewares.Authenticate(jwt))
	r.GET("", semua, c.Daftar)
	r.GET("/:id", semua, c.Detail)
	r.POST("", pengelola, c.Tambah)
	r.PUT("/:id", pengelola, c.Ubah)
}
