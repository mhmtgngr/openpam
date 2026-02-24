package analytics

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

// TestNewAggregationWorker tests aggregation worker creation
func TestNewAggregationWorker(t *testing.T) {
	t.Run("creates aggregation worker", func(t *testing.T) {
		logger := zerolog.Nop()

		worker := NewAggregationWorker(nil, nil, logger)

		assert.NotNil(t, worker)
		assert.NotNil(t, worker.logger)
		assert.NotNil(t, worker.stopCh)
		assert.False(t, worker.running)
	})
}

// TestAggregationWorker_StartStop tests worker lifecycle
func TestAggregationWorker_StartStop(t *testing.T) {
	t.Run("starts worker", func(t *testing.T) {
		logger := zerolog.Nop()

		worker := NewAggregationWorker(nil, nil, logger)
		ctx := context.Background()

		worker.Start(ctx)

		assert.True(t, worker.running)

		// Clean up
		worker.Stop()
	})

	t.Run("stops running worker", func(t *testing.T) {
		logger := zerolog.Nop()

		worker := NewAggregationWorker(nil, nil, logger)
		ctx := context.Background()

		worker.Start(ctx)
		assert.True(t, worker.running)

		worker.Stop()
		assert.False(t, worker.running)
	})

	t.Run("stop is idempotent", func(t *testing.T) {
		logger := zerolog.Nop()

		worker := NewAggregationWorker(nil, nil, logger)

		// Multiple stops should not panic
		worker.Stop()
		worker.Stop()
		worker.Stop()

		assert.False(t, worker.running)
	})
}

// TestAggregationWorker_GetStatus tests status retrieval
func TestAggregationWorker_GetStatus(t *testing.T) {
	t.Run("gets status for running worker", func(t *testing.T) {
		logger := zerolog.Nop()

		worker := NewAggregationWorker(nil, nil, logger)
		ctx := context.Background()

		worker.Start(ctx)
		defer worker.Stop()

		status := worker.GetStatus()

		assert.True(t, status.Running)
		assert.False(t, status.LastRun.IsZero())
		assert.False(t, status.NextRun.IsZero())
	})

	t.Run("gets status for stopped worker", func(t *testing.T) {
		logger := zerolog.Nop()

		worker := NewAggregationWorker(nil, nil, logger)

		status := worker.GetStatus()

		assert.False(t, status.Running)
	})
}

// TestAggregationWorker_AggregateForTenant tests manual aggregation
func TestAggregationWorker_AggregateForTenant(t *testing.T) {
	t.Run("aggregates for tenant with valid period", func(t *testing.T) {
		logger := zerolog.Nop()

		worker := NewAggregationWorker(nil, nil, logger)
		ctx := context.Background()

		tenantID := uuid.New()
		startDate := time.Now().Add(-24 * time.Hour)
		endDate := time.Now()

		err := worker.AggregateForTenant(ctx, tenantID, startDate, endDate)

		// Without database, aggregation methods will succeed (they return nil)
		assert.NoError(t, err)
	})

	t.Run("aggregates for tenant with short period", func(t *testing.T) {
		logger := zerolog.Nop()

		worker := NewAggregationWorker(nil, nil, logger)
		ctx := context.Background()

		tenantID := uuid.New()
		startDate := time.Now().Add(-time.Hour)
		endDate := time.Now()

		err := worker.AggregateForTenant(ctx, tenantID, startDate, endDate)

		assert.NoError(t, err)
	})
}

// TestNewAlertEvaluator tests alert evaluator creation
func TestNewAlertEvaluator(t *testing.T) {
	t.Run("creates alert evaluator", func(t *testing.T) {
		logger := zerolog.Nop()

		evaluator := NewAlertEvaluator(nil, nil, logger)

		assert.NotNil(t, evaluator)
		assert.NotNil(t, evaluator.logger)
		assert.NotNil(t, evaluator.stopCh)
		assert.False(t, evaluator.running)
	})
}

