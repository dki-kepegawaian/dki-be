package dto

type (
	// LoginRequest: isi username (layar L1) atau email.
	LoginRequest struct {
		Username string `json:"username" binding:"omitempty,max=50"`
		Email    string `json:"email" binding:"omitempty,email,max=255"`
		Password string `json:"password" binding:"required,max=72"`
	}

	RefreshTokenRequest struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	TokenResponse struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		Role         string `json:"role"`
	}
)
