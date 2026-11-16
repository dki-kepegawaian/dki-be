package controller

import (
	"net/http"

	"github.com/Caknoooo/go-gin-clean-starter/middlewares"
	"github.com/Caknoooo/go-gin-clean-starter/modules/user/dto"
	"github.com/Caknoooo/go-gin-clean-starter/modules/user/service"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/utils"
	"github.com/gin-gonic/gin"
)

type UserController interface {
	Me(ctx *gin.Context)
	GantiPassword(ctx *gin.Context)
	RiwayatMasuk(ctx *gin.Context)
	KeluarSesiLain(ctx *gin.Context)
}

type userController struct {
	userService service.UserService
}

func NewUserController(us service.UserService) UserController {
	return &userController{userService: us}
}

func (c *userController) Me(ctx *gin.Context) {
	res, err := c.userService.GetUserById(ctx.Request.Context(), middlewares.Actor(ctx).ID.String())
	if err != nil {
		utils.RespondError(ctx, "gagal mengambil akun", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("berhasil mengambil akun", res))
}

func (c *userController) GantiPassword(ctx *gin.Context) {
	var req dto.GantiPasswordRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.RespondError(ctx, "data tidak valid", utils.ErrBadRequest(err.Error()))
		return
	}
	if err := c.userService.GantiPassword(ctx.Request.Context(), middlewares.Actor(ctx).ID.String(), req); err != nil {
		utils.RespondError(ctx, "gagal mengganti password", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("password berhasil diganti", nil))
}

func (c *userController) RiwayatMasuk(ctx *gin.Context) {
	res, err := c.userService.RiwayatMasuk(ctx.Request.Context(), middlewares.Actor(ctx).ID.String())
	if err != nil {
		utils.RespondError(ctx, "gagal mengambil riwayat masuk", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("berhasil mengambil riwayat masuk", res))
}

func (c *userController) KeluarSesiLain(ctx *gin.Context) {
	var req dto.KeluarSesiLainRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.RespondError(ctx, "data tidak valid", utils.ErrBadRequest(err.Error()))
		return
	}
	if err := c.userService.KeluarSesiLain(ctx.Request.Context(), middlewares.Actor(ctx).ID.String(), req); err != nil {
		utils.RespondError(ctx, "gagal mengeluarkan sesi lain", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("semua sesi lain dikeluarkan", nil))
}
