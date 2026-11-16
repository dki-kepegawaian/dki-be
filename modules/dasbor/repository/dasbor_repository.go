package repository

import (
	"context"
	"time"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"github.com/Caknoooo/go-gin-clean-starter/modules/dasbor/spek"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/akses"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/constants"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Baris adalah satu pengajuan dari tabel modul mana pun.
type Baris struct {
	ID           uuid.UUID
	Nomor        string
	Status       string
	PendidikNama *string
	SekolahNama  string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Tenggat      *time.Time
}

type Saring struct {
	StatusIn    []string // kosong = tanpa filter
	StatusNotIn []string
	Search      string
	Offset      int
	Limit       int
}

type StatusStat struct {
	Status    string
	Jumlah    int64
	RataDetik *float64 // rata-rata updated_at - created_at
}

type AnjabRingkas struct {
	PeriodeID    uuid.UUID
	TahunAjaran  string
	TanggalTutup time.Time
	PerStatus    map[string]int64
}

type DasborRepository interface {
	Baris(ctx context.Context, actor entities.User, m spek.Modul, s Saring) ([]Baris, int64, error)
	Statistik(ctx context.Context, actor entities.User, m spek.Modul, dari, sampai time.Time) ([]StatusStat, error)
	TrenBulanan(ctx context.Context, actor entities.User, m spek.Modul, dari time.Time) (map[string]int64, error)
	AnjabTugas(ctx context.Context, actor entities.User, status []string, hanyaPeriodeTerbuka bool) ([]Baris, error)
	AnjabTerkini(ctx context.Context, actor entities.User) (*AnjabRingkas, error)
	SudinNama(ctx context.Context, id uuid.UUID) string
	// PenggunaPengingat: akun aktif yang punya daftar tugas (staf Sudin, pimpinan, operator sekolah).
	PenggunaPengingat(ctx context.Context) ([]entities.User, error)
	// DenganKunciPengingat menjalankan fn dalam transaksi yang memegang advisory lock,
	// supaya dua instance aplikasi tidak mengirim pengingat bersamaan. false = instance lain sedang jalan.
	DenganKunciPengingat(ctx context.Context, fn func(tx *gorm.DB) error) (bool, error)
	// BuatPengingat membuat notifikasi kecuali akun itu sudah menerima yang sama (judul + tautan) sejak waktu tersebut.
	BuatPengingat(tx *gorm.DB, userID uuid.UUID, judul, pesan, tautan string, sejak time.Time) (bool, error)
}

type dasborRepository struct {
	db *gorm.DB
}

func NewDasborRepository(db *gorm.DB) DasborRepository {
	return &dasborRepository{db: db}
}

// Nama tabel dan kolom berasal dari spek (konstanta kode), bukan dari input.
func (r *dasborRepository) dasar(ctx context.Context, actor entities.User, m spek.Modul, s Saring) *gorm.DB {
	t := m.Tabel
	q := r.db.WithContext(ctx).Table(t).
		Joins("LEFT JOIN pendidik ON pendidik.id = " + t + ".pendidik_id").
		Joins("JOIN sekolah ON sekolah.id = " + t + ".sekolah_id").
		Scopes(akses.Scope(actor, t+".sekolah_id"))
	if len(s.StatusIn) > 0 {
		q = q.Where(t+".status IN ?", s.StatusIn)
	}
	if len(s.StatusNotIn) > 0 {
		q = q.Where(t+".status NOT IN ?", s.StatusNotIn)
	}
	if s.Search != "" {
		q = q.Where("("+t+".nomor ILIKE ? OR pendidik.nama ILIKE ? OR pendidik.nip LIKE ?)", "%"+s.Search+"%", "%"+s.Search+"%", s.Search+"%")
	}
	return q
}

func (r *dasborRepository) Baris(ctx context.Context, actor entities.User, m spek.Modul, s Saring) ([]Baris, int64, error) {
	var total int64
	if err := r.dasar(ctx, actor, m, s).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	t := m.Tabel
	tenggat := "NULL::date AS tenggat"
	if m.PunyaTenggat {
		tenggat = t + ".tenggat"
	}
	var rows []Baris
	err := r.dasar(ctx, actor, m, s).
		Select(t + ".id, " + t + ".nomor, " + t + ".status, pendidik.nama AS pendidik_nama, sekolah.nama AS sekolah_nama, " +
			t + ".created_at, " + t + ".updated_at, " + tenggat).
		Order(t + ".updated_at DESC").Offset(s.Offset).Limit(s.Limit).Scan(&rows).Error
	return rows, total, err
}

func (r *dasborRepository) Statistik(ctx context.Context, actor entities.User, m spek.Modul, dari, sampai time.Time) ([]StatusStat, error) {
	t := m.Tabel
	var rows []StatusStat
	err := r.db.WithContext(ctx).Table(t).Scopes(akses.Scope(actor, t+".sekolah_id")).
		Where(t+".created_at >= ? AND "+t+".created_at < ?", dari, sampai).
		Select(t + ".status, COUNT(*) AS jumlah, AVG(EXTRACT(EPOCH FROM (" + t + ".updated_at - " + t + ".created_at))) AS rata_detik").
		Group(t + ".status").Scan(&rows).Error
	return rows, err
}

func (r *dasborRepository) TrenBulanan(ctx context.Context, actor entities.User, m spek.Modul, dari time.Time) (map[string]int64, error) {
	t := m.Tabel
	var rows []struct {
		Bulan  string
		Jumlah int64
	}
	err := r.db.WithContext(ctx).Table(t).Scopes(akses.Scope(actor, t+".sekolah_id")).
		Where(t+".created_at >= ?", dari).
		Select("to_char(date_trunc('month', " + t + ".created_at), 'YYYY-MM') AS bulan, COUNT(*) AS jumlah").
		Group("bulan").Scan(&rows).Error
	out := map[string]int64{}
	for _, row := range rows {
		out[row.Bulan] = row.Jumlah
	}
	return out, err
}

// AnjabTugas: data Anjab sekolah berstatus tertentu; nomor = "Anjab <tahun ajaran>",
// tenggat = tanggal tutup periode. hanyaPeriodeTerbuka membatasi ke periode yang sedang dibuka.
func (r *dasborRepository) AnjabTugas(ctx context.Context, actor entities.User, status []string, hanyaPeriodeTerbuka bool) ([]Baris, error) {
	q := r.db.WithContext(ctx).Table("anjab_sekolah").
		Joins("JOIN anjab_periode ON anjab_periode.id = anjab_sekolah.periode_id").
		Joins("JOIN sekolah ON sekolah.id = anjab_sekolah.sekolah_id").
		Scopes(akses.Scope(actor, "anjab_sekolah.sekolah_id")).
		Where("anjab_sekolah.status IN ?", status)
	if hanyaPeriodeTerbuka {
		q = q.Where("anjab_periode.dibuka_at IS NOT NULL AND anjab_periode.tanggal_tutup >= CURRENT_DATE")
	}
	var rows []Baris
	err := q.Select(`anjab_sekolah.id, 'Anjab ' || anjab_periode.tahun_ajaran AS nomor, anjab_sekolah.status,
		NULL AS pendidik_nama, sekolah.nama AS sekolah_nama, anjab_sekolah.created_at, anjab_sekolah.updated_at,
		anjab_periode.tanggal_tutup AS tenggat`).Limit(500).Scan(&rows).Error
	return rows, err
}

func (r *dasborRepository) AnjabTerkini(ctx context.Context, actor entities.User) (*AnjabRingkas, error) {
	var p struct {
		ID           uuid.UUID
		TahunAjaran  string
		TanggalTutup time.Time
	}
	err := r.db.WithContext(ctx).Table("anjab_periode").
		Joins("JOIN anjab_sekolah ON anjab_sekolah.periode_id = anjab_periode.id").
		Scopes(akses.Scope(actor, "anjab_sekolah.sekolah_id")).
		Select("anjab_periode.id, anjab_periode.tahun_ajaran, anjab_periode.tanggal_tutup").
		Order("anjab_periode.tanggal_buka DESC").Limit(1).Scan(&p).Error
	if err != nil || p.ID == uuid.Nil {
		return nil, err
	}
	var rows []struct {
		Status string
		Jumlah int64
	}
	err = r.db.WithContext(ctx).Table("anjab_sekolah").Scopes(akses.Scope(actor, "anjab_sekolah.sekolah_id")).
		Where("anjab_sekolah.periode_id = ?", p.ID).
		Select("anjab_sekolah.status, COUNT(*) AS jumlah").Group("anjab_sekolah.status").Scan(&rows).Error
	out := &AnjabRingkas{PeriodeID: p.ID, TahunAjaran: p.TahunAjaran, TanggalTutup: p.TanggalTutup, PerStatus: map[string]int64{}}
	for _, row := range rows {
		out.PerStatus[row.Status] = row.Jumlah
	}
	return out, err
}

func (r *dasborRepository) SudinNama(ctx context.Context, id uuid.UUID) string {
	var nama string
	r.db.WithContext(ctx).Model(&entities.Sudin{}).Where("id = ?", id).Pluck("nama", &nama)
	return nama
}

func (r *dasborRepository) PenggunaPengingat(ctx context.Context) ([]entities.User, error) {
	var users []entities.User
	err := r.db.WithContext(ctx).
		Where("is_aktif = ? AND role IN ?", true, []string{constants.ENUM_ROLE_STAF_SUDIN, constants.ENUM_ROLE_PIMPINAN, constants.ENUM_ROLE_OPERATOR_SEKOLAH}).
		Find(&users).Error
	return users, err
}

// kunciPengingat adalah kunci advisory lock Postgres untuk pengingat tenggat (angka tetap, sembarang).
const kunciPengingat = 7302611

func (r *dasborRepository) DenganKunciPengingat(ctx context.Context, fn func(tx *gorm.DB) error) (bool, error) {
	dapat := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Raw("SELECT pg_try_advisory_xact_lock(?)", kunciPengingat).Scan(&dapat).Error; err != nil || !dapat {
			return err
		}
		return fn(tx)
	})
	return dapat, err
}

func (r *dasborRepository) BuatPengingat(tx *gorm.DB, userID uuid.UUID, judul, pesan, tautan string, sejak time.Time) (bool, error) {
	var n int64
	if err := tx.Model(&entities.Notifikasi{}).
		Where("user_id = ? AND judul = ? AND tautan = ? AND created_at >= ?", userID, judul, tautan, sejak).
		Count(&n).Error; err != nil || n > 0 {
		return false, err
	}
	return true, tx.Create(&entities.Notifikasi{UserID: userID, Judul: judul, Pesan: pesan, Tautan: tautan}).Error
}
