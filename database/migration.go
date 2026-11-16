package database

import (
	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&entities.Migration{},
		&entities.Sudin{},
		&entities.Sekolah{},
		&entities.User{},
		&entities.RefreshToken{},
		&entities.Pendidik{},
		&entities.Dokumen{},
		&entities.Notifikasi{},
		&entities.LogAktivitas{},
		&entities.NomorUrut{},
		&entities.Cuti{},
		&entities.CutiRiwayat{},
		&entities.DapodikTiket{},
		&entities.DapodikLampiran{},
		&entities.DapodikRiwayat{},
		&entities.KinerjaPenilaian{},
		&entities.KinerjaRiwayat{},
		&entities.AnjabPeriode{},
		&entities.AnjabBeban{},
		&entities.AnjabSekolah{},
		&entities.AnjabRombel{},
		&entities.AnjabGuru{},
		&entities.AnjabRiwayat{},
		&entities.Pensiun{},
		&entities.PensiunBerkas{},
		&entities.PensiunRiwayat{},
		&entities.KenaikanPangkat{},
		&entities.KenaikanPangkatBerkas{},
		&entities.KenaikanPangkatRiwayat{},
		&entities.Disiplin{},
		&entities.DisiplinBerkas{},
		&entities.DisiplinRiwayat{},
		&entities.Arsip{},
		&entities.ArsipVersi{},
		&entities.ArsipLog{},
		&entities.Surat{},
		&entities.SuratRiwayat{},
	); err != nil {
		return err
	}

	manager := NewMigrationManager(db)
	return manager.Run()
}
