package v2

import (
	"time"
	"unicode/utf8"
)

// LexerMode represents the lexing mode
type LexerMode int

const (
	ModeCommand LexerMode = iota // Command mode: organized tasks (commands.cli)
	ModeScript                   // Script mode: direct execution scripts
)

// LexerOpt represents a lexer configuration option
type LexerOpt func(*LexerConfig)

// TelemetryMode controls telemetry collection (production-safe)
type TelemetryMode int

const (
	TelemetryOff    TelemetryMode = iota // Zero overhead (default)
	TelemetryBasic                       // Token counts only
	TelemetryTiming                      // Token counts + timing per type
)

// DebugLevel controls debug tracing (development only)
type DebugLevel int

const (
	DebugOff      DebugLevel = iota // No debug info (default)
	DebugPaths                      // Method call tracing
	DebugDetailed                   // Character-level tracing
)

// LexerConfig holds lexer configuration
type LexerConfig struct {
	telemetry TelemetryMode
	debug     DebugLevel
	mode      LexerMode
}

// WithTelemetryBasic enables basic telemetry (token counts only)
func WithTelemetryBasic() LexerOpt {
	return func(c *LexerConfig) {
		c.telemetry = TelemetryBasic
	}
}

// WithTelemetryTiming enables timing telemetry (counts + timing per type)
func WithTelemetryTiming() LexerOpt {
	return func(c *LexerConfig) {
		c.telemetry = TelemetryTiming
	}
}

// WithDebugPaths enables debug path tracing (development only)
func WithDebugPaths() LexerOpt {
	return func(c *LexerConfig) {
		c.debug = DebugPaths
	}
}

// WithDebugDetailed enables detailed debug tracing (development only)
func WithDebugDetailed() LexerOpt {
	return func(c *LexerConfig) {
		c.debug = DebugDetailed
	}
}

// WithScriptMode sets the lexer to script mode (direct execution)
func WithScriptMode() LexerOpt {
	return func(c *LexerConfig) {
		c.mode = ModeScript
	}
}

// WithCommandMode sets the lexer to command mode (organized tasks) - default
func WithCommandMode() LexerOpt {
	return func(c *LexerConfig) {
		c.mode = ModeCommand
	}
}

// TokenTelemetry holds per-token type telemetry (production-safe)
type TokenTelemetry struct {
	Type      TokenType
	Count     int
	TotalTime time.Duration
	AvgTime   time.Duration
	MinTime   time.Duration
	MaxTime   time.Duration
}

// DebugEvent holds debug tracing information (development only)
type DebugEvent struct {
	Timestamp time.Time
	Event     string   // "enter_lexNumber", "found_digit", "exit_lexNumber"
	Position  Position // Current lexer position
	Context   string   // Current character, token being built, etc.
}

// Lexer represents the v2 lexer
type Lexer struct {
	// Core lexing state
	input    []byte // Use []byte for zero-allocation performance
	position int
	line     int
	column   int

	// Simple newline state
	lastWasNewline bool // Track if last emitted token was NEWLINE (to skip consecutive ones)

	// Buffering for efficient token access
	tokens     []Token // Internal token buffer
	tokenIndex int     // Current position in buffer
	bufferSize int     // Number of tokens to buffer at once (default: 2500)

	// Telemetry (nil when disabled for zero allocation)
	telemetryMode  TelemetryMode
	tokenTelemetry map[TokenType]*TokenTelemetry // Per-token type telemetry (production safe)

	// Debug (nil when disabled for zero allocation)
	debugLevel  DebugLevel
	debugEvents []DebugEvent // Debug event tracing (development only)
}

// NewLexer creates a new lexer instance with optional configuration
func NewLexer(input string, opts ...LexerOpt) *Lexer {
	config := &LexerConfig{}
	for _, opt := range opts {
		opt(config)
	}

	lexer := &Lexer{
		bufferSize:    2500,                   // Large enough for 90%+ of devcmd files
		tokens:        make([]Token, 0, 2500), // Pre-allocate capacity
		telemetryMode: config.telemetry,       // Default is TelemetryOff (0)
		debugLevel:    config.debug,           // Default is DebugOff (0)
	}

	// Only allocate telemetry structures when needed
	if config.telemetry > TelemetryOff {
		lexer.tokenTelemetry = make(map[TokenType]*TokenTelemetry)
	}

	// Only allocate debug structures when needed
	if config.debug > DebugOff {
		lexer.debugEvents = make([]DebugEvent, 0, 1000) // Pre-allocate debug events
	}

	lexer.Init([]byte(input))
	return lexer
}

