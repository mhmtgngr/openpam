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
	"github.com/openpam/openpam/cmd/gateway/handlers"
	"github.com/openpam/openpam/internal/auth"
	"github.com/openpam/openpam/internal/audit"
	"github.com/openpam/openpam/internal/cache"
	"github.com/openpam/openpam/internal/database"
	"github.com/openpam/openpam/internal/events"
	middleware2 "github.com/openpam/openpam/internal/middleware"
	"github.com/openpam/openpam/internal/pam/approval"
	"github.com/openpam/openpam/internal/pam/target"
	"github.com/openpam/openpam/internal/pam/vault"
	"github.com/openpam/openpam/internal/session"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Configuration
	config := loadConfig()

	// Setup logging
	setupLogger(config.LogLevel)
	logger := log.With().Str("service", "gateway").Logger()

	// Initialize dependencies
	db, err := database.New(database.Config{
		Host:            config.DBHost,
		Port:            config.DBPort,
		User:            config.DBUser,
		Password:        config.DBPassword,
		Database:        config.DBName,
		SSLMode:         config.DBSSLMode,
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: time.Minute * 5,
	}, logger)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}

	redisCache, err := cache.New(cache.Config{
		Host:     config.RedisHost,
		Port:     config.RedisPort,
		Password: config.RedisPassword,
		DB:       config.RedisDB,
		PoolSize: 100,
	}, logger)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to Redis")
	}

	// Initialize services
	eventBus := events.New(redisCache, logger)
	eventPublisher := events.NewPublisher(eventBus)

	// Load or generate RSA keys for JWT
	privateKey, publicKey, err := auth.LoadOrGenerateRSAKeys(config.JWTPrivateKeyPath, config.JWTPublicKeyPath, logger)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize RSA keys")
	}

	jwtManager := auth.NewJWTManager(privateKey, publicKey, redisCache, logger)

	totpManager := auth.NewTOTPManager(auth.TOTPConfig{
		Issuer:     "OpenPAM",
		Algorithm:  "SHA256",
		Digits:     6,
		Period:     30,
		SecretSize: 32,
		Leeway:     1,
	}, redisCache, logger)

	mfaManager := auth.NewMFAManager(totpManager, nil, redisCache, logger)

	authService := auth.NewService(db.DB, jwtManager, mfaManager, redisCache, logger)

	// Initialize services
	targetRepo := target.NewTargetRepository(db.DB, redisCache, logger)
	targetService := target.NewTargetService(targetRepo, logger)

	// Initialize vault service
	secretRepo := vault.NewSecretRepository(db.DB, redisCache, logger)
	// For now, generate a master key from environment or use a default
	// In production, load from secure storage like HSM or KMS
	masterKey := make([]byte, 32)
	masterKeyStr := getEnv("VAULT_MASTER_KEY", "")
	if masterKeyStr != "" {
		// Use provided key (should be 64 hex chars for 32 bytes)
		for i := 0; i < 32 && i*2 < len(masterKeyStr); i++ {
			var b byte
			if _, err := fmt.Sscanf(masterKeyStr[i*2:i*2+2], "%02x", &b); err == nil {
				masterKey[i] = b
			}
		}
	} else {
		// Fallback: use a hash of the JWT private key path as seed
		// This is NOT secure for production - replace with proper key management
		seed := getEnv("JWT_PRIVATE_KEY_PATH", "/etc/openpam/jwt/private.pem")
		for i, c := range seed {
			masterKey[i%32] += byte(c)
		}
		logger.Warn().Msg("Using insecure master key generation - set VAULT_MASTER_KEY in production")
	}
	envelopeEncryption, err := vault.NewEnvelopeEncryption(masterKey, "master-1", db.DB, redisCache, logger)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize envelope encryption")
	}
	vaultService := vault.NewVaultService(secretRepo, envelopeEncryption, logger)

	// Initialize session service
	sessionRepo := session.NewRepository(db.DB, redisCache, logger)
	sessionService := session.NewService(sessionRepo, redisCache, eventPublisher, logger)

	// Initialize audit service
	auditRepo := audit.NewRepository(db.DB, redisCache, logger)
	auditService := audit.NewService(auditRepo, logger)

	// Initialize approval service
	approvalRepo := approval.NewRepository(db.DB, redisCache, logger)
	approvalService := approval.NewWorkflowService(approvalRepo, eventPublisher, redisCache, logger)

	// Setup router
	router := setupRouter(config, db, redisCache, jwtManager, mfaManager, eventPublisher, authService, targetService, vaultService, sessionService, auditService, approvalService, logger)

	// Start server
	srv := &http.Server{
		Addr:    ":" + config.Port,
		Handler: router,
	}

	go func() {
		logger.Info().Str("port", config.Port).Msg("Starting gateway server")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server failed")
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info().Msg("Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error().Err(err).Msg("Forced shutdown")
	}

	logger.Info().Msg("Goodbye.")
}

