package dto

import (
	"mime/multipart"
	"strings"
	"time"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"github.com/Caknoooo/go-gin-clean-starter/modules/anjab/hitung"
)

const TglFormat = "2006-01-02"

type (
	BebanItem struct {
		Jenjang string `json:"jenjang" binding:"required,oneof=SD SMP SMA SMK"`
		Mapel   string `json:"mapel" binding:"required,max=100"`
		Tingkat int    `json:"tingkat" binding:"required,min=1,max=12"`
		JP      int    `json:"jp" binding:"min=0,max=40"`
	}

	// PeriodeRequest dipakai untuk membuat dan mengubah periode beserta parameternya (A1).
	PeriodeRequest struct {
		TahunAjaran     string      `json:"tahun_ajaran" binding:"required,len=9"`
		Jenjang         []string    `json:"jenjang" binding:"required,min=1,max=4,dive,oneof=SD SMP SMA SMK"`
		TanggalBuka     string      `json:"tanggal_buka" binding:"required,datetime=2006-01-02"`
		TanggalTutup    string      `json:"tanggal_tutup" binding:"required,datetime=2006-01-02"`
		AcuanPeraturan  string      `json:"acuan_peraturan" binding:"max=200"`
		JamWajib        int         `json:"jam_wajib" binding:"required,min=1,max=60"`
		RasioSD         int         `json:"rasio_sd" binding:"required,min=1,max=100"`
		RasioSMP        int         `json:"rasio_smp" binding:"required,min=1,max=100"`
		RasioSMA        int         `json:"rasio_sma" binding:"required,min=1,max=100"`
		BatasTMTPensiun string      `json:"batas_tmt_pensiun" binding:"required,datetime=2006-01-02"`
		Beban           []BebanItem `json:"beban" binding:"max=600,dive"`
	}

	TenggatRequest struct {
		TanggalTutup string `json:"tanggal_tutup" binding:"required,datetime=2006-01-02"`
	}

	ImporRequest struct {
		Berkas *multipart.FileHeader `form:"berkas" binding:"required"`
		Simpan bool                  `form:"simpan"`
	}

	RombelItem struct {
		Tingkat      int `json:"tingkat" binding:"required,min=1,max=12"`
		JumlahRombel int `json:"jumlah_rombel" binding:"min=0,max=200"`
		JumlahSiswa  int `json:"jumlah_siswa" binding:"min=0,max=20000"`
	}
	RombelRequest struct {
		Rombel []RombelItem `json:"rombel" binding:"required,min=1,max=12,dive"`
	}

	GuruTambahRequest struct {
		PendidikID string `json:"pendidik_id" binding:"required,uuid"`
		MapelSK    string `json:"mapel_sk" binding:"required,max=100"`
		MapelRiil  string `json:"mapel_riil" binding:"max=100"`
		JPRiil     int    `json:"jp_riil" binding:"min=0,max=60"`
	}
	// GuruUbahRequest: operator hanya mengubah kondisi riil; mapel SK mengikuti SK (lewat Sudin).
	GuruUbahRequest struct {
		MapelRiil string `json:"mapel_riil" binding:"required,max=100"`
		JPRiil    int    `json:"jp_riil" binding:"min=0,max=60"`
	}

	TinjauRequest struct {
		Keputusan string `json:"keputusan" binding:"required,oneof=ditetapkan dikembalikan"`
		Catatan   string `json:"catatan" binding:"max=2000"`
	}

	Filter struct {
		PeriodeID string
		Status    []string
		Kecamatan string
		Search    string
		Offset    int
		Limit     int
	}
)

