package decorators

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aledsdavies/devcmd/core/ast"
)

// ValidateParameterType validates that a parameter value matches the expected type
// Allows both literal values and identifiers (which can resolve at runtime)
func ValidateParameterType(paramName string, paramValue ast.Expression, expectedType ast.ExpressionType, decoratorName string) error {
	switch expectedType {
	case ast.StringType:
		switch paramValue.(type) {
		case *ast.StringLiteral, *ast.Identifier:
			return nil
		default:
			return fmt.Errorf("@%s '%s' parameter must be of type string", decoratorName, paramName)
		}
	case ast.NumberType:
		switch paramValue.(type) {
		case *ast.NumberLiteral, *ast.Identifier:
			return nil
		default:
			return fmt.Errorf("@%s '%s' parameter must be of type number", decoratorName, paramName)
		}
	case ast.DurationType:
		switch paramValue.(type) {
		case *ast.DurationLiteral, *ast.Identifier:
			return nil
		default:
			return fmt.Errorf("@%s '%s' parameter must be of type duration", decoratorName, paramName)
		}
	case ast.BooleanType:
		switch paramValue.(type) {
		case *ast.BooleanLiteral, *ast.Identifier:
			return nil
		default:
			return fmt.Errorf("@%s '%s' parameter must be of type boolean", decoratorName, paramName)
		}
	case ast.IdentifierType:
		switch paramValue.(type) {
		case *ast.Identifier:
			return nil
		default:
			return fmt.Errorf("@%s '%s' parameter must be an identifier", decoratorName, paramName)
		}
	default:
		return fmt.Errorf("@%s '%s' parameter has unsupported type %v", decoratorName, paramName, expectedType)
	}
}

// ValidateRequiredParameter validates that a required parameter exists and has the correct type
func ValidateRequiredParameter(params []ast.NamedParameter, paramName string, expectedType ast.ExpressionType, decoratorName string) error {
	param := ast.FindParameter(params, paramName)
	if param == nil {
		return fmt.Errorf("@%s requires '%s' parameter", decoratorName, paramName)
	}
	return ValidateParameterType(paramName, param.Value, expectedType, decoratorName)
}

// ValidateOptionalParameter validates that an optional parameter (if present) has the correct type
func ValidateOptionalParameter(params []ast.NamedParameter, paramName string, expectedType ast.ExpressionType, decoratorName string) error {
	param := ast.FindParameter(params, paramName)
	if param == nil {
		return nil // Optional parameter not provided, which is fine
	}
	return ValidateParameterType(paramName, param.Value, expectedType, decoratorName)
}

// ValidateParameterCount validates that the number of parameters is within expected bounds
func ValidateParameterCount(params []ast.NamedParameter, minParams, maxParams int, decoratorName string) error {
	count := len(params)
	if count < minParams {
		if minParams == maxParams {
			return fmt.Errorf("@%s requires exactly %d parameter(s), got %d", decoratorName, minParams, count)
		}
		return fmt.Errorf("@%s requires at least %d parameter(s), got %d", decoratorName, minParams, count)
	}
	if count > maxParams {
		if minParams == maxParams {
			return fmt.Errorf("@%s requires exactly %d parameter(s), got %d", decoratorName, maxParams, count)
		}
		return fmt.Errorf("@%s accepts at most %d parameter(s), got %d", decoratorName, maxParams, count)
	}
	return nil
}

// ValidatePositiveInteger validates that a numeric parameter is positive
func ValidatePositiveInteger(params []ast.NamedParameter, paramName string, decoratorName string) error {
	param := ast.FindParameter(params, paramName)
	if param == nil {
		return fmt.Errorf("@%s '%s' parameter is required", decoratorName, paramName)
	}

	// For identifiers, we can't validate at parse time
	if _, isIdentifier := param.Value.(*ast.Identifier); isIdentifier {
		return nil
	}

	// For literals, validate the value
	if numLit, ok := param.Value.(*ast.NumberLiteral); ok {
		if value, err := strconv.Atoi(numLit.Value); err != nil {
			return fmt.Errorf("@%s '%s' parameter must be a valid integer", decoratorName, paramName)
		} else if value <= 0 {
			return fmt.Errorf("@%s '%s' parameter must be positive, got %d", decoratorName, paramName, value)
		}
		return nil
	}

	return fmt.Errorf("@%s '%s' parameter must be a number", decoratorName, paramName)
}

