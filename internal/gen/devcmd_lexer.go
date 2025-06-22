// Code generated from DevcmdLexer.g4 by ANTLR 4.13.2. DO NOT EDIT.

package gen

import (
	"fmt"
	"github.com/antlr4-go/antlr/v4"
	"sync"
	"unicode"
)

// Suppress unused import error
var _ = fmt.Printf
var _ = sync.Once{}
var _ = unicode.IsLetter

type DevcmdLexer struct {
	*antlr.BaseLexer
	channelNames []string
	modeNames    []string
	// TODO: EOF string
}

var DevcmdLexerLexerStaticData struct {
	once                   sync.Once
	serializedATN          []int32
	ChannelNames           []string
	ModeNames              []string
	LiteralNames           []string
	SymbolicNames          []string
	RuleNames              []string
	PredictionContextCache *antlr.PredictionContextCache
	atn                    *antlr.ATN
	decisionToDFA          []*antlr.DFA
}

func devcmdlexerLexerInit() {
	staticData := &DevcmdLexerLexerStaticData
	staticData.ChannelNames = []string{
		"DEFAULT_TOKEN_CHANNEL", "HIDDEN",
	}
	staticData.ModeNames = []string{
		"DEFAULT_MODE",
	}
	staticData.LiteralNames = []string{
		"", "'def'", "'watch'", "'stop'", "'@'", "'='", "':'", "';'", "'{'",
		"'}'", "'('", "')'", "'\\'", "'&'", "'|'", "'<'", "'>'", "", "", "",
		"", "", "'.'", "','", "'/'", "'-'", "'*'", "'+'", "'?'", "'!'", "'%'",
		"'^'", "'~'", "'_'", "'['", "']'", "'$'", "'#'", "'\"'",
	}
	staticData.SymbolicNames = []string{
		"", "DEF", "WATCH", "STOP", "AT", "EQUALS", "COLON", "SEMICOLON", "LBRACE",
		"RBRACE", "LPAREN", "RPAREN", "BACKSLASH", "AMPERSAND", "PIPE", "LT",
		"GT", "STRING", "SINGLE_STRING", "NAME", "NUMBER", "PATH_CONTENT", "DOT",
		"COMMA", "SLASH", "DASH", "STAR", "PLUS", "QUESTION", "EXCLAIM", "PERCENT",
		"CARET", "TILDE", "UNDERSCORE", "LBRACKET", "RBRACKET", "DOLLAR", "HASH",
		"DOUBLEQUOTE", "CONTENT", "COMMENT", "NEWLINE", "WS",
	}
	staticData.RuleNames = []string{
		"DEF", "WATCH", "STOP", "AT", "EQUALS", "COLON", "SEMICOLON", "LBRACE",
		"RBRACE", "LPAREN", "RPAREN", "BACKSLASH", "AMPERSAND", "PIPE", "LT",
		"GT", "STRING", "SINGLE_STRING", "NAME", "NUMBER", "PATH_CONTENT", "DOT",
		"COMMA", "SLASH", "DASH", "STAR", "PLUS", "QUESTION", "EXCLAIM", "PERCENT",
		"CARET", "TILDE", "UNDERSCORE", "LBRACKET", "RBRACKET", "DOLLAR", "HASH",
		"DOUBLEQUOTE", "CONTENT", "COMMENT", "NEWLINE", "WS",
	}
	staticData.PredictionContextCache = antlr.NewPredictionContextCache()
	staticData.serializedATN = []int32{
		4, 0, 42, 235, 6, -1, 2, 0, 7, 0, 2, 1, 7, 1, 2, 2, 7, 2, 2, 3, 7, 3, 2,
		4, 7, 4, 2, 5, 7, 5, 2, 6, 7, 6, 2, 7, 7, 7, 2, 8, 7, 8, 2, 9, 7, 9, 2,
		10, 7, 10, 2, 11, 7, 11, 2, 12, 7, 12, 2, 13, 7, 13, 2, 14, 7, 14, 2, 15,
		7, 15, 2, 16, 7, 16, 2, 17, 7, 17, 2, 18, 7, 18, 2, 19, 7, 19, 2, 20, 7,
		20, 2, 21, 7, 21, 2, 22, 7, 22, 2, 23, 7, 23, 2, 24, 7, 24, 2, 25, 7, 25,
		2, 26, 7, 26, 2, 27, 7, 27, 2, 28, 7, 28, 2, 29, 7, 29, 2, 30, 7, 30, 2,
		31, 7, 31, 2, 32, 7, 32, 2, 33, 7, 33, 2, 34, 7, 34, 2, 35, 7, 35, 2, 36,
		7, 36, 2, 37, 7, 37, 2, 38, 7, 38, 2, 39, 7, 39, 2, 40, 7, 40, 2, 41, 7,
		41, 1, 0, 1, 0, 1, 0, 1, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 2, 1,
		2, 1, 2, 1, 2, 1, 2, 1, 3, 1, 3, 1, 4, 1, 4, 1, 5, 1, 5, 1, 6, 1, 6, 1,
		7, 1, 7, 1, 8, 1, 8, 1, 9, 1, 9, 1, 10, 1, 10, 1, 11, 1, 11, 1, 12, 1,
		12, 1, 13, 1, 13, 1, 14, 1, 14, 1, 15, 1, 15, 1, 16, 1, 16, 1, 16, 1, 16,
		5, 16, 131, 8, 16, 10, 16, 12, 16, 134, 9, 16, 1, 16, 1, 16, 1, 17, 1,
		17, 1, 17, 1, 17, 5, 17, 142, 8, 17, 10, 17, 12, 17, 145, 9, 17, 1, 17,
		1, 17, 1, 18, 1, 18, 5, 18, 151, 8, 18, 10, 18, 12, 18, 154, 9, 18, 1,
		19, 3, 19, 157, 8, 19, 1, 19, 4, 19, 160, 8, 19, 11, 19, 12, 19, 161, 1,
		19, 1, 19, 4, 19, 166, 8, 19, 11, 19, 12, 19, 167, 3, 19, 170, 8, 19, 1,
		20, 1, 20, 5, 20, 174, 8, 20, 10, 20, 12, 20, 177, 9, 20, 1, 21, 1, 21,
		1, 22, 1, 22, 1, 23, 1, 23, 1, 24, 1, 24, 1, 25, 1, 25, 1, 26, 1, 26, 1,
		27, 1, 27, 1, 28, 1, 28, 1, 29, 1, 29, 1, 30, 1, 30, 1, 31, 1, 31, 1, 32,
		1, 32, 1, 33, 1, 33, 1, 34, 1, 34, 1, 35, 1, 35, 1, 36, 1, 36, 1, 37, 1,
		37, 1, 38, 1, 38, 1, 39, 1, 39, 5, 39, 217, 8, 39, 10, 39, 12, 39, 220,
		9, 39, 1, 39, 1, 39, 1, 40, 3, 40, 225, 8, 40, 1, 40, 1, 40, 1, 41, 4,
		41, 230, 8, 41, 11, 41, 12, 41, 231, 1, 41, 1, 41, 0, 0, 42, 1, 1, 3, 2,
		5, 3, 7, 4, 9, 5, 11, 6, 13, 7, 15, 8, 17, 9, 19, 10, 21, 11, 23, 12, 25,
		13, 27, 14, 29, 15, 31, 16, 33, 17, 35, 18, 37, 19, 39, 20, 41, 21, 43,
		22, 45, 23, 47, 24, 49, 25, 51, 26, 53, 27, 55, 28, 57, 29, 59, 30, 61,
		31, 63, 32, 65, 33, 67, 34, 69, 35, 71, 36, 73, 37, 75, 38, 77, 39, 79,
		40, 81, 41, 83, 42, 1, 0, 10, 4, 0, 10, 10, 13, 13, 34, 34, 92, 92, 4,
		0, 10, 10, 13, 13, 39, 39, 92, 92, 2, 0, 65, 90, 97, 122, 5, 0, 45, 45,
		48, 57, 65, 90, 95, 95, 97, 122, 1, 0, 48, 57, 2, 0, 46, 47, 126, 126,
		5, 0, 42, 42, 45, 57, 65, 90, 95, 95, 97, 122, 9, 0, 9, 10, 13, 13, 32,
		32, 34, 35, 39, 41, 58, 59, 64, 64, 123, 123, 125, 125, 2, 0, 10, 10, 13,
		13, 2, 0, 9, 9, 32, 32, 247, 0, 1, 1, 0, 0, 0, 0, 3, 1, 0, 0, 0, 0, 5,
		1, 0, 0, 0, 0, 7, 1, 0, 0, 0, 0, 9, 1, 0, 0, 0, 0, 11, 1, 0, 0, 0, 0, 13,
		1, 0, 0, 0, 0, 15, 1, 0, 0, 0, 0, 17, 1, 0, 0, 0, 0, 19, 1, 0, 0, 0, 0,
		21, 1, 0, 0, 0, 0, 23, 1, 0, 0, 0, 0, 25, 1, 0, 0, 0, 0, 27, 1, 0, 0, 0,
		0, 29, 1, 0, 0, 0, 0, 31, 1, 0, 0, 0, 0, 33, 1, 0, 0, 0, 0, 35, 1, 0, 0,
		0, 0, 37, 1, 0, 0, 0, 0, 39, 1, 0, 0, 0, 0, 41, 1, 0, 0, 0, 0, 43, 1, 0,
		0, 0, 0, 45, 1, 0, 0, 0, 0, 47, 1, 0, 0, 0, 0, 49, 1, 0, 0, 0, 0, 51, 1,
		0, 0, 0, 0, 53, 1, 0, 0, 0, 0, 55, 1, 0, 0, 0, 0, 57, 1, 0, 0, 0, 0, 59,
		1, 0, 0, 0, 0, 61, 1, 0, 0, 0, 0, 63, 1, 0, 0, 0, 0, 65, 1, 0, 0, 0, 0,
		67, 1, 0, 0, 0, 0, 69, 1, 0, 0, 0, 0, 71, 1, 0, 0, 0, 0, 73, 1, 0, 0, 0,
		0, 75, 1, 0, 0, 0, 0, 77, 1, 0, 0, 0, 0, 79, 1, 0, 0, 0, 0, 81, 1, 0, 0,
		0, 0, 83, 1, 0, 0, 0, 1, 85, 1, 0, 0, 0, 3, 89, 1, 0, 0, 0, 5, 95, 1, 0,
		0, 0, 7, 100, 1, 0, 0, 0, 9, 102, 1, 0, 0, 0, 11, 104, 1, 0, 0, 0, 13,
		106, 1, 0, 0, 0, 15, 108, 1, 0, 0, 0, 17, 110, 1, 0, 0, 0, 19, 112, 1,
		0, 0, 0, 21, 114, 1, 0, 0, 0, 23, 116, 1, 0, 0, 0, 25, 118, 1, 0, 0, 0,
		27, 120, 1, 0, 0, 0, 29, 122, 1, 0, 0, 0, 31, 124, 1, 0, 0, 0, 33, 126,
		1, 0, 0, 0, 35, 137, 1, 0, 0, 0, 37, 148, 1, 0, 0, 0, 39, 156, 1, 0, 0,
		0, 41, 171, 1, 0, 0, 0, 43, 178, 1, 0, 0, 0, 45, 180, 1, 0, 0, 0, 47, 182,
		1, 0, 0, 0, 49, 184, 1, 0, 0, 0, 51, 186, 1, 0, 0, 0, 53, 188, 1, 0, 0,
		0, 55, 190, 1, 0, 0, 0, 57, 192, 1, 0, 0, 0, 59, 194, 1, 0, 0, 0, 61, 196,
		1, 0, 0, 0, 63, 198, 1, 0, 0, 0, 65, 200, 1, 0, 0, 0, 67, 202, 1, 0, 0,
		0, 69, 204, 1, 0, 0, 0, 71, 206, 1, 0, 0, 0, 73, 208, 1, 0, 0, 0, 75, 210,
		1, 0, 0, 0, 77, 212, 1, 0, 0, 0, 79, 214, 1, 0, 0, 0, 81, 224, 1, 0, 0,
		0, 83, 229, 1, 0, 0, 0, 85, 86, 5, 100, 0, 0, 86, 87, 5, 101, 0, 0, 87,
		88, 5, 102, 0, 0, 88, 2, 1, 0, 0, 0, 89, 90, 5, 119, 0, 0, 90, 91, 5, 97,
		0, 0, 91, 92, 5, 116, 0, 0, 92, 93, 5, 99, 0, 0, 93, 94, 5, 104, 0, 0,
		94, 4, 1, 0, 0, 0, 95, 96, 5, 115, 0, 0, 96, 97, 5, 116, 0, 0, 97, 98,
		5, 111, 0, 0, 98, 99, 5, 112, 0, 0, 99, 6, 1, 0, 0, 0, 100, 101, 5, 64,
		0, 0, 101, 8, 1, 0, 0, 0, 102, 103, 5, 61, 0, 0, 103, 10, 1, 0, 0, 0, 104,
		105, 5, 58, 0, 0, 105, 12, 1, 0, 0, 0, 106, 107, 5, 59, 0, 0, 107, 14,
		1, 0, 0, 0, 108, 109, 5, 123, 0, 0, 109, 16, 1, 0, 0, 0, 110, 111, 5, 125,
		0, 0, 111, 18, 1, 0, 0, 0, 112, 113, 5, 40, 0, 0, 113, 20, 1, 0, 0, 0,
		114, 115, 5, 41, 0, 0, 115, 22, 1, 0, 0, 0, 116, 117, 5, 92, 0, 0, 117,
		24, 1, 0, 0, 0, 118, 119, 5, 38, 0, 0, 119, 26, 1, 0, 0, 0, 120, 121, 5,
		124, 0, 0, 121, 28, 1, 0, 0, 0, 122, 123, 5, 60, 0, 0, 123, 30, 1, 0, 0,
		0, 124, 125, 5, 62, 0, 0, 125, 32, 1, 0, 0, 0, 126, 132, 5, 34, 0, 0, 127,
		131, 8, 0, 0, 0, 128, 129, 5, 92, 0, 0, 129, 131, 9, 0, 0, 0, 130, 127,
		1, 0, 0, 0, 130, 128, 1, 0, 0, 0, 131, 134, 1, 0, 0, 0, 132, 130, 1, 0,
		0, 0, 132, 133, 1, 0, 0, 0, 133, 135, 1, 0, 0, 0, 134, 132, 1, 0, 0, 0,
		135, 136, 5, 34, 0, 0, 136, 34, 1, 0, 0, 0, 137, 143, 5, 39, 0, 0, 138,
		142, 8, 1, 0, 0, 139, 140, 5, 92, 0, 0, 140, 142, 9, 0, 0, 0, 141, 138,
		1, 0, 0, 0, 141, 139, 1, 0, 0, 0, 142, 145, 1, 0, 0, 0, 143, 141, 1, 0,
		0, 0, 143, 144, 1, 0, 0, 0, 144, 146, 1, 0, 0, 0, 145, 143, 1, 0, 0, 0,
		146, 147, 5, 39, 0, 0, 147, 36, 1, 0, 0, 0, 148, 152, 7, 2, 0, 0, 149,
		151, 7, 3, 0, 0, 150, 149, 1, 0, 0, 0, 151, 154, 1, 0, 0, 0, 152, 150,
		1, 0, 0, 0, 152, 153, 1, 0, 0, 0, 153, 38, 1, 0, 0, 0, 154, 152, 1, 0,
		0, 0, 155, 157, 5, 45, 0, 0, 156, 155, 1, 0, 0, 0, 156, 157, 1, 0, 0, 0,
		157, 159, 1, 0, 0, 0, 158, 160, 7, 4, 0, 0, 159, 158, 1, 0, 0, 0, 160,
		161, 1, 0, 0, 0, 161, 159, 1, 0, 0, 0, 161, 162, 1, 0, 0, 0, 162, 169,
		1, 0, 0, 0, 163, 165, 5, 46, 0, 0, 164, 166, 7, 4, 0, 0, 165, 164, 1, 0,
		0, 0, 166, 167, 1, 0, 0, 0, 167, 165, 1, 0, 0, 0, 167, 168, 1, 0, 0, 0,
		168, 170, 1, 0, 0, 0, 169, 163, 1, 0, 0, 0, 169, 170, 1, 0, 0, 0, 170,
		40, 1, 0, 0, 0, 171, 175, 7, 5, 0, 0, 172, 174, 7, 6, 0, 0, 173, 172, 1,
		0, 0, 0, 174, 177, 1, 0, 0, 0, 175, 173, 1, 0, 0, 0, 175, 176, 1, 0, 0,
		0, 176, 42, 1, 0, 0, 0, 177, 175, 1, 0, 0, 0, 178, 179, 5, 46, 0, 0, 179,
		44, 1, 0, 0, 0, 180, 181, 5, 44, 0, 0, 181, 46, 1, 0, 0, 0, 182, 183, 5,
		47, 0, 0, 183, 48, 1, 0, 0, 0, 184, 185, 5, 45, 0, 0, 185, 50, 1, 0, 0,
		0, 186, 187, 5, 42, 0, 0, 187, 52, 1, 0, 0, 0, 188, 189, 5, 43, 0, 0, 189,
		54, 1, 0, 0, 0, 190, 191, 5, 63, 0, 0, 191, 56, 1, 0, 0, 0, 192, 193, 5,
		33, 0, 0, 193, 58, 1, 0, 0, 0, 194, 195, 5, 37, 0, 0, 195, 60, 1, 0, 0,
		0, 196, 197, 5, 94, 0, 0, 197, 62, 1, 0, 0, 0, 198, 199, 5, 126, 0, 0,
		199, 64, 1, 0, 0, 0, 200, 201, 5, 95, 0, 0, 201, 66, 1, 0, 0, 0, 202, 203,
		5, 91, 0, 0, 203, 68, 1, 0, 0, 0, 204, 205, 5, 93, 0, 0, 205, 70, 1, 0,
		0, 0, 206, 207, 5, 36, 0, 0, 207, 72, 1, 0, 0, 0, 208, 209, 5, 35, 0, 0,
		209, 74, 1, 0, 0, 0, 210, 211, 5, 34, 0, 0, 211, 76, 1, 0, 0, 0, 212, 213,
		8, 7, 0, 0, 213, 78, 1, 0, 0, 0, 214, 218, 5, 35, 0, 0, 215, 217, 8, 8,
		0, 0, 216, 215, 1, 0, 0, 0, 217, 220, 1, 0, 0, 0, 218, 216, 1, 0, 0, 0,
		218, 219, 1, 0, 0, 0, 219, 221, 1, 0, 0, 0, 220, 218, 1, 0, 0, 0, 221,
		222, 6, 39, 0, 0, 222, 80, 1, 0, 0, 0, 223, 225, 5, 13, 0, 0, 224, 223,
		1, 0, 0, 0, 224, 225, 1, 0, 0, 0, 225, 226, 1, 0, 0, 0, 226, 227, 5, 10,
		0, 0, 227, 82, 1, 0, 0, 0, 228, 230, 7, 9, 0, 0, 229, 228, 1, 0, 0, 0,
		230, 231, 1, 0, 0, 0, 231, 229, 1, 0, 0, 0, 231, 232, 1, 0, 0, 0, 232,
		233, 1, 0, 0, 0, 233, 234, 6, 41, 0, 0, 234, 84, 1, 0, 0, 0, 14, 0, 130,
		132, 141, 143, 152, 156, 161, 167, 169, 175, 218, 224, 231, 1, 0, 1, 0,
	}
	deserializer := antlr.NewATNDeserializer(nil)
	staticData.atn = deserializer.Deserialize(staticData.serializedATN)
	atn := staticData.atn
	staticData.decisionToDFA = make([]*antlr.DFA, len(atn.DecisionToState))
	decisionToDFA := staticData.decisionToDFA
	for index, state := range atn.DecisionToState {
		decisionToDFA[index] = antlr.NewDFA(state, index)
	}
}

