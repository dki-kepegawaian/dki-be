package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/akses"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/arsipkan"
	"strings"
	"time"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	dokumenService "github.com/Caknoooo/go-gin-clean-starter/modules/dokumen/service"
	"github.com/Caknoooo/go-gin-clean-starter/modules/kenaikanpangkat/dto"
	"github.com/Caknoooo/go-gin-clean-starter/modules/kenaikanpangkat/repository"
	notifikasiService "github.com/Caknoooo/go-gin-clean-starter/modules/notifikasi/service"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/aktivitas"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/helpers"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/persyaratan"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/utils"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const tglFormat = "2006-01-02"

var daftarBerkas = entities.KPBerkasWajib

type KenaikanPangkatService interface {
	UnggahBerkas(ctx context.Context, actor entities.User, req dto.UnggahBerkasRequest) (*persyaratan.DokumenRingkas, error)
	Perkiraan(ctx context.Context, actor entities.User, pendidikID string) (dto.PerkiraanResponse, error)
	// MemenuhiSyarat: guru PNS dalam scope yang sudah 4 tahun di golongannya pada periode berikut (K1).
	MemenuhiSyarat(ctx context.Context, actor entities.User) ([]dto.MemenuhiSyaratResponse, error)
	Ajukan(ctx context.Context, actor entities.User, req dto.AjukanRequest) (dto.KenaikanPangkatResponse, error)
	Perbaiki(ctx context.Context, actor entities.User, id string, req dto.PerbaikiRequest) (dto.KenaikanPangkatResponse, error)
	Verifikasi(ctx context.Context, actor entities.User, id string, req dto.VerifikasiRequest) (dto.KenaikanPangkatResponse, error)
	KirimBKD(ctx context.Context, actor entities.User, req dto.KirimBKDRequest) (dto.KirimBKDResponse, error)
	UnggahSK(ctx context.Context, actor entities.User, id string, req dto.SKRequest) (dto.KenaikanPangkatResponse, error)
	Detail(ctx context.Context, actor entities.User, id string) (dto.KenaikanPangkatResponse, error)
	Daftar(ctx context.Context, actor entities.User, f dto.Filter) ([]dto.KenaikanPangkatResponse, int64, error)
	Ringkasan(ctx context.Context, actor entities.User) (map[string]int64, error)
}

type kpService struct {
	repo       repository.KenaikanPangkatRepository
	dokumen    dokumenService.DokumenService
	notifikasi notifikasiService.NotifikasiService
	db         *gorm.DB
}

func NewKenaikanPangkatService(
	repo repository.KenaikanPangkatRepository,
	dokumen dokumenService.DokumenService,
	notifikasi notifikasiService.NotifikasiService,
	db *gorm.DB,
) KenaikanPangkatService {
	return &kpService{repo: repo, dokumen: dokumen, notifikasi: notifikasi, db: db}
}

// transaksi menjalankan fn dalam satu transaksi. Berkas yang sudah tersimpan
// lewat simpan() dihapus dari storage kalau transaksi gagal.
func (s *kpService) transaksi(ctx context.Context, fn func(tx *gorm.DB, simpan func(*entities.Dokumen)) error) error {
	var tersimpan []entities.Dokumen
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(tx, func(d *entities.Dokumen) { tersimpan = append(tersimpan, *d) })
	})
	if err != nil {
		s.dokumen.Hapus(tersimpan...)
	}
	return err
}

func (s *kpService) catatTransisi(ctx context.Context, tx *gorm.DB, actor entities.User, k *entities.KenaikanPangkat, ke, aksi, catatan string) error {
	dari := k.Status
	k.Status = ke
	if err := s.repo.Update(ctx, tx, k); err != nil {
		return err
	}
	if err := s.repo.AddRiwayat(ctx, tx, &entities.KenaikanPangkatRiwayat{
		KenaikanPangkatID: k.ID, DariStatus: dari, KeStatus: ke, Oleh: actor.ID, Catatan: catatan,
	}); err != nil {
		return err
	}
	return aktivitas.Catat(tx, actor.ID, aksi, "kenaikan_pangkat", k.ID, fmt.Sprintf("%s: %s -> %s", k.Nomor, dari, ke))
}

