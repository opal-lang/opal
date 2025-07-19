package lexer

import (
	"fmt"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/aledsdavies/devcmd/pkgs/stdlib"
)

// ASCII character lookup tables for fast classification
var (
	isWhitespace      [128]bool // Only ASCII range
	isLetter          [128]bool
	isDigit           [128]bool
	isIdentStart      [128]bool
	isIdentPart       [128]bool
	singleCharTokens  [128]TokenType // Fast lookup for single-char tokens
	singleCharStrings [128]string    // Pre-allocated single-char strings
)

func init() {
	for i := 0; i < 128; i++ {
		ch := byte(i)
		isWhitespace[i] = ch == ' ' || ch == '\t' || ch == '\r' || ch == '\f'
		isLetter[i] = ('a' <= ch && ch <= 'z') || ('A' <= ch && ch <= 'Z') || ch == '_'
		isDigit[i] = '0' <= ch && ch <= '9'
		isIdentStart[i] = isLetter[i] || ch == '_'
		isIdentPart[i] = isIdentStart[i] || isDigit[i] || ch == '-'
		singleCharTokens[i] = ILLEGAL     // Default to ILLEGAL for non-single-char tokens
		singleCharStrings[i] = string(ch) // Pre-allocate single char strings
	}

	// Initialize single character token mappings
	singleCharTokens['@'] = AT
	singleCharTokens[':'] = COLON
	singleCharTokens['='] = EQUALS
	singleCharTokens[','] = COMMA
	singleCharTokens['('] = LPAREN
	singleCharTokens[')'] = RPAREN
	singleCharTokens['{'] = LBRACE
	singleCharTokens['}'] = RBRACE
	singleCharTokens['*'] = ASTERISK
}

// Object pools for memory optimization
var (
	// Pool for token slices with different capacity tiers
	smallSlicePool = sync.Pool{
		New: func() interface{} {
			slice := make([]Token, 0, 16)
			return &slice
		},
	}
	mediumSlicePool = sync.Pool{
		New: func() interface{} {
			slice := make([]Token, 0, 64)
			return &slice
		},
	}
	largeSlicePool = sync.Pool{
		New: func() interface{} {
			slice := make([]Token, 0, 256)
			return &slice
		},
	}
)

// getTokenSlice returns a token slice from the appropriate pool
func getTokenSlice(estimatedSize int) *[]Token {
	if estimatedSize <= 16 {
		return smallSlicePool.Get().(*[]Token)
	} else if estimatedSize <= 64 {
		return mediumSlicePool.Get().(*[]Token)
	} else {
		return largeSlicePool.Get().(*[]Token)
	}
}

// putTokenSlice returns a token slice to the appropriate pool
func putTokenSlice(slice *[]Token) {
	*slice = (*slice)[:0] // Reset length but keep capacity

	cap := cap(*slice)
	if cap <= 16 {
		smallSlicePool.Put(slice)
	} else if cap <= 64 {
		mediumSlicePool.Put(slice)
	} else if cap <= 256 {
		largeSlicePool.Put(slice)
	}
	// For larger slices, let GC handle them
}

// LexerMode represents the current parsing context
type LexerMode int

const (
	// LanguageMode: Structural parsing of Devcmd syntax
	LanguageMode LexerMode = iota
	// CommandMode: Shell content capture
	CommandMode
	// PatternMode: Inside pattern-matching blocks (@when, @try, etc.)
	PatternMode
)

// String implements Stringer for LexerMode
func (lm LexerMode) String() string {
	switch lm {
	case LanguageMode:
		return "LanguageMode"
	case CommandMode:
		return "CommandMode"
	case PatternMode:
		return "PatternMode"
	default:
		return fmt.Sprintf("LexerMode(%d)", int(lm))
	}
}

// Lexer tokenizes Devcmd source code with rune-based parsing
type Lexer struct {
	input        string        // Source text
	position     int           // Current position in input (byte offset)
	readPos      int           // Current reading position in input (byte offset)
	ch           rune          // Current rune under examination
	line         int           // Current line number
	column       int           // Current column number
	stateMachine *StateMachine // State machine for parsing context
	braceLevel   int           // Track brace nesting for command mode
	patternLevel int           // Track pattern-matching decorator nesting

	// Structural brace tracking - only track braces that are part of Devcmd structure
	structuralBraceStack []int // positions of structural { braces

	// Infinite loop detection
	lastPosition     int
	lastLine         int
	lastColumn       int
	stuckCounter     int
	maxStuckAttempts int // Set to 3-5 typically
}

// New creates a new lexer instance with state machine
func New(input string) *Lexer {
	l := &Lexer{
		input:                input,
		line:                 1,
		column:               0, // Start at column 0, will be incremented to 1 on first readChar
		stateMachine:         NewStateMachine(),
		braceLevel:           0,
		patternLevel:         0,
		structuralBraceStack: make([]int, 0, 8), // Pre-allocate small capacity
		maxStuckAttempts:     3,                 // Allow 3 attempts before panicking
		lastPosition:         -1,                // Start with -1 so first token is always progress
	}
	l.readChar()
	return l
}

// NewWithDebug creates a new lexer instance with debugging enabled
func NewWithDebug(input string) *Lexer {
	l := New(input)
	l.stateMachine.SetDebug(true)
	return l
}

