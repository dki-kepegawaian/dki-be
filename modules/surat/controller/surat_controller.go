package controller

import (
	"mime"
	"net/http"
	"strings"

	"github.com/Caknoooo/go-gin-clean-starter/middlewares"
	"github.com/Caknoooo/go-gin-clean-starter/modules/surat/dto"
	"github.com/Caknoooo/go-gin-clean-starter/modules/surat/service"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/utils"
	"github.com/Caknoooo/go-pagination"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type SuratController interface {
	Jenis(ctx *gin.Context)
	Daftar(ctx *gin.Context)
	Detail(ctx *gin.Context)
	Pratinjau(ctx *gin.Context)
	Buat(ctx *gin.Context)
	Ubah(ctx *gin.Context)
	BuatDokumen(ctx *gin.Context)
	TetapkanFinal(ctx *gin.Context)
	Batalkan(ctx *gin.Context)
	Serahkan(ctx *gin.Context)
}

type suratController struct {
	service service.SuratService
}

func NewSuratController(s service.SuratService) SuratController {
	return &suratController{service: s}
}

func bind(ctx *gin.Context, req any, b binding.Binding) bool {
	if err := ctx.ShouldBindWith(req, b); err != nil {
		utils.RespondError(ctx, "data tidak valid", utils.ErrBadRequest(err.Error()))
		return false
	}
	return true
}

func (c *suratController) Jenis(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("berhasil mengambil jenis surat", c.service.Jenis()))
}

func (c *suratController) Daftar(ctx *gin.Context) {
	p := pagination.BindPagination(ctx)
	var status []string
	if s := ctx.Query("status"); s != "" {
		status = strings.Split(s, ",")
	}
	items, total, err := c.service.Daftar(ctx.Request.Context(), middlewares.Actor(ctx), dto.Filter{
		Status:     status,
		Jenis:      ctx.Query("jenis"),
		PendidikID: ctx.Query("pendidik_id"),
		Search:     p.Search,
		Offset:     p.GetOffset(),
		Limit:      p.GetLimit(),
	})
	if err != nil {
		utils.RespondError(ctx, "gagal mengambil daftar surat", err)
		return
	}
	ctx.JSON(http.StatusOK, pagination.NewPaginatedResponse(http.StatusOK, "berhasil mengambil daftar surat", items, pagination.CalculatePagination(p, total)))
}

func (c *suratController) Detail(ctx *gin.Context) {
	res, err := c.service.Detail(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"))
	if err != nil {
		utils.RespondError(ctx, "gagal mengambil surat", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("berhasil mengambil surat", res))
}

func (c *suratController) Pratinjau(ctx *gin.Context) {
	nama, data, err := c.service.Pratinjau(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"))
	if err != nil {
		utils.RespondError(ctx, "gagal membuat pratinjau surat", err)
		return
	}
	ctx.Header("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": nama}))
	ctx.Header("Cache-Control", "no-store")
	ctx.Data(http.StatusOK, "application/pdf", data)
}

func (c *suratController) Buat(ctx *gin.Context) {
	var req dto.BuatRequest
	if !bind(ctx, &req, binding.JSON) {
		return
	}
	res, err := c.service.Buat(ctx.Request.Context(), middlewares.Actor(ctx), req)
	if err != nil {
		utils.RespondError(ctx, "gagal menyimpan draf surat", err)
		return
	}
	ctx.JSON(http.StatusCreated, utils.BuildResponseSuccess("draf surat tersimpan", res))
}

func (c *suratController) Ubah(ctx *gin.Context) {
	var req dto.UbahRequest
	if !bind(ctx, &req, binding.JSON) {
		return
	}
	res, err := c.service.Ubah(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"), req)
	if err != nil {
		utils.RespondError(ctx, "gagal mengubah surat", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("isian surat tersimpan", res))
}

func (c *suratController) BuatDokumen(ctx *gin.Context) {
	res, err := c.service.BuatDokumen(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"))
	if err != nil {
		utils.RespondError(ctx, "gagal membuat dokumen surat", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("dokumen surat dibuat, menunggu tanda tangan", res))
}

func (c *suratController) TetapkanFinal(ctx *gin.Context) {
	var req dto.FinalRequest
	if !bind(ctx, &req, binding.FormMultipart) {
		return
	}
	res, err := c.service.TetapkanFinal(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"), req)
	if err != nil {
		utils.RespondError(ctx, "gagal menetapkan surat final", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("surat ditetapkan final", res))
}

func (c *suratController) Batalkan(ctx *gin.Context) {
	var req dto.BatalRequest
	if !bind(ctx, &req, binding.JSON) {
		return
	}
	res, err := c.service.Batalkan(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"), req)
	if err != nil {
		utils.RespondError(ctx, "gagal membatalkan surat", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("surat dibatalkan", res))
}

func (c *suratController) Serahkan(ctx *gin.Context) {
	res, err := c.service.Serahkan(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"))
	if err != nil {
		utils.RespondError(ctx, "gagal menandai surat diserahkan", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("surat ditandai sudah diserahkan", res))
}