// Init resets the lexer with new input (following Go scanner pattern)
func (l *Lexer) Init(input []byte) {
	l.input = input
	l.position = 0
	l.line = 1
	l.column = 1

	// Reset buffering state
	l.tokens = l.tokens[:0] // Reset slice but keep capacity
	l.tokenIndex = 0

	// Reset telemetry stats if enabled
	if l.telemetryMode > TelemetryOff && l.tokenTelemetry != nil {
		// Clear existing stats without reallocating map
		for k := range l.tokenTelemetry {
			delete(l.tokenTelemetry, k)
		}
	}

	// Reset debug events if enabled
	if l.debugLevel > DebugOff && l.debugEvents != nil {
		l.debugEvents = l.debugEvents[:0] // Reset slice but keep capacity
	}
}

// GetTokenTelemetry returns per-token type telemetry (production safe)
func (l *Lexer) GetTokenTelemetry() map[TokenType]*TokenTelemetry {
	if l.telemetryMode == TelemetryOff || l.tokenTelemetry == nil {
		return nil
	}

	// Return a copy to prevent external modification
	result := make(map[TokenType]*TokenTelemetry, len(l.tokenTelemetry))
	for k, v := range l.tokenTelemetry {
		// Copy the telemetry struct
		telemetryCopy := *v
		result[k] = &telemetryCopy
	}
	return result
}

// GetDebugEvents returns debug events (development only)
func (l *Lexer) GetDebugEvents() []DebugEvent {
	if l.debugLevel == DebugOff || l.debugEvents == nil {
		return nil
	}

	// Return a copy to prevent external modification
	result := make([]DebugEvent, len(l.debugEvents))
	copy(result, l.debugEvents)
	return result
}

// NextToken returns the next token using streaming interface
func (l *Lexer) NextToken() Token {
	// Ensure buffer has tokens
	if l.tokenIndex >= len(l.tokens) {
		l.fillBuffer()
	}

	// If still no tokens, return EOF
	if l.tokenIndex >= len(l.tokens) {
		return Token{Type: EOF, Text: nil, Position: Position{Line: l.line, Column: l.column}}
	}

	token := l.tokens[l.tokenIndex]
	l.tokenIndex++
	return token
}

// GetTokens returns all tokens using batch interface
// If tokens have already been consumed via NextToken(), this includes those tokens
// No timing logic - timing is handled by NextToken() calls
func (l *Lexer) GetTokens() []Token {
	var tokens []Token

	// First, collect any tokens already consumed via NextToken()
	for i := 0; i < l.tokenIndex; i++ {
		tokens = append(tokens, l.tokens[i])
	}

	// Then continue collecting remaining tokens via NextToken()
	for {
		token := l.NextToken()
		tokens = append(tokens, token)
		if token.Type == EOF {
			break
		}
	}

	return tokens
}

// fillBuffer fills the internal token buffer with the next batch of tokens
func (l *Lexer) fillBuffer() {
	var start time.Time
	if l.telemetryMode >= TelemetryTiming {
		start = time.Now()
	}

	// Reset buffer but keep capacity
	l.tokens = l.tokens[:0]
	l.tokenIndex = 0

	// Fill buffer up to current capacity
	targetSize := cap(l.tokens)
	for len(l.tokens) < targetSize {
		token := l.nextToken()

		// Check if we need to grow the buffer
		if len(l.tokens) == cap(l.tokens) {
			// Double the capacity for very large files
			newCapacity := cap(l.tokens) * 2
			newTokens := make([]Token, len(l.tokens), newCapacity)
			copy(newTokens, l.tokens)
			l.tokens = newTokens
		}

		l.tokens = append(l.tokens, token)

		if token.Type == EOF {
			break
		}
	}

	// Update timing (accumulate across buffer fills)
	if l.telemetryMode >= TelemetryTiming {
		// TODO: Implement buffer-level timing telemetry
		_ = time.Since(start)
	}
}

