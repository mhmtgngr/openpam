package policy

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// PolicyType represents the type/category of a policy
type PolicyType string

const (
	PolicyTypeAccess         PolicyType = "access"
	PolicyTypeSession        PolicyType = "session"
	PolicyTypeCredential     PolicyType = "credential"
	PolicyTypeApproval       PolicyType = "approval"
	PolicyTypeCompliance     PolicyType = "compliance"
	PolicyTypeCommandFilter  PolicyType = "command_filter"
)

// PolicyTypeFromString converts a string to PolicyType
func PolicyTypeFromString(s string) (PolicyType, error) {
	switch strings.ToLower(s) {
	case "access":
		return PolicyTypeAccess, nil
	case "session":
		return PolicyTypeSession, nil
	case "credential":
		return PolicyTypeCredential, nil
	case "approval":
		return PolicyTypeApproval, nil
	case "compliance":
		return PolicyTypeCompliance, nil
	case "command_filter", "command-filter", "commandfilter":
		return PolicyTypeCommandFilter, nil
	default:
		return "", fmt.Errorf("policy: invalid policy type '%s'", s)
	}
}

// String returns the string representation
func (p PolicyType) String() string {
	return string(p)
}

// IsValid checks if the policy type is valid
func (p PolicyType) IsValid() bool {
	switch p {
	case PolicyTypeAccess, PolicyTypeSession, PolicyTypeCredential,
		PolicyTypeApproval, PolicyTypeCompliance, PolicyTypeCommandFilter:
		return true
	}
	return false
}

// MarshalJSON implements json.Marshaler
func (p PolicyType) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(p))
}

// UnmarshalJSON implements json.Unmarshaler
func (p *PolicyType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	pt, err := PolicyTypeFromString(s)
	if err != nil {
		return err
	}
	*p = pt
	return nil
}

// PolicyEffect represents the effect of a policy (allow or deny)
type PolicyEffect string

const (
	PolicyEffectAllow PolicyEffect = "allow"
	PolicyEffectDeny  PolicyEffect = "deny"
)

// EffectFromString converts a string to PolicyEffect
func EffectFromString(s string) (PolicyEffect, error) {
	switch strings.ToLower(s) {
	case "allow":
		return PolicyEffectAllow, nil
	case "deny":
		return PolicyEffectDeny, nil
	default:
		return "", fmt.Errorf("policy: invalid effect '%s', must be 'allow' or 'deny'", s)
	}
}

// String returns the string representation
func (p PolicyEffect) String() string {
	return string(p)
}

// IsAllow returns true if the effect is allow
func (p PolicyEffect) IsAllow() bool {
	return p == PolicyEffectAllow
}

// IsDeny returns true if the effect is deny
func (p PolicyEffect) IsDeny() bool {
	return p == PolicyEffectDeny
}

// MarshalJSON implements json.Marshaler
func (p PolicyEffect) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(p))
}

// UnmarshalJSON implements json.Unmarshaler
func (p *PolicyEffect) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	effect, err := EffectFromString(s)
	if err != nil {
		return err
	}
	*p = effect
	return nil
}

// RuleType represents the type of a rule within a policy
type RuleType string

const (
	RuleTypeTimeRestriction     RuleType = "time_restriction"
	RuleTypeIPRestriction       RuleType = "ip_restriction"
	RuleTypeMFARequirement      RuleType = "mfa_requirement"
	RuleTypeCommandFilter       RuleType = "command_filter"
	RuleTypeSessionRecording    RuleType = "session_recording"
	RuleTypeApprovalRequirement RuleType = "approval_requirement"
	RuleTypeDurationLimit       RuleType = "duration_limit"
	RuleTypeConcurrentLimit     RuleType = "concurrent_limit"
)

