package analytics

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDB sets up a test database connection
func setupTestDB(t *testing.T) *sqlx.DB {
	db, err := sqlx.Connect("postgres", "dbname=openpam_test sslmode=disable user=openpam password=openpam host=localhost port=5432")
	if err != nil {
		t.Skipf("Skipping test: cannot connect to test database: %v", err)
	}
	return db
}

// TestReportRepository_CreateComplianceReport tests creating a compliance report
func TestReportRepository_CreateComplianceReport(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewReportRepository(db, testLogger())
	ctx := context.Background()

	tenantID := uuid.New()
	reportData := mustMarshalJSON(map[string]interface{}{
		"test": "data",
	})

	report := &ComplianceReport{
		TenantID:       tenantID,
		Framework:      "SOC2",
		Status:         "completed",
		GeneratedAt:    time.Now(),
		PeriodStart:    time.Now().Add(-30 * 24 * time.Hour),
		PeriodEnd:      time.Now(),
		OverallScore:   85.5,
		PassedControls: 50,
		FailedControls: 5,
		Data:           reportData,
	}

	// Set tenant context for RLS
	ctx = context.WithValue(ctx, "app.tenant_id", tenantID.String())

	err := repo.CreateComplianceReport(ctx, report)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, report.ID)
	assert.False(t, report.CreatedAt.IsZero())
	assert.False(t, report.UpdatedAt.IsZero())

	// Cleanup
	_, _ = db.Exec("DELETE FROM compliance_reports WHERE id = $1", report.ID)
}

// TestReportRepository_GetComplianceReportByID tests retrieving a compliance report
func TestReportRepository_GetComplianceReportByID(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewReportRepository(db, testLogger())
	ctx := context.Background()

	tenantID := uuid.New()
	reportData := mustMarshalJSON(map[string]interface{}{
		"test": "data",
	})

	// Create a report first
	report := &ComplianceReport{
		TenantID:       tenantID,
		Framework:      "ISO27001",
		Status:         "completed",
		GeneratedAt:    time.Now(),
		PeriodStart:    time.Now().Add(-30 * 24 * time.Hour),
		PeriodEnd:      time.Now(),
		OverallScore:   90.0,
		PassedControls: 100,
		FailedControls: 2,
		Data:           reportData,
	}

	ctx = context.WithValue(ctx, "app.tenant_id", tenantID.String())
	err := repo.CreateComplianceReport(ctx, report)
	require.NoError(t, err)

	// Retrieve the report
	fetched, err := repo.GetComplianceReportByID(ctx, report.ID, tenantID)
	assert.NoError(t, err)
	assert.Equal(t, report.ID, fetched.ID)
	assert.Equal(t, report.Framework, fetched.Framework)
	assert.Equal(t, report.Status, fetched.Status)
	assert.Equal(t, report.OverallScore, fetched.OverallScore)

	// Cleanup
	_, _ = db.Exec("DELETE FROM compliance_reports WHERE id = $1", report.ID)
}

// TestReportRepository_ListComplianceReports tests listing compliance reports with filters
func TestReportRepository_ListComplianceReports(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewReportRepository(db, testLogger())
	ctx := context.Background()

	tenantID := uuid.New()

	// Create multiple reports
	for i := 0; i < 5; i++ {
		report := &ComplianceReport{
			TenantID:       tenantID,
			Framework:      []string{"SOC2", "ISO27001"}[i%2],
			Status:         "completed",
			GeneratedAt:    time.Now().Add(-time.Duration(i) * 24 * time.Hour),
			PeriodStart:    time.Now().Add(-30 * 24 * time.Hour),
			PeriodEnd:      time.Now(),
			OverallScore:   float64(80 + i),
			PassedControls: 50 + i,
			FailedControls: 5,
			Data:           mustMarshalJSON(map[string]interface{}{"index": i}),
		}
		ctx = context.WithValue(ctx, "app.tenant_id", tenantID.String())
		err := repo.CreateComplianceReport(ctx, report)
		require.NoError(t, err)
	}

	// List all reports
	filter := ComplianceReportFilter{
		TenantID: tenantID,
		Limit:    10,
		Offset:   0,
	}
	ctx = context.WithValue(ctx, "app.tenant_id", tenantID.String())
	reports, total, err := repo.ListComplianceReports(ctx, tenantID, filter)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, total, 5)
	assert.GreaterOrEqual(t, len(reports), 5)

	// Filter by framework
	filter.Framework = "SOC2"
	reports, total, err = repo.ListComplianceReports(ctx, tenantID, filter)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, total, 2)

	// Cleanup
	_, _ = db.Exec("DELETE FROM compliance_reports WHERE tenant_id = $1", tenantID)
}

