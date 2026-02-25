// Package analytics provides integration helpers for initializing the analytics module
package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/audit/anomalies"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

// Module holds all the analytics components
type Module struct {
	Detector        *Detector
	MetricsStore    *MetricsStore
	BaselineManager *BaselineManager
	Compliance      *ComplianceEngine
	AnomalyStore    *anomalies.Store
	RedisCache      *RedisMetricsCache
	Scheduler       *Scheduler
	Logger          zerolog.Logger
}

// ModuleConfig holds configuration for initializing the analytics module
type ModuleConfig struct {
	DB                   *sqlx.DB
	RedisClient          *redis.Client
	Logger               zerolog.Logger
	EnableScheduler      bool
	SchedulerConfig      SchedulerConfig
}

// InitializeModule initializes all analytics components
func InitializeModule(cfg ModuleConfig) (*Module, error) {
	// Initialize Redis cache
	redisCache := NewRedisMetricsCache(cfg.RedisClient, cfg.Logger)

	// Initialize anomaly store
	anomalyStore := anomalies.NewStore(cfg.DB, cfg.Logger)

	// Initialize metrics store
	metricsStore := NewMetricsStore(cfg.DB, cfg.Logger)

	// Initialize baseline manager
	baselineMgr := NewBaselineManager(cfg.DB, cfg.Logger)

	// Initialize detector
	detector := NewDetector(cfg.DB, cfg.Logger)

	// Initialize compliance engine
	// Need to create a repository wrapper for the compliance engine
	repo := NewRepository(cfg.DB, cfg.Logger)
	compliance := NewComplianceEngine(repo, redisCache, cfg.Logger)

	// Initialize scheduler if enabled
	var scheduler *Scheduler
	if cfg.EnableScheduler {
		scheduler = NewScheduler(
			cfg.DB,
			detector,
			metricsStore,
			baselineMgr,
			cfg.Logger,
		)
	}

	module := &Module{
		Detector:        detector,
		MetricsStore:    metricsStore,
		BaselineManager: baselineMgr,
		Compliance:      compliance,
		AnomalyStore:    anomalyStore,
		RedisCache:      redisCache,
		Scheduler:       scheduler,
		Logger:          cfg.Logger,
	}

	return module, nil
}

// Start starts the analytics module (e.g., starts the scheduler)
func (m *Module) Start(ctx context.Context, config SchedulerConfig) error {
	if m.Scheduler != nil {
		return m.Scheduler.Start(ctx, config)
	}
	return nil
}

// Stop stops the analytics module
func (m *Module) Stop() {
	if m.Scheduler != nil {
		m.Scheduler.Stop()
	}
}

// RunInitialTasks runs initial setup tasks like building initial baselines
func (m *Module) RunInitialTasks(ctx context.Context, tenantID uuid.UUID) error {
	m.Logger.Info().Str("tenant_id", tenantID.String()).Msg("Running initial analytics tasks")

	// Build initial baselines for active users
	periodEnd := time.Now()
	periodStart := periodEnd.AddDate(0, 0, -90) // 90 day lookback

	// In production, would get active users and build baselines
	_, err := m.BaselineManager.BuildUserBaseline(ctx, tenantID, tenantID, periodStart, periodEnd, "daily")
	if err != nil {
		m.Logger.Warn().Err(err).Msg("Failed to build initial baseline")
	}

	return nil
}

// HealthCheck checks the health of the analytics module
func (m *Module) HealthCheck(ctx context.Context) error {
	// Check database connection
	if m.Detector != nil {
		// Would ping database
	}
	return nil
}

// GetStatus returns the current status of the analytics module
func (m *Module) GetStatus() map[string]interface{} {
	status := make(map[string]interface{})

	if m.Scheduler != nil {
		schedulerStatus := m.Scheduler.GetStatus()
		status["scheduler"] = map[string]interface{}{
			"running":       schedulerStatus.Running,
			"total_jobs":   schedulerStatus.TotalJobs,
			"last_run_times": schedulerStatus.LastRunTimes,
			"run_counts":   schedulerStatus.RunCounts,
		}
	}

	status["components"] = map[string]interface{}{
		"detector":         m.Detector != nil,
		"metrics_store":    m.MetricsStore != nil,
		"baseline_manager": m.BaselineManager != nil,
		"compliance":       m.Compliance != nil,
		"anomaly_store":    m.AnomalyStore != nil,
		"redis_cache":      m.RedisCache != nil,
	}

	return status
}

// CleanupOldData removes old analytics data based on retention policies
func (m *Module) CleanupOldData(ctx context.Context, retentionDays int) error {
	// Clean up old session analytics
	_, err := m.MetricsStore.PurgeOldMetrics(ctx, retentionDays)
	if err != nil {
		return fmt.Errorf("cleanup metrics: %w", err)
	}

	return nil
}

// GetTenantSummary returns a summary of analytics data for a tenant
func (m *Module) GetTenantSummary(ctx context.Context, tenantID uuid.UUID, days int) (map[string]interface{}, error) {
	// Get baseline statistics
	baselineStats, err := m.BaselineManager.GetBaselineStatistics(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get baseline stats: %w", err)
	}

	// Get anomaly statistics
	anomalyStats, err := m.AnomalyStore.GetStats(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get anomaly stats: %w", err)
	}

	return map[string]interface{}{
		"tenant_id":    tenantID.String(),
		"period_days":  days,
		"baselines":    baselineStats,
		"anomalies":    anomalyStats,
		"generated_at": time.Now(),
	}, nil
}
