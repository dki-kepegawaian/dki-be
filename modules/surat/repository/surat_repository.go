package repository

import (
	"context"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"github.com/Caknoooo/go-gin-clean-starter/modules/surat/dto"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/akses"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/penerima"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SuratRepository interface {
	Create(ctx context.Context, tx *gorm.DB, s *entities.Surat) error
	Update(ctx context.Context, tx *gorm.DB, s *entities.Surat) error
	AddRiwayat(ctx context.Context, tx *gorm.DB, r *entities.SuratRiwayat) error
	// GetForUpdate mengunci baris surat (SELECT ... FOR UPDATE) dalam scope actor.
	GetForUpdate(ctx context.Context, tx *gorm.DB, actor entities.User, id string) (entities.Surat, error)
	GetDetail(ctx context.Context, actor entities.User, id string) (entities.Surat, error)
	List(ctx context.Context, actor entities.User, f dto.Filter) ([]entities.Surat, int64, error)
	// GetPendidik mengambil pendidik dalam scope actor beserta sekolah dan Sudin-nya.
	GetPendidik(ctx context.Context, tx *gorm.DB, actor entities.User, id string) (entities.Pendidik, error)
	// NomorDipakai true bila nomor sudah dipakai surat lain (kecuali kecuali).
	NomorDipakai(ctx context.Context, tx *gorm.DB, nomor string, kecuali uuid.UUID) (bool, error)
	StafSudinSekolah(ctx context.Context, tx *gorm.DB, sekolahID uuid.UUID) ([]uuid.UUID, error)
	OperatorSekolah(ctx context.Context, tx *gorm.DB, sekolahID uuid.UUID) ([]uuid.UUID, error)
}

type suratRepository struct {
	db *gorm.DB
}

func NewSuratRepository(db *gorm.DB) SuratRepository {
	return &suratRepository{db: db}
}

func (r *suratRepository) tx(tx *gorm.DB) *gorm.DB {
	if tx == nil {
		return r.db
	}
	return tx
}

func (r *suratRepository) Create(ctx context.Context, tx *gorm.DB, s *entities.Surat) error {
	return r.tx(tx).WithContext(ctx).Omit(clause.Associations).Create(s).Error
}

func (r *suratRepository) Update(ctx context.Context, tx *gorm.DB, s *entities.Surat) error {
	return r.tx(tx).WithContext(ctx).Omit(clause.Associations).Save(s).Error
}

func (r *suratRepository) AddRiwayat(ctx context.Context, tx *gorm.DB, h *entities.SuratRiwayat) error {
	return r.tx(tx).WithContext(ctx).Omit(clause.Associations).Create(h).Error
}

func (r *suratRepository) GetForUpdate(ctx context.Context, tx *gorm.DB, actor entities.User, id string) (entities.Surat, error) {
	var s entities.Surat
	if _, err := uuid.Parse(id); err != nil {
		return s, gorm.ErrRecordNotFound
	}
	err := r.tx(tx).WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE", Table: clause.Table{Name: "surat"}}).
		Scopes(akses.Scope(actor, "surat.sekolah_id")).
		Preload("Pendidik").Preload("Sekolah.Sudin").
		Take(&s, "surat.id = ?", id).Error
	return s, err
}

func (r *suratRepository) GetDetail(ctx context.Context, actor entities.User, id string) (entities.Surat, error) {
	var s entities.Surat
	if _, err := uuid.Parse(id); err != nil {
		return s, gorm.ErrRecordNotFound
	}
	err := r.db.WithContext(ctx).
		Scopes(akses.Scope(actor, "surat.sekolah_id")).
		Preload("Pendidik").Preload("Sekolah.Sudin").Preload("Dokumen").Preload("DokumenTTD").
		Preload("Riwayat", func(db *gorm.DB) *gorm.DB { return db.Order("created_at ASC") }).
		Preload("Riwayat.Pengguna").
		Take(&s, "surat.id = ?", id).Error
	return s, err
}

func (r *suratRepository) List(ctx context.Context, actor entities.User, f dto.Filter) ([]entities.Surat, int64, error) {
	q := r.db.WithContext(ctx).Model(&entities.Surat{}).Scopes(akses.Scope(actor, "surat.sekolah_id"))
	if len(f.Status) > 0 {
		q = q.Where("surat.status IN ?", f.Status)
	}
	if f.Jenis != "" {
		q = q.Where("surat.jenis = ?", f.Jenis)
	}
	if f.PendidikID != "" {
		if _, err := uuid.Parse(f.PendidikID); err != nil {
			return nil, 0, nil
		}
		q = q.Where("surat.pendidik_id = ?", f.PendidikID)
	}
	if f.Search != "" {
		q = q.Joins("JOIN pendidik ON pendidik.id = surat.pendidik_id").
			Where("(surat.nomor ILIKE ? OR surat.perihal ILIKE ? OR pendidik.nama ILIKE ? OR pendidik.nip LIKE ?)",
				"%"+f.Search+"%", "%"+f.Search+"%", "%"+f.Search+"%", f.Search+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []entities.Surat
	err := q.Preload("Pendidik").Preload("Sekolah.Sudin").Preload("Dokumen").Preload("DokumenTTD").
		Order("surat.created_at DESC").Offset(f.Offset).Limit(f.Limit).Find(&items).Error
	return items, total, err
}

func (r *suratRepository) GetPendidik(ctx context.Context, tx *gorm.DB, actor entities.User, id string) (entities.Pendidik, error) {
	var p entities.Pendidik
	if _, err := uuid.Parse(id); err != nil {
		return p, gorm.ErrRecordNotFound
	}
	err := r.tx(tx).WithContext(ctx).
		Scopes(akses.Scope(actor, "pendidik.sekolah_id")).
		Preload("Sekolah.Sudin").
		Take(&p, "pendidik.id = ?", id).Error
	return p, err
}

func (r *suratRepository) NomorDipakai(ctx context.Context, tx *gorm.DB, nomor string, kecuali uuid.UUID) (bool, error) {
	var n int64
	err := r.tx(tx).WithContext(ctx).Model(&entities.Surat{}).
		Where("nomor = ? AND id <> ?", nomor, kecuali).Count(&n).Error
	return n > 0, err
}

func (r *suratRepository) StafSudinSekolah(ctx context.Context, tx *gorm.DB, sekolahID uuid.UUID) ([]uuid.UUID, error) {
	return penerima.StafSudinSekolah(ctx, r.tx(tx), sekolahID)
}

func (r *suratRepository) OperatorSekolah(ctx context.Context, tx *gorm.DB, sekolahID uuid.UUID) ([]uuid.UUID, error) {
	return penerima.OperatorSekolah(ctx, r.tx(tx), sekolahID)
}
