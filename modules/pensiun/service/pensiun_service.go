package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/arsipkan"
	"strings"
	"time"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	dokumenService "github.com/Caknoooo/go-gin-clean-starter/modules/dokumen/service"
	notifikasiService "github.com/Caknoooo/go-gin-clean-starter/modules/notifikasi/service"
	"github.com/Caknoooo/go-gin-clean-starter/modules/pensiun/dto"
	"github.com/Caknoooo/go-gin-clean-starter/modules/pensiun/repository"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/aktivitas"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/helpers"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/persyaratan"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/utils"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const tglFormat = "2006-01-02"

var daftarBerkas = entities.PensiunBerkasWajib

type PensiunService interface {
	UnggahBerkas(ctx context.Context, actor entities.User, req dto.UnggahBerkasRequest) (*persyaratan.DokumenRingkas, error)
	Perkiraan(ctx context.Context, actor entities.User, pendidikID string) (dto.PerkiraanResponse, error)
	Ajukan(ctx context.Context, actor entities.User, req dto.AjukanRequest) (dto.PensiunResponse, error)
	Perbaiki(ctx context.Context, actor entities.User, id string, req dto.PerbaikiRequest) (dto.PensiunResponse, error)
	Verifikasi(ctx context.Context, actor entities.User, id string, req dto.VerifikasiRequest) (dto.PensiunResponse, error)
	KirimBKD(ctx context.Context, actor entities.User, id string, req dto.KirimBKDRequest) (dto.PensiunResponse, error)
	UnggahSK(ctx context.Context, actor entities.User, id string, req dto.SKRequest) (dto.PensiunResponse, error)
	Detail(ctx context.Context, actor entities.User, id string) (dto.PensiunResponse, error)
	Daftar(ctx context.Context, actor entities.User, f dto.Filter) ([]dto.PensiunResponse, int64, error)
	Ringkasan(ctx context.Context, actor entities.User) (map[string]int64, error)
}

type pensiunService struct {
	repo       repository.PensiunRepository
	dokumen    dokumenService.DokumenService
	notifikasi notifikasiService.NotifikasiService
	db         *gorm.DB
}

func NewPensiunService(
	repo repository.PensiunRepository,
	dokumen dokumenService.DokumenService,
	notifikasi notifikasiService.NotifikasiService,
	db *gorm.DB,
) PensiunService {
	return &pensiunService{repo: repo, dokumen: dokumen, notifikasi: notifikasi, db: db}
}

// transaksi menjalankan fn dalam satu transaksi. Berkas yang sudah tersimpan
// lewat simpan() dihapus dari storage kalau transaksi gagal.
func (s *pensiunService) transaksi(ctx context.Context, fn func(tx *gorm.DB, simpan func(*entities.Dokumen)) error) error {
	var tersimpan []entities.Dokumen
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(tx, func(d *entities.Dokumen) { tersimpan = append(tersimpan, *d) })
	})
	if err != nil {
		s.dokumen.Hapus(tersimpan...)
	}
	return err
}

func (s *pensiunService) catatTransisi(ctx context.Context, tx *gorm.DB, actor entities.User, p *entities.Pensiun, ke, aksi, catatan string) error {
	dari := p.Status
	p.Status = ke
	if err := s.repo.Update(ctx, tx, p); err != nil {
		return err
	}
	if err := s.repo.AddRiwayat(ctx, tx, &entities.PensiunRiwayat{
		PensiunID: p.ID, DariStatus: dari, KeStatus: ke, Oleh: actor.ID, Catatan: catatan,
	}); err != nil {
		return err
	}
	return aktivitas.Catat(tx, actor.ID, aksi, "pensiun", p.ID, fmt.Sprintf("%s: %s -> %s", p.Nomor, dari, ke))
}

// tmtBatasUsia: tanggal 1 bulan berikutnya setelah pendidik mencapai batas usia pensiun.
func tmtBatasUsia(lahir time.Time) time.Time {
	ultah := lahir.AddDate(entities.PensiunBatasUsia, 0, 0)
	return time.Date(ultah.Year(), ultah.Month()+1, 1, 0, 0, 0, 0, time.UTC)
}

