package service

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"github.com/go-pdf/fpdf"
)

// LaporanPDF (A6 "Unduh laporan Anjab (PDF)") memuat angka yang sama dengan Monitoring:
// ringkasan periode, rekap per sekolah, dan pasangan pemerataan guru.
func (s *anjabService) LaporanPDF(ctx context.Context, actor entities.User, id string) ([]byte, string, error) {
	m, err := s.Monitoring(ctx, actor, id, "")
	if err != nil {
		return nil, "", err
	}
	pdf := fpdf.New("L", "mm", "A4", "")
	tr := pdf.UnicodeTranslatorFromDescriptor("")
	pdf.SetTitle("Laporan Analisis Jabatan "+m.TahunAjaran, true)
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 14)
	pdf.CellFormat(0, 8, tr("Laporan Analisis Jabatan "+m.TahunAjaran), "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(0, 6, tr(fmt.Sprintf("%d dari %d sekolah ditetapkan · kekurangan %d guru · kelebihan %d guru · dicetak %s",
		m.SekolahDitetapkan, m.TotalSekolah, m.KekuranganGuru, m.KelebihanGuru, time.Now().Format("02-01-2006 15:04"))), "", 1, "L", false, 0, "")

	tabel := func(judul string, head []string, w []float64, baris [][]string) {
		pdf.Ln(4)
		pdf.SetFont("Helvetica", "B", 11)
		pdf.CellFormat(0, 7, tr(judul), "", 1, "L", false, 0, "")
		kepala := func() {
			pdf.SetFont("Helvetica", "B", 9)
			for i, h := range head {
				pdf.CellFormat(w[i], 7, tr(h), "1", 0, "L", false, 0, "")
			}
			pdf.Ln(-1)
			pdf.SetFont("Helvetica", "", 9)
		}
		kepala()
		_, tinggi := pdf.GetPageSize()
		for _, b := range baris {
			if pdf.GetY() > tinggi-20 {
				pdf.AddPage()
				kepala()
			}
			for i, c := range b {
				// Potong teks yang melebihi lebar kolom supaya tidak menimpa kolom sebelah.
				for pdf.GetStringWidth(tr(c)) > w[i]-2 && len(c) > 1 {
					c = c[:len(c)-1]
				}
				pdf.CellFormat(w[i], 6, tr(c), "1", 0, "L", false, 0, "")
			}
			pdf.Ln(-1)
		}
		if len(baris) == 0 {
			pdf.CellFormat(0, 6, tr("Belum ada data."), "", 1, "L", false, 0, "")
		}
	}

	sekolah := make([][]string, 0, len(m.Sekolah))
	for _, r := range m.Sekolah {
		sekolah = append(sekolah, []string{r.Sekolah.NPSN, r.Sekolah.Nama, r.Sekolah.Jenjang, r.Sekolah.Kecamatan, r.Status, fmt.Sprint(r.Kurang), fmt.Sprint(r.Lebih)})
	}
	tabel("Rekap per sekolah (versi kondisi riil)",
		[]string{"NPSN", "Sekolah", "Jenjang", "Kecamatan", "Status", "Kurang", "Lebih"},
		[]float64{24, 95, 18, 45, 50, 20, 20}, sekolah)

	saran := make([][]string, 0, len(m.Rekomendasi))
	for _, r := range m.Rekomendasi {
		jarak := "Beda kecamatan"
		if r.KecamatanSama {
			jarak = "Kecamatan sama"
		}
		saran = append(saran, []string{r.Mapel,
			fmt.Sprintf("%s (lebih %d, %s)", r.Dari.Nama, r.Dari.Jumlah, r.Dari.Kecamatan),
			fmt.Sprintf("%s (kurang %d, %s)", r.Ke.Nama, r.Ke.Jumlah, r.Ke.Kecamatan),
			fmt.Sprint(r.Jumlah), jarak})
	}
	tabel("Rekomendasi pemerataan guru",
		[]string{"Mapel", "Guru lebih di", "Kurang di", "Jumlah", "Jarak"},
		[]float64{45, 95, 95, 17, 20}, saran)

	pdf.Ln(3)
	pdf.SetFont("Helvetica", "I", 8)
	pdf.MultiCell(0, 5, tr("Pasangan adalah saran berdasarkan mapel dan kecamatan. Mutasi tetap diproses di luar sistem."), "", "L", false)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), "laporan_anjab_" + strings.ReplaceAll(m.TahunAjaran, "/", "-") + ".pdf", nil
}
