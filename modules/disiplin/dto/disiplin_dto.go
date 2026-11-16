package dto

import (
	"mime/multipart"
	"time"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
)

const TglFormat = "2006-01-02"

// Hukuman adalah satu jenis hukuman disiplin (PP 94/2021 Pasal 8).
type Hukuman struct {
	Tingkat string
	Label   string
	Bulan   int // lama hukuman; 0 = tidak bermasa
}

var KatalogHukuman = map[string]Hukuman{
	"teguran_lisan":         {entities.DisiplinTingkatRingan, "Teguran lisan", 0},
	"teguran_tertulis":      {entities.DisiplinTingkatRingan, "Teguran tertulis", 0},
	"pernyataan_tidak_puas": {entities.DisiplinTingkatRingan, "Pernyataan tidak puas secara tertulis", 0},
	"potong_tukin_6_bulan":  {entities.DisiplinTingkatSedang, "Pemotongan tunjangan kinerja 25% selama 6 bulan", 6},
	"potong_tukin_9_bulan":  {entities.DisiplinTingkatSedang, "Pemotongan tunjangan kinerja 25% selama 9 bulan", 9},
	"potong_tukin_12_bulan": {entities.DisiplinTingkatSedang, "Pemotongan tunjangan kinerja 25% selama 12 bulan", 12},
	"penurunan_jabatan":     {entities.DisiplinTingkatBerat, "Penurunan jabatan setingkat lebih rendah selama 12 bulan", 12},
	"pembebasan_jabatan":    {entities.DisiplinTingkatBerat, "Pembebasan dari jabatan menjadi jabatan pelaksana selama 12 bulan", 12},
	"pemberhentian":         {entities.DisiplinTingkatBerat, "Pemberhentian dengan hormat tidak atas permintaan sendiri", 0},
}

var LabelPelanggaran = map[string]string{
	"ketidakhadiran":          "Ketidakhadiran tanpa keterangan",
	"jam_kerja":               "Tidak menaati ketentuan jam kerja",
	"target_kinerja":          "Tidak mencapai target kinerja",
	"penyalahgunaan_wewenang": "Penyalahgunaan wewenang",
	"gratifikasi":             "Menerima hadiah atau gratifikasi terkait jabatan",
	"netralitas":              "Tidak netral dalam pemilu atau pemilihan kepala daerah",
	"barang_milik_negara":     "Penyalahgunaan barang milik negara",
	"lainnya":                 "Pelanggaran kewajiban atau larangan lainnya",
}