// DevcmdLexerInit initializes any static state used to implement DevcmdLexer. By default the
// static state used to implement the lexer is lazily initialized during the first call to
// NewDevcmdLexer(). You can call this function if you wish to initialize the static state ahead
// of time.
func DevcmdLexerInit() {
	staticData := &DevcmdLexerLexerStaticData
	staticData.once.Do(devcmdlexerLexerInit)
}

// NewDevcmdLexer produces a new lexer instance for the optional input antlr.CharStream.
func NewDevcmdLexer(input antlr.CharStream) *DevcmdLexer {
	DevcmdLexerInit()
	l := new(DevcmdLexer)
	l.BaseLexer = antlr.NewBaseLexer(input)
	staticData := &DevcmdLexerLexerStaticData
	l.Interpreter = antlr.NewLexerATNSimulator(l, staticData.atn, staticData.decisionToDFA, staticData.PredictionContextCache)
	l.channelNames = staticData.ChannelNames
	l.modeNames = staticData.ModeNames
	l.RuleNames = staticData.RuleNames
	l.LiteralNames = staticData.LiteralNames
	l.SymbolicNames = staticData.SymbolicNames
	l.GrammarFileName = "DevcmdLexer.g4"
	// TODO: l.EOF = antlr.TokenEOF

	return l
}

