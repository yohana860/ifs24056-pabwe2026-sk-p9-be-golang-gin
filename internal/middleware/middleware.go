// Package middleware berisi CORS, autentikasi JWT, role guard, rate limiter, dan header keamanan.
package middleware

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"app/internal/response"
)

const userKey = "user"

// Claims adalah isi token JWT.
type Claims struct {
	UserID   uint   `json:"uid"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken menerbitkan JWT HS256 dengan masa berlaku ttl.
func GenerateToken(secret string, id uint, username, role string, ttl time.Duration) (string, error) {
	claims := Claims{
		UserID:   id,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// CORS mengizinkan origin frontend. origins berisi "*" atau daftar origin eksplisit.
func CORS(origins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if slices.Contains(origins, "*") {
			c.Header("Access-Control-Allow-Origin", "*")
		} else if origin != "" && slices.Contains(origins, origin) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
		}
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization,Content-Type")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// SecurityHeaders menambahkan header pengaman standar.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		c.Next()
	}
}

// Timeout membatasi durasi konteks request.
func Timeout(d time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), d)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// Auth memverifikasi Bearer token JWT dan menyimpan Claims pada context.
func Auth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		parts := strings.SplitN(c.GetHeader("Authorization"), " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			response.AbortFail(c, http.StatusUnauthorized, "Token autentikasi tidak ditemukan")
			return
		}
		claims := &Claims{}
		_, err := jwt.ParseWithClaims(parts[1], claims, func(*jwt.Token) (any, error) {
			return []byte(secret), nil
		}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())
		if err != nil {
			response.AbortFail(c, http.StatusUnauthorized, "Token tidak valid atau sudah kedaluwarsa")
			return
		}
		c.Set(userKey, claims)
		c.Next()
	}
}

// CurrentUser mengambil Claims pengguna aktif (nil jika belum terautentikasi).
func CurrentUser(c *gin.Context) *Claims {
	v, _ := c.Get(userKey)
	claims, _ := v.(*Claims)
	return claims
}

// RequireRole hanya meloloskan pengguna dengan salah satu role yang diizinkan.
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := CurrentUser(c)
		if claims == nil || !slices.Contains(roles, claims.Role) {
			response.AbortFail(c, http.StatusForbidden, "Anda tidak memiliki akses ke sumber daya ini")
			return
		}
		c.Next()
	}
}

type bucket struct {
	count   int
	resetAt time.Time
}

var now = time.Now

// RateLimiter membatasi limit request per window untuk setiap alamat IP.
func RateLimiter(limit int, window time.Duration) gin.HandlerFunc {
	var mu sync.Mutex
	buckets := map[string]*bucket{}
	return func(c *gin.Context) {
		key := c.ClientIP()
		mu.Lock()
		b, ok := buckets[key]
		if !ok || now().After(b.resetAt) {
			b = &bucket{resetAt: now().Add(window)}
			buckets[key] = b
		}
		b.count++
		exceeded := b.count > limit
		mu.Unlock()
		if exceeded {
			response.AbortFail(c, http.StatusTooManyRequests, "Terlalu banyak permintaan, coba lagi nanti")
			return
		}
		c.Next()
	}
}
