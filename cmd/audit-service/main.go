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
	analyticsmiddleware "github.com/openpam/openpam/internal/audit/middleware"
	"github.com/openpam/openpam/internal/audit/model"
	"github.com/openpam/openpam/internal/cache"
	"github.com/openpam/openpam/internal/database"
	"github.com/openpam/openpam/internal/middleware"
	pamanalytics "github.com/openpam/openpam/internal/pam/analytics"
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

	// Initialize audit service
	auditRepo := audit.NewRepository(db.DB, redisCache, logger)
	auditSvc := audit.NewService(auditRepo, logger)

	// Initialize analytics service (audit analytics for anomaly persistence)
	analyticsSvc := audit.NewAnalyticsService(db.DB, redisCache, logger)

	// Wire up report generator with storage configuration
	reportGenerator := audit.NewReportGenerator(db.DB, logger, &audit.StorageConfig{
		BaseURL:       config.ReportStorageBaseURL,
		StoragePath:   config.ReportStoragePath,
		MaxFileSize:   config.ReportStorageMaxSize,
		RetentionDays: config.ReportRetentionDays,
	})
	analyticsSvc.SetReportGenerator(reportGenerator)

	// Initialize PAM analytics service
	pamAnalyticsRepo := pamanalytics.NewRepository(db.DB, logger)
	pamAnomalyRepo := pamanalytics.NewAnomalyRepository(db.DB, logger)
	pamReportRepo := pamanalytics.NewReportRepository(db.DB, logger)
	pamAnalyticsSvc := pamanalytics.NewService(pamAnalyticsRepo, pamAnomalyRepo, pamReportRepo, redisCache, logger)

	// Setup router
	router := setupRouter(config, db, redisCache, auditSvc, analyticsSvc, pamAnalyticsSvc, logger)

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
	DBSSLRootCert string

	RedisHost     string
	RedisPort     int
	RedisPassword string
	RedisDB       int

	// Report Storage Configuration
	ReportStorageBaseURL   string
	ReportStoragePath      string
	ReportStorageMaxSize   int64
	ReportRetentionDays    int
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
		DBSSLMode:  getEnv("DB_SSLMODE", "require"), // SECURITY: Default to require SSL
		DBSSLRootCert: getEnv("DB_SSLROOTCERT", "/etc/ssl/certs/postgresql-ca.crt"),

		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnvInt("REDIS_PORT", 6379),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvInt("REDIS_DB", 0),

		// Report storage defaults
		ReportStorageBaseURL:  getEnv("REPORT_STORAGE_BASE_URL", "/api/v1/analytics/reports/download"),
		ReportStoragePath:     getEnv("REPORT_STORAGE_PATH", "/var/lib/openpam/reports"),
		ReportStorageMaxSize:  int64(getEnvInt("REPORT_STORAGE_MAX_SIZE_MB", 100) * 1024 * 1024),
		ReportRetentionDays:   getEnvInt("REPORT_RETENTION_DAYS", 90),
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
	analyticsSvc *audit.AnalyticsService,
	pamAnalyticsSvc *pamanalytics.Service,
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

			// Analytics routes (with tenant and correlation middleware)
			analytics := protected.Group("/analytics")
			analytics.Use(analyticsmiddleware.RequireTenantID())
			analytics.Use(analyticsmiddleware.CorrelationID())
			{
				// Dashboard
				analytics.GET("/dashboard", handleAnalyticsDashboard(analyticsSvc, logger))
				analytics.GET("/summary", handleAnalyticsSummary(analyticsSvc, logger))

				// Anomalies
				analytics.GET("/anomalies", handleListAnomalies(analyticsSvc, logger))
				analytics.GET("/anomalies/:id", handleGetAnomaly(analyticsSvc, logger))
				analytics.PUT("/anomalies/:id", handleUpdateAnomaly(analyticsSvc, logger))
				analytics.DELETE("/anomalies/:id", handleDeleteAnomaly(analyticsSvc, logger))
				analytics.GET("/anomalies/stats", handleGetAnomalyStats(analyticsSvc, logger))
				analytics.POST("/anomalies/detect", handleRunAnomalyDetection(pamAnalyticsSvc, logger))

				// Anomaly acknowledge/resolve operations
				analytics.POST("/anomalies/:id/acknowledge", handleAcknowledgeAnomaly(pamAnalyticsSvc, logger))
				analytics.POST("/anomalies/:id/resolve", handleResolveAnomaly(pamAnalyticsSvc, logger))
				analytics.POST("/anomalies/:id/merge", handleMergeDuplicateAnomalies(analyticsSvc, logger))

				// Anomaly bulk operations
				analytics.PUT("/anomalies/bulk", handleBulkUpdateAnomalies(analyticsSvc, logger))

				// Anomaly correlation and deduplication
				analytics.GET("/anomalies/correlation/:id", handleGetAnomaliesByCorrelation(analyticsSvc, logger))
				analytics.GET("/anomalies/types", handleGetAnomalyTypes(analyticsSvc, logger))
				analytics.GET("/anomalies/top-users", handleGetTopAnomalyUsers(analyticsSvc, logger))

				// SSH Key Analytics
				analytics.GET("/ssh-keys/:id/analytics", handleGetSSHKeyAnalytics(analyticsSvc, logger))
				analytics.GET("/ssh-keys/analytics", handleListSSHKeyAnalytics(analyticsSvc, logger))
				analytics.GET("/ssh-keys/:id/summary", handleGetSSHKeyUsageSummary(analyticsSvc, logger))
				analytics.GET("/ssh-keys/most-used", handleGetMostUsedSSHKeys(analyticsSvc, logger))
				analytics.GET("/ssh-keys/anomalous", handleGetAnomalousSSHKeys(analyticsSvc, logger))

				// Command Blacklist
				analytics.POST("/blacklist", handleCreateCommandBlacklist(analyticsSvc, logger))
				analytics.GET("/blacklist", handleListCommandBlacklist(analyticsSvc, logger))
				analytics.GET("/blacklist/:id", handleGetCommandBlacklist(analyticsSvc, logger))
				analytics.PUT("/blacklist/:id", handleUpdateCommandBlacklist(analyticsSvc, logger))
				analytics.DELETE("/blacklist/:id", handleDeleteCommandBlacklist(analyticsSvc, logger))
				analytics.GET("/blacklist/stats", handleGetBlacklistStats(analyticsSvc, logger))
				analytics.POST("/blacklist/:id/enable", handleEnableCommandBlacklist(analyticsSvc, logger))
				analytics.POST("/blacklist/:id/disable", handleDisableCommandBlacklist(analyticsSvc, logger))

				// Compliance Reports
				analytics.GET("/reports", handleListComplianceReports(pamAnalyticsSvc, logger))
				analytics.GET("/reports/:id", handleGetComplianceReport(pamAnalyticsSvc, logger))
				analytics.POST("/reports", handleGenerateComplianceReport(pamAnalyticsSvc, logger))
				analytics.DELETE("/reports/:id", handleDeleteComplianceReport(pamAnalyticsSvc, logger))

				// Report Snapshots
				analytics.GET("/snapshots", handleListReportSnapshots(pamAnalyticsSvc, logger))
				analytics.GET("/snapshots/:id", handleGetReportSnapshot(pamAnalyticsSvc, logger))
				analytics.GET("/snapshots/stats", handleGetReportSnapshotStats(pamAnalyticsSvc, logger))
			}

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