// periodeBerikut: periode KP reguler 1 April dan 1 Oktober. Usul sampai akhir Januari
// masuk periode April tahun itu, sampai akhir Juli masuk Oktober, sesudahnya April tahun depan.
func periodeBerikut(now time.Time) time.Time {
	switch {
	case now.Month() <= time.January:
		return time.Date(now.Year(), time.April, 1, 0, 0, 0, 0, time.UTC)
	case now.Month() <= time.July:
		return time.Date(now.Year(), time.October, 1, 0, 0, 0, 0, time.UTC)
	default:
		return time.Date(now.Year()+1, time.April, 1, 0, 0, 0, 0, time.UTC)
	}
}

// tahunDiGolongan adalah masa kerja minimal di golongan sekarang untuk KP reguler (K1).
const tahunDiGolongan = 4

// masaKerja menghitung selisih tahun dan bulan penuh dari tmt sampai batas.
func masaKerja(tmt, sampai time.Time) dto.MasaKerja {
	bulan := (sampai.Year()-tmt.Year())*12 + int(sampai.Month()) - int(tmt.Month())
	if sampai.Day() < tmt.Day() {
		bulan--
	}
	if bulan < 0 {
		bulan = 0
	}
	return dto.MasaKerja{Tahun: bulan / 12, Bulan: bulan % 12}
}

// usulan mengembalikan pangkat lama dan golongan/pangkat berikutnya, atau kendala.
// Syarat 4 tahun di golongan hanya dicek bila tmt_golongan terisi di data induk.
func usulan(p entities.Pendidik, periode time.Time) (pangkatLama, golBaru, pangkatBaru string, kendala []string) {
	if p.StatusKepegawaian != entities.StatusKepegawaianPNS {
		return "", "", "", []string{"kenaikan pangkat lewat BKD hanya untuk PNS"}
	}
	for i, g := range entities.PangkatPNS {
		if g.Golongan != p.Golongan {
			continue
		}
		if i == len(entities.PangkatPNS)-1 {
			return "", "", "", []string{"golongan " + p.Golongan + " sudah tertinggi"}
		}
		lama := p.Pangkat
		if lama == "" {
			lama = g.Pangkat
		}
		next := entities.PangkatPNS[i+1]
		if p.TMTGolongan != nil && p.TMTGolongan.AddDate(tahunDiGolongan, 0, 0).After(periode) {
			m := masaKerja(*p.TMTGolongan, periode)
			kendala = []string{fmt.Sprintf("masa kerja di golongan %s baru %d tahun %d bulan pada periode %s; minimal %d tahun",
				p.Golongan, m.Tahun, m.Bulan, periode.Format("2006-01"), tahunDiGolongan)}
		}
		return lama, next.Golongan, next.Pangkat, kendala
	}
	return "", "", "", []string{"golongan pendidik belum diisi atau tidak dikenal di data induk"}
}

func berkasPtr(k *entities.KenaikanPangkat) []*entities.BerkasPersyaratan {
	out := make([]*entities.BerkasPersyaratan, len(k.Berkas))
	for i := range k.Berkas {
		out[i] = &k.Berkas[i].BerkasPersyaratan
	}
	return out
}

func (s *kpService) simpanSemuaBerkas(ctx context.Context, tx *gorm.DB, k *entities.KenaikanPangkat) error {
	for i := range k.Berkas {
		if err := s.repo.SaveBerkas(ctx, tx, &k.Berkas[i]); err != nil {
			return err
		}
	}
	return nil
}

func (s *kpService) UnggahBerkas(ctx context.Context, actor entities.User, req dto.UnggahBerkasRequest) (*persyaratan.DokumenRingkas, error) {
	doc, err := persyaratan.Unggah(ctx, s.db, s.dokumen, actor, req.Berkas)
	if err != nil {
		return nil, err
	}
	return persyaratan.Dokumen(&doc), nil
}

func (s *kpService) Perkiraan(ctx context.Context, actor entities.User, pendidikID string) (dto.PerkiraanResponse, error) {
	p, err := s.repo.GetPendidik(ctx, nil, actor, pendidikID, false)
	if err != nil {
		return dto.PerkiraanResponse{}, err
	}
	nomor, err := s.repo.NomorPengajuan(ctx, nil, p.ID)
	if err != nil {
		return dto.PerkiraanResponse{}, err
	}
	periode := periodeBerikut(time.Now())
	lama, golBaru, pangkatBaru, kendala := usulan(p, periode)
	if kendala == nil {
		kendala = []string{}
	}
	res := dto.PerkiraanResponse{
		Pendidik: *dto.Pendidik(&p), Sekolah: dto.Sekolah(p.Sekolah),
		Periode:     periode.Format("2006-01"),
		PangkatLama: lama, GolonganLama: p.Golongan, PangkatBaru: pangkatBaru, GolonganBaru: golBaru,
		JumlahBerkasWajib: len(daftarBerkas), Kendala: kendala,
		TMTGolongan: dto.Tgl(p.TMTGolongan),
	}
	if p.TMTGolongan != nil {
		m := masaKerja(*p.TMTGolongan, periode)
		res.MasaKerjaGolongan = &m
	}
	if nomor != "" {
		res.PengajuanNomor = &nomor
		res.Kendala = append(res.Kendala, "pendidik masih punya pengajuan kenaikan pangkat "+nomor)
	}
	return res, nil
}