// checkStuck detects infinite loops in lexing
func (l *Lexer) checkStuck(context string) {
	if l.position == l.lastPosition &&
		l.line == l.lastLine &&
		l.column == l.lastColumn {
		l.stuckCounter++
		if l.stuckCounter >= l.maxStuckAttempts {
			// Fatal log with detailed context
			panic(fmt.Sprintf(
				"LEXER STUCK: Infinite loop detected in %s\n"+
					"Position: %d, Line: %d, Column: %d\n"+
					"Current char: %q (U+%04X)\n"+
					"State: %s, Mode: %s\n"+
					"Brace level: %d, Pattern level: %d\n"+
					"Input around position: %s\n"+
					"Context stack: %+v",
				context,
				l.position, l.line, l.column,
				l.ch, l.ch,
				l.stateMachine.Current().String(),
				l.stateMachine.GetMode().String(),
				l.braceLevel, l.patternLevel,
				l.getContextWindow(),
				l.stateMachine.contextStack,
			))
		}
	} else {
		// Reset counter when we make progress
		l.stuckCounter = 0
		l.lastPosition = l.position
		l.lastLine = l.line
		l.lastColumn = l.column
	}
}

// getContextWindow returns input context around current position
func (l *Lexer) getContextWindow() string {
	start := l.position - 20
	if start < 0 {
		start = 0
	}
	end := l.position + 20
	if end > len(l.input) {
		end = len(l.input)
	}

	context := l.input[start:end]
	result := make([]rune, 0, len(context)+1)

	// Mark current position with »
	runePos := 0
	for _, r := range context {
		if start+runePos == l.position {
			result = append(result, '»')
		}
		result = append(result, r)
		runePos += utf8.RuneLen(r)
	}

	return string(result)
}

// TokenizeToSlice tokenizes to pre-allocated slice with memory optimization
func (l *Lexer) TokenizeToSlice() []Token {
	// Better estimation based on input characteristics
	estimatedTokens := l.estimateTokenCount()

	// Get a pooled slice
	resultPtr := getTokenSlice(estimatedTokens)
	result := *resultPtr

	// Iteration limit as safeguard
	maxTokens := len(l.input) * 2
	tokenCount := 0

	for {
		tok := l.NextToken()

		// Smart doubling if we exceed capacity
		if len(result) == cap(result) {
			newCap := cap(result) * 2
			if newCap > maxTokens {
				newCap = maxTokens
			}
			newResult := make([]Token, len(result), newCap)
			copy(newResult, result)

			// Return old slice to pool if it's a pooled size
			putTokenSlice(resultPtr)
			result = newResult
			resultPtr = &result
		}

		result = append(result, tok)
		tokenCount++

		if tok.Type == EOF {
			break
		}

		if tokenCount > maxTokens {
			putTokenSlice(resultPtr)
			panic(fmt.Sprintf("Too many tokens: %d (input size: %d)", tokenCount, len(l.input)))
		}
	}

	// Create final result and return slice to pool
	finalResult := make([]Token, len(result))
	copy(finalResult, result)
	putTokenSlice(resultPtr)

	return finalResult
}

// estimateTokenCount provides better token count estimation
func (l *Lexer) estimateTokenCount() int {
	inputLen := len(l.input)
	if inputLen == 0 {
		return 4
	}

	// Count structural characters that likely generate tokens
	structuralChars := 0
	for i := 0; i < inputLen; i++ {
		ch := l.input[i]
		if ch == ':' || ch == '{' || ch == '}' || ch == '@' || ch == '(' || ch == ')' {
			structuralChars++
		}
	}

	// Base estimate on structural chars + some factor for identifiers/shell text
	estimated := structuralChars + (inputLen / 20) // Assume avg 20 chars per token

	if estimated < 4 {
		estimated = 4
	}
	if estimated > 500 {
		estimated = 500
	}

	return estimated
}

// NextToken returns the next token from the input
func (l *Lexer) NextToken() Token {
	l.checkStuck("NextToken")
	return l.lexToken()
}

// lexToken performs token lexing with state machine-aware logic
func (l *Lexer) lexToken() Token {
	// Skip whitespace in most modes
	mode := l.stateMachine.GetMode()
	if mode == LanguageMode || mode == PatternMode {
		l.skipWhitespace()
	}

	start := l.position

	// Check if we should enter shell content mode based on current state
	currentState := l.stateMachine.Current()
	if l.shouldLexShellContent(currentState) {
		return l.lexShellText(start)
	}

	switch mode {
	case LanguageMode:
		return l.lexLanguageMode(start)
	case CommandMode:
		return l.lexCommandMode(start)
	case PatternMode:
		return l.lexPatternMode(start)
	default:
		return l.lexLanguageMode(start)
	}
}

// shouldLexShellContent determines if we should lex shell content based on state
func (l *Lexer) shouldLexShellContent(state LexerState) bool {
	// Don't lex shell content if we're at structural tokens
	switch l.ch {
	case 0, '\n', '{', '}':
		return false
	case ':':
		// Colon is structural in pattern mode
		if l.stateMachine.GetMode() == PatternMode {
			return false
		}
		return false
	case '*':
		// Asterisk is structural in pattern mode
		if l.stateMachine.GetMode() == PatternMode {
			return false
		}
		// In other modes, continue to check state
	case '@':
		// Special handling for @ - check if it's followed by a function decorator
		if l.stateMachine.GetMode() == CommandMode || state == StateAfterColon || state == StateAfterPatternColon {
			// Look ahead to see if this is a function decorator
			if l.isFunctionDecorator() {
				return true // Treat as shell content
			}
		}
		return false
	}

	// Lex shell content in these states when we're not at structural boundaries
	switch state {
	case StateAfterColon:
		// After colon, if we see content that isn't a block decorator or brace, it's shell content
		return l.ch != '{'
	case StateAfterPatternColon:
		// After pattern colon, if we see content that isn't a block decorator or brace, it's shell content
		return l.ch != '{'
	case StateCommandContent:
		// In command content, everything except structural tokens is shell content
		return true
	case StateAfterDecorator:
		// After decorator, if we see content that isn't a brace, it might be shell content
		return l.ch != '{'
	case StatePatternBlock:
		// In pattern block, don't lex shell content - parse pattern structure
		return false
	default:
		return false
	}
}

