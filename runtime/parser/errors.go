package parser

import (
	"fmt"
	"strings"

	"github.com/opal-lang/opal/runtime/lexer"
)

// ErrorFormatter formats parse errors for user-friendly output
type ErrorFormatter struct {
	Source   []byte // Original source for context
	Filename string // Filename to display
	Compact  bool   // Use compact format (default: detailed)
	Color    bool   // Use ANSI color codes (default: false)
}

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
)

// Format produces user-friendly error output
func (f *ErrorFormatter) Format(err ParseError) string {
	if f.Compact {
		return f.formatCompact(err)
	}
	return f.formatDetailed(err)
}

// formatCompact produces concise error output
// Format: "file:line:col: message"
func (f *ErrorFormatter) formatCompact(err ParseError) string {
	var b strings.Builder

	// Location: file:line:col
	f.writeLocation(&b, err)
	b.WriteString(": ")

	// Message
	f.colorize(&b, colorRed, err.Message)
	if err.Context != "" {
		fmt.Fprintf(&b, " in %s", err.Context)
	}
	b.WriteString("\n")

	// Source snippet with caret
	f.writeSourceSnippet(&b, err, true)

	// Suggestion
	if err.Suggestion != "" {
		b.WriteString("   ")
		f.colorize(&b, colorBlue, err.Suggestion)
		b.WriteString("\n")
	}

	return b.String()
}

// formatDetailed produces Rust-style detailed error output
func (f *ErrorFormatter) formatDetailed(err ParseError) string {
	var b strings.Builder

	// Error header
	f.colorize(&b, colorRed, "Error: "+err.Message)
	b.WriteString("\n")

	// Location with arrow
	b.WriteString("  ")
	f.colorize(&b, colorCyan, "--> ")
	f.writeLocation(&b, err)
	b.WriteString("\n")

	// Source snippet with context
	f.writeSourceSnippet(&b, err, false)

	// Suggestion
	if err.Suggestion != "" {
		b.WriteString("   ")
		f.colorize(&b, colorBlue, "= Suggestion: "+err.Suggestion)
		b.WriteString("\n")
	}

	// Example
	if err.Example != "" {
		b.WriteString("   ")
		f.colorize(&b, colorGreen, "= Example: "+err.Example)
		b.WriteString("\n")
	}

	// Note
	if err.Note != "" {
		b.WriteString("   ")
		f.colorize(&b, colorYellow, "= Note: "+err.Note)
		b.WriteString("\n")
	}

	return b.String()
}

// writeLocation writes file:line:col
func (f *ErrorFormatter) writeLocation(b *strings.Builder, err ParseError) {
	if err.Filename != "" {
		b.WriteString(err.Filename)
		b.WriteString(":")
	}
	fmt.Fprintf(b, "%d:%d", err.Position.Line, err.Position.Column)
}

// writeSourceSnippet writes source line with caret pointer
func (f *ErrorFormatter) writeSourceSnippet(b *strings.Builder, err ParseError, compact bool) {
	sourceLine := f.extractSourceLine(err.Position)
	if sourceLine == "" {
		return
	}

	if !compact {
		b.WriteString("   |\n")
	}

	// Source line with line number
	fmt.Fprintf(b, " %d | %s\n", err.Position.Line, sourceLine)

	// Caret pointer
	fmt.Fprintf(b, "   | %s", strings.Repeat(" ", err.Position.Column-1))

	caretMsg := "^"
	if len(err.Expected) > 0 {
		caretMsg += " expected " + f.formatTokenList(err.Expected)
	} else if err.Got != lexer.EOF {
		caretMsg += " unexpected " + tokenName(err.Got)
	}

	f.colorize(b, colorRed, caretMsg)
	b.WriteString("\n")

	if !compact {
		b.WriteString("   |\n")
	}
}

// colorize writes text with optional color
func (f *ErrorFormatter) colorize(b *strings.Builder, color, text string) {
	if f.Color {
		b.WriteString(color)
	}
	b.WriteString(text)
	if f.Color {
		b.WriteString(colorReset)
	}
}

// extractSourceLine gets the source line for display
func (f *ErrorFormatter) extractSourceLine(pos lexer.Position) string {
	if len(f.Source) == 0 {
		return ""
	}

	// Find the start of the target line
	lineStart := 0
	currentLine := 1
	for i := 0; i < len(f.Source); i++ {
		if currentLine == pos.Line {
			lineStart = i
			break
		}
		if f.Source[i] == '\n' {
			currentLine++
		}
	}

	// Find line end
	lineEnd := lineStart
	for lineEnd < len(f.Source) && f.Source[lineEnd] != '\n' {
		lineEnd++
	}

	if lineStart >= len(f.Source) {
		return ""
	}

	return string(f.Source[lineStart:lineEnd])
}

// formatTokenList formats expected tokens: "a", "a or b", "a, b, or c"
func (f *ErrorFormatter) formatTokenList(tokens []lexer.TokenType) string {
	if len(tokens) == 0 {
		return ""
	}
	if len(tokens) == 1 {
		return tokenName(tokens[0])
	}

	var parts []string
	for _, t := range tokens {
		parts = append(parts, tokenName(t))
	}

	if len(parts) == 2 {
		return parts[0] + " or " + parts[1]
	}

	// More than 2: "a, b, or c"
	last := parts[len(parts)-1]
	rest := strings.Join(parts[:len(parts)-1], ", ")
	return rest + ", or " + last
}

// tokenName returns user-friendly token name
func tokenName(t lexer.TokenType) string {
	switch t {
	case lexer.LPAREN:
		return "'('"
	case lexer.RPAREN:
		return "')'"
	case lexer.LBRACE:
		return "'{'"
	case lexer.RBRACE:
		return "'}'"
	case lexer.LSQUARE:
		return "'['"
	case lexer.RSQUARE:
		return "']'"
	case lexer.COMMA:
		return "','"
	case lexer.COLON:
		return "':'"
	case lexer.EQUALS:
		return "'='"
	case lexer.SEMICOLON:
		return "';'"
	case lexer.IDENTIFIER:
		return "identifier"
	case lexer.STRING:
		return "string"
	case lexer.INTEGER:
		return "number"
	case lexer.EOF:
		return "end of file"
	default:
		return t.String()
	}
}