// SanitizeNumericValue clamps a numeric value to a safe range
// Returns the clamped value and whether it was modified
func SanitizeNumericValue(value, min, max int) (int, bool) {
	if value < min {
		return min, true
	}
	if value > max {
		return max, true
	}
	return value, false
}

// ValidateNumericBounds validates that a numeric parameter is within safe bounds
// This is more flexible than ValidateIntegerRange as it allows auto-clamping
func ValidateNumericBounds(params []ast.NamedParameter, paramName string, min, max int, decoratorName string, allowClamping bool) (int, error) {
	param := ast.FindParameter(params, paramName)
	if param == nil {
		return 0, fmt.Errorf("@%s '%s' parameter is required", decoratorName, paramName)
	}

	// For identifiers, we can't validate at parse time
	if _, isIdentifier := param.Value.(*ast.Identifier); isIdentifier {
		return 0, nil // Return 0 but no error - will be validated at runtime
	}

	// For literals, validate the range
	if numLit, ok := param.Value.(*ast.NumberLiteral); ok {
		if value, err := strconv.Atoi(numLit.Value); err != nil {
			return 0, fmt.Errorf("@%s '%s' parameter must be a valid integer", decoratorName, paramName)
		} else {
			sanitized, wasClamped := SanitizeNumericValue(value, min, max)
			if wasClamped && !allowClamping {
				return 0, fmt.Errorf("@%s '%s' parameter must be between %d and %d, got %d", decoratorName, paramName, min, max, value)
			}
			return sanitized, nil
		}
	}

	return 0, fmt.Errorf("@%s '%s' parameter must be a number", decoratorName, paramName)
}

// ValidateIntegerRange validates that a numeric parameter is within a specific range
func ValidateIntegerRange(params []ast.NamedParameter, paramName string, min, max int, decoratorName string) error {
	param := ast.FindParameter(params, paramName)
	if param == nil {
		return fmt.Errorf("@%s '%s' parameter is required", decoratorName, paramName)
	}

	// For identifiers, we can't validate at parse time
	if _, isIdentifier := param.Value.(*ast.Identifier); isIdentifier {
		return nil
	}

	// For literals, validate the range
	if numLit, ok := param.Value.(*ast.NumberLiteral); ok {
		if value, err := strconv.Atoi(numLit.Value); err != nil {
			return fmt.Errorf("@%s '%s' parameter must be a valid integer", decoratorName, paramName)
		} else if value < min || value > max {
			return fmt.Errorf("@%s '%s' parameter must be between %d and %d, got %d", decoratorName, paramName, min, max, value)
		}
		return nil
	}

	return fmt.Errorf("@%s '%s' parameter must be a number", decoratorName, paramName)
}

// ValidateDuration validates that a duration parameter is valid and within reasonable bounds
func ValidateDuration(params []ast.NamedParameter, paramName string, minDuration, maxDuration time.Duration, decoratorName string) error {
	param := ast.FindParameter(params, paramName)
	if param == nil {
		return nil // Duration parameters are typically optional
	}

	// For identifiers, we can't validate at parse time
	if _, isIdentifier := param.Value.(*ast.Identifier); isIdentifier {
		return nil
	}

	// For literals, validate the duration
	if durLit, ok := param.Value.(*ast.DurationLiteral); ok {
		if duration, err := time.ParseDuration(durLit.Value); err != nil {
			return fmt.Errorf("@%s '%s' parameter must be a valid duration (e.g., '1s', '5m')", decoratorName, paramName)
		} else if duration < minDuration {
			return fmt.Errorf("@%s '%s' parameter must be at least %v, got %v", decoratorName, paramName, minDuration, duration)
		} else if maxDuration > 0 && duration > maxDuration {
			return fmt.Errorf("@%s '%s' parameter must be at most %v, got %v", decoratorName, paramName, maxDuration, duration)
		}
		return nil
	}

	return fmt.Errorf("@%s '%s' parameter must be a duration", decoratorName, paramName)
}

