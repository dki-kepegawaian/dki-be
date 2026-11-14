package controller

import (
	"net/http"

	"github.com/Caknoooo/go-gin-clean-starter/middlewares"
	"github.com/Caknoooo/go-gin-clean-starter/modules/notifikasi/service"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/utils"
	"github.com/Caknoooo/go-pagination"
	"github.com/gin-gonic/gin"
)

type NotifikasiController interface {
	Daftar(ctx *gin.Context)
	TandaiDibaca(ctx *gin.Context)
	TandaiSemuaDibaca(ctx *gin.Context)
}

type notifikasiController struct {
	service service.NotifikasiService
}

func NewNotifikasiController(s service.NotifikasiService) NotifikasiController {
	return &notifikasiController{service: s}
}

func (c *notifikasiController) Daftar(ctx *gin.Context) {
	p := pagination.BindPagination(ctx)
	actor := middlewares.Actor(ctx)
	items, total, belum, err := c.service.Daftar(ctx.Request.Context(), actor.ID, ctx.Query("belum_dibaca") == "1", p.GetOffset(), p.GetLimit())
	if err != nil {
		utils.RespondError(ctx, "gagal mengambil notifikasi", err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"code":         http.StatusOK,
		"status":       "success",
		"message":      "berhasil mengambil notifikasi",
		"data":         items,
		"pagination":   pagination.CalculatePagination(p, total),
		"belum_dibaca": belum,
	})
}

func (c *notifikasiController) TandaiDibaca(ctx *gin.Context) {
	if err := c.service.TandaiDibaca(ctx.Request.Context(), middlewares.Actor(ctx).ID, ctx.Param("id")); err != nil {
		utils.RespondError(ctx, "gagal menandai notifikasi", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("notifikasi ditandai dibaca", nil))
}

func (c *notifikasiController) TandaiSemuaDibaca(ctx *gin.Context) {
	if err := c.service.TandaiSemuaDibaca(ctx.Request.Context(), middlewares.Actor(ctx).ID); err != nil {
		utils.RespondError(ctx, "gagal menandai notifikasi", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("semua notifikasi ditandai dibaca", nil))
}
