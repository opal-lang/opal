package decorators

import (
	"testing"
	"time"

	"github.com/aledsdavies/devcmd/core/ast"
)

func TestValidateShellCommandSafety(t *testing.T) {
	tests := []struct {
		name          string
		params        []ast.NamedParameter
		paramName     string
		decoratorName string
		shouldFail    bool
		expectedError string
	}{
		{
			name: "Safe command",
			params: []ast.NamedParameter{
				{Name: "command", Value: &ast.StringLiteral{Value: "echo hello"}},
			},
			paramName:     "command",
			decoratorName: "test",
			shouldFail:    false,
		},
		{
			name: "Command with semicolon injection",
			params: []ast.NamedParameter{
				{Name: "command", Value: &ast.StringLiteral{Value: "echo hello; rm -rf /"}},
			},
			paramName:     "command",
			decoratorName: "test",
			shouldFail:    true,
			expectedError: "command chaining",
		},
		{
			name: "Command with pipe injection",
			params: []ast.NamedParameter{
				{Name: "command", Value: &ast.StringLiteral{Value: "echo hello | cat /etc/passwd"}},
			},
			paramName:     "command",
			decoratorName: "test",
			shouldFail:    true,
			expectedError: "command piping",
		},
		{
			name: "Command substitution attempt",
			params: []ast.NamedParameter{
				{Name: "command", Value: &ast.StringLiteral{Value: "echo $(whoami)"}},
			},
			paramName:     "command",
			decoratorName: "test",
			shouldFail:    true,
			expectedError: "command substitution",
		},
		{
			name: "Quote escape attempt",
			params: []ast.NamedParameter{
				{Name: "command", Value: &ast.StringLiteral{Value: `echo "test"`}},
			},
			paramName:     "command",
			decoratorName: "test",
			shouldFail:    true,
			expectedError: "quote or escape characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateShellCommandSafety(tt.params, tt.paramName, tt.decoratorName)

			if tt.shouldFail {
				if err == nil {
					t.Errorf("Expected validation to fail, but it passed")
				} else if tt.expectedError != "" && !containsSubstring(err.Error(), tt.expectedError) {
					t.Errorf("Expected error to contain '%s', got: %s", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected validation to pass, but got error: %s", err.Error())
				}
			}
		})
	}
}

func TestValidateNoPrivilegeEscalation(t *testing.T) {
	tests := []struct {
		name          string
		params        []ast.NamedParameter
		paramName     string
		decoratorName string
		shouldFail    bool
		expectedError string
	}{
		{
			name: "Safe command",
			params: []ast.NamedParameter{
				{Name: "command", Value: &ast.StringLiteral{Value: "ls -la"}},
			},
			paramName:     "command",
			decoratorName: "test",
			shouldFail:    false,
		},
		{
			name: "Sudo attempt",
			params: []ast.NamedParameter{
				{Name: "command", Value: &ast.StringLiteral{Value: "sudo rm file"}},
			},
			paramName:     "command",
			decoratorName: "test",
			shouldFail:    true,
			expectedError: "privilege escalation",
		},
		{
			name: "Setuid attempt",
			params: []ast.NamedParameter{
				{Name: "command", Value: &ast.StringLiteral{Value: "chmod +s /bin/bash"}},
			},
			paramName:     "command",
			decoratorName: "test",
			shouldFail:    true,
			expectedError: "privilege escalation",
		},
		{
			name: "Password file access",
			params: []ast.NamedParameter{
				{Name: "command", Value: &ast.StringLiteral{Value: "cat /etc/passwd"}},
			},
			paramName:     "command",
			decoratorName: "test",
			shouldFail:    true,
			expectedError: "privilege escalation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNoPrivilegeEscalation(tt.params, tt.paramName, tt.decoratorName)

			if tt.shouldFail {
				if err == nil {
					t.Errorf("Expected validation to fail, but it passed")
				} else if tt.expectedError != "" && !containsSubstring(err.Error(), tt.expectedError) {
					t.Errorf("Expected error to contain '%s', got: %s", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected validation to pass, but got error: %s", err.Error())
				}
			}
		})
	}
}

