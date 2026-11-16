package migrations

import (
	"github.com/Caknoooo/go-gin-clean-starter/database"
	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"gorm.io/gorm"
)

func init() {
	database.RegisterMigration("20261115060000_create_arsip_tables", Up20261115060000CreateArsipTables, Down20261115060000CreateArsipTables)
}

func Up20261115060000CreateArsipTables(db *gorm.DB) error {
	return db.AutoMigrate(&entities.Arsip{}, &entities.ArsipVersi{}, &entities.ArsipLog{})
}

func Down20261115060000CreateArsipTables(db *gorm.DB) error {
	return db.Migrator().DropTable(&entities.ArsipLog{}, &entities.ArsipVersi{}, &entities.Arsip{})
}
