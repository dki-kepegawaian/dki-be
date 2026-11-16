package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"strings"
	"time"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"github.com/Caknoooo/go-gin-clean-starter/modules/disiplin/dto"
	"github.com/Caknoooo/go-gin-clean-starter/modules/disiplin/repository"
	dokumenService "github.com/Caknoooo/go-gin-clean-starter/modules/dokumen/service"
	notifikasiService "github.com/Caknoooo/go-gin-clean-starter/modules/notifikasi/service"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/aktivitas"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/constants"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/helpers"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/storage"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/utils"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Tenggat per tahap dalam hari kerja (asumsi, lihat NOTES.md).
const (
	tenggatVerifikasi   = 5
	tenggatKelengkapan  = 10
	tenggatPemeriksaan  = 20
	tenggatKeputusan    = 5
	tenggatCatat        = 5
	tenggatPelaksanaan  = 7
	maksBarisKronologi  = 30
	maksKejadianPanjang = 500
)

var wib = time.FixedZone("WIB", 7*3600)

type DisiplinService interface {
	Lapor(ctx context.Context, actor entities.User, req dto.LaporRequest) (dto.DisiplinResponse, error)
	Verifikasi(ctx context.Context, actor entities.User, id string, req dto.VerifikasiRequest) (dto.DisiplinResponse, error)
	Lengkapi(ctx context.Context, actor entities.User, id string, req dto.KelengkapanRequest) (dto.DisiplinResponse, error)
	Pemeriksaan(ctx context.Context, actor entities.User, id string, req dto.PemeriksaanRequest) (dto.DisiplinResponse, error)
	Rekomendasi(ctx context.Context, actor entities.User, id string, req dto.RekomendasiRequest) (dto.DisiplinResponse, error)
	Putuskan(ctx context.Context, actor entities.User, id string, req dto.KeputusanRequest) (dto.DisiplinResponse, error)
	CatatKeputusan(ctx context.Context, actor entities.User, id string, req dto.CatatKeputusanRequest) (dto.DisiplinResponse, error)
	TindakLanjut(ctx context.Context, actor entities.User, id string, req dto.TindakLanjutRequest) (dto.DisiplinResponse, error)
	LaporanPelaksanaan(ctx context.Context, actor entities.User, id string, req dto.LaporanPelaksanaanRequest) (dto.DisiplinResponse, error)
	Tutup(ctx context.Context, actor entities.User, id string, req dto.TutupRequest) (dto.DisiplinResponse, error)
	Detail(ctx context.Context, actor entities.User, id string) (dto.DisiplinResponse, error)
	Daftar(ctx context.Context, actor entities.User, f dto.Filter) ([]dto.DisiplinResponse, int64, error)
	Ringkasan(ctx context.Context, actor entities.User) (dto.RingkasanResponse, error)
	// BukaBerkas mengembalikan isi berkas kasus bila actor berhak. Pemanggil wajib menutup reader.
	BukaBerkas(ctx context.Context, actor entities.User, id, dokumenID string) (entities.Dokumen, io.ReadCloser, int64, error)
}

type disiplinService struct {
	repo       repository.DisiplinRepository
	dokumen    dokumenService.DokumenService
	notifikasi notifikasiService.NotifikasiService
	storage    storage.Storage
	db         *gorm.DB
}

func NewDisiplinService(
	repo repository.DisiplinRepository,
	dokumen dokumenService.DokumenService,
	notifikasi notifikasiService.NotifikasiService,
	st storage.Storage,
	db *gorm.DB,
) DisiplinService {
	return &disiplinService{repo: repo, dokumen: dokumen, notifikasi: notifikasi, storage: st, db: db}
}

func internal(actor entities.User) bool { return actor.Role != constants.ENUM_ROLE_OPERATOR_SEKOLAH }

func hariIni() time.Time {
	n := time.Now().In(wib)
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC)
}

// tambahHariKerja menghitung tanggal n hari kerja (Senin–Jumat) setelah t.
func tambahHariKerja(t time.Time, n int) *time.Time {
	d := t
	for n > 0 {
		d = d.AddDate(0, 0, 1)
		if wd := d.Weekday(); wd != time.Saturday && wd != time.Sunday {
			n--
		}
	}
	return &d
}

func tenggat(n int) *time.Time { return tambahHariKerja(hariIni(), n) }

func parseTgl(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, _ := time.Parse(dto.TglFormat, s) // format sudah divalidasi binding
	return &t
}

