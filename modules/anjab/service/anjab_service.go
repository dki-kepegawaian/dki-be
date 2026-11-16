package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"github.com/Caknoooo/go-gin-clean-starter/modules/anjab/dto"
	"github.com/Caknoooo/go-gin-clean-starter/modules/anjab/hitung"
	"github.com/Caknoooo/go-gin-clean-starter/modules/anjab/repository"
	notifikasiService "github.com/Caknoooo/go-gin-clean-starter/modules/notifikasi/service"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/aktivitas"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// Urutan baku jenjang dan rentang tingkat kelasnya.
var (
	urutanJenjang  = []string{"SD", "SMP", "SMA", "SMK"}
	rentangTingkat = map[string][2]int{"SD": {1, 6}, "SMP": {7, 9}, "SMA": {10, 12}, "SMK": {10, 12}}
	polaTahun      = regexp.MustCompile(`^(\d{4})/(\d{4})$`)
)

type AnjabService interface {
	BuatPeriode(ctx context.Context, actor entities.User, req dto.PeriodeRequest) (dto.PeriodeResponse, error)
	UbahPeriode(ctx context.Context, actor entities.User, id string, req dto.PeriodeRequest) (dto.PeriodeResponse, error)
	DaftarPeriode(ctx context.Context, actor entities.User, offset, limit int) ([]dto.PeriodeResponse, int64, error)
	DetailPeriode(ctx context.Context, actor entities.User, id string) (dto.PeriodeResponse, error)
	BukaPembaruan(ctx context.Context, actor entities.User, id string) (dto.PeriodeResponse, error)
	PerpanjangTenggat(ctx context.Context, actor entities.User, id string, req dto.TenggatRequest) (dto.PeriodeResponse, error)
	KirimPengingat(ctx context.Context, actor entities.User, id string) (int, error)
	Templat(ctx context.Context, actor entities.User, id string) ([]byte, string, error)
	Impor(ctx context.Context, actor entities.User, id string, req dto.ImporRequest) (dto.ImporResponse, error)
	Monitoring(ctx context.Context, actor entities.User, id, mapel string) (dto.MonitoringResponse, error)
	MonitoringExcel(ctx context.Context, actor entities.User, id string) ([]byte, string, error)
	LaporanPDF(ctx context.Context, actor entities.User, id string) ([]byte, string, error)

	Daftar(ctx context.Context, actor entities.User, f dto.Filter) ([]dto.AnjabSekolahResponse, int64, error)
	Detail(ctx context.Context, actor entities.User, id string) (dto.AnjabSekolahResponse, error)
	UbahRombel(ctx context.Context, actor entities.User, id string, req dto.RombelRequest) (dto.AnjabSekolahResponse, error)
	TambahGuru(ctx context.Context, actor entities.User, id string, req dto.GuruTambahRequest) (dto.AnjabSekolahResponse, error)
	UbahGuru(ctx context.Context, actor entities.User, id, guruID string, req dto.GuruUbahRequest) (dto.AnjabSekolahResponse, error)
	HapusGuru(ctx context.Context, actor entities.User, id, guruID string) (dto.AnjabSekolahResponse, error)
	Kirim(ctx context.Context, actor entities.User, id string) (dto.AnjabSekolahResponse, error)
	Tinjau(ctx context.Context, actor entities.User, id string, req dto.TinjauRequest) (dto.AnjabSekolahResponse, error)
}

type anjabService struct {
	repo       repository.AnjabRepository
	notifikasi notifikasiService.NotifikasiService
	db         *gorm.DB
}

func NewAnjabService(repo repository.AnjabRepository, notifikasi notifikasiService.NotifikasiService, db *gorm.DB) AnjabService {
	return &anjabService{repo: repo, notifikasi: notifikasi, db: db}
}

func (s *anjabService) transaksi(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return s.db.WithContext(ctx).Transaction(fn)
}

func duplikat(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func parseTgl(s string) time.Time {
	t, _ := time.Parse(dto.TglFormat, s) // format sudah divalidasi binding
	return t
}

// hariIni dalam WIB, format 2006-01-02 (dibandingkan sebagai string).
func hariIni() string {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		loc = time.FixedZone("WIB", 7*3600)
	}
	return time.Now().In(loc).Format(dto.TglFormat)
}

