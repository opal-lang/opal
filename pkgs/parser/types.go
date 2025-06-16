package parser

import (
	"fmt"
	"strings"
)

// BlockStatement represents a statement within a block command
// Supports both regular commands and decorated commands
type BlockStatement struct {
	// For regular commands
	Command string // The command text to execute

	// For decorated commands
	IsDecorated    bool             // Whether this is a decorated command
	Decorator      string           // The decorator name (sh, parallel, retry, etc.)
	DecoratorType  string           // "function", "simple", or "block"
	DecoratedBlock []BlockStatement // For block-type decorators like @parallel: { }
}

// Helper methods for BlockStatement
func (bs *BlockStatement) IsFunction() bool {
	return bs.IsDecorated && bs.DecoratorType == "function"
}

func (bs *BlockStatement) IsSimpleDecorator() bool {
	return bs.IsDecorated && bs.DecoratorType == "simple"
}

func (bs *BlockStatement) IsBlockDecorator() bool {
	return bs.IsDecorated && bs.DecoratorType == "block"
}

func (bs *BlockStatement) GetCommand() string {
	return bs.Command
}

func (bs *BlockStatement) GetDecorator() string {
	return bs.Decorator
}

func (bs *BlockStatement) GetNestedBlock() []BlockStatement {
	return bs.DecoratedBlock
}

// Definition represents a variable definition in the command file
type Definition struct {
	Name  string // The variable name
	Value string // The variable value
	Line  int    // The line number in the source file
}

// Command represents a command definition in the command file
type Command struct {
	Name    string           // The command name
	Command string           // The command text for simple commands
	Line    int              // The line number in the source file
	IsWatch bool             // Whether this is a watch command
	IsStop  bool             // Whether this is a stop command
	IsBlock bool             // Whether this is a block command
	Block   []BlockStatement // The statements for block commands
}

// CommandFile represents the parsed command file
type CommandFile struct {
	Definitions []Definition // All variable definitions
	Commands    []Command    // All command definitions
	Lines       []string     // Original file lines for error reporting
}

// ExpandVariables expands variable references in commands
func (cf *CommandFile) ExpandVariables() error {
	// Create lookup map for variables
	vars := make(map[string]string)
	for _, def := range cf.Definitions {
		vars[def.Name] = def.Value
	}

	// Expand variables in simple commands
	for i := range cf.Commands {
		cmd := &cf.Commands[i]
		if !cmd.IsBlock {
			expanded, err := expandVariablesInText(cmd.Command, vars, cmd.Line)
			if err != nil {
				return err
			}
			cmd.Command = expanded
		} else {
			// Expand variables in block statements
			if err := cf.expandVariablesInBlockStatements(cmd.Block, vars, cmd.Line); err != nil {
				return err
			}
		}
	}

	return nil
}

// expandVariablesInBlockStatements handles variable expansion in block statements
func (cf *CommandFile) expandVariablesInBlockStatements(statements []BlockStatement, vars map[string]string, line int) error {
	for i := range statements {
		stmt := &statements[i]

		if stmt.IsDecorated {
			// Handle decorated commands
			switch stmt.DecoratorType {
			case "function", "simple":
				// Expand variables in the command text
				if stmt.Command != "" {
					expanded, err := expandVariablesInText(stmt.Command, vars, line)
					if err != nil {
						return err
					}
					stmt.Command = expanded
				}
			case "block":
				// Recursively expand variables in nested block
				if err := cf.expandVariablesInBlockStatements(stmt.DecoratedBlock, vars, line); err != nil {
					return err
				}
			}
		} else {
			// Handle regular commands
			if stmt.Command != "" {
				expanded, err := expandVariablesInText(stmt.Command, vars, line)
				if err != nil {
					return err
				}
				stmt.Command = expanded
			}
		}
	}
	return nil
}

