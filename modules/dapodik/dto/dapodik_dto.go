package dto

import (
	"mime/multipart"
	"time"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
)

const tglFormat = "2006-01-02"

type (
	// AjukanRequest dikirim operator sebagai multipart/form-data (DP1 + DP2).
	// isian berupa string JSON objek, mis. {"nama_operator_baru":"Rizky"}.
	AjukanRequest struct {
		Jenis      string                  `form:"jenis" binding:"required,oneof=revisi_akun_operator autentikasi_ptk kode_referral_yayasan perbaikan_data_ptk lainnya"`
		PendidikID string                  `form:"pendidik_id" binding:"omitempty,uuid"`
		Isian      map[string]string       `form:"isian" binding:"max=20,dive,keys,max=50,endkeys,max=1000"`
		Uraian     string                  `form:"uraian" binding:"required,max=2000"`
		Lampiran   []*multipart.FileHeader `form:"lampiran" binding:"max=5"`
	}

	// LengkapiRequest dikirim operator saat status perlu_info_tambahan (DP5).
	LengkapiRequest struct {
		Balasan  string                  `form:"balasan" binding:"required,max=2000"`
		Lampiran []*multipart.FileHeader `form:"lampiran" binding:"max=5"`
	}

	// TinjauRequest adalah keputusan staf Sudin (DP4).
	TinjauRequest struct {
		Keputusan string `json:"keputusan" binding:"required,oneof=selesai perlu_info_tambahan"`
		Catatan   string `json:"catatan" binding:"required,max=2000"`
	}

	Filter struct {
		Status    []string
		Jenis     string
		SekolahID string
		Search    string
		Urut      string // "tenggat" = tenggat terdekat dulu; default terbaru dulu
		Offset    int
		Limit     int
	}
)

type (
	SekolahRingkas struct {
		ID        string `json:"id"`
		NPSN      string `json:"npsn"`
		Nama      string `json:"nama"`
		Kecamatan string `json:"kecamatan"`
	}

	PendidikRingkas struct {
		ID                string  `json:"id"`
		Nama              string  `json:"nama"`
		NIP               *string `json:"nip"`
		StatusKepegawaian string  `json:"status_kepegawaian"`
		Jabatan           string  `json:"jabatan"`
	}

	DokumenRingkas struct {
		ID       string `json:"id"`
		NamaAsli string `json:"nama_asli"`
		MimeType string `json:"mime_type"`
		Ukuran   int64  `json:"ukuran"`
		URL      string `json:"url"`
	}

	LampiranResponse struct {
		Tahap   string          `json:"tahap"`
		Dokumen *DokumenRingkas `json:"dokumen"`
	}

	RiwayatResponse struct {
		DariStatus string    `json:"dari_status"`
		KeStatus   string    `json:"ke_status"`
		Oleh       string    `json:"oleh"`
		OlehNama   string    `json:"oleh_nama"`
		Catatan    string    `json:"catatan"`
		CreatedAt  time.Time `json:"created_at"`
	}

	TiketResponse struct {
		ID                  string             `json:"id"`
		Nomor               string             `json:"nomor"`
		Jenis               string             `json:"jenis"`
		Status              string             `json:"status"`
		Isian               map[string]string  `json:"isian"`
		Uraian              string             `json:"uraian"`
		Tenggat             string             `json:"tenggat"`
		Terlambat           bool               `json:"terlambat"`
		PengajuNama         string             `json:"pengaju_nama,omitempty"`
		Sekolah             *SekolahRingkas    `json:"sekolah,omitempty"`
		Pendidik            *PendidikRingkas   `json:"pendidik,omitempty"`
		PermintaanInfo      string             `json:"permintaan_info"`
		BalasanInfo         string             `json:"balasan_info"`
		CatatanPenyelesaian string             `json:"catatan_penyelesaian"`
		PenyelesaiNama      string             `json:"penyelesai_nama,omitempty"`
		DiselesaikanAt      *time.Time         `json:"diselesaikan_at"`
		Lampiran            []LampiranResponse `json:"lampiran,omitempty"`
		Riwayat             []RiwayatResponse  `json:"riwayat,omitempty"`
		CreatedAt           time.Time          `json:"created_at"`
		UpdatedAt           time.Time          `json:"updated_at"`
	}
)

func dokumen(d *entities.Dokumen) *DokumenRingkas {
	if d == nil {
		return nil
	}
	return &DokumenRingkas{
		ID: d.ID.String(), NamaAsli: d.NamaAsli, MimeType: d.MimeType, Ukuran: d.Ukuran,
		URL: "/api/dokumen/" + d.ID.String(),
	}
}

func FromEntity(t entities.DapodikTiket) TiketResponse {
	hariIni := time.Now().Format(tglFormat)
	r := TiketResponse{
		ID: t.ID.String(), Nomor: t.Nomor, Jenis: t.Jenis, Status: t.Status,
		Isian: t.Isian, Uraian: t.Uraian, Tenggat: t.Tenggat.Format(tglFormat),
		Terlambat:           t.Status != entities.DapodikStatusSelesai && t.Tenggat.Format(tglFormat) < hariIni,
		PermintaanInfo:      t.PermintaanInfo,
		BalasanInfo:         t.BalasanInfo,
		CatatanPenyelesaian: t.CatatanPenyelesaian,
		DiselesaikanAt:      t.DiselesaikanAt,
		CreatedAt:           t.CreatedAt,
		UpdatedAt:           t.UpdatedAt,
	}
	if r.Isian == nil {
		r.Isian = map[string]string{}
	}
	if t.Pengaju != nil {
		r.PengajuNama = t.Pengaju.Name
	}
	if t.Penyelesai != nil {
		r.PenyelesaiNama = t.Penyelesai.Name
	}
	if s := t.Sekolah; s != nil {
		r.Sekolah = &SekolahRingkas{ID: s.ID.String(), NPSN: s.NPSN, Nama: s.Nama, Kecamatan: s.Kecamatan}
	}
	if p := t.Pendidik; p != nil {
		r.Pendidik = &PendidikRingkas{ID: p.ID.String(), Nama: p.Nama, NIP: p.NIP, StatusKepegawaian: p.StatusKepegawaian, Jabatan: p.Jabatan}
	}
	for _, l := range t.Lampiran {
		r.Lampiran = append(r.Lampiran, LampiranResponse{Tahap: l.Tahap, Dokumen: dokumen(l.Dokumen)})
	}
	for _, h := range t.Riwayat {
		item := RiwayatResponse{
			DariStatus: h.DariStatus, KeStatus: h.KeStatus, Oleh: h.Oleh.String(),
			Catatan: h.Catatan, CreatedAt: h.CreatedAt,
		}
		if h.Pengguna != nil {
			item.OlehNama = h.Pengguna.Name
		}
		r.Riwayat = append(r.Riwayat, item)
	}
	return r
}
