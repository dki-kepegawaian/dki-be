package migrations

import (
	"github.com/Caknoooo/go-gin-clean-starter/database"
	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"gorm.io/gorm"
)

func init() {
	database.RegisterMigration("20261114000000_create_fondasi_tables", Up20261114000000CreateFondasiTables, Down20261114000000CreateFondasiTables)
}

func Up20261114000000CreateFondasiTables(db *gorm.DB) error {
	return db.AutoMigrate(
		&entities.Sudin{},
		&entities.Sekolah{},
		&entities.User{},
		&entities.Pendidik{},
		&entities.Dokumen{},
		&entities.Notifikasi{},
		&entities.LogAktivitas{},
		&entities.NomorUrut{},
	)
}

func Down20261114000000CreateFondasiTables(db *gorm.DB) error {
	if err := db.Migrator().DropColumn(&entities.User{}, "sekolah_id"); err != nil {
		return err
	}
	if err := db.Migrator().DropColumn(&entities.User{}, "sudin_id"); err != nil {
		return err
	}
	return db.Migrator().DropTable(
		&entities.NomorUrut{},
		&entities.LogAktivitas{},
		&entities.Notifikasi{},
		&entities.Dokumen{},
		&entities.Pendidik{},
		&entities.Sekolah{},
		&entities.Sudin{},
	)
}
