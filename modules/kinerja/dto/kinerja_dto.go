package dto

import (
	"fmt"
	"mime/multipart"
	"time"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
)

type (
	// Nilai BerAKHLAK skala 1–5. Field kosong (null) berarti belum diisi.
	NilaiInput struct {
		BerorientasiPelayanan *int `json:"berorientasi_pelayanan" binding:"omitempty,min=1,max=5"`
		Akuntabel             *int `json:"akuntabel" binding:"omitempty,min=1,max=5"`
		Kompeten              *int `json:"kompeten" binding:"omitempty,min=1,max=5"`
		Harmonis              *int `json:"harmonis" binding:"omitempty,min=1,max=5"`
		Loyal                 *int `json:"loyal" binding:"omitempty,min=1,max=5"`
		Adaptif               *int `json:"adaptif" binding:"omitempty,min=1,max=5"`
		Kolaboratif           *int `json:"kolaboratif" binding:"omitempty,min=1,max=5"`
	}

	// BuatRequest membuat draf penilaian (KP1 > Nilai, KP2 > Simpan draf).
	BuatRequest struct {
		PendidikID string     `json:"pendidik_id" binding:"required,uuid"`
		Tahun      int        `json:"tahun" binding:"required,min=2020,max=2100"`
		Semester   int        `json:"semester" binding:"required,oneof=1 2"`
		Nilai      NilaiInput `json:"nilai"`
	}

	// UbahNilaiRequest mengganti seluruh nilai draf (KP2 > Simpan draf).
	UbahNilaiRequest struct {
		Nilai NilaiInput `json:"nilai"`
	}

	// FinalRequest dikirim staf Sudin sebagai multipart/form-data (KP4).
	FinalRequest struct {
		Berkas     *multipart.FileHeader `form:"berkas" binding:"required"`
		TTDJelas   bool                  `form:"ttd_jelas"`
		DataSesuai bool                  `form:"data_sesuai"`
	}

	Filter struct {
		Status    []string
		Tahun     int
		Semester  int
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
		Jabatan           string  `json:"jabatan"`
	}

	SekolahRingkas struct {
		ID        string `json:"id"`
		NPSN      string `json:"npsn"`
		Nama      string `json:"nama"`
		Kecamatan string `json:"kecamatan"`
	}

	DokumenRingkas struct {
		ID       string `json:"id"`
		NamaAsli string `json:"nama_asli"`
		MimeType string `json:"mime_type"`
		Ukuran   int64  `json:"ukuran"`
		URL      string `json:"url"`
	}

	NilaiResponse struct {
		BerorientasiPelayanan *int `json:"berorientasi_pelayanan"`
		Akuntabel             *int `json:"akuntabel"`
		Kompeten              *int `json:"kompeten"`
		Harmonis              *int `json:"harmonis"`
		Loyal                 *int `json:"loyal"`
		Adaptif               *int `json:"adaptif"`
		Kolaboratif           *int `json:"kolaboratif"`
	}

	RiwayatResponse struct {
		DariStatus string    `json:"dari_status"`
		KeStatus   string    `json:"ke_status"`
		Oleh       string    `json:"oleh"`
		OlehNama   string    `json:"oleh_nama"`
		Catatan    string    `json:"catatan"`
		CreatedAt  time.Time `json:"created_at"`
	}

	PenilaianResponse struct {
		ID              string            `json:"id"`
		Nomor           string            `json:"nomor"`
		Status          string            `json:"status"`
		Tahun           int               `json:"tahun"`
		Semester        int               `json:"semester"`
		Periode         string            `json:"periode"`
		Pendidik        *PendidikRingkas  `json:"pendidik,omitempty"`
		Sekolah         *SekolahRingkas   `json:"sekolah,omitempty"`
		Nilai           NilaiResponse     `json:"nilai"`
		ParameterTerisi int               `json:"parameter_terisi"`
		NilaiAkhir      *float64          `json:"nilai_akhir"`
		Predikat        string            `json:"predikat"`
		DokumenSistem   *DokumenRingkas   `json:"dokumen_sistem,omitempty"`
		DokumenDibuatAt *time.Time        `json:"dokumen_dibuat_at"`
		DokumenTTD      *DokumenRingkas   `json:"dokumen_ttd,omitempty"`
		DifinalkanAt    *time.Time        `json:"difinalkan_at"`
		Riwayat         []RiwayatResponse `json:"riwayat,omitempty"`
		CreatedAt       time.Time         `json:"created_at"`
		UpdatedAt       time.Time         `json:"updated_at"`
	}

	PenilaianRingkas struct {
		ID         string   `json:"id"`
		Nomor      string   `json:"nomor"`
		Status     string   `json:"status"`
		NilaiAkhir *float64 `json:"nilai_akhir"`
		Predikat   string   `json:"predikat"`
	}

	// PegawaiResponse adalah satu baris daftar pegawai PPPK PW untuk satu periode (KP1).
	PegawaiResponse struct {
		Pendidik  PendidikRingkas   `json:"pendidik"`
		Sekolah   *SekolahRingkas   `json:"sekolah,omitempty"`
		Penilaian *PenilaianRingkas `json:"penilaian"`
	}
)

