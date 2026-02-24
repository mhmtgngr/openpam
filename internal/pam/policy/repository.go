package policy

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

// Repository handles policy data operations
type Repository struct {
	db     *sqlx.DB
	logger zerolog.Logger
}

// NewRepository creates a new policy repository
func NewRepository(db *sqlx.DB, logger zerolog.Logger) *Repository {
	return &Repository{
		db:     db,
		logger: logger,
	}
}

// Create creates a new policy
func (r *Repository) Create(ctx context.Context, policy *Policy) error {
	policy.ID = uuid.New()
	policy.CreatedAt = time.Now()
	policy.UpdatedAt = policy.CreatedAt
	policy.Version = 1

	// Marshal JSONB fields
	userIDsJSON, err := json.Marshal(policy.UserIDs)
	if err != nil {
		return fmt.Errorf("policy.Create.MarshalUserIDs: %w", err)
	}
	roleIDsJSON, err := json.Marshal(policy.RoleIDs)
	if err != nil {
		return fmt.Errorf("policy.Create.MarshalRoleIDs: %w", err)
	}
	groupIDsJSON, err := json.Marshal(policy.GroupIDs)
	if err != nil {
		return fmt.Errorf("policy.Create.MarshalGroupIDs: %w", err)
	}
	targetIDsJSON, err := json.Marshal(policy.TargetIDs)
	if err != nil {
		return fmt.Errorf("policy.Create.MarshalTargetIDs: %w", err)
	}
	credentialIDsJSON, err := json.Marshal(policy.CredentialIDs)
	if err != nil {
		return fmt.Errorf("policy.Create.MarshalCredentialIDs: %w", err)
	}
	rulesJSON, err := json.Marshal(policy.Rules)
	if err != nil {
		return fmt.Errorf("policy.Create.MarshalRules: %w", err)
	}
	mfaMethodsJSON, err := json.Marshal(policy.MFAMethods)
	if err != nil {
		return fmt.Errorf("policy.Create.MarshalMFAMethods: %w", err)
	}
	approvalApproversJSON, err := json.Marshal(policy.ApprovalApprovers)
	if err != nil {
		return fmt.Errorf("policy.Create.MarshalApprovalApprovers: %w", err)
	}
	metadataJSON, err := json.Marshal(policy.Metadata)
	if err != nil {
		return fmt.Errorf("policy.Create.MarshalMetadata: %w", err)
	}
	tagsJSON, err := json.Marshal(policy.Tags)
	if err != nil {
		return fmt.Errorf("policy.Create.MarshalTags: %w", err)
	}

	query := `
		INSERT INTO policies (
			id, name, description, type, effect, priority,
			tenant_id, user_ids, role_ids, group_ids, target_ids, credential_ids,
			rules, max_session_duration_seconds, session_extension_allowed, max_extensions,
			mfa_required, mfa_methods, approval_required, approval_approvers, approval_timeout_minutes,
			recording_required, recording_mode, enabled, system_policy,
			metadata, tags, version, created_by, updated_by, created_at, updated_at
		) VALUES (
			:id, :name, :description, :type, :effect, :priority,
			:tenant_id, :user_ids, :role_ids, :group_ids, :target_ids, :credential_ids,
			:rules, :max_session_duration_seconds, :session_extension_allowed, :max_extensions,
			:mfa_required, :mfa_methods, :approval_required, :approval_approvers, :approval_timeout_minutes,
			:recording_required, :recording_mode, :enabled, :system_policy,
			:metadata, :tags, :version, :created_by, :updated_by, :created_at, :updated_at
		)
	`

	args := map[string]interface{}{
		"id":                            policy.ID,
		"name":                          policy.Name,
		"description":                   policy.Description,
		"type":                          policy.Type,
		"effect":                        policy.Effect,
		"priority":                      policy.Priority,
		"tenant_id":                     policy.TenantID,
		"user_ids":                      userIDsJSON,
		"role_ids":                      roleIDsJSON,
		"group_ids":                     groupIDsJSON,
		"target_ids":                    targetIDsJSON,
		"credential_ids":                credentialIDsJSON,
		"rules":                         rulesJSON,
		"max_session_duration_seconds":  policy.MaxSessionDurationSeconds,
		"session_extension_allowed":     policy.SessionExtensionAllowed,
		"max_extensions":                policy.MaxExtensions,
		"mfa_required":                  policy.MFARequired,
		"mfa_methods":                   mfaMethodsJSON,
		"approval_required":             policy.ApprovalRequired,
		"approval_approvers":            approvalApproversJSON,
		"approval_timeout_minutes":      policy.ApprovalTimeoutMinutes,
		"recording_required":            policy.RecordingRequired,
		"recording_mode":                policy.RecordingMode,
		"enabled":                       policy.Enabled,
		"system_policy":                 policy.SystemPolicy,
		"metadata":                      metadataJSON,
		"tags":                          tagsJSON,
		"version":                       policy.Version,
		"created_by":                    policy.CreatedBy,
		"updated_by":                    policy.UpdatedBy,
		"created_at":                    policy.CreatedAt,
		"updated_at":                    policy.UpdatedAt,
	}

	_, err = r.db.NamedExecContext(ctx, query, args)
	if err != nil {
		return fmt.Errorf("policy.Create: %w", err)
	}

	return nil
}

