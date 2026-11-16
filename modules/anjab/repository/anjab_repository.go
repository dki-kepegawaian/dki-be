package repository

import (
	"context"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"github.com/Caknoooo/go-gin-clean-starter/modules/anjab/dto"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/akses"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/constants"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/penerima"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AnjabRepository interface {
	// Periode (milik Sudin)
	CreatePeriode(ctx context.Context, tx *gorm.DB, p *entities.AnjabPeriode) error
	UpdatePeriode(ctx context.Context, tx *gorm.DB, p *entities.AnjabPeriode) error
	ReplaceBeban(ctx context.Context, tx *gorm.DB, periodeID uuid.UUID, beban []entities.AnjabBeban) error
	// GetPeriode mengambil periode dalam scope actor; kunci=true menambah FOR UPDATE.
	GetPeriode(ctx context.Context, tx *gorm.DB, actor entities.User, id string, kunci bool) (entities.AnjabPeriode, error)
	ListPeriode(ctx context.Context, actor entities.User, offset, limit int) ([]entities.AnjabPeriode, int64, error)
	Progres(ctx context.Context, tx *gorm.DB, periodeID uuid.UUID) (map[string]int64, error)

	// Sekolah dalam Sudin
	SekolahSudin(ctx context.Context, tx *gorm.DB, sudinID uuid.UUID, jenjang []string) ([]entities.Sekolah, error)

	// Data Anjab per sekolah
	GetAnjabSekolahByPeriode(ctx context.Context, tx *gorm.DB, periodeID, sekolahID uuid.UUID) (entities.AnjabSekolah, error)
	CreateAnjabSekolah(ctx context.Context, tx *gorm.DB, a *entities.AnjabSekolah) error
	UpdateAnjabSekolah(ctx context.Context, tx *gorm.DB, a *entities.AnjabSekolah) error
	AddRiwayat(ctx context.Context, tx *gorm.DB, r *entities.AnjabRiwayat) error
	// GetForUpdate mengunci baris anjab_sekolah (SELECT ... FOR UPDATE) dalam scope actor.
	GetForUpdate(ctx context.Context, tx *gorm.DB, actor entities.User, id string) (entities.AnjabSekolah, error)
	GetDetail(ctx context.Context, tx *gorm.DB, actor entities.User, id string) (entities.AnjabSekolah, error)
	List(ctx context.Context, actor entities.User, f dto.Filter) ([]entities.AnjabSekolah, int64, error)
	ListByPeriode(ctx context.Context, tx *gorm.DB, periodeID uuid.UUID, status []string) ([]entities.AnjabSekolah, error)
	Hitungan(ctx context.Context, ids []uuid.UUID) (rombel map[uuid.UUID]int, guru map[uuid.UUID]int, err error)

	ReplaceRombel(ctx context.Context, tx *gorm.DB, a entities.AnjabSekolah, rows []entities.AnjabRombel) error
	GetGuru(ctx context.Context, tx *gorm.DB, anjabSekolahID uuid.UUID, id string) (entities.AnjabGuru, error)
	SaveGuru(ctx context.Context, tx *gorm.DB, g *entities.AnjabGuru) error
	DeleteGuru(ctx context.Context, tx *gorm.DB, g entities.AnjabGuru) error
	ReplaceGuru(ctx context.Context, tx *gorm.DB, a entities.AnjabSekolah, rows []entities.AnjabGuru) error

	// Pendidik (tabel fondasi) untuk impor dan tambah guru
	PendidikDiSekolah(ctx context.Context, tx *gorm.DB, id string, sekolahID uuid.UUID) (entities.Pendidik, error)
	PendidikByNIP(ctx context.Context, tx *gorm.DB, nips []string) ([]entities.Pendidik, error)
	PendidikTanpaNIP(ctx context.Context, tx *gorm.DB, sekolahIDs []uuid.UUID) ([]entities.Pendidik, error)
	SavePendidik(ctx context.Context, tx *gorm.DB, p *entities.Pendidik) error

	OperatorSekolah(ctx context.Context, tx *gorm.DB, sekolahID uuid.UUID) ([]uuid.UUID, error)
	StafSudinSekolah(ctx context.Context, tx *gorm.DB, sekolahID uuid.UUID) ([]uuid.UUID, error)
}

type anjabRepository struct {
	db *gorm.DB
}

func NewAnjabRepository(db *gorm.DB) AnjabRepository {
	return &anjabRepository{db: db}
}

func (r *anjabRepository) tx(tx *gorm.DB) *gorm.DB {
	if tx == nil {
		return r.db
	}
	return tx
}