func berkasPtr(p *entities.Pensiun) []*entities.BerkasPersyaratan {
	out := make([]*entities.BerkasPersyaratan, len(p.Berkas))
	for i := range p.Berkas {
		out[i] = &p.Berkas[i].BerkasPersyaratan
	}
	return out
}

func (s *pensiunService) simpanSemuaBerkas(ctx context.Context, tx *gorm.DB, p *entities.Pensiun) error {
	for i := range p.Berkas {
		if err := s.repo.SaveBerkas(ctx, tx, &p.Berkas[i]); err != nil {
			return err
		}
	}
	return nil
}

func (s *pensiunService) UnggahBerkas(ctx context.Context, actor entities.User, req dto.UnggahBerkasRequest) (*persyaratan.DokumenRingkas, error) {
	doc, err := persyaratan.Unggah(ctx, s.db, s.dokumen, actor, req.Berkas)
	if err != nil {
		return nil, err
	}
	return persyaratan.Dokumen(&doc), nil
}

// kendala mengembalikan alasan pendidik belum bisa diajukan pensiun.
func kendala(p entities.Pendidik, nomor string) []string {
	out := []string{}
	if p.StatusKepegawaian != entities.StatusKepegawaianPNS {
		out = append(out, "pengajuan pensiun lewat BKD hanya untuk PNS")
	}
	if nomor != "" {
		out = append(out, "pendidik sudah memiliki pengajuan pensiun "+nomor)
	}
	return out
}

func (s *pensiunService) Perkiraan(ctx context.Context, actor entities.User, pendidikID string) (dto.PerkiraanResponse, error) {
	p, err := s.repo.GetPendidik(ctx, nil, actor, pendidikID, false)
	if err != nil {
		return dto.PerkiraanResponse{}, err
	}
	nomor, err := s.repo.NomorPengajuan(ctx, nil, p.ID)
	if err != nil {
		return dto.PerkiraanResponse{}, err
	}
	res := dto.PerkiraanResponse{
		Pendidik: *dto.Pendidik(&p), Sekolah: dto.Sekolah(p.Sekolah), BatasUsia: entities.PensiunBatasUsia,
		JumlahBerkasWajib: len(daftarBerkas), Kendala: kendala(p, nomor),
	}
	if nomor != "" {
		res.PengajuanNomor = &nomor
	}
	if p.TanggalLahir != nil {
		tmt := tmtBatasUsia(*p.TanggalLahir)
		res.TMTPensiunBUP = dto.Tgl(&tmt)
		res.MasaKerja = dto.HitungMasaKerja(&p, tmt)
	} else {
		res.Kendala = append(res.Kendala, "tanggal lahir belum diisi di data induk (wajib untuk pensiun batas usia)")
	}
	return res, nil
}

