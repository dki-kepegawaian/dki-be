package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LogAktivitas struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	UserID     uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	Aksi       string    `gorm:"type:varchar(50);not null" json:"aksi"`
	Entitas    string    `gorm:"type:varchar(50);not null;index:idx_log_entitas" json:"entitas"`
	EntitasID  uuid.UUID `gorm:"type:uuid;not null;index:idx_log_entitas" json:"entitas_id"`
	Keterangan string    `gorm:"type:text" json:"keterangan"`
	CreatedAt  time.Time `gorm:"type:timestamp with time zone" json:"created_at"`

	User *User `gorm:"foreignKey:UserID" json:"-"`
}

func (LogAktivitas) TableName() string { return "log_aktivitas" }

func (l *LogAktivitas) BeforeCreate(_ *gorm.DB) error {
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	return nil
}
