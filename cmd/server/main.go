package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/HW618/mdict-server/internal/auth"
	"github.com/HW618/mdict-server/internal/config"
	"github.com/HW618/mdict-server/internal/dict"
	"github.com/HW618/mdict-server/internal/handlers"
	"github.com/HW618/mdict-server/internal/middleware"
	"github.com/HW618/mdict-server/internal/models"
	"github.com/HW618/mdict-server/internal/store"
	"golang.org/x/crypto/bcrypt"
)

const version = "1.0.0"

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Setup logger
	setupLogger(cfg.LogLevel, cfg.LogFormat)

	log.Info().
		Str("version", version).
		Str("addr", cfg.GetServerAddress()).
		Msg("Starting Mdict Server")

	// Initialize stores
	sqliteStore, err := store.NewSQLiteStore(cfg.DataDir)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize database")
	}
	defer sqliteStore.Close()

	userStore := store.NewUserStore(sqliteStore)
	dictStore := store.NewDictStore(sqliteStore)

	// Create default admin user if not exists
	if err := createDefaultAdmin(userStore, cfg.AdminUser, cfg.AdminPass); err != nil {
		log.Fatal().Err(err).Msg("Failed to create default admin")
	}

	// Initialize dictionary engine
	dictEngine := dict.NewEngine(cfg.DictDir, dictStore)
	if err := dictEngine.LoadAll(); err != nil {
		log.Error().Err(err).Msg("Failed to load some dictionaries")
	}

	// Initialize JWT manager
	jwtManager := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)

	// Initialize middleware
	authMiddleware := auth.NewAuthMiddleware(jwtManager, userStore)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(userStore, jwtManager)
	searchHandler := handlers.NewSearchHandler(dictEngine)
	dictHandler := handlers.NewDictHandler(dictEngine, dictStore, cfg.DictDir, cfg.MaxUploadSizeBytes)
	userHandler := handlers.NewUserHandler(userStore, jwtManager)
	skillHandler := handlers.NewSkillHandler(cfg.SkillServerURL)
	healthHandler := handlers.NewHealthHandler(dictStore, userStore, version)

	// Setup Gin
	if cfg.LogLevel != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())

	// Apply middleware
	router.Use(middleware.CORS(middleware.CORSConfig{
		AllowOrigins: cfg.CORSOrigins,
	}))
	router.Use(middleware.Logger())

	// Serve static files (templates)
	router.StaticFile("/", "./templates/index.html")
	router.StaticFile("/admin", "./templates/admin.html")

	// API routes
	api := router.Group("/api/v1")
	{
		// Public routes
		api.POST("/auth/login", middleware.LoginRateLimit(), authHandler.Login)
		api.POST("/auth/refresh", authHandler.Refresh)
		api.GET("/health", healthHandler.Health)
		api.GET("/skill.json", skillHandler.GetSkill)

		// Authenticated routes
		auth := api.Group("")
		auth.Use(authMiddleware.RequireAuth())
		{
		auth.POST("/auth/logout", authHandler.Logout)
		auth.PUT("/users/me/password", userHandler.ChangePassword)
		}

		// API access routes (JWT or API token)
		apiAccess := api.Group("")
		apiAccess.Use(authMiddleware.RequireAPIAccess())
		apiAccess.Use(middleware.RateLimit(cfg.RateLimit))
		{
			apiAccess.GET("/search", searchHandler.Search)
			apiAccess.GET("/search/fuzzy", searchHandler.FuzzySearch)
			apiAccess.GET("/dicts", dictHandler.List)
		}

		// Dict admin routes
		dictAdmin := api.Group("")
		dictAdmin.Use(authMiddleware.RequireDictAdmin())
		{
			dictAdmin.PATCH("/dicts/:id/status", dictHandler.UpdateStatus)
			dictAdmin.PUT("/dicts/:id/title", dictHandler.UpdateTitle)
			dictAdmin.POST("/dicts/upload", dictHandler.Upload)
			dictAdmin.POST("/dicts/upload/init", dictHandler.UploadInit)
			dictAdmin.PUT("/dicts/upload/chunk", dictHandler.UploadChunk)
			dictAdmin.POST("/dicts/upload/complete", dictHandler.UploadComplete)
			dictAdmin.GET("/dicts/:id/download", dictHandler.Download)
			dictAdmin.DELETE("/dicts/:id", dictHandler.Delete)
		}

		// User admin routes
		userAdmin := api.Group("")
		userAdmin.Use(authMiddleware.RequireUserAdmin())
		userAdmin.Use(middleware.RateLimit(cfg.RateLimit))
		{
			userAdmin.GET("/users", userHandler.List)
			userAdmin.POST("/users", userHandler.Create)
			userAdmin.DELETE("/users/:id", userHandler.Delete)
			userAdmin.PUT("/users/:id/permissions", userHandler.UpdatePermissions)
			userAdmin.POST("/users/:id/reset-token", userHandler.ResetToken)
			userAdmin.PUT("/users/:id/password", userHandler.AdminResetPassword)
			userAdmin.GET("/stats", healthHandler.Stats)
		}
	}

	// Create HTTP server
	srv := &http.Server{
		Addr:         cfg.GetServerAddress(),
		Handler:      router,
		ReadTimeout:  300 * time.Second,
		WriteTimeout: 300 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start expired token cleanup goroutine
	tokenCleanupDone := make(chan struct{})
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := userStore.CleanExpiredTokens(); err != nil {
					log.Error().Err(err).Msg("Failed to clean expired tokens")
				} else {
					log.Debug().Msg("Cleaned expired refresh tokens")
				}
				dictHandler.CleanupExpiredUploads(1 * time.Hour)
			case <-tokenCleanupDone:
				return
			}
		}
	}()

	// Start server in goroutine
	go func() {
		log.Info().Str("addr", cfg.GetServerAddress()).Msg("Server listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server failed to start")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server...")

	// Stop token cleanup goroutine
	close(tokenCleanupDone)

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server exited")
}

// setupLogger configures the logger
func setupLogger(level, format string) {
	// Set log level
	switch level {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	// Set log format
	if format == "text" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
}

// createDefaultAdmin creates the default admin user if it doesn't exist
func createDefaultAdmin(userStore *store.UserStore, username, password string) error {
	// Check if admin exists
	exists, err := userStore.ExistsByUsername(username)
	if err != nil {
		return fmt.Errorf("failed to check admin existence: %w", err)
	}

	if exists {
		log.Info().Str("username", username).Msg("Admin user already exists")
		return nil
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Create admin user
	admin := &models.User{
		ID:          "admin-001",
		Username:    username,
		Password:    string(hashedPassword),
		APIToken:    models.GenerateAPIToken(),
		CanUseAPI:   true,
		IsDictAdmin: true,
		IsUserAdmin: true,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := userStore.Create(admin); err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	log.Info().
		Str("username", username).
		Msg("Created default admin user — credentials printed below for initial setup")
	fmt.Fprintf(os.Stderr, "\n"+
		"╔══════════════════════════════════════════════════════╗\n"+
		"║           DEFAULT ADMIN CREDENTIALS                  ║\n"+
		"╠══════════════════════════════════════════════════════╣\n"+
		"║  Username: %-41s ║\n"+
		"║  Password: %-41s ║\n"+
		"╠══════════════════════════════════════════════════════╣\n"+
		"║  Set ADMIN_USER and ADMIN_PASS env vars to persist.  ║\n"+
		"╚══════════════════════════════════════════════════════╝\n\n",
		username, password)

	return nil
}
