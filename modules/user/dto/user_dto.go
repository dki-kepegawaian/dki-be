package dto

import "github.com/Caknoooo/go-gin-clean-starter/database/entities"

type (
	UserResponse struct {
		ID          string  `json:"id"`
		Name        string  `json:"name"`
		Email       string  `json:"email"`
		TelpNumber  string  `json:"telp_number"`
		Role        string  `json:"role"`
		IsAktif     bool    `json:"is_aktif"`
		SudinID     *string `json:"sudin_id"`
		SudinNama   string  `json:"sudin_nama,omitempty"`
		SekolahID   *string `json:"sekolah_id"`
		SekolahNama string  `json:"sekolah_nama,omitempty"`
	}

	GantiPasswordRequest struct {
		PasswordLama string `json:"password_lama" binding:"required"`
		PasswordBaru string `json:"password_baru" binding:"required,min=8,max=72"`
	}
)

func FromEntity(u entities.User) UserResponse {
	r := UserResponse{
		ID: u.ID.String(), Name: u.Name, Email: u.Email, TelpNumber: u.TelpNumber,
		Role: u.Role, IsAktif: u.IsAktif,
	}
	if u.SudinID != nil {
		s := u.SudinID.String()
		r.SudinID = &s
	}
	if u.Sudin != nil {
		r.SudinNama = u.Sudin.Nama
	}
	if u.SekolahID != nil {
		s := u.SekolahID.String()
		r.SekolahID = &s
	}
	if u.Sekolah != nil {
		r.SekolahNama = u.Sekolah.Nama
	}
	return r
}