// transaksi menjalankan fn dalam satu transaksi. Berkas yang sudah tersimpan
// dihapus dari storage kalau transaksi gagal.
func (s *disiplinService) transaksi(ctx context.Context, fn func(tx *gorm.DB, simpan func(*entities.Dokumen)) error) error {
	var tersimpan []entities.Dokumen
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(tx, func(d *entities.Dokumen) { tersimpan = append(tersimpan, *d) })
	})
	if err != nil {
		s.dokumen.Hapus(tersimpan...)
	}
	return err
}

// lampirkan menyimpan berkas dan menautkannya ke kasus. Berkas internal disimpan tanpa
// sekolah_id supaya tidak bisa diunduh sekolah lewat /api/dokumen; aksesnya hanya lewat
// /api/disiplin/:id/dokumen/:dokumen_id.
func (s *disiplinService) lampirkan(ctx context.Context, tx *gorm.DB, simpan func(*entities.Dokumen), actor entities.User, d *entities.Disiplin, jenis string, fh *multipart.FileHeader) error {
	if fh == nil {
		return nil
	}
	isInternal := entities.DisiplinBerkasInternal(jenis)
	var sekolahID *uuid.UUID
	if !isInternal {
		sekolahID = &d.SekolahID
	}
	doc, err := s.dokumen.Simpan(ctx, tx, fh, sekolahID, actor.ID)
	if err != nil {
		return err
	}
	simpan(&doc)
	return s.repo.AddBerkas(ctx, tx, &entities.DisiplinBerkas{
		DisiplinID: d.ID, DokumenID: doc.ID, Jenis: jenis, Internal: isInternal,
	})
}

func (s *disiplinService) catatTransisi(ctx context.Context, tx *gorm.DB, actor entities.User, d *entities.Disiplin, ke, aksi, catatan string) error {
	dari := d.Status
	d.Status = ke
	if err := s.repo.Update(ctx, tx, d); err != nil {
		return err
	}
	if err := s.repo.AddRiwayat(ctx, tx, &entities.DisiplinRiwayat{
		DisiplinID: d.ID, DariStatus: dari, KeStatus: ke, Oleh: actor.ID, Catatan: catatan,
	}); err != nil {
		return err
	}
	return aktivitas.Catat(tx, actor.ID, aksi, "disiplin", d.ID, fmt.Sprintf("%s: %s -> %s", d.Nomor, dari, ke))
}

// kirim mengirim notifikasi. Isi notifikasi sengaja tanpa nama pegawai (data sensitif).
func (s *disiplinService) kirim(ctx context.Context, tx *gorm.DB, cari func(context.Context, *gorm.DB, uuid.UUID) ([]uuid.UUID, error), d *entities.Disiplin, judul, pesan string) error {
	penerima, err := cari(ctx, tx, d.SekolahID)
	if err != nil {
		return err
	}
	return s.notifikasi.Kirim(ctx, tx, penerima, judul, d.Nomor+" · "+pesan, "/disiplin/"+d.ID.String())
}

func (s *disiplinService) ambil(ctx context.Context, tx *gorm.DB, actor entities.User, id, status, pesan string) (entities.Disiplin, error) {
	d, err := s.repo.GetForUpdate(ctx, tx, actor, id)
	if err != nil {
		return d, err
	}
	if d.Status != status {
		return d, utils.ErrConflict(pesan)
	}
	return d, nil
}

func parseKronologi(raw string) ([]entities.DisiplinKronologi, error) {
	var rows []entities.DisiplinKronologi
	if err := json.Unmarshal([]byte(raw), &rows); err != nil {
		return nil, utils.ErrBadRequest("kronologi harus berupa JSON array [{tanggal, kejadian}]")
	}
	if len(rows) == 0 || len(rows) > maksBarisKronologi {
		return nil, utils.ErrBadRequest(fmt.Sprintf("kronologi wajib berisi 1 sampai %d baris", maksBarisKronologi))
	}
	besok := hariIni().AddDate(0, 0, 1)
	for i := range rows {
		t, err := time.Parse(dto.TglFormat, rows[i].Tanggal)
		if err != nil {
			return nil, utils.ErrBadRequest(fmt.Sprintf("tanggal kronologi baris %d harus berformat YYYY-MM-DD", i+1))
		}
		if !t.Before(besok) {
			return nil, utils.ErrBadRequest(fmt.Sprintf("tanggal kronologi baris %d tidak boleh di masa depan", i+1))
		}
		rows[i].Kejadian = strings.TrimSpace(rows[i].Kejadian)
		if n := len([]rune(rows[i].Kejadian)); n == 0 || n > maksKejadianPanjang {
			return nil, utils.ErrBadRequest(fmt.Sprintf("kejadian kronologi baris %d wajib diisi, maksimal %d karakter", i+1, maksKejadianPanjang))
		}
	}
	return rows, nil
}