// =============================================================================
// Analytics Handler Functions
// =============================================================================

func handleAnalyticsDashboard(svc *audit.AnalyticsService, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := getTenantUUID(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
			return
		}

		summary, err := svc.GetDashboardSummary(c.Request.Context(), tenantID)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to get dashboard summary")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get dashboard summary"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"summary": summary})
	}
}

func handleAnalyticsSummary(svc *audit.AnalyticsService, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := getTenantUUID(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
			return
		}

		summary, err := svc.GetDashboardSummary(c.Request.Context(), tenantID)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to get analytics summary")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get summary"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"summary": summary})
	}
}

// Anomaly handlers

func handleListAnomalies(svc *audit.AnalyticsService, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := getTenantUUID(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
			return
		}

		limit := getIntQuery(c, "limit", 50)
		offset := getIntQuery(c, "offset", 0)

		// Build filter
		var filter model.AnomalyFilter
		if userID := c.Query("user_id"); userID != "" {
			if uid, err := uuid.Parse(userID); err == nil {
				filter.UserID = &uid
			}
		}
		if anomalyType := c.Query("anomaly_type"); anomalyType != "" {
			filter.AnomalyType = &anomalyType
		}
		if severity := c.Query("severity"); severity != "" {
			filter.Severity = &severity
		}
		if status := c.Query("status"); status != "" {
			filter.Status = &status
		}
		if assignedTo := c.Query("assigned_to"); assignedTo != "" {
			if uid, err := uuid.Parse(assignedTo); err == nil {
				filter.AssignedTo = &uid
			}
		}
		if dateFrom := c.Query("date_from"); dateFrom != "" {
			if t, err := time.Parse(time.RFC3339, dateFrom); err == nil {
				filter.DateFrom = &t
			}
		}
		if dateTo := c.Query("date_to"); dateTo != "" {
			if t, err := time.Parse(time.RFC3339, dateTo); err == nil {
				filter.DateTo = &t
			}
		}
		filter.Search = c.Query("search")

		anomalies, total, err := svc.ListAnomalies(c.Request.Context(), tenantID, filter, limit, offset)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to list anomalies")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to list anomalies"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"anomalies": anomalies,
			"total":     total,
			"limit":     limit,
			"offset":    offset,
		})
	}
}

