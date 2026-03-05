package validation

import (
	"testing"
)

func TestRequiredString(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid", "hello", false},
		{"empty", "", true},
		{"whitespace only", "   ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := RequiredString{Field: "test", Value: tt.value}
			err := v.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("RequiredString.Validate() = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidUUID(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid uuid", "550e8400-e29b-41d4-a716-446655440000", false},
		{"invalid", "not-a-uuid", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := ValidUUID{Field: "test", Value: tt.value}
			err := v.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidUUID.Validate() = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidEmail(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid", "user@example.com", false},
		{"missing @", "userexample.com", true},
		{"missing domain", "user@", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := ValidEmail{Field: "test", Value: tt.value}
			err := v.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidEmail.Validate() = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChain(t *testing.T) {
	// Test with all valid inputs
	chain := NewChain().
		Add(RequiredString{Field: "name", Value: "Alice"}).
		Add(ValidEmail{Field: "email", Value: "alice@example.com"}).
		Add(IntRange{Field: "age", Value: 25, Min: 1, Max: 150})

	errs := chain.Run()
	if errs.HasErrors() {
		t.Errorf("expected no errors, got %v", errs)
	}

	// Test with some invalid inputs
	chain2 := NewChain().
		Add(RequiredString{Field: "name", Value: ""}).
		Add(ValidEmail{Field: "email", Value: "not-an-email"}).
		Add(IntRange{Field: "age", Value: 200, Min: 1, Max: 150})

	errs2 := chain2.Run()
	if !errs2.HasErrors() {
		t.Error("expected errors, got none")
	}
	if len(errs2) != 3 {
		t.Errorf("expected 3 errors, got %d", len(errs2))
	}
}

func TestPasswordComplexity(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"strong", "MyStr0ng!Pass#1", false},
		{"too short", "Ab1!", false},
		{"short", "Ab1!cd", true},
		{"no upper", "mystr0ng!pass#1", true},
		{"no digit", "MyStrong!Pass#x", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := PasswordComplexity{Field: "password", Value: tt.value}
			err := v.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("PasswordComplexity(%q) = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestOneOf(t *testing.T) {
	v := OneOf{Field: "status", Value: "active", Allowed: []string{"active", "inactive", "suspended"}}
	if err := v.Validate(); err != nil {
		t.Errorf("expected valid, got %v", err)
	}

	v2 := OneOf{Field: "status", Value: "unknown", Allowed: []string{"active", "inactive"}}
	if err := v2.Validate(); err == nil {
		t.Error("expected error for invalid value")
	}
}

func TestNoSQLInjection(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"clean", "hello world", false},
		{"drop table", "DROP TABLE users", true},
		{"semicolon", "value; DELETE", true},
		{"comment", "value -- comment", true},
		{"union", "1 UNION SELECT", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NoSQLInjection{Field: "test", Value: tt.value}
			err := v.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("NoSQLInjection(%q) = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}
