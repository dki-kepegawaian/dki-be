package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Dokumen adalah metadata berkas yang diunggah. Isinya disimpan di storage/dokumen
// dan hanya bisa diunduh lewat endpoint yang mengecek akses berdasarkan SekolahID.
type Dokumen struct {
	ID           uuid.UUID  `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	SekolahID    *uuid.UUID `gorm:"type:uuid;index" json:"sekolah_id"`
	DiunggahOleh uuid.UUID  `gorm:"type:uuid;not null" json:"diunggah_oleh"`
	NamaAsli     string     `gorm:"type:varchar(255);not null" json:"nama_asli"`
	Path         string     `gorm:"type:varchar(255);not null" json:"-"`
	MimeType     string     `gorm:"type:varchar(100)" json:"mime_type"`
	Ukuran       int64      `gorm:"not null" json:"ukuran"`
	CreatedAt    time.Time  `gorm:"type:timestamp with time zone" json:"created_at"`

	Sekolah    *Sekolah `gorm:"foreignKey:SekolahID" json:"-"`
	Pengunggah *User    `gorm:"foreignKey:DiunggahOleh" json:"-"`
}

func (Dokumen) TableName() string { return "dokumen" }

func (d *Dokumen) BeforeCreate(_ *gorm.DB) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return nil
}