func handleGetAnomaly(svc *audit.AnalyticsService, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid anomaly ID"}})
			return
		}

		anomaly, err := svc.GetAnomaly(c.Request.Context(), id)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to get anomaly")
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Anomaly not found"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"anomaly": anomaly})
	}
}

func handleUpdateAnomaly(svc *audit.AnalyticsService, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid anomaly ID"}})
			return
		}

		var req model.UpdateAnomalyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
			return
		}

		// Get existing anomaly
		anomaly, err := svc.GetAnomaly(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Anomaly not found"}})
			return
		}

		// Update fields
		if req.Status != nil {
			anomaly.Status = *req.Status
		}
		if req.AssignedTo != nil {
			anomaly.AssignedTo = req.AssignedTo
		}
		if req.ResolutionNotes != nil {
			anomaly.ResolutionNotes = req.ResolutionNotes
		}

		if err := svc.UpdateAnomaly(c.Request.Context(), anomaly); err != nil {
			logger.Error().Err(err).Msg("Failed to update anomaly")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to update anomaly"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"anomaly": anomaly})
	}
}

func handleDeleteAnomaly(svc *audit.AnalyticsService, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid anomaly ID"}})
			return
		}

		if err := svc.DeleteAnomaly(c.Request.Context(), id); err != nil {
			logger.Error().Err(err).Msg("Failed to delete anomaly")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to delete anomaly"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Anomaly deleted"})
	}
}

func handleGetAnomalyStats(svc *audit.AnalyticsService, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := getTenantUUID(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
			return
		}

		stats, err := svc.GetAnomalyStats(c.Request.Context(), tenantID)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to get anomaly stats")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get stats"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"stats": stats})
	}
}

func handleRunAnomalyDetection(svc *pamanalytics.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := getTenantUUID(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
			return
		}

		anomalies, err := svc.RunAnomalyDetection(c.Request.Context(), tenantID)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to run anomaly detection")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to run anomaly detection"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"anomalies": anomalies,
			"count":     len(anomalies),
		})
	}
}