// nextToken returns the next token from the input (internal implementation)
func (l *Lexer) nextToken() Token {
	var start time.Time
	if l.telemetryMode >= TelemetryTiming {
		start = time.Now()
	}

	token := l.lexToken() // Do the actual lexing work

	// Record telemetry when enabled
	if l.telemetryMode > TelemetryOff {
		var elapsed time.Duration
		if l.telemetryMode >= TelemetryTiming {
			elapsed = time.Since(start)
		}
		l.recordTokenTelemetry(token.Type, elapsed)
	}

	return token
}

// recordTokenTelemetry records per-token type telemetry (production safe)
func (l *Lexer) recordTokenTelemetry(tokenType TokenType, elapsed time.Duration) {
	telemetry, exists := l.tokenTelemetry[tokenType]
	if !exists {
		// Allocate new telemetry (only when telemetry enabled)
		telemetry = &TokenTelemetry{
			Type:      tokenType,
			Count:     0,
			TotalTime: 0,
			MinTime:   elapsed,
			MaxTime:   elapsed,
		}
		l.tokenTelemetry[tokenType] = telemetry
	}

	telemetry.Count++

	// Update timing if we're collecting it
	if l.telemetryMode >= TelemetryTiming {
		telemetry.TotalTime += elapsed
		telemetry.AvgTime = telemetry.TotalTime / time.Duration(telemetry.Count)

		// Update min/max
		if elapsed < telemetry.MinTime || telemetry.Count == 1 {
			telemetry.MinTime = elapsed
		}
		if elapsed > telemetry.MaxTime || telemetry.Count == 1 {
			telemetry.MaxTime = elapsed
		}
	}
}

// recordDebugEvent records debug events when debug tracing is enabled
func (l *Lexer) recordDebugEvent(event, context string) {
	if l.debugLevel == DebugOff || l.debugEvents == nil {
		return
	}

	l.debugEvents = append(l.debugEvents, DebugEvent{
		Timestamp: time.Now(),
		Event:     event,
		Position:  Position{Line: l.line, Column: l.column},
		Context:   context,
	})
}

