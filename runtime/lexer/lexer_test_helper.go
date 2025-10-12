package lexer

// newTestLexer is a test helper that creates and initializes a lexer with string input
func newTestLexer(input string, opts ...LexerOpt) *Lexer {
	lex := NewLexer(opts...)
	lex.Init([]byte(input))
	return lex
}
