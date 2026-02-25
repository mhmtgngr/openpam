package models

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

// BehaviorProfile represents a user's behavioral baseline for anomaly detection
type BehaviorProfile struct {
	ID                uuid.UUID              `json:"id" db:"id"`
	TenantID          uuid.UUID              `json:"tenant_id" db:"tenant_id"`
	UserID            uuid.UUID              `json:"user_id" db:"user_id"`
	ProfileType       string                 `json:"profile_type" db:"profile_type"` // session, credential, access
	Status            string                 `json:"status" db:"status"`             // active, stale, disabled

	// Time patterns
	PeakHours         []int                  `json:"peak_hours" db:"peak_hours"`           // Hours of day (0-23)
	PeakDays          []int                  `json:"peak_days" db:"peak_days"`             // Days of week (0-6)
	AverageSessionDuration int               `json:"average_session_duration" db:"average_session_duration"` // minutes
	SessionDurationStdDev  float64           `json:"session_duration_stddev" db:"session_duration_stddev"`

	// Access patterns
	FrequentResources  []ResourceFrequency   `json:"frequent_resources" db:"frequent_resources"`
	FrequentCommands   []CommandFrequency    `json:"frequent_commands" db:"frequent_commands"`
	TypicalResourceCount  MinMaxAvg          `json:"typical_resource_count" db:"typical_resource_count"`

	// Geographic patterns
	TypicalIPs        []string               `json:"typical_ips" db:"typical_ips"`
	TypicalLocations  []string               `json:"typical_locations" db:"typical_locations"`
	TypicalUserAgents []string               `json:"typical_user_agents" db:"typical_user_agents"`

	// Statistical data points
	SampleCount       int                    `json:"sample_count" db:"sample_count"`
	LastUpdated       time.Time              `json:"last_updated" db:"last_updated"`
	StaleAt           time.Time              `json:"stale_at" db:"stale_at"`

	// Model metadata
	ModelVersion      string                 `json:"model_version" db:"model_version"`
	Confidence        float64                `json:"confidence" db:"confidence"` // 0-1
	Metadata          map[string]string      `json:"metadata" db:"metadata"`
}

// ResourceFrequency represents how often a resource is accessed
type ResourceFrequency struct {
	ResourceID   string  `json:"resource_id"`
	ResourceType string  `json:"resource_type"`
	Count        int     `json:"count"`
	Frequency    float64 `json:"frequency"` // 0-1
}

// CommandFrequency represents how often a command is executed
type CommandFrequency struct {
	Command   string  `json:"command"`
	Count     int     `json:"count"`
	Frequency float64 `json:"frequency"`
}

// MinMaxAvg represents min, max, and average statistics
type MinMaxAvg struct {
	Min   int     `json:"min"`
	Max   int     `json:"max"`
	Avg   float64 `json:"avg"`
	StdDev float64 `json:"stddev"`
}

// BehaviorSample represents a single behavior data point
type BehaviorSample struct {
	ID           uuid.UUID              `json:"id" db:"id"`
	TenantID     uuid.UUID              `json:"tenant_id" db:"tenant_id"`
	UserID       uuid.UUID              `json:"user_id" db:"user_id"`
	SessionID    uuid.UUID              `json:"session_id" db:"session_id"`
	Timestamp    time.Time              `json:"timestamp" db:"timestamp"`

	// Session data
	Duration     int                    `json:"duration" db:"duration"` // minutes
	ResourceCount int                   `json:"resource_count" db:"resource_count"`
	CommandCount  int                   `json:"command_count" db:"command_count"`

	// Context data
	IPAddress    string                 `json:"ip_address" db:"ip_address"`
	Location     string                 `json:"location" db:"location"`
	UserAgent    string                 `json:"user_agent" db:"user_agent"`
	HourOfDay    int                    `json:"hour_of_day" db:"hour_of_day"`
	DayOfWeek    int                    `json:"day_of_week" db:"day_of_week"`

	// Access data
	Resources    []string               `json:"resources" db:"resources"`
	Commands     []string               `json:"commands" db:"commands"`

	// Flags
	AnomalyScore float64                `json:"anomaly_score" db:"anomaly_score"`
	IsAnomaly    bool                   `json:"is_anomaly" db:"is_anomaly"`
}

// BehaviorProfileService manages user behavior profiles
type BehaviorProfileService struct {
	db     *sqlx.DB
	logger *zerolog.Logger

	// Configuration
	minSamples        int
	staleThreshold   time.Duration
	updateThreshold  int // Minimum new samples before updating profile
}

