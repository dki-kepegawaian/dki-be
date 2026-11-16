package hitung

import (
	"sort"
	"strings"
	"time"
)

// Fungsi di file ini murni (tanpa DB) supaya rumus Anjab mudah diuji dan diaudit.

const (
	// MapelGuruKelas dipakai guru kelas SD; kebutuhannya = jumlah rombel.
	MapelGuruKelas = "Guru Kelas"
	// BatasUsiaPensiun guru (tahun).
	BatasUsiaPensiun = 60
)

type (
	BebanInput struct {
		Mapel   string
		Tingkat int
		JP      int
	}
	RombelInput struct {
		Tingkat      int
		JumlahRombel int
		JumlahSiswa  int
	}
	GuruInput struct {
		Nama         string
		MapelSK      string
		MapelRiil    string
		TanggalLahir *time.Time
	}
	Param struct {
		Jenjang         string
		JamWajib        int // JP per minggu
		Rasio           int // siswa per rombel untuk jenjang ini
		BatasTMTPensiun time.Time
		Beban           []BebanInput // beban jenjang ini saja
	}

	BarisMapel struct {
		Mapel     string `json:"mapel"`
		Kebutuhan int    `json:"kebutuhan"`
		Tersedia  int    `json:"tersedia"`
		Selisih   int    `json:"selisih"` // tersedia - kebutuhan; negatif = kurang
	}
	VersiHasil struct {
		Mapel  []BarisMapel `json:"mapel"`
		Kurang int          `json:"kurang"`
		Lebih  int          `json:"lebih"`
	}
	RombelHasil struct {
		Tingkat        int     `json:"tingkat"`
		JumlahRombel   int     `json:"jumlah_rombel"`
		JumlahSiswa    int     `json:"jumlah_siswa"`
		RataRata       float64 `json:"rata_rata"`
		RombelDihitung int     `json:"rombel_dihitung"`
	}
	Perbedaan struct {
		Nama      string `json:"nama"`
		MapelSK   string `json:"mapel_sk"`
		MapelRiil string `json:"mapel_riil"`
	}
	// Hasil hitungan satu sekolah dalam dua basis: mapel menurut SK dan mapel riil.
	Hasil struct {
		GuruTotal    int           `json:"guru_total"`
		GuruDihitung int           `json:"guru_dihitung"`
		GuruPensiun  int           `json:"guru_pensiun"`
		TotalRombel  int           `json:"total_rombel"`
		TotalSiswa   int           `json:"total_siswa"`
		Rombel       []RombelHasil `json:"rombel"`
		SK           VersiHasil    `json:"versi_sk"`
		Riil         VersiHasil    `json:"versi_riil"`
		Perbedaan    []Perbedaan   `json:"perbedaan"`
	}
)

// TMTPensiun: awal bulan berikutnya setelah guru mencapai batas usia.
func TMTPensiun(lahir time.Time, batasUsia int) time.Time {
	y, m, _ := lahir.Date()
	return time.Date(y+batasUsia, m+1, 1, 0, 0, 0, 0, time.UTC) // bulan 13 dinormalisasi ke Januari
}

// Dikeluarkan: true bila TMT pensiun guru jatuh pada atau sebelum batas periode.
// Guru tanpa tanggal lahir tetap dihitung.
func Dikeluarkan(lahir *time.Time, batas time.Time) bool {
	if lahir == nil {
		return false
	}
	b := time.Date(batas.Year(), batas.Month(), batas.Day(), 0, 0, 0, 0, time.UTC)
	return !TMTPensiun(*lahir, BatasUsiaPensiun).After(b)
}

// RombelDihitung membatasi jumlah rombel dengan rasio: rombel yang dihitung tidak lebih
// dari ceil(siswa / rasio). Tanpa data siswa atau rasio, jumlah rombel dipakai apa adanya.
func RombelDihitung(rombel, siswa, rasio int) int {
	if rasio <= 0 || siswa <= 0 {
		return rombel
	}
	return min(rombel, ceilDiv(siswa, rasio))
}

func ceilDiv(a, b int) int {
	if b <= 0 || a <= 0 {
		return 0
	}
	return (a + b - 1) / b
}

// KunciMapel menyamakan penulisan nama mapel (huruf besar/kecil, spasi ganda).
func KunciMapel(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}

