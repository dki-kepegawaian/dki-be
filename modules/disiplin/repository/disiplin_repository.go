package repository

import (
	"context"
	"time"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"github.com/Caknoooo/go-gin-clean-starter/modules/disiplin/dto"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/akses"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/penerima"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DisiplinRepository interface {
	Create(ctx context.Context, tx *gorm.DB, d *entities.Disiplin) error
	Update(ctx context.Context, tx *gorm.DB, d *entities.Disiplin) error
	AddRiwayat(ctx context.Context, tx *gorm.DB, r *entities.DisiplinRiwayat) error
	AddBerkas(ctx context.Context, tx *gorm.DB, b *entities.DisiplinBerkas) error
	JumlahBerkas(ctx context.Context, tx *gorm.DB, disiplinID uuid.UUID, jenis string) (int64, error)
	// GetForUpdate mengunci baris kasus (SELECT ... FOR UPDATE) dalam scope actor.
	GetForUpdate(ctx context.Context, tx *gorm.DB, actor entities.User, id string) (entities.Disiplin, error)
	GetDetail(ctx context.Context, actor entities.User, id string) (entities.Disiplin, error)
	// GetBerkas mengambil berkas kasus dalam scope actor. internal=false menyaring berkas internal.
	GetBerkas(ctx context.Context, actor entities.User, id, dokumenID string, internal bool) (entities.DisiplinBerkas, error)
	List(ctx context.Context, actor entities.User, f dto.Filter) ([]entities.Disiplin, int64, error)
	PerStatus(ctx context.Context, actor entities.User) (map[string]int64, error)
	TenggatDekat(ctx context.Context, actor entities.User, batas time.Time) ([]string, error)
	// RiwayatHukuman: kasus lain pegawai yang sama yang sudah dijatuhi hukuman, dalam scope actor.
	RiwayatHukuman(ctx context.Context, actor entities.User, pendidikID, kecuali uuid.UUID) ([]entities.Disiplin, error)
	GetPendidik(ctx context.Context, tx *gorm.DB, actor entities.User, id string) (entities.Pendidik, error)
	StafSudinSekolah(ctx context.Context, tx *gorm.DB, sekolahID uuid.UUID) ([]uuid.UUID, error)
	PimpinanSekolah(ctx context.Context, tx *gorm.DB, sekolahID uuid.UUID) ([]uuid.UUID, error)
	OperatorSekolah(ctx context.Context, tx *gorm.DB, sekolahID uuid.UUID) ([]uuid.UUID, error)
}

type disiplinRepository struct {
	db *gorm.DB
}

func NewDisiplinRepository(db *gorm.DB) DisiplinRepository {
	return &disiplinRepository{db: db}
}

func (r *disiplinRepository) tx(tx *gorm.DB) *gorm.DB {
	if tx == nil {
		return r.db
	}
	return tx
}

func (r *disiplinRepository) Create(ctx context.Context, tx *gorm.DB, d *entities.Disiplin) error {
	return r.tx(tx).WithContext(ctx).Omit(clause.Associations).Create(d).Error
}

func (r *disiplinRepository) Update(ctx context.Context, tx *gorm.DB, d *entities.Disiplin) error {
	return r.tx(tx).WithContext(ctx).Omit(clause.Associations).Save(d).Error
}

func (r *disiplinRepository) AddRiwayat(ctx context.Context, tx *gorm.DB, h *entities.DisiplinRiwayat) error {
	return r.tx(tx).WithContext(ctx).Omit(clause.Associations).Create(h).Error
}

func (r *disiplinRepository) AddBerkas(ctx context.Context, tx *gorm.DB, b *entities.DisiplinBerkas) error {
	return r.tx(tx).WithContext(ctx).Omit(clause.Associations).Create(b).Error
}

func (r *disiplinRepository) JumlahBerkas(ctx context.Context, tx *gorm.DB, disiplinID uuid.UUID, jenis string) (int64, error) {
	var n int64
	err := r.tx(tx).WithContext(ctx).Model(&entities.DisiplinBerkas{}).
		Where("disiplin_id = ? AND jenis = ?", disiplinID, jenis).Count(&n).Error
	return n, err
}

func (r *disiplinRepository) GetForUpdate(ctx context.Context, tx *gorm.DB, actor entities.User, id string) (entities.Disiplin, error) {
	var d entities.Disiplin
	if _, err := uuid.Parse(id); err != nil {
		return d, gorm.ErrRecordNotFound
	}
	err := r.tx(tx).WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE", Table: clause.Table{Name: "disiplin"}}).
		Scopes(akses.Scope(actor, "disiplin.sekolah_id")).
		Preload("Pendidik").Preload("Sekolah").
		Take(&d, "disiplin.id = ?", id).Error
	return d, err
}

func (r *disiplinRepository) GetDetail(ctx context.Context, actor entities.User, id string) (entities.Disiplin, error) {
	var d entities.Disiplin
	if _, err := uuid.Parse(id); err != nil {
		return d, gorm.ErrRecordNotFound
	}
	err := r.db.WithContext(ctx).
		Scopes(akses.Scope(actor, "disiplin.sekolah_id")).
		Preload("Pendidik").Preload("Sekolah").
		Preload("Pelapor").Preload("Verifikator").Preload("Perekomendasi").Preload("Pemutus").
		Preload("PencatatIDIS").Preload("Penutup").
		Preload("Berkas", func(db *gorm.DB) *gorm.DB { return db.Order("created_at ASC") }).
		Preload("Berkas.Dokumen").
		Preload("Riwayat", func(db *gorm.DB) *gorm.DB { return db.Order("created_at ASC") }).
		Preload("Riwayat.Pengguna").
		Take(&d, "disiplin.id = ?", id).Error
	return d, err
}

func (r *disiplinRepository) GetBerkas(ctx context.Context, actor entities.User, id, dokumenID string, internal bool) (entities.DisiplinBerkas, error) {
	var b entities.DisiplinBerkas
	if _, err := uuid.Parse(id); err != nil {
		return b, gorm.ErrRecordNotFound
	}
	if _, err := uuid.Parse(dokumenID); err != nil {
		return b, gorm.ErrRecordNotFound
	}
	q := r.db.WithContext(ctx).
		Joins("JOIN disiplin ON disiplin.id = disiplin_berkas.disiplin_id").
		Scopes(akses.Scope(actor, "disiplin.sekolah_id")).
		Where("disiplin_berkas.disiplin_id = ? AND disiplin_berkas.dokumen_id = ?", id, dokumenID)
	if !internal {
		q = q.Where("disiplin_berkas.internal = false")
	}
	err := q.Preload("Dokumen").Take(&b).Error
	return b, err
}

func (r *disiplinRepository) List(ctx context.Context, actor entities.User, f dto.Filter) ([]entities.Disiplin, int64, error) {
	q := r.db.WithContext(ctx).Model(&entities.Disiplin{}).Scopes(akses.Scope(actor, "disiplin.sekolah_id"))
	if len(f.Status) > 0 {
		q = q.Where("disiplin.status IN ?", f.Status)
	}
	if f.SekolahID != "" {
		if _, err := uuid.Parse(f.SekolahID); err != nil {
			return nil, 0, nil
		}
		q = q.Where("disiplin.sekolah_id = ?", f.SekolahID)
	}
	if f.Tahun > 0 {
		q = q.Where("EXTRACT(YEAR FROM disiplin.created_at) = ?", f.Tahun)
	}
	if f.Search != "" {
		q = q.Joins("JOIN pendidik ON pendidik.id = disiplin.pendidik_id").
			Where("(disiplin.nomor ILIKE ? OR pendidik.nama ILIKE ? OR pendidik.nip LIKE ?)",
				"%"+f.Search+"%", "%"+f.Search+"%", f.Search+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []entities.Disiplin
	err := q.Preload("Pendidik").Preload("Sekolah").Preload("Berkas").
		Order("disiplin.created_at DESC").Offset(f.Offset).Limit(f.Limit).Find(&items).Error
	return items, total, err
}

func (r *disiplinRepository) PerStatus(ctx context.Context, actor entities.User) (map[string]int64, error) {
	var rows []struct {
		Status string
		Jumlah int64
	}
	err := r.db.WithContext(ctx).Model(&entities.Disiplin{}).
		Scopes(akses.Scope(actor, "disiplin.sekolah_id")).
		Select("status, COUNT(*) AS jumlah").Group("status").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := map[string]int64{
		entities.DisiplinStatusRegistrasi: 0, entities.DisiplinStatusPerluKelengkapan: 0,
		entities.DisiplinStatusPemeriksaan: 0, entities.DisiplinStatusMenungguKeputusan: 0,
		entities.DisiplinStatusSanksiDitetapkan: 0, entities.DisiplinStatusTindakLanjut: 0,
		entities.DisiplinStatusSelesai: 0, entities.DisiplinStatusTidakDilanjutkan: 0,
	}
	for _, row := range rows {
		out[row.Status] = row.Jumlah
	}
	return out, nil
}

func (r *disiplinRepository) TenggatDekat(ctx context.Context, actor entities.User, batas time.Time) ([]string, error) {
	out := []string{}
	err := r.db.WithContext(ctx).Model(&entities.Disiplin{}).
		Scopes(akses.Scope(actor, "disiplin.sekolah_id")).
		Where("disiplin.status NOT IN ?", []string{entities.DisiplinStatusSelesai, entities.DisiplinStatusTidakDilanjutkan}).
		Where("disiplin.tenggat IS NOT NULL AND disiplin.tenggat <= ?", batas).
		Order("disiplin.tenggat ASC").Limit(50).Pluck("disiplin.nomor", &out).Error
	return out, err
}

func (r *disiplinRepository) RiwayatHukuman(ctx context.Context, actor entities.User, pendidikID, kecuali uuid.UUID) ([]entities.Disiplin, error) {
	var out []entities.Disiplin
	err := r.db.WithContext(ctx).
		Scopes(akses.Scope(actor, "disiplin.sekolah_id")).
		Where("disiplin.pendidik_id = ? AND disiplin.id <> ? AND disiplin.jenis_hukuman <> ''", pendidikID, kecuali).
		Where("disiplin.status IN ?", []string{entities.DisiplinStatusSanksiDitetapkan, entities.DisiplinStatusTindakLanjut, entities.DisiplinStatusSelesai}).
		Order("disiplin.created_at DESC").Limit(20).Find(&out).Error
	return out, err
}

func (r *disiplinRepository) GetPendidik(ctx context.Context, tx *gorm.DB, actor entities.User, id string) (entities.Pendidik, error) {
	var p entities.Pendidik
	if _, err := uuid.Parse(id); err != nil {
		return p, gorm.ErrRecordNotFound
	}
	err := r.tx(tx).WithContext(ctx).Scopes(akses.Scope(actor, "pendidik.sekolah_id")).
		Preload("Sekolah").Take(&p, "pendidik.id = ?", id).Error
	return p, err
}

func (r *disiplinRepository) StafSudinSekolah(ctx context.Context, tx *gorm.DB, sekolahID uuid.UUID) ([]uuid.UUID, error) {
	return penerima.StafSudinSekolah(ctx, r.tx(tx), sekolahID)
}

func (r *disiplinRepository) PimpinanSekolah(ctx context.Context, tx *gorm.DB, sekolahID uuid.UUID) ([]uuid.UUID, error) {
	return penerima.PimpinanSekolah(ctx, r.tx(tx), sekolahID)
}

func (r *disiplinRepository) OperatorSekolah(ctx context.Context, tx *gorm.DB, sekolahID uuid.UUID) ([]uuid.UUID, error) {
	return penerima.OperatorSekolah(ctx, r.tx(tx), sekolahID)
}