// NewBehaviorProfileService creates a new behavior profile service
func NewBehaviorProfileService(db *sqlx.DB, logger *zerolog.Logger) *BehaviorProfileService {
	return &BehaviorProfileService{
		db:              db,
		logger:          logger,
		minSamples:      10,
		staleThreshold:  30 * 24 * time.Hour, // 30 days
		updateThreshold: 5,
	}
}

// GetProfile retrieves a user's behavior profile
func (s *BehaviorProfileService) GetProfile(ctx context.Context, tenantID, userID uuid.UUID, profileType string) (*BehaviorProfile, error) {
	query := `
		SELECT id, tenant_id, user_id, profile_type, status,
			peak_hours, peak_days, average_session_duration, session_duration_stddev,
			frequent_resources, frequent_commands, typical_resource_count,
			typical_ips, typical_locations, typical_user_agents,
			sample_count, last_updated, stale_at,
			model_version, confidence, metadata
		FROM behavior_profiles
		WHERE tenant_id = $1 AND user_id = $2 AND profile_type = $3
			AND deleted_at IS NULL
		ORDER BY last_updated DESC
		LIMIT 1
	`

	var profile BehaviorProfile
	err := s.db.GetContext(ctx, &profile, query, tenantID, userID, profileType)
	if err != nil {
		return nil, fmt.Errorf("get profile: %w", err)
	}

	// Check if profile is stale
	if time.Since(profile.LastUpdated) > s.staleThreshold {
		profile.Status = "stale"
	}

	return &profile, nil
}

// CreateProfile creates a new behavior profile
func (s *BehaviorProfileService) CreateProfile(ctx context.Context, profile *BehaviorProfile) error {
	profile.ID = uuid.New()
	profile.LastUpdated = time.Now()
	profile.StaleAt = time.Now().Add(s.staleThreshold)
	profile.SampleCount = 0
	profile.ModelVersion = "1.0"

	query := `
		INSERT INTO behavior_profiles (
			id, tenant_id, user_id, profile_type, status,
			peak_hours, peak_days, average_session_duration, session_duration_stddev,
			frequent_resources, frequent_commands, typical_resource_count,
			typical_ips, typical_locations, typical_user_agents,
			sample_count, last_updated, stale_at,
			model_version, confidence, metadata
		) VALUES (
			:id, :tenant_id, :user_id, :profile_type, :status,
			:peak_hours, :peak_days, :average_session_duration, :session_duration_stddev,
			:frequent_resources, :frequent_commands, :typical_resource_count,
			:typical_ips, :typical_locations, :typical_user_agents,
			:sample_count, :last_updated, :stale_at,
			:model_version, :confidence, :metadata
		)
	`

	_, err := s.db.NamedExecContext(ctx, query, profile)
	if err != nil {
		return fmt.Errorf("create profile: %w", err)
	}

	return nil
}

// UpdateProfile updates a behavior profile with new sample data
func (s *BehaviorProfileService) UpdateProfile(ctx context.Context, profile *BehaviorProfile, samples []BehaviorSample) error {
	if len(samples) < s.updateThreshold {
		return fmt.Errorf("insufficient samples for update")
	}

	// Calculate new statistics
	newStats := s.calculateStatistics(samples, profile)

	// Update profile with weighted average of old and new data
	alpha := 0.3 // Weight for new data

	// Update session duration stats
	profile.AverageSessionDuration = int(float64(profile.AverageSessionDuration)*(1-alpha) + float64(newStats.AvgDuration)*alpha)
	profile.SessionDurationStdDev = profile.SessionDurationStdDev*(1-alpha) + newStats.StdDevDuration*alpha

	// Update peak hours
	profile.PeakHours = mergeSlices(profile.PeakHours, newStats.PeakHours, alpha)

	// Update peak days
	profile.PeakDays = mergeSlices(profile.PeakDays, newStats.PeakDays, alpha)

	// Update frequent resources
	profile.FrequentResources = updateFrequencies(profile.FrequentResources, newStats.ResourceFreq)

	// Update frequent commands
	profile.FrequentCommands = updateFrequenciesCommands(profile.FrequentCommands, newStats.CommandFreq)

	// Update metadata
	profile.SampleCount += len(samples)
	profile.LastUpdated = time.Now()
	profile.StaleAt = time.Now().Add(s.staleThreshold)
	profile.Confidence = math.Min(1.0, float64(profile.SampleCount)/float64(s.minSamples*2))

	query := `
		UPDATE behavior_profiles
		SET peak_hours = :peak_hours,
			peak_days = :peak_days,
			average_session_duration = :average_session_duration,
			session_duration_stddev = :session_duration_stddev,
			frequent_resources = :frequent_resources,
			frequent_commands = :frequent_commands,
			typical_resource_count = :typical_resource_count,
			sample_count = :sample_count,
			last_updated = :last_updated,
			stale_at = :stale_at,
			confidence = :confidence,
			metadata = :metadata
		WHERE id = :id
	`

	_, err := s.db.NamedExecContext(ctx, query, profile)
	if err != nil {
		return fmt.Errorf("update profile: %w", err)
	}

	return nil
}

