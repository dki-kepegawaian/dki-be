package dto

import (
	"mime/multipart"
	"time"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
)

const tglFormat = "2006-01-02"

type (
	// AjukanRequest dikirim sebagai multipart/form-data (C1).
	AjukanRequest struct {
		PendidikID      string                `form:"pendidik_id" binding:"required,uuid"`
		Jenis           string                `form:"jenis" binding:"required,oneof=tahunan besar sakit melahirkan alasan_penting di_luar_tanggungan_negara"`
		TanggalMulai    string                `form:"tanggal_mulai" binding:"required,datetime=2006-01-02"`
		TanggalSelesai  string                `form:"tanggal_selesai" binding:"required,datetime=2006-01-02"`
		Alasan          string                `form:"alasan" binding:"required,max=1000"`
		DisetujuiKepsek bool                  `form:"disetujui_kepsek"`
		Berkas          *multipart.FileHeader `form:"berkas" binding:"required"`
	}

	// PerbaikiRequest dikirim operator saat status perlu_perbaikan (C6 > Perbaiki).
	PerbaikiRequest struct {
		Alasan string                `form:"alasan" binding:"omitempty,max=1000"`
		Berkas *multipart.FileHeader `form:"berkas" binding:"required"`
	}

	// PeriksaRequest adalah hasil pemeriksaan berkas digital oleh Sudin (C3).
	PeriksaRequest struct {
		BerkasLengkap *bool  `json:"berkas_lengkap" binding:"required"`
		BerkasTerbaca *bool  `json:"berkas_terbaca" binding:"required"`
		Catatan       string `json:"catatan" binding:"max=1000"`
	}

	// TerimaSuratFisikRequest menandai beberapa pengajuan sekaligus (C4).
	TerimaSuratFisikRequest struct {
		CutiIDs         []string `json:"cuti_ids" binding:"required,min=1,max=100,dive,uuid"`
		TanggalDiterima string   `json:"tanggal_diterima" binding:"required,datetime=2006-01-02"`
		PenerimaID      string   `json:"penerima_id" binding:"omitempty,uuid"`
		NotaDinasAda    *bool    `json:"nota_dinas_ada" binding:"required"`
		Catatan         string   `json:"catatan" binding:"max=1000"`
	}

	// KeputusanRequest dikirim sebagai multipart/form-data (C5).
	KeputusanRequest struct {
		Keputusan        string                `form:"keputusan" binding:"required,oneof=disetujui ditolak"`
		NomorSuratIzin   string                `form:"nomor_surat_izin" binding:"required,max=100"`
		TanggalSuratIzin string                `form:"tanggal_surat_izin" binding:"required,datetime=2006-01-02"`
		Catatan          string                `form:"catatan" binding:"max=1000"`
		SuratIzin        *multipart.FileHeader `form:"surat_izin" binding:"required"`
	}

	Filter struct {
		Status    []string
		Jenis     string
		SekolahID string
		Search    string
		Offset    int
		Limit     int
	}
)