// SanitizePath sanitizes a file path by removing dangerous patterns
// Returns the sanitized path and any issues found for logging
func SanitizePath(path string) (string, []string) {
	issues := make([]string, 0)
	sanitized := path

	// Remove dangerous patterns
	if strings.Contains(sanitized, "..") {
		issues = append(issues, "directory traversal (..) removed")
		// Replace .. with single dot to stay in current directory
		sanitized = strings.ReplaceAll(sanitized, "..", ".")
	}

	// Remove null bytes
	if strings.Contains(sanitized, "\x00") {
		issues = append(issues, "null bytes removed")
		sanitized = strings.ReplaceAll(sanitized, "\x00", "")
	}

	// Clean path using filepath.Clean
	cleaned := filepath.Clean(sanitized)
	if cleaned != sanitized {
		issues = append(issues, "path cleaned by filepath.Clean")
		sanitized = cleaned
	}

	return sanitized, issues
}

// ValidatePathSafety validates that a path parameter is safe (no directory traversal, etc.)
func ValidatePathSafety(params []ast.NamedParameter, paramName string, decoratorName string) error {
	param := ast.FindParameter(params, paramName)
	if param == nil {
		return fmt.Errorf("@%s '%s' parameter is required", decoratorName, paramName)
	}

	// For identifiers, we can't validate at parse time
	if _, isIdentifier := param.Value.(*ast.Identifier); isIdentifier {
		return nil
	}

	// For string literals, validate path safety
	if strLit, ok := param.Value.(*ast.StringLiteral); ok {
		path := strLit.Value

		// Check for empty path
		if strings.TrimSpace(path) == "" {
			return fmt.Errorf("@%s '%s' parameter cannot be empty", decoratorName, paramName)
		}

		// Check for dangerous patterns
		if strings.Contains(path, "..") {
			return fmt.Errorf("@%s '%s' parameter contains directory traversal (..), which is not allowed", decoratorName, paramName)
		}

		// Check for null bytes (security issue)
		if strings.Contains(path, "\x00") {
			return fmt.Errorf("@%s '%s' parameter contains null bytes, which is not allowed", decoratorName, paramName)
		}

		// Clean and validate the path
		cleanPath := filepath.Clean(path)
		if cleanPath != path && path != "." && path != ".." {
			// Allow common cases but warn about others
			// This is informational rather than blocking since Clean() may change valid paths
			_ = cleanPath // Suppress unused variable warning
		}

		return nil
	}

	return fmt.Errorf("@%s '%s' parameter must be a string", decoratorName, paramName)
}

// SanitizeEnvironmentVariableName sanitizes an environment variable name
// Returns the sanitized name and any issues found for logging
func SanitizeEnvironmentVariableName(envName string) (string, []string) {
	issues := make([]string, 0)
	sanitized := envName

	// Remove whitespace
	sanitized = strings.TrimSpace(sanitized)

	// Replace invalid characters with underscores
	envNameRegex := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	if envNameRegex.MatchString(sanitized) {
		issues = append(issues, "invalid characters replaced with underscores")
		sanitized = envNameRegex.ReplaceAllString(sanitized, "_")
	}

	// Ensure it starts with letter or underscore
	if len(sanitized) > 0 && sanitized[0] >= '0' && sanitized[0] <= '9' {
		issues = append(issues, "prefixed with underscore (cannot start with digit)")
		sanitized = "_" + sanitized
	}

	// Limit length
	if len(sanitized) > 255 {
		issues = append(issues, "truncated to 255 characters")
		sanitized = sanitized[:255]
	}

	return sanitized, issues
}

// ValidateEnvironmentVariableName validates that an environment variable name is safe
func ValidateEnvironmentVariableName(params []ast.NamedParameter, paramName string, decoratorName string) error {
	param := ast.FindParameter(params, paramName)
	if param == nil {
		return fmt.Errorf("@%s '%s' parameter is required", decoratorName, paramName)
	}

	// For identifiers, we can't validate at parse time
	if _, isIdentifier := param.Value.(*ast.Identifier); isIdentifier {
		return nil
	}

	// For string literals, validate environment variable name
	if strLit, ok := param.Value.(*ast.StringLiteral); ok {
		envName := strLit.Value

		// Check for empty name
		if strings.TrimSpace(envName) == "" {
			return fmt.Errorf("@%s '%s' parameter cannot be empty", decoratorName, paramName)
		}

		// Validate environment variable name format (POSIX standard)
		// Must start with letter or underscore, followed by letters, digits, or underscores
		envNameRegex := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
		if !envNameRegex.MatchString(envName) {
			return fmt.Errorf("@%s '%s' parameter must be a valid environment variable name (letters, digits, underscore only, cannot start with digit)", decoratorName, paramName)
		}

		// Check for reasonable length (environment variable names shouldn't be too long)
		if len(envName) > 255 {
			return fmt.Errorf("@%s '%s' parameter is too long (max 255 characters)", decoratorName, paramName)
		}

		return nil
	}

	return fmt.Errorf("@%s '%s' parameter must be a string", decoratorName, paramName)
}

