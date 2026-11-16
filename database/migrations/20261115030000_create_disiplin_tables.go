package migrations

import (
	"github.com/Caknoooo/go-gin-clean-starter/database"
	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"gorm.io/gorm"
)

func init() {
	database.RegisterMigration("20261115030000_create_disiplin_tables", Up20261115030000CreateDisiplinTables, Down20261115030000CreateDisiplinTables)
}

func Up20261115030000CreateDisiplinTables(db *gorm.DB) error {
	return db.AutoMigrate(&entities.Disiplin{}, &entities.DisiplinBerkas{}, &entities.DisiplinRiwayat{})
}

func Down20261115030000CreateDisiplinTables(db *gorm.DB) error {
	return db.Migrator().DropTable(&entities.DisiplinRiwayat{}, &entities.DisiplinBerkas{}, &entities.Disiplin{})
}