// AddSample adds a behavior sample for analysis
func (s *BehaviorProfileService) AddSample(ctx context.Context, sample *BehaviorSample) error {
	sample.ID = uuid.New()

	query := `
		INSERT INTO behavior_samples (
			id, tenant_id, user_id, session_id, timestamp,
			duration, resource_count, command_count,
			ip_address, location, user_agent, hour_of_day, day_of_week,
			resources, commands, anomaly_score, is_anomaly
		) VALUES (
			:id, :tenant_id, :user_id, :session_id, :timestamp,
			:duration, :resource_count, :command_count,
			:ip_address, :location, :user_agent, :hour_of_day, :day_of_week,
			:resources, :commands, :anomaly_score, :is_anomaly
		)
	`

	_, err := s.db.NamedExecContext(ctx, query, sample)
	if err != nil {
		return fmt.Errorf("add sample: %w", err)
	}

	return nil
}

// GetSamples retrieves behavior samples for a user
func (s *BehaviorProfileService) GetSamples(ctx context.Context, tenantID, userID uuid.UUID, limit int) ([]BehaviorSample, error) {
	query := `
		SELECT id, tenant_id, user_id, session_id, timestamp,
			duration, resource_count, command_count,
			ip_address, location, user_agent, hour_of_day, day_of_week,
			resources, commands, anomaly_score, is_anomaly
		FROM behavior_samples
		WHERE tenant_id = $1 AND user_id = $2
		ORDER BY timestamp DESC
		LIMIT $3
	`

	var samples []BehaviorSample
	err := s.db.SelectContext(ctx, &samples, query, tenantID, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("get samples: %w", err)
	}

	return samples, nil
}

// CalculateStatistics calculates statistics from behavior samples
type ProfileStatistics struct {
	AvgDuration    float64
	StdDevDuration float64
	PeakHours      []int
	PeakDays       []int
	ResourceFreq   []ResourceFrequency
	CommandFreq    []CommandFrequency
	TypicalIPs     []string
}

func (s *BehaviorProfileService) calculateStatistics(samples []BehaviorSample, profile *BehaviorProfile) ProfileStatistics {
	stats := ProfileStatistics{}

	if len(samples) == 0 {
		return stats
	}

	// Calculate duration statistics
	sumDuration := 0
	sumSquaredDuration := 0
	for _, s := range samples {
		sumDuration += s.Duration
		sumSquaredDuration += s.Duration * s.Duration
	}
	stats.AvgDuration = float64(sumDuration) / float64(len(samples))
	variance := float64(sumSquaredDuration)/float64(len(samples)) - stats.AvgDuration*stats.AvgDuration
	stats.StdDevDuration = math.Sqrt(math.Max(0, variance))

	// Calculate peak hours
	hourCounts := make(map[int]int)
	for _, s := range samples {
		hourCounts[s.HourOfDay]++
	}
	stats.PeakHours = topNItems(hourCounts, 6) // Top 6 hours

	// Calculate peak days
	dayCounts := make(map[int]int)
	for _, s := range samples {
		dayCounts[s.DayOfWeek]++
	}
	stats.PeakDays = topNItems(dayCounts, 5) // Top 5 days

	// Calculate resource frequencies
	resourceCounts := make(map[string]int)
	for _, s := range samples {
		for _, r := range s.Resources {
			resourceCounts[r]++
		}
	}
	stats.ResourceFreq = topResources(resourceCounts, len(samples), 20)

	// Calculate command frequencies
	commandCounts := make(map[string]int)
	for _, s := range samples {
		for _, c := range s.Commands {
			commandCounts[c]++
		}
	}
	stats.CommandFreq = topCommands(commandCounts, len(samples), 20)

	return stats
}