// scopePeriode: periode milik Sudin actor. Operator melihat periode Sudin sekolahnya.
func scopePeriode(actor entities.User) func(*gorm.DB) *gorm.DB {
	return func(q *gorm.DB) *gorm.DB {
		switch actor.Role {
		case constants.ENUM_ROLE_ADMIN:
			return q
		case constants.ENUM_ROLE_STAF_SUDIN, constants.ENUM_ROLE_PIMPINAN:
			if actor.SudinID == nil {
				return q.Where("1 = 0")
			}
			return q.Where("anjab_periode.sudin_id = ?", *actor.SudinID)
		case constants.ENUM_ROLE_OPERATOR_SEKOLAH:
			if actor.SekolahID == nil {
				return q.Where("1 = 0")
			}
			return q.Where("anjab_periode.sudin_id = (SELECT sudin_id FROM sekolah WHERE id = ?)", *actor.SekolahID)
		default:
			return q.Where("1 = 0")
		}
	}
}

func bebanUrut(db *gorm.DB) *gorm.DB { return db.Order("jenjang, mapel, tingkat") }

func (r *anjabRepository) CreatePeriode(ctx context.Context, tx *gorm.DB, p *entities.AnjabPeriode) error {
	return r.tx(tx).WithContext(ctx).Omit(clause.Associations).Create(p).Error
}

func (r *anjabRepository) UpdatePeriode(ctx context.Context, tx *gorm.DB, p *entities.AnjabPeriode) error {
	return r.tx(tx).WithContext(ctx).Omit(clause.Associations).Save(p).Error
}

func (r *anjabRepository) ReplaceBeban(ctx context.Context, tx *gorm.DB, periodeID uuid.UUID, beban []entities.AnjabBeban) error {
	db := r.tx(tx).WithContext(ctx)
	if err := db.Where("periode_id = ?", periodeID).Delete(&entities.AnjabBeban{}).Error; err != nil {
		return err
	}
	if len(beban) == 0 {
		return nil
	}
	return db.Omit(clause.Associations).Create(&beban).Error
}

func (r *anjabRepository) GetPeriode(ctx context.Context, tx *gorm.DB, actor entities.User, id string, kunci bool) (entities.AnjabPeriode, error) {
	var p entities.AnjabPeriode
	if _, err := uuid.Parse(id); err != nil {
		return p, gorm.ErrRecordNotFound
	}
	q := r.tx(tx).WithContext(ctx)
	if kunci {
		q = q.Clauses(clause.Locking{Strength: "UPDATE", Table: clause.Table{Name: "anjab_periode"}})
	}
	err := q.Scopes(scopePeriode(actor)).Preload("Beban", bebanUrut).Take(&p, "anjab_periode.id = ?", id).Error
	return p, err
}

func (r *anjabRepository) ListPeriode(ctx context.Context, actor entities.User, offset, limit int) ([]entities.AnjabPeriode, int64, error) {
	q := r.db.WithContext(ctx).Model(&entities.AnjabPeriode{}).Scopes(scopePeriode(actor))
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []entities.AnjabPeriode
	err := q.Order("tahun_ajaran DESC, created_at DESC").Offset(offset).Limit(limit).Find(&items).Error
	return items, total, err
}

func (r *anjabRepository) Progres(ctx context.Context, tx *gorm.DB, periodeID uuid.UUID) (map[string]int64, error) {
	var rows []struct {
		Status string
		Jumlah int64
	}
	err := r.tx(tx).WithContext(ctx).Model(&entities.AnjabSekolah{}).
		Where("periode_id = ?", periodeID).
		Select("status, COUNT(*) AS jumlah").Group("status").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := map[string]int64{
		entities.AnjabStatusDraf: 0, entities.AnjabStatusDikirim: 0,
		entities.AnjabStatusDikembalikan: 0, entities.AnjabStatusDitetapkan: 0, "total": 0,
	}
	for _, row := range rows {
		out[row.Status] = row.Jumlah
		out["total"] += row.Jumlah
	}
	return out, nil
}

func (r *anjabRepository) SekolahSudin(ctx context.Context, tx *gorm.DB, sudinID uuid.UUID, jenjang []string) ([]entities.Sekolah, error) {
	var out []entities.Sekolah
	err := r.tx(tx).WithContext(ctx).Where("sudin_id = ? AND jenjang IN ?", sudinID, jenjang).Order("nama").Find(&out).Error
	return out, err
}

func (r *anjabRepository) GetAnjabSekolahByPeriode(ctx context.Context, tx *gorm.DB, periodeID, sekolahID uuid.UUID) (entities.AnjabSekolah, error) {
	var a entities.AnjabSekolah
	err := r.tx(tx).WithContext(ctx).Take(&a, "periode_id = ? AND sekolah_id = ?", periodeID, sekolahID).Error
	return a, err
}

func (r *anjabRepository) CreateAnjabSekolah(ctx context.Context, tx *gorm.DB, a *entities.AnjabSekolah) error {
	return r.tx(tx).WithContext(ctx).Omit(clause.Associations).Create(a).Error
}