type (
	// LaporRequest (DS1), multipart/form-data. Kronologi berupa JSON array
	// [{"tanggal":"2026-03-13","kejadian":"..."}].
	LaporRequest struct {
		PendidikID       string                  `form:"pendidik_id" binding:"required,uuid"`
		JenisPelanggaran string                  `form:"jenis_pelanggaran" binding:"required,oneof=ketidakhadiran jam_kerja target_kinerja penyalahgunaan_wewenang gratifikasi netralitas barang_milik_negara lainnya"`
		Uraian           string                  `form:"uraian" binding:"required,max=5000"`
		Kronologi        string                  `form:"kronologi" binding:"required,max=20000"`
		Pernyataan       bool                    `form:"pernyataan"`
		BuktiAwal        *multipart.FileHeader   `form:"bukti_awal" binding:"required"`
		BuktiPendukung   []*multipart.FileHeader `form:"bukti_pendukung" binding:"max=5"`
	}

	// VerifikasiRequest (DS2).
	VerifikasiRequest struct {
		Keputusan        string `json:"keputusan" binding:"required,oneof=layak perlu_kelengkapan tidak_layak"`
		CekIdentitas     *bool  `json:"identitas_sesuai" binding:"required"`
		CekKronologi     *bool  `json:"kronologi_jelas" binding:"required"`
		CekBukti         *bool  `json:"bukti_memadai" binding:"required"`
		CekKewenangan    *bool  `json:"dalam_kewenangan" binding:"required"`
		Catatan          string `json:"catatan" binding:"max=2000"`
		BatasKelengkapan string `json:"batas_kelengkapan" binding:"omitempty,datetime=2006-01-02"`
	}

	// KelengkapanRequest (DS3), multipart/form-data.
	KelengkapanRequest struct {
		Tanggapan string                  `form:"tanggapan" binding:"required,max=2000"`
		Berkas    []*multipart.FileHeader `form:"berkas" binding:"required,min=1,max=5"`
	}

	// PemeriksaanRequest (DS4), multipart/form-data. Boleh disimpan berulang selama
	// status pemeriksaan; berkas yang dikirim ditambahkan.
	PemeriksaanRequest struct {
		NomorSuratPanggilan  string                `form:"nomor_surat_panggilan" binding:"required,max=100"`
		WaktuPemeriksaan     string                `form:"waktu_pemeriksaan" binding:"required,datetime=2006-01-02T15:04"`
		RingkasanPemeriksaan string                `form:"ringkasan_pemeriksaan" binding:"max=5000"`
		BAP                  *multipart.FileHeader `form:"bap"`
		TanggapanPegawai     *multipart.FileHeader `form:"tanggapan_pegawai"`
		BuktiTambahan        *multipart.FileHeader `form:"bukti_tambahan"`
	}

	// RekomendasiRequest (DS5).
	RekomendasiRequest struct {
		RingkasanFakta string `json:"ringkasan_fakta" binding:"required,max=5000"`
		DugaanTerbukti string `json:"dugaan_terbukti" binding:"required,max=500"`
		DasarHukum     string `json:"dasar_hukum" binding:"required,max=500"`
		Tingkat        string `json:"tingkat" binding:"required,oneof=ringan sedang berat"`
		JenisHukuman   string `json:"jenis_hukuman" binding:"required,oneof=teguran_lisan teguran_tertulis pernyataan_tidak_puas potong_tukin_6_bulan potong_tukin_9_bulan potong_tukin_12_bulan penurunan_jabatan pembebasan_jabatan pemberhentian"`
		Justifikasi    string `json:"justifikasi" binding:"required,max=5000"`
	}

	// KeputusanRequest (DS6, pimpinan).
	KeputusanRequest struct {
		Keputusan    string `json:"keputusan" binding:"required,oneof=setujui ubah kembalikan"`
		Tingkat      string `json:"tingkat" binding:"omitempty,oneof=ringan sedang berat"`
		JenisHukuman string `json:"jenis_hukuman" binding:"omitempty,oneof=teguran_lisan teguran_tertulis pernyataan_tidak_puas potong_tukin_6_bulan potong_tukin_9_bulan potong_tukin_12_bulan penurunan_jabatan pembebasan_jabatan pemberhentian"`
		Catatan      string `json:"catatan" binding:"max=2000"`
	}

	// CatatKeputusanRequest (DS7 bagian keputusan final), multipart/form-data.
	CatatKeputusanRequest struct {
		NomorKeputusan   string                `form:"nomor_keputusan" binding:"required,max=100"`
		TanggalKeputusan string                `form:"tanggal_keputusan" binding:"required,datetime=2006-01-02"`
		MasaMulai        string                `form:"masa_mulai" binding:"omitempty,datetime=2006-01-02"`
		SalinanKeputusan *multipart.FileHeader `form:"salinan_keputusan" binding:"required"`
	}

	// TindakLanjutRequest (DS7 bagian tindak lanjut), multipart/form-data. Field kosong tidak mengubah data.
	TindakLanjutRequest struct {
		TanggalDiterimaPegawai string                `form:"tanggal_diterima_pegawai" binding:"omitempty,datetime=2006-01-02"`
		BuktiPenyampaian       *multipart.FileHeader `form:"bukti_penyampaian"`
		DicatatIDIS            *bool                 `form:"dicatat_idis"`
		UpayaAdministratif     string                `form:"upaya_administratif" binding:"omitempty,oneof=tidak_ada ada"`
		CatatanUpaya           string                `form:"catatan_upaya" binding:"max=2000"`
	}

	// LaporanPelaksanaanRequest (DS8, operator), multipart/form-data.
	LaporanPelaksanaanRequest struct {
		Catatan string                `form:"catatan" binding:"required,max=2000"`
		Berkas  *multipart.FileHeader `form:"berkas" binding:"required"`
	}

	TutupRequest struct {
		Catatan string `json:"catatan" binding:"max=2000"`
	}

	Filter struct {
		Status    []string
		SekolahID string
		Tahun     int
		Search    string
		Offset    int
		Limit     int
	}
)

