package helpers

import "time"

// HariKerja menghitung hari Senin–Jumat dari mulai sampai selesai (inklusif).
// Libur nasional dan cuti bersama belum diperhitungkan.
func HariKerja(mulai, selesai time.Time) int {
	if selesai.Before(mulai) {
		return 0
	}
	n := 0
	for d := mulai; !d.After(selesai); d = d.AddDate(0, 0, 1) {
		if wd := d.Weekday(); wd != time.Saturday && wd != time.Sunday {
			n++
		}
	}
	return n
}