// GetByID retrieves a policy by ID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Policy, error) {
	var policy Policy
	query := `SELECT * FROM policies WHERE id = $1 AND deleted_at IS NULL`
	err := r.db.GetContext(ctx, &policy, query, id)
	if err != nil {
		return nil, fmt.Errorf("policy.GetByID: %w", err)
	}

	if err := r.unmarshalPolicyFields(&policy); err != nil {
		return nil, err
	}

	return &policy, nil
}

// GetByIDForTenant retrieves a policy by ID with tenant isolation
func (r *Repository) GetByIDForTenant(ctx context.Context, id, tenantID uuid.UUID) (*Policy, error) {
	var policy Policy
	query := `SELECT * FROM policies WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`
	err := r.db.GetContext(ctx, &policy, query, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("policy.GetByIDForTenant: %w", err)
	}

	if err := r.unmarshalPolicyFields(&policy); err != nil {
		return nil, err
	}

	return &policy, nil
}

// List retrieves policies with filtering and pagination
func (r *Repository) List(ctx context.Context, tenantID uuid.UUID, filter PolicyFilter, limit, offset int) ([]Policy, int, error) {
	baseQuery := `
		SELECT * FROM policies
		WHERE tenant_id = $1 AND deleted_at IS NULL
	`
	countQuery := `
		SELECT COUNT(*) FROM policies
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

	if filter.Effect != nil {
		baseQuery += fmt.Sprintf(" AND effect = $%d", argCount)
		countQuery += fmt.Sprintf(" AND effect = $%d", argCount)
		args = append(args, *filter.Effect)
		argCount++
	}

	if filter.Enabled != nil {
		baseQuery += fmt.Sprintf(" AND enabled = $%d", argCount)
		countQuery += fmt.Sprintf(" AND enabled = $%d", argCount)
		args = append(args, *filter.Enabled)
		argCount++
	}

	if filter.Search != "" {
		baseQuery += fmt.Sprintf(" AND (name ILIKE $%d OR description ILIKE $%d)", argCount, argCount)
		countQuery += fmt.Sprintf(" AND (name ILIKE $%d OR description ILIKE $%d)", argCount, argCount)
		searchPattern := "%" + filter.Search + "%"
		args = append(args, searchPattern, searchPattern)
		argCount += 2
	}

	if len(filter.Tags) > 0 {
		baseQuery += fmt.Sprintf(" AND tags ?| $%d", argCount)
		countQuery += fmt.Sprintf(" AND tags ?| $%d", argCount)
		args = append(args, filter.Tags)
		argCount++
	}

	// Get count
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("policy.List.Count: %w", err)
	}

	// Add pagination and ordering
	baseQuery += fmt.Sprintf(" ORDER BY priority ASC, created_at DESC LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, limit, offset)

	var policies []Policy
	if err := r.db.SelectContext(ctx, &policies, baseQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("policy.List: %w", err)
	}

	for i := range policies {
		if err := r.unmarshalPolicyFields(&policies[i]); err != nil {
			return nil, 0, err
		}
	}

	return policies, total, nil
}

// GetActiveForTenant retrieves all active policies for a tenant
func (r *Repository) GetActiveForTenant(ctx context.Context, tenantID uuid.UUID) ([]Policy, error) {
	var policies []Policy
	query := `
		SELECT * FROM policies
		WHERE tenant_id = $1 AND enabled = true AND deleted_at IS NULL
		ORDER BY priority ASC
	`
	err := r.db.SelectContext(ctx, &policies, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("policy.GetActiveForTenant: %w", err)
	}

	for i := range policies {
		if err := r.unmarshalPolicyFields(&policies[i]); err != nil {
			return nil, err
		}
	}

	return policies, nil
}

// GetApplicablePolicies retrieves policies that apply to a specific user/resource context
func (r *Repository) GetApplicablePolicies(ctx context.Context, tenantID, userID uuid.UUID, userRoles []uuid.UUID, resourceID uuid.UUID) ([]Policy, error) {
	var policies []Policy

	// Get policies that match:
	// 1. Apply to all users (user_ids is empty or contains null)
	// 2. Apply to this user specifically
	// 3. Apply to this user's roles
	// 4. Apply to this resource (if specified)
	query := `
		SELECT DISTINCT p.* FROM policies p
		WHERE p.tenant_id = $1
			AND p.enabled = true
			AND p.deleted_at IS NULL
			AND (
				p.user_ids = '[]'::jsonb
				OR p.user_ids @> $2::jsonb
				OR p.role_ids && $3::jsonb
			)
		ORDER BY p.priority ASC
	`

	userIDJSON, _ := json.Marshal([]uuid.UUID{userID})
	roleIdsJSON, _ := json.Marshal(userRoles)

	err := r.db.SelectContext(ctx, &policies, query, tenantID, userIDJSON, roleIdsJSON)
	if err != nil {
		return nil, fmt.Errorf("policy.GetApplicablePolicies: %w", err)
	}

	// Further filter by target/credential if resource ID is provided
	var filtered []Policy
	for _, p := range policies {
		if err := r.unmarshalPolicyFields(&p); err != nil {
			return nil, err
		}

		// If resource ID provided, check if policy applies to it
		if resourceID != uuid.Nil {
			applies := len(p.TargetIDs) == 0 && len(p.CredentialIDs) == 0
			for _, tid := range p.TargetIDs {
				if tid == resourceID {
					applies = true
					break
				}
			}
			for _, cid := range p.CredentialIDs {
				if cid == resourceID {
					applies = true
					break
				}
			}
			if !applies {
				continue
			}
		}
		filtered = append(filtered, p)
	}

	return filtered, nil
}

// Update updates a policy
func (r *Repository) Update(ctx context.Context, policy *Policy) error {
	policy.UpdatedAt = time.Now()

	// Marshal JSONB fields
	userIDsJSON, err := json.Marshal(policy.UserIDs)
	if err != nil {
		return fmt.Errorf("policy.Update.MarshalUserIDs: %w", err)
	}
	roleIDsJSON, err := json.Marshal(policy.RoleIDs)
	if err != nil {
		return fmt.Errorf("policy.Update.MarshalRoleIDs: %w", err)
	}
	groupIDsJSON, err := json.Marshal(policy.GroupIDs)
	if err != nil {
		return fmt.Errorf("policy.Update.MarshalGroupIDs: %w", err)
	}
	targetIDsJSON, err := json.Marshal(policy.TargetIDs)
	if err != nil {
		return fmt.Errorf("policy.Update.MarshalTargetIDs: %w", err)
	}
	credentialIDsJSON, err := json.Marshal(policy.CredentialIDs)
	if err != nil {
		return fmt.Errorf("policy.Update.MarshalCredentialIDs: %w", err)
	}
	rulesJSON, err := json.Marshal(policy.Rules)
	if err != nil {
		return fmt.Errorf("policy.Update.MarshalRules: %w", err)
	}
	mfaMethodsJSON, err := json.Marshal(policy.MFAMethods)
	if err != nil {
		return fmt.Errorf("policy.Update.MarshalMFAMethods: %w", err)
	}
	approvalApproversJSON, err := json.Marshal(policy.ApprovalApprovers)
	if err != nil {
		return fmt.Errorf("policy.Update.MarshalApprovalApprovers: %w", err)
	}
	metadataJSON, err := json.Marshal(policy.Metadata)
	if err != nil {
		return fmt.Errorf("policy.Update.MarshalMetadata: %w", err)
	}
	tagsJSON, err := json.Marshal(policy.Tags)
	if err != nil {
		return fmt.Errorf("policy.Update.MarshalTags: %w", err)
	}

	query := `
		UPDATE policies SET
			name = :name,
			description = :description,
			type = :type,
			effect = :effect,
			priority = :priority,
			user_ids = :user_ids,
			role_ids = :role_ids,
			group_ids = :group_ids,
			target_ids = :target_ids,
			credential_ids = :credential_ids,
			rules = :rules,
			max_session_duration_seconds = :max_session_duration_seconds,
			session_extension_allowed = :session_extension_allowed,
			max_extensions = :max_extensions,
			mfa_required = :mfa_required,
			mfa_methods = :mfa_methods,
			approval_required = :approval_required,
			approval_approvers = :approval_approvers,
			approval_timeout_minutes = :approval_timeout_minutes,
			recording_required = :recording_required,
			recording_mode = :recording_mode,
			enabled = :enabled,
			metadata = :metadata,
			tags = :tags,
			updated_by = :updated_by,
			updated_at = :updated_at
		WHERE id = :id AND tenant_id = :tenant_id AND deleted_at IS NULL
	`

	args := map[string]interface{}{
		"id":                            policy.ID,
		"name":                          policy.Name,
		"description":                   policy.Description,
		"type":                          policy.Type,
		"effect":                        policy.Effect,
		"priority":                      policy.Priority,
		"tenant_id":                     policy.TenantID,
		"user_ids":                      userIDsJSON,
		"role_ids":                      roleIDsJSON,
		"group_ids":                     groupIDsJSON,
		"target_ids":                    targetIDsJSON,
		"credential_ids":                credentialIDsJSON,
		"rules":                         rulesJSON,
		"max_session_duration_seconds":  policy.MaxSessionDurationSeconds,
		"session_extension_allowed":     policy.SessionExtensionAllowed,
		"max_extensions":                policy.MaxExtensions,
		"mfa_required":                  policy.MFARequired,
		"mfa_methods":                   mfaMethodsJSON,
		"approval_required":             policy.ApprovalRequired,
		"approval_approvers":            approvalApproversJSON,
		"approval_timeout_minutes":      policy.ApprovalTimeoutMinutes,
		"recording_required":            policy.RecordingRequired,
		"recording_mode":                policy.RecordingMode,
		"enabled":                       policy.Enabled,
		"metadata":                      metadataJSON,
		"tags":                          tagsJSON,
		"updated_by":                    policy.UpdatedBy,
		"updated_at":                    policy.UpdatedAt,
	}

	result, err := r.db.NamedExecContext(ctx, query, args)
	if err != nil {
		return fmt.Errorf("policy.Update: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("policy.Update: no rows affected (policy may not exist)")
	}

	return nil
}

// Delete soft deletes a policy
func (r *Repository) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	query := `
		UPDATE policies
		SET deleted_at = NOW()
		WHERE id = $1 AND tenant_id = $2 AND system_policy = false
	`

	result, err := r.db.ExecContext(ctx, query, id, tenantID)
	if err != nil {
		return fmt.Errorf("policy.Delete: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("policy.Delete: no rows affected (policy may not exist or is a system policy)")
	}

	return nil
}

// Enable enables a policy
func (r *Repository) Enable(ctx context.Context, id, tenantID uuid.UUID) error {
	query := `
		UPDATE policies
		SET enabled = true, updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, id, tenantID)
	if err != nil {
		return fmt.Errorf("policy.Enable: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("policy.Enable: no rows affected")
	}

	return nil
}

