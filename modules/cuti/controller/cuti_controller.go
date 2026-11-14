package controller

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Caknoooo/go-gin-clean-starter/middlewares"
	"github.com/Caknoooo/go-gin-clean-starter/modules/cuti/dto"
	"github.com/Caknoooo/go-gin-clean-starter/modules/cuti/service"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/utils"
	"github.com/Caknoooo/go-pagination"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type CutiController interface {
	Ajukan(ctx *gin.Context)
	Daftar(ctx *gin.Context)
	Ringkasan(ctx *gin.Context)
	SisaTahunan(ctx *gin.Context)
	Detail(ctx *gin.Context)
	Perbaiki(ctx *gin.Context)
	Periksa(ctx *gin.Context)
	TerimaSuratFisik(ctx *gin.Context)
	Putuskan(ctx *gin.Context)
}

type cutiController struct {
	service service.CutiService
}

func NewCutiController(s service.CutiService) CutiController {
	return &cutiController{service: s}
}

func bind(ctx *gin.Context, req any, b binding.Binding) bool {
	if err := ctx.ShouldBindWith(req, b); err != nil {
		utils.RespondError(ctx, "data tidak valid", utils.ErrBadRequest(err.Error()))
		return false
	}
	return true
}

func (c *cutiController) Ajukan(ctx *gin.Context) {
	var req dto.AjukanRequest
	if !bind(ctx, &req, binding.FormMultipart) {
		return
	}
	res, err := c.service.Ajukan(ctx.Request.Context(), middlewares.Actor(ctx), req)
	if err != nil {
		utils.RespondError(ctx, "gagal mengajukan cuti", err)
		return
	}
	ctx.JSON(http.StatusCreated, utils.BuildResponseSuccess("pengajuan cuti terkirim", res))
}

func (c *cutiController) Daftar(ctx *gin.Context) {
	p := pagination.BindPagination(ctx)
	var status []string
	if s := ctx.Query("status"); s != "" {
		status = strings.Split(s, ",")
	}
	items, total, err := c.service.Daftar(ctx.Request.Context(), middlewares.Actor(ctx), dto.Filter{
		Status:    status,
		Jenis:     ctx.Query("jenis"),
		SekolahID: ctx.Query("sekolah_id"),
		Search:    p.Search,
		Offset:    p.GetOffset(),
		Limit:     p.GetLimit(),
	})
	if err != nil {
		utils.RespondError(ctx, "gagal mengambil daftar cuti", err)
		return
	}
	ctx.JSON(http.StatusOK, pagination.NewPaginatedResponse(http.StatusOK, "berhasil mengambil daftar cuti", items, pagination.CalculatePagination(p, total)))
}

func (c *cutiController) Ringkasan(ctx *gin.Context) {
	res, err := c.service.Ringkasan(ctx.Request.Context(), middlewares.Actor(ctx))
	if err != nil {
		utils.RespondError(ctx, "gagal mengambil ringkasan cuti", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("berhasil mengambil ringkasan cuti", res))
}

func (c *cutiController) SisaTahunan(ctx *gin.Context) {
	tahun := time.Now().Year()
	if t := ctx.Query("tahun"); t != "" {
		v, err := strconv.Atoi(t)
		if err != nil || v < 2000 || v > 2100 {
			utils.RespondError(ctx, "data tidak valid", utils.ErrBadRequest("tahun tidak valid"))
			return
		}
		tahun = v
	}
	res, err := c.service.SisaTahunan(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Query("pendidik_id"), tahun)
	if err != nil {
		utils.RespondError(ctx, "gagal menghitung sisa cuti", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("berhasil menghitung sisa cuti tahunan", res))
}

func (c *cutiController) Detail(ctx *gin.Context) {
	res, err := c.service.Detail(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"))
	if err != nil {
		utils.RespondError(ctx, "gagal mengambil pengajuan cuti", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("berhasil mengambil pengajuan cuti", res))
}

func (c *cutiController) Perbaiki(ctx *gin.Context) {
	var req dto.PerbaikiRequest
	if !bind(ctx, &req, binding.FormMultipart) {
		return
	}
	res, err := c.service.Perbaiki(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"), req)
	if err != nil {
		utils.RespondError(ctx, "gagal memperbaiki pengajuan cuti", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("perbaikan berkas terkirim", res))
}

func (c *cutiController) Periksa(ctx *gin.Context) {
	var req dto.PeriksaRequest
	if !bind(ctx, &req, binding.JSON) {
		return
	}
	res, err := c.service.Periksa(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"), req)
	if err != nil {
		utils.RespondError(ctx, "gagal menyimpan hasil pemeriksaan", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("hasil pemeriksaan tersimpan", res))
}

func (c *cutiController) TerimaSuratFisik(ctx *gin.Context) {
	var req dto.TerimaSuratFisikRequest
	if !bind(ctx, &req, binding.JSON) {
		return
	}
	res, err := c.service.TerimaSuratFisik(ctx.Request.Context(), middlewares.Actor(ctx), req)
	if err != nil {
		utils.RespondError(ctx, "gagal mencatat penerimaan surat fisik", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("penerimaan surat fisik tercatat", res))
}

func (c *cutiController) Putuskan(ctx *gin.Context) {
	var req dto.KeputusanRequest
	if !bind(ctx, &req, binding.FormMultipart) {
		return
	}
	res, err := c.service.Putuskan(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"), req)
	if err != nil {
		utils.RespondError(ctx, "gagal menyimpan keputusan cuti", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("keputusan cuti tersimpan", res))
}
