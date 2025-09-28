package lexer

import (
	"io"
	"log/slog"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/aledsdavies/opal/core/decorators"
	"github.com/aledsdavies/opal/core/types"
)

// ASCII character lookup tables for fast classification
var (
	isWhitespace      [128]bool // Only ASCII range
	isLetter          [128]bool
	isDigit           [128]bool
	isIdentStart      [128]bool
	isIdentPart       [128]bool
	singleCharTokens  [128]types.TokenType // Fast lookup for single-char tokens
	singleCharStrings [128]string          // Pre-allocated single-char strings
)

func init() {
	for i := 0; i < 128; i++ {
		ch := byte(i)
		isWhitespace[i] = ch == ' ' || ch == '\t' || ch == '\r' || ch == '\f'
		isLetter[i] = ('a' <= ch && ch <= 'z') || ('A' <= ch && ch <= 'Z') || ch == '_'
		isDigit[i] = '0' <= ch && ch <= '9'
		isIdentStart[i] = isLetter[i] || ch == '_'
		isIdentPart[i] = isIdentStart[i] || isDigit[i] || ch == '-'
		singleCharTokens[i] = types.ILLEGAL // Default to ILLEGAL for non-single-char tokens
		singleCharStrings[i] = string(ch)   // Pre-allocate single char strings
	}

	// Initialize single character token mappings
	singleCharTokens['@'] = types.AT
	singleCharTokens[':'] = types.COLON
	singleCharTokens['='] = types.EQUALS
	singleCharTokens[','] = types.COMMA
	singleCharTokens['('] = types.LPAREN
	singleCharTokens[')'] = types.RPAREN
	singleCharTokens['{'] = types.LBRACE
	singleCharTokens['}'] = types.RBRACE
	singleCharTokens['*'] = types.ASTERISK
}

// LexerMode represents the lexer's parsing modes
type LexerMode int

const (
	LanguageMode LexerMode = iota // Top-level parsing and decorator parsing
	CommandMode                   // Simple shell content parsing inside command bodies
	ShellMode                     // Complex shell content with decorator expansion and maintained shell context
	PatternMode                   // Pattern decorator parsing (@when, @try blocks)
)

// Lexer implements the three-mode system with simple context-free transitions
type Lexer struct {
	input    string // Complete input (read once from Reader)
	position int    // Current position in input (byte offset)
	readPos  int    // Current reading position in input (byte offset)
	ch       rune   // Current rune under examination
	line     int    // Current line number
	column   int    // Current column number

	// Simple three-mode system
	mode LexerMode

	// Debug logger
	logger *slog.Logger

	// Minimal context tracking
	braceLevel        int // Track brace nesting for mode transitions
	patternBraceLevel int // Track the brace level where we entered pattern decorator
	commandBlockLevel int // Track the brace level where we entered a command block

	// Function decorator state
	inFunctionDecorator bool // True when we're inside a function decorator sequence

	// Shell context tracking (maintained across decorator breaks in ShellMode)
	shellBraceLevel    int  // Track ${...} parameter expansion braces globally
	shellParenLevel    int  // Track $(...) command substitution globally
	shellAnyBraceLevel int  // Track any {...} constructs in shell context globally
	needsShellEnd      bool // True when we need to emit SHELL_END before next non-shell token

	// Quote state tracking (maintained across decorator breaks in ShellMode)
	shellInSingleQuote bool
	shellInDoubleQuote bool
	shellInBacktick    bool

	// Interpolated string state tracking
	inInterpolatedString bool      // True when we're inside an interpolated string (between STRING_START and STRING_END)
	interpolatedQuote    rune      // The quote character of the current interpolated string (" or `)
	preInterpolatedMode  LexerMode // The mode we were in before entering interpolated string
	inStringDecorator    bool      // True when we're parsing a decorator inside a string

	// Literal string state (for simple quoted strings in decorator parameters)
	inLiteralString bool      // True when we're inside a literal string (no interpolation)
	literalQuote    rune      // The quote character of the current literal string
	originalMode    LexerMode // The original mode before any LanguageMode transitions (for proper restoration)

	// Token queue for complex tokenization scenarios
	tokenQueue []types.Token // Queue of tokens to return before continuing normal parsing

	// SHELL_END position tracking - the end position of the last shell content
	shellEndPosition int
	shellEndLine     int
	shellEndColumn   int

	// Position tracking for error reporting
	lastPosition int
	lastLine     int
	lastColumn   int
}

// getShellEndPosition returns a safe position for SHELL_END tokens
func (l *Lexer) getShellEndPosition(fallbackPos, fallbackLine, fallbackColumn int) (int, int, int) {
	// If shell end position is set, use it
	if l.shellEndPosition > 0 {
		return l.shellEndPosition, l.shellEndLine, l.shellEndColumn
	}
	// Otherwise use fallback position but ensure column > 0
	return fallbackPos, fallbackLine, max(1, fallbackColumn)
}

// New creates a new Lexer from an io.Reader
func New(reader io.Reader) *Lexer {
	// Read entire input into string (simpler approach for now)
	data, err := io.ReadAll(reader)
	if err != nil {
		// Handle error by creating empty lexer
		data = []byte{}
	}

	// Create debug logger - check if DEVCMD_DEBUG_LEXER environment variable is set
	logLevel := slog.LevelInfo
	if os.Getenv("DEVCMD_DEBUG_LEXER") != "" {
		logLevel = slog.LevelDebug
	}

	// Custom lexer-friendly handler
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Remove timestamp for cleaner output
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			// Simplify level display
			if a.Key == slog.LevelKey {
				return slog.Attr{}
			}
			return a
		},
	}))

	l := &Lexer{
		input:  string(data),
		line:   1,
		column: 0,            // Will be incremented to 1 by initial readChar()
		mode:   LanguageMode, // Start in LanguageMode
		logger: logger,
	}
	l.readChar()
	return l
}

// isAfterPatternDecorator checks if we just parsed a pattern decorator by looking back
func (l *Lexer) isAfterPatternDecorator() bool {
	// Look back through recent input to find any pattern decorator using the registry
	pos := l.position - 1

	// Skip backwards through whitespace and closing paren to find the decorator
	for pos >= 0 && (l.input[pos] == ' ' || l.input[pos] == '\t' || l.input[pos] == ')') {
		pos--
	}

	// Look back to find @ symbol and extract decorator name
	if pos >= 4 {
		// Find the @ symbol by scanning backwards
		atPos := -1
		for i := pos; i >= max(0, pos-20); i-- {
			if l.input[i] == '@' {
				atPos = i
				break
			}
		}

		if atPos >= 0 {
			// Extract potential decorator name after @
			nameStart := atPos + 1
			nameEnd := nameStart

			// Find end of identifier (decorator name)
			for nameEnd < len(l.input) && nameEnd <= pos+1 {
				ch := l.input[nameEnd]
				if ch >= 128 || (!isLetter[ch] && !isDigit[ch]) {
					break
				}
				nameEnd++
			}

			if nameEnd > nameStart {
				decoratorName := l.input[nameStart:nameEnd]
				// Use decorator registry to check if this is a pattern decorator
				return decorators.IsPatternDecorator(decoratorName)
			}
		}
	}
	return false
}

// isInPatternContext determines if we're currently inside a pattern decorator context
func (l *Lexer) isInPatternContext() bool {
	// Simple check: are we at or below the brace level where we entered pattern mode?
	return l.patternBraceLevel > 0 && l.braceLevel >= l.patternBraceLevel
}

// max helper function
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// readChar reads the next character and advances position
func (l *Lexer) readChar() {
	l.position = l.readPos

	if l.readPos >= len(l.input) {
		l.ch = 0 // EOF
	} else {
		var size int
		l.ch, size = utf8.DecodeRuneInString(l.input[l.readPos:])
		if l.ch == utf8.RuneError {
			l.ch = rune(l.input[l.readPos])
			size = 1
		}
		l.readPos += size
	}

	// Track line/column for current character
	if l.ch == '\n' {
		l.line++
		l.column = 0 // Will be incremented to 1 for next character
	} else {
		l.column++
	}
}

// peekChar returns the next character without advancing position
func (l *Lexer) peekChar() rune {
	if l.readPos >= len(l.input) {
		return 0
	}
	ch, _ := utf8.DecodeRuneInString(l.input[l.readPos:])
	return ch
}