func handleBulkUpdateAnomalies(svc *audit.AnalyticsService, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			AnomalyIDs      []string `json:"anomaly_ids" binding:"required"`
			Status          *string  `json:"status"`
			AssignedTo      *string  `json:"assigned_to"`
			ResolutionNotes *string  `json:"resolution_notes"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
			return
		}

		var assignedTo *uuid.UUID
		if req.AssignedTo != nil {
			if uid, err := uuid.Parse(*req.AssignedTo); err == nil {
				assignedTo = &uid
			}
		}

		updated := 0
		for _, idStr := range req.AnomalyIDs {
			id, err := uuid.Parse(idStr)
			if err != nil {
				continue
			}

			anomaly, err := svc.GetAnomaly(c.Request.Context(), id)
			if err != nil {
				continue
			}

			if req.Status != nil {
				anomaly.Status = *req.Status
			}
			if assignedTo != nil {
				anomaly.AssignedTo = assignedTo
			}
			if req.ResolutionNotes != nil {
				anomaly.ResolutionNotes = req.ResolutionNotes
			}

			if err := svc.UpdateAnomaly(c.Request.Context(), anomaly); err == nil {
				updated++
			}
		}

		c.JSON(http.StatusOK, gin.H{"updated": updated})
	}
}

// handleAcknowledgeAnomaly handles POST /api/v1/anomalies/:id/acknowledge
func handleAcknowledgeAnomaly(svc *pamanalytics.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid anomaly ID"}})
			return
		}

		// Get user ID from context
		userIDStr, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "User ID not found in context"}})
			return
		}

		userID, err := uuid.Parse(userIDStr.(string))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_USER", "message": "Invalid user ID"}})
			return
		}

		var req model.AcknowledgeAnomalyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
			return
		}

		if err := svc.AcknowledgeAnomaly(c.Request.Context(), id, userID, req.Notes); err != nil {
			logger.Error().Err(err).Str("anomaly_id", id.String()).Msg("Failed to acknowledge anomaly")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to acknowledge anomaly"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Anomaly acknowledged",
			"id":      id.String(),
		})
	}
}

// handleResolveAnomaly handles POST /api/v1/anomalies/:id/resolve
func handleResolveAnomaly(svc *pamanalytics.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid anomaly ID"}})
			return
		}

		// Get user ID from context
		userIDStr, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "User ID not found in context"}})
			return
		}

		userID, err := uuid.Parse(userIDStr.(string))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_USER", "message": "Invalid user ID"}})
			return
		}

		var req model.ResolveAnomalyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
			return
		}

		if err := svc.ResolveAnomaly(c.Request.Context(), id, userID, pamanalytics.AnomalyStatus(req.Status), req.Notes); err != nil {
			logger.Error().Err(err).Str("anomaly_id", id.String()).Msg("Failed to resolve anomaly")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to resolve anomaly"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Anomaly resolved",
			"id":      id.String(),
			"status":  req.Status,
		})
	}
}

// SSH Key Analytics handlers

func handleGetSSHKeyAnalytics(svc *audit.AnalyticsService, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := getTenantUUID(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
			return
		}

		sshKeyID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid SSH key ID"}})
			return
		}

		// Default to today
		date := time.Now().Truncate(24 * time.Hour)
		if dateStr := c.Query("date"); dateStr != "" {
			if t, err := time.Parse("2006-01-02", dateStr); err == nil {
				date = t
			}
		}

		analytics, err := svc.GetSSHKeyAnalytics(c.Request.Context(), tenantID, sshKeyID, date)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to get SSH key analytics")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get analytics"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"analytics": analytics})
	}
}

func handleListSSHKeyAnalytics(svc *audit.AnalyticsService, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := getTenantUUID(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
			return
		}

		filter := model.SSHKeyAnalyticsFilter{TenantID: &tenantID}

		if sshKeyID := c.Query("ssh_key_id"); sshKeyID != "" {
			if id, err := uuid.Parse(sshKeyID); err == nil {
				filter.SSHKeyID = &id
			}
		}
		if dateFrom := c.Query("date_from"); dateFrom != "" {
			if t, err := time.Parse("2006-01-02", dateFrom); err == nil {
				filter.DateFrom = &t
			}
		}
		if dateTo := c.Query("date_to"); dateTo != "" {
			if t, err := time.Parse("2006-01-02", dateTo); err == nil {
				filter.DateTo = &t
			}
		}

		analytics, err := svc.ListSSHKeyAnalytics(c.Request.Context(), filter)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to list SSH key analytics")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to list analytics"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"analytics": analytics})
	}
}

func handleGetSSHKeyUsageSummary(svc *audit.AnalyticsService, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := getTenantUUID(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
			return
		}

		// Default to last 30 days
		dateTo := time.Now()
		dateFrom := dateTo.AddDate(0, 0, -30)

		if dateFromStr := c.Query("date_from"); dateFromStr != "" {
			if t, err := time.Parse("2006-01-02", dateFromStr); err == nil {
				dateFrom = t
			}
		}
		if dateToStr := c.Query("date_to"); dateToStr != "" {
			if t, err := time.Parse("2006-01-02", dateToStr); err == nil {
				dateTo = t
			}
		}

		summary, err := svc.GetSSHKeyUsageSummary(c.Request.Context(), tenantID, dateFrom, dateTo)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to get SSH key usage summary")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get summary"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"summary": summary})
	}
}

func handleGetMostUsedSSHKeys(svc *audit.AnalyticsService, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := getTenantUUID(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
			return
		}

		dateTo := time.Now()
		dateFrom := dateTo.AddDate(0, 0, -30)

		if dateFromStr := c.Query("date_from"); dateFromStr != "" {
			if t, err := time.Parse("2006-01-02", dateFromStr); err == nil {
				dateFrom = t
			}
		}
		if dateToStr := c.Query("date_to"); dateToStr != "" {
			if t, err := time.Parse("2006-01-02", dateToStr); err == nil {
				dateTo = t
			}
		}

		limit := getIntQuery(c, "limit", 10)

		ranks, err := svc.GetMostUsedSSHKeys(c.Request.Context(), tenantID, dateFrom, dateTo, limit)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to get most used SSH keys")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get most used keys"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"keys": ranks})
	}
}

func handleGetAnomalousSSHKeys(svc *audit.AnalyticsService, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := getTenantUUID(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
			return
		}

		dateTo := time.Now()
		dateFrom := dateTo.AddDate(0, 0, -30)

		if dateFromStr := c.Query("date_from"); dateFromStr != "" {
			if t, err := time.Parse("2006-01-02", dateFromStr); err == nil {
				dateFrom = t
			}
		}
		if dateToStr := c.Query("date_to"); dateToStr != "" {
			if t, err := time.Parse("2006-01-02", dateToStr); err == nil {
				dateTo = t
			}
		}

		threshold := getIntQuery(c, "threshold", 5)

		analytics, err := svc.GetAnomalousSSHKeys(c.Request.Context(), tenantID, dateFrom, dateTo, threshold)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to get anomalous SSH keys")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get anomalous keys"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"analytics": analytics})
	}
}

// Command Blacklist handlers

func handleCreateCommandBlacklist(svc *audit.AnalyticsService, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req model.CreateCommandBlacklistRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
			return
		}

		tenantID, err := getTenantUUID(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
			return
		}

		userID, _ := c.Get("user_id")
		userIDUUID, _ := uuid.Parse(userID.(string))

		blacklist := &model.CommandBlacklist{
			TenantID:        &tenantID,
			CommandPattern:  req.CommandPattern,
			PatternType:     req.PatternType,
			Action:          req.Action,
			Severity:        req.Severity,
			AppliesToUsers:  req.AppliesToUsers,
			AppliesToGroups: req.AppliesToGroups,
			AppliesToTargets: req.AppliesToTargets,
			AllowOverride:   req.AllowOverride,
			OverrideRoles:   req.OverrideRoles,
			Reason:          req.Reason,
			CreatedBy:       userIDUUID,
			Enabled:         true,
		}

		if req.BaseCommand != "" {
			blacklist.BaseCommand = &req.BaseCommand
		}
		if req.RiskCategory != "" {
			blacklist.RiskCategory = &req.RiskCategory
		}

		if err := svc.CreateCommandBlacklist(c.Request.Context(), blacklist); err != nil {
			logger.Error().Err(err).Msg("Failed to create command blacklist")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to create blacklist entry"}})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"blacklist": blacklist})
	}
}

func handleListCommandBlacklist(svc *audit.AnalyticsService, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := getTenantUUID(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
			return
		}

		filter := model.CommandBlacklistFilter{TenantID: &tenantID}

		if enabled := c.Query("enabled"); enabled != "" {
			if e, err := parseBool(enabled); err == nil {
				filter.Enabled = &e
			}
		}
		if patternType := c.Query("pattern_type"); patternType != "" {
			filter.PatternType = &patternType
		}
		if action := c.Query("action"); action != "" {
			filter.Action = &action
		}
		if severity := c.Query("severity"); severity != "" {
			filter.Severity = &severity
		}

		blacklists, err := svc.ListCommandBlacklist(c.Request.Context(), filter)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to list command blacklist")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to list blacklist"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"blacklist": blacklists})
	}
}

func handleGetCommandBlacklist(svc *audit.AnalyticsService, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid blacklist ID"}})
			return
		}

		blacklist, err := svc.GetCommandBlacklist(c.Request.Context(), id)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to get command blacklist")
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Blacklist entry not found"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"blacklist": blacklist})
	}
}

func handleUpdateCommandBlacklist(svc *audit.AnalyticsService, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid blacklist ID"}})
			return
		}

		blacklist, err := svc.GetCommandBlacklist(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Blacklist entry not found"}})
			return
		}

		var req model.UpdateCommandBlacklistRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
			return
		}

		if req.CommandPattern != nil {
			blacklist.CommandPattern = *req.CommandPattern
		}
		if req.PatternType != nil {
			blacklist.PatternType = *req.PatternType
		}
		if req.BaseCommand != nil {
			blacklist.BaseCommand = req.BaseCommand
		}
		if req.Action != nil {
			blacklist.Action = *req.Action
		}
		if req.Severity != nil {
			blacklist.Severity = *req.Severity
		}
		if req.AppliesToUsers != nil {
			blacklist.AppliesToUsers = req.AppliesToUsers
		}
		if req.AppliesToGroups != nil {
			blacklist.AppliesToGroups = req.AppliesToGroups
		}
		if req.AppliesToTargets != nil {
			blacklist.AppliesToTargets = req.AppliesToTargets
		}
		if req.AllowOverride != nil {
			blacklist.AllowOverride = *req.AllowOverride
		}
		if req.OverrideRoles != nil {
			blacklist.OverrideRoles = req.OverrideRoles
		}
		if req.Reason != nil {
			blacklist.Reason = *req.Reason
		}
		if req.RiskCategory != nil {
			blacklist.RiskCategory = req.RiskCategory
		}
		if req.Enabled != nil {
			blacklist.Enabled = *req.Enabled
		}

		if err := svc.UpdateCommandBlacklist(c.Request.Context(), blacklist); err != nil {
			logger.Error().Err(err).Msg("Failed to update command blacklist")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to update blacklist"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"blacklist": blacklist})
	}
}

func handleDeleteCommandBlacklist(svc *audit.AnalyticsService, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid blacklist ID"}})
			return
		}

		if err := svc.DeleteCommandBlacklist(c.Request.Context(), id); err != nil {
			logger.Error().Err(err).Msg("Failed to delete command blacklist")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to delete blacklist"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Blacklist entry deleted"})
	}
}

func handleGetBlacklistStats(svc *audit.AnalyticsService, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := getTenantUUID(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
			return
		}

		stats, err := svc.GetBlacklistStats(c.Request.Context(), tenantID)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to get blacklist stats")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get stats"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"stats": stats})
	}
}

func handleEnableCommandBlacklist(svc *audit.AnalyticsService, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid blacklist ID"}})
			return
		}

		if err := svc.EnableCommandBlacklist(c.Request.Context(), id); err != nil {
			logger.Error().Err(err).Msg("Failed to enable command blacklist")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to enable blacklist"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Blacklist entry enabled"})
	}
}

func handleDisableCommandBlacklist(svc *audit.AnalyticsService, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid blacklist ID"}})
			return
		}

		if err := svc.DisableCommandBlacklist(c.Request.Context(), id); err != nil {
			logger.Error().Err(err).Msg("Failed to disable command blacklist")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to disable blacklist"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Blacklist entry disabled"})
	}
}

// =============================================================================
// Anomaly Correlation Handlers
// =============================================================================

// handleGetAnomaliesByCorrelation handles GET /api/v1/analytics/anomalies/correlation/:id
func handleGetAnomaliesByCorrelation(svc *audit.AnalyticsService, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		correlationID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid correlation ID"}})
			return
		}

		anomalies, err := svc.GetAnomaliesByCorrelationID(c.Request.Context(), correlationID)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to get anomalies by correlation")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get anomalies"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"anomalies":       anomalies,
			"correlation_id":  correlationID,
			"count":          len(anomalies),
		})
	}
}

// handleGetAnomalyTypes handles GET /api/v1/analytics/anomalies/types
func handleGetAnomalyTypes(svc *audit.AnalyticsService, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := getTenantUUID(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
			return
		}

		types, err := svc.GetAnomalyTypes(c.Request.Context(), tenantID)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to get anomaly types")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get anomaly types"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"types": types})
	}
}

// handleGetTopAnomalyUsers handles GET /api/v1/analytics/anomalies/top-users
func handleGetTopAnomalyUsers(svc *audit.AnalyticsService, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := getTenantUUID(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
			return
		}

		limit := getIntQuery(c, "limit", 10)

		var dateFrom, dateTo *time.Time
		if dateFromStr := c.Query("date_from"); dateFromStr != "" {
			if t, err := time.Parse(time.RFC3339, dateFromStr); err == nil {
				dateFrom = &t
			}
		}
		if dateToStr := c.Query("date_to"); dateToStr != "" {
			if t, err := time.Parse(time.RFC3339, dateToStr); err == nil {
				dateTo = &t
			}
		}

		users, err := svc.GetTopUsersByAnomalyCount(c.Request.Context(), tenantID, limit, dateFrom, dateTo)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to get top anomaly users")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get top users"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"users": users})
	}
}

// handleMergeDuplicateAnomalies handles POST /api/v1/analytics/anomalies/:id/merge
func handleMergeDuplicateAnomalies(svc *audit.AnalyticsService, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid anomaly ID"}})
			return
		}

		count, err := svc.MergeDuplicateAnomalies(c.Request.Context(), id)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to merge duplicate anomalies")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to merge duplicates"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":          "Duplicates merged",
			"anomaly_id":       id.String(),
			"duplicates_merged": count,
		})
	}
}

// =============================================================================
// Compliance Report Handlers
// =============================================================================

// handleListComplianceReports handles GET /api/v1/analytics/reports
func handleListComplianceReports(svc *pamanalytics.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := getTenantUUID(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
			return
		}

		limit := getIntQuery(c, "limit", 50)
		offset := getIntQuery(c, "offset", 0)
		framework := c.Query("framework")

		reports, total, err := svc.ListComplianceReports(c.Request.Context(), tenantID, framework, limit, offset)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to list compliance reports")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to list reports"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"reports": reports,
			"total":   total,
			"limit":   limit,
			"offset":  offset,
		})
	}
}

// handleGetComplianceReport handles GET /api/v1/analytics/reports/:id
func handleGetComplianceReport(svc *pamanalytics.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := getTenantUUID(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
			return
		}

		reportID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid report ID"}})
			return
		}

		report, err := svc.GetComplianceReport(c.Request.Context(), reportID, tenantID)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to get compliance report")
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Report not found"}})
			return
		}

		c.JSON(http.StatusOK, report)
	}
}

// handleGenerateComplianceReport handles POST /api/v1/analytics/reports
func handleGenerateComplianceReport(svc *pamanalytics.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := getTenantUUID(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
			return
		}

		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_USER", "message": "User ID not found in context"}})
			return
		}
		userIDUUID, err := uuid.Parse(userID.(string))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_USER", "message": "Invalid user ID"}})
			return
		}

		var req struct {
			Framework  string `json:"framework" binding:"required"`
			PeriodStart string `json:"period_start" binding:"required"`
			PeriodEnd   string `json:"period_end" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
			return
		}

		startDate, err := time.Parse(time.RFC3339, req.PeriodStart)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_DATE", "message": "Invalid period_start format"}})
			return
		}

		endDate, err := time.Parse(time.RFC3339, req.PeriodEnd)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_DATE", "message": "Invalid period_end format"}})
			return
		}

		report, err := svc.GenerateComplianceReport(c.Request.Context(), tenantID, userIDUUID, req.Framework, startDate, endDate)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to generate compliance report")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to generate report"}})
			return
		}

		c.JSON(http.StatusCreated, report)
	}
}