func (s *disiplinService) Lapor(ctx context.Context, actor entities.User, req dto.LaporRequest) (dto.DisiplinResponse, error) {
	if !req.Pernyataan {
		return dto.DisiplinResponse{}, utils.ErrBadRequest("centang pernyataan bahwa laporan disusun berdasarkan data yang benar")
	}
	uraian := strings.TrimSpace(req.Uraian)
	if uraian == "" {
		return dto.DisiplinResponse{}, utils.ErrBadRequest("uraian wajib diisi")
	}
	kronologi, err := parseKronologi(req.Kronologi)
	if err != nil {
		return dto.DisiplinResponse{}, err
	}

	var id uuid.UUID
	err = s.transaksi(ctx, func(tx *gorm.DB, simpan func(*entities.Dokumen)) error {
		p, err := s.repo.GetPendidik(ctx, tx, actor, req.PendidikID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return utils.ErrBadRequest("pendidik tidak ditemukan di sekolah Anda")
			}
			return err
		}
		if p.StatusKepegawaian == entities.StatusKepegawaianKKI {
			return utils.ErrBadRequest("Sikap Disiplin hanya untuk ASN (PNS dan PPPK)")
		}
		nomor, err := helpers.NomorRegistrasi(tx, "DSP", time.Now().Year())
		if err != nil {
			return err
		}
		d := entities.Disiplin{
			Nomor: nomor, PendidikID: p.ID, SekolahID: p.SekolahID, DilaporkanOleh: actor.ID,
			Status: entities.DisiplinStatusRegistrasi, Tenggat: tenggat(tenggatVerifikasi),
			JenisPelanggaran: req.JenisPelanggaran, Uraian: uraian, Kronologi: kronologi,
		}
		if err := s.repo.Create(ctx, tx, &d); err != nil {
			return err
		}
		if err := s.lampirkan(ctx, tx, simpan, actor, &d, entities.DisiplinBerkasBuktiAwal, req.BuktiAwal); err != nil {
			return err
		}
		for _, fh := range req.BuktiPendukung {
			if err := s.lampirkan(ctx, tx, simpan, actor, &d, entities.DisiplinBerkasBuktiPendukung, fh); err != nil {
				return err
			}
		}
		if err := s.repo.AddRiwayat(ctx, tx, &entities.DisiplinRiwayat{DisiplinID: d.ID, KeStatus: d.Status, Oleh: actor.ID}); err != nil {
			return err
		}
		if err := aktivitas.Catat(tx, actor.ID, "disiplin.lapor", "disiplin", d.ID, nomor); err != nil {
			return err
		}
		id = d.ID
		d.Sekolah = p.Sekolah
		return s.kirim(ctx, tx, s.repo.StafSudinSekolah, &d, "Laporan dugaan pelanggaran disiplin",
			p.Sekolah.Nama+" mengirim laporan baru untuk verifikasi awal.")
	})
	if err != nil {
		return dto.DisiplinResponse{}, err
	}
	return s.Detail(ctx, actor, id.String())
}