type (
	PendidikRingkas struct {
		ID                string  `json:"id"`
		Nama              string  `json:"nama"`
		NIP               *string `json:"nip"`
		StatusKepegawaian string  `json:"status_kepegawaian"`
		Pangkat           string  `json:"pangkat"`
		Golongan          string  `json:"golongan"`
		Jabatan           string  `json:"jabatan"`
	}

	SekolahRingkas struct {
		ID   string `json:"id"`
		NPSN string `json:"npsn"`
		Nama string `json:"nama"`
	}

	DokumenRingkas struct {
		ID       string `json:"id"`
		NamaAsli string `json:"nama_asli"`
		MimeType string `json:"mime_type"`
		Ukuran   int64  `json:"ukuran"`
		URL      string `json:"url"`
	}

	RiwayatResponse struct {
		DariStatus string    `json:"dari_status"`
		KeStatus   string    `json:"ke_status"`
		Oleh       string    `json:"oleh"`
		OlehNama   string    `json:"oleh_nama"`
		Catatan    string    `json:"catatan"`
		CreatedAt  time.Time `json:"created_at"`
	}

	CutiResponse struct {
		ID               string           `json:"id"`
		Nomor            string           `json:"nomor"`
		Status           string           `json:"status"`
		Jenis            string           `json:"jenis"`
		TanggalMulai     string           `json:"tanggal_mulai"`
		TanggalSelesai   string           `json:"tanggal_selesai"`
		HariKerja        int              `json:"hari_kerja"`
		Alasan           string           `json:"alasan"`
		DisetujuiKepsek  bool             `json:"disetujui_kepsek"`
		Pendidik         *PendidikRingkas `json:"pendidik,omitempty"`
		Sekolah          *SekolahRingkas  `json:"sekolah,omitempty"`
		BerkasPermohonan *DokumenRingkas  `json:"berkas_permohonan,omitempty"`

		BerkasLengkap    *bool      `json:"berkas_lengkap"`
		BerkasTerbaca    *bool      `json:"berkas_terbaca"`
		CatatanPerbaikan string     `json:"catatan_perbaikan"`
		DiperiksaAt      *time.Time `json:"diperiksa_at"`

		SuratFisikDiterimaTanggal *string `json:"surat_fisik_diterima_tanggal"`
		SuratFisikPenerimaNama    string  `json:"surat_fisik_penerima_nama,omitempty"`
		NotaDinasAda              *bool   `json:"nota_dinas_ada"`
		CatatanPenerimaan         string  `json:"catatan_penerimaan"`

		NomorSuratIzin   string          `json:"nomor_surat_izin"`
		TanggalSuratIzin *string         `json:"tanggal_surat_izin"`
		SuratIzin        *DokumenRingkas `json:"surat_izin,omitempty"`
		CatatanKeputusan string          `json:"catatan_keputusan"`
		DiputuskanAt     *time.Time      `json:"diputuskan_at"`

		Riwayat   []RiwayatResponse `json:"riwayat,omitempty"`
		CreatedAt time.Time         `json:"created_at"`
		UpdatedAt time.Time         `json:"updated_at"`
	}

	SisaTahunanResponse struct {
		PendidikID string `json:"pendidik_id"`
		Tahun      int    `json:"tahun"`
		Jatah      int    `json:"jatah"`
		Terpakai   int    `json:"terpakai"`
		Sisa       int    `json:"sisa"`
	}

	TerimaSuratFisikResponse struct {
		Diterima []string `json:"diterima"`
	}
)

func tglPtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(tglFormat)
	return &s
}

func dokumen(d *entities.Dokumen) *DokumenRingkas {
	if d == nil {
		return nil
	}
	return &DokumenRingkas{
		ID:       d.ID.String(),
		NamaAsli: d.NamaAsli,
		MimeType: d.MimeType,
		Ukuran:   d.Ukuran,
		URL:      "/api/dokumen/" + d.ID.String(),
	}
}

func FromEntity(c entities.Cuti) CutiResponse {
	r := CutiResponse{
		ID:                        c.ID.String(),
		Nomor:                     c.Nomor,
		Status:                    c.Status,
		Jenis:                     c.Jenis,
		TanggalMulai:              c.TanggalMulai.Format(tglFormat),
		TanggalSelesai:            c.TanggalSelesai.Format(tglFormat),
		HariKerja:                 c.HariKerja,
		Alasan:                    c.Alasan,
		DisetujuiKepsek:           c.DisetujuiKepsek,
		BerkasPermohonan:          dokumen(c.BerkasPermohonan),
		BerkasLengkap:             c.BerkasLengkap,
		BerkasTerbaca:             c.BerkasTerbaca,
		CatatanPerbaikan:          c.CatatanPerbaikan,
		DiperiksaAt:               c.DiperiksaAt,
		SuratFisikDiterimaTanggal: tglPtr(c.SuratFisikDiterimaTanggal),
		NotaDinasAda:              c.NotaDinasAda,
		CatatanPenerimaan:         c.CatatanPenerimaan,
		NomorSuratIzin:            c.NomorSuratIzin,
		TanggalSuratIzin:          tglPtr(c.TanggalSuratIzin),
		SuratIzin:                 dokumen(c.SuratIzin),
		CatatanKeputusan:          c.CatatanKeputusan,
		DiputuskanAt:              c.DiputuskanAt,
		CreatedAt:                 c.CreatedAt,
		UpdatedAt:                 c.UpdatedAt,
	}
	if p := c.Pendidik; p != nil {
		r.Pendidik = &PendidikRingkas{
			ID: p.ID.String(), Nama: p.Nama, NIP: p.NIP, StatusKepegawaian: p.StatusKepegawaian,
			Pangkat: p.Pangkat, Golongan: p.Golongan, Jabatan: p.Jabatan,
		}
	}
	if s := c.Sekolah; s != nil {
		r.Sekolah = &SekolahRingkas{ID: s.ID.String(), NPSN: s.NPSN, Nama: s.Nama}
	}
	if c.PenerimaSurat != nil {
		r.SuratFisikPenerimaNama = c.PenerimaSurat.Name
	}
	for _, h := range c.Riwayat {
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
