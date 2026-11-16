package service

import (
	"context"
	"errors"
	"time"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"github.com/Caknoooo/go-gin-clean-starter/modules/pendidik/dto"
	"github.com/Caknoooo/go-gin-clean-starter/modules/pendidik/repository"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/akses"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type PendidikService interface {
	Daftar(ctx context.Context, actor entities.User, f repository.Filter) ([]entities.Pendidik, int64, error)
	Detail(ctx context.Context, actor entities.User, id string) (entities.Pendidik, error)
	Tambah(ctx context.Context, actor entities.User, req dto.PendidikRequest) (entities.Pendidik, error)
	Ubah(ctx context.Context, actor entities.User, id string, req dto.PendidikRequest) (entities.Pendidik, error)
}

type pendidikService struct {
	repo repository.PendidikRepository
	db   *gorm.DB
}

func NewPendidikService(repo repository.PendidikRepository, db *gorm.DB) PendidikService {
	return &pendidikService{repo: repo, db: db}
}

func (s *pendidikService) Daftar(ctx context.Context, actor entities.User, f repository.Filter) ([]entities.Pendidik, int64, error) {
	return s.repo.List(ctx, actor, f)
}

func (s *pendidikService) Detail(ctx context.Context, actor entities.User, id string) (entities.Pendidik, error) {
	return s.repo.GetByID(ctx, actor, id)
}

func (s *pendidikService) Tambah(ctx context.Context, actor entities.User, req dto.PendidikRequest) (entities.Pendidik, error) {
	var p entities.Pendidik
	if err := s.isi(ctx, actor, &p, req); err != nil {
		return p, err
	}
	if err := s.simpan(ctx, &p); err != nil {
		return p, err
	}
	return s.repo.GetByID(ctx, actor, p.ID.String())
}

func (s *pendidikService) Ubah(ctx context.Context, actor entities.User, id string, req dto.PendidikRequest) (entities.Pendidik, error) {
	p, err := s.repo.GetByID(ctx, actor, id)
	if err != nil {
		return p, err
	}
	if err := s.isi(ctx, actor, &p, req); err != nil {
		return p, err
	}
	if err := s.simpan(ctx, &p); err != nil {
		return p, err
	}
	return s.repo.GetByID(ctx, actor, id)
}

func (s *pendidikService) isi(ctx context.Context, actor entities.User, p *entities.Pendidik, req dto.PendidikRequest) error {
	sekolahID := uuid.MustParse(req.SekolahID) // sudah divalidasi binding:"uuid"
	ok, err := akses.BolehSekolah(s.db.WithContext(ctx), actor, sekolahID)
	if err != nil {
		return err
	}
	if !ok {
		return utils.ErrBadRequest("sekolah tidak ditemukan di wilayah Anda")
	}
	if req.NIP != nil && *req.NIP == "" {
		req.NIP = nil
	}

	p.SekolahID = sekolahID
	p.Sekolah = nil
	p.Nama = req.Nama
	p.NIP = req.NIP
	p.NIK = req.NIK
	p.NUPTK = req.NUPTK
	p.NRK = req.NRK
	p.NIKKI = req.NIKKI
	p.StatusKepegawaian = req.StatusKepegawaian
	p.Pangkat = req.Pangkat
	p.Golongan = req.Golongan
	p.Jabatan = req.Jabatan
	p.JabatanDapodik = req.JabatanDapodik
	p.ProgramStudi = req.ProgramStudi
	p.Sertifikasi = req.Sertifikasi
	p.Email = req.Email
	p.Telepon = req.Telepon
	p.AlamatDomisili = req.AlamatDomisili
	p.KotaDomisili = req.KotaDomisili
	p.TanggalLahir = tanggal(req.TanggalLahir)
	p.TMTCPNS = tanggal(req.TMTCPNS)
	p.TMTGolongan = tanggal(req.TMTGolongan)
	return nil
}

func (s *pendidikService) simpan(ctx context.Context, p *entities.Pendidik) error {
	err := s.repo.Save(ctx, p)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return utils.ErrConflict("NIP sudah terdaftar")
	}
	return err
}

// tanggal mengubah "2006-01-02" (sudah divalidasi binding) menjadi *time.Time; kosong = nil.
func tanggal(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil
	}
	return &t
}
