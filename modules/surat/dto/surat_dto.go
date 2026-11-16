package dto

import (
	"mime/multipart"
	"time"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
)

const TglFormat = "2006-01-02"

type (
	// BuatRequest menyimpan draf surat (S2 > Simpan draf).
	// Nomor boleh kosong; server lalu memberi nomor urut berikutnya.
	BuatRequest struct {
		Jenis        string `json:"jenis" binding:"required,oneof=tunjangan aktif_mengajar pengantar_kenaikan_pangkat izin_cuti pengantar_pensiun"`
		PendidikID   string `json:"pendidik_id" binding:"required,uuid"`
		Perihal      string `json:"perihal" binding:"required,max=300"`
		TanggalSurat string `json:"tanggal_surat" binding:"required,datetime=2006-01-02"`
		Nomor        string `json:"nomor" binding:"omitempty,max=50"`
		Keterangan   string `json:"keterangan" binding:"max=2000"`
	}

	// UbahRequest mengubah isian surat yang masih draf (S3 > Ubah isian).
	// Jenis dan pendidik tidak bisa diganti; buat draf baru bila salah pilih.
	UbahRequest struct {
		Perihal      string `json:"perihal" binding:"required,max=300"`
		TanggalSurat string `json:"tanggal_surat" binding:"required,datetime=2006-01-02"`
		Nomor        string `json:"nomor" binding:"omitempty,max=50"`
		Keterangan   string `json:"keterangan" binding:"max=2000"`
	}

	// FinalRequest dikirim sebagai multipart/form-data (S4 > Unggah & tetapkan Final).
	FinalRequest struct {
		Berkas *multipart.FileHeader `form:"berkas" binding:"required"`
	}

	BatalRequest struct {
		Alasan string `json:"alasan" binding:"required,max=1000"`
	}

	Filter struct {
		Status     []string
		Jenis      string
		PendidikID string
		Search     string
		Offset     int
		Limit      int
	}
)

type (
	JenisResponse struct {
		Kode      string `json:"kode"`
		Label     string `json:"label"`
		Deskripsi string `json:"deskripsi"`
	}

	PendidikRingkas struct {
		ID                string  `json:"id"`
		Nama              string  `json:"nama"`
		NIP               *string `json:"nip"`
		StatusKepegawaian string  `json:"status_kepegawaian"`
		Pangkat           string  `json:"pangkat"`
		Golongan          string  `json:"golongan"`
		Jabatan           string  `json:"jabatan"`
		JabatanDapodik    string  `json:"jabatan_dapodik"`
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

	SuratResponse struct {
		ID            string           `json:"id"`
		Nomor         string           `json:"nomor"`
		Jenis         string           `json:"jenis"`
		JenisLabel    string           `json:"jenis_label"`
		Status        string           `json:"status"`
		Perihal       string           `json:"perihal"`
		TanggalSurat  string           `json:"tanggal_surat"`
		Keterangan    string           `json:"keterangan"`
		Kop           string           `json:"kop"`
		PenandaTangan string           `json:"penanda_tangan"`
		Pendidik      *PendidikRingkas `json:"pendidik,omitempty"`
		Sekolah       *SekolahRingkas  `json:"sekolah,omitempty"`
		Dokumen       *DokumenRingkas  `json:"dokumen,omitempty"`
		DokumenTTD    *DokumenRingkas  `json:"dokumen_ttd,omitempty"`
		FinalAt       *time.Time       `json:"final_at"`
		DiserahkanAt  *time.Time       `json:"diserahkan_at"`
		AlasanBatal   string           `json:"alasan_batal"`

		Riwayat   []RiwayatResponse `json:"riwayat,omitempty"`
		CreatedAt time.Time         `json:"created_at"`
		UpdatedAt time.Time         `json:"updated_at"`
	}
)

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

// FromEntity memetakan surat ke respons. jenisLabel, kop, dan penandaTangan
// berasal dari template di service supaya teksnya satu sumber.
func FromEntity(s entities.Surat, jenisLabel, kop, penandaTangan string) SuratResponse {
	r := SuratResponse{
		ID:            s.ID.String(),
		Nomor:         s.Nomor,
		Jenis:         s.Jenis,
		JenisLabel:    jenisLabel,
		Status:        s.Status,
		Perihal:       s.Perihal,
		TanggalSurat:  s.TanggalSurat.Format(TglFormat),
		Keterangan:    s.Keterangan,
		Kop:           kop,
		PenandaTangan: penandaTangan,
		Dokumen:       dokumen(s.Dokumen),
		DokumenTTD:    dokumen(s.DokumenTTD),
		FinalAt:       s.FinalAt,
		DiserahkanAt:  s.DiserahkanAt,
		AlasanBatal:   s.AlasanBatal,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
	}
	if p := s.Pendidik; p != nil {
		r.Pendidik = &PendidikRingkas{
			ID: p.ID.String(), Nama: p.Nama, NIP: p.NIP, StatusKepegawaian: p.StatusKepegawaian,
			Pangkat: p.Pangkat, Golongan: p.Golongan, Jabatan: p.Jabatan, JabatanDapodik: p.JabatanDapodik,
		}
	}
	if sk := s.Sekolah; sk != nil {
		r.Sekolah = &SekolahRingkas{ID: sk.ID.String(), NPSN: sk.NPSN, Nama: sk.Nama}
	}
	for _, h := range s.Riwayat {
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
