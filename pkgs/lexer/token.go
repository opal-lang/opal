package lexer

import (
	"fmt"
	"strings"
)

// TokenType represents the type of token in Devcmd
type TokenType int

const (
	// Special tokens
	EOF TokenType = iota
	ILLEGAL

	// Language structure tokens
	VAR      // var
	WATCH    // watch
	STOP     // stop
	WHEN     // when (for @when pattern decorator)
	TRY      // try (for @try pattern decorator)
	AT       // @
	COLON    // :
	EQUALS   // =
	COMMA    // ,
	LPAREN   // (
	RPAREN   // )
	LBRACE   // {
	RBRACE   // }
	ASTERISK // * (wildcard in patterns)

	// Literals and Content
	IDENTIFIER // command names, variable names, decorator names, patterns
	SHELL_TEXT // shell command text
	NUMBER     // 8080, 3.14, -100
	STRING     // "hello", 'world', `template`
	DURATION   // 30s, 5m, 1h
	BOOLEAN    // true, false

	// Structure
	// NEWLINE removed - handled as whitespace

	// Comments
	COMMENT           // #
	MULTILINE_COMMENT // /* */
)

// Pre-computed token name lookup for fast debugging
var tokenNames = [...]string{
	EOF:        "EOF",
	ILLEGAL:    "ILLEGAL",
	VAR:        "VAR",
	WATCH:      "WATCH",
	STOP:       "STOP",
	WHEN:       "WHEN",
	TRY:        "TRY",
	AT:         "AT",
	COLON:      "COLON",
	EQUALS:     "EQUALS",
	COMMA:      "COMMA",
	LPAREN:     "LPAREN",
	RPAREN:     "RPAREN",
	LBRACE:     "LBRACE",
	RBRACE:     "RBRACE",
	ASTERISK:   "ASTERISK",
	IDENTIFIER: "IDENTIFIER",
	SHELL_TEXT: "SHELL_TEXT",
	NUMBER:     "NUMBER",
	STRING:     "STRING",
	DURATION:   "DURATION",
	BOOLEAN:    "BOOLEAN",
	// NEWLINE removed
	COMMENT:           "COMMENT",
	MULTILINE_COMMENT: "MULTILINE_COMMENT",
}

func (t TokenType) String() string {
	if int(t) < len(tokenNames) && int(t) >= 0 {
		return tokenNames[t]
	}
	return fmt.Sprintf("TokenType(%d)", int(t))
}

// StringType represents the type of string literal
type StringType int

const (
	DoubleQuoted StringType = iota // "string"
	SingleQuoted                   // 'string'
	Backtick                       // `string`
)

// SemanticTokenType represents semantic categories for syntax highlighting
type SemanticTokenType int

const (
	SemKeyword   SemanticTokenType = iota // var, watch, stop, when, try
	SemCommand                            // command names
	SemVariable                           // variable names
	SemString                             // string literals
	SemNumber                             // numeric literals
	SemComment                            // comments
	SemOperator                           // :, =, {, }, (, ), @, *
	SemShellText                          // shell text content
	SemDecorator                          // decorators like @timeout, @retry
	SemPattern                            // pattern-matching decorators (@when, @try)
	SemBoolean                            // boolean literals (true, false)
)

// SourceSpan represents a precise location in source code
type SourceSpan struct {
	Start SourcePosition `json:"start"`
	End   SourcePosition `json:"end"`
}

// SourcePosition represents a position in source code
type SourcePosition struct {
	Line   int `json:"line"`   // 1-based
	Column int `json:"column"` // 1-based
	Offset int `json:"offset"` // 0-based byte offset
}

// ShellSegment represents a portion of shell text with precise positioning
type ShellSegment struct {
	Text    string     `json:"text"`     // The processed text
	Span    SourceSpan `json:"span"`     // Original source location
	RawText string     `json:"raw_text"` // Original raw text (with continuations)
	Offset  int        `json:"offset"`   // Offset within processed shell text
}

