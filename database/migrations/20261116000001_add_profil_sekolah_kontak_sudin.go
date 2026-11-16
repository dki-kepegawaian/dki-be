package migrations

import (
	"github.com/Caknoooo/go-gin-clean-starter/database"
	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"gorm.io/gorm"
)

func init() {
	database.RegisterMigration("20261116000001_add_profil_sekolah_kontak_sudin", Up20261116000001AddProfilSekolah, Down20261116000001AddProfilSekolah)
}

// Menambah sekolah.kepala_sekolah dan kontak layanan di sudin (profil sekolah, H5).
func Up20261116000001AddProfilSekolah(db *gorm.DB) error {
	return db.AutoMigrate(&entities.Sudin{}, &entities.Sekolah{})
}

func Down20261116000001AddProfilSekolah(db *gorm.DB) error {
	if err := db.Migrator().DropColumn(&entities.Sekolah{}, "kepala_sekolah"); err != nil {
		return err
	}
	for _, c := range []string{"kontak_unit", "kontak_telepon", "kontak_email", "jam_layanan"} {
		if err := db.Migrator().DropColumn(&entities.Sudin{}, c); err != nil {
			return err
		}
	}
	return nil
}
