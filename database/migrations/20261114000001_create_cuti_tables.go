package migrations

import (
	"github.com/Caknoooo/go-gin-clean-starter/database"
	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"gorm.io/gorm"
)

func init() {
	database.RegisterMigration("20261114000001_create_cuti_tables", Up20261114000001CreateCutiTables, Down20261114000001CreateCutiTables)
}

func Up20261114000001CreateCutiTables(db *gorm.DB) error {
	return db.AutoMigrate(&entities.Cuti{}, &entities.CutiRiwayat{})
}

func Down20261114000001CreateCutiTables(db *gorm.DB) error {
	return db.Migrator().DropTable(&entities.CutiRiwayat{}, &entities.Cuti{})
}
