package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	DisiplinStatusRegistrasi        = "registrasi"
	DisiplinStatusPerluKelengkapan  = "perlu_kelengkapan"
	DisiplinStatusPemeriksaan       = "pemeriksaan"
	DisiplinStatusMenungguKeputusan = "menunggu_keputusan"
	DisiplinStatusSanksiDitetapkan  = "sanksi_ditetapkan"
	DisiplinStatusTindakLanjut      = "tindak_lanjut"
	DisiplinStatusSelesai           = "selesai"
	DisiplinStatusTidakDilanjutkan  = "tidak_dilanjutkan"

	DisiplinTingkatRingan = "ringan"
	DisiplinTingkatSedang = "sedang"
	DisiplinTingkatBerat  = "berat"

	// Jenis berkas pada kasus. Berkas internal (BAP, tanggapan pegawai, bukti tambahan,
	// bukti penyampaian) hanya untuk Sudin dan pimpinan.
	DisiplinBerkasBuktiAwal          = "bukti_awal"
	DisiplinBerkasBuktiPendukung     = "bukti_pendukung"
	DisiplinBerkasKelengkapan        = "kelengkapan"
	DisiplinBerkasBAP                = "bap"
	DisiplinBerkasTanggapanPegawai   = "tanggapan_pegawai"
	DisiplinBerkasBuktiTambahan      = "bukti_tambahan"
	DisiplinBerkasSalinanKeputusan   = "salinan_keputusan"
	DisiplinBerkasBuktiPenyampaian   = "bukti_penyampaian"
	DisiplinBerkasLaporanPelaksanaan = "laporan_pelaksanaan"
)

// DisiplinStatusFinal: status yang tidak bisa diubah lagi.
func DisiplinStatusFinal(status string) bool {
	return status == DisiplinStatusSelesai || status == DisiplinStatusTidakDilanjutkan
}

// DisiplinBerkasInternal: berkas yang tidak boleh dilihat sekolah.
func DisiplinBerkasInternal(jenis string) bool {
	switch jenis {
	case DisiplinBerkasBAP, DisiplinBerkasTanggapanPegawai, DisiplinBerkasBuktiTambahan, DisiplinBerkasBuktiPenyampaian:
		return true
	}
	return false
}

type DisiplinKronologi struct {
	Tanggal  string `json:"tanggal"` // YYYY-MM-DD
	Kejadian string `json:"kejadian"`
}

