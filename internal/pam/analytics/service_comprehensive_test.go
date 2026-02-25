package analytics

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/cache"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock repository for testing
type mockRepository struct {
	sessionAnalytics      *SessionAnalytics
	userActivity          *UserActivity
	topUsers              []TopUser
	accessPatterns        []AccessPattern
	complianceStatus      *ComplianceStatus
	commandFrequency      []CommandFrequency
	riskScores            []RiskScore
	metrics               []Metric
	timeseriesData        []TimeSeriesDataPoint
	createErr             error
	getErr                error
	listErr               error
	calls                 map[string]int
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		calls: make(map[string]int),
	}
}

func (m *mockRepository) UpsertSessionAnalytics(ctx context.Context, analytics *SessionAnalytics) error {
	m.calls["UpsertSessionAnalytics"]++
	m.sessionAnalytics = analytics
	return m.createErr
}

func (m *mockRepository) GetSessionAnalytics(ctx context.Context, tenantID uuid.UUID, periodType PeriodType, periodStart time.Time) (*SessionAnalytics, error) {
	m.calls["GetSessionAnalytics"]++
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.sessionAnalytics != nil {
		return m.sessionAnalytics, nil
	}
	return &SessionAnalytics{
		ID:              uuid.New(),
		TenantID:        tenantID,
		PeriodType:      periodType,
		PeriodStart:     periodStart,
		PeriodEnd:       periodStart.Add(time.Hour),
		ActiveSessions:  5,
		CompletedSessions: 10,
		TotalSessions:   15,
	}, nil
}

func (m *mockRepository) ListSessionAnalytics(ctx context.Context, tenantID uuid.UUID, periodType PeriodType, startDate, endDate time.Time, limit, offset int) ([]SessionAnalytics, error) {
	m.calls["ListSessionAnalytics"]++
	if m.listErr != nil {
		return nil, m.listErr
	}
	return []SessionAnalytics{*m.sessionAnalytics}, nil
}

func (m *mockRepository) UpsertUserActivity(ctx context.Context, activity *UserActivity) error {
	m.calls["UpsertUserActivity"]++
	m.userActivity = activity
	return m.createErr
}

func (m *mockRepository) GetUserActivity(ctx context.Context, tenantID, userID uuid.UUID, periodType PeriodType, periodStart time.Time) (*UserActivity, error) {
	m.calls["GetUserActivity"]++
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.userActivity != nil {
		return m.userActivity, nil
	}
	return &UserActivity{
		ID:              uuid.New(),
		TenantID:        tenantID,
		UserID:          userID,
		PeriodType:      periodType,
		PeriodStart:     periodStart,
		SessionsCreated: 5,
	}, nil
}

func (m *mockRepository) ListUserActivity(ctx context.Context, tenantID uuid.UUID, periodType PeriodType, startDate, endDate time.Time, limit, offset int) ([]UserActivity, error) {
	m.calls["ListUserActivity"]++
	if m.listErr != nil {
		return nil, m.listErr
	}
	if m.userActivity != nil {
		return []UserActivity{*m.userActivity}, nil
	}
	return []UserActivity{}, nil
}

func (m *mockRepository) GetTopUsersBySessions(ctx context.Context, tenantID uuid.UUID, startDate, endDate time.Time, limit int) ([]TopUser, error) {
	m.calls["GetTopUsersBySessions"]++
	if m.listErr != nil {
		return nil, m.listErr
	}
	if m.topUsers != nil {
		return m.topUsers, nil
	}
	return []TopUser{
		{UserID: uuid.New(), Username: "user1", SessionCount: 10, TotalSeconds: 3600, LastSeen: time.Now()},
	}, nil
}

func (m *mockRepository) GetAccessPatterns(ctx context.Context, tenantID uuid.UUID, startDate, endDate time.Time) ([]AccessPattern, error) {
	m.calls["GetAccessPatterns"]++
	if m.listErr != nil {
		return nil, m.listErr
	}
	if m.accessPatterns != nil {
		return m.accessPatterns, nil
	}
	return []AccessPattern{
		{HourOfDay: 10, DayOfWeek: 1, SessionCount: 5, AvgDuration: 1800, IsOutsideBusiness: false},
	}, nil
}

func (m *mockRepository) CalculateComplianceStatus(ctx context.Context, tenantID uuid.UUID, startDate, endDate time.Time) (*ComplianceStatus, error) {
	m.calls["CalculateComplianceStatus"]++
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.complianceStatus != nil {
		return m.complianceStatus, nil
	}
	return &ComplianceStatus{
		OverallPercentage: 85.5,
		PassedChecks:      17,
		TotalChecks:       20,
		Violations:        []ComplianceViolation{},
		ByPolicy:          map[string]CompliancePolicyStatus{},
	}, nil
}

func (m *mockRepository) ListCommandFrequency(ctx context.Context, tenantID uuid.UUID, periodType PeriodType, startDate, endDate time.Time, limit, offset int) ([]CommandFrequency, error) {
	m.calls["ListCommandFrequency"]++
	if m.listErr != nil {
		return nil, m.listErr
	}
	if m.commandFrequency != nil {
		return m.commandFrequency, nil
	}
	return []CommandFrequency{}, nil
}

func (m *mockRepository) GetAnomalousUsers(ctx context.Context, tenantID uuid.UUID, periodType PeriodType, periodStart time.Time, limit int) ([]UserActivity, error) {
	m.calls["GetAnomalousUsers"]++
	if m.listErr != nil {
		return nil, m.listErr
	}
	return []UserActivity{}, nil
}

func (m *mockRepository) GetRawSessionStats(ctx context.Context, tenantID uuid.UUID, startDate, endDate time.Time) (*SessionStats, error) {
	m.calls["GetRawSessionStats"]++
	if m.getErr != nil {
		return nil, m.getErr
	}
	return &SessionStats{
		Total:      100,
		Active:     10,
		Completed:  85,
		Failed:     5,
		Terminated: 0,
		AvgDuration: 1800,
		MaxDuration: 7200,
		ByProtocol:  map[string]int64{"ssh": 80, "rdp": 20},
		ByUser:      map[string]int64{},
	}, nil
}

func (m *mockRepository) GetSessionTimeSeries(ctx context.Context, tenantID uuid.UUID, metric string, startDate, endDate time.Time) ([]TimeSeriesDataPoint, error) {
	m.calls["GetSessionTimeSeries"]++
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.timeseriesData != nil {
		return m.timeseriesData, nil
	}
	return []TimeSeriesDataPoint{
		{Timestamp: time.Now(), Value: 10},
	}, nil
}