func (s *kpService) MemenuhiSyarat(ctx context.Context, actor entities.User) ([]dto.MemenuhiSyaratResponse, error) {
	periode := periodeBerikut(time.Now())
	rows, err := s.repo.CalonKP(ctx, actor, periode.AddDate(-tahunDiGolongan, 0, 0))
	if err != nil {
		return nil, err
	}
	out := []dto.MemenuhiSyaratResponse{}
	for _, p := range rows {
		_, golBaru, pangkatBaru, kendala := usulan(p, periode)
		if len(kendala) > 0 {
			continue // golongan tertinggi atau tidak dikenal
		}
		out = append(out, dto.MemenuhiSyaratResponse{
			Pendidik: *dto.Pendidik(&p), Sekolah: dto.Sekolah(p.Sekolah), Periode: periode.Format("2006-01"),
			GolonganLama: p.Golongan, GolonganBaru: golBaru, PangkatBaru: pangkatBaru,
			TMTGolongan: *dto.Tgl(p.TMTGolongan), MasaKerjaGolongan: masaKerja(*p.TMTGolongan, periode),
		})
	}
	return out, nil
}

func (s *kpService) Ajukan(ctx context.Context, actor entities.User, req dto.AjukanRequest) (dto.KenaikanPangkatResponse, error) {
	if err := persyaratan.Lengkap(req.Berkas, daftarBerkas); err != nil {
		return dto.KenaikanPangkatResponse{}, err
	}

	var id uuid.UUID
	err := s.transaksi(ctx, func(tx *gorm.DB, _ func(*entities.Dokumen)) error {
		p, err := s.repo.GetPendidik(ctx, tx, actor, req.PendidikID, true)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return utils.ErrBadRequest("pendidik tidak ditemukan di sekolah Anda")
			}
			return err
		}
		nomorLama, err := s.repo.NomorPengajuan(ctx, tx, p.ID)
		if err != nil {
			return err
		}
		if nomorLama != "" {
			return utils.ErrConflict("pendidik masih punya pengajuan kenaikan pangkat " + nomorLama)
		}
		lama, golBaru, pangkatBaru, kendala := usulan(p, periodeBerikut(time.Now()))
		if len(kendala) > 0 {
			return utils.ErrBadRequest(kendala[0])
		}

		dok, err := persyaratan.Ambil(ctx, tx, actor, p.SekolahID, req.Berkas, daftarBerkas, entities.KenaikanPangkatBerkas{}.TableName())
		if err != nil {
			return err
		}

		nomor, err := helpers.NomorRegistrasi(tx, "KP", time.Now().Year())
		if err != nil {
			return err
		}
		k := entities.KenaikanPangkat{
			Nomor: nomor, PendidikID: p.ID, SekolahID: p.SekolahID, DiajukanOleh: actor.ID,
			Periode:     periodeBerikut(time.Now()),
			PangkatLama: lama, GolonganLama: p.Golongan, PangkatBaru: pangkatBaru, GolonganBaru: golBaru,
			Status: entities.KPStatusDiajukan,
		}
		if err := s.repo.Create(ctx, tx, &k); err != nil {
			return err
		}
		for _, j := range daftarBerkas {
			b := entities.KenaikanPangkatBerkas{KenaikanPangkatID: k.ID, BerkasPersyaratan: entities.BerkasPersyaratan{Jenis: j.Kode, DokumenID: dok[j.Kode].ID}}
			if err := s.repo.SaveBerkas(ctx, tx, &b); err != nil {
				return err
			}
		}
		if err := s.repo.AddRiwayat(ctx, tx, &entities.KenaikanPangkatRiwayat{KenaikanPangkatID: k.ID, KeStatus: k.Status, Oleh: actor.ID}); err != nil {
			return err
		}
		if err := aktivitas.Catat(tx, actor.ID, "kenaikan_pangkat.ajukan", "kenaikan_pangkat", k.ID, nomor); err != nil {
			return err
		}
		staf, err := s.repo.StafSudinSekolah(ctx, tx, p.SekolahID)
		if err != nil {
			return err
		}
		id = k.ID
		return s.notifikasi.Kirim(ctx, tx, staf, "Usulan kenaikan pangkat baru",
			fmt.Sprintf("%s · %s · %s → %s · %s", nomor, p.Nama, k.GolonganLama, k.GolonganBaru, p.Sekolah.Nama), tautan(k.ID))
	})
	if err != nil {
		return dto.KenaikanPangkatResponse{}, err
	}
	return s.Detail(ctx, actor, id.String())
}

