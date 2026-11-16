package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	KPStatusDiajukan       = "diajukan"
	KPStatusPerluPerbaikan = "perlu_perbaikan"
	KPStatusTerverifikasi  = "terverifikasi"
	KPStatusDiusulkanKeBKD = "diusulkan_ke_bkd"
	KPStatusSelesai        = "selesai"
)

// KPBerkasWajib adalah 6 berkas persyaratan kenaikan pangkat (Figma K2).
var KPBerkasWajib = []JenisBerkas{
	{"sk_pangkat_terakhir", "SK pangkat terakhir", "Persyaratan"},
	{"skp_2_tahun", "SKP 2 tahun terakhir", "Persyaratan"},
	{"pak", "PAK (Penetapan Angka Kredit)", "Persyaratan"},
	{"ijazah_transkrip", "Ijazah & transkrip terakhir", "Persyaratan"},
	{"sertifikat_pendidik", "Sertifikat pendidik", "Persyaratan"},
	{"pengantar_kepsek", "Surat pengantar kepala sekolah", "Persyaratan"},
}

// PangkatPNS memetakan golongan ruang PNS ke nama pangkat, urut dari terendah.
var PangkatPNS = []struct{ Golongan, Pangkat string }{
	{"I/a", "Juru Muda"}, {"I/b", "Juru Muda Tk. I"}, {"I/c", "Juru"}, {"I/d", "Juru Tk. I"},
	{"II/a", "Pengatur Muda"}, {"II/b", "Pengatur Muda Tk. I"}, {"II/c", "Pengatur"}, {"II/d", "Pengatur Tk. I"},
	{"III/a", "Penata Muda"}, {"III/b", "Penata Muda Tk. I"}, {"III/c", "Penata"}, {"III/d", "Penata Tk. I"},
	{"IV/a", "Pembina"}, {"IV/b", "Pembina Tk. I"}, {"IV/c", "Pembina Utama Muda"}, {"IV/d", "Pembina Utama Madya"},
	{"IV/e", "Pembina Utama"},
}

type KenaikanPangkat struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Nomor        string    `gorm:"type:varchar(20);uniqueIndex;not null" json:"nomor"`
	PendidikID   uuid.UUID `gorm:"type:uuid;not null;index" json:"pendidik_id"`
	SekolahID    uuid.UUID `gorm:"type:uuid;not null;index" json:"sekolah_id"`
	DiajukanOleh uuid.UUID `gorm:"type:uuid;not null" json:"diajukan_oleh"`
	Periode      time.Time `gorm:"type:date;not null;index" json:"periode"` // 1 April atau 1 Oktober
	PangkatLama  string    `gorm:"type:varchar(50)" json:"pangkat_lama"`
	GolonganLama string    `gorm:"type:varchar(10);not null" json:"golongan_lama"`
	PangkatBaru  string    `gorm:"type:varchar(50);not null" json:"pangkat_baru"`
	GolonganBaru string    `gorm:"type:varchar(10);not null" json:"golongan_baru"`
	Status       string    `gorm:"type:varchar(30);not null;index" json:"status"`

	// K3: verifikasi berkas
	DiverifikasiOleh *uuid.UUID `gorm:"type:uuid" json:"diverifikasi_oleh"`
	DiverifikasiAt   *time.Time `gorm:"type:timestamp with time zone" json:"diverifikasi_at"`

	// K4: pengiriman ke BKD (di luar sistem, dicatat per batch)
	NomorSuratPengantar string     `gorm:"type:varchar(100)" json:"nomor_surat_pengantar"`
	TanggalKirim        *time.Time `gorm:"type:date" json:"tanggal_kirim"`
	CaraKirim           string     `gorm:"type:varchar(100)" json:"cara_kirim"`
	BuktiKirimID        *uuid.UUID `gorm:"type:uuid" json:"bukti_kirim_id"`
	CatatanKirim        string     `gorm:"type:text" json:"catatan_kirim"`
	DikirimOleh         *uuid.UUID `gorm:"type:uuid" json:"dikirim_oleh"`

	// K5: SK dari BKD; pangkat pendidik diperbarui
	NomorSK     string     `gorm:"column:nomor_sk;type:varchar(100)" json:"nomor_sk"`
	TanggalSK   *time.Time `gorm:"column:tanggal_sk;type:date" json:"tanggal_sk"`
	TMTPangkat  *time.Time `gorm:"column:tmt_pangkat;type:date" json:"tmt_pangkat"`
	SKID        *uuid.UUID `gorm:"column:sk_id;type:uuid" json:"sk_id"`
	SelesaiOleh *uuid.UUID `gorm:"type:uuid" json:"selesai_oleh"`
	SelesaiAt   *time.Time `gorm:"type:timestamp with time zone" json:"selesai_at"`

	Pendidik   *Pendidik                `gorm:"foreignKey:PendidikID;constraint:OnDelete:RESTRICT" json:"pendidik,omitempty"`
	Sekolah    *Sekolah                 `gorm:"foreignKey:SekolahID;constraint:OnDelete:RESTRICT" json:"sekolah,omitempty"`
	Pengaju    *User                    `gorm:"foreignKey:DiajukanOleh" json:"-"`
	BuktiKirim *Dokumen                 `gorm:"foreignKey:BuktiKirimID" json:"-"`
	SK         *Dokumen                 `gorm:"foreignKey:SKID" json:"-"`
	Berkas     []KenaikanPangkatBerkas  `gorm:"foreignKey:KenaikanPangkatID" json:"-"`
	Riwayat    []KenaikanPangkatRiwayat `gorm:"foreignKey:KenaikanPangkatID" json:"-"`

	Timestamp
}

func (KenaikanPangkat) TableName() string { return "kenaikan_pangkat" }

func (k *KenaikanPangkat) BeforeCreate(_ *gorm.DB) error {
	if k.ID == uuid.Nil {
		k.ID = uuid.New()
	}
	return nil
}

type KenaikanPangkatBerkas struct {
	ID                uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	KenaikanPangkatID uuid.UUID `gorm:"type:uuid;not null;index" json:"kenaikan_pangkat_id"`
	BerkasPersyaratan
	Timestamp

	KenaikanPangkat *KenaikanPangkat `gorm:"foreignKey:KenaikanPangkatID;constraint:OnDelete:CASCADE" json:"-"`
	Dokumen         *Dokumen         `gorm:"foreignKey:DokumenID" json:"-"`
}

func (KenaikanPangkatBerkas) TableName() string { return "kenaikan_pangkat_berkas" }

func (b *KenaikanPangkatBerkas) BeforeCreate(_ *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

type KenaikanPangkatRiwayat struct {
	ID                uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	KenaikanPangkatID uuid.UUID `gorm:"type:uuid;not null;index" json:"kenaikan_pangkat_id"`
	DariStatus        string    `gorm:"type:varchar(30)" json:"dari_status"`
	KeStatus          string    `gorm:"type:varchar(30);not null" json:"ke_status"`
	Oleh              uuid.UUID `gorm:"type:uuid;not null" json:"oleh"`
	Catatan           string    `gorm:"type:text" json:"catatan"`
	CreatedAt         time.Time `gorm:"type:timestamp with time zone" json:"created_at"`

	KenaikanPangkat *KenaikanPangkat `gorm:"foreignKey:KenaikanPangkatID;constraint:OnDelete:CASCADE" json:"-"`
	Pengguna        *User            `gorm:"foreignKey:Oleh" json:"-"`
}

func (KenaikanPangkatRiwayat) TableName() string { return "kenaikan_pangkat_riwayat" }

func (r *KenaikanPangkatRiwayat) BeforeCreate(_ *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
