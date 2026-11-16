package service

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"github.com/Caknoooo/go-gin-clean-starter/modules/anjab/dto"
	"github.com/Caknoooo/go-gin-clean-starter/modules/anjab/hitung"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/aktivitas"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/utils"
	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

const (
	lembarSekolah = "Sekolah"
	lembarRombel  = "Rombel"
	lembarGuru    = "Guru"

	maksUkuranImpor = 10 << 20
	maksBarisImpor  = 20000
)

var (
	kolomSekolah = []string{"NPSN", "Nama Sekolah", "Jenjang", "Kecamatan"}
	kolomRombel  = []string{"NPSN", "Tingkat", "Jumlah Rombel", "Jumlah Siswa"}
	kolomGuru    = []string{"NPSN", "NIP", "NIK", "Nama", "Status Kepegawaian", "Golongan", "Tanggal Lahir", "TMT CPNS", "Mapel SK", "Mapel Riil", "JP Riil"}
	// Kolom identitas diformat teks di templat supaya nol di depan dan 18 digit NIP tidak rusak.
	kolomTeks = map[string]bool{"NPSN": true, "NIP": true, "NIK": true}

	polaAngka = regexp.MustCompile(`^\d+$`)
	polaNPSN  = regexp.MustCompile(`^\d{8}$`)
	romawi    = map[string]int{"I": 1, "II": 2, "III": 3, "IV": 4, "V": 5, "VI": 6, "VII": 7, "VIII": 8, "IX": 9, "X": 10, "XI": 11, "XII": 12}
	statusPeg = map[string]string{
		"PNS": entities.StatusKepegawaianPNS, "PPPK": entities.StatusKepegawaianPPPK,
		"PPPK_PW": entities.StatusKepegawaianPPPKPW, "PPPK PW": entities.StatusKepegawaianPPPKPW,
		"PPPK PARUH WAKTU": entities.StatusKepegawaianPPPKPW, "KKI": entities.StatusKepegawaianKKI,
	}
)

func (s *anjabService) Templat(ctx context.Context, actor entities.User, id string) ([]byte, string, error) {
	p, err := s.repo.GetPeriode(ctx, nil, actor, id, false)
	if err != nil {
		return nil, "", err
	}
	f := excelize.NewFile()
	defer f.Close()
	teks, err := f.NewStyle(&excelize.Style{NumFmt: 49})
	if err != nil {
		return nil, "", err
	}
	tebal, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err != nil {
		return nil, "", err
	}
	lembar := []struct {
		nama  string
		kolom []string
	}{{lembarSekolah, kolomSekolah}, {lembarRombel, kolomRombel}, {lembarGuru, kolomGuru}}
	for i, l := range lembar {
		if i == 0 {
			if err := f.SetSheetName("Sheet1", l.nama); err != nil {
				return nil, "", err
			}
		} else if _, err := f.NewSheet(l.nama); err != nil {
			return nil, "", err
		}
		if err := f.SetSheetRow(l.nama, "A1", &l.kolom); err != nil {
			return nil, "", err
		}
		for j, k := range l.kolom {
			col, _ := excelize.ColumnNumberToName(j + 1)
			_ = f.SetColWidth(l.nama, col, col, 20)
			if kolomTeks[k] {
				if err := f.SetColStyle(l.nama, col, teks); err != nil {
					return nil, "", err
				}
			}
		}
		_ = f.SetCellStyle(l.nama, "A1", fmt.Sprintf("%s1", mustCol(len(l.kolom))), tebal)
	}

	petunjuk := [][]any{
		{"Petunjuk impor data master Anjab " + p.TahunAjaran},
		{"Jenjang periode: " + strings.ReplaceAll(p.Jenjang, ",", ", ")},
		{"Sekolah: NPSN 8 digit yang terdaftar di Sudin; Jenjang harus sama dengan data sekolah."},
		{"Rombel: Tingkat 1–6 (SD), 7–9 (SMP), 10–12 (SMA/SMK), boleh angka romawi. Satu baris per tingkat."},
		{"Guru: NIP 18 digit (wajib untuk PNS/PPPK), NIK 16 digit, Status PNS / PPPK / PPPK_PW / KKI."},
		{"Tanggal ditulis YYYY-MM-DD atau sel bertipe tanggal. Guru kelas SD: isi Mapel SK \"" + hitung.MapelGuruKelas + "\"."},
		{"Data rombel dan guru sekolah yang ada di berkas menggantikan data sebelumnya untuk periode ini."},
	}
	if _, err := f.NewSheet("Petunjuk"); err != nil {
		return nil, "", err
	}
	for i, row := range petunjuk {
		if err := f.SetSheetRow("Petunjuk", fmt.Sprintf("A%d", i+1), &row); err != nil {
			return nil, "", err
		}
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, "", err
	}
	return buf.Bytes(), "templat_anjab_" + strings.ReplaceAll(p.TahunAjaran, "/", "-") + ".xlsx", nil
}

