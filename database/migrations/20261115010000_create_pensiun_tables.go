package migrations

import (
	"github.com/Caknoooo/go-gin-clean-starter/database"
	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"gorm.io/gorm"
)

func init() {
	database.RegisterMigration("20261115010000_create_pensiun_tables", Up20261115010000CreatePensiunTables, Down20261115010000CreatePensiunTables)
}

func Up20261115010000CreatePensiunTables(db *gorm.DB) error {
	return db.AutoMigrate(&entities.Pensiun{}, &entities.PensiunBerkas{}, &entities.PensiunRiwayat{})
}

func Down20261115010000CreatePensiunTables(db *gorm.DB) error {
	return db.Migrator().DropTable(&entities.PensiunRiwayat{}, &entities.PensiunBerkas{}, &entities.Pensiun{})
}
