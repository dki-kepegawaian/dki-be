package repository

import (
	"context"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"github.com/Caknoooo/go-gin-clean-starter/modules/kinerja/dto"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/akses"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/penerima"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type KinerjaRepository interface {
	Create(ctx context.Context, tx *gorm.DB, k *entities.KinerjaPenilaian) error
	Update(ctx context.Context, tx *gorm.DB, k *entities.KinerjaPenilaian) error
	AddRiwayat(ctx context.Context, tx *gorm.DB, r *entities.KinerjaRiwayat) error
	// GetForUpdate mengunci baris penilaian (SELECT ... FOR UPDATE) dalam scope actor.
	GetForUpdate(ctx context.Context, tx *gorm.DB, actor entities.User, id string) (entities.KinerjaPenilaian, error)
	GetDetail(ctx context.Context, actor entities.User, id string) (entities.KinerjaPenilaian, error)
	List(ctx context.Context, actor entities.User, f dto.Filter) ([]entities.KinerjaPenilaian, int64, error)
	Ringkasan(ctx context.Context, actor entities.User, tahun, semester int) (map[string]int64, error)
	// GetPendidikForUpdate mengunci baris pendidik dalam scope actor supaya cek
	// "satu penilaian per periode" aman dari race.
	GetPendidikForUpdate(ctx context.Context, tx *gorm.DB, actor entities.User, id string) (entities.Pendidik, error)
	AdaPenilaian(ctx context.Context, tx *gorm.DB, pendidikID uuid.UUID, tahun, semester int) (bool, error)
	// Pegawai mengembalikan pendidik PPPK_PW dalam scope beserta penilaiannya pada periode tersebut.
	Pegawai(ctx context.Context, actor entities.User, sekolahID, search string, tahun, semester, offset, limit int) ([]entities.Pendidik, map[uuid.UUID]entities.KinerjaPenilaian, int64, error)
	StafSudinSekolah(ctx context.Context, tx *gorm.DB, sekolahID uuid.UUID) ([]uuid.UUID, error)
	OperatorSekolah(ctx context.Context, tx *gorm.DB, sekolahID uuid.UUID) ([]uuid.UUID, error)
}

type kinerjaRepository struct {
	db *gorm.DB
}

func NewKinerjaRepository(db *gorm.DB) KinerjaRepository {
	return &kinerjaRepository{db: db}
}

func (r *kinerjaRepository) tx(tx *gorm.DB) *gorm.DB {
	if tx == nil {
		return r.db
	}
	return tx
}

func (r *kinerjaRepository) Create(ctx context.Context, tx *gorm.DB, k *entities.KinerjaPenilaian) error {
	return r.tx(tx).WithContext(ctx).Omit(clause.Associations).Create(k).Error
}

func (r *kinerjaRepository) Update(ctx context.Context, tx *gorm.DB, k *entities.KinerjaPenilaian) error {
	return r.tx(tx).WithContext(ctx).Omit(clause.Associations).Save(k).Error
}

func (r *kinerjaRepository) AddRiwayat(ctx context.Context, tx *gorm.DB, h *entities.KinerjaRiwayat) error {
	return r.tx(tx).WithContext(ctx).Omit(clause.Associations).Create(h).Error
}

func (r *kinerjaRepository) GetForUpdate(ctx context.Context, tx *gorm.DB, actor entities.User, id string) (entities.KinerjaPenilaian, error) {
	var k entities.KinerjaPenilaian
	if _, err := uuid.Parse(id); err != nil {
		return k, gorm.ErrRecordNotFound
	}
	err := r.tx(tx).WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE", Table: clause.Table{Name: "kinerja_penilaian"}}).
		Scopes(akses.Scope(actor, "kinerja_penilaian.sekolah_id")).
		Preload("Pendidik").Preload("Sekolah").Preload("Sekolah.Sudin").
		Take(&k, "kinerja_penilaian.id = ?", id).Error
	return k, err
}

func (r *kinerjaRepository) GetDetail(ctx context.Context, actor entities.User, id string) (entities.KinerjaPenilaian, error) {
	var k entities.KinerjaPenilaian
	if _, err := uuid.Parse(id); err != nil {
		return k, gorm.ErrRecordNotFound
	}
	err := r.db.WithContext(ctx).
		Scopes(akses.Scope(actor, "kinerja_penilaian.sekolah_id")).
		Preload("Pendidik").Preload("Sekolah").Preload("DokumenSistem").Preload("DokumenTTD").
		Preload("Riwayat", func(db *gorm.DB) *gorm.DB { return db.Order("created_at ASC") }).
		Preload("Riwayat.Pengguna").
		Take(&k, "kinerja_penilaian.id = ?", id).Error
	return k, err
}

func (r *kinerjaRepository) List(ctx context.Context, actor entities.User, f dto.Filter) ([]entities.KinerjaPenilaian, int64, error) {
	q := r.db.WithContext(ctx).Model(&entities.KinerjaPenilaian{}).Scopes(akses.Scope(actor, "kinerja_penilaian.sekolah_id"))
	if len(f.Status) > 0 {
		q = q.Where("kinerja_penilaian.status IN ?", f.Status)
	}
	if f.Tahun != 0 {
		q = q.Where("kinerja_penilaian.tahun = ?", f.Tahun)
	}
	if f.Semester != 0 {
		q = q.Where("kinerja_penilaian.semester = ?", f.Semester)
	}
	if f.SekolahID != "" {
		if _, err := uuid.Parse(f.SekolahID); err != nil {
			return nil, 0, nil
		}
		q = q.Where("kinerja_penilaian.sekolah_id = ?", f.SekolahID)
	}
	if f.Search != "" {
		q = q.Joins("JOIN pendidik ON pendidik.id = kinerja_penilaian.pendidik_id").
			Where("(kinerja_penilaian.nomor ILIKE ? OR pendidik.nama ILIKE ?)", "%"+f.Search+"%", "%"+f.Search+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []entities.KinerjaPenilaian
	err := q.Preload("Pendidik").Preload("Sekolah").Preload("DokumenSistem").Preload("DokumenTTD").
		Order("kinerja_penilaian.updated_at DESC").Offset(f.Offset).Limit(f.Limit).Find(&items).Error
	return items, total, err
}

func (r *kinerjaRepository) Ringkasan(ctx context.Context, actor entities.User, tahun, semester int) (map[string]int64, error) {
	var rows []struct {
		Status string
		Jumlah int64
	}
	q := r.db.WithContext(ctx).Model(&entities.KinerjaPenilaian{}).
		Scopes(akses.Scope(actor, "kinerja_penilaian.sekolah_id"))
	if tahun != 0 {
		q = q.Where("tahun = ?", tahun)
	}
	if semester != 0 {
		q = q.Where("semester = ?", semester)
	}
	if err := q.Select("status, COUNT(*) AS jumlah").Group("status").Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := map[string]int64{
		entities.KinerjaStatusDraf: 0, entities.KinerjaStatusMenungguTTD: 0, entities.KinerjaStatusFinal: 0,
	}
	for _, row := range rows {
		out[row.Status] = row.Jumlah
	}
	return out, nil
}

func (r *kinerjaRepository) GetPendidikForUpdate(ctx context.Context, tx *gorm.DB, actor entities.User, id string) (entities.Pendidik, error) {
	var p entities.Pendidik
	if _, err := uuid.Parse(id); err != nil {
		return p, gorm.ErrRecordNotFound
	}
	err := r.tx(tx).WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE", Table: clause.Table{Name: "pendidik"}}).
		Scopes(akses.Scope(actor, "pendidik.sekolah_id")).
		Take(&p, "pendidik.id = ?", id).Error
	return p, err
}

func (r *kinerjaRepository) AdaPenilaian(ctx context.Context, tx *gorm.DB, pendidikID uuid.UUID, tahun, semester int) (bool, error) {
	var n int64
	err := r.tx(tx).WithContext(ctx).Model(&entities.KinerjaPenilaian{}).
		Where("pendidik_id = ? AND tahun = ? AND semester = ?", pendidikID, tahun, semester).
		Count(&n).Error
	return n > 0, err
}

func (r *kinerjaRepository) Pegawai(ctx context.Context, actor entities.User, sekolahID, search string, tahun, semester, offset, limit int) ([]entities.Pendidik, map[uuid.UUID]entities.KinerjaPenilaian, int64, error) {
	q := r.db.WithContext(ctx).Model(&entities.Pendidik{}).
		Scopes(akses.Scope(actor, "pendidik.sekolah_id")).
		Where("pendidik.status_kepegawaian = ?", entities.StatusKepegawaianPPPKPW)
	if sekolahID != "" {
		if _, err := uuid.Parse(sekolahID); err != nil {
			return nil, nil, 0, nil
		}
		q = q.Where("pendidik.sekolah_id = ?", sekolahID)
	}
	if search != "" {
		q = q.Where("pendidik.nama ILIKE ?", "%"+search+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, nil, 0, err
	}
	var items []entities.Pendidik
	if err := q.Preload("Sekolah").Order("pendidik.nama ASC").Offset(offset).Limit(limit).Find(&items).Error; err != nil {
		return nil, nil, 0, err
	}
	out := map[uuid.UUID]entities.KinerjaPenilaian{}
	if len(items) == 0 {
		return items, out, total, nil
	}
	ids := make([]uuid.UUID, 0, len(items))
	for _, p := range items {
		ids = append(ids, p.ID)
	}
	var nilai []entities.KinerjaPenilaian
	err := r.db.WithContext(ctx).
		Scopes(akses.Scope(actor, "kinerja_penilaian.sekolah_id")).
		Where("pendidik_id IN ? AND tahun = ? AND semester = ?", ids, tahun, semester).
		Find(&nilai).Error
	for _, k := range nilai {
		out[k.PendidikID] = k
	}
	return items, out, total, err
}

func (r *kinerjaRepository) StafSudinSekolah(ctx context.Context, tx *gorm.DB, sekolahID uuid.UUID) ([]uuid.UUID, error) {
	return penerima.StafSudinSekolah(ctx, r.tx(tx), sekolahID)
}

func (r *kinerjaRepository) OperatorSekolah(ctx context.Context, tx *gorm.DB, sekolahID uuid.UUID) ([]uuid.UUID, error) {
	return penerima.OperatorSekolah(ctx, r.tx(tx), sekolahID)
}
