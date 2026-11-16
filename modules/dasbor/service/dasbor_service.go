package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"sort"
	"time"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"github.com/Caknoooo/go-gin-clean-starter/modules/dasbor/repository"
	"github.com/Caknoooo/go-gin-clean-starter/modules/dasbor/spek"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/constants"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/helpers"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/utils"
	"github.com/go-pdf/fpdf"
	"github.com/google/uuid"
)

const batasPerModul = 500 // tugas per modul yang dibaca; cukup untuk antrean satu Sudin

type TugasItem struct {
	Modul      string    `json:"modul"`
	ModulLabel string    `json:"modul_label"`
	ID         uuid.UUID `json:"id"`
	Nomor      string    `json:"nomor"`
	Pegawai    *string   `json:"pegawai"`
	Sekolah    string    `json:"sekolah"`
	Status     string    `json:"status"`
	Tindakan   string    `json:"tindakan"`
	Kategori   string    `json:"kategori"`
	Tenggat    string    `json:"tenggat"`
	SisaHari   int       `json:"sisa_hari"` // negatif = terlewat
	Tautan     string    `json:"tautan"`
}

type TugasResponse struct {
	Items       []TugasItem    `json:"items"`
	Total       int            `json:"total"`
	PerKategori map[string]int `json:"per_kategori"`
}

type PengajuanItem struct {
	Modul      string    `json:"modul"`
	ModulLabel string    `json:"modul_label"`
	ID         uuid.UUID `json:"id"`
	Nomor      string    `json:"nomor"`
	Pegawai    *string   `json:"pegawai"`
	Sekolah    string    `json:"sekolah"`
	Status     string    `json:"status"`
	Kelompok   string    `json:"kelompok"`
	Diajukan   time.Time `json:"diajukan"`
	Diperbarui time.Time `json:"diperbarui"`
	Tautan     string    `json:"tautan"`
}

type ModulStat struct {
	Kode      string           `json:"kode"`
	Label     string           `json:"label"`
	Total     int64            `json:"total"`
	Selesai   int64            `json:"selesai"`
	PerStatus map[string]int64 `json:"per_status"`
	RataHari  *float64         `json:"rata_hari"`
}

type RingkasanResponse struct {
	Bulan string      `json:"bulan"`
	Modul []ModulStat `json:"modul"`
	// Semua: Cuti, Pensiun, KP, Dapodik, Disiplin (B2); Anjab dan Kinerja dihitung terpisah.
	Semua ModulStat `json:"semua"`
	Tren  []struct {
		Bulan  string `json:"bulan"`
		Jumlah int64  `json:"jumlah"`
	} `json:"tren"`
	Anjab *AnjabStat `json:"anjab"`
}

type AnjabStat struct {
	TahunAjaran  string           `json:"tahun_ajaran"`
	TanggalTutup string           `json:"tanggal_tutup"`
	Total        int64            `json:"total"`
	Ditetapkan   int64            `json:"ditetapkan"`
	PerStatus    map[string]int64 `json:"per_status"`
}

type DasborService interface {
	Tugas(ctx context.Context, actor entities.User, kategori, modul string, offset, limit int) (TugasResponse, error)
	Pengajuan(ctx context.Context, actor entities.User, modul []string, kelompok, search string, offset, limit int) ([]PengajuanItem, int64, error)
	PengajuanCSV(ctx context.Context, actor entities.User, modul []string, kelompok, search string) ([]byte, error)
	KirimPengingatTenggat(ctx context.Context) (int, error)
	JalankanPengingat(ctx context.Context)
	Ringkasan(ctx context.Context, actor entities.User, bulan string) (RingkasanResponse, error)
	RekapPDF(ctx context.Context, actor entities.User, bulan string) ([]byte, error)
}

type dasborService struct {
	repo repository.DasborRepository
	now  func() time.Time
}

func NewDasborService(repo repository.DasborRepository) DasborService {
	return &dasborService{repo: repo, now: time.Now}
}

// peranTugas: admin melihat tugas staf Sudin.
func peranTugas(actor entities.User) string {
	if actor.Role == constants.ENUM_ROLE_ADMIN {
		return constants.ENUM_ROLE_STAF_SUDIN
	}
	return actor.Role
}

func pegawai(actor entities.User, m spek.Modul, nama *string) *string {
	if m.SembunyikanUntukPimpinan && actor.Role == constants.ENUM_ROLE_PIMPINAN {
		return nil
	}
	return nama
}

func (s *dasborService) hariIni() time.Time {
	n := s.now()
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, n.Location())
}