// Disable disables a policy
func (r *Repository) Disable(ctx context.Context, id, tenantID uuid.UUID) error {
	query := `
		UPDATE policies
		SET enabled = false, updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2 AND system_policy = false
	`

	result, err := r.db.ExecContext(ctx, query, id, tenantID)
	if err != nil {
		return fmt.Errorf("policy.Disable: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("policy.Disable: no rows affected")
	}

	return nil
}

// CreateEvaluationLog creates a policy evaluation log entry
func (r *Repository) CreateEvaluationLog(ctx context.Context, log *PolicyEvalLog) error {
	log.ID = uuid.New()
	log.CreatedAt = time.Now()

	denialReasonsJSON, err := json.Marshal(log.DenialReasons)
	if err != nil {
		return fmt.Errorf("policy.CreateEvalLog.MarshalDenialReasons: %w", err)
	}

	matchedRulesJSON, err := json.Marshal(log.MatchedRules)
	if err != nil {
		return fmt.Errorf("policy.CreateEvalLog.MarshalMatchedRules: %w", err)
	}

	query := `
		INSERT INTO policy_evaluations (
			id, tenant_id, policy_id, user_id, target_type, target_id, action,
			result, denial_reasons, matched_rules, client_ip, user_agent,
			session_id, evaluation_duration_ms, created_at
		) VALUES (
			:id, :tenant_id, :policy_id, :user_id, :target_type, :target_id, :action,
			:result, :denial_reasons, :matched_rules, :client_ip, :user_agent,
			:session_id, :evaluation_duration_ms, :created_at
		)
	`

	args := map[string]interface{}{
		"id":                         log.ID,
		"tenant_id":                  log.TenantID,
		"policy_id":                  log.PolicyID,
		"user_id":                    log.UserID,
		"target_type":                log.TargetType,
		"target_id":                  log.TargetID,
		"action":                     log.Action,
		"result":                     log.Result,
		"denial_reasons":             denialReasonsJSON,
		"matched_rules":              matchedRulesJSON,
		"client_ip":                  log.ClientIP,
		"user_agent":                 log.UserAgent,
		"session_id":                 log.SessionID,
		"evaluation_duration_ms":     log.EvaluationDurationMs,
		"created_at":                 log.CreatedAt,
	}

	_, err = r.db.NamedExecContext(ctx, query, args)
	if err != nil {
		return fmt.Errorf("policy.CreateEvalLog: %w", err)
	}

	return nil
}

// GetEvaluationLogs retrieves evaluation logs with pagination
func (r *Repository) GetEvaluationLogs(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, limit, offset int) ([]PolicyEvalLog, int, error) {
	baseQuery := `
		SELECT * FROM policy_evaluations
		WHERE tenant_id = $1
	`
	countQuery := `
		SELECT COUNT(*) FROM policy_evaluations
		WHERE tenant_id = $1
	`

	args := []interface{}{tenantID}
	argCount := 2

	if userID != nil {
		baseQuery += fmt.Sprintf(" AND user_id = $%d", argCount)
		countQuery += fmt.Sprintf(" AND user_id = $%d", argCount)
		args = append(args, *userID)
		argCount++
	}

	// Get count
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("policy.GetEvaluationLogs.Count: %w", err)
	}

	// Add pagination
	baseQuery += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, limit, offset)

	var logs []PolicyEvalLog
	if err := r.db.SelectContext(ctx, &logs, baseQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("policy.GetEvaluationLogs: %w", err)
	}

	for i := range logs {
		if err := r.unmarshalEvalLogFields(&logs[i]); err != nil {
			return nil, 0, err
		}
	}

	return logs, total, nil
}