// isFunctionDecorator looks ahead to determine if @ is followed by a function decorator
func (l *Lexer) isFunctionDecorator() bool {
	if l.ch != '@' {
		return false
	}

	// Save current state
	savePos := l.position
	saveReadPos := l.readPos
	saveCh := l.ch
	saveLine := l.line
	saveColumn := l.column

	// Move past @
	l.readChar()

	// Skip any whitespace (though there shouldn't be any)
	for unicode.IsSpace(l.ch) && l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}

	// Read identifier if present
	var decoratorName string
	if unicode.IsLetter(l.ch) || l.ch == '_' {
		identStart := l.position
		for (unicode.IsLetter(l.ch) || unicode.IsDigit(l.ch) || l.ch == '_' || l.ch == '-') && l.ch != 0 {
			l.readChar()
		}
		decoratorName = l.input[identStart:l.position]
	}

	// Restore state
	l.position = savePos
	l.readPos = saveReadPos
	l.ch = saveCh
	l.line = saveLine
	l.column = saveColumn

	// Check if it's a function decorator
	return decoratorName != "" && stdlib.IsFunctionDecorator(decoratorName)
}

// lexLanguageMode handles structural Devcmd syntax
func (l *Lexer) lexLanguageMode(start int) Token {
	startLine, startColumn := l.line, l.column

	// Fast path for ASCII single-character tokens (but skip context-sensitive ones)
	if l.ch < 128 && l.ch != 0 && l.ch != '\n' && l.ch != '{' && l.ch != '}' {
		if tokenType := singleCharTokens[l.ch]; tokenType != ILLEGAL {
			value := singleCharStrings[l.ch] // Use pre-allocated string
			var tok Token
			if tokenType == AT {
				tok = l.createTokenWithSemantic(AT, SemOperator, value, start, startLine, startColumn)
			} else {
				tok = l.createSimpleToken(tokenType, value, start, startLine, startColumn)
			}
			l.readChar()
			l.updateTokenEnd(&tok)
			l.updateStateMachine(tokenType, value)
			return tok
		}
	}

	switch l.ch {
	case 0:
		tok := l.createSimpleToken(EOF, "", start, startLine, startColumn)
		l.updateStateMachine(EOF, "")
		return tok
	case '\n':
		// Skip newlines - they're not needed in the parser/AST
		l.readChar()
		l.skipWhitespace()
		return l.lexToken() // Get the next meaningful token
	case '@':
		tok := l.createTokenWithSemantic(AT, SemOperator, "@", start, startLine, startColumn)
		l.readChar()
		l.updateTokenEnd(&tok)
		l.updateStateMachine(AT, "@")
		return tok
	case ':':
		tok := l.createSimpleToken(COLON, ":", start, startLine, startColumn)
		l.readChar()
		l.updateTokenEnd(&tok)
		l.updateStateMachine(COLON, ":")
		return tok
	case '=':
		tok := l.createSimpleToken(EQUALS, "=", start, startLine, startColumn)
		l.readChar()
		l.updateTokenEnd(&tok)
		l.updateStateMachine(EQUALS, "=")
		return tok
	case ',':
		tok := l.createSimpleToken(COMMA, ",", start, startLine, startColumn)
		l.readChar()
		l.updateTokenEnd(&tok)
		l.updateStateMachine(COMMA, ",")
		return tok
	case '(':
		tok := l.createSimpleToken(LPAREN, "(", start, startLine, startColumn)
		l.readChar()
		l.updateTokenEnd(&tok)
		l.updateStateMachine(LPAREN, "(")
		return tok
	case ')':
		tok := l.createSimpleToken(RPAREN, ")", start, startLine, startColumn)
		l.readChar()
		l.updateTokenEnd(&tok)
		l.updateStateMachine(RPAREN, ")")
		return tok
	case '{':
		tok := l.createSimpleToken(LBRACE, "{", start, startLine, startColumn)
		l.braceLevel++

		// Track if this is a structural brace
		if l.isStructuralContext() {
			l.pushStructuralBrace()
		}

		l.readChar()
		l.skipWhitespace() // Skip whitespace after opening brace
		l.updateTokenEnd(&tok)
		l.updateStateMachine(LBRACE, "{")
		return tok
	case '}':
		tok := l.createSimpleToken(RBRACE, "}", start, startLine, startColumn)
		if l.braceLevel > 0 {
			l.braceLevel--
		}

		// Pop structural brace if this closes one
		if l.hasStructuralBraces() {
			l.popStructuralBrace()
		}

		l.readChar()
		l.updateTokenEnd(&tok)
		l.updateStateMachine(RBRACE, "}")
		return tok
	case '*':
		// Always treat * as ASTERISK token for wildcard patterns
		tok := l.createSimpleToken(ASTERISK, "*", start, startLine, startColumn)
		l.readChar()
		l.updateTokenEnd(&tok)
		l.updateStateMachine(ASTERISK, "*")
		return tok
	case '"':
		return l.lexString('"', DoubleQuoted, start)
	case '\'':
		return l.lexString('\'', SingleQuoted, start)
	case '`':
		return l.lexString('`', Backtick, start)
	case '#':
		return l.lexComment(start)
	case '/':
		if l.peekChar() == '*' {
			return l.lexMultilineComment(start)
		}
		fallthrough
	case '\\':
		if l.peekChar() == '\n' {
			// Line continuation in language mode - treat as single char
			return l.lexSingleChar(start)
		}
		fallthrough
	default:
		if unicode.IsLetter(l.ch) || l.ch == '_' {
			return l.lexIdentifierOrKeyword(start)
		} else if unicode.IsDigit(l.ch) || (l.ch == '-' && unicode.IsDigit(l.peekChar())) {
			return l.lexNumberOrDuration(start)
		} else {
			return l.lexSingleChar(start)
		}
	}
}