func (s *dasborService) Tugas(ctx context.Context, actor entities.User, kategori, modul string, offset, limit int) (TugasResponse, error) {
	peran := peranTugas(actor)
	hariIni := s.hariIni()
	var items []TugasItem
	tambah := func(kode, label string, b repository.Baris, t spek.Tugas, tenggat time.Time, nama *string) {
		tg := time.Date(tenggat.Year(), tenggat.Month(), tenggat.Day(), 0, 0, 0, 0, hariIni.Location())
		items = append(items, TugasItem{
			Modul: kode, ModulLabel: label, ID: b.ID, Nomor: b.Nomor, Pegawai: nama, Sekolah: b.SekolahNama,
			Status: b.Status, Tindakan: t.Tindakan, Kategori: t.Kategori, Tenggat: tg.Format("2006-01-02"),
			SisaHari: int(tg.Sub(hariIni).Hours() / 24), Tautan: "/" + kode + "/" + b.ID.String(),
		})
	}

	for _, m := range spek.DaftarModul {
		if modul != "" && modul != m.Kode {
			continue
		}
		byStatus := map[string]spek.Tugas{}
		var status []string
		for _, t := range m.Tugas {
			if t.Peran == peran {
				byStatus[t.Status] = t
				status = append(status, t.Status)
			}
		}
		if len(status) == 0 {
			continue
		}
		rows, _, err := s.repo.Baris(ctx, actor, m, repository.Saring{StatusIn: status, Limit: batasPerModul})
		if err != nil {
			return TugasResponse{}, err
		}
		for _, b := range rows {
			t := byStatus[b.Status]
			tenggat := helpers.TambahHariKerja(b.UpdatedAt, t.HariKerja)
			if m.PunyaTenggat && b.Tenggat != nil {
				tenggat = *b.Tenggat
			}
			tambah(m.Kode, m.Label, b, t, tenggat, pegawai(actor, m, b.PendidikNama))
		}
	}

	// Anjab: staf meninjau kiriman sekolah; operator memperbarui data selama periode terbuka.
	if modul == "" || modul == "anjab" {
		var t spek.Tugas
		var status []string
		terbuka := false
		switch peran {
		case constants.ENUM_ROLE_STAF_SUDIN:
			t, status = spek.Tugas{Tindakan: "Tinjau hasil Anjab sekolah", Kategori: spek.KategoriPerluTindakan, HariKerja: 5}, []string{"dikirim"}
		case constants.ENUM_ROLE_OPERATOR_SEKOLAH:
			t, status, terbuka = spek.Tugas{Tindakan: "Perbarui data Analisis Jabatan sekolah", Kategori: spek.KategoriPerluTindakan}, []string{"draf", "dikembalikan"}, true
		}
		if len(status) > 0 {
			rows, err := s.repo.AnjabTugas(ctx, actor, status, terbuka)
			if err != nil {
				return TugasResponse{}, err
			}
			for _, b := range rows {
				tenggat := helpers.TambahHariKerja(b.UpdatedAt, t.HariKerja)
				if terbuka && b.Tenggat != nil {
					tenggat = *b.Tenggat
				}
				tambah("anjab", "Analisis Jabatan", b, t, tenggat, nil)
			}
		}
	}

	sort.SliceStable(items, func(i, j int) bool { return items[i].Tenggat < items[j].Tenggat })
	per := map[string]int{spek.KategoriPerluTindakan: 0, spek.KategoriMenungguFisik: 0, spek.KategoriMenungguTTD: 0}
	var hasil []TugasItem
	for _, it := range items {
		per[it.Kategori]++
		if kategori == "" || kategori == it.Kategori {
			hasil = append(hasil, it)
		}
	}
	total := len(hasil)
	if offset > len(hasil) {
		offset = len(hasil)
	}
	end := min(offset+limit, len(hasil))
	return TugasResponse{Items: append([]TugasItem{}, hasil[offset:end]...), Total: total, PerKategori: per}, nil
}

// maksEkspor membatasi baris rekap CSV.
const maksEkspor = 5000

// PengajuanCSV adalah "Unduh rekap" di antrean tiap modul: isi Daftar Pengajuan dengan saringan yang sama.
func (s *dasborService) PengajuanCSV(ctx context.Context, actor entities.User, modul []string, kelompok, search string) ([]byte, error) {
	items, _, err := s.Pengajuan(ctx, actor, modul, kelompok, search, 0, maksEkspor)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"modul", "nomor", "pegawai", "sekolah", "status", "kelompok", "diajukan", "diperbarui"})
	for _, it := range items {
		pegawai := ""
		if it.Pegawai != nil {
			pegawai = *it.Pegawai
		}
		_ = w.Write([]string{it.ModulLabel, helpers.AmanCSV(it.Nomor), helpers.AmanCSV(pegawai), helpers.AmanCSV(it.Sekolah),
			it.Status, it.Kelompok, it.Diajukan.Format(time.RFC3339), it.Diperbarui.Format(time.RFC3339)})
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}

