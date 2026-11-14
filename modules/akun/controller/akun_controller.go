package controller

import (
	"net/http"

	"github.com/Caknoooo/go-gin-clean-starter/middlewares"
	"github.com/Caknoooo/go-gin-clean-starter/modules/akun/dto"
	"github.com/Caknoooo/go-gin-clean-starter/modules/akun/service"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/utils"
	"github.com/Caknoooo/go-pagination"
	"github.com/gin-gonic/gin"
)

type AkunController interface {
	Daftar(ctx *gin.Context)
	Detail(ctx *gin.Context)
	Tambah(ctx *gin.Context)
	Ubah(ctx *gin.Context)
	ResetPassword(ctx *gin.Context)
	Nonaktifkan(ctx *gin.Context)
}

type akunController struct {
	service service.AkunService
}

func NewAkunController(s service.AkunService) AkunController {
	return &akunController{service: s}
}

func (c *akunController) Daftar(ctx *gin.Context) {
	p := pagination.BindPagination(ctx)
	f := dto.Filter{
		Search: p.Search, Role: ctx.Query("role"), SekolahID: ctx.Query("sekolah_id"),
		Offset: p.GetOffset(), Limit: p.GetLimit(),
	}
	switch ctx.Query("aktif") {
	case "true":
		v := true
		f.Aktif = &v
	case "false":
		v := false
		f.Aktif = &v
	}
	items, total, err := c.service.Daftar(ctx.Request.Context(), middlewares.Actor(ctx), f)
	if err != nil {
		utils.RespondError(ctx, "gagal mengambil daftar akun", err)
		return
	}
	ctx.JSON(http.StatusOK, pagination.NewPaginatedResponse(http.StatusOK, "berhasil mengambil daftar akun", items, pagination.CalculatePagination(p, total)))
}

func (c *akunController) Detail(ctx *gin.Context) {
	res, err := c.service.Detail(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"))
	if err != nil {
		utils.RespondError(ctx, "gagal mengambil akun", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("berhasil mengambil akun", res))
}

func (c *akunController) Tambah(ctx *gin.Context) {
	var req dto.TambahRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.RespondError(ctx, "data akun tidak valid", utils.ErrBadRequest(err.Error()))
		return
	}
	res, err := c.service.Tambah(ctx.Request.Context(), middlewares.Actor(ctx), req)
	if err != nil {
		utils.RespondError(ctx, "gagal menambah akun", err)
		return
	}
	ctx.JSON(http.StatusCreated, utils.BuildResponseSuccess("akun berhasil dibuat", res))
}

func (c *akunController) Ubah(ctx *gin.Context) {
	var req dto.UbahRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.RespondError(ctx, "data akun tidak valid", utils.ErrBadRequest(err.Error()))
		return
	}
	res, err := c.service.Ubah(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"), req)
	if err != nil {
		utils.RespondError(ctx, "gagal mengubah akun", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("akun berhasil diubah", res))
}

func (c *akunController) ResetPassword(ctx *gin.Context) {
	var req dto.ResetPasswordRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.RespondError(ctx, "data tidak valid", utils.ErrBadRequest(err.Error()))
		return
	}
	if err := c.service.ResetPassword(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"), req); err != nil {
		utils.RespondError(ctx, "gagal mengganti password", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("password berhasil diganti", nil))
}

func (c *akunController) Nonaktifkan(ctx *gin.Context) {
	if err := c.service.Nonaktifkan(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id")); err != nil {
		utils.RespondError(ctx, "gagal menonaktifkan akun", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("akun dinonaktifkan", nil))
}
