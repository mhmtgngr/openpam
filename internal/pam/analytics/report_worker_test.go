package analytics

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReportWorker_Construction tests the construction of ReportWorker
func TestReportWorker_Construction(t *testing.T) {
	logger := zerolog.New(&testWriter{t: t})

	tests := []struct {
		name     string
		workers  int
		expected int
	}{
		{
			name:     "default worker count",
			workers:  0,
			expected: 3,
		},
		{
			name:     "custom worker count",
			workers:  5,
			expected: 5,
		},
		{
			name:     "negative worker count uses default",
			workers:  -1,
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			worker := NewReportWorker(nil, nil, logger, tt.workers)

			assert.NotNil(t, worker)
			assert.Equal(t, tt.expected, worker.workers)
			assert.NotNil(t, worker.jobQueue)
			assert.NotNil(t, worker.stopCh)
			assert.False(t, worker.running, "Worker should not be running initially")
		})
	}
}

// TestReportWorker_GetStatus tests the GetStatus method
func TestReportWorker_GetStatus(t *testing.T) {
	logger := zerolog.New(&testWriter{t: t})
	worker := NewReportWorker(nil, nil, logger, 3)

	t.Run("status when not running", func(t *testing.T) {
		status := worker.GetStatus()

		assert.False(t, status.Running)
		assert.NotEmpty(t, status.WorkerID)
		assert.Equal(t, 3, status.Workers)
		assert.Equal(t, 0, status.ActiveJobs)
		assert.Equal(t, 0, status.QueuedJobs)
		assert.False(t, status.LastActivity.IsZero())
	})

	t.Run("status when running with active jobs", func(t *testing.T) {
		worker.running = true

		// Add fake active jobs
		worker.activeJobs.Store("test-job-1", true)
		worker.activeJobs.Store("test-job-2", true)

		status := worker.GetStatus()

		assert.True(t, status.Running)
		assert.Equal(t, 2, status.ActiveJobs)
	})
}

// TestReportWorker_Stop tests the Stop method
func TestReportWorker_Stop(t *testing.T) {
	logger := zerolog.New(&testWriter{t: t})
	worker := NewReportWorker(nil, nil, logger, 2)

	// Start the worker
	worker.running = true
	worker.jobQueue = make(chan *ReportGenerationJob, 100)

	// Stop the worker
	worker.Stop()

	assert.False(t, worker.running, "Worker should not be running after stop")
}

// TestReportWorker_EnqueueJob tests the EnqueueJob method
func TestReportWorker_EnqueueJob(t *testing.T) {
	logger := zerolog.New(&testWriter{t: t})
	worker := NewReportWorker(nil, nil, logger, 2)
	worker.running = true

	jobID := uuid.New()
	tenantID := uuid.New()
	snapshotID := uuid.New()
	reportID := uuid.New()

	job := &ReportGenerationJob{
		ID:         jobID,
		TenantID:   tenantID,
		SnapshotID: &snapshotID,
		ReportID:   &reportID,
		JobType:    "compliance",
		Format:     "pdf",
		Status:     ReportJobStatusQueued,
		MaxRetries: 3,
	}

	t.Run("enqueue when running", func(t *testing.T) {
		worker.EnqueueJob(job)

		select {
		case <-time.After(100 * time.Millisecond):
			// Job should be in queue
			assert.Equal(t, 1, len(worker.jobQueue))
		case <-worker.jobQueue:
			// Job was dequeued
		}
	})

	t.Run("enqueue when not running", func(t *testing.T) {
		worker.running = false
		worker.jobQueue = make(chan *ReportGenerationJob, 1000)

		// This should not panic
		worker.EnqueueJob(job)

		select {
		case j := <-worker.jobQueue:
			t.Fatal("Job should not be enqueued when worker is not running, got:", j)
		default:
			// Expected - no job in queue
		}
	})
}

// TestReportJobStatusEnum tests ReportJobStatus enum values
func TestReportJobStatusEnum(t *testing.T) {
	tests := []struct {
		status   ReportJobStatus
		expected string
	}{
		{ReportJobStatusQueued, "queued"},
		{ReportJobStatusProcessing, "processing"},
		{ReportJobStatusCompleted, "completed"},
		{ReportJobStatusFailed, "failed"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.status))
		})
	}
}

// TestReportSnapshotStatusEnum tests ReportSnapshotStatus enum values
func TestReportSnapshotStatusEnum(t *testing.T) {
	tests := []struct {
		status   ReportSnapshotStatus
		expected string
	}{
		{ReportSnapshotStatusPending, "pending"},
		{ReportSnapshotStatusCompleted, "completed"},
		{ReportSnapshotStatusFailed, "failed"},
		{ReportSnapshotStatusExpired, "expired"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.status))
		})
	}
}

// TestHelperFunctions tests helper functions
func TestHelperFunctions(t *testing.T) {
	t.Run("stringPtr", func(t *testing.T) {
		s := "test string"
		ptr := stringPtr(s)

		require.NotNil(t, ptr)
		assert.Equal(t, s, *ptr)
	})

	t.Run("stringPtr with empty string", func(t *testing.T) {
		s := ""
		ptr := stringPtr(s)

		require.NotNil(t, ptr)
		assert.Equal(t, s, *ptr)
	})

	t.Run("snapshotIDPtr with nil", func(t *testing.T) {
		ptr := snapshotIDPtr(nil)
		assert.Nil(t, ptr)
	})

	t.Run("snapshotIDPtr with snapshot", func(t *testing.T) {
		snapshot := &ReportSnapshot{
			ID: uuid.New(),
		}
		ptr := snapshotIDPtr(snapshot)

		require.NotNil(t, ptr)
		assert.Equal(t, snapshot.ID, *ptr)
	})
}

// TestReportGenerationJob_Defaults tests ReportGenerationJob default values
func TestReportGenerationJob_Defaults(t *testing.T) {
	jobID := uuid.New()
	tenantID := uuid.New()
	snapshotID := uuid.New()
	reportID := uuid.New()

	job := ReportGenerationJob{
		ID:         jobID,
		TenantID:   tenantID,
		SnapshotID: &snapshotID,
		ReportID:   &reportID,
		JobType:    "compliance",
		Format:     "pdf",
		Status:     ReportJobStatusQueued,
		MaxRetries: 3,
	}

	assert.Equal(t, jobID, job.ID)
	assert.Equal(t, tenantID, job.TenantID)
	assert.Equal(t, snapshotID, *job.SnapshotID)
	assert.Equal(t, reportID, *job.ReportID)
	assert.Equal(t, "compliance", job.JobType)
	assert.Equal(t, "pdf", job.Format)
	assert.Equal(t, ReportJobStatusQueued, job.Status)
	assert.Equal(t, 3, job.MaxRetries)
}

// testWriter implements io.Writer for zerolog
type testWriter struct {
	t *testing.T
}

func (w *testWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}
