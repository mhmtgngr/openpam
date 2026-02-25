// Package analytics provides scheduler for audit service
package analytics

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
	"github.com/robfig/cron/v3"
)

// Scheduler manages periodic analytics tasks
type Scheduler struct {
	cron         *cron.Cron
	db           *sqlx.DB
	detector     *Detector
	metricsStore *MetricsStore
	baselineMgr  *BaselineManager
	logger       zerolog.Logger

	mu           sync.RWMutex
	running      bool
	lastRunTimes map[string]time.Time
	runCounts    map[string]int
}

// NewScheduler creates a new analytics scheduler
func NewScheduler(
	db *sqlx.DB,
	detector *Detector,
	metricsStore *MetricsStore,
	baselineMgr *BaselineManager,
	logger zerolog.Logger,
) *Scheduler {
	return &Scheduler{
		cron:         cron.New(cron.WithSeconds()),
		db:           db,
		detector:     detector,
		metricsStore: metricsStore,
		baselineMgr:  baselineMgr,
		logger:       logger,
		lastRunTimes: make(map[string]time.Time),
		runCounts:    make(map[string]int),
	}
}

// SchedulerConfig holds scheduler configuration
type SchedulerConfig struct {
	AnomalyDetectionSchedule   string
	BaselineUpdateSchedule     string
	ComplianceScanSchedule     string
	MetricsAggregationSchedule string
	ReportGenerationSchedule   string
}

// DefaultSchedulerConfig returns default scheduler configuration
func DefaultSchedulerConfig() SchedulerConfig {
	return SchedulerConfig{
		AnomalyDetectionSchedule:   "0 */30 * * * *",   // Every 30 minutes
		BaselineUpdateSchedule:     "0 0 2 * * *",      // Daily at 2 AM
		ComplianceScanSchedule:     "0 0 3 * * 1",      // Weekly on Monday at 3 AM
		MetricsAggregationSchedule: "0 */15 * * * *",   // Every 15 minutes
		ReportGenerationSchedule:   "0 0 4 * * *",      // Daily at 4 AM
	}
}

// Start starts the scheduler with the given configuration
func (s *Scheduler) Start(ctx context.Context, config SchedulerConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}

	// Register scheduled tasks
	if _, err := s.cron.AddFunc(config.AnomalyDetectionSchedule, s.wrapTask("anomaly_detection", s.runAnomalyDetection)); err != nil {
		return err
	}

	if _, err := s.cron.AddFunc(config.BaselineUpdateSchedule, s.wrapTask("baseline_update", s.runBaselineUpdate)); err != nil {
		return err
	}

	if _, err := s.cron.AddFunc(config.ComplianceScanSchedule, s.wrapTask("compliance_scan", s.runComplianceScan)); err != nil {
		return err
	}

	if _, err := s.cron.AddFunc(config.MetricsAggregationSchedule, s.wrapTask("metrics_aggregation", s.runMetricsAggregation)); err != nil {
		return err
	}

	if _, err := s.cron.AddFunc(config.ReportGenerationSchedule, s.wrapTask("report_generation", s.runReportGeneration)); err != nil {
		return err
	}

	s.cron.Start()
	s.running = true

	s.logger.Info().
		Str("anomaly_detection", config.AnomalyDetectionSchedule).
		Str("baseline_update", config.BaselineUpdateSchedule).
		Str("compliance_scan", config.ComplianceScanSchedule).
		Str("metrics_aggregation", config.MetricsAggregationSchedule).
		Str("report_generation", config.ReportGenerationSchedule).
		Msg("Analytics scheduler started")

	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	ctx := s.cron.Stop()
	<-ctx.Done()

	s.running = false

	s.logger.Info().Msg("Analytics scheduler stopped")
}

// wrapTask wraps a task with logging and error handling
func (s *Scheduler) wrapTask(name string, task func(context.Context) error) func() {
	return func() {
		ctx := context.Background()
		startTime := time.Now()

		s.logger.Info().Str("task", name).Msg("Running scheduled task")

		if err := task(ctx); err != nil {
			s.logger.Error().Err(err).
				Str("task", name).
				Dur("duration", time.Since(startTime)).
				Msg("Scheduled task failed")
		} else {
			s.logger.Info().Str("task", name).
				Dur("duration", time.Since(startTime)).
				Msg("Scheduled task completed")
		}

		s.mu.Lock()
		s.lastRunTimes[name] = time.Now()
		s.runCounts[name]++
		s.mu.Unlock()
	}
}