// RuleTypeFromString converts a string to RuleType
func RuleTypeFromString(s string) (RuleType, error) {
	switch strings.ToLower(strings.ReplaceAll(s, "-", "_")) {
	case "time_restriction", "timerestriction", "time":
		return RuleTypeTimeRestriction, nil
	case "ip_restriction", "iprestriction", "ip":
		return RuleTypeIPRestriction, nil
	case "mfa_requirement", "mfarequirement", "mfa":
		return RuleTypeMFARequirement, nil
	case "command_filter", "commandfilter", "command":
		return RuleTypeCommandFilter, nil
	case "session_recording", "sessionrecording", "recording":
		return RuleTypeSessionRecording, nil
	case "approval_requirement", "approvalrequirement", "approval":
		return RuleTypeApprovalRequirement, nil
	case "duration_limit", "durationlimit", "duration":
		return RuleTypeDurationLimit, nil
	case "concurrent_limit", "concurrentlimit", "concurrent":
		return RuleTypeConcurrentLimit, nil
	default:
		return "", fmt.Errorf("policy: invalid rule type '%s'", s)
	}
}

// String returns the string representation
func (r RuleType) String() string {
	return string(r)
}

// MarshalJSON implements json.Marshaler
func (r RuleType) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(r))
}

// UnmarshalJSON implements json.Unmarshaler
func (r *RuleType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	rt, err := RuleTypeFromString(s)
	if err != nil {
		return err
	}
	*r = rt
	return nil
}

// ApprovalStatus represents the status of an approval request
type ApprovalStatus string

const (
	ApprovalStatusPending  ApprovalStatus = "pending"
	ApprovalStatusApproved ApprovalStatus = "approved"
	ApprovalStatusDenied   ApprovalStatus = "denied"
	ApprovalStatusCancelled ApprovalStatus = "cancelled"
	ApprovalStatusExpired  ApprovalStatus = "expired"
)

// ApprovalStatusFromString converts a string to ApprovalStatus
func ApprovalStatusFromString(s string) (ApprovalStatus, error) {
	switch strings.ToLower(s) {
	case "pending":
		return ApprovalStatusPending, nil
	case "approved", "approve":
		return ApprovalStatusApproved, nil
	case "denied", "deny", "rejected":
		return ApprovalStatusDenied, nil
	case "cancelled", "canceled", "cancel":
		return ApprovalStatusCancelled, nil
	case "expired":
		return ApprovalStatusExpired, nil
	default:
		return "", fmt.Errorf("policy: invalid approval status '%s'", s)
	}
}

// String returns the string representation
func (a ApprovalStatus) String() string {
	return string(a)
}

// IsPending returns true if the approval is pending
func (a ApprovalStatus) IsPending() bool {
	return a == ApprovalStatusPending
}

// IsApproved returns true if the approval is approved
func (a ApprovalStatus) IsApproved() bool {
	return a == ApprovalStatusApproved
}

// IsDenied returns true if the approval is denied
func (a ApprovalStatus) IsDenied() bool {
	return a == ApprovalStatusDenied
}

// IsFinal returns true if the approval has a final decision
func (a ApprovalStatus) IsFinal() bool {
	return a == ApprovalStatusApproved || a == ApprovalStatusDenied ||
		a == ApprovalStatusCancelled || a == ApprovalStatusExpired
}

// MarshalJSON implements json.Marshaler
func (a ApprovalStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(a))
}

// UnmarshalJSON implements json.Unmarshaler
func (a *ApprovalStatus) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	status, err := ApprovalStatusFromString(s)
	if err != nil {
		return err
	}
	*a = status
	return nil
}

// EvaluationResult represents the result of a policy evaluation
type EvaluationResult string

const (
	EvaluationResultAllow            EvaluationResult = "allow"
	EvaluationResultDeny             EvaluationResult = "deny"
	EvaluationResultApprovalRequired EvaluationResult = "approval_required"
)

// String returns the string representation
func (e EvaluationResult) String() string {
	return string(e)
}

