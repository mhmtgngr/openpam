package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/cache"
	"github.com/openpam/openpam/internal/database"
	"github.com/openpam/openpam/internal/events"
	"github.com/openpam/openpam/internal/middleware"
	"github.com/openpam/openpam/internal/session"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Configuration
	config := loadConfig()

	// Setup logging
	setupLogger(config.LogLevel)
	logger := log.With().Str("service", "session-service").Logger()

	// Initialize dependencies
	db, err := database.New(database.Config{
		Host:            config.DBHost,
		Port:            config.DBPort,
		User:            config.DBUser,
		Password:        config.DBPassword,
		Database:        config.DBName,
		SSLMode:         config.DBSSLMode,
		SSLRootCert:     config.DBSSLRootCert,
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

	// Initialize services with secure event bus (HMAC signing)
	eventSigningKey := os.Getenv("EVENT_SIGNING_KEY")
	if len(eventSigningKey) < 32 {
		log.Fatal().Msg("EVENT_SIGNING_KEY environment variable must be set and at least 32 bytes for HMAC-SHA256 security")
	}
	eventBus, err := events.New(events.EventConfig{
		Cache:      redisCache,
		Logger:     logger,
		SigningKey: []byte(eventSigningKey),
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize event bus")
	}
	eventPublisher := events.NewPublisher(eventBus)

	sessionRepo := session.NewRepository(db.DB, redisCache, logger)
	sessionSvc := session.NewService(sessionRepo, redisCache, eventPublisher, logger)

	// Setup router
	router := setupRouter(config, db, redisCache, sessionSvc, logger)

	// Start server
	srv := &http.Server{
		Addr:    ":" + config.Port,
		Handler: router,
	}

	go func() {
		logger.Info().Str("port", config.Port).Msg("Starting session-service server")
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
	DBSSLRootCert string

	RedisHost     string
	RedisPort     int
	RedisPassword string
	RedisDB       int
}

func loadConfig() Config {
	return Config{
		Port:     getEnv("PORT", "8501"),
		LogLevel: getEnv("LOG_LEVEL", "info"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnvInt("DB_PORT", 5432),
		DBUser:     getEnv("DB_USER", "openpam"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "openpam"),
		DBSSLMode:  getEnv("DB_SSLMODE", "require"), // SECURITY: Default to require SSL
		DBSSLRootCert: getEnv("DB_SSLROOTCERT", "/etc/ssl/certs/postgresql-ca.crt"),

		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnvInt("REDIS_PORT", 6379),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvInt("REDIS_DB", 0),
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
	sessionSvc *session.Service,
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
		AllowedOrigins:  []string{"http://localhost:3000", "http://localhost:8580"}, // SECURITY: No wildcard CORS on internal services
		AllowedMethods:  []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:  []string{"Origin", "Content-Type", "Authorization", "X-Request-ID"},
		ExposeHeaders:   []string{"Content-Length", "X-Request-ID"},
	}))
	r.Use(middleware.RequestID())

	// Health endpoints
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "up", "service": "session-service"})
	})
	r.GET("/ready", func(c *gin.Context) {
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
		// Public routes
		public := v1.Group("")
		{
			public.GET("/sessions/:id/stream", handleSessionStream(sessionSvc, logger))
		}

		// Protected routes (require auth)
		protected := v1.Group("")
		protected.Use(middleware.Auth())
		{
			// Session management
			protected.POST("/sessions", handleStartSession(sessionSvc, logger))
			protected.GET("/sessions", handleListSessions(sessionSvc, logger))
			protected.GET("/sessions/:id", handleGetSession(sessionSvc, logger))
			protected.POST("/sessions/:id/end", handleEndSession(sessionSvc, logger))
			protected.POST("/sessions/:id/terminate", handleTerminateSession(sessionSvc, logger))

			// Session activity
			protected.POST("/sessions/:id/activity", handleUpdateActivity(sessionSvc, logger))

			// Recording
			protected.GET("/sessions/:id/recording", handleGetRecording(sessionSvc, logger))

			// Admin routes
			admin := protected.Group("/admin")
			admin.Use(middleware.RequireRole("admin", "super_admin"))
			{
				admin.POST("/sessions/terminate-user", handleTerminateUserSessions(sessionSvc, logger))
				admin.GET("/sessions/stats", handleSessionStats(sessionSvc, logger))
			}
		}
	}

	return r
}

// Handler functions

