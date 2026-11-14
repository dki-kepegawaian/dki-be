package seeds

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type fondasiSeed struct {
	Sudin   []entities.Sudin `json:"sudin"`
	Sekolah []struct {
		entities.Sekolah
		SudinKode string `json:"sudin_kode"`
	} `json:"sekolah"`
	Pendidik []struct {
		entities.Pendidik
		SekolahNPSN string `json:"sekolah_npsn"`
	} `json:"pendidik"`
	Users []struct {
		entities.User
		Password    string `json:"password"`
		SudinKode   string `json:"sudin_kode"`
		SekolahNPSN string `json:"sekolah_npsn"`
	} `json:"users"`
}

// FondasiSeeder mengisi daftar Sudin, dan bila demo=true juga sekolah, pendidik,
// serta akun contoh per peran. Aman dijalankan berulang: data yang sudah ada dilewati.
func FondasiSeeder(db *gorm.DB, demo bool) error {
	raw, err := os.ReadFile("./database/seeders/json/fondasi.json")
	if err != nil {
		return err
	}
	var data fondasiSeed
	if err := json.Unmarshal(raw, &data); err != nil {
		return err
	}

	return db.Transaction(func(tx *gorm.DB) error {
		sudinID := map[string]uuid.UUID{}
		for _, s := range data.Sudin {
			row := s
			if err := tx.Where(entities.Sudin{Kode: s.Kode}).FirstOrCreate(&row).Error; err != nil {
				return err
			}
			sudinID[row.Kode] = row.ID
		}
		if !demo {
			return nil
		}

		sekolahID := map[string]uuid.UUID{}
		for _, s := range data.Sekolah {
			id, ok := sudinID[s.SudinKode]
			if !ok {
				return fmt.Errorf("sudin %q tidak ada untuk sekolah %s", s.SudinKode, s.NPSN)
			}
			row := s.Sekolah
			row.SudinID = id
			if err := tx.Where(entities.Sekolah{NPSN: s.NPSN}).FirstOrCreate(&row).Error; err != nil {
				return err
			}
			sekolahID[row.NPSN] = row.ID
		}

		for _, p := range data.Pendidik {
			id, ok := sekolahID[p.SekolahNPSN]
			if !ok {
				return fmt.Errorf("sekolah %q tidak ada untuk pendidik %s", p.SekolahNPSN, p.Nama)
			}
			row := p.Pendidik
			row.SekolahID = id
			if err := tx.Where(entities.Pendidik{Nama: p.Nama, SekolahID: id}).FirstOrCreate(&row).Error; err != nil {
				return err
			}
		}

		for _, u := range data.Users {
			row := u.User
			row.Password = u.Password
			row.IsVerified, row.IsAktif = true, true
			if u.SudinKode != "" {
				id := sudinID[u.SudinKode]
				row.SudinID = &id
			}
			if u.SekolahNPSN != "" {
				id := sekolahID[u.SekolahNPSN]
				row.SekolahID = &id
			}
			// FirstOrCreate memicu BeforeCreate (hash password) hanya untuk baris baru.
			if err := tx.Where(entities.User{Email: u.Email}).FirstOrCreate(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
