package migrations

import (
	"github.com/Caknoooo/go-gin-clean-starter/database"
	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"gorm.io/gorm"
)

func init() {
	database.RegisterMigration("20261115050000_create_anjab_tables", Up20261115050000CreateAnjabTables, Down20261115050000CreateAnjabTables)
}

func Up20261115050000CreateAnjabTables(db *gorm.DB) error {
	return db.AutoMigrate(
		&entities.AnjabPeriode{}, &entities.AnjabBeban{}, &entities.AnjabSekolah{},
		&entities.AnjabRombel{}, &entities.AnjabGuru{}, &entities.AnjabRiwayat{},
	)
}

func Down20261115050000CreateAnjabTables(db *gorm.DB) error {
	return db.Migrator().DropTable(
		&entities.AnjabRiwayat{}, &entities.AnjabGuru{}, &entities.AnjabRombel{},
		&entities.AnjabSekolah{}, &entities.AnjabBeban{}, &entities.AnjabPeriode{},
	)
}