// lexPatternMode handles pattern-matching decorator blocks (@when, @try, etc.)
func (l *Lexer) lexPatternMode(start int) Token {
	startLine, startColumn := l.line, l.column

	switch l.ch {
	case 0:
		tok := l.createSimpleToken(EOF, "", start, startLine, startColumn)
		l.updateStateMachine(EOF, "")
		return tok
	case '\n':
		// Skip newlines - they're not needed in the parser/AST
		l.readChar()
		l.skipWhitespace()
		return l.lexToken() // Get the next meaningful token
	case ':':
		tok := l.createSimpleToken(COLON, ":", start, startLine, startColumn)
		l.readChar()
		l.updateTokenEnd(&tok)
		l.updateStateMachine(COLON, ":")
		return tok
	case '}':
		tok := l.createSimpleToken(RBRACE, "}", start, startLine, startColumn)
		if l.braceLevel > 0 {
			l.braceLevel--
		}

		// Pop structural brace if this closes one
		if l.hasStructuralBraces() {
			l.popStructuralBrace()
		}

		l.readChar()
		l.updateTokenEnd(&tok)
		l.updateStateMachine(RBRACE, "}")
		return tok
	case '{':
		tok := l.createSimpleToken(LBRACE, "{", start, startLine, startColumn)
		l.braceLevel++

		// Track if this is a structural brace
		if l.isStructuralContext() {
			l.pushStructuralBrace()
		}

		l.readChar()
		l.skipWhitespace()
		l.updateTokenEnd(&tok)
		l.updateStateMachine(LBRACE, "{")
		return tok
	case '@':
		tok := l.createTokenWithSemantic(AT, SemOperator, "@", start, startLine, startColumn)
		l.readChar()
		l.updateTokenEnd(&tok)
		l.updateStateMachine(AT, "@")
		return tok
	case '*':
		// Always treat * as ASTERISK token for wildcard patterns
		tok := l.createSimpleToken(ASTERISK, "*", start, startLine, startColumn)
		l.readChar()
		l.updateTokenEnd(&tok)
		l.updateStateMachine(ASTERISK, "*")
		return tok
	case '(':
		tok := l.createSimpleToken(LPAREN, "(", start, startLine, startColumn)
		l.readChar()
		l.updateTokenEnd(&tok)
		l.updateStateMachine(LPAREN, "(")
		return tok
	case ')':
		tok := l.createSimpleToken(RPAREN, ")", start, startLine, startColumn)
		l.readChar()
		l.updateTokenEnd(&tok)
		l.updateStateMachine(RPAREN, ")")
		return tok
	case '"':
		return l.lexString('"', DoubleQuoted, start)
	case '\'':
		return l.lexString('\'', SingleQuoted, start)
	case '`':
		return l.lexString('`', Backtick, start)
	default:
		// In pattern mode, identifiers should be treated as pattern identifiers
		if unicode.IsLetter(l.ch) || l.ch == '_' {
			return l.lexPatternIdentifier(start)
		} else if unicode.IsDigit(l.ch) || (l.ch == '-' && unicode.IsDigit(l.peekChar())) {
			return l.lexNumberOrDuration(start)
		} else {
			return l.lexSingleChar(start)
		}
	}
}

// lexPatternIdentifier lexes identifiers in pattern mode
func (l *Lexer) lexPatternIdentifier(start int) Token {
	startLine, startColumn := l.line, l.column

	// Read the full identifier
	l.readIdentifier()

	value := l.input[start:l.position]

	tok := Token{
		Type:      IDENTIFIER,
		Value:     value,
		Line:      startLine,
		Column:    startColumn,
		EndLine:   l.line,
		EndColumn: l.column,
		Semantic:  SemPattern, // Mark as pattern semantic in pattern mode
		Span: SourceSpan{
			Start: SourcePosition{Line: startLine, Column: startColumn, Offset: start},
			End:   SourcePosition{Line: l.line, Column: l.column, Offset: l.position},
		},
	}
	l.updateStateMachine(IDENTIFIER, value)
	return tok
}

// lexCommandMode handles shell content capture with proper newline handling
func (l *Lexer) lexCommandMode(start int) Token {
	startLine, startColumn := l.line, l.column

	switch l.ch {
	case 0:
		tok := l.createSimpleToken(EOF, "", start, startLine, startColumn)
		l.updateStateMachine(EOF, "")
		return tok
	case '\n':
		// Skip newlines - they're not needed in the parser/AST
		l.readChar()
		l.skipWhitespace()
		return l.lexToken() // Get the next meaningful token
	case '}':
		// Only recognize } as structural if it closes a structural Devcmd brace
		if l.hasStructuralBraces() {
			tok := l.createSimpleToken(RBRACE, "}", start, startLine, startColumn)
			l.braceLevel--
			l.popStructuralBrace()
			l.readChar()
			l.updateTokenEnd(&tok)
			l.updateStateMachine(RBRACE, "}")
			return tok
		}
		// Otherwise, treat as shell content
		return l.lexShellText(start)
	case '@':
		// Check if this is a function decorator
		if l.isFunctionDecorator() {
			// Function decorator should be part of shell text
			return l.lexShellText(start)
		}

		// It's a block/pattern decorator, treat it as structural.
		tok := l.createTokenWithSemantic(AT, SemOperator, "@", start, startLine, startColumn)
		l.readChar()
		l.updateTokenEnd(&tok)
		l.updateStateMachine(AT, "@")
		return tok
	case '{':
		// Handle opening brace in command mode
		tok := l.createSimpleToken(LBRACE, "{", start, startLine, startColumn)
		l.braceLevel++

		// Track if this is a structural brace
		if l.isStructuralContext() {
			l.pushStructuralBrace()
		}

		l.readChar()
		l.skipWhitespace()
		l.updateTokenEnd(&tok)
		l.updateStateMachine(LBRACE, "{")
		return tok
	default:
		// All other content is handled as shell text
		return l.lexShellText(start)
	}
}