type (
	PendidikRingkas struct {
		ID                string  `json:"id"`
		Nama              string  `json:"nama"`
		NIP               *string `json:"nip"`
		StatusKepegawaian string  `json:"status_kepegawaian"`
		Pangkat           string  `json:"pangkat"`
		Golongan          string  `json:"golongan"`
		Jabatan           string  `json:"jabatan"`
	}

	SekolahRingkas struct {
		ID   string `json:"id"`
		NPSN string `json:"npsn"`
		Nama string `json:"nama"`
	}

	BerkasResponse struct {
		ID        string    `json:"id"` // id dokumen
		Jenis     string    `json:"jenis"`
		NamaAsli  string    `json:"nama_asli"`
		MimeType  string    `json:"mime_type"`
		Ukuran    int64     `json:"ukuran"`
		URL       string    `json:"url"`
		CreatedAt time.Time `json:"created_at"`
	}

	Kontak struct {
		Nama  string `json:"nama"`
		Email string `json:"email,omitempty"`
	}

	VerifikasiResponse struct {
		Keputusan        string     `json:"keputusan"`
		Catatan          string     `json:"catatan"`
		BatasKelengkapan *string    `json:"batas_kelengkapan"`
		IdentitasSesuai  *bool      `json:"identitas_sesuai,omitempty"`
		KronologiJelas   *bool      `json:"kronologi_jelas,omitempty"`
		BuktiMemadai     *bool      `json:"bukti_memadai,omitempty"`
		DalamKewenangan  *bool      `json:"dalam_kewenangan,omitempty"`
		Oleh             *Kontak    `json:"oleh,omitempty"`
		At               *time.Time `json:"at"`
	}

	PemeriksaanResponse struct {
		NomorSuratPanggilan  string     `json:"nomor_surat_panggilan"`
		WaktuPemeriksaan     *time.Time `json:"waktu_pemeriksaan"`
		RingkasanPemeriksaan string     `json:"ringkasan_pemeriksaan"`
	}

	RekomendasiResponse struct {
		RingkasanFakta string     `json:"ringkasan_fakta"`
		DugaanTerbukti string     `json:"dugaan_terbukti"`
		DasarHukum     string     `json:"dasar_hukum"`
		Tingkat        string     `json:"tingkat"`
		JenisHukuman   string     `json:"jenis_hukuman"`
		JenisLabel     string     `json:"jenis_hukuman_label"`
		Justifikasi    string     `json:"justifikasi"`
		Oleh           *Kontak    `json:"oleh,omitempty"`
		At             *time.Time `json:"at"`
	}

	KeputusanPimpinanResponse struct {
		Keputusan string     `json:"keputusan"`
		Catatan   string     `json:"catatan"`
		Oleh      *Kontak    `json:"oleh,omitempty"`
		At        *time.Time `json:"at"`
	}

	HukumanResponse struct {
		Tingkat          string  `json:"tingkat"`
		JenisHukuman     string  `json:"jenis_hukuman"`
		JenisLabel       string  `json:"jenis_hukuman_label"`
		NomorKeputusan   string  `json:"nomor_keputusan"`
		TanggalKeputusan *string `json:"tanggal_keputusan"`
		MasaMulai        *string `json:"masa_mulai"`
		MasaSelesai      *string `json:"masa_selesai"`
	}

	TindakLanjutResponse struct {
		TanggalDiterimaPegawai *string    `json:"tanggal_diterima_pegawai"`
		DicatatIDIS            bool       `json:"dicatat_idis"`
		IDISOleh               *Kontak    `json:"idis_oleh,omitempty"`
		IDISAt                 *time.Time `json:"idis_at"`
		UpayaAdministratif     string     `json:"upaya_administratif"`
		CatatanUpaya           string     `json:"catatan_upaya"`
		DitutupAt              *time.Time `json:"ditutup_at"`
	}

	RiwayatHukuman struct {
		ID               string  `json:"id"`
		Nomor            string  `json:"nomor"`
		Tingkat          string  `json:"tingkat"`
		JenisLabel       string  `json:"jenis_hukuman_label"`
		TanggalKeputusan *string `json:"tanggal_keputusan"`
	}

	RiwayatResponse struct {
		DariStatus string    `json:"dari_status"`
		KeStatus   string    `json:"ke_status"`
		OlehNama   string    `json:"oleh_nama"`
		Catatan    string    `json:"catatan"`
		CreatedAt  time.Time `json:"created_at"`
	}

	// DisiplinResponse. Bagian internal (pemeriksaan, rekomendasi, keputusan pimpinan,
	// tindak lanjut, checklist verifikasi, riwayat hukuman, berkas internal) dikosongkan
	// untuk operator sekolah.
	DisiplinResponse struct {
		ID                    string                       `json:"id"`
		Nomor                 string                       `json:"nomor"`
		Status                string                       `json:"status"`
		Tahap                 string                       `json:"tahap"`
		Tenggat               *string                      `json:"tenggat"`
		JenisPelanggaran      string                       `json:"jenis_pelanggaran"`
		JenisPelanggaranLabel string                       `json:"jenis_pelanggaran_label"`
		Uraian                string                       `json:"uraian,omitempty"`
		Kronologi             []entities.DisiplinKronologi `json:"kronologi,omitempty"`
		Pendidik              *PendidikRingkas             `json:"pendidik,omitempty"`
		Sekolah               *SekolahRingkas              `json:"sekolah,omitempty"`
		Pelapor               *Kontak                      `json:"pelapor,omitempty"`
		Berkas                []BerkasResponse             `json:"berkas,omitempty"`

		Verifikasi         *VerifikasiResponse        `json:"verifikasi,omitempty"`
		TanggapanSekolah   string                     `json:"tanggapan_sekolah,omitempty"`
		DilengkapiAt       *time.Time                 `json:"dilengkapi_at,omitempty"`
		Pemeriksaan        *PemeriksaanResponse       `json:"pemeriksaan,omitempty"`
		Rekomendasi        *RekomendasiResponse       `json:"rekomendasi,omitempty"`
		KeputusanPimpinan  *KeputusanPimpinanResponse `json:"keputusan_pimpinan,omitempty"`
		Hukuman            *HukumanResponse           `json:"hukuman,omitempty"`
		TindakLanjut       *TindakLanjutResponse      `json:"tindak_lanjut,omitempty"`
		CatatanPelaksanaan string                     `json:"catatan_pelaksanaan,omitempty"`
		PelaksanaanAt      *time.Time                 `json:"pelaksanaan_at,omitempty"`
		PetugasSudin       *Kontak                    `json:"petugas_sudin,omitempty"`
		RiwayatHukuman     []RiwayatHukuman           `json:"riwayat_hukuman,omitempty"`

		Riwayat   []RiwayatResponse `json:"riwayat,omitempty"`
		CreatedAt time.Time         `json:"created_at"`
		UpdatedAt time.Time         `json:"updated_at"`
	}

	RingkasanResponse struct {
		PerStatus         map[string]int64 `json:"per_status"`
		PerluTindakan     int64            `json:"perlu_tindakan"`
		MenungguKeputusan int64            `json:"menunggu_keputusan"`
		TenggatDekat      []string         `json:"tenggat_dekat"`
	}
)