// Token represents a single token with enhanced position information
type Token struct {
	Type       TokenType
	Semantic   SemanticTokenType
	Line       int
	Column     int
	EndLine    int
	EndColumn  int
	Value      string
	Raw        string
	StringType StringType

	// Enhanced positioning for shell content
	Span          SourceSpan     `json:"span"`
	ShellSegments []ShellSegment `json:"shell_segments,omitempty"` // For SHELL_TEXT tokens
}

// Position returns a formatted position string for error reporting
func (t Token) Position() string {
	if t.Line == t.EndLine {
		return fmt.Sprintf("%d:%d-%d", t.Line, t.Column, t.EndColumn)
	}
	return fmt.Sprintf("%d:%d-%d:%d", t.Line, t.Column, t.EndLine, t.EndColumn)
}

// GetPositionAt returns the source position for a character offset within this token's value
func (t *Token) GetPositionAt(offset int) SourcePosition {
	if t.Type != SHELL_TEXT || len(t.ShellSegments) == 0 {
		// For non-shell tokens, interpolate position
		return t.interpolatePosition(offset)
	}

	// For shell tokens, find the segment containing this offset
	for _, segment := range t.ShellSegments {
		segmentEnd := segment.Offset + len(segment.Text)
		if offset >= segment.Offset && offset < segmentEnd {
			// Offset is within this segment
			localOffset := offset - segment.Offset
			return t.mapToSourcePosition(segment, localOffset)
		}
	}

	// Fallback to end position
	return t.Span.End
}

// interpolatePosition calculates position for simple tokens
func (t *Token) interpolatePosition(offset int) SourcePosition {
	if offset <= 0 {
		return t.Span.Start
	}
	if offset >= len(t.Value) {
		return t.Span.End
	}

	// Simple interpolation for single-line tokens
	if t.Line == t.EndLine {
		return SourcePosition{
			Line:   t.Line,
			Column: t.Column + offset,
			Offset: t.Span.Start.Offset + offset,
		}
	}

	// For multi-line tokens, we need to count newlines
	lines := strings.Split(t.Value[:offset], "\n")
	if len(lines) == 1 {
		return SourcePosition{
			Line:   t.Line,
			Column: t.Column + offset,
			Offset: t.Span.Start.Offset + offset,
		}
	}

	return SourcePosition{
		Line:   t.Line + len(lines) - 1,
		Column: len(lines[len(lines)-1]) + 1,
		Offset: t.Span.Start.Offset + offset,
	}
}

// mapToSourcePosition maps from processed text position back to original source
func (t *Token) mapToSourcePosition(segment ShellSegment, localOffset int) SourcePosition {
	// Handle line continuations and other transformations
	if segment.RawText == segment.Text {
		// No transformations, direct mapping
		return SourcePosition{
			Line:   segment.Span.Start.Line,
			Column: segment.Span.Start.Column + localOffset,
			Offset: segment.Span.Start.Offset + localOffset,
		}
	}

	// Complex mapping for line continuations
	return t.mapThroughTransformations(segment, localOffset)
}

// mapThroughTransformations handles complex position mapping through text transformations
func (t *Token) mapThroughTransformations(segment ShellSegment, localOffset int) SourcePosition {
	// Walk through the raw text and processed text simultaneously
	rawPos := 0
	processedPos := 0
	line := segment.Span.Start.Line
	column := segment.Span.Start.Column

	for rawPos < len(segment.RawText) && processedPos < localOffset {
		if rawPos < len(segment.RawText)-1 &&
			segment.RawText[rawPos] == '\\' &&
			segment.RawText[rawPos+1] == '\n' {
			// Line continuation: \\ + \n becomes space in processed text
			rawPos += 2 // Skip \\ and \n
			line++
			column = 1

			// Skip any following whitespace in raw text
			for rawPos < len(segment.RawText) &&
				(segment.RawText[rawPos] == ' ' || segment.RawText[rawPos] == '\t') {
				rawPos++
				column++
			}

			processedPos++ // Advance processed position (for the space)
		} else {
			// Normal character
			if segment.RawText[rawPos] == '\n' {
				line++
				column = 1
			} else {
				column++
			}
			rawPos++
			processedPos++
		}
	}

	return SourcePosition{
		Line:   line,
		Column: column,
		Offset: segment.Span.Start.Offset + rawPos,
	}
}

