package sekolah

import (
	"github.com/Caknoooo/go-gin-clean-starter/middlewares"
	"github.com/Caknoooo/go-gin-clean-starter/modules/auth/service"
	"github.com/Caknoooo/go-gin-clean-starter/modules/sekolah/controller"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/constants"
	"github.com/gin-gonic/gin"
	"github.com/samber/do"
	"gorm.io/gorm"
)

func RegisterRoutes(server *gin.Engine, injector *do.Injector) {
	c := do.MustInvoke[controller.SekolahController](injector)
	jwt := do.MustInvokeNamed[service.JWTService](injector, constants.JWTService)
	db := do.MustInvokeNamed[*gorm.DB](injector, constants.DB)

	r := server.Group("/api/sekolah", middlewares.Authenticate(jwt), middlewares.Authorize(db))
	r.GET("", c.Daftar)
	r.GET("/:id", c.Detail)
}
