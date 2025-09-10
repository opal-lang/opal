package codegen

import (
	"fmt"
	"strconv"
	"strings"
)

// Common helper functions for code generation

// QuoteString safely quotes a string for code generation
func QuoteString(s string) string {
	return strconv.Quote(s)
}

// SanitizeIdentifier converts a name to a valid identifier
func SanitizeIdentifier(name string) string {
	result := make([]rune, 0, len(name))
	for i, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (i > 0 && r >= '0' && r <= '9') || r == '_' {
			result = append(result, r)
		} else {
			result = append(result, '_')
		}
	}
	return string(result)
}

// FormatArgs converts decorator parameters to a code representation
func FormatArgs(args []DecoratorParam) string {
	if len(args) == 0 {
		return "nil"
	}

	var parts []string
	for _, arg := range args {
		if arg.Name != "" {
			// Named parameter
			parts = append(parts, fmt.Sprintf("%s: %s", arg.Name, formatValue(arg.Value)))
		} else {
			// Positional parameter
			parts = append(parts, formatValue(arg.Value))
		}
	}

	return fmt.Sprintf("[]DecoratorParam{%s}", strings.Join(parts, ", "))
}

// formatValue converts a generic value to a code representation
func formatValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return QuoteString(v)
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%v", v)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%v", v)
	case float32, float64:
		return fmt.Sprintf("%v", v)
	case bool:
		return fmt.Sprintf("%t", v)
	default:
		return fmt.Sprintf("%#v", v)
	}
}

// IndentCode adds indentation to generated code
func IndentCode(code string, level int) string {
	if code == "" {
		return code
	}

	indent := strings.Repeat("\t", level)
	lines := strings.Split(code, "\n")

	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			lines[i] = indent + line
		}
	}

	return strings.Join(lines, "\n")
}

// JoinResults combines multiple temp results with a separator
func JoinResults(results []TempResult, separator string) string {
	if len(results) == 0 {
		return ""
	}

	var parts []string
	for _, result := range results {
		parts = append(parts, result.String())
	}

	return strings.Join(parts, separator)
}
