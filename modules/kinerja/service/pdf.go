package service

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"github.com/Caknoooo/go-gin-clean-starter/modules/kinerja/dto"
	"github.com/go-pdf/fpdf"
)

// wib dipakai untuk tanggal cetak; zona tetap supaya tidak bergantung tzdata di container.
var wib = time.FixedZone("WIB", 7*60*60)

// pejabatPenilai:Kepala Sudin tempat sekolah pegawai (Figma KP2 "Pejabat Penilai").
func pejabatPenilai(k *entities.KinerjaPenilaian) string {
	if k.Sekolah != nil && k.Sekolah.Sudin != nil {
		return "Kepala Sudin Pendidikan " + k.Sekolah.Sudin.Nama
	}
	return "Kepala Sudin Pendidikan"
}

func namaBerkas(k *entities.KinerjaPenilaian) string {
	nama := strings.NewReplacer("/", "-", "\\", "-").Replace(k.Pendidik.Nama)
	return fmt.Sprintf("Penilaian Kinerja PPPK PW · %s · Smt %d %d.pdf", nama, k.Semester, k.Tahun)
}

// buatPDF membuat dokumen penilaian untuk dicetak dan ditandatangani basah.
// k harus sudah memuat Pendidik dan Sekolah (beserta Sudin).
func buatPDF(k *entities.KinerjaPenilaian, dibuat time.Time) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.SetAutoPageBreak(true, 15)
	pdf.SetCreationDate(dibuat)
	tr := pdf.UnicodeTranslatorFromDescriptor("") // cp1252: cukup untuk "–" dan "·"
	pdf.SetTitle(tr("Penilaian Kinerja PPPK PW "+k.Nomor), false)
	pdf.AliasNbPages("")
	pdf.SetFooterFunc(func() {
		pdf.SetY(-12)
		pdf.SetFont("Helvetica", "I", 7)
		pdf.CellFormat(0, 4, tr(fmt.Sprintf("%s · dibuat Teralur %s WIB · halaman %d dari {nb}",
			k.Nomor, dibuat.In(wib).Format("02-01-2006 15:04"), pdf.PageNo())), "", 0, "C", false, 0, "")
	})
	pdf.AddPage()

	p := k.Pendidik
	periode := dto.Periode(k.Tahun, k.Semester)

	pdf.SetFont("Helvetica", "B", 13)
	pdf.CellFormat(0, 7, tr("PENILAIAN KINERJA PPPK PARUH WAKTU"), "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(0, 5, tr("Nomor "+k.Nomor+" · "+periode), "", 1, "C", false, 0, "")
	pdf.Ln(5)

	judul := func(s string) {
		pdf.SetFont("Helvetica", "B", 10)
		pdf.CellFormat(0, 6, tr(s), "", 1, "L", false, 0, "")
	}

	judul("A. Identitas pegawai")
	nip := "Tanpa NIP"
	if p.NIP != nil && *p.NIP != "" {
		nip = *p.NIP
	}
	unit := ""
	if k.Sekolah != nil {
		unit = k.Sekolah.Nama
	}
	pdf.SetFont("Helvetica", "", 9)
	for _, row := range [][2]string{
		{"Nama", p.Nama},
		{"NIP", nip},
		{"Status kepegawaian", "PPPK Paruh Waktu"},
		{"Jabatan", p.Jabatan},
		{"Unit kerja", unit},
		{"Periode penilaian", periode},
		{"Pejabat penilai", pejabatPenilai(k)},
	} {
		pdf.CellFormat(45, 5.5, tr(row[0]), "", 0, "L", false, 0, "")
		pdf.CellFormat(0, 5.5, tr(": "+row[1]), "", 1, "L", false, 0, "")
	}
	pdf.Ln(4)

	judul("B. Nilai BerAKHLAK (skala 1–5)")
	lebar := []float64{9, 38, 95, 12, 26} // total 180 mm = lebar A4 dikurangi margin
	pdf.SetFont("Helvetica", "B", 8.5)
	pdf.SetFillColor(230, 230, 230)
	for i, h := range []string{"No", "Aspek", "Uraian perilaku", "Nilai", "Keterangan"} {
		pdf.CellFormat(lebar[i], 7, tr(h), "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)
	pdf.SetFont("Helvetica", "", 8.5)
	for i, a := range daftarAspek(k) {
		nilai, ket := "-", ""
		if a.Nilai != nil {
			nilai, ket = fmt.Sprint(*a.Nilai), labelSkor[*a.Nilai]
		}
		pdf.CellFormat(lebar[0], 7, fmt.Sprint(i+1), "1", 0, "C", false, 0, "")
		pdf.CellFormat(lebar[1], 7, tr(a.Nama), "1", 0, "L", false, 0, "")
		pdf.CellFormat(lebar[2], 7, tr(a.Uraian), "1", 0, "L", false, 0, "")
		pdf.CellFormat(lebar[3], 7, nilai, "1", 0, "C", false, 0, "")
		pdf.CellFormat(lebar[4], 7, tr(ket), "1", 1, "L", false, 0, "")
	}
	akhir, pred := "-", ""
	if k.NilaiAkhir != nil {
		akhir, pred = koma(*k.NilaiAkhir), k.Predikat
	}
	pdf.SetFont("Helvetica", "B", 9)
	pdf.CellFormat(lebar[0]+lebar[1]+lebar[2], 8, tr("Nilai akhir (rata-rata tujuh aspek)"), "1", 0, "R", true, 0, "")
	pdf.CellFormat(lebar[3], 8, akhir, "1", 0, "C", true, 0, "")
	pdf.CellFormat(lebar[4], 8, tr(pred), "1", 1, "L", true, 0, "")
	pdf.SetFont("Helvetica", "", 7.5)
	pdf.Ln(1)
	pdf.MultiCell(0, 4, tr("Predikat: Sangat kurang (1,0–1,4) · Kurang (1,5–2,4) · Cukup (2,5–3,4) · Baik (3,5–4,4) · Sangat baik (4,5–5,0)."), "", "L", false)
	pdf.Ln(10)

	judul("C. Pengesahan")
	pdf.SetFont("Helvetica", "", 9)
	kiri, kanan := 15.0, 115.0
	y := pdf.GetY()
	pdf.SetXY(kiri, y)
	pdf.MultiCell(80, 5, tr("Pegawai yang dinilai,"), "", "L", false)
	pdf.SetXY(kanan, y)
	pdf.MultiCell(80, 5, tr("Jakarta, ...............................\nPejabat Penilai,\n"+pejabatPenilai(k)), "", "L", false)
	y = pdf.GetY() + 22
	pdf.SetXY(kiri, y)
	pdf.MultiCell(80, 5, tr(p.Nama), "", "L", false)
	pdf.SetXY(kanan, y)
	pdf.MultiCell(80, 5, tr("(.............................................)\nNIP"), "", "L", false)
	pdf.Ln(8)
	pdf.SetFont("Helvetica", "I", 7.5)
	pdf.MultiCell(0, 4, tr("Dokumen ini sah setelah ditandatangani basah dan distempel Pejabat Penilai di Sudin, lalu hasil pindainya diunggah ke Teralur."), "", "L", false)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
