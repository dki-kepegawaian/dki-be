package entities

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Sekolah struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	SudinID   uuid.UUID `gorm:"type:uuid;not null;index" json:"sudin_id"`
	NPSN      string    `gorm:"column:npsn;type:varchar(8);uniqueIndex;not null" json:"npsn"`
	Nama      string    `gorm:"type:varchar(150);not null" json:"nama"`
	Jenjang   string    `gorm:"type:varchar(10);not null" json:"jenjang"`
	Kecamatan string    `gorm:"type:varchar(100)" json:"kecamatan"`
	Alamat    string    `gorm:"type:text" json:"alamat"`
	// Nama kepala sekolah untuk profil sekolah (H5); diubah oleh Sudin.
	KepalaSekolah string `gorm:"type:varchar(150)" json:"kepala_sekolah"`

	Sudin *Sudin `gorm:"foreignKey:SudinID;constraint:OnDelete:RESTRICT" json:"sudin,omitempty"`

	Timestamp
}

func (Sekolah) TableName() string { return "sekolah" }

func (s *Sekolah) BeforeCreate(_ *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}
