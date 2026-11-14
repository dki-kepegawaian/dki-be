package controller

import (
	"mime"
	"net/http"

	"github.com/Caknoooo/go-gin-clean-starter/middlewares"
	"github.com/Caknoooo/go-gin-clean-starter/modules/dokumen/service"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/utils"
	"github.com/gin-gonic/gin"
)

type DokumenController interface {
	Unduh(ctx *gin.Context)
}

type dokumenController struct {
	service service.DokumenService
}

func NewDokumenController(s service.DokumenService) DokumenController {
	return &dokumenController{service: s}
}

func (c *dokumenController) Unduh(ctx *gin.Context) {
	doc, r, size, err := c.service.Buka(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"))
	if err != nil {
		utils.RespondError(ctx, "gagal mengunduh dokumen", err)
		return
	}
	defer r.Close()

	disposition := "attachment"
	if ctx.Query("inline") == "1" {
		disposition = "inline"
	}
	ctx.DataFromReader(http.StatusOK, size, doc.MimeType, r, map[string]string{
		"Content-Disposition": mime.FormatMediaType(disposition, map[string]string{"filename": doc.NamaAsli}),
	})
}