// GetErrorPosition returns a formatted error message with precise position
func (t *Token) GetErrorPosition(message string, offset int) string {
	pos := t.GetPositionAt(offset)
	return fmt.Sprintf("%s at %d:%d", message, pos.Line, pos.Column)
}

// ToLSPSemanticToken converts to Language Server Protocol format
func (t Token) ToLSPSemanticToken() LSPSemanticToken {
	return LSPSemanticToken{
		Line:      uint32(t.Line - 1),
		Character: uint32(t.Column - 1),
		Length:    uint32(len(t.Value)),
		TokenType: uint32(t.Semantic),
	}
}

// LSPSemanticToken represents a token in LSP format
type LSPSemanticToken struct {
	Line      uint32
	Character uint32
	Length    uint32
	TokenType uint32
}

// GetSemanticTokens extracts all tokens with semantic information for syntax highlighting
func GetSemanticTokens(input string) ([]Token, error) {
	lexer := New(input)
	tokens := lexer.TokenizeToSlice()
	if len(tokens) > 0 && tokens[len(tokens)-1].Type == EOF {
		tokens = tokens[:len(tokens)-1]
	}
	return tokens, nil
}

// ToLSPSemanticTokensArray converts tokens to LSP semantic tokens array format
func ToLSPSemanticTokensArray(tokens []Token) []uint32 {
	if len(tokens) == 0 {
		return []uint32{}
	}
	result := make([]uint32, 0, len(tokens)*5)
	var prevLine, prevChar uint32
	for _, token := range tokens {
		line := uint32(token.Line - 1)
		char := uint32(token.Column - 1)
		length := uint32(len(token.Value))
		tokenType := uint32(token.Semantic)
		deltaLine := line - prevLine
		var deltaChar uint32
		if deltaLine == 0 {
			deltaChar = char - prevChar
		} else {
			deltaChar = char
		}
		result = append(result, deltaLine, deltaChar, length, tokenType, 0)
		prevLine = line
		prevChar = char
	}
	return result
}

// GetTextMateGrammarScopes returns all unique TextMate scopes used
func GetTextMateGrammarScopes(tokens []Token) []string {
	scopes := make(map[string]bool)
	for _, token := range tokens {
		switch token.Type {
		case VAR, WATCH, STOP:
			scopes["keyword.control.devcmd"] = true
		case WHEN, TRY:
			scopes["keyword.control.pattern.devcmd"] = true
		case STRING:
			scopes["string.quoted.devcmd"] = true
		case NUMBER:
			scopes["constant.numeric.devcmd"] = true
		case DURATION:
			scopes["constant.numeric.duration.devcmd"] = true
		case BOOLEAN:
			scopes["constant.language.boolean.devcmd"] = true
		case COMMENT:
			scopes["comment.line.hash.devcmd"] = true
		case MULTILINE_COMMENT:
			scopes["comment.block.devcmd"] = true
		case SHELL_TEXT:
			scopes["source.shell.embedded.devcmd"] = true
		case IDENTIFIER:
			scopes["entity.name.function.devcmd"] = true
		case AT:
			scopes["punctuation.definition.decorator.devcmd"] = true
		case ASTERISK:
			scopes["keyword.operator.wildcard.devcmd"] = true
		}
	}

	result := make([]string, 0, len(scopes))
	for scope := range scopes {
		result = append(result, scope)
	}
	return result
}

// TokenClassification provides helper methods for token analysis

// IsStructuralToken checks if a token represents Devcmd structure
func IsStructuralToken(tokenType TokenType) bool {
	switch tokenType {
	case VAR, WATCH, STOP, WHEN, TRY, AT, COLON, EQUALS, COMMA, LPAREN, RPAREN, LBRACE, RBRACE, ASTERISK:
		return true
	default:
		return false
	}
}

