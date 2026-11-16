package migrations

import (
	"github.com/Caknoooo/go-gin-clean-starter/database"
	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"gorm.io/gorm"
)

func init() {
	database.RegisterMigration("20261115000001_add_user_username_nip", Up20261115000001AddUserUsernameNip, Down20261115000001AddUserUsernameNip)
}

// Menambah username (unik, untuk masuk) dan nip ke users.
func Up20261115000001AddUserUsernameNip(db *gorm.DB) error {
	return db.AutoMigrate(&entities.User{})
}

func Down20261115000001AddUserUsernameNip(db *gorm.DB) error {
	if err := db.Migrator().DropColumn(&entities.User{}, "nip"); err != nil {
		return err
	}
	return db.Migrator().DropColumn(&entities.User{}, "username")
}