// skipWhitespace skips whitespace characters except newlines (using fast ASCII lookups)
func (l *Lexer) skipWhitespace() {
	for l.ch != '\n' && l.ch != 0 {
		// Fast path for ASCII
		if l.ch < 128 && isWhitespace[l.ch] {
			l.readChar()
		} else if l.ch >= 128 && unicode.IsSpace(l.ch) {
			// Fallback for non-ASCII
			l.readChar()
		} else {
			break
		}
	}
}

// TokenizeToSlice tokenizes the entire input and returns a slice of tokens
func (l *Lexer) TokenizeToSlice() []types.Token {
	var tokens []types.Token
	for {
		token := l.NextToken()
		tokens = append(tokens, token)
		if token.Type == types.EOF {
			break
		}
	}
	return tokens
}

// NextToken returns the next token from the input
func (l *Lexer) NextToken() types.Token {
	// Check token queue first
	if len(l.tokenQueue) > 0 {
		token := l.tokenQueue[0]
		l.tokenQueue = l.tokenQueue[1:]
		return token
	}

	// Prevent infinite loops, but allow SHELL_END transitions
	if l.position == l.lastPosition && l.line == l.lastLine && l.column == l.lastColumn && !l.needsShellEnd {
		// We haven't advanced - force EOF to prevent infinite loop
		return l.createToken(types.EOF, "", l.position, l.line, l.column)
	}
	l.lastPosition = l.position
	l.lastLine = l.line
	l.lastColumn = l.column

	// Check if we're inside a literal string (decorator parameters)
	if l.inLiteralString {
		return l.lexLiteralStringContent()
	}

	// Check if we're inside an interpolated string
	// BUT allow LanguageMode to take precedence when parsing decorators within strings
	if l.inInterpolatedString && !l.inStringDecorator {
		return l.lexInterpolatedStringContent()
	}

	// Dispatch based on current mode
	// Debug logging for mode transitions and current state
	if l.position < len(l.input) {
		nextChars := ""
		end := l.position + 10
		if end > len(l.input) {
			end = len(l.input)
		}
		nextChars = strings.ReplaceAll(l.input[l.position:end], "\n", "\\n")
		l.logger.Debug("[LEXER] Token dispatch",
			"mode", l.mode,
			"ch", string(l.ch),
			"next", nextChars,
			"braceLevel", l.braceLevel,
			"inInterpolatedString", l.inInterpolatedString,
			"inStringDecorator", l.inStringDecorator,
			"inFunctionDecorator", l.inFunctionDecorator)
	}

	switch l.mode {
	case LanguageMode:
		return l.lexLanguageMode()
	case CommandMode:
		return l.lexCommandMode()
	case ShellMode:
		return l.lexShellMode()
	case PatternMode:
		return l.lexPatternMode()
	default:
		return l.createToken(types.EOF, "", l.position, l.line, l.column)
	}
}

// createToken creates a token with position information
func (l *Lexer) createToken(tokenType types.TokenType, value string, start, line, column int) types.Token {
	return types.Token{
		Type:   tokenType,
		Value:  value,
		Line:   line,
		Column: column,
		Span: types.SourceSpan{
			Start: types.SourcePosition{Line: line, Column: column, Offset: start},
			End:   types.SourcePosition{Line: l.line, Column: l.column, Offset: l.position},
		},
	}
}

// lexLanguageMode handles top-level parsing and decorator parsing
func (l *Lexer) lexLanguageMode() types.Token {
	l.logger.Debug("[LEXER] Entering LanguageMode",
		"position", l.position,
		"ch", string(l.ch),
		"braceLevel", l.braceLevel,
		"inFunctionDecorator", l.inFunctionDecorator,
		"inStringDecorator", l.inStringDecorator)

	l.skipWhitespace()

	start := l.position
	startLine, startColumn := l.line, l.column

	switch l.ch {
	case 0:
		// Check if we're ending a function decorator in shell content
		if l.inFunctionDecorator {
			// Transition back to ShellMode to emit SHELL_END before EOF
			l.inFunctionDecorator = false
			l.mode = ShellMode
			l.needsShellEnd = true
			// Reset position tracking to allow SHELL_END processing
			l.lastPosition = -1
			pos, line, col := l.getShellEndPosition(start, startLine, startColumn)
			return l.createToken(types.SHELL_END, "", pos, line, col)
		}
		return l.createToken(types.EOF, "", start, startLine, startColumn)

	case '\n':
		// Skip newlines in language mode
		l.readChar()
		return l.NextToken()

	case ':':
		l.readChar()
		// Transition to CommandMode after colon (CommandMode will decide between decorators and shell content)
		l.mode = CommandMode
		return l.createToken(types.COLON, ":", start, startLine, startColumn)

	case '=':
		l.readChar()
		return l.createToken(types.EQUALS, "=", start, startLine, startColumn)

	case ',':
		l.readChar()
		return l.createToken(types.COMMA, ",", start, startLine, startColumn)

	case '(':
		l.readChar()
		return l.createToken(types.LPAREN, "(", start, startLine, startColumn)

	case ')':
		l.readChar()
		// Check if we're ending a function decorator sequence
		if l.inFunctionDecorator {
			l.logger.Debug("[LEXER] Ending function decorator",
				"inStringDecorator", l.inStringDecorator,
				"inInterpolatedString", l.inInterpolatedString,
				"currentMode", l.mode)

			l.inFunctionDecorator = false
			l.originalMode = 0 // Reset originalMode when function decorator ends

			// Check if we're in a decorator within a string - return to string parsing
			if l.inStringDecorator {
				l.inStringDecorator = false
				l.logger.Debug("[LEXER] Ended decorator in string - staying in LanguageMode for string continuation")
				// Stay in LanguageMode but next call to NextToken will continue string parsing
				// since inInterpolatedString is still true
			} else {
				// Return to the original mode we were in before the function decorator
				if l.originalMode != 0 {
					l.mode = l.originalMode
				} else {
					l.mode = ShellMode // Fallback if originalMode wasn't set
				}
				l.logger.Debug("[LEXER] Ended decorator - transitioning to original mode",
					"newMode", l.mode, "originalMode", l.originalMode)
				// Check if this decorator is the last shell token before newline (command end)
				// Skip whitespace to see what comes next
				pos := l.position
				ch := l.ch
				for ch == ' ' || ch == '\t' {
					pos++
					if pos >= len(l.input) {
						ch = 0
						break
					}
					ch = rune(l.input[pos])
				}
				// If next non-whitespace is newline, EOF, or closing brace (but not inside shell parameter expansion), this decorator ends the command
				if ch == '\n' || ch == 0 || (ch == '}' && l.shellBraceLevel == 0) {
					l.needsShellEnd = true
				}
			}
		}
		return l.createToken(types.RPAREN, ")", start, startLine, startColumn)

	case '{':
		l.readChar()
		l.braceLevel++
		// Simple rule: { after pattern decorator → PatternMode, otherwise → CommandMode
		if l.isAfterPatternDecorator() {
			l.mode = PatternMode
			l.patternBraceLevel = l.braceLevel // Remember where we entered pattern mode
		} else {
			l.mode = CommandMode
		}
		return l.createToken(types.LBRACE, "{", start, startLine, startColumn)

	case '}':
		l.readChar()
		l.braceLevel--
		// Simple rule: completely exited all braces → LanguageMode
		if l.braceLevel <= 0 {
			l.mode = LanguageMode
			l.patternBraceLevel = 0 // Clear pattern context
			l.commandBlockLevel = 0 // Clear command block context
		} else if l.commandBlockLevel > 0 && l.braceLevel >= l.commandBlockLevel {
			// We're still inside a command block, return to CommandMode
			l.mode = CommandMode
		}
		// Otherwise stay in current mode - parent context will handle mode transitions
		return l.createToken(types.RBRACE, "}", start, startLine, startColumn)

	case '*':
		l.readChar()
		return l.createToken(types.ASTERISK, "*", start, startLine, startColumn)

	case '@':
		return l.lexDecorator(start, startLine, startColumn)

	case '"', '\'', '`':
		// When inside a string decorator, use simple literal string parsing
		// to avoid nested interpolated string contexts
		if l.inStringDecorator {
			return l.lexLiteralStringInDecorator(l.ch, start, startLine, startColumn)
		}
		return l.lexString(l.ch, start, startLine, startColumn)

	case '#':
		return l.lexComment(start, startLine, startColumn)

	case '/':
		// Check for multi-line comment /* */
		if l.peekChar() == '*' {
			return l.lexMultilineComment(start, startLine, startColumn)
		}
		// Not a comment - treat as unknown character
		char := string(l.ch)
		l.readChar()
		return l.createToken(types.ILLEGAL, char, start, startLine, startColumn)

	case '-':
		// Check if this is a negative number
		if l.readPos < len(l.input) {
			nextCh, _ := utf8.DecodeRuneInString(l.input[l.readPos:])
			if (nextCh < 128 && isDigit[nextCh]) || (nextCh >= 128 && unicode.IsDigit(nextCh)) {
				return l.lexNumber(start, startLine, startColumn)
			}
		}
		// Not a negative number - treat as unknown character
		char := string(l.ch)
		l.readChar()
		return l.createToken(types.ILLEGAL, char, start, startLine, startColumn)

	default:
		// Fast path for ASCII identifier start
		if (l.ch < 128 && isIdentStart[l.ch]) || (l.ch >= 128 && (unicode.IsLetter(l.ch) || l.ch == '_')) {
			return l.lexIdentifierOrKeyword(start, startLine, startColumn)
		}
		// Fast path for ASCII digits
		if (l.ch < 128 && isDigit[l.ch]) || (l.ch >= 128 && unicode.IsDigit(l.ch)) {
			return l.lexNumber(start, startLine, startColumn)
		}

		// Unknown character
		char := string(l.ch)
		l.readChar()
		return l.createToken(types.ILLEGAL, char, start, startLine, startColumn)
	}
}

