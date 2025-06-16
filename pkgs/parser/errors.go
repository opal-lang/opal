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

	// Build variable names map
	varNames := make(map[string]bool)
	for _, def := range file.Definitions {
		varNames[def.Name] = true
	}

	// Build command maps for validation
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

	// Variable reference checker function
	checkVarReferences := func(text string, line int, lineContent string) {
		if debug != nil {
			debug.Log("Checking variables in: %s", text)
		}

		var inVar bool
		var varName strings.Builder

		for i := 0; i < len(text); i++ {
			if !inVar {
				// Skip escaped dollar signs
				if i+1 < len(text) && text[i] == '\\' && text[i+1] == '$' {
					i += 1
					continue
				}

				// Look for variable start
				if i+1 < len(text) && text[i] == '$' && text[i+1] == '(' {
					inVar = true
					varName.Reset()
					i++
				}
			} else {
				if text[i] == ')' {
					name := varName.String()
					if debug != nil {
						debug.Log("Found variable reference: %s", name)
					}
					if !varNames[name] {
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

	// Validate variable references in commands
	for _, cmd := range file.Commands {
		lineContent := ""
		if cmd.Line > 0 && cmd.Line <= len(file.Lines) {
			lineContent = file.Lines[cmd.Line-1]
		}

		if !cmd.IsBlock {
			checkVarReferences(cmd.Command, cmd.Line, lineContent)
		} else {
			checkBlockStatements(cmd.Block, cmd.Line, lineContent, checkVarReferences, debug)
		}
	}

	if validationError.HasErrors() {
		return validationError
	}

	return nil
}

func checkBlockStatements(statements []BlockStatement, line int, lineContent string,
	checkVarReferences func(string, int, string), debug *DebugTrace,
) {
	for i, stmt := range statements {
		if debug != nil {
			debug.Log("Checking block statement %d: decorated=%v", i, stmt.IsDecorated)
		}

		if stmt.IsDecorated {
			switch stmt.DecoratorType {
			case "function", "simple":
				if stmt.Command != "" {
					checkVarReferences(stmt.Command, line, lineContent)
				}
			case "block":
				checkBlockStatements(stmt.DecoratedBlock, line, lineContent, checkVarReferences, debug)
			}
		} else {
			if stmt.Command != "" {
				checkVarReferences(stmt.Command, line, lineContent)
			}
		}
	}
}
