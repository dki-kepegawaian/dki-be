package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Status Anjab per sekolah dalam satu periode (Kamus Status: Draf, Dikirim, Dikembalikan, Ditetapkan).
const (
	AnjabStatusDraf         = "draf"
	AnjabStatusDikirim      = "dikirim"
	AnjabStatusDikembalikan = "dikembalikan"
	AnjabStatusDitetapkan   = "ditetapkan"
)

// AnjabPeriode adalah satu periode analisis jabatan milik satu Sudin (A1).
// Parameter perhitungan terkunci setelah DibukaAt terisi (A3).
type AnjabPeriode struct {
	ID              uuid.UUID  `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	SudinID         uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex:uq_anjab_periode_sudin_tahun" json:"sudin_id"`
	TahunAjaran     string     `gorm:"type:varchar(9);not null;uniqueIndex:uq_anjab_periode_sudin_tahun" json:"tahun_ajaran"`
	Jenjang         string     `gorm:"type:varchar(30);not null" json:"jenjang"` // dipisah koma, mis. "SD,SMP"
	TanggalBuka     time.Time  `gorm:"type:date;not null" json:"tanggal_buka"`
	TanggalTutup    time.Time  `gorm:"type:date;not null" json:"tanggal_tutup"`
	AcuanPeraturan  string     `gorm:"type:varchar(200)" json:"acuan_peraturan"`
	JamWajib        int        `gorm:"not null" json:"jam_wajib"`
	RasioSD         int        `gorm:"column:rasio_sd;not null" json:"rasio_sd"`
	RasioSMP        int        `gorm:"column:rasio_smp;not null" json:"rasio_smp"`
	RasioSMA        int        `gorm:"column:rasio_sma;not null" json:"rasio_sma"` // SMA dan SMK
	BatasTMTPensiun time.Time  `gorm:"column:batas_tmt_pensiun;type:date;not null" json:"batas_tmt_pensiun"`
	DibuatOleh      uuid.UUID  `gorm:"type:uuid;not null" json:"dibuat_oleh"`
	DibukaOleh      *uuid.UUID `gorm:"type:uuid" json:"dibuka_oleh"`
	DibukaAt        *time.Time `gorm:"type:timestamp with time zone" json:"dibuka_at"`

	Sudin *Sudin       `gorm:"foreignKey:SudinID;constraint:OnDelete:RESTRICT" json:"-"`
	Beban []AnjabBeban `gorm:"foreignKey:PeriodeID" json:"-"`

	Timestamp
}

func (AnjabPeriode) TableName() string { return "anjab_periode" }

func (p *AnjabPeriode) BeforeCreate(_ *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// AnjabBeban: beban JP per rombel per minggu untuk satu mapel di satu tingkat kelas.
type AnjabBeban struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	PeriodeID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:uq_anjab_beban" json:"periode_id"`
	Jenjang   string    `gorm:"type:varchar(10);not null;uniqueIndex:uq_anjab_beban" json:"jenjang"`
	Mapel     string    `gorm:"type:varchar(100);not null;uniqueIndex:uq_anjab_beban" json:"mapel"`
	Tingkat   int       `gorm:"not null;uniqueIndex:uq_anjab_beban" json:"tingkat"`
	JP        int       `gorm:"column:jp;not null" json:"jp"`

	Periode *AnjabPeriode `gorm:"foreignKey:PeriodeID;constraint:OnDelete:CASCADE" json:"-"`
}

func (AnjabBeban) TableName() string { return "anjab_beban" }