// lexCommandMode handles shell content parsing inside command bodies
// Recognizes: Shell text, Line continuations, Decorators, Block boundaries
func (l *Lexer) lexCommandMode() types.Token {
	l.logger.Debug("[LEXER] Entering CommandMode",
		"position", l.position,
		"ch", string(l.ch),
		"braceLevel", l.braceLevel,
		"commandBlockLevel", l.commandBlockLevel,
		"patternBraceLevel", l.patternBraceLevel)

	l.skipWhitespace()

	start := l.position
	startLine, startColumn := l.line, l.column

	switch l.ch {
	case 0:
		return l.createToken(types.EOF, "", start, startLine, startColumn)

	case '\n':
		// Newlines end shell content in command mode (unless line continuation)
		l.readChar()
		// Simple rule: determine next mode based on context
		if l.braceLevel == 0 {
			l.mode = LanguageMode
		} else if l.isInPatternContext() && l.braceLevel == l.patternBraceLevel {
			// Only return to PatternMode if we're at the exact pattern brace level
			// (not inside nested blocks within the pattern)
			l.mode = PatternMode
		}
		// Otherwise stay in CommandMode for regular braced blocks or nested blocks within patterns
		return l.NextToken()

	case '}':
		// Closing brace - exit command mode
		l.readChar()
		l.braceLevel--
		// Simple rule: determine next mode based on context
		if l.braceLevel <= 0 {
			l.mode = LanguageMode
			l.patternBraceLevel = 0 // Clear pattern context
			l.commandBlockLevel = 0 // Clear command block context
		} else if l.commandBlockLevel > 0 && l.braceLevel >= l.commandBlockLevel {
			// We're still inside a command block, stay in CommandMode
			l.mode = CommandMode
		} else if l.isInPatternContext() && l.braceLevel == l.patternBraceLevel {
			// Only return to PatternMode if we're back to the exact pattern brace level
			// (exiting a pattern branch block, not a nested block within the pattern)
			l.mode = PatternMode
		}
		// Otherwise stay in CommandMode for nested command blocks
		return l.createToken(types.RBRACE, "}", start, startLine, startColumn)

	case '{':
		// Opening brace in command mode - start new block
		l.readChar()
		l.braceLevel++
		// Track that we're entering a command block
		if l.commandBlockLevel == 0 {
			l.commandBlockLevel = l.braceLevel
		}
		return l.createToken(types.LBRACE, "{", start, startLine, startColumn)

	case '@':
		// Handle Decorator path: check if Block or Pattern decorator
		return l.lexDecoratorInCommand(start, startLine, startColumn)

	case ')':
		// Closing parenthesis in command mode (e.g., end of function decorator)
		l.readChar()
		return l.createToken(types.RPAREN, ")", start, startLine, startColumn)

	default:
		// Handle Shell path: all other content as shell text
		l.mode = ShellMode // Switch to ShellMode for proper SHELL_END handling
		return l.lexShellTextWithContext(start, startLine, startColumn)
	}
}

// lexDecorator handles decorator parsing in LanguageMode
func (l *Lexer) lexDecorator(start, startLine, startColumn int) types.Token {
	// Skip @ character
	l.readChar()

	// Skip whitespace after @
	l.skipWhitespace()

	// Read decorator identifier using fast ASCII lookups
	if (l.ch >= 128 || !isIdentStart[l.ch]) && (l.ch < 128 || (!unicode.IsLetter(l.ch) && l.ch != '_')) {
		return l.createToken(types.ILLEGAL, "@", start, startLine, startColumn)
	}

	// Return AT token, let next token be the identifier
	return l.createToken(types.AT, "@", start, startLine, startColumn)
}

// lexDecoratorInCommand checks if @identifier is a decorator in CommandMode
func (l *Lexer) lexDecoratorInCommand(start, startLine, startColumn int) types.Token {
	// Look ahead to check if this is @identifier pattern
	savedPos := l.position
	savedReadPos := l.readPos
	savedCh := l.ch
	savedLine := l.line
	savedColumn := l.column

	// Skip @
	l.readChar()
	l.skipWhitespace()

	// Check if followed by identifier using fast ASCII lookups
	if (l.ch < 128 && isIdentStart[l.ch]) || (l.ch >= 128 && (unicode.IsLetter(l.ch) || l.ch == '_')) {
		// Read the identifier to check if it's a decorator
		identStart := l.position
		for {
			if l.ch < 128 && isIdentPart[l.ch] {
				l.readChar()
			} else if l.ch >= 128 && (unicode.IsLetter(l.ch) || unicode.IsDigit(l.ch)) {
				l.readChar()
			} else {
				break
			}
		}
		identifier := l.input[identStart:l.position]

		// Check if it's a registered decorator
		if decorators.IsDecorator(identifier) {
			// Handle different decorator types appropriately
			if decorators.IsBlockDecorator(identifier) || decorators.IsPatternDecorator(identifier) {
				// Block and pattern decorators need LanguageMode for proper IDENTIFIER parsing
				l.mode = LanguageMode

				// Advance past @ character (don't restore position)
				l.position = savedPos
				l.readPos = savedReadPos
				l.ch = savedCh
				l.line = savedLine
				l.column = savedColumn
				l.readChar() // Skip the @ character

				return l.createToken(types.AT, "@", start, startLine, startColumn)
			} else if decorators.IsDecorator(identifier) {
				// Check if followed by parentheses for function decorators
				if l.ch == '(' {
					// This is a function decorator - switch to LanguageMode for the decorator sequence
					l.originalMode = l.mode // Save the original mode before switching to LanguageMode
					l.mode = LanguageMode
					l.inFunctionDecorator = true

					// Reset position to @ and advance past it
					l.position = savedPos
					l.readPos = savedReadPos
					l.ch = savedCh
					l.line = savedLine
					l.column = savedColumn
					l.readChar() // Skip the @ character

					return l.createToken(types.AT, "@", start, startLine, startColumn)
				}
			}
		}
	}

	// Restore position - this is shell text starting with @
	l.position = savedPos
	l.readPos = savedReadPos
	l.ch = savedCh
	l.line = savedLine
	l.column = savedColumn

	return l.lexShellText(start, startLine, startColumn)
}

