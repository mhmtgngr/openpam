package recording

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/cache"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestRecordingType_Constants(t *testing.T) {
	t.Run("recording type constants", func(t *testing.T) {
		assert.Equal(t, RecordingType("keystrokes"), RecordingTypeKeystrokes)
		assert.Equal(t, RecordingType("video"), RecordingTypeVideo)
		assert.Equal(t, RecordingType("audio"), RecordingTypeAudio)
		assert.Equal(t, RecordingType("metadata"), RecordingTypeMetadata)
	})
}

func TestRecordingStatus_Constants(t *testing.T) {
	t.Run("recording status constants", func(t *testing.T) {
		assert.Equal(t, RecordingStatus("recording"), RecordingStatusRecording)
		assert.Equal(t, RecordingStatus("processing"), RecordingStatusProcessing)
		assert.Equal(t, RecordingStatus("ready"), RecordingStatusReady)
		assert.Equal(t, RecordingStatus("failed"), RecordingStatusFailed)
		assert.Equal(t, RecordingStatus("deleted"), RecordingStatusDeleted)
	})
}

func TestRecording_Struct(t *testing.T) {
	t.Run("recording with all fields", func(t *testing.T) {
		id := uuid.New()
		sessionID := uuid.New()
		tenantID := uuid.New()
		now := time.Now()
		endTime := now.Add(5 * time.Minute)

		recording := &Recording{
			ID:              id,
			SessionID:       sessionID,
			Type:            RecordingTypeVideo,
			Status:          RecordingStatusReady,
			StorageURL:      "s3://bucket/session-123.mp4",
			StorageProvider: "s3",
			FileSize:        1024000,
			Duration:        300,
			StartTime:       now,
			EndTime:         &endTime,
			Format:          "mp4",
			Encrypted:       true,
			TenantID:        tenantID,
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		assert.Equal(t, id, recording.ID)
		assert.Equal(t, sessionID, recording.SessionID)
		assert.Equal(t, RecordingTypeVideo, recording.Type)
		assert.Equal(t, RecordingStatusReady, recording.Status)
		assert.Equal(t, "s3://bucket/session-123.mp4", recording.StorageURL)
		assert.Equal(t, "s3", recording.StorageProvider)
		assert.Equal(t, int64(1024000), recording.FileSize)
		assert.Equal(t, 300, recording.Duration)
	})
}

func TestRecordingSegment_Struct(t *testing.T) {
	t.Run("recording segment", func(t *testing.T) {
		id := uuid.New()
		recordingID := uuid.New()
		now := time.Now()

		segment := &RecordingSegment{
			ID:          id,
			RecordingID: recordingID,
			SegmentNum:  1,
			StorageURL:  "s3://bucket/segment-001.ts",
			StartTime:   now,
			EndTime:     now.Add(30 * time.Second),
			FileSize:    102400,
			CreatedAt:   now,
		}

		assert.Equal(t, id, segment.ID)
		assert.Equal(t, recordingID, segment.RecordingID)
		assert.Equal(t, 1, segment.SegmentNum)
	})
}

func TestNewRepository(t *testing.T) {
	t.Run("creates repository with nil dependencies", func(t *testing.T) {
		var c *cache.Cache // nil for testing
		logger := zerolog.Nop()

		repo := NewRepository(nil, c, logger)

		assert.NotNil(t, repo)
	})
}

func TestNewService(t *testing.T) {
	t.Run("creates service with nil dependencies", func(t *testing.T) {
		var c *cache.Cache
		logger := zerolog.Nop()

		repo := NewRepository(nil, c, logger)
		svc := NewService(repo, nil, logger)

		assert.NotNil(t, svc)
	})
}

func TestService_StartRecording(t *testing.T) {
	t.Run("start recording panics with nil db", func(t *testing.T) {
		var c *cache.Cache
		logger := zerolog.Nop()
		repo := NewRepository(nil, c, logger)
		svc := NewService(repo, nil, logger)

		ctx := context.Background()
		sessionID := uuid.New()
		tenantID := uuid.New()

		assert.Panics(t, func() {
			_, _ = svc.StartRecording(ctx, sessionID, RecordingTypeVideo, tenantID)
		})
	})
}

func TestService_EndRecording(t *testing.T) {
	t.Run("end recording panics with nil db", func(t *testing.T) {
		var c *cache.Cache
		logger := zerolog.Nop()
		repo := NewRepository(nil, c, logger)
		svc := NewService(repo, nil, logger)

		ctx := context.Background()
		recordingID := uuid.New()

		assert.Panics(t, func() {
			_ = svc.EndRecording(ctx, recordingID)
		})
	})
}

func TestService_GetRecordingURL(t *testing.T) {
	t.Run("get recording URL panics with nil db", func(t *testing.T) {
		var c *cache.Cache
		logger := zerolog.Nop()
		repo := NewRepository(nil, c, logger)
		svc := NewService(repo, nil, logger)

		ctx := context.Background()
		recordingID := uuid.New()

		assert.Panics(t, func() {
			_, _ = svc.GetRecordingURL(ctx, recordingID)
		})
	})
}

func TestService_SearchRecordings(t *testing.T) {
	t.Run("search recordings panics with nil db", func(t *testing.T) {
		var c *cache.Cache
		logger := zerolog.Nop()
		repo := NewRepository(nil, c, logger)
		svc := NewService(repo, nil, logger)

		ctx := context.Background()
		tenantID := uuid.New()

		assert.Panics(t, func() {
			_, _, _ = svc.SearchRecordings(ctx, tenantID, "test query", 10, 0)
		})
	})
}

func TestRecordingEvent_Struct(t *testing.T) {
	t.Run("recording event with all fields", func(t *testing.T) {
		id := uuid.New()
		recordingID := uuid.New()
		now := time.Now()

		event := &RecordingEvent{
			ID:          id,
			RecordingID: recordingID,
			Timestamp:   now,
			Type:        "keystroke",
			Data:        []byte(`{"key": "a"}`),
		}

		assert.Equal(t, id, event.ID)
		assert.Equal(t, recordingID, event.RecordingID)
		assert.Equal(t, "keystroke", event.Type)
	})
}

func TestS3Storage_Struct(t *testing.T) {
	t.Run("S3 storage initialization with static credentials (deprecated)", func(t *testing.T) {
		storage := NewS3Storage("test-bucket", "endpoint", "key", "secret", "us-east-1")

		assert.NotNil(t, storage)
		assert.Equal(t, "test-bucket", storage.bucket)
		assert.Equal(t, "endpoint", storage.endpoint)
		assert.Equal(t, "us-east-1", storage.region)
		assert.False(t, storage.useIAM) // Static credentials mode
	})

	t.Run("S3 storage initialization with IAM (recommended)", func(t *testing.T) {
		storage := NewS3StorageWithIAM("test-bucket", "endpoint", "us-east-1", "arn:aws:iam::123456789012:role/MyRole")

		assert.NotNil(t, storage)
		assert.Equal(t, "test-bucket", storage.bucket)
		assert.Equal(t, "endpoint", storage.endpoint)
		assert.Equal(t, "us-east-1", storage.region)
		assert.True(t, storage.useIAM) // IAM mode
		assert.Equal(t, "arn:aws:iam::123456789012:role/MyRole", storage.roleARN)
	})

	t.Run("S3 storage Upload returns URL", func(t *testing.T) {
		storage := NewS3Storage("test-bucket", "endpoint", "key", "secret", "us-east-1")

		ctx := context.Background()
		recordingID := uuid.New()
		recording := &Recording{
			ID:     recordingID,
			Type:   RecordingTypeVideo,
			Status: RecordingStatusRecording,
		}

		url, err := storage.Upload(ctx, recordingID, recording)

		assert.NoError(t, err)
		assert.Contains(t, url, "s3://test-bucket/")
		assert.Contains(t, url, recordingID.String())
	})

	t.Run("S3 storage GetSignedURL", func(t *testing.T) {
		storage := NewS3Storage("test-bucket", "endpoint", "key", "secret", "us-east-1")

		recording := &Recording{
			StorageURL: "s3://test-bucket/test.mp4",
		}

		url, err := storage.GetSignedURL(nil, recording, 1*time.Hour)

		assert.NoError(t, err)
		assert.Equal(t, "s3://test-bucket/test.mp4", url)
	})

	t.Run("S3 storage Delete", func(t *testing.T) {
		storage := NewS3Storage("test-bucket", "endpoint", "key", "secret", "us-east-1")

		err := storage.Delete(nil, uuid.New())

		assert.NoError(t, err)
	})
}

func TestRecordingFilter_Struct(t *testing.T) {
	t.Run("recording filter with all fields", func(t *testing.T) {
		status := RecordingStatusReady
		recType := RecordingTypeVideo
		sessionID := uuid.New()

		filter := RecordingFilter{
			Type:       &recType,
			Status:     &status,
			SessionID:  &sessionID,
			SearchTerm: "test",
		}

		assert.Equal(t, &recType, filter.Type)
		assert.Equal(t, &status, filter.Status)
		assert.Equal(t, &sessionID, filter.SessionID)
		assert.Equal(t, "test", filter.SearchTerm)
	})
}
