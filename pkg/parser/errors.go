package parser

import (
	"fmt"
)

// ParseError represents an error that occurred during parsing
type ParseError struct {
	Line    int
	Message string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("line %d: %s", e.Line, e.Message)
}

// NewParseError creates a new ParseError
func NewParseError(line int, format string, args ...interface{}) *ParseError {
	return &ParseError{
		Line:    line,
		Message: fmt.Sprintf(format, args...),
	}
}