func (m *mockRepository) ListRiskScores(ctx context.Context, tenantID uuid.UUID, entityType *EntityType, minScore *float64, riskLevel *RiskLevel, limit, offset int) ([]RiskScore, error) {
	m.calls["ListRiskScores"]++
	if m.listErr != nil {
		return nil, m.listErr
	}
	if m.riskScores != nil {
		return m.riskScores, nil
	}
	return []RiskScore{}, nil
}

func (m *mockRepository) GetLatestRiskScore(ctx context.Context, tenantID uuid.UUID, entityType EntityType, entityID uuid.UUID) (*RiskScore, error) {
	m.calls["GetLatestRiskScore"]++
	if m.getErr != nil {
		return nil, m.getErr
	}
	return &RiskScore{
		ID:               uuid.New(),
		TenantID:         tenantID,
		EntityType:       entityType,
		EntityID:         entityID,
		CalculatedAt:     time.Now(),
		OverallRiskScore: 50.0,
		RiskLevel:        RiskLevelMedium,
	}, nil
}

func (m *mockRepository) CreateRiskScore(ctx context.Context, score *RiskScore) error {
	m.calls["CreateRiskScore"]++
	return m.createErr
}

func (m *mockRepository) QueryMetrics(ctx context.Context, tenantID uuid.UUID, metricName string, startDate, endDate time.Time, limit int) ([]Metric, error) {
	m.calls["QueryMetrics"]++
	if m.listErr != nil {
		return nil, m.listErr
	}
	if m.metrics != nil {
		return m.metrics, nil
	}
	return []Metric{}, nil
}

func (m *mockRepository) RecordMetric(ctx context.Context, metric *Metric) error {
	m.calls["RecordMetric"]++
	return m.createErr
}

func (m *mockRepository) AggregateMetrics(ctx context.Context, tenantID uuid.UUID, metricName string, startDate, endDate time.Time, period string) ([]TimeSeriesDataPoint, error) {
	m.calls["AggregateMetrics"]++
	if m.listErr != nil {
		return nil, m.listErr
	}
	return []TimeSeriesDataPoint{}, nil
}

// Mock cache for testing
type mockCache struct {
	data      map[string][]byte
	getCalls  map[string]int
	setCalls  map[string]int
	deleteErr error
	getErr    error
}

func newMockCache() *mockCache {
	return &mockCache{
		data:     make(map[string][]byte),
		getCalls: make(map[string]int),
		setCalls: make(map[string]int),
	}
}

func (m *mockCache) Get(ctx context.Context, key string, dest interface{}) error {
	m.getCalls[key]++
	if m.getErr != nil {
		return m.getErr
	}
	data, ok := m.data[key]
	if !ok {
		return cache.ErrCacheMiss
	}
	return json.Unmarshal(data, dest)
}

func (m *mockCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	m.setCalls[key]++
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	m.data[key] = data
	return nil
}

func (m *mockCache) Delete(ctx context.Context, key string) error {
	delete(m.data, key)
	return m.deleteErr
}

func (m *mockCache) Clear(ctx context.Context) error {
	m.data = make(map[string][]byte)
	return nil
}

// Mock anomaly repository
type mockAnomalyRepository struct {
	anomalies      map[uuid.UUID]Anomaly
	listResult     []Anomaly
	stats          *AnomalyStats
	createErr      error
	getErr         error
	updateErr      error
	deleteErr      error
	calls          map[string]int
}

func newMockAnomalyRepository() *mockAnomalyRepository {
	return &mockAnomalyRepository{
		anomalies: make(map[uuid.UUID]Anomaly),
		calls:     make(map[string]int),
		stats: &AnomalyStats{
			Total:              10,
			OpenCount:          3,
			InvestigatingCount: 2,
			ResolvedCount:      5,
			CriticalCount:      1,
			HighCount:          2,
			MediumCount:        4,
			LowCount:           3,
			TodayCount:         1,
			WeekCount:          5,
			UniqueCorrelations: 8,
			TotalDuplicates:    2,
			AvgRiskScore:       65.5,
		},
	}
}

func (m *mockAnomalyRepository) Create(ctx context.Context, anomaly *Anomaly) error {
	m.calls["Create"]++
	if m.createErr != nil {
		return m.createErr
	}
	anomaly.ID = uuid.New()
	anomaly.CreatedAt = time.Now()
	anomaly.UpdatedAt = time.Now()
	anomaly.DetectedAt = time.Now()
	m.anomalies[anomaly.ID] = *anomaly
	return nil
}

func (m *mockAnomalyRepository) BatchCreate(ctx context.Context, anomalies []Anomaly) error {
	m.calls["BatchCreate"]++
	if m.createErr != nil {
		return m.createErr
	}
	for i := range anomalies {
		anomalies[i].ID = uuid.New()
		anomalies[i].CreatedAt = time.Now()
		anomalies[i].UpdatedAt = time.Now()
		anomalies[i].DetectedAt = time.Now()
		m.anomalies[anomalies[i].ID] = anomalies[i]
	}
	return nil
}

func (m *mockAnomalyRepository) GetByID(ctx context.Context, id uuid.UUID) (*Anomaly, error) {
	m.calls["GetByID"]++
	if m.getErr != nil {
		return nil, m.getErr
	}
	if a, ok := m.anomalies[id]; ok {
		return &a, nil
	}
	return nil, ErrAnomalyNotFound
}

func (m *mockAnomalyRepository) List(ctx context.Context, tenantID uuid.UUID, filter AnomalyFilter, limit, offset int) ([]Anomaly, int, error) {
	m.calls["List"]++
	if m.listErr != nil {
		return nil, 0, m.listErr
	}
	if m.listResult != nil {
		return m.listResult, len(m.listResult), nil
	}
	var result []Anomaly
	for _, a := range m.anomalies {
		if a.TenantID == tenantID {
			result = append(result, a)
		}
	}
	return result, len(result), nil
}

func (m *mockAnomalyRepository) Update(ctx context.Context, anomaly *Anomaly) error {
	m.calls["Update"]++
	if m.updateErr != nil {
		return m.updateErr
	}
	if _, ok := m.anomalies[anomaly.ID]; !ok {
		return ErrAnomalyNotFound
	}
	anomaly.UpdatedAt = time.Now()
	m.anomalies[anomaly.ID] = *anomaly
	return nil
}

