package seeds

import (
	"errors"
	"log"
	"os"
	"strings"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/constants"
	"gorm.io/gorm"
)

// AdminSeeder membuat akun admin awal dari ADMIN_EMAIL dan ADMIN_PASSWORD bila belum ada.
// Akun yang sudah ada tidak diubah, jadi password yang diganti lewat aplikasi tetap aman.
func AdminSeeder(db *gorm.DB) error {
	email := strings.ToLower(strings.TrimSpace(os.Getenv("ADMIN_EMAIL")))
	password := os.Getenv("ADMIN_PASSWORD")
	if email == "" || password == "" {
		log.Println("ADMIN_EMAIL/ADMIN_PASSWORD kosong, akun admin awal dilewati")
		return nil
	}
	if len(password) < 8 {
		return errors.New("ADMIN_PASSWORD minimal 8 karakter")
	}
	admin := entities.User{
		Name: "Administrator", Email: email, Password: password,
		Role: constants.ENUM_ROLE_ADMIN, IsVerified: true, IsAktif: true,
	}
	return db.Where(entities.User{Email: email}).FirstOrCreate(&admin).Error
}