// ValidateStringContent validates that a string parameter doesn't contain dangerous content
func ValidateStringContent(params []ast.NamedParameter, paramName string, decoratorName string) error {
	param := ast.FindParameter(params, paramName)
	if param == nil {
		return nil // String content parameters are typically optional
	}

	// For identifiers, we can't validate at parse time
	if _, isIdentifier := param.Value.(*ast.Identifier); isIdentifier {
		return nil
	}

	// For string literals, validate content safety
	if strLit, ok := param.Value.(*ast.StringLiteral); ok {
		content := strLit.Value

		// Check for null bytes (security issue)
		if strings.Contains(content, "\x00") {
			return fmt.Errorf("@%s '%s' parameter contains null bytes, which is not allowed", decoratorName, paramName)
		}

		// Check for excessively long strings (potential DoS)
		if len(content) > 10000 {
			return fmt.Errorf("@%s '%s' parameter is too long (max 10000 characters)", decoratorName, paramName)
		}

		return nil
	}

	return fmt.Errorf("@%s '%s' parameter must be a string", decoratorName, paramName)
}

// SanitizeShellCommand sanitizes a shell command string by removing dangerous patterns
// Returns the sanitized command and a list of removed patterns for logging
func SanitizeShellCommand(command string) (string, []string) {
	removed := make([]string, 0)
	sanitized := command

	// Remove dangerous patterns (be conservative)
	dangerousPatterns := []string{
		";", "&", "|", "`", "$(", "||", "&&", ">", "<", ">>", "2>",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(sanitized, pattern) {
			removed = append(removed, pattern)
			// Replace with safe equivalent or remove
			switch pattern {
			case ";", "&":
				sanitized = strings.ReplaceAll(sanitized, pattern, " ")
			default:
				sanitized = strings.ReplaceAll(sanitized, pattern, "")
			}
		}
	}

	return strings.TrimSpace(sanitized), removed
}

// ValidateShellCommandSafety validates that a string doesn't contain shell injection patterns
// This is for parameters that might be used directly in shell commands
func ValidateShellCommandSafety(params []ast.NamedParameter, paramName string, decoratorName string) error {
	param := ast.FindParameter(params, paramName)
	if param == nil {
		return nil // Optional parameter
	}

	// For identifiers, we can't validate at parse time
	if _, isIdentifier := param.Value.(*ast.Identifier); isIdentifier {
		return nil
	}

	// For string literals, validate shell command safety
	if strLit, ok := param.Value.(*ast.StringLiteral); ok {
		content := strLit.Value

		// Basic string content validation first
		if err := ValidateStringContent(params, paramName, decoratorName); err != nil {
			return err
		}

		// Check for dangerous shell injection patterns
		dangerousPatterns := []struct {
			pattern string
			reason  string
		}{
			{";", "command chaining"},
			{"&", "background execution"},
			{"|", "command piping"},
			{"`", "command substitution"},
			{"$(", "command substitution"},
			{"\n", "command injection via newline"},
			{"\r", "command injection via carriage return"},
			{"&&", "conditional execution"},
			{"||", "conditional execution"},
			{">", "output redirection"},
			{"<", "input redirection"},
			{">>", "output redirection"},
			{"2>", "error redirection"},
			{"${", "variable expansion"},
			{"*", "glob expansion"},
			{"?", "glob expansion"},
		}

		for _, dangerous := range dangerousPatterns {
			if strings.Contains(content, dangerous.pattern) {
				return fmt.Errorf("@%s '%s' parameter contains potentially dangerous shell pattern '%s' (%s), which is not allowed",
					decoratorName, paramName, dangerous.pattern, dangerous.reason)
			}
		}

		// Check for suspicious patterns that might be attempts to break out of quotes
		suspiciousPatterns := []string{
			`"`, `'`, `\"`, `\'`, `\\`,
		}

		for _, pattern := range suspiciousPatterns {
			if strings.Contains(content, pattern) {
				return fmt.Errorf("@%s '%s' parameter contains quote or escape characters, which could lead to shell injection",
					decoratorName, paramName)
			}
		}

		return nil
	}

	return fmt.Errorf("@%s '%s' parameter must be a string", decoratorName, paramName)
}

