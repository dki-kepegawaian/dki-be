package migrations

import (
	"github.com/Caknoooo/go-gin-clean-starter/database"
	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"gorm.io/gorm"
)

func init() {
	database.RegisterMigration("20261115000000_add_pendidik_tanggal", Up20261115000000AddPendidikTanggal, Down20261115000000AddPendidikTanggal)
}

// Menambah tanggal_lahir dan tmt_cpns (nullable) ke pendidik.
func Up20261115000000AddPendidikTanggal(db *gorm.DB) error {
	return db.AutoMigrate(&entities.Pendidik{})
}

func Down20261115000000AddPendidikTanggal(db *gorm.DB) error {
	if err := db.Migrator().DropColumn(&entities.Pendidik{}, "tmt_cpns"); err != nil {
		return err
	}
	return db.Migrator().DropColumn(&entities.Pendidik{}, "tanggal_lahir")
}
