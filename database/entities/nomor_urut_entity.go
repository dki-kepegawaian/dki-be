package entities

// NomorUrut menyimpan nomor registrasi terakhir per prefix dan tahun (CUT-2026-0142).
type NomorUrut struct {
	Prefix   string `gorm:"type:varchar(10);primaryKey"`
	Tahun    int    `gorm:"primaryKey;autoIncrement:false"`
	Terakhir int    `gorm:"not null;default:0"`
}

func (NomorUrut) TableName() string { return "nomor_urut" }