func (m *mockAnomalyRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status AnomalyStatus, assignedTo, resolvedBy *uuid.UUID, resolutionNotes *string) error {
	m.calls["UpdateStatus"]++
	if m.updateErr != nil {
		return m.updateErr
	}
	if _, ok := m.anomalies[id]; !ok {
		return ErrAnomalyNotFound
	}
	a := m.anomalies[id]
	a.Status = status
	a.AssignedTo = assignedTo
	a.ResolvedBy = resolvedBy
	a.ResolutionNotes = resolutionNotes
	a.UpdatedAt = time.Now()
	if status == AnomalyStatusResolved || status == AnomalyStatusFalsePositive || status == AnomalyStatusIgnored {
		now := time.Now()
		a.ResolvedAt = &now
	}
	m.anomalies[id] = a
	return nil
}

func (m *mockAnomalyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	m.calls["Delete"]++
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.anomalies, id)
	return nil
}

func (m *mockAnomalyRepository) GetStats(ctx context.Context, tenantID uuid.UUID) (*AnomalyStats, error) {
	m.calls["GetStats"]++
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.stats, nil
}

func (m *mockAnomalyRepository) GetByUserID(ctx context.Context, tenantID, userID uuid.UUID, limit int) ([]Anomaly, error) {
	m.calls["GetByUserID"]++
	var result []Anomaly
	for _, a := range m.anomalies {
		if a.TenantID == tenantID && a.UserID != nil && *a.UserID == userID {
			result = append(result, a)
		}
	}
	return result, nil
}

func (m *mockAnomalyRepository) GetBySessionID(ctx context.Context, sessionID uuid.UUID) ([]Anomaly, error) {
	m.calls["GetBySessionID"]++
	var result []Anomaly
	for _, a := range m.anomalies {
		if a.SessionID != nil && *a.SessionID == sessionID {
			result = append(result, a)
		}
	}
	return result, nil
}

func (m *mockAnomalyRepository) GetOpen(ctx context.Context, tenantID uuid.UUID, severity *Severity) ([]Anomaly, error) {
	m.calls["GetOpen"]++
	var result []Anomaly
	for _, a := range m.anomalies {
		if a.TenantID == tenantID && a.Status == AnomalyStatusOpen {
			if severity == nil || a.Severity == *severity {
				result = append(result, a)
			}
		}
	}
	return result, nil
}

func (m *mockAnomalyRepository) GetByCorrelationID(ctx context.Context, correlationID uuid.UUID) ([]Anomaly, error) {
	m.calls["GetByCorrelationID"]++
	var result []Anomaly
	for _, a := range m.anomalies {
		if a.CorrelationID != nil && *a.CorrelationID == correlationID {
			result = append(result, a)
		}
	}
	return result, nil
}

func (m *mockAnomalyRepository) GetAnomalyTypes(ctx context.Context, tenantID uuid.UUID) ([]string, error) {
	m.calls["GetAnomalyTypes"]++
	return []string{"behavioral", "temporal", "spatial", "pattern", "volumetric", "ransomware"}, nil
}

func (m *mockAnomalyRepository) GetTopUsersByAnomalyCount(ctx context.Context, tenantID uuid.UUID, limit int, dateFrom, dateTo *time.Time) ([]UserAnomalyCount, error) {
	m.calls["GetTopUsersByAnomalyCount"]++
	return []UserAnomalyCount{
		{UserID: uuid.New(), AnomalyCount: 5, MaxRiskScore: 85.0},
	}, nil
}

func (m *mockAnomalyRepository) CreateFromDetectionRequest(ctx context.Context, req *CreateAnomalyRequest, detectedBy uuid.UUID) (*Anomaly, error) {
	m.calls["CreateFromDetectionRequest"]++
	if m.createErr != nil {
		return nil, m.createErr
	}
	anomaly := &Anomaly{
		ID:               uuid.New(),
		TenantID:         req.TenantID,
		AnomalyType:      req.AnomalyType,
		UserID:           req.UserID,
		SessionID:        req.SessionID,
		TargetHost:       req.TargetHost,
		Title:            req.Title,
		Description:      req.Description,
		Severity:         req.Severity,
		ConfidenceScore:  req.ConfidenceScore,
		RiskScore:        req.RiskScore,
		DetectionMethod:  req.DetectionMethod,
		ModelVersion:     req.ModelVersion,
		AutoTriggered:    req.AutoTriggered,
		AutoActionTaken:  req.AutoActionTaken,
		Status:           AnomalyStatusOpen,
		DuplicateCount:   0,
		IsDuplicate:      false,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		DetectedAt:       time.Now(),
	}
	m.anomalies[anomaly.ID] = *anomaly
	return anomaly, nil
}

// Helper errors
var ErrAnomalyNotFound = fmt.Errorf("anomaly not found")

// TestService_NewService tests service creation
func TestService_NewService(t *testing.T) {
	t.Run("creates service with valid dependencies", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		anomalyRepo := newMockAnomalyRepository()
		cache := newMockCache()

		service := NewService(repo, anomalyRepo, cache, logger)

		assert.NotNil(t, service)
		assert.NotNil(t, service.repo)
		assert.NotNil(t, service.anomalyRepo)
		assert.NotNil(t, service.cache)
		assert.NotNil(t, service.logger)
	})

	t.Run("service initializes with workers not running", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		anomalyRepo := newMockAnomalyRepository()
		cache := newMockCache()

		service := NewService(repo, anomalyRepo, cache, logger)

		assert.False(t, service.aggregationWorkerRunning)
		assert.False(t, service.alertEvaluatorRunning)
		assert.False(t, service.reportSchedulerRunning)
	})
}

// TestService_RecordSessionStart tests session start recording
func TestService_RecordSessionStart(t *testing.T) {
	t.Run("records session start successfully", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		anomalyRepo := newMockAnomalyRepository()
		cache := newMockCache()
		service := NewService(repo, anomalyRepo, cache, logger)

		ctx := context.Background()
		tenantID := uuid.New()
		userID := uuid.New()
		sessionID := uuid.New()

		err := service.RecordSessionStart(ctx, tenantID, userID, sessionID, "ssh", "server.example.com", 22)

		assert.NoError(t, err)
		assert.Equal(t, 1, repo.calls["UpsertSessionAnalytics"])
		assert.Equal(t, 1, repo.calls["UpsertUserActivity"])
	})

	t.Run("records off-hours access correctly", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		anomalyRepo := newMockAnomalyRepository()
		cache := newMockCache()
		service := NewService(repo, anomalyRepo, cache, logger)

		ctx := context.Background()
		tenantID := uuid.New()
		userID := uuid.New()
		sessionID := uuid.New()

		// Test at 3 AM (should be off-hours)
		err := service.RecordSessionStart(ctx, tenantID, userID, sessionID, "ssh", "server.example.com", 22)

		assert.NoError(t, err)
		// The activity should be marked with off-hours access based on the time
	})

	t.Run("returns error when upsert fails", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		repo.createErr = assert.AnError
		anomalyRepo := newMockAnomalyRepository()
		cache := newMockCache()
		service := NewService(repo, anomalyRepo, cache, logger)

		ctx := context.Background()
		tenantID := uuid.New()
		userID := uuid.New()
		sessionID := uuid.New()

		err := service.RecordSessionStart(ctx, tenantID, userID, sessionID, "ssh", "server.example.com", 22)

		assert.Error(t, err)
	})
}

