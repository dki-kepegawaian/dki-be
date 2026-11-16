package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/arsipkan"
	"time"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	dokumenService "github.com/Caknoooo/go-gin-clean-starter/modules/dokumen/service"
	"github.com/Caknoooo/go-gin-clean-starter/modules/kinerja/dto"
	"github.com/Caknoooo/go-gin-clean-starter/modules/kinerja/repository"
	notifikasiService "github.com/Caknoooo/go-gin-clean-starter/modules/notifikasi/service"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/aktivitas"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/helpers"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/utils"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type KinerjaService interface {
	Buat(ctx context.Context, actor entities.User, req dto.BuatRequest) (dto.PenilaianResponse, error)
	UbahNilai(ctx context.Context, actor entities.User, id string, req dto.UbahNilaiRequest) (dto.PenilaianResponse, error)
	BuatDokumen(ctx context.Context, actor entities.User, id string) (dto.PenilaianResponse, error)
	Finalkan(ctx context.Context, actor entities.User, id string, req dto.FinalRequest) (dto.PenilaianResponse, error)
	Detail(ctx context.Context, actor entities.User, id string) (dto.PenilaianResponse, error)
	Daftar(ctx context.Context, actor entities.User, f dto.Filter) ([]dto.PenilaianResponse, int64, error)
	Ringkasan(ctx context.Context, actor entities.User, tahun, semester int) (map[string]int64, error)
	Pegawai(ctx context.Context, actor entities.User, sekolahID, search string, tahun, semester, offset, limit int) ([]dto.PegawaiResponse, int64, error)
}

type kinerjaService struct {
	repo       repository.KinerjaRepository
	dokumen    dokumenService.DokumenService
	notifikasi notifikasiService.NotifikasiService
	db         *gorm.DB
}

func NewKinerjaService(
	repo repository.KinerjaRepository,
	dokumen dokumenService.DokumenService,
	notifikasi notifikasiService.NotifikasiService,
	db *gorm.DB,
) KinerjaService {
	return &kinerjaService{repo: repo, dokumen: dokumen, notifikasi: notifikasi, db: db}
}

// transaksi menjalankan fn dalam satu transaksi. Berkas yang sudah tersimpan
// lewat simpan() dihapus dari storage kalau transaksi gagal.
func (s *kinerjaService) transaksi(ctx context.Context, fn func(tx *gorm.DB, simpan func(*entities.Dokumen)) error) error {
	var tersimpan []entities.Dokumen
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(tx, func(d *entities.Dokumen) { tersimpan = append(tersimpan, *d) })
	})
	if err != nil {
		s.dokumen.Hapus(tersimpan...)
	}
	return err
}

func (s *kinerjaService) catatTransisi(ctx context.Context, tx *gorm.DB, actor entities.User, k *entities.KinerjaPenilaian, ke, aksi, catatan string) error {
	dari := k.Status
	k.Status = ke
	if err := s.repo.Update(ctx, tx, k); err != nil {
		return err
	}
	if err := s.repo.AddRiwayat(ctx, tx, &entities.KinerjaRiwayat{
		PenilaianID: k.ID, DariStatus: dari, KeStatus: ke, Oleh: actor.ID, Catatan: catatan,
	}); err != nil {
		return err
	}
	return aktivitas.Catat(tx, actor.ID, aksi, "kinerja_penilaian", k.ID, fmt.Sprintf("%s: %s -> %s", k.Nomor, dari, ke))
}

