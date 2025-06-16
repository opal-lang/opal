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
		"", "'def'", "'watch'", "'stop'", "", "'@'", "'='", "':'", "';'", "'{'",
		"'}'", "'('", "')'", "'\\'", "'&'", "'|'", "'<'", "'>'", "", "", "'\\$'",
		"'\\;'", "", "", "", "", "", "", "'.'", "','", "'/'", "'-'", "'*'",
		"'+'", "'?'", "'!'", "'%'", "'^'", "'~'", "'_'", "'['", "']'", "'$'",
		"'#'", "'\"'",
	}
	staticData.SymbolicNames = []string{
		"", "DEF", "WATCH", "STOP", "AT_NAME_LPAREN", "AT", "EQUALS", "COLON",
		"SEMICOLON", "LBRACE", "RBRACE", "LPAREN", "RPAREN", "BACKSLASH", "AMPERSAND",
		"PIPE", "LT", "GT", "VAR_REF", "SHELL_VAR", "ESCAPED_DOLLAR", "ESCAPED_SEMICOLON",
		"ESCAPED_BRACE", "STRING", "SINGLE_STRING", "NAME", "NUMBER", "PATH_CONTENT",
		"DOT", "COMMA", "SLASH", "DASH", "STAR", "PLUS", "QUESTION", "EXCLAIM",
		"PERCENT", "CARET", "TILDE", "UNDERSCORE", "LBRACKET", "RBRACKET", "DOLLAR",
		"HASH", "DOUBLEQUOTE", "COMMENT", "NEWLINE", "WS",
	}
	staticData.RuleNames = []string{
		"DEF", "WATCH", "STOP", "AT_NAME_LPAREN", "AT", "EQUALS", "COLON", "SEMICOLON",
		"LBRACE", "RBRACE", "LPAREN", "RPAREN", "BACKSLASH", "AMPERSAND", "PIPE",
		"LT", "GT", "VAR_REF", "SHELL_VAR", "ESCAPED_DOLLAR", "ESCAPED_SEMICOLON",
		"ESCAPED_BRACE", "STRING", "SINGLE_STRING", "NAME", "NUMBER", "PATH_CONTENT",
		"DOT", "COMMA", "SLASH", "DASH", "STAR", "PLUS", "QUESTION", "EXCLAIM",
		"PERCENT", "CARET", "TILDE", "UNDERSCORE", "LBRACKET", "RBRACKET", "DOLLAR",
		"HASH", "DOUBLEQUOTE", "COMMENT", "NEWLINE", "WS",
	}
	staticData.PredictionContextCache = antlr.NewPredictionContextCache()
	staticData.serializedATN = []int32{
		4, 0, 47, 285, 6, -1, 2, 0, 7, 0, 2, 1, 7, 1, 2, 2, 7, 2, 2, 3, 7, 3, 2,
		4, 7, 4, 2, 5, 7, 5, 2, 6, 7, 6, 2, 7, 7, 7, 2, 8, 7, 8, 2, 9, 7, 9, 2,
		10, 7, 10, 2, 11, 7, 11, 2, 12, 7, 12, 2, 13, 7, 13, 2, 14, 7, 14, 2, 15,
		7, 15, 2, 16, 7, 16, 2, 17, 7, 17, 2, 18, 7, 18, 2, 19, 7, 19, 2, 20, 7,
		20, 2, 21, 7, 21, 2, 22, 7, 22, 2, 23, 7, 23, 2, 24, 7, 24, 2, 25, 7, 25,
		2, 26, 7, 26, 2, 27, 7, 27, 2, 28, 7, 28, 2, 29, 7, 29, 2, 30, 7, 30, 2,
		31, 7, 31, 2, 32, 7, 32, 2, 33, 7, 33, 2, 34, 7, 34, 2, 35, 7, 35, 2, 36,
		7, 36, 2, 37, 7, 37, 2, 38, 7, 38, 2, 39, 7, 39, 2, 40, 7, 40, 2, 41, 7,
		41, 2, 42, 7, 42, 2, 43, 7, 43, 2, 44, 7, 44, 2, 45, 7, 45, 2, 46, 7, 46,
		1, 0, 1, 0, 1, 0, 1, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 2, 1, 2,
		1, 2, 1, 2, 1, 2, 1, 3, 1, 3, 1, 3, 5, 3, 114, 8, 3, 10, 3, 12, 3, 117,
		9, 3, 1, 3, 1, 3, 1, 4, 1, 4, 1, 5, 1, 5, 1, 6, 1, 6, 1, 7, 1, 7, 1, 8,
		1, 8, 1, 9, 1, 9, 1, 10, 1, 10, 1, 11, 1, 11, 1, 12, 1, 12, 1, 13, 1, 13,
		1, 14, 1, 14, 1, 15, 1, 15, 1, 16, 1, 16, 1, 17, 1, 17, 1, 17, 1, 17, 1,
		17, 5, 17, 152, 8, 17, 10, 17, 12, 17, 155, 9, 17, 1, 17, 1, 17, 1, 18,
		1, 18, 1, 18, 5, 18, 162, 8, 18, 10, 18, 12, 18, 165, 9, 18, 1, 19, 1,
		19, 1, 19, 1, 20, 1, 20, 1, 20, 1, 21, 1, 21, 1, 21, 1, 21, 3, 21, 177,
		8, 21, 1, 22, 1, 22, 1, 22, 1, 22, 5, 22, 183, 8, 22, 10, 22, 12, 22, 186,
		9, 22, 1, 22, 1, 22, 1, 23, 1, 23, 1, 23, 1, 23, 5, 23, 194, 8, 23, 10,
		23, 12, 23, 197, 9, 23, 1, 23, 1, 23, 1, 24, 1, 24, 5, 24, 203, 8, 24,
		10, 24, 12, 24, 206, 9, 24, 1, 25, 3, 25, 209, 8, 25, 1, 25, 4, 25, 212,
		8, 25, 11, 25, 12, 25, 213, 1, 25, 1, 25, 4, 25, 218, 8, 25, 11, 25, 12,
		25, 219, 3, 25, 222, 8, 25, 1, 26, 1, 26, 5, 26, 226, 8, 26, 10, 26, 12,
		26, 229, 9, 26, 1, 27, 1, 27, 1, 28, 1, 28, 1, 29, 1, 29, 1, 30, 1, 30,
		1, 31, 1, 31, 1, 32, 1, 32, 1, 33, 1, 33, 1, 34, 1, 34, 1, 35, 1, 35, 1,
		36, 1, 36, 1, 37, 1, 37, 1, 38, 1, 38, 1, 39, 1, 39, 1, 40, 1, 40, 1, 41,
		1, 41, 1, 42, 1, 42, 1, 43, 1, 43, 1, 44, 1, 44, 5, 44, 267, 8, 44, 10,
		44, 12, 44, 270, 9, 44, 1, 44, 1, 44, 1, 45, 3, 45, 275, 8, 45, 1, 45,
		1, 45, 1, 46, 4, 46, 280, 8, 46, 11, 46, 12, 46, 281, 1, 46, 1, 46, 0,
		0, 47, 1, 1, 3, 2, 5, 3, 7, 4, 9, 5, 11, 6, 13, 7, 15, 8, 17, 9, 19, 10,
		21, 11, 23, 12, 25, 13, 27, 14, 29, 15, 31, 16, 33, 17, 35, 18, 37, 19,
		39, 20, 41, 21, 43, 22, 45, 23, 47, 24, 49, 25, 51, 26, 53, 27, 55, 28,
		57, 29, 59, 30, 61, 31, 63, 32, 65, 33, 67, 34, 69, 35, 71, 36, 73, 37,
		75, 38, 77, 39, 79, 40, 81, 41, 83, 42, 85, 43, 87, 44, 89, 45, 91, 46,
		93, 47, 1, 0, 11, 2, 0, 65, 90, 97, 122, 5, 0, 45, 45, 48, 57, 65, 90,
		95, 95, 97, 122, 3, 0, 65, 90, 95, 95, 97, 122, 4, 0, 48, 57, 65, 90, 95,
		95, 97, 122, 4, 0, 10, 10, 13, 13, 34, 34, 92, 92, 4, 0, 10, 10, 13, 13,
		39, 39, 92, 92, 1, 0, 48, 57, 2, 0, 46, 47, 126, 126, 5, 0, 42, 42, 45,
		57, 65, 90, 95, 95, 97, 122, 2, 0, 10, 10, 13, 13, 2, 0, 9, 9, 32, 32,
		301, 0, 1, 1, 0, 0, 0, 0, 3, 1, 0, 0, 0, 0, 5, 1, 0, 0, 0, 0, 7, 1, 0,
		0, 0, 0, 9, 1, 0, 0, 0, 0, 11, 1, 0, 0, 0, 0, 13, 1, 0, 0, 0, 0, 15, 1,
		0, 0, 0, 0, 17, 1, 0, 0, 0, 0, 19, 1, 0, 0, 0, 0, 21, 1, 0, 0, 0, 0, 23,
		1, 0, 0, 0, 0, 25, 1, 0, 0, 0, 0, 27, 1, 0, 0, 0, 0, 29, 1, 0, 0, 0, 0,
		31, 1, 0, 0, 0, 0, 33, 1, 0, 0, 0, 0, 35, 1, 0, 0, 0, 0, 37, 1, 0, 0, 0,
		0, 39, 1, 0, 0, 0, 0, 41, 1, 0, 0, 0, 0, 43, 1, 0, 0, 0, 0, 45, 1, 0, 0,
		0, 0, 47, 1, 0, 0, 0, 0, 49, 1, 0, 0, 0, 0, 51, 1, 0, 0, 0, 0, 53, 1, 0,
		0, 0, 0, 55, 1, 0, 0, 0, 0, 57, 1, 0, 0, 0, 0, 59, 1, 0, 0, 0, 0, 61, 1,
		0, 0, 0, 0, 63, 1, 0, 0, 0, 0, 65, 1, 0, 0, 0, 0, 67, 1, 0, 0, 0, 0, 69,
		1, 0, 0, 0, 0, 71, 1, 0, 0, 0, 0, 73, 1, 0, 0, 0, 0, 75, 1, 0, 0, 0, 0,
		77, 1, 0, 0, 0, 0, 79, 1, 0, 0, 0, 0, 81, 1, 0, 0, 0, 0, 83, 1, 0, 0, 0,
		0, 85, 1, 0, 0, 0, 0, 87, 1, 0, 0, 0, 0, 89, 1, 0, 0, 0, 0, 91, 1, 0, 0,
		0, 0, 93, 1, 0, 0, 0, 1, 95, 1, 0, 0, 0, 3, 99, 1, 0, 0, 0, 5, 105, 1,
		0, 0, 0, 7, 110, 1, 0, 0, 0, 9, 120, 1, 0, 0, 0, 11, 122, 1, 0, 0, 0, 13,
		124, 1, 0, 0, 0, 15, 126, 1, 0, 0, 0, 17, 128, 1, 0, 0, 0, 19, 130, 1,
		0, 0, 0, 21, 132, 1, 0, 0, 0, 23, 134, 1, 0, 0, 0, 25, 136, 1, 0, 0, 0,
		27, 138, 1, 0, 0, 0, 29, 140, 1, 0, 0, 0, 31, 142, 1, 0, 0, 0, 33, 144,
		1, 0, 0, 0, 35, 146, 1, 0, 0, 0, 37, 158, 1, 0, 0, 0, 39, 166, 1, 0, 0,
		0, 41, 169, 1, 0, 0, 0, 43, 176, 1, 0, 0, 0, 45, 178, 1, 0, 0, 0, 47, 189,
		1, 0, 0, 0, 49, 200, 1, 0, 0, 0, 51, 208, 1, 0, 0, 0, 53, 223, 1, 0, 0,
		0, 55, 230, 1, 0, 0, 0, 57, 232, 1, 0, 0, 0, 59, 234, 1, 0, 0, 0, 61, 236,
		1, 0, 0, 0, 63, 238, 1, 0, 0, 0, 65, 240, 1, 0, 0, 0, 67, 242, 1, 0, 0,
		0, 69, 244, 1, 0, 0, 0, 71, 246, 1, 0, 0, 0, 73, 248, 1, 0, 0, 0, 75, 250,
		1, 0, 0, 0, 77, 252, 1, 0, 0, 0, 79, 254, 1, 0, 0, 0, 81, 256, 1, 0, 0,
		0, 83, 258, 1, 0, 0, 0, 85, 260, 1, 0, 0, 0, 87, 262, 1, 0, 0, 0, 89, 264,
		1, 0, 0, 0, 91, 274, 1, 0, 0, 0, 93, 279, 1, 0, 0, 0, 95, 96, 5, 100, 0,
		0, 96, 97, 5, 101, 0, 0, 97, 98, 5, 102, 0, 0, 98, 2, 1, 0, 0, 0, 99, 100,
		5, 119, 0, 0, 100, 101, 5, 97, 0, 0, 101, 102, 5, 116, 0, 0, 102, 103,
		5, 99, 0, 0, 103, 104, 5, 104, 0, 0, 104, 4, 1, 0, 0, 0, 105, 106, 5, 115,
		0, 0, 106, 107, 5, 116, 0, 0, 107, 108, 5, 111, 0, 0, 108, 109, 5, 112,
		0, 0, 109, 6, 1, 0, 0, 0, 110, 111, 5, 64, 0, 0, 111, 115, 7, 0, 0, 0,
		112, 114, 7, 1, 0, 0, 113, 112, 1, 0, 0, 0, 114, 117, 1, 0, 0, 0, 115,
		113, 1, 0, 0, 0, 115, 116, 1, 0, 0, 0, 116, 118, 1, 0, 0, 0, 117, 115,
		1, 0, 0, 0, 118, 119, 5, 40, 0, 0, 119, 8, 1, 0, 0, 0, 120, 121, 5, 64,
		0, 0, 121, 10, 1, 0, 0, 0, 122, 123, 5, 61, 0, 0, 123, 12, 1, 0, 0, 0,
		124, 125, 5, 58, 0, 0, 125, 14, 1, 0, 0, 0, 126, 127, 5, 59, 0, 0, 127,
		16, 1, 0, 0, 0, 128, 129, 5, 123, 0, 0, 129, 18, 1, 0, 0, 0, 130, 131,
		5, 125, 0, 0, 131, 20, 1, 0, 0, 0, 132, 133, 5, 40, 0, 0, 133, 22, 1, 0,
		0, 0, 134, 135, 5, 41, 0, 0, 135, 24, 1, 0, 0, 0, 136, 137, 5, 92, 0, 0,
		137, 26, 1, 0, 0, 0, 138, 139, 5, 38, 0, 0, 139, 28, 1, 0, 0, 0, 140, 141,
		5, 124, 0, 0, 141, 30, 1, 0, 0, 0, 142, 143, 5, 60, 0, 0, 143, 32, 1, 0,
		0, 0, 144, 145, 5, 62, 0, 0, 145, 34, 1, 0, 0, 0, 146, 147, 5, 36, 0, 0,
		147, 148, 5, 40, 0, 0, 148, 149, 1, 0, 0, 0, 149, 153, 7, 0, 0, 0, 150,
		152, 7, 1, 0, 0, 151, 150, 1, 0, 0, 0, 152, 155, 1, 0, 0, 0, 153, 151,
		1, 0, 0, 0, 153, 154, 1, 0, 0, 0, 154, 156, 1, 0, 0, 0, 155, 153, 1, 0,
		0, 0, 156, 157, 5, 41, 0, 0, 157, 36, 1, 0, 0, 0, 158, 159, 5, 36, 0, 0,
		159, 163, 7, 2, 0, 0, 160, 162, 7, 3, 0, 0, 161, 160, 1, 0, 0, 0, 162,
		165, 1, 0, 0, 0, 163, 161, 1, 0, 0, 0, 163, 164, 1, 0, 0, 0, 164, 38, 1,
		0, 0, 0, 165, 163, 1, 0, 0, 0, 166, 167, 5, 92, 0, 0, 167, 168, 5, 36,
		0, 0, 168, 40, 1, 0, 0, 0, 169, 170, 5, 92, 0, 0, 170, 171, 5, 59, 0, 0,
		171, 42, 1, 0, 0, 0, 172, 173, 5, 92, 0, 0, 173, 177, 5, 123, 0, 0, 174,
		175, 5, 92, 0, 0, 175, 177, 5, 125, 0, 0, 176, 172, 1, 0, 0, 0, 176, 174,
		1, 0, 0, 0, 177, 44, 1, 0, 0, 0, 178, 184, 5, 34, 0, 0, 179, 183, 8, 4,
		0, 0, 180, 181, 5, 92, 0, 0, 181, 183, 9, 0, 0, 0, 182, 179, 1, 0, 0, 0,
		182, 180, 1, 0, 0, 0, 183, 186, 1, 0, 0, 0, 184, 182, 1, 0, 0, 0, 184,
		185, 1, 0, 0, 0, 185, 187, 1, 0, 0, 0, 186, 184, 1, 0, 0, 0, 187, 188,
		5, 34, 0, 0, 188, 46, 1, 0, 0, 0, 189, 195, 5, 39, 0, 0, 190, 194, 8, 5,
		0, 0, 191, 192, 5, 92, 0, 0, 192, 194, 9, 0, 0, 0, 193, 190, 1, 0, 0, 0,
		193, 191, 1, 0, 0, 0, 194, 197, 1, 0, 0, 0, 195, 193, 1, 0, 0, 0, 195,
		196, 1, 0, 0, 0, 196, 198, 1, 0, 0, 0, 197, 195, 1, 0, 0, 0, 198, 199,
		5, 39, 0, 0, 199, 48, 1, 0, 0, 0, 200, 204, 7, 0, 0, 0, 201, 203, 7, 1,
		0, 0, 202, 201, 1, 0, 0, 0, 203, 206, 1, 0, 0, 0, 204, 202, 1, 0, 0, 0,
		204, 205, 1, 0, 0, 0, 205, 50, 1, 0, 0, 0, 206, 204, 1, 0, 0, 0, 207, 209,
		5, 45, 0, 0, 208, 207, 1, 0, 0, 0, 208, 209, 1, 0, 0, 0, 209, 211, 1, 0,
		0, 0, 210, 212, 7, 6, 0, 0, 211, 210, 1, 0, 0, 0, 212, 213, 1, 0, 0, 0,
		213, 211, 1, 0, 0, 0, 213, 214, 1, 0, 0, 0, 214, 221, 1, 0, 0, 0, 215,
		217, 5, 46, 0, 0, 216, 218, 7, 6, 0, 0, 217, 216, 1, 0, 0, 0, 218, 219,
		1, 0, 0, 0, 219, 217, 1, 0, 0, 0, 219, 220, 1, 0, 0, 0, 220, 222, 1, 0,
		0, 0, 221, 215, 1, 0, 0, 0, 221, 222, 1, 0, 0, 0, 222, 52, 1, 0, 0, 0,
		223, 227, 7, 7, 0, 0, 224, 226, 7, 8, 0, 0, 225, 224, 1, 0, 0, 0, 226,
		229, 1, 0, 0, 0, 227, 225, 1, 0, 0, 0, 227, 228, 1, 0, 0, 0, 228, 54, 1,
		0, 0, 0, 229, 227, 1, 0, 0, 0, 230, 231, 5, 46, 0, 0, 231, 56, 1, 0, 0,
		0, 232, 233, 5, 44, 0, 0, 233, 58, 1, 0, 0, 0, 234, 235, 5, 47, 0, 0, 235,
		60, 1, 0, 0, 0, 236, 237, 5, 45, 0, 0, 237, 62, 1, 0, 0, 0, 238, 239, 5,
		42, 0, 0, 239, 64, 1, 0, 0, 0, 240, 241, 5, 43, 0, 0, 241, 66, 1, 0, 0,
		0, 242, 243, 5, 63, 0, 0, 243, 68, 1, 0, 0, 0, 244, 245, 5, 33, 0, 0, 245,
		70, 1, 0, 0, 0, 246, 247, 5, 37, 0, 0, 247, 72, 1, 0, 0, 0, 248, 249, 5,
		94, 0, 0, 249, 74, 1, 0, 0, 0, 250, 251, 5, 126, 0, 0, 251, 76, 1, 0, 0,
		0, 252, 253, 5, 95, 0, 0, 253, 78, 1, 0, 0, 0, 254, 255, 5, 91, 0, 0, 255,
		80, 1, 0, 0, 0, 256, 257, 5, 93, 0, 0, 257, 82, 1, 0, 0, 0, 258, 259, 5,
		36, 0, 0, 259, 84, 1, 0, 0, 0, 260, 261, 5, 35, 0, 0, 261, 86, 1, 0, 0,
		0, 262, 263, 5, 34, 0, 0, 263, 88, 1, 0, 0, 0, 264, 268, 5, 35, 0, 0, 265,
		267, 8, 9, 0, 0, 266, 265, 1, 0, 0, 0, 267, 270, 1, 0, 0, 0, 268, 266,
		1, 0, 0, 0, 268, 269, 1, 0, 0, 0, 269, 271, 1, 0, 0, 0, 270, 268, 1, 0,
		0, 0, 271, 272, 6, 44, 0, 0, 272, 90, 1, 0, 0, 0, 273, 275, 5, 13, 0, 0,
		274, 273, 1, 0, 0, 0, 274, 275, 1, 0, 0, 0, 275, 276, 1, 0, 0, 0, 276,
		277, 5, 10, 0, 0, 277, 92, 1, 0, 0, 0, 278, 280, 7, 10, 0, 0, 279, 278,
		1, 0, 0, 0, 280, 281, 1, 0, 0, 0, 281, 279, 1, 0, 0, 0, 281, 282, 1, 0,
		0, 0, 282, 283, 1, 0, 0, 0, 283, 284, 6, 46, 0, 0, 284, 94, 1, 0, 0, 0,
		18, 0, 115, 153, 163, 176, 182, 184, 193, 195, 204, 208, 213, 219, 221,
		227, 268, 274, 281, 1, 0, 1, 0,
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
	DevcmdLexerDEF               = 1
	DevcmdLexerWATCH             = 2
	DevcmdLexerSTOP              = 3
	DevcmdLexerAT_NAME_LPAREN    = 4
	DevcmdLexerAT                = 5
	DevcmdLexerEQUALS            = 6
	DevcmdLexerCOLON             = 7
	DevcmdLexerSEMICOLON         = 8
	DevcmdLexerLBRACE            = 9
	DevcmdLexerRBRACE            = 10
	DevcmdLexerLPAREN            = 11
	DevcmdLexerRPAREN            = 12
	DevcmdLexerBACKSLASH         = 13
	DevcmdLexerAMPERSAND         = 14
	DevcmdLexerPIPE              = 15
	DevcmdLexerLT                = 16
	DevcmdLexerGT                = 17
	DevcmdLexerVAR_REF           = 18
	DevcmdLexerSHELL_VAR         = 19
	DevcmdLexerESCAPED_DOLLAR    = 20
	DevcmdLexerESCAPED_SEMICOLON = 21
	DevcmdLexerESCAPED_BRACE     = 22
	DevcmdLexerSTRING            = 23
	DevcmdLexerSINGLE_STRING     = 24
	DevcmdLexerNAME              = 25
	DevcmdLexerNUMBER            = 26
	DevcmdLexerPATH_CONTENT      = 27
	DevcmdLexerDOT               = 28
	DevcmdLexerCOMMA             = 29
	DevcmdLexerSLASH             = 30
	DevcmdLexerDASH              = 31
	DevcmdLexerSTAR              = 32
	DevcmdLexerPLUS              = 33
	DevcmdLexerQUESTION          = 34
	DevcmdLexerEXCLAIM           = 35
	DevcmdLexerPERCENT           = 36
	DevcmdLexerCARET             = 37
	DevcmdLexerTILDE             = 38
	DevcmdLexerUNDERSCORE        = 39
	DevcmdLexerLBRACKET          = 40
	DevcmdLexerRBRACKET          = 41
	DevcmdLexerDOLLAR            = 42
	DevcmdLexerHASH              = 43
	DevcmdLexerDOUBLEQUOTE       = 44
	DevcmdLexerCOMMENT           = 45
	DevcmdLexerNEWLINE           = 46
	DevcmdLexerWS                = 47
)
