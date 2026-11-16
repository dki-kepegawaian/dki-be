package controller

import (
	"net/http"
	"strings"

	"github.com/Caknoooo/go-gin-clean-starter/middlewares"
	"github.com/Caknoooo/go-gin-clean-starter/modules/anjab/dto"
	"github.com/Caknoooo/go-gin-clean-starter/modules/anjab/service"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/utils"
	"github.com/Caknoooo/go-pagination"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

const mimeXLSX = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

type AnjabController interface {
	DaftarPeriode(ctx *gin.Context)
	BuatPeriode(ctx *gin.Context)
	DetailPeriode(ctx *gin.Context)
	UbahPeriode(ctx *gin.Context)
	BukaPembaruan(ctx *gin.Context)
	PerpanjangTenggat(ctx *gin.Context)
	KirimPengingat(ctx *gin.Context)
	Templat(ctx *gin.Context)
	Impor(ctx *gin.Context)
	Monitoring(ctx *gin.Context)
	MonitoringExcel(ctx *gin.Context)
	LaporanPDF(ctx *gin.Context)

	Daftar(ctx *gin.Context)
	Detail(ctx *gin.Context)
	UbahRombel(ctx *gin.Context)
	TambahGuru(ctx *gin.Context)
	UbahGuru(ctx *gin.Context)
	HapusGuru(ctx *gin.Context)
	Kirim(ctx *gin.Context)
	Tinjau(ctx *gin.Context)
}

type anjabController struct {
	service service.AnjabService
}

func NewAnjabController(s service.AnjabService) AnjabController {
	return &anjabController{service: s}
}

func bind(ctx *gin.Context, req any, b binding.Binding) bool {
	if err := ctx.ShouldBindWith(req, b); err != nil {
		utils.RespondError(ctx, "data tidak valid", utils.ErrBadRequest(err.Error()))
		return false
	}
	return true
}

func ok(ctx *gin.Context, status int, pesan string, data any) {
	ctx.JSON(status, utils.BuildResponseSuccess(pesan, data))
}

func unduh(ctx *gin.Context, data []byte, nama string) {
	ctx.Header("Content-Disposition", `attachment; filename="`+nama+`"`)
	ctx.Data(http.StatusOK, mimeXLSX, data)
}

func (c *anjabController) DaftarPeriode(ctx *gin.Context) {
	p := pagination.BindPagination(ctx)
	items, total, err := c.service.DaftarPeriode(ctx.Request.Context(), middlewares.Actor(ctx), p.GetOffset(), p.GetLimit())
	if err != nil {
		utils.RespondError(ctx, "gagal mengambil daftar periode", err)
		return
	}
	ctx.JSON(http.StatusOK, pagination.NewPaginatedResponse(http.StatusOK, "berhasil mengambil daftar periode", items, pagination.CalculatePagination(p, total)))
}

func (c *anjabController) BuatPeriode(ctx *gin.Context) {
	var req dto.PeriodeRequest
	if !bind(ctx, &req, binding.JSON) {
		return
	}
	res, err := c.service.BuatPeriode(ctx.Request.Context(), middlewares.Actor(ctx), req)
	if err != nil {
		utils.RespondError(ctx, "gagal membuat periode", err)
		return
	}
	ok(ctx, http.StatusCreated, "periode tersimpan", res)
}

func (c *anjabController) DetailPeriode(ctx *gin.Context) {
	res, err := c.service.DetailPeriode(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"))
	if err != nil {
		utils.RespondError(ctx, "gagal mengambil periode", err)
		return
	}
	ok(ctx, http.StatusOK, "berhasil mengambil periode", res)
}

func (c *anjabController) UbahPeriode(ctx *gin.Context) {
	var req dto.PeriodeRequest
	if !bind(ctx, &req, binding.JSON) {
		return
	}
	res, err := c.service.UbahPeriode(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"), req)
	if err != nil {
		utils.RespondError(ctx, "gagal menyimpan parameter", err)
		return
	}
	ok(ctx, http.StatusOK, "parameter tersimpan", res)
}

func (c *anjabController) BukaPembaruan(ctx *gin.Context) {
	res, err := c.service.BukaPembaruan(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"))
	if err != nil {
		utils.RespondError(ctx, "gagal membuka masa pembaruan", err)
		return
	}
	ok(ctx, http.StatusOK, "masa pembaruan dibuka", res)
}

func (c *anjabController) PerpanjangTenggat(ctx *gin.Context) {
	var req dto.TenggatRequest
	if !bind(ctx, &req, binding.JSON) {
		return
	}
	res, err := c.service.PerpanjangTenggat(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"), req)
	if err != nil {
		utils.RespondError(ctx, "gagal memperpanjang tenggat", err)
		return
	}
	ok(ctx, http.StatusOK, "tenggat diperpanjang", res)
}

func (c *anjabController) KirimPengingat(ctx *gin.Context) {
	n, err := c.service.KirimPengingat(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"))
	if err != nil {
		utils.RespondError(ctx, "gagal mengirim pengingat", err)
		return
	}
	ok(ctx, http.StatusOK, "pengingat terkirim", gin.H{"jumlah_sekolah": n})
}

func (c *anjabController) Templat(ctx *gin.Context) {
	data, nama, err := c.service.Templat(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"))
	if err != nil {
		utils.RespondError(ctx, "gagal membuat templat", err)
		return
	}
	unduh(ctx, data, nama)
}

func (c *anjabController) Impor(ctx *gin.Context) {
	var req dto.ImporRequest
	if !bind(ctx, &req, binding.FormMultipart) {
		return
	}
	res, err := c.service.Impor(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"), req)
	if err != nil {
		utils.RespondError(ctx, "gagal mengimpor data master", err)
		return
	}
	pesan := "hasil validasi impor"
	if res.Tersimpan {
		pesan = "baris valid tersimpan"
	}
	ok(ctx, http.StatusOK, pesan, res)
}

func (c *anjabController) Monitoring(ctx *gin.Context) {
	res, err := c.service.Monitoring(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"), ctx.Query("mapel"))
	if err != nil {
		utils.RespondError(ctx, "gagal mengambil monitoring", err)
		return
	}
	ok(ctx, http.StatusOK, "berhasil mengambil monitoring", res)
}

func (c *anjabController) MonitoringExcel(ctx *gin.Context) {
	data, nama, err := c.service.MonitoringExcel(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"))
	if err != nil {
		utils.RespondError(ctx, "gagal membuat rekap", err)
		return
	}
	unduh(ctx, data, nama)
}

func (c *anjabController) LaporanPDF(ctx *gin.Context) {
	data, nama, err := c.service.LaporanPDF(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"))
	if err != nil {
		utils.RespondError(ctx, "gagal membuat laporan", err)
		return
	}
	ctx.Header("Content-Disposition", `attachment; filename="`+nama+`"`)
	ctx.Data(http.StatusOK, "application/pdf", data)
}

func (c *anjabController) Daftar(ctx *gin.Context) {
	p := pagination.BindPagination(ctx)
	var status []string
	if s := ctx.Query("status"); s != "" {
		status = strings.Split(s, ",")
	}
	items, total, err := c.service.Daftar(ctx.Request.Context(), middlewares.Actor(ctx), dto.Filter{
		PeriodeID: ctx.Query("periode_id"),
		Status:    status,
		Kecamatan: ctx.Query("kecamatan"),
		Search:    p.Search,
		Offset:    p.GetOffset(),
		Limit:     p.GetLimit(),
	})
	if err != nil {
		utils.RespondError(ctx, "gagal mengambil daftar sekolah", err)
		return
	}
	ctx.JSON(http.StatusOK, pagination.NewPaginatedResponse(http.StatusOK, "berhasil mengambil daftar sekolah", items, pagination.CalculatePagination(p, total)))
}

func (c *anjabController) Detail(ctx *gin.Context) {
	res, err := c.service.Detail(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"))
	if err != nil {
		utils.RespondError(ctx, "gagal mengambil data Anjab sekolah", err)
		return
	}
	ok(ctx, http.StatusOK, "berhasil mengambil data Anjab sekolah", res)
}

func (c *anjabController) UbahRombel(ctx *gin.Context) {
	var req dto.RombelRequest
	if !bind(ctx, &req, binding.JSON) {
		return
	}
	res, err := c.service.UbahRombel(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"), req)
	if err != nil {
		utils.RespondError(ctx, "gagal menyimpan rombel", err)
		return
	}
	ok(ctx, http.StatusOK, "rombel tersimpan", res)
}

func (c *anjabController) TambahGuru(ctx *gin.Context) {
	var req dto.GuruTambahRequest
	if !bind(ctx, &req, binding.JSON) {
		return
	}
	res, err := c.service.TambahGuru(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"), req)
	if err != nil {
		utils.RespondError(ctx, "gagal menambah guru", err)
		return
	}
	ok(ctx, http.StatusCreated, "guru ditambahkan", res)
}

func (c *anjabController) UbahGuru(ctx *gin.Context) {
	var req dto.GuruUbahRequest
	if !bind(ctx, &req, binding.JSON) {
		return
	}
	res, err := c.service.UbahGuru(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"), ctx.Param("guruId"), req)
	if err != nil {
		utils.RespondError(ctx, "gagal mengubah data guru", err)
		return
	}
	ok(ctx, http.StatusOK, "data guru tersimpan", res)
}

func (c *anjabController) HapusGuru(ctx *gin.Context) {
	res, err := c.service.HapusGuru(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"), ctx.Param("guruId"))
	if err != nil {
		utils.RespondError(ctx, "gagal menghapus guru dari daftar", err)
		return
	}
	ok(ctx, http.StatusOK, "guru dihapus dari daftar", res)
}

func (c *anjabController) Kirim(ctx *gin.Context) {
	res, err := c.service.Kirim(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"))
	if err != nil {
		utils.RespondError(ctx, "gagal mengirim pembaruan", err)
		return
	}
	ok(ctx, http.StatusOK, "pembaruan terkirim", res)
}

func (c *anjabController) Tinjau(ctx *gin.Context) {
	var req dto.TinjauRequest
	if !bind(ctx, &req, binding.JSON) {
		return
	}
	res, err := c.service.Tinjau(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"), req)
	if err != nil {
		utils.RespondError(ctx, "gagal menyimpan hasil tinjauan", err)
		return
	}
	ok(ctx, http.StatusOK, "hasil tinjauan tersimpan", res)
}