// handleDeleteComplianceReport handles DELETE /api/v1/analytics/reports/:id
func handleDeleteComplianceReport(svc *pamanalytics.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := getTenantUUID(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
			return
		}

		reportID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid report ID"}})
			return
		}

		if err := svc.DeleteComplianceReport(c.Request.Context(), reportID, tenantID); err != nil {
			logger.Error().Err(err).Msg("Failed to delete compliance report")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to delete report"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Report deleted successfully"})
	}
}

// =============================================================================
// Report Snapshot Handlers
// =============================================================================

// handleListReportSnapshots handles GET /api/v1/analytics/snapshots
func handleListReportSnapshots(svc *pamanalytics.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := getTenantUUID(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
			return
		}

		limit := getIntQuery(c, "limit", 50)
		offset := getIntQuery(c, "offset", 0)

		// Parse optional filters
		var filter pamanalytics.ReportSnapshotFilter
		filter.TenantID = tenantID
		filter.Limit = limit
		filter.Offset = offset

		if reportID := c.Query("report_id"); reportID != "" {
			if id, err := uuid.Parse(reportID); err == nil {
				filter.ReportID = &id
			}
		}

		if framework := c.Query("framework"); framework != "" {
			filter.Framework = framework
		}

		if status := c.Query("status"); status != "" {
			filter.Status = status
		}

		snapshots, total, err := svc.ListReportSnapshots(c.Request.Context(), tenantID, filter)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to list report snapshots")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to list snapshots"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"snapshots": snapshots,
			"total":     total,
			"limit":     limit,
			"offset":    offset,
		})
	}
}

