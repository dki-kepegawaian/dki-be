package dto

import (
	"mime/multipart"
	"time"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/persyaratan"
)

const tglFormat = "2006-01-02"

type (
	// UnggahBerkasRequest: satu berkas persyaratan (PDF, maks 2 MB) sebelum diajukan (P2).
	UnggahBerkasRequest struct {
		Berkas *multipart.FileHeader `form:"berkas" binding:"required"`
	}

	// AjukanRequest (P1+P2). Berkas: kode jenis -> id dokumen dari POST /api/pensiun/berkas.
	AjukanRequest struct {
		PendidikID          string            `json:"pendidik_id" binding:"required,uuid"`
		Jenis               string            `json:"jenis" binding:"required,oneof=batas_usia atas_permintaan_sendiri"`
		TMTPensiun          string            `json:"tmt_pensiun" binding:"omitempty,datetime=2006-01-02"`
		PernyataanDiperiksa bool              `json:"pernyataan_diperiksa"`
		Berkas              map[string]string `json:"berkas" binding:"required,max=14,dive,keys,max=40,endkeys,uuid"`
	}

	// PerbaikiRequest: unggah ulang hanya berkas yang dinyatakan tidak valid (P6).
	PerbaikiRequest struct {
		Berkas map[string]string `json:"berkas" binding:"required,min=1,max=14,dive,keys,max=40,endkeys,uuid"`
	}

	// VerifikasiRequest: penilaian per berkas (P3).
	VerifikasiRequest struct {
		Penilaian []persyaratan.Penilaian `json:"penilaian" binding:"required,min=1,max=14,dive"`
	}

	// KirimBKDRequest (P4), multipart.
	KirimBKDRequest struct {
		NomorSuratPengantar string                `form:"nomor_surat_pengantar" binding:"required,max=100"`
		TanggalKirim        string                `form:"tanggal_kirim" binding:"required,datetime=2006-01-02"`
		CaraKirim           string                `form:"cara_kirim" binding:"max=100"`
		Catatan             string                `form:"catatan" binding:"max=1000"`
		Bukti               *multipart.FileHeader `form:"bukti"`
	}

	// SKRequest (P5), multipart. TMTPensiun opsional, mengikuti SK bila berbeda.
	SKRequest struct {
		NomorSK    string                `form:"nomor_sk" binding:"required,max=100"`
		TanggalSK  string                `form:"tanggal_sk" binding:"required,datetime=2006-01-02"`
		TMTPensiun string                `form:"tmt_pensiun" binding:"omitempty,datetime=2006-01-02"`
		SK         *multipart.FileHeader `form:"sk" binding:"required"`
	}

	Filter struct {
		Status    []string
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
		TanggalLahir      *string `json:"tanggal_lahir"`
		TMTCPNS           *string `json:"tmt_cpns"`
	}

	SekolahRingkas struct {
		ID   string `json:"id"`
		NPSN string `json:"npsn"`
		Nama string `json:"nama"`
	}

	MasaKerja struct {
		Tahun int `json:"tahun"`
		Bulan int `json:"bulan"`
	}

	RiwayatResponse struct {
		DariStatus string    `json:"dari_status"`
		KeStatus   string    `json:"ke_status"`
		Oleh       string    `json:"oleh"`
		OlehNama   string    `json:"oleh_nama"`
		Catatan    string    `json:"catatan"`
		CreatedAt  time.Time `json:"created_at"`
	}

	PensiunResponse struct {
		ID         string           `json:"id"`
		Nomor      string           `json:"nomor"`
		Status     string           `json:"status"`
		Jenis      string           `json:"jenis"`
		TMTPensiun string           `json:"tmt_pensiun"`
		MasaKerja  *MasaKerja       `json:"masa_kerja"`
		Pendidik   *PendidikRingkas `json:"pendidik,omitempty"`
		Sekolah    *SekolahRingkas  `json:"sekolah,omitempty"`

		JumlahBerkas     int                          `json:"jumlah_berkas"`
		BerkasTidakValid int                          `json:"berkas_tidak_valid"`
		Berkas           []persyaratan.BerkasResponse `json:"berkas,omitempty"`
		DiverifikasiAt   *time.Time                   `json:"diverifikasi_at"`

		NomorSuratPengantar string                      `json:"nomor_surat_pengantar"`
		TanggalKirim        *string                     `json:"tanggal_kirim"`
		CaraKirim           string                      `json:"cara_kirim"`
		BuktiKirim          *persyaratan.DokumenRingkas `json:"bukti_kirim,omitempty"`
		CatatanKirim        string                      `json:"catatan_kirim"`

		NomorSK   string                      `json:"nomor_sk"`
		TanggalSK *string                     `json:"tanggal_sk"`
		SK        *persyaratan.DokumenRingkas `json:"sk,omitempty"`
		SelesaiAt *time.Time                  `json:"selesai_at"`

		Riwayat   []RiwayatResponse `json:"riwayat,omitempty"`
		CreatedAt time.Time         `json:"created_at"`
		UpdatedAt time.Time         `json:"updated_at"`
	}

	// PerkiraanResponse untuk P1: data guru dan TMT pensiun batas usia sebelum mengajukan.
	PerkiraanResponse struct {
		Pendidik          PendidikRingkas `json:"pendidik"`
		Sekolah           *SekolahRingkas `json:"sekolah,omitempty"`
		BatasUsia         int             `json:"batas_usia"`
		TMTPensiunBUP     *string         `json:"tmt_pensiun_bup"`
		MasaKerja         *MasaKerja      `json:"masa_kerja"`
		PengajuanNomor    *string         `json:"pengajuan_nomor"`
		JumlahBerkasWajib int             `json:"jumlah_berkas_wajib"`
		// Kendala berisi alasan pengajuan belum bisa dibuat; kosong = bisa diajukan.
		Kendala []string `json:"kendala"`
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
		TanggalLahir: Tgl(p.TanggalLahir), TMTCPNS: Tgl(p.TMTCPNS),
	}
}