// TestAlertEvaluator_StartStop tests evaluator lifecycle
func TestAlertEvaluator_StartStop(t *testing.T) {
	t.Run("starts evaluator", func(t *testing.T) {
		logger := zerolog.Nop()

		evaluator := NewAlertEvaluator(nil, nil, logger)
		ctx := context.Background()

		evaluator.Start(ctx)

		assert.True(t, evaluator.running)

		// Clean up
		evaluator.Stop()
	})

	t.Run("stops running evaluator", func(t *testing.T) {
		logger := zerolog.Nop()

		evaluator := NewAlertEvaluator(nil, nil, logger)
		ctx := context.Background()

		evaluator.Start(ctx)
		assert.True(t, evaluator.running)

		evaluator.Stop()
		assert.False(t, evaluator.running)
	})

	t.Run("stop is idempotent", func(t *testing.T) {
		logger := zerolog.Nop()

		evaluator := NewAlertEvaluator(nil, nil, logger)

		evaluator.Stop()
		evaluator.Stop()
		evaluator.Stop()

		assert.False(t, evaluator.running)
	})
}

// TestAlertEvaluator_GetStatus tests evaluator status
func TestAlertEvaluator_GetStatus(t *testing.T) {
	t.Run("gets status for running evaluator", func(t *testing.T) {
		logger := zerolog.Nop()

		evaluator := NewAlertEvaluator(nil, nil, logger)
		ctx := context.Background()

		evaluator.Start(ctx)
		defer evaluator.Stop()

		status := evaluator.GetStatus()

		assert.True(t, status.Running)
		assert.False(t, status.LastEvaluation.IsZero())
		assert.False(t, status.NextEvaluation.IsZero())
	})

	t.Run("gets status for stopped evaluator", func(t *testing.T) {
		logger := zerolog.Nop()

		evaluator := NewAlertEvaluator(nil, nil, logger)

		status := evaluator.GetStatus()

		assert.False(t, status.Running)
	})
}

// TestAlertEvaluator_AlertStructure tests alert structure validation
func TestAlertEvaluator_AlertStructure(t *testing.T) {
	t.Run("creates valid threshold alert", func(t *testing.T) {
		tenantID := uuid.New()
		conditions := json.RawMessage(`{"metric": "failed_auth_rate", "operator": "greater_than", "threshold": 10}`)

		alert := &Alert{
			ID:          uuid.New(),
			TenantID:    tenantID,
			Name:        "High Failed Auth Rate",
			Description: "Alert when failed auth rate exceeds threshold",
			AlertType:   AlertTypeThreshold,
			Severity:    SeverityHigh,
			Conditions:  conditions,
			EvaluationIntervalMinutes: 15,
			Enabled:     true,
		}

		assert.Equal(t, tenantID, alert.TenantID)
		assert.Equal(t, AlertTypeThreshold, alert.AlertType)
		assert.Equal(t, SeverityHigh, alert.Severity)
		assert.True(t, alert.Enabled)
	})

	t.Run("creates valid pattern alert", func(t *testing.T) {
		tenantID := uuid.New()
		conditions := json.RawMessage(`{"pattern": "after_hours_access", "hours": {"start": 18, "end": 6}}`)

		alert := &Alert{
			ID:          uuid.New(),
			TenantID:    tenantID,
			Name:        "After Hours Access",
			AlertType:   AlertTypePattern,
			Severity:    SeverityMedium,
			Conditions:  conditions,
			EvaluationIntervalMinutes: 30,
			Enabled:     true,
		}

		assert.Equal(t, AlertTypePattern, alert.AlertType)
		assert.Equal(t, SeverityMedium, alert.Severity)
	})
}

// TestNewReportScheduler tests report scheduler creation
func TestNewReportScheduler(t *testing.T) {
	t.Run("creates report scheduler", func(t *testing.T) {
		logger := zerolog.Nop()

		scheduler := NewReportScheduler(nil, nil, logger)

		assert.NotNil(t, scheduler)
		assert.NotNil(t, scheduler.logger)
		assert.NotNil(t, scheduler.stopCh)
		assert.False(t, scheduler.running)
	})
}