func (s *kpService) Perbaiki(ctx context.Context, actor entities.User, id string, req dto.PerbaikiRequest) (dto.KenaikanPangkatResponse, error) {
	err := s.transaksi(ctx, func(tx *gorm.DB, _ func(*entities.Dokumen)) error {
		k, err := s.repo.GetForUpdate(ctx, tx, actor, id)
		if err != nil {
			return err
		}
		if k.Status != entities.KPStatusPerluPerbaikan {
			return utils.ErrConflict("hanya pengajuan berstatus perlu perbaikan yang bisa diperbaiki")
		}
		dok, err := persyaratan.Ambil(ctx, tx, actor, k.SekolahID, req.Berkas, daftarBerkas, entities.KenaikanPangkatBerkas{}.TableName())
		if err != nil {
			return err
		}
		if err := persyaratan.Ganti(berkasPtr(&k), dok, daftarBerkas); err != nil {
			return err
		}
		if err := s.simpanSemuaBerkas(ctx, tx, &k); err != nil {
			return err
		}
		catatan := fmt.Sprintf("%d berkas diunggah ulang", len(dok))
		if err := s.catatTransisi(ctx, tx, actor, &k, entities.KPStatusDiajukan, "kenaikan_pangkat.perbaiki", catatan); err != nil {
			return err
		}
		staf, err := s.repo.StafSudinSekolah(ctx, tx, k.SekolahID)
		if err != nil {
			return err
		}
		return s.notifikasi.Kirim(ctx, tx, staf, "Berkas kenaikan pangkat diperbaiki",
			fmt.Sprintf("%s · %s mengunggah ulang %s", k.Nomor, k.Sekolah.Nama, catatan), tautan(k.ID))
	})
	if err != nil {
		return dto.KenaikanPangkatResponse{}, err
	}
	return s.Detail(ctx, actor, id)
}

func (s *kpService) Verifikasi(ctx context.Context, actor entities.User, id string, req dto.VerifikasiRequest) (dto.KenaikanPangkatResponse, error) {
	err := s.transaksi(ctx, func(tx *gorm.DB, _ func(*entities.Dokumen)) error {
		k, err := s.repo.GetForUpdate(ctx, tx, actor, id)
		if err != nil {
			return err
		}
		if k.Status != entities.KPStatusDiajukan {
			return utils.ErrConflict("hanya pengajuan berstatus diajukan yang bisa diverifikasi")
		}
		tidakValid, err := persyaratan.Nilai(berkasPtr(&k), req.Penilaian, actor.ID, daftarBerkas)
		if err != nil {
			return err
		}
		if err := s.simpanSemuaBerkas(ctx, tx, &k); err != nil {
			return err
		}
		now := time.Now()
		k.DiverifikasiOleh, k.DiverifikasiAt = &actor.ID, &now

		ke, judul, catatan := entities.KPStatusTerverifikasi, "Berkas kenaikan pangkat terverifikasi",
			fmt.Sprintf("%d berkas valid", len(k.Berkas))
		pesan := fmt.Sprintf("%s · %s · berkas valid, masuk antrean usulan ke BKD.", k.Nomor, k.Pendidik.Nama)
		if len(tidakValid) > 0 {
			ke, judul = entities.KPStatusPerluPerbaikan, "Berkas kenaikan pangkat perlu perbaikan"
			catatan = persyaratan.RingkasTidakValid(tidakValid, daftarBerkas)
			pesan = fmt.Sprintf("%s · %d berkas tidak valid: %s", k.Nomor, len(tidakValid), catatan)
		}
		if err := s.catatTransisi(ctx, tx, actor, &k, ke, "kenaikan_pangkat.verifikasi", catatan); err != nil {
			return err
		}
		operator, err := s.repo.OperatorSekolah(ctx, tx, k.SekolahID)
		if err != nil {
			return err
		}
		return s.notifikasi.Kirim(ctx, tx, operator, judul, pesan, tautan(k.ID))
	})
	if err != nil {
		return dto.KenaikanPangkatResponse{}, err
	}
	return s.Detail(ctx, actor, id)
}

