package dto

// PendidikRequest dipakai untuk tambah dan ubah. Saat ubah, semua kolom dikirim ulang (PUT).
type PendidikRequest struct {
	SekolahID         string  `json:"sekolah_id" binding:"required,uuid"`
	Nama              string  `json:"nama" binding:"required,min=2,max=150"`
	NIP               *string `json:"nip" binding:"omitempty,numeric,len=18"`
	NIK               string  `json:"nik" binding:"omitempty,numeric,len=16"`
	NUPTK             string  `json:"nuptk" binding:"omitempty,numeric,len=16"`
	NRK               string  `json:"nrk" binding:"omitempty,max=20"`
	NIKKI             string  `json:"nikki" binding:"omitempty,max=30"`
	StatusKepegawaian string  `json:"status_kepegawaian" binding:"required,oneof=PNS PPPK PPPK_PW KKI"`
	Pangkat           string  `json:"pangkat" binding:"omitempty,max=50"`
	Golongan          string  `json:"golongan" binding:"omitempty,max=10"`
	Jabatan           string  `json:"jabatan" binding:"omitempty,max=150"`
	JabatanDapodik    string  `json:"jabatan_dapodik" binding:"omitempty,max=150"`
	ProgramStudi      string  `json:"program_studi" binding:"omitempty,max=150"`
	Sertifikasi       string  `json:"sertifikasi" binding:"omitempty,max=150"`
	Email             string  `json:"email" binding:"omitempty,email,max=255"`
	Telepon           string  `json:"telepon" binding:"omitempty,max=20"`
	AlamatDomisili    string  `json:"alamat_domisili" binding:"omitempty,max=500"`
	KotaDomisili      string  `json:"kota_domisili" binding:"omitempty,max=100"`
	TanggalLahir      string  `json:"tanggal_lahir" binding:"omitempty,datetime=2006-01-02"`
	TMTCPNS           string  `json:"tmt_cpns" binding:"omitempty,datetime=2006-01-02"`
	TMTGolongan       string  `json:"tmt_golongan" binding:"omitempty,datetime=2006-01-02"`
}