// lexToken performs the actual tokenization work
func (l *Lexer) lexToken() Token {
	if l.debugLevel > DebugOff {
		l.recordDebugEvent("enter_lexToken", "starting tokenization")
	}

	// Skip whitespace (except newlines which are significant)
	hadWhitespace := l.skipWhitespace()

	// Handle newlines - emit NEWLINE token, skip consecutive ones
	if l.position < len(l.input) && l.input[l.position] == '\n' {
		if !l.lastWasNewline {
			// Emit first newline
			start := Position{Line: l.line, Column: l.column}
			l.advanceChar() // Consume the newline
			l.lastWasNewline = true
			return Token{
				Type:     NEWLINE,
				Text:     nil, // Self-identifying token
				Position: start,
			}
		} else {
			// Skip consecutive newlines
			for l.position < len(l.input) && l.input[l.position] == '\n' {
				l.advanceChar()
			}
			// After skipping newlines, recurse to continue lexing
			return l.lexToken()
		}
	}

	// Reset newline flag for non-newline tokens
	l.lastWasNewline = false

	// Check for EOF
	if l.position >= len(l.input) {
		if l.debugLevel > DebugOff {
			l.recordDebugEvent("found_EOF", "end of input")
		}
		return Token{
			Type:     EOF,
			Text:     nil,
			Position: Position{Line: l.line, Column: l.column},
		}
	}

	// Capture current position for token
	start := Position{Line: l.line, Column: l.column}
	ch := l.currentChar()
	if l.debugLevel > DebugOff {
		l.recordDebugEvent("current_char", string(ch))
	}

	// Identifier or keyword
	if ch < 128 && isIdentStart[ch] {
		return l.lexIdentifier(start, hadWhitespace)
	}

	// String literals
	if ch == '"' || ch == '\'' || ch == '`' {
		return l.lexString(start, ch, hadWhitespace)
	}

	// Numbers (integers, floats, etc.) - no longer handle negative sign here
	if ch < 128 && isDigit[ch] {
		return l.lexNumber(start, hadWhitespace)
	}

	// Decimal numbers starting with dot (.5, .123)
	if ch == '.' && l.position+1 < len(l.input) && l.input[l.position+1] < 128 && isDigit[l.input[l.position+1]] {
		return l.lexNumber(start, hadWhitespace)
	}

	// Single character punctuation
	switch ch {
	case '=':
		return l.lexEquals(start, hadWhitespace)
	case '<':
		return l.lexLessThan(start, hadWhitespace)
	case '>':
		return l.lexGreaterThan(start, hadWhitespace)
	case '!':
		return l.lexExclamation(start, hadWhitespace)
	case '&':
		return l.lexAmpersand(start, hadWhitespace)
	case '|':
		return l.lexPipe(start, hadWhitespace)
	case ':':
		l.advanceChar()
		return Token{Type: COLON, Text: nil, Position: start, HasSpaceBefore: hadWhitespace}
	case '{':
		l.advanceChar()
		return Token{Type: LBRACE, Text: nil, Position: start, HasSpaceBefore: hadWhitespace}
	case '}':
		l.advanceChar()
		return Token{Type: RBRACE, Text: nil, Position: start, HasSpaceBefore: hadWhitespace}
	case '(':
		l.advanceChar()
		return Token{Type: LPAREN, Text: nil, Position: start}
	case ')':
		l.advanceChar()
		return Token{Type: RPAREN, Text: nil, Position: start}
	case '[':
		l.advanceChar()
		return Token{Type: LSQUARE, Text: nil, Position: start}
	case ']':
		l.advanceChar()
		return Token{Type: RSQUARE, Text: nil, Position: start}
	case ',':
		l.advanceChar()
		return Token{Type: COMMA, Text: nil, Position: start}
	case ';':
		l.advanceChar()
		return Token{Type: SEMICOLON, Text: nil, Position: start}
	case '-':
		return l.lexMinus(start, hadWhitespace)
	case '+':
		return l.lexPlus(start, hadWhitespace)
	case '*':
		return l.lexMultiply(start, hadWhitespace)
	case '/':
		return l.lexDivide(start, hadWhitespace)
	case '%':
		return l.lexModulo(start, hadWhitespace)
	case '@':
		l.advanceChar()
		return Token{Type: AT, Text: nil, Position: start}
	case '.':
		// Only handle as DOT if not followed by a digit (which would be a decimal number)
		if l.position+1 >= len(l.input) || l.input[l.position+1] >= 128 || !isDigit[l.input[l.position+1]] {
			l.advanceChar()
			return Token{Type: DOT, Text: nil, Position: start}
		}
		// If followed by digit, it will be handled by the decimal number case above
		// This is an edge case that shouldn't happen due to the check above, but we'll return ILLEGAL
		l.advanceChar()
		return Token{Type: ILLEGAL, Text: []byte{'.'}, Position: start}
		// NOTE: '\n' is now handled as whitespace and skipped
		// Meaningful newlines will be implemented when we add statement parsing
	}

	// Unrecognized character - advance and mark as illegal
	l.advanceChar()
	return Token{
		Type:     ILLEGAL,
		Text:     []byte{ch},
		Position: start,
	}
}

// skipWhitespace skips whitespace characters except newlines
// Returns true if any whitespace was skipped
func (l *Lexer) skipWhitespace() bool {
	start := l.position

	// Array jumping: fast scan for non-whitespace
	for l.position < len(l.input) {
		ch := l.input[l.position]
		if ch >= 128 || !isWhitespace[ch] {
			break
		}
		l.position++
	}

	// Update column position based on characters skipped
	l.updateColumnFromWhitespace(start, l.position)

	// Return true if we skipped any characters
	return l.position > start
}

// updateColumnFromWhitespace updates column position after array jumping
func (l *Lexer) updateColumnFromWhitespace(start, end int) {
	for i := start; i < end; i++ {
		ch := l.input[i]
		switch ch {
		case '\n':
			l.line++
			l.column = 1
		case '\t':
			l.column++ // Go standard: column = byte count, tab = 1 byte
		default:
			l.column++
		}
	}
}