// updateStateMachine notifies the state machine about the current token
func (l *Lexer) updateStateMachine(tokenType TokenType, value string) {
	if _, err := l.stateMachine.HandleToken(tokenType, value); err != nil {
		// In production, you might want to handle this error differently
		// For debugging, log state machine errors
		if l.stateMachine.debug {
			println("State machine error:", err.Error(), "- token:", tokenType.String(), "value:", value)
		}
	}
}

// lexShellText captures shell content as a single token
// It handles POSIX quoting rules and line continuations structurally
func (l *Lexer) lexShellText(start int) Token {
	startLine, startColumn := l.line, l.column
	startOffset := start

	var inSingleQuotes, inDoubleQuotes, inBackticks bool
	var prevWasBackslash bool

	// Track shell-level brace nesting separate from structural braces
	// This handles things like find -exec cmd {} +
	shellBraceLevel := 0

	for {
		switch l.ch {
		case 0:
			// EOF - return what we have
			tok := l.makeShellToken(start, startOffset, startLine, startColumn)
			l.updateStateMachine(SHELL_TEXT, tok.Value)
			return tok

		case '\n':
			// Handle line continuation outside quotes
			if !inSingleQuotes && !inDoubleQuotes && !inBackticks && prevWasBackslash {
				prevWasBackslash = false
				l.readChar()
				// Skip following whitespace per GNU make behavior
				for l.ch == ' ' || l.ch == '\t' {
					l.readChar()
				}
				continue
			}

			// Newlines inside quotes are part of shell text
			if inSingleQuotes || inDoubleQuotes || inBackticks {
				prevWasBackslash = false
				l.readChar()
				continue
			}

			// Otherwise, newline ends shell text
			tok := l.makeShellToken(start, startOffset, startLine, startColumn)
			l.updateStateMachine(SHELL_TEXT, tok.Value)
			// Also handle the newline for state transitions (but don't generate NEWLINE token)
			if _, err := l.stateMachine.handleNewline(); err != nil {
				// Log error but continue
				if l.stateMachine.debug {
					println("Newline state transition error:", err.Error())
				}
			}
			return tok

		case '\'':
			if !inDoubleQuotes && !inBackticks {
				inSingleQuotes = !inSingleQuotes
			}
			prevWasBackslash = false
			l.readChar()

		case '"':
			if !inSingleQuotes && !inBackticks {
				inDoubleQuotes = !inDoubleQuotes
			}
			prevWasBackslash = false
			l.readChar()

		case '`':
			if !inSingleQuotes && !inDoubleQuotes {
				inBackticks = !inBackticks
			}
			prevWasBackslash = false
			l.readChar()

		case '\\':
			if inSingleQuotes {
				// In single quotes, backslash is literal
				prevWasBackslash = false
				l.readChar()
			} else {
				// Mark potential line continuation
				prevWasBackslash = true
				l.readChar()
				// In double quotes or backticks, consume escaped character
				if (inDoubleQuotes || inBackticks) && l.ch != 0 {
					prevWasBackslash = false // Not a line continuation
					l.readChar()
				}
			}

		case '{':
			// Track shell-level braces (like in find -exec cmd {} +)
			if !inSingleQuotes && !inDoubleQuotes && !inBackticks {
				shellBraceLevel++
			}
			prevWasBackslash = false
			l.readChar()

		case '}':
			// Handle shell-level and structural braces
			if !inSingleQuotes && !inDoubleQuotes && !inBackticks {
				if shellBraceLevel > 0 {
					// This closes a shell-level brace, continue in shell text
					shellBraceLevel--
					prevWasBackslash = false
					l.readChar()
				} else if len(l.structuralBraceStack) > 0 {
					// This could close a structural brace, exit shell text
					tok := l.makeShellToken(start, startOffset, startLine, startColumn)
					l.updateStateMachine(SHELL_TEXT, tok.Value)
					return tok
				} else {
					// No braces to close, continue as shell text
					prevWasBackslash = false
					l.readChar()
				}
			} else {
				prevWasBackslash = false
				l.readChar()
			}

		default:
			// Any other character resets line continuation and continues as shell content
			// This includes semicolons and the '@' symbol.
			if l.ch != ' ' && l.ch != '\t' {
				prevWasBackslash = false
			}
			l.readChar()
		}
	}
}

// makeShellToken creates a shell text token from the captured range
func (l *Lexer) makeShellToken(start, startOffset, startLine, startColumn int) Token {
	// Get the raw text
	rawText := l.input[start:l.position]

	// Process line continuations
	processedText := l.processLineContinuations(rawText)

	// Trim whitespace
	processedText = strings.TrimSpace(processedText)

	// Don't emit empty tokens - but ensure we've actually consumed something
	if processedText == "" {
		// If we haven't moved forward, we need to consume at least one character
		// to avoid infinite loops
		if l.position == start && l.ch != 0 {
			l.readChar()
		}
		return l.lexToken()
	}

	return Token{
		Type:      SHELL_TEXT,
		Value:     processedText,
		Line:      startLine,
		Column:    startColumn,
		EndLine:   l.line,
		EndColumn: l.column,
		Raw:       rawText, // Keep original for formatting tools
		Semantic:  SemShellText,
		Span: SourceSpan{
			Start: SourcePosition{Line: startLine, Column: startColumn, Offset: startOffset},
			End:   SourcePosition{Line: l.line, Column: l.column, Offset: l.position},
		},
	}
}