// KirimBKD mencatat satu surat pengantar untuk beberapa usulan sekaligus (K4).
// Semua usulan harus terverifikasi; jika satu gagal, tidak ada yang tersimpan.
func (s *kpService) KirimBKD(ctx context.Context, actor entities.User, req dto.KirimBKDRequest) (dto.KirimBKDResponse, error) {
	tanggal, _ := time.Parse(tglFormat, req.TanggalKirim)
	if tanggal.After(time.Now()) {
		return dto.KirimBKDResponse{}, utils.ErrBadRequest("tanggal kirim tidak boleh di masa depan")
	}
	nomorSurat := strings.TrimSpace(req.NomorSuratPengantar)

	var diusulkan []string
	err := s.transaksi(ctx, func(tx *gorm.DB, simpan func(*entities.Dokumen)) error {
		// Bukti disimpan sekali per sekolah supaya hak unduhnya mengikuti scope sekolah.
		bukti := map[uuid.UUID]uuid.UUID{}
		seen := map[string]bool{}
		for _, id := range req.IDs {
			if seen[id] {
				continue
			}
			seen[id] = true
			k, err := s.repo.GetForUpdate(ctx, tx, actor, id)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return utils.ErrNotFound("usulan " + id + " tidak ditemukan")
				}
				return err
			}
			if k.Status != entities.KPStatusTerverifikasi {
				return utils.ErrConflict(k.Nomor + " belum terverifikasi atau sudah diusulkan")
			}
			if req.Bukti != nil {
				docID, ada := bukti[k.SekolahID]
				if !ada {
					doc, err := s.dokumen.Simpan(ctx, tx, req.Bukti, &k.SekolahID, actor.ID)
					if err != nil {
						return err
					}
					simpan(&doc)
					docID = doc.ID
					bukti[k.SekolahID] = docID
				}
				k.BuktiKirimID = &docID
			}
			k.NomorSuratPengantar = nomorSurat
			k.TanggalKirim = &tanggal
			k.CaraKirim = strings.TrimSpace(req.CaraKirim)
			k.CatatanKirim = strings.TrimSpace(req.Catatan)
			k.DikirimOleh = &actor.ID
			if err := s.catatTransisi(ctx, tx, actor, &k, entities.KPStatusDiusulkanKeBKD, "kenaikan_pangkat.kirim_bkd",
				"Surat pengantar "+nomorSurat); err != nil {
				return err
			}
			operator, err := s.repo.OperatorSekolah(ctx, tx, k.SekolahID)
			if err != nil {
				return err
			}
			if err := s.notifikasi.Kirim(ctx, tx, operator, "Usulan kenaikan pangkat dikirim ke BKD",
				fmt.Sprintf("%s · %s · surat pengantar %s, menunggu SK dari BKD.", k.Nomor, k.Pendidik.Nama, nomorSurat), tautan(k.ID)); err != nil {
				return err
			}
			diusulkan = append(diusulkan, k.Nomor)
		}
		return nil
	})
	if err != nil {
		return dto.KirimBKDResponse{}, err
	}
	return dto.KirimBKDResponse{Diusulkan: diusulkan}, nil
}