type (
	BebanResponse struct {
		Jenjang string `json:"jenjang"`
		Mapel   string `json:"mapel"`
		Tingkat int    `json:"tingkat"`
		JP      int    `json:"jp"`
	}

	PeriodeResponse struct {
		ID              string           `json:"id"`
		SudinID         string           `json:"sudin_id"`
		TahunAjaran     string           `json:"tahun_ajaran"`
		Jenjang         []string         `json:"jenjang"`
		TanggalBuka     string           `json:"tanggal_buka"`
		TanggalTutup    string           `json:"tanggal_tutup"`
		AcuanPeraturan  string           `json:"acuan_peraturan"`
		JamWajib        int              `json:"jam_wajib"`
		RasioSD         int              `json:"rasio_sd"`
		RasioSMP        int              `json:"rasio_smp"`
		RasioSMA        int              `json:"rasio_sma"`
		BatasTMTPensiun string           `json:"batas_tmt_pensiun"`
		Terkunci        bool             `json:"terkunci"`
		DibukaAt        *time.Time       `json:"dibuka_at"`
		Beban           []BebanResponse  `json:"beban,omitempty"`
		Progres         map[string]int64 `json:"progres,omitempty"`
		CreatedAt       time.Time        `json:"created_at"`
	}

	SekolahRingkas struct {
		ID        string `json:"id"`
		NPSN      string `json:"npsn"`
		Nama      string `json:"nama"`
		Jenjang   string `json:"jenjang"`
		Kecamatan string `json:"kecamatan"`
	}

	RombelResponse struct {
		Tingkat      int `json:"tingkat"`
		JumlahRombel int `json:"jumlah_rombel"`
		JumlahSiswa  int `json:"jumlah_siswa"`
	}

	GuruResponse struct {
		ID                string  `json:"id"`
		PendidikID        string  `json:"pendidik_id"`
		Nama              string  `json:"nama"`
		NIP               *string `json:"nip"`
		StatusKepegawaian string  `json:"status_kepegawaian"`
		Golongan          string  `json:"golongan"`
		MapelSK           string  `json:"mapel_sk"`
		MapelRiil         string  `json:"mapel_riil"`
		JPRiil            int     `json:"jp_riil"`
		TMTPensiun        *string `json:"tmt_pensiun"`
		Dihitung          bool    `json:"dihitung"`
	}

	RiwayatResponse struct {
		DariStatus string    `json:"dari_status"`
		KeStatus   string    `json:"ke_status"`
		OlehNama   string    `json:"oleh_nama"`
		Catatan    string    `json:"catatan"`
		CreatedAt  time.Time `json:"created_at"`
	}

	AnjabSekolahResponse struct {
		ID           string         `json:"id"`
		PeriodeID    string         `json:"periode_id"`
		TahunAjaran  string         `json:"tahun_ajaran"`
		Tenggat      string         `json:"tenggat"`
		Sekolah      SekolahRingkas `json:"sekolah"`
		Status       string         `json:"status"`
		KirimanKe    int            `json:"kiriman_ke"`
		DikirimAt    *time.Time     `json:"dikirim_at"`
		CatatanSudin string         `json:"catatan_sudin"`
		DitinjauAt   *time.Time     `json:"ditinjau_at"`
		JumlahRombel int            `json:"jumlah_rombel"`
		JumlahGuru   int            `json:"jumlah_guru"`

		Rombel  []RombelResponse  `json:"rombel,omitempty"`
		Guru    []GuruResponse    `json:"guru,omitempty"`
		Hasil   *hitung.Hasil     `json:"hasil,omitempty"`
		Riwayat []RiwayatResponse `json:"riwayat,omitempty"`
	}

	GalatImpor struct {
		Baris   int    `json:"baris"`
		Lembar  string `json:"lembar"`
		Kolom   string `json:"kolom"`
		Nilai   string `json:"nilai"`
		Masalah string `json:"masalah"`
	}
	ImporResponse struct {
		NamaBerkas      string         `json:"nama_berkas"`
		TotalBaris      int            `json:"total_baris"`
		BarisValid      int            `json:"baris_valid"`
		BarisBermasalah int            `json:"baris_bermasalah"`
		PerLembar       map[string]int `json:"valid_per_lembar"`
		Galat           []GalatImpor   `json:"galat"`
		Tersimpan       bool           `json:"tersimpan"`
	}

	MonitoringSekolah struct {
		AnjabSekolahID string         `json:"anjab_sekolah_id"`
		Sekolah        SekolahRingkas `json:"sekolah"`
		Status         string         `json:"status"`
		Kurang         int            `json:"kurang"`
		Lebih          int            `json:"lebih"`
	}
	MonitoringResponse struct {
		PeriodeID         string               `json:"periode_id"`
		TahunAjaran       string               `json:"tahun_ajaran"`
		TotalSekolah      int                  `json:"total_sekolah"`
		SekolahDitetapkan int                  `json:"sekolah_ditetapkan"`
		KekuranganGuru    int                  `json:"kekurangan_guru"`
		KelebihanGuru     int                  `json:"kelebihan_guru"`
		Rekomendasi       []hitung.Rekomendasi `json:"rekomendasi"`
		Sekolah           []MonitoringSekolah  `json:"sekolah"`
	}
)