func tglID(t time.Time) string {
	bulan := []string{"Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}
	return fmt.Sprintf("%d %s %d", t.Day(), bulan[t.Month()-1], t.Year())
}

func jenjangPeriode(p entities.AnjabPeriode) []string { return strings.Split(p.Jenjang, ",") }

func dalamJenjang(p entities.AnjabPeriode, jenjang string) bool {
	for _, j := range jenjangPeriode(p) {
		if j == jenjang {
			return true
		}
	}
	return false
}

// isiPeriode memvalidasi request A1 dan mengisi entitas beserta beban JP-nya.
func isiPeriode(p *entities.AnjabPeriode, req dto.PeriodeRequest) ([]entities.AnjabBeban, error) {
	m := polaTahun.FindStringSubmatch(req.TahunAjaran)
	if m == nil {
		return nil, utils.ErrBadRequest("tahun ajaran harus berformat 2026/2027")
	}
	awal, _ := strconv.Atoi(m[1])
	akhir, _ := strconv.Atoi(m[2])
	if akhir != awal+1 || awal < 2000 || awal > 2100 {
		return nil, utils.ErrBadRequest("tahun ajaran harus dua tahun berurutan, mis. 2026/2027")
	}
	pilih := map[string]bool{}
	for _, j := range req.Jenjang {
		pilih[j] = true
	}
	var jenjang []string
	for _, j := range urutanJenjang {
		if pilih[j] {
			jenjang = append(jenjang, j)
		}
	}
	buka, tutup := parseTgl(req.TanggalBuka), parseTgl(req.TanggalTutup)
	if tutup.Before(buka) {
		return nil, utils.ErrBadRequest("tanggal tutup pembaruan tidak boleh sebelum tanggal buka")
	}

	var beban []entities.AnjabBeban
	seen := map[string]bool{}
	for i, b := range req.Beban {
		mapel := strings.Join(strings.Fields(b.Mapel), " ")
		if mapel == "" {
			return nil, utils.ErrBadRequest(fmt.Sprintf("beban ke-%d: mapel wajib diisi", i+1))
		}
		if !pilih[b.Jenjang] {
			return nil, utils.ErrBadRequest(fmt.Sprintf("beban ke-%d: jenjang %s tidak termasuk periode ini", i+1, b.Jenjang))
		}
		if r := rentangTingkat[b.Jenjang]; b.Tingkat < r[0] || b.Tingkat > r[1] {
			return nil, utils.ErrBadRequest(fmt.Sprintf("beban ke-%d: tingkat %s harus %d–%d", i+1, b.Jenjang, r[0], r[1]))
		}
		k := fmt.Sprintf("%s|%s|%d", b.Jenjang, hitung.KunciMapel(mapel), b.Tingkat)
		if seen[k] {
			return nil, utils.ErrBadRequest(fmt.Sprintf("beban ke-%d: %s %s tingkat %d ganda", i+1, b.Jenjang, mapel, b.Tingkat))
		}
		seen[k] = true
		beban = append(beban, entities.AnjabBeban{PeriodeID: p.ID, Jenjang: b.Jenjang, Mapel: mapel, Tingkat: b.Tingkat, JP: b.JP})
	}

	p.TahunAjaran = req.TahunAjaran
	p.Jenjang = strings.Join(jenjang, ",")
	p.TanggalBuka, p.TanggalTutup = buka, tutup
	p.AcuanPeraturan = strings.TrimSpace(req.AcuanPeraturan)
	p.JamWajib = req.JamWajib
	p.RasioSD, p.RasioSMP, p.RasioSMA = req.RasioSD, req.RasioSMP, req.RasioSMA
	p.BatasTMTPensiun = parseTgl(req.BatasTMTPensiun)
	return beban, nil
}

func (s *anjabService) BuatPeriode(ctx context.Context, actor entities.User, req dto.PeriodeRequest) (dto.PeriodeResponse, error) {
	if actor.SudinID == nil {
		return dto.PeriodeResponse{}, utils.ErrForbidden("akun Anda tidak terhubung ke Sudin")
	}
	p := entities.AnjabPeriode{ID: uuid.New(), SudinID: *actor.SudinID, DibuatOleh: actor.ID}
	beban, err := isiPeriode(&p, req)
	if err != nil {
		return dto.PeriodeResponse{}, err
	}
	err = s.transaksi(ctx, func(tx *gorm.DB) error {
		if err := s.repo.CreatePeriode(ctx, tx, &p); err != nil {
			if duplikat(err) {
				return utils.ErrConflict("periode " + p.TahunAjaran + " sudah ada")
			}
			return err
		}
		if err := s.repo.ReplaceBeban(ctx, tx, p.ID, beban); err != nil {
			return err
		}
		return aktivitas.Catat(tx, actor.ID, "anjab.buat_periode", "anjab_periode", p.ID, p.TahunAjaran)
	})
	if err != nil {
		return dto.PeriodeResponse{}, err
	}
	return s.DetailPeriode(ctx, actor, p.ID.String())
}

func (s *anjabService) UbahPeriode(ctx context.Context, actor entities.User, id string, req dto.PeriodeRequest) (dto.PeriodeResponse, error) {
	err := s.transaksi(ctx, func(tx *gorm.DB) error {
		p, err := s.repo.GetPeriode(ctx, tx, actor, id, true)
		if err != nil {
			return err
		}
		if p.DibukaAt != nil {
			return utils.ErrConflict("parameter terkunci karena masa pembaruan sudah dibuka")
		}
		p.Beban = nil
		beban, err := isiPeriode(&p, req)
		if err != nil {
			return err
		}
		if err := s.repo.UpdatePeriode(ctx, tx, &p); err != nil {
			if duplikat(err) {
				return utils.ErrConflict("periode " + p.TahunAjaran + " sudah ada")
			}
			return err
		}
		if err := s.repo.ReplaceBeban(ctx, tx, p.ID, beban); err != nil {
			return err
		}
		return aktivitas.Catat(tx, actor.ID, "anjab.ubah_periode", "anjab_periode", p.ID, p.TahunAjaran)
	})
	if err != nil {
		return dto.PeriodeResponse{}, err
	}
	return s.DetailPeriode(ctx, actor, id)
}

func (s *anjabService) DaftarPeriode(ctx context.Context, actor entities.User, offset, limit int) ([]dto.PeriodeResponse, int64, error) {
	items, total, err := s.repo.ListPeriode(ctx, actor, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	out := make([]dto.PeriodeResponse, 0, len(items))
	for _, p := range items {
		out = append(out, dto.PeriodeFromEntity(p))
	}
	return out, total, nil
}

func (s *anjabService) DetailPeriode(ctx context.Context, actor entities.User, id string) (dto.PeriodeResponse, error) {
	p, err := s.repo.GetPeriode(ctx, nil, actor, id, false)
	if err != nil {
		return dto.PeriodeResponse{}, err
	}
	r := dto.PeriodeFromEntity(p)
	if r.Progres, err = s.repo.Progres(ctx, nil, p.ID); err != nil {
		return dto.PeriodeResponse{}, err
	}
	return r, nil
}

func (s *anjabService) kirimKeOperator(ctx context.Context, tx *gorm.DB, sekolahID uuid.UUID, judul, pesan, tautan string) error {
	ids, err := s.repo.OperatorSekolah(ctx, tx, sekolahID)
	if err != nil {
		return err
	}
	return s.notifikasi.Kirim(ctx, tx, ids, judul, pesan, tautan)
}

// pastikanPeserta membuat data Anjab berstatus draf bila sekolah belum terdaftar di periode.
func (s *anjabService) pastikanPeserta(ctx context.Context, tx *gorm.DB, actor entities.User, periodeID, sekolahID uuid.UUID) (entities.AnjabSekolah, bool, error) {
	a, err := s.repo.GetAnjabSekolahByPeriode(ctx, tx, periodeID, sekolahID)
	if err == nil {
		return a, false, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return a, false, err
	}
	a = entities.AnjabSekolah{PeriodeID: periodeID, SekolahID: sekolahID, Status: entities.AnjabStatusDraf}
	if err := s.repo.CreateAnjabSekolah(ctx, tx, &a); err != nil {
		return a, false, err
	}
	err = s.repo.AddRiwayat(ctx, tx, &entities.AnjabRiwayat{AnjabSekolahID: a.ID, KeStatus: a.Status, Oleh: actor.ID})
	return a, true, err
}

func (s *anjabService) BukaPembaruan(ctx context.Context, actor entities.User, id string) (dto.PeriodeResponse, error) {
	err := s.transaksi(ctx, func(tx *gorm.DB) error {
		p, err := s.repo.GetPeriode(ctx, tx, actor, id, true)
		if err != nil {
			return err
		}
		if p.DibukaAt != nil {
			return utils.ErrConflict("masa pembaruan periode ini sudah dibuka")
		}
		if len(p.Beban) == 0 {
			return utils.ErrBadRequest("isi beban JP per mapel sebelum membuka masa pembaruan")
		}
		if hariIni() > p.TanggalTutup.Format(dto.TglFormat) {
			return utils.ErrBadRequest("tanggal tutup pembaruan sudah lewat; ubah periode dulu")
		}
		now := time.Now()
		p.DibukaAt, p.DibukaOleh = &now, &actor.ID
		p.Beban = nil
		if err := s.repo.UpdatePeriode(ctx, tx, &p); err != nil {
			return err
		}
		sekolah, err := s.repo.SekolahSudin(ctx, tx, p.SudinID, jenjangPeriode(p))
		if err != nil {
			return err
		}
		for _, sk := range sekolah {
			a, _, err := s.pastikanPeserta(ctx, tx, actor, p.ID, sk.ID)
			if err != nil {
				return err
			}
			if err := s.kirimKeOperator(ctx, tx, sk.ID, "Masa pembaruan Anjab dibuka",
				fmt.Sprintf("Masa pembaruan Analisis Jabatan %s dibuka hingga %s. Perbarui rombel dan data guru.", p.TahunAjaran, tglID(p.TanggalTutup)),
				tautan(a.ID)); err != nil {
				return err
			}
		}
		return aktivitas.Catat(tx, actor.ID, "anjab.buka_pembaruan", "anjab_periode", p.ID,
			fmt.Sprintf("%s · %d sekolah", p.TahunAjaran, len(sekolah)))
	})
	if err != nil {
		return dto.PeriodeResponse{}, err
	}
	return s.DetailPeriode(ctx, actor, id)
}

var statusBelumKirim = []string{entities.AnjabStatusDraf, entities.AnjabStatusDikembalikan}

func (s *anjabService) PerpanjangTenggat(ctx context.Context, actor entities.User, id string, req dto.TenggatRequest) (dto.PeriodeResponse, error) {
	tutup := parseTgl(req.TanggalTutup)
	err := s.transaksi(ctx, func(tx *gorm.DB) error {
		p, err := s.repo.GetPeriode(ctx, tx, actor, id, true)
		if err != nil {
			return err
		}
		if p.DibukaAt == nil {
			return utils.ErrConflict("masa pembaruan belum dibuka; ubah tanggal lewat pengaturan periode")
		}
		if !tutup.After(p.TanggalTutup) {
			return utils.ErrBadRequest("tenggat baru harus setelah " + tglID(p.TanggalTutup))
		}
		p.TanggalTutup = tutup
		p.Beban = nil
		if err := s.repo.UpdatePeriode(ctx, tx, &p); err != nil {
			return err
		}
		belum, err := s.repo.ListByPeriode(ctx, tx, p.ID, statusBelumKirim)
		if err != nil {
			return err
		}
		for _, a := range belum {
			if err := s.kirimKeOperator(ctx, tx, a.SekolahID, "Tenggat Anjab diperpanjang",
				fmt.Sprintf("Tenggat pembaruan Analisis Jabatan %s diperpanjang hingga %s.", p.TahunAjaran, tglID(tutup)), tautan(a.ID)); err != nil {
				return err
			}
		}
		return aktivitas.Catat(tx, actor.ID, "anjab.perpanjang_tenggat", "anjab_periode", p.ID, req.TanggalTutup)
	})
	if err != nil {
		return dto.PeriodeResponse{}, err
	}
	return s.DetailPeriode(ctx, actor, id)
}

func (s *anjabService) KirimPengingat(ctx context.Context, actor entities.User, id string) (int, error) {
	var n int
	err := s.transaksi(ctx, func(tx *gorm.DB) error {
		p, err := s.repo.GetPeriode(ctx, tx, actor, id, false)
		if err != nil {
			return err
		}
		if p.DibukaAt == nil {
			return utils.ErrConflict("masa pembaruan belum dibuka")
		}
		belum, err := s.repo.ListByPeriode(ctx, tx, p.ID, statusBelumKirim)
		if err != nil {
			return err
		}
		for _, a := range belum {
			if err := s.kirimKeOperator(ctx, tx, a.SekolahID, "Pengingat pembaruan Anjab",
				fmt.Sprintf("Data Anjab %s sekolah Anda belum dikirim. Tenggat %s.", p.TahunAjaran, tglID(p.TanggalTutup)), tautan(a.ID)); err != nil {
				return err
			}
		}
		n = len(belum)
		return aktivitas.Catat(tx, actor.ID, "anjab.kirim_pengingat", "anjab_periode", p.ID, fmt.Sprintf("%d sekolah", n))
	})
	return n, err
}

// paramHitung menyusun parameter perhitungan periode untuk satu jenjang.
func paramHitung(p entities.AnjabPeriode, jenjang string) hitung.Param {
	rasio := p.RasioSMA
	switch jenjang {
	case "SD":
		rasio = p.RasioSD
	case "SMP":
		rasio = p.RasioSMP
	}
	hp := hitung.Param{Jenjang: jenjang, JamWajib: p.JamWajib, Rasio: rasio, BatasTMTPensiun: p.BatasTMTPensiun}
	for _, b := range p.Beban {
		if b.Jenjang == jenjang {
			hp.Beban = append(hp.Beban, hitung.BebanInput{Mapel: b.Mapel, Tingkat: b.Tingkat, JP: b.JP})
		}
	}
	return hp
}

// hasilSekolah: hasil yang dibekukan bila sudah ditetapkan, selain itu dihitung dari data saat ini.
// a harus dimuat lewat GetDetail (periode+beban, sekolah, rombel, guru+pendidik).
func hasilSekolah(a entities.AnjabSekolah) (hitung.Hasil, error) {
	if a.Status == entities.AnjabStatusDitetapkan && a.Hasil != "" {
		var h hitung.Hasil
		err := json.Unmarshal([]byte(a.Hasil), &h)
		return h, err
	}
	rombel := make([]hitung.RombelInput, 0, len(a.Rombel))
	for _, r := range a.Rombel {
		rombel = append(rombel, hitung.RombelInput{Tingkat: r.Tingkat, JumlahRombel: r.JumlahRombel, JumlahSiswa: r.JumlahSiswa})
	}
	guru := make([]hitung.GuruInput, 0, len(a.Guru))
	for _, g := range a.Guru {
		in := hitung.GuruInput{MapelSK: g.MapelSK, MapelRiil: g.MapelRiil}
		if g.Pendidik != nil {
			in.Nama, in.TanggalLahir = g.Pendidik.Nama, g.Pendidik.TanggalLahir
		}
		guru = append(guru, in)
	}
	return hitung.Sekolah(paramHitung(*a.Periode, a.Sekolah.Jenjang), rombel, guru), nil
}

func detailResponse(a entities.AnjabSekolah) (dto.AnjabSekolahResponse, error) {
	totalRombel := 0
	for _, r := range a.Rombel {
		totalRombel += r.JumlahRombel
	}
	res := dto.AnjabSekolahFromEntity(a, totalRombel, len(a.Guru))
	res.Rombel = []dto.RombelResponse{}
	for _, r := range a.Rombel {
		res.Rombel = append(res.Rombel, dto.RombelResponse{Tingkat: r.Tingkat, JumlahRombel: r.JumlahRombel, JumlahSiswa: r.JumlahSiswa})
	}
	res.Guru = []dto.GuruResponse{}
	for _, g := range a.Guru {
		item := dto.GuruResponse{
			ID: g.ID.String(), PendidikID: g.PendidikID.String(), MapelSK: g.MapelSK, MapelRiil: g.MapelRiil,
			JPRiil: g.JPRiil, Dihitung: true,
		}
		if p := g.Pendidik; p != nil {
			item.Nama, item.NIP, item.StatusKepegawaian, item.Golongan = p.Nama, p.NIP, p.StatusKepegawaian, p.Golongan
			if p.TanggalLahir != nil {
				t := hitung.TMTPensiun(*p.TanggalLahir, hitung.BatasUsiaPensiun).Format(dto.TglFormat)
				item.TMTPensiun = &t
				item.Dihitung = !hitung.Dikeluarkan(p.TanggalLahir, a.Periode.BatasTMTPensiun)
			}
		}
		res.Guru = append(res.Guru, item)
	}
	h, err := hasilSekolah(a)
	if err != nil {
		return res, err
	}
	res.Hasil = &h
	return res, nil
}

func (s *anjabService) Daftar(ctx context.Context, actor entities.User, f dto.Filter) ([]dto.AnjabSekolahResponse, int64, error) {
	items, total, err := s.repo.List(ctx, actor, f)
	if err != nil {
		return nil, 0, err
	}
	ids := make([]uuid.UUID, 0, len(items))
	for _, a := range items {
		ids = append(ids, a.ID)
	}
	rombel, guru, err := s.repo.Hitungan(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	out := make([]dto.AnjabSekolahResponse, 0, len(items))
	for _, a := range items {
		out = append(out, dto.AnjabSekolahFromEntity(a, rombel[a.ID], guru[a.ID]))
	}
	return out, total, nil
}

func (s *anjabService) Detail(ctx context.Context, actor entities.User, id string) (dto.AnjabSekolahResponse, error) {
	a, err := s.repo.GetDetail(ctx, nil, actor, id)
	if err != nil {
		return dto.AnjabSekolahResponse{}, err
	}
	return detailResponse(a)
}

// bisaDiubah: operator hanya mengubah data saat masa pembaruan terbuka dan status draf/dikembalikan.
func bisaDiubah(a entities.AnjabSekolah) error {
	if a.Periode.DibukaAt == nil {
		return utils.ErrConflict("masa pembaruan belum dibuka")
	}
	if hariIni() > a.Periode.TanggalTutup.Format(dto.TglFormat) {
		return utils.ErrConflict("masa pembaruan sudah ditutup pada " + tglID(a.Periode.TanggalTutup))
	}
	if a.Status != entities.AnjabStatusDraf && a.Status != entities.AnjabStatusDikembalikan {
		return utils.ErrConflict("data hanya bisa diubah saat berstatus draf atau dikembalikan")
	}
	return nil
}

// ubahData menjalankan perubahan data sekolah oleh operator dengan baris anjab_sekolah terkunci.
func (s *anjabService) ubahData(ctx context.Context, actor entities.User, id, aksi string, fn func(tx *gorm.DB, a entities.AnjabSekolah) error) (dto.AnjabSekolahResponse, error) {
	err := s.transaksi(ctx, func(tx *gorm.DB) error {
		a, err := s.repo.GetForUpdate(ctx, tx, actor, id)
		if err != nil {
			return err
		}
		if err := bisaDiubah(a); err != nil {
			return err
		}
		if err := fn(tx, a); err != nil {
			return err
		}
		return aktivitas.Catat(tx, actor.ID, aksi, "anjab_sekolah", a.ID, a.Sekolah.NPSN+" "+a.Periode.TahunAjaran)
	})
	if err != nil {
		return dto.AnjabSekolahResponse{}, err
	}
	return s.Detail(ctx, actor, id)
}

func (s *anjabService) UbahRombel(ctx context.Context, actor entities.User, id string, req dto.RombelRequest) (dto.AnjabSekolahResponse, error) {
	return s.ubahData(ctx, actor, id, "anjab.ubah_rombel", func(tx *gorm.DB, a entities.AnjabSekolah) error {
		r := rentangTingkat[a.Sekolah.Jenjang]
		seen := map[int]bool{}
		rows := make([]entities.AnjabRombel, 0, len(req.Rombel))
		for _, x := range req.Rombel {
			if x.Tingkat < r[0] || x.Tingkat > r[1] {
				return utils.ErrBadRequest(fmt.Sprintf("tingkat %s harus %d–%d", a.Sekolah.Jenjang, r[0], r[1]))
			}
			if seen[x.Tingkat] {
				return utils.ErrBadRequest(fmt.Sprintf("tingkat %d ganda", x.Tingkat))
			}
			seen[x.Tingkat] = true
			rows = append(rows, entities.AnjabRombel{Tingkat: x.Tingkat, JumlahRombel: x.JumlahRombel, JumlahSiswa: x.JumlahSiswa})
		}
		return s.repo.ReplaceRombel(ctx, tx, a, rows)
	})
}

func rapikan(s string) string { return strings.Join(strings.Fields(s), " ") }

func (s *anjabService) TambahGuru(ctx context.Context, actor entities.User, id string, req dto.GuruTambahRequest) (dto.AnjabSekolahResponse, error) {
	return s.ubahData(ctx, actor, id, "anjab.tambah_guru", func(tx *gorm.DB, a entities.AnjabSekolah) error {
		p, err := s.repo.PendidikDiSekolah(ctx, tx, req.PendidikID, a.SekolahID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return utils.ErrBadRequest("pendidik tidak ditemukan di sekolah ini")
			}
			return err
		}
		if rapikan(req.MapelSK) == "" {
			return utils.ErrBadRequest("mapel SK wajib diisi")
		}
		g := entities.AnjabGuru{
			AnjabSekolahID: a.ID, SekolahID: a.SekolahID, PendidikID: p.ID,
			MapelSK: rapikan(req.MapelSK), MapelRiil: rapikan(req.MapelRiil), JPRiil: req.JPRiil,
		}
		if err := s.repo.SaveGuru(ctx, tx, &g); err != nil {
			if duplikat(err) {
				return utils.ErrConflict(p.Nama + " sudah ada di daftar guru")
			}
			return err
		}
		return nil
	})
}

func (s *anjabService) UbahGuru(ctx context.Context, actor entities.User, id, guruID string, req dto.GuruUbahRequest) (dto.AnjabSekolahResponse, error) {
	return s.ubahData(ctx, actor, id, "anjab.ubah_guru", func(tx *gorm.DB, a entities.AnjabSekolah) error {
		g, err := s.repo.GetGuru(ctx, tx, a.ID, guruID)
		if err != nil {
			return err
		}
		if rapikan(req.MapelRiil) == "" {
			return utils.ErrBadRequest("mapel riil wajib diisi")
		}
		g.MapelRiil, g.JPRiil = rapikan(req.MapelRiil), req.JPRiil
		return s.repo.SaveGuru(ctx, tx, &g)
	})
}

func (s *anjabService) HapusGuru(ctx context.Context, actor entities.User, id, guruID string) (dto.AnjabSekolahResponse, error) {
	return s.ubahData(ctx, actor, id, "anjab.hapus_guru", func(tx *gorm.DB, a entities.AnjabSekolah) error {
		g, err := s.repo.GetGuru(ctx, tx, a.ID, guruID)
		if err != nil {
			return err
		}
		return s.repo.DeleteGuru(ctx, tx, g)
	})
}

func (s *anjabService) catatTransisi(ctx context.Context, tx *gorm.DB, actor entities.User, a *entities.AnjabSekolah, ke, aksi, catatan string) error {
	dari := a.Status
	a.Status = ke
	if err := s.repo.UpdateAnjabSekolah(ctx, tx, a); err != nil {
		return err
	}
	if err := s.repo.AddRiwayat(ctx, tx, &entities.AnjabRiwayat{
		AnjabSekolahID: a.ID, DariStatus: dari, KeStatus: ke, Oleh: actor.ID, Catatan: catatan,
	}); err != nil {
		return err
	}
	return aktivitas.Catat(tx, actor.ID, aksi, "anjab_sekolah", a.ID,
		fmt.Sprintf("%s %s: %s -> %s", a.Sekolah.NPSN, a.Periode.TahunAjaran, dari, ke))
}

func (s *anjabService) Kirim(ctx context.Context, actor entities.User, id string) (dto.AnjabSekolahResponse, error) {
	err := s.transaksi(ctx, func(tx *gorm.DB) error {
		a, err := s.repo.GetForUpdate(ctx, tx, actor, id)
		if err != nil {
			return err
		}
		if err := bisaDiubah(a); err != nil {
			return err
		}
		d, err := s.repo.GetDetail(ctx, tx, actor, id)
		if err != nil {
			return err
		}
		rombel := 0
		for _, r := range d.Rombel {
			rombel += r.JumlahRombel
		}
		if rombel == 0 {
			return utils.ErrBadRequest("isi jumlah rombel sebelum mengirim")
		}
		if len(d.Guru) == 0 {
			return utils.ErrBadRequest("daftar guru masih kosong")
		}
		var kosong []string
		for _, g := range d.Guru {
			if g.MapelRiil == "" && g.Pendidik != nil && !hitung.Dikeluarkan(g.Pendidik.TanggalLahir, a.Periode.BatasTMTPensiun) {
				kosong = append(kosong, g.Pendidik.Nama)
			}
		}
		if len(kosong) > 0 {
			if len(kosong) > 3 {
				kosong = append(kosong[:3], fmt.Sprintf("dan %d lainnya", len(kosong)-3))
			}
			return utils.ErrBadRequest("mapel riil belum diisi: " + strings.Join(kosong, ", "))
		}

		now := time.Now()
		a.KirimanKe++
		a.DikirimAt, a.DikirimOleh = &now, &actor.ID
		if err := s.catatTransisi(ctx, tx, actor, &a, entities.AnjabStatusDikirim, "anjab.kirim",
			fmt.Sprintf("Kiriman ke-%d", a.KirimanKe)); err != nil {
			return err
		}
		staf, err := s.repo.StafSudinSekolah(ctx, tx, a.SekolahID)
		if err != nil {
			return err
		}
		return s.notifikasi.Kirim(ctx, tx, staf, "Pembaruan Anjab dikirim",
			fmt.Sprintf("%s mengirim data Anjab %s (kiriman ke-%d).", a.Sekolah.Nama, a.Periode.TahunAjaran, a.KirimanKe), tautan(a.ID))
	})
	if err != nil {
		return dto.AnjabSekolahResponse{}, err
	}
	return s.Detail(ctx, actor, id)
}

func (s *anjabService) Tinjau(ctx context.Context, actor entities.User, id string, req dto.TinjauRequest) (dto.AnjabSekolahResponse, error) {
	catatan := strings.TrimSpace(req.Catatan)
	if req.Keputusan == entities.AnjabStatusDikembalikan && catatan == "" {
		return dto.AnjabSekolahResponse{}, utils.ErrBadRequest("catatan untuk sekolah wajib diisi jika dikembalikan")
	}
	err := s.transaksi(ctx, func(tx *gorm.DB) error {
		a, err := s.repo.GetForUpdate(ctx, tx, actor, id)
		if err != nil {
			return err
		}
		if a.Status != entities.AnjabStatusDikirim {
			return utils.ErrConflict("hanya data berstatus dikirim yang bisa ditinjau")
		}
		now := time.Now()
		a.DitinjauAt, a.DitinjauOleh = &now, &actor.ID
		a.CatatanSudin = catatan

		judul, pesan := "Anjab dikembalikan", fmt.Sprintf("Data Anjab %s dikembalikan Sudin: %s", a.Periode.TahunAjaran, catatan)
		if req.Keputusan == entities.AnjabStatusDitetapkan {
			d, err := s.repo.GetDetail(ctx, tx, actor, id)
			if err != nil {
				return err
			}
			h, err := hasilSekolah(d)
			if err != nil {
				return err
			}
			raw, err := json.Marshal(h)
			if err != nil {
				return err
			}
			a.Hasil = string(raw)
			judul = "Anjab ditetapkan"
			pesan = fmt.Sprintf("Hasil Anjab %s ditetapkan Sudin: kurang %d, lebih %d guru (kondisi riil).",
				a.Periode.TahunAjaran, h.Riil.Kurang, h.Riil.Lebih)
		}
		if err := s.catatTransisi(ctx, tx, actor, &a, req.Keputusan, "anjab.tinjau", catatan); err != nil {
			return err
		}
		return s.kirimKeOperator(ctx, tx, a.SekolahID, judul, pesan, tautan(a.ID))
	})
	if err != nil {
		return dto.AnjabSekolahResponse{}, err
	}
	return s.Detail(ctx, actor, id)
}

func tautan(id uuid.UUID) string { return "/anjab/sekolah/" + id.String() }