// CreateApprovalRequest creates an approval request
func (r *Repository) CreateApprovalRequest(ctx context.Context, req *ApprovalRequest) error {
	req.ID = uuid.New()
	req.CreatedAt = time.Now()
	req.UpdatedAt = req.CreatedAt
	req.Status = ApprovalStatusPending

	metadataJSON, err := json.Marshal(req.Metadata)
	if err != nil {
		return fmt.Errorf("policy.CreateApprovalRequest.MarshalMetadata: %w", err)
	}

	query := `
		INSERT INTO approval_requests (
			id, policy_id, requester_id, tenant_id, target_type, target_id, reason,
			requested_duration_seconds, requested_start_time, requested_end_time,
			status, expires_at, metadata, created_at, updated_at
		) VALUES (
			:id, :policy_id, :requester_id, :tenant_id, :target_type, :target_id, :reason,
			:requested_duration_seconds, :requested_start_time, :requested_end_time,
			:status, :expires_at, :metadata, :created_at, :updated_at
		)
	`

	args := map[string]interface{}{
		"id":                         req.ID,
		"policy_id":                  req.PolicyID,
		"requester_id":               req.RequesterID,
		"tenant_id":                  req.TenantID,
		"target_type":                req.TargetType,
		"target_id":                  req.TargetID,
		"reason":                     req.Reason,
		"requested_duration_seconds": req.RequestedDurationSeconds,
		"requested_start_time":       req.RequestedStartTime,
		"requested_end_time":         req.RequestedEndTime,
		"status":                     req.Status,
		"expires_at":                 req.ExpiresAt,
		"metadata":                   metadataJSON,
		"created_at":                 req.CreatedAt,
		"updated_at":                 req.UpdatedAt,
	}

	_, err = r.db.NamedExecContext(ctx, query, args)
	if err != nil {
		return fmt.Errorf("policy.CreateApprovalRequest: %w", err)
	}

	return nil
}