func TestValidateResourceLimits(t *testing.T) {
	tests := []struct {
		name          string
		params        []ast.NamedParameter
		paramName     string
		maxValue      int
		decoratorName string
		shouldFail    bool
		expectedError string
	}{
		{
			name: "Valid concurrency",
			params: []ast.NamedParameter{
				{Name: "concurrency", Value: &ast.NumberLiteral{Value: "4"}},
			},
			paramName:     "concurrency",
			maxValue:      100,
			decoratorName: "test",
			shouldFail:    false,
		},
		{
			name: "Excessive concurrency",
			params: []ast.NamedParameter{
				{Name: "concurrency", Value: &ast.NumberLiteral{Value: "10000"}},
			},
			paramName:     "concurrency",
			maxValue:      1000,
			decoratorName: "test",
			shouldFail:    true,
			expectedError: "exceeds maximum safe limit",
		},
		{
			name: "Zero concurrency",
			params: []ast.NamedParameter{
				{Name: "concurrency", Value: &ast.NumberLiteral{Value: "0"}},
			},
			paramName:     "concurrency",
			maxValue:      100,
			decoratorName: "test",
			shouldFail:    true,
			expectedError: "must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateResourceLimits(tt.params, tt.paramName, tt.maxValue, tt.decoratorName)

			if tt.shouldFail {
				if err == nil {
					t.Errorf("Expected validation to fail, but it passed")
				} else if tt.expectedError != "" && !containsSubstring(err.Error(), tt.expectedError) {
					t.Errorf("Expected error to contain '%s', got: %s", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected validation to pass, but got error: %s", err.Error())
				}
			}
		})
	}
}

func TestValidateTimeoutSafety(t *testing.T) {
	tests := []struct {
		name          string
		params        []ast.NamedParameter
		paramName     string
		maxTimeout    time.Duration
		decoratorName string
		shouldFail    bool
		expectedError string
	}{
		{
			name: "Valid timeout",
			params: []ast.NamedParameter{
				{Name: "timeout", Value: &ast.DurationLiteral{Value: "30s"}},
			},
			paramName:     "timeout",
			maxTimeout:    time.Hour,
			decoratorName: "test",
			shouldFail:    false,
		},
		{
			name: "Excessive timeout",
			params: []ast.NamedParameter{
				{Name: "timeout", Value: &ast.DurationLiteral{Value: "25h"}},
			},
			paramName:     "timeout",
			maxTimeout:    24 * time.Hour,
			decoratorName: "test",
			shouldFail:    true,
			expectedError: "exceeds maximum safe timeout",
		},
		{
			name: "Zero timeout",
			params: []ast.NamedParameter{
				{Name: "timeout", Value: &ast.DurationLiteral{Value: "0s"}},
			},
			paramName:     "timeout",
			maxTimeout:    time.Hour,
			decoratorName: "test",
			shouldFail:    true,
			expectedError: "must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTimeoutSafety(tt.params, tt.paramName, tt.maxTimeout, tt.decoratorName)

			if tt.shouldFail {
				if err == nil {
					t.Errorf("Expected validation to fail, but it passed")
				} else if tt.expectedError != "" && !containsSubstring(err.Error(), tt.expectedError) {
					t.Errorf("Expected error to contain '%s', got: %s", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected validation to pass, but got error: %s", err.Error())
				}
			}
		})
	}
}

func TestPerformComprehensiveSecurityValidation(t *testing.T) {
	schema := []ParameterSchema{
		{Name: "path", Type: ast.StringType, Required: true},
		{Name: "concurrency", Type: ast.NumberType, Required: false},
		{Name: "timeout", Type: ast.DurationType, Required: false},
	}

	tests := []struct {
		name           string
		params         []ast.NamedParameter
		decoratorName  string
		shouldFail     bool
		expectedChecks int
	}{
		{
			name: "Safe parameters",
			params: []ast.NamedParameter{
				{Name: "path", Value: &ast.StringLiteral{Value: "/tmp/safe"}},
				{Name: "concurrency", Value: &ast.NumberLiteral{Value: "4"}},
			},
			decoratorName:  "test",
			shouldFail:     false,
			expectedChecks: 2, // 1 path + 1 resource limit
		},
		{
			name: "Unsafe path with directory traversal",
			params: []ast.NamedParameter{
				{Name: "path", Value: &ast.StringLiteral{Value: "../../../etc/passwd"}},
			},
			decoratorName:  "test",
			shouldFail:     true,
			expectedChecks: 1, // 1 path validation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary, err := PerformComprehensiveSecurityValidation(tt.params, schema, tt.decoratorName)

			if tt.shouldFail {
				if err == nil {
					t.Errorf("Expected validation to fail, but it passed")
				}
			} else {
				if err != nil {
					t.Errorf("Expected validation to pass, but got error: %s", err.Error())
				}
			}

			if summary != nil {
				totalChecks := summary.PathValidations + summary.ShellValidations + summary.ResourceLimitChecks + summary.PrivilegeChecks
				if totalChecks != tt.expectedChecks {
					t.Errorf("Expected %d security checks, got %d", tt.expectedChecks, totalChecks)
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring
func containsSubstring(haystack, needle string) bool {
	return len(needle) == 0 || len(haystack) >= len(needle) &&
		(haystack == needle ||
			containsSubstring(haystack[1:], needle) ||
			(len(haystack) > 0 && haystack[:len(needle)] == needle))
}
