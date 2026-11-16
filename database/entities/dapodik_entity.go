package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	DapodikStatusBaru              = "baru"
	DapodikStatusPerluInfoTambahan = "perlu_info_tambahan"
	DapodikStatusSelesai           = "selesai"

	DapodikJenisRevisiAkunOperator = "revisi_akun_operator"
	DapodikJenisAutentikasiPTK     = "autentikasi_ptk"
	DapodikJenisKodeReferral       = "kode_referral_yayasan"
	DapodikJenisPerbaikanDataPTK   = "perbaikan_data_ptk"
	DapodikJenisLainnya            = "lainnya"

	DapodikTenggatHariKerja = 3 // target tinjau sejak tiket (ulang) masuk

	DapodikLampiranPengajuan    = "pengajuan"
	DapodikLampiranInfoTambahan = "info_tambahan"
)

// DapodikTiket adalah tiket layanan Dapodik yang diajukan operator sekolah dan
// ditinjau staf Sudin (admin Dapodik). Pemrosesan di aplikasi Dapodik terjadi di luar sistem.
type DapodikTiket struct {
	ID           uuid.UUID  `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Nomor        string     `gorm:"type:varchar(20);uniqueIndex;not null" json:"nomor"`
	SekolahID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"sekolah_id"`
	DiajukanOleh uuid.UUID  `gorm:"type:uuid;not null" json:"diajukan_oleh"`
	Jenis        string     `gorm:"type:varchar(30);not null;index" json:"jenis"`
	PendidikID   *uuid.UUID `gorm:"type:uuid;index" json:"pendidik_id"` // PTK terkait, bila ada
	// Isian adalah field formulir yang berbeda per jenis layanan (mis. nama_operator_baru).
	Isian   map[string]string `gorm:"type:jsonb;serializer:json;not null" json:"isian"`
	Uraian  string            `gorm:"type:text;not null" json:"uraian"`
	Status  string            `gorm:"type:varchar(30);not null;index" json:"status"`
	Tenggat time.Time         `gorm:"type:date;not null;index" json:"tenggat"`

	PermintaanInfo      string     `gorm:"type:text" json:"permintaan_info"`
	BalasanInfo         string     `gorm:"type:text" json:"balasan_info"`
	CatatanPenyelesaian string     `gorm:"type:text" json:"catatan_penyelesaian"`
	DiselesaikanOleh    *uuid.UUID `gorm:"type:uuid" json:"diselesaikan_oleh"`
	DiselesaikanAt      *time.Time `gorm:"type:timestamp with time zone" json:"diselesaikan_at"`

	Sekolah    *Sekolah          `gorm:"foreignKey:SekolahID;constraint:OnDelete:RESTRICT" json:"sekolah,omitempty"`
	Pendidik   *Pendidik         `gorm:"foreignKey:PendidikID;constraint:OnDelete:RESTRICT" json:"pendidik,omitempty"`
	Pengaju    *User             `gorm:"foreignKey:DiajukanOleh" json:"-"`
	Penyelesai *User             `gorm:"foreignKey:DiselesaikanOleh" json:"-"`
	Lampiran   []DapodikLampiran `gorm:"foreignKey:TiketID" json:"lampiran,omitempty"`
	Riwayat    []DapodikRiwayat  `gorm:"foreignKey:TiketID" json:"riwayat,omitempty"`

	Timestamp
}

func (DapodikTiket) TableName() string { return "dapodik_tiket" }

func (t *DapodikTiket) BeforeCreate(_ *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

// DapodikLampiran menautkan dokumen ke tiket. Tahap membedakan lampiran awal
// dan dokumen pendukung yang dikirim saat melengkapi info.
type DapodikLampiran struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	TiketID   uuid.UUID `gorm:"type:uuid;not null;index" json:"tiket_id"`
	DokumenID uuid.UUID `gorm:"type:uuid;not null" json:"dokumen_id"`
	Tahap     string    `gorm:"type:varchar(20);not null" json:"tahap"`
	CreatedAt time.Time `gorm:"type:timestamp with time zone" json:"created_at"`

	Tiket   *DapodikTiket `gorm:"foreignKey:TiketID;constraint:OnDelete:CASCADE" json:"-"`
	Dokumen *Dokumen      `gorm:"foreignKey:DokumenID;constraint:OnDelete:RESTRICT" json:"dokumen,omitempty"`
}

func (DapodikLampiran) TableName() string { return "dapodik_lampiran" }

func (l *DapodikLampiran) BeforeCreate(_ *gorm.DB) error {
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	return nil
}

type DapodikRiwayat struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	TiketID    uuid.UUID `gorm:"type:uuid;not null;index" json:"tiket_id"`
	DariStatus string    `gorm:"type:varchar(30)" json:"dari_status"`
	KeStatus   string    `gorm:"type:varchar(30);not null" json:"ke_status"`
	Oleh       uuid.UUID `gorm:"type:uuid;not null" json:"oleh"`
	Catatan    string    `gorm:"type:text" json:"catatan"`
	CreatedAt  time.Time `gorm:"type:timestamp with time zone" json:"created_at"`

	Tiket    *DapodikTiket `gorm:"foreignKey:TiketID;constraint:OnDelete:CASCADE" json:"-"`
	Pengguna *User         `gorm:"foreignKey:Oleh" json:"-"`
}

func (DapodikRiwayat) TableName() string { return "dapodik_riwayat" }

func (r *DapodikRiwayat) BeforeCreate(_ *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