// Sekolah menghitung kebutuhan dan ketersediaan guru per mapel.
//
//	kebutuhan mapel   = ceil(Σ_tingkat rombel_dihitung × JP mapel tingkat itu / jam wajib)
//	kebutuhan guru kelas (SD) = Σ rombel_dihitung
//	tersedia          = guru dihitung (bukan pensiun ≤ batas) dengan mapel itu
//
// Versi SK memakai mapel SK, versi riil memakai mapel riil (mapel SK bila riil kosong).
func Sekolah(p Param, rombel []RombelInput, guru []GuruInput) Hasil {
	var h Hasil
	perTingkat := map[int]int{}
	rombel = append([]RombelInput(nil), rombel...)
	sort.SliceStable(rombel, func(i, j int) bool { return rombel[i].Tingkat < rombel[j].Tingkat })
	for _, r := range rombel {
		d := RombelDihitung(r.JumlahRombel, r.JumlahSiswa, p.Rasio)
		perTingkat[r.Tingkat] += d
		rata := 0.0
		if r.JumlahRombel > 0 {
			rata = float64(int(float64(r.JumlahSiswa)/float64(r.JumlahRombel)*10+0.5)) / 10
		}
		h.Rombel = append(h.Rombel, RombelHasil{
			Tingkat: r.Tingkat, JumlahRombel: r.JumlahRombel, JumlahSiswa: r.JumlahSiswa,
			RataRata: rata, RombelDihitung: d,
		})
		h.TotalRombel += r.JumlahRombel
		h.TotalSiswa += r.JumlahSiswa
	}

	// Urutan baris: guru kelas (SD), mapel di parameter beban, lalu mapel lain dari data guru.
	var urutan []string
	nama := map[string]string{}
	tambah := func(m string) string {
		k := KunciMapel(m)
		if k == "" {
			return ""
		}
		if _, ok := nama[k]; !ok {
			nama[k] = strings.Join(strings.Fields(m), " ")
			urutan = append(urutan, k)
		}
		return k
	}

	jp := map[string]int{}
	if p.Jenjang == "SD" {
		tambah(MapelGuruKelas)
	}
	for _, b := range p.Beban {
		if k := tambah(b.Mapel); k != "" {
			jp[k] += perTingkat[b.Tingkat] * b.JP
		}
	}
	kebutuhan := map[string]int{}
	for k, total := range jp {
		kebutuhan[k] = ceilDiv(total, p.JamWajib)
	}
	if p.Jenjang == "SD" {
		total := 0
		for _, n := range perTingkat {
			total += n
		}
		kebutuhan[KunciMapel(MapelGuruKelas)] = total
	}

	tersediaSK, tersediaRiil := map[string]int{}, map[string]int{}
	var lain []string
	for _, g := range guru {
		h.GuruTotal++
		if Dikeluarkan(g.TanggalLahir, p.BatasTMTPensiun) {
			h.GuruPensiun++
			continue
		}
		h.GuruDihitung++
		riil := g.MapelRiil
		if KunciMapel(riil) == "" {
			riil = g.MapelSK
		}
		for _, m := range []string{g.MapelSK, riil} {
			if _, ok := nama[KunciMapel(m)]; !ok && KunciMapel(m) != "" {
				lain = append(lain, m)
			}
		}
		tersediaSK[KunciMapel(g.MapelSK)]++
		tersediaRiil[KunciMapel(riil)]++
		if KunciMapel(riil) != KunciMapel(g.MapelSK) {
			h.Perbedaan = append(h.Perbedaan, Perbedaan{Nama: g.Nama, MapelSK: g.MapelSK, MapelRiil: riil})
		}
	}
	sort.Slice(lain, func(i, j int) bool { return KunciMapel(lain[i]) < KunciMapel(lain[j]) })
	for _, m := range lain {
		tambah(m)
	}

	versi := func(tersedia map[string]int) VersiHasil {
		v := VersiHasil{Mapel: []BarisMapel{}}
		for _, k := range urutan {
			b := BarisMapel{Mapel: nama[k], Kebutuhan: kebutuhan[k], Tersedia: tersedia[k]}
			if b.Kebutuhan == 0 && b.Tersedia == 0 {
				continue
			}
			b.Selisih = b.Tersedia - b.Kebutuhan
			if b.Selisih < 0 {
				v.Kurang -= b.Selisih
			} else {
				v.Lebih += b.Selisih
			}
			v.Mapel = append(v.Mapel, b)
		}
		return v
	}
	h.SK = versi(tersediaSK)
	h.Riil = versi(tersediaRiil)
	if h.Rombel == nil {
		h.Rombel = []RombelHasil{}
	}
	if h.Perbedaan == nil {
		h.Perbedaan = []Perbedaan{}
	}
	return h
}

