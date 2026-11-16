package hitung

import (
	"testing"
	"time"
)

func tgl(s string) *time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return &t
}

func TestTMTPensiun(t *testing.T) {
	cases := []struct{ lahir, want string }{
		{"1967-03-05", "2027-04-01"},
		{"1967-03-01", "2027-04-01"},
		{"1966-12-20", "2027-01-01"}, // Desember → Januari tahun berikutnya
		{"1964-02-29", "2024-03-01"},
	}
	for _, c := range cases {
		got := TMTPensiun(*tgl(c.lahir), 60).Format("2006-01-02")
		if got != c.want {
			t.Errorf("TMTPensiun(%s) = %s, want %s", c.lahir, got, c.want)
		}
	}
}

func TestDikeluarkan(t *testing.T) {
	batas := *tgl("2027-07-31")
	if !Dikeluarkan(tgl("1967-06-15"), batas) { // TMT 2027-07-01
		t.Error("TMT sebelum batas harus dikeluarkan")
	}
	if !Dikeluarkan(tgl("1967-06-30"), *tgl("2027-07-01")) { // TMT tepat sama dengan batas
		t.Error("TMT sama dengan batas harus dikeluarkan")
	}
	if Dikeluarkan(tgl("1967-07-01"), batas) { // TMT 2027-08-01
		t.Error("TMT setelah batas tetap dihitung")
	}
	if Dikeluarkan(nil, batas) {
		t.Error("tanpa tanggal lahir tetap dihitung")
	}
}

func TestRombelDihitung(t *testing.T) {
	cases := []struct{ rombel, siswa, rasio, want int }{
		{9, 288, 32, 9},  // pas rasio
		{10, 288, 32, 9}, // rombel kecil-kecil dibatasi rasio
		{8, 288, 32, 8},  // rombel padat: pakai rombel riil
		{12, 0, 28, 12},  // tanpa data siswa
		{3, 70, 0, 3},    // tanpa rasio
		{3, 65, 32, 3},   // ceil(65/32) = 3
	}
	for _, c := range cases {
		if got := RombelDihitung(c.rombel, c.siswa, c.rasio); got != c.want {
			t.Errorf("RombelDihitung(%d,%d,%d) = %d, want %d", c.rombel, c.siswa, c.rasio, got, c.want)
		}
	}
}

func baris(v VersiHasil, mapel string) (BarisMapel, bool) {
	for _, b := range v.Mapel {
		if b.Mapel == mapel {
			return b, true
		}
	}
	return BarisMapel{}, false
}

func cek(t *testing.T, v VersiHasil, mapel string, kebutuhan, tersedia int) {
	t.Helper()
	b, ok := baris(v, mapel)
	if !ok {
		t.Fatalf("mapel %s tidak ada di hasil %+v", mapel, v.Mapel)
	}
	if b.Kebutuhan != kebutuhan || b.Tersedia != tersedia || b.Selisih != tersedia-kebutuhan {
		t.Errorf("%s = %+v, want kebutuhan %d tersedia %d", mapel, b, kebutuhan, tersedia)
	}
}

func TestHitungSMP(t *testing.T) {
	p := Param{
		Jenjang: "SMP", JamWajib: 24, Rasio: 32, BatasTMTPensiun: *tgl("2027-07-31"),
		Beban: []BebanInput{
			{"IPA", 7, 5}, {"IPA", 8, 5}, {"IPA", 9, 5},
			{"Bahasa Indonesia", 7, 6}, {"Bahasa Indonesia", 8, 6}, {"Bahasa Indonesia", 9, 6},
			{"PJOK", 7, 3}, {"PJOK", 8, 3}, {"PJOK", 9, 3},
			{"Prakarya", 7, 2}, {"Prakarya", 8, 2}, {"Prakarya", 9, 2},
		},
	}
	rombel := []RombelInput{{9, 3, 96}, {7, 3, 96}, {8, 3, 96}}
	guru := []GuruInput{
		{Nama: "A", MapelSK: "IPA", MapelRiil: "IPA"},
		{Nama: "B", MapelSK: "IPA", MapelRiil: "IPA", TanggalLahir: tgl("1967-03-05")}, // pensiun 2027-04-01
		{Nama: "C", MapelSK: "ipa ", MapelRiil: ""},                                    // riil kosong → pakai SK
		{Nama: "D", MapelSK: "Bahasa Indonesia", MapelRiil: "Prakarya"},
		{Nama: "E", MapelSK: "PJOK", MapelRiil: "PJOK", TanggalLahir: tgl("1990-01-01")},
		{Nama: "F", MapelSK: "Seni Budaya", MapelRiil: "Seni Budaya"},
	}
	h := Sekolah(p, rombel, guru)

	if h.GuruTotal != 6 || h.GuruDihitung != 5 || h.GuruPensiun != 1 {
		t.Errorf("guru total/dihitung/pensiun = %d/%d/%d", h.GuruTotal, h.GuruDihitung, h.GuruPensiun)
	}
	if h.TotalRombel != 9 || h.TotalSiswa != 288 || h.Rombel[0].Tingkat != 7 || h.Rombel[0].RataRata != 32 {
		t.Errorf("rombel = %+v", h.Rombel)
	}
	// IPA: 9 rombel × 5 JP = 45 / 24 → 2
	cek(t, h.SK, "IPA", 2, 2)
	// B. Indonesia: 9 × 6 = 54 / 24 → 3
	cek(t, h.SK, "Bahasa Indonesia", 3, 1)
	cek(t, h.Riil, "Bahasa Indonesia", 3, 0)
	// PJOK: 27 / 24 → 2; Prakarya: 18 / 24 → 1
	cek(t, h.SK, "PJOK", 2, 1)
	cek(t, h.SK, "Prakarya", 1, 0)
	cek(t, h.Riil, "Prakarya", 1, 1)
	// Mapel tanpa beban tetap tampil sebagai kelebihan
	cek(t, h.SK, "Seni Budaya", 0, 1)

	if h.SK.Kurang != 4 || h.SK.Lebih != 1 {
		t.Errorf("SK kurang/lebih = %d/%d, want 4/1", h.SK.Kurang, h.SK.Lebih)
	}
	if h.Riil.Kurang != 4 || h.Riil.Lebih != 1 {
		t.Errorf("riil kurang/lebih = %d/%d, want 4/1", h.Riil.Kurang, h.Riil.Lebih)
	}
	if len(h.Perbedaan) != 1 || h.Perbedaan[0].Nama != "D" {
		t.Errorf("perbedaan = %+v", h.Perbedaan)
	}
	if h.SK.Mapel[0].Mapel != "IPA" {
		t.Errorf("urutan mapel mengikuti parameter beban, dapat %s", h.SK.Mapel[0].Mapel)
	}
}