func (s *pensiunService) Ajukan(ctx context.Context, actor entities.User, req dto.AjukanRequest) (dto.PensiunResponse, error) {
	if !req.PernyataanDiperiksa {
		return dto.PensiunResponse{}, utils.ErrBadRequest("centang pernyataan bahwa isi setiap berkas sudah diperiksa")
	}
	if err := persyaratan.Lengkap(req.Berkas, daftarBerkas); err != nil {
		return dto.PensiunResponse{}, err
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
			return utils.ErrConflict("pendidik sudah memiliki pengajuan pensiun " + nomorLama)
		}
		if p.StatusKepegawaian != entities.StatusKepegawaianPNS {
			return utils.ErrBadRequest("pengajuan pensiun lewat BKD hanya untuk PNS")
		}

		var tmt time.Time
		if req.Jenis == entities.PensiunJenisBatasUsia {
			if p.TanggalLahir == nil {
				return utils.ErrBadRequest("tanggal lahir pendidik belum diisi di data induk; minta Sudin melengkapinya")
			}
			tmt = tmtBatasUsia(*p.TanggalLahir)
		} else {
			if req.TMTPensiun == "" {
				return utils.ErrBadRequest("TMT pensiun wajib diisi untuk pensiun atas permintaan sendiri")
			}
			tmt, _ = time.Parse(tglFormat, req.TMTPensiun) // format sudah divalidasi binding
			if tmt.Day() != 1 {
				return utils.ErrBadRequest("TMT pensiun harus tanggal 1")
			}
			if !tmt.After(time.Now()) {
				return utils.ErrBadRequest("TMT pensiun harus di masa depan")
			}
		}

		dok, err := persyaratan.Ambil(ctx, tx, actor, p.SekolahID, req.Berkas, daftarBerkas, entities.PensiunBerkas{}.TableName())
		if err != nil {
			return err
		}

		nomor, err := helpers.NomorRegistrasi(tx, "PEN", time.Now().Year())
		if err != nil {
			return err
		}
		pen := entities.Pensiun{
			Nomor: nomor, PendidikID: p.ID, SekolahID: p.SekolahID, DiajukanOleh: actor.ID,
			Jenis: req.Jenis, TMTPensiun: tmt, Status: entities.PensiunStatusDiajukan,
		}
		if err := s.repo.Create(ctx, tx, &pen); err != nil {
			return err
		}
		for _, j := range daftarBerkas {
			b := entities.PensiunBerkas{PensiunID: pen.ID, BerkasPersyaratan: entities.BerkasPersyaratan{Jenis: j.Kode, DokumenID: dok[j.Kode].ID}}
			if err := s.repo.SaveBerkas(ctx, tx, &b); err != nil {
				return err
			}
		}
		if err := s.repo.AddRiwayat(ctx, tx, &entities.PensiunRiwayat{PensiunID: pen.ID, KeStatus: pen.Status, Oleh: actor.ID}); err != nil {
			return err
		}
		if err := aktivitas.Catat(tx, actor.ID, "pensiun.ajukan", "pensiun", pen.ID, nomor); err != nil {
			return err
		}
		staf, err := s.repo.StafSudinSekolah(ctx, tx, p.SekolahID)
		if err != nil {
			return err
		}
		id = pen.ID
		return s.notifikasi.Kirim(ctx, tx, staf, "Pengajuan pensiun baru",
			fmt.Sprintf("%s · %s · TMT %s · %s", nomor, p.Nama, tmt.Format(tglFormat), p.Sekolah.Nama), tautan(pen.ID))
	})
	if err != nil {
		return dto.PensiunResponse{}, err
	}
	return s.Detail(ctx, actor, id.String())
}

func (s *pensiunService) Perbaiki(ctx context.Context, actor entities.User, id string, req dto.PerbaikiRequest) (dto.PensiunResponse, error) {
	err := s.transaksi(ctx, func(tx *gorm.DB, _ func(*entities.Dokumen)) error {
		p, err := s.repo.GetForUpdate(ctx, tx, actor, id)
		if err != nil {
			return err
		}
		if p.Status != entities.PensiunStatusPerluPerbaikan {
			return utils.ErrConflict("hanya pengajuan berstatus perlu perbaikan yang bisa diperbaiki")
		}
		dok, err := persyaratan.Ambil(ctx, tx, actor, p.SekolahID, req.Berkas, daftarBerkas, entities.PensiunBerkas{}.TableName())
		if err != nil {
			return err
		}
		if err := persyaratan.Ganti(berkasPtr(&p), dok, daftarBerkas); err != nil {
			return err
		}
		if err := s.simpanSemuaBerkas(ctx, tx, &p); err != nil {
			return err
		}
		catatan := fmt.Sprintf("%d berkas diunggah ulang", len(dok))
		if err := s.catatTransisi(ctx, tx, actor, &p, entities.PensiunStatusDiajukan, "pensiun.perbaiki", catatan); err != nil {
			return err
		}
		staf, err := s.repo.StafSudinSekolah(ctx, tx, p.SekolahID)
		if err != nil {
			return err
		}
		return s.notifikasi.Kirim(ctx, tx, staf, "Berkas pensiun diperbaiki",
			fmt.Sprintf("%s · %s mengunggah ulang %s", p.Nomor, p.Sekolah.Nama, catatan), tautan(p.ID))
	})
	if err != nil {
		return dto.PensiunResponse{}, err
	}
	return s.Detail(ctx, actor, id)
}