// GetApprovalRequestByID retrieves an approval request by ID
func (r *Repository) GetApprovalRequestByID(ctx context.Context, id uuid.UUID) (*ApprovalRequest, error) {
	var req ApprovalRequest
	query := `SELECT * FROM approval_requests WHERE id = $1`
	err := r.db.GetContext(ctx, &req, query, id)
	if err != nil {
		return nil, fmt.Errorf("policy.GetApprovalRequestByID: %w", err)
	}

	if err := r.unmarshalApprovalRequestFields(&req); err != nil {
		return nil, err
	}

	return &req, nil
}

// UpdateApprovalRequestStatus updates the status of an approval request
func (r *Repository) UpdateApprovalRequestStatus(ctx context.Context, id uuid.UUID, status ApprovalStatus, approverID *uuid.UUID, denialReason string) error {
	query := `
		UPDATE approval_requests
		SET status = $1, approver_id = $2, approved_at = CASE WHEN $1 = 'approved' THEN NOW() ELSE approved_at END,
		    denial_reason = $3, updated_at = NOW()
		WHERE id = $4 AND status = 'pending'
	`

	result, err := r.db.ExecContext(ctx, query, status, approverID, denialReason, id)
	if err != nil {
		return fmt.Errorf("policy.UpdateApprovalRequestStatus: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("policy.UpdateApprovalRequestStatus: no rows affected")
	}

	return nil
}

// GetPendingApprovalRequests retrieves pending approval requests for a tenant
func (r *Repository) GetPendingApprovalRequests(ctx context.Context, tenantID uuid.UUID, approverID *uuid.UUID) ([]ApprovalRequest, error) {
	var reqs []ApprovalRequest
	query := `
		SELECT * FROM approval_requests
		WHERE tenant_id = $1 AND status = 'pending' AND expires_at > NOW()
		ORDER BY created_at ASC
	`
	err := r.db.SelectContext(ctx, &reqs, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("policy.GetPendingApprovalRequests: %w", err)
	}

	for i := range reqs {
		if err := r.unmarshalApprovalRequestFields(&reqs[i]); err != nil {
			return nil, err
		}
	}

	return reqs, nil
}

// CreatePolicyTemplate creates a policy template
func (r *Repository) CreatePolicyTemplate(ctx context.Context, template *PolicyTemplate) error {
	template.ID = uuid.New()
	template.CreatedAt = time.Now()
	template.UpdatedAt = template.CreatedAt

	templatePolicyJSON, err := json.Marshal(template.TemplatePolicy)
	if err != nil {
		return fmt.Errorf("policy.CreatePolicyTemplate.MarshalTemplatePolicy: %w", err)
	}

	tagsJSON, err := json.Marshal(template.Tags)
	if err != nil {
		return fmt.Errorf("policy.CreatePolicyTemplate.MarshalTags: %w", err)
	}

	query := `
		INSERT INTO policy_templates (id, name, description, category, template_policy, is_system, tags, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err = r.db.ExecContext(ctx, query, template.ID, template.Name, template.Description,
		template.Category, templatePolicyJSON, template.IsSystem, tagsJSON, template.CreatedAt, template.UpdatedAt)
	if err != nil {
		return fmt.Errorf("policy.CreatePolicyTemplate: %w", err)
	}

	return nil
}

// GetPolicyTemplates retrieves all policy templates
func (r *Repository) GetPolicyTemplates(ctx context.Context, category *string) ([]PolicyTemplate, error) {
	var templates []PolicyTemplate
	var query string
	var err error

	if category != nil {
		query = `SELECT * FROM policy_templates WHERE category = $1 ORDER BY name`
		err = r.db.SelectContext(ctx, &templates, query, *category)
	} else {
		query = `SELECT * FROM policy_templates ORDER BY category, name`
		err = r.db.SelectContext(ctx, &templates, query)
	}

	if err != nil {
		return nil, fmt.Errorf("policy.GetPolicyTemplates: %w", err)
	}

	for i := range templates {
		if err := r.unmarshalPolicyTemplateFields(&templates[i]); err != nil {
			return nil, err
		}
	}

	return templates, nil
}

// CreateCommandFilterPattern creates a command filter pattern
func (r *Repository) CreateCommandFilterPattern(ctx context.Context, pattern *CommandFilterPattern) error {
	pattern.ID = uuid.New()
	pattern.CreatedAt = time.Now()

	query := `
		INSERT INTO command_filter_patterns (id, policy_id, pattern, is_whitelist, description, compiled_hash, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (policy_id, pattern) DO NOTHING
	`

	_, err := r.db.ExecContext(ctx, query, pattern.ID, pattern.PolicyID, pattern.Pattern,
		pattern.IsWhitelist, pattern.Description, pattern.CompiledHash, pattern.CreatedAt)
	if err != nil {
		return fmt.Errorf("policy.CreateCommandFilterPattern: %w", err)
	}

	return nil
}

// GetCommandFilterPatterns retrieves patterns for a policy
func (r *Repository) GetCommandFilterPatterns(ctx context.Context, policyID uuid.UUID) ([]CommandFilterPattern, error) {
	var patterns []CommandFilterPattern
	query := `SELECT * FROM command_filter_patterns WHERE policy_id = $1 ORDER BY pattern`
	err := r.db.SelectContext(ctx, &patterns, query, policyID)
	if err != nil {
		return nil, fmt.Errorf("policy.GetCommandFilterPatterns: %w", err)
	}
	return patterns, nil
}

// Helper methods for JSONB unmarshaling

func (r *Repository) unmarshalPolicyFields(policy *Policy) error {
	if err := json.Unmarshal(policy.UserIDsRaw, &policy.UserIDs); err != nil && policy.UserIDsRaw != nil {
		return fmt.Errorf("unmarshal UserIDs: %w", err)
	}
	if err := json.Unmarshal(policy.RoleIDsRaw, &policy.RoleIDs); err != nil && policy.RoleIDsRaw != nil {
		return fmt.Errorf("unmarshal RoleIDs: %w", err)
	}
	if err := json.Unmarshal(policy.GroupIDsRaw, &policy.GroupIDs); err != nil && policy.GroupIDsRaw != nil {
		return fmt.Errorf("unmarshal GroupIDs: %w", err)
	}
	if err := json.Unmarshal(policy.TargetIDsRaw, &policy.TargetIDs); err != nil && policy.TargetIDsRaw != nil {
		return fmt.Errorf("unmarshal TargetIDs: %w", err)
	}
	if err := json.Unmarshal(policy.CredentialIDsRaw, &policy.CredentialIDs); err != nil && policy.CredentialIDsRaw != nil {
		return fmt.Errorf("unmarshal CredentialIDs: %w", err)
	}
	if err := json.Unmarshal(policy.RulesRaw, &policy.Rules); err != nil && policy.RulesRaw != nil {
		return fmt.Errorf("unmarshal Rules: %w", err)
	}
	if err := json.Unmarshal(policy.MFAMethodsRaw, &policy.MFAMethods); err != nil && policy.MFAMethodsRaw != nil {
		return fmt.Errorf("unmarshal MFAMethods: %w", err)
	}
	if err := json.Unmarshal(policy.ApprovalApproversRaw, &policy.ApprovalApprovers); err != nil && policy.ApprovalApproversRaw != nil {
		return fmt.Errorf("unmarshal ApprovalApprovers: %w", err)
	}
	if err := json.Unmarshal(policy.MetadataRaw, &policy.Metadata); err != nil && policy.MetadataRaw != nil {
		return fmt.Errorf("unmarshal Metadata: %w", err)
	}
	if err := json.Unmarshal(policy.TagsRaw, &policy.Tags); err != nil && policy.TagsRaw != nil {
		return fmt.Errorf("unmarshal Tags: %w", err)
	}
	return nil
}

func (r *Repository) unmarshalEvalLogFields(log *PolicyEvalLog) error {
	if err := json.Unmarshal(log.DenialReasonsRaw, &log.DenialReasons); err != nil && log.DenialReasonsRaw != nil {
		return fmt.Errorf("unmarshal DenialReasons: %w", err)
	}
	if err := json.Unmarshal(log.MatchedRulesRaw, &log.MatchedRules); err != nil && log.MatchedRulesRaw != nil {
		return fmt.Errorf("unmarshal MatchedRules: %w", err)
	}
	return nil
}

func (r *Repository) unmarshalApprovalRequestFields(req *ApprovalRequest) error {
	if err := json.Unmarshal(req.MetadataRaw, &req.Metadata); err != nil && req.MetadataRaw != nil {
		return fmt.Errorf("unmarshal Metadata: %w", err)
	}
	return nil
}

func (r *Repository) unmarshalPolicyTemplateFields(template *PolicyTemplate) error {
	if err := json.Unmarshal(template.TemplatePolicyRaw, &template.TemplatePolicy); err != nil && template.TemplatePolicyRaw != nil {
		return fmt.Errorf("unmarshal TemplatePolicy: %w", err)
	}
	if err := json.Unmarshal(template.TagsRaw, &template.Tags); err != nil && template.TagsRaw != nil {
		return fmt.Errorf("unmarshal Tags: %w", err)
	}
	return nil
}

// PolicyDB represents the raw database structure for policies
type PolicyDB struct {
	ID                        uuid.UUID    `db:"id"`
	Name                      string       `db:"name"`
	Description               string       `db:"description"`
	Type                      PolicyType   `db:"type"`
	Effect                    PolicyEffect `db:"effect"`
	Priority                  int          `db:"priority"`
	TenantID                  uuid.UUID    `db:"tenant_id"`
	UserIDsRaw                []byte       `db:"user_ids"`
	RoleIDsRaw                []byte       `db:"role_ids"`
	GroupIDsRaw               []byte       `db:"group_ids"`
	TargetIDsRaw              []byte       `db:"target_ids"`
	CredentialIDsRaw          []byte       `db:"credential_ids"`
	RulesRaw                  []byte       `db:"rules"`
	MaxSessionDurationSeconds *int         `db:"max_session_duration_seconds"`
	SessionExtensionAllowed   bool         `db:"session_extension_allowed"`
	MaxExtensions             *int         `db:"max_extensions"`
	MFARequired               bool         `db:"mfa_required"`
	MFAMethodsRaw             []byte       `db:"mfa_methods"`
	ApprovalRequired          bool         `db:"approval_required"`
	ApprovalApproversRaw      []byte       `db:"approval_approvers"`
	ApprovalTimeoutMinutes    int          `db:"approval_timeout_minutes"`
	RecordingRequired         bool         `db:"recording_required"`
	RecordingMode             RecordingMode `db:"recording_mode"`
	Enabled                   bool         `db:"enabled"`
	SystemPolicy              bool         `db:"system_policy"`
	MetadataRaw               []byte       `db:"metadata"`
	TagsRaw                   []byte       `db:"tags"`
	Version                   int          `db:"version"`
	CreatedBy                 uuid.UUID    `db:"created_by"`
	UpdatedBy                 uuid.UUID    `db:"updated_by"`
	CreatedAt                 time.Time    `db:"created_at"`
	UpdatedAt                 time.Time    `db:"updated_at"`
	DeletedAt                 *time.Time   `db:"deleted_at"`
}

// Scan implements sql.Scanner for Policy
func (p *Policy) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("policy.Scan: expected []byte, got %T", value)
	}
	return json.Unmarshal(bytes, p)
}
