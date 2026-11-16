package repository

import (
	"context"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"github.com/Caknoooo/go-gin-clean-starter/modules/dapodik/dto"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/akses"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/penerima"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DapodikRepository interface {
	Create(ctx context.Context, tx *gorm.DB, t *entities.DapodikTiket) error
	Update(ctx context.Context, tx *gorm.DB, t *entities.DapodikTiket) error
	AddRiwayat(ctx context.Context, tx *gorm.DB, r *entities.DapodikRiwayat) error
	AddLampiran(ctx context.Context, tx *gorm.DB, l *entities.DapodikLampiran) error
	// GetForUpdate mengunci baris tiket (SELECT ... FOR UPDATE) dalam scope actor.
	GetForUpdate(ctx context.Context, tx *gorm.DB, actor entities.User, id string) (entities.DapodikTiket, error)
	GetDetail(ctx context.Context, actor entities.User, id string) (entities.DapodikTiket, error)
	List(ctx context.Context, actor entities.User, f dto.Filter) ([]entities.DapodikTiket, int64, error)
	Ringkasan(ctx context.Context, actor entities.User) (map[string]int64, error)
	GetPendidik(ctx context.Context, tx *gorm.DB, actor entities.User, id string) (entities.Pendidik, error)
	StafSudinSekolah(ctx context.Context, tx *gorm.DB, sekolahID uuid.UUID) ([]uuid.UUID, error)
	OperatorSekolah(ctx context.Context, tx *gorm.DB, sekolahID uuid.UUID) ([]uuid.UUID, error)
}

type dapodikRepository struct {
	db *gorm.DB
}

func NewDapodikRepository(db *gorm.DB) DapodikRepository {
	return &dapodikRepository{db: db}
}

func (r *dapodikRepository) tx(tx *gorm.DB) *gorm.DB {
	if tx == nil {
		return r.db
	}
	return tx
}

func (r *dapodikRepository) Create(ctx context.Context, tx *gorm.DB, t *entities.DapodikTiket) error {
	return r.tx(tx).WithContext(ctx).Omit(clause.Associations).Create(t).Error
}

func (r *dapodikRepository) Update(ctx context.Context, tx *gorm.DB, t *entities.DapodikTiket) error {
	return r.tx(tx).WithContext(ctx).Omit(clause.Associations).Save(t).Error
}

func (r *dapodikRepository) AddRiwayat(ctx context.Context, tx *gorm.DB, h *entities.DapodikRiwayat) error {
	return r.tx(tx).WithContext(ctx).Omit(clause.Associations).Create(h).Error
}

func (r *dapodikRepository) AddLampiran(ctx context.Context, tx *gorm.DB, l *entities.DapodikLampiran) error {
	return r.tx(tx).WithContext(ctx).Omit(clause.Associations).Create(l).Error
}

func (r *dapodikRepository) GetForUpdate(ctx context.Context, tx *gorm.DB, actor entities.User, id string) (entities.DapodikTiket, error) {
	var t entities.DapodikTiket
	if _, err := uuid.Parse(id); err != nil {
		return t, gorm.ErrRecordNotFound
	}
	err := r.tx(tx).WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE", Table: clause.Table{Name: "dapodik_tiket"}}).
		Scopes(akses.Scope(actor, "dapodik_tiket.sekolah_id")).
		Preload("Sekolah").
		Take(&t, "dapodik_tiket.id = ?", id).Error
	return t, err
}

func (r *dapodikRepository) GetDetail(ctx context.Context, actor entities.User, id string) (entities.DapodikTiket, error) {
	var t entities.DapodikTiket
	if _, err := uuid.Parse(id); err != nil {
		return t, gorm.ErrRecordNotFound
	}
	err := r.db.WithContext(ctx).
		Scopes(akses.Scope(actor, "dapodik_tiket.sekolah_id")).
		Preload("Sekolah").Preload("Pendidik").Preload("Pengaju").Preload("Penyelesai").
		Preload("Lampiran", func(db *gorm.DB) *gorm.DB { return db.Order("created_at ASC") }).
		Preload("Lampiran.Dokumen").
		Preload("Riwayat", func(db *gorm.DB) *gorm.DB { return db.Order("created_at ASC") }).
		Preload("Riwayat.Pengguna").
		Take(&t, "dapodik_tiket.id = ?", id).Error
	return t, err
}

func (r *dapodikRepository) List(ctx context.Context, actor entities.User, f dto.Filter) ([]entities.DapodikTiket, int64, error) {
	q := r.db.WithContext(ctx).Model(&entities.DapodikTiket{}).Scopes(akses.Scope(actor, "dapodik_tiket.sekolah_id"))
	if len(f.Status) > 0 {
		q = q.Where("dapodik_tiket.status IN ?", f.Status)
	}
	if f.Jenis != "" {
		q = q.Where("dapodik_tiket.jenis = ?", f.Jenis)
	}
	if f.SekolahID != "" {
		if _, err := uuid.Parse(f.SekolahID); err != nil {
			return nil, 0, nil
		}
		q = q.Where("dapodik_tiket.sekolah_id = ?", f.SekolahID)
	}
	if f.Search != "" {
		q = q.Joins("JOIN sekolah ON sekolah.id = dapodik_tiket.sekolah_id").
			Where("(dapodik_tiket.nomor ILIKE ? OR sekolah.nama ILIKE ?)", "%"+f.Search+"%", "%"+f.Search+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	order := "dapodik_tiket.created_at DESC"
	if f.Urut == "tenggat" {
		order = "dapodik_tiket.tenggat ASC, dapodik_tiket.created_at ASC"
	}
	var items []entities.DapodikTiket
	err := q.Preload("Sekolah").Preload("Pendidik").
		Order(order).Offset(f.Offset).Limit(f.Limit).Find(&items).Error
	return items, total, err
}

func (r *dapodikRepository) Ringkasan(ctx context.Context, actor entities.User) (map[string]int64, error) {
	var rows []struct {
		Status string
		Jumlah int64
	}
	err := r.db.WithContext(ctx).Model(&entities.DapodikTiket{}).
		Scopes(akses.Scope(actor, "dapodik_tiket.sekolah_id")).
		Select("status, COUNT(*) AS jumlah").Group("status").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := map[string]int64{
		entities.DapodikStatusBaru: 0, entities.DapodikStatusPerluInfoTambahan: 0, entities.DapodikStatusSelesai: 0,
	}
	for _, row := range rows {
		out[row.Status] = row.Jumlah
	}
	return out, nil
}

func (r *dapodikRepository) GetPendidik(ctx context.Context, tx *gorm.DB, actor entities.User, id string) (entities.Pendidik, error) {
	var p entities.Pendidik
	if _, err := uuid.Parse(id); err != nil {
		return p, gorm.ErrRecordNotFound
	}
	err := r.tx(tx).WithContext(ctx).Scopes(akses.Scope(actor, "pendidik.sekolah_id")).
		Take(&p, "pendidik.id = ?", id).Error
	return p, err
}

func (r *dapodikRepository) StafSudinSekolah(ctx context.Context, tx *gorm.DB, sekolahID uuid.UUID) ([]uuid.UUID, error) {
	return penerima.StafSudinSekolah(ctx, r.tx(tx), sekolahID)
}

func (r *dapodikRepository) OperatorSekolah(ctx context.Context, tx *gorm.DB, sekolahID uuid.UUID) ([]uuid.UUID, error) {
	return penerima.OperatorSekolah(ctx, r.tx(tx), sekolahID)
}