func (s *disiplinService) Verifikasi(ctx context.Context, actor entities.User, id string, req dto.VerifikasiRequest) (dto.DisiplinResponse, error) {
	catatan := strings.TrimSpace(req.Catatan)
	semuaSesuai := *req.CekIdentitas && *req.CekKronologi && *req.CekBukti && *req.CekKewenangan
	switch req.Keputusan {
	case "layak":
		if !semuaSesuai {
			return dto.DisiplinResponse{}, utils.ErrBadRequest("kasus hanya layak diproses jika semua butir daftar periksa terpenuhi")
		}
	case "perlu_kelengkapan":
		if catatan == "" {
			return dto.DisiplinResponse{}, utils.ErrBadRequest("daftar kekurangan wajib diisi")
		}
	case "tidak_layak":
		if catatan == "" {
			return dto.DisiplinResponse{}, utils.ErrBadRequest("alasan tidak layak wajib diisi")
		}
	}
	batas := parseTgl(req.BatasKelengkapan)
	if batas != nil && batas.Before(hariIni()) {
		return dto.DisiplinResponse{}, utils.ErrBadRequest("batas waktu kelengkapan tidak boleh di masa lalu")
	}

	err := s.transaksi(ctx, func(tx *gorm.DB, _ func(*entities.Dokumen)) error {
		d, err := s.ambil(ctx, tx, actor, id, entities.DisiplinStatusRegistrasi, "verifikasi awal hanya untuk kasus berstatus registrasi")
		if err != nil {
			return err
		}
		now := time.Now()
		d.CekIdentitas, d.CekKronologi, d.CekBukti, d.CekKewenangan = req.CekIdentitas, req.CekKronologi, req.CekBukti, req.CekKewenangan
		d.KeputusanVerifikasi, d.CatatanVerifikasi = req.Keputusan, catatan
		d.DiverifikasiOleh, d.DiverifikasiAt = &actor.ID, &now
		d.BatasKelengkapan = nil

		var ke, judul, pesan string
		switch req.Keputusan {
		case "layak":
			ke, judul, pesan = entities.DisiplinStatusPemeriksaan, "Kasus disiplin diproses Sudin",
				"Laporan layak diproses dan masuk tahap pemeriksaan."
			d.Tenggat = tenggat(tenggatPemeriksaan)
		case "perlu_kelengkapan":
			if batas == nil {
				batas = tenggat(tenggatKelengkapan)
			}
			d.BatasKelengkapan, d.Tenggat = batas, batas
			ke, judul, pesan = entities.DisiplinStatusPerluKelengkapan, "Laporan disiplin perlu kelengkapan",
				"Lengkapi dokumen paling lambat "+batas.Format(dto.TglFormat)+"."
		default:
			ke, judul, pesan = entities.DisiplinStatusTidakDilanjutkan, "Laporan disiplin tidak dilanjutkan", catatan
			d.Tenggat = nil
		}
		if err := s.catatTransisi(ctx, tx, actor, &d, ke, "disiplin.verifikasi", catatan); err != nil {
			return err
		}
		return s.kirim(ctx, tx, s.repo.OperatorSekolah, &d, judul, pesan)
	})
	if err != nil {
		return dto.DisiplinResponse{}, err
	}
	return s.Detail(ctx, actor, id)
}

func (s *disiplinService) Lengkapi(ctx context.Context, actor entities.User, id string, req dto.KelengkapanRequest) (dto.DisiplinResponse, error) {
	tanggapan := strings.TrimSpace(req.Tanggapan)
	if tanggapan == "" {
		return dto.DisiplinResponse{}, utils.ErrBadRequest("tanggapan untuk Sudin wajib diisi")
	}
	err := s.transaksi(ctx, func(tx *gorm.DB, simpan func(*entities.Dokumen)) error {
		d, err := s.ambil(ctx, tx, actor, id, entities.DisiplinStatusPerluKelengkapan, "hanya kasus berstatus perlu kelengkapan yang bisa dilengkapi")
		if err != nil {
			return err
		}
		for _, fh := range req.Berkas {
			if err := s.lampirkan(ctx, tx, simpan, actor, &d, entities.DisiplinBerkasKelengkapan, fh); err != nil {
				return err
			}
		}
		now := time.Now()
		d.TanggapanSekolah, d.DilengkapiAt = tanggapan, &now
		d.Tenggat = tenggat(tenggatVerifikasi)
		if err := s.catatTransisi(ctx, tx, actor, &d, entities.DisiplinStatusRegistrasi, "disiplin.lengkapi", tanggapan); err != nil {
			return err
		}
		return s.kirim(ctx, tx, s.repo.StafSudinSekolah, &d, "Kelengkapan kasus disiplin diterima",
			d.Sekolah.Nama+" mengirim kelengkapan laporan untuk diverifikasi ulang.")
	})
	if err != nil {
		return dto.DisiplinResponse{}, err
	}
	return s.Detail(ctx, actor, id)
}

func (s *disiplinService) Pemeriksaan(ctx context.Context, actor entities.User, id string, req dto.PemeriksaanRequest) (dto.DisiplinResponse, error) {
	waktu, _ := time.ParseInLocation("2006-01-02T15:04", req.WaktuPemeriksaan, wib)
	err := s.transaksi(ctx, func(tx *gorm.DB, simpan func(*entities.Dokumen)) error {
		d, err := s.ambil(ctx, tx, actor, id, entities.DisiplinStatusPemeriksaan, "hasil pemeriksaan hanya bisa dicatat saat kasus berstatus pemeriksaan")
		if err != nil {
			return err
		}
		for jenis, fh := range map[string]*multipart.FileHeader{
			entities.DisiplinBerkasBAP:              req.BAP,
			entities.DisiplinBerkasTanggapanPegawai: req.TanggapanPegawai,
			entities.DisiplinBerkasBuktiTambahan:    req.BuktiTambahan,
		} {
			if err := s.lampirkan(ctx, tx, simpan, actor, &d, jenis, fh); err != nil {
				return err
			}
		}
		d.NomorSuratPanggilan = strings.TrimSpace(req.NomorSuratPanggilan)
		d.WaktuPemeriksaan = &waktu
		if r := strings.TrimSpace(req.RingkasanPemeriksaan); r != "" {
			d.RingkasanPemeriksaan = r
		}
		if err := s.repo.Update(ctx, tx, &d); err != nil {
			return err
		}
		return aktivitas.Catat(tx, actor.ID, "disiplin.pemeriksaan", "disiplin", d.ID, d.Nomor+": hasil pemeriksaan disimpan")
	})
	if err != nil {
		return dto.DisiplinResponse{}, err
	}
	return s.Detail(ctx, actor, id)
}