// IsAllowed returns true if access is allowed
func (e EvaluationResult) IsAllowed() bool {
	return e == EvaluationResultAllow
}

// IsDenied returns true if access is denied
func (e EvaluationResult) IsDenied() bool {
	return e == EvaluationResultDeny
}

// RequiresApproval returns true if approval is required
func (e EvaluationResult) RequiresApproval() bool {
	return e == EvaluationResultApprovalRequired
}

// MarshalJSON implements json.Marshaler
func (e EvaluationResult) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(e))
}

// UnmarshalJSON implements json.Unmarshaler
func (e *EvaluationResult) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	switch s {
	case "allow":
		*e = EvaluationResultAllow
	case "deny":
		*e = EvaluationResultDeny
	case "approval_required":
		*e = EvaluationResultApprovalRequired
	default:
		return fmt.Errorf("policy: invalid evaluation result '%s'", s)
	}
	return nil
}

// Weekday represents a day of the week for time restrictions
type Weekday string

const (
	WeekdayMonday    Weekday = "monday"
	WeekdayTuesday   Weekday = "tuesday"
	WeekdayWednesday Weekday = "wednesday"
	WeekdayThursday  Weekday = "thursday"
	WeekdayFriday    Weekday = "friday"
	WeekdaySaturday  Weekday = "saturday"
	WeekdaySunday    Weekday = "sunday"
)

// WeekdayFromString converts a string to Weekday
func WeekdayFromString(s string) (Weekday, error) {
	switch strings.ToLower(s) {
	case "monday", "mon":
		return WeekdayMonday, nil
	case "tuesday", "tue":
		return WeekdayTuesday, nil
	case "wednesday", "wed":
		return WeekdayWednesday, nil
	case "thursday", "thu":
		return WeekdayThursday, nil
	case "friday", "fri":
		return WeekdayFriday, nil
	case "saturday", "sat":
		return WeekdaySaturday, nil
	case "sunday", "sun":
		return WeekdaySunday, nil
	default:
		return "", fmt.Errorf("policy: invalid weekday '%s'", s)
	}
}

// String returns the string representation
func (w Weekday) String() string {
	return string(w)
}

// MarshalJSON implements json.Marshaler
func (w Weekday) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(w))
}

// UnmarshalJSON implements json.Unmarshaler
func (w *Weekday) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	wd, err := WeekdayFromString(s)
	if err != nil {
		return err
	}
	*w = wd
	return nil
}

// RecordingMode represents how sessions should be recorded
type RecordingMode string

const (
	RecordingModeAll        RecordingMode = "all"
	RecordingModeOnCommand  RecordingMode = "on_command"
	RecordingModeNone       RecordingMode = "none"
)

// RecordingModeFromString converts a string to RecordingMode
func RecordingModeFromString(s string) (RecordingMode, error) {
	switch strings.ToLower(s) {
	case "all":
		return RecordingModeAll, nil
	case "on_command", "oncommand":
		return RecordingModeOnCommand, nil
	case "none":
		return RecordingModeNone, nil
	default:
		return "", fmt.Errorf("policy: invalid recording mode '%s'", s)
	}
}

// String returns the string representation
func (r RecordingMode) String() string {
	return string(r)
}

// ShouldRecord returns true if recording should happen
func (r RecordingMode) ShouldRecord() bool {
	return r == RecordingModeAll || r == RecordingModeOnCommand
}

// MarshalJSON implements json.Marshaler
func (r RecordingMode) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(r))
}

// UnmarshalJSON implements json.Unmarshaler
func (r *RecordingMode) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	rm, err := RecordingModeFromString(s)
	if err != nil {
		return err
	}
	*r = rm
	return nil
}

// MFAMethod represents an MFA method type
type MFAMethod string

const (
	MFAMethodTOTP        MFAMethod = "totp"
	MFAMethodPush        MFAMethod = "push"
	MFAMethodSMS         MFAMethod = "sms"
	MFAMethodHardwareKey MFAMethod = "hardware_key"
	MFAMethodEmail       MFAMethod = "email"
	MFAMethodWebAuthn    MFAMethod = "webauthn"
)

