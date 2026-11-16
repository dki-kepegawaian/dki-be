package entities

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Sudin adalah Suku Dinas Pendidikan per kota administrasi (mis. Jakarta Barat).
type Sudin struct {
	ID   uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Kode string    `gorm:"type:varchar(10);uniqueIndex;not null" json:"kode"`
	Nama string    `gorm:"type:varchar(100);uniqueIndex;not null" json:"nama"`
	// Kontak layanan kepegawaian yang ditampilkan ke sekolah (H5 "Kontak Sudin").
	KontakUnit    string `gorm:"type:varchar(150)" json:"kontak_unit"`
	KontakTelepon string `gorm:"type:varchar(30)" json:"kontak_telepon"`
	KontakEmail   string `gorm:"type:varchar(255)" json:"kontak_email"`
	JamLayanan    string `gorm:"type:varchar(100)" json:"jam_layanan"`

	Timestamp
}

func (Sudin) TableName() string { return "sudin" }

func (s *Sudin) BeforeCreate(_ *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}
