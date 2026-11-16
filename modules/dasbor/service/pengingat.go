package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
)

const (
	judulPengingat = "Tenggat terlewat"
	// Tugas yang sama diingatkan lagi paling cepat setelah selang ini.
	selangPengingat = 7 * 24 * time.Hour
	jamPengingat    = 8 // 08.00 WIB
)

var wib = time.FixedZone("WIB", 7*3600)

// KirimPengingatTenggat mengirim notifikasi "tenggat terlewat" ke setiap akun aktif untuk
// tugas di daftar Tugas Saya yang sudah lewat tenggat. Aman dijalankan berulang dan dari
// beberapa instance: satu tugas diingatkan sekali per selangPengingat per akun.
func (s *dasborService) KirimPengingatTenggat(ctx context.Context) (int, error) {
	users, err := s.repo.PenggunaPengingat(ctx)
	if err != nil {
		return 0, err
	}
	terkirim := 0
	sejak := s.now().Add(-selangPengingat)
	_, err = s.repo.DenganKunciPengingat(ctx, func(tx *gorm.DB) error {
		for _, u := range users {
			res, err := s.Tugas(ctx, u, "", "", 0, batasPerModul*10)
			if err != nil {
				return err
			}
			for _, it := range res.Items {
				if it.SisaHari >= 0 {
					continue
				}
				siapa := ""
				if it.Pegawai != nil && *it.Pegawai != "" {
					siapa = " (" + *it.Pegawai + ")"
				}
				pesan := fmt.Sprintf("Tenggat %s %s%s terlewat %d hari: %s.", it.ModulLabel, it.Nomor, siapa, -it.SisaHari, it.Tindakan)
				baru, err := s.repo.BuatPengingat(tx, u.ID, judulPengingat, pesan, it.Tautan, sejak)
				if err != nil {
					return err
				}
				if baru {
					terkirim++
				}
			}
		}
		return nil
	})
	return terkirim, err
}

// JalankanPengingat memanggil KirimPengingatTenggat setiap hari pukul 08.00 WIB sampai ctx selesai.
func (s *dasborService) JalankanPengingat(ctx context.Context) {
	for {
		now := s.now().In(wib)
		next := time.Date(now.Year(), now.Month(), now.Day(), jamPengingat, 0, 0, 0, wib)
		if !next.After(now) {
			next = next.AddDate(0, 0, 1)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(next.Sub(now)):
		}
		n, err := s.KirimPengingatTenggat(ctx)
		if err != nil {
			log.Printf("pengingat tenggat gagal: %v", err)
			continue
		}
		log.Printf("pengingat tenggat: %d notifikasi terkirim", n)
	}
}
