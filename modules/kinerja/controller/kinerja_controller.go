package controller

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Caknoooo/go-gin-clean-starter/middlewares"
	"github.com/Caknoooo/go-gin-clean-starter/modules/kinerja/dto"
	"github.com/Caknoooo/go-gin-clean-starter/modules/kinerja/service"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/utils"
	"github.com/Caknoooo/go-pagination"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type KinerjaController interface {
	Buat(ctx *gin.Context)
	Daftar(ctx *gin.Context)
	Ringkasan(ctx *gin.Context)
	Pegawai(ctx *gin.Context)
	Detail(ctx *gin.Context)
	UbahNilai(ctx *gin.Context)
	BuatDokumen(ctx *gin.Context)
	Finalkan(ctx *gin.Context)
}

type kinerjaController struct {
	service service.KinerjaService
}

func NewKinerjaController(s service.KinerjaService) KinerjaController {
	return &kinerjaController{service: s}
}

func bind(ctx *gin.Context, req any, b binding.Binding) bool {
	if err := ctx.ShouldBindWith(req, b); err != nil {
		utils.RespondError(ctx, "data tidak valid", utils.ErrBadRequest(err.Error()))
		return false
	}
	return true
}

// periode membaca ?tahun= dan ?semester=. wajib=false membolehkan keduanya kosong (0).
func periode(ctx *gin.Context, wajib bool) (int, int, bool) {
	tahun, semester := 0, 0
	if t := ctx.Query("tahun"); t != "" || wajib {
		v, err := strconv.Atoi(t)
		if err != nil || v < 2020 || v > 2100 {
			utils.RespondError(ctx, "data tidak valid", utils.ErrBadRequest("tahun tidak valid"))
			return 0, 0, false
		}
		tahun = v
	}
	if sm := ctx.Query("semester"); sm != "" || wajib {
		if sm != "1" && sm != "2" {
			utils.RespondError(ctx, "data tidak valid", utils.ErrBadRequest("semester harus 1 atau 2"))
			return 0, 0, false
		}
		semester, _ = strconv.Atoi(sm)
	}
	return tahun, semester, true
}

func (c *kinerjaController) Buat(ctx *gin.Context) {
	var req dto.BuatRequest
	if !bind(ctx, &req, binding.JSON) {
		return
	}
	res, err := c.service.Buat(ctx.Request.Context(), middlewares.Actor(ctx), req)
	if err != nil {
		utils.RespondError(ctx, "gagal membuat penilaian kinerja", err)
		return
	}
	ctx.JSON(http.StatusCreated, utils.BuildResponseSuccess("draf penilaian tersimpan", res))
}

func (c *kinerjaController) Daftar(ctx *gin.Context) {
	tahun, semester, ok := periode(ctx, false)
	if !ok {
		return
	}
	p := pagination.BindPagination(ctx)
	var status []string
	if s := ctx.Query("status"); s != "" {
		status = strings.Split(s, ",")
	}
	items, total, err := c.service.Daftar(ctx.Request.Context(), middlewares.Actor(ctx), dto.Filter{
		Status:    status,
		Tahun:     tahun,
		Semester:  semester,
		SekolahID: ctx.Query("sekolah_id"),
		Search:    p.Search,
		Offset:    p.GetOffset(),
		Limit:     p.GetLimit(),
	})
	if err != nil {
		utils.RespondError(ctx, "gagal mengambil daftar penilaian kinerja", err)
		return
	}
	ctx.JSON(http.StatusOK, pagination.NewPaginatedResponse(http.StatusOK, "berhasil mengambil daftar penilaian kinerja", items, pagination.CalculatePagination(p, total)))
}

func (c *kinerjaController) Ringkasan(ctx *gin.Context) {
	tahun, semester, ok := periode(ctx, false)
	if !ok {
		return
	}
	res, err := c.service.Ringkasan(ctx.Request.Context(), middlewares.Actor(ctx), tahun, semester)
	if err != nil {
		utils.RespondError(ctx, "gagal mengambil ringkasan penilaian kinerja", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("berhasil mengambil ringkasan penilaian kinerja", res))
}

func (c *kinerjaController) Pegawai(ctx *gin.Context) {
	tahun, semester, ok := periode(ctx, true)
	if !ok {
		return
	}
	p := pagination.BindPagination(ctx)
	items, total, err := c.service.Pegawai(ctx.Request.Context(), middlewares.Actor(ctx),
		ctx.Query("sekolah_id"), p.Search, tahun, semester, p.GetOffset(), p.GetLimit())
	if err != nil {
		utils.RespondError(ctx, "gagal mengambil daftar pegawai P3K PW", err)
		return
	}
	ctx.JSON(http.StatusOK, pagination.NewPaginatedResponse(http.StatusOK, "berhasil mengambil daftar pegawai P3K PW", items, pagination.CalculatePagination(p, total)))
}

func (c *kinerjaController) Detail(ctx *gin.Context) {
	res, err := c.service.Detail(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"))
	if err != nil {
		utils.RespondError(ctx, "gagal mengambil penilaian kinerja", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("berhasil mengambil penilaian kinerja", res))
}

func (c *kinerjaController) UbahNilai(ctx *gin.Context) {
	var req dto.UbahNilaiRequest
	if !bind(ctx, &req, binding.JSON) {
		return
	}
	res, err := c.service.UbahNilai(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"), req)
	if err != nil {
		utils.RespondError(ctx, "gagal menyimpan nilai", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("draf penilaian tersimpan", res))
}

func (c *kinerjaController) BuatDokumen(ctx *gin.Context) {
	res, err := c.service.BuatDokumen(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"))
	if err != nil {
		utils.RespondError(ctx, "gagal membuat dokumen penilaian", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("dokumen penilaian dibuat", res))
}

func (c *kinerjaController) Finalkan(ctx *gin.Context) {
	var req dto.FinalRequest
	if !bind(ctx, &req, binding.FormMultipart) {
		return
	}
	res, err := c.service.Finalkan(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"), req)
	if err != nil {
		utils.RespondError(ctx, "gagal menetapkan dokumen Final", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("dokumen kinerja ditetapkan Final", res))
}
