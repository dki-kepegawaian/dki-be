package migrations

import (
	"github.com/Caknoooo/go-gin-clean-starter/database"
	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"gorm.io/gorm"
)

func init() {
	database.RegisterMigration("20261116000000_add_pendidik_tmt_golongan", Up20261116000000AddPendidikTMTGolongan, Down20261116000000AddPendidikTMTGolongan)
}

// Menambah tmt_golongan (nullable) ke pendidik untuk syarat 4 tahun kenaikan pangkat.
func Up20261116000000AddPendidikTMTGolongan(db *gorm.DB) error {
	return db.AutoMigrate(&entities.Pendidik{})
}

func Down20261116000000AddPendidikTMTGolongan(db *gorm.DB) error {
	return db.Migrator().DropColumn(&entities.Pendidik{}, "tmt_golongan")
}
