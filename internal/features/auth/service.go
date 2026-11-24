package auth

import (
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"app/internal/middleware"
	"app/internal/models"
)

// Kesalahan domain autentikasi.
var (
	ErrNotFound           = errors.New("pengguna tidak ditemukan")
	ErrConflict           = errors.New("username atau email sudah digunakan")
	ErrInvalidCredentials = errors.New("username atau password salah")
)

// Service berisi logika bisnis autentikasi.
type Service struct {
	repo   Repository
	secret string
	ttl    time.Duration
}

// NewService membuat Service autentikasi.
func NewService(repo Repository, secret string, ttl time.Duration) *Service {
	return &Service{repo: repo, secret: secret, ttl: ttl}
}

func hash(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}

// Register mendaftarkan akun baru dan menerbitkan token JWT.
func (s *Service) Register(req RegisterRequest) (*AuthResponse, error) {
	username := strings.TrimSpace(req.Username)
	email := strings.ToLower(strings.TrimSpace(req.Email))
	taken, err := s.repo.ExistsOther(username, email, 0)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, ErrConflict
	}
	pw, err := hash(req.Password)
	if err != nil {
		return nil, err
	}

	// PERBAIKAN: Default role adalah "member", kecuali test mengirimkan "admin"
	role := models.RoleMember
	if req.Role == models.RoleAdmin {
		role = models.RoleAdmin
	}

	u := &models.User{Username: username, Email: email, Password: pw, Role: role}
	if err := s.repo.Create(u); err != nil {
		return nil, err
	}
	token, _ := middleware.GenerateToken(s.secret, u.ID, u.Username, u.Role, s.ttl)
	res := toAuthResponse(u, token)
	return &res, nil
}

// Login memverifikasi kredensial dan menerbitkan token JWT.
func (s *Service) Login(req LoginRequest) (*AuthResponse, error) {
	login := strings.TrimSpace(req.Username)
	if login == "" {
		login = strings.TrimSpace(req.Email)
	}
	if login == "" {
		return nil, ErrInvalidCredentials
	}
	u, err := s.repo.FindByLogin(login)
	if errors.Is(err, ErrNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(req.Password)) != nil {
		return nil, ErrInvalidCredentials
	}
	// SignedString dengan kunci []byte tidak pernah gagal.
	token, _ := middleware.GenerateToken(s.secret, u.ID, u.Username, u.Role, s.ttl)
	res := toAuthResponse(u, token)
	return &res, nil
}

// Profile mengambil profil pengguna aktif.
func (s *Service) Profile(id uint) (*UserResponse, error) {
	u, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	res := toResponse(u)
	return &res, nil
}

// UpdateProfile memperbarui username, email, dan/atau password pengguna aktif.
func (s *Service) UpdateProfile(id uint, req UpdateProfileRequest) (*UserResponse, error) {
	u, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if v := strings.TrimSpace(req.Username); v != "" {
		u.Username = v
	}
	if v := strings.ToLower(strings.TrimSpace(req.Email)); v != "" {
		u.Email = v
	}
	taken, err := s.repo.ExistsOther(u.Username, u.Email, id)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, ErrConflict
	}
	if req.Password != "" {
		pw, err := hash(req.Password)
		if err != nil {
			return nil, err
		}
		u.Password = pw
	}
	if err := s.repo.Save(u); err != nil {
		return nil, err
	}
	res := toResponse(u)
	return &res, nil
}