// MFAMethodFromString converts a string to MFAMethod
func MFAMethodFromString(s string) (MFAMethod, error) {
	switch strings.ToLower(s) {
	case "totp":
		return MFAMethodTOTP, nil
	case "push":
		return MFAMethodPush, nil
	case "sms":
		return MFAMethodSMS, nil
	case "hardware_key", "hardwarekey", "yubikey", "u2f":
		return MFAMethodHardwareKey, nil
	case "email":
		return MFAMethodEmail, nil
	case "webauthn", "fido2":
		return MFAMethodWebAuthn, nil
	default:
		return "", fmt.Errorf("policy: invalid MFA method '%s'", s)
	}
}

// String returns the string representation
func (m MFAMethod) String() string {
	return string(m)
}

// MarshalJSON implements json.Marshaler
func (m MFAMethod) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(m))
}

// UnmarshalJSON implements json.Unmarshaler
func (m *MFAMethod) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	method, err := MFAMethodFromString(s)
	if err != nil {
		return err
	}
	*m = method
	return nil
}

// ConditionType represents the type of condition in a rule
type ConditionType string

const (
	ConditionTypeTime        ConditionType = "time"
	ConditionTypeIP          ConditionType = "ip"
	ConditionTypeRole        ConditionType = "role"
	ConditionTypeResource    ConditionType = "resource"
	ConditionTypeUser        ConditionType = "user"
	ConditionTypeGroup       ConditionType = "group"
	ConditionTypeMFAMethod   ConditionType = "mfa_method"
	ConditionTypeMFATrust    ConditionType = "mfa_trust"
	ConditionTypeGeo         ConditionType = "geo"
	ConditionTypeDeviceTrust ConditionType = "device_trust"
)

// String returns the string representation
func (c ConditionType) String() string {
	return string(c)
}

// MarshalJSON implements json.Marshaler
func (c ConditionType) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(c))
}

// UnmarshalJSON implements json.Unmarshaler
func (c *ConditionType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*c = ConditionType(s)
	return nil
}

// LogicalOperator represents how conditions are combined
type LogicalOperator string

const (
	LogicalOperatorAND LogicalOperator = "and"
	LogicalOperatorOR  LogicalOperator = "or"
)

// String returns the string representation
func (l LogicalOperator) String() string {
	return string(l)
}

// MarshalJSON implements json.Marshaler
func (l LogicalOperator) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(l))
}

// UnmarshalJSON implements json.Unmarshaler
func (l *LogicalOperator) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	switch s {
	case "and", "AND":
		*l = LogicalOperatorAND
	case "or", "OR":
		*l = LogicalOperatorOR
	default:
		return fmt.Errorf("policy: invalid logical operator '%s'", s)
	}
	return nil
}

// Condition represents a single condition that can be evaluated
type Condition struct {
	ID       uuid.UUID       `json:"id" db:"id"`
	Type     ConditionType   `json:"type" db:"type"`
	Operator LogicalOperator `json:"operator" db:"operator"`
	Negate   bool            `json:"negate" db:"negate"`

	// Condition-specific data
	TimeData      *TimeCondition      `json:"time_data,omitempty"`
	IPData        *IPCondition        `json:"ip_data,omitempty"`
	RoleData      *RoleCondition      `json:"role_data,omitempty"`
	ResourceData  *ResourceCondition  `json:"resource_data,omitempty"`
	UserData      *UserCondition      `json:"user_data,omitempty"`
	GroupData     *GroupCondition     `json:"group_data,omitempty"`
	MFAMethodData *MFAMethodCondition `json:"mfa_method_data,omitempty"`
	MFATrustData  *MFATrustCondition  `json:"mfa_trust_data,omitempty"`
	GeoData       *GeoCondition       `json:"geo_data,omitempty"`
	DeviceData    *DeviceCondition    `json:"device_data,omitempty"`
}