// ValidateResourceLimits validates that resource-related parameters are within safe bounds
func ValidateResourceLimits(params []ast.NamedParameter, paramName string, maxValue int, decoratorName string) error {
	param := ast.FindParameter(params, paramName)
	if param == nil {
		return nil // Optional parameter
	}

	// For identifiers, we can't validate at parse time
	if _, isIdentifier := param.Value.(*ast.Identifier); isIdentifier {
		return nil
	}

	// For literals, validate resource limits
	if numLit, ok := param.Value.(*ast.NumberLiteral); ok {
		if value, err := strconv.Atoi(numLit.Value); err != nil {
			return fmt.Errorf("@%s '%s' parameter must be a valid integer", decoratorName, paramName)
		} else if value > maxValue {
			return fmt.Errorf("@%s '%s' parameter exceeds maximum safe limit of %d, got %d", decoratorName, paramName, maxValue, value)
		} else if value <= 0 {
			return fmt.Errorf("@%s '%s' parameter must be positive, got %d", decoratorName, paramName, value)
		}
		return nil
	}

	return fmt.Errorf("@%s '%s' parameter must be a number", decoratorName, paramName)
}

// ValidateTimeoutSafety validates that timeout values are reasonable and safe
func ValidateTimeoutSafety(params []ast.NamedParameter, paramName string, maxTimeout time.Duration, decoratorName string) error {
	param := ast.FindParameter(params, paramName)
	if param == nil {
		return nil // Optional parameter
	}

	// For identifiers, we can't validate at parse time
	if _, isIdentifier := param.Value.(*ast.Identifier); isIdentifier {
		return nil
	}

	// For literals, validate timeout safety
	if durLit, ok := param.Value.(*ast.DurationLiteral); ok {
		if duration, err := time.ParseDuration(durLit.Value); err != nil {
			return fmt.Errorf("@%s '%s' parameter must be a valid duration (e.g., '1s', '5m')", decoratorName, paramName)
		} else if duration <= 0 {
			return fmt.Errorf("@%s '%s' parameter must be positive", decoratorName, paramName)
		} else if duration > maxTimeout {
			return fmt.Errorf("@%s '%s' parameter exceeds maximum safe timeout of %v, got %v", decoratorName, paramName, maxTimeout, duration)
		}
		return nil
	}

	return fmt.Errorf("@%s '%s' parameter must be a duration", decoratorName, paramName)
}

// ValidateNoPrivilegeEscalation validates that parameters don't attempt privilege escalation
func ValidateNoPrivilegeEscalation(params []ast.NamedParameter, paramName string, decoratorName string) error {
	param := ast.FindParameter(params, paramName)
	if param == nil {
		return nil // Optional parameter
	}

	// For identifiers, we can't validate at parse time
	if _, isIdentifier := param.Value.(*ast.Identifier); isIdentifier {
		return nil
	}

	// For string literals, validate no privilege escalation attempts
	if strLit, ok := param.Value.(*ast.StringLiteral); ok {
		content := strings.ToLower(strLit.Value)

		// Check for common privilege escalation patterns
		dangerousCommands := []string{
			"sudo", "su", "doas", "runas",
			"chmod +s", "chmod 4", "setuid",
			"/etc/passwd", "/etc/shadow", "/etc/sudoers",
			"chown root", "chgrp root",
		}

		for _, dangerous := range dangerousCommands {
			if strings.Contains(content, dangerous) {
				return fmt.Errorf("@%s '%s' parameter contains potentially dangerous privilege escalation pattern '%s'",
					decoratorName, paramName, dangerous)
			}
		}

		return nil
	}

	return fmt.Errorf("@%s '%s' parameter must be a string", decoratorName, paramName)
}