func mustCol(n int) string {
	c, _ := excelize.ColumnNumberToName(n)
	return c
}

// lembarData adalah isi satu lembar: baris data (tanpa header) beserta nomor baris Excel-nya.
type lembarData struct {
	nama  string
	idx   map[string]int
	baris [][]string
	nomor []int
}

func (l lembarData) sel(row []string, kolom string) string {
	i := l.idx[kolom]
	if i >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[i])
}

// bacaLembar membaca lembar baris demi baris dan hanya menyimpan kolom yang dipakai,
// supaya berkas yang direkayasa (sel di kolom jauh, jutaan baris) tidak menghabiskan memori.
func bacaLembar(f *excelize.File, nama string, kolom []string) (lembarData, error) {
	l := lembarData{nama: nama, idx: map[string]int{}}
	rows, err := f.Rows(nama)
	if err != nil {
		return l, utils.ErrBadRequest("lembar " + nama + " tidak ditemukan; pakai templat resmi")
	}
	defer rows.Close()
	lebar, n := 0, 0
	for rows.Next() {
		n++
		if n > maksBarisImpor+1 {
			return l, utils.ErrBadRequest(fmt.Sprintf("maksimal %d baris per impor", maksBarisImpor))
		}
		row, err := rows.Columns(excelize.Options{RawCellValue: true})
		if err != nil {
			return l, utils.ErrBadRequest("lembar " + nama + " tidak bisa dibaca; pakai templat resmi")
		}
		if n == 1 {
			header := map[string]int{}
			for i, h := range row {
				header[strings.ToLower(strings.TrimSpace(h))] = i
			}
			for _, k := range kolom {
				i, ok := header[strings.ToLower(k)]
				if !ok {
					return l, utils.ErrBadRequest(fmt.Sprintf("kolom %q tidak ada di lembar %s; pakai templat resmi", k, nama))
				}
				l.idx[k] = i
				lebar = max(lebar, i+1)
			}
			continue
		}
		if len(row) > lebar {
			row = row[:lebar]
		}
		kosong := true
		for _, v := range row {
			if strings.TrimSpace(v) != "" {
				kosong = false
				break
			}
		}
		if kosong {
			continue
		}
		l.baris = append(l.baris, append([]string(nil), row...))
		l.nomor = append(l.nomor, n)
	}
	if n == 0 {
		return l, utils.ErrBadRequest("lembar " + nama + " kosong; pakai templat resmi")
	}
	return l, nil
}

// pemeriksa mengumpulkan galat per baris.
type pemeriksa struct {
	galat []dto.GalatImpor
	salah map[string]bool // "lembar:baris"
}

func (p *pemeriksa) tambah(lembar string, baris int, kolom, nilai, masalah string) {
	p.galat = append(p.galat, dto.GalatImpor{Baris: baris, Lembar: lembar, Kolom: kolom, Nilai: nilai, Masalah: masalah})
	p.salah[fmt.Sprintf("%s:%d", lembar, baris)] = true
}

func (p *pemeriksa) valid(lembar string, baris int) bool {
	return !p.salah[fmt.Sprintf("%s:%d", lembar, baris)]
}

