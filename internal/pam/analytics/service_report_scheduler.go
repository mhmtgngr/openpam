package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/pam/analytics/repository"
	"github.com/rs/zerolog"
)

// ReportScheduler handles automated report generation based on schedules
type ReportScheduler struct {
	reportRepo *repository.ReportRepository
	reportSvc  *ReportService
	logger     zerolog.Logger

	// Worker state
	running bool
	stopCh  chan struct{}
}

// NewReportScheduler creates a new report scheduler
func NewReportScheduler(
	reportRepo *repository.ReportRepository,
	reportSvc *ReportService,
	logger zerolog.Logger,
) *ReportScheduler {
	return &ReportScheduler{
		reportRepo: reportRepo,
		reportSvc:  reportSvc,
		logger:     logger,
		stopCh:     make(chan struct{}),
	}
}

// Start begins the report scheduler
func (s *ReportScheduler) Start(ctx context.Context) {
	if s.running {
		return
	}

	s.running = true
	s.logger.Info().Msg("Starting report scheduler")

	go s.run(ctx)
}

// Stop gracefully stops the report scheduler
func (s *ReportScheduler) Stop() {
	if !s.running {
		return
	}

	s.logger.Info().Msg("Stopping report scheduler")
	close(s.stopCh)
	s.running = false
}

// run is the main scheduler loop
func (s *ReportScheduler) run(ctx context.Context) {
	// Initial check on startup
	time.Sleep(10 * time.Second)
	s.checkDueSchedules(ctx)

	// Check every minute
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info().Msg("Report scheduler context cancelled")
			return

		case <-s.stopCh:
			s.logger.Info().Msg("Report scheduler stopped")
			return

		case <-ticker.C:
			s.checkDueSchedules(ctx)
		}
	}
}

// checkDueSchedules processes all schedules that are due for execution
func (s *ReportScheduler) checkDueSchedules(ctx context.Context) {
	schedules, err := s.reportRepo.GetDueSchedules(ctx, 100)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to get due schedules")
		return
	}

	if len(schedules) == 0 {
		return
	}

	s.logger.Debug().Int("count", len(schedules)).Msg("Processing due report schedules")

	for _, schedule := range schedules {
		if err := s.processSchedule(ctx, &schedule); err != nil {
			s.logger.Error().
				Str("schedule_id", schedule.ID.String()).
				Err(err).
				Msg("Failed to process schedule")
		}
	}
}

// processSchedule processes a single report schedule
func (s *ReportScheduler) processSchedule(ctx context.Context, schedule *ReportSchedule) error {
	s.logger.Info().
		Str("schedule_id", schedule.ID.String()).
		Str("tenant_id", schedule.TenantID.String()).
		Str("framework", schedule.Framework).
		Msg("Processing report schedule")

	// Calculate report period based on schedule type
	startDate, endDate := s.calculateReportPeriod(schedule.ScheduleType)

	// Generate the report
	snapshot, err := s.reportSvc.GenerateReportSnapshot(
		ctx,
		schedule.TenantID,
		schedule.CreatedBy, // Use creator as generator
		schedule.ReportID,
		schedule.Framework,
		startDate,
		endDate,
		ReportFormat(schedule.Format),
	)

	if err != nil {
		s.logger.Error().
			Str("schedule_id", schedule.ID.String()).
			Err(err).
			Msg("Failed to generate report snapshot")

		// Update schedule as failed
		_ = s.reportRepo.UpdateScheduleAfterRun(ctx, schedule.ID, false, nil)
		return err
	}

	// Calculate next run time
	nextRunAt, err := s.calculateNextRunTime(schedule.ScheduleType, schedule.CronExpression)
	if err != nil {
		s.logger.Error().
			Str("schedule_id", schedule.ID.String()).
			Err(err).
			Msg("Failed to calculate next run time")
		nextRunAt = nil
	}

	// Update schedule after successful run
	if err := s.reportRepo.UpdateScheduleAfterRun(ctx, schedule.ID, true, nextRunAt); err != nil {
		s.logger.Error().
			Str("schedule_id", schedule.ID.String()).
			Err(err).
			Msg("Failed to update schedule after run")
	}

	s.logger.Info().
		Str("schedule_id", schedule.ID.String()).
		Str("snapshot_id", snapshot.ID.String()).
		Msg("Report generated from schedule")

	// Send notification if configured
	if schedule.NotifyOnCompletion && len(schedule.Recipients) > 0 {
		// In production, would send email/webhook notification
		s.logger.Info().
			Str("schedule_id", schedule.ID.String()).
			Strs("recipients", schedule.Recipients).
			Msg("Report completion notification would be sent")
	}

	return nil
}

