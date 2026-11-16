package controller

import (
	"net/http"

	"github.com/Caknoooo/go-gin-clean-starter/middlewares"
	"github.com/Caknoooo/go-gin-clean-starter/modules/sekolah/repository"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/utils"
	"github.com/Caknoooo/go-pagination"
	"github.com/gin-gonic/gin"
)

// Modul sekolah tipis (baca data dan ubah profil), jadi controller langsung memakai repository.
type SekolahController interface {
	Daftar(ctx *gin.Context)
	Detail(ctx *gin.Context)
	Profil(ctx *gin.Context)
	Ubah(ctx *gin.Context)
	UbahKontakSudin(ctx *gin.Context)
}

type sekolahController struct {
	repo repository.SekolahRepository
}

func NewSekolahController(repo repository.SekolahRepository) SekolahController {
	return &sekolahController{repo: repo}
}

func (c *sekolahController) Daftar(ctx *gin.Context) {
	p := pagination.BindPagination(ctx)
	items, total, err := c.repo.List(ctx.Request.Context(), middlewares.Actor(ctx), repository.Filter{
		Search:  p.Search,
		Jenjang: ctx.Query("jenjang"),
		Offset:  p.GetOffset(),
		Limit:   p.GetLimit(),
	})
	if err != nil {
		utils.RespondError(ctx, "gagal mengambil daftar sekolah", err)
		return
	}
	ctx.JSON(http.StatusOK, pagination.NewPaginatedResponse(http.StatusOK, "berhasil mengambil daftar sekolah", items, pagination.CalculatePagination(p, total)))
}

func (c *sekolahController) Detail(ctx *gin.Context) {
	item, err := c.repo.GetByID(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"))
	if err != nil {
		utils.RespondError(ctx, "gagal mengambil sekolah", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("berhasil mengambil sekolah", item))
}

func (c *sekolahController) Profil(ctx *gin.Context) {
	item, err := c.repo.Profil(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"))
	if err != nil {
		utils.RespondError(ctx, "gagal mengambil profil sekolah", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("berhasil mengambil profil sekolah", item))
}

func (c *sekolahController) Ubah(ctx *gin.Context) {
	var req repository.UbahData
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.RespondError(ctx, "data sekolah tidak valid", utils.ErrBadRequest(err.Error()))
		return
	}
	item, err := c.repo.Ubah(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"), req)
	if err != nil {
		utils.RespondError(ctx, "gagal mengubah sekolah", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("berhasil mengubah sekolah", item))
}

func (c *sekolahController) UbahKontakSudin(ctx *gin.Context) {
	var req repository.KontakData
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.RespondError(ctx, "data kontak tidak valid", utils.ErrBadRequest(err.Error()))
		return
	}
	item, err := c.repo.UbahKontakSudin(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"), req)
	if err != nil {
		utils.RespondError(ctx, "gagal mengubah kontak Sudin", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("berhasil mengubah kontak Sudin", item))
}