// TestReportRepository_UpdateComplianceReportStatus tests updating report status
func TestReportRepository_UpdateComplianceReportStatus(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewReportRepository(db, testLogger())
	ctx := context.Background()

	tenantID := uuid.New()
	report := &ComplianceReport{
		TenantID:       tenantID,
		Framework:      "SOC2",
		Status:         "pending",
		GeneratedAt:    time.Now(),
		PeriodStart:    time.Now().Add(-30 * 24 * time.Hour),
		PeriodEnd:      time.Now(),
		OverallScore:   0,
		PassedControls: 0,
		FailedControls: 0,
	}

	ctx = context.WithValue(ctx, "app.tenant_id", tenantID.String())
	err := repo.CreateComplianceReport(ctx, report)
	require.NoError(t, err)

	newData := mustMarshalJSON(map[string]interface{}{"updated": true})
	err = repo.UpdateComplianceReportStatus(ctx, report.ID, tenantID, "completed", &newData)
	assert.NoError(t, err)

	// Verify update
	updated, err := repo.GetComplianceReportByID(ctx, report.ID, tenantID)
	assert.NoError(t, err)
	assert.Equal(t, "completed", updated.Status)

	// Cleanup
	_, _ = db.Exec("DELETE FROM compliance_reports WHERE id = $1", report.ID)
}

// TestReportRepository_CreateReportSnapshot tests creating a report snapshot
func TestReportRepository_CreateReportSnapshot(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewReportRepository(db, testLogger())
	ctx := context.Background()

	tenantID := uuid.New()
	reportID := uuid.New()
	generatedBy := uuid.New()
	format := "pdf"

	snapshot := &ReportSnapshot{
		TenantID:     tenantID,
		ReportID:     reportID,
		SnapshotName: "Test Report",
		Framework:    "SOC2",
		GeneratedAt:  time.Now(),
		GeneratedBy:  generatedBy,
		PeriodStart:  time.Now().Add(-30 * 24 * time.Hour),
		PeriodEnd:    time.Now(),
		Status:       ReportSnapshotStatusPending,
		FileFormat:   &format,
	}

	ctx = context.WithValue(ctx, "app.tenant_id", tenantID.String())
	err := repo.CreateReportSnapshot(ctx, snapshot)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, snapshot.ID)

	// Cleanup
	_, _ = db.Exec("DELETE FROM report_snapshots WHERE id = $1", snapshot.ID)
}

// TestReportRepository_GetReportSnapshotByID tests retrieving a report snapshot
func TestReportRepository_GetReportSnapshotByID(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewReportRepository(db, testLogger())
	ctx := context.Background()

	tenantID := uuid.New()
	reportID := uuid.New()
	generatedBy := uuid.New()
	format := "xlsx"

	snapshot := &ReportSnapshot{
		TenantID:     tenantID,
		ReportID:     reportID,
		SnapshotName: "Test Report",
		Framework:    "ISO27001",
		GeneratedAt:  time.Now(),
		GeneratedBy:  generatedBy,
		PeriodStart:  time.Now().Add(-30 * 24 * time.Hour),
		PeriodEnd:    time.Now(),
		Status:       ReportSnapshotStatusCompleted,
		FileFormat:   &format,
	}

	ctx = context.WithValue(ctx, "app.tenant_id", tenantID.String())
	err := repo.CreateReportSnapshot(ctx, snapshot)
	require.NoError(t, err)

	// Retrieve the snapshot
	fetched, err := repo.GetReportSnapshotByID(ctx, snapshot.ID, tenantID)
	assert.NoError(t, err)
	assert.Equal(t, snapshot.ID, fetched.ID)
	assert.Equal(t, snapshot.SnapshotName, fetched.SnapshotName)
	assert.Equal(t, snapshot.Framework, fetched.Framework)

	// Cleanup
	_, _ = db.Exec("DELETE FROM report_snapshots WHERE id = $1", snapshot.ID)
}

