package dto

import (
	"mime/multipart"
	"strconv"
	"time"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
)

type (
	// UnggahRequest (R3) dikirim sebagai multipart/form-data.
	UnggahRequest struct {
		PendidikID     string                `form:"pendidik_id" binding:"required,uuid"`
		Kategori       string                `form:"kategori" binding:"required,oneof=sk tunjangan ijazah kinerja surat lainnya"`
		JenisDokumen   string                `form:"jenis_dokumen" binding:"required,max=150"`
		Nomor          string                `form:"nomor" binding:"omitempty,max=100"`
		TanggalDokumen string                `form:"tanggal_dokumen" binding:"omitempty,datetime=2006-01-02"`
		Keterangan     string                `form:"keterangan" binding:"omitempty,max=1000"`
		Berkas         *multipart.FileHeader `form:"berkas" binding:"required"`
		Lampiran       *multipart.FileHeader `form:"lampiran"`
	}

	// VersiRequest (R2/R3 "simpan sebagai versi baru") dikirim sebagai multipart/form-data.
	VersiRequest struct {
		Catatan  string                `form:"catatan" binding:"omitempty,max=500"`
		Berkas   *multipart.FileHeader `form:"berkas" binding:"required"`
		Lampiran *multipart.FileHeader `form:"lampiran"`
	}

	Filter struct {
		PendidikID string
		Kategori   string
		Search     string
		Offset     int
		Limit      int
	}

	LogFilter struct {
		Dari   string // 2006-01-02, inklusif
		Sampai string // 2006-01-02, inklusif
		UserID string
		Aksi   string
		Search string
		Offset int
		Limit  int
	}
)

type (
	DokumenRingkas struct {
		ID       string `json:"id"`
		NamaAsli string `json:"nama_asli"`
		MimeType string `json:"mime_type"`
		Ukuran   int64  `json:"ukuran"`
	}

	VersiResponse struct {
		Versi        int             `json:"versi"`
		Catatan      string          `json:"catatan"`
		DiunggahOleh string          `json:"diunggah_oleh"`
		Dokumen      *DokumenRingkas `json:"dokumen,omitempty"`
		Lampiran     *DokumenRingkas `json:"lampiran,omitempty"`
		URL          string          `json:"url"`
		CreatedAt    time.Time       `json:"created_at"`
	}

	ArsipResponse struct {
		ID             string          `json:"id"`
		PendidikID     string          `json:"pendidik_id"`
		PendidikNama   string          `json:"pendidik_nama"`
		PendidikNIP    *string         `json:"pendidik_nip"`
		SekolahID      string          `json:"sekolah_id"`
		SekolahNama    string          `json:"sekolah_nama"`
		Kategori       string          `json:"kategori"`
		JenisDokumen   string          `json:"jenis_dokumen"`
		Nomor          string          `json:"nomor"`
		TanggalDokumen *string         `json:"tanggal_dokumen"`
		Keterangan     string          `json:"keterangan"`
		VersiTerkini   int             `json:"versi_terkini"`
		SumberModul    string          `json:"sumber_modul"`
		DiunggahOleh   string          `json:"diunggah_oleh"`
		URL            string          `json:"url"`
		Versi          []VersiResponse `json:"versi,omitempty"`
		UpdatedAt      time.Time       `json:"updated_at"`
	}

	RingkasanResponse struct {
		Total       int64            `json:"total"`
		PerKategori map[string]int64 `json:"per_kategori"`
	}

	LogResponse struct {
		Waktu        time.Time `json:"waktu"`
		Aksi         string    `json:"aksi"`
		Versi        int       `json:"versi"`
		Keterangan   string    `json:"keterangan"`
		Pengguna     string    `json:"pengguna"`
		Peran        string    `json:"peran"`
		ArsipID      string    `json:"arsip_id"`
		JenisDokumen string    `json:"jenis_dokumen"`
		Nomor        string    `json:"nomor"`
		PendidikNama string    `json:"pendidik_nama"`
	}
)

func dokumen(d *entities.Dokumen) *DokumenRingkas {
	if d == nil {
		return nil
	}
	return &DokumenRingkas{ID: d.ID.String(), NamaAsli: d.NamaAsli, MimeType: d.MimeType, Ukuran: d.Ukuran}
}

// Pengunggah: nama akun, atau "Sistem · modul X" untuk dokumen yang diarsipkan otomatis di versi 1.
func Pengunggah(a entities.Arsip, v entities.ArsipVersi) string {
	if a.SumberModul != "" && v.Versi == 1 {
		return "Sistem · modul " + a.SumberModul
	}
	if v.Pengunggah != nil {
		return v.Pengunggah.Name
	}
	return ""
}

func FromEntity(a entities.Arsip, denganVersi bool) ArsipResponse {
	r := ArsipResponse{
		ID: a.ID.String(), PendidikID: a.PendidikID.String(), SekolahID: a.SekolahID.String(),
		Kategori: a.Kategori, JenisDokumen: a.JenisDokumen, Nomor: a.Nomor, Keterangan: a.Keterangan,
		VersiTerkini: a.VersiTerkini, SumberModul: a.SumberModul, UpdatedAt: a.UpdatedAt,
		URL: "/api/arsip/" + a.ID.String() + "/unduh",
	}
	if a.TanggalDokumen != nil {
		s := a.TanggalDokumen.Format("2006-01-02")
		r.TanggalDokumen = &s
	}
	if a.Pendidik != nil {
		r.PendidikNama, r.PendidikNIP = a.Pendidik.Nama, a.Pendidik.NIP
	}
	if a.Sekolah != nil {
		r.SekolahNama = a.Sekolah.Nama
	}
	for _, v := range a.Versi { // diurutkan versi terbaru dulu oleh repository
		if v.Versi == a.VersiTerkini {
			r.DiunggahOleh = Pengunggah(a, v)
		}
		if denganVersi {
			r.Versi = append(r.Versi, VersiResponse{
				Versi: v.Versi, Catatan: v.Catatan, DiunggahOleh: Pengunggah(a, v),
				Dokumen: dokumen(v.Dokumen), Lampiran: dokumen(v.LampiranDokumen), CreatedAt: v.CreatedAt,
				URL: "/api/arsip/" + a.ID.String() + "/unduh?versi=" + strconv.Itoa(v.Versi),
			})
		}
	}
	return r
}
