package entities

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

const (
	StatusKepegawaianPNS    = "PNS"
	StatusKepegawaianPPPK   = "PPPK"
	StatusKepegawaianPPPKPW = "PPPK_PW"
	StatusKepegawaianKKI    = "KKI"
)

type Pendidik struct {
	ID                uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	SekolahID         uuid.UUID `gorm:"type:uuid;not null;index" json:"sekolah_id"`
	Nama              string    `gorm:"type:varchar(150);not null" json:"nama"`
	NIP               *string   `gorm:"column:nip;type:varchar(18);uniqueIndex" json:"nip"`
	NIK               string    `gorm:"column:nik;type:varchar(16)" json:"nik"`
	NUPTK             string    `gorm:"column:nuptk;type:varchar(16)" json:"nuptk"`
	NRK               string    `gorm:"column:nrk;type:varchar(20)" json:"nrk"`
	NIKKI             string    `gorm:"column:nikki;type:varchar(30)" json:"nikki"`
	StatusKepegawaian string    `gorm:"type:varchar(10);not null;index" json:"status_kepegawaian"`
	Pangkat           string    `gorm:"type:varchar(50)" json:"pangkat"`
	Golongan          string    `gorm:"type:varchar(10)" json:"golongan"`
	Jabatan           string    `gorm:"type:varchar(150)" json:"jabatan"`
	JabatanDapodik    string    `gorm:"type:varchar(150)" json:"jabatan_dapodik"`
	ProgramStudi      string    `gorm:"type:varchar(150)" json:"program_studi"`
	Sertifikasi       string    `gorm:"type:varchar(150)" json:"sertifikasi"`
	Email             string    `gorm:"type:varchar(255)" json:"email"`
	Telepon           string    `gorm:"type:varchar(20)" json:"telepon"`
	AlamatDomisili    string    `gorm:"type:text" json:"alamat_domisili"`
	KotaDomisili      string    `gorm:"type:varchar(100)" json:"kota_domisili"`
	// Untuk TMT pensiun (batas usia) dan masa kerja; null bila belum diisi.
	TanggalLahir *time.Time `gorm:"type:date" json:"tanggal_lahir"`
	TMTCPNS      *time.Time `gorm:"column:tmt_cpns;type:date" json:"tmt_cpns"`
	// TMT golongan sekarang, untuk syarat 4 tahun kenaikan pangkat; null bila belum diisi.
	TMTGolongan *time.Time `gorm:"column:tmt_golongan;type:date" json:"tmt_golongan"`

	Sekolah *Sekolah `gorm:"foreignKey:SekolahID;constraint:OnDelete:RESTRICT" json:"sekolah,omitempty"`

	Timestamp
}

func (Pendidik) TableName() string { return "pendidik" }

func (p *Pendidik) BeforeCreate(_ *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}