// lexPatternMode handles pattern decorator content (@when, @try blocks)
func (l *Lexer) lexPatternMode() types.Token {
	l.skipWhitespace()

	start := l.position
	startLine, startColumn := l.line, l.column

	switch l.ch {
	case 0:
		return l.createToken(types.EOF, "", start, startLine, startColumn)

	case '\n':
		// Skip newlines in pattern mode
		l.readChar()
		return l.NextToken()

	case '}':
		// Closing brace - exit pattern mode
		l.readChar()
		l.braceLevel--
		// Simple rule: completely exited → LanguageMode, otherwise determine by context
		if l.braceLevel <= 0 {
			l.mode = LanguageMode
			l.patternBraceLevel = 0 // Clear pattern context
		} else if l.isInPatternContext() {
			// Still inside a pattern decorator, return to PatternMode for more pattern branches
			l.mode = PatternMode
		} else {
			// Regular block context, return to CommandMode
			l.mode = CommandMode
		}
		return l.createToken(types.RBRACE, "}", start, startLine, startColumn)

	case ':':
		l.readChar()
		// After colon in pattern mode, switch to CommandMode for shell content
		l.mode = CommandMode
		return l.createToken(types.COLON, ":", start, startLine, startColumn)

	case '{':
		l.readChar()
		l.braceLevel++
		// Transition to CommandMode for block content inside patterns
		l.mode = CommandMode
		return l.createToken(types.LBRACE, "{", start, startLine, startColumn)

	default:
		// Pattern identifiers (prod, dev, main, error, finally, default)
		if (l.ch < 128 && isIdentStart[l.ch]) || (l.ch >= 128 && (unicode.IsLetter(l.ch) || l.ch == '_')) {
			return l.lexIdentifierOrKeyword(start, startLine, startColumn)
		}

		// Unknown character
		char := string(l.ch)
		l.readChar()
		return l.createToken(types.ILLEGAL, char, start, startLine, startColumn)
	}
}

// lexShellMode handles complex shell content with decorator expansion and maintained shell context
func (l *Lexer) lexShellMode() types.Token {
	l.logger.Debug("[LEXER] Entering ShellMode",
		"position", l.position,
		"ch", string(l.ch),
		"braceLevel", l.braceLevel,
		"needsShellEnd", l.needsShellEnd,
		"shellBraceLevel", l.shellBraceLevel,
		"shellInSingleQuote", l.shellInSingleQuote,
		"shellInDoubleQuote", l.shellInDoubleQuote)

	// Check if we need to emit SHELL_END first
	if l.needsShellEnd {
		l.needsShellEnd = false
		// After emitting SHELL_END, transition to appropriate mode based on context
		if l.braceLevel > 0 {
			// Still inside braces - return to CommandMode for more shell content
			l.mode = CommandMode
		} else {
			// Exited all braces - return to LanguageMode
			l.mode = LanguageMode
		}
		// Create SHELL_END token using stored shell end position
		pos, line, col := l.getShellEndPosition(l.position, l.line, l.column)
		token := l.createToken(types.SHELL_END, "", pos, line, col)
		// Reset position tracking to allow the next token to be processed
		l.lastPosition = -1 // Force position check to pass
		return token
	}

	// Skip whitespace at start of shell content or start of lines, but preserve mid-command whitespace
	// Check if we're at start of shell content (after colon) or start of new line
	shouldSkipWhitespace := false
	if l.position > 0 {
		prevChar := l.input[l.position-1]
		if prevChar == ':' || prevChar == '{' || prevChar == '\n' {
			shouldSkipWhitespace = true
		}
	}

	if shouldSkipWhitespace {
		l.skipWhitespace()
	}

	start := l.position
	startLine, startColumn := l.line, l.column

	switch l.ch {
	case 0:
		return l.createToken(types.EOF, "", start, startLine, startColumn)

	case '\n':
		// Newlines end shell content in shell mode (unless line continuation)
		l.readChar()
		// Simple rule: determine next mode based on context
		if l.braceLevel == 0 {
			l.mode = LanguageMode
			// Only reset shell context when truly exiting shell mode, not during temporary decorator parsing
			if !l.inFunctionDecorator {
				l.shellBraceLevel = 0
				l.shellParenLevel = 0
				l.shellAnyBraceLevel = 0
				// Reset quote state when exiting shell mode
				l.shellInSingleQuote = false
				l.shellInDoubleQuote = false
				l.shellInBacktick = false
			}
		} else if l.isInPatternContext() && l.braceLevel == l.patternBraceLevel {
			// Only return to PatternMode if we're at the exact pattern brace level
			l.mode = PatternMode
		}
		// Otherwise stay in ShellMode for regular braced blocks
		return l.NextToken()

	case '}':
		// Closing brace - need to check if it's shell syntax or block boundary
		if l.shellBraceLevel > 0 || l.shellAnyBraceLevel > 0 {
			// This is shell syntax, not a block boundary - continue with shell text
			return l.lexShellTextWithContext(start, startLine, startColumn)
		} else {
			// This is a block boundary - check if we need SHELL_END first
			if l.needsShellEnd {
				// Emit SHELL_END first, then handle } on next call
				l.needsShellEnd = false
				// Set up mode transition for after SHELL_END
				if l.braceLevel <= 0 {
					l.mode = LanguageMode
					l.patternBraceLevel = 0 // Clear pattern context
				} else if l.isInPatternContext() && l.braceLevel == l.patternBraceLevel {
					l.mode = PatternMode
				} else {
					l.mode = CommandMode
				}
				// Only reset shell context when truly exiting shell mode, not during temporary decorator parsing
				if !l.inFunctionDecorator {
					l.shellBraceLevel = 0
					l.shellParenLevel = 0
					l.shellAnyBraceLevel = 0
				}
				// Reset position tracking to allow next token processing
				l.lastPosition = -1
				pos, line, col := l.getShellEndPosition(start, startLine, startColumn)
				return l.createToken(types.SHELL_END, "", pos, line, col)
			} else {
				// No SHELL_END needed - set up mode transition and handle }
				if l.braceLevel <= 0 {
					l.mode = LanguageMode
					l.patternBraceLevel = 0 // Clear pattern context
				} else if l.isInPatternContext() && l.braceLevel == l.patternBraceLevel {
					l.mode = PatternMode
				} else {
					l.mode = CommandMode
				}
				// Only reset shell context when truly exiting shell mode, not during temporary decorator parsing
				if !l.inFunctionDecorator {
					l.shellBraceLevel = 0
					l.shellParenLevel = 0
					l.shellAnyBraceLevel = 0
				}
				// Let the appropriate mode handle the } token
				return l.NextToken()
			}
		}

	case '{':
		// Opening brace in shell mode - start new block
		l.readChar()
		l.braceLevel++
		return l.createToken(types.LBRACE, "{", start, startLine, startColumn)

	case '@':
		// Handle decorators inline - check if it's a registered decorator
		return l.lexDecoratorInShell(start, startLine, startColumn)

	default:
		// Handle shell content with maintained context
		return l.lexShellTextWithContext(start, startLine, startColumn)
	}
}

// lexDecoratorInShell handles decorators inline in ShellMode
func (l *Lexer) lexDecoratorInShell(start, startLine, startColumn int) types.Token {
	// Look ahead to check if this is @identifier pattern
	savedPos := l.position
	savedReadPos := l.readPos
	savedCh := l.ch
	savedLine := l.line
	savedColumn := l.column

	// Skip @
	l.readChar()
	l.skipWhitespace()

	// Check if followed by identifier
	if (l.ch < 128 && isIdentStart[l.ch]) || (l.ch >= 128 && (unicode.IsLetter(l.ch) || l.ch == '_')) {
		// Read the identifier to check if it's a decorator
		identStart := l.position
		for {
			if l.ch < 128 && isIdentPart[l.ch] {
				l.readChar()
			} else if l.ch >= 128 && (unicode.IsLetter(l.ch) || unicode.IsDigit(l.ch)) {
				l.readChar()
			} else {
				break
			}
		}
		identifier := l.input[identStart:l.position]

		// Check if it's a registered decorator
		if decorators.IsDecorator(identifier) {
			// Handle different decorator types appropriately
			if decorators.IsBlockDecorator(identifier) || decorators.IsPatternDecorator(identifier) {
				// Switch to LanguageMode for block/pattern decorator parsing
				l.mode = LanguageMode

				// Reset position to @ and advance past it
				l.position = savedPos
				l.readPos = savedReadPos
				l.ch = savedCh
				l.line = savedLine
				l.column = savedColumn
				l.readChar() // Skip the @ character

				return l.createToken(types.AT, "@", start, startLine, startColumn)
			} else if decorators.IsDecorator(identifier) {
				// Check if followed by parentheses for function decorators
				if l.ch == '(' {
					// This is a function decorator - switch to LanguageMode for the decorator sequence
					l.originalMode = l.mode // Save the original mode before switching to LanguageMode
					l.mode = LanguageMode
					l.inFunctionDecorator = true

					// Reset position to @ and advance past it
					l.position = savedPos
					l.readPos = savedReadPos
					l.ch = savedCh
					l.line = savedLine
					l.column = savedColumn
					l.readChar() // Skip the @ character

					return l.createToken(types.AT, "@", start, startLine, startColumn)
				}
			}
		}
	}

	// Restore position - this is shell text starting with @
	l.position = savedPos
	l.readPos = savedReadPos
	l.ch = savedCh
	l.line = savedLine
	l.column = savedColumn

	return l.lexShellTextWithContext(start, startLine, startColumn)
}

