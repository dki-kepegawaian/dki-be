package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	ArsipKategoriSK        = "sk"
	ArsipKategoriTunjangan = "tunjangan"
	ArsipKategoriIjazah    = "ijazah"
	ArsipKategoriKinerja   = "kinerja"
	ArsipKategoriSurat     = "surat"
	ArsipKategoriLainnya   = "lainnya"

	ArsipAksiMelihat       = "melihat"
	ArsipAksiMengunduh     = "mengunduh"
	ArsipAksiMengunggah    = "mengunggah"
	ArsipAksiMenggantiVers = "mengganti_versi"
)

// Arsip adalah satu dokumen kepegawaian seorang pendidik; isinya ada di ArsipVersi.
type Arsip struct {
	ID             uuid.UUID  `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	PendidikID     uuid.UUID  `gorm:"type:uuid;not null;index" json:"pendidik_id"`
	SekolahID      uuid.UUID  `gorm:"type:uuid;not null;index" json:"sekolah_id"`
	Kategori       string     `gorm:"type:varchar(20);not null;index" json:"kategori"`
	JenisDokumen   string     `gorm:"type:varchar(150);not null" json:"jenis_dokumen"`
	Nomor          string     `gorm:"type:varchar(100)" json:"nomor"`
	TanggalDokumen *time.Time `gorm:"type:date" json:"tanggal_dokumen"`
	Keterangan     string     `gorm:"type:text" json:"keterangan"`
	VersiTerkini   int        `gorm:"not null;default:1" json:"versi_terkini"`
	// SumberModul diisi bila diarsipkan otomatis oleh modul lain (mis. "cuti").
	SumberModul string     `gorm:"type:varchar(30)" json:"sumber_modul"`
	SumberID    *uuid.UUID `gorm:"type:uuid;index" json:"sumber_id"`

	Pendidik *Pendidik    `gorm:"foreignKey:PendidikID;constraint:OnDelete:RESTRICT" json:"-"`
	Sekolah  *Sekolah     `gorm:"foreignKey:SekolahID;constraint:OnDelete:RESTRICT" json:"-"`
	Versi    []ArsipVersi `gorm:"foreignKey:ArsipID" json:"-"`

	Timestamp
}

func (Arsip) TableName() string { return "arsip" }

func (a *Arsip) BeforeCreate(_ *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

type ArsipVersi struct {
	ID                uuid.UUID  `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	ArsipID           uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex:idx_arsip_versi" json:"arsip_id"`
	Versi             int        `gorm:"not null;uniqueIndex:idx_arsip_versi" json:"versi"`
	DokumenID         uuid.UUID  `gorm:"type:uuid;not null" json:"dokumen_id"`
	LampiranDokumenID *uuid.UUID `gorm:"type:uuid" json:"lampiran_dokumen_id"`
	Catatan           string     `gorm:"type:text" json:"catatan"`
	DiunggahOleh      uuid.UUID  `gorm:"type:uuid;not null" json:"diunggah_oleh"`
	CreatedAt         time.Time  `gorm:"type:timestamp with time zone" json:"created_at"`

	Dokumen         *Dokumen `gorm:"foreignKey:DokumenID" json:"-"`
	LampiranDokumen *Dokumen `gorm:"foreignKey:LampiranDokumenID" json:"-"`
	Pengunggah      *User    `gorm:"foreignKey:DiunggahOleh" json:"-"`
}

func (ArsipVersi) TableName() string { return "arsip_versi" }

func (v *ArsipVersi) BeforeCreate(_ *gorm.DB) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	return nil
}

// ArsipLog mencatat setiap aksi pada dokumen arsip (R4). Hanya ditambah, tidak pernah diubah.
type ArsipLog struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	ArsipID    uuid.UUID `gorm:"type:uuid;not null;index" json:"arsip_id"`
	Versi      int       `gorm:"not null" json:"versi"`
	Aksi       string    `gorm:"type:varchar(20);not null;index" json:"aksi"`
	UserID     uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	Keterangan string    `gorm:"type:varchar(255)" json:"keterangan"`
	CreatedAt  time.Time `gorm:"type:timestamp with time zone;index" json:"created_at"`

	Arsip *Arsip `gorm:"foreignKey:ArsipID;constraint:OnDelete:CASCADE" json:"-"`
	User  *User  `gorm:"foreignKey:UserID" json:"-"`
}

func (ArsipLog) TableName() string { return "arsip_log" }

func (l *ArsipLog) BeforeCreate(_ *gorm.DB) error {
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	return nil
}
