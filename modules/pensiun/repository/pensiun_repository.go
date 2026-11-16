package repository

import (
	"context"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"github.com/Caknoooo/go-gin-clean-starter/modules/pensiun/dto"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/akses"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/penerima"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PensiunRepository interface {
	Create(ctx context.Context, tx *gorm.DB, p *entities.Pensiun) error
	Update(ctx context.Context, tx *gorm.DB, p *entities.Pensiun) error
	SaveBerkas(ctx context.Context, tx *gorm.DB, b *entities.PensiunBerkas) error
	AddRiwayat(ctx context.Context, tx *gorm.DB, r *entities.PensiunRiwayat) error
	// GetForUpdate mengunci baris pensiun (SELECT ... FOR UPDATE) dalam scope actor.
	GetForUpdate(ctx context.Context, tx *gorm.DB, actor entities.User, id string) (entities.Pensiun, error)
	GetDetail(ctx context.Context, actor entities.User, id string) (entities.Pensiun, error)
	List(ctx context.Context, actor entities.User, f dto.Filter) ([]entities.Pensiun, int64, error)
	Ringkasan(ctx context.Context, actor entities.User) (map[string]int64, error)
	// GetPendidik mengambil pendidik dalam scope actor; kunci=true mengunci barisnya
	// supaya cek pengajuan ganda aman dari race.
	GetPendidik(ctx context.Context, tx *gorm.DB, actor entities.User, id string, kunci bool) (entities.Pendidik, error)
	// NomorPengajuan mengembalikan nomor pengajuan pensiun pendidik (status apa pun), "" bila belum ada.
	NomorPengajuan(ctx context.Context, tx *gorm.DB, pendidikID uuid.UUID) (string, error)
	StafSudinSekolah(ctx context.Context, tx *gorm.DB, sekolahID uuid.UUID) ([]uuid.UUID, error)
	OperatorSekolah(ctx context.Context, tx *gorm.DB, sekolahID uuid.UUID) ([]uuid.UUID, error)
}

type pensiunRepository struct {
	db *gorm.DB
}

func NewPensiunRepository(db *gorm.DB) PensiunRepository {
	return &pensiunRepository{db: db}
}

func (r *pensiunRepository) tx(tx *gorm.DB) *gorm.DB {
	if tx == nil {
		return r.db
	}
	return tx
}

func (r *pensiunRepository) Create(ctx context.Context, tx *gorm.DB, p *entities.Pensiun) error {
	return r.tx(tx).WithContext(ctx).Omit(clause.Associations).Create(p).Error
}

func (r *pensiunRepository) Update(ctx context.Context, tx *gorm.DB, p *entities.Pensiun) error {
	return r.tx(tx).WithContext(ctx).Omit(clause.Associations).Save(p).Error
}

func (r *pensiunRepository) SaveBerkas(ctx context.Context, tx *gorm.DB, b *entities.PensiunBerkas) error {
	return r.tx(tx).WithContext(ctx).Omit(clause.Associations).Save(b).Error
}

func (r *pensiunRepository) AddRiwayat(ctx context.Context, tx *gorm.DB, h *entities.PensiunRiwayat) error {
	return r.tx(tx).WithContext(ctx).Omit(clause.Associations).Create(h).Error
}

func (r *pensiunRepository) GetForUpdate(ctx context.Context, tx *gorm.DB, actor entities.User, id string) (entities.Pensiun, error) {
	var p entities.Pensiun
	if _, err := uuid.Parse(id); err != nil {
		return p, gorm.ErrRecordNotFound
	}
	err := r.tx(tx).WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE", Table: clause.Table{Name: "pensiun"}}).
		Scopes(akses.Scope(actor, "pensiun.sekolah_id")).
		Preload("Pendidik").Preload("Sekolah").Preload("Berkas").
		Take(&p, "pensiun.id = ?", id).Error
	return p, err
}

func (r *pensiunRepository) GetDetail(ctx context.Context, actor entities.User, id string) (entities.Pensiun, error) {
	var p entities.Pensiun
	if _, err := uuid.Parse(id); err != nil {
		return p, gorm.ErrRecordNotFound
	}
	err := r.db.WithContext(ctx).
		Scopes(akses.Scope(actor, "pensiun.sekolah_id")).
		Preload("Pendidik").Preload("Sekolah").Preload("BuktiKirim").Preload("SK").
		Preload("Berkas").Preload("Berkas.Dokumen").
		Preload("Riwayat", func(db *gorm.DB) *gorm.DB { return db.Order("created_at ASC") }).
		Preload("Riwayat.Pengguna").
		Take(&p, "pensiun.id = ?", id).Error
	return p, err
}

func (r *pensiunRepository) List(ctx context.Context, actor entities.User, f dto.Filter) ([]entities.Pensiun, int64, error) {
	q := r.db.WithContext(ctx).Model(&entities.Pensiun{}).Scopes(akses.Scope(actor, "pensiun.sekolah_id"))
	if len(f.Status) > 0 {
		q = q.Where("pensiun.status IN ?", f.Status)
	}
	if f.SekolahID != "" {
		if _, err := uuid.Parse(f.SekolahID); err != nil {
			return nil, 0, nil
		}
		q = q.Where("pensiun.sekolah_id = ?", f.SekolahID)
	}
	if f.Search != "" {
		q = q.Joins("JOIN pendidik ON pendidik.id = pensiun.pendidik_id").
			Where("(pensiun.nomor ILIKE ? OR pendidik.nama ILIKE ? OR pendidik.nip LIKE ?)",
				"%"+f.Search+"%", "%"+f.Search+"%", f.Search+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []entities.Pensiun
	err := q.Preload("Pendidik").Preload("Sekolah").Preload("Berkas").
		Order("pensiun.created_at DESC").Offset(f.Offset).Limit(f.Limit).Find(&items).Error
	return items, total, err
}

func (r *pensiunRepository) Ringkasan(ctx context.Context, actor entities.User) (map[string]int64, error) {
	var rows []struct {
		Status string
		Jumlah int64
	}
	err := r.db.WithContext(ctx).Model(&entities.Pensiun{}).
		Scopes(akses.Scope(actor, "pensiun.sekolah_id")).
		Select("status, COUNT(*) AS jumlah").Group("status").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := map[string]int64{
		entities.PensiunStatusDiajukan: 0, entities.PensiunStatusPerluPerbaikan: 0,
		entities.PensiunStatusTerverifikasi: 0, entities.PensiunStatusDiusulkanKeBKD: 0,
		entities.PensiunStatusSelesai: 0,
	}
	for _, row := range rows {
		out[row.Status] = row.Jumlah
	}
	return out, nil
}

func (r *pensiunRepository) GetPendidik(ctx context.Context, tx *gorm.DB, actor entities.User, id string, kunci bool) (entities.Pendidik, error) {
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

func (r *pensiunRepository) NomorPengajuan(ctx context.Context, tx *gorm.DB, pendidikID uuid.UUID) (string, error) {
	var nomor []string
	err := r.tx(tx).WithContext(ctx).Model(&entities.Pensiun{}).
		Where("pendidik_id = ?", pendidikID).Order("created_at DESC").Limit(1).Pluck("nomor", &nomor).Error
	if err != nil || len(nomor) == 0 {
		return "", err
	}
	return nomor[0], nil
}

func (r *pensiunRepository) StafSudinSekolah(ctx context.Context, tx *gorm.DB, sekolahID uuid.UUID) ([]uuid.UUID, error) {
	return penerima.StafSudinSekolah(ctx, r.tx(tx), sekolahID)
}

func (r *pensiunRepository) OperatorSekolah(ctx context.Context, tx *gorm.DB, sekolahID uuid.UUID) ([]uuid.UUID, error) {
	return penerima.OperatorSekolah(ctx, r.tx(tx), sekolahID)
}
