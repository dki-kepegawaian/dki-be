package controller

import (
	"net/http"

	"github.com/Caknoooo/go-gin-clean-starter/modules/auth/dto"
	"github.com/Caknoooo/go-gin-clean-starter/modules/auth/service"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/utils"
	"github.com/gin-gonic/gin"
)

type AuthController interface {
	Login(ctx *gin.Context)
	RefreshToken(ctx *gin.Context)
	Logout(ctx *gin.Context)
}

type authController struct {
	authService service.AuthService
}

func NewAuthController(as service.AuthService) AuthController {
	return &authController{authService: as}
}

func (c *authController) Login(ctx *gin.Context) {
	var req dto.LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.RespondError(ctx, "gagal masuk", utils.ErrBadRequest(err.Error()))
		return
	}
	res, err := c.authService.Login(ctx.Request.Context(), req)
	if err != nil {
		utils.RespondError(ctx, "gagal masuk", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("berhasil masuk", res))
}

func (c *authController) RefreshToken(ctx *gin.Context) {
	var req dto.RefreshTokenRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.RespondError(ctx, "gagal memperbarui token", utils.ErrBadRequest(err.Error()))
		return
	}
	res, err := c.authService.RefreshToken(ctx.Request.Context(), req)
	if err != nil {
		utils.RespondError(ctx, "gagal memperbarui token", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("berhasil memperbarui token", res))
}

func (c *authController) Logout(ctx *gin.Context) {
	if err := c.authService.Logout(ctx.Request.Context(), ctx.GetString("user_id")); err != nil {
		utils.RespondError(ctx, "gagal keluar", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("berhasil keluar", nil))
}
