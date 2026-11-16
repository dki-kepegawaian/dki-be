// Package spek mendefinisikan modul yang tampil di dasbor lintas modul. Dasbor menyajikan tampilan lintas modul (T1 Tugas Saya, D02/H1/B1 beranda,
// D03 Daftar Pengajuan, B2 Rekap) dari tabel modul yang sudah ada, tanpa tabel sendiri.
package spek

import "github.com/Caknoooo/go-gin-clean-starter/pkg/constants"

const (
	KategoriPerluTindakan = "perlu_tindakan"
	KategoriMenungguFisik = "menunggu_fisik"
	KategoriMenungguTTD   = "menunggu_ttd"

	KelompokPerluPerbaikan = "perlu_perbaikan"
	KelompokMenungguFisik  = "menunggu_surat_fisik"
	KelompokDiproses       = "diproses"
	KelompokSelesai        = "selesai"
)

// Tugas: status yang menunggu tindakan satu peran, dengan tenggat N hari kerja sejak perubahan terakhir.
type Tugas struct {
	Status, Peran, Tindakan, Kategori string
	HariKerja                         int
}

// Modul menjelaskan satu tabel pengajuan. Semua tabel punya kolom id, nomor, status,
// pendidik_id, sekolah_id, created_at, updated_at; status final tidak berubah lagi,
// sehingga updated_at status final = waktu selesai.
type Modul struct {
	Kode, Label, Tabel string
	// Pengajuan: ikut Daftar Pengajuan dan total "semua pengajuan" di rekap (B2).
	Pengajuan                bool
	Final, Perbaikan, Fisik  []string
	Tugas                    []Tugas
	PunyaTenggat             bool // kolom tenggat sendiri (disiplin)
	SembunyikanUntukPimpinan bool // identitas guru tidak tampil di daftar pimpinan (B1)
}

const (
	staf     = constants.ENUM_ROLE_STAF_SUDIN
	operator = constants.ENUM_ROLE_OPERATOR_SEKOLAH
	pimpinan = constants.ENUM_ROLE_PIMPINAN
)

// DaftarModul: urutan juga urutan tampil di rekap.
var DaftarModul = []Modul{
	{
		Kode: "cuti", Label: "Cuti", Tabel: "cuti", Pengajuan: true,
		Final: []string{"disetujui", "ditolak"}, Perbaikan: []string{"perlu_perbaikan"}, Fisik: []string{"menunggu_surat_fisik"},
		Tugas: []Tugas{
			{"diajukan", staf, "Periksa berkas digital", KategoriPerluTindakan, 2},
			{"menunggu_surat_fisik", staf, "Terima surat fisik & nota dinas", KategoriMenungguFisik, 3},
			{"diproses_sudin", staf, "Catat keputusan pejabat", KategoriMenungguTTD, 5},
			{"perlu_perbaikan", operator, "Unggah ulang berkas sesuai catatan Sudin", KategoriPerluTindakan, 3},
			{"menunggu_surat_fisik", operator, "Antar surat fisik & nota dinas ke Sudin", KategoriMenungguFisik, 3},
		},
	},
	{
		Kode: "pensiun", Label: "Pensiun", Tabel: "pensiun", Pengajuan: true,
		Final: []string{"selesai"}, Perbaikan: []string{"perlu_perbaikan"},
		Tugas: []Tugas{
			{"diajukan", staf, "Verifikasi berkas pensiun", KategoriPerluTindakan, 3},
			{"terverifikasi", staf, "Kirim usulan ke BKD", KategoriPerluTindakan, 5},
			{"perlu_perbaikan", operator, "Unggah ulang berkas yang ditolak", KategoriPerluTindakan, 5},
		},
	},
	{
		Kode: "kenaikan_pangkat", Label: "Kenaikan Pangkat", Tabel: "kenaikan_pangkat", Pengajuan: true,
		Final: []string{"selesai"}, Perbaikan: []string{"perlu_perbaikan"},
		Tugas: []Tugas{
			{"diajukan", staf, "Verifikasi berkas usulan", KategoriPerluTindakan, 3},
			{"terverifikasi", staf, "Kirim usulan ke BKD", KategoriPerluTindakan, 5},
			{"perlu_perbaikan", operator, "Unggah ulang berkas yang ditolak", KategoriPerluTindakan, 5},
		},
	},
	{
		Kode: "disiplin", Label: "Sikap Disiplin", Tabel: "disiplin", Pengajuan: true, PunyaTenggat: true, SembunyikanUntukPimpinan: true,
		Final: []string{"selesai", "tidak_dilanjutkan"}, Perbaikan: []string{"perlu_kelengkapan"},
		Tugas: []Tugas{
			{"registrasi", staf, "Verifikasi awal kasus", KategoriPerluTindakan, 3},
			{"pemeriksaan", staf, "Pemeriksaan & rekomendasi hukuman", KategoriPerluTindakan, 10},
			{"sanksi_ditetapkan", staf, "Catat keputusan & sampaikan", KategoriPerluTindakan, 5},
			{"tindak_lanjut", staf, "Pantau tindak lanjut & tutup kasus", KategoriPerluTindakan, 10},
			{"perlu_kelengkapan", operator, "Lengkapi laporan kasus", KategoriPerluTindakan, 5},
			{"menunggu_keputusan", pimpinan, "Tetapkan keputusan", KategoriPerluTindakan, 5},
		},
	},
	{
		Kode: "dapodik", Label: "Layanan Dapodik", Tabel: "dapodik_tiket", Pengajuan: true, PunyaTenggat: true,
		Final: []string{"selesai"}, Perbaikan: []string{"perlu_info_tambahan"},
		Tugas: []Tugas{
			{"baru", staf, "Proses tiket Dapodik", KategoriPerluTindakan, 3},
			{"perlu_info_tambahan", operator, "Lengkapi info tambahan tiket", KategoriPerluTindakan, 3},
		},
	},
	{
		Kode: "kinerja", Label: "Kinerja P3K PW", Tabel: "kinerja_penilaian",
		Final: []string{"final"},
		Tugas: []Tugas{
			{"draf", operator, "Selesaikan nilai & buat dokumen penilaian", KategoriPerluTindakan, 5},
			{"menunggu_ttd", staf, "Unggah penilaian bertanda tangan", KategoriMenungguTTD, 5},
		},
	},
	{
		Kode: "surat", Label: "Generator Surat", Tabel: "surat",
		Final: []string{"final", "dibatalkan"},
		Tugas: []Tugas{
			{"menunggu_ttd", staf, "Unggah pindaian surat bertanda tangan", KategoriMenungguTTD, 3},
		},
	},
}

func ByKode(kode string) (Modul, bool) {
	for _, m := range DaftarModul {
		if m.Kode == kode {
			return m, true
		}
	}
	return Modul{}, false
}

// Kelompok memetakan status modul ke kelompok tab Daftar Pengajuan / C6.
func (m Modul) Kelompok(status string) string {
	switch {
	case has(m.Final, status):
		return KelompokSelesai
	case has(m.Perbaikan, status):
		return KelompokPerluPerbaikan
	case has(m.Fisik, status):
		return KelompokMenungguFisik
	default:
		return KelompokDiproses
	}
}

// StatusKelompok: kebalikan Kelompok; nil = semua status.
func (m Modul) StatusKelompok(k string, semua []string) []string {
	var out []string
	for _, s := range semua {
		if m.Kelompok(s) == k {
			out = append(out, s)
		}
	}
	return out
}

func has(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}
