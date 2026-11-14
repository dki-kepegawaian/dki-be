package dto

type (
	TambahRequest struct {
		Name       string `json:"name" binding:"required,min=2,max=100"`
		Email      string `json:"email" binding:"required,email,max=255"`
		TelpNumber string `json:"telp_number" binding:"omitempty,max=20"`
		Role       string `json:"role" binding:"required,oneof=staf_sudin operator_sekolah pimpinan admin"`
		// SekolahID wajib untuk operator_sekolah.
		SekolahID string `json:"sekolah_id" binding:"omitempty,uuid"`
		// SudinID untuk staf_sudin/pimpinan. Staf Sudin selalu memakai Sudin-nya sendiri; hanya admin yang mengisi.
		SudinID  string `json:"sudin_id" binding:"omitempty,uuid"`
		Password string `json:"password" binding:"required,min=8,max=72"`
	}

	UbahRequest struct {
		Name       string `json:"name" binding:"required,min=2,max=100"`
		Email      string `json:"email" binding:"required,email,max=255"`
		TelpNumber string `json:"telp_number" binding:"omitempty,max=20"`
		Role       string `json:"role" binding:"required,oneof=staf_sudin operator_sekolah pimpinan admin"`
		SekolahID  string `json:"sekolah_id" binding:"omitempty,uuid"`
		SudinID    string `json:"sudin_id" binding:"omitempty,uuid"`
		IsAktif    *bool  `json:"is_aktif" binding:"required"`
	}

	ResetPasswordRequest struct {
		PasswordBaru string `json:"password_baru" binding:"required,min=8,max=72"`
	}

	Filter struct {
		Search    string
		Role      string
		SekolahID string
		Aktif     *bool
		Offset    int
		Limit     int
	}
)