func Sekolah(s *entities.Sekolah) *SekolahRingkas {
	if s == nil {
		return nil
	}
	return &SekolahRingkas{ID: s.ID.String(), NPSN: s.NPSN, Nama: s.Nama}
}

// HitungMasaKerja: dari TMT CPNS sampai tanggal tertentu; nil bila TMT CPNS belum diisi.
func HitungMasaKerja(p *entities.Pendidik, sampai time.Time) *MasaKerja {
	if p == nil || p.TMTCPNS == nil {
		return nil
	}
	t, b := persyaratan.MasaKerja(*p.TMTCPNS, sampai)
	return &MasaKerja{Tahun: t, Bulan: b}
}

func FromEntity(p entities.Pensiun) PensiunResponse {
	r := PensiunResponse{
		ID: p.ID.String(), Nomor: p.Nomor, Status: p.Status, Jenis: p.Jenis,
		TMTPensiun: p.TMTPensiun.Format(tglFormat),
		MasaKerja:  HitungMasaKerja(p.Pendidik, p.TMTPensiun),
		Pendidik:   Pendidik(p.Pendidik), Sekolah: Sekolah(p.Sekolah),
		DiverifikasiAt:      p.DiverifikasiAt,
		NomorSuratPengantar: p.NomorSuratPengantar, TanggalKirim: Tgl(p.TanggalKirim), CaraKirim: p.CaraKirim,
		BuktiKirim: persyaratan.Dokumen(p.BuktiKirim), CatatanKirim: p.CatatanKirim,
		NomorSK: p.NomorSK, TanggalSK: Tgl(p.TanggalSK), SK: persyaratan.Dokumen(p.SK), SelesaiAt: p.SelesaiAt,
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
	items := make([]persyaratan.Item, 0, len(p.Berkas))
	for _, b := range p.Berkas {
		items = append(items, persyaratan.Item{Berkas: b.BerkasPersyaratan, Dokumen: b.Dokumen})
		if b.Valid != nil && !*b.Valid {
			r.BerkasTidakValid++
		}
	}
	r.JumlahBerkas = len(p.Berkas)
	r.Berkas = persyaratan.KeResponse(items, entities.PensiunBerkasWajib)
	for _, h := range p.Riwayat {
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
