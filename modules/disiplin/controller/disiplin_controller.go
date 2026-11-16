package controller

import (
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/Caknoooo/go-gin-clean-starter/middlewares"
	"github.com/Caknoooo/go-gin-clean-starter/modules/disiplin/dto"
	"github.com/Caknoooo/go-gin-clean-starter/modules/disiplin/service"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/utils"
	"github.com/Caknoooo/go-pagination"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type DisiplinController interface {
	Lapor(ctx *gin.Context)
	Daftar(ctx *gin.Context)
	Ringkasan(ctx *gin.Context)
	Detail(ctx *gin.Context)
	UnduhBerkas(ctx *gin.Context)
	Verifikasi(ctx *gin.Context)
	Lengkapi(ctx *gin.Context)
	Pemeriksaan(ctx *gin.Context)
	Rekomendasi(ctx *gin.Context)
	Putuskan(ctx *gin.Context)
	CatatKeputusan(ctx *gin.Context)
	TindakLanjut(ctx *gin.Context)
	LaporanPelaksanaan(ctx *gin.Context)
	Tutup(ctx *gin.Context)
}

type disiplinController struct {
	service service.DisiplinService
}

func NewDisiplinController(s service.DisiplinService) DisiplinController {
	return &disiplinController{service: s}
}

func bind(ctx *gin.Context, req any, b binding.Binding) bool {
	if err := ctx.ShouldBindWith(req, b); err != nil {
		utils.RespondError(ctx, "data tidak valid", utils.ErrBadRequest(err.Error()))
		return false
	}
	return true
}

// aksi menjalankan satu langkah alur pada kasus :id dan menulis responsnya.
func aksi[T any](ctx *gin.Context, b binding.Binding, gagal, sukses string,
	fn func(ctx *gin.Context, id string, req T) (dto.DisiplinResponse, error)) {
	var req T
	if !bind(ctx, &req, b) {
		return
	}
	res, err := fn(ctx, ctx.Param("id"), req)
	if err != nil {
		utils.RespondError(ctx, gagal, err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess(sukses, res))
}

func (c *disiplinController) Lapor(ctx *gin.Context) {
	var req dto.LaporRequest
	if !bind(ctx, &req, binding.FormMultipart) {
		return
	}
	res, err := c.service.Lapor(ctx.Request.Context(), middlewares.Actor(ctx), req)
	if err != nil {
		utils.RespondError(ctx, "gagal mengirim laporan", err)
		return
	}
	ctx.JSON(http.StatusCreated, utils.BuildResponseSuccess("laporan dugaan pelanggaran terkirim", res))
}

func (c *disiplinController) Daftar(ctx *gin.Context) {
	p := pagination.BindPagination(ctx)
	var status []string
	if s := ctx.Query("status"); s != "" {
		status = strings.Split(s, ",")
	}
	tahun := 0
	if t := ctx.Query("tahun"); t != "" {
		v, err := strconv.Atoi(t)
		if err != nil || v < 2000 || v > 2100 {
			utils.RespondError(ctx, "data tidak valid", utils.ErrBadRequest("tahun tidak valid"))
			return
		}
		tahun = v
	}
	items, total, err := c.service.Daftar(ctx.Request.Context(), middlewares.Actor(ctx), dto.Filter{
		Status:    status,
		SekolahID: ctx.Query("sekolah_id"),
		Tahun:     tahun,
		Search:    p.Search,
		Offset:    p.GetOffset(),
		Limit:     p.GetLimit(),
	})
	if err != nil {
		utils.RespondError(ctx, "gagal mengambil daftar kasus", err)
		return
	}
	ctx.JSON(http.StatusOK, pagination.NewPaginatedResponse(http.StatusOK, "berhasil mengambil daftar kasus disiplin", items, pagination.CalculatePagination(p, total)))
}

func (c *disiplinController) Ringkasan(ctx *gin.Context) {
	res, err := c.service.Ringkasan(ctx.Request.Context(), middlewares.Actor(ctx))
	if err != nil {
		utils.RespondError(ctx, "gagal mengambil ringkasan kasus", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("berhasil mengambil ringkasan kasus disiplin", res))
}

func (c *disiplinController) Detail(ctx *gin.Context) {
	res, err := c.service.Detail(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"))
	if err != nil {
		utils.RespondError(ctx, "gagal mengambil kasus", err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("berhasil mengambil kasus disiplin", res))
}

func (c *disiplinController) UnduhBerkas(ctx *gin.Context) {
	doc, r, size, err := c.service.BukaBerkas(ctx.Request.Context(), middlewares.Actor(ctx), ctx.Param("id"), ctx.Param("dokumen_id"))
	if err != nil {
		utils.RespondError(ctx, "gagal mengunduh berkas", err)
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

func (c *disiplinController) Verifikasi(ctx *gin.Context) {
	aksi(ctx, binding.JSON, "gagal menyimpan verifikasi awal", "verifikasi awal tersimpan",
		func(ctx *gin.Context, id string, req dto.VerifikasiRequest) (dto.DisiplinResponse, error) {
			return c.service.Verifikasi(ctx.Request.Context(), middlewares.Actor(ctx), id, req)
		})
}

func (c *disiplinController) Lengkapi(ctx *gin.Context) {
	aksi(ctx, binding.FormMultipart, "gagal mengirim kelengkapan", "kelengkapan laporan terkirim",
		func(ctx *gin.Context, id string, req dto.KelengkapanRequest) (dto.DisiplinResponse, error) {
			return c.service.Lengkapi(ctx.Request.Context(), middlewares.Actor(ctx), id, req)
		})
}

func (c *disiplinController) Pemeriksaan(ctx *gin.Context) {
	aksi(ctx, binding.FormMultipart, "gagal menyimpan hasil pemeriksaan", "hasil pemeriksaan tersimpan",
		func(ctx *gin.Context, id string, req dto.PemeriksaanRequest) (dto.DisiplinResponse, error) {
			return c.service.Pemeriksaan(ctx.Request.Context(), middlewares.Actor(ctx), id, req)
		})
}

func (c *disiplinController) Rekomendasi(ctx *gin.Context) {
	aksi(ctx, binding.JSON, "gagal mengajukan rekomendasi", "rekomendasi diajukan ke pimpinan",
		func(ctx *gin.Context, id string, req dto.RekomendasiRequest) (dto.DisiplinResponse, error) {
			return c.service.Rekomendasi(ctx.Request.Context(), middlewares.Actor(ctx), id, req)
		})
}

func (c *disiplinController) Putuskan(ctx *gin.Context) {
	aksi(ctx, binding.JSON, "gagal menetapkan keputusan", "keputusan pimpinan tersimpan",
		func(ctx *gin.Context, id string, req dto.KeputusanRequest) (dto.DisiplinResponse, error) {
			return c.service.Putuskan(ctx.Request.Context(), middlewares.Actor(ctx), id, req)
		})
}

func (c *disiplinController) CatatKeputusan(ctx *gin.Context) {
	aksi(ctx, binding.FormMultipart, "gagal mencatat keputusan", "keputusan final tercatat",
		func(ctx *gin.Context, id string, req dto.CatatKeputusanRequest) (dto.DisiplinResponse, error) {
			return c.service.CatatKeputusan(ctx.Request.Context(), middlewares.Actor(ctx), id, req)
		})
}

func (c *disiplinController) TindakLanjut(ctx *gin.Context) {
	aksi(ctx, binding.FormMultipart, "gagal menyimpan tindak lanjut", "tindak lanjut tersimpan",
		func(ctx *gin.Context, id string, req dto.TindakLanjutRequest) (dto.DisiplinResponse, error) {
			return c.service.TindakLanjut(ctx.Request.Context(), middlewares.Actor(ctx), id, req)
		})
}

func (c *disiplinController) LaporanPelaksanaan(ctx *gin.Context) {
	aksi(ctx, binding.FormMultipart, "gagal mengirim laporan pelaksanaan", "laporan pelaksanaan terkirim",
		func(ctx *gin.Context, id string, req dto.LaporanPelaksanaanRequest) (dto.DisiplinResponse, error) {
			return c.service.LaporanPelaksanaan(ctx.Request.Context(), middlewares.Actor(ctx), id, req)
		})
}

func (c *disiplinController) Tutup(ctx *gin.Context) {
	aksi(ctx, binding.JSON, "gagal menutup kasus", "kasus ditutup",
		func(ctx *gin.Context, id string, req dto.TutupRequest) (dto.DisiplinResponse, error) {
			return c.service.Tutup(ctx.Request.Context(), middlewares.Actor(ctx), id, req)
		})
}
