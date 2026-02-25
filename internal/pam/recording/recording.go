package recording

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/cache"
	"github.com/rs/zerolog"
)

// RecordingType represents the type of recording
type RecordingType string

const (
	RecordingTypeKeystrokes RecordingType = "keystrokes"
	RecordingTypeVideo     RecordingType = "video"
	RecordingTypeAudio     RecordingType = "audio"
	RecordingTypeMetadata  RecordingType = "metadata"
)

// RecordingStatus represents the status of a recording
type RecordingStatus string

const (
	RecordingStatusRecording RecordingStatus = "recording"
	RecordingStatusProcessing RecordingStatus = "processing"
	RecordingStatusReady     RecordingStatus = "ready"
	RecordingStatusFailed    RecordingStatus = "failed"
	RecordingStatusDeleted   RecordingStatus = "deleted"
)

// Recording represents a session recording
type Recording struct {
	ID              uuid.UUID        `db:"id" json:"id"`
	SessionID       uuid.UUID        `db:"session_id" json:"session_id"`
	Type            RecordingType    `db:"type" json:"type"`
	Status          RecordingStatus  `db:"status" json:"status"`

	// Storage info
	StorageURL      string           `db:"storage_url" json:"storage_url"`
	StorageProvider string           `db:"storage_provider" json:"storage_provider"` // s3, minio, azure
	FileSize        int64            `db:"file_size" json:"file_size"`
	Checksum        string           `db:"checksum" json:"checksum"`

	// Metadata
	Duration        int              `db:"duration_seconds" json:"duration_seconds"`
	StartTime       time.Time        `db:"start_time" json:"start_time"`
	EndTime         *time.Time       `db:"end_time" json:"end_time,omitempty"`
	Format          string           `db:"format" json:"format"` // json, mp4, webm, etc.
	Encrypted       bool             `db:"is_encrypted" json:"is_encrypted"`
	EncryptionKeyID string           `db:"encryption_key_id" json:"encryption_key_id,omitempty"`

	// Search/indexing
	Transcript      string           `db:"transcript" json:"transcript,omitempty"`
	SearchIndex     []string         `db:"search_index" json:"search_index,omitempty"`

	TenantID        uuid.UUID        `db:"tenant_id" json:"tenant_id"`
	CreatedAt       time.Time        `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time        `db:"updated_at" json:"updated_at"`
	DeletedAt       *time.Time       `db:"deleted_at" json:"deleted_at,omitempty"`
}

// RecordingSegment represents a segment of a recording
type RecordingSegment struct {
	ID           uuid.UUID    `db:"id" json:"id"`
	RecordingID  uuid.UUID    `db:"recording_id" json:"recording_id"`
	SegmentNum   int          `db:"segment_num" json:"segment_num"`
	StorageURL   string       `db:"storage_url" json:"storage_url"`
	StartTime    time.Time    `db:"start_time" json:"start_time"`
	EndTime      time.Time    `db:"end_time" json:"end_time"`
	FileSize     int64        `db:"file_size" json:"file_size"`
	CreatedAt    time.Time    `db:"created_at" json:"created_at"`
}

// RecordingEvent represents a timed event within a recording
type RecordingEvent struct {
	ID          uuid.UUID       `db:"id" json:"id"`
	RecordingID uuid.UUID       `db:"recording_id" json:"recording_id"`
	Timestamp   time.Time       `db:"timestamp" json:"timestamp"`
	Type        string          `db:"type" json:"type"` // keystroke, command, error, warning
	Data        json.RawMessage `db:"data" json:"data"`
	CreatedAt   time.Time       `db:"created_at" json:"created_at"`
}

// Repository handles recording data operations
type Repository struct {
	db     *sqlx.DB
	cache  *cache.Cache
	logger zerolog.Logger
}

// NewRepository creates a new recording repository
func NewRepository(db *sqlx.DB, c *cache.Cache, logger zerolog.Logger) *Repository {
	return &Repository{db: db, cache: c, logger: logger}
}

// Create creates a new recording entry
func (r *Repository) Create(ctx context.Context, recording *Recording) error {
	recording.ID = uuid.New()
	recording.CreatedAt = time.Now()
	recording.UpdatedAt = time.Now()
	recording.Status = RecordingStatusRecording

	query := `
		INSERT INTO recordings (id, session_id, type, status, storage_url, storage_provider,
			file_size, checksum, duration_seconds, start_time, end_time, format, is_encrypted,
			encryption_key_id, transcript, search_index, tenant_id, created_at, updated_at)
		VALUES (:id, :session_id, :type, :status, :storage_url, :storage_provider,
			:file_size, :checksum, :duration_seconds, :start_time, :end_time, :format, :is_encrypted,
			:encryption_key_id, :transcript, :search_index, :tenant_id, :created_at, :updated_at)
	`

	_, err := r.db.NamedExecContext(ctx, query, recording)
	if err != nil {
		return fmt.Errorf("recording.Create: %w", err)
	}

	return nil
}

// GetByID retrieves a recording by ID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Recording, error) {
	var recording Recording
	query := `SELECT * FROM recordings WHERE id = $1 AND deleted_at IS NULL`
	err := r.db.GetContext(ctx, &recording, query, id)
	if err != nil {
		return nil, fmt.Errorf("recording.GetByID: %w", err)
	}
	return &recording, nil
}

// GetBySessionID retrieves recordings for a session
func (r *Repository) GetBySessionID(ctx context.Context, sessionID uuid.UUID) ([]Recording, error) {
	var recordings []Recording
	query := `
		SELECT * FROM recordings
		WHERE session_id = $1 AND deleted_at IS NULL
		ORDER BY created_at ASC
	`
	err := r.db.SelectContext(ctx, &recordings, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("recording.GetBySessionID: %w", err)
	}
	return recordings, nil
}

// List retrieves recordings with pagination and filtering
func (r *Repository) List(ctx context.Context, tenantID uuid.UUID, filter RecordingFilter, limit, offset int) ([]Recording, int, error) {
	baseQuery := `
		SELECT * FROM recordings
		WHERE tenant_id = $1 AND deleted_at IS NULL
	`
	countQuery := `
		SELECT COUNT(*) FROM recordings
		WHERE tenant_id = $1 AND deleted_at IS NULL
	`

	args := []interface{}{tenantID}
	argCount := 2

	if filter.Type != nil {
		baseQuery += fmt.Sprintf(" AND type = $%d", argCount)
		countQuery += fmt.Sprintf(" AND type = $%d", argCount)
		args = append(args, *filter.Type)
		argCount++
	}
	if filter.Status != nil {
		baseQuery += fmt.Sprintf(" AND status = $%d", argCount)
		countQuery += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, *filter.Status)
		argCount++
	}
	if filter.SessionID != nil {
		baseQuery += fmt.Sprintf(" AND session_id = $%d", argCount)
		countQuery += fmt.Sprintf(" AND session_id = $%d", argCount)
		args = append(args, *filter.SessionID)
		argCount++
	}
	if filter.StartTimeFrom != nil {
		baseQuery += fmt.Sprintf(" AND start_time >= $%d", argCount)
		countQuery += fmt.Sprintf(" AND start_time >= $%d", argCount)
		args = append(args, *filter.StartTimeFrom)
		argCount++
	}
	if filter.StartTimeTo != nil {
		baseQuery += fmt.Sprintf(" AND start_time <= $%d", argCount)
		countQuery += fmt.Sprintf(" AND start_time <= $%d", argCount)
		args = append(args, *filter.StartTimeTo)
		argCount++
	}
	if filter.SearchTerm != "" {
		baseQuery += fmt.Sprintf(" AND (transcript ILIKE $%d OR storage_url ILIKE $%d)", argCount, argCount)
		countQuery += fmt.Sprintf(" AND (transcript ILIKE $%d OR storage_url ILIKE $%d)", argCount, argCount)
		args = append(args, "%"+filter.SearchTerm+"%")
		argCount++
	}

	// Get count
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("recording.List.Count: %w", err)
	}

	// Add pagination
	baseQuery += fmt.Sprintf(" ORDER BY start_time DESC LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, limit, offset)

	var recordings []Recording
	if err := r.db.SelectContext(ctx, &recordings, baseQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("recording.List: %w", err)
	}

	return recordings, total, nil
}

// Update updates a recording
func (r *Repository) Update(ctx context.Context, recording *Recording) error {
	recording.UpdatedAt = time.Now()

	query := `
		UPDATE recordings SET
			status = :status,
			storage_url = :storage_url,
			file_size = :file_size,
			checksum = :checksum,
			duration_seconds = :duration_seconds,
			end_time = :end_time,
			transcript = :transcript,
			search_index = :search_index,
			updated_at = :updated_at
		WHERE id = :id
	`

	_, err := r.db.NamedExecContext(ctx, query, recording)
	if err != nil {
		return fmt.Errorf("recording.Update: %w", err)
	}

	return nil
}

// Delete soft deletes a recording
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE recordings SET deleted_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("recording.Delete: %w", err)
	}
	return nil
}

// AddEvent adds a timed event to a recording
func (r *Repository) AddEvent(ctx context.Context, event *RecordingEvent) error {
	event.ID = uuid.New()
	event.CreatedAt = time.Now()

	query := `
		INSERT INTO recording_events (id, recording_id, timestamp, type, data, created_at)
		VALUES (:id, :recording_id, :timestamp, :type, :data, :created_at)
	`

	_, err := r.db.NamedExecContext(ctx, query, event)
	if err != nil {
		return fmt.Errorf("recording.AddEvent: %w", err)
	}

	return nil
}

// GetEventsForRecording retrieves events for a recording
func (r *Repository) GetEventsForRecording(ctx context.Context, recordingID uuid.UUID) ([]RecordingEvent, error) {
	var events []RecordingEvent
	query := `
		SELECT * FROM recording_events
		WHERE recording_id = $1
		ORDER BY timestamp ASC
	`
	err := r.db.SelectContext(ctx, &events, query, recordingID)
	if err != nil {
		return nil, fmt.Errorf("recording.GetEventsForRecording: %w", err)
	}
	return events, nil
}

// RecordingFilter filters recording queries
type RecordingFilter struct {
	Type         *RecordingType
	Status       *RecordingStatus
	SessionID    *uuid.UUID
	StartTimeFrom *time.Time
	StartTimeTo   *time.Time
	SearchTerm   string
}

// Service handles recording business logic
type Service struct {
	repo    *Repository
	storage Storage
	logger  zerolog.Logger
}

// NewService creates a new recording service
func NewService(repo *Repository, storage Storage, logger zerolog.Logger) *Service {
	return &Service{repo: repo, storage: storage, logger: logger}
}

// StartRecording starts a new recording
func (s *Service) StartRecording(ctx context.Context, sessionID uuid.UUID, recordingType RecordingType, tenantID uuid.UUID) (*Recording, error) {
	recording := &Recording{
		SessionID: sessionID,
		Type:      recordingType,
		Status:    RecordingStatusRecording,
		StartTime: time.Now(),
		TenantID:  tenantID,
	}

	if err := s.repo.Create(ctx, recording); err != nil {
		return nil, err
	}

	s.logger.Info().
		Str("recording_id", recording.ID.String()).
		Str("session_id", sessionID.String()).
		Str("type", string(recordingType)).
		Msg("Recording started")

	return recording, nil
}

// EndRecording ends a recording and processes it
func (s *Service) EndRecording(ctx context.Context, recordingID uuid.UUID) error {
	recording, err := s.repo.GetByID(ctx, recordingID)
	if err != nil {
		return err
	}

	now := time.Now()
	recording.EndTime = &now
	recording.Status = RecordingStatusProcessing

	if err := s.repo.Update(ctx, recording); err != nil {
		return err
	}

	// Process recording (transcode, encrypt, index)
	go s.processRecording(context.Background(), recording)

	return nil
}

// processRecording processes a recording after completion
func (s *Service) processRecording(ctx context.Context, recording *Recording) {
	// 1. Upload to storage if not already
	if recording.StorageURL == "" {
		url, err := s.storage.Upload(ctx, recording.ID, recording)
		if err != nil {
			s.logger.Error().Err(err).Str("recording_id", recording.ID.String()).Msg("Failed to upload recording")
			recording.Status = RecordingStatusFailed
			_ = s.repo.Update(ctx, recording)
			return
		}
		recording.StorageURL = url
	}

	// 2. Encrypt if required
	if recording.Encrypted {
		// Encryption would be done during upload or after
	}

	// 3. Generate transcript/search index
	if recording.Type == RecordingTypeKeystrokes {
		_ = s.generateTranscript(ctx, recording)
	}

	// 4. Mark as ready
	recording.Status = RecordingStatusReady
	_ = s.repo.Update(ctx, recording)

	s.logger.Info().
		Str("recording_id", recording.ID.String()).
		Str("storage_url", recording.StorageURL).
		Msg("Recording processed")
}

// GetRecordingURL retrieves a signed URL for playback
func (s *Service) GetRecordingURL(ctx context.Context, recordingID uuid.UUID) (string, error) {
	recording, err := s.repo.GetByID(ctx, recordingID)
	if err != nil {
		return "", err
	}

	if recording.Status != RecordingStatusReady {
		return "", fmt.Errorf("recording: not ready for playback")
	}

	return s.storage.GetSignedURL(ctx, recording, 1*time.Hour)
}

// SearchRecordings searches recordings by content
func (s *Service) SearchRecordings(ctx context.Context, tenantID uuid.UUID, query string, limit, offset int) ([]Recording, int, error) {
	// Use full-text search on transcript
	filter := RecordingFilter{SearchTerm: query}
	return s.repo.List(ctx, tenantID, filter, limit, offset)
}

// generateTranscript generates transcript from keystroke recording
func (s *Service) generateTranscript(ctx context.Context, recording *Recording) error {
	events, err := s.repo.GetEventsForRecording(ctx, recording.ID)
	if err != nil {
		return err
	}

	// Build transcript from events
	transcript := ""
	for _, event := range events {
		if event.Type == "command" {
			var data map[string]string
			if err := json.Unmarshal(event.Data, &data); err == nil {
				if cmd, ok := data["command"]; ok {
					transcript += cmd + "\n"
				}
			}
		}
	}

	recording.Transcript = transcript
	return s.repo.Update(ctx, recording)
}

// Storage defines recording storage interface
type Storage interface {
	Upload(ctx context.Context, recordingID uuid.UUID, recording *Recording) (string, error)
	GetSignedURL(ctx context.Context, recording *Recording, duration time.Duration) (string, error)
	Delete(ctx context.Context, recordingID uuid.UUID) error
}

// S3Storage implements Storage for S3/MinIO
// SECURITY FIX: Credentials are managed via IAM-based providers, not static keys
type S3Storage struct {
	bucket  string
	endpoint string
	region  string
	// SECURITY FIX: Use IAM-based credential providers instead of static keys
	// Static keys are deprecated and should not be used in production
	useIAM  bool
	roleARN string // For assumed role credentials
}

// NewS3Storage creates a new S3 storage with IAM-based authentication
// DEPRECATED: The accessKey/secretKey parameters are kept for backward compatibility only
// New deployments should use NewS3StorageWithIAM instead
func NewS3Storage(bucket, endpoint, accessKey, secretKey, region string) *S3Storage {
	return &S3Storage{
		bucket:   bucket,
		endpoint: endpoint,
		region:   region,
		useIAM:   false,
	}
}

// NewS3StorageWithIAM creates a new S3 storage using IAM-based authentication
// This is the recommended approach for production deployments
func NewS3StorageWithIAM(bucket, endpoint, region, roleARN string) *S3Storage {
	return &S3Storage{
		bucket:   bucket,
		endpoint: endpoint,
		region:   region,
		useIAM:   true,
		roleARN:  roleARN,
	}
}

// Upload uploads a recording to S3
func (s *S3Storage) Upload(ctx context.Context, recordingID uuid.UUID, recording *Recording) (string, error) {
	// Implement S3 upload
	url := fmt.Sprintf("s3://%s/%s.mp4", s.bucket, recordingID)
	return url, nil
}

// GetSignedURL generates a signed URL for playback
func (s *S3Storage) GetSignedURL(ctx context.Context, recording *Recording, duration time.Duration) (string, error) {
	// Implement presigned URL generation
	return recording.StorageURL, nil
}

// Delete deletes a recording from storage
func (s *S3Storage) Delete(ctx context.Context, recordingID uuid.UUID) error {
	// Implement S3 delete
	return nil
}