// TestReportRepository_ListReportSnapshots tests listing report snapshots with filters
func TestReportRepository_ListReportSnapshots(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewReportRepository(db, testLogger())
	ctx := context.Background()

	tenantID := uuid.New()
	reportID := uuid.New()
	generatedBy := uuid.New()

	// Create multiple snapshots
	for i := 0; i < 5; i++ {
		format := "pdf"
		snapshot := &ReportSnapshot{
			TenantID:     tenantID,
			ReportID:     reportID,
			SnapshotName: "Test Report",
			Framework:    []string{"SOC2", "ISO27001"}[i%2],
			GeneratedAt:  time.Now().Add(-time.Duration(i) * 24 * time.Hour),
			GeneratedBy:  generatedBy,
			PeriodStart:  time.Now().Add(-30 * 24 * time.Hour),
			PeriodEnd:    time.Now(),
			Status:       ReportSnapshotStatus([]ReportSnapshotStatus{ReportSnapshotStatusPending, ReportSnapshotStatusCompleted, ReportSnapshotStatusFailed, ReportSnapshotStatusExpired}[i%4]),
			FileFormat:   &format,
		}
		ctx = context.WithValue(ctx, "app.tenant_id", tenantID.String())
		err := repo.CreateReportSnapshot(ctx, snapshot)
		require.NoError(t, err)
	}

	// List all snapshots
	filter := ReportSnapshotFilter{
		TenantID: tenantID,
		Limit:    10,
		Offset:   0,
	}
	ctx = context.WithValue(ctx, "app.tenant_id", tenantID.String())
	snapshots, total, err := repo.ListReportSnapshots(ctx, tenantID, filter)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, total, 5)
	assert.GreaterOrEqual(t, len(snapshots), 5)

	// Filter by status
	filter.Status = "completed"
	snapshots, total, err = repo.ListReportSnapshots(ctx, tenantID, filter)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, total, 1)

	// Cleanup
	_, _ = db.Exec("DELETE FROM report_snapshots WHERE tenant_id = $1", tenantID)
}

// TestReportRepository_UpdateReportSnapshotStatus tests updating snapshot status
func TestReportRepository_UpdateReportSnapshotStatus(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewReportRepository(db, testLogger())
	ctx := context.Background()

	tenantID := uuid.New()
	reportID := uuid.New()
	generatedBy := uuid.New()
	format := "pdf"

	snapshot := &ReportSnapshot{
		TenantID:     tenantID,
		ReportID:     reportID,
		SnapshotName: "Test Report",
		Framework:    "SOC2",
		GeneratedAt:  time.Now(),
		GeneratedBy:  generatedBy,
		PeriodStart:  time.Now().Add(-30 * 24 * time.Hour),
		PeriodEnd:    time.Now(),
		Status:       ReportSnapshotStatusPending,
		FileFormat:   &format,
	}

	ctx = context.WithValue(ctx, "app.tenant_id", tenantID.String())
	err := repo.CreateReportSnapshot(ctx, snapshot)
	require.NoError(t, err)

	// Update status to completed
	fileURL := "https://storage.example.com/report.pdf"
	fileSize := int64(12345)
	err = repo.UpdateReportSnapshotStatus(ctx, snapshot.ID, tenantID, ReportSnapshotStatusCompleted, &fileURL, &fileSize, nil)
	assert.NoError(t, err)

	// Verify update
	updated, err := repo.GetReportSnapshotByID(ctx, snapshot.ID, tenantID)
	assert.NoError(t, err)
	assert.Equal(t, ReportSnapshotStatusCompleted, updated.Status)
	assert.Equal(t, &fileURL, updated.FileURL)
	assert.Equal(t, &fileSize, updated.FileSizeBytes)

	// Cleanup
	_, _ = db.Exec("DELETE FROM report_snapshots WHERE id = $1", snapshot.ID)
}