// lexIdentifier reads an identifier or keyword starting at current position
func (l *Lexer) lexIdentifier(start Position, hasSpaceBefore bool) Token {
	if l.debugLevel > DebugOff {
		l.recordDebugEvent("enter_lexIdentifier", "reading identifier/keyword")
	}
	startPos := l.position

	// Read all identifier characters
	for l.position < len(l.input) {
		ch := l.input[l.position]
		if ch >= 128 || !isIdentPart[ch] {
			break
		}
		l.advanceChar()
	}

	// Extract the text as byte slice (zero allocation)
	text := l.input[startPos:l.position]
	if l.debugLevel > DebugOff {
		l.recordDebugEvent("found_identifier", string(text))
	}

	// Check if it's a keyword (need string for map lookup)
	tokenType := l.lookupKeyword(string(text))

	return Token{
		Type:           tokenType,
		Text:           text,
		Position:       start,
		HasSpaceBefore: hasSpaceBefore,
	}
}

// lexString reads a string literal starting at current position
func (l *Lexer) lexString(start Position, quote byte, hasSpaceBefore bool) Token {
	startPos := l.position
	l.advanceChar() // Skip opening quote

	// Read until closing quote
	for l.position < len(l.input) {
		ch := l.currentChar()

		// Found closing quote
		if ch == quote {
			l.advanceChar() // Include closing quote
			break
		}

		// Handle escape sequences
		if ch == '\\' && l.position+1 < len(l.input) {
			l.advanceChar() // Skip backslash
			l.advanceChar() // Skip escaped character
			continue
		}

		// For backticks, newlines are allowed
		if quote == '`' && ch == '\n' {
			l.advanceChar()
			continue
		}

		// For double/single quotes, newlines end the string (error case)
		if ch == '\n' && quote != '`' {
			break // Unterminated string
		}

		l.advanceChar()
	}

	// Extract the full string including quotes as byte slice (zero allocation)
	text := l.input[startPos:l.position]

	return Token{
		Type:           STRING,
		Text:           text,
		Position:       start,
		HasSpaceBefore: hasSpaceBefore,
	}
}

// lookupKeyword returns the appropriate token type for keywords, or IDENTIFIER
func (l *Lexer) lookupKeyword(text string) TokenType {
	switch text {
	case "var":
		return VAR
	case "for":
		return FOR
	case "in":
		return IN
	case "if":
		return IF
	case "else":
		return ELSE
	case "when":
		return WHEN
	case "try":
		return TRY
	case "catch":
		return CATCH
	case "finally":
		return FINALLY
	default:
		return IDENTIFIER
	}
}

// currentChar returns the current character being examined (ASCII fast path)
func (l *Lexer) currentChar() byte {
	if l.position >= len(l.input) {
		return 0 // EOF
	}
	return l.input[l.position]
}

// advanceChar moves to the next character, handling Unicode for position tracking only
func (l *Lexer) advanceChar() {
	if l.position >= len(l.input) {
		return
	}

	ch := l.input[l.position]

	// Fast path for ASCII (majority case)
	if ch < 128 {
		switch ch {
		case '\n':
			l.line++
			l.column = 1
		case '\t':
			l.column++ // Go standard: column = byte count, tab = 1 byte
		default:
			l.column++
		}
		l.position++
		return
	}

	// Unicode character - we only need size for position tracking
	// Content goes into tokens as raw bytes
	_, size := utf8.DecodeRune(l.input[l.position:])
	if size <= 0 {
		size = 1 // Invalid UTF-8, treat as single byte
	}

	l.position += size
	l.column++ // Unicode characters count as 1 column for display
}

