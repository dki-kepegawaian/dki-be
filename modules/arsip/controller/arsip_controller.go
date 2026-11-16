package controller

import (
	"mime"
	"net/http"
	"strconv"

	"github.com/Caknoooo/go-gin-clean-starter/middlewares"
	"github.com/Caknoooo/go-gin-clean-starter/modules/arsip/dto"
	"github.com/Caknoooo/go-gin-clean-starter/modules/arsip/service"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/utils"
	"github.com/Caknoooo/go-pagination"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type ArsipController interface {
	Daftar(ctx *gin.Context)
	Ringkasan(ctx *gin.Context)
	Detail(ctx *gin.Context)
	Unduh(ctx *gin.Context)
	Unggah(ctx *gin.Context)
	VersiBaru(ctx *gin.Context)
	Log(ctx *gin.Context)
	LogCSV(ctx *gin.Context)
}

type arsipController struct {
	service service.ArsipService
}

func NewArsipController(s service.ArsipService) ArsipController {
	return &arsipController{service: s}
}

func (c *arsipController) Daftar(ctx *gin.Context) {
	p := pagination.BindPagination(ctx)
	items, total, err := c.service.Daftar(ctx.Request.Context(), middlewares.Actor(ctx), dto.Filter{
		PendidikID: ctx.Query("pendidik_id"), Kategori: ctx.Query("kategori"), Search: p.Search,
		Offset: p.GetOffset(), Limit: p.GetLimit(),
	})
	if err != nil {
		utils.RespondError(ctx, "gagal mengambil arsip", err)
		return
	}
	ctx.JSON(http.StatusOK, pagination.NewPaginatedResponse(http.StatusOK, "berhasil mengambil arsip", items, pagination.CalculatePagination(p, total)))
}

func (c *arsipController) Ringkasan(ctx *gin.Context) {
	res, err := c.service.Ringkasan(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Query("pendidik_id"))
	if err != nil {
		utils.RespondError(ctx, "gagal mengambil ringkasan arsip", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("berhasil mengambil ringkasan arsip", res))
}

func (c *arsipController) Detail(ctx *gin.Context) {
	res, err := c.service.Detail(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"))
	if err != nil {
		utils.RespondError(ctx, "gagal mengambil arsip", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("berhasil mengambil arsip", res))
}

func (c *arsipController) Unduh(ctx *gin.Context) {
	versi := 0
	if v := ctx.Query("versi"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			utils.RespondError(ctx, "data tidak valid", utils.ErrBadRequest("versi tidak valid"))
			return
		}
		versi = n
	}
	inline := ctx.Query("inline") == "1"
	doc, r, size, err := c.service.Buka(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"), versi, inline)
	if err != nil {
		utils.RespondError(ctx, "gagal mengunduh arsip", err)
		return
	}
	defer r.Close()
	disposition := "attachment"
	if inline {
		disposition = "inline"
	}
	ctx.DataFromReader(http.StatusOK, size, doc.MimeType, r, map[string]string{
		"Content-Disposition":    mime.FormatMediaType(disposition, map[string]string{"filename": doc.NamaAsli}),
		"X-Content-Type-Options": "nosniff",
	})
}

func (c *arsipController) Unggah(ctx *gin.Context) {
	var req dto.UnggahRequest
	if err := ctx.ShouldBindWith(&req, binding.FormMultipart); err != nil {
		utils.RespondError(ctx, "data tidak valid", utils.ErrBadRequest(err.Error()))
		return
	}
	res, err := c.service.Unggah(ctx.Request.Context(), middlewares.Actor(ctx), req)
	if err != nil {
		utils.RespondError(ctx, "gagal mengarsipkan dokumen", err)
		return
	}
	ctx.JSON(http.StatusCreated, utils.BuildResponseSuccess("dokumen diarsipkan", res))
}

func (c *arsipController) VersiBaru(ctx *gin.Context) {
	var req dto.VersiRequest
	if err := ctx.ShouldBindWith(&req, binding.FormMultipart); err != nil {
		utils.RespondError(ctx, "data tidak valid", utils.ErrBadRequest(err.Error()))
		return
	}
	res, err := c.service.VersiBaru(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"), req)
	if err != nil {
		utils.RespondError(ctx, "gagal menyimpan versi baru", err)
		return
	}
	ctx.JSON(http.StatusCreated, utils.BuildResponseSuccess("versi baru tersimpan", res))
}

func logFilter(ctx *gin.Context, p pagination.PaginationRequest) dto.LogFilter {
	return dto.LogFilter{
		Dari: ctx.Query("dari"), Sampai: ctx.Query("sampai"), UserID: ctx.Query("user_id"),
		Aksi: ctx.Query("aksi"), Search: p.Search, Offset: p.GetOffset(), Limit: p.GetLimit(),
	}
}

func (c *arsipController) Log(ctx *gin.Context) {
	p := pagination.BindPagination(ctx)
	rows, total, err := c.service.Log(ctx.Request.Context(), middlewares.Actor(ctx), logFilter(ctx, p))
	if err != nil {
		utils.RespondError(ctx, "gagal mengambil log arsip", err)
		return
	}
	ctx.JSON(http.StatusOK, pagination.NewPaginatedResponse(http.StatusOK, "berhasil mengambil log arsip", rows, pagination.CalculatePagination(p, total)))
}

func (c *arsipController) LogCSV(ctx *gin.Context) {
	p := pagination.BindPagination(ctx)
	data, err := c.service.LogCSV(ctx.Request.Context(), middlewares.Actor(ctx), logFilter(ctx, p))
	if err != nil {
		utils.RespondError(ctx, "gagal mengekspor log arsip", err)
		return
	}
	ctx.Header("Content-Disposition", `attachment; filename="log-arsip.csv"`)
	ctx.Data(http.StatusOK, "text/csv; charset=utf-8", data)
}