func (r *anjabRepository) UpdateAnjabSekolah(ctx context.Context, tx *gorm.DB, a *entities.AnjabSekolah) error {
	return r.tx(tx).WithContext(ctx).Omit(clause.Associations).Save(a).Error
}

func (r *anjabRepository) AddRiwayat(ctx context.Context, tx *gorm.DB, h *entities.AnjabRiwayat) error {
	return r.tx(tx).WithContext(ctx).Omit(clause.Associations).Create(h).Error
}

func (r *anjabRepository) GetForUpdate(ctx context.Context, tx *gorm.DB, actor entities.User, id string) (entities.AnjabSekolah, error) {
	var a entities.AnjabSekolah
	if _, err := uuid.Parse(id); err != nil {
		return a, gorm.ErrRecordNotFound
	}
	err := r.tx(tx).WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE", Table: clause.Table{Name: "anjab_sekolah"}}).
		Scopes(akses.Scope(actor, "anjab_sekolah.sekolah_id")).
		Preload("Sekolah").Preload("Periode").Preload("Periode.Beban", bebanUrut).
		Take(&a, "anjab_sekolah.id = ?", id).Error
	return a, err
}

func (r *anjabRepository) GetDetail(ctx context.Context, tx *gorm.DB, actor entities.User, id string) (entities.AnjabSekolah, error) {
	var a entities.AnjabSekolah
	if _, err := uuid.Parse(id); err != nil {
		return a, gorm.ErrRecordNotFound
	}
	err := r.tx(tx).WithContext(ctx).
		Scopes(akses.Scope(actor, "anjab_sekolah.sekolah_id")).
		Preload("Sekolah").Preload("Periode").Preload("Periode.Beban", bebanUrut).
		Preload("Rombel", func(db *gorm.DB) *gorm.DB { return db.Order("tingkat") }).
		Preload("Guru", func(db *gorm.DB) *gorm.DB { return db.Order("created_at") }).Preload("Guru.Pendidik").
		Preload("Riwayat", func(db *gorm.DB) *gorm.DB { return db.Order("created_at ASC") }).
		Preload("Riwayat.Pengguna").
		Take(&a, "anjab_sekolah.id = ?", id).Error
	return a, err
}

