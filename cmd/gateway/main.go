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
	"github.com/openpam/openpam/internal/auth"
	"github.com/openpam/openpam/internal/cache"
	"github.com/openpam/openpam/internal/database"
	"github.com/openpam/openpam/internal/events"
	"github.com/openpam/openpam/internal/middleware"
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

	jwtManager := auth.NewJWTManager(/* load RSA keys */ nil, nil, redisCache, logger)

	totpManager := auth.NewTOTPManager(auth.TOTPConfig{
		Issuer:     "OpenPAM",
		Algorithm:  "SHA256",
		Digits:     6,
		Period:     30,
		SecretSize: 32,
		Leeway:     1,
	}, redisCache, logger)

	mfaManager := auth.NewMFAManager(totpManager, nil, redisCache, logger)

	// Setup router
	router := setupRouter(config, db, redisCache, jwtManager, mfaManager, eventPublisher, logger)

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
	logger zerolog.Logger,
) *gin.Engine {
	if config.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// Middleware
	r.Use(middleware.Logger(logger))
	r.Use(middleware.Recovery(logger))
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(middleware.Config{
		AllowedOrigins:  []string{"*"},
		AllowedMethods:  []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:  []string{"Origin", "Content-Type", "Authorization", "X-Request-ID"},
		ExposeHeaders:   []string{"Content-Length", "X-Request-ID"},
	}))
	r.Use(middleware.RequestID())

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

	// API v1 routes
	v1 := r.Group("/api/v1")
	{
		// Public routes (no auth)
		public := v1.Group("")
		{
			public.POST("/auth/login", handleLogin(jwt, mfa, db, logger))
			public.POST("/auth/logout", handleLogout(cache, logger))
			public.POST("/auth/refresh", handleRefresh(jwt, cache, logger))
			public.POST("/auth/mfa/verify", handleMFAVerify(mfa, cache, logger))
		}

		// Protected routes (require auth)
		protected := v1.Group("")
		protected.Use(middleware.Auth())
		{
			// User management
			protected.GET("/users", handleListUsers(db, logger))
			protected.GET("/users/:id", handleGetUser(db, logger))
			protected.POST("/users", handleCreateUser(db, logger))
			protected.PUT("/users/:id", handleUpdateUser(db, logger))
			protected.DELETE("/users/:id", handleDeleteUser(db, logger))

			// Targets
			protected.GET("/targets", handleListTargets(db, logger))
			protected.GET("/targets/:id", handleGetTarget(db, logger))
			protected.POST("/targets", handleCreateTarget(db, logger))
			protected.PUT("/targets/:id", handleUpdateTarget(db, logger))
			protected.DELETE("/targets/:id", handleDeleteTarget(db, logger))

			// Credentials
			protected.GET("/credentials", handleListCredentials(db, logger))
			protected.GET("/credentials/:id", handleGetCredential(db, logger))
			protected.POST("/credentials", handleCreateCredential(db, logger))
			protected.PUT("/credentials/:id", handleUpdateCredential(db, logger))
			protected.DELETE("/credentials/:id", handleDeleteCredential(db, logger))

			// Checkouts
			protected.GET("/checkouts", handleListCheckouts(db, logger))
			protected.POST("/checkouts", handleCreateCheckout(db, publisher, logger))
			protected.POST("/checkouts/:id/checkout", handleCheckoutCredential(db, logger))
			protected.POST("/checkouts/:id/checkin", handleCheckinCredential(db, logger))

			// Sessions
			protected.GET("/sessions", handleListSessions(db, logger))
			protected.GET("/sessions/:id", handleGetSession(db, logger))
			protected.POST("/sessions/:id/terminate", handleTerminateSession(db, logger))

			// Approval requests
			protected.GET("/approvals/requests", handleListApprovalRequests(db, logger))
			protected.GET("/approvals/requests/:id", handleGetApprovalRequest(db, logger))
			protected.POST("/approvals/requests", handleCreateApprovalRequest(db, logger))
			protected.POST("/approvals/requests/:id/approve", handleApproveRequest(db, logger))
			protected.POST("/approvals/requests/:id/deny", handleDenyRequest(db, logger))

			// Audit logs
			protected.GET("/audit/events", handleListAuditEvents(db, logger))
			protected.GET("/audit/events/:id", handleGetAuditEvent(db, logger))
			protected.GET("/audit/export", handleExportAuditEvents(db, logger))

			// Discovery
			protected.GET("/discovery/scans", handleListDiscoveryScans(db, logger))
			protected.POST("/discovery/scans", handleCreateDiscoveryScan(db, logger))
			protected.POST("/discovery/scans/:id/run", handleRunDiscoveryScan(db, logger))
			protected.GET("/discovery/assets", handleListDiscoveredAssets(db, logger))

			// Admin routes (require admin role)
			admin := protected.Group("/admin")
			admin.Use(middleware.RequireRole("admin", "super_admin"))
			{
				admin.GET("/tenants", handleListTenants(db, logger))
				admin.POST("/tenants", handleCreateTenant(db, logger))
				admin.PUT("/tenants/:id", handleUpdateTenant(db, logger))
				admin.GET("/stats", handleSystemStats(db, logger))
			}
		}
	}

	return r
}

