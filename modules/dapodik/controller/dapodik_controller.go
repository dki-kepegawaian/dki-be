package controller

import (
	"net/http"
	"strings"

	"github.com/Caknoooo/go-gin-clean-starter/middlewares"
	"github.com/Caknoooo/go-gin-clean-starter/modules/dapodik/dto"
	"github.com/Caknoooo/go-gin-clean-starter/modules/dapodik/service"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/utils"
	"github.com/Caknoooo/go-pagination"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type DapodikController interface {
	Ajukan(ctx *gin.Context)
	Daftar(ctx *gin.Context)
	Ringkasan(ctx *gin.Context)
	Detail(ctx *gin.Context)
	Lengkapi(ctx *gin.Context)
	Tinjau(ctx *gin.Context)
}

type dapodikController struct {
	service service.DapodikService
}

func NewDapodikController(s service.DapodikService) DapodikController {
	return &dapodikController{service: s}
}

func bind(ctx *gin.Context, req any, b binding.Binding) bool {
	if err := ctx.ShouldBindWith(req, b); err != nil {
		utils.RespondError(ctx, "data tidak valid", utils.ErrBadRequest(err.Error()))
		return false
	}
	return true
}

func (c *dapodikController) Ajukan(ctx *gin.Context) {
	var req dto.AjukanRequest
	if !bind(ctx, &req, binding.FormMultipart) {
		return
	}
	res, err := c.service.Ajukan(ctx.Request.Context(), middlewares.Actor(ctx), req)
	if err != nil {
		utils.RespondError(ctx, "gagal mengirim tiket Dapodik", err)
		return
	}
	ctx.JSON(http.StatusCreated, utils.BuildResponseSuccess("tiket Dapodik terkirim", res))
}

func (c *dapodikController) Daftar(ctx *gin.Context) {
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
		Urut:      ctx.Query("urut"),
		Offset:    p.GetOffset(),
		Limit:     p.GetLimit(),
	})
	if err != nil {
		utils.RespondError(ctx, "gagal mengambil daftar tiket Dapodik", err)
		return
	}
	ctx.JSON(http.StatusOK, pagination.NewPaginatedResponse(http.StatusOK, "berhasil mengambil daftar tiket Dapodik", items, pagination.CalculatePagination(p, total)))
}

func (c *dapodikController) Ringkasan(ctx *gin.Context) {
	res, err := c.service.Ringkasan(ctx.Request.Context(), middlewares.Actor(ctx))
	if err != nil {
		utils.RespondError(ctx, "gagal mengambil ringkasan tiket Dapodik", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("berhasil mengambil ringkasan tiket Dapodik", res))
}

func (c *dapodikController) Detail(ctx *gin.Context) {
	res, err := c.service.Detail(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"))
	if err != nil {
		utils.RespondError(ctx, "gagal mengambil tiket Dapodik", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("berhasil mengambil tiket Dapodik", res))
}

func (c *dapodikController) Lengkapi(ctx *gin.Context) {
	var req dto.LengkapiRequest
	if !bind(ctx, &req, binding.FormMultipart) {
		return
	}
	res, err := c.service.Lengkapi(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"), req)
	if err != nil {
		utils.RespondError(ctx, "gagal melengkapi tiket Dapodik", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("kelengkapan tiket terkirim", res))
}

func (c *dapodikController) Tinjau(ctx *gin.Context) {
	var req dto.TinjauRequest
	if !bind(ctx, &req, binding.JSON) {
		return
	}
	res, err := c.service.Tinjau(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"), req)
	if err != nil {
		utils.RespondError(ctx, "gagal menyimpan tinjauan tiket Dapodik", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("tinjauan tiket tersimpan", res))
}
