package migrations

import (
	"github.com/Caknoooo/go-gin-clean-starter/database"
	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"gorm.io/gorm"
)

func init() {
	database.RegisterMigration("20261115040000_create_surat_tables", Up20261115040000CreateSuratTables, Down20261115040000CreateSuratTables)
}

func Up20261115040000CreateSuratTables(db *gorm.DB) error {
	return db.AutoMigrate(&entities.Surat{}, &entities.SuratRiwayat{})
}

func Down20261115040000CreateSuratTables(db *gorm.DB) error {
	return db.Migrator().DropTable(&entities.SuratRiwayat{}, &entities.Surat{})
}
