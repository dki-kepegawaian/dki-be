package service

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"github.com/go-pdf/fpdf"
)

// template satu jenis surat. Teks boleh memuat placeholder {penanda_tangan},
// {nama}, {unit_kerja}, dan {perihal}; isian "keterangan" dari form disisipkan
// apa adanya di antara data pegawai dan penutup.
type template struct {
	Label     string
	Deskripsi string
	Judul     string
	Berkas    string // awalan nama file PDF
	Pembuka   string
	Penutup   string
}

// Urutan dan teks label/deskripsi mengikuti S1 Figma.
var urutanJenis = []string{
	entities.SuratJenisTunjangan,
	entities.SuratJenisAktifMengajar,
	entities.SuratJenisPengantarKenaikanPangkat,
	entities.SuratJenisIzinCuti,
	entities.SuratJenisPengantarPensiun,
}

var templates = map[string]template{
	entities.SuratJenisTunjangan: {
		Label:     "Surat keterangan tunjangan",
		Deskripsi: "Keterangan untuk pengurusan tunjangan profesi atau tunjangan lain guru.",
		Judul:     "SURAT KETERANGAN",
		Berkas:    "SK_Tunjangan",
		Pembuka:   "Yang bertanda tangan di bawah ini, {penanda_tangan}, menerangkan bahwa:",
		Penutup:   "Demikian surat keterangan ini dibuat untuk pengurusan tunjangan dan dipergunakan sebagaimana mestinya.",
	},
	entities.SuratJenisAktifMengajar: {
		Label:     "Surat keterangan aktif mengajar",
		Deskripsi: "Menerangkan guru aktif mengajar di satuan pendidikan tertentu.",
		Judul:     "SURAT KETERANGAN",
		Berkas:    "SK_Aktif_Mengajar",
		Pembuka:   "Yang bertanda tangan di bawah ini, {penanda_tangan}, menerangkan bahwa:",
		Penutup:   "Demikian surat keterangan ini dibuat untuk dipergunakan sebagaimana mestinya.",
	},
	entities.SuratJenisPengantarKenaikanPangkat: {
		Label:     "Surat pengantar kenaikan pangkat",
		Deskripsi: "Pengantar usulan kenaikan pangkat ke BKD Provinsi DKI Jakarta.",
		Judul:     "SURAT PENGANTAR",
		Berkas:    "Pengantar_KP",
		Pembuka:   "Bersama ini {penanda_tangan} menyampaikan usulan kenaikan pangkat pegawai berikut kepada Kepala Badan Kepegawaian Daerah Provinsi DKI Jakarta:",
		Penutup:   "Demikian surat pengantar ini disampaikan untuk diproses lebih lanjut. Atas perhatiannya diucapkan terima kasih.",
	},
	entities.SuratJenisIzinCuti: {
		Label:     "Surat izin cuti",
		Deskripsi: "Izin cuti guru berdasarkan pengajuan cuti yang disetujui.",
		Judul:     "SURAT IZIN CUTI",
		Berkas:    "Izin_Cuti",
		Pembuka:   "Yang bertanda tangan di bawah ini, {penanda_tangan}, memberikan izin cuti kepada:",
		Penutup:   "Demikian surat izin cuti ini dibuat untuk dipergunakan sebagaimana mestinya.",
	},
	entities.SuratJenisPengantarPensiun: {
		Label:     "Surat pengantar pensiun",
		Deskripsi: "Pengantar usulan pensiun guru ke BKD Provinsi DKI Jakarta.",
		Judul:     "SURAT PENGANTAR",
		Berkas:    "Pengantar_Pensiun",
		Pembuka:   "Bersama ini {penanda_tangan} menyampaikan usulan pensiun pegawai berikut kepada Kepala Badan Kepegawaian Daerah Provinsi DKI Jakarta:",
		Penutup:   "Demikian surat pengantar ini disampaikan untuk diproses lebih lanjut. Atas perhatiannya diucapkan terima kasih.",
	},
}

var namaBulan = [...]string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni",
	"Juli", "Agustus", "September", "Oktober", "November", "Desember"}

func tanggalIndonesia(t time.Time) string {
	return fmt.Sprintf("%d %s %d", t.Day(), namaBulan[t.Month()], t.Year())
}

// kop dan penandaTangan diturunkan dari Sudin sekolah si pendidik.
func kop(sudin *entities.Sudin) string {
	if sudin == nil {
		return "Suku Dinas Pendidikan"
	}
	return "Suku Dinas Pendidikan Kota Administrasi " + sudin.Nama
}

func penandaTangan(sudin *entities.Sudin) string {
	return "Kepala " + kop(sudin)
}