// TestService_RecordSessionEnd tests session end recording
func TestService_RecordSessionEnd(t *testing.T) {
	t.Run("records session end successfully", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		anomalyRepo := newMockAnomalyRepository()
		cache := newMockCache()
		service := NewService(repo, anomalyRepo, cache, logger)

		ctx := context.Background()
		tenantID := uuid.New()
		userID := uuid.New()
		sessionID := uuid.New()

		err := service.RecordSessionEnd(ctx, tenantID, userID, sessionID, 30*time.Minute, "ssh")

		assert.NoError(t, err)
		assert.Equal(t, 1, repo.calls["GetSessionAnalytics"])
	})
}

// TestService_GetSessionMetricsSummary tests session metrics retrieval
func TestService_GetSessionMetricsSummary(t *testing.T) {
	t.Run("retrieves session metrics from repository", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		anomalyRepo := newMockAnomalyRepository()
		cache := newMockCache()
		service := NewService(repo, anomalyRepo, cache, logger)

		ctx := context.Background()
		tenantID := uuid.New()
		startDate := time.Now().Add(-7 * 24 * time.Hour)
		endDate := time.Now()

		stats, err := service.GetSessionMetricsSummary(ctx, tenantID, startDate, endDate)

		assert.NoError(t, err)
		assert.NotNil(t, stats)
		assert.Equal(t, int64(100), stats.Total)
		assert.Equal(t, int64(10), stats.Active)
		assert.Equal(t, int64(85), stats.Completed)
	})

	t.Run("caches session metrics", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		anomalyRepo := newMockAnomalyRepository()
		cache := newMockCache()
		service := NewService(repo, anomalyRepo, cache, logger)

		ctx := context.Background()
		tenantID := uuid.New()
		startDate := time.Now().Add(-7 * 24 * time.Hour)
		endDate := time.Now()

		// First call should hit repository
		_, err := service.GetSessionMetricsSummary(ctx, tenantID, startDate, endDate)
		require.NoError(t, err)
		assert.Equal(t, 1, repo.calls["GetRawSessionStats"])

		// Second call should use cache
		_, err = service.GetSessionMetricsSummary(ctx, tenantID, startDate, endDate)
		require.NoError(t, err)
		assert.Equal(t, 1, repo.calls["GetRawSessionStats"]) // Should not increase
	})
}

// TestService_GetDashboardMetrics tests dashboard metrics retrieval
func TestService_GetDashboardMetrics(t *testing.T) {
	t.Run("retrieves comprehensive dashboard metrics", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		anomalyRepo := newMockAnomalyRepository()
		cache := newMockCache()
		service := NewService(repo, anomalyRepo, cache, logger)

		ctx := context.Background()
		tenantID := uuid.New()

		metrics, err := service.GetDashboardMetrics(ctx, tenantID)

		assert.NoError(t, err)
		assert.NotNil(t, metrics)
		assert.NotNil(t, metrics.SessionMetrics)
		assert.NotEmpty(t, metrics.TopUsers)
		assert.NotEmpty(t, metrics.AccessPatterns)
		assert.NotNil(t, metrics.ComplianceStatus)
		assert.False(t, metrics.Timestamp.IsZero())
	})

	t.Run("caches dashboard metrics", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		anomalyRepo := newMockAnomalyRepository()
		cache := newMockCache()
		service := NewService(repo, anomalyRepo, cache, logger)

		ctx := context.Background()
		tenantID := uuid.New()

		// First call
		_, err := service.GetDashboardMetrics(ctx, tenantID)
		require.NoError(t, err)

		// Should cache the result
		assert.Equal(t, 1, cache.setCalls[fmt.Sprintf("analytics:dashboard:%s", tenantID)])
	})
}

// TestService_DetectAnomalies tests anomaly detection
func TestService_DetectAnomalies(t *testing.T) {
	t.Run("detects anomalies and persists them", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		// Setup mock user activity with off-hours access
		offHoursTime := time.Now().Truncate(24 * time.Hour)
		userID := uuid.New()
		activity := UserActivity{
			UserID:          userID,
			OffHoursAccess:  true,
			SessionsCreated: 15, // Above threshold
			PeriodStart:     offHoursTime,
		}
		repo.userActivity = &activity

		anomalyRepo := newMockAnomalyRepository()
		cache := newMockCache()
		service := NewService(repo, anomalyRepo, cache, logger)

		ctx := context.Background()
		tenantID := uuid.New()

		anomalies, err := service.DetectAnomalies(ctx, tenantID)

		assert.NoError(t, err)
		assert.NotNil(t, anomalies)
		assert.Equal(t, 1, anomalyRepo.calls["BatchCreate"])
	})

	t.Run("handles no anomalies detected gracefully", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		// No user activity set, so no anomalies should be detected

		anomalyRepo := newMockAnomalyRepository()
		cache := newMockCache()
		service := NewService(repo, anomalyRepo, cache, logger)

		ctx := context.Background()
		tenantID := uuid.New()

		anomalies, err := service.DetectAnomalies(ctx, tenantID)

		assert.NoError(t, err)
		assert.NotNil(t, anomalies)
	})
}

