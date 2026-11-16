package controller

import (
	"net/http"
	"strings"

	"github.com/Caknoooo/go-gin-clean-starter/middlewares"
	"github.com/Caknoooo/go-gin-clean-starter/modules/dasbor/service"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/utils"
	"github.com/Caknoooo/go-pagination"
	"github.com/gin-gonic/gin"
)

type DasborController interface {
	Tugas(ctx *gin.Context)
	Pengajuan(ctx *gin.Context)
	PengajuanCSV(ctx *gin.Context)
	KirimPengingat(ctx *gin.Context)
	Ringkasan(ctx *gin.Context)
	RekapPDF(ctx *gin.Context)
}

type dasborController struct {
	service service.DasborService
}

func NewDasborController(s service.DasborService) DasborController {
	return &dasborController{service: s}
}

func (c *dasborController) Tugas(ctx *gin.Context) {
	p := pagination.BindPagination(ctx)
	res, err := c.service.Tugas(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Query("kategori"), ctx.Query("modul"), p.GetOffset(), p.GetLimit())
	if err != nil {
		utils.RespondError(ctx, "gagal mengambil tugas", err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"code": http.StatusOK, "status": "success", "message": "berhasil mengambil tugas",
		"data": res.Items, "per_kategori": res.PerKategori,
		"pagination": pagination.CalculatePagination(p, int64(res.Total)),
	})
}

// modulDipilih membaca ?modul=a,b; ok=false bila respons galat sudah dikirim.
func modulDipilih(ctx *gin.Context) (modul []string, ok bool) {
	if m := ctx.Query("modul"); m != "" {
		modul = strings.Split(m, ",")
		if len(modul) > 10 {
			utils.RespondError(ctx, "data tidak valid", utils.ErrBadRequest("terlalu banyak modul"))
			return nil, false
		}
	}
	return modul, true
}

// KirimPengingat menjalankan pengingat tenggat saat itu juga (biasanya otomatis tiap pagi).
func (c *dasborController) KirimPengingat(ctx *gin.Context) {
	n, err := c.service.KirimPengingatTenggat(ctx.Request.Context())
	if err != nil {
		utils.RespondError(ctx, "gagal mengirim pengingat tenggat", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("pengingat tenggat terkirim", gin.H{"terkirim": n}))
}

func (c *dasborController) PengajuanCSV(ctx *gin.Context) {
	p := pagination.BindPagination(ctx)
	modul, ok := modulDipilih(ctx)
	if !ok {
		return
	}
	data, err := c.service.PengajuanCSV(ctx.Request.Context(), middlewares.Actor(ctx), modul, ctx.Query("kelompok"), p.Search)
	if err != nil {
		utils.RespondError(ctx, "gagal mengekspor rekap pengajuan", err)
		return
	}
	ctx.Header("Content-Disposition", `attachment; filename="rekap-pengajuan.csv"`)
	ctx.Data(http.StatusOK, "text/csv; charset=utf-8", data)
}

func (c *dasborController) Pengajuan(ctx *gin.Context) {
	p := pagination.BindPagination(ctx)
	modul, ok := modulDipilih(ctx)
	if !ok {
		return
	}
	items, total, err := c.service.Pengajuan(ctx.Request.Context(), middlewares.Actor(ctx), modul, ctx.Query("kelompok"), p.Search, p.GetOffset(), p.GetLimit())
	if err != nil {
		utils.RespondError(ctx, "gagal mengambil daftar pengajuan", err)
		return
	}
	ctx.JSON(http.StatusOK, pagination.NewPaginatedResponse(http.StatusOK, "berhasil mengambil daftar pengajuan", items, pagination.CalculatePagination(p, total)))
}

func (c *dasborController) Ringkasan(ctx *gin.Context) {
	res, err := c.service.Ringkasan(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Query("bulan"))
	if err != nil {
		utils.RespondError(ctx, "gagal mengambil ringkasan", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("berhasil mengambil ringkasan", res))
}

func (c *dasborController) RekapPDF(ctx *gin.Context) {
	data, err := c.service.RekapPDF(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Query("bulan"))
	if err != nil {
		utils.RespondError(ctx, "gagal membuat rekap", err)
		return
	}
	ctx.Header("Content-Disposition", `attachment; filename="rekap-kepegawaian.pdf"`)
	ctx.Data(http.StatusOK, "application/pdf", data)
}