func (s *pensiunService) Verifikasi(ctx context.Context, actor entities.User, id string, req dto.VerifikasiRequest) (dto.PensiunResponse, error) {
	err := s.transaksi(ctx, func(tx *gorm.DB, _ func(*entities.Dokumen)) error {
		p, err := s.repo.GetForUpdate(ctx, tx, actor, id)
		if err != nil {
			return err
		}
		if p.Status != entities.PensiunStatusDiajukan {
			return utils.ErrConflict("hanya pengajuan berstatus diajukan yang bisa diverifikasi")
		}
		tidakValid, err := persyaratan.Nilai(berkasPtr(&p), req.Penilaian, actor.ID, daftarBerkas)
		if err != nil {
			return err
		}
		if err := s.simpanSemuaBerkas(ctx, tx, &p); err != nil {
			return err
		}
		now := time.Now()
		p.DiverifikasiOleh, p.DiverifikasiAt = &actor.ID, &now

		ke, judul, catatan := entities.PensiunStatusTerverifikasi, "Berkas pensiun terverifikasi",
			fmt.Sprintf("%d berkas valid", len(p.Berkas))
		pesan := fmt.Sprintf("%s · %s · berkas lengkap dan valid, menunggu usulan ke BKD.", p.Nomor, p.Pendidik.Nama)
		if len(tidakValid) > 0 {
			ke, judul = entities.PensiunStatusPerluPerbaikan, "Berkas pensiun perlu perbaikan"
			catatan = persyaratan.RingkasTidakValid(tidakValid, daftarBerkas)
			pesan = fmt.Sprintf("%s · %d berkas tidak valid: %s", p.Nomor, len(tidakValid), catatan)
		}
		if err := s.catatTransisi(ctx, tx, actor, &p, ke, "pensiun.verifikasi", catatan); err != nil {
			return err
		}
		operator, err := s.repo.OperatorSekolah(ctx, tx, p.SekolahID)
		if err != nil {
			return err
		}
		return s.notifikasi.Kirim(ctx, tx, operator, judul, pesan, tautan(p.ID))
	})
	if err != nil {
		return dto.PensiunResponse{}, err
	}
	return s.Detail(ctx, actor, id)
}

func (s *pensiunService) KirimBKD(ctx context.Context, actor entities.User, id string, req dto.KirimBKDRequest) (dto.PensiunResponse, error) {
	tanggal, _ := time.Parse(tglFormat, req.TanggalKirim)
	if tanggal.After(time.Now()) {
		return dto.PensiunResponse{}, utils.ErrBadRequest("tanggal kirim tidak boleh di masa depan")
	}
	err := s.transaksi(ctx, func(tx *gorm.DB, simpan func(*entities.Dokumen)) error {
		p, err := s.repo.GetForUpdate(ctx, tx, actor, id)
		if err != nil {
			return err
		}
		if p.Status != entities.PensiunStatusTerverifikasi {
			return utils.ErrConflict("hanya pengajuan terverifikasi yang bisa diusulkan ke BKD")
		}
		if req.Bukti != nil {
			doc, err := s.dokumen.Simpan(ctx, tx, req.Bukti, &p.SekolahID, actor.ID)
			if err != nil {
				return err
			}
			simpan(&doc)
			p.BuktiKirimID = &doc.ID
		}
		p.NomorSuratPengantar = strings.TrimSpace(req.NomorSuratPengantar)
		p.TanggalKirim = &tanggal
		p.CaraKirim = strings.TrimSpace(req.CaraKirim)
		p.CatatanKirim = strings.TrimSpace(req.Catatan)
		p.DikirimOleh = &actor.ID
		if err := s.catatTransisi(ctx, tx, actor, &p, entities.PensiunStatusDiusulkanKeBKD, "pensiun.kirim_bkd",
			"Surat pengantar "+p.NomorSuratPengantar); err != nil {
			return err
		}
		operator, err := s.repo.OperatorSekolah(ctx, tx, p.SekolahID)
		if err != nil {
			return err
		}
		return s.notifikasi.Kirim(ctx, tx, operator, "Usulan pensiun dikirim ke BKD",
			fmt.Sprintf("%s · %s · surat pengantar %s, menunggu SK dari BKD.", p.Nomor, p.Pendidik.Nama, p.NomorSuratPengantar), tautan(p.ID))
	})
	if err != nil {
		return dto.PensiunResponse{}, err
	}
	return s.Detail(ctx, actor, id)
}