func (r *anjabRepository) List(ctx context.Context, actor entities.User, f dto.Filter) ([]entities.AnjabSekolah, int64, error) {
	q := r.db.WithContext(ctx).Model(&entities.AnjabSekolah{}).
		Scopes(akses.Scope(actor, "anjab_sekolah.sekolah_id")).
		Joins("JOIN sekolah ON sekolah.id = anjab_sekolah.sekolah_id")
	if f.PeriodeID != "" {
		if _, err := uuid.Parse(f.PeriodeID); err != nil {
			return nil, 0, nil
		}
		q = q.Where("anjab_sekolah.periode_id = ?", f.PeriodeID)
	}
	if len(f.Status) > 0 {
		q = q.Where("anjab_sekolah.status IN ?", f.Status)
	}
	if f.Kecamatan != "" {
		q = q.Where("sekolah.kecamatan = ?", f.Kecamatan)
	}
	if f.Search != "" {
		q = q.Where("(sekolah.nama ILIKE ? OR sekolah.npsn LIKE ?)", "%"+f.Search+"%", f.Search+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []entities.AnjabSekolah
	err := q.Preload("Sekolah").Preload("Periode").
		Order("sekolah.nama ASC").Offset(f.Offset).Limit(f.Limit).Find(&items).Error
	return items, total, err
}

func (r *anjabRepository) ListByPeriode(ctx context.Context, tx *gorm.DB, periodeID uuid.UUID, status []string) ([]entities.AnjabSekolah, error) {
	q := r.tx(tx).WithContext(ctx).Preload("Sekolah").Where("periode_id = ?", periodeID)
	if len(status) > 0 {
		q = q.Where("status IN ?", status)
	}
	var out []entities.AnjabSekolah
	err := q.Find(&out).Error
	return out, err
}

func (r *anjabRepository) Hitungan(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]int, map[uuid.UUID]int, error) {
	type row struct {
		ID     uuid.UUID
		Jumlah int
	}
	rombel, guru := map[uuid.UUID]int{}, map[uuid.UUID]int{}
	if len(ids) == 0 {
		return rombel, guru, nil
	}
	var rows []row
	if err := r.db.WithContext(ctx).Model(&entities.AnjabRombel{}).
		Select("anjab_sekolah_id AS id, COALESCE(SUM(jumlah_rombel), 0) AS jumlah").
		Where("anjab_sekolah_id IN ?", ids).Group("anjab_sekolah_id").Scan(&rows).Error; err != nil {
		return nil, nil, err
	}
	for _, x := range rows {
		rombel[x.ID] = x.Jumlah
	}
	rows = nil
	if err := r.db.WithContext(ctx).Model(&entities.AnjabGuru{}).
		Select("anjab_sekolah_id AS id, COUNT(*) AS jumlah").
		Where("anjab_sekolah_id IN ?", ids).Group("anjab_sekolah_id").Scan(&rows).Error; err != nil {
		return nil, nil, err
	}
	for _, x := range rows {
		guru[x.ID] = x.Jumlah
	}
	return rombel, guru, nil
}

func (r *anjabRepository) ReplaceRombel(ctx context.Context, tx *gorm.DB, a entities.AnjabSekolah, rows []entities.AnjabRombel) error {
	db := r.tx(tx).WithContext(ctx)
	if err := db.Where("anjab_sekolah_id = ?", a.ID).Delete(&entities.AnjabRombel{}).Error; err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	for i := range rows {
		rows[i].AnjabSekolahID, rows[i].SekolahID = a.ID, a.SekolahID
	}
	return db.Omit(clause.Associations).Create(&rows).Error
}

func (r *anjabRepository) GetGuru(ctx context.Context, tx *gorm.DB, anjabSekolahID uuid.UUID, id string) (entities.AnjabGuru, error) {
	var g entities.AnjabGuru
	if _, err := uuid.Parse(id); err != nil {
		return g, gorm.ErrRecordNotFound
	}
	err := r.tx(tx).WithContext(ctx).Take(&g, "id = ? AND anjab_sekolah_id = ?", id, anjabSekolahID).Error
	return g, err
}

func (r *anjabRepository) SaveGuru(ctx context.Context, tx *gorm.DB, g *entities.AnjabGuru) error {
	return r.tx(tx).WithContext(ctx).Omit(clause.Associations).Save(g).Error
}

func (r *anjabRepository) DeleteGuru(ctx context.Context, tx *gorm.DB, g entities.AnjabGuru) error {
	return r.tx(tx).WithContext(ctx).Delete(&entities.AnjabGuru{}, "id = ?", g.ID).Error
}

func (r *anjabRepository) ReplaceGuru(ctx context.Context, tx *gorm.DB, a entities.AnjabSekolah, rows []entities.AnjabGuru) error {
	db := r.tx(tx).WithContext(ctx)
	if err := db.Where("anjab_sekolah_id = ?", a.ID).Delete(&entities.AnjabGuru{}).Error; err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	for i := range rows {
		rows[i].AnjabSekolahID, rows[i].SekolahID = a.ID, a.SekolahID
	}
	return db.Omit(clause.Associations).Create(&rows).Error
}

func (r *anjabRepository) PendidikDiSekolah(ctx context.Context, tx *gorm.DB, id string, sekolahID uuid.UUID) (entities.Pendidik, error) {
	var p entities.Pendidik
	if _, err := uuid.Parse(id); err != nil {
		return p, gorm.ErrRecordNotFound
	}
	err := r.tx(tx).WithContext(ctx).Take(&p, "id = ? AND sekolah_id = ?", id, sekolahID).Error
	return p, err
}

// PendidikByNIP sengaja tanpa scope: dipakai untuk mendeteksi NIP yang sudah terdaftar di Sudin lain.
func (r *anjabRepository) PendidikByNIP(ctx context.Context, tx *gorm.DB, nips []string) ([]entities.Pendidik, error) {
	var out []entities.Pendidik
	if len(nips) == 0 {
		return out, nil
	}
	err := r.tx(tx).WithContext(ctx).Where("nip IN ?", nips).Find(&out).Error
	return out, err
}

func (r *anjabRepository) PendidikTanpaNIP(ctx context.Context, tx *gorm.DB, sekolahIDs []uuid.UUID) ([]entities.Pendidik, error) {
	var out []entities.Pendidik
	if len(sekolahIDs) == 0 {
		return out, nil
	}
	err := r.tx(tx).WithContext(ctx).Where("nip IS NULL AND sekolah_id IN ?", sekolahIDs).Find(&out).Error
	return out, err
}

func (r *anjabRepository) SavePendidik(ctx context.Context, tx *gorm.DB, p *entities.Pendidik) error {
	return r.tx(tx).WithContext(ctx).Omit(clause.Associations).Save(p).Error
}

func (r *anjabRepository) OperatorSekolah(ctx context.Context, tx *gorm.DB, sekolahID uuid.UUID) ([]uuid.UUID, error) {
	return penerima.OperatorSekolah(ctx, r.tx(tx), sekolahID)
}

func (r *anjabRepository) StafSudinSekolah(ctx context.Context, tx *gorm.DB, sekolahID uuid.UUID) ([]uuid.UUID, error) {
	return penerima.StafSudinSekolah(ctx, r.tx(tx), sekolahID)
}