// SecurityValidationSummary provides a summary of security validations performed
type SecurityValidationSummary struct {
	ParameterCount      int
	PathValidations     int
	ShellValidations    int
	PrivilegeChecks     int
	ResourceLimitChecks int
	ValidationErrors    []string
}

// PerformComprehensiveSecurityValidation performs all relevant security validations
func PerformComprehensiveSecurityValidation(params []ast.NamedParameter, schema []ParameterSchema, decoratorName string) (*SecurityValidationSummary, error) {
	summary := &SecurityValidationSummary{
		ParameterCount:   len(params),
		ValidationErrors: make([]string, 0),
	}

	// Basic schema compliance
	if err := ValidateSchemaCompliance(params, schema, decoratorName); err != nil {
		return summary, err
	}

	// Security validations based on parameter names and types
	for _, param := range params {
		switch {
		case strings.Contains(strings.ToLower(param.Name), "path") ||
			strings.Contains(strings.ToLower(param.Name), "dir") ||
			strings.Contains(strings.ToLower(param.Name), "file"):
			// Path-related parameters
			if err := ValidatePathSafety(params, param.Name, decoratorName); err != nil {
				summary.ValidationErrors = append(summary.ValidationErrors, err.Error())
			}
			summary.PathValidations++

		case strings.Contains(strings.ToLower(param.Name), "command") ||
			strings.Contains(strings.ToLower(param.Name), "cmd") ||
			strings.Contains(strings.ToLower(param.Name), "script"):
			// Command-related parameters need shell safety
			if err := ValidateShellCommandSafety(params, param.Name, decoratorName); err != nil {
				summary.ValidationErrors = append(summary.ValidationErrors, err.Error())
			}
			if err := ValidateNoPrivilegeEscalation(params, param.Name, decoratorName); err != nil {
				summary.ValidationErrors = append(summary.ValidationErrors, err.Error())
			}
			summary.ShellValidations++
			summary.PrivilegeChecks++

		case strings.Contains(strings.ToLower(param.Name), "concurrency") ||
			strings.Contains(strings.ToLower(param.Name), "parallel") ||
			strings.Contains(strings.ToLower(param.Name), "workers"):
			// Resource limit parameters
			if err := ValidateResourceLimits(params, param.Name, 1000, decoratorName); err != nil {
				summary.ValidationErrors = append(summary.ValidationErrors, err.Error())
			}
			summary.ResourceLimitChecks++

		case strings.Contains(strings.ToLower(param.Name), "timeout") ||
			strings.Contains(strings.ToLower(param.Name), "duration"):
			// Timeout parameters
			if err := ValidateTimeoutSafety(params, param.Name, 24*time.Hour, decoratorName); err != nil {
				summary.ValidationErrors = append(summary.ValidationErrors, err.Error())
			}
			summary.ResourceLimitChecks++
		}
	}

	// Return error if any validation failed
	if len(summary.ValidationErrors) > 0 {
		return summary, fmt.Errorf("security validation failed: %v", summary.ValidationErrors)
	}

	return summary, nil
}

// ValidateSchemaCompliance validates parameters against a decorator's parameter schema
func ValidateSchemaCompliance(params []ast.NamedParameter, schema []ParameterSchema, decoratorName string) error {
	// First, resolve positional parameters to named parameters based on schema
	resolvedParams, err := ResolvePositionalParameters(params, schema)
	if err != nil {
		return fmt.Errorf("@%s parameter resolution error: %w", decoratorName, err)
	}

	// Check for unknown parameters
	for _, param := range resolvedParams {
		found := false
		for _, schemaParam := range schema {
			if schemaParam.Name == param.Name {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("@%s does not accept parameter '%s'", decoratorName, param.Name)
		}
	}

	// Validate required parameters and types using resolved parameters
	for _, schemaParam := range schema {
		if schemaParam.Required {
			if err := ValidateRequiredParameter(resolvedParams, schemaParam.Name, schemaParam.Type, decoratorName); err != nil {
				return err
			}
		} else {
			if err := ValidateOptionalParameter(resolvedParams, schemaParam.Name, schemaParam.Type, decoratorName); err != nil {
				return err
			}
		}
	}

	return nil
}