func cekHukuman(tingkat, jenis string) error {
	h, ok := dto.KatalogHukuman[jenis]
	if !ok || h.Tingkat != tingkat {
		return utils.ErrBadRequest("jenis hukuman tidak sesuai dengan tingkat hukuman " + tingkat)
	}
	return nil
}

func (s *disiplinService) Rekomendasi(ctx context.Context, actor entities.User, id string, req dto.RekomendasiRequest) (dto.DisiplinResponse, error) {
	if err := cekHukuman(req.Tingkat, req.JenisHukuman); err != nil {
		return dto.DisiplinResponse{}, err
	}
	err := s.transaksi(ctx, func(tx *gorm.DB, _ func(*entities.Dokumen)) error {
		d, err := s.ambil(ctx, tx, actor, id, entities.DisiplinStatusPemeriksaan, "rekomendasi hanya bisa diajukan saat kasus berstatus pemeriksaan")
		if err != nil {
			return err
		}
		if d.RingkasanPemeriksaan == "" {
			return utils.ErrBadRequest("simpan ringkasan hasil pemeriksaan terlebih dahulu")
		}
		for jenis, label := range map[string]string{
			entities.DisiplinBerkasBAP:              "BAP bertanda tangan",
			entities.DisiplinBerkasTanggapanPegawai: "tanggapan tertulis pegawai",
		} {
			n, err := s.repo.JumlahBerkas(ctx, tx, d.ID, jenis)
			if err != nil {
				return err
			}
			if n == 0 {
				return utils.ErrBadRequest("unggah " + label + " sebelum mengajukan rekomendasi")
			}
		}
		now := time.Now()
		d.RingkasanFakta = strings.TrimSpace(req.RingkasanFakta)
		d.DugaanTerbukti = strings.TrimSpace(req.DugaanTerbukti)
		d.DasarHukum = strings.TrimSpace(req.DasarHukum)
		d.Justifikasi = strings.TrimSpace(req.Justifikasi)
		d.RekomTingkat, d.RekomJenis = req.Tingkat, req.JenisHukuman
		d.DirekomendasikanOleh, d.DirekomendasikanAt = &actor.ID, &now
		d.Tenggat = tenggat(tenggatKeputusan)
		if err := s.catatTransisi(ctx, tx, actor, &d, entities.DisiplinStatusMenungguKeputusan, "disiplin.rekomendasi", ""); err != nil {
			return err
		}
		return s.kirim(ctx, tx, s.repo.PimpinanSekolah, &d, "Kasus disiplin menunggu keputusan",
			"Rekomendasi hukuman disiplin "+d.RekomTingkat+" dari "+d.Sekolah.Nama+" menunggu keputusan Anda.")
	})
	if err != nil {
		return dto.DisiplinResponse{}, err
	}
	return s.Detail(ctx, actor, id)
}