// Config holds application configuration
type Config struct {
	Port     string
	LogLevel string

	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	RedisHost     string
	RedisPort     int
	RedisPassword string
	RedisDB       int

	JWTPrivateKeyPath string
	JWTPublicKeyPath  string
}

func loadConfig() Config {
	return Config{
		Port:     getEnv("PORT", "8500"),
		LogLevel: getEnv("LOG_LEVEL", "info"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnvInt("DB_PORT", 5432),
		DBUser:     getEnv("DB_USER", "openpam"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "openpam"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),

		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnvInt("REDIS_PORT", 6379),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvInt("REDIS_DB", 0),

		JWTPrivateKeyPath: getEnv("JWT_PRIVATE_KEY_PATH", "/etc/openpam/jwt/private.pem"),
		JWTPublicKeyPath:  getEnv("JWT_PUBLIC_KEY_PATH", "/etc/openpam/jwt/public.pem"),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		var intVal int
		if _, err := fmt.Sscanf(val, "%d", &intVal); err == nil {
			return intVal
		}
	}
	return defaultVal
}

func setupLogger(level string) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(lvl)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}

func setupRouter(
	config Config,
	db *database.DB,
	cache *cache.Cache,
	jwt *auth.JWTManager,
	mfa *auth.MFAManager,
	publisher *events.Publisher,
	authService *auth.Service,
	targetService *target.TargetService,
	vaultService *vault.VaultService,
	sessionService *session.Service,
	auditService *audit.Service,
	approvalService *approval.WorkflowService,
	logger zerolog.Logger,
) *gin.Engine {
	if config.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// Initialize rate limiter
	rateLimiter := middleware2.NewRateLimiter(cache, logger)

	// Middleware
	r.Use(middleware2.Logger(logger))
	r.Use(middleware2.Recovery(logger))
	r.Use(middleware2.SecurityHeaders())
	r.Use(middleware2.CORS(middleware2.Config{
		AllowedOrigins:  []string{"*"},
		AllowedMethods:  []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:  []string{"Origin", "Content-Type", "Authorization", "X-Request-ID"},
		ExposeHeaders:   []string{"Content-Length", "X-Request-ID"},
	}))
	r.Use(middleware2.RequestID())

	// Health endpoints
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "up", "service": "gateway"})
	})
	r.GET("/ready", func(c *gin.Context) {
		// Check dependencies
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		ready := true
		details := gin.H{}

		if err := db.Health(ctx); err != nil {
			ready = false
			details["database"] = "unhealthy"
		} else {
			details["database"] = "healthy"
		}

		if err := cache.Health(ctx); err != nil {
			ready = false
			details["cache"] = "unhealthy"
		} else {
			details["cache"] = "healthy"
		}

		if ready {
			c.JSON(http.StatusOK, gin.H{"ready": true, "details": details})
		} else {
			c.JSON(http.StatusServiceUnavailable, gin.H{"ready": false, "details": details})
		}
	})

	// Create handlers
	authHandler := handlers.NewAuthHandler(authService, logger)
	targetHandler := handlers.NewTargetHandler(targetService, logger)
	credentialHandler := handlers.NewCredentialHandler(vaultService, logger)
	sessionHandler := handlers.NewSessionHandler(sessionService, logger)
	auditHandler := handlers.NewAuditHandler(auditService, logger)
	approvalHandler := handlers.NewApprovalHandler(approvalService, logger)

	// API v1 routes
	v1 := r.Group("/api/v1")
	{
		// Public routes (no auth)
		public := v1.Group("")
		{
			// Use stricter rate limiting for auth endpoints
			public.Use(rateLimiter.RateLimit(middleware2.AuthRateLimitConfig()))

			public.POST("/auth/login", authHandler.Login)
			public.POST("/auth/logout", authHandler.Logout)
			public.POST("/auth/refresh", authHandler.Refresh)
			public.POST("/auth/verify-mfa", authHandler.VerifyMFA)
		}

		// Protected routes (require auth)
		protected := v1.Group("")
		protected.Use(middleware2.Auth())
		{
			// Auth routes
			protected.GET("/auth/me", authHandler.Me)
			protected.POST("/auth/mfa/setup", authHandler.SetupMFA)
			protected.POST("/auth/mfa/verify", authHandler.VerifyAndEnableMFA)
			protected.POST("/auth/change-password", authHandler.ChangePassword)
			protected.GET("/auth/backup-codes", authHandler.GetBackupCodes)

			// User management (privileged)
			users := protected.Group("/users")
			users.Use(middleware2.RequireRole("admin", "super_admin"))
			{
				users.GET("", authHandler.ListUsers)
				users.POST("", authHandler.Register)
				users.GET("/:id", authHandler.Me) // Reuse Me for single user
				users.PUT("/:id", authHandler.UpdateUser)
				users.DELETE("/:id", authHandler.DeleteUser)
			}

			// Targets
			protected.GET("/targets", targetHandler.List)
			protected.GET("/targets/:id", targetHandler.Get)
			protected.POST("/targets", middleware2.RequireRole("admin", "super_admin"), targetHandler.Create)
			protected.PUT("/targets/:id", middleware2.RequireRole("admin", "super_admin"), targetHandler.Update)
			protected.DELETE("/targets/:id", middleware2.RequireRole("admin", "super_admin"), targetHandler.Delete)
			protected.POST("/targets/:id/verify", targetHandler.Verify)

			// Credentials
			protected.GET("/credentials", credentialHandler.List)
			protected.GET("/credentials/:id", credentialHandler.Get)
			protected.GET("/credentials/:id/reveal", credentialHandler.Reveal)
			protected.POST("/credentials", middleware2.RequireRole("admin", "super_admin"), credentialHandler.Create)
			protected.PUT("/credentials/:id", middleware2.RequireRole("admin", "super_admin"), credentialHandler.Update)
			protected.DELETE("/credentials/:id", middleware2.RequireRole("admin", "super_admin"), credentialHandler.Delete)
			protected.POST("/credentials/:id/rotate", credentialHandler.Rotate)
			protected.POST("/credentials/:id/compromised", credentialHandler.MarkCompromised)

			// Checkouts (placeholder - to be implemented)
			protected.GET("/checkouts", handleListCheckouts)
			protected.POST("/checkouts", handleCreateCheckout)
			protected.POST("/checkouts/:id/checkout", handleCheckoutCredential)
			protected.POST("/checkouts/:id/checkin", handleCheckinCredential)

			// Sessions
			protected.GET("/sessions", sessionHandler.List)
			protected.GET("/sessions/:id", sessionHandler.Get)
			protected.POST("/sessions", sessionHandler.Create)
			protected.POST("/sessions/:id/terminate", sessionHandler.Terminate)
			protected.GET("/sessions/active", sessionHandler.GetActive)
			protected.GET("/sessions/stats", sessionHandler.GetStats)

			// Approval requests
			protected.GET("/approvals/requests", approvalHandler.List)
			protected.GET("/approvals/requests/pending", approvalHandler.GetPending)
			protected.GET("/approvals/requests/:id", approvalHandler.Get)
			protected.POST("/approvals/requests", approvalHandler.Create)
			protected.POST("/approvals/requests/:id/approve", approvalHandler.Approve)
			protected.POST("/approvals/requests/:id/deny", approvalHandler.Approve)
			protected.POST("/approvals/requests/:id/cancel", approvalHandler.Cancel)
			protected.POST("/approvals/requests/:id/delegate", approvalHandler.Delegate)

			// Audit logs
			protected.GET("/audit/events", auditHandler.List)
			protected.GET("/audit/events/:id", auditHandler.Get)
			protected.GET("/audit/export", auditHandler.Export)
			protected.GET("/audit/integrity", auditHandler.VerifyIntegrity)
			protected.GET("/audit/compliance/report", auditHandler.GenerateComplianceReport)
			protected.GET("/audit/stats", auditHandler.GetStats)

			// Discovery
			protected.GET("/discovery/scans", handleListDiscoveryScans)
			protected.POST("/discovery/scans", handleCreateDiscoveryScan)
			protected.POST("/discovery/scans/:id/run", handleRunDiscoveryScan)
			protected.GET("/discovery/assets", handleListDiscoveredAssets)

			// Admin routes (require admin role)
			admin := protected.Group("/admin")
			admin.Use(middleware2.RequireRole("admin", "super_admin"))
			{
				admin.GET("/tenants", handleListTenants)
				admin.POST("/tenants", handleCreateTenant)
				admin.PUT("/tenants/:id", handleUpdateTenant)
				admin.GET("/stats", handleSystemStats)
			}
		}
	}

	return r
}

// Placeholder handlers for routes not yet implemented
// These will be replaced with proper handler implementations

var (
	handleListCheckouts          = notImplemented
	handleCreateCheckout         = notImplemented
	handleCheckoutCredential     = notImplemented
	handleCheckinCredential      = notImplemented
	handleListDiscoveryScans     = notImplemented
	handleCreateDiscoveryScan    = notImplemented
	handleRunDiscoveryScan       = notImplemented
	handleListDiscoveredAssets   = notImplemented
	handleListTenants            = notImplemented
	handleCreateTenant           = notImplemented
	handleUpdateTenant           = notImplemented
	handleSystemStats            = notImplemented
)

// notImplemented returns a 501 Not Implemented response
func notImplemented(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": gin.H{
			"code":    "NOT_IMPLEMENTED",
			"message": "This endpoint is not yet implemented",
		},
	})
}
