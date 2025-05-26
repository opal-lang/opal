// Code generated from devcmd.g4 by ANTLR 4.13.2. DO NOT EDIT.

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

type devcmdLexer struct {
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
		"", "'def'", "'='", "':'", "'watch'", "'stop'", "", "", "'\\;'", "'\\$'",
		"", "'{'", "'}'", "';'", "'&'", "'\\'",
	}
	staticData.SymbolicNames = []string{
		"", "DEF", "EQUALS", "COLON", "WATCH", "STOP", "OUR_VARIABLE_REFERENCE",
		"SHELL_VARIABLE_REFERENCE", "ESCAPED_SEMICOLON", "ESCAPED_DOLLAR", "ESCAPED_CHAR",
		"LBRACE", "RBRACE", "SEMICOLON", "AMPERSAND", "BACKSLASH", "NAME", "NUMBER",
		"COMMAND_TEXT", "COMMENT", "NEWLINE", "WS",
	}
	staticData.RuleNames = []string{
		"DEF", "EQUALS", "COLON", "WATCH", "STOP", "OUR_VARIABLE_REFERENCE",
		"SHELL_VARIABLE_REFERENCE", "ESCAPED_SEMICOLON", "ESCAPED_DOLLAR", "ESCAPED_CHAR",
		"LBRACE", "RBRACE", "SEMICOLON", "AMPERSAND", "BACKSLASH", "NAME", "NUMBER",
		"COMMAND_TEXT", "COMMENT", "NEWLINE", "WS",
	}
	staticData.PredictionContextCache = antlr.NewPredictionContextCache()
	staticData.serializedATN = []int32{
		4, 0, 21, 156, 6, -1, 2, 0, 7, 0, 2, 1, 7, 1, 2, 2, 7, 2, 2, 3, 7, 3, 2,
		4, 7, 4, 2, 5, 7, 5, 2, 6, 7, 6, 2, 7, 7, 7, 2, 8, 7, 8, 2, 9, 7, 9, 2,
		10, 7, 10, 2, 11, 7, 11, 2, 12, 7, 12, 2, 13, 7, 13, 2, 14, 7, 14, 2, 15,
		7, 15, 2, 16, 7, 16, 2, 17, 7, 17, 2, 18, 7, 18, 2, 19, 7, 19, 2, 20, 7,
		20, 1, 0, 1, 0, 1, 0, 1, 0, 1, 1, 1, 1, 1, 2, 1, 2, 1, 3, 1, 3, 1, 3, 1,
		3, 1, 3, 1, 3, 1, 4, 1, 4, 1, 4, 1, 4, 1, 4, 1, 5, 1, 5, 1, 5, 1, 5, 1,
		5, 1, 5, 1, 6, 1, 6, 1, 6, 5, 6, 72, 8, 6, 10, 6, 12, 6, 75, 9, 6, 1, 7,
		1, 7, 1, 7, 1, 8, 1, 8, 1, 8, 1, 9, 1, 9, 1, 9, 1, 9, 1, 9, 1, 9, 1, 9,
		1, 9, 1, 9, 1, 9, 3, 9, 93, 8, 9, 1, 10, 1, 10, 1, 11, 1, 11, 1, 12, 1,
		12, 1, 13, 1, 13, 1, 14, 1, 14, 1, 15, 1, 15, 5, 15, 107, 8, 15, 10, 15,
		12, 15, 110, 9, 15, 1, 16, 5, 16, 113, 8, 16, 10, 16, 12, 16, 116, 9, 16,
		1, 16, 1, 16, 4, 16, 120, 8, 16, 11, 16, 12, 16, 121, 1, 16, 4, 16, 125,
		8, 16, 11, 16, 12, 16, 126, 3, 16, 129, 8, 16, 1, 17, 4, 17, 132, 8, 17,
		11, 17, 12, 17, 133, 1, 18, 1, 18, 5, 18, 138, 8, 18, 10, 18, 12, 18, 141,
		9, 18, 1, 18, 1, 18, 1, 19, 3, 19, 146, 8, 19, 1, 19, 1, 19, 1, 20, 4,
		20, 151, 8, 20, 11, 20, 12, 20, 152, 1, 20, 1, 20, 0, 0, 21, 1, 1, 3, 2,
		5, 3, 7, 4, 9, 5, 11, 6, 13, 7, 15, 8, 17, 9, 19, 10, 21, 11, 23, 12, 25,
		13, 27, 14, 29, 15, 31, 16, 33, 17, 35, 18, 37, 19, 39, 20, 41, 21, 1,
		0, 9, 2, 0, 65, 90, 97, 122, 4, 0, 48, 57, 65, 90, 95, 95, 97, 122, 9,
		0, 34, 34, 36, 36, 40, 41, 92, 92, 110, 110, 114, 114, 116, 116, 123, 123,
		125, 125, 3, 0, 48, 57, 65, 70, 97, 102, 5, 0, 45, 45, 48, 57, 65, 90,
		95, 95, 97, 122, 1, 0, 48, 57, 6, 0, 9, 10, 13, 13, 32, 32, 58, 59, 61,
		61, 92, 92, 2, 0, 10, 10, 13, 13, 2, 0, 9, 9, 32, 32, 167, 0, 1, 1, 0,
		0, 0, 0, 3, 1, 0, 0, 0, 0, 5, 1, 0, 0, 0, 0, 7, 1, 0, 0, 0, 0, 9, 1, 0,
		0, 0, 0, 11, 1, 0, 0, 0, 0, 13, 1, 0, 0, 0, 0, 15, 1, 0, 0, 0, 0, 17, 1,
		0, 0, 0, 0, 19, 1, 0, 0, 0, 0, 21, 1, 0, 0, 0, 0, 23, 1, 0, 0, 0, 0, 25,
		1, 0, 0, 0, 0, 27, 1, 0, 0, 0, 0, 29, 1, 0, 0, 0, 0, 31, 1, 0, 0, 0, 0,
		33, 1, 0, 0, 0, 0, 35, 1, 0, 0, 0, 0, 37, 1, 0, 0, 0, 0, 39, 1, 0, 0, 0,
		0, 41, 1, 0, 0, 0, 1, 43, 1, 0, 0, 0, 3, 47, 1, 0, 0, 0, 5, 49, 1, 0, 0,
		0, 7, 51, 1, 0, 0, 0, 9, 57, 1, 0, 0, 0, 11, 62, 1, 0, 0, 0, 13, 68, 1,
		0, 0, 0, 15, 76, 1, 0, 0, 0, 17, 79, 1, 0, 0, 0, 19, 82, 1, 0, 0, 0, 21,
		94, 1, 0, 0, 0, 23, 96, 1, 0, 0, 0, 25, 98, 1, 0, 0, 0, 27, 100, 1, 0,
		0, 0, 29, 102, 1, 0, 0, 0, 31, 104, 1, 0, 0, 0, 33, 128, 1, 0, 0, 0, 35,
		131, 1, 0, 0, 0, 37, 135, 1, 0, 0, 0, 39, 145, 1, 0, 0, 0, 41, 150, 1,
		0, 0, 0, 43, 44, 5, 100, 0, 0, 44, 45, 5, 101, 0, 0, 45, 46, 5, 102, 0,
		0, 46, 2, 1, 0, 0, 0, 47, 48, 5, 61, 0, 0, 48, 4, 1, 0, 0, 0, 49, 50, 5,
		58, 0, 0, 50, 6, 1, 0, 0, 0, 51, 52, 5, 119, 0, 0, 52, 53, 5, 97, 0, 0,
		53, 54, 5, 116, 0, 0, 54, 55, 5, 99, 0, 0, 55, 56, 5, 104, 0, 0, 56, 8,
		1, 0, 0, 0, 57, 58, 5, 115, 0, 0, 58, 59, 5, 116, 0, 0, 59, 60, 5, 111,
		0, 0, 60, 61, 5, 112, 0, 0, 61, 10, 1, 0, 0, 0, 62, 63, 5, 36, 0, 0, 63,
		64, 5, 40, 0, 0, 64, 65, 1, 0, 0, 0, 65, 66, 3, 31, 15, 0, 66, 67, 5, 41,
		0, 0, 67, 12, 1, 0, 0, 0, 68, 69, 5, 36, 0, 0, 69, 73, 7, 0, 0, 0, 70,
		72, 7, 1, 0, 0, 71, 70, 1, 0, 0, 0, 72, 75, 1, 0, 0, 0, 73, 71, 1, 0, 0,
		0, 73, 74, 1, 0, 0, 0, 74, 14, 1, 0, 0, 0, 75, 73, 1, 0, 0, 0, 76, 77,
		5, 92, 0, 0, 77, 78, 5, 59, 0, 0, 78, 16, 1, 0, 0, 0, 79, 80, 5, 92, 0,
		0, 80, 81, 5, 36, 0, 0, 81, 18, 1, 0, 0, 0, 82, 92, 5, 92, 0, 0, 83, 93,
		7, 2, 0, 0, 84, 85, 5, 120, 0, 0, 85, 86, 7, 3, 0, 0, 86, 93, 7, 3, 0,
		0, 87, 88, 5, 117, 0, 0, 88, 89, 7, 3, 0, 0, 89, 90, 7, 3, 0, 0, 90, 91,
		7, 3, 0, 0, 91, 93, 7, 3, 0, 0, 92, 83, 1, 0, 0, 0, 92, 84, 1, 0, 0, 0,
		92, 87, 1, 0, 0, 0, 93, 20, 1, 0, 0, 0, 94, 95, 5, 123, 0, 0, 95, 22, 1,
		0, 0, 0, 96, 97, 5, 125, 0, 0, 97, 24, 1, 0, 0, 0, 98, 99, 5, 59, 0, 0,
		99, 26, 1, 0, 0, 0, 100, 101, 5, 38, 0, 0, 101, 28, 1, 0, 0, 0, 102, 103,
		5, 92, 0, 0, 103, 30, 1, 0, 0, 0, 104, 108, 7, 0, 0, 0, 105, 107, 7, 4,
		0, 0, 106, 105, 1, 0, 0, 0, 107, 110, 1, 0, 0, 0, 108, 106, 1, 0, 0, 0,
		108, 109, 1, 0, 0, 0, 109, 32, 1, 0, 0, 0, 110, 108, 1, 0, 0, 0, 111, 113,
		7, 5, 0, 0, 112, 111, 1, 0, 0, 0, 113, 116, 1, 0, 0, 0, 114, 112, 1, 0,
		0, 0, 114, 115, 1, 0, 0, 0, 115, 117, 1, 0, 0, 0, 116, 114, 1, 0, 0, 0,
		117, 119, 5, 46, 0, 0, 118, 120, 7, 5, 0, 0, 119, 118, 1, 0, 0, 0, 120,
		121, 1, 0, 0, 0, 121, 119, 1, 0, 0, 0, 121, 122, 1, 0, 0, 0, 122, 129,
		1, 0, 0, 0, 123, 125, 7, 5, 0, 0, 124, 123, 1, 0, 0, 0, 125, 126, 1, 0,
		0, 0, 126, 124, 1, 0, 0, 0, 126, 127, 1, 0, 0, 0, 127, 129, 1, 0, 0, 0,
		128, 114, 1, 0, 0, 0, 128, 124, 1, 0, 0, 0, 129, 34, 1, 0, 0, 0, 130, 132,
		8, 6, 0, 0, 131, 130, 1, 0, 0, 0, 132, 133, 1, 0, 0, 0, 133, 131, 1, 0,
		0, 0, 133, 134, 1, 0, 0, 0, 134, 36, 1, 0, 0, 0, 135, 139, 5, 35, 0, 0,
		136, 138, 8, 7, 0, 0, 137, 136, 1, 0, 0, 0, 138, 141, 1, 0, 0, 0, 139,
		137, 1, 0, 0, 0, 139, 140, 1, 0, 0, 0, 140, 142, 1, 0, 0, 0, 141, 139,
		1, 0, 0, 0, 142, 143, 6, 18, 0, 0, 143, 38, 1, 0, 0, 0, 144, 146, 5, 13,
		0, 0, 145, 144, 1, 0, 0, 0, 145, 146, 1, 0, 0, 0, 146, 147, 1, 0, 0, 0,
		147, 148, 5, 10, 0, 0, 148, 40, 1, 0, 0, 0, 149, 151, 7, 8, 0, 0, 150,
		149, 1, 0, 0, 0, 151, 152, 1, 0, 0, 0, 152, 150, 1, 0, 0, 0, 152, 153,
		1, 0, 0, 0, 153, 154, 1, 0, 0, 0, 154, 155, 6, 20, 0, 0, 155, 42, 1, 0,
		0, 0, 12, 0, 73, 92, 108, 114, 121, 126, 128, 133, 139, 145, 152, 1, 0,
		1, 0,
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

// devcmdLexerInit initializes any static state used to implement devcmdLexer. By default the
// static state used to implement the lexer is lazily initialized during the first call to
// NewdevcmdLexer(). You can call this function if you wish to initialize the static state ahead
// of time.
func DevcmdLexerInit() {
	staticData := &DevcmdLexerLexerStaticData
	staticData.once.Do(devcmdlexerLexerInit)
}

// NewdevcmdLexer produces a new lexer instance for the optional input antlr.CharStream.
func NewdevcmdLexer(input antlr.CharStream) *devcmdLexer {
	DevcmdLexerInit()
	l := new(devcmdLexer)
	l.BaseLexer = antlr.NewBaseLexer(input)
	staticData := &DevcmdLexerLexerStaticData
	l.Interpreter = antlr.NewLexerATNSimulator(l, staticData.atn, staticData.decisionToDFA, staticData.PredictionContextCache)
	l.channelNames = staticData.ChannelNames
	l.modeNames = staticData.ModeNames
	l.RuleNames = staticData.RuleNames
	l.LiteralNames = staticData.LiteralNames
	l.SymbolicNames = staticData.SymbolicNames
	l.GrammarFileName = "devcmd.g4"
	// TODO: l.EOF = antlr.TokenEOF

	return l
}

// devcmdLexer tokens.
const (
	devcmdLexerDEF                      = 1
	devcmdLexerEQUALS                   = 2
	devcmdLexerCOLON                    = 3
	devcmdLexerWATCH                    = 4
	devcmdLexerSTOP                     = 5
	devcmdLexerOUR_VARIABLE_REFERENCE   = 6
	devcmdLexerSHELL_VARIABLE_REFERENCE = 7
	devcmdLexerESCAPED_SEMICOLON        = 8
	devcmdLexerESCAPED_DOLLAR           = 9
	devcmdLexerESCAPED_CHAR             = 10
	devcmdLexerLBRACE                   = 11
	devcmdLexerRBRACE                   = 12
	devcmdLexerSEMICOLON                = 13
	devcmdLexerAMPERSAND                = 14
	devcmdLexerBACKSLASH                = 15
	devcmdLexerNAME                     = 16
	devcmdLexerNUMBER                   = 17
	devcmdLexerCOMMAND_TEXT             = 18
	devcmdLexerCOMMENT                  = 19
	devcmdLexerNEWLINE                  = 20
	devcmdLexerWS                       = 21
)
