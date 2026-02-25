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
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/openpam/openpam/internal/cache"
	"github.com/openpam/openpam/internal/config"
	"github.com/openpam/openpam/internal/database"
	"github.com/openpam/openpam/internal/identity/handler"
	"github.com/openpam/openpam/internal/identity/itsm"
	"github.com/openpam/openpam/internal/identity/repository"
	"github.com/openpam/openpam/internal/identity/service"
	"github.com/openpam/openpam/internal/middleware"
	"github.com/openpam/openpam/internal/models"
)

const (
	serviceName = "identity-service"
	servicePort = "8507"
)

func main() {
	// Load configuration
	cfg := config.Load(serviceName)

	// Setup logger
	log.Logger = zerolog.New(os.Stdout).With().
		Timestamp().
		Str("service", serviceName).
		Logger()

	level, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		log.Warn().Str("level", cfg.LogLevel).Msg("Invalid log level, defaulting to info")
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	logger := log.With().Str("service", serviceName).Logger()
	logger.Info().Msg("Starting OpenPAM Identity Service")

	// Initialize database
	db, err := database.NewPostgres(cfg.Database.URL)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	// Initialize Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer rdb.Close()

	// Test Redis connection
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Fatal().Err(err).Msg("Failed to connect to Redis")
	}

	// Initialize cache
	redisCache := cache.NewRedisCache(rdb, &logger)

	// Initialize repositories
	accessRequestRepo := repository.NewAccessRequestRepository(db, &logger)
	workflowRepo := repository.NewWorkflowRepository(db, &logger)
	userRepo := repository.NewUserRepository(db, &logger)
	approvalRepo := repository.NewApprovalRepository(db, &logger)

	// Initialize ITSM clients
	var itsmClients []service.ITSMClient
	if cfg.ServiceNow.Enabled {
		snowClient := itsm.NewServiceNowClient(
			cfg.ServiceNow.InstanceURL,
			cfg.ServiceNow.Username,
			cfg.ServiceNow.Password,
			&logger,
		)
		itsmClients = append(itsmClients, snowClient)
	}
	if cfg.Jira.Enabled {
		jiraClient := itsm.NewJiraClient(
			cfg.Jira.BaseURL,
			cfg.Jira.Username,
			cfg.Jira.APIToken,
			cfg.Jira.ProjectKey,
			&logger,
		)
		itsmClients = append(itsmClients, jiraClient)
	}

	// Initialize services
	accessRequestSvc := service.NewAccessRequestService(
		accessRequestRepo,
		workflowRepo,
		approvalRepo,
		userRepo,
		redisCache,
		itsmClients,
		&logger,
	)

	workflowSvc := service.NewWorkflowEngine(workflowRepo, userRepo, &logger)
	breakGlassSvc := service.NewBreakGlassService(accessRequestRepo, approvalRepo, userRepo, redisCache, &logger)

	// Initialize handlers
	accessHandler := handler.NewAccessRequestHandler(accessRequestSvc, &logger)
	workflowHandler := handler.NewWorkflowHandler(workflowSvc, &logger)
	breakGlassHandler := handler.NewBreakGlassHandler(breakGlassSvc, &logger)

	// Setup router
	router := setupRouter(cfg, accessHandler, workflowHandler, breakGlassHandler)

	// Start ITSM sync worker
	itsmWorker := NewITSMSyncWorker(accessRequestSvc, itsmClients, cfg.ITSM.SyncInterval, &logger)
	itsmWorker.Start(ctx)
	defer itsmWorker.Stop()

	// Start HTTP server
	srv := &http.Server{
		Addr:    ":" + servicePort,
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		logger.Info().Str("port", servicePort).Msg("HTTP server starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("Failed to start HTTP server")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("Server forced to shutdown")
	}

	logger.Info().Msg("Server exited")
}