func tgl(t time.Time) string { return t.Format(TglFormat) }

func PeriodeFromEntity(p entities.AnjabPeriode) PeriodeResponse {
	r := PeriodeResponse{
		ID: p.ID.String(), SudinID: p.SudinID.String(), TahunAjaran: p.TahunAjaran,
		Jenjang: strings.Split(p.Jenjang, ","), TanggalBuka: tgl(p.TanggalBuka), TanggalTutup: tgl(p.TanggalTutup),
		AcuanPeraturan: p.AcuanPeraturan, JamWajib: p.JamWajib,
		RasioSD: p.RasioSD, RasioSMP: p.RasioSMP, RasioSMA: p.RasioSMA,
		BatasTMTPensiun: tgl(p.BatasTMTPensiun), Terkunci: p.DibukaAt != nil, DibukaAt: p.DibukaAt,
		CreatedAt: p.CreatedAt,
	}
	for _, b := range p.Beban {
		r.Beban = append(r.Beban, BebanResponse{Jenjang: b.Jenjang, Mapel: b.Mapel, Tingkat: b.Tingkat, JP: b.JP})
	}
	return r
}

func SekolahFromEntity(s *entities.Sekolah) SekolahRingkas {
	if s == nil {
		return SekolahRingkas{}
	}
	return SekolahRingkas{ID: s.ID.String(), NPSN: s.NPSN, Nama: s.Nama, Jenjang: s.Jenjang, Kecamatan: s.Kecamatan}
}

// AnjabSekolahFromEntity mengisi field ringkas; rombel/guru/hasil diisi service untuk detail.
func AnjabSekolahFromEntity(a entities.AnjabSekolah, jumlahRombel, jumlahGuru int) AnjabSekolahResponse {
	r := AnjabSekolahResponse{
		ID: a.ID.String(), PeriodeID: a.PeriodeID.String(), Sekolah: SekolahFromEntity(a.Sekolah),
		Status: a.Status, KirimanKe: a.KirimanKe, DikirimAt: a.DikirimAt, CatatanSudin: a.CatatanSudin,
		DitinjauAt: a.DitinjauAt, JumlahRombel: jumlahRombel, JumlahGuru: jumlahGuru,
	}
	if a.Periode != nil {
		r.TahunAjaran = a.Periode.TahunAjaran
		r.Tenggat = tgl(a.Periode.TanggalTutup)
	}
	for _, h := range a.Riwayat {
		item := RiwayatResponse{DariStatus: h.DariStatus, KeStatus: h.KeStatus, Catatan: h.Catatan, CreatedAt: h.CreatedAt}
		if h.Pengguna != nil {
			item.OlehNama = h.Pengguna.Name
		}
		r.Riwayat = append(r.Riwayat, item)
	}
	return r
}
