package migrations

import (
	"github.com/Caknoooo/go-gin-clean-starter/database"
	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"gorm.io/gorm"
)

func init() {
	database.RegisterMigration("20261115020001_create_kinerja_tables", Up20261115020001CreateKinerjaTables, Down20261115020001CreateKinerjaTables)
}

func Up20261115020001CreateKinerjaTables(db *gorm.DB) error {
	return db.AutoMigrate(&entities.KinerjaPenilaian{}, &entities.KinerjaRiwayat{})
}

func Down20261115020001CreateKinerjaTables(db *gorm.DB) error {
	return db.Migrator().DropTable(&entities.KinerjaRiwayat{}, &entities.KinerjaPenilaian{})
}