func (s *dasborService) Pengajuan(ctx context.Context, actor entities.User, modul []string, kelompok, search string, offset, limit int) ([]PengajuanItem, int64, error) {
	pilih := map[string]bool{}
	for _, m := range modul {
		if _, ok := spek.ByKode(m); !ok {
			return nil, 0, utils.ErrBadRequest("modul tidak dikenal: " + m)
		}
		pilih[m] = true
	}
	all := []PengajuanItem{}
	var total int64
	for _, m := range spek.DaftarModul {
		if !m.Pengajuan || (len(pilih) > 0 && !pilih[m.Kode]) {
			continue
		}
		sar := repository.Saring{Search: search, Limit: offset + limit}
		switch kelompok {
		case "":
		case spek.KelompokSelesai:
			sar.StatusIn = m.Final
		case spek.KelompokPerluPerbaikan:
			sar.StatusIn = m.Perbaikan
		case spek.KelompokMenungguFisik:
			sar.StatusIn = m.Fisik
		case spek.KelompokDiproses:
			sar.StatusNotIn = append(append(append([]string{}, m.Final...), m.Perbaikan...), m.Fisik...)
		default:
			return nil, 0, utils.ErrBadRequest("kelompok tidak dikenal")
		}
		if kelompok != "" && kelompok != spek.KelompokDiproses && len(sar.StatusIn) == 0 {
			continue // modul ini tidak punya status di kelompok tsb.
		}
		rows, n, err := s.repo.Baris(ctx, actor, m, sar)
		if err != nil {
			return nil, 0, err
		}
		total += n
		for _, b := range rows {
			all = append(all, PengajuanItem{
				Modul: m.Kode, ModulLabel: m.Label, ID: b.ID, Nomor: b.Nomor, Pegawai: pegawai(actor, m, b.PendidikNama),
				Sekolah: b.SekolahNama, Status: b.Status, Kelompok: m.Kelompok(b.Status), Diajukan: b.CreatedAt,
				Diperbarui: b.UpdatedAt, Tautan: "/" + m.Kode + "/" + b.ID.String(),
			})
		}
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].Diperbarui.After(all[j].Diperbarui) })
	if offset > len(all) {
		offset = len(all)
	}
	return all[offset:min(offset+limit, len(all))], total, nil
}

func parseBulan(bulan string, now time.Time) (time.Time, error) {
	if bulan == "" {
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()), nil
	}
	t, err := time.ParseInLocation("2006-01", bulan, now.Location())
	if err != nil {
		return t, utils.ErrBadRequest("bulan harus berformat YYYY-MM")
	}
	return t, nil
}

func (s *dasborService) Ringkasan(ctx context.Context, actor entities.User, bulan string) (RingkasanResponse, error) {
	awal, err := parseBulan(bulan, s.now())
	if err != nil {
		return RingkasanResponse{}, err
	}
	akhir := awal.AddDate(0, 1, 0)
	res := RingkasanResponse{Bulan: awal.Format("2006-01"), Semua: ModulStat{Kode: "semua", Label: "Semua pengajuan", PerStatus: map[string]int64{}}}
	var semuaDetik float64
	var semuaSelesai int64
	tren := map[string]int64{}
	trenDari := awal.AddDate(0, -5, 0)

	for _, m := range spek.DaftarModul {
		if m.Kode == "surat" && actor.Role == constants.ENUM_ROLE_OPERATOR_SEKOLAH {
			continue // modul Surat tertutup untuk operator sekolah
		}
		rows, err := s.repo.Statistik(ctx, actor, m, awal, akhir)
		if err != nil {
			return RingkasanResponse{}, err
		}
		st := ModulStat{Kode: m.Kode, Label: m.Label, PerStatus: map[string]int64{}}
		var detik float64
		for _, r := range rows {
			st.PerStatus[r.Status] = r.Jumlah
			st.Total += r.Jumlah
			if m.Kelompok(r.Status) == spek.KelompokSelesai {
				st.Selesai += r.Jumlah
				if r.RataDetik != nil {
					detik += *r.RataDetik * float64(r.Jumlah)
				}
			}
		}
		if st.Selesai > 0 {
			h := round1(detik / float64(st.Selesai) / 86400)
			st.RataHari = &h
		}
		res.Modul = append(res.Modul, st)
		if m.Pengajuan {
			res.Semua.Total += st.Total
			res.Semua.Selesai += st.Selesai
			semuaDetik += detik
			semuaSelesai += st.Selesai
			t, err := s.repo.TrenBulanan(ctx, actor, m, trenDari)
			if err != nil {
				return RingkasanResponse{}, err
			}
			for k, v := range t {
				tren[k] += v
			}
		}
	}
	if semuaSelesai > 0 {
		h := round1(semuaDetik / float64(semuaSelesai) / 86400)
		res.Semua.RataHari = &h
	}
	for i := 0; i < 6; i++ {
		k := trenDari.AddDate(0, i, 0).Format("2006-01")
		res.Tren = append(res.Tren, struct {
			Bulan  string `json:"bulan"`
			Jumlah int64  `json:"jumlah"`
		}{k, tren[k]})
	}

	a, err := s.repo.AnjabTerkini(ctx, actor)
	if err != nil {
		return RingkasanResponse{}, err
	}
	if a != nil {
		as := &AnjabStat{TahunAjaran: a.TahunAjaran, TanggalTutup: a.TanggalTutup.Format("2006-01-02"), PerStatus: a.PerStatus}
		for st, n := range a.PerStatus {
			as.Total += n
			if st == "ditetapkan" {
				as.Ditetapkan = n
			}
		}
		res.Anjab = as
	}
	return res, nil
}