func (s *disiplinService) Putuskan(ctx context.Context, actor entities.User, id string, req dto.KeputusanRequest) (dto.DisiplinResponse, error) {
	catatan := strings.TrimSpace(req.Catatan)
	switch req.Keputusan {
	case "ubah":
		if req.Tingkat == "" || req.JenisHukuman == "" {
			return dto.DisiplinResponse{}, utils.ErrBadRequest("tingkat dan jenis hukuman wajib diisi jika tingkat hukuman diubah")
		}
		if err := cekHukuman(req.Tingkat, req.JenisHukuman); err != nil {
			return dto.DisiplinResponse{}, err
		}
		if catatan == "" {
			return dto.DisiplinResponse{}, utils.ErrBadRequest("catatan pimpinan wajib diisi jika tingkat hukuman diubah")
		}
	case "kembalikan":
		if catatan == "" {
			return dto.DisiplinResponse{}, utils.ErrBadRequest("catatan pimpinan wajib diisi jika kasus dikembalikan untuk telaah")
		}
	}

	err := s.transaksi(ctx, func(tx *gorm.DB, _ func(*entities.Dokumen)) error {
		d, err := s.ambil(ctx, tx, actor, id, entities.DisiplinStatusMenungguKeputusan, "keputusan hanya bisa ditetapkan untuk kasus yang menunggu keputusan")
		if err != nil {
			return err
		}
		now := time.Now()
		d.KeputusanPimpinan, d.CatatanPimpinan = req.Keputusan, catatan
		d.DiputuskanOleh, d.DiputuskanAt = &actor.ID, &now

		if req.Keputusan == "kembalikan" {
			d.Tenggat = tenggat(tenggatPemeriksaan)
			if err := s.catatTransisi(ctx, tx, actor, &d, entities.DisiplinStatusPemeriksaan, "disiplin.keputusan", catatan); err != nil {
				return err
			}
			return s.kirim(ctx, tx, s.repo.StafSudinSekolah, &d, "Kasus disiplin dikembalikan pimpinan", "Telaah ulang: "+catatan)
		}

		d.Tingkat, d.JenisHukuman = d.RekomTingkat, d.RekomJenis
		if req.Keputusan == "ubah" {
			d.Tingkat, d.JenisHukuman = req.Tingkat, req.JenisHukuman
		}
		d.Tenggat = tenggat(tenggatCatat)
		if err := s.catatTransisi(ctx, tx, actor, &d, entities.DisiplinStatusSanksiDitetapkan, "disiplin.keputusan", catatan); err != nil {
			return err
		}
		label := dto.KatalogHukuman[d.JenisHukuman].Label
		if err := s.kirim(ctx, tx, s.repo.StafSudinSekolah, &d, "Sanksi disiplin ditetapkan pimpinan",
			"Hukuman disiplin "+d.Tingkat+": "+label+". Catat surat keputusan dan tindak lanjutnya."); err != nil {
			return err
		}
		return s.kirim(ctx, tx, s.repo.OperatorSekolah, &d, "Sanksi disiplin ditetapkan",
			"Pimpinan menetapkan hukuman disiplin "+d.Tingkat+". Salinan keputusan menyusul dari Sudin.")
	})
	if err != nil {
		return dto.DisiplinResponse{}, err
	}
	return s.Detail(ctx, actor, id)
}

func (s *disiplinService) CatatKeputusan(ctx context.Context, actor entities.User, id string, req dto.CatatKeputusanRequest) (dto.DisiplinResponse, error) {
	tanggal := parseTgl(req.TanggalKeputusan)
	if tanggal.After(hariIni()) {
		return dto.DisiplinResponse{}, utils.ErrBadRequest("tanggal keputusan tidak boleh di masa depan")
	}
	mulai := parseTgl(req.MasaMulai)

	err := s.transaksi(ctx, func(tx *gorm.DB, simpan func(*entities.Dokumen)) error {
		d, err := s.ambil(ctx, tx, actor, id, entities.DisiplinStatusSanksiDitetapkan, "keputusan final hanya bisa dicatat setelah sanksi ditetapkan pimpinan")
		if err != nil {
			return err
		}
		d.MasaMulai, d.MasaSelesai = nil, nil
		if bulan := dto.KatalogHukuman[d.JenisHukuman].Bulan; bulan > 0 {
			if mulai == nil {
				return utils.ErrBadRequest("masa_mulai wajib diisi untuk hukuman yang bermasa")
			}
			if mulai.Before(*tanggal) {
				return utils.ErrBadRequest("masa hukuman tidak boleh dimulai sebelum tanggal keputusan")
			}
			selesai := mulai.AddDate(0, bulan, -1)
			d.MasaMulai, d.MasaSelesai = mulai, &selesai
		}
		if err := s.lampirkan(ctx, tx, simpan, actor, &d, entities.DisiplinBerkasSalinanKeputusan, req.SalinanKeputusan); err != nil {
			return err
		}
		d.NomorKeputusan = strings.TrimSpace(req.NomorKeputusan)
		d.TanggalKeputusan = tanggal
		d.Tenggat = tenggat(tenggatPelaksanaan)
		if err := s.catatTransisi(ctx, tx, actor, &d, entities.DisiplinStatusTindakLanjut, "disiplin.catat_keputusan", d.NomorKeputusan); err != nil {
			return err
		}
		return s.kirim(ctx, tx, s.repo.OperatorSekolah, &d, "Keputusan hukuman disiplin terbit",
			"Keputusan "+d.NomorKeputusan+" terbit. Kirim laporan pelaksanaan paling lambat "+d.Tenggat.Format(dto.TglFormat)+".")
	})
	if err != nil {
		return dto.DisiplinResponse{}, err
	}
	return s.Detail(ctx, actor, id)
}