// checkShellOperator checks if the current position starts a shell operator
// Returns the token type and length if found, otherwise returns ILLEGAL and 0
func (l *Lexer) checkShellOperator() (types.TokenType, int) {
	if l.ch == '&' && l.peekChar() == '&' {
		return types.AND, 2
	}
	if l.ch == '|' {
		next := l.peekChar()
		if next == '|' {
			return types.OR, 2
		}
		// Single pipe
		return types.PIPE, 1
	}
	if l.ch == '>' && l.peekChar() == '>' {
		return types.APPEND, 2
	}
	return types.ILLEGAL, 0
}

// lexShellTextWithContext handles shell content in ShellMode with maintained global context
func (l *Lexer) lexShellTextWithContext(start, startLine, startColumn int) types.Token {
	var result strings.Builder
	// Use global quote state to maintain context across decorator breaks
	inSingleQuote := l.shellInSingleQuote
	inDoubleQuote := l.shellInDoubleQuote
	inBacktick := l.shellInBacktick

	for l.ch != 0 {
		// Check for quoted strings first - these should be extracted as STRING tokens
		if (l.ch == '"' || l.ch == '\'' || l.ch == '`') && !inSingleQuote && !inDoubleQuote && !inBacktick {
			// If we have accumulated shell text, return it first
			if result.Len() > 0 {
				text := result.String()
				return l.createToken(types.SHELL_TEXT, text, start, startLine, startColumn)
			}
			// Extract the string as a separate STRING token
			return l.lexString(l.ch, l.position, l.line, l.column)
		}

		// Stop at newline (unless line continuation or inside quotes)
		if l.ch == '\n' {
			// Check for line continuation (backslash before newline)
			if l.position > 0 && l.input[l.position-1] == '\\' && !inSingleQuote {
				// Line continuation - remove the backslash
				text := result.String()
				if len(text) > 0 && text[len(text)-1] == '\\' {
					result.Reset()
					result.WriteString(text[:len(text)-1]) // Remove the backslash
				}
				l.readChar() // Skip newline
				// Skip leading whitespace on the next line
				for l.ch == ' ' || l.ch == '\t' {
					l.readChar()
				}
				continue
			}

			// If inside single quotes, include the newline literally
			if inSingleQuote {
				result.WriteRune(l.ch)
				l.readChar()
				continue
			}

			// If inside double quotes or backticks without line continuation, include newline
			if inDoubleQuote || inBacktick {
				result.WriteRune(l.ch)
				l.readChar()
				continue
			}

			// Not in quotes and no line continuation - end of shell text
			break
		}

		// Stop at closing brace (block boundary) - unless inside quotes or shell constructs
		if l.ch == '}' && !inSingleQuote && !inDoubleQuote && !inBacktick {
			if l.shellBraceLevel > 0 {
				// This is closing a shell parameter expansion ${...}
				l.shellBraceLevel--
			} else if l.shellAnyBraceLevel > 0 {
				// This is closing some other shell brace construct
				l.shellAnyBraceLevel--
			} else {
				// This is a block boundary - only break if we're not inside any shell constructs
				break
			}
		}

		// Check for shell operators when not inside quotes
		if !inSingleQuote && !inDoubleQuote && !inBacktick {
			if opType, opLen := l.checkShellOperator(); opType != types.ILLEGAL {
				// Found a shell operator - return shell text so far if we have any
				if result.Len() > 0 {
					text := strings.TrimSpace(result.String())
					if text != "" {
						// Save quote state before returning
						l.shellInSingleQuote = inSingleQuote
						l.shellInDoubleQuote = inDoubleQuote
						l.shellInBacktick = inBacktick
						return l.createToken(types.SHELL_TEXT, text, start, startLine, startColumn)
					}
				}
				// No shell text before operator - return the operator token
				operatorStart := l.position
				operatorLine, operatorColumn := l.line, l.column
				operatorValue := l.input[l.position : l.position+opLen]
				// Advance past the operator
				for i := 0; i < opLen; i++ {
					l.readChar()
				}
				// Save quote state before returning
				l.shellInSingleQuote = inSingleQuote
				l.shellInDoubleQuote = inDoubleQuote
				l.shellInBacktick = inBacktick
				return l.createToken(opType, operatorValue, operatorStart, operatorLine, operatorColumn)
			}
		}

		// Stop at @ if it starts any registered decorator - allow inside double quotes for decorator expansion
		if l.ch == '@' && !inSingleQuote {
			// Look ahead to see if this is @identifier for any registered decorator
			if l.readPos < len(l.input) {
				nextCh, _ := utf8.DecodeRuneInString(l.input[l.readPos:])
				if (nextCh < 128 && isIdentStart[nextCh]) || (nextCh >= 128 && (unicode.IsLetter(nextCh) || nextCh == '_')) {
					// Check if this is any registered decorator by reading ahead
					savedPos := l.position
					savedReadPos := l.readPos
					savedCh := l.ch

					// Skip @ and read identifier
					l.readChar()
					identStart := l.position
					for {
						if l.ch < 128 && isIdentPart[l.ch] {
							l.readChar()
						} else if l.ch >= 128 && (unicode.IsLetter(l.ch) || unicode.IsDigit(l.ch)) {
							l.readChar()
						} else {
							break
						}
					}
					identifier := l.input[identStart:l.position]

					// Check what follows the identifier
					hasOpenParen := l.ch == '('

					// Restore position
					l.position = savedPos
					l.readPos = savedReadPos
					l.ch = savedCh

					// Break for block/pattern decorators (they switch to LanguageMode)
					if decorators.IsBlockDecorator(identifier) || decorators.IsPatternDecorator(identifier) {
						break
					}

					// Break for function decorators with parentheses (we'll tokenize them here)
					if decorators.IsDecorator(identifier) && hasOpenParen {
						break
					}
				}
			}
		}

		// Track various shell constructs BEFORE adding character
		if !inSingleQuote && !inDoubleQuote && !inBacktick {
			// Track shell parameter expansion ${...}
			if l.ch == '$' && l.peekChar() == '{' {
				l.shellBraceLevel++
			}
			// Track command substitution $(...)
			if l.ch == '$' && l.peekChar() == '(' {
				l.shellParenLevel++
			}
			// Track closing parentheses for command substitution
			if l.ch == ')' && l.shellParenLevel > 0 {
				l.shellParenLevel--
			}
			// Track standalone braces in shell context (brace expansion, find {})
			if l.ch == '{' {
				// Check if this is part of ${...} (already handled above)
				if result.Len() > 0 {
					lastChar := result.String()[result.Len()-1]
					if lastChar != '$' {
						l.shellAnyBraceLevel++
					}
				} else {
					l.shellAnyBraceLevel++
				}
			}
		}

		// Add character to result
		result.WriteRune(l.ch)

		// Track quote state AFTER adding character
		if l.ch == '\'' && !inDoubleQuote && !inBacktick {
			inSingleQuote = !inSingleQuote
		} else if l.ch == '"' && !inSingleQuote && !inBacktick {
			inDoubleQuote = !inDoubleQuote
		} else if l.ch == '`' && !inSingleQuote && !inDoubleQuote {
			inBacktick = !inBacktick
		}

		l.readChar()
	}

	text := result.String()
	if text == "" {
		return l.createToken(types.ILLEGAL, "", start, startLine, startColumn)
	}

	// Minimal whitespace trimming for ShellMode - preserve shell syntax
	originalText := text

	// Determine why we stopped (what comes next)
	stoppedAtNewline := l.ch == '\n'
	stoppedAtBrace := l.ch == '}'
	stoppedAtEOF := l.ch == 0

	// Only trim trailing whitespace when ending command (not when continuing after decorators)
	if stoppedAtNewline || stoppedAtBrace || stoppedAtEOF {
		// Trim trailing whitespace when ending command
		text = strings.TrimRight(text, " \t")

		// If we have shell text, return it first, SHELL_END will come next
		if text != "" {
			l.needsShellEnd = true
			// Set SHELL_END position to end of current line (ensures column > 0)
			l.shellEndPosition = l.position
			l.shellEndLine = l.line
			l.shellEndColumn = max(1, l.column) // Ensure column is at least 1
			// Save quote state before returning
			l.shellInSingleQuote = inSingleQuote
			l.shellInDoubleQuote = inDoubleQuote
			l.shellInBacktick = inBacktick
			return l.createToken(types.SHELL_TEXT, text, start, startLine, startColumn)
		} else {
			// No shell text, emit SHELL_END directly with safe position
			pos, line, col := l.getShellEndPosition(start, startLine, startColumn)
			// Save quote state before returning
			l.shellInSingleQuote = inSingleQuote
			l.shellInDoubleQuote = inDoubleQuote
			l.shellInBacktick = inBacktick
			return l.createToken(types.SHELL_END, "", pos, line, col)
		}
	}
	// Preserve all other whitespace to maintain shell syntax

	// If trimming resulted in empty string, fall back to original
	if text == "" && originalText != "" {
		text = originalText
	}

	if text == "" {
		// Save quote state before returning
		l.shellInSingleQuote = inSingleQuote
		l.shellInDoubleQuote = inDoubleQuote
		l.shellInBacktick = inBacktick
		return l.createToken(types.ILLEGAL, "", start, startLine, startColumn)
	}

	// Save quote state before returning
	l.shellInSingleQuote = inSingleQuote
	l.shellInDoubleQuote = inDoubleQuote
	l.shellInBacktick = inBacktick
	return l.createToken(types.SHELL_TEXT, text, start, startLine, startColumn)
}

