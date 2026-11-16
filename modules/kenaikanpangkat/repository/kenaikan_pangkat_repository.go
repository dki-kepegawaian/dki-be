package repository

import (
	"context"
	"time"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"github.com/Caknoooo/go-gin-clean-starter/modules/kenaikanpangkat/dto"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/akses"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/penerima"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type KenaikanPangkatRepository interface {
	Create(ctx context.Context, tx *gorm.DB, p *entities.KenaikanPangkat) error
	Update(ctx context.Context, tx *gorm.DB, p *entities.KenaikanPangkat) error
	SaveBerkas(ctx context.Context, tx *gorm.DB, b *entities.KenaikanPangkatBerkas) error
	AddRiwayat(ctx context.Context, tx *gorm.DB, r *entities.KenaikanPangkatRiwayat) error
	// GetForUpdate mengunci baris kenaikan pangkat (SELECT ... FOR UPDATE) dalam scope actor.
	GetForUpdate(ctx context.Context, tx *gorm.DB, actor entities.User, id string) (entities.KenaikanPangkat, error)
	GetDetail(ctx context.Context, actor entities.User, id string) (entities.KenaikanPangkat, error)
	List(ctx context.Context, actor entities.User, f dto.Filter) ([]entities.KenaikanPangkat, int64, error)
	Ringkasan(ctx context.Context, actor entities.User) (map[string]int64, error)
	// GetPendidik mengambil pendidik dalam scope actor; kunci=true mengunci barisnya
	// supaya cek pengajuan ganda aman dari race.
	GetPendidik(ctx context.Context, tx *gorm.DB, actor entities.User, id string, kunci bool) (entities.Pendidik, error)
	// NomorPengajuan mengembalikan nomor pengajuan KP pendidik yang belum selesai, "" bila tidak ada.
	NomorPengajuan(ctx context.Context, tx *gorm.DB, pendidikID uuid.UUID) (string, error)
	// KunciPendidik mengunci pendidik milik pengajuan (tanpa scope: pengajuannya sudah dicek scope-nya).
	KunciPendidik(ctx context.Context, tx *gorm.DB, id uuid.UUID) (entities.Pendidik, error)
	// UbahPangkat memperbarui pangkat, golongan, dan tmt_golongan di data induk.
	UbahPangkat(ctx context.Context, tx *gorm.DB, pendidikID uuid.UUID, pangkat, golongan string, tmt time.Time) error
	// CalonKP: PNS dalam scope dengan tmt_golongan <= batas dan tanpa pengajuan KP yang masih berjalan.
	CalonKP(ctx context.Context, actor entities.User, batas time.Time) ([]entities.Pendidik, error)
	StafSudinSekolah(ctx context.Context, tx *gorm.DB, sekolahID uuid.UUID) ([]uuid.UUID, error)
	OperatorSekolah(ctx context.Context, tx *gorm.DB, sekolahID uuid.UUID) ([]uuid.UUID, error)
}

type kpRepository struct {
	db *gorm.DB
}

func NewKenaikanPangkatRepository(db *gorm.DB) KenaikanPangkatRepository {
	return &kpRepository{db: db}
}

func (r *kpRepository) tx(tx *gorm.DB) *gorm.DB {
	if tx == nil {
		return r.db
	}
	return tx
}

func (r *kpRepository) Create(ctx context.Context, tx *gorm.DB, p *entities.KenaikanPangkat) error {
	return r.tx(tx).WithContext(ctx).Omit(clause.Associations).Create(p).Error
}

func (r *kpRepository) Update(ctx context.Context, tx *gorm.DB, p *entities.KenaikanPangkat) error {
	return r.tx(tx).WithContext(ctx).Omit(clause.Associations).Save(p).Error
}

func (r *kpRepository) SaveBerkas(ctx context.Context, tx *gorm.DB, b *entities.KenaikanPangkatBerkas) error {
	return r.tx(tx).WithContext(ctx).Omit(clause.Associations).Save(b).Error
}

func (r *kpRepository) AddRiwayat(ctx context.Context, tx *gorm.DB, h *entities.KenaikanPangkatRiwayat) error {
	return r.tx(tx).WithContext(ctx).Omit(clause.Associations).Create(h).Error
}

func (r *kpRepository) GetForUpdate(ctx context.Context, tx *gorm.DB, actor entities.User, id string) (entities.KenaikanPangkat, error) {
	var p entities.KenaikanPangkat
	if _, err := uuid.Parse(id); err != nil {
		return p, gorm.ErrRecordNotFound
	}
	err := r.tx(tx).WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE", Table: clause.Table{Name: "kenaikan_pangkat"}}).
		Scopes(akses.Scope(actor, "kenaikan_pangkat.sekolah_id")).
		Preload("Pendidik").Preload("Sekolah").Preload("Berkas").
		Take(&p, "kenaikan_pangkat.id = ?", id).Error
	return p, err
}

func (r *kpRepository) GetDetail(ctx context.Context, actor entities.User, id string) (entities.KenaikanPangkat, error) {
	var p entities.KenaikanPangkat
	if _, err := uuid.Parse(id); err != nil {
		return p, gorm.ErrRecordNotFound
	}
	err := r.db.WithContext(ctx).
		Scopes(akses.Scope(actor, "kenaikan_pangkat.sekolah_id")).
		Preload("Pendidik").Preload("Sekolah").Preload("BuktiKirim").Preload("SK").
		Preload("Berkas").Preload("Berkas.Dokumen").
		Preload("Riwayat", func(db *gorm.DB) *gorm.DB { return db.Order("created_at ASC") }).
		Preload("Riwayat.Pengguna").
		Take(&p, "kenaikan_pangkat.id = ?", id).Error
	return p, err
}

func (r *kpRepository) List(ctx context.Context, actor entities.User, f dto.Filter) ([]entities.KenaikanPangkat, int64, error) {
	q := r.db.WithContext(ctx).Model(&entities.KenaikanPangkat{}).Scopes(akses.Scope(actor, "kenaikan_pangkat.sekolah_id"))
	if len(f.Status) > 0 {
		q = q.Where("kenaikan_pangkat.status IN ?", f.Status)
	}
	if f.SekolahID != "" {
		if _, err := uuid.Parse(f.SekolahID); err != nil {
			return nil, 0, nil
		}
		q = q.Where("kenaikan_pangkat.sekolah_id = ?", f.SekolahID)
	}
	if f.Periode != "" {
		periode, err := time.Parse("2006-01", f.Periode)
		if err != nil {
			return nil, 0, nil
		}
		q = q.Where("kenaikan_pangkat.periode = ?", periode)
	}
	if f.Search != "" {
		q = q.Joins("JOIN pendidik ON pendidik.id = kenaikan_pangkat.pendidik_id").
			Where("(kenaikan_pangkat.nomor ILIKE ? OR pendidik.nama ILIKE ? OR pendidik.nip LIKE ?)",
				"%"+f.Search+"%", "%"+f.Search+"%", f.Search+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []entities.KenaikanPangkat
	err := q.Preload("Pendidik").Preload("Sekolah").Preload("Berkas").
		Order("kenaikan_pangkat.created_at DESC").Offset(f.Offset).Limit(f.Limit).Find(&items).Error
	return items, total, err
}

func (r *kpRepository) Ringkasan(ctx context.Context, actor entities.User) (map[string]int64, error) {
	var rows []struct {
		Status string
		Jumlah int64
	}
	err := r.db.WithContext(ctx).Model(&entities.KenaikanPangkat{}).
		Scopes(akses.Scope(actor, "kenaikan_pangkat.sekolah_id")).
		Select("status, COUNT(*) AS jumlah").Group("status").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := map[string]int64{
		entities.KPStatusDiajukan: 0, entities.KPStatusPerluPerbaikan: 0,
		entities.KPStatusTerverifikasi: 0, entities.KPStatusDiusulkanKeBKD: 0,
		entities.KPStatusSelesai: 0,
	}
	for _, row := range rows {
		out[row.Status] = row.Jumlah
	}
	return out, nil
}

func (r *kpRepository) GetPendidik(ctx context.Context, tx *gorm.DB, actor entities.User, id string, kunci bool) (entities.Pendidik, error) {
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

func (r *kpRepository) NomorPengajuan(ctx context.Context, tx *gorm.DB, pendidikID uuid.UUID) (string, error) {
	var nomor []string
	err := r.tx(tx).WithContext(ctx).Model(&entities.KenaikanPangkat{}).
		Where("pendidik_id = ? AND status <> ?", pendidikID, entities.KPStatusSelesai).
		Order("created_at DESC").Limit(1).Pluck("nomor", &nomor).Error
	if err != nil || len(nomor) == 0 {
		return "", err
	}
	return nomor[0], nil
}

func (r *kpRepository) KunciPendidik(ctx context.Context, tx *gorm.DB, id uuid.UUID) (entities.Pendidik, error) {
	var p entities.Pendidik
	err := r.tx(tx).WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE", Table: clause.Table{Name: "pendidik"}}).
		Take(&p, "pendidik.id = ?", id).Error
	return p, err
}

func (r *kpRepository) UbahPangkat(ctx context.Context, tx *gorm.DB, pendidikID uuid.UUID, pangkat, golongan string, tmt time.Time) error {
	return r.tx(tx).WithContext(ctx).Model(&entities.Pendidik{}).Where("id = ?", pendidikID).
		Updates(map[string]any{"pangkat": pangkat, "golongan": golongan, "tmt_golongan": tmt, "updated_at": time.Now()}).Error
}

func (r *kpRepository) CalonKP(ctx context.Context, actor entities.User, batas time.Time) ([]entities.Pendidik, error) {
	var items []entities.Pendidik
	err := r.db.WithContext(ctx).
		Scopes(akses.Scope(actor, "pendidik.sekolah_id")).
		Where("pendidik.status_kepegawaian = ? AND pendidik.tmt_golongan <= ?", entities.StatusKepegawaianPNS, batas).
		Where("NOT EXISTS (SELECT 1 FROM kenaikan_pangkat k WHERE k.pendidik_id = pendidik.id AND k.status <> ?)", entities.KPStatusSelesai).
		Preload("Sekolah").Order("pendidik.tmt_golongan ASC, pendidik.nama ASC").Limit(500).Find(&items).Error
	return items, err
}

func (r *kpRepository) StafSudinSekolah(ctx context.Context, tx *gorm.DB, sekolahID uuid.UUID) ([]uuid.UUID, error) {
	return penerima.StafSudinSekolah(ctx, r.tx(tx), sekolahID)
}

func (r *kpRepository) OperatorSekolah(ctx context.Context, tx *gorm.DB, sekolahID uuid.UUID) ([]uuid.UUID, error) {
	return penerima.OperatorSekolah(ctx, r.tx(tx), sekolahID)
}
