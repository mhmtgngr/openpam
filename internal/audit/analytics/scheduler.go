// Package analytics provides scheduled task execution for analytics
package analytics

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/audit/anomalies"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
)

// Scheduler manages periodic analytics tasks
type Scheduler struct {
	cron          *cron.Cron
	db            *sqlx.DB
	detector      *Detector
	metricsStore  *MetricsStore
	baselineMgr   *BaselineManager
	compliance    *ComplianceEngine
	anomalyStore  *anomalies.Store
	redisCache    *RedisMetricsCache
	logger        zerolog.Logger

	// Running state
	mu            sync.RWMutex
	running       bool
	lastRunTimes  map[string]time.Time
	runCounts     map[string]int
}

// NewScheduler creates a new analytics scheduler
func NewScheduler(
	db *sqlx.DB,
	detector *Detector,
	metricsStore *MetricsStore,
	baselineMgr *BaselineManager,
	compliance *ComplianceEngine,
	anomalyStore *anomalies.Store,
	redisCache *RedisMetricsCache,
	logger zerolog.Logger,
) *Scheduler {
	// Configure cron with seconds precision
	cronScheduler := cron.New(cron.WithSeconds())

	return &Scheduler{
		cron:         cronScheduler,
		db:           db,
		detector:     detector,
		metricsStore: metricsStore,
		baselineMgr:  baselineMgr,
		compliance:   compliance,
		anomalyStore: anomalyStore,
		redisCache:   redisCache,
		logger:       logger,
		lastRunTimes: make(map[string]time.Time),
		runCounts:    make(map[string]int),
	}
}

// SchedulerConfig holds scheduler configuration
type SchedulerConfig struct {
	// Anomaly detection schedule (cron format)
	AnomalyDetectionSchedule string
	// Baseline update schedule
	BaselineUpdateSchedule string
	// Compliance scan schedule
	ComplianceScanSchedule string
	// Metrics aggregation schedule
	MetricsAggregationSchedule string
	// Report generation schedule
	ReportGenerationSchedule string
}

// DefaultSchedulerConfig returns default scheduler configuration
func DefaultSchedulerConfig() SchedulerConfig {
	return SchedulerConfig{
		AnomalyDetectionSchedule:    "0 */30 * * * *",   // Every 30 minutes
		BaselineUpdateSchedule:      "0 0 2 * * *",      // Daily at 2 AM
		ComplianceScanSchedule:      "0 0 3 * * 1",      // Weekly on Monday at 3 AM
		MetricsAggregationSchedule:  "0 */15 * * * *",   // Every 15 minutes
		ReportGenerationSchedule:    "0 0 4 * * *",      // Daily at 4 AM
	}
}

// Start starts the scheduler with the given configuration
func (s *Scheduler) Start(ctx context.Context, config SchedulerConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("scheduler: already running")
	}

	// Register scheduled tasks
	if _, err := s.cron.AddFunc(config.AnomalyDetectionSchedule, s.wrapTask("anomaly_detection", s.runAnomalyDetection)); err != nil {
		return fmt.Errorf("scheduler: failed to schedule anomaly detection: %w", err)
	}

	if _, err := s.cron.AddFunc(config.BaselineUpdateSchedule, s.wrapTask("baseline_update", s.runBaselineUpdate)); err != nil {
		return fmt.Errorf("scheduler: failed to schedule baseline update: %w", err)
	}

	if _, err := s.cron.AddFunc(config.ComplianceScanSchedule, s.wrapTask("compliance_scan", s.runComplianceScan)); err != nil {
		return fmt.Errorf("scheduler: failed to schedule compliance scan: %w", err)
	}

	if _, err := s.cron.AddFunc(config.MetricsAggregationSchedule, s.wrapTask("metrics_aggregation", s.runMetricsAggregation)); err != nil {
		return fmt.Errorf("scheduler: failed to schedule metrics aggregation: %w", err)
	}

	if _, err := s.cron.AddFunc(config.ReportGenerationSchedule, s.wrapTask("report_generation", s.runReportGeneration)); err != nil {
		return fmt.Errorf("scheduler: failed to schedule report generation: %w", err)
	}

	// Start cron scheduler
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
	// Get all active tenants
	tenants, err := s.getActiveTenants(ctx)
	if err != nil {
		return fmt.Errorf("run_anomaly_detection: failed to get tenants: %w", err)
	}

	config := DefaultDetectionConfig()

	totalAnomalies := 0
	for _, tenantID := range tenants {
		// Run detection
		result, err := s.detector.DetectAnomalies(ctx, tenantID, config)
		if err != nil {
			s.logger.Error().Err(err).
				Str("tenant_id", tenantID.String()).
				Msg("Failed to run anomaly detection for tenant")
			continue
		}

		// Convert and store anomalies
		anomalies := s.detector.ConvertToModelAnomalies(result)
		for _, anomaly := range anomalies {
			if err := s.anomalyStore.Create(ctx, &anomaly); err != nil {
				s.logger.Error().Err(err).
					Str("anomaly_id", anomaly.ID.String()).
					Msg("Failed to store anomaly")
			} else {
				totalAnomalies++
			}
		}

		s.logger.Info().
			Str("tenant_id", tenantID.String()).
			Int("anomalies_detected", len(anomalies)).
			Msg("Anomaly detection completed for tenant")
	}

	s.logger.Info().
		Int("total_tenants", len(tenants)).
		Int("total_anomalies", totalAnomalies).
		Msg("Anomaly detection completed")

	return nil
}