func setupRouter(
	cfg *config.Config,
	accessHandler *handler.AccessRequestHandler,
	workflowHandler *handler.WorkflowHandler,
	breakGlassHandler *handler.BreakGlassHandler,
) *gin.Engine {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Middleware
	router.Use(gin.Recovery())
	router.Use(middleware.CORS())
	router.Use(middleware.CorrelationID())
	router.Use(middleware.RequestLogger())
	router.Use(middleware.RateLimit(cfg.RateLimit))
	router.Use(middleware.Auth(cfg.JWT.Secret))
	router.Use(middleware.TenantContext())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": serviceName})
	})

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Access Requests
		accessRequests := v1.Group("/access-requests")
		{
			accessRequests.POST("", accessHandler.Create)
			accessRequests.GET("", accessHandler.List)
			accessRequests.GET("/:id", accessHandler.Get)
			accessRequests.PUT("/:id", accessHandler.Update)
			accessRequests.DELETE("/:id", accessHandler.Delete)
			accessRequests.POST("/:id/approve", accessHandler.Approve)
			accessRequests.POST("/:id/deny", accessHandler.Deny)
			accessRequests.POST("/:id/cancel", accessHandler.Cancel)
			accessRequests.GET("/:id/history", accessHandler.GetHistory)
		}

		// Workflows
		workflows := v1.Group("/workflows")
		{
			workflows.POST("", workflowHandler.Create)
			workflows.GET("", workflowHandler.List)
			workflows.GET("/:id", workflowHandler.Get)
			workflows.PUT("/:id", workflowHandler.Update)
			workflows.DELETE("/:id", workflowHandler.Delete)
			workflows.POST("/:id/activate", workflowHandler.Activate)
			workflows.POST("/:id/deactivate", workflowHandler.Deactivate)
		}

		// Break Glass (Emergency Access)
		breakGlass := v1.Group("/break-glass")
		{
			breakGlass.POST("", breakGlassHandler.Request)
			breakGlass.GET("", breakGlassHandler.List)
			breakGlass.GET("/:id", breakGlassHandler.Get)
			breakGlass.POST("/:id/activate", breakGlassHandler.Activate)
			breakGlass.POST("/:id/revoke", breakGlassHandler.Revoke)
			breakGlass.GET("/audit", breakGlassHandler.AuditLog)
		}
	}

	return router
}

// ITMSSyncWorker handles background synchronization with ITSM systems
type ITMSSyncWorker struct {
	accessRequestSvc *service.AccessRequestService
	clients          []service.ITSMClient
	interval         time.Duration
	logger           *zerolog.Logger
	running          bool
	stopCh           chan struct{}
}

func NewITSMSyncWorker(
	accessRequestSvc *service.AccessRequestService,
	clients []service.ITSMClient,
	interval time.Duration,
	logger *zerolog.Logger,
) *ITMSSyncWorker {
	if interval == 0 {
		interval = 5 * time.Minute
	}
	return &ITMSSyncWorker{
		accessRequestSvc: accessRequestSvc,
		clients:          clients,
		interval:         interval,
		logger:           logger,
		stopCh:           make(chan struct{}),
	}
}

func (w *ITMSSyncWorker) Start(ctx context.Context) {
	w.running = true
	go w.run(ctx)
}

func (w *ITMSSyncWorker) Stop() {
	if !w.running {
		return
	}
	close(w.stopCh)
	w.running = false
}

func (w *ITMSSyncWorker) run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	w.logger.Info().Dur("interval", w.interval).Msg("ITMSSyncWorker started")

	for {
		select {
		case <-ctx.Done():
			w.logger.Info().Msg("ITMSSyncWorker context done")
			return
		case <-w.stopCh:
			w.logger.Info().Msg("ITMSSyncWorker stopped")
			return
		case <-ticker.C:
			w.sync(ctx)
		}
	}
}

func (w *ITMSSyncWorker) sync(ctx context.Context) {
	w.logger.Debug().Msg("Syncing with ITSM systems")

	for _, client := range w.clients {
		if err := w.accessRequestSvc.SyncITSMStatus(ctx, client); err != nil {
			w.logger.Error().Err(err).Str("itsm_type", client.Type()).Msg("ITSM sync failed")
		}
	}

	w.logger.Debug().Msg("ITSM sync complete")
}