// Pilihan tampilan DTO.
type Tampilan struct {
	Internal      bool // staf Sudin, pimpinan, admin
	SembunyikanID bool // daftar untuk pimpinan: identitas guru hanya di halaman kasus (B1)
	Detail        bool
}

func tgl(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(TglFormat)
	return &s
}

func kontak(u *entities.User, email bool) *Kontak {
	if u == nil {
		return nil
	}
	k := &Kontak{Nama: u.Name}
	if email {
		k.Email = u.Email
	}
	return k
}

func punyaBerkas(d entities.Disiplin, jenis string) bool {
	for _, b := range d.Berkas {
		if b.Jenis == jenis {
			return true
		}
	}
	return false
}

// Tahap adalah label tahap yang dipakai di rekap DS9.
func Tahap(d entities.Disiplin) string {
	switch d.Status {
	case entities.DisiplinStatusRegistrasi:
		return "Verifikasi awal"
	case entities.DisiplinStatusPerluKelengkapan:
		return "Menunggu kelengkapan sekolah"
	case entities.DisiplinStatusPemeriksaan:
		if punyaBerkas(d, entities.DisiplinBerkasBAP) {
			return "Rekomendasi hukuman"
		}
		return "Pemanggilan & BAP"
	case entities.DisiplinStatusMenungguKeputusan:
		return "Keputusan pimpinan"
	case entities.DisiplinStatusSanksiDitetapkan:
		return "Pencatatan keputusan"
	case entities.DisiplinStatusTindakLanjut:
		if punyaBerkas(d, entities.DisiplinBerkasLaporanPelaksanaan) {
			return "Penutupan kasus"
		}
		return "Laporan pelaksanaan"
	case entities.DisiplinStatusSelesai:
		return "Ditutup"
	case entities.DisiplinStatusTidakDilanjutkan:
		return "Verifikasi awal: tidak layak"
	}
	return ""
}

