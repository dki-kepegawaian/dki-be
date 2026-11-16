package controller

import (
	"net/http"
	"strings"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"github.com/Caknoooo/go-gin-clean-starter/middlewares"
	"github.com/Caknoooo/go-gin-clean-starter/modules/kenaikanpangkat/dto"
	"github.com/Caknoooo/go-gin-clean-starter/modules/kenaikanpangkat/service"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/utils"
	"github.com/Caknoooo/go-pagination"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type KenaikanPangkatController interface {
	Persyaratan(ctx *gin.Context)
	Perkiraan(ctx *gin.Context)
	MemenuhiSyarat(ctx *gin.Context)
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

type kpController struct {
	service service.KenaikanPangkatService
}

func NewKenaikanPangkatController(s service.KenaikanPangkatService) KenaikanPangkatController {
	return &kpController{service: s}
}

func bind(ctx *gin.Context, req any, b binding.Binding) bool {
	if err := ctx.ShouldBindWith(req, b); err != nil {
		utils.RespondError(ctx, "data tidak valid", utils.ErrBadRequest(err.Error()))
		return false
	}
	return true
}

func (c *kpController) Persyaratan(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("berkas persyaratan kenaikan pangkat", entities.KPBerkasWajib))
}

func (c *kpController) Perkiraan(ctx *gin.Context) {
	res, err := c.service.Perkiraan(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Query("pendidik_id"))
	if err != nil {
		utils.RespondError(ctx, "gagal mengambil data kenaikan pangkat guru", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("berhasil mengambil data kenaikan pangkat guru", res))
}

func (c *kpController) MemenuhiSyarat(ctx *gin.Context) {
	res, err := c.service.MemenuhiSyarat(ctx.Request.Context(), middlewares.Actor(ctx))
	if err != nil {
		utils.RespondError(ctx, "gagal mengambil daftar guru memenuhi syarat", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("berhasil mengambil daftar guru memenuhi syarat", res))
}

func (c *kpController) UnggahBerkas(ctx *gin.Context) {
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

func (c *kpController) Ajukan(ctx *gin.Context) {
	var req dto.AjukanRequest
	if !bind(ctx, &req, binding.JSON) {
		return
	}
	res, err := c.service.Ajukan(ctx.Request.Context(), middlewares.Actor(ctx), req)
	if err != nil {
		utils.RespondError(ctx, "gagal mengajukan kenaikan pangkat", err)
		return
	}
	ctx.JSON(http.StatusCreated, utils.BuildResponseSuccess("pengajuan kenaikan pangkat terkirim", res))
}

func (c *kpController) Daftar(ctx *gin.Context) {
	p := pagination.BindPagination(ctx)
	var status []string
	if s := ctx.Query("status"); s != "" {
		status = strings.Split(s, ",")
	}
	items, total, err := c.service.Daftar(ctx.Request.Context(), middlewares.Actor(ctx), dto.Filter{
		Status:    status,
		SekolahID: ctx.Query("sekolah_id"),
		Periode:   ctx.Query("periode"),
		Search:    p.Search,
		Offset:    p.GetOffset(),
		Limit:     p.GetLimit(),
	})
	if err != nil {
		utils.RespondError(ctx, "gagal mengambil daftar kenaikan pangkat", err)
		return
	}
	ctx.JSON(http.StatusOK, pagination.NewPaginatedResponse(http.StatusOK, "berhasil mengambil daftar kenaikan pangkat", items, pagination.CalculatePagination(p, total)))
}

func (c *kpController) Ringkasan(ctx *gin.Context) {
	res, err := c.service.Ringkasan(ctx.Request.Context(), middlewares.Actor(ctx))
	if err != nil {
		utils.RespondError(ctx, "gagal mengambil ringkasan kenaikan pangkat", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("berhasil mengambil ringkasan kenaikan pangkat", res))
}

func (c *kpController) Detail(ctx *gin.Context) {
	res, err := c.service.Detail(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"))
	if err != nil {
		utils.RespondError(ctx, "gagal mengambil pengajuan kenaikan pangkat", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("berhasil mengambil pengajuan kenaikan pangkat", res))
}

func (c *kpController) Perbaiki(ctx *gin.Context) {
	var req dto.PerbaikiRequest
	if !bind(ctx, &req, binding.JSON) {
		return
	}
	res, err := c.service.Perbaiki(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"), req)
	if err != nil {
		utils.RespondError(ctx, "gagal memperbaiki pengajuan kenaikan pangkat", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("perbaikan berkas terkirim", res))
}

func (c *kpController) Verifikasi(ctx *gin.Context) {
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

func (c *kpController) KirimBKD(ctx *gin.Context) {
	var req dto.KirimBKDRequest
	if !bind(ctx, &req, binding.FormMultipart) {
		return
	}
	res, err := c.service.KirimBKD(ctx.Request.Context(), middlewares.Actor(ctx), req)
	if err != nil {
		utils.RespondError(ctx, "gagal mencatat pengiriman ke BKD", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("pengiriman ke BKD tercatat", res))
}

func (c *kpController) UnggahSK(ctx *gin.Context) {
	var req dto.SKRequest
	if !bind(ctx, &req, binding.FormMultipart) {
		return
	}
	res, err := c.service.UnggahSK(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"), req)
	if err != nil {
		utils.RespondError(ctx, "gagal menyimpan SK kenaikan pangkat", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("SK kenaikan pangkat tersimpan, pengajuan selesai", res))
}