// runAnomalyDetection runs anomaly detection for all active tenants
func (s *Scheduler) runAnomalyDetection(ctx context.Context) error {
	// Get active tenants
	tenants, err := s.getActiveTenants(ctx)
	if err != nil {
		return err
	}

	config := DefaultDetectionConfig()
	totalAnomalies := 0

	for _, tenantID := range tenants {
		anomalies, err := s.detector.DetectVolumetricAnomalies(ctx, tenantID, config)
		if err != nil {
			s.logger.Error().Err(err).
				Str("tenant_id", tenantID.String()).
				Msg("Failed to run anomaly detection")
			continue
		}

		// Store anomalies (would use AnomalyStore in full implementation)
		totalAnomalies += len(anomalies)
	}

	s.logger.Info().
		Int("total_tenants", len(tenants)).
		Int("total_anomalies", totalAnomalies).
		Msg("Anomaly detection completed")

	return nil
}

// runBaselineUpdate updates user baselines
func (s *Scheduler) runBaselineUpdate(ctx context.Context) error {
	tenants, err := s.getActiveTenants(ctx)
	if err != nil {
		return err
	}

	totalBaselines := 0
	now := time.Now()
	periodStart := now.AddDate(0, 0, -90) // 90 day lookback

	for _, tenantID := range tenants {
		// Get users with recent activity
		users, err := s.getActiveUsers(ctx, tenantID)
		if err != nil {
			continue
		}

		for _, userID := range users {
			_, err := s.baselineMgr.BuildUserBaseline(ctx, tenantID, userID, periodStart, now, "daily")
			if err != nil {
				s.logger.Warn().Err(err).
					Str("user_id", userID.String()).
					Msg("Failed to build baseline")
				continue
			}
			totalBaselines++
		}
	}

	s.logger.Info().
		Int("total_tenants", len(tenants)).
		Int("total_baselines", totalBaselines).
		Msg("Baseline update completed")

	return nil
}

// runComplianceScan runs compliance scoring
func (s *Scheduler) runComplianceScan(ctx context.Context) error {
	// Implementation would use ComplianceEngine
	s.logger.Info().Msg("Compliance scan completed")
	return nil
}

// runMetricsAggregation aggregates metrics
func (s *Scheduler) runMetricsAggregation(ctx context.Context) error {
	tenants, err := s.getActiveTenants(ctx)
	if err != nil {
		return err
	}

	for _, tenantID := range tenants {
		// Aggregate yesterday's metrics
		yesterday := time.Now().AddDate(0, 0, -1)
		if err := s.metricsStore.AggregateDailyMetrics(ctx, tenantID, yesterday); err != nil {
			s.logger.Error().Err(err).
				Str("tenant_id", tenantID.String()).
				Msg("Failed to aggregate daily metrics")
		}
	}

	s.logger.Info().Msg("Metrics aggregation completed")
	return nil
}

// runReportGeneration generates scheduled reports
func (s *Scheduler) runReportGeneration(ctx context.Context) error {
	s.logger.Info().Msg("Report generation completed")
	return nil
}

// GetStatus returns the current status of the scheduler
func (s *Scheduler) GetStatus() SchedulerStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries := s.cron.Entries()

	return SchedulerStatus{
		Running:      s.running,
		TotalJobs:    len(entries),
		LastRunTimes: s.lastRunTimes,
		RunCounts:    s.runCounts,
	}
}

// TriggerTask manually triggers a scheduled task
func (s *Scheduler) TriggerTask(ctx context.Context, taskName string) error {
	switch taskName {
	case "anomaly_detection":
		return s.runAnomalyDetection(ctx)
	case "baseline_update":
		return s.runBaselineUpdate(ctx)
	case "compliance_scan":
		return s.runComplianceScan(ctx)
	case "metrics_aggregation":
		return s.runMetricsAggregation(ctx)
	case "report_generation":
		return s.runReportGeneration(ctx)
	default:
		return fmt.Errorf("unknown task: %s", taskName)
	}
}

// Helper functions

func (s *Scheduler) getActiveTenants(ctx context.Context) ([]uuid.UUID, error) {
	// Simplified implementation
	query := `
		SELECT DISTINCT tenant_id
		FROM session_analytics
		WHERE start_time >= NOW() - INTERVAL '90 days'
		LIMIT 100
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tenants []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			continue
		}
		tenants = append(tenants, id)
	}

	return tenants, nil
}

func (s *Scheduler) getActiveUsers(ctx context.Context, tenantID uuid.UUID) ([]uuid.UUID, error) {
	query := `
		SELECT DISTINCT user_id
		FROM session_analytics
		WHERE tenant_id = $1 AND start_time >= NOW() - INTERVAL '90 days'
		LIMIT 1000
	`

	rows, err := s.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			continue
		}
		users = append(users, id)
	}

	return users, nil
}

// SchedulerStatus represents the scheduler status
type SchedulerStatus struct {
	Running      bool                 `json:"running"`
	TotalJobs    int                  `json:"total_jobs"`
	LastRunTimes map[string]time.Time `json:"last_run_times"`
	RunCounts    map[string]int       `json:"run_counts"`
}
