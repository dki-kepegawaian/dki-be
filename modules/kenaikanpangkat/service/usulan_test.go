package service

import (
	"testing"
	"time"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
)

func tgl(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func TestMasaKerja(t *testing.T) {
	for _, c := range []struct {
		tmt, sampai  time.Time
		tahun, bulan int
	}{
		{tgl(2023, 4, 1), tgl(2027, 4, 1), 4, 0},
		{tgl(2022, 4, 1), tgl(2027, 4, 1), 5, 0},
		{tgl(2023, 10, 1), tgl(2027, 4, 1), 3, 6},
		{tgl(2023, 4, 15), tgl(2027, 4, 1), 3, 11}, // bulan terakhir belum penuh
		{tgl(2028, 1, 1), tgl(2027, 4, 1), 0, 0},   // TMT di masa depan
	} {
		got := masaKerja(c.tmt, c.sampai)
		if got.Tahun != c.tahun || got.Bulan != c.bulan {
			t.Errorf("masaKerja(%s, %s) = %d th %d bln, mau %d th %d bln", c.tmt.Format("2006-01-02"), c.sampai.Format("2006-01-02"), got.Tahun, got.Bulan, c.tahun, c.bulan)
		}
	}
}

func TestUsulanSyaratEmpatTahun(t *testing.T) {
	periode := tgl(2027, 4, 1)
	guru := func(tmt *time.Time) entities.Pendidik {
		return entities.Pendidik{StatusKepegawaian: entities.StatusKepegawaianPNS, Golongan: "III/b", TMTGolongan: tmt}
	}
	tepat, kurang := tgl(2023, 4, 1), tgl(2023, 4, 2)

	for nama, c := range map[string]struct {
		p       entities.Pendidik
		kendala bool
	}{
		"tepat 4 tahun boleh":          {guru(&tepat), false},
		"kurang sehari ditolak":        {guru(&kurang), true},
		"tanpa tmt_golongan tak dicek": {guru(nil), false},
	} {
		_, gol, _, kendala := usulan(c.p, periode)
		if (len(kendala) > 0) != c.kendala {
			t.Errorf("%s: kendala = %v", nama, kendala)
		}
		if gol != "III/c" {
			t.Errorf("%s: golongan baru = %q, mau III/c", nama, gol)
		}
	}
}
