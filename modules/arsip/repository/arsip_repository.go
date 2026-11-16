package repository

import (
	"context"
	"time"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"github.com/Caknoooo/go-gin-clean-starter/modules/arsip/dto"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/akses"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ArsipRepository interface {
	List(ctx context.Context, actor entities.User, f dto.Filter) ([]entities.Arsip, int64, error)
	Get(ctx context.Context, tx *gorm.DB, actor entities.User, id string, kunci bool) (entities.Arsip, error)
	Ringkasan(ctx context.Context, actor entities.User, pendidikID string) (map[string]int64, error)
	GetPendidik(ctx context.Context, actor entities.User, id string) (entities.Pendidik, error)
	Log(ctx context.Context, actor entities.User, f dto.LogFilter) ([]dto.LogResponse, int64, error)
	CatatLog(ctx context.Context, tx *gorm.DB, l *entities.ArsipLog) error
}

type arsipRepository struct {
	db *gorm.DB
}

func NewArsipRepository(db *gorm.DB) ArsipRepository {
	return &arsipRepository{db: db}
}

func (r *arsipRepository) tx(tx *gorm.DB) *gorm.DB {
	if tx == nil {
		return r.db
	}
	return tx
}

func valid(id string) bool {
	_, err := uuid.Parse(id)
	return err == nil
}

func preloadVersi(db *gorm.DB) *gorm.DB {
	return db.Order("versi DESC")
}

func (r *arsipRepository) List(ctx context.Context, actor entities.User, f dto.Filter) ([]entities.Arsip, int64, error) {
	q := r.db.WithContext(ctx).Model(&entities.Arsip{}).Scopes(akses.Scope(actor, "arsip.sekolah_id"))
	if f.PendidikID != "" {
		if !valid(f.PendidikID) {
			return nil, 0, nil
		}
		q = q.Where("arsip.pendidik_id = ?", f.PendidikID)
	}
	if f.Kategori != "" {
		q = q.Where("arsip.kategori = ?", f.Kategori)
	}
	if f.Search != "" {
		q = q.Joins("JOIN pendidik ON pendidik.id = arsip.pendidik_id").
			Where("(arsip.jenis_dokumen ILIKE ? OR arsip.nomor ILIKE ? OR pendidik.nama ILIKE ? OR pendidik.nip LIKE ?)",
				"%"+f.Search+"%", "%"+f.Search+"%", "%"+f.Search+"%", f.Search+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []entities.Arsip
	err := q.Preload("Pendidik").Preload("Sekolah").Preload("Versi", preloadVersi).Preload("Versi.Pengunggah").
		Order("arsip.updated_at DESC").Offset(f.Offset).Limit(f.Limit).Find(&items).Error
	return items, total, err
}

func (r *arsipRepository) Get(ctx context.Context, tx *gorm.DB, actor entities.User, id string, kunci bool) (entities.Arsip, error) {
	var a entities.Arsip
	if !valid(id) {
		return a, gorm.ErrRecordNotFound
	}
	q := r.tx(tx).WithContext(ctx)
	if kunci {
		q = q.Clauses(clause.Locking{Strength: "UPDATE", Table: clause.Table{Name: "arsip"}})
	}
	err := q.Scopes(akses.Scope(actor, "arsip.sekolah_id")).
		Preload("Pendidik").Preload("Sekolah").Preload("Versi", preloadVersi).
		Preload("Versi.Pengunggah").Preload("Versi.Dokumen").Preload("Versi.LampiranDokumen").
		Take(&a, "arsip.id = ?", id).Error
	return a, err
}

func (r *arsipRepository) Ringkasan(ctx context.Context, actor entities.User, pendidikID string) (map[string]int64, error) {
	out := map[string]int64{}
	if !valid(pendidikID) {
		return out, gorm.ErrRecordNotFound
	}
	var rows []struct {
		Kategori string
		Jumlah   int64
	}
	err := r.db.WithContext(ctx).Model(&entities.Arsip{}).Scopes(akses.Scope(actor, "arsip.sekolah_id")).
		Where("arsip.pendidik_id = ?", pendidikID).
		Select("kategori, COUNT(*) AS jumlah").Group("kategori").Scan(&rows).Error
	for _, row := range rows {
		out[row.Kategori] = row.Jumlah
	}
	return out, err
}

func (r *arsipRepository) GetPendidik(ctx context.Context, actor entities.User, id string) (entities.Pendidik, error) {
	var p entities.Pendidik
	if !valid(id) {
		return p, gorm.ErrRecordNotFound
	}
	err := r.db.WithContext(ctx).Scopes(akses.Scope(actor, "pendidik.sekolah_id")).Take(&p, "pendidik.id = ?", id).Error
	return p, err
}

func (r *arsipRepository) Log(ctx context.Context, actor entities.User, f dto.LogFilter) ([]dto.LogResponse, int64, error) {
	q := r.db.WithContext(ctx).Table("arsip_log").
		Joins("JOIN arsip ON arsip.id = arsip_log.arsip_id").
		Joins("JOIN pendidik ON pendidik.id = arsip.pendidik_id").
		Joins("JOIN users ON users.id = arsip_log.user_id").
		Scopes(akses.Scope(actor, "arsip.sekolah_id"))
	if t, err := time.Parse("2006-01-02", f.Dari); err == nil {
		q = q.Where("arsip_log.created_at >= ?", t)
	}
	if t, err := time.Parse("2006-01-02", f.Sampai); err == nil {
		q = q.Where("arsip_log.created_at < ?", t.AddDate(0, 0, 1))
	}
	if f.UserID != "" {
		if !valid(f.UserID) {
			return nil, 0, nil
		}
		q = q.Where("arsip_log.user_id = ?", f.UserID)
	}
	if f.Aksi != "" {
		q = q.Where("arsip_log.aksi = ?", f.Aksi)
	}
	if f.Search != "" {
		q = q.Where("(arsip.jenis_dokumen ILIKE ? OR arsip.nomor ILIKE ? OR pendidik.nama ILIKE ? OR pendidik.nip LIKE ?)",
			"%"+f.Search+"%", "%"+f.Search+"%", "%"+f.Search+"%", f.Search+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []dto.LogResponse
	err := q.Select(`arsip_log.created_at AS waktu, arsip_log.aksi, arsip_log.versi, arsip_log.keterangan,
		users.name AS pengguna, users.role AS peran, arsip.id AS arsip_id, arsip.jenis_dokumen, arsip.nomor,
		pendidik.nama AS pendidik_nama`).
		Order("arsip_log.created_at DESC").Offset(f.Offset).Limit(f.Limit).Scan(&rows).Error
	return rows, total, err
}

func (r *arsipRepository) CatatLog(ctx context.Context, tx *gorm.DB, l *entities.ArsipLog) error {
	return r.tx(tx).WithContext(ctx).Omit(clause.Associations).Create(l).Error
}
