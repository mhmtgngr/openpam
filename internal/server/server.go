package server

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/openpam/openpam/internal/cache"
	"github.com/openpam/openpam/internal/config"
	"github.com/openpam/openpam/internal/database"
	"github.com/openpam/openpam/internal/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Dependencies holds shared infrastructure dependencies initialized by Bootstrap.
// Services use this to access the database, cache, logger, and configuration
// without duplicating initialization logic.
type Dependencies struct {
	Config config.Config
	DB     *database.DB
	Cache  *cache.Cache
	Logger zerolog.Logger
}

// Bootstrap initializes shared infrastructure (config, logging, database, Redis)
// for any service. It returns Dependencies that the service uses to wire up
// its domain-specific components.
func Bootstrap(serviceName string) *Dependencies {
	cfg := config.Load(serviceName)

	setupLogger(cfg.LogLevel)
	logger := log.With().Str("service", serviceName).Logger()

	db, err := database.New(database.Config{
		Host:            cfg.DBHost,
		Port:            cfg.DBPort,
		User:            cfg.DBUser,
		Password:        cfg.DBPassword,
		Database:        cfg.DBName,
		SSLMode:         cfg.DBSSLMode,
		SSLRootCert:     cfg.DBSSLRootCert,
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: time.Minute * 5,
	}, logger)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}

	redisCache, err := cache.New(cache.Config{
		Host:     cfg.RedisHost,
		Port:     cfg.RedisPort,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
		PoolSize: 100,
	}, logger)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to Redis")
	}

	return &Dependencies{
		Config: cfg,
		DB:     db,
		Cache:  redisCache,
		Logger: logger,
	}
}

// NewRouter creates a gin.Engine with standard middleware (logging, recovery,
// security headers, CORS, request IDs). Services add their own routes to the
// returned engine.
func NewRouter(deps *Dependencies) *gin.Engine {
	if deps.Config.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(middleware.Logger(deps.Logger))
	r.Use(middleware.Recovery(deps.Logger))
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(middleware.Config{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Origin", "Content-Type", "Authorization", "X-Request-ID"},
		ExposeHeaders:  []string{"Content-Length", "X-Request-ID"},
	}))
	r.Use(middleware.RequestID())

	registerHealthEndpoints(r, deps)

	return r
}

// ListenAndServe starts the HTTP server and blocks until a shutdown signal
// (SIGINT/SIGTERM) is received, then performs graceful shutdown.
func ListenAndServe(router *gin.Engine, deps *Dependencies) {
	srv := &http.Server{
		Addr:    ":" + deps.Config.Port,
		Handler: router,
	}

	go func() {
		deps.Logger.Info().Str("port", deps.Config.Port).Msg("Starting server")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server failed")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	deps.Logger.Info().Msg("Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		deps.Logger.Error().Err(err).Msg("Forced shutdown")
	}

	deps.Logger.Info().Msg("Goodbye.")
}

func registerHealthEndpoints(r *gin.Engine, deps *Dependencies) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "up", "service": deps.Config.ServiceName})
	})
	r.GET("/ready", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		ready := true
		details := gin.H{}

		if err := deps.DB.Health(ctx); err != nil {
			ready = false
			details["database"] = "unhealthy"
		} else {
			details["database"] = "healthy"
		}

		if err := deps.Cache.Health(ctx); err != nil {
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