// handleGetReportSnapshot handles GET /api/v1/analytics/snapshots/:id
func handleGetReportSnapshot(svc *pamanalytics.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := getTenantUUID(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
			return
		}

		snapshotID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid snapshot ID"}})
			return
		}

		snapshot, err := svc.GetReportSnapshot(c.Request.Context(), snapshotID, tenantID)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to get report snapshot")
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Snapshot not found"}})
			return
		}

		c.JSON(http.StatusOK, snapshot)
	}
}

// handleGetReportSnapshotStats handles GET /api/v1/analytics/snapshots/stats
func handleGetReportSnapshotStats(svc *pamanalytics.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := getTenantUUID(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
			return
		}

		stats, err := svc.GetReportSnapshotStats(c.Request.Context(), tenantID)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to get report snapshot stats")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get stats"}})
			return
		}

		c.JSON(http.StatusOK, stats)
	}
}

// Helper functions

func getTenantUUID(c *gin.Context) (uuid.UUID, error) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		return uuid.Nil, fmt.Errorf("tenant_id not found in context")
	}
	return uuid.Parse(tenantID.(string))
}

func parseBool(s string) (bool, error) {
	switch s {
	case "true", "1", "yes", "on":
		return true, nil
	case "false", "0", "no", "off":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value: %s", s)
	}
}
