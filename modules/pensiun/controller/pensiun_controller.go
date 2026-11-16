package controller

import (
	"net/http"
	"strings"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"github.com/Caknoooo/go-gin-clean-starter/middlewares"
	"github.com/Caknoooo/go-gin-clean-starter/modules/pensiun/dto"
	"github.com/Caknoooo/go-gin-clean-starter/modules/pensiun/service"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/utils"
	"github.com/Caknoooo/go-pagination"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type PensiunController interface {
	Persyaratan(ctx *gin.Context)
	Perkiraan(ctx *gin.Context)
	UnggahBerkas(ctx *gin.Context)
	Ajukan(ctx *gin.Context)
	Daftar(ctx *gin.Context)
	Ringkasan(ctx *gin.Context)
	Detail(ctx *gin.Context)
	Perbaiki(ctx *gin.Context)
	Verifikasi(ctx *gin.Context)
	KirimBKD(ctx *gin.Context)
	UnggahSK(ctx *gin.Context)
}

type pensiunController struct {
	service service.PensiunService
}

func NewPensiunController(s service.PensiunService) PensiunController {
	return &pensiunController{service: s}
}

func bind(ctx *gin.Context, req any, b binding.Binding) bool {
	if err := ctx.ShouldBindWith(req, b); err != nil {
		utils.RespondError(ctx, "data tidak valid", utils.ErrBadRequest(err.Error()))
		return false
	}
	return true
}

func (c *pensiunController) Persyaratan(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("berkas persyaratan pensiun", entities.PensiunBerkasWajib))
}

func (c *pensiunController) Perkiraan(ctx *gin.Context) {
	res, err := c.service.Perkiraan(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Query("pendidik_id"))
	if err != nil {
		utils.RespondError(ctx, "gagal mengambil data pensiun guru", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("berhasil mengambil data pensiun guru", res))
}

func (c *pensiunController) UnggahBerkas(ctx *gin.Context) {
	var req dto.UnggahBerkasRequest
	if !bind(ctx, &req, binding.FormMultipart) {
		return
	}
	res, err := c.service.UnggahBerkas(ctx.Request.Context(), middlewares.Actor(ctx), req)
	if err != nil {
		utils.RespondError(ctx, "gagal mengunggah berkas", err)
		return
	}
	ctx.JSON(http.StatusCreated, utils.BuildResponseSuccess("berkas terunggah", res))
}

func (c *pensiunController) Ajukan(ctx *gin.Context) {
	var req dto.AjukanRequest
	if !bind(ctx, &req, binding.JSON) {
		return
	}
	res, err := c.service.Ajukan(ctx.Request.Context(), middlewares.Actor(ctx), req)
	if err != nil {
		utils.RespondError(ctx, "gagal mengajukan pensiun", err)
		return
	}
	ctx.JSON(http.StatusCreated, utils.BuildResponseSuccess("pengajuan pensiun terkirim", res))
}

func (c *pensiunController) Daftar(ctx *gin.Context) {
	p := pagination.BindPagination(ctx)
	var status []string
	if s := ctx.Query("status"); s != "" {
		status = strings.Split(s, ",")
	}
	items, total, err := c.service.Daftar(ctx.Request.Context(), middlewares.Actor(ctx), dto.Filter{
		Status:    status,
		SekolahID: ctx.Query("sekolah_id"),
		Search:    p.Search,
		Offset:    p.GetOffset(),
		Limit:     p.GetLimit(),
	})
	if err != nil {
		utils.RespondError(ctx, "gagal mengambil daftar pensiun", err)
		return
	}
	ctx.JSON(http.StatusOK, pagination.NewPaginatedResponse(http.StatusOK, "berhasil mengambil daftar pensiun", items, pagination.CalculatePagination(p, total)))
}

func (c *pensiunController) Ringkasan(ctx *gin.Context) {
	res, err := c.service.Ringkasan(ctx.Request.Context(), middlewares.Actor(ctx))
	if err != nil {
		utils.RespondError(ctx, "gagal mengambil ringkasan pensiun", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("berhasil mengambil ringkasan pensiun", res))
}

func (c *pensiunController) Detail(ctx *gin.Context) {
	res, err := c.service.Detail(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"))
	if err != nil {
		utils.RespondError(ctx, "gagal mengambil pengajuan pensiun", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("berhasil mengambil pengajuan pensiun", res))
}

func (c *pensiunController) Perbaiki(ctx *gin.Context) {
	var req dto.PerbaikiRequest
	if !bind(ctx, &req, binding.JSON) {
		return
	}
	res, err := c.service.Perbaiki(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"), req)
	if err != nil {
		utils.RespondError(ctx, "gagal memperbaiki pengajuan pensiun", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("perbaikan berkas terkirim", res))
}

func (c *pensiunController) Verifikasi(ctx *gin.Context) {
	var req dto.VerifikasiRequest
	if !bind(ctx, &req, binding.JSON) {
		return
	}
	res, err := c.service.Verifikasi(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"), req)
	if err != nil {
		utils.RespondError(ctx, "gagal menyimpan verifikasi berkas", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("verifikasi berkas tersimpan", res))
}

func (c *pensiunController) KirimBKD(ctx *gin.Context) {
	var req dto.KirimBKDRequest
	if !bind(ctx, &req, binding.FormMultipart) {
		return
	}
	res, err := c.service.KirimBKD(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"), req)
	if err != nil {
		utils.RespondError(ctx, "gagal mencatat pengiriman ke BKD", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("pengiriman ke BKD tercatat", res))
}

func (c *pensiunController) UnggahSK(ctx *gin.Context) {
	var req dto.SKRequest
	if !bind(ctx, &req, binding.FormMultipart) {
		return
	}
	res, err := c.service.UnggahSK(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"), req)
	if err != nil {
		utils.RespondError(ctx, "gagal menyimpan SK pensiun", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("SK pensiun tersimpan, pengajuan selesai", res))
}