func round1(f float64) float64 { return float64(int(f*10+0.5)) / 10 }

// RekapPDF (B2 "Unduh rekap"): ringkasan bulan per modul dalam satu halaman.
func (s *dasborService) RekapPDF(ctx context.Context, actor entities.User, bulan string) ([]byte, error) {
	r, err := s.Ringkasan(ctx, actor, bulan)
	if err != nil {
		return nil, err
	}
	pdf := fpdf.New("P", "mm", "A4", "")
	tr := pdf.UnicodeTranslatorFromDescriptor("")
	pdf.SetTitle("Rekap kepegawaian "+r.Bulan, true)
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 14)
	pdf.CellFormat(0, 8, tr("Rekap Kepegawaian"), "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	wilayah := "Semua wilayah"
	if actor.SudinID != nil {
		wilayah = "Sudin Pendidikan " + s.repo.SudinNama(ctx, *actor.SudinID)
	}
	pdf.CellFormat(0, 6, tr(fmt.Sprintf("%s · bulan %s · dicetak %s", wilayah, r.Bulan, s.now().Format("02-01-2006 15:04"))), "", 1, "L", false, 0, "")
	pdf.Ln(4)

	head := []string{"Modul", "Masuk", "Selesai", "Rata-rata (hari)", "Rincian status"}
	w := []float64{40, 18, 18, 30, 84}
	pdf.SetFont("Helvetica", "B", 9)
	for i, h := range head {
		pdf.CellFormat(w[i], 7, tr(h), "1", 0, "L", false, 0, "")
	}
	pdf.Ln(-1)
	pdf.SetFont("Helvetica", "", 9)
	baris := append(append([]ModulStat{}, r.Modul...), r.Semua)
	for _, m := range baris {
		rata := "-"
		if m.RataHari != nil {
			rata = fmt.Sprintf("%.1f", *m.RataHari)
		}
		keys := make([]string, 0, len(m.PerStatus))
		for k := range m.PerStatus {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		rinci := ""
		for _, k := range keys {
			rinci += fmt.Sprintf("%s %d; ", k, m.PerStatus[k])
		}
		cells := []string{m.Label, fmt.Sprint(m.Total), fmt.Sprint(m.Selesai), rata, rinci}
		for i, c := range cells {
			pdf.CellFormat(w[i], 7, tr(c), "1", 0, "L", false, 0, "")
		}
		pdf.Ln(-1)
	}
	if r.Anjab != nil {
		pdf.Ln(4)
		pdf.CellFormat(0, 6, tr(fmt.Sprintf("Analisis Jabatan %s: %d dari %d sekolah ditetapkan.", r.Anjab.TahunAjaran, r.Anjab.Ditetapkan, r.Anjab.Total)), "", 1, "L", false, 0, "")
	}
	pdf.Ln(2)
	pdf.SetFont("Helvetica", "I", 8)
	pdf.MultiCell(0, 5, tr("Angka dihitung dari pengajuan yang masuk pada bulan tersebut. Semua pengajuan = Cuti, Pensiun, Kenaikan Pangkat, Layanan Dapodik, dan Sikap Disiplin; Anjab dan Kinerja P3K PW dihitung terpisah."), "", "L", false)
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