func (s *pensiunService) UnggahSK(ctx context.Context, actor entities.User, id string, req dto.SKRequest) (dto.PensiunResponse, error) {
	tanggal, _ := time.Parse(tglFormat, req.TanggalSK)
	if tanggal.After(time.Now()) {
		return dto.PensiunResponse{}, utils.ErrBadRequest("tanggal SK tidak boleh di masa depan")
	}
	err := s.transaksi(ctx, func(tx *gorm.DB, simpan func(*entities.Dokumen)) error {
		p, err := s.repo.GetForUpdate(ctx, tx, actor, id)
		if err != nil {
			return err
		}
		if p.Status != entities.PensiunStatusDiusulkanKeBKD {
			return utils.ErrConflict("SK hanya bisa dicatat setelah usulan dikirim ke BKD")
		}
		doc, err := s.dokumen.Simpan(ctx, tx, req.SK, &p.SekolahID, actor.ID)
		if err != nil {
			return err
		}
		simpan(&doc)

		now := time.Now()
		p.NomorSK = strings.TrimSpace(req.NomorSK)
		p.TanggalSK = &tanggal
		p.SKID = &doc.ID
		if req.TMTPensiun != "" {
			p.TMTPensiun, _ = time.Parse(tglFormat, req.TMTPensiun)
		}
		p.SelesaiOleh, p.SelesaiAt = &actor.ID, &now
		if err := s.catatTransisi(ctx, tx, actor, &p, entities.PensiunStatusSelesai, "pensiun.sk", "SK "+p.NomorSK); err != nil {
			return err
		}
		if err := arsipkan.Otomatis(tx, arsipkan.Data{
			PendidikID: p.PendidikID, SekolahID: p.SekolahID, Kategori: entities.ArsipKategoriSK,
			JenisDokumen: "SK Pensiun", Nomor: p.NomorSK, TanggalDokumen: p.TanggalSK,
			Keterangan: p.Nomor + " · TMT " + p.TMTPensiun.Format(tglFormat),
			DokumenID:  doc.ID, SumberModul: "pensiun", SumberID: p.ID, Oleh: actor.ID,
		}); err != nil {
			return err
		}
		operator, err := s.repo.OperatorSekolah(ctx, tx, p.SekolahID)
		if err != nil {
			return err
		}
		return s.notifikasi.Kirim(ctx, tx, operator, "SK pensiun terbit",
			fmt.Sprintf("%s · %s · SK %s · TMT %s. SK dapat diunduh.", p.Nomor, p.Pendidik.Nama, p.NomorSK, p.TMTPensiun.Format(tglFormat)), tautan(p.ID))
	})
	if err != nil {
		return dto.PensiunResponse{}, err
	}
	return s.Detail(ctx, actor, id)
}

func (s *pensiunService) Detail(ctx context.Context, actor entities.User, id string) (dto.PensiunResponse, error) {
	p, err := s.repo.GetDetail(ctx, actor, id)
	if err != nil {
		return dto.PensiunResponse{}, err
	}
	return dto.FromEntity(p), nil
}

func (s *pensiunService) Daftar(ctx context.Context, actor entities.User, f dto.Filter) ([]dto.PensiunResponse, int64, error) {
	items, total, err := s.repo.List(ctx, actor, f)
	if err != nil {
		return nil, 0, err
	}
	out := make([]dto.PensiunResponse, 0, len(items))
	for _, p := range items {
		r := dto.FromEntity(p)
		r.Berkas = nil // daftar cukup jumlah_berkas & berkas_tidak_valid
		out = append(out, r)
	}
	return out, total, nil
}

func (s *pensiunService) Ringkasan(ctx context.Context, actor entities.User) (map[string]int64, error) {
	return s.repo.Ringkasan(ctx, actor)
}

func tautan(id uuid.UUID) string { return "/pensiun/" + id.String() }
