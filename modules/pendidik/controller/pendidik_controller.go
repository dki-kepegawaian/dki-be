package controller

import (
	"net/http"

	"github.com/Caknoooo/go-gin-clean-starter/middlewares"
	"github.com/Caknoooo/go-gin-clean-starter/modules/pendidik/dto"
	"github.com/Caknoooo/go-gin-clean-starter/modules/pendidik/repository"
	"github.com/Caknoooo/go-gin-clean-starter/modules/pendidik/service"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/utils"
	"github.com/Caknoooo/go-pagination"
	"github.com/gin-gonic/gin"
)

type PendidikController interface {
	Daftar(ctx *gin.Context)
	Detail(ctx *gin.Context)
	Tambah(ctx *gin.Context)
	Ubah(ctx *gin.Context)
}

type pendidikController struct {
	service service.PendidikService
}

func NewPendidikController(s service.PendidikService) PendidikController {
	return &pendidikController{service: s}
}

func (c *pendidikController) Daftar(ctx *gin.Context) {
	p := pagination.BindPagination(ctx)
	items, total, err := c.service.Daftar(ctx.Request.Context(), middlewares.Actor(ctx), repository.Filter{
		Search:            p.Search,
		SekolahID:         ctx.Query("sekolah_id"),
		StatusKepegawaian: ctx.Query("status_kepegawaian"),
		Offset:            p.GetOffset(),
		Limit:             p.GetLimit(),
	})
	if err != nil {
		utils.RespondError(ctx, "gagal mengambil daftar pendidik", err)
		return
	}
	ctx.JSON(http.StatusOK, pagination.NewPaginatedResponse(http.StatusOK, "berhasil mengambil daftar pendidik", items, pagination.CalculatePagination(p, total)))
}

func (c *pendidikController) Detail(ctx *gin.Context) {
	item, err := c.service.Detail(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"))
	if err != nil {
		utils.RespondError(ctx, "gagal mengambil pendidik", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("berhasil mengambil pendidik", item))
}

func (c *pendidikController) Tambah(ctx *gin.Context) {
	var req dto.PendidikRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.RespondError(ctx, "data pendidik tidak valid", utils.ErrBadRequest(err.Error()))
		return
	}
	item, err := c.service.Tambah(ctx.Request.Context(), middlewares.Actor(ctx), req)
	if err != nil {
		utils.RespondError(ctx, "gagal menambah pendidik", err)
		return
	}
	ctx.JSON(http.StatusCreated, utils.BuildResponseSuccess("berhasil menambah pendidik", item))
}

func (c *pendidikController) Ubah(ctx *gin.Context) {
	var req dto.PendidikRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.RespondError(ctx, "data pendidik tidak valid", utils.ErrBadRequest(err.Error()))
		return
	}
	item, err := c.service.Ubah(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"), req)
	if err != nil {
		utils.RespondError(ctx, "gagal mengubah pendidik", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("berhasil mengubah pendidik", item))
}
