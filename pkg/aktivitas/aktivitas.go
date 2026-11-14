package aktivitas

import (
	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Catat menulis satu baris log_aktivitas. Panggil di transaksi yang sama
// dengan perubahan datanya supaya log dan data selalu konsisten.
func Catat(tx *gorm.DB, userID uuid.UUID, aksi, entitas string, entitasID uuid.UUID, keterangan string) error {
	return tx.Create(&entities.LogAktivitas{
		UserID:     userID,
		Aksi:       aksi,
		Entitas:    entitas,
		EntitasID:  entitasID,
		Keterangan: keterangan,
	}).Error
}