// Periode mengembalikan label periode, mis. "Semester 2 · Juli–Desember 2026".
func Periode(tahun, semester int) string {
	bulan := "Januari–Juni"
	if semester == 2 {
		bulan = "Juli–Desember"
	}
	return fmt.Sprintf("Semester %d · %s %d", semester, bulan, tahun)
}

func dokumen(d *entities.Dokumen) *DokumenRingkas {
	if d == nil {
		return nil
	}
	return &DokumenRingkas{
		ID: d.ID.String(), NamaAsli: d.NamaAsli, MimeType: d.MimeType, Ukuran: d.Ukuran,
		URL: "/api/dokumen/" + d.ID.String(),
	}
}

func pendidik(p *entities.Pendidik) *PendidikRingkas {
	if p == nil {
		return nil
	}
	return &PendidikRingkas{ID: p.ID.String(), Nama: p.Nama, NIP: p.NIP, StatusKepegawaian: p.StatusKepegawaian, Jabatan: p.Jabatan}
}

func sekolah(s *entities.Sekolah) *SekolahRingkas {
	if s == nil {
		return nil
	}
	return &SekolahRingkas{ID: s.ID.String(), NPSN: s.NPSN, Nama: s.Nama, Kecamatan: s.Kecamatan}
}

func FromEntity(k entities.KinerjaPenilaian, terisi int) PenilaianResponse {
	r := PenilaianResponse{
		ID: k.ID.String(), Nomor: k.Nomor, Status: k.Status, Tahun: k.Tahun, Semester: k.Semester,
		Periode:  Periode(k.Tahun, k.Semester),
		Pendidik: pendidik(k.Pendidik),
		Sekolah:  sekolah(k.Sekolah),
		Nilai: NilaiResponse{
			BerorientasiPelayanan: k.BerorientasiPelayanan, Akuntabel: k.Akuntabel, Kompeten: k.Kompeten,
			Harmonis: k.Harmonis, Loyal: k.Loyal, Adaptif: k.Adaptif, Kolaboratif: k.Kolaboratif,
		},
		ParameterTerisi: terisi,
		NilaiAkhir:      k.NilaiAkhir,
		Predikat:        k.Predikat,
		DokumenSistem:   dokumen(k.DokumenSistem),
		DokumenDibuatAt: k.DokumenDibuatAt,
		DokumenTTD:      dokumen(k.DokumenTTD),
		DifinalkanAt:    k.DifinalkanAt,
		CreatedAt:       k.CreatedAt,
		UpdatedAt:       k.UpdatedAt,
	}
	for _, h := range k.Riwayat {
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

func FromPegawai(p entities.Pendidik, k *entities.KinerjaPenilaian) PegawaiResponse {
	r := PegawaiResponse{Pendidik: *pendidik(&p), Sekolah: sekolah(p.Sekolah)}
	if k != nil {
		r.Penilaian = &PenilaianRingkas{ID: k.ID.String(), Nomor: k.Nomor, Status: k.Status, NilaiAkhir: k.NilaiAkhir, Predikat: k.Predikat}
	}
	return r
}
