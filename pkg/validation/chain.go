package validation

import (
	"fmt"
	"net"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// ValidationError represents a validation failure
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationErrors collects multiple validation errors
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	msgs := make([]string, len(e))
	for i, err := range e {
		msgs[i] = err.Error()
	}
	return strings.Join(msgs, "; ")
}

// HasErrors returns true if there are any validation errors
func (e ValidationErrors) HasErrors() bool {
	return len(e) > 0
}

// Validator validates a specific field or constraint
type Validator interface {
	Validate() *ValidationError
}

// Chain implements Chain of Responsibility for request validation
// Validators are executed in order; all errors are collected
type Chain struct {
	validators []Validator
}

// NewChain creates a new validation chain
func NewChain() *Chain {
	return &Chain{}
}

// Add appends a validator to the chain
func (c *Chain) Add(v Validator) *Chain {
	c.validators = append(c.validators, v)
	return c
}

// Run executes all validators and returns collected errors
func (c *Chain) Run() ValidationErrors {
	var errors ValidationErrors
	for _, v := range c.validators {
		if err := v.Validate(); err != nil {
			errors = append(errors, *err)
		}
	}
	return errors
}

// --- Built-in Validators ---

// RequiredString validates that a string field is not empty
type RequiredString struct {
	Field string
	Value string
}

func (v RequiredString) Validate() *ValidationError {
	if strings.TrimSpace(v.Value) == "" {
		return &ValidationError{Field: v.Field, Message: "is required", Code: "REQUIRED"}
	}
	return nil
}

// MinLength validates minimum string length
type MinLength struct {
	Field  string
	Value  string
	Min    int
}

func (v MinLength) Validate() *ValidationError {
	if len(v.Value) < v.Min {
		return &ValidationError{
			Field:   v.Field,
			Message: fmt.Sprintf("must be at least %d characters", v.Min),
			Code:    "MIN_LENGTH",
		}
	}
	return nil
}

// MaxLength validates maximum string length
type MaxLength struct {
	Field string
	Value string
	Max   int
}

func (v MaxLength) Validate() *ValidationError {
	if len(v.Value) > v.Max {
		return &ValidationError{
			Field:   v.Field,
			Message: fmt.Sprintf("must be at most %d characters", v.Max),
			Code:    "MAX_LENGTH",
		}
	}
	return nil
}

// ValidUUID validates that a string is a valid UUID
type ValidUUID struct {
	Field string
	Value string
}

func (v ValidUUID) Validate() *ValidationError {
	if _, err := uuid.Parse(v.Value); err != nil {
		return &ValidationError{Field: v.Field, Message: "must be a valid UUID", Code: "INVALID_UUID"}
	}
	return nil
}

// ValidEmail validates email format
type ValidEmail struct {
	Field string
	Value string
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

func (v ValidEmail) Validate() *ValidationError {
	if !emailRegex.MatchString(v.Value) {
		return &ValidationError{Field: v.Field, Message: "must be a valid email address", Code: "INVALID_EMAIL"}
	}
	return nil
}

// ValidIP validates IP address format
type ValidIP struct {
	Field string
	Value string
}

func (v ValidIP) Validate() *ValidationError {
	if net.ParseIP(v.Value) == nil {
		return &ValidationError{Field: v.Field, Message: "must be a valid IP address", Code: "INVALID_IP"}
	}
	return nil
}

// ValidPort validates port number range
type ValidPort struct {
	Field string
	Value int
}

func (v ValidPort) Validate() *ValidationError {
	if v.Value < 1 || v.Value > 65535 {
		return &ValidationError{Field: v.Field, Message: "must be between 1 and 65535", Code: "INVALID_PORT"}
	}
	return nil
}

// OneOf validates that a value is one of the allowed values
type OneOf struct {
	Field   string
	Value   string
	Allowed []string
}

func (v OneOf) Validate() *ValidationError {
	for _, a := range v.Allowed {
		if v.Value == a {
			return nil
		}
	}
	return &ValidationError{
		Field:   v.Field,
		Message: fmt.Sprintf("must be one of: %s", strings.Join(v.Allowed, ", ")),
		Code:    "INVALID_VALUE",
	}
}

// PositiveInt validates that an integer is positive
type PositiveInt struct {
	Field string
	Value int
}

func (v PositiveInt) Validate() *ValidationError {
	if v.Value <= 0 {
		return &ValidationError{Field: v.Field, Message: "must be a positive number", Code: "INVALID_VALUE"}
	}
	return nil
}

// IntRange validates an integer is within a range
type IntRange struct {
	Field string
	Value int
	Min   int
	Max   int
}

func (v IntRange) Validate() *ValidationError {
	if v.Value < v.Min || v.Value > v.Max {
		return &ValidationError{
			Field:   v.Field,
			Message: fmt.Sprintf("must be between %d and %d", v.Min, v.Max),
			Code:    "OUT_OF_RANGE",
		}
	}
	return nil
}

// NoSQLInjection validates that a string doesn't contain SQL injection patterns
type NoSQLInjection struct {
	Field string
	Value string
}

var sqlInjectionPatterns = regexp.MustCompile(`(?i)(;|--|\b(DROP|DELETE|INSERT|UPDATE|ALTER|EXEC|UNION)\b)`)

func (v NoSQLInjection) Validate() *ValidationError {
	if sqlInjectionPatterns.MatchString(v.Value) {
		return &ValidationError{Field: v.Field, Message: "contains invalid characters", Code: "INVALID_INPUT"}
	}
	return nil
}

// PasswordComplexity validates password complexity requirements
type PasswordComplexity struct {
	Field string
	Value string
}

func (v PasswordComplexity) Validate() *ValidationError {
	if len(v.Value) < 12 {
		return &ValidationError{Field: v.Field, Message: "must be at least 12 characters", Code: "WEAK_PASSWORD"}
	}

	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(v.Value)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(v.Value)
	hasDigit := regexp.MustCompile(`[0-9]`).MatchString(v.Value)
	hasSpecial := regexp.MustCompile(`[^a-zA-Z0-9]`).MatchString(v.Value)

	if !hasUpper || !hasLower || !hasDigit || !hasSpecial {
		return &ValidationError{
			Field:   v.Field,
			Message: "must contain uppercase, lowercase, digit, and special character",
			Code:    "WEAK_PASSWORD",
		}
	}
	return nil
}