type (
	// SekolahSelisih adalah hasil (versi riil) satu sekolah yang sudah ditetapkan.
	SekolahSelisih struct {
		SekolahID string
		Nama      string
		Kecamatan string
		Mapel     []BarisMapel
	}
	SekolahRef struct {
		SekolahID string `json:"sekolah_id"`
		Nama      string `json:"nama"`
		Kecamatan string `json:"kecamatan"`
		Jumlah    int    `json:"jumlah"` // lebih (asal) atau kurang (tujuan) di mapel itu
	}
	Rekomendasi struct {
		Mapel         string     `json:"mapel"`
		Dari          SekolahRef `json:"dari"`
		Ke            SekolahRef `json:"ke"`
		Jumlah        int        `json:"jumlah"`
		KecamatanSama bool       `json:"kecamatan_sama"`
	}
)

// Rekomendasikan memasangkan sekolah kelebihan dengan sekolah kekurangan per mapel.
// Sekolah tujuan dengan kekurangan terbesar dilayani dulu; asal di kecamatan yang sama
// diutamakan, lalu asal dengan kelebihan terbesar. Hasilnya saran, bukan keputusan mutasi.
func Rekomendasikan(sekolah []SekolahSelisih) []Rekomendasi {
	type sisi struct {
		idx  int
		awal int
		sisa int
	}
	lebih, kurang := map[string][]*sisi{}, map[string][]*sisi{}
	nama := map[string]string{}
	for i, s := range sekolah {
		for _, m := range s.Mapel {
			k := KunciMapel(m.Mapel)
			if _, ok := nama[k]; !ok {
				nama[k] = m.Mapel
			}
			switch {
			case m.Selisih > 0:
				lebih[k] = append(lebih[k], &sisi{i, m.Selisih, m.Selisih})
			case m.Selisih < 0:
				kurang[k] = append(kurang[k], &sisi{i, -m.Selisih, -m.Selisih})
			}
		}
	}

	keys := make([]string, 0, len(kurang))
	for k := range kurang {
		if len(lebih[k]) > 0 {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	out := []Rekomendasi{}
	for _, k := range keys {
		tujuan := kurang[k]
		sort.SliceStable(tujuan, func(i, j int) bool {
			if tujuan[i].awal != tujuan[j].awal {
				return tujuan[i].awal > tujuan[j].awal
			}
			return sekolah[tujuan[i].idx].Nama < sekolah[tujuan[j].idx].Nama
		})
		for _, t := range tujuan {
			kec := sekolah[t.idx].Kecamatan
			asal := append([]*sisi(nil), lebih[k]...)
			sort.SliceStable(asal, func(i, j int) bool {
				si, sj := sekolah[asal[i].idx].Kecamatan == kec, sekolah[asal[j].idx].Kecamatan == kec
				if si != sj {
					return si
				}
				if asal[i].sisa != asal[j].sisa {
					return asal[i].sisa > asal[j].sisa
				}
				return sekolah[asal[i].idx].Nama < sekolah[asal[j].idx].Nama
			})
			for _, a := range asal {
				if t.sisa == 0 {
					break
				}
				n := min(a.sisa, t.sisa)
				if n == 0 {
					continue
				}
				a.sisa -= n
				t.sisa -= n
				da, ke := sekolah[a.idx], sekolah[t.idx]
				out = append(out, Rekomendasi{
					Mapel:         nama[k],
					Dari:          SekolahRef{SekolahID: da.SekolahID, Nama: da.Nama, Kecamatan: da.Kecamatan, Jumlah: a.awal},
					Ke:            SekolahRef{SekolahID: ke.SekolahID, Nama: ke.Nama, Kecamatan: ke.Kecamatan, Jumlah: t.awal},
					Jumlah:        n,
					KecamatanSama: da.Kecamatan != "" && da.Kecamatan == ke.Kecamatan,
				})
			}
		}
	}
	return out
}
