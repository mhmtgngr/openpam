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
	"github.com/openpam/openpam/cmd/analytics-service/handlers"
	"github.com/openpam/openpam/internal/analytics"
	"github.com/openpam/openpam/internal/cache"
	"github.com/openpam/openpam/internal/database"
	"github.com/openpam/openpam/internal/events"
	middleware2 "github.com/openpam/openpam/internal/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Configuration
	config := loadConfig()

	// Setup logging
	setupLogger(config.LogLevel)
	logger := log.With().Str("service", "analytics-service").Logger()

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

	// Initialize event bus for subscribing to session events with HMAC signing
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

	// Initialize analytics repository
	repo := analytics.NewPostgresRepository(db.DB, logger)

	// Initialize analytics cache layer
	analyticsCache := analytics.NewRedisCache(redisCache, logger)

	// Wrap with caching
	cachedRepo := analytics.NewCachedRepository(repo, analyticsCache, logger)

	// Initialize analytics service with default config
	serviceConfig := analytics.DefaultServiceConfig()
	analyticsService := analytics.NewService(cachedRepo, analyticsCache, eventPublisher, logger, serviceConfig)

	// Subscribe to analytics events
	eventBus.Subscribe(events.EventTypeAnalyticsSessionStarted, func(ctx context.Context, event events.Event) error {
		return analyticsService.HandleSessionStarted(ctx, event)
	})
	eventBus.Subscribe(events.EventTypeAnalyticsSessionEnded, func(ctx context.Context, event events.Event) error {
		return analyticsService.HandleSessionEnded(ctx, event)
	})
	eventBus.Subscribe(events.EventTypeAnalyticsCommandExecuted, func(ctx context.Context, event events.Event) error {
		return analyticsService.HandleCommandExecuted(ctx, event)
	})

	// Setup router
	router := setupRouter(config, redisCache, analyticsService, logger)

	// Start server
	srv := &http.Server{
		Addr:    ":" + config.Port,
		Handler: router,
	}

	go func() {
		logger.Info().Str("port", config.Port).Msg("Starting analytics service")
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
		Port:     getEnv("PORT", "8507"),
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

func setupRouter(config Config, cache *cache.Cache, service *analytics.Service, logger zerolog.Logger) *gin.Engine {
	if config.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// Initialize rate limiter
	_ = middleware2.NewRateLimiter(cache, logger) // TODO: integrate rate limiting

	// Middleware
	r.Use(middleware2.Logger(logger))
	r.Use(middleware2.Recovery(logger))
	r.Use(middleware2.SecurityHeaders())
	r.Use(middleware2.CORS(middleware2.Config{
		AllowedOrigins:  []string{"*"},
		AllowedMethods:  []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:  []string{"Origin", "Content-Type", "Authorization", "X-Request-ID", "X-Tenant-ID", "X-User-ID"},
		ExposeHeaders:   []string{"Content-Length", "X-Request-ID"},
	}))
	r.Use(middleware2.RequestID())

	// Health endpoints
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "up", "service": "analytics-service"})
	})
	r.GET("/ready", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		ready := true
		details := gin.H{}

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
	analyticsHandler := handlers.NewAnalyticsHandler(service, logger)
	complianceHandler := handlers.NewComplianceHandler(service, logger)
	anomalyHandler := handlers.NewAnomalyHandler(service, logger)

	// API v1 routes
	v1 := r.Group("/api/v1")
	{
		// Protected routes (require auth - tokens validated by gateway)
		protected := v1.Group("")
		{
			// Analytics endpoints
			protected.GET("/analytics/sessions", analyticsHandler.GetSessions)
			protected.GET("/analytics/sessions/trends", analyticsHandler.GetSessionTrends)
			protected.GET("/analytics/sessions/summary", analyticsHandler.GetSessionSummary)
			protected.GET("/analytics/users/:id/activity", analyticsHandler.GetUserActivity)
			protected.GET("/analytics/users/:id/risk", analyticsHandler.GetUserRiskScore)
			protected.GET("/analytics/users/top", analyticsHandler.GetTopUsers)
			protected.GET("/analytics/commands", analyticsHandler.GetCommands)
			protected.GET("/analytics/commands/top", analyticsHandler.GetTopCommands)
			protected.GET("/analytics/commands/blacklist", analyticsHandler.GetCommandBlacklist)
			protected.POST("/analytics/commands/blacklist", analyticsHandler.CreateCommandBlacklist)
			protected.PUT("/analytics/commands/blacklist/:id", analyticsHandler.UpdateCommandBlacklist)
			protected.DELETE("/analytics/commands/blacklist/:id", analyticsHandler.DeleteCommandBlacklist)
			protected.GET("/analytics/dashboard", analyticsHandler.GetDashboard)
			protected.GET("/analytics/timeseries", analyticsHandler.GetTimeSeries)
			protected.GET("/analytics/ssh-keys/:id", analyticsHandler.GetSSHKeyAnalytics)

			// Compliance endpoints
			protected.GET("/compliance/reports", complianceHandler.ListReports)
			protected.POST("/compliance/reports", complianceHandler.GenerateReport)
			protected.GET("/compliance/reports/:id", complianceHandler.GetReport)
			protected.GET("/compliance/reports/:id/controls", complianceHandler.GetControls)
			protected.GET("/compliance/summary", complianceHandler.GetSummary)
			protected.GET("/compliance/exceptions", complianceHandler.ListExceptions)
			protected.POST("/compliance/exceptions", complianceHandler.CreateException)
			protected.PUT("/compliance/exceptions/:id/approve", complianceHandler.ApproveException)
			protected.PUT("/compliance/exceptions/:id/deny", complianceHandler.DenyException)

			// Anomaly Detection endpoints
			protected.GET("/anomalies", anomalyHandler.ListAnomalies)
			protected.GET("/anomalies/:id", anomalyHandler.GetAnomaly)
			protected.POST("/anomalies/detect", anomalyHandler.RunDetection)
			protected.PUT("/anomalies/:id/status", anomalyHandler.UpdateStatus)
			protected.GET("/anomalies/trends", anomalyHandler.GetTrends)
			protected.GET("/anomalies/users/:id", anomalyHandler.GetUserAnomalies)

			// Ransomware Events endpoints
			protected.GET("/ransomware/events", anomalyHandler.ListRansomwareEvents)
			protected.GET("/ransomware/events/:id", anomalyHandler.GetRansomwareEvent)
			protected.POST("/ransomware/events/:id/emergency", anomalyHandler.TriggerEmergency)

			// Internal event ingestion endpoint (called by other services)
			protected.POST("/analytics/events/ingest", analyticsHandler.IngestEvent)
		}
	}

	return r
}