func BerkasURL(disiplinID, dokumenID string) string {
	return "/api/disiplin/" + disiplinID + "/dokumen/" + dokumenID
}

// riwayatUntukSekolah: catatan transisi yang memang ditujukan ke sekolah.
var riwayatUntukSekolah = map[string]bool{
	entities.DisiplinStatusRegistrasi:       true,
	entities.DisiplinStatusPerluKelengkapan: true,
	entities.DisiplinStatusTidakDilanjutkan: true,
}

func FromEntity(d entities.Disiplin, t Tampilan, riwayatHukuman []entities.Disiplin) DisiplinResponse {
	id := d.ID.String()
	r := DisiplinResponse{
		ID:                    id,
		Nomor:                 d.Nomor,
		Status:                d.Status,
		Tahap:                 Tahap(d),
		Tenggat:               tgl(d.Tenggat),
		JenisPelanggaran:      d.JenisPelanggaran,
		JenisPelanggaranLabel: LabelPelanggaran[d.JenisPelanggaran],
		CreatedAt:             d.CreatedAt,
		UpdatedAt:             d.UpdatedAt,
	}
	if entities.DisiplinStatusFinal(d.Status) {
		r.Tenggat = nil
	}
	if p := d.Pendidik; p != nil && !t.SembunyikanID {
		r.Pendidik = &PendidikRingkas{
			ID: p.ID.String(), Nama: p.Nama, NIP: p.NIP, StatusKepegawaian: p.StatusKepegawaian,
			Pangkat: p.Pangkat, Golongan: p.Golongan, Jabatan: p.Jabatan,
		}
	}
	if s := d.Sekolah; s != nil {
		r.Sekolah = &SekolahRingkas{ID: s.ID.String(), NPSN: s.NPSN, Nama: s.Nama}
	}
	// Tingkat/jenis hukuman hanya terisi setelah pimpinan menetapkan (DS6); sekolah diberi tahu.
	if d.JenisHukuman != "" {
		r.Hukuman = &HukumanResponse{
			Tingkat: d.Tingkat, JenisHukuman: d.JenisHukuman, JenisLabel: KatalogHukuman[d.JenisHukuman].Label,
			NomorKeputusan: d.NomorKeputusan, TanggalKeputusan: tgl(d.TanggalKeputusan),
			MasaMulai: tgl(d.MasaMulai), MasaSelesai: tgl(d.MasaSelesai),
		}
	}
	if !t.Detail {
		return r
	}

	r.Uraian = d.Uraian
	r.Kronologi = d.Kronologi
	r.Pelapor = kontak(d.Pelapor, false)
	r.TanggapanSekolah = d.TanggapanSekolah
	r.DilengkapiAt = d.DilengkapiAt
	r.CatatanPelaksanaan = d.CatatanPelaksanaan
	r.PelaksanaanAt = d.PelaksanaanAt
	r.PetugasSudin = kontak(d.Verifikator, true)

	for _, b := range d.Berkas {
		if (b.Internal && !t.Internal) || b.Dokumen == nil {
			continue
		}
		r.Berkas = append(r.Berkas, BerkasResponse{
			ID: b.DokumenID.String(), Jenis: b.Jenis, NamaAsli: b.Dokumen.NamaAsli, MimeType: b.Dokumen.MimeType,
			Ukuran: b.Dokumen.Ukuran, URL: BerkasURL(id, b.DokumenID.String()), CreatedAt: b.CreatedAt,
		})
	}

	if d.KeputusanVerifikasi != "" {
		v := &VerifikasiResponse{
			Keputusan: d.KeputusanVerifikasi, BatasKelengkapan: tgl(d.BatasKelengkapan),
			Oleh: kontak(d.Verifikator, false), At: d.DiverifikasiAt,
		}
		if t.Internal || d.KeputusanVerifikasi != "layak" {
			v.Catatan = d.CatatanVerifikasi
		}
		if t.Internal {
			v.IdentitasSesuai, v.KronologiJelas, v.BuktiMemadai, v.DalamKewenangan =
				d.CekIdentitas, d.CekKronologi, d.CekBukti, d.CekKewenangan
		}
		r.Verifikasi = v
	}

	if t.Internal {
		if d.NomorSuratPanggilan != "" {
			r.Pemeriksaan = &PemeriksaanResponse{
				NomorSuratPanggilan: d.NomorSuratPanggilan, WaktuPemeriksaan: d.WaktuPemeriksaan,
				RingkasanPemeriksaan: d.RingkasanPemeriksaan,
			}
		}
		if d.RekomJenis != "" {
			r.Rekomendasi = &RekomendasiResponse{
				RingkasanFakta: d.RingkasanFakta, DugaanTerbukti: d.DugaanTerbukti, DasarHukum: d.DasarHukum,
				Tingkat: d.RekomTingkat, JenisHukuman: d.RekomJenis, JenisLabel: KatalogHukuman[d.RekomJenis].Label,
				Justifikasi: d.Justifikasi, Oleh: kontak(d.Perekomendasi, false), At: d.DirekomendasikanAt,
			}
		}
		if d.KeputusanPimpinan != "" {
			r.KeputusanPimpinan = &KeputusanPimpinanResponse{
				Keputusan: d.KeputusanPimpinan, Catatan: d.CatatanPimpinan,
				Oleh: kontak(d.Pemutus, false), At: d.DiputuskanAt,
			}
		}
		if d.Status == entities.DisiplinStatusTindakLanjut || d.Status == entities.DisiplinStatusSelesai {
			r.TindakLanjut = &TindakLanjutResponse{
				TanggalDiterimaPegawai: tgl(d.TanggalDiterimaPegawai), DicatatIDIS: d.DicatatIDIS,
				IDISOleh: kontak(d.PencatatIDIS, false), IDISAt: d.IDISAt,
				UpayaAdministratif: d.UpayaAdministratif, CatatanUpaya: d.CatatanUpaya, DitutupAt: d.DitutupAt,
			}
		}
		r.RiwayatHukuman = []RiwayatHukuman{}
		for _, h := range riwayatHukuman {
			r.RiwayatHukuman = append(r.RiwayatHukuman, RiwayatHukuman{
				ID: h.ID.String(), Nomor: h.Nomor, Tingkat: h.Tingkat,
				JenisLabel: KatalogHukuman[h.JenisHukuman].Label, TanggalKeputusan: tgl(h.TanggalKeputusan),
			})
		}
	}

	for _, h := range d.Riwayat {
		item := RiwayatResponse{DariStatus: h.DariStatus, KeStatus: h.KeStatus, CreatedAt: h.CreatedAt}
		if t.Internal || riwayatUntukSekolah[h.KeStatus] {
			item.Catatan = h.Catatan
		}
		if h.Pengguna != nil {
			item.OlehNama = h.Pengguna.Name
		}
		r.Riwayat = append(r.Riwayat, item)
	}
	return r
}
