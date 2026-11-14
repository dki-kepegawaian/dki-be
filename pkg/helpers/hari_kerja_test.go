package helpers

import (
	"testing"
	"time"
)

func tgl(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}

func TestHariKerja(t *testing.T) {
	cases := []struct {
		name           string
		mulai, selesai string
		want           int
	}{
		{"Senin-Jumat (contoh C1)", "2026-11-02", "2026-11-06", 5},
		{"melewati akhir pekan", "2026-10-28", "2026-11-06", 8},
		{"satu hari kerja", "2026-11-03", "2026-11-03", 1},
		{"Sabtu-Minggu saja", "2026-11-07", "2026-11-08", 0},
		{"selesai sebelum mulai", "2026-11-06", "2026-11-02", 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := HariKerja(tgl(c.mulai), tgl(c.selesai)); got != c.want {
				t.Fatalf("HariKerja(%s, %s) = %d, want %d", c.mulai, c.selesai, got, c.want)
			}
		})
	}
}