// TestReportRepository_CreateReportGenerationJob tests creating a report generation job
func TestReportRepository_CreateReportGenerationJob(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewReportRepository(db, testLogger())
	ctx := context.Background()

	tenantID := uuid.New()
	reportID := uuid.New()
	options := mustMarshalJSON(map[string]interface{}{"include_charts": true})

	job := &ReportGenerationJob{
		TenantID:  tenantID,
		JobType:   "compliance",
		ReportID:  &reportID,
		Status:    ReportJobStatusQueued,
		Format:    "pdf",
		Options:   options,
		MaxRetries: 3,
	}

	ctx = context.WithValue(ctx, "app.tenant_id", tenantID.String())
	err := repo.CreateReportGenerationJob(ctx, job)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, job.ID)
	assert.False(t, job.CreatedAt.IsZero())

	// Cleanup
	_, _ = db.Exec("DELETE FROM report_generation_jobs WHERE id = $1", job.ID)
}

// TestReportRepository_GetReportGenerationJobByID tests retrieving a report generation job
func TestReportRepository_GetReportGenerationJobByID(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewReportRepository(db, testLogger())
	ctx := context.Background()

	tenantID := uuid.New()
	reportID := uuid.New()

	job := &ReportGenerationJob{
		TenantID:  tenantID,
		JobType:   "compliance",
		ReportID:  &reportID,
		Status:    ReportJobStatusQueued,
		Format:    "pdf",
		Options:   mustMarshalJSON(map[string]interface{}{}),
	}

	ctx = context.WithValue(ctx, "app.tenant_id", tenantID.String())
	err := repo.CreateReportGenerationJob(ctx, job)
	require.NoError(t, err)

	// Retrieve the job
	fetched, err := repo.GetReportGenerationJobByID(ctx, job.ID, tenantID)
	assert.NoError(t, err)
	assert.Equal(t, job.ID, fetched.ID)
	assert.Equal(t, job.JobType, fetched.JobType)
	assert.Equal(t, job.Format, fetched.Format)

	// Cleanup
	_, _ = db.Exec("DELETE FROM report_generation_jobs WHERE id = $1", job.ID)
}

// TestReportRepository_ListReportGenerationJobs tests listing report generation jobs
func TestReportRepository_ListReportGenerationJobs(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewReportRepository(db, testLogger())
	ctx := context.Background()

	tenantID := uuid.New()

	// Create multiple jobs
	for i := 0; i < 5; i++ {
		job := &ReportGenerationJob{
			TenantID: tenantID,
			JobType:  []string{"compliance", "analytics"}[i%2],
			Status:   []ReportJobStatus{ReportJobStatusQueued, ReportJobStatusProcessing, ReportJobStatusCompleted, ReportJobStatusFailed, ReportJobStatusCancelled}[i%5],
			Format:   "pdf",
			Options:  mustMarshalJSON(map[string]interface{}{"index": i}),
		}
		ctx = context.WithValue(ctx, "app.tenant_id", tenantID.String())
		err := repo.CreateReportGenerationJob(ctx, job)
		require.NoError(t, err)
	}

	// List all jobs
	filter := ReportJobFilter{
		TenantID: tenantID,
		Limit:    10,
		Offset:   0,
	}
	ctx = context.WithValue(ctx, "app.tenant_id", tenantID.String())
	jobs, total, err := repo.ListReportGenerationJobs(ctx, tenantID, filter)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, total, 5)
	assert.GreaterOrEqual(t, len(jobs), 5)

	// Filter by status
	filter.Status = "queued"
	jobs, total, err = repo.ListReportGenerationJobs(ctx, tenantID, filter)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, total, 1)

	// Cleanup
	_, _ = db.Exec("DELETE FROM report_generation_jobs WHERE tenant_id = $1", tenantID)
}