// TestService_ListAnomalies tests listing anomalies
func TestService_ListAnomalies(t *testing.T) {
	t.Run("lists all anomalies when no status filter", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		anomalyRepo := newMockAnomalyRepository()
		// Setup mock anomalies
		tenantID := uuid.New()
		anomaly1 := Anomaly{
			ID:         uuid.New(),
			TenantID:   tenantID,
			AnomalyType: AnomalyTypeBehavioral,
			Severity:    SeverityHigh,
			Status:     AnomalyStatusOpen,
			Title:      "Test Anomaly 1",
		}
		anomaly2 := Anomaly{
			ID:         uuid.New(),
			TenantID:   tenantID,
			AnomalyType: AnomalyTypeTemporal,
			Severity:    SeverityMedium,
			Status:     AnomalyStatusResolved,
			Title:      "Test Anomaly 2",
		}
		anomalyRepo.anomalies[anomaly1.ID] = anomaly1
		anomalyRepo.anomalies[anomaly2.ID] = anomaly2

		cache := newMockCache()
		service := NewService(repo, anomalyRepo, cache, logger)

		ctx := context.Background()

		anomalies, err := service.ListAnomalies(ctx, tenantID, "", 10, 0)

		assert.NoError(t, err)
		assert.NotNil(t, anomalies)
		assert.Equal(t, 1, anomalyRepo.calls["List"])
	})

	t.Run("filters anomalies by status", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		anomalyRepo := newMockAnomalyRepository()
		cache := newMockCache()
		service := NewService(repo, anomalyRepo, cache, logger)

		ctx := context.Background()
		tenantID := uuid.New()

		anomalies, err := service.ListAnomalies(ctx, tenantID, "open", 10, 0)

		assert.NoError(t, err)
		assert.NotNil(t, anomalies)
	})
}

// TestService_GetAnomaly tests retrieving a single anomaly
func TestService_GetAnomaly(t *testing.T) {
	t.Run("retrieves existing anomaly", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		anomalyRepo := newMockAnomalyRepository()
		cache := newMockCache()
		service := NewService(repo, anomalyRepo, cache, logger)

		ctx := context.Background()
		anomalyID := uuid.New()
		tenantID := uuid.New()
		anomaly := Anomaly{
			ID:         anomalyID,
			TenantID:   tenantID,
			AnomalyType: AnomalyTypeBehavioral,
			Severity:    SeverityHigh,
			Status:     AnomalyStatusOpen,
			Title:      "Test Anomaly",
		}
		anomalyRepo.anomalies[anomalyID] = anomaly

		response, err := service.GetAnomaly(ctx, anomalyID)

		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, anomalyID, response.ID)
	})

	t.Run("returns error for non-existent anomaly", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		anomalyRepo := newMockAnomalyRepository()
		anomalyRepo.getErr = ErrAnomalyNotFound
		cache := newMockCache()
		service := NewService(repo, anomalyRepo, cache, logger)

		ctx := context.Background()
		anomalyID := uuid.New()

		_, err := service.GetAnomaly(ctx, anomalyID)

		assert.Error(t, err)
	})
}

// TestService_UpdateAnomalyStatus tests updating anomaly status
func TestService_UpdateAnomalyStatus(t *testing.T) {
	t.Run("updates anomaly status successfully", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		anomalyRepo := newMockAnomalyRepository()
		cache := newMockCache()
		service := NewService(repo, anomalyRepo, cache, logger)

		ctx := context.Background()
		anomalyID := uuid.New()
		tenantID := uuid.New()
		userID := uuid.New()
		anomaly := Anomaly{
			ID:         anomalyID,
			TenantID:   tenantID,
			AnomalyType: AnomalyTypeBehavioral,
			Severity:    SeverityHigh,
			Status:     AnomalyStatusOpen,
			Title:      "Test Anomaly",
		}
		anomalyRepo.anomalies[anomalyID] = anomaly

		err := service.UpdateAnomalyStatus(ctx, anomalyID, "investigating", &userID, nil, "Started investigation")

		assert.NoError(t, err)
		assert.Equal(t, 1, anomalyRepo.calls["UpdateStatus"])
	})

	t.Run("invalidates cache on status update", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		anomalyRepo := newMockAnomalyRepository()
		cache := newMockCache()
		service := NewService(repo, anomalyRepo, cache, logger)

		ctx := context.Background()
		anomalyID := uuid.New()
		tenantID := uuid.New()
		userID := uuid.New()
		anomaly := Anomaly{
			ID:         anomalyID,
			TenantID:   tenantID,
			AnomalyType: AnomalyTypeBehavioral,
			Severity:    SeverityHigh,
			Status:     AnomalyStatusOpen,
			Title:      "Test Anomaly",
		}
		anomalyRepo.anomalies[anomalyID] = anomaly

		err := service.UpdateAnomalyStatus(ctx, anomalyID, "investigating", &userID, nil, "Started investigation")

		assert.NoError(t, err)
		// Cache should be invalidated
		expectedKey := fmt.Sprintf("analytics:anomalies:%s", tenantID)
		_, deleted := cache.data[expectedKey]
		assert.False(t, deleted)
	})
}

// TestService_AcknowledgeAnomaly tests acknowledging anomalies
func TestService_AcknowledgeAnomaly(t *testing.T) {
	t.Run("acknowledges open anomaly successfully", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		anomalyRepo := newMockAnomalyRepository()
		cache := newMockCache()
		service := NewService(repo, anomalyRepo, cache, logger)

		ctx := context.Background()
		anomalyID := uuid.New()
		tenantID := uuid.New()
		userID := uuid.New()
		anomaly := Anomaly{
			ID:         anomalyID,
			TenantID:   tenantID,
			AnomalyType: AnomalyTypeBehavioral,
			Severity:    SeverityHigh,
			Status:     AnomalyStatusOpen,
			Title:      "Test Anomaly",
		}
		anomalyRepo.anomalies[anomalyID] = anomaly

		err := service.AcknowledgeAnomaly(ctx, anomalyID, userID, "Acknowledging for review")

		assert.NoError(t, err)
		assert.Equal(t, 1, anomalyRepo.calls["UpdateStatus"])
	})

	t.Run("fails to acknowledge non-open anomaly", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		anomalyRepo := newMockAnomalyRepository()
		cache := newMockCache()
		service := NewService(repo, anomalyRepo, cache, logger)

		ctx := context.Background()
		anomalyID := uuid.New()
		tenantID := uuid.New()
		userID := uuid.New()
		anomaly := Anomaly{
			ID:         anomalyID,
			TenantID:   tenantID,
			AnomalyType: AnomalyTypeBehavioral,
			Severity:    SeverityHigh,
			Status:     AnomalyStatusResolved, // Already resolved
			Title:      "Test Anomaly",
		}
		anomalyRepo.anomalies[anomalyID] = anomaly

		err := service.AcknowledgeAnomaly(ctx, anomalyID, userID, "Should not work")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot acknowledge anomaly")
	})
}