// lexNumber tokenizes numeric literals (integers, floats, scientific notation)
func (l *Lexer) lexNumber(start Position, hasSpaceBefore bool) Token {
	if l.debugLevel > DebugOff {
		l.recordDebugEvent("enter_lexNumber", "reading numeric literal")
	}
	startPos := l.position

	// Handle both integer and decimal number patterns
	isFloat := false

	// Check if starting with decimal point
	if l.currentChar() == '.' {
		l.advanceChar()
		if !l.readDigits() {
			// No digits after decimal - shouldn't happen given our caller check
			return Token{Type: ILLEGAL, Text: l.input[startPos:l.position], Position: start, HasSpaceBefore: hasSpaceBefore}
		}
		isFloat = true
	} else {
		// Read integer part
		if !l.readDigits() {
			// No digits found - this shouldn't happen given our caller check
			return Token{Type: ILLEGAL, Text: l.input[startPos:l.position], Position: start, HasSpaceBefore: hasSpaceBefore}
		}

		// Check for decimal point
		if l.position < len(l.input) && l.currentChar() == '.' {
			l.advanceChar()
			// Read decimal part (optional - Go allows 5.)
			l.readDigits()
			isFloat = true
		}
	}

	// Check for scientific notation (e/E)
	if l.position < len(l.input) {
		ch := l.currentChar()
		if ch == 'e' || ch == 'E' {
			l.advanceChar() // consume 'e' or 'E'

			// Check for optional sign (+/-)
			if l.position < len(l.input) {
				signChar := l.currentChar()
				if signChar == '+' || signChar == '-' {
					l.advanceChar() // consume sign
				}
			}

			// Read exponent digits (Go allows incomplete exponents like "1e")
			l.readDigits()

			// This is scientific notation
			return Token{
				Type:           SCIENTIFIC,
				Text:           l.input[startPos:l.position],
				Position:       start,
				HasSpaceBefore: hasSpaceBefore,
			}
		}
	}

	// Check for duration units (only for integers, not floats or scientific)
	if !isFloat {
		if l.tryParseDuration(startPos) {
			return Token{
				Type:           DURATION,
				Text:           l.input[startPos:l.position],
				Position:       start,
				HasSpaceBefore: hasSpaceBefore,
			}
		}
	}

	// Return appropriate type based on whether we found a decimal point
	if isFloat {
		return Token{
			Type:           FLOAT,
			Text:           l.input[startPos:l.position],
			Position:       start,
			HasSpaceBefore: hasSpaceBefore,
		}
	}

	// Just an integer
	return Token{
		Type:           INTEGER,
		Text:           l.input[startPos:l.position],
		Position:       start,
		HasSpaceBefore: hasSpaceBefore,
	}
}

// readDigits reads a sequence of digits and returns true if at least one was found
func (l *Lexer) readDigits() bool {
	startPos := l.position

	for l.position < len(l.input) {
		ch := l.currentChar()
		if ch >= 128 || !isDigit[ch] {
			break
		}
		l.advanceChar()
	}

	return l.position > startPos
}

// tryParseDuration attempts to parse duration units after a number
// Returns true if duration units were found and consumed
func (l *Lexer) tryParseDuration(startPos int) bool {
	savedPosition := l.position

	// Try to read compound duration units
	hasUnits := false
	for l.readDurationUnit() {
		hasUnits = true

		// After reading a unit, check if there are more digits for compound duration
		if l.position >= len(l.input) {
			break // End of input
		}

		// Check if next character could start another unit (digit)
		ch := l.currentChar()
		if ch >= 128 || !isDigit[ch] {
			break // No more units - this is normal for simple durations
		}

		// Read digits for next unit
		if !l.readDigits() {
			break // No digits found, we're done
		}
	}

	// If we found at least one unit, this is a duration
	if hasUnits {
		return true
	}

	// No units found, restore position
	l.position = savedPosition
	return false
}

// readDurationUnit reads a single duration unit (s, m, h, d, w, y, ms, us, ns)
// Returns true if a valid unit was consumed
func (l *Lexer) readDurationUnit() bool {
	if l.position >= len(l.input) {
		return false
	}

	// Check for two-character units first (ms, us, ns)
	if l.position+1 < len(l.input) {
		twoChar := string(l.input[l.position : l.position+2])
		switch twoChar {
		case "ms", "us", "ns":
			l.advanceChar() // Advance first character
			l.advanceChar() // Advance second character
			return true
		}
	}

	// Check for single-character units
	ch := l.currentChar()
	switch ch {
	case 's', 'm', 'h', 'd', 'w', 'y':
		l.advanceChar() // Properly advance with line/column tracking
		return true
	}

	return false
}

// lexMinus handles '-', '--', and '-=' operators
func (l *Lexer) lexMinus(start Position, hasSpaceBefore bool) Token {
	l.advanceChar() // consume '-'

	// Check for '--' (decrement)
	if l.position < len(l.input) && l.currentChar() == '-' {
		l.advanceChar() // consume second '-'
		return Token{Type: DECREMENT, Text: nil, Position: start, HasSpaceBefore: hasSpaceBefore}
	}

	// Check for '-=' (minus assign)
	if l.position < len(l.input) && l.currentChar() == '=' {
		l.advanceChar() // consume '='
		return Token{Type: MINUS_ASSIGN, Text: nil, Position: start, HasSpaceBefore: hasSpaceBefore}
	}

	// Just '-' (minus)
	return Token{Type: MINUS, Text: nil, Position: start, HasSpaceBefore: hasSpaceBefore}
}