// TimeCondition represents time-based access restrictions
type TimeCondition struct {
	DaysOfWeek []Weekday `json:"days_of_week"`
	StartTime  string    `json:"start_time"` // HH:MM format
	EndTime    string    `json:"end_time"`   // HH:MM format
	Timezone   string    `json:"timezone"`   // IANA timezone (e.g., "America/New_York")
	DateRange  *DateRange `json:"date_range,omitempty"`
}

// DateRange represents a specific date range
type DateRange struct {
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
}

// IPCondition represents IP-based restrictions
type IPCondition struct {
	CIDRs      []string `json:"cidrs"`       // e.g., ["192.168.1.0/24", "10.0.0.0/16"]
	IPRanges   []IPRange `json:"ip_ranges"`  // For specific ranges
	Countries  []string `json:"countries"`   // ISO country codes
	Source     string   `json:"source"`      // "client_ip", "forwarded_for", etc.
}

// IPRange represents a start and end IP
type IPRange struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// RoleCondition represents role-based conditions
type RoleCondition struct {
	RoleIDs []uuid.UUID `json:"role_ids"`
	Roles   []string    `json:"roles"` // Role names
	Match   string      `json:"match"` // "any", "all", "exact"
}

// ResourceCondition represents resource/target-based conditions
type ResourceCondition struct {
	ResourceIDs []uuid.UUID `json:"resource_ids"`
	Targets     []uuid.UUID `json:"targets"`
	Credentials []uuid.UUID `json:"credentials"`
	Tags        []string    `json:"tags"`
	Type        string      `json:"type"` // "ssh", "rdp", "database", etc.
}

// UserCondition represents user-based conditions
type UserCondition struct {
	UserIDs   []uuid.UUID `json:"user_ids"`
	Emails    []string    `json:"emails"`
	Attribute string      `json:"attribute"` // For custom attributes
	Value     string      `json:"value"`
}

// GroupCondition represents group-based conditions
type GroupCondition struct {
	GroupIDs []uuid.UUID `json:"group_ids"`
	Groups   []string    `json:"groups"` // Group names
}

// MFAMethodCondition represents MFA method requirements
type MFAMethodCondition struct {
	Methods []MFAMethod `json:"methods"`
	Match   string      `json:"match"` // "any", "all"
}

// MFATrustCondition represents MFA trust levels
type MFATrustCondition struct {
	MinimumTrustLevel string    `json:"minimum_trust_level"` // "low", "medium", "high"
	MaxTrustDuration  int       `json:"max_trust_duration"`   // In minutes
	LastVerifiedAfter *time.Time `json:"last_verified_after,omitempty"`
}

// GeoCondition represents geographical restrictions
type GeoCondition struct {
	Countries     []string `json:"countries"`      // ISO country codes
	Regions       []string `json:"regions"`        // e.g., "US-West", "EU"
	ExcludeVPN    bool     `json:"exclude_vpn"`
	ExcludeProxy  bool     `json:"exclude_proxy"`
}

// DeviceCondition represents device trust conditions
type DeviceCondition struct {
	Managed         bool     `json:"managed"`
	MinimumTrustScore int    `json:"minimum_trust_score"`
	AllowedPlatforms []string `json:"allowed_platforms"` // "windows", "macos", "linux", "ios", "android"
}

// Rule represents a policy rule with multiple conditions
type Rule struct {
	ID          uuid.UUID       `json:"id" db:"id"`
	Name        string          `json:"name" db:"name"`
	Description string          `json:"description" db:"description"`
	Type        RuleType        `json:"type" db:"type"`
	Operator    LogicalOperator `json:"operator" db:"operator"` // How to combine conditions
	Conditions  []Condition     `json:"conditions" db:"conditions"`
	Action      string          `json:"action" db:"action"` // "allow", "deny", "require_approval", "require_mfa"
	Priority    int             `json:"priority" db:"priority"`
	Enabled     bool            `json:"enabled" db:"enabled"`
}

