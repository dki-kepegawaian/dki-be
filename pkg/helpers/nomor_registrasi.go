package helpers

import (
	"fmt"

	"gorm.io/gorm"
)

// NomorRegistrasi mengambil nomor berikutnya untuk prefix dan tahun secara atomik,
// mis. NomorRegistrasi(tx, "CUT", 2026) -> "CUT-2026-0142". Panggil di dalam transaksi
// yang sama dengan insert pengajuan supaya nomor tidak terbuang saat insert gagal.
func NomorRegistrasi(tx *gorm.DB, prefix string, tahun int) (string, error) {
	var urut int
	err := tx.Raw(`
		INSERT INTO nomor_urut (prefix, tahun, terakhir) VALUES (?, ?, 1)
		ON CONFLICT (prefix, tahun) DO UPDATE SET terakhir = nomor_urut.terakhir + 1
		RETURNING terakhir`, prefix, tahun).Scan(&urut).Error
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%d-%04d", prefix, tahun, urut), nil
}
