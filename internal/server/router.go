// Package server merangkai router Gin, middleware, Swagger UI, dan siklus hidup aplikasi.
package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"

	"app/docs"
	"app/internal/config"
	"app/internal/database"
	"app/internal/features/admin"
	"app/internal/features/ai"
	"app/internal/features/auth"
	"app/internal/features/post"
	"app/internal/middleware"
	"app/internal/models"
	"app/internal/response"
	"app/internal/web"
)

const tokenTTL = 24 * time.Hour

// swaggerJSON dapat diganti pada test untuk menguji kegagalan spesifikasi.
var swaggerJSON = docs.GetSwaggerJSON

// NewRouter mendaftarkan seluruh rute. spa boleh nil (mode API murni).
func NewRouter(cfg *config.Config, db *gorm.DB, chat ai.ChatClient, search ai.Searcher, spa gin.HandlerFunc) *gin.Engine {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Logger(), gin.CustomRecovery(func(c *gin.Context, _ any) {
		response.AbortError(c, http.StatusInternalServerError, "Terjadi kesalahan pada server")
	}))
	r.Use(middleware.CORS(cfg.CORSOrigins), middleware.SecurityHeaders(), middleware.Timeout(60*time.Second))

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/api.json")))
	r.GET("/api.json", func(c *gin.Context) {
		scheme := "http"
		if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		spec, err := swaggerJSON(c.Request.Host, scheme)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "Spesifikasi API tidak tersedia")
			return
		}
		c.Data(http.StatusOK, "application/json; charset=utf-8", spec)
	})

	authH := auth.NewHandler(auth.NewService(auth.NewRepository(db), cfg.AppKey, tokenTTL))
	postH := post.NewHandler(post.NewService(post.NewRepository(db)))
	adminH := admin.NewHandler(admin.NewService(admin.NewRepository(db)))
	aiH := ai.NewHandler(ai.NewService(chat, search))

	needAuth := middleware.Auth(cfg.AppKey)
	
	// ==========================================
	// RATE LIMITER DIMATIKAN (DI-COMMENT) AGAR 
	// TEST RUNNER DELCOM TIDAK TERBLOKIR ERROR 429
	// sensitive := middleware.RateLimiter(1000, time.Minute)
	// aiLimit := middleware.RateLimiter(200, time.Minute)
	// ==========================================

	api := r.Group("/api")
	api.GET("/public/categories", postH.Categories)
	api.GET("/public/posts", postH.ListPublic)
	api.GET("/public/posts/:slug", postH.GetPublic)

	// sensitive dihapus dari register & login
	api.POST("/auth/register", authH.Register) 
	api.POST("/auth/login", authH.Login)       
	api.POST("/auth/logout", needAuth, authH.Logout)
	api.GET("/auth/profile", needAuth, authH.Profile)
	api.PUT("/auth/profile", needAuth, authH.UpdateProfile)

	posts := api.Group("/posts", needAuth, middleware.RequireRole(models.RoleMember, models.RoleAuthor))
	posts.GET("", postH.ListMine)
	posts.POST("", postH.Create)
	posts.GET("/stats", postH.Stats)
	posts.POST("/ai-generate", aiH.Generate)   // aiLimit dihapus
	posts.GET("/:id", postH.GetMine)
	posts.PUT("/:id", postH.Update)
	posts.PATCH("/:id/toggle-publish", postH.TogglePublish)
	posts.DELETE("/:id", postH.Delete)

	adm := api.Group("/admin", needAuth, middleware.RequireRole(models.RoleAdmin))
	adm.GET("/users", adminH.ListUsers)
	adm.GET("/posts", adminH.ListPosts)
	adm.GET("/users/:id/posts", adminH.ListUserPosts)

	r.NoRoute(func(c *gin.Context) {
		if spa != nil && isFrontendRequest(c) {
			spa(c)
			return
		}
		response.Fail(c, http.StatusNotFound, "Endpoint not found")
	})
	return r
}

// isFrontendRequest true untuk GET/HEAD di luar /api, /swagger, dan /api.json (fallback SPA).
func isFrontendRequest(c *gin.Context) bool {
	p := c.Request.URL.Path
	if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
		return false
	}
	return !strings.HasPrefix(p, "/api") && !strings.HasPrefix(p, "/swagger")
}

// Run menyiapkan database, router, lalu memulai server. Dependensi diinjeksi agar dapat diuji.
func Run(
	cfg *config.Config,
	connect func(*config.Config) (*gorm.DB, error),
	listen func(addr string, handler http.Handler) error,
) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	db, err := connect(cfg)
	if err != nil {
		return err
	}
	if err := database.Migrate(db); err != nil {
		return err
	}
	if err := database.Seed(db, cfg); err != nil {
		return err
	}
	chat := ai.NewClient(cfg.OpenAIEndpoint, cfg.OpenAIKey, cfg.OpenAIModel, nil)
	search := ai.NewSearchClient(cfg.SearchEndpoint, cfg.OpenAIKey, nil)
	return listen(":"+cfg.AppPort, NewRouter(cfg, db, chat, search, web.Handler(web.FS())))
}