// processLineContinuations handles backslash-newline sequences in shell text
func (l *Lexer) processLineContinuations(text string) string {
	// Fast path: no backslashes means no continuations
	if !strings.Contains(text, "\\") {
		return text
	}

	var result strings.Builder
	result.Grow(len(text))

	runes := []rune(text)
	i := 0
	inSingleQuotes := false

	for i < len(runes) {
		ch := runes[i]

		// Track single quote state
		if ch == '\'' {
			inSingleQuotes = !inSingleQuotes
			result.WriteRune(ch)
			i++
			continue
		}

		// In single quotes, everything is literal
		if inSingleQuotes {
			result.WriteRune(ch)
			i++
			continue
		}

		// Check for line continuation outside single quotes
		if ch == '\\' && i+1 < len(runes) && runes[i+1] == '\n' {
			// Skip the backslash and newline
			i += 2

			// Skip following whitespace
			for i < len(runes) && (runes[i] == ' ' || runes[i] == '\t') {
				i++
			}

			// Add a space to join the lines
			if result.Len() > 0 && i < len(runes) {
				str := result.String()
				lastCh := rune(str[len(str)-1])
				if lastCh != ' ' && lastCh != '\t' {
					result.WriteRune(' ')
				}
			}
		} else {
			result.WriteRune(ch)
			i++
		}
	}

	return result.String()
}

// lexIdentifierOrKeyword lexes identifiers and keywords with optimized lookahead
func (l *Lexer) lexIdentifierOrKeyword(start int) Token {
	startLine, startColumn := l.line, l.column

	// Read the full identifier
	l.readIdentifier()

	value := l.input[start:l.position]

	var tokenType TokenType
	var semantic SemanticTokenType

	// Check for boolean literals first
	if value == "true" || value == "false" {
		tok := Token{
			Type:      BOOLEAN,
			Value:     value,
			Line:      startLine,
			Column:    startColumn,
			EndLine:   l.line,
			EndColumn: l.column,
			Semantic:  SemBoolean,
			Span: SourceSpan{
				Start: SourcePosition{Line: startLine, Column: startColumn, Offset: start},
				End:   SourcePosition{Line: l.line, Column: l.column, Offset: l.position},
			},
		}
		l.updateStateMachine(BOOLEAN, value)
		return tok
	}

	// Check for keywords
	if keywordType, isKeyword := keywords[value]; isKeyword {
		tokenType = keywordType
		semantic = SemKeyword
		// Special handling for pattern-matching decorators
		if value == "when" || value == "try" {
			// Track that we're entering a pattern-matching decorator
			l.patternLevel++
		}
	} else {
		tokenType = IDENTIFIER
		semantic = SemCommand // Default to command name
	}

	tok := Token{
		Type:      tokenType,
		Value:     value,
		Line:      startLine,
		Column:    startColumn,
		EndLine:   l.line,
		EndColumn: l.column,
		Semantic:  semantic,
		Span: SourceSpan{
			Start: SourcePosition{Line: startLine, Column: startColumn, Offset: start},
			End:   SourcePosition{Line: l.line, Column: l.column, Offset: l.position},
		},
	}
	l.updateStateMachine(tokenType, value)
	return tok
}

// Keywords map - includes pattern-matching decorator keywords
var keywords = map[string]TokenType{
	"var":   VAR,
	"stop":  STOP,
	"watch": WATCH,
	"when":  WHEN,
	"try":   TRY,
}

// lexNumberOrDuration lexes numbers and durations with rune-based scanning
func (l *Lexer) lexNumberOrDuration(start int) Token {
	startLine, startColumn := l.line, l.column

	// Handle negative numbers
	if l.ch == '-' {
		l.readChar()
	}

	// Scan integer part
	for unicode.IsDigit(l.ch) {
		l.readChar()
	}

	// Check for decimal part
	if l.ch == '.' && unicode.IsDigit(l.peekChar()) {
		l.readChar() // consume '.'
		for unicode.IsDigit(l.ch) {
			l.readChar()
		}
	}

	// Check for duration unit
	isDuration := false
	if l.isDurationUnit() {
		isDuration = true
		l.readDurationUnit()
	}

	value := l.input[start:l.position]

	tokenType := NUMBER
	if isDuration {
		tokenType = DURATION
	}

	tok := Token{
		Type:      tokenType,
		Value:     value,
		Line:      startLine,
		Column:    startColumn,
		EndLine:   l.line,
		EndColumn: l.column,
		Semantic:  SemNumber,
		Span: SourceSpan{
			Start: SourcePosition{Line: startLine, Column: startColumn, Offset: start},
			End:   SourcePosition{Line: l.line, Column: l.column, Offset: l.position},
		},
	}
	l.updateStateMachine(tokenType, value)
	return tok
}

