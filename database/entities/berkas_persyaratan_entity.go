package entities

import (
	"time"

	"github.com/google/uuid"
)

// JenisBerkas adalah satu berkas persyaratan yang wajib diunggah sekolah (Pensiun P2, KP K2).
type JenisBerkas struct {
	Kode     string `json:"kode"`
	Label    string `json:"label"`
	Kelompok string `json:"kelompok"`
}

// BerkasPersyaratan disematkan di tabel berkas Pensiun dan Kenaikan Pangkat:
// satu baris per jenis berkas, beserta penilaian Sudin per berkas (P3/K3).
type BerkasPersyaratan struct {
	Jenis       string     `gorm:"type:varchar(40);not null" json:"jenis"`
	DokumenID   uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex" json:"dokumen_id"`
	Valid       *bool      `json:"valid"`
	Catatan     string     `gorm:"type:text" json:"catatan"`
	DinilaiOleh *uuid.UUID `gorm:"type:uuid" json:"dinilai_oleh"`
	DinilaiAt   *time.Time `gorm:"type:timestamp with time zone" json:"dinilai_at"`
}
