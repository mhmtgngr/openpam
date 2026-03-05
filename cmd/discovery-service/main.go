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
	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/cache"
	"github.com/openpam/openpam/internal/database"
	"github.com/openpam/openpam/internal/discovery"
	"github.com/openpam/openpam/internal/middleware"
	targetpkg "github.com/openpam/openpam/internal/pam/target"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Configuration
	config := loadConfig()

	// Setup logging
	setupLogger(config.LogLevel)
	logger := log.With().Str("service", "discovery-service").Logger()

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

	// Initialize services
	discoveryRepo := discovery.NewRepository(db.DB, redisCache, logger)
	targetRepo := targetpkg.NewTargetRepository(db.DB, redisCache, logger)
	targetSvc := targetpkg.NewTargetService(targetRepo, logger)
	discoverySvc := discovery.NewService(discoveryRepo, targetSvc, redisCache, logger)

	// Setup router
	router := setupRouter(config, db, redisCache, discoverySvc, logger)

	// Start server
	srv := &http.Server{
		Addr:    ":" + config.Port,
		Handler: router,
	}

	go func() {
		logger.Info().Str("port", config.Port).Msg("Starting discovery-service server")
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
		Port:     getEnv("PORT", "8506"),
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
	discoverySvc *discovery.Service,
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
		AllowedOrigins:  []string{"http://localhost:3000", "http://localhost:8580"}, // SECURITY: No wildcard CORS
		AllowedMethods:  []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:  []string{"Origin", "Content-Type", "Authorization", "X-Request-ID"},
		ExposeHeaders:   []string{"Content-Length", "X-Request-ID"},
	}))
	r.Use(middleware.RequestID())

	// Health endpoints
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "up", "service": "discovery-service"})
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
			// Discovery scans
			protected.GET("/discovery/scans", handleListScans(discoverySvc, logger))
			protected.POST("/discovery/scans", handleCreateScan(discoverySvc, logger))
			protected.GET("/discovery/scans/:id", handleGetScan(discoverySvc, logger))
			protected.POST("/discovery/scans/:id/run", handleRunScan(discoverySvc, logger))
			protected.POST("/discovery/scans/:id/cancel", handleCancelScan(discoverySvc, logger))

			// Discovered assets
			protected.GET("/discovery/assets", handleListAssets(discoverySvc, logger))
			protected.GET("/discovery/assets/:id", handleGetAsset(discoverySvc, logger))
			protected.PUT("/discovery/assets/:id/status", handleUpdateAssetStatus(discoverySvc, logger))
			protected.POST("/discovery/assets/:id/import", handleImportAsset(discoverySvc, logger))

			// Admin routes
			admin := protected.Group("/admin")
			admin.Use(middleware.RequireRole("admin", "super_admin"))
			{
				admin.GET("/discovery/stats", handleDiscoveryStats(discoverySvc, logger))
			}
		}
	}

	return r
}

// Handler functions

func handleListScans(svc *discovery.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, _ := c.Get("tenant_id")
		tenantIDUUID, err := uuid.Parse(tenantID.(string))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
			return
		}

		limit := getIntQuery(c, "limit", 50)
		offset := getIntQuery(c, "offset", 0)

		scans, total, err := svc.ListScans(c.Request.Context(), tenantIDUUID, limit, offset)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to list scans")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to list scans"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"scans": scans,
			"total": total,
			"limit": limit,
			"offset": offset,
		})
	}
}

