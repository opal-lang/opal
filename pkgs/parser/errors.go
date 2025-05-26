package parser

import (
	"fmt"
	"strings"
)

// ValidationError checks if commands and definitions are valid
type ValidationError struct {
	Errors []ValidationErrorEntry
}

// ValidationErrorEntry represents a single validation error
type ValidationErrorEntry struct {
	Line    int
	Column  int
	Message string
	Context string
}

// Error formats all validation errors as a single string
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
	return builder.String()
}

// NewValidationError creates a new ValidationError
func NewValidationError() *ValidationError {
	return &ValidationError{
		Errors: []ValidationErrorEntry{},
	}
}

// Add adds a new error message to the validation error
func (e *ValidationError) Add(line int, column int, context string, format string, args ...interface{}) {
	e.Errors = append(e.Errors, ValidationErrorEntry{
		Line:    line,
		Column:  column,
		Context: context,
		Message: fmt.Sprintf(format, args...),
	})
}

// AddSimple adds a simple error message without context
func (e *ValidationError) AddSimple(line int, format string, args ...interface{}) {
	e.Errors = append(e.Errors, ValidationErrorEntry{
		Line:    line,
		Message: fmt.Sprintf(format, args...),
	})
}

// HasErrors returns true if there are validation errors
func (e *ValidationError) HasErrors() bool {
	return len(e.Errors) > 0
}

// Validate performs semantic validation on a command file
func Validate(file *CommandFile) error {
	validationError := NewValidationError()

	// Create variable name lookup
	varNames := make(map[string]bool)
	for _, def := range file.Definitions {
		varNames[def.Name] = true
	}

	// 1. Check for matching watch/stop commands
	watchCmds := make(map[string]int)
	stopCmds := make(map[string]int)
	for _, cmd := range file.Commands {
		name := strings.TrimPrefix(cmd.Name, ".")
		if cmd.IsWatch {
			watchCmds[name] = cmd.Line
		}
		if cmd.IsStop {
			stopCmds[name] = cmd.Line
		}
	}

	// We no longer enforce validation requirements for watch/stop commands
	// Watch commands don't need matching stop commands (stop is optional)
	// Stop commands without matching watch commands will be ignored

	// 2. Check for variable references in command text
	checkVarReferences := func(text string, line int, lineContent string) {
		// Find all $(var) references in text, but skip escaped sequences
		var inVar bool
		var varName strings.Builder

		for i := 0; i < len(text); i++ {
			if !inVar {
				// Check for escaped dollar sign \$ - skip over it
				if i+1 < len(text) && text[i] == '\\' && text[i+1] == '$' {
					i += 1 // Skip both '\' and '$', will be incremented by for loop
					continue
				}

				// Look for unescaped variable reference $(
				if i+1 < len(text) && text[i] == '$' && text[i+1] == '(' {
					inVar = true
					varName.Reset()
					i++ // Skip the '('
				}
			} else {
				if text[i] == ')' {
					// End of variable reference
					name := varName.String()
					if !varNames[name] {
						// Find the position of this variable in the original line
						varPos := strings.Index(lineContent, "$("+name+")")
						if varPos >= 0 {
							validationError.Add(line, varPos, lineContent,
								"undefined variable '%s'", name)
						} else {
							validationError.AddSimple(line, "undefined variable '%s'", name)
						}
					}
					inVar = false
				} else {
					varName.WriteByte(text[i])
				}
			}
		}

		if inVar {
			validationError.AddSimple(line, "unclosed variable reference at line %d", line)
		}
	}

	// Check variables in commands
	for _, cmd := range file.Commands {
		lineContent := ""
		if cmd.Line > 0 && cmd.Line <= len(file.Lines) {
			lineContent = file.Lines[cmd.Line-1]
		}

		if !cmd.IsBlock {
			checkVarReferences(cmd.Command, cmd.Line, lineContent)
		} else {
			for _, stmt := range cmd.Block {
				checkVarReferences(stmt.Command, cmd.Line, lineContent)
			}
		}
	}

	if validationError.HasErrors() {
		return validationError
	}

	return nil
}