// formatNIP: 198706142010012015 -> 19870614 201001 2 015.
func formatNIP(nip *string) string {
	if nip == nil || *nip == "" {
		return "-"
	}
	n := *nip
	if len(n) != 18 {
		return n
	}
	return n[:8] + " " + n[8:14] + " " + n[14:15] + " " + n[15:]
}

func gabung(sep string, bagian ...string) string {
	var isi []string
	for _, b := range bagian {
		if b = strings.TrimSpace(b); b != "" {
			isi = append(isi, b)
		}
	}
	if len(isi) == 0 {
		return "-"
	}
	return strings.Join(isi, sep)
}

func namaBerkas(s entities.Surat) string {
	nomor := strings.NewReplacer("/", "-", " ", "_").Replace(s.Nomor)
	return templates[s.Jenis].Berkas + "_" + nomor + ".pdf"
}

// renderPDF membuat PDF surat dari template jenisnya. Surat wajib sudah memuat
// Pendidik dan Sekolah.Sudin.
func renderPDF(s entities.Surat) ([]byte, error) {
	t := templates[s.Jenis]
	var sudin *entities.Sudin
	unitKerja := "-"
	if s.Sekolah != nil {
		sudin = s.Sekolah.Sudin
		unitKerja = s.Sekolah.Nama
	}
	p := s.Pendidik
	ttd := penandaTangan(sudin)
	isi := strings.NewReplacer(
		"{penanda_tangan}", ttd,
		"{nama}", p.Nama,
		"{unit_kerja}", unitKerja,
		"{perihal}", s.Perihal,
	)

	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetTitle(t.Label+" "+s.Nomor, true)
	pdf.SetMargins(25, 20, 25)
	pdf.SetAutoPageBreak(true, 20)
	pdf.AddPage()
	tr := pdf.UnicodeTranslatorFromDescriptor("") // UTF-8 -> cp1252 untuk font inti
	lebar, _ := pdf.GetPageSize()
	lebar -= 50

	// Kop surat (S3)
	pdf.SetFont("Helvetica", "B", 12)
	pdf.CellFormat(lebar, 6, "PEMERINTAH PROVINSI DKI JAKARTA", "", 1, "C", false, 0, "")
	pdf.CellFormat(lebar, 6, "DINAS PENDIDIKAN", "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "B", 11)
	pdf.MultiCell(lebar, 6, tr(strings.ToUpper(kop(sudin))), "", "C", false)
	y := pdf.GetY() + 2
	pdf.SetLineWidth(0.6)
	pdf.Line(25, y, 25+lebar, y)
	pdf.SetY(y + 8)

	pdf.SetFont("Helvetica", "BU", 12)
	pdf.CellFormat(lebar, 6, tr(t.Judul), "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "", 11)
	pdf.CellFormat(lebar, 6, tr("Nomor: "+s.Nomor), "", 1, "C", false, 0, "")
	pdf.Ln(4)
	pdf.MultiCell(lebar, 6, tr("Perihal: "+s.Perihal), "", "L", false)
	pdf.Ln(2)
	pdf.MultiCell(lebar, 6, tr(isi.Replace(t.Pembuka)), "", "J", false)
	pdf.Ln(2)

	baris := [][2]string{
		{"Nama", p.Nama},
		{"NIP", formatNIP(p.NIP)},
		{"Pangkat/golongan", gabung(" · ", p.Pangkat, p.Golongan)},
		{"Jabatan", gabung(" · ", p.Jabatan, p.JabatanDapodik)},
		{"Unit kerja", unitKerja},
	}
	for _, b := range baris {
		pdf.SetX(35)
		pdf.CellFormat(40, 6, tr(b[0]), "", 0, "L", false, 0, "")
		pdf.CellFormat(4, 6, ":", "", 0, "L", false, 0, "")
		pdf.MultiCell(lebar-54, 6, tr(b[1]), "", "L", false)
	}
	pdf.Ln(3)
	if k := strings.TrimSpace(s.Keterangan); k != "" {
		pdf.MultiCell(lebar, 6, tr(k), "", "J", false)
		pdf.Ln(2)
	}
	pdf.MultiCell(lebar, 6, tr(isi.Replace(t.Penutup)), "", "J", false)

	// Blok tanda tangan basah, kanan bawah
	pdf.Ln(10)
	kolom := 25 + lebar/2
	pdf.SetX(kolom)
	pdf.MultiCell(lebar/2, 6, tr("Jakarta, "+tanggalIndonesia(s.TanggalSurat)), "", "L", false)
	pdf.SetX(kolom)
	pdf.MultiCell(lebar/2, 6, tr(ttd+","), "", "L", false)
	pdf.Ln(22)
	pdf.SetX(kolom)
	pdf.MultiCell(lebar/2, 6, tr("(nama & NIP pejabat penanda tangan)"), "", "L", false)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