// lexString lexes string literals with rune-based scanning
func (l *Lexer) lexString(quote rune, stringType StringType, start int) Token {
	startLine, startColumn := l.line, l.column
	l.readChar() // consume opening quote

	var value string
	var hasEscapes bool

	// For single-quoted strings, just find the next quote
	if stringType == SingleQuoted {
		valueStart := l.position
		for l.ch != quote && l.ch != 0 {
			l.readChar()
		}
		value = l.input[valueStart:l.position]
	} else {
		// For double-quoted and backtick strings, handle escapes
		var escaped strings.Builder
		valueStart := l.position

		for l.ch != quote && l.ch != 0 {
			if l.ch == '\\' {
				if !hasEscapes {
					hasEscapes = true
					escaped.WriteString(l.input[valueStart:l.position])
				}
				l.readChar()
				if l.ch == 0 {
					break
				}
				escapeStr := l.handleEscape(stringType)
				escaped.WriteString(escapeStr)
				l.readChar()
				valueStart = l.position
			} else {
				l.readChar()
			}
		}

		if hasEscapes {
			escaped.WriteString(l.input[valueStart:l.position])
			value = escaped.String()
		} else {
			value = l.input[valueStart:l.position]
		}
	}

	if l.ch == quote {
		l.readChar() // consume closing quote
	}

	tok := Token{
		Type:       STRING,
		Value:      value,
		Line:       startLine,
		Column:     startColumn,
		EndLine:    l.line,
		EndColumn:  l.column,
		StringType: stringType,
		Raw:        l.input[start:l.position],
		Semantic:   SemString,
		Span: SourceSpan{
			Start: SourcePosition{Line: startLine, Column: startColumn, Offset: start},
			End:   SourcePosition{Line: l.line, Column: l.column, Offset: l.position},
		},
	}
	l.updateStateMachine(STRING, value)
	return tok
}

// lexComment lexes single-line comments
func (l *Lexer) lexComment(start int) Token {
	startLine, startColumn := l.line, l.column
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
	tok := Token{
		Type:      COMMENT,
		Value:     l.input[start:l.position],
		Line:      startLine,
		Column:    startColumn,
		EndLine:   l.line,
		EndColumn: l.column,
		Semantic:  SemComment,
		Span: SourceSpan{
			Start: SourcePosition{Line: startLine, Column: startColumn, Offset: start},
			End:   SourcePosition{Line: l.line, Column: l.column, Offset: l.position},
		},
	}
	l.updateStateMachine(COMMENT, tok.Value)
	return tok
}

// lexMultilineComment lexes multi-line comments
func (l *Lexer) lexMultilineComment(start int) Token {
	startLine, startColumn := l.line, l.column
	l.readChar() // consume '/'
	l.readChar() // consume '*'

	for l.ch != 0 {
		if l.ch == '*' && l.peekChar() == '/' {
			l.readChar()
			l.readChar()
			break
		}
		l.readChar()
	}

	tok := Token{
		Type:      MULTILINE_COMMENT,
		Value:     l.input[start:l.position],
		Line:      startLine,
		Column:    startColumn,
		EndLine:   l.line,
		EndColumn: l.column,
		Semantic:  SemComment,
		Span: SourceSpan{
			Start: SourcePosition{Line: startLine, Column: startColumn, Offset: start},
			End:   SourcePosition{Line: l.line, Column: l.column, Offset: l.position},
		},
	}
	l.updateStateMachine(MULTILINE_COMMENT, tok.Value)
	return tok
}

// lexSingleChar lexes single character tokens
func (l *Lexer) lexSingleChar(start int) Token {
	startLine, startColumn := l.line, l.column
	char := l.ch
	l.readChar()

	token := Token{
		Type:      IDENTIFIER,
		Value:     string(char),
		Line:      startLine,
		Column:    startColumn,
		EndLine:   l.line,
		EndColumn: l.column,
		Semantic:  SemOperator,
		Span: SourceSpan{
			Start: SourcePosition{Line: startLine, Column: startColumn, Offset: start},
			End:   SourcePosition{Line: l.line, Column: l.column, Offset: l.position},
		},
	}
	l.updateStateMachine(IDENTIFIER, token.Value)
	return token
}

// Helper methods for creating tokens with proper position tracking

// createSimpleToken creates a token with basic type and value
func (l *Lexer) createSimpleToken(tokenType TokenType, value string, start, startLine, startColumn int) Token {
	return Token{
		Type:      tokenType,
		Value:     value,
		Line:      startLine,
		Column:    startColumn,
		EndLine:   startLine,   // Will be updated by updateTokenEnd
		EndColumn: startColumn, // Will be updated by updateTokenEnd
		Span: SourceSpan{
			Start: SourcePosition{Line: startLine, Column: startColumn, Offset: start},
			End:   SourcePosition{Line: startLine, Column: startColumn, Offset: start}, // Will be updated
		},
	}
}

// createTokenWithSemantic creates a token with specific semantic type
func (l *Lexer) createTokenWithSemantic(tokenType TokenType, semantic SemanticTokenType, value string, start, startLine, startColumn int) Token {
	return Token{
		Type:      tokenType,
		Value:     value,
		Line:      startLine,
		Column:    startColumn,
		EndLine:   startLine,   // Will be updated by updateTokenEnd
		EndColumn: startColumn, // Will be updated by updateTokenEnd
		Semantic:  semantic,
		Span: SourceSpan{
			Start: SourcePosition{Line: startLine, Column: startColumn, Offset: start},
			End:   SourcePosition{Line: startLine, Column: startColumn, Offset: start}, // Will be updated
		},
	}
}

// updateTokenEnd updates the end position of a token
func (l *Lexer) updateTokenEnd(token *Token) {
	token.EndLine = l.line
	token.EndColumn = l.column
	token.Span.End = SourcePosition{Line: l.line, Column: l.column, Offset: l.position}
}

// Helper methods

func (l *Lexer) readIdentifier() {
	for l.ch != 0 {
		// ASCII fast path using lookup table
		if l.ch < 128 {
			if !isIdentPart[l.ch] {
				break
			}
		} else {
			// Unicode fallback for non-ASCII characters
			if !unicode.IsLetter(l.ch) && !unicode.IsDigit(l.ch) {
				break
			}
		}
		l.readChar()
	}
}

// skipWhitespace with hybrid ASCII/Unicode scanning
func (l *Lexer) skipWhitespace() {
	for l.ch != '\n' && l.ch != 0 {
		// ASCII fast path using lookup table
		if l.ch < 128 {
			if !isWhitespace[l.ch] {
				break
			}
		} else {
			// Unicode fallback for non-ASCII characters
			if !unicode.IsSpace(l.ch) {
				break
			}
		}
		l.readChar()
	}
}