// Helper functions

func mergeSlices(existing, new []int, alpha float64) []int {
	merged := make(map[int]float64)

	// Add existing with weight
	for _, v := range existing {
		merged[v] += (1 - alpha)
	}

	// Add new with weight
	for _, v := range new {
		merged[v] += alpha
	}

	// Convert back to slice sorted by value
	result := make([]int, 0, len(merged))
	for k := range merged {
		result = append(result, k)
	}
	return result
}

func updateFrequencies(existing []ResourceFrequency, new []ResourceFrequency) []ResourceFrequency {
	freqMap := make(map[string]float64)

	// Add existing
	for _, f := range existing {
		freqMap[f.ResourceID] = f.Frequency
	}

	// Update with new (simple max)
	for _, f := range new {
		if f.Frequency > freqMap[f.ResourceID] {
			freqMap[f.ResourceID] = f.Frequency
		}
	}

	// Convert back to slice
	result := make([]ResourceFrequency, 0, len(freqMap))
	for id, freq := range freqMap {
		result = append(result, ResourceFrequency{
			ResourceID: id,
			Frequency:  freq,
		})
	}

	return result
}

func updateFrequenciesCommands(existing []CommandFrequency, new []CommandFrequency) []CommandFrequency {
	freqMap := make(map[string]float64)

	// Add existing
	for _, f := range existing {
		freqMap[f.Command] = f.Frequency
	}

	// Update with new
	for _, f := range new {
		if f.Frequency > freqMap[f.Command] {
			freqMap[f.Command] = f.Frequency
		}
	}

	// Convert back to slice
	result := make([]CommandFrequency, 0, len(freqMap))
	for cmd, freq := range freqMap {
		result = append(result, CommandFrequency{
			Command:   cmd,
			Frequency: freq,
		})
	}

	return result
}

func topNItems(m map[int]int, n int) []int {
	type kv struct {
		key   int
		value int
	}

	var pairs []kv
	for k, v := range m {
		pairs = append(pairs, kv{k, v})
	}

	// Simple sort (should use proper sort in production)
	for i := 0; i < len(pairs)-1; i++ {
		for j := i + 1; j < len(pairs); j++ {
			if pairs[j].value > pairs[i].value {
				pairs[i], pairs[j] = pairs[j], pairs[i]
			}
		}
	}

	result := make([]int, 0, n)
	for i := 0; i < n && i < len(pairs); i++ {
		result = append(result, pairs[i].key)
	}
	return result
}

func topResources(m map[string]int, total int, n int) []ResourceFrequency {
	type kv struct {
		key   string
		value int
	}

	var pairs []kv
	for k, v := range m {
		pairs = append(pairs, kv{k, v})
	}

	// Sort by count
	for i := 0; i < len(pairs)-1; i++ {
		for j := i + 1; j < len(pairs); j++ {
			if pairs[j].value > pairs[i].value {
				pairs[i], pairs[j] = pairs[j], pairs[i]
			}
		}
	}

	result := make([]ResourceFrequency, 0, n)
	for i := 0; i < n && i < len(pairs); i++ {
		result = append(result, ResourceFrequency{
			ResourceID: pairs[i].key,
			Count:      pairs[i].value,
			Frequency:  float64(pairs[i].value) / float64(total),
		})
	}
	return result
}

func topCommands(m map[string]int, total int, n int) []CommandFrequency {
	type kv struct {
		key   string
		value int
	}

	var pairs []kv
	for k, v := range m {
		pairs = append(pairs, kv{k, v})
	}

	// Sort by count
	for i := 0; i < len(pairs)-1; i++ {
		for j := i + 1; j < len(pairs); j++ {
			if pairs[j].value > pairs[i].value {
				pairs[i], pairs[j] = pairs[j], pairs[i]
			}
		}
	}

	result := make([]CommandFrequency, 0, n)
	for i := 0; i < n && i < len(pairs); i++ {
		result = append(result, CommandFrequency{
			Command:   pairs[i].key,
			Count:     pairs[i].value,
			Frequency: float64(pairs[i].value) / float64(total),
		})
	}
	return result
}

func (p *BehaviorProfile) MarshalJSON() ([]byte, error) {
	type Alias BehaviorProfile
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(p),
	})
}