// lexPlus handles '+', '++', and '+=' operators
func (l *Lexer) lexPlus(start Position, hasSpaceBefore bool) Token {
	l.advanceChar() // consume '+'

	// Check for '++' (increment)
	if l.position < len(l.input) && l.currentChar() == '+' {
		l.advanceChar() // consume second '+'
		return Token{Type: INCREMENT, Text: nil, Position: start, HasSpaceBefore: hasSpaceBefore}
	}

	// Check for '+=' (plus assign)
	if l.position < len(l.input) && l.currentChar() == '=' {
		l.advanceChar() // consume '='
		return Token{Type: PLUS_ASSIGN, Text: nil, Position: start, HasSpaceBefore: hasSpaceBefore}
	}

	// Just '+' (plus)
	return Token{Type: PLUS, Text: nil, Position: start, HasSpaceBefore: hasSpaceBefore}
}

// lexMultiply handles '*' and '*=' operators
func (l *Lexer) lexMultiply(start Position, hasSpaceBefore bool) Token {
	l.advanceChar() // consume '*'

	// Check for '*=' (multiply assign)
	if l.position < len(l.input) && l.currentChar() == '=' {
		l.advanceChar() // consume '='
		return Token{Type: MULTIPLY_ASSIGN, Text: nil, Position: start, HasSpaceBefore: hasSpaceBefore}
	}

	// Just '*' (multiply)
	return Token{Type: MULTIPLY, Text: nil, Position: start, HasSpaceBefore: hasSpaceBefore}
}

// lexDivide handles '/', '/=', '//', and '/*' operators and comments
func (l *Lexer) lexDivide(start Position, hasSpaceBefore bool) Token {
	l.advanceChar() // consume '/'

	if l.position < len(l.input) {
		nextChar := l.currentChar()

		// Check for '/*' (block comment start)
		if nextChar == '*' {
			return l.lexBlockComment(start, hasSpaceBefore)
		}

		// Check for '//' (line comment start)
		if nextChar == '/' {
			return l.lexLineComment(start, hasSpaceBefore)
		}

		// Check for '/=' (divide assign)
		if nextChar == '=' {
			l.advanceChar() // consume '='
			return Token{Type: DIVIDE_ASSIGN, Text: nil, Position: start, HasSpaceBefore: hasSpaceBefore}
		}
	}

	// Just '/' (divide)
	return Token{Type: DIVIDE, Text: nil, Position: start, HasSpaceBefore: hasSpaceBefore}
}

// lexModulo handles '%' and '%=' operators
func (l *Lexer) lexModulo(start Position, hasSpaceBefore bool) Token {
	l.advanceChar() // consume '%'

	// Check for '%=' (modulo assign)
	if l.position < len(l.input) && l.currentChar() == '=' {
		l.advanceChar() // consume '='
		return Token{Type: MODULO_ASSIGN, Text: nil, Position: start, HasSpaceBefore: hasSpaceBefore}
	}

	// Just '%' (modulo)
	return Token{Type: MODULO, Text: nil, Position: start, HasSpaceBefore: hasSpaceBefore}
}

// lexEquals handles '=' and '==' operators
func (l *Lexer) lexEquals(start Position, hasSpaceBefore bool) Token {
	l.advanceChar() // consume '='

	// Check for '==' (equality)
	if l.position < len(l.input) && l.currentChar() == '=' {
		l.advanceChar() // consume second '='
		return Token{Type: EQ_EQ, Text: nil, Position: start, HasSpaceBefore: hasSpaceBefore}
	}

	// Just '=' (assignment)
	return Token{Type: EQUALS, Text: nil, Position: start, HasSpaceBefore: hasSpaceBefore}
}

// lexLessThan handles '<' and '<=' operators
func (l *Lexer) lexLessThan(start Position, hasSpaceBefore bool) Token {
	l.advanceChar() // consume '<'

	// Check for '<=' (less than or equal)
	if l.position < len(l.input) && l.currentChar() == '=' {
		l.advanceChar() // consume '='
		return Token{Type: LT_EQ, Text: nil, Position: start, HasSpaceBefore: hasSpaceBefore}
	}

	// Just '<' (less than)
	return Token{Type: LT, Text: nil, Position: start, HasSpaceBefore: hasSpaceBefore}
}

