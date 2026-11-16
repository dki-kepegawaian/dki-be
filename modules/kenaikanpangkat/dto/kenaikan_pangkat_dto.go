package dto

import (
	"mime/multipart"
	"time"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/persyaratan"
)

const tglFormat = "2006-01-02"

type (
	// UnggahBerkasRequest: satu berkas persyaratan (PDF, maks 2 MB) sebelum diajukan (K2).
	UnggahBerkasRequest struct {
		Berkas *multipart.FileHeader `form:"berkas" binding:"required"`
	}

	// AjukanRequest (K2). Berkas: kode jenis -> id dokumen dari POST /api/kenaikan-pangkat/berkas.
	AjukanRequest struct {
		PendidikID string            `json:"pendidik_id" binding:"required,uuid"`
		Berkas     map[string]string `json:"berkas" binding:"required,max=6,dive,keys,max=40,endkeys,uuid"`
	}

	// PerbaikiRequest: unggah ulang hanya berkas yang dinyatakan tidak valid (K6).
	PerbaikiRequest struct {
		Berkas map[string]string `json:"berkas" binding:"required,min=1,max=6,dive,keys,max=40,endkeys,uuid"`
	}

	// VerifikasiRequest: penilaian per berkas (K3).
	VerifikasiRequest struct {
		Penilaian []persyaratan.Penilaian `json:"penilaian" binding:"required,min=1,max=6,dive"`
	}

	// KirimBKDRequest (K4), multipart; satu surat pengantar untuk beberapa usulan.
	KirimBKDRequest struct {
		IDs                 []string              `form:"ids" binding:"required,min=1,max=100,dive,uuid"`
		NomorSuratPengantar string                `form:"nomor_surat_pengantar" binding:"required,max=100"`
		TanggalKirim        string                `form:"tanggal_kirim" binding:"required,datetime=2006-01-02"`
		CaraKirim           string                `form:"cara_kirim" binding:"max=100"`
		Catatan             string                `form:"catatan" binding:"max=1000"`
		Bukti               *multipart.FileHeader `form:"bukti"`
	}

	// SKRequest (K5), multipart.
	SKRequest struct {
		NomorSK    string                `form:"nomor_sk" binding:"required,max=100"`
		TanggalSK  string                `form:"tanggal_sk" binding:"required,datetime=2006-01-02"`
		TMTPangkat string                `form:"tmt_pangkat" binding:"required,datetime=2006-01-02"`
		SK         *multipart.FileHeader `form:"sk" binding:"required"`
	}

	Filter struct {
		Status    []string
		SekolahID string
		Periode   string // YYYY-MM
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

	RiwayatResponse struct {
		DariStatus string    `json:"dari_status"`
		KeStatus   string    `json:"ke_status"`
		Oleh       string    `json:"oleh"`
		OlehNama   string    `json:"oleh_nama"`
		Catatan    string    `json:"catatan"`
		CreatedAt  time.Time `json:"created_at"`
	}

	KenaikanPangkatResponse struct {
		ID           string           `json:"id"`
		Nomor        string           `json:"nomor"`
		Status       string           `json:"status"`
		Periode      string           `json:"periode"`
		PangkatLama  string           `json:"pangkat_lama"`
		GolonganLama string           `json:"golongan_lama"`
		PangkatBaru  string           `json:"pangkat_baru"`
		GolonganBaru string           `json:"golongan_baru"`
		Pendidik     *PendidikRingkas `json:"pendidik,omitempty"`
		Sekolah      *SekolahRingkas  `json:"sekolah,omitempty"`

		JumlahBerkas     int                          `json:"jumlah_berkas"`
		BerkasTidakValid int                          `json:"berkas_tidak_valid"`
		Berkas           []persyaratan.BerkasResponse `json:"berkas,omitempty"`
		DiverifikasiAt   *time.Time                   `json:"diverifikasi_at"`

		NomorSuratPengantar string                      `json:"nomor_surat_pengantar"`
		TanggalKirim        *string                     `json:"tanggal_kirim"`
		CaraKirim           string                      `json:"cara_kirim"`
		BuktiKirim          *persyaratan.DokumenRingkas `json:"bukti_kirim,omitempty"`
		CatatanKirim        string                      `json:"catatan_kirim"`

		NomorSK    string                      `json:"nomor_sk"`
		TanggalSK  *string                     `json:"tanggal_sk"`
		TMTPangkat *string                     `json:"tmt_pangkat"`
		SK         *persyaratan.DokumenRingkas `json:"sk,omitempty"`
		SelesaiAt  *time.Time                  `json:"selesai_at"`

		Riwayat   []RiwayatResponse `json:"riwayat,omitempty"`
		CreatedAt time.Time         `json:"created_at"`
		UpdatedAt time.Time         `json:"updated_at"`
	}

	// PerkiraanResponse untuk K2 "Cari data": usulan golongan sebelum mengajukan.
	PerkiraanResponse struct {
		Pendidik          PendidikRingkas `json:"pendidik"`
		Sekolah           *SekolahRingkas `json:"sekolah,omitempty"`
		Periode           string          `json:"periode"`
		PangkatLama       string          `json:"pangkat_lama"`
		GolonganLama      string          `json:"golongan_lama"`
		PangkatBaru       string          `json:"pangkat_baru"`
		GolonganBaru      string          `json:"golongan_baru"`
		PengajuanNomor    *string         `json:"pengajuan_nomor"`
		JumlahBerkasWajib int             `json:"jumlah_berkas_wajib"`
		// Kendala berisi alasan pengajuan belum bisa dibuat; kosong = bisa diajukan.
		Kendala []string `json:"kendala"`
		// Null bila tmt_golongan belum diisi di data induk (syarat 4 tahun tidak bisa dicek).
		TMTGolongan       *string    `json:"tmt_golongan"`
		MasaKerjaGolongan *MasaKerja `json:"masa_kerja_golongan"`
	}

	// MasaKerja dihitung sampai tanggal periode KP.
	MasaKerja struct {
		Tahun int `json:"tahun"`
		Bulan int `json:"bulan"`
	}

	// MemenuhiSyaratResponse: satu baris "Guru memenuhi syarat" di K1.
	MemenuhiSyaratResponse struct {
		Pendidik          PendidikRingkas `json:"pendidik"`
		Sekolah           *SekolahRingkas `json:"sekolah,omitempty"`
		Periode           string          `json:"periode"`
		GolonganLama      string          `json:"golongan_lama"`
		GolonganBaru      string          `json:"golongan_baru"`
		PangkatBaru       string          `json:"pangkat_baru"`
		TMTGolongan       string          `json:"tmt_golongan"`
		MasaKerjaGolongan MasaKerja       `json:"masa_kerja_golongan"`
	}

	KirimBKDResponse struct {
		Diusulkan []string `json:"diusulkan"`
	}
)

func Tgl(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(tglFormat)
	return &s
}

func Pendidik(p *entities.Pendidik) *PendidikRingkas {
	if p == nil {
		return nil
	}
	return &PendidikRingkas{
		ID: p.ID.String(), Nama: p.Nama, NIP: p.NIP, StatusKepegawaian: p.StatusKepegawaian,
		Pangkat: p.Pangkat, Golongan: p.Golongan, Jabatan: p.Jabatan,
	}
}

func Sekolah(s *entities.Sekolah) *SekolahRingkas {
	if s == nil {
		return nil
	}
	return &SekolahRingkas{ID: s.ID.String(), NPSN: s.NPSN, Nama: s.Nama}
}

func FromEntity(k entities.KenaikanPangkat) KenaikanPangkatResponse {
	r := KenaikanPangkatResponse{
		ID: k.ID.String(), Nomor: k.Nomor, Status: k.Status, Periode: k.Periode.Format("2006-01"),
		PangkatLama: k.PangkatLama, GolonganLama: k.GolonganLama, PangkatBaru: k.PangkatBaru, GolonganBaru: k.GolonganBaru,
		Pendidik: Pendidik(k.Pendidik), Sekolah: Sekolah(k.Sekolah),
		DiverifikasiAt:      k.DiverifikasiAt,
		NomorSuratPengantar: k.NomorSuratPengantar, TanggalKirim: Tgl(k.TanggalKirim), CaraKirim: k.CaraKirim,
		BuktiKirim: persyaratan.Dokumen(k.BuktiKirim), CatatanKirim: k.CatatanKirim,
		NomorSK: k.NomorSK, TanggalSK: Tgl(k.TanggalSK), TMTPangkat: Tgl(k.TMTPangkat),
		SK: persyaratan.Dokumen(k.SK), SelesaiAt: k.SelesaiAt,
		CreatedAt: k.CreatedAt, UpdatedAt: k.UpdatedAt,
	}
	items := make([]persyaratan.Item, 0, len(k.Berkas))
	for _, b := range k.Berkas {
		items = append(items, persyaratan.Item{Berkas: b.BerkasPersyaratan, Dokumen: b.Dokumen})
		if b.Valid != nil && !*b.Valid {
			r.BerkasTidakValid++
		}
	}
	r.JumlahBerkas = len(k.Berkas)
	r.Berkas = persyaratan.KeResponse(items, entities.KPBerkasWajib)
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