// TestReportRepository_UpdateReportGenerationJobStatus tests updating job status
func TestReportRepository_UpdateReportGenerationJobStatus(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewReportRepository(db, testLogger())
	ctx := context.Background()

	tenantID := uuid.New()

	job := &ReportGenerationJob{
		TenantID: tenantID,
		JobType:  "compliance",
		Status:   ReportJobStatusQueued,
		Format:   "pdf",
		Options:  mustMarshalJSON(map[string]interface{}{}),
	}

	ctx = context.WithValue(ctx, "app.tenant_id", tenantID.String())
	err := repo.CreateReportGenerationJob(ctx, job)
	require.NoError(t, err)

	// Update status to processing
	snapshotID := uuid.New()
	err = repo.UpdateReportGenerationJobStatus(ctx, job.ID, tenantID, ReportJobStatusProcessing, 50, nil, &snapshotID)
	assert.NoError(t, err)

	// Verify update
	updated, err := repo.GetReportGenerationJobByID(ctx, job.ID, tenantID)
	assert.NoError(t, err)
	assert.Equal(t, ReportJobStatusProcessing, updated.Status)
	assert.Equal(t, 50, updated.Progress)
	assert.Equal(t, &snapshotID, updated.SnapshotID)

	// Cleanup
	_, _ = db.Exec("DELETE FROM report_generation_jobs WHERE id = $1", job.ID)
}

// TestReportRepository_CreateReportSchedule tests creating a report schedule
func TestReportRepository_CreateReportSchedule(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewReportRepository(db, testLogger())
	ctx := context.Background()

	tenantID := uuid.New()
	reportID := uuid.New()
	createdBy := uuid.New()
	ownedBy := uuid.New()

	schedule := &ReportSchedule{
		TenantID:           tenantID,
		ScheduleName:       "Weekly SOC2 Report",
		Framework:          "SOC2",
		ReportID:           reportID,
		ScheduleType:       ReportScheduleWeekly,
		Format:             "pdf",
		Recipients:         []string{"admin@example.com"},
		NotifyOnCompletion: true,
		NotifyOnFailure:    true,
		Status:             ReportScheduleStatusActive,
		NextRunAt:          time.Now().Add(7 * 24 * time.Hour),
		RetentionDays:      90,
		CreatedBy:          createdBy,
		OwnedBy:            ownedBy,
	}

	ctx = context.WithValue(ctx, "app.tenant_id", tenantID.String())
	err := repo.CreateReportSchedule(ctx, schedule)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, schedule.ID)

	// Cleanup
	_, _ = db.Exec("DELETE FROM report_schedules WHERE id = $1", schedule.ID)
}

// TestReportRepository_GetReportScheduleByID tests retrieving a report schedule
func TestReportRepository_GetReportScheduleByID(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewReportRepository(db, testLogger())
	ctx := context.Background()

	tenantID := uuid.New()
	reportID := uuid.New()
	createdBy := uuid.New()
	ownedBy := uuid.New()

	schedule := &ReportSchedule{
		TenantID:           tenantID,
		ScheduleName:       "Monthly ISO Report",
		Framework:          "ISO27001",
		ReportID:           reportID,
		ScheduleType:       ReportScheduleMonthly,
		Format:             "xlsx",
		Recipients:         []string{"admin@example.com"},
		NotifyOnCompletion: true,
		NotifyOnFailure:    true,
		Status:             ReportScheduleStatusActive,
		NextRunAt:          time.Now().Add(30 * 24 * time.Hour),
		RetentionDays:      90,
		CreatedBy:          createdBy,
		OwnedBy:            ownedBy,
	}

	ctx = context.WithValue(ctx, "app.tenant_id", tenantID.String())
	err := repo.CreateReportSchedule(ctx, schedule)
	require.NoError(t, err)

	// Retrieve the schedule
	fetched, err := repo.GetReportScheduleByID(ctx, schedule.ID, tenantID)
	assert.NoError(t, err)
	assert.Equal(t, schedule.ID, fetched.ID)
	assert.Equal(t, schedule.ScheduleName, fetched.ScheduleName)
	assert.Equal(t, schedule.Framework, fetched.Framework)

	// Cleanup
	_, _ = db.Exec("DELETE FROM report_schedules WHERE id = $1", schedule.ID)
}