// IsLiteralToken checks if a token represents a literal value
func IsLiteralToken(tokenType TokenType) bool {
	switch tokenType {
	case STRING, NUMBER, DURATION, IDENTIFIER, BOOLEAN:
		return true
	default:
		return false
	}
}

// IsShellContent checks if a token represents shell content
func IsShellContent(tokenType TokenType) bool {
	return tokenType == SHELL_TEXT
}

// IsPatternToken checks if a token is related to pattern-matching decorators
func IsPatternToken(tokenType TokenType) bool {
	switch tokenType {
	case WHEN, TRY, ASTERISK:
		return true
	default:
		return false
	}
}

// IsKeywordToken checks if a token is a language keyword
func IsKeywordToken(tokenType TokenType) bool {
	switch tokenType {
	case VAR, WATCH, STOP, WHEN, TRY:
		return true
	default:
		return false
	}
}

// IsDecoratorKeyword checks if a token is a decorator keyword
func IsDecoratorKeyword(tokenType TokenType) bool {
	switch tokenType {
	case WHEN, TRY:
		return true
	default:
		return false
	}
}

// IsPatternIdentifier checks if an identifier token could be a pattern
func IsPatternIdentifier(token Token) bool {
	if token.Type != IDENTIFIER {
		return false
	}

	// Common pattern identifiers
	commonPatterns := map[string]bool{
		"main":        true,
		"error":       true,
		"finally":     true,
		"production":  true,
		"development": true,
		"staging":     true,
		"test":        true,
		"prod":        true,
		"dev":         true,
	}

	return commonPatterns[token.Value]
}

// GetPatternType returns the type of pattern for a token
func GetPatternType(token Token) PatternType {
	if token.Type == ASTERISK {
		return WildcardPattern
	}

	if token.Type == IDENTIFIER {
		switch token.Value {
		case "main", "error", "finally":
			return TryPattern
		case "production", "development", "staging", "test", "prod", "dev":
			return WhenPattern
		default:
			return CustomPattern
		}
	}

	return UnknownPattern
}

// PatternType represents different types of patterns
type PatternType int

const (
	UnknownPattern  PatternType = iota
	WildcardPattern             // *
	WhenPattern                 // production, development, test, etc.
	TryPattern                  // main, error, finally
	CustomPattern               // user-defined patterns
)

func (pt PatternType) String() string {
	switch pt {
	case WildcardPattern:
		return "wildcard"
	case WhenPattern:
		return "when"
	case TryPattern:
		return "try"
	case CustomPattern:
		return "custom"
	default:
		return "unknown"
	}
}

// ValidatePatternSequence validates a sequence of pattern tokens
func ValidatePatternSequence(tokens []Token, decoratorType string) []PatternError {
	var errors []PatternError

	patterns := make(map[string]Token)
	hasWildcard := false

	for _, token := range tokens {
		switch token.Type {
		case ASTERISK:
			if hasWildcard {
				errors = append(errors, PatternError{
					Message: "multiple wildcard patterns not allowed",
					Token:   token,
					Code:    "duplicate-wildcard",
				})
			}
			hasWildcard = true
		case IDENTIFIER:
			if existing, exists := patterns[token.Value]; exists {
				errors = append(errors, PatternError{
					Message: fmt.Sprintf("duplicate pattern '%s'", token.Value),
					Token:   token,
					Code:    "duplicate-pattern",
					Related: &existing,
				})
			}
			patterns[token.Value] = token
		}
	}

	// Decorator-specific validation
	switch decoratorType {
	case "try":
		if _, hasMain := patterns["main"]; !hasMain {
			errors = append(errors, PatternError{
				Message: "@try decorator requires 'main' pattern",
				Code:    "missing-main-pattern",
			})
		}
	}

	return errors
}

// PatternError represents a pattern validation error
type PatternError struct {
	Message string `json:"message"`
	Token   Token  `json:"token"`
	Code    string `json:"code"`
	Related *Token `json:"related,omitempty"`
}

func (pe PatternError) Error() string {
	return fmt.Sprintf("%s at %s", pe.Message, pe.Token.Position())
}
