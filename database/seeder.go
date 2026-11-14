package database

import (
	"os"

	"github.com/Caknoooo/go-gin-clean-starter/database/seeders/seeds"
	"gorm.io/gorm"
)

func Seeder(db *gorm.DB) error {
	if err := seeds.AdminSeeder(db); err != nil {
		return err
	}

	// Data Sudin selalu diisi; sekolah, pendidik, dan akun contoh hanya bila SEED_DEMO=true.
	if err := seeds.FondasiSeeder(db, os.Getenv("SEED_DEMO") == "true"); err != nil {
		return err
	}

	return nil
}