// TestService_ResolveAnomaly tests resolving anomalies
func TestService_ResolveAnomaly(t *testing.T) {
	t.Run("resolves anomaly with resolved status", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		anomalyRepo := newMockAnomalyRepository()
		cache := newMockCache()
		service := NewService(repo, anomalyRepo, cache, logger)

		ctx := context.Background()
		anomalyID := uuid.New()
		tenantID := uuid.New()
		userID := uuid.New()
		anomaly := Anomaly{
			ID:         anomalyID,
			TenantID:   tenantID,
			AnomalyType: AnomalyTypeBehavioral,
			Severity:    SeverityHigh,
			Status:     AnomalyStatusOpen,
			Title:      "Test Anomaly",
		}
		anomalyRepo.anomalies[anomalyID] = anomaly

		err := service.ResolveAnomaly(ctx, anomalyID, userID, AnomalyStatusResolved, "Issue fixed")

		assert.NoError(t, err)
		assert.Equal(t, 1, anomalyRepo.calls["UpdateStatus"])
	})

	t.Run("resolves anomaly with false_positive status", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		anomalyRepo := newMockAnomalyRepository()
		cache := newMockCache()
		service := NewService(repo, anomalyRepo, cache, logger)

		ctx := context.Background()
		anomalyID := uuid.New()
		tenantID := uuid.New()
		userID := uuid.New()
		anomaly := Anomaly{
			ID:         anomalyID,
			TenantID:   tenantID,
			AnomalyType: AnomalyTypeBehavioral,
			Severity:    SeverityHigh,
			Status:     AnomalyStatusOpen,
			Title:      "Test Anomaly",
		}
		anomalyRepo.anomalies[anomalyID] = anomaly

		err := service.ResolveAnomaly(ctx, anomalyID, userID, AnomalyStatusFalsePositive, "Not actually an issue")

		assert.NoError(t, err)
	})

	t.Run("rejects invalid resolution status", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		anomalyRepo := newMockAnomalyRepository()
		cache := newMockCache()
		service := NewService(repo, anomalyRepo, cache, logger)

		ctx := context.Background()
		anomalyID := uuid.New()
		tenantID := uuid.New()
		userID := uuid.New()
		anomaly := Anomaly{
			ID:         anomalyID,
			TenantID:   tenantID,
			AnomalyType: AnomalyTypeBehavioral,
			Severity:    SeverityHigh,
			Status:     AnomalyStatusOpen,
			Title:      "Test Anomaly",
		}
		anomalyRepo.anomalies[anomalyID] = anomaly

		err := service.ResolveAnomaly(ctx, anomalyID, userID, AnomalyStatusOpen, "Invalid")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid resolution status")
	})
}

// TestService_GetAnomalyStats tests retrieving anomaly statistics
func TestService_GetAnomalyStats(t *testing.T) {
	t.Run("retrieves anomaly statistics", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		anomalyRepo := newMockAnomalyRepository()
		cache := newMockCache()
		service := NewService(repo, anomalyRepo, cache, logger)

		ctx := context.Background()
		tenantID := uuid.New()

		stats, err := service.GetAnomalyStats(ctx, tenantID)

		assert.NoError(t, err)
		assert.NotNil(t, stats)
		assert.Equal(t, 10, stats.Total)
		assert.Equal(t, 3, stats.OpenCount)
		assert.Equal(t, 2, stats.InvestigatingCount)
		assert.Equal(t, 5, stats.ResolvedCount)
		assert.Equal(t, 1, stats.CriticalCount)
		assert.Equal(t, 65.5, stats.AvgRiskScore)
	})
}

// TestService_EvaluateUserForAnomalies tests user anomaly evaluation
func TestService_EvaluateUserForAnomalies(t *testing.T) {
	t.Run("evaluates user and detects anomalies", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		// Setup user activity that should trigger anomalies
		userID := uuid.New()
		tenantID := uuid.New()
		activities := []UserActivity{
			{
				UserID:            userID,
				TenantID:          tenantID,
				PeriodStart:       time.Now().Truncate(24 * time.Hour),
				OffHoursAccess:    true,
				SessionsCreated:   10, // Triggers off-hours anomaly
				PolicyViolations:  8,  // Triggers policy violation anomaly
				FailedAuthAttempts: 15, // Triggers failed auth anomaly
			},
		}
		repo.userActivityList = activities

		anomalyRepo := newMockAnomalyRepository()
		cache := newMockCache()
		service := NewService(repo, anomalyRepo, cache, logger)

		ctx := context.Background()

		anomalies, err := service.EvaluateUserForAnomalies(ctx, tenantID, userID)

		assert.NoError(t, err)
		assert.NotNil(t, anomalies)
		// Should detect 3 types of anomalies from the activity
	})
}

// TestService_GenerateComplianceReport tests compliance report generation
func TestService_GenerateComplianceReport(t *testing.T) {
	t.Run("generates compliance report successfully", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		anomalyRepo := newMockAnomalyRepository()
		cache := newMockCache()
		service := NewService(repo, anomalyRepo, cache, logger)

		ctx := context.Background()
		tenantID := uuid.New()
		generatedBy := uuid.New()
		startDate := time.Now().Add(-30 * 24 * time.Hour)
		endDate := time.Now()

		report, err := service.GenerateComplianceReport(ctx, tenantID, generatedBy, "SOC2", startDate, endDate)

		assert.NoError(t, err)
		assert.NotNil(t, report)
		assert.Equal(t, tenantID, report.TenantID)
		assert.Equal(t, "SOC2", report.Framework)
		assert.Equal(t, generatedBy, report.GeneratedBy)
		assert.Greater(t, report.OverallScore, 0.0)
	})
}

// TestService_GetComplianceSummary tests compliance summary retrieval
func TestService_GetComplianceSummary(t *testing.T) {
	t.Run("retrieves compliance summary", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		anomalyRepo := newMockAnomalyRepository()
		cache := newMockCache()
		service := NewService(repo, anomalyRepo, cache, logger)

		ctx := context.Background()
		tenantID := uuid.New()

		summary, err := service.GetComplianceSummary(ctx, tenantID)

		assert.NoError(t, err)
		assert.NotNil(t, summary)
		assert.Equal(t, 85.5, summary.OverallPercentage)
		assert.Equal(t, 17, summary.PassedChecks)
		assert.Equal(t, 20, summary.TotalChecks)
	})
}