// lexGreaterThan handles '>' and '>=' operators
func (l *Lexer) lexGreaterThan(start Position, hasSpaceBefore bool) Token {
	l.advanceChar() // consume '>'

	// Check for '>=' (greater than or equal)
	if l.position < len(l.input) && l.currentChar() == '=' {
		l.advanceChar() // consume '='
		return Token{Type: GT_EQ, Text: nil, Position: start, HasSpaceBefore: hasSpaceBefore}
	}

	// Just '>' (greater than)
	return Token{Type: GT, Text: nil, Position: start, HasSpaceBefore: hasSpaceBefore}
}

// lexExclamation handles '!' and '!=' operators
func (l *Lexer) lexExclamation(start Position, hasSpaceBefore bool) Token {
	l.advanceChar() // consume '!'

	// Check for '!=' (not equal)
	if l.position < len(l.input) && l.currentChar() == '=' {
		l.advanceChar() // consume '='
		return Token{Type: NOT_EQ, Text: nil, Position: start, HasSpaceBefore: hasSpaceBefore}
	}

	// Just '!' (logical not)
	return Token{Type: NOT, Text: nil, Position: start, HasSpaceBefore: hasSpaceBefore}
}

// lexAmpersand handles '&' and '&&' operators
func (l *Lexer) lexAmpersand(start Position, hasSpaceBefore bool) Token {
	l.advanceChar() // consume '&'

	// Check for '&&' (logical and)
	if l.position < len(l.input) && l.currentChar() == '&' {
		l.advanceChar() // consume second '&'
		return Token{Type: AND_AND, Text: nil, Position: start, HasSpaceBefore: hasSpaceBefore}
	}

	// Single '&' is illegal for now (future: bitwise and)
	return Token{Type: ILLEGAL, Text: nil, Position: start, HasSpaceBefore: hasSpaceBefore}
}

// lexPipe handles '|' and '||' operators
func (l *Lexer) lexPipe(start Position, hasSpaceBefore bool) Token {
	l.advanceChar() // consume '|'

	// Check for '||' (logical or)
	if l.position < len(l.input) && l.currentChar() == '|' {
		l.advanceChar() // consume second '|'
		return Token{Type: OR_OR, Text: nil, Position: start, HasSpaceBefore: hasSpaceBefore}
	}

	// Single '|' is illegal for now (future: bitwise or)
	return Token{Type: ILLEGAL, Text: nil, Position: start, HasSpaceBefore: hasSpaceBefore}
}

// lexLineComment handles // style comments, excluding the // prefix
func (l *Lexer) lexLineComment(start Position, hasSpaceBefore bool) Token {
	l.advanceChar() // consume second '/'

	startContentPos := l.position

	// Read until end of line or EOF
	for l.position < len(l.input) {
		ch := l.currentChar()
		if ch == '\n' {
			break // Don't consume the newline - it's meaningful whitespace
		}
		l.advanceChar()
	}

	// Extract comment content (without the // prefix)
	content := l.input[startContentPos:l.position]

	return Token{
		Type:     COMMENT,
		Text:     content,
		Position: start,
	}
}

// lexBlockComment handles /* */ style comments, excluding the delimiters
func (l *Lexer) lexBlockComment(start Position, hasSpaceBefore bool) Token {
	l.advanceChar() // consume '*'

	startContentPos := l.position

	// Read until */ or EOF
	for l.position < len(l.input) {
		ch := l.currentChar()
		if ch == '*' && l.position+1 < len(l.input) && l.input[l.position+1] == '/' {
			// Found closing */
			endContentPos := l.position
			l.advanceChar() // consume '*'
			l.advanceChar() // consume '/'

			// Extract comment content (without the /* */ delimiters)
			content := l.input[startContentPos:endContentPos]

			return Token{
				Type:           COMMENT,
				Text:           content,
				Position:       start,
				HasSpaceBefore: hasSpaceBefore,
			}
		}
		l.advanceChar()
	}

	// Unterminated block comment - return content up to EOF
	content := l.input[startContentPos:l.position]

	return Token{
		Type:     COMMENT,
		Text:     content,
		Position: start,
	}
}
