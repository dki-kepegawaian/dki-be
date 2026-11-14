package database

import (
	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&entities.Migration{},
		&entities.Sudin{},
		&entities.Sekolah{},
		&entities.User{},
		&entities.RefreshToken{},
		&entities.Pendidik{},
		&entities.Dokumen{},
		&entities.Notifikasi{},
		&entities.LogAktivitas{},
		&entities.NomorUrut{},
		&entities.Cuti{},
		&entities.CutiRiwayat{},
	); err != nil {
		return err
	}

	manager := NewMigrationManager(db)
	return manager.Run()
}