func TestHitungSDGuruKelas(t *testing.T) {
	p := Param{
		Jenjang: "SD", JamWajib: 24, Rasio: 28, BatasTMTPensiun: *tgl("2027-07-31"),
		Beban: []BebanInput{{"PJOK", 1, 4}, {"PJOK", 2, 4}},
	}
	rombel := []RombelInput{{1, 2, 56}, {2, 3, 56}} // tingkat 2: ceil(56/28)=2 rombel dihitung
	guru := []GuruInput{
		{Nama: "a", MapelSK: "Guru Kelas", MapelRiil: "Guru Kelas"},
		{Nama: "b", MapelSK: "guru kelas", MapelRiil: "Guru Kelas"},
		{Nama: "c", MapelSK: "PJOK", MapelRiil: "PJOK"},
	}
	h := Sekolah(p, rombel, guru)
	cek(t, h.SK, MapelGuruKelas, 4, 2)
	cek(t, h.SK, "PJOK", 1, 1) // (2+2) × 4 = 16 / 24 → 1
	if h.SK.Mapel[0].Mapel != MapelGuruKelas {
		t.Errorf("guru kelas harus baris pertama, dapat %+v", h.SK.Mapel)
	}
	if h.Rombel[1].RombelDihitung != 2 || h.TotalRombel != 5 {
		t.Errorf("rombel = %+v total %d", h.Rombel, h.TotalRombel)
	}
}

func TestHitungKosong(t *testing.T) {
	h := Sekolah(Param{Jenjang: "SMA", JamWajib: 24}, nil, nil)
	if h.SK.Mapel == nil || len(h.SK.Mapel) != 0 || h.Perbedaan == nil || h.Rombel == nil {
		t.Errorf("hasil kosong harus slice kosong, bukan nil: %+v", h)
	}
}

func TestRekomendasikan(t *testing.T) {
	sekolah := []SekolahSelisih{
		{SekolahID: "1", Nama: "SMA 33", Kecamatan: "Cengkareng", Mapel: []BarisMapel{{Mapel: "IPA", Selisih: 2}, {Mapel: "PJOK", Selisih: -1}}},
		{SekolahID: "2", Nama: "SD Rawa Buaya", Kecamatan: "Cengkareng", Mapel: []BarisMapel{{Mapel: "IPA", Selisih: -1}}},
		{SekolahID: "3", Nama: "SMP 45", Kecamatan: "Kebon Jeruk", Mapel: []BarisMapel{{Mapel: "ipa", Selisih: -3}, {Mapel: "PJOK", Selisih: 1}}},
		{SekolahID: "4", Nama: "SMA 16", Kecamatan: "Kebon Jeruk", Mapel: []BarisMapel{{Mapel: "IPA", Selisih: 1}}},
		{SekolahID: "5", Nama: "SMK 13", Kecamatan: "Kembangan", Mapel: []BarisMapel{{Mapel: "Matematika", Selisih: 1}}},
	}
	got := Rekomendasikan(sekolah)
	// IPA: tujuan SMP 45 (kurang 3) dilayani dulu: SMA 16 (kecamatan sama, 1) lalu SMA 33 (2).
	// SD Rawa Buaya tidak kebagian. PJOK: SMP 45 → SMA 33 (1). Matematika: tidak ada yang kurang.
	want := []struct {
		mapel, dari, ke string
		n               int
		sama            bool
	}{
		{"IPA", "SMA 16", "SMP 45", 1, true},
		{"IPA", "SMA 33", "SMP 45", 2, false},
		{"PJOK", "SMP 45", "SMA 33", 1, false},
	}
	if len(got) != len(want) {
		t.Fatalf("dapat %d rekomendasi, want %d: %+v", len(got), len(want), got)
	}
	for i, w := range want {
		g := got[i]
		if g.Mapel != w.mapel || g.Dari.Nama != w.dari || g.Ke.Nama != w.ke || g.Jumlah != w.n || g.KecamatanSama != w.sama {
			t.Errorf("rekomendasi %d = %+v, want %+v", i, g, w)
		}
	}
	if got[1].Dari.Jumlah != 2 || got[1].Ke.Jumlah != 3 {
		t.Errorf("jumlah lebih/kurang awal salah: %+v", got[1])
	}
}
