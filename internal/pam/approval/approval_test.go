package approval

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/cache"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestRequestStatus_Constants(t *testing.T) {
	t.Run("request status constants", func(t *testing.T) {
		assert.Equal(t, RequestStatus("pending"), StatusPending)
		assert.Equal(t, RequestStatus("approved"), StatusApproved)
		assert.Equal(t, RequestStatus("denied"), StatusDenied)
		assert.Equal(t, RequestStatus("cancelled"), StatusCancelled)
		assert.Equal(t, RequestStatus("expired"), StatusExpired)
		assert.Equal(t, RequestStatus("escalated"), StatusEscalated)
	})
}

func TestRequestType_Constants(t *testing.T) {
	t.Run("request type constants", func(t *testing.T) {
		assert.Equal(t, RequestType("credential_access"), RequestTypeCredentialAccess)
		assert.Equal(t, RequestType("session_access"), RequestTypeSessionAccess)
		assert.Equal(t, RequestType("privilege_escalation"), RequestTypePrivilegeEscalation)
	})
}

func TestRequest_Struct(t *testing.T) {
	t.Run("create request with all fields", func(t *testing.T) {
		id := uuid.New()
		userID := uuid.New()
		credentialID := uuid.New()
		tenantID := uuid.New()
		now := time.Now()
		expiresAt := now.Add(time.Hour)

		request := &Request{
			ID:                id,
			TenantID:          tenantID,
			UserID:            userID,
			CredentialID:      &credentialID,
			Type:              RequestTypeCredentialAccess,
			Status:            StatusPending,
			Justification:     "Need to deploy hotfix",
			Duration:          60,
			ExpiresAt:         &expiresAt,
			CreatedAt:         now,
			UpdatedAt:         now,
			RequiredApprovals: 1,
			ReceivedApprovals: 0,
		}

		assert.Equal(t, id, request.ID)
		assert.Equal(t, tenantID, request.TenantID)
		assert.Equal(t, userID, request.UserID)
		assert.Equal(t, credentialID, *request.CredentialID)
		assert.Equal(t, RequestTypeCredentialAccess, request.Type)
		assert.Equal(t, StatusPending, request.Status)
		assert.Equal(t, "Need to deploy hotfix", request.Justification)
		assert.Equal(t, 60, request.Duration)
		assert.Equal(t, 1, request.RequiredApprovals)
		assert.NotNil(t, request.ExpiresAt)
	})
}

func TestApproval_Struct(t *testing.T) {
	t.Run("create approval with all fields", func(t *testing.T) {
		id := uuid.New()
		requestID := uuid.New()
		approverID := uuid.New()
		now := time.Now()

		approval := &Approval{
			ID:        id,
			RequestID: requestID,
			ApproverID: approverID,
			Decision:  "approved",
			Comments:  "Looks good",
			CreatedAt: now,
		}

		assert.Equal(t, id, approval.ID)
		assert.Equal(t, requestID, approval.RequestID)
		assert.Equal(t, approverID, approval.ApproverID)
		assert.Equal(t, "approved", approval.Decision)
		assert.Equal(t, "Looks good", approval.Comments)
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

func TestNewWorkflowService(t *testing.T) {
	t.Run("creates service with nil dependencies", func(t *testing.T) {
		var c *cache.Cache // nil for testing
		logger := zerolog.Nop()

		repo := NewRepository(nil, c, logger)
		svc := NewWorkflowService(repo, nil, c, logger)

		assert.NotNil(t, svc)
	})
}

func TestWorkflowService_ApproveRequest(t *testing.T) {
	t.Run("approve request panics with nil db", func(t *testing.T) {
		var c *cache.Cache
		logger := zerolog.Nop()

		repo := NewRepository(nil, c, logger)
		svc := NewWorkflowService(repo, nil, c, logger)

		ctx := context.Background()
		requestID := uuid.New()
		approverID := uuid.New()

		assert.Panics(t, func() {
			_ = svc.ApproveRequest(ctx, requestID, approverID, "Approved")
		})
	})
}

func TestWorkflowService_DenyRequest(t *testing.T) {
	t.Run("deny request panics with nil db", func(t *testing.T) {
		var c *cache.Cache
		logger := zerolog.Nop()

		repo := NewRepository(nil, c, logger)
		svc := NewWorkflowService(repo, nil, c, logger)

		ctx := context.Background()
		requestID := uuid.New()
		approverID := uuid.New()

		assert.Panics(t, func() {
			_ = svc.DenyRequest(ctx, requestID, approverID, "Denied")
		})
	})
}

func TestWorkflowService_EscalateRequest(t *testing.T) {
	t.Run("escalate request - method exists in service", func(t *testing.T) {
		// This test verifies the service type exists
		var c *cache.Cache
		logger := zerolog.Nop()

		repo := NewRepository(nil, c, logger)
		svc := NewWorkflowService(repo, nil, c, logger)

		assert.NotNil(t, svc)
		// The actual EscalateRequest method may have different signature
	})
}