// lexShellText handles shell content in CommandMode
func (l *Lexer) lexShellText(start, startLine, startColumn int) types.Token {
	var result strings.Builder
	var inSingleQuote, inDoubleQuote, inBacktick bool
	var shellBraceLevel int // Track ${...} parameter expansion braces
	var parenLevel int      // Track $(...) command substitution
	var anyBraceLevel int   // Track any {...} constructs in shell context

	// Determine if this is the start of shell content by checking if we're at beginning of line or after colon
	isStartOfCommand := false
	if start > 0 {
		// Look back to see if this follows a colon (start of command) or whitespace after colon
		pos := start - 1
		for pos >= 0 && (l.input[pos] == ' ' || l.input[pos] == '\t') {
			pos--
		}
		if pos >= 0 && l.input[pos] == ':' {
			isStartOfCommand = true
		}
	}

	for l.ch != 0 {
		// Check for quoted strings first - these should be extracted as STRING tokens
		if (l.ch == '"' || l.ch == '\'' || l.ch == '`') && !inSingleQuote && !inDoubleQuote && !inBacktick {
			// If we have accumulated shell text, return it first
			if result.Len() > 0 {
				text := result.String()
				return l.createToken(types.SHELL_TEXT, text, start, startLine, startColumn)
			}
			// Extract the string as a separate STRING token
			return l.lexString(l.ch, l.position, l.line, l.column)
		}

		// Stop at newline (unless line continuation or inside quotes)
		if l.ch == '\n' {
			// Check for line continuation (backslash before newline)
			// Process line continuation when NOT inside single quotes (but do process in double quotes and backticks)
			if l.position > 0 && l.input[l.position-1] == '\\' && !inSingleQuote {
				// Line continuation - remove the backslash
				text := result.String()
				if len(text) > 0 && text[len(text)-1] == '\\' {
					result.Reset()
					result.WriteString(text[:len(text)-1]) // Remove the backslash
				}
				l.readChar() // Skip newline
				// Skip leading whitespace on the next line
				for l.ch == ' ' || l.ch == '\t' {
					l.readChar()
				}
				continue
			}

			// If inside single quotes, include the newline literally
			if inSingleQuote {
				result.WriteRune(l.ch)
				l.readChar()
				continue
			}

			// If inside double quotes or backticks without line continuation, include newline
			if inDoubleQuote || inBacktick {
				result.WriteRune(l.ch)
				l.readChar()
				continue
			}

			// Not in quotes and no line continuation - end of shell text
			break
		}

		// Stop at closing brace (block boundary) - unless inside quotes or shell constructs
		if l.ch == '}' && !inSingleQuote && !inDoubleQuote && !inBacktick {
			if shellBraceLevel > 0 {
				// This is closing a shell parameter expansion ${...}
				shellBraceLevel--
			} else if anyBraceLevel > 0 {
				// This is closing some other shell brace construct
				anyBraceLevel--
			} else {
				// This is a block boundary - only break if we're not inside any shell constructs
				break
			}
		}

		// Check for shell operators when not inside quotes
		if !inSingleQuote && !inDoubleQuote && !inBacktick {
			if opType, opLen := l.checkShellOperator(); opType != types.ILLEGAL {
				// Found a shell operator - return shell text so far if we have any
				if result.Len() > 0 {
					text := strings.TrimSpace(result.String())
					if text != "" {
						return l.createToken(types.SHELL_TEXT, text, start, startLine, startColumn)
					}
				}
				// No shell text before operator - return the operator token
				operatorStart := l.position
				operatorLine, operatorColumn := l.line, l.column
				operatorValue := l.input[l.position : l.position+opLen]
				// Advance past the operator
				for i := 0; i < opLen; i++ {
					l.readChar()
				}
				return l.createToken(opType, operatorValue, operatorStart, operatorLine, operatorColumn)
			}
		}

		// Stop at @ if it starts any registered decorator - allow inside double quotes for decorator expansion
		if l.ch == '@' && !inSingleQuote {
			// Look ahead to see if this is @identifier for any registered decorator
			if l.readPos < len(l.input) {
				nextCh, _ := utf8.DecodeRuneInString(l.input[l.readPos:])
				if (nextCh < 128 && isIdentStart[nextCh]) || (nextCh >= 128 && (unicode.IsLetter(nextCh) || nextCh == '_')) {
					// Check if this is any registered decorator by reading ahead
					savedPos := l.position
					savedReadPos := l.readPos
					savedCh := l.ch

					// Skip @ and read identifier
					l.readChar()
					identStart := l.position
					for {
						if l.ch < 128 && isIdentPart[l.ch] {
							l.readChar()
						} else if l.ch >= 128 && (unicode.IsLetter(l.ch) || unicode.IsDigit(l.ch)) {
							l.readChar()
						} else {
							break
						}
					}
					identifier := l.input[identStart:l.position]

					// Check what follows the identifier
					hasOpenParen := l.ch == '('

					// Restore position
					l.position = savedPos
					l.readPos = savedReadPos
					l.ch = savedCh

					// Break for block/pattern decorators (they switch to LanguageMode)
					if decorators.IsBlockDecorator(identifier) || decorators.IsPatternDecorator(identifier) {
						break
					}

					// Break for function decorators with parentheses (we'll tokenize them here)
					if decorators.IsDecorator(identifier) && hasOpenParen {
						break
					}
				}
			}
		}

		// Track various shell constructs BEFORE adding character
		if !inSingleQuote && !inDoubleQuote && !inBacktick {
			// Track shell parameter expansion ${...}
			if l.ch == '$' && l.peekChar() == '{' {
				shellBraceLevel++
			}
			// Track command substitution $(...)
			if l.ch == '$' && l.peekChar() == '(' {
				parenLevel++
			}
			// Track closing parentheses for command substitution
			if l.ch == ')' && parenLevel > 0 {
				parenLevel--
			}
			// Track standalone braces in shell context (brace expansion, find {})
			if l.ch == '{' {
				// Check if this is part of ${...} (already handled above)
				if result.Len() > 0 {
					lastChar := result.String()[result.Len()-1]
					if lastChar != '$' {
						anyBraceLevel++
					}
				} else {
					anyBraceLevel++
				}
			}
		}

		// Add character to result
		result.WriteRune(l.ch)

		// Track quote state AFTER adding character
		if l.ch == '\'' && !inDoubleQuote && !inBacktick {
			inSingleQuote = !inSingleQuote
		} else if l.ch == '"' && !inSingleQuote && !inBacktick {
			inDoubleQuote = !inDoubleQuote
		} else if l.ch == '`' && !inSingleQuote && !inDoubleQuote {
			inBacktick = !inBacktick
		}

		l.readChar()
	}

	text := result.String()
	if text == "" {
		return l.createToken(types.ILLEGAL, "", start, startLine, startColumn)
	}

	// Context-aware whitespace trimming
	originalText := text

	// Determine why we stopped (what comes next)
	stoppedAtDecorator := l.ch == '@'
	stoppedAtNewline := l.ch == '\n'
	stoppedAtBrace := l.ch == '}'
	stoppedAtEOF := l.ch == 0

	// Apply trimming rules based on context:
	// 1. If we're at start of command AND stopping at decorator, preserve trailing space
	// 2. If we're stopping at newline/brace/EOF, trim trailing whitespace
	// 3. If we're in middle (after decorator), trim leading whitespace

	if isStartOfCommand && stoppedAtDecorator {
		// Keep trailing space for cases like "cd @var(DIR)" - preserve the "cd " part
		text = strings.TrimLeft(text, " \t")
	} else if stoppedAtNewline || stoppedAtBrace || stoppedAtEOF {
		// Trim trailing whitespace when ending command
		text = strings.TrimRight(text, " \t")
		if !isStartOfCommand {
			// Also trim leading if not start of command (middle of sequence)
			text = strings.TrimLeft(text, " \t")
		}
	} else if stoppedAtDecorator && !isStartOfCommand {
		// Middle of sequence, trim both sides but keep internal spaces
		text = strings.TrimSpace(text)
	}

	// If trimming resulted in empty string, fall back to original
	if text == "" && originalText != "" {
		text = originalText
	}

	if text == "" {
		return l.createToken(types.ILLEGAL, "", start, startLine, startColumn)
	}

	return l.createToken(types.SHELL_TEXT, text, start, startLine, startColumn)
}

