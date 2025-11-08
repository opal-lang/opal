package lexer

// ASCII character lookup tables for fast classification (zero-allocation)
//
// Performance: Use inline bounds-checked lookups for maximum speed:
//
//	if ch < 128 && isLetter[ch] { ... }  // Fastest approach
//
// For Unicode characters (ch >= 128), use unicode package functions.
//
// Benchmarks show:
//   - Inline bounds check: 9.17 ns/op (fastest)
//   - Function calls: 11.00 ns/op (20% slower)
//   - Direct access: 9.82 ns/op (7% slower, unsafe)
var (
	isWhitespace  [128]bool // Space, tab, carriage return, newline
	isLetter      [128]bool // a-z, A-Z, _
	isDigit       [128]bool // 0-9
	isIdentStart  [128]bool // Letter or _
	isIdentPart   [128]bool // Letter, digit, _ or -
	isHexDigit    [128]bool // 0-9, a-f, A-F
	isPunctuation [128]bool // Single character punctuation
)

func init() {
	// Pre-compute ASCII character classification tables
	for i := 0; i < 128; i++ {
		ch := byte(i)

		// Whitespace (excluding newline - newlines are meaningful tokens)
		isWhitespace[i] = ch == ' ' || ch == '\t' || ch == '\r' || ch == '\f'

		// Letters (ASCII + underscore)
		isLetter[i] = ('a' <= ch && ch <= 'z') || ('A' <= ch && ch <= 'Z') || ch == '_'

		// Digits
		isDigit[i] = '0' <= ch && ch <= '9'

		// Identifier characters (no hyphens per specification)
		isIdentStart[i] = isLetter[i]
		isIdentPart[i] = isLetter[i] || isDigit[i]

		// Hex digits
		isHexDigit[i] = isDigit[i] || ('a' <= ch && ch <= 'f') || ('A' <= ch && ch <= 'F')

		// Single character punctuation/operators
		isPunctuation[i] = (ch >= '!' && ch <= '/') || (ch >= ':' && ch <= '@') ||
			(ch >= '[' && ch <= '`') || (ch >= '{' && ch <= '~')
	}
}

// Identifier specification: ASCII-only for maximum compatibility
//
// Identifiers: [a-zA-Z_][a-zA-Z0-9_]*
// - Must start with letter or underscore
// - Can contain letters, digits, underscore (no hyphens per spec)
// - No case requirements (user choice)
//
// Unicode handling: Only for position tracking and string content
// - Position tracking: Use utf8.DecodeRune for proper advancement
// - String content: Preserve as raw bytes in tokens

// isValidASCIIIdentifier checks if a string is a valid ASCII identifier
func isValidASCIIIdentifier(s string) bool {
	if s == "" {
		return false
	}

	// First character must be letter or underscore
	first := s[0]
	if first >= 128 || !isIdentStart[first] {
		return false
	}

	// Remaining characters must be letters, digits, underscore, or hyphen
	for i := 1; i < len(s); i++ {
		ch := s[i]
		if ch >= 128 || !isIdentPart[ch] {
			return false
		}
	}

	return true
}
