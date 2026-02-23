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
	"github.com/openpam/openpam/internal/audit"
	"github.com/openpam/openpam/internal/cache"
	"github.com/openpam/openpam/internal/database"
	"github.com/openpam/openpam/internal/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Configuration
	config := loadConfig()

	// Setup logging
	setupLogger(config.LogLevel)
	logger := log.With().Str("service", "audit-service").Logger()

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

	// Initialize audit service
	auditRepo := audit.NewRepository(db.DB, redisCache, logger)
	auditSvc := audit.NewService(auditRepo, logger)

	// Setup router
	router := setupRouter(config, db, redisCache, auditSvc, logger)

	// Start server
	srv := &http.Server{
		Addr:    ":" + config.Port,
		Handler: router,
	}

	go func() {
		logger.Info().Str("port", config.Port).Msg("Starting audit-service server")
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
}

func loadConfig() Config {
	return Config{
		Port:     getEnv("PORT", "8504"),
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
	auditSvc *audit.Service,
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
		c.JSON(http.StatusOK, gin.H{"status": "up", "service": "audit-service"})
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
		// Protected routes (require auth)
		protected := v1.Group("")
		protected.Use(middleware.Auth())
		{
			// Audit events
			protected.GET("/audit/events", handleListAuditEvents(auditSvc, logger))
			protected.GET("/audit/events/:id", handleGetAuditEvent(auditSvc, logger))
			protected.POST("/audit/events", handleCreateAuditEvent(auditSvc, logger))

			// Export
			protected.GET("/audit/export", handleExportAuditEvents(auditSvc, logger))

			// Compliance
			protected.GET("/audit/compliance/report", handleComplianceReport(auditSvc, logger))

			// Integrity
			protected.GET("/audit/integrity/verify", handleVerifyIntegrity(auditSvc, logger))

			// Admin routes
			admin := protected.Group("/admin")
			admin.Use(middleware.RequireRole("admin", "super_admin"))
			{
				admin.GET("/audit/stats", handleAuditStats(auditSvc, logger))
			}
		}
	}

	return r
}

// Handler functions

func handleListAuditEvents(svc *audit.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, _ := c.Get("tenant_id")
		tenantIDUUID, err := uuid.Parse(tenantID.(string))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
			return
		}

		limit := getIntQuery(c, "limit", 50)
		offset := getIntQuery(c, "offset", 0)

		var filter audit.EventFilter
		if actorID := c.Query("actor_id"); actorID != "" {
			uid, _ := uuid.Parse(actorID)
			filter.ActorID = &uid
		}
		if action := c.Query("action"); action != "" {
			filter.Action = &action
		}
		if resourceType := c.Query("resource_type"); resourceType != "" {
			filter.ResourceType = &resourceType
		}
		if outcome := c.Query("outcome"); outcome != "" {
			o := audit.EventOutcome(outcome)
			filter.Outcome = &o
		}

		events, total, err := svc.Query(c.Request.Context(), tenantIDUUID, filter, limit, offset)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to list audit events")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to list events"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"events": events,
			"total":  total,
			"limit":  limit,
			"offset": offset,
		})
	}
}

func handleGetAuditEvent(svc *audit.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid event ID"}})
			return
		}

		// Need to use repository directly for GetByID
		c.JSON(http.StatusNotImplemented, gin.H{"error": gin.H{"code": "NOT_IMPLEMENTED", "message": "Use list endpoint"}})
	}
}

func handleCreateAuditEvent(svc *audit.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			TenantID     uuid.UUID       `json:"tenant_id" binding:"required"`
			ActorID      uuid.UUID       `json:"actor_id" binding:"required"`
			ActorType    string          `json:"actor_type" binding:"required"`
			Action       string          `json:"action" binding:"required"`
			ResourceType string          `json:"resource_type" binding:"required"`
			ResourceID   string          `json:"resource_id"`
			Outcome      string          `json:"outcome" binding:"required"`
			IP           string          `json:"ip"`
			UserAgent    string          `json:"user_agent"`
			Details      json.RawMessage `json:"details"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
			return
		}

		event := &audit.Event{
			TenantID:     req.TenantID,
			ActorID:      req.ActorID,
			ActorType:    req.ActorType,
			Action:       req.Action,
			ResourceType: req.ResourceType,
			ResourceID:   req.ResourceID,
			Outcome:      audit.EventOutcome(req.Outcome),
			IP:           req.IP,
			UserAgent:    req.UserAgent,
			Details:      req.Details,
		}

		if err := svc.Log(c.Request.Context(), event); err != nil {
			logger.Error().Err(err).Msg("Failed to create audit event")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to create event"}})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"event": event})
	}
}

func handleExportAuditEvents(svc *audit.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, _ := c.Get("tenant_id")
		tenantIDUUID, err := uuid.Parse(tenantID.(string))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
			return
		}

		var filter audit.EventFilter
		data, _, err := svc.Query(c.Request.Context(), tenantIDUUID, filter, 10000, 0)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to export audit events")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to export events"}})
			return
		}

		c.Header("Content-Type", "application/json")
		c.Header("Content-Disposition", "attachment; filename=audit_export.json")
		json.NewEncoder(c.Writer).Encode(data)
	}
}

func handleComplianceReport(svc *audit.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, _ := c.Get("tenant_id")
		tenantIDUUID, err := uuid.Parse(tenantID.(string))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
			return
		}

		// Parse time range
		startTime := time.Now().AddDate(0, -1, 0) // Default: last 30 days
		endTime := time.Now()

		if startStr := c.Query("start"); startStr != "" {
			if t, err := time.Parse(time.RFC3339, startStr); err == nil {
				startTime = t
			}
		}
		if endStr := c.Query("end"); endStr != "" {
			if t, err := time.Parse(time.RFC3339, endStr); err == nil {
				endTime = t
			}
		}

		report, err := svc.GenerateComplianceReport(c.Request.Context(), tenantIDUUID, startTime, endTime)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to generate compliance report")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to generate report"}})
			return
		}

		// Verify integrity for the report
		valid, errors, _ := svc.VerifyIntegrity(c.Request.Context(), tenantIDUUID)
		report.IntegrityValid = valid
		report.IntegrityErrors = errors

		c.JSON(http.StatusOK, gin.H{"report": report})
	}
}

func handleVerifyIntegrity(svc *audit.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, _ := c.Get("tenant_id")
		tenantIDUUID, err := uuid.Parse(tenantID.(string))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
			return
		}

		valid, errors, err := svc.VerifyIntegrity(c.Request.Context(), tenantIDUUID)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to verify integrity")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to verify integrity"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"valid":  valid,
			"errors": errors,
		})
	}
}

func handleAuditStats(svc *audit.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, _ := c.Get("tenant_id")
		tenantIDUUID, err := uuid.Parse(tenantID.(string))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
			return
		}

		// Get events for last 30 days for stats
		endTime := time.Now()
		startTime := endTime.AddDate(0, -1, 0)

		report, err := svc.GenerateComplianceReport(c.Request.Context(), tenantIDUUID, startTime, endTime)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to get audit stats")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get stats"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"stats": gin.H{
				"total_events":        report.TotalEvents,
				"successful_events":   report.SuccessfulEvents,
				"failed_events":       report.FailedEvents,
				"denied_events":       report.DeniedEvents,
				"event_types":         report.EventTypes,
				"credential_events":   report.CredentialEvents,
				"session_events":      report.SessionEvents,
				"user_events":         report.UserEvents,
			},
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