// TestReportScheduler_StartStop tests scheduler lifecycle
func TestReportScheduler_StartStop(t *testing.T) {
	t.Run("starts scheduler", func(t *testing.T) {
		logger := zerolog.Nop()

		scheduler := NewReportScheduler(nil, nil, logger)
		ctx := context.Background()

		scheduler.Start(ctx)

		assert.True(t, scheduler.running)

		// Clean up
		scheduler.Stop()
	})

	t.Run("stops running scheduler", func(t *testing.T) {
		logger := zerolog.Nop()

		scheduler := NewReportScheduler(nil, nil, logger)
		ctx := context.Background()

		scheduler.Start(ctx)
		assert.True(t, scheduler.running)

		scheduler.Stop()
		assert.False(t, scheduler.running)
	})

	t.Run("stop is idempotent", func(t *testing.T) {
		logger := zerolog.Nop()

		scheduler := NewReportScheduler(nil, nil, logger)

		scheduler.Stop()
		scheduler.Stop()
		scheduler.Stop()

		assert.False(t, scheduler.running)
	})
}

// TestReportScheduler_GetStatus tests scheduler status
func TestReportScheduler_GetStatus(t *testing.T) {
	t.Run("gets status for running scheduler", func(t *testing.T) {
		logger := zerolog.Nop()

		scheduler := NewReportScheduler(nil, nil, logger)
		ctx := context.Background()

		scheduler.Start(ctx)
		defer scheduler.Stop()

		status := scheduler.GetStatus()

		assert.True(t, status.Running)
		assert.False(t, status.LastCheck.IsZero())
	})

	t.Run("gets status for stopped scheduler", func(t *testing.T) {
		logger := zerolog.Nop()

		scheduler := NewReportScheduler(nil, nil, logger)

		status := scheduler.GetStatus()

		assert.False(t, status.Running)
	})
}

// TestAggregationWorkerStatus tests status structure
func TestAggregationWorkerStatus(t *testing.T) {
	t.Run("creates aggregation worker status", func(t *testing.T) {
		now := time.Now()
		nextRun := now.Add(time.Hour)

		status := AggregationWorkerStatus{
			Running:           true,
			LastRun:           now,
			LastRunDuration:   1500,
			NextRun:           nextRun,
			ProcessedTenants:  5,
			SessionsProcessed: 1000,
			EventsProcessed:   5000,
		}

		assert.True(t, status.Running)
		assert.Equal(t, int64(1500), status.LastRunDuration)
		assert.Equal(t, 5, status.ProcessedTenants)
		assert.Equal(t, int64(1000), status.SessionsProcessed)
		assert.Equal(t, int64(5000), status.EventsProcessed)
	})
}

// TestAlertEvaluatorStatus tests evaluator status structure
func TestAlertEvaluatorStatus(t *testing.T) {
	t.Run("creates alert evaluator status", func(t *testing.T) {
		now := time.Now()
		nextEval := now.Add(15 * time.Minute)

		status := AlertEvaluatorStatus{
			Running:          true,
			LastEvaluation:   now,
			AlertsEvaluated:  10,
			AlertsTriggered:  2,
			NextEvaluation:   nextEval,
		}

		assert.True(t, status.Running)
		assert.Equal(t, 10, status.AlertsEvaluated)
		assert.Equal(t, 2, status.AlertsTriggered)
	})
}

// TestReportSchedulerStatus tests scheduler status structure
func TestReportSchedulerStatus(t *testing.T) {
	t.Run("creates report scheduler status", func(t *testing.T) {
		now := time.Now()
		nextScheduled := now.Add(time.Hour)

		status := ReportSchedulerStatus{
			Running:           true,
			LastCheck:         now,
			ReportsProcessed:  3,
			ReportsScheduled:  5,
			NextScheduledTime: &nextScheduled,
		}

		assert.True(t, status.Running)
		assert.Equal(t, 3, status.ReportsProcessed)
		assert.Equal(t, 5, status.ReportsScheduled)
		assert.NotNil(t, status.NextScheduledTime)
	})
}