// Handler functions

func handleLogin(jwt *auth.JWTManager, mfa *auth.MFAManager, db *database.DB, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Email    string `json:"email" binding:"required"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
			return
		}

		// Authenticate user
		// user, err := userService.Authenticate(c.Request.Context(), req.Email, req.Password)
		// if err != nil {
		// 	c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "Invalid credentials"}})
		// 	return
		// }

		// Generate tokens
		// accessToken, refreshToken, err := jwt.GenerateTokenPair(...)
		// if err != nil {
		// 	c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to generate tokens"}})
		// 	return
		// }

		c.JSON(http.StatusOK, gin.H{
			"access_token":  "accessToken",
			"refresh_token": "refreshToken",
			"user":          nil, // user info without password
		})
	}
}

func handleLogout(cache *cache.Cache, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Revoke refresh token
		c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
	}
}

func handleRefresh(jwt *auth.JWTManager, cache *cache.Cache, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Refresh access token using refresh token
		c.JSON(http.StatusOK, gin.H{"access_token": "newAccessToken"})
	}
}

func handleMFAVerify(mfa *auth.MFAManager, cache *cache.Cache, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Code  string `json:"code" binding:"required"`
			Token string `json:"token" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
			return
		}

		// Verify MFA code
		c.JSON(http.StatusOK, gin.H{"verified": true})
	}
}

func handleListUsers(db *database.DB, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"users": []interface{}{}})
	}
}

func handleGetUser(db *database.DB, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"user": nil})
	}
}

func handleCreateUser(db *database.DB, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"user": nil})
	}
}

func handleUpdateUser(db *database.DB, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"user": nil})
	}
}

func handleDeleteUser(db *database.DB, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
	}
}

func handleListTargets(db *database.DB, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"targets": []interface{}{}, "total": 0})
	}
}

func handleGetTarget(db *database.DB, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"target": nil})
	}
}

func handleCreateTarget(db *database.DB, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"target": nil})
	}
}

func handleUpdateTarget(db *database.DB, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"target": nil})
	}
}

func handleDeleteTarget(db *database.DB, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Target deleted"})
	}
}

func handleListCredentials(db *database.DB, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"credentials": []interface{}{}, "total": 0})
	}
}

func handleGetCredential(db *database.DB, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"credential": nil})
	}
}

func handleCreateCredential(db *database.DB, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"credential": nil})
	}
}

func handleUpdateCredential(db *database.DB, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"credential": nil})
	}
}

func handleDeleteCredential(db *database.DB, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Credential deleted"})
	}
}

func handleListCheckouts(db *database.DB, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"checkouts": []interface{}{}, "total": 0})
	}
}

func handleCreateCheckout(db *database.DB, publisher *events.Publisher, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"checkout": nil})
	}
}

func handleCheckoutCredential(db *database.DB, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"credential": nil})
	}
}

func handleCheckinCredential(db *database.DB, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Credential checked in"})
	}
}

func handleListSessions(db *database.DB, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"sessions": []interface{}{}, "total": 0})
	}
}

func handleGetSession(db *database.DB, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"session": nil})
	}
}

func handleTerminateSession(db *database.DB, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Session terminated"})
	}
}

func handleListApprovalRequests(db *database.DB, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"requests": []interface{}{}, "total": 0})
	}
}

func handleGetApprovalRequest(db *database.DB, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"request": nil})
	}
}

func handleCreateApprovalRequest(db *database.DB, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"request": nil})
	}
}

func handleApproveRequest(db *database.DB, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Request approved"})
	}
}

func handleDenyRequest(db *database.DB, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Request denied"})
	}
}

func handleListAuditEvents(db *database.DB, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"events": []interface{}{}, "total": 0})
	}
}

func handleGetAuditEvent(db *database.DB, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"event": nil})
	}
}

func handleExportAuditEvents(db *database.DB, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Content-Type", "application/json")
		c.Header("Content-Disposition", "attachment; filename=audit_export.json")
		c.String(http.StatusOK, "[]")
	}
}

func handleListDiscoveryScans(db *database.DB, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"scans": []interface{}{}, "total": 0})
	}
}

func handleCreateDiscoveryScan(db *database.DB, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"scan": nil})
	}
}

func handleRunDiscoveryScan(db *database.DB, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Scan started"})
	}
}

func handleListDiscoveredAssets(db *database.DB, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"assets": []interface{}{}, "total": 0})
	}
}

func handleListTenants(db *database.DB, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"tenants": []interface{}{}, "total": 0})
	}
}

func handleCreateTenant(db *database.DB, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"tenant": nil})
	}
}

func handleUpdateTenant(db *database.DB, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"tenant": nil})
	}
}

func handleSystemStats(db *database.DB, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"stats": nil})
	}
}