func (l *Lexer) readChar() {
	if l.readPos >= len(l.input) {
		l.ch = 0
		l.position = l.readPos
	} else {
		// ASCII fast path - most config files are primarily ASCII
		if b := l.input[l.readPos]; b < 0x80 {
			l.ch = rune(b)
			l.position = l.readPos
			l.readPos++
		} else {
			// Unicode slow path
			r, size := utf8.DecodeRuneInString(l.input[l.readPos:])
			l.ch = r
			l.position = l.readPos
			l.readPos += size
		}
	}

	// Position tracking: increment column before handling newline
	l.column++
	if l.ch == '\n' {
		l.line++
		l.column = 0 // Reset to 0, will be incremented to 1 on next readChar
	}
}

func (l *Lexer) peekChar() rune {
	if l.readPos >= len(l.input) {
		return 0
	}
	// ASCII fast path
	if b := l.input[l.readPos]; b < 0x80 {
		return rune(b)
	}
	// Unicode slow path
	r, _ := utf8.DecodeRuneInString(l.input[l.readPos:])
	return r
}

func (l *Lexer) isDurationUnit() bool {
	if l.ch == 0 {
		return false
	}
	next := l.peekChar()
	switch l.ch {
	case 'n', 'u':
		return next == 's'
	case 'm':
		return next == 's' || next == 0 || !unicode.IsLetter(next)
	case 's', 'h':
		return next == 0 || !unicode.IsLetter(next)
	}
	return false
}

func (l *Lexer) readDurationUnit() {
	switch l.ch {
	case 'n', 'u':
		if l.peekChar() == 's' {
			l.readChar()
			l.readChar()
		}
	case 'm':
		l.readChar()
		if l.ch == 's' {
			l.readChar()
		}
	case 's', 'h':
		l.readChar()
	}
}

func (l *Lexer) handleEscape(stringType StringType) string {
	switch stringType {
	case SingleQuoted:
		// In single-quoted strings, a backslash is a literal character.
		return "\\" + string(l.ch)
	case DoubleQuoted:
		switch l.ch {
		case 'n':
			return "\n"
		case 't':
			return "\t"
		case 'r':
			return "\r"
		case '\\':
			return "\\"
		case '"':
			return "\""
		default:
			return "\\" + string(l.ch)
		}
	case Backtick:
		switch l.ch {
		case 'n':
			return "\n"
		case 't':
			return "\t"
		case 'r':
			return "\r"
		case 'b':
			return "\b"
		case 'f':
			return "\f"
		case 'v':
			return "\v"
		case '0':
			return "\x00"
		case '\\':
			return "\\"
		case '`':
			return "`"
		case '"':
			return "\""
		case '\'':
			return "'"
		case 'x':
			return l.readHexEscape()
		case 'u':
			if l.peekChar() == '{' {
				return l.readUnicodeEscape()
			}
			return "\\u"
		default:
			return "\\" + string(l.ch)
		}
	}
	return "\\" + string(l.ch)
}

func (l *Lexer) readHexEscape() string {
	if !isHexDigit(l.peekChar()) {
		return "\\x"
	}
	l.readChar()
	hex1 := l.ch
	l.readChar()
	if !isHexDigit(l.ch) {
		return "\\x" + string(hex1)
	}
	hex2 := l.ch
	value := hexValue(hex1)*16 + hexValue(hex2)
	return string(rune(value))
}

func (l *Lexer) readUnicodeEscape() string {
	l.readChar()
	l.readChar()
	start := l.position
	for l.ch != '}' && l.ch != 0 && isHexDigit(l.ch) {
		l.readChar()
	}
	if l.ch != '}' {
		return "\\u{"
	}
	hexDigits := l.input[start:l.position]
	l.readChar()
	if len(hexDigits) == 0 {
		return "\\u{}"
	}
	var value rune
	for _, ch := range hexDigits {
		value = value*16 + rune(hexValue(ch))
	}
	if !utf8.ValidRune(value) {
		return "\\u{" + hexDigits + "}"
	}
	return string(value)
}

func isHexDigit(ch rune) bool {
	return unicode.IsDigit(ch) || ('a' <= ch && ch <= 'f') || ('A' <= ch && ch <= 'F')
}

func hexValue(ch rune) int {
	switch {
	case unicode.IsDigit(ch):
		return int(ch - '0')
	case 'a' <= ch && ch <= 'f':
		return int(ch - 'a' + 10)
	case 'A' <= ch && ch <= 'F':
		return int(ch - 'A' + 10)
	}
	return 0
}

// pushStructuralBrace adds a structural brace to the stack
func (l *Lexer) pushStructuralBrace() {
	l.structuralBraceStack = append(l.structuralBraceStack, l.position)
}

// popStructuralBrace removes the most recent structural brace from the stack
func (l *Lexer) popStructuralBrace() {
	if len(l.structuralBraceStack) > 0 {
		l.structuralBraceStack = l.structuralBraceStack[:len(l.structuralBraceStack)-1]
	}
}

// hasStructuralBraces returns true if there are structural braces on the stack
func (l *Lexer) hasStructuralBraces() bool {
	return len(l.structuralBraceStack) > 0
}

// isStructuralContext determines if the current context expects a structural brace
func (l *Lexer) isStructuralContext() bool {
	currentState := l.stateMachine.Current()
	switch currentState {
	case StateAfterColon:
		// After command: or pattern:, a { is structural
		return true
	case StateAfterDecorator:
		// After @parallel, @timeout, etc., a { is structural
		return true
	case StateAfterPatternColon:
		// After pattern:, a { is structural
		return true
	default:
		return false
	}
}