func (s *disiplinService) TindakLanjut(ctx context.Context, actor entities.User, id string, req dto.TindakLanjutRequest) (dto.DisiplinResponse, error) {
	diterima := parseTgl(req.TanggalDiterimaPegawai)
	if diterima != nil && diterima.After(hariIni()) {
		return dto.DisiplinResponse{}, utils.ErrBadRequest("tanggal diterima pegawai tidak boleh di masa depan")
	}
	err := s.transaksi(ctx, func(tx *gorm.DB, simpan func(*entities.Dokumen)) error {
		d, err := s.ambil(ctx, tx, actor, id, entities.DisiplinStatusTindakLanjut, "tindak lanjut hanya bisa dicatat saat kasus berstatus tindak lanjut")
		if err != nil {
			return err
		}
		if diterima != nil {
			if d.TanggalKeputusan != nil && diterima.Before(*d.TanggalKeputusan) {
				return utils.ErrBadRequest("tanggal diterima pegawai tidak boleh sebelum tanggal keputusan")
			}
			d.TanggalDiterimaPegawai = diterima
		}
		if err := s.lampirkan(ctx, tx, simpan, actor, &d, entities.DisiplinBerkasBuktiPenyampaian, req.BuktiPenyampaian); err != nil {
			return err
		}
		if req.DicatatIDIS != nil {
			if *req.DicatatIDIS && d.Pendidik.StatusKepegawaian != entities.StatusKepegawaianPNS {
				return utils.ErrBadRequest("pencatatan di I'DIS BKN hanya untuk PNS")
			}
			if *req.DicatatIDIS != d.DicatatIDIS {
				now := time.Now()
				d.DicatatIDIS, d.IDISOleh, d.IDISAt = *req.DicatatIDIS, &actor.ID, &now
				if !*req.DicatatIDIS {
					d.IDISOleh, d.IDISAt = nil, nil
				}
			}
		}
		if req.UpayaAdministratif != "" {
			d.UpayaAdministratif = req.UpayaAdministratif
		}
		if c := strings.TrimSpace(req.CatatanUpaya); c != "" {
			d.CatatanUpaya = c
		}
		if err := s.repo.Update(ctx, tx, &d); err != nil {
			return err
		}
		return aktivitas.Catat(tx, actor.ID, "disiplin.tindak_lanjut", "disiplin", d.ID, d.Nomor+": tindak lanjut diperbarui")
	})
	if err != nil {
		return dto.DisiplinResponse{}, err
	}
	return s.Detail(ctx, actor, id)
}

func (s *disiplinService) LaporanPelaksanaan(ctx context.Context, actor entities.User, id string, req dto.LaporanPelaksanaanRequest) (dto.DisiplinResponse, error) {
	catatan := strings.TrimSpace(req.Catatan)
	if catatan == "" {
		return dto.DisiplinResponse{}, utils.ErrBadRequest("catatan untuk Sudin wajib diisi")
	}
	err := s.transaksi(ctx, func(tx *gorm.DB, simpan func(*entities.Dokumen)) error {
		d, err := s.ambil(ctx, tx, actor, id, entities.DisiplinStatusTindakLanjut, "laporan pelaksanaan hanya bisa dikirim setelah keputusan dicatat Sudin")
		if err != nil {
			return err
		}
		if err := s.lampirkan(ctx, tx, simpan, actor, &d, entities.DisiplinBerkasLaporanPelaksanaan, req.Berkas); err != nil {
			return err
		}
		now := time.Now()
		d.CatatanPelaksanaan, d.PelaksanaanAt = catatan, &now
		d.Tenggat = nil
		if err := s.repo.Update(ctx, tx, &d); err != nil {
			return err
		}
		if err := aktivitas.Catat(tx, actor.ID, "disiplin.laporan_pelaksanaan", "disiplin", d.ID, d.Nomor+": laporan pelaksanaan dikirim"); err != nil {
			return err
		}
		return s.kirim(ctx, tx, s.repo.StafSudinSekolah, &d, "Laporan pelaksanaan hukuman disiplin",
			d.Sekolah.Nama+" mengirim laporan pelaksanaan hukuman.")
	})
	if err != nil {
		return dto.DisiplinResponse{}, err
	}
	return s.Detail(ctx, actor, id)
}

