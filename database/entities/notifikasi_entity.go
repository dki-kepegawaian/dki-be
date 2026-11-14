package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Notifikasi struct {
	ID        uuid.UUID  `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index:idx_notifikasi_user_dibaca" json:"user_id"`
	Judul     string     `gorm:"type:varchar(150);not null" json:"judul"`
	Pesan     string     `gorm:"type:text" json:"pesan"`
	Tautan    string     `gorm:"type:varchar(255)" json:"tautan"`
	DibacaAt  *time.Time `gorm:"type:timestamp with time zone;index:idx_notifikasi_user_dibaca" json:"dibaca_at"`
	CreatedAt time.Time  `gorm:"type:timestamp with time zone;index" json:"created_at"`

	User *User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
}

func (Notifikasi) TableName() string { return "notifikasi" }

func (n *Notifikasi) BeforeCreate(_ *gorm.DB) error {
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	return nil
}