func angka(s string, minV, maxV int) (int, bool) {
	if s == "" {
		return 0, false
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || f != float64(int(f)) || int(f) < minV || int(f) > maxV {
		return 0, false
	}
	return int(f), true
}

// tanggalSel menerima YYYY-MM-DD, DD/MM/YYYY, DD-MM-YYYY, atau nomor seri tanggal Excel.
func tanggalSel(s string) (*time.Time, bool) {
	if s == "" {
		return nil, true
	}
	var t time.Time
	var err error
	if f, e := strconv.ParseFloat(s, 64); e == nil {
		t, err = excelize.ExcelDateToTime(f, false)
	} else {
		for _, layout := range []string{"2006-01-02", "02/01/2006", "02-01-2006"} {
			if t, err = time.Parse(layout, s); err == nil {
				break
			}
		}
	}
	if err != nil || t.Year() < 1940 || t.Year() > 2100 {
		return nil, false
	}
	t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	return &t, true
}

type guruImpor struct {
	sekolah   entities.Sekolah
	nip       string
	nik       string
	nama      string
	status    string
	golongan  string
	lahir     *time.Time
	tmtCPNS   *time.Time
	mapelSK   string
	mapelRiil string
	jpRiil    int
}

type rombelImpor struct {
	sekolahID uuid.UUID
	row       entities.AnjabRombel
}

func (s *anjabService) Impor(ctx context.Context, actor entities.User, id string, req dto.ImporRequest) (dto.ImporResponse, error) {
	res := dto.ImporResponse{NamaBerkas: filepath.Base(strings.ReplaceAll(req.Berkas.Filename, "\\", "/")), Galat: []dto.GalatImpor{}}
	if req.Berkas.Size > maksUkuranImpor {
		return res, utils.ErrBadRequest("ukuran berkas maksimal 10 MB")
	}
	if !strings.EqualFold(filepath.Ext(req.Berkas.Filename), ".xlsx") {
		return res, utils.ErrBadRequest("berkas harus spreadsheet .xlsx dari templat resmi")
	}

	p, err := s.repo.GetPeriode(ctx, nil, actor, id, false)
	if err != nil {
		return res, err
	}
	if p.DibukaAt != nil {
		return res, utils.ErrConflict("impor hanya bisa sebelum masa pembaruan dibuka")
	}

	src, err := req.Berkas.Open()
	if err != nil {
		return res, err
	}
	defer src.Close()
	f, err := excelize.OpenReader(src, excelize.Options{UnzipSizeLimit: 100 << 20, UnzipXMLSizeLimit: 50 << 20})
	if err != nil {
		return res, utils.ErrBadRequest("berkas bukan spreadsheet .xlsx yang valid")
	}
	defer f.Close()

	lSekolah, err := bacaLembar(f, lembarSekolah, kolomSekolah)
	if err != nil {
		return res, err
	}
	lRombel, err := bacaLembar(f, lembarRombel, kolomRombel)
	if err != nil {
		return res, err
	}
	lGuru, err := bacaLembar(f, lembarGuru, kolomGuru)
	if err != nil {
		return res, err
	}
	res.TotalBaris = len(lSekolah.baris) + len(lRombel.baris) + len(lGuru.baris)
	if res.TotalBaris > maksBarisImpor {
		return res, utils.ErrBadRequest(fmt.Sprintf("maksimal %d baris per impor", maksBarisImpor))
	}

	semua, err := s.repo.SekolahSudin(ctx, nil, p.SudinID, urutanJenjang)
	if err != nil {
		return res, err
	}
	sekolahNPSN := map[string]entities.Sekolah{}
	sekolahID := map[uuid.UUID]bool{}
	for _, sk := range semua {
		sekolahNPSN[sk.NPSN] = sk
		sekolahID[sk.ID] = true
	}

	pm := &pemeriksa{salah: map[string]bool{}}
	// cekNPSN memvalidasi kolom NPSN dan mengembalikan sekolahnya.
	cekNPSN := func(lembar string, baris int, npsn string) (entities.Sekolah, bool) {
		sk, ok := sekolahNPSN[npsn]
		switch {
		case !polaNPSN.MatchString(npsn):
			pm.tambah(lembar, baris, "NPSN", npsn, "NPSN harus 8 digit")
		case !ok:
			pm.tambah(lembar, baris, "NPSN", npsn, "NPSN tidak terdaftar di Sudin ini")
		case !dalamJenjang(p, sk.Jenjang):
			pm.tambah(lembar, baris, "NPSN", npsn, "jenjang "+sk.Jenjang+" tidak termasuk periode ini")
		default:
			return sk, true
		}
		return sk, false
	}

	peserta := map[uuid.UUID]bool{}
	seenNPSN := map[string]bool{}
	for i, row := range lSekolah.baris {
		b := lSekolah.nomor[i]
		npsn := lSekolah.sel(row, "NPSN")
		sk, ok := cekNPSN(lembarSekolah, b, npsn)
		if ok && seenNPSN[npsn] {
			pm.tambah(lembarSekolah, b, "NPSN", npsn, "NPSN ganda di lembar Sekolah")
		}
		if j := strings.ToUpper(lSekolah.sel(row, "Jenjang")); ok && j != "" && j != sk.Jenjang {
			pm.tambah(lembarSekolah, b, "Jenjang", j, "tidak sesuai data sekolah ("+sk.Jenjang+")")
		}
		seenNPSN[npsn] = true
		if pm.valid(lembarSekolah, b) {
			peserta[sk.ID] = true
		}
	}

	var rombel []rombelImpor
	seenTingkat := map[string]bool{}
	for i, row := range lRombel.baris {
		b := lRombel.nomor[i]
		sk, ok := cekNPSN(lembarRombel, b, lRombel.sel(row, "NPSN"))
		tRaw := lRombel.sel(row, "Tingkat")
		tingkat, okT := angka(tRaw, 1, 12)
		if !okT {
			tingkat, okT = romawi[strings.ToUpper(tRaw)]
		}
		if r := rentangTingkat[sk.Jenjang]; ok && (!okT || tingkat < r[0] || tingkat > r[1]) {
			pm.tambah(lembarRombel, b, "Tingkat", tRaw, fmt.Sprintf("tingkat %s harus %d–%d", sk.Jenjang, r[0], r[1]))
		}
		jr, okR := angka(lRombel.sel(row, "Jumlah Rombel"), 0, 200)
		if !okR {
			pm.tambah(lembarRombel, b, "Jumlah Rombel", lRombel.sel(row, "Jumlah Rombel"), "harus bilangan bulat 0–200")
		}
		js, okS := angka(lRombel.sel(row, "Jumlah Siswa"), 0, 20000)
		if !okS {
			pm.tambah(lembarRombel, b, "Jumlah Siswa", lRombel.sel(row, "Jumlah Siswa"), "harus bilangan bulat 0–20000")
		}
		k := fmt.Sprintf("%s|%d", sk.NPSN, tingkat)
		if ok && okT && seenTingkat[k] {
			pm.tambah(lembarRombel, b, "Tingkat", tRaw, "tingkat ganda untuk sekolah ini")
		}
		seenTingkat[k] = true
		if pm.valid(lembarRombel, b) {
			rombel = append(rombel, rombelImpor{sk.ID, entities.AnjabRombel{Tingkat: tingkat, JumlahRombel: jr, JumlahSiswa: js}})
		}
	}

	var nips []string
	for _, row := range lGuru.baris {
		if nip := strings.Join(strings.Fields(lGuru.sel(row, "NIP")), ""); nip != "" {
			nips = append(nips, nip)
		}
	}
	terdaftar, err := s.repo.PendidikByNIP(ctx, nil, nips)
	if err != nil {
		return res, err
	}
	nipLain := map[string]bool{}
	for _, x := range terdaftar {
		if !sekolahID[x.SekolahID] {
			nipLain[*x.NIP] = true
		}
	}

	var guru []guruImpor
	seenNIP := map[string]bool{}
	for i, row := range lGuru.baris {
		b := lGuru.nomor[i]
		sk, _ := cekNPSN(lembarGuru, b, lGuru.sel(row, "NPSN"))
		g := guruImpor{
			sekolah: sk, nip: strings.Join(strings.Fields(lGuru.sel(row, "NIP")), ""), nik: lGuru.sel(row, "NIK"),
			nama: rapikan(lGuru.sel(row, "Nama")), golongan: lGuru.sel(row, "Golongan"),
			mapelSK: rapikan(lGuru.sel(row, "Mapel SK")), mapelRiil: rapikan(lGuru.sel(row, "Mapel Riil")),
		}
		stRaw := lGuru.sel(row, "Status Kepegawaian")
		g.status = statusPeg[strings.ToUpper(rapikan(stRaw))]
		if g.status == "" {
			pm.tambah(lembarGuru, b, "Status Kepegawaian", stRaw, "harus PNS, PPPK, PPPK_PW, atau KKI")
		}
		switch {
		case g.nip == "" && (g.status == entities.StatusKepegawaianPNS || g.status == entities.StatusKepegawaianPPPK):
			pm.tambah(lembarGuru, b, "NIP", "(kosong)", "NIP wajib untuk PNS dan PPPK")
		case g.nip != "" && (len(g.nip) != 18 || !polaAngka.MatchString(g.nip)):
			pm.tambah(lembarGuru, b, "NIP", lGuru.sel(row, "NIP"), "NIP harus 18 digit")
		case g.nip != "" && seenNIP[g.nip]:
			pm.tambah(lembarGuru, b, "NIP", lGuru.sel(row, "NIP"), "NIP ganda di lembar Guru")
		case nipLain[g.nip]:
			pm.tambah(lembarGuru, b, "NIP", lGuru.sel(row, "NIP"), "NIP terdaftar di sekolah di luar Sudin ini")
		}
		if g.nip != "" {
			seenNIP[g.nip] = true
		}
		switch {
		case g.nik == "":
			pm.tambah(lembarGuru, b, "NIK", "(kosong)", "NIK wajib diisi")
		case len(g.nik) != 16 || !polaAngka.MatchString(g.nik):
			pm.tambah(lembarGuru, b, "NIK", g.nik, "NIK harus 16 digit")
		}
		if g.nama == "" || len([]rune(g.nama)) > 150 {
			pm.tambah(lembarGuru, b, "Nama", g.nama, "nama wajib diisi, maksimal 150 karakter")
		}
		if len(g.golongan) > 10 {
			pm.tambah(lembarGuru, b, "Golongan", g.golongan, "maksimal 10 karakter")
		}
		var ok bool
		if g.lahir, ok = tanggalSel(lGuru.sel(row, "Tanggal Lahir")); !ok {
			pm.tambah(lembarGuru, b, "Tanggal Lahir", lGuru.sel(row, "Tanggal Lahir"), "format tanggal harus YYYY-MM-DD")
		}
		if g.tmtCPNS, ok = tanggalSel(lGuru.sel(row, "TMT CPNS")); !ok {
			pm.tambah(lembarGuru, b, "TMT CPNS", lGuru.sel(row, "TMT CPNS"), "format tanggal harus YYYY-MM-DD")
		}
		if g.mapelSK == "" || len([]rune(g.mapelSK)) > 100 {
			pm.tambah(lembarGuru, b, "Mapel SK", g.mapelSK, "mapel SK wajib diisi, maksimal 100 karakter")
		}
		if len([]rune(g.mapelRiil)) > 100 {
			pm.tambah(lembarGuru, b, "Mapel Riil", g.mapelRiil, "maksimal 100 karakter")
		}
		if v := lGuru.sel(row, "JP Riil"); v != "" {
			if g.jpRiil, ok = angka(v, 0, 60); !ok {
				pm.tambah(lembarGuru, b, "JP Riil", v, "harus bilangan bulat 0–60")
			}
		}
		if pm.valid(lembarGuru, b) {
			guru = append(guru, g)
		}
	}

	res.Galat = pm.galat
	res.BarisBermasalah = len(pm.salah)
	res.BarisValid = res.TotalBaris - res.BarisBermasalah
	res.PerLembar = map[string]int{
		lembarSekolah: len(lSekolah.baris) - hitungSalah(pm, lembarSekolah, lSekolah.nomor),
		lembarRombel:  len(rombel),
		lembarGuru:    len(guru),
	}
	if !req.Simpan {
		return res, nil
	}
	if res.BarisValid == 0 {
		return res, utils.ErrBadRequest("tidak ada baris valid untuk disimpan")
	}

	err = s.transaksi(ctx, func(tx *gorm.DB) error {
		p, err := s.repo.GetPeriode(ctx, tx, actor, id, true)
		if err != nil {
			return err
		}
		if p.DibukaAt != nil {
			return utils.ErrConflict("impor hanya bisa sebelum masa pembaruan dibuka")
		}
		for _, r := range rombel {
			peserta[r.sekolahID] = true
		}
		for _, g := range guru {
			peserta[g.sekolah.ID] = true
		}
		anjab := map[uuid.UUID]entities.AnjabSekolah{}
		for sid := range peserta {
			a, _, err := s.pastikanPeserta(ctx, tx, actor, p.ID, sid)
			if err != nil {
				return err
			}
			anjab[sid] = a
		}

		rombelPer := map[uuid.UUID][]entities.AnjabRombel{}
		for _, r := range rombel {
			rombelPer[r.sekolahID] = append(rombelPer[r.sekolahID], r.row)
		}
		for sid, rows := range rombelPer {
			if err := s.repo.ReplaceRombel(ctx, tx, anjab[sid], rows); err != nil {
				return err
			}
		}

		guruPer, err := s.upsertPendidik(ctx, tx, guru)
		if err != nil {
			return err
		}
		for sid, rows := range guruPer {
			if err := s.repo.ReplaceGuru(ctx, tx, anjab[sid], rows); err != nil {
				return err
			}
		}
		return aktivitas.Catat(tx, actor.ID, "anjab.impor", "anjab_periode", p.ID,
			fmt.Sprintf("%s · %d baris valid dari %d", res.NamaBerkas, res.BarisValid, res.TotalBaris))
	})
	if err != nil {
		return res, err
	}
	res.Tersimpan = true
	return res, nil
}

func hitungSalah(pm *pemeriksa, lembar string, nomor []int) int {
	n := 0
	for _, b := range nomor {
		if !pm.valid(lembar, b) {
			n++
		}
	}
	return n
}

// upsertPendidik menyimpan guru hasil impor ke tabel pendidik: cocok berdasar NIP, atau
// (tanpa NIP) berdasar nama di sekolah yang sama. Mengembalikan baris anjab_guru per sekolah.
func (s *anjabService) upsertPendidik(ctx context.Context, tx *gorm.DB, guru []guruImpor) (map[uuid.UUID][]entities.AnjabGuru, error) {
	var nips []string
	var sekolahTanpaNIP []uuid.UUID
	for _, g := range guru {
		if g.nip != "" {
			nips = append(nips, g.nip)
		} else {
			sekolahTanpaNIP = append(sekolahTanpaNIP, g.sekolah.ID)
		}
	}
	adaNIP, err := s.repo.PendidikByNIP(ctx, tx, nips)
	if err != nil {
		return nil, err
	}
	byNIP := map[string]entities.Pendidik{}
	for _, x := range adaNIP {
		byNIP[*x.NIP] = x
	}
	tanpaNIP, err := s.repo.PendidikTanpaNIP(ctx, tx, sekolahTanpaNIP)
	if err != nil {
		return nil, err
	}
	byNama := map[string]entities.Pendidik{}
	for _, x := range tanpaNIP {
		byNama[x.SekolahID.String()+"|"+strings.ToLower(x.Nama)] = x
	}

	out := map[uuid.UUID][]entities.AnjabGuru{}
	seen := map[uuid.UUID]map[uuid.UUID]bool{}
	for _, g := range guru {
		var pd entities.Pendidik
		var ada bool
		if g.nip != "" {
			pd, ada = byNIP[g.nip]
		} else {
			pd, ada = byNama[g.sekolah.ID.String()+"|"+strings.ToLower(g.nama)]
		}
		if !ada {
			pd = entities.Pendidik{Jabatan: "Guru Mapel"}
			if hitung.KunciMapel(g.mapelSK) == hitung.KunciMapel(hitung.MapelGuruKelas) {
				pd.Jabatan = hitung.MapelGuruKelas
			}
		}
		pd.Sekolah = nil
		pd.SekolahID, pd.Nama, pd.NIK, pd.StatusKepegawaian = g.sekolah.ID, g.nama, g.nik, g.status
		if g.nip != "" {
			nip := g.nip
			pd.NIP = &nip
		}
		if g.golongan != "" {
			pd.Golongan = g.golongan
		}
		if g.lahir != nil {
			pd.TanggalLahir = g.lahir
		}
		if g.tmtCPNS != nil {
			pd.TMTCPNS = g.tmtCPNS
		}
		if err := s.repo.SavePendidik(ctx, tx, &pd); err != nil {
			if duplikat(err) {
				return nil, utils.ErrConflict("NIP " + g.nip + " sudah terdaftar")
			}
			return nil, err
		}
		if g.nip == "" {
			byNama[g.sekolah.ID.String()+"|"+strings.ToLower(g.nama)] = pd
		}
		if seen[g.sekolah.ID] == nil {
			seen[g.sekolah.ID] = map[uuid.UUID]bool{}
		}
		if seen[g.sekolah.ID][pd.ID] {
			continue // nama sama tanpa NIP di sekolah yang sama: dihitung sekali
		}
		seen[g.sekolah.ID][pd.ID] = true
		out[g.sekolah.ID] = append(out[g.sekolah.ID], entities.AnjabGuru{
			PendidikID: pd.ID, MapelSK: g.mapelSK, MapelRiil: g.mapelRiil, JPRiil: g.jpRiil,
		})
	}
	return out, nil
}