func handleStartSession(svc *session.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			UserID       uuid.UUID          `json:"user_id" binding:"required"`
			CredentialID uuid.UUID         `json:"credential_id" binding:"required"`
			TargetID     *uuid.UUID        `json:"target_id"`
			Type         string            `json:"type" binding:"required"`
			TargetHost   string            `json:"target_host" binding:"required"`
			TargetPort   int               `json:"target_port" binding:"required"`
			TenantID     uuid.UUID         `json:"tenant_id" binding:"required"`
			ClientIP     string            `json:"client_ip"`
			UserAgent    string            `json:"user_agent"`
			Metadata     json.RawMessage   `json:"metadata"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
			return
		}

		s := &session.Session{
			UserID:       req.UserID,
			CredentialID: req.CredentialID,
			TargetID:     req.TargetID,
			Type:         session.SessionType(req.Type),
			TargetHost:   req.TargetHost,
			TargetPort:   req.TargetPort,
			ClientIP:     req.ClientIP,
			UserAgent:    req.UserAgent,
			Metadata:     req.Metadata,
			TenantID:     req.TenantID,
		}

		if err := svc.StartSession(c.Request.Context(), s); err != nil {
			logger.Error().Err(err).Msg("Failed to start session")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to start session"}})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"session": s})
	}
}

func handleListSessions(svc *session.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, _ := c.Get("tenant_id")
		tenantIDUUID, err := uuid.Parse(tenantID.(string))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
			return
		}

		limit := getIntQuery(c, "limit", 50)
		offset := getIntQuery(c, "offset", 0)

		var filter session.SessionFilter
		if status := c.Query("status"); status != "" {
			filter.Status = (*session.SessionStatus)(&status)
		}
		if sessionType := c.Query("type"); sessionType != "" {
			filter.Type = (*session.SessionType)(&sessionType)
		}
		if userID := c.Query("user_id"); userID != "" {
			uid, _ := uuid.Parse(userID)
			filter.UserID = &uid
		}

		sessions, total, err := svc.ListSessions(c.Request.Context(), tenantIDUUID, filter, limit, offset)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to list sessions")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to list sessions"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"sessions": sessions,
			"total":    total,
			"limit":    limit,
			"offset":   offset,
		})
	}
}

func handleGetSession(svc *session.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid session ID"}})
			return
		}

		s, err := svc.GetSession(c.Request.Context(), id)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to get session")
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Session not found"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"session": s})
	}
}

func handleEndSession(svc *session.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid session ID"}})
			return
		}

		userID, _ := c.Get("user_id")

		var req struct {
			Reason string `json:"reason"`
		}
		c.ShouldBindJSON(&req)

		if err := svc.EndSession(c.Request.Context(), id, nil, req.Reason); err != nil {
			logger.Error().Err(err).Msg("Failed to end session")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to end session"}})
			return
		}

		logger.Info().
			Str("session_id", id.String()).
			Str("user_id", userID.(string)).
			Msg("Session ended")

		c.JSON(http.StatusOK, gin.H{"message": "Session ended"})
	}
}

func handleTerminateSession(svc *session.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid session ID"}})
			return
		}

		terminatedBy, _ := c.Get("user_id")
		terminatedByID, _ := uuid.Parse(terminatedBy.(string))

		var req struct {
			Reason string `json:"reason"`
		}
		_ = c.ShouldBindJSON(&req)

		if err := svc.TerminateSession(c.Request.Context(), id, terminatedByID, req.Reason); err != nil {
			logger.Error().Err(err).Msg("Failed to terminate session")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to terminate session"}})
			return
		}

		logger.Info().
			Str("session_id", id.String()).
			Str("terminated_by", terminatedBy.(string)).
			Str("reason", req.Reason).
			Msg("Session terminated")

		c.JSON(http.StatusOK, gin.H{"message": "Session terminated"})
	}
}

func handleUpdateActivity(svc *session.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid session ID"}})
			return
		}

		if err := svc.UpdateSessionActivity(c.Request.Context(), id); err != nil {
			logger.Error().Err(err).Msg("Failed to update activity")
			// Don't return error, activity updates are optional
		}

		c.JSON(http.StatusOK, gin.H{"message": "Activity updated"})
	}
}

func handleTerminateUserSessions(svc *session.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			UserID uuid.UUID `json:"user_id" binding:"required"`
			Reason string    `json:"reason"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
			return
		}

		terminatedBy, _ := c.Get("user_id")
		terminatedByID, _ := uuid.Parse(terminatedBy.(string))

		if err := svc.TerminateUserSessions(c.Request.Context(), req.UserID, terminatedByID, req.Reason); err != nil {
			logger.Error().Err(err).Msg("Failed to terminate user sessions")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to terminate sessions"}})
			return
		}

		logger.Info().
			Str("target_user_id", req.UserID.String()).
			Str("terminated_by", terminatedBy.(string)).
			Str("reason", req.Reason).
			Msg("All user sessions terminated")

		c.JSON(http.StatusOK, gin.H{"message": "All user sessions terminated"})
	}
}

func handleSessionStats(svc *session.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, _ := c.Get("tenant_id")
		tenantIDUUID, err := uuid.Parse(tenantID.(string))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
			return
		}

		stats, err := svc.GetSessionStats(c.Request.Context(), tenantIDUUID)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to get session stats")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get stats"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"stats": stats})
	}
}

func handleSessionStream(svc *session.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid session ID"}})
			return
		}

		// Check if session is active
		if !svc.IsSessionActive(c.Request.Context(), id) {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Session not active"}})
			return
		}

		// TODO: Implement WebSocket streaming for session I/O
		c.JSON(http.StatusNotImplemented, gin.H{"error": gin.H{"code": "NOT_IMPLEMENTED", "message": "Session streaming not yet implemented"}})
	}
}

func handleGetRecording(svc *session.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid session ID"}})
			return
		}

		s, err := svc.GetSession(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Session not found"}})
			return
		}

		if s.RecordingURL == "" {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "No recording available"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"recording_url": s.RecordingURL,
			"recording_id":  s.RecordingID,
		})
	}
}

func getIntQuery(c *gin.Context, key string, defaultVal int) int {
	val := c.Query(key)
	if val == "" {
		return defaultVal
	}
	var intVal int
	if _, err := fmt.Sscanf(val, "%d", &intVal); err == nil {
		return intVal
	}
	return defaultVal
}