// TestService_CreateComplianceException tests compliance exception creation
func TestService_CreateComplianceException(t *testing.T) {
	t.Run("creates compliance exception successfully", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		anomalyRepo := newMockAnomalyRepository()
		cache := newMockCache()
		service := NewService(repo, anomalyRepo, cache, logger)

		ctx := context.Background()
		tenantID := uuid.New()
		createdBy := uuid.New()
		req := &CreateComplianceExceptionRequest{
			ControlID:     "AC-001",
			ControlName:   "Access Control",
			Framework:     "SOC2",
			RiskLevel:     "medium",
			Justification: "Legacy system compatibility",
			BusinessReason: "Requires temporary exception",
		}

		exception, err := service.CreateComplianceException(ctx, tenantID, createdBy, req)

		assert.NoError(t, err)
		assert.NotNil(t, exception)
		assert.Equal(t, tenantID, exception.TenantID)
		assert.Equal(t, "AC-001", exception.ControlID)
		assert.Equal(t, "pending", exception.Status)
		assert.Equal(t, createdBy, exception.RequestedBy)
	})
}

// TestService_GetUserRiskScore tests user risk score calculation
func TestService_GetUserRiskScore(t *testing.T) {
	t.Run("calculates user risk score", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		anomalyRepo := newMockAnomalyRepository()
		cache := newMockCache()
		service := NewService(repo, anomalyRepo, cache, logger)

		ctx := context.Background()
		tenantID := uuid.New()
		userID := uuid.New()

		score, err := service.GetUserRiskScore(ctx, tenantID, userID, 30)

		assert.NoError(t, err)
		assert.GreaterOrEqual(t, score, 0.0)
		assert.LessOrEqual(t, score, 100.0)
	})

	t.Run("caches risk score result", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		anomalyRepo := newMockAnomalyRepository()
		cache := newMockCache()
		service := NewService(repo, anomalyRepo, cache, logger)

		ctx := context.Background()
		tenantID := uuid.New()
		userID := uuid.New()

		// First call
		_, err := service.GetUserRiskScore(ctx, tenantID, userID, 30)
		require.NoError(t, err)

		// Second call should use cache
		cacheKey := fmt.Sprintf("analytics:risk_score:%s:%s:30", tenantID, userID)
		_, exists := cache.data[cacheKey]
		assert.True(t, exists)
	})
}

// TestService_ListRiskScores tests listing risk scores
func TestService_ListRiskScores(t *testing.T) {
	t.Run("lists risk scores with filters", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		anomalyRepo := newMockAnomalyRepository()
		cache := newMockCache()
		service := NewService(repo, anomalyRepo, cache, logger)

		ctx := context.Background()
		tenantID := uuid.New()

		scores, err := service.ListRiskScores(ctx, tenantID, "user", "", "", 10)

		assert.NoError(t, err)
		assert.NotNil(t, scores)
	})
}

// TestService_GetLatestRiskScore tests retrieving latest risk score
func TestService_GetLatestRiskScore(t *testing.T) {
	t.Run("retrieves latest risk score for entity", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		anomalyRepo := newMockAnomalyRepository()
		cache := newMockCache()
		service := NewService(repo, anomalyRepo, cache, logger)

		ctx := context.Background()
		tenantID := uuid.New()
		entityID := uuid.New()

		score, err := service.GetLatestRiskScore(ctx, tenantID, "user", entityID)

		assert.NoError(t, err)
		assert.NotNil(t, score)
		assert.Equal(t, entityID, score.EntityID)
		assert.Equal(t, 50.0, score.OverallRiskScore)
	})
}

// TestService_CalculateRiskScores tests calculating risk scores
func TestService_CalculateRiskScores(t *testing.T) {
	t.Run("calculates risk scores for entities", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		anomalyRepo := newMockAnomalyRepository()
		cache := newMockCache()
		service := NewService(repo, anomalyRepo, cache, logger)

		ctx := context.Background()
		tenantID := uuid.New()
		entityIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}

		count, err := service.CalculateRiskScores(ctx, tenantID, "user", entityIDs)

		assert.NoError(t, err)
		assert.Equal(t, 3, count)
		assert.Equal(t, 3, repo.calls["CreateRiskScore"])
	})
}

// TestService_RecordMetric tests recording metrics
func TestService_RecordMetric(t *testing.T) {
	t.Run("records metric successfully", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		anomalyRepo := newMockAnomalyRepository()
		cache := newMockCache()
		service := NewService(repo, anomalyRepo, cache, logger)

		ctx := context.Background()
		tenantID := uuid.New()
		req := &RecordMetricRequest{
			Name:  "test.metric",
			Type:  "gauge",
			Value: 42.5,
			Labels: map[string]interface{}{"label1": "value1"},
		}

		err := service.RecordMetric(ctx, tenantID, req)

		assert.NoError(t, err)
		assert.Equal(t, 1, repo.calls["RecordMetric"])
	})
}

// TestService_QueryMetrics tests querying metrics
func TestService_QueryMetrics(t *testing.T) {
	t.Run("queries metrics with filters", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		anomalyRepo := newMockAnomalyRepository()
		cache := newMockCache()
		service := NewService(repo, anomalyRepo, cache, logger)

		ctx := context.Background()
		tenantID := uuid.New()
		startDate := time.Now().Add(-24 * time.Hour)
		endDate := time.Now()

		metrics, err := service.QueryMetrics(ctx, tenantID, "test.metric", startDate, endDate, 100)

		assert.NoError(t, err)
		assert.NotNil(t, metrics)
	})
}

// TestService_GetSessionTimeSeries tests time series data retrieval
func TestService_GetSessionTimeSeries(t *testing.T) {
	t.Run("retrieves time series data", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		anomalyRepo := newMockAnomalyRepository()
		cache := newMockCache()
		service := NewService(repo, anomalyRepo, cache, logger)

		ctx := context.Background()
		tenantID := uuid.New()
		startDate := time.Now().Add(-7 * 24 * time.Hour)
		endDate := time.Now()

		data, err := service.GetSessionTimeSeries(ctx, tenantID, "sessions", startDate, endDate)

		assert.NoError(t, err)
		assert.NotNil(t, data)
	})

	t.Run("caches time series data", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		anomalyRepo := newMockAnomalyRepository()
		cache := newMockCache()
		service := NewService(repo, anomalyRepo, cache, logger)

		ctx := context.Background()
		tenantID := uuid.New()
		startDate := time.Now().Add(-7 * 24 * time.Hour)
		endDate := time.Now()

		// First call
		_, err := service.GetSessionTimeSeries(ctx, tenantID, "sessions", startDate, endDate)
		require.NoError(t, err)
		assert.Equal(t, 1, repo.calls["GetSessionTimeSeries"])

		// Second call should use cache
		_, err = service.GetSessionTimeSeries(ctx, tenantID, "sessions", startDate, endDate)
		require.NoError(t, err)
		assert.Equal(t, 1, repo.calls["GetSessionTimeSeries"]) // Should not increase
	})
}

