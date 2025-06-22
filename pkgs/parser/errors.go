package parser

import (
	"fmt"
	"strings"
)

type ValidationError struct {
	Errors []ValidationErrorEntry
	Debug  *DebugTrace
}

type ValidationErrorEntry struct {
	Line    int
	Column  int
	Message string
	Context string
}

type DebugTrace struct {
	Enabled bool
	Tokens  []string
	Rules   []string
	Errors  []string
}

func (d *DebugTrace) Log(format string, args ...interface{}) {
	if d != nil {
		d.Rules = append(d.Rules, fmt.Sprintf(format, args...))
	}
}

func (d *DebugTrace) LogToken(token string) {
	if d != nil {
		d.Tokens = append(d.Tokens, token)
	}
}

func (d *DebugTrace) LogError(format string, args ...interface{}) {
	if d != nil {
		d.Errors = append(d.Errors, fmt.Sprintf(format, args...))
	}
}

func (d *DebugTrace) HasTrace() bool {
	return d != nil && (len(d.Tokens) > 0 || len(d.Rules) > 0 || len(d.Errors) > 0)
}

func (d *DebugTrace) String() string {
	if !d.HasTrace() {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("\n=== DEBUG TRACE ===\n")

	if len(d.Rules) > 0 {
		builder.WriteString("Rules: ")
		builder.WriteString(strings.Join(d.Rules, " → "))
		builder.WriteString("\n")
	}

	if len(d.Tokens) > 0 {
		builder.WriteString("Tokens: [")
		builder.WriteString(strings.Join(d.Tokens, ", "))
		builder.WriteString("]\n")
	}

	if len(d.Errors) > 0 {
		builder.WriteString("Debug Errors: ")
		builder.WriteString(strings.Join(d.Errors, "; "))
		builder.WriteString("\n")
	}

	builder.WriteString("==================")
	return builder.String()
}

func (e *ValidationError) Error() string {
	if len(e.Errors) == 0 {
		return ""
	}

	var builder strings.Builder
	for i, err := range e.Errors {
		if i > 0 {
			builder.WriteString("\n")
		}

		if err.Context != "" {
			pointer := strings.Repeat(" ", err.Column) + "^"
			builder.WriteString(fmt.Sprintf("line %d: %s\n%s\n%s",
				err.Line,
				err.Message,
				err.Context,
				pointer))
		} else {
			builder.WriteString(fmt.Sprintf("line %d: %s", err.Line, err.Message))
		}
	}

	// Add debug trace if available and enabled
	if e.Debug != nil && e.Debug.Enabled && e.Debug.HasTrace() {
		builder.WriteString(e.Debug.String())
	}

	return builder.String()
}

func NewValidationError(debug *DebugTrace) *ValidationError {
	return &ValidationError{
		Errors: []ValidationErrorEntry{},
		Debug:  debug,
	}
}

func (e *ValidationError) Add(line int, column int, context string, format string, args ...interface{}) {
	e.Errors = append(e.Errors, ValidationErrorEntry{
		Line:    line,
		Column:  column,
		Context: context,
		Message: fmt.Sprintf(format, args...),
	})
}

func (e *ValidationError) AddSimple(line int, format string, args ...interface{}) {
	e.Errors = append(e.Errors, ValidationErrorEntry{
		Line:    line,
		Message: fmt.Sprintf(format, args...),
	})
}

func (e *ValidationError) HasErrors() bool {
	return len(e.Errors) > 0
}

func Validate(file *CommandFile) error {
	return ValidateWithDebug(file, nil)
}

func ValidateWithDebug(file *CommandFile, debug *DebugTrace) error {
	validationError := NewValidationError(debug)

	if debug != nil {
		debug.Log("Validation complete - no variable reference checks needed")
	}

	if validationError.HasErrors() {
		return validationError
	}

	return nil
}