// TestReportRepository_ListReportSchedules tests listing report schedules
func TestReportRepository_ListReportSchedules(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewReportRepository(db, testLogger())
	ctx := context.Background()

	tenantID := uuid.New()
	reportID := uuid.New()
	createdBy := uuid.New()
	ownedBy := uuid.New()

	// Create multiple schedules
	for i := 0; i < 5; i++ {
		schedule := &ReportSchedule{
			TenantID:           tenantID,
			ScheduleName:       "Schedule " + string(rune('A'+i)),
			Framework:          []string{"SOC2", "ISO27001"}[i%2],
			ReportID:           reportID,
			ScheduleType:       []ReportScheduleType{ReportScheduleDaily, ReportScheduleWeekly, ReportScheduleMonthly, ReportScheduleQuarterly, ReportScheduleYearly}[i%5],
			Format:             "pdf",
			Recipients:         []string{"admin@example.com"},
			NotifyOnCompletion: true,
			NotifyOnFailure:    true,
			Status:             ReportScheduleStatusActive,
			NextRunAt:          time.Now().Add(time.Duration(i+1) * 24 * time.Hour),
			RetentionDays:      90,
			CreatedBy:          createdBy,
			OwnedBy:            ownedBy,
		}
		ctx = context.WithValue(ctx, "app.tenant_id", tenantID.String())
		err := repo.CreateReportSchedule(ctx, schedule)
		require.NoError(t, err)
	}

	// List all schedules
	filter := ReportScheduleFilter{
		TenantID: tenantID,
		Limit:    10,
		Offset:   0,
	}
	ctx = context.WithValue(ctx, "app.tenant_id", tenantID.String())
	schedules, total, err := repo.ListReportSchedules(ctx, tenantID, filter)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, total, 5)
	assert.GreaterOrEqual(t, len(schedules), 5)

	// Filter by framework
	filter.Framework = "SOC2"
	schedules, total, err = repo.ListReportSchedules(ctx, tenantID, filter)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, total, 2)

	// Cleanup
	_, _ = db.Exec("DELETE FROM report_schedules WHERE tenant_id = $1", tenantID)
}

// TestReportRepository_UpdateReportSchedule tests updating a report schedule
func TestReportRepository_UpdateReportSchedule(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewReportRepository(db, testLogger())
	ctx := context.Background()

	tenantID := uuid.New()
	reportID := uuid.New()
	createdBy := uuid.New()
	ownedBy := uuid.New()

	schedule := &ReportSchedule{
		TenantID:           tenantID,
		ScheduleName:       "Original Name",
		Framework:          "SOC2",
		ReportID:           reportID,
		ScheduleType:       ReportScheduleWeekly,
		Format:             "pdf",
		Recipients:         []string{"admin@example.com"},
		NotifyOnCompletion: true,
		NotifyOnFailure:    true,
		Status:             ReportScheduleStatusActive,
		NextRunAt:          time.Now().Add(7 * 24 * time.Hour),
		RetentionDays:      90,
		CreatedBy:          createdBy,
		OwnedBy:            ownedBy,
	}

	ctx = context.WithValue(ctx, "app.tenant_id", tenantID.String())
	err := repo.CreateReportSchedule(ctx, schedule)
	require.NoError(t, err)

	// Update schedule
	schedule.ScheduleName = "Updated Name"
	schedule.Recipients = []string{"newadmin@example.com"}

	err = repo.UpdateReportSchedule(ctx, schedule)
	assert.NoError(t, err)

	// Verify update
	updated, err := repo.GetReportScheduleByID(ctx, schedule.ID, tenantID)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Name", updated.ScheduleName)
	assert.Equal(t, []string{"newadmin@example.com"}, updated.Recipients)

	// Cleanup
	_, _ = db.Exec("DELETE FROM report_schedules WHERE id = $1", schedule.ID)
}

