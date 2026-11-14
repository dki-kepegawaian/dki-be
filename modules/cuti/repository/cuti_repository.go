package repository

import (
	"context"
	"time"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"github.com/Caknoooo/go-gin-clean-starter/modules/cuti/dto"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/akses"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/constants"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CutiRepository interface {
	Create(ctx context.Context, tx *gorm.DB, c *entities.Cuti) error
	Update(ctx context.Context, tx *gorm.DB, c *entities.Cuti) error
	AddRiwayat(ctx context.Context, tx *gorm.DB, r *entities.CutiRiwayat) error
	// GetForUpdate mengunci baris cuti (SELECT ... FOR UPDATE) dalam scope actor.
	GetForUpdate(ctx context.Context, tx *gorm.DB, actor entities.User, id string) (entities.Cuti, error)
	GetDetail(ctx context.Context, actor entities.User, id string) (entities.Cuti, error)
	List(ctx context.Context, actor entities.User, f dto.Filter) ([]entities.Cuti, int64, error)
	Ringkasan(ctx context.Context, actor entities.User) (map[string]int64, error)
	// GetPendidik mengambil pendidik dalam scope actor. kunci=true mengunci barisnya
	// supaya cek sisa cuti & tumpang tindih saat pengajuan aman dari race.
	GetPendidik(ctx context.Context, tx *gorm.DB, actor entities.User, id string, kunci bool) (entities.Pendidik, error)
	HariTahunanTerpakai(ctx context.Context, tx *gorm.DB, pendidikID uuid.UUID, tahun int) (int, error)
	AdaTumpangTindih(ctx context.Context, tx *gorm.DB, pendidikID uuid.UUID, mulai, selesai time.Time) (bool, error)
	StafSudinSekolah(ctx context.Context, tx *gorm.DB, sekolahID uuid.UUID) ([]uuid.UUID, error)
	StafSudinValid(ctx context.Context, tx *gorm.DB, userID uuid.UUID, sudinID uuid.UUID) (bool, error)
}

type cutiRepository struct {
	db *gorm.DB
}

func NewCutiRepository(db *gorm.DB) CutiRepository {
	return &cutiRepository{db: db}
}

func (r *cutiRepository) tx(tx *gorm.DB) *gorm.DB {
	if tx == nil {
		return r.db
	}
	return tx
}

func (r *cutiRepository) Create(ctx context.Context, tx *gorm.DB, c *entities.Cuti) error {
	return r.tx(tx).WithContext(ctx).Omit(clause.Associations).Create(c).Error
}

func (r *cutiRepository) Update(ctx context.Context, tx *gorm.DB, c *entities.Cuti) error {
	return r.tx(tx).WithContext(ctx).Omit(clause.Associations).Save(c).Error
}

func (r *cutiRepository) AddRiwayat(ctx context.Context, tx *gorm.DB, h *entities.CutiRiwayat) error {
	return r.tx(tx).WithContext(ctx).Omit(clause.Associations).Create(h).Error
}

func (r *cutiRepository) GetForUpdate(ctx context.Context, tx *gorm.DB, actor entities.User, id string) (entities.Cuti, error) {
	var c entities.Cuti
	if _, err := uuid.Parse(id); err != nil {
		return c, gorm.ErrRecordNotFound
	}
	err := r.tx(tx).WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE", Table: clause.Table{Name: "cuti"}}).
		Scopes(akses.Scope(actor, "cuti.sekolah_id")).
		Preload("Pendidik").Preload("Sekolah").
		Take(&c, "cuti.id = ?", id).Error
	return c, err
}

func (r *cutiRepository) GetDetail(ctx context.Context, actor entities.User, id string) (entities.Cuti, error) {
	var c entities.Cuti
	if _, err := uuid.Parse(id); err != nil {
		return c, gorm.ErrRecordNotFound
	}
	err := r.db.WithContext(ctx).
		Scopes(akses.Scope(actor, "cuti.sekolah_id")).
		Preload("Pendidik").Preload("Sekolah").Preload("BerkasPermohonan").Preload("SuratIzin").
		Preload("PenerimaSurat").
		Preload("Riwayat", func(db *gorm.DB) *gorm.DB { return db.Order("created_at ASC") }).
		Preload("Riwayat.Pengguna").
		Take(&c, "cuti.id = ?", id).Error
	return c, err
}

func (r *cutiRepository) List(ctx context.Context, actor entities.User, f dto.Filter) ([]entities.Cuti, int64, error) {
	q := r.db.WithContext(ctx).Model(&entities.Cuti{}).Scopes(akses.Scope(actor, "cuti.sekolah_id"))
	if len(f.Status) > 0 {
		q = q.Where("cuti.status IN ?", f.Status)
	}
	if f.Jenis != "" {
		q = q.Where("cuti.jenis = ?", f.Jenis)
	}
	if f.SekolahID != "" {
		if _, err := uuid.Parse(f.SekolahID); err != nil {
			return nil, 0, nil
		}
		q = q.Where("cuti.sekolah_id = ?", f.SekolahID)
	}
	if f.Search != "" {
		q = q.Joins("JOIN pendidik ON pendidik.id = cuti.pendidik_id").
			Where("(cuti.nomor ILIKE ? OR pendidik.nama ILIKE ? OR pendidik.nip LIKE ?)",
				"%"+f.Search+"%", "%"+f.Search+"%", f.Search+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []entities.Cuti
	err := q.Preload("Pendidik").Preload("Sekolah").
		Order("cuti.created_at DESC").Offset(f.Offset).Limit(f.Limit).Find(&items).Error
	return items, total, err
}

func (r *cutiRepository) Ringkasan(ctx context.Context, actor entities.User) (map[string]int64, error) {
	var rows []struct {
		Status string
		Jumlah int64
	}
	err := r.db.WithContext(ctx).Model(&entities.Cuti{}).
		Scopes(akses.Scope(actor, "cuti.sekolah_id")).
		Select("status, COUNT(*) AS jumlah").Group("status").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := map[string]int64{
		entities.CutiStatusDiajukan: 0, entities.CutiStatusPerluPerbaikan: 0,
		entities.CutiStatusMenungguSuratFisik: 0, entities.CutiStatusDiprosesSudin: 0,
		entities.CutiStatusDisetujui: 0, entities.CutiStatusDitolak: 0,
	}
	for _, row := range rows {
		out[row.Status] = row.Jumlah
	}
	return out, nil
}

func (r *cutiRepository) GetPendidik(ctx context.Context, tx *gorm.DB, actor entities.User, id string, kunci bool) (entities.Pendidik, error) {
	var p entities.Pendidik
	if _, err := uuid.Parse(id); err != nil {
		return p, gorm.ErrRecordNotFound
	}
	q := r.tx(tx).WithContext(ctx)
	if kunci {
		q = q.Clauses(clause.Locking{Strength: "UPDATE", Table: clause.Table{Name: "pendidik"}})
	}
	err := q.Scopes(akses.Scope(actor, "pendidik.sekolah_id")).
		Preload("Sekolah").
		Take(&p, "pendidik.id = ?", id).Error
	return p, err
}

func (r *cutiRepository) HariTahunanTerpakai(ctx context.Context, tx *gorm.DB, pendidikID uuid.UUID, tahun int) (int, error) {
	var n int
	err := r.tx(tx).WithContext(ctx).Model(&entities.Cuti{}).
		Select("COALESCE(SUM(hari_kerja), 0)").
		Where("pendidik_id = ? AND jenis = ? AND status <> ?", pendidikID, entities.CutiJenisTahunan, entities.CutiStatusDitolak).
		Where("EXTRACT(YEAR FROM tanggal_mulai) = ?", tahun).
		Scan(&n).Error
	return n, err
}

func (r *cutiRepository) AdaTumpangTindih(ctx context.Context, tx *gorm.DB, pendidikID uuid.UUID, mulai, selesai time.Time) (bool, error) {
	var n int64
	err := r.tx(tx).WithContext(ctx).Model(&entities.Cuti{}).
		Where("pendidik_id = ? AND status <> ?", pendidikID, entities.CutiStatusDitolak).
		Where("tanggal_mulai <= ? AND tanggal_selesai >= ?", selesai, mulai).
		Count(&n).Error
	return n > 0, err
}

func (r *cutiRepository) StafSudinSekolah(ctx context.Context, tx *gorm.DB, sekolahID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := r.tx(tx).WithContext(ctx).Model(&entities.User{}).
		Where("role = ? AND sudin_id = (SELECT sudin_id FROM sekolah WHERE id = ?)", constants.ENUM_ROLE_STAF_SUDIN, sekolahID).
		Pluck("id", &ids).Error
	return ids, err
}

func (r *cutiRepository) StafSudinValid(ctx context.Context, tx *gorm.DB, userID uuid.UUID, sudinID uuid.UUID) (bool, error) {
	var n int64
	err := r.tx(tx).WithContext(ctx).Model(&entities.User{}).
		Where("id = ? AND role = ? AND sudin_id = ?", userID, constants.ENUM_ROLE_STAF_SUDIN, sudinID).
		Count(&n).Error
	return n > 0, err
}
