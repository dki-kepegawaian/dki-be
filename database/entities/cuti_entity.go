package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	CutiStatusDiajukan           = "diajukan"
	CutiStatusPerluPerbaikan     = "perlu_perbaikan"
	CutiStatusMenungguSuratFisik = "menunggu_surat_fisik"
	CutiStatusDiprosesSudin      = "diproses_sudin"
	CutiStatusDisetujui          = "disetujui"
	CutiStatusDitolak            = "ditolak"

	CutiJenisTahunan                = "tahunan"
	CutiJenisBesar                  = "besar"
	CutiJenisSakit                  = "sakit"
	CutiJenisMelahirkan             = "melahirkan"
	CutiJenisAlasanPenting          = "alasan_penting"
	CutiJenisDiLuarTanggunganNegara = "di_luar_tanggungan_negara"

	CutiJatahTahunan = 12 // hari kerja per tahun
)

// CutiStatusSelesai adalah status final yang tidak bisa diubah lagi.
func CutiStatusSelesai(status string) bool {
	return status == CutiStatusDisetujui || status == CutiStatusDitolak
}

type Cuti struct {
	ID                 uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Nomor              string    `gorm:"type:varchar(20);uniqueIndex;not null" json:"nomor"`
	PendidikID         uuid.UUID `gorm:"type:uuid;not null;index" json:"pendidik_id"`
	SekolahID          uuid.UUID `gorm:"type:uuid;not null;index" json:"sekolah_id"`
	DiajukanOleh       uuid.UUID `gorm:"type:uuid;not null" json:"diajukan_oleh"`
	Jenis              string    `gorm:"type:varchar(30);not null" json:"jenis"`
	TanggalMulai       time.Time `gorm:"type:date;not null" json:"tanggal_mulai"`
	TanggalSelesai     time.Time `gorm:"type:date;not null" json:"tanggal_selesai"`
	HariKerja          int       `gorm:"not null" json:"hari_kerja"`
	Alasan             string    `gorm:"type:text;not null" json:"alasan"`
	DisetujuiKepsek    bool      `gorm:"not null" json:"disetujui_kepsek"`
	BerkasPermohonanID uuid.UUID `gorm:"type:uuid;not null" json:"berkas_permohonan_id"`
	Status             string    `gorm:"type:varchar(30);not null;index" json:"status"`

	// C3: periksa berkas digital
	BerkasLengkap    *bool      `json:"berkas_lengkap"`
	BerkasTerbaca    *bool      `json:"berkas_terbaca"`
	CatatanPerbaikan string     `gorm:"type:text" json:"catatan_perbaikan"`
	DiperiksaOleh    *uuid.UUID `gorm:"type:uuid" json:"diperiksa_oleh"`
	DiperiksaAt      *time.Time `gorm:"type:timestamp with time zone" json:"diperiksa_at"`

	// C4: penerimaan surat fisik + nota dinas
	SuratFisikDiterimaTanggal *time.Time `gorm:"type:date" json:"surat_fisik_diterima_tanggal"`
	SuratFisikPenerimaID      *uuid.UUID `gorm:"type:uuid" json:"surat_fisik_penerima_id"`
	NotaDinasAda              *bool      `json:"nota_dinas_ada"`
	CatatanPenerimaan         string     `gorm:"type:text" json:"catatan_penerimaan"`

	// C5: keputusan pejabat
	NomorSuratIzin   string     `gorm:"type:varchar(100)" json:"nomor_surat_izin"`
	TanggalSuratIzin *time.Time `gorm:"type:date" json:"tanggal_surat_izin"`
	SuratIzinID      *uuid.UUID `gorm:"type:uuid" json:"surat_izin_id"`
	CatatanKeputusan string     `gorm:"type:text" json:"catatan_keputusan"`
	DiputuskanOleh   *uuid.UUID `gorm:"type:uuid" json:"diputuskan_oleh"`
	DiputuskanAt     *time.Time `gorm:"type:timestamp with time zone" json:"diputuskan_at"`

	Pendidik         *Pendidik     `gorm:"foreignKey:PendidikID;constraint:OnDelete:RESTRICT" json:"pendidik,omitempty"`
	Sekolah          *Sekolah      `gorm:"foreignKey:SekolahID;constraint:OnDelete:RESTRICT" json:"sekolah,omitempty"`
	Pengaju          *User         `gorm:"foreignKey:DiajukanOleh" json:"-"`
	BerkasPermohonan *Dokumen      `gorm:"foreignKey:BerkasPermohonanID" json:"berkas_permohonan,omitempty"`
	SuratIzin        *Dokumen      `gorm:"foreignKey:SuratIzinID" json:"surat_izin,omitempty"`
	Pemeriksa        *User         `gorm:"foreignKey:DiperiksaOleh" json:"-"`
	PenerimaSurat    *User         `gorm:"foreignKey:SuratFisikPenerimaID" json:"-"`
	Pemutus          *User         `gorm:"foreignKey:DiputuskanOleh" json:"-"`
	Riwayat          []CutiRiwayat `gorm:"foreignKey:CutiID" json:"riwayat,omitempty"`

	Timestamp
}

func (Cuti) TableName() string { return "cuti" }

func (c *Cuti) BeforeCreate(_ *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

type CutiRiwayat struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	CutiID     uuid.UUID `gorm:"type:uuid;not null;index" json:"cuti_id"`
	DariStatus string    `gorm:"type:varchar(30)" json:"dari_status"`
	KeStatus   string    `gorm:"type:varchar(30);not null" json:"ke_status"`
	Oleh       uuid.UUID `gorm:"type:uuid;not null" json:"oleh"`
	Catatan    string    `gorm:"type:text" json:"catatan"`
	CreatedAt  time.Time `gorm:"type:timestamp with time zone" json:"created_at"`

	Cuti     *Cuti `gorm:"foreignKey:CutiID;constraint:OnDelete:CASCADE" json:"-"`
	Pengguna *User `gorm:"foreignKey:Oleh" json:"-"`
}

func (CutiRiwayat) TableName() string { return "cuti_riwayat" }

func (r *CutiRiwayat) BeforeCreate(_ *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