// TestWorkerConcurrency tests concurrent worker operations
func TestWorkerConcurrency(t *testing.T) {
	t.Run("handles concurrent start/stop", func(t *testing.T) {
		logger := zerolog.Nop()

		worker := NewAggregationWorker(nil, nil, logger)
		ctx := context.Background()

		// Start worker
		worker.Start(ctx)
		assert.True(t, worker.running)

		// Try to start again (should be idempotent)
		worker.Start(ctx)
		assert.True(t, worker.running)

		// Stop worker
		worker.Stop()
		assert.False(t, worker.running)
	})

	t.Run("handles concurrent evaluator start/stop", func(t *testing.T) {
		logger := zerolog.Nop()

		evaluator := NewAlertEvaluator(nil, nil, logger)
		ctx := context.Background()

		evaluator.Start(ctx)
		evaluator.Start(ctx)
		assert.True(t, evaluator.running)

		evaluator.Stop()
		assert.False(t, evaluator.running)
	})
}

// TestWorkerContextCancellation tests context cancellation handling
func TestWorkerContextCancellation(t *testing.T) {
	t.Run("aggregation worker stops on context cancel", func(t *testing.T) {
		logger := zerolog.Nop()

		worker := NewAggregationWorker(nil, nil, logger)
		ctx, cancel := context.WithCancel(context.Background())

		worker.Start(ctx)
		assert.True(t, worker.running)

		// Cancel context
		cancel()

		// Give worker time to stop
		time.Sleep(10 * time.Millisecond)

		worker.Stop()
		assert.False(t, worker.running)
	})

	t.Run("alert evaluator stops on context cancel", func(t *testing.T) {
		logger := zerolog.Nop()

		evaluator := NewAlertEvaluator(nil, nil, logger)
		ctx, cancel := context.WithCancel(context.Background())

		evaluator.Start(ctx)
		assert.True(t, evaluator.running)

		cancel()

		time.Sleep(10 * time.Millisecond)

		evaluator.Stop()
		assert.False(t, evaluator.running)
	})

	t.Run("report scheduler stops on context cancel", func(t *testing.T) {
		logger := zerolog.Nop()

		scheduler := NewReportScheduler(nil, nil, logger)
		ctx, cancel := context.WithCancel(context.Background())

		scheduler.Start(ctx)
		assert.True(t, scheduler.running)

		cancel()

		time.Sleep(10 * time.Millisecond)

		scheduler.Stop()
		assert.False(t, scheduler.running)
	})
}

// TestWorkerTiming tests worker timing behavior
func TestWorkerTiming(t *testing.T) {
	t.Run("calculates next run time", func(t *testing.T) {
		logger := zerolog.Nop()

		worker := NewAggregationWorker(nil, nil, logger)

		status := worker.GetStatus()

		// Next run should be approximately 1 hour from now
		expectedNext := time.Now().Add(time.Hour)
		diff := status.NextRun.Sub(expectedNext)

		// Allow 1 second tolerance
		assert.Less(t, diff, time.Second)
		assert.Greater(t, diff, -time.Second)
	})

	t.Run("evaluator calculates next evaluation time", func(t *testing.T) {
		logger := zerolog.Nop()

		evaluator := NewAlertEvaluator(nil, nil, logger)

		status := evaluator.GetStatus()

		// Next evaluation should be approximately 15 minutes from now
		expectedNext := time.Now().Add(15 * time.Minute)
		diff := status.NextEvaluation.Sub(expectedNext)

		// Allow 1 second tolerance
		assert.Less(t, diff, time.Second)
		assert.Greater(t, diff, -time.Second)
	})
}