// DevcmdLexer tokens.
const (
	DevcmdLexerDEF           = 1
	DevcmdLexerWATCH         = 2
	DevcmdLexerSTOP          = 3
	DevcmdLexerAT            = 4
	DevcmdLexerEQUALS        = 5
	DevcmdLexerCOLON         = 6
	DevcmdLexerSEMICOLON     = 7
	DevcmdLexerLBRACE        = 8
	DevcmdLexerRBRACE        = 9
	DevcmdLexerLPAREN        = 10
	DevcmdLexerRPAREN        = 11
	DevcmdLexerBACKSLASH     = 12
	DevcmdLexerAMPERSAND     = 13
	DevcmdLexerPIPE          = 14
	DevcmdLexerLT            = 15
	DevcmdLexerGT            = 16
	DevcmdLexerSTRING        = 17
	DevcmdLexerSINGLE_STRING = 18
	DevcmdLexerNAME          = 19
	DevcmdLexerNUMBER        = 20
	DevcmdLexerPATH_CONTENT  = 21
	DevcmdLexerDOT           = 22
	DevcmdLexerCOMMA         = 23
	DevcmdLexerSLASH         = 24
	DevcmdLexerDASH          = 25
	DevcmdLexerSTAR          = 26
	DevcmdLexerPLUS          = 27
	DevcmdLexerQUESTION      = 28
	DevcmdLexerEXCLAIM       = 29
	DevcmdLexerPERCENT       = 30
	DevcmdLexerCARET         = 31
	DevcmdLexerTILDE         = 32
	DevcmdLexerUNDERSCORE    = 33
	DevcmdLexerLBRACKET      = 34
	DevcmdLexerRBRACKET      = 35
	DevcmdLexerDOLLAR        = 36
	DevcmdLexerHASH          = 37
	DevcmdLexerDOUBLEQUOTE   = 38
	DevcmdLexerCONTENT       = 39
	DevcmdLexerCOMMENT       = 40
	DevcmdLexerNEWLINE       = 41
	DevcmdLexerWS            = 42
)