// Policy represents an access control policy
type Policy struct {
	ID        uuid.UUID   `json:"id" db:"id"`
	Name      string     `json:"name" db:"name"`
	Description string   `json:"description" db:"description"`
	Type      PolicyType `json:"type" db:"type"`
	Effect    PolicyEffect `json:"effect" db:"effect"`
	Priority  int        `json:"priority" db:"priority"`

	// Policy scope - who/what this applies to
	TenantID     uuid.UUID   `json:"tenant_id" db:"tenant_id"`
	UserIDs      []uuid.UUID `json:"user_ids" db:"user_ids"`
	RoleIDs      []uuid.UUID `json:"role_ids" db:"role_ids"`
	GroupIDs     []uuid.UUID `json:"group_ids" db:"group_ids"`
	TargetIDs    []uuid.UUID `json:"target_ids" db:"target_ids"`
	CredentialIDs []uuid.UUID `json:"credential_ids" db:"credential_ids"`

	// Policy rules
	Rules []Rule `json:"rules" db:"rules"`

	// Duration settings
	MaxSessionDurationSeconds *int  `json:"max_session_duration_seconds,omitempty" db:"max_session_duration_seconds"`
	SessionExtensionAllowed   bool  `json:"session_extension_allowed" db:"session_extension_allowed"`
	MaxExtensions             *int  `json:"max_extensions,omitempty" db:"max_extensions"`

	// MFA settings
	MFARequired bool        `json:"mfa_required" db:"mfa_required"`
	MFAMethods  []MFAMethod `json:"mfa_methods" db:"mfa_methods"`

	// Approval settings
	ApprovalRequired    bool       `json:"approval_required" db:"approval_required"`
	ApprovalApprovers   []uuid.UUID `json:"approval_approvers" db:"approval_approvers"`
	ApprovalTimeoutMinutes int     `json:"approval_timeout_minutes" db:"approval_timeout_minutes"`

	// Session recording
	RecordingRequired bool          `json:"recording_required" db:"recording_required"`
	RecordingMode     RecordingMode `json:"recording_mode" db:"recording_mode"`

	// Status
	Enabled     bool      `json:"enabled" db:"enabled"`
	SystemPolicy bool     `json:"system_policy" db:"system_policy"`

	// Metadata
	Metadata     map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	Tags         []string               `json:"tags,omitempty" db:"tags"`
	Version      int                    `json:"version" db:"version"`

	// Audit
	CreatedBy   uuid.UUID  `json:"created_by" db:"created_by"`
	UpdatedBy   uuid.UUID  `json:"updated_by" db:"updated_by"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`

	// Raw JSONB fields for database scanning (not exported in JSON)
	UserIDsRaw              []byte `json:"-" db:"user_ids"`
	RoleIDsRaw              []byte `json:"-" db:"role_ids"`
	GroupIDsRaw             []byte `json:"-" db:"group_ids"`
	TargetIDsRaw            []byte `json:"-" db:"target_ids"`
	CredentialIDsRaw        []byte `json:"-" db:"credential_ids"`
	RulesRaw                []byte `json:"-" db:"rules"`
	MFAMethodsRaw           []byte `json:"-" db:"mfa_methods"`
	ApprovalApproversRaw    []byte `json:"-" db:"approval_approvers"`
	MetadataRaw             []byte `json:"-" db:"metadata"`
	TagsRaw                 []byte `json:"-" db:"tags"`
}