// TestReportRepository_UpdateScheduleAfterRun tests updating schedule statistics after a run
func TestReportRepository_UpdateScheduleAfterRun(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewReportRepository(db, testLogger())
	ctx := context.Background()

	tenantID := uuid.New()
	reportID := uuid.New()
	createdBy := uuid.New()
	ownedBy := uuid.New()
	nextRunAt := time.Now().Add(7 * 24 * time.Hour)

	schedule := &ReportSchedule{
		TenantID:           tenantID,
		ScheduleName:       "Test Schedule",
		Framework:          "SOC2",
		ReportID:           reportID,
		ScheduleType:       ReportScheduleWeekly,
		Format:             "pdf",
		Recipients:         []string{"admin@example.com"},
		NotifyOnCompletion: true,
		NotifyOnFailure:    true,
		Status:             ReportScheduleStatusActive,
		NextRunAt:          nextRunAt,
		RetentionDays:      90,
		CreatedBy:          createdBy,
		OwnedBy:            ownedBy,
	}

	ctx = context.WithValue(ctx, "app.tenant_id", tenantID.String())
	err := repo.CreateReportSchedule(ctx, schedule)
	require.NoError(t, err)

	// Update after successful run
	newNextRunAt := time.Now().Add(14 * 24 * time.Hour)
	err = repo.UpdateScheduleAfterRun(ctx, schedule.ID, true, &newNextRunAt)
	assert.NoError(t, err)

	// Verify update
	updated, err := repo.GetReportScheduleByID(ctx, schedule.ID, tenantID)
	assert.NoError(t, err)
	assert.Equal(t, 1, updated.TotalRuns)
	assert.Equal(t, 1, updated.SuccessfulRuns)
	assert.Equal(t, 0, updated.FailedRuns)
	assert.NotNil(t, updated.LastRunAt)
	assert.NotNil(t, updated.LastSuccessfulRunAt)

	// Cleanup
	_, _ = db.Exec("DELETE FROM report_schedules WHERE id = $1", schedule.ID)
}

// TestReportRepository_DeleteReportSchedule tests deleting a report schedule
func TestReportRepository_DeleteReportSchedule(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewReportRepository(db, testLogger())
	ctx := context.Background()

	tenantID := uuid.New()
	reportID := uuid.New()
	createdBy := uuid.New()
	ownedBy := uuid.New()

	schedule := &ReportSchedule{
		TenantID:           tenantID,
		ScheduleName:       "Test Schedule",
		Framework:          "SOC2",
		ReportID:           reportID,
		ScheduleType:       ReportScheduleWeekly,
		Format:             "pdf",
		Recipients:         []string{"admin@example.com"},
		NotifyOnCompletion: true,
		NotifyOnFailure:    true,
		Status:             ReportScheduleStatusActive,
		NextRunAt:          time.Now().Add(7 * 24 * time.Hour),
		RetentionDays:      90,
		CreatedBy:          createdBy,
		OwnedBy:            ownedBy,
	}

	ctx = context.WithValue(ctx, "app.tenant_id", tenantID.String())
	err := repo.CreateReportSchedule(ctx, schedule)
	require.NoError(t, err)

	// Delete schedule
	err = repo.DeleteReportSchedule(ctx, schedule.ID, tenantID)
	assert.NoError(t, err)

	// Verify deletion
	_, err = repo.GetReportScheduleByID(ctx, schedule.ID, tenantID)
	assert.Error(t, err)
}

// testLogger returns a test logger
func testLogger() zerolog.Logger {
	return zerolog.New(zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
		w.NoColor = true
		w.Out = nil // Discard output
	}))
}