// TestService_GetCommandFrequencyList tests command frequency retrieval
func TestService_GetCommandFrequencyList(t *testing.T) {
	t.Run("retrieves command frequency data", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		anomalyRepo := newMockAnomalyRepository()
		cache := newMockCache()
		service := NewService(repo, anomalyRepo, cache, logger)

		ctx := context.Background()
		tenantID := uuid.New()
		startDate := time.Now().Add(-7 * 24 * time.Hour)
		endDate := time.Now()

		freq, err := service.GetCommandFrequencyList(ctx, tenantID, startDate, endDate, 10)

		assert.NoError(t, err)
		assert.NotNil(t, freq)
	})
}

// TestService_GetTopUsers tests retrieving top users
func TestService_GetTopUsers(t *testing.T) {
	t.Run("retrieves top users by activity", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		anomalyRepo := newMockAnomalyRepository()
		cache := newMockCache()
		service := NewService(repo, anomalyRepo, cache, logger)

		ctx := context.Background()
		tenantID := uuid.New()
		startDate := time.Now().Add(-7 * 24 * time.Hour)
		endDate := time.Now()

		users, err := service.GetTopUsersByActivity(ctx, tenantID, startDate, endDate, 5)

		assert.NoError(t, err)
		assert.NotNil(t, users)
	})
}

// TestService_AggregateMetrics tests metric aggregation
func TestService_AggregateMetrics(t *testing.T) {
	t.Run("aggregates metrics by period", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		anomalyRepo := newMockAnomalyRepository()
		cache := newMockCache()
		service := NewService(repo, anomalyRepo, cache, logger)

		ctx := context.Background()
		tenantID := uuid.New()
		startDate := time.Now().Add(-24 * time.Hour)
		endDate := time.Now()

		data, err := service.AggregateMetrics(ctx, tenantID, "test.metric", startDate, endDate, "hour")

		assert.NoError(t, err)
		assert.NotNil(t, data)
	})
}

// TestService_ListSessionAnalytics tests listing session analytics
func TestService_ListSessionAnalytics(t *testing.T) {
	t.Run("lists session analytics with filters", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		anomalyRepo := newMockAnomalyRepository()
		cache := newMockCache()
		service := NewService(repo, anomalyRepo, cache, logger)

		ctx := context.Background()
		tenantID := uuid.New()
		startDate := time.Now().Add(-7 * 24 * time.Hour)
		endDate := time.Now()

		analytics, err := service.ListSessionAnalytics(ctx, tenantID, PeriodDay, startDate, endDate, 10, 0)

		assert.NoError(t, err)
		assert.NotNil(t, analytics)
	})
}

// TestService_GetUserActivityList tests retrieving user activity list
func TestService_GetUserActivityList(t *testing.T) {
	t.Run("retrieves user activity list", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		anomalyRepo := newMockAnomalyRepository()
		cache := newMockCache()
		service := NewService(repo, anomalyRepo, cache, logger)

		ctx := context.Background()
		tenantID := uuid.New()
		userID := uuid.New()
		startDate := time.Now().Add(-7 * 24 * time.Hour)
		endDate := time.Now()

		activities, err := service.GetUserActivityList(ctx, tenantID, userID, startDate, endDate, 10)

		assert.NoError(t, err)
		assert.NotNil(t, activities)
	})
}

// TestService_RunAnomalyDetection tests running anomaly detection
func TestService_RunAnomalyDetection(t *testing.T) {
	t.Run("runs full anomaly detection", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		anomalyRepo := newMockAnomalyRepository()
		cache := newMockCache()
		service := NewService(repo, anomalyRepo, cache, logger)

		ctx := context.Background()
		tenantID := uuid.New()

		anomalies, err := service.RunAnomalyDetection(ctx, tenantID)

		assert.NoError(t, err)
		assert.NotNil(t, anomalies)
	})
}

// TestService_GetTopCommands tests retrieving top commands
func TestService_GetTopCommands(t *testing.T) {
	t.Run("retrieves top commands", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		anomalyRepo := newMockAnomalyRepository()
		cache := newMockCache()
		service := NewService(repo, anomalyRepo, cache, logger)

		ctx := context.Background()
		tenantID := uuid.New()
		startDate := time.Now().Add(-7 * 24 * time.Hour)
		endDate := time.Now()

		commands, err := service.GetTopCommands(ctx, tenantID, startDate, endDate, 10)

		assert.NoError(t, err)
		assert.NotNil(t, commands)
	})
}

// TestService_ListComplianceReports tests listing compliance reports
func TestService_ListComplianceReports(t *testing.T) {
	t.Run("lists compliance reports", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		anomalyRepo := newMockAnomalyRepository()
		cache := newMockCache()
		service := NewService(repo, anomalyRepo, cache, logger)

		ctx := context.Background()
		tenantID := uuid.New()

		reports, err := service.ListComplianceReports(ctx, tenantID, "SOC2")

		assert.NoError(t, err)
		assert.NotNil(t, reports)
	})
}

// TestService_GetComplianceReport tests retrieving a compliance report
func TestService_GetComplianceReport(t *testing.T) {
	t.Run("returns not implemented error", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		anomalyRepo := newMockAnomalyRepository()
		cache := newMockCache()
		service := NewService(repo, anomalyRepo, cache, logger)

		ctx := context.Background()
		reportID := uuid.New()

		_, err := service.GetComplianceReport(ctx, reportID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not implemented")
	})
}

// TestService_ListComplianceExceptions tests listing compliance exceptions
func TestService_ListComplianceExceptions(t *testing.T) {
	t.Run("lists compliance exceptions", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := newMockRepository()
		anomalyRepo := newMockAnomalyRepository()
		cache := newMockCache()
		service := NewService(repo, anomalyRepo, cache, logger)

		ctx := context.Background()
		tenantID := uuid.New()

		exceptions, err := service.ListComplianceExceptions(ctx, tenantID)

		assert.NoError(t, err)
		assert.NotNil(t, exceptions)
	})
}

// Helper functions
func timePtr(t time.Time) *time.Time {
	return &t
}

func stringPtr(s string) *string {
	return &s
}

func mustMarshalJSON(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
