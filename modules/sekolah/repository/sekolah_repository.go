package repository

import (
	"context"
	"time"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/akses"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/aktivitas"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/constants"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Filter struct {
	Search  string
	Jenjang string
	Offset  int
	Limit   int
}

type SekolahRepository interface {
	List(ctx context.Context, actor entities.User, f Filter) ([]entities.Sekolah, int64, error)
	GetByID(ctx context.Context, actor entities.User, id string) (entities.Sekolah, error)
	// Profil: data sekolah, kontak Sudin, jumlah guru per status, operator, dan petugas Sudin (H5).
	Profil(ctx context.Context, actor entities.User, id string) (Profil, error)
	Ubah(ctx context.Context, actor entities.User, id string, d UbahData) (entities.Sekolah, error)
	// UbahKontakSudin: staf hanya Sudin-nya sendiri; di luar itu ErrRecordNotFound.
	UbahKontakSudin(ctx context.Context, actor entities.User, id string, d KontakData) (entities.Sudin, error)
}

type Profil struct {
	entities.Sekolah
	JumlahGuru    int64            `json:"jumlah_guru"`
	GuruPerStatus map[string]int64 `json:"guru_per_status"`
	Operator      []string         `json:"operator"`
	PetugasSudin  []string         `json:"petugas_sudin"`
}

type UbahData struct {
	KepalaSekolah string `json:"kepala_sekolah" binding:"max=150"`
	Kecamatan     string `json:"kecamatan" binding:"max=100"`
	Alamat        string `json:"alamat" binding:"max=500"`
}

type KontakData struct {
	KontakUnit    string `json:"kontak_unit" binding:"max=150"`
	KontakTelepon string `json:"kontak_telepon" binding:"max=30"`
	KontakEmail   string `json:"kontak_email" binding:"omitempty,email,max=255"`
	JamLayanan    string `json:"jam_layanan" binding:"max=100"`
}

type sekolahRepository struct {
	db *gorm.DB
}

func NewSekolahRepository(db *gorm.DB) SekolahRepository {
	return &sekolahRepository{db: db}
}

func (r *sekolahRepository) List(ctx context.Context, actor entities.User, f Filter) ([]entities.Sekolah, int64, error) {
	q := r.db.WithContext(ctx).Model(&entities.Sekolah{}).Scopes(akses.Scope(actor, "sekolah.id"))
	if f.Search != "" {
		q = q.Where("(nama ILIKE ? OR npsn = ?)", "%"+f.Search+"%", f.Search)
	}
	if f.Jenjang != "" {
		q = q.Where("jenjang = ?", f.Jenjang)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []entities.Sekolah
	err := q.Preload("Sudin").Order("nama ASC").Offset(f.Offset).Limit(f.Limit).Find(&items).Error
	return items, total, err
}

func (r *sekolahRepository) GetByID(ctx context.Context, actor entities.User, id string) (entities.Sekolah, error) {
	var s entities.Sekolah
	if _, err := uuid.Parse(id); err != nil {
		return s, gorm.ErrRecordNotFound
	}
	err := r.db.WithContext(ctx).Scopes(akses.Scope(actor, "sekolah.id")).Preload("Sudin").Take(&s, "sekolah.id = ?", id).Error
	return s, err
}

func (r *sekolahRepository) Profil(ctx context.Context, actor entities.User, id string) (Profil, error) {
	s, err := r.GetByID(ctx, actor, id)
	if err != nil {
		return Profil{}, err
	}
	p := Profil{Sekolah: s, GuruPerStatus: map[string]int64{}, Operator: []string{}, PetugasSudin: []string{}}
	db := r.db.WithContext(ctx)
	var rows []struct {
		Status string
		Jumlah int64
	}
	if err := db.Model(&entities.Pendidik{}).Select("status_kepegawaian AS status, COUNT(*) AS jumlah").
		Where("sekolah_id = ?", s.ID).Group("status_kepegawaian").Scan(&rows).Error; err != nil {
		return Profil{}, err
	}
	for _, x := range rows {
		p.GuruPerStatus[x.Status] = x.Jumlah
		p.JumlahGuru += x.Jumlah
	}
	akun := db.Model(&entities.User{}).Where("is_aktif = ?", true).Order("name ASC").Limit(20)
	if err := akun.Session(&gorm.Session{}).Where("role = ? AND sekolah_id = ?", constants.ENUM_ROLE_OPERATOR_SEKOLAH, s.ID).
		Pluck("name", &p.Operator).Error; err != nil {
		return Profil{}, err
	}
	err = akun.Session(&gorm.Session{}).Where("role = ? AND sudin_id = ?", constants.ENUM_ROLE_STAF_SUDIN, s.SudinID).
		Pluck("name", &p.PetugasSudin).Error
	return p, err
}

func (r *sekolahRepository) Ubah(ctx context.Context, actor entities.User, id string, d UbahData) (entities.Sekolah, error) {
	s, err := r.GetByID(ctx, actor, id)
	if err != nil {
		return s, err
	}
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&entities.Sekolah{}).Where("id = ?", s.ID).Updates(map[string]any{
			"kepala_sekolah": d.KepalaSekolah, "kecamatan": d.Kecamatan, "alamat": d.Alamat, "updated_at": time.Now(),
		}).Error; err != nil {
			return err
		}
		return aktivitas.Catat(tx, actor.ID, "sekolah.ubah", "sekolah", s.ID, s.Nama)
	})
	if err != nil {
		return s, err
	}
	return r.GetByID(ctx, actor, id)
}

func (r *sekolahRepository) UbahKontakSudin(ctx context.Context, actor entities.User, id string, d KontakData) (entities.Sudin, error) {
	var s entities.Sudin
	sid, err := uuid.Parse(id)
	if err != nil || (actor.Role != constants.ENUM_ROLE_ADMIN && (actor.SudinID == nil || *actor.SudinID != sid)) {
		return s, gorm.ErrRecordNotFound
	}
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Take(&s, "id = ?", sid).Error; err != nil {
			return err
		}
		s.KontakUnit, s.KontakTelepon, s.KontakEmail, s.JamLayanan = d.KontakUnit, d.KontakTelepon, d.KontakEmail, d.JamLayanan
		if err := tx.Save(&s).Error; err != nil {
			return err
		}
		return aktivitas.Catat(tx, actor.ID, "sudin.ubah_kontak", "sudin", s.ID, s.Nama)
	})
	return s, err
}