func (s *disiplinService) Tutup(ctx context.Context, actor entities.User, id string, req dto.TutupRequest) (dto.DisiplinResponse, error) {
	err := s.transaksi(ctx, func(tx *gorm.DB, _ func(*entities.Dokumen)) error {
		d, err := s.ambil(ctx, tx, actor, id, entities.DisiplinStatusTindakLanjut, "hanya kasus berstatus tindak lanjut yang bisa ditutup")
		if err != nil {
			return err
		}
		var kurang []string
		for jenis, label := range map[string]string{
			entities.DisiplinBerkasBuktiPenyampaian:   "bukti penyampaian keputusan",
			entities.DisiplinBerkasLaporanPelaksanaan: "laporan pelaksanaan dari sekolah",
		} {
			n, err := s.repo.JumlahBerkas(ctx, tx, d.ID, jenis)
			if err != nil {
				return err
			}
			if n == 0 {
				kurang = append(kurang, label)
			}
		}
		if d.Pendidik.StatusKepegawaian == entities.StatusKepegawaianPNS && !d.DicatatIDIS {
			kurang = append(kurang, "pencatatan di I'DIS BKN")
		}
		if d.UpayaAdministratif == "" {
			kurang = append(kurang, "status upaya administratif")
		}
		if d.UpayaAdministratif == "ada" && d.CatatanUpaya == "" {
			kurang = append(kurang, "catatan hasil upaya administratif")
		}
		if len(kurang) > 0 {
			return utils.ErrConflict("kasus belum bisa ditutup, lengkapi: " + strings.Join(kurang, ", "))
		}
		now := time.Now()
		d.DitutupOleh, d.DitutupAt = &actor.ID, &now
		d.Tenggat = nil
		catatan := strings.TrimSpace(req.Catatan)
		if err := s.catatTransisi(ctx, tx, actor, &d, entities.DisiplinStatusSelesai, "disiplin.tutup", catatan); err != nil {
			return err
		}
		return s.kirim(ctx, tx, s.repo.OperatorSekolah, &d, "Kasus disiplin ditutup", "Semua tindak lanjut selesai dan berkas diarsipkan.")
	})
	if err != nil {
		return dto.DisiplinResponse{}, err
	}
	return s.Detail(ctx, actor, id)
}

func (s *disiplinService) Detail(ctx context.Context, actor entities.User, id string) (dto.DisiplinResponse, error) {
	d, err := s.repo.GetDetail(ctx, actor, id)
	if err != nil {
		return dto.DisiplinResponse{}, err
	}
	var riwayat []entities.Disiplin
	if internal(actor) {
		riwayat, err = s.repo.RiwayatHukuman(ctx, actor, d.PendidikID, d.ID)
		if err != nil {
			return dto.DisiplinResponse{}, err
		}
	}
	return dto.FromEntity(d, dto.Tampilan{Internal: internal(actor), Detail: true}, riwayat), nil
}

func (s *disiplinService) Daftar(ctx context.Context, actor entities.User, f dto.Filter) ([]dto.DisiplinResponse, int64, error) {
	items, total, err := s.repo.List(ctx, actor, f)
	if err != nil {
		return nil, 0, err
	}
	t := dto.Tampilan{Internal: internal(actor), SembunyikanID: actor.Role == constants.ENUM_ROLE_PIMPINAN}
	out := make([]dto.DisiplinResponse, 0, len(items))
	for _, d := range items {
		out = append(out, dto.FromEntity(d, t, nil))
	}
	return out, total, nil
}

func (s *disiplinService) Ringkasan(ctx context.Context, actor entities.User) (dto.RingkasanResponse, error) {
	per, err := s.repo.PerStatus(ctx, actor)
	if err != nil {
		return dto.RingkasanResponse{}, err
	}
	dekat, err := s.repo.TenggatDekat(ctx, actor, *tenggat(5))
	if err != nil {
		return dto.RingkasanResponse{}, err
	}
	return dto.RingkasanResponse{
		PerStatus: per,
		PerluTindakan: per[entities.DisiplinStatusRegistrasi] + per[entities.DisiplinStatusPemeriksaan] +
			per[entities.DisiplinStatusSanksiDitetapkan],
		MenungguKeputusan: per[entities.DisiplinStatusMenungguKeputusan],
		TenggatDekat:      dekat,
	}, nil
}

func (s *disiplinService) BukaBerkas(ctx context.Context, actor entities.User, id, dokumenID string) (entities.Dokumen, io.ReadCloser, int64, error) {
	b, err := s.repo.GetBerkas(ctx, actor, id, dokumenID, internal(actor))
	if err != nil {
		return entities.Dokumen{}, nil, 0, err
	}
	if b.Dokumen == nil {
		return entities.Dokumen{}, nil, 0, gorm.ErrRecordNotFound
	}
	r, size, err := s.storage.Get(ctx, b.Dokumen.Path)
	if err != nil {
		return entities.Dokumen{}, nil, 0, err
	}
	return *b.Dokumen, r, size, nil
}
