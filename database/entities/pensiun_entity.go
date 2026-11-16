package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	PensiunStatusDiajukan       = "diajukan"
	PensiunStatusPerluPerbaikan = "perlu_perbaikan"
	PensiunStatusTerverifikasi  = "terverifikasi"
	PensiunStatusDiusulkanKeBKD = "diusulkan_ke_bkd"
	PensiunStatusSelesai        = "selesai"

	PensiunJenisBatasUsia             = "batas_usia"
	PensiunJenisAtasPermintaanSendiri = "atas_permintaan_sendiri"

	PensiunBatasUsia = 60 // tahun, BUP guru
)

// PensiunBerkasWajib adalah 14 berkas persyaratan pensiun (Figma P2), urut seperti di layar.
var PensiunBerkasWajib = []JenisBerkas{
	{"sk_cpns", "SK CPNS", "Kepegawaian"},
	{"sk_pns", "SK PNS", "Kepegawaian"},
	{"sk_kp_terakhir", "SK kenaikan pangkat terakhir", "Kepegawaian"},
	{"karpeg", "Kartu Pegawai (Karpeg)", "Kepegawaian"},
	{"skp_terakhir", "SKP tahun terakhir", "Kepegawaian"},
	{"dpcp", "DPCP (Data Perorangan Calon Penerima Pensiun)", "Kepegawaian"},
	{"ktp", "KTP", "Identitas & keluarga"},
	{"kartu_keluarga", "Kartu Keluarga", "Identitas & keluarga"},
	{"akta_nikah", "Akta nikah", "Identitas & keluarga"},
	{"karsu_karis", "Karsu / Karis", "Identitas & keluarga"},
	{"npwp", "NPWP", "Identitas & keluarga"},
	{"sp_tidak_hukdis", "Surat pernyataan tidak pernah dijatuhi hukuman disiplin", "Surat pernyataan & pengantar"},
	{"sp_tidak_pidana", "Surat pernyataan tidak sedang menjalani proses pidana", "Surat pernyataan & pengantar"},
	{"pengantar_skpd", "Surat pengantar SKPD", "Surat pernyataan & pengantar"},
}

type Pensiun struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Nomor        string    `gorm:"type:varchar(20);uniqueIndex;not null" json:"nomor"`
	PendidikID   uuid.UUID `gorm:"type:uuid;not null;index" json:"pendidik_id"`
	SekolahID    uuid.UUID `gorm:"type:uuid;not null;index" json:"sekolah_id"`
	DiajukanOleh uuid.UUID `gorm:"type:uuid;not null" json:"diajukan_oleh"`
	Jenis        string    `gorm:"type:varchar(30);not null" json:"jenis"`
	TMTPensiun   time.Time `gorm:"column:tmt_pensiun;type:date;not null" json:"tmt_pensiun"`
	Status       string    `gorm:"type:varchar(30);not null;index" json:"status"`

	// P3: verifikasi berkas
	DiverifikasiOleh *uuid.UUID `gorm:"type:uuid" json:"diverifikasi_oleh"`
	DiverifikasiAt   *time.Time `gorm:"type:timestamp with time zone" json:"diverifikasi_at"`

	// P4: pengiriman ke BKD (di luar sistem, dicatat)
	NomorSuratPengantar string     `gorm:"type:varchar(100)" json:"nomor_surat_pengantar"`
	TanggalKirim        *time.Time `gorm:"type:date" json:"tanggal_kirim"`
	CaraKirim           string     `gorm:"type:varchar(100)" json:"cara_kirim"`
	BuktiKirimID        *uuid.UUID `gorm:"type:uuid" json:"bukti_kirim_id"`
	CatatanKirim        string     `gorm:"type:text" json:"catatan_kirim"`
	DikirimOleh         *uuid.UUID `gorm:"type:uuid" json:"dikirim_oleh"`

	// P5: SK dari BKD
	NomorSK     string     `gorm:"column:nomor_sk;type:varchar(100)" json:"nomor_sk"`
	TanggalSK   *time.Time `gorm:"column:tanggal_sk;type:date" json:"tanggal_sk"`
	SKID        *uuid.UUID `gorm:"column:sk_id;type:uuid" json:"sk_id"`
	SelesaiOleh *uuid.UUID `gorm:"type:uuid" json:"selesai_oleh"`
	SelesaiAt   *time.Time `gorm:"type:timestamp with time zone" json:"selesai_at"`

	Pendidik   *Pendidik        `gorm:"foreignKey:PendidikID;constraint:OnDelete:RESTRICT" json:"pendidik,omitempty"`
	Sekolah    *Sekolah         `gorm:"foreignKey:SekolahID;constraint:OnDelete:RESTRICT" json:"sekolah,omitempty"`
	Pengaju    *User            `gorm:"foreignKey:DiajukanOleh" json:"-"`
	BuktiKirim *Dokumen         `gorm:"foreignKey:BuktiKirimID" json:"-"`
	SK         *Dokumen         `gorm:"foreignKey:SKID" json:"-"`
	Berkas     []PensiunBerkas  `gorm:"foreignKey:PensiunID" json:"-"`
	Riwayat    []PensiunRiwayat `gorm:"foreignKey:PensiunID" json:"-"`

	Timestamp
}

func (Pensiun) TableName() string { return "pensiun" }

func (p *Pensiun) BeforeCreate(_ *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

type PensiunBerkas struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	PensiunID uuid.UUID `gorm:"type:uuid;not null;index" json:"pensiun_id"`
	BerkasPersyaratan
	Timestamp

	Pensiun *Pensiun `gorm:"foreignKey:PensiunID;constraint:OnDelete:CASCADE" json:"-"`
	Dokumen *Dokumen `gorm:"foreignKey:DokumenID" json:"-"`
}

func (PensiunBerkas) TableName() string { return "pensiun_berkas" }

func (b *PensiunBerkas) BeforeCreate(_ *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

type PensiunRiwayat struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	PensiunID  uuid.UUID `gorm:"type:uuid;not null;index" json:"pensiun_id"`
	DariStatus string    `gorm:"type:varchar(30)" json:"dari_status"`
	KeStatus   string    `gorm:"type:varchar(30);not null" json:"ke_status"`
	Oleh       uuid.UUID `gorm:"type:uuid;not null" json:"oleh"`
	Catatan    string    `gorm:"type:text" json:"catatan"`
	CreatedAt  time.Time `gorm:"type:timestamp with time zone" json:"created_at"`

	Pensiun  *Pensiun `gorm:"foreignKey:PensiunID;constraint:OnDelete:CASCADE" json:"-"`
	Pengguna *User    `gorm:"foreignKey:Oleh" json:"-"`
}

func (PensiunRiwayat) TableName() string { return "pensiun_riwayat" }

func (r *PensiunRiwayat) BeforeCreate(_ *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