func (s *kinerjaService) Buat(ctx context.Context, actor entities.User, req dto.BuatRequest) (dto.PenilaianResponse, error) {
	var id uuid.UUID
	err := s.transaksi(ctx, func(tx *gorm.DB, _ func(*entities.Dokumen)) error {
		p, err := s.repo.GetPendidikForUpdate(ctx, tx, actor, req.PendidikID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return utils.ErrBadRequest("pegawai tidak ditemukan di sekolah Anda")
			}
			return err
		}
		if p.StatusKepegawaian != entities.StatusKepegawaianPPPKPW {
			return utils.ErrBadRequest("penilaian ini hanya untuk pegawai PPPK Paruh Waktu")
		}
		ada, err := s.repo.AdaPenilaian(ctx, tx, p.ID, req.Tahun, req.Semester)
		if err != nil {
			return err
		}
		if ada {
			return utils.ErrConflict("pegawai sudah punya penilaian untuk periode ini")
		}
		nomor, err := helpers.NomorRegistrasi(tx, "P3K", time.Now().Year())
		if err != nil {
			return err
		}
		k := entities.KinerjaPenilaian{
			Nomor: nomor, PendidikID: p.ID, SekolahID: p.SekolahID, DinilaiOleh: actor.ID,
			Tahun: req.Tahun, Semester: req.Semester, Status: entities.KinerjaStatusDraf,
		}
		terapkanNilai(&k, req.Nilai)
		hitung(&k)
		if err := s.repo.Create(ctx, tx, &k); err != nil {
			return err
		}
		if err := s.repo.AddRiwayat(ctx, tx, &entities.KinerjaRiwayat{PenilaianID: k.ID, KeStatus: k.Status, Oleh: actor.ID}); err != nil {
			return err
		}
		id = k.ID
		return aktivitas.Catat(tx, actor.ID, "kinerja.buat", "kinerja_penilaian", k.ID, nomor)
	})
	if err != nil {
		return dto.PenilaianResponse{}, err
	}
	return s.Detail(ctx, actor, id.String())
}

func (s *kinerjaService) UbahNilai(ctx context.Context, actor entities.User, id string, req dto.UbahNilaiRequest) (dto.PenilaianResponse, error) {
	err := s.transaksi(ctx, func(tx *gorm.DB, _ func(*entities.Dokumen)) error {
		k, err := s.repo.GetForUpdate(ctx, tx, actor, id)
		if err != nil {
			return err
		}
		if k.Status != entities.KinerjaStatusDraf {
			return utils.ErrConflict("nilai hanya bisa diubah saat penilaian masih draf")
		}
		terapkanNilai(&k, req.Nilai)
		hitung(&k)
		if err := s.repo.Update(ctx, tx, &k); err != nil {
			return err
		}
		return aktivitas.Catat(tx, actor.ID, "kinerja.ubah_nilai", "kinerja_penilaian", k.ID, k.Nomor)
	})
	if err != nil {
		return dto.PenilaianResponse{}, err
	}
	return s.Detail(ctx, actor, id)
}

func (s *kinerjaService) BuatDokumen(ctx context.Context, actor entities.User, id string) (dto.PenilaianResponse, error) {
	err := s.transaksi(ctx, func(tx *gorm.DB, simpan func(*entities.Dokumen)) error {
		k, err := s.repo.GetForUpdate(ctx, tx, actor, id)
		if err != nil {
			return err
		}
		if k.Status != entities.KinerjaStatusDraf {
			return utils.ErrConflict("dokumen hanya bisa dibuat dari penilaian berstatus draf")
		}
		if n := hitung(&k); n < len(daftarAspek(&k)) {
			return utils.ErrBadRequest(fmt.Sprintf("%d parameter belum diisi; semua parameter wajib diisi 1–5", len(daftarAspek(&k))-n))
		}

		now := time.Now()
		data, err := buatPDF(&k, now)
		if err != nil {
			return err
		}
		doc, err := s.dokumen.SimpanBytes(ctx, tx, namaBerkas(&k), data, &k.SekolahID, actor.ID)
		if err != nil {
			return err
		}
		simpan(&doc)

		k.DokumenSistemID, k.DokumenDibuatAt = &doc.ID, &now
		if err := s.catatTransisi(ctx, tx, actor, &k, entities.KinerjaStatusMenungguTTD, "kinerja.buat_dokumen", ""); err != nil {
			return err
		}
		staf, err := s.repo.StafSudinSekolah(ctx, tx, k.SekolahID)
		if err != nil {
			return err
		}
		return s.notifikasi.Kirim(ctx, tx, staf, "Penilaian kinerja menunggu tanda tangan",
			fmt.Sprintf("%s · %s · %s · nilai akhir %s %s", k.Nomor, k.Pendidik.Nama, k.Sekolah.Nama, koma(*k.NilaiAkhir), k.Predikat),
			tautan(k.ID))
	})
	if err != nil {
		return dto.PenilaianResponse{}, err
	}
	return s.Detail(ctx, actor, id)
}

