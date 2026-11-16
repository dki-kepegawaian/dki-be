package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	SuratStatusDraf        = "draf"
	SuratStatusMenungguTTD = "menunggu_ttd"
	SuratStatusFinal       = "final"
	SuratStatusDibatalkan  = "dibatalkan"

	SuratJenisTunjangan                = "tunjangan"
	SuratJenisAktifMengajar            = "aktif_mengajar"
	SuratJenisPengantarKenaikanPangkat = "pengantar_kenaikan_pangkat"
	SuratJenisIzinCuti                 = "izin_cuti"
	SuratJenisPengantarPensiun         = "pengantar_pensiun"
)

// Surat adalah surat yang dibuat staf Sudin dari Generator Surat (S1-S5).
// Alur: draf -> menunggu_ttd (PDF dibuat) -> final (hasil pindai bertanda tangan diunggah).
// Draf dan menunggu_ttd bisa dibatalkan.
type Surat struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Nomor        string    `gorm:"type:varchar(50);uniqueIndex;not null" json:"nomor"`
	Jenis        string    `gorm:"type:varchar(40);not null;index" json:"jenis"`
	PendidikID   uuid.UUID `gorm:"type:uuid;not null;index" json:"pendidik_id"`
	SekolahID    uuid.UUID `gorm:"type:uuid;not null;index" json:"sekolah_id"`
	DibuatOleh   uuid.UUID `gorm:"type:uuid;not null" json:"dibuat_oleh"`
	Perihal      string    `gorm:"type:varchar(300);not null" json:"perihal"`
	TanggalSurat time.Time `gorm:"type:date;not null" json:"tanggal_surat"`
	Keterangan   string    `gorm:"type:text" json:"keterangan"`
	Status       string    `gorm:"type:varchar(20);not null;index" json:"status"`

	// S3: PDF yang dibuat sistem untuk dicetak dan ditandatangani basah.
	DokumenID *uuid.UUID `gorm:"type:uuid" json:"dokumen_id"`
	// S4: hasil pindai surat bertanda tangan.
	DokumenTTDID *uuid.UUID `gorm:"column:dokumen_ttd_id;type:uuid" json:"dokumen_ttd_id"`
	FinalOleh    *uuid.UUID `gorm:"type:uuid" json:"final_oleh"`
	FinalAt      *time.Time `gorm:"type:timestamp with time zone" json:"final_at"`

	// S5: penyerahan surat asli dilakukan di luar sistem, hanya ditandai.
	DiserahkanOleh *uuid.UUID `gorm:"type:uuid" json:"diserahkan_oleh"`
	DiserahkanAt   *time.Time `gorm:"type:timestamp with time zone" json:"diserahkan_at"`

	AlasanBatal string `gorm:"type:text" json:"alasan_batal"`

	Pendidik   *Pendidik      `gorm:"foreignKey:PendidikID;constraint:OnDelete:RESTRICT" json:"pendidik,omitempty"`
	Sekolah    *Sekolah       `gorm:"foreignKey:SekolahID;constraint:OnDelete:RESTRICT" json:"sekolah,omitempty"`
	Pembuat    *User          `gorm:"foreignKey:DibuatOleh" json:"-"`
	Dokumen    *Dokumen       `gorm:"foreignKey:DokumenID" json:"dokumen,omitempty"`
	DokumenTTD *Dokumen       `gorm:"foreignKey:DokumenTTDID" json:"dokumen_ttd,omitempty"`
	Riwayat    []SuratRiwayat `gorm:"foreignKey:SuratID" json:"riwayat,omitempty"`

	Timestamp
}

func (Surat) TableName() string { return "surat" }

func (s *Surat) BeforeCreate(_ *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

type SuratRiwayat struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	SuratID    uuid.UUID `gorm:"type:uuid;not null;index" json:"surat_id"`
	DariStatus string    `gorm:"type:varchar(20)" json:"dari_status"`
	KeStatus   string    `gorm:"type:varchar(20);not null" json:"ke_status"`
	Oleh       uuid.UUID `gorm:"type:uuid;not null" json:"oleh"`
	Catatan    string    `gorm:"type:text" json:"catatan"`
	CreatedAt  time.Time `gorm:"type:timestamp with time zone" json:"created_at"`

	Surat    *Surat `gorm:"foreignKey:SuratID;constraint:OnDelete:CASCADE" json:"-"`
	Pengguna *User  `gorm:"foreignKey:Oleh" json:"-"`
}

func (SuratRiwayat) TableName() string { return "surat_riwayat" }

func (r *SuratRiwayat) BeforeCreate(_ *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