// UnggahSK mencatat SK dari BKD, memperbarui pangkat/golongan di data induk, dan menutup pengajuan (K5).
func (s *kpService) UnggahSK(ctx context.Context, actor entities.User, id string, req dto.SKRequest) (dto.KenaikanPangkatResponse, error) {
	tanggal, _ := time.Parse(tglFormat, req.TanggalSK)
	if tanggal.After(time.Now()) {
		return dto.KenaikanPangkatResponse{}, utils.ErrBadRequest("tanggal SK tidak boleh di masa depan")
	}
	tmt, _ := time.Parse(tglFormat, req.TMTPangkat)

	err := s.transaksi(ctx, func(tx *gorm.DB, simpan func(*entities.Dokumen)) error {
		k, err := s.repo.GetForUpdate(ctx, tx, actor, id)
		if err != nil {
			return err
		}
		if k.Status != entities.KPStatusDiusulkanKeBKD {
			return utils.ErrConflict("SK hanya bisa dicatat setelah usulan dikirim ke BKD")
		}
		p, err := s.repo.KunciPendidik(ctx, tx, k.PendidikID)
		if err != nil {
			return err
		}
		// Pendidik bisa sudah pindah sejak usulan dibuat; data induknya hanya boleh diubah
		// oleh yang berwenang atas sekolahnya sekarang.
		if boleh, err := akses.BolehSekolah(tx, actor, p.SekolahID); err != nil {
			return err
		} else if !boleh {
			return utils.ErrConflict("pendidik sudah pindah ke luar wilayah Anda; hubungi admin untuk mencatat SK ini")
		}
		if p.Golongan != k.GolonganLama && p.Golongan != k.GolonganBaru {
			return utils.ErrConflict(fmt.Sprintf("golongan di data induk (%s) sudah berubah dari usulan (%s); periksa data pendidik", p.Golongan, k.GolonganLama))
		}

		doc, err := s.dokumen.Simpan(ctx, tx, req.SK, &k.SekolahID, actor.ID)
		if err != nil {
			return err
		}
		simpan(&doc)

		now := time.Now()
		k.NomorSK = strings.TrimSpace(req.NomorSK)
		k.TanggalSK = &tanggal
		k.TMTPangkat = &tmt
		k.SKID = &doc.ID
		k.SelesaiOleh, k.SelesaiAt = &actor.ID, &now
		if err := s.catatTransisi(ctx, tx, actor, &k, entities.KPStatusSelesai, "kenaikan_pangkat.sk", "SK "+k.NomorSK); err != nil {
			return err
		}
		if err := arsipkan.Otomatis(tx, arsipkan.Data{
			PendidikID: k.PendidikID, SekolahID: k.SekolahID, Kategori: entities.ArsipKategoriSK,
			JenisDokumen: "SK Kenaikan Pangkat " + k.GolonganBaru, Nomor: k.NomorSK, TanggalDokumen: k.TanggalSK,
			Keterangan: fmt.Sprintf("%s · %s -> %s, TMT %s", k.Nomor, k.GolonganLama, k.GolonganBaru, tmt.Format(tglFormat)),
			DokumenID:  doc.ID, SumberModul: "kenaikan_pangkat", SumberID: k.ID, Oleh: actor.ID,
		}); err != nil {
			return err
		}

		if err := s.repo.UbahPangkat(ctx, tx, p.ID, k.PangkatBaru, k.GolonganBaru, tmt); err != nil {
			return err
		}
		if err := aktivitas.Catat(tx, actor.ID, "pendidik.ubah_pangkat", "pendidik", p.ID,
			fmt.Sprintf("%s · %s %s -> %s %s", k.Nomor, p.Pangkat, p.Golongan, k.PangkatBaru, k.GolonganBaru)); err != nil {
			return err
		}

		operator, err := s.repo.OperatorSekolah(ctx, tx, k.SekolahID)
		if err != nil {
			return err
		}
		return s.notifikasi.Kirim(ctx, tx, operator, "SK kenaikan pangkat terbit",
			fmt.Sprintf("%s · %s · pangkat menjadi %s · %s (TMT %s).", k.Nomor, p.Nama, k.PangkatBaru, k.GolonganBaru, tmt.Format(tglFormat)), tautan(k.ID))
	})
	if err != nil {
		return dto.KenaikanPangkatResponse{}, err
	}
	return s.Detail(ctx, actor, id)
}

func (s *kpService) Detail(ctx context.Context, actor entities.User, id string) (dto.KenaikanPangkatResponse, error) {
	k, err := s.repo.GetDetail(ctx, actor, id)
	if err != nil {
		return dto.KenaikanPangkatResponse{}, err
	}
	return dto.FromEntity(k), nil
}

func (s *kpService) Daftar(ctx context.Context, actor entities.User, f dto.Filter) ([]dto.KenaikanPangkatResponse, int64, error) {
	items, total, err := s.repo.List(ctx, actor, f)
	if err != nil {
		return nil, 0, err
	}
	out := make([]dto.KenaikanPangkatResponse, 0, len(items))
	for _, k := range items {
		r := dto.FromEntity(k)
		r.Berkas = nil // daftar cukup jumlah_berkas & berkas_tidak_valid
		out = append(out, r)
	}
	return out, total, nil
}

func (s *kpService) Ringkasan(ctx context.Context, actor entities.User) (map[string]int64, error) {
	return s.repo.Ringkasan(ctx, actor)
}

func tautan(id uuid.UUID) string { return "/kenaikan-pangkat/" + id.String() }