// lexIdentifierOrKeyword handles identifiers and keywords (using fast ASCII lookups)
func (l *Lexer) lexIdentifierOrKeyword(start, startLine, startColumn int) types.Token {
	for {
		// Fast path for ASCII
		if l.ch < 128 && isIdentPart[l.ch] {
			l.readChar()
		} else if l.ch >= 128 && (unicode.IsLetter(l.ch) || unicode.IsDigit(l.ch)) {
			// Fallback for non-ASCII
			l.readChar()
		} else {
			break
		}
	}

	value := l.input[start:l.position]

	// No need to track decorator names with simplified approach

	// Check for keywords - but only in appropriate contexts
	switch value {
	case "var":
		// Only treat 'var' as VAR token if we're not in decorator context
		if l.inFunctionDecorator {
			return l.createToken(types.IDENTIFIER, value, start, startLine, startColumn)
		}
		return l.createToken(types.VAR, value, start, startLine, startColumn)
	case "watch":
		return l.createToken(types.WATCH, value, start, startLine, startColumn)
	case "stop":
		return l.createToken(types.STOP, value, start, startLine, startColumn)
	case "true", "false":
		return l.createToken(types.BOOLEAN, value, start, startLine, startColumn)
	default:
		return l.createToken(types.IDENTIFIER, value, start, startLine, startColumn)
	}
}

// lexString handles string literals (quoted strings)
// All strings now use the unified STRING_START/TEXT/END token system
func (l *Lexer) lexString(quote rune, start, startLine, startColumn int) types.Token {
	// Both single and double quotes now use the same tokenization system
	// Single quotes disable interpolation, double quotes enable it
	return l.lexInterpolatedStringStart(quote, start, startLine, startColumn)
}

// lexInterpolatedStringStart handles the beginning of interpolated strings (double quotes and backticks)
func (l *Lexer) lexInterpolatedStringStart(quote rune, start, startLine, startColumn int) types.Token {
	l.logger.Debug("[LEXER] Starting interpolated string",
		"quote", string(quote),
		"currentMode", l.mode,
		"position", l.position,
		"braceLevel", l.braceLevel)

	// Return STRING_START token with the opening quote
	quoteStr := string(quote)
	l.readChar() // Skip the opening quote

	// Set state for interpolated string parsing
	l.inInterpolatedString = true
	l.interpolatedQuote = quote
	// Use originalMode if we're inside a function decorator, otherwise use current mode
	if l.inFunctionDecorator && l.originalMode != 0 {
		l.preInterpolatedMode = l.originalMode
	} else {
		l.preInterpolatedMode = l.mode
	}

	l.logger.Debug("[LEXER] Set interpolated string state",
		"inInterpolatedString", l.inInterpolatedString,
		"interpolatedQuote", string(l.interpolatedQuote),
		"preInterpolatedMode", l.preInterpolatedMode)

	// Create STRING_START token
	token := l.createToken(types.STRING_START, quoteStr, start, startLine, startColumn)

	// Set the string type based on quote character
	switch quote {
	case '"':
		token.StringType = types.DoubleQuoted
	case '\'':
		token.StringType = types.SingleQuoted
	case '`':
		token.StringType = types.Backtick
	}

	// Store the raw quote character
	token.Raw = quoteStr

	return token
}

// lexInterpolatedStringContent parses content inside interpolated strings
func (l *Lexer) lexInterpolatedStringContent() types.Token {
	start := l.position
	startLine, startColumn := l.line, l.column

	// Check for closing quote first
	if l.ch == l.interpolatedQuote {
		return l.lexInterpolatedStringEnd(start, startLine, startColumn)
	}

	// Check for EOF
	if l.ch == 0 {
		// Unterminated string
		l.inInterpolatedString = false
		return l.createToken(types.ILLEGAL, "Unterminated string", start, startLine, startColumn)
	}

	// Check for value decorator (@)
	if l.ch == '@' {
		// Look ahead to check if this is a value decorator (@var or @env)
		savedPos := l.position
		savedReadPos := l.readPos
		savedCh := l.ch
		savedLine := l.line
		savedColumn := l.column

		// Skip @
		l.readChar()

		// Check if followed by identifier
		if (l.ch >= 128 || !isIdentStart[l.ch]) && (l.ch < 128 || (!unicode.IsLetter(l.ch) && l.ch != '_')) {
			// Not an identifier after @, treat as regular text
			l.position = savedPos
			l.readPos = savedReadPos
			l.ch = savedCh
			l.line = savedLine
			l.column = savedColumn
			return l.lexStringText(start, startLine, startColumn)
		}

		// Read the identifier
		identStart := l.position
		for (l.ch < 128 && isIdentPart[l.ch]) || (l.ch >= 128 && (unicode.IsLetter(l.ch) || unicode.IsDigit(l.ch))) {
			l.readChar()
		}
		identifier := l.input[identStart:l.position]

		// Check if this is a registered value decorator
		if decorators.IsValueDecorator(identifier) && l.ch == '(' {
			// It's a valid value decorator, restore position and parse it
			l.position = savedPos
			l.readPos = savedReadPos
			l.ch = savedCh
			l.line = savedLine
			l.column = savedColumn
			return l.lexDecoratorInInterpolatedString(start, startLine, startColumn)
		}

		// Not a value decorator, treat @ as regular text
		l.position = savedPos
		l.readPos = savedReadPos
		l.ch = savedCh
		l.line = savedLine
		l.column = savedColumn
		return l.lexStringText(start, startLine, startColumn)
	}

	// Parse regular text content
	return l.lexStringText(start, startLine, startColumn)
}

// lexInterpolatedStringEnd handles the closing quote of interpolated strings
func (l *Lexer) lexInterpolatedStringEnd(start, startLine, startColumn int) types.Token {
	l.logger.Debug("[LEXER] Ending interpolated string",
		"quote", string(l.interpolatedQuote),
		"currentMode", l.mode,
		"preInterpolatedMode", l.preInterpolatedMode,
		"inStringDecorator", l.inStringDecorator,
		"position", l.position)

	quoteStr := string(l.interpolatedQuote)
	l.readChar() // Skip the closing quote

	// Reset interpolated string state and restore previous mode
	l.inInterpolatedString = false
	l.interpolatedQuote = 0

	// If we're still inside a function decorator, stay in LanguageMode
	// Otherwise, restore to the original mode
	if l.inFunctionDecorator {
		l.mode = LanguageMode
	} else {
		l.mode = l.preInterpolatedMode
	}

	l.logger.Debug("[LEXER] Mode restored after string end",
		"newMode", l.mode,
		"preInterpolatedMode", l.preInterpolatedMode,
		"inFunctionDecorator", l.inFunctionDecorator,
		"inInterpolatedString", l.inInterpolatedString,
		"braceLevel", l.braceLevel)

	// Check if we're at the end of a shell command and should emit SHELL_END next
	if l.mode == ShellMode {
		// Skip whitespace to see what comes next
		pos := l.position
		ch := l.ch
		for ch == ' ' || ch == '\t' {
			pos++
			if pos >= len(l.input) {
				ch = 0
				break
			}
			ch = rune(l.input[pos])
		}
		// If next non-whitespace is newline, EOF, or closing brace, this ends the shell command
		if ch == '\n' || ch == 0 || (ch == '}' && l.braceLevel > 0) {
			l.needsShellEnd = true
		}
	}

	// Create STRING_END token
	token := l.createToken(types.STRING_END, quoteStr, start, startLine, startColumn)
	token.Raw = quoteStr

	return token
}

