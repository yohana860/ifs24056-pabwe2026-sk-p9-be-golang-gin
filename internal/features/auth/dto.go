package auth

import (
	"time"

	"app/internal/models"
)

// RegisterRequest adalah payload registrasi. Role tidak pernah diterima dari klien.
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email,max=255"`
	Password string `json:"password" binding:"required,min=6,max=72"`
}

// LoginRequest adalah payload login (username atau email).
type LoginRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password" binding:"required"`
}

// UpdateProfileRequest adalah payload pembaruan profil; field kosong tidak diubah.
type UpdateProfileRequest struct {
	Username string `json:"username" binding:"omitempty,min=3,max=50"`
	Email    string `json:"email" binding:"omitempty,email,max=255"`
	Password string `json:"password" binding:"omitempty,min=6,max=72"`
}

// UserResponse adalah representasi pengguna tanpa password.
type UserResponse struct {
	ID        uint      `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// AuthResponse berisi token JWT dan profil pengguna.
type AuthResponse struct {
	Token       string       `json:"token"`
	AccessToken string       `json:"access_token,omitempty"`
	User        UserResponse `json:"user"`
	ID          uint         `json:"id"`
	Username    string       `json:"username"`
	Email       string       `json:"email"`
	Role        string       `json:"role"`
	CreatedAt   time.Time    `json:"created_at"`
}

func toAuthResponse(u *models.User, token string) AuthResponse {
	ur := toResponse(u)
	return AuthResponse{
		Token:       token,
		AccessToken: token,
		User:        ur,
		ID:          ur.ID,
		Username:    ur.Username,
		Email:       ur.Email,
		Role:        ur.Role,
		CreatedAt:   ur.CreatedAt,
	}
}

func toResponse(u *models.User) UserResponse {
	return UserResponse{ID: u.ID, Username: u.Username, Email: u.Email, Role: u.Role, CreatedAt: u.CreatedAt}
}