func (s *kinerjaService) Finalkan(ctx context.Context, actor entities.User, id string, req dto.FinalRequest) (dto.PenilaianResponse, error) {
	if !req.TTDJelas || !req.DataSesuai {
		return dto.PenilaianResponse{}, utils.ErrBadRequest("pastikan tanda tangan dan stempel terlihat jelas serta data sama dengan dokumen sistem")
	}
	err := s.transaksi(ctx, func(tx *gorm.DB, simpan func(*entities.Dokumen)) error {
		k, err := s.repo.GetForUpdate(ctx, tx, actor, id)
		if err != nil {
			return err
		}
		if k.Status != entities.KinerjaStatusMenungguTTD {
			return utils.ErrConflict("hanya penilaian berstatus menunggu TTD yang bisa ditetapkan Final")
		}
		doc, err := s.dokumen.Simpan(ctx, tx, req.Berkas, &k.SekolahID, actor.ID)
		if err != nil {
			return err
		}
		simpan(&doc)

		now := time.Now()
		k.DokumenTTDID = &doc.ID
		k.DifinalkanOleh, k.DifinalkanAt = &actor.ID, &now
		if err := s.catatTransisi(ctx, tx, actor, &k, entities.KinerjaStatusFinal, "kinerja.final", ""); err != nil {
			return err
		}
		if err := arsipkan.Otomatis(tx, arsipkan.Data{
			PendidikID: k.PendidikID, SekolahID: k.SekolahID, Kategori: entities.ArsipKategoriKinerja,
			JenisDokumen: "Penilaian kinerja " + dto.Periode(k.Tahun, k.Semester), Nomor: k.Nomor, TanggalDokumen: &now,
			DokumenID: doc.ID, SumberModul: "kinerja", SumberID: k.ID, Oleh: actor.ID,
		}); err != nil {
			return err
		}
		operator, err := s.repo.OperatorSekolah(ctx, tx, k.SekolahID)
		if err != nil {
			return err
		}
		return s.notifikasi.Kirim(ctx, tx, operator, "Dokumen kinerja Final",
			fmt.Sprintf("%s · Penilaian %s %s sudah Final dan bisa diunduh.", k.Nomor, k.Pendidik.Nama, dto.Periode(k.Tahun, k.Semester)),
			tautan(k.ID))
	})
	if err != nil {
		return dto.PenilaianResponse{}, err
	}
	return s.Detail(ctx, actor, id)
}

func (s *kinerjaService) Detail(ctx context.Context, actor entities.User, id string) (dto.PenilaianResponse, error) {
	k, err := s.repo.GetDetail(ctx, actor, id)
	if err != nil {
		return dto.PenilaianResponse{}, err
	}
	return dto.FromEntity(k, jumlahTerisi(&k)), nil
}

func (s *kinerjaService) Daftar(ctx context.Context, actor entities.User, f dto.Filter) ([]dto.PenilaianResponse, int64, error) {
	items, total, err := s.repo.List(ctx, actor, f)
	if err != nil {
		return nil, 0, err
	}
	out := make([]dto.PenilaianResponse, 0, len(items))
	for _, k := range items {
		out = append(out, dto.FromEntity(k, jumlahTerisi(&k)))
	}
	return out, total, nil
}

func (s *kinerjaService) Ringkasan(ctx context.Context, actor entities.User, tahun, semester int) (map[string]int64, error) {
	return s.repo.Ringkasan(ctx, actor, tahun, semester)
}

func (s *kinerjaService) Pegawai(ctx context.Context, actor entities.User, sekolahID, search string, tahun, semester, offset, limit int) ([]dto.PegawaiResponse, int64, error) {
	items, nilai, total, err := s.repo.Pegawai(ctx, actor, sekolahID, search, tahun, semester, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	out := make([]dto.PegawaiResponse, 0, len(items))
	for _, p := range items {
		var k *entities.KinerjaPenilaian
		if v, ok := nilai[p.ID]; ok {
			k = &v
		}
		out = append(out, dto.FromPegawai(p, k))
	}
	return out, total, nil
}

func tautan(id uuid.UUID) string { return "/kinerja-p3k/" + id.String() }
