package migrations

import (
	"github.com/Caknoooo/go-gin-clean-starter/database"
	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"gorm.io/gorm"
)

func init() {
	database.RegisterMigration("20261115020000_create_dapodik_tables", Up20261115020000CreateDapodikTables, Down20261115020000CreateDapodikTables)
}

func Up20261115020000CreateDapodikTables(db *gorm.DB) error {
	return db.AutoMigrate(&entities.DapodikTiket{}, &entities.DapodikLampiran{}, &entities.DapodikRiwayat{})
}

func Down20261115020000CreateDapodikTables(db *gorm.DB) error {
	return db.Migrator().DropTable(&entities.DapodikRiwayat{}, &entities.DapodikLampiran{}, &entities.DapodikTiket{})
}
