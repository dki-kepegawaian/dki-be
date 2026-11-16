package service

import (
	"fmt"
	"math"
	"strings"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"github.com/Caknoooo/go-gin-clean-starter/modules/kinerja/dto"
)

// aspek adalah tujuh parameter BerAKHLAK (Figma KP2), urut seperti di layar.
type aspek struct {
	Nama   string
	Uraian string
	Nilai  *int
}

func daftarAspek(k *entities.KinerjaPenilaian) []aspek {
	return []aspek{
		{"Berorientasi Pelayanan", "Memahami dan memenuhi kebutuhan warga sekolah dengan ramah.", k.BerorientasiPelayanan},
		{"Akuntabel", "Melaksanakan tugas dengan jujur, cermat, dan bertanggung jawab.", k.Akuntabel},
		{"Kompeten", "Terus belajar dan mengembangkan kapabilitas kerja.", k.Kompeten},
		{"Harmonis", "Saling peduli dan menghargai perbedaan.", k.Harmonis},
		{"Loyal", "Berdedikasi dan mengutamakan kepentingan bangsa.", k.Loyal},
		{"Adaptif", "Terus berinovasi dan antusias menghadapi perubahan.", k.Adaptif},
		{"Kolaboratif", "Membangun kerja sama yang sinergis.", k.Kolaboratif},
	}
}

// labelSkor mengikuti skala di KP2: 1 Sangat kurang ... 5 Sangat baik.
var labelSkor = map[int]string{1: "Sangat kurang", 2: "Kurang", 3: "Cukup", 4: "Baik", 5: "Sangat baik"}

// predikat untuk nilai akhir 1 desimal. Rentang Baik (3,5–4,4) dari Figma KP2;
// rentang lain diturunkan dengan lebar yang sama.
func predikat(n float64) string {
	switch {
	case n >= 4.5:
		return "Sangat baik"
	case n >= 3.5:
		return "Baik"
	case n >= 2.5:
		return "Cukup"
	case n >= 1.5:
		return "Kurang"
	default:
		return "Sangat kurang"
	}
}

func terapkanNilai(k *entities.KinerjaPenilaian, n dto.NilaiInput) {
	k.BerorientasiPelayanan, k.Akuntabel, k.Kompeten = n.BerorientasiPelayanan, n.Akuntabel, n.Kompeten
	k.Harmonis, k.Loyal, k.Adaptif, k.Kolaboratif = n.Harmonis, n.Loyal, n.Adaptif, n.Kolaboratif
}

// hitung mengisi NilaiAkhir (rata-rata parameter terisi, 1 desimal) dan Predikat,
// lalu mengembalikan jumlah parameter yang terisi.
func hitung(k *entities.KinerjaPenilaian) int {
	total, terisi := 0, 0
	for _, a := range daftarAspek(k) {
		if a.Nilai != nil {
			total += *a.Nilai
			terisi++
		}
	}
	if terisi == 0 {
		k.NilaiAkhir, k.Predikat = nil, ""
		return 0
	}
	n := math.Round(float64(total)/float64(terisi)*10) / 10
	k.NilaiAkhir, k.Predikat = &n, predikat(n)
	return terisi
}

func jumlahTerisi(k *entities.KinerjaPenilaian) int {
	n := 0
	for _, a := range daftarAspek(k) {
		if a.Nilai != nil {
			n++
		}
	}
	return n
}

// koma memformat angka gaya Indonesia, mis. 4.3 -> "4,3".
func koma(n float64) string {
	return strings.Replace(fmt.Sprintf("%.1f", n), ".", ",", 1)
}