// Disiplin adalah satu kasus dugaan pelanggaran disiplin (Figma DS1–DS9).
type Disiplin struct {
	ID               uuid.UUID           `gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	Nomor            string              `gorm:"type:varchar(20);uniqueIndex;not null"`
	PendidikID       uuid.UUID           `gorm:"type:uuid;not null;index"`
	SekolahID        uuid.UUID           `gorm:"type:uuid;not null;index"`
	DilaporkanOleh   uuid.UUID           `gorm:"type:uuid;not null"`
	Status           string              `gorm:"type:varchar(30);not null;index"`
	Tenggat          *time.Time          `gorm:"type:date"` // tenggat tahap yang sedang berjalan
	JenisPelanggaran string              `gorm:"type:varchar(40);not null"`
	Uraian           string              `gorm:"type:text;not null"`
	Kronologi        []DisiplinKronologi `gorm:"type:jsonb;serializer:json;not null"`

	// DS2: verifikasi awal
	CekIdentitas        *bool
	CekKronologi        *bool
	CekBukti            *bool
	CekKewenangan       *bool
	KeputusanVerifikasi string     `gorm:"type:varchar(30)"`
	CatatanVerifikasi   string     `gorm:"type:text"` // daftar kekurangan atau alasan tidak layak
	BatasKelengkapan    *time.Time `gorm:"type:date"`
	DiverifikasiOleh    *uuid.UUID `gorm:"type:uuid"`
	DiverifikasiAt      *time.Time `gorm:"type:timestamp with time zone"`

	// DS3: kelengkapan dari sekolah
	TanggapanSekolah string     `gorm:"type:text"`
	DilengkapiAt     *time.Time `gorm:"type:timestamp with time zone"`

	// DS4: pemanggilan dan pemeriksaan (internal Sudin)
	NomorSuratPanggilan  string     `gorm:"type:varchar(100)"`
	WaktuPemeriksaan     *time.Time `gorm:"type:timestamp with time zone"`
	RingkasanPemeriksaan string     `gorm:"type:text"`

	// DS5: rekomendasi (internal Sudin)
	RingkasanFakta       string     `gorm:"type:text"`
	DugaanTerbukti       string     `gorm:"type:varchar(500)"`
	DasarHukum           string     `gorm:"type:varchar(500)"`
	RekomTingkat         string     `gorm:"type:varchar(10)"`
	RekomJenis           string     `gorm:"type:varchar(40)"`
	Justifikasi          string     `gorm:"type:text"`
	DirekomendasikanOleh *uuid.UUID `gorm:"type:uuid"`
	DirekomendasikanAt   *time.Time `gorm:"type:timestamp with time zone"`

	// DS6: keputusan pimpinan
	KeputusanPimpinan string     `gorm:"type:varchar(20)"`
	CatatanPimpinan   string     `gorm:"type:text"`
	Tingkat           string     `gorm:"type:varchar(10)"`
	JenisHukuman      string     `gorm:"type:varchar(40)"`
	DiputuskanOleh    *uuid.UUID `gorm:"type:uuid"`
	DiputuskanAt      *time.Time `gorm:"type:timestamp with time zone"`

	// DS7: keputusan final dan tindak lanjut
	NomorKeputusan         string     `gorm:"type:varchar(100)"`
	TanggalKeputusan       *time.Time `gorm:"type:date"`
	MasaMulai              *time.Time `gorm:"type:date"`
	MasaSelesai            *time.Time `gorm:"type:date"`
	TanggalDiterimaPegawai *time.Time `gorm:"type:date"`
	DicatatIDIS            bool       `gorm:"not null;default:false"`
	IDISOleh               *uuid.UUID `gorm:"type:uuid"`
	IDISAt                 *time.Time `gorm:"type:timestamp with time zone"`
	UpayaAdministratif     string     `gorm:"type:varchar(10)"` // "", tidak_ada, ada
	CatatanUpaya           string     `gorm:"type:text"`
	DitutupOleh            *uuid.UUID `gorm:"type:uuid"`
	DitutupAt              *time.Time `gorm:"type:timestamp with time zone"`

	// DS8: laporan pelaksanaan dari sekolah
	CatatanPelaksanaan string     `gorm:"type:text"`
	PelaksanaanAt      *time.Time `gorm:"type:timestamp with time zone"`

	Pendidik      *Pendidik         `gorm:"foreignKey:PendidikID;constraint:OnDelete:RESTRICT"`
	Sekolah       *Sekolah          `gorm:"foreignKey:SekolahID;constraint:OnDelete:RESTRICT"`
	Pelapor       *User             `gorm:"foreignKey:DilaporkanOleh"`
	Verifikator   *User             `gorm:"foreignKey:DiverifikasiOleh"`
	Perekomendasi *User             `gorm:"foreignKey:DirekomendasikanOleh"`
	Pemutus       *User             `gorm:"foreignKey:DiputuskanOleh"`
	PencatatIDIS  *User             `gorm:"foreignKey:IDISOleh"`
	Penutup       *User             `gorm:"foreignKey:DitutupOleh"`
	Berkas        []DisiplinBerkas  `gorm:"foreignKey:DisiplinID"`
	Riwayat       []DisiplinRiwayat `gorm:"foreignKey:DisiplinID"`

	Timestamp
}

func (Disiplin) TableName() string { return "disiplin" }

func (d *Disiplin) BeforeCreate(_ *gorm.DB) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return nil
}

// DisiplinBerkas menautkan dokumen ke kasus beserta jenisnya.
type DisiplinBerkas struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	DisiplinID uuid.UUID `gorm:"type:uuid;not null;index"`
	DokumenID  uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
	Jenis      string    `gorm:"type:varchar(30);not null"`
	Internal   bool      `gorm:"not null"`
	CreatedAt  time.Time `gorm:"type:timestamp with time zone"`

	Disiplin *Disiplin `gorm:"foreignKey:DisiplinID;constraint:OnDelete:CASCADE"`
	Dokumen  *Dokumen  `gorm:"foreignKey:DokumenID;constraint:OnDelete:RESTRICT"`
}

func (DisiplinBerkas) TableName() string { return "disiplin_berkas" }

func (b *DisiplinBerkas) BeforeCreate(_ *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

type DisiplinRiwayat struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	DisiplinID uuid.UUID `gorm:"type:uuid;not null;index"`
	DariStatus string    `gorm:"type:varchar(30)"`
	KeStatus   string    `gorm:"type:varchar(30);not null"`
	Oleh       uuid.UUID `gorm:"type:uuid;not null"`
	Catatan    string    `gorm:"type:text"`
	CreatedAt  time.Time `gorm:"type:timestamp with time zone"`

	Disiplin *Disiplin `gorm:"foreignKey:DisiplinID;constraint:OnDelete:CASCADE"`
	Pengguna *User     `gorm:"foreignKey:Oleh"`
}

func (DisiplinRiwayat) TableName() string { return "disiplin_riwayat" }

func (r *DisiplinRiwayat) BeforeCreate(_ *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
