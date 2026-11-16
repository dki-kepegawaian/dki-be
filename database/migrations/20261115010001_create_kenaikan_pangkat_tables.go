package migrations

import (
	"github.com/Caknoooo/go-gin-clean-starter/database"
	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"gorm.io/gorm"
)

func init() {
	database.RegisterMigration("20261115010001_create_kenaikan_pangkat_tables", Up20261115010001CreateKenaikanPangkatTables, Down20261115010001CreateKenaikanPangkatTables)
}

func Up20261115010001CreateKenaikanPangkatTables(db *gorm.DB) error {
	return db.AutoMigrate(&entities.KenaikanPangkat{}, &entities.KenaikanPangkatBerkas{}, &entities.KenaikanPangkatRiwayat{})
}

func Down20261115010001CreateKenaikanPangkatTables(db *gorm.DB) error {
	return db.Migrator().DropTable(&entities.KenaikanPangkatRiwayat{}, &entities.KenaikanPangkatBerkas{}, &entities.KenaikanPangkat{})
}