// PolicyEvalLog represents a policy evaluation log entry for audit
type PolicyEvalLog struct {
	ID            uuid.UUID        `json:"id" db:"id"`
	TenantID      uuid.UUID        `json:"tenant_id" db:"tenant_id"`
	PolicyID      *uuid.UUID       `json:"policy_id,omitempty" db:"policy_id"`

	// Request context
	UserID        uuid.UUID        `json:"user_id" db:"user_id"`
	TargetType    string           `json:"target_type" db:"target_type"`
	TargetID      *uuid.UUID       `json:"target_id,omitempty" db:"target_id"`
	Action        string           `json:"action" db:"action"`

	// Evaluation result
	Result        EvaluationResult `json:"result" db:"result"`
	DenialReasons []string         `json:"denial_reasons,omitempty" db:"denial_reasons"`
	MatchedRules  []MatchedRule    `json:"matched_rules,omitempty" db:"matched_rules"`

	// Context at evaluation time
	ClientIP      string           `json:"client_ip,omitempty" db:"client_ip"`
	UserAgent     string           `json:"user_agent,omitempty" db:"user_agent"`
	SessionID     *uuid.UUID       `json:"session_id,omitempty" db:"session_id"`

	// Performance tracking
	EvaluationDurationMs *int      `json:"evaluation_duration_ms,omitempty" db:"evaluation_duration_ms"`

	CreatedAt    time.Time        `json:"created_at" db:"created_at"`

	// Raw JSONB fields for database scanning
	DenialReasonsRaw []byte `json:"-" db:"denial_reasons"`
	MatchedRulesRaw  []byte `json:"-" db:"matched_rules"`
}

// ApprovalRequest represents an approval workflow request
type ApprovalRequest struct {
	ID              uuid.UUID       `json:"id" db:"id"`
	PolicyID        *uuid.UUID      `json:"policy_id,omitempty" db:"policy_id"`
	RequesterID     uuid.UUID       `json:"requester_id" db:"requester_id"`
	TenantID        uuid.UUID       `json:"tenant_id" db:"tenant_id"`

	// Request details
	TargetType      string          `json:"target_type" db:"target_type"` // session, credential, target
	TargetID        uuid.UUID       `json:"target_id" db:"target_id"`
	Reason          string          `json:"reason" db:"reason"`

	// Timing
	RequestedDurationSeconds *int      `json:"requested_duration_seconds,omitempty" db:"requested_duration_seconds"`
	RequestedStartTime       *time.Time `json:"requested_start_time,omitempty" db:"requested_start_time"`
	RequestedEndTime         *time.Time `json:"requested_end_time,omitempty" db:"requested_end_time"`

	// Status
	Status          ApprovalStatus  `json:"status" db:"status"`

	// Approval
	ApproverID      *uuid.UUID      `json:"approver_id,omitempty" db:"approver_id"`
	ApprovedAt      *time.Time      `json:"approved_at,omitempty" db:"approved_at"`
	DenialReason    string          `json:"denial_reason,omitempty" db:"denial_reason"`

	// Expiry
	ExpiresAt       time.Time       `json:"expires_at" db:"expires_at"`

	// Metadata
	Metadata        map[string]interface{} `json:"metadata,omitempty" db:"metadata"`

	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at" db:"updated_at"`

	// Raw JSONB fields for database scanning
	MetadataRaw     []byte `json:"-" db:"metadata"`
}

// PolicyTemplate represents a reusable policy template
type PolicyTemplate struct {
	ID          uuid.UUID              `json:"id" db:"id"`
	Name        string                 `json:"name" db:"name"`
	Description string                 `json:"description" db:"description"`
	Category    string                 `json:"category" db:"category"`
	TemplatePolicy map[string]interface{} `json:"template_policy" db:"template_policy"`
	IsSystem    bool                   `json:"is_system" db:"is_system"`
	Tags        []string               `json:"tags" db:"tags"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`

	// Raw JSONB fields for database scanning
	TemplatePolicyRaw []byte `json:"-" db:"template_policy"`
	TagsRaw           []byte `json:"-" db:"tags"`
}