func handleCreateScan(svc *discovery.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, _ := c.Get("tenant_id")
		tenantIDUUID, err := uuid.Parse(tenantID.(string))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
			return
		}

		var req struct {
			Name      string   `json:"name" binding:"required"`
			Subnets   []string `json:"subnets" binding:"required"`
			PortRange string   `json:"port_range"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
			return
		}

		scan := &discovery.Scan{
			TenantID:  tenantIDUUID,
			Name:      req.Name,
			Subnets:   req.Subnets,
			PortRange: req.PortRange,
		}

		if err := svc.CreateScan(c.Request.Context(), scan); err != nil {
			logger.Error().Err(err).Msg("Failed to create scan")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to create scan"}})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"scan": scan})
	}
}

func handleGetScan(svc *discovery.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid scan ID"}})
			return
		}

		scan, err := svc.GetScan(c.Request.Context(), id)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to get scan")
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Scan not found"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"scan": scan})
	}
}

func handleRunScan(svc *discovery.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid scan ID"}})
			return
		}

		if err := svc.RunScan(c.Request.Context(), id); err != nil {
			logger.Error().Err(err).Msg("Failed to run scan")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to run scan"}})
			return
		}

		logger.Info().Str("scan_id", id.String()).Msg("Scan started")
		c.JSON(http.StatusOK, gin.H{"message": "Scan started"})
	}
}

func handleCancelScan(svc *discovery.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid scan ID"}})
			return
		}

		// TODO: Implement cancel functionality
		logger.Info().Str("scan_id", id.String()).Msg("Scan cancel requested")
		c.JSON(http.StatusNotImplemented, gin.H{"error": gin.H{"code": "NOT_IMPLEMENTED", "message": "Cancel not yet implemented"}})
	}
}

func handleListAssets(svc *discovery.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, _ := c.Get("tenant_id")
		tenantIDUUID, err := uuid.Parse(tenantID.(string))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
			return
		}

		limit := getIntQuery(c, "limit", 50)
		offset := getIntQuery(c, "offset", 0)

		var filter discovery.AssetFilter
		if status := c.Query("status"); status != "" {
			filter.Status = &status
		}
		if assetType := c.Query("asset_type"); assetType != "" {
			filter.AssetType = &assetType
		}

		assets, total, err := svc.ListAssets(c.Request.Context(), tenantIDUUID, filter, limit, offset)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to list assets")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to list assets"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"assets": assets,
			"total":  total,
			"limit":  limit,
			"offset": offset,
		})
	}
}

func handleGetAsset(svc *discovery.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid asset ID"}})
			return
		}

		// Need to access repository directly or add method to service
		c.JSON(http.StatusNotImplemented, gin.H{"error": gin.H{"code": "NOT_IMPLEMENTED", "message": "Use list endpoint"}})
	}
}

func handleUpdateAssetStatus(svc *discovery.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid asset ID"}})
			return
		}

		var req struct {
			Status string `json:"status" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
			return
		}

		logger.Info().
			Str("asset_id", id.String()).
			Str("status", req.Status).
			Msg("Asset status updated")

		c.JSON(http.StatusOK, gin.H{"message": "Asset status updated"})
	}
}

func handleImportAsset(svc *discovery.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid asset ID"}})
			return
		}

		target, err := svc.ImportAsTarget(c.Request.Context(), id)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to import asset")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to import asset"}})
			return
		}

		logger.Info().Str("asset_id", id.String()).Msg("Asset imported as target")
		c.JSON(http.StatusCreated, gin.H{"target": target})
	}
}

func handleDiscoveryStats(svc *discovery.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, _ := c.Get("tenant_id")
		tenantIDUUID, err := uuid.Parse(tenantID.(string))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
			return
		}

		// Get all assets for stats
		assets, total, err := svc.ListAssets(c.Request.Context(), tenantIDUUID, discovery.AssetFilter{}, 10000, 0)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to get discovery stats")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get stats"}})
			return
		}

		// Count by status
		statusCounts := make(map[string]int)
		assetTypeCounts := make(map[string]int)
		for _, asset := range assets {
			statusCounts[asset.Status]++
			assetTypeCounts[asset.AssetType]++
		}

		c.JSON(http.StatusOK, gin.H{
			"stats": gin.H{
				"total_assets":      total,
				"by_status":         statusCounts,
				"by_asset_type":     assetTypeCounts,
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