// lexStringText parses text content within interpolated strings (until @ or closing quote)
func (l *Lexer) lexStringText(start, startLine, startColumn int) types.Token {
	// Read until we hit the closing quote or a potential decorator
	for l.ch != 0 && l.ch != l.interpolatedQuote {
		// Only check for interpolation in double quotes and backticks, not single quotes
		if l.ch == '@' && (l.interpolatedQuote == '"' || l.interpolatedQuote == '`') {
			// Check if this is really a decorator
			savedPos := l.position
			savedReadPos := l.readPos
			savedCh := l.ch

			// Skip @
			l.readChar()

			// Check if it's var or env followed by (
			if l.ch == 'v' || l.ch == 'e' {
				identStart := l.position
				for (l.ch < 128 && isIdentPart[l.ch]) || (l.ch >= 128 && (unicode.IsLetter(l.ch) || unicode.IsDigit(l.ch))) {
					l.readChar()
				}
				identifier := l.input[identStart:l.position]

				if decorators.IsValueDecorator(identifier) && l.ch == '(' {
					// It's a registered value decorator, stop here
					l.position = savedPos
					l.readPos = savedReadPos
					l.ch = savedCh
					break
				}
			}

			// Not a decorator, restore and continue
			l.position = savedPos
			l.readPos = savedReadPos
			l.ch = savedCh
		}
		l.readChar()
	}

	text := l.input[start:l.position]

	// Fix position tracking: if we started with column 0, adjust to 1 (1-based indexing)
	if startColumn == 0 {
		startColumn = 1
	}

	return l.createToken(types.STRING_TEXT, text, start, startLine, startColumn)
}

// lexDecoratorInInterpolatedString handles decorators within interpolated strings
// by switching to LanguageMode for proper parameter parsing
func (l *Lexer) lexDecoratorInInterpolatedString(start, startLine, startColumn int) types.Token {
	l.logger.Debug("[LEXER] Found decorator in interpolated string",
		"currentMode", l.mode,
		"preInterpolatedMode", l.preInterpolatedMode,
		"position", l.position,
		"ch", string(l.ch))

	// Switch to LanguageMode for decorator parsing - this allows full parameter parsing
	// including quoted strings, named parameters, etc.
	l.mode = LanguageMode
	l.inFunctionDecorator = true
	l.inStringDecorator = true // Flag that we're in a decorator within a string

	l.logger.Debug("[LEXER] Set flags for decorator in string",
		"mode", l.mode,
		"inFunctionDecorator", l.inFunctionDecorator,
		"inStringDecorator", l.inStringDecorator)

	// Return the AT token, subsequent calls will parse the decorator normally
	l.readChar() // Skip @
	return l.createToken(types.AT, "@", start, startLine, startColumn)
}

// lexLiteralStringInDecorator handles quoted strings within decorator parameters
// This avoids creating nested interpolated string contexts
func (l *Lexer) lexLiteralStringInDecorator(quote rune, start, startLine, startColumn int) types.Token {
	l.logger.Debug("[LEXER] Parsing literal string in decorator",
		"quote", string(quote),
		"position", l.position)

	// Return STRING_START token with the opening quote
	quoteStr := string(quote)
	l.readChar() // Skip the opening quote

	// Create STRING_START token
	token := l.createToken(types.STRING_START, quoteStr, start, startLine, startColumn)

	// Set the string type based on quote character
	switch quote {
	case '"':
		token.StringType = types.DoubleQuoted
	case '\'':
		token.StringType = types.SingleQuoted
	case '`':
		token.StringType = types.Backtick
	}

	token.Raw = quoteStr

	// Set state for simple string parsing (no interpolation)
	l.inLiteralString = true
	l.literalQuote = quote

	return token
}

// lexLiteralStringContent handles the content inside literal strings (no interpolation)
func (l *Lexer) lexLiteralStringContent() types.Token {
	start := l.position
	startLine, startColumn := l.line, l.column

	// Check for closing quote first
	if l.ch == l.literalQuote {
		return l.lexLiteralStringEnd(start, startLine, startColumn)
	}

	// Check for EOF
	if l.ch == 0 {
		// Unterminated string
		l.inLiteralString = false
		return l.createToken(types.ILLEGAL, "Unterminated string", start, startLine, startColumn)
	}

	// Read until we hit the closing quote (no interpolation checks)
	for l.ch != 0 && l.ch != l.literalQuote {
		l.readChar()
	}

	text := l.input[start:l.position]

	// Fix position tracking: if we started with column 0, adjust to 1 (1-based indexing)
	if startColumn == 0 {
		startColumn = 1
	}

	return l.createToken(types.STRING_TEXT, text, start, startLine, startColumn)
}

// lexLiteralStringEnd handles the closing quote of a literal string
func (l *Lexer) lexLiteralStringEnd(start, startLine, startColumn int) types.Token {
	quoteStr := string(l.literalQuote)
	l.readChar() // Skip the closing quote

	// Reset literal string state
	l.inLiteralString = false
	l.literalQuote = 0

	return l.createToken(types.STRING_END, quoteStr, start, startLine, startColumn)
}

// lexNumber handles number literals (using fast ASCII lookups)
func (l *Lexer) lexNumber(start, startLine, startColumn int) types.Token {
	hasDecimal := false

	// Handle negative sign if present
	if l.ch == '-' {
		l.readChar()
	}

	for {
		// Fast path for ASCII digits
		if l.ch < 128 && isDigit[l.ch] {
			l.readChar()
		} else if l.ch == '.' && !hasDecimal {
			hasDecimal = true
			l.readChar()
		} else if l.ch >= 128 && unicode.IsDigit(l.ch) {
			// Fallback for non-ASCII digits
			l.readChar()
		} else {
			break
		}
	}

	// Check for duration suffix using fast ASCII lookups
	if (l.ch < 128 && isLetter[l.ch]) || (l.ch >= 128 && unicode.IsLetter(l.ch)) {
		durStart := l.position
		for {
			if l.ch < 128 && isLetter[l.ch] {
				l.readChar()
			} else if l.ch >= 128 && unicode.IsLetter(l.ch) {
				l.readChar()
			} else {
				break
			}
		}
		suffix := l.input[durStart:l.position]

		// Valid duration suffixes
		switch suffix {
		case "ns", "us", "ms", "s", "m", "h":
			value := l.input[start:l.position]
			return l.createToken(types.DURATION, value, start, startLine, startColumn)
		default:
			// Invalid suffix - treat as separate tokens
			l.position = durStart
			l.readPos = durStart + utf8.RuneLen(l.ch)
			l.ch, _ = utf8.DecodeRuneInString(l.input[durStart:])
		}
	}

	value := l.input[start:l.position]
	return l.createToken(types.NUMBER, value, start, startLine, startColumn)
}

// lexComment handles comment lines starting with #
func (l *Lexer) lexComment(start, startLine, startColumn int) types.Token {
	// Read from # to end of line
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}

	value := l.input[start:l.position]
	return l.createToken(types.COMMENT, value, start, startLine, startColumn)
}

// lexMultilineComment handles multi-line comments /* */
func (l *Lexer) lexMultilineComment(start, startLine, startColumn int) types.Token {
	// Skip /*
	l.readChar() // Skip /
	l.readChar() // Skip *

	// Read until */
	for {
		if l.ch == 0 {
			// Unterminated comment
			return l.createToken(types.ILLEGAL, "unterminated comment", start, startLine, startColumn)
		}

		if l.ch == '*' && l.peekChar() == '/' {
			l.readChar() // Skip *
			l.readChar() // Skip /
			break
		}

		l.readChar()
	}

	value := l.input[start:l.position]
	return l.createToken(types.MULTILINE_COMMENT, value, start, startLine, startColumn)
}