// expandVariablesInText replaces $(name) in a string with its value
// Also converts \\; (devcmd input syntax) to \; (shell output syntax)
// Only processes devcmd escapes defined in the grammar: \\ \n \r \t \$ \{ \} \( \) \"
func expandVariablesInText(text string, vars map[string]string, line int) (string, error) {
	// First convert \\; (devcmd syntax) to \; (shell syntax)
	text = strings.ReplaceAll(text, "\\\\;", "\\;")

	var result []byte
	var varName []byte
	inVar := false
	i := 0

	for i < len(text) {
		if text[i] == '\\' && i+1 < len(text) {
			// Check if this is a devcmd escape sequence (not shell escaping)
			nextChar := text[i+1]
			switch nextChar {
			case '\\':
				result = append(result, '\\')
				i += 2
			case 'n':
				result = append(result, '\n')
				i += 2
			case 'r':
				result = append(result, '\r')
				i += 2
			case 't':
				result = append(result, '\t')
				i += 2
			case '$':
				result = append(result, '$')
				i += 2
			case '{':
				result = append(result, '{')
				i += 2
			case '}':
				result = append(result, '}')
				i += 2
			case '(':
				result = append(result, '(')
				i += 2
			case ')':
				result = append(result, ')')
				i += 2
			case '"':
				result = append(result, '"')
				i += 2
			case 'x':
				// Hex escape: \xXX
				if i+3 < len(text) && isHexDigit(text[i+2]) && isHexDigit(text[i+3]) {
					hex := text[i+2 : i+4]
					if val, err := parseHexByte(hex); err == nil {
						result = append(result, val)
						i += 4
					} else {
						// Not valid hex, preserve as-is
						result = append(result, '\\')
						i++
					}
				} else {
					// Not valid hex escape, preserve as-is
					result = append(result, '\\')
					i++
				}
			case 'u':
				// Unicode escape: \uXXXX
				if i+5 < len(text) &&
					isHexDigit(text[i+2]) && isHexDigit(text[i+3]) &&
					isHexDigit(text[i+4]) && isHexDigit(text[i+5]) {
					hex := text[i+2 : i+6]
					if val, err := parseHexRune(hex); err == nil {
						result = append(result, []byte(string(val))...)
						i += 6
					} else {
						// Not valid unicode, preserve as-is
						result = append(result, '\\')
						i++
					}
				} else {
					// Not valid unicode escape, preserve as-is
					result = append(result, '\\')
					i++
				}
			default:
				// Not a devcmd escape - preserve both backslash and next char for shell
				result = append(result, '\\')
				i++
			}
		} else if !inVar {
			// Look for variable start: $(
			if i+1 < len(text) && text[i] == '$' && text[i+1] == '(' {
				inVar = true
				varName = varName[:0] // Reset variable name buffer
				i += 2                // Skip '$('
			} else {
				result = append(result, text[i])
				i++
			}
		} else {
			// Inside variable reference
			if text[i] == ')' {
				// End of variable reference
				name := string(varName)
				if value, ok := vars[name]; ok {
					// Process escape sequences in the variable value as well
					processedValue, err := processEscapeSequences(value)
					if err != nil {
						return "", &ParseError{
							Line:    line,
							Message: fmt.Sprintf("error processing escapes in variable %s: %v", name, err),
						}
					}
					result = append(result, processedValue...)
				} else {
					return "", &ParseError{
						Line:    line,
						Message: fmt.Sprintf("undefined variable: %s", name),
					}
				}
				inVar = false
				i++
			} else {
				varName = append(varName, text[i])
				i++
			}
		}
	}

	// Check for unclosed variable reference
	if inVar {
		return "", &ParseError{
			Line:    line,
			Message: fmt.Sprintf("unclosed variable reference: $(%s", string(varName)),
		}
	}

	return string(result), nil
}

// processEscapeSequences processes escape sequences in variable values
func processEscapeSequences(text string) ([]byte, error) {
	var result []byte
	i := 0

	for i < len(text) {
		if text[i] == '\\' && i+1 < len(text) {
			nextChar := text[i+1]
			switch nextChar {
			case '\\':
				result = append(result, '\\')
				i += 2
			case 'n':
				result = append(result, '\n')
				i += 2
			case 'r':
				result = append(result, '\r')
				i += 2
			case 't':
				result = append(result, '\t')
				i += 2
			case '$':
				result = append(result, '$')
				i += 2
			case '{':
				result = append(result, '{')
				i += 2
			case '}':
				result = append(result, '}')
				i += 2
			case '(':
				result = append(result, '(')
				i += 2
			case ')':
				result = append(result, ')')
				i += 2
			case '"':
				result = append(result, '"')
				i += 2
			case 'x':
				// Hex escape: \xXX
				if i+3 < len(text) && isHexDigit(text[i+2]) && isHexDigit(text[i+3]) {
					hex := text[i+2 : i+4]
					if val, err := parseHexByte(hex); err == nil {
						result = append(result, val)
						i += 4
					} else {
						// Not valid hex, preserve as-is
						result = append(result, '\\')
						i++
					}
				} else {
					// Not valid hex escape, preserve as-is
					result = append(result, '\\')
					i++
				}
			case 'u':
				// Unicode escape: \uXXXX
				if i+5 < len(text) &&
					isHexDigit(text[i+2]) && isHexDigit(text[i+3]) &&
					isHexDigit(text[i+4]) && isHexDigit(text[i+5]) {
					hex := text[i+2 : i+6]
					if val, err := parseHexRune(hex); err == nil {
						result = append(result, []byte(string(val))...)
						i += 6
					} else {
						// Not valid unicode, preserve as-is
						result = append(result, '\\')
						i++
					}
				} else {
					// Not valid unicode escape, preserve as-is
					result = append(result, '\\')
					i++
				}
			default:
				// Not a devcmd escape - preserve both backslash and next char
				result = append(result, '\\')
				i++
			}
		} else {
			result = append(result, text[i])
			i++
		}
	}

	return result, nil
}

// isHexDigit checks if a character is a valid hexadecimal digit
func isHexDigit(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

// parseHexByte parses a 2-character hex string into a byte
func parseHexByte(hex string) (byte, error) {
	if len(hex) != 2 {
		return 0, fmt.Errorf("invalid hex length: expected 2, got %d", len(hex))
	}

	var result byte
	for _, c := range []byte(hex) {
		result <<= 4
		if c >= '0' && c <= '9' {
			result |= c - '0'
		} else if c >= 'a' && c <= 'f' {
			result |= c - 'a' + 10
		} else if c >= 'A' && c <= 'F' {
			result |= c - 'A' + 10
		} else {
			return 0, fmt.Errorf("invalid hex character: %c", c)
		}
	}
	return result, nil
}

// parseHexRune parses a 4-character hex string into a rune
func parseHexRune(hex string) (rune, error) {
	if len(hex) != 4 {
		return 0, fmt.Errorf("invalid hex length: expected 4, got %d", len(hex))
	}

	var result rune
	for _, c := range []byte(hex) {
		result <<= 4
		if c >= '0' && c <= '9' {
			result |= rune(c - '0')
		} else if c >= 'a' && c <= 'f' {
			result |= rune(c - 'a' + 10)
		} else if c >= 'A' && c <= 'F' {
			result |= rune(c - 'A' + 10)
		} else {
			return 0, fmt.Errorf("invalid hex character: %c", c)
		}
	}
	return result, nil
}