// calculateReportPeriod calculates the date range for a report based on schedule type
func (s *ReportScheduler) calculateReportPeriod(scheduleType ReportScheduleType) (time.Time, time.Time) {
	now := time.Now()

	switch scheduleType {
	case ReportScheduleDaily:
		// Previous day
		endDate := now.Truncate(24 * time.Hour)
		startDate := endDate.Add(-24 * time.Hour)
		return startDate, endDate

	case ReportScheduleWeekly:
		// Previous week (Sunday to Saturday)
		endDate := now.Truncate(24 * time.Hour)
		daysSinceSunday := int(now.Weekday())
		if daysSinceSunday == 0 {
			daysSinceSunday = 7
		}
		endDate = endDate.Add(time.Duration(-daysSinceSunday) * 24 * time.Hour)
		startDate := endDate.Add(-7 * 24 * time.Hour)
		return startDate, endDate

	case ReportScheduleMonthly:
		// Previous month
		endDate := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		startDate := endDate.AddDate(0, -1, 0)
		return startDate, endDate

	case ReportScheduleQuarterly:
		// Previous quarter
		currentQuarter := (now.Month()-1)/3 + 1
		var startMonth time.Month
		if currentQuarter == 1 {
			// Q4 of previous year
			startDate := time.Date(now.Year()-1, 10, 1, 0, 0, 0, 0, now.Location())
			endDate := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
			return startDate, endDate
		}
		startMonth = (currentQuarter-2)*3 + 1
		endDate := time.Date(now.Year(), startMonth+3, 1, 0, 0, 0, 0, now.Location())
		startDate := time.Date(now.Year(), startMonth, 1, 0, 0, 0, 0, now.Location())
		return startDate, endDate

	case ReportScheduleYearly:
		// Previous year
		endDate := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		startDate := endDate.AddDate(-1, 0, 0)
		return startDate, endDate

	default:
		// Default to last 7 days
		endDate := now.Truncate(24 * time.Hour)
		startDate := endDate.Add(-7 * 24 * time.Hour)
		return startDate, endDate
	}
}

// calculateNextRunTime calculates the next run time for a schedule
func (s *ReportScheduler) calculateNextRunTime(scheduleType ReportScheduleType, cronExpr *string) (time.Time, error) {
	now := time.Now()

	// If custom cron expression provided, use it
	if cronExpr != nil && scheduleType == ReportScheduleCustom {
		// In production, would parse cron expression
		// For now, return tomorrow at same time
		return now.Add(24 * time.Hour), nil
	}

	switch scheduleType {
	case ReportScheduleDaily:
		// Next run at midnight tomorrow
		next := now.Add(24 * time.Hour)
		return time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, next.Location()), nil

	case ReportScheduleWeekly:
		// Next run at next Sunday midnight
		daysUntilSunday := (7 - int(now.Weekday())) % 7
		if daysUntilSunday == 0 {
			daysUntilSunday = 7
		}
		next := now.AddDate(0, 0, daysUntilSunday)
		return time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, next.Location()), nil

	case ReportScheduleMonthly:
		// Next run at first of next month midnight
		next := now.AddDate(0, 1, 0)
		return time.Date(next.Year(), next.Month(), 1, 0, 0, 0, 0, next.Location()), nil

	case ReportScheduleQuarterly:
		// Next run at first of next quarter
		nextMonth := now.Month() + 1
		if nextMonth > 12 {
			nextMonth = 1
		}
		quarterStartMonth := ((nextMonth-1)/3)*3 + 1
		if quarterStartMonth <= nextMonth {
			// Move to next quarter
			quarterStartMonth += 3
			if quarterStartMonth > 12 {
				quarterStartMonth = 1
			}
		}
		next := time.Date(now.Year(), quarterStartMonth, 1, 0, 0, 0, 0, now.Location())
		return next, nil

	case ReportScheduleYearly:
		// Next run at January 1 of next year
		next := time.Date(now.Year()+1, 1, 1, 0, 0, 0, 0, now.Location())
		return next, nil

	default:
		return now.Add(24 * time.Hour), nil
	}
}

// PauseSchedule pauses a report schedule
func (s *ReportScheduler) PauseSchedule(ctx context.Context, id uuid.UUID) error {
	schedule, err := s.reportRepo.GetReportSchedule(ctx, id)
	if err != nil {
		return err
	}

	schedule.Status = ReportScheduleStatusPaused
	return s.reportRepo.UpdateReportSchedule(ctx, schedule)
}

// ResumeSchedule resumes a paused report schedule
func (s *ReportScheduler) ResumeSchedule(ctx context.Context, id uuid.UUID) error {
	schedule, err := s.reportRepo.GetReportSchedule(ctx, id)
	if err != nil {
		return err
	}

	schedule.Status = ReportScheduleStatusActive
	return s.reportRepo.UpdateReportSchedule(ctx, schedule)
}

// TriggerSchedule manually triggers a schedule to run immediately
func (s *ReportScheduler) TriggerSchedule(ctx context.Context, id uuid.UUID) (*ReportSnapshot, error) {
	schedule, err := s.reportRepo.GetReportSchedule(ctx, id)
	if err != nil {
		return nil, err
	}

	startDate, endDate := s.calculateReportPeriod(schedule.ScheduleType)

	snapshot, err := s.reportSvc.GenerateReportSnapshot(
		ctx,
		schedule.TenantID,
		schedule.CreatedBy,
		schedule.ReportID,
		schedule.Framework,
		startDate,
		endDate,
		ReportFormat(schedule.Format),
	)

	if err != nil {
		return nil, err
	}

	// Update schedule run count
	_ = s.reportRepo.UpdateScheduleAfterRun(ctx, schedule.ID, true, nil)

	return snapshot, nil
}
