package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	KinerjaStatusDraf        = "draf"
	KinerjaStatusMenungguTTD = "menunggu_ttd"
	KinerjaStatusFinal       = "final"
)

// KinerjaPenilaian adalah penilaian kinerja satu pegawai PPPK Paruh Waktu untuk satu
// periode semester. Nilai BerAKHLAK berskala 1–5; nil berarti belum diisi.
type KinerjaPenilaian struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Nomor       string    `gorm:"type:varchar(20);uniqueIndex;not null" json:"nomor"`
	PendidikID  uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_kinerja_periode" json:"pendidik_id"`
	Tahun       int       `gorm:"not null;uniqueIndex:idx_kinerja_periode" json:"tahun"`
	Semester    int       `gorm:"not null;uniqueIndex:idx_kinerja_periode" json:"semester"`
	SekolahID   uuid.UUID `gorm:"type:uuid;not null;index" json:"sekolah_id"`
	DinilaiOleh uuid.UUID `gorm:"type:uuid;not null" json:"dinilai_oleh"`
	Status      string    `gorm:"type:varchar(20);not null;index" json:"status"`

	BerorientasiPelayanan *int `json:"berorientasi_pelayanan"`
	Akuntabel             *int `json:"akuntabel"`
	Kompeten              *int `json:"kompeten"`
	Harmonis              *int `json:"harmonis"`
	Loyal                 *int `json:"loyal"`
	Adaptif               *int `json:"adaptif"`
	Kolaboratif           *int `json:"kolaboratif"`

	// NilaiAkhir = rata-rata parameter yang terisi, dibulatkan 1 desimal.
	NilaiAkhir *float64 `gorm:"type:numeric(2,1)" json:"nilai_akhir"`
	Predikat   string   `gorm:"type:varchar(20)" json:"predikat"`

	DokumenSistemID *uuid.UUID `gorm:"type:uuid" json:"dokumen_sistem_id"`
	DokumenDibuatAt *time.Time `gorm:"type:timestamp with time zone" json:"dokumen_dibuat_at"`
	DokumenTTDID    *uuid.UUID `gorm:"column:dokumen_ttd_id;type:uuid" json:"dokumen_ttd_id"`
	DifinalkanOleh  *uuid.UUID `gorm:"type:uuid" json:"difinalkan_oleh"`
	DifinalkanAt    *time.Time `gorm:"type:timestamp with time zone" json:"difinalkan_at"`

	Pendidik      *Pendidik        `gorm:"foreignKey:PendidikID;constraint:OnDelete:RESTRICT" json:"pendidik,omitempty"`
	Sekolah       *Sekolah         `gorm:"foreignKey:SekolahID;constraint:OnDelete:RESTRICT" json:"sekolah,omitempty"`
	Penilai       *User            `gorm:"foreignKey:DinilaiOleh" json:"-"`
	DokumenSistem *Dokumen         `gorm:"foreignKey:DokumenSistemID" json:"dokumen_sistem,omitempty"`
	DokumenTTD    *Dokumen         `gorm:"foreignKey:DokumenTTDID" json:"dokumen_ttd,omitempty"`
	Pemfinal      *User            `gorm:"foreignKey:DifinalkanOleh" json:"-"`
	Riwayat       []KinerjaRiwayat `gorm:"foreignKey:PenilaianID" json:"riwayat,omitempty"`

	Timestamp
}

func (KinerjaPenilaian) TableName() string { return "kinerja_penilaian" }

func (k *KinerjaPenilaian) BeforeCreate(_ *gorm.DB) error {
	if k.ID == uuid.Nil {
		k.ID = uuid.New()
	}
	return nil
}

type KinerjaRiwayat struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	PenilaianID uuid.UUID `gorm:"type:uuid;not null;index" json:"penilaian_id"`
	DariStatus  string    `gorm:"type:varchar(20)" json:"dari_status"`
	KeStatus    string    `gorm:"type:varchar(20);not null" json:"ke_status"`
	Oleh        uuid.UUID `gorm:"type:uuid;not null" json:"oleh"`
	Catatan     string    `gorm:"type:text" json:"catatan"`
	CreatedAt   time.Time `gorm:"type:timestamp with time zone" json:"created_at"`

	Penilaian *KinerjaPenilaian `gorm:"foreignKey:PenilaianID;constraint:OnDelete:CASCADE" json:"-"`
	Pengguna  *User             `gorm:"foreignKey:Oleh" json:"-"`
}

func (KinerjaRiwayat) TableName() string { return "kinerja_riwayat" }

func (r *KinerjaRiwayat) BeforeCreate(_ *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