// runBaselineUpdate updates user baselines for all tenants
func (s *Scheduler) runBaselineUpdate(ctx context.Context) error {
	tenants, err := s.getActiveTenants(ctx)
	if err != nil {
		return fmt.Errorf("run_baseline_update: failed to get tenants: %w", err)
	}

	totalBaselines := 0
	for _, tenantID := range tenants {
		count, err := s.baselineMgr.RebuildAllBaselinesForTenant(ctx, tenantID, "daily")
		if err != nil {
			s.logger.Error().Err(err).
				Str("tenant_id", tenantID.String()).
				Msg("Failed to rebuild baselines for tenant")
			continue
		}
		totalBaselines += count
	}

	s.logger.Info().
		Int("total_tenants", len(tenants)).
		Int("total_baselines", totalBaselines).
		Msg("Baseline update completed")

	return nil
}

// runComplianceScan runs compliance scoring for all tenants
func (s *Scheduler) runComplianceScan(ctx context.Context) error {
	tenants, err := s.getActiveTenants(ctx)
	if err != nil {
		return fmt.Errorf("run_compliance_scan: failed to get tenants: %w", err)
	}

	frameworks := []string{"SOC2", "ISO27001", "PCI-DSS", "HIPAA"}

	totalReports := 0
	for _, tenantID := range tenants {
		for _, framework := range frameworks {
			// Get tenant's configured frameworks (simplified)
			periodEnd := time.Now()
			periodStart := periodEnd.AddDate(0, 0, -30) // 30 day lookback

			score, err := s.compliance.CalculateComplianceScore(ctx, tenantID, framework, periodStart, periodEnd)
			if err != nil {
				s.logger.Error().Err(err).
					Str("tenant_id", tenantID.String()).
					Str("framework", framework).
					Msg("Failed to calculate compliance score")
				continue
			}

			// Convert to report and store
			report := s.compliance.ConvertToComplianceReport(score, uuid.Nil) // System-generated

			if err := s.storeComplianceReport(ctx, report); err != nil {
				s.logger.Error().Err(err).
					Str("report_id", report.ID.String()).
					Msg("Failed to store compliance report")
			} else {
				totalReports++
			}

			// Cache the score
			_ = s.redisCache.StoreComplianceScore(ctx, tenantID, framework, score)
		}
	}

	s.logger.Info().
		Int("total_tenants", len(tenants)).
		Int("total_reports", totalReports).
		Msg("Compliance scan completed")

	return nil
}

// runMetricsAggregation aggregates metrics into time buckets
func (s *Scheduler) runMetricsAggregation(ctx context.Context) error {
	tenants, err := s.getActiveTenants(ctx)
	if err != nil {
		return fmt.Errorf("run_metrics_aggregation: failed to get tenants: %w", err)
	}

	now := time.Now()
	buckets := []time.Duration{time.Minute, time.Hour, 24 * time.Hour}

	for _, tenantID := range tenants {
		for _, bucket := range buckets {
			from := now.Add(-bucket * 100) // Look back for aggregation
			to := now

			_, err := s.metricsStore.AggregateMetricsByTimeBucket(ctx, tenantID, from, to, bucket)
			if err != nil {
				s.logger.Error().Err(err).
					Str("tenant_id", tenantID.String()).
					Str("bucket", bucket.String()).
					Msg("Failed to aggregate metrics")
			}
		}
	}

	s.logger.Info().
		Int("total_tenants", len(tenants)).
		Msg("Metrics aggregation completed")

	return nil
}

// runReportGeneration generates scheduled reports
func (s *Scheduler) runReportGeneration(ctx context.Context) error {
	// Get due report schedules
	schedules := s.getDueReportSchedules(ctx)

	for _, schedule := range schedules {
		if err := s.generateScheduledReport(ctx, &schedule); err != nil {
			s.logger.Error().Err(err).
				Str("schedule_id", schedule.ID.String()).
				Msg("Failed to generate scheduled report")
		}
	}

	s.logger.Info().
		Int("schedules_processed", len(schedules)).
		Msg("Report generation completed")

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
		NextRuns:     s.getNextRunTimes(entries),
	}
}

// SchedulerStatus represents the scheduler status
type SchedulerStatus struct {
	Running      bool                 `json:"running"`
	TotalJobs    int                  `json:"total_jobs"`
	LastRunTimes map[string]time.Time `json:"last_run_times"`
	RunCounts    map[string]int       `json:"run_counts"`
	NextRuns     map[string]time.Time `json:"next_runs"`
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
		return fmt.Errorf("scheduler: unknown task: %s", taskName)
	}
}

// Helper functions

func (s *Scheduler) getActiveTenants(ctx context.Context) ([]uuid.UUID, error) {
	query := `
		SELECT DISTINCT tenant_id
		FROM tenants
		WHERE status = 'active'
			AND deleted_at IS NULL
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

func (s *Scheduler) storeComplianceReport(ctx context.Context, report interface{}) error {
	// Simplified - would use repository in full implementation
	return nil
}

func (s *Scheduler) getDueReportSchedules(ctx context.Context) []ReportSchedule {
	// Implementation would scan and return schedules
	return []ReportSchedule{}
}

func (s *Scheduler) generateScheduledReport(ctx context.Context, schedule *ReportSchedule) error {
	// Generate report based on schedule configuration
	return nil
}

func (s *Scheduler) getNextRunTimes(entries []cron.Entry) map[string]time.Time {
	nextRuns := make(map[string]time.Time)
	for _, entry := range entries {
		if !entry.Next.IsZero() {
			// Use entry ID as key since job names aren't easily extracted
			nextRuns[fmt.Sprintf("job-%d", entry.ID)] = entry.Next
		}
	}
	return nextRuns
}

// ReportSchedule represents a scheduled report
type ReportSchedule struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Name      string    `json:"name"`
	Framework string    `json:"framework"`
	NextRunAt time.Time `json:"next_run_at"`
}