// PolicyFilter filters policy queries
type PolicyFilter struct {
	Type      *PolicyType
	Effect    *PolicyEffect
	Enabled   *bool
	UserID    *uuid.UUID
	RoleID    *uuid.UUID
	TargetID  *uuid.UUID
	Tags      []string
	Search    string
}

// EvaluationRequest represents a request to evaluate policies
type EvaluationRequest struct {
	TenantID      uuid.UUID `json:"tenant_id"`
	UserID        uuid.UUID `json:"user_id"`
	UserRoles     []uuid.UUID `json:"user_roles"`
	UserGroups    []uuid.UUID `json:"user_groups"`
	Action        string    `json:"action"`         // checkout, extend, terminate, etc.
	ResourceType  string    `json:"resource_type"`  // session, credential, target
	ResourceID    uuid.UUID `json:"resource_id"`
	ClientIP      string    `json:"client_ip"`
	UserAgent     string    `json:"user_agent"`
	SessionID     *uuid.UUID `json:"session_id,omitempty"`
	Time          time.Time `json:"time"`
	MFAVerified   bool      `json:"mfa_verified"`
	MFAMethod     MFAMethod `json:"mfa_method"`
	DeviceTrust   int       `json:"device_trust"`
	Context       map[string]interface{} `json:"context,omitempty"`
}

// EvaluationResponse represents the result of policy evaluation
type EvaluationResponse struct {
	Effect          EvaluationResult `json:"effect"`
	MatchedPolicies []MatchedPolicy `json:"matched_policies"`
	DenialReasons   []string        `json:"denial_reasons,omitempty"`
	RequiredActions []RequiredAction `json:"required_actions,omitempty"`
	EvaluatedAt     time.Time       `json:"evaluated_at"`
	DurationMs      int64           `json:"duration_ms"`
}

// MatchedPolicy represents a policy that matched during evaluation
type MatchedPolicy struct {
	PolicyID   uuid.UUID       `json:"policy_id"`
	PolicyName string          `json:"policy_name"`
	Effect     PolicyEffect    `json:"effect"`
	Rules      []MatchedRule   `json:"matched_rules"`
	Priority   int             `json:"priority"`
}

// MatchedRule represents a rule that matched during evaluation
type MatchedRule struct {
	RuleID      uuid.UUID `json:"rule_id"`
	RuleName    string    `json:"rule_name"`
	Action      string    `json:"action"`
	Conditions  []uuid.UUID `json:"matched_conditions"`
}

// RequiredAction represents an action required by policy
type RequiredAction struct {
	Type        string `json:"type"` // mfa, approval, justification, etc.
	Description string `json:"description"`
	RequiredBy  uuid.UUID `json:"required_by"` // Policy ID
	Timeout     *int    `json:"timeout,omitempty"` // In seconds
}

// CommandFilterPattern represents a command filter pattern
type CommandFilterPattern struct {
	ID          uuid.UUID `json:"id" db:"id"`
	PolicyID    uuid.UUID `json:"policy_id" db:"policy_id"`
	Pattern     string    `json:"pattern" db:"pattern"`
	IsWhitelist bool      `json:"is_whitelist" db:"is_whitelist"`
	Description string    `json:"description,omitempty" db:"description"`
	CompiledHash string   `json:"compiled_hash" db:"compiled_hash"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// PolicyAssignment represents explicit user-policy mappings
type PolicyAssignment struct {
	ID         uuid.UUID              `json:"id" db:"id"`
	PolicyID   uuid.UUID              `json:"policy_id" db:"policy_id"`
	UserID     uuid.UUID              `json:"user_id" db:"user_id"`
	TenantID   uuid.UUID              `json:"tenant_id" db:"tenant_id"`
	Overrides  map[string]interface{} `json:"overrides,omitempty" db:"overrides"`
	CreatedAt  time.Time              `json:"created_at" db:"created_at"`
	CreatedBy  uuid.UUID              `json:"created_by" db:"created_by"`
}