func (b *AnjabBeban) BeforeCreate(_ *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

// AnjabSekolah adalah data Anjab satu sekolah dalam satu periode; pemegang status.
type AnjabSekolah struct {
	ID           uuid.UUID  `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	PeriodeID    uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex:uq_anjab_sekolah_periode" json:"periode_id"`
	SekolahID    uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex:uq_anjab_sekolah_periode;index" json:"sekolah_id"`
	Status       string     `gorm:"type:varchar(20);not null;index" json:"status"`
	KirimanKe    int        `gorm:"not null;default:0" json:"kiriman_ke"`
	DikirimOleh  *uuid.UUID `gorm:"type:uuid" json:"dikirim_oleh"`
	DikirimAt    *time.Time `gorm:"type:timestamp with time zone" json:"dikirim_at"`
	CatatanSudin string     `gorm:"type:text" json:"catatan_sudin"`
	DitinjauOleh *uuid.UUID `gorm:"type:uuid" json:"ditinjau_oleh"`
	DitinjauAt   *time.Time `gorm:"type:timestamp with time zone" json:"ditinjau_at"`
	// Hasil hitungan (JSON) yang dibekukan saat ditetapkan.
	Hasil string `gorm:"type:text" json:"-"`

	Periode *AnjabPeriode  `gorm:"foreignKey:PeriodeID;constraint:OnDelete:RESTRICT" json:"-"`
	Sekolah *Sekolah       `gorm:"foreignKey:SekolahID;constraint:OnDelete:RESTRICT" json:"-"`
	Rombel  []AnjabRombel  `gorm:"foreignKey:AnjabSekolahID" json:"-"`
	Guru    []AnjabGuru    `gorm:"foreignKey:AnjabSekolahID" json:"-"`
	Riwayat []AnjabRiwayat `gorm:"foreignKey:AnjabSekolahID" json:"-"`

	Timestamp
}

func (AnjabSekolah) TableName() string { return "anjab_sekolah" }

func (a *AnjabSekolah) BeforeCreate(_ *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

// AnjabRombel: jumlah rombel dan siswa per tingkat kelas.
type AnjabRombel struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	AnjabSekolahID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:uq_anjab_rombel" json:"anjab_sekolah_id"`
	SekolahID      uuid.UUID `gorm:"type:uuid;not null;index" json:"sekolah_id"`
	Tingkat        int       `gorm:"not null;uniqueIndex:uq_anjab_rombel" json:"tingkat"`
	JumlahRombel   int       `gorm:"not null" json:"jumlah_rombel"`
	JumlahSiswa    int       `gorm:"not null" json:"jumlah_siswa"`

	AnjabSekolah *AnjabSekolah `gorm:"foreignKey:AnjabSekolahID;constraint:OnDelete:CASCADE" json:"-"`
}

func (AnjabRombel) TableName() string { return "anjab_rombel" }

func (r *AnjabRombel) BeforeCreate(_ *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

// AnjabGuru: guru yang dihitung di sekolah pada periode itu, dengan mapel SK dan mapel riil.
type AnjabGuru struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	AnjabSekolahID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:uq_anjab_guru" json:"anjab_sekolah_id"`
	SekolahID      uuid.UUID `gorm:"type:uuid;not null;index" json:"sekolah_id"`
	PendidikID     uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:uq_anjab_guru;index" json:"pendidik_id"`
	MapelSK        string    `gorm:"column:mapel_sk;type:varchar(100);not null" json:"mapel_sk"`
	MapelRiil      string    `gorm:"type:varchar(100)" json:"mapel_riil"`
	JPRiil         int       `gorm:"column:jp_riil;not null;default:0" json:"jp_riil"`

	AnjabSekolah *AnjabSekolah `gorm:"foreignKey:AnjabSekolahID;constraint:OnDelete:CASCADE" json:"-"`
	Pendidik     *Pendidik     `gorm:"foreignKey:PendidikID;constraint:OnDelete:RESTRICT" json:"-"`

	Timestamp
}

func (AnjabGuru) TableName() string { return "anjab_guru" }

func (g *AnjabGuru) BeforeCreate(_ *gorm.DB) error {
	if g.ID == uuid.Nil {
		g.ID = uuid.New()
	}
	return nil
}

type AnjabRiwayat struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	AnjabSekolahID uuid.UUID `gorm:"type:uuid;not null;index" json:"anjab_sekolah_id"`
	DariStatus     string    `gorm:"type:varchar(20)" json:"dari_status"`
	KeStatus       string    `gorm:"type:varchar(20);not null" json:"ke_status"`
	Oleh           uuid.UUID `gorm:"type:uuid;not null" json:"oleh"`
	Catatan        string    `gorm:"type:text" json:"catatan"`
	CreatedAt      time.Time `gorm:"type:timestamp with time zone" json:"created_at"`

	AnjabSekolah *AnjabSekolah `gorm:"foreignKey:AnjabSekolahID;constraint:OnDelete:CASCADE" json:"-"`
	Pengguna     *User         `gorm:"foreignKey:Oleh" json:"-"`
}

func (AnjabRiwayat) TableName() string { return "anjab_riwayat" }

func (r *AnjabRiwayat) BeforeCreate(_ *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
