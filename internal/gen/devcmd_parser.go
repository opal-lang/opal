// Code generated from DevcmdParser.g4 by ANTLR 4.13.2. DO NOT EDIT.

package gen // DevcmdParser
import (
	"fmt"
	"strconv"
	"sync"

	"github.com/antlr4-go/antlr/v4"
)

// Suppress unused import errors
var _ = fmt.Printf
var _ = strconv.Itoa
var _ = sync.Once{}

type DevcmdParser struct {
	*antlr.BaseParser
}

var DevcmdParserParserStaticData struct {
	once                   sync.Once
	serializedATN          []int32
	LiteralNames           []string
	SymbolicNames          []string
	RuleNames              []string
	PredictionContextCache *antlr.PredictionContextCache
	atn                    *antlr.ATN
	decisionToDFA          []*antlr.DFA
}

func devcmdparserParserInit() {
	staticData := &DevcmdParserParserStaticData
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
		"program", "line", "variableDefinition", "variableValue", "commandDefinition",
		"commandBody", "decoratedCommand", "functionDecorator", "blockDecorator",
		"simpleDecorator", "decorator", "decoratorContent", "decoratorElement",
		"nestedDecorator", "decoratorTextElement", "simpleCommand", "decoratorCommand",
		"blockCommand", "blockStatements", "nonEmptyBlockStatements", "blockStatement",
		"continuationLine", "commandText", "commandTextElement", "inlineDecorator",
	}
	staticData.PredictionContextCache = antlr.NewPredictionContextCache()
	staticData.serializedATN = []int32{
		4, 1, 42, 251, 2, 0, 7, 0, 2, 1, 7, 1, 2, 2, 7, 2, 2, 3, 7, 3, 2, 4, 7,
		4, 2, 5, 7, 5, 2, 6, 7, 6, 2, 7, 7, 7, 2, 8, 7, 8, 2, 9, 7, 9, 2, 10, 7,
		10, 2, 11, 7, 11, 2, 12, 7, 12, 2, 13, 7, 13, 2, 14, 7, 14, 2, 15, 7, 15,
		2, 16, 7, 16, 2, 17, 7, 17, 2, 18, 7, 18, 2, 19, 7, 19, 2, 20, 7, 20, 2,
		21, 7, 21, 2, 22, 7, 22, 2, 23, 7, 23, 2, 24, 7, 24, 1, 0, 5, 0, 52, 8,
		0, 10, 0, 12, 0, 55, 9, 0, 1, 0, 1, 0, 1, 1, 1, 1, 1, 1, 3, 1, 62, 8, 1,
		1, 2, 1, 2, 1, 2, 1, 2, 3, 2, 68, 8, 2, 1, 2, 1, 2, 1, 3, 1, 3, 1, 4, 3,
		4, 75, 8, 4, 1, 4, 1, 4, 1, 4, 1, 4, 1, 5, 1, 5, 1, 5, 3, 5, 84, 8, 5,
		1, 6, 1, 6, 1, 6, 3, 6, 89, 8, 6, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 3,
		7, 97, 8, 7, 1, 8, 1, 8, 1, 8, 1, 8, 1, 8, 1, 9, 1, 9, 1, 9, 1, 9, 1, 9,
		1, 10, 1, 10, 1, 11, 5, 11, 112, 8, 11, 10, 11, 12, 11, 115, 9, 11, 1,
		12, 1, 12, 1, 12, 1, 12, 1, 12, 1, 12, 1, 12, 3, 12, 124, 8, 12, 1, 13,
		1, 13, 1, 13, 1, 13, 1, 13, 1, 13, 1, 14, 1, 14, 1, 15, 1, 15, 5, 15, 136,
		8, 15, 10, 15, 12, 15, 139, 9, 15, 1, 15, 1, 15, 1, 16, 1, 16, 5, 16, 145,
		8, 16, 10, 16, 12, 16, 148, 9, 16, 1, 17, 1, 17, 3, 17, 152, 8, 17, 1,
		17, 1, 17, 1, 17, 1, 18, 1, 18, 3, 18, 159, 8, 18, 1, 19, 1, 19, 1, 19,
		5, 19, 164, 8, 19, 10, 19, 12, 19, 167, 9, 19, 1, 19, 5, 19, 170, 8, 19,
		10, 19, 12, 19, 173, 9, 19, 1, 19, 3, 19, 176, 8, 19, 1, 19, 5, 19, 179,
		8, 19, 10, 19, 12, 19, 182, 9, 19, 1, 20, 1, 20, 1, 20, 5, 20, 187, 8,
		20, 10, 20, 12, 20, 190, 9, 20, 3, 20, 192, 8, 20, 1, 21, 1, 21, 1, 21,
		1, 21, 1, 22, 5, 22, 199, 8, 22, 10, 22, 12, 22, 202, 9, 22, 1, 23, 1,
		23, 1, 23, 1, 23, 1, 23, 1, 23, 1, 23, 1, 23, 1, 23, 1, 23, 1, 23, 1, 23,
		1, 23, 1, 23, 1, 23, 1, 23, 1, 23, 1, 23, 1, 23, 1, 23, 1, 23, 1, 23, 1,
		23, 1, 23, 1, 23, 1, 23, 1, 23, 1, 23, 1, 23, 1, 23, 1, 23, 1, 23, 1, 23,
		1, 23, 1, 23, 1, 23, 1, 23, 1, 23, 1, 23, 3, 23, 243, 8, 23, 1, 24, 1,
		24, 1, 24, 1, 24, 1, 24, 1, 24, 1, 24, 0, 0, 25, 0, 2, 4, 6, 8, 10, 12,
		14, 16, 18, 20, 22, 24, 26, 28, 30, 32, 34, 36, 38, 40, 42, 44, 46, 48,
		0, 2, 1, 0, 2, 3, 3, 0, 1, 3, 5, 9, 12, 39, 288, 0, 53, 1, 0, 0, 0, 2,
		61, 1, 0, 0, 0, 4, 63, 1, 0, 0, 0, 6, 71, 1, 0, 0, 0, 8, 74, 1, 0, 0, 0,
		10, 83, 1, 0, 0, 0, 12, 88, 1, 0, 0, 0, 14, 90, 1, 0, 0, 0, 16, 98, 1,
		0, 0, 0, 18, 103, 1, 0, 0, 0, 20, 108, 1, 0, 0, 0, 22, 113, 1, 0, 0, 0,
		24, 123, 1, 0, 0, 0, 26, 125, 1, 0, 0, 0, 28, 131, 1, 0, 0, 0, 30, 133,
		1, 0, 0, 0, 32, 142, 1, 0, 0, 0, 34, 149, 1, 0, 0, 0, 36, 158, 1, 0, 0,
		0, 38, 160, 1, 0, 0, 0, 40, 191, 1, 0, 0, 0, 42, 193, 1, 0, 0, 0, 44, 200,
		1, 0, 0, 0, 46, 242, 1, 0, 0, 0, 48, 244, 1, 0, 0, 0, 50, 52, 3, 2, 1,
		0, 51, 50, 1, 0, 0, 0, 52, 55, 1, 0, 0, 0, 53, 51, 1, 0, 0, 0, 53, 54,
		1, 0, 0, 0, 54, 56, 1, 0, 0, 0, 55, 53, 1, 0, 0, 0, 56, 57, 5, 0, 0, 1,
		57, 1, 1, 0, 0, 0, 58, 62, 3, 4, 2, 0, 59, 62, 3, 8, 4, 0, 60, 62, 5, 41,
		0, 0, 61, 58, 1, 0, 0, 0, 61, 59, 1, 0, 0, 0, 61, 60, 1, 0, 0, 0, 62, 3,
		1, 0, 0, 0, 63, 64, 5, 1, 0, 0, 64, 65, 5, 19, 0, 0, 65, 67, 5, 5, 0, 0,
		66, 68, 3, 6, 3, 0, 67, 66, 1, 0, 0, 0, 67, 68, 1, 0, 0, 0, 68, 69, 1,
		0, 0, 0, 69, 70, 5, 7, 0, 0, 70, 5, 1, 0, 0, 0, 71, 72, 3, 44, 22, 0, 72,
		7, 1, 0, 0, 0, 73, 75, 7, 0, 0, 0, 74, 73, 1, 0, 0, 0, 74, 75, 1, 0, 0,
		0, 75, 76, 1, 0, 0, 0, 76, 77, 5, 19, 0, 0, 77, 78, 5, 6, 0, 0, 78, 79,
		3, 10, 5, 0, 79, 9, 1, 0, 0, 0, 80, 84, 3, 12, 6, 0, 81, 84, 3, 34, 17,
		0, 82, 84, 3, 30, 15, 0, 83, 80, 1, 0, 0, 0, 83, 81, 1, 0, 0, 0, 83, 82,
		1, 0, 0, 0, 84, 11, 1, 0, 0, 0, 85, 89, 3, 14, 7, 0, 86, 89, 3, 16, 8,
		0, 87, 89, 3, 18, 9, 0, 88, 85, 1, 0, 0, 0, 88, 86, 1, 0, 0, 0, 88, 87,
		1, 0, 0, 0, 89, 13, 1, 0, 0, 0, 90, 91, 5, 4, 0, 0, 91, 92, 5, 19, 0, 0,
		92, 93, 5, 10, 0, 0, 93, 94, 3, 22, 11, 0, 94, 96, 5, 11, 0, 0, 95, 97,
		5, 7, 0, 0, 96, 95, 1, 0, 0, 0, 96, 97, 1, 0, 0, 0, 97, 15, 1, 0, 0, 0,
		98, 99, 5, 4, 0, 0, 99, 100, 3, 20, 10, 0, 100, 101, 5, 6, 0, 0, 101, 102,
		3, 34, 17, 0, 102, 17, 1, 0, 0, 0, 103, 104, 5, 4, 0, 0, 104, 105, 3, 20,
		10, 0, 105, 106, 5, 6, 0, 0, 106, 107, 3, 32, 16, 0, 107, 19, 1, 0, 0,
		0, 108, 109, 5, 19, 0, 0, 109, 21, 1, 0, 0, 0, 110, 112, 3, 24, 12, 0,
		111, 110, 1, 0, 0, 0, 112, 115, 1, 0, 0, 0, 113, 111, 1, 0, 0, 0, 113,
		114, 1, 0, 0, 0, 114, 23, 1, 0, 0, 0, 115, 113, 1, 0, 0, 0, 116, 124, 3,
		26, 13, 0, 117, 118, 5, 10, 0, 0, 118, 119, 3, 22, 11, 0, 119, 120, 5,
		11, 0, 0, 120, 124, 1, 0, 0, 0, 121, 124, 5, 41, 0, 0, 122, 124, 3, 28,
		14, 0, 123, 116, 1, 0, 0, 0, 123, 117, 1, 0, 0, 0, 123, 121, 1, 0, 0, 0,
		123, 122, 1, 0, 0, 0, 124, 25, 1, 0, 0, 0, 125, 126, 5, 4, 0, 0, 126, 127,
		5, 19, 0, 0, 127, 128, 5, 10, 0, 0, 128, 129, 3, 22, 11, 0, 129, 130, 5,
		11, 0, 0, 130, 27, 1, 0, 0, 0, 131, 132, 7, 1, 0, 0, 132, 29, 1, 0, 0,
		0, 133, 137, 3, 44, 22, 0, 134, 136, 3, 42, 21, 0, 135, 134, 1, 0, 0, 0,
		136, 139, 1, 0, 0, 0, 137, 135, 1, 0, 0, 0, 137, 138, 1, 0, 0, 0, 138,
		140, 1, 0, 0, 0, 139, 137, 1, 0, 0, 0, 140, 141, 5, 7, 0, 0, 141, 31, 1,
		0, 0, 0, 142, 146, 3, 44, 22, 0, 143, 145, 3, 42, 21, 0, 144, 143, 1, 0,
		0, 0, 145, 148, 1, 0, 0, 0, 146, 144, 1, 0, 0, 0, 146, 147, 1, 0, 0, 0,
		147, 33, 1, 0, 0, 0, 148, 146, 1, 0, 0, 0, 149, 151, 5, 8, 0, 0, 150, 152,
		5, 41, 0, 0, 151, 150, 1, 0, 0, 0, 151, 152, 1, 0, 0, 0, 152, 153, 1, 0,
		0, 0, 153, 154, 3, 36, 18, 0, 154, 155, 5, 9, 0, 0, 155, 35, 1, 0, 0, 0,
		156, 159, 1, 0, 0, 0, 157, 159, 3, 38, 19, 0, 158, 156, 1, 0, 0, 0, 158,
		157, 1, 0, 0, 0, 159, 37, 1, 0, 0, 0, 160, 171, 3, 40, 20, 0, 161, 165,
		5, 7, 0, 0, 162, 164, 5, 41, 0, 0, 163, 162, 1, 0, 0, 0, 164, 167, 1, 0,
		0, 0, 165, 163, 1, 0, 0, 0, 165, 166, 1, 0, 0, 0, 166, 168, 1, 0, 0, 0,
		167, 165, 1, 0, 0, 0, 168, 170, 3, 40, 20, 0, 169, 161, 1, 0, 0, 0, 170,
		173, 1, 0, 0, 0, 171, 169, 1, 0, 0, 0, 171, 172, 1, 0, 0, 0, 172, 175,
		1, 0, 0, 0, 173, 171, 1, 0, 0, 0, 174, 176, 5, 7, 0, 0, 175, 174, 1, 0,
		0, 0, 175, 176, 1, 0, 0, 0, 176, 180, 1, 0, 0, 0, 177, 179, 5, 41, 0, 0,
		178, 177, 1, 0, 0, 0, 179, 182, 1, 0, 0, 0, 180, 178, 1, 0, 0, 0, 180,
		181, 1, 0, 0, 0, 181, 39, 1, 0, 0, 0, 182, 180, 1, 0, 0, 0, 183, 192, 3,
		12, 6, 0, 184, 188, 3, 44, 22, 0, 185, 187, 3, 42, 21, 0, 186, 185, 1,
		0, 0, 0, 187, 190, 1, 0, 0, 0, 188, 186, 1, 0, 0, 0, 188, 189, 1, 0, 0,
		0, 189, 192, 1, 0, 0, 0, 190, 188, 1, 0, 0, 0, 191, 183, 1, 0, 0, 0, 191,
		184, 1, 0, 0, 0, 192, 41, 1, 0, 0, 0, 193, 194, 5, 12, 0, 0, 194, 195,
		5, 41, 0, 0, 195, 196, 3, 44, 22, 0, 196, 43, 1, 0, 0, 0, 197, 199, 3,
		46, 23, 0, 198, 197, 1, 0, 0, 0, 199, 202, 1, 0, 0, 0, 200, 198, 1, 0,
		0, 0, 200, 201, 1, 0, 0, 0, 201, 45, 1, 0, 0, 0, 202, 200, 1, 0, 0, 0,
		203, 243, 3, 48, 24, 0, 204, 243, 5, 19, 0, 0, 205, 243, 5, 20, 0, 0, 206,
		243, 5, 17, 0, 0, 207, 243, 5, 18, 0, 0, 208, 243, 5, 21, 0, 0, 209, 243,
		5, 10, 0, 0, 210, 243, 5, 11, 0, 0, 211, 243, 5, 8, 0, 0, 212, 243, 5,
		9, 0, 0, 213, 243, 5, 34, 0, 0, 214, 243, 5, 35, 0, 0, 215, 243, 5, 13,
		0, 0, 216, 243, 5, 14, 0, 0, 217, 243, 5, 15, 0, 0, 218, 243, 5, 16, 0,
		0, 219, 243, 5, 6, 0, 0, 220, 243, 5, 5, 0, 0, 221, 243, 5, 12, 0, 0, 222,
		243, 5, 22, 0, 0, 223, 243, 5, 23, 0, 0, 224, 243, 5, 24, 0, 0, 225, 243,
		5, 25, 0, 0, 226, 243, 5, 26, 0, 0, 227, 243, 5, 27, 0, 0, 228, 243, 5,
		28, 0, 0, 229, 243, 5, 29, 0, 0, 230, 243, 5, 30, 0, 0, 231, 243, 5, 31,
		0, 0, 232, 243, 5, 32, 0, 0, 233, 243, 5, 33, 0, 0, 234, 243, 5, 36, 0,
		0, 235, 243, 5, 37, 0, 0, 236, 243, 5, 38, 0, 0, 237, 243, 5, 4, 0, 0,
		238, 243, 5, 2, 0, 0, 239, 243, 5, 3, 0, 0, 240, 243, 5, 1, 0, 0, 241,
		243, 5, 39, 0, 0, 242, 203, 1, 0, 0, 0, 242, 204, 1, 0, 0, 0, 242, 205,
		1, 0, 0, 0, 242, 206, 1, 0, 0, 0, 242, 207, 1, 0, 0, 0, 242, 208, 1, 0,
		0, 0, 242, 209, 1, 0, 0, 0, 242, 210, 1, 0, 0, 0, 242, 211, 1, 0, 0, 0,
		242, 212, 1, 0, 0, 0, 242, 213, 1, 0, 0, 0, 242, 214, 1, 0, 0, 0, 242,
		215, 1, 0, 0, 0, 242, 216, 1, 0, 0, 0, 242, 217, 1, 0, 0, 0, 242, 218,
		1, 0, 0, 0, 242, 219, 1, 0, 0, 0, 242, 220, 1, 0, 0, 0, 242, 221, 1, 0,
		0, 0, 242, 222, 1, 0, 0, 0, 242, 223, 1, 0, 0, 0, 242, 224, 1, 0, 0, 0,
		242, 225, 1, 0, 0, 0, 242, 226, 1, 0, 0, 0, 242, 227, 1, 0, 0, 0, 242,
		228, 1, 0, 0, 0, 242, 229, 1, 0, 0, 0, 242, 230, 1, 0, 0, 0, 242, 231,
		1, 0, 0, 0, 242, 232, 1, 0, 0, 0, 242, 233, 1, 0, 0, 0, 242, 234, 1, 0,
		0, 0, 242, 235, 1, 0, 0, 0, 242, 236, 1, 0, 0, 0, 242, 237, 1, 0, 0, 0,
		242, 238, 1, 0, 0, 0, 242, 239, 1, 0, 0, 0, 242, 240, 1, 0, 0, 0, 242,
		241, 1, 0, 0, 0, 243, 47, 1, 0, 0, 0, 244, 245, 5, 4, 0, 0, 245, 246, 5,
		19, 0, 0, 246, 247, 5, 10, 0, 0, 247, 248, 3, 22, 11, 0, 248, 249, 5, 11,
		0, 0, 249, 49, 1, 0, 0, 0, 21, 53, 61, 67, 74, 83, 88, 96, 113, 123, 137,
		146, 151, 158, 165, 171, 175, 180, 188, 191, 200, 242,
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

// DevcmdParserInit initializes any static state used to implement DevcmdParser. By default the
// static state used to implement the parser is lazily initialized during the first call to
// NewDevcmdParser(). You can call this function if you wish to initialize the static state ahead
// of time.
func DevcmdParserInit() {
	staticData := &DevcmdParserParserStaticData
	staticData.once.Do(devcmdparserParserInit)
}

// NewDevcmdParser produces a new parser instance for the optional input antlr.TokenStream.
func NewDevcmdParser(input antlr.TokenStream) *DevcmdParser {
	DevcmdParserInit()
	this := new(DevcmdParser)
	this.BaseParser = antlr.NewBaseParser(input)
	staticData := &DevcmdParserParserStaticData
	this.Interpreter = antlr.NewParserATNSimulator(this, staticData.atn, staticData.decisionToDFA, staticData.PredictionContextCache)
	this.RuleNames = staticData.RuleNames
	this.LiteralNames = staticData.LiteralNames
	this.SymbolicNames = staticData.SymbolicNames
	this.GrammarFileName = "DevcmdParser.g4"

	return this
}

// DevcmdParser tokens.
const (
	DevcmdParserEOF           = antlr.TokenEOF
	DevcmdParserDEF           = 1
	DevcmdParserWATCH         = 2
	DevcmdParserSTOP          = 3
	DevcmdParserAT            = 4
	DevcmdParserEQUALS        = 5
	DevcmdParserCOLON         = 6
	DevcmdParserSEMICOLON     = 7
	DevcmdParserLBRACE        = 8
	DevcmdParserRBRACE        = 9
	DevcmdParserLPAREN        = 10
	DevcmdParserRPAREN        = 11
	DevcmdParserBACKSLASH     = 12
	DevcmdParserAMPERSAND     = 13
	DevcmdParserPIPE          = 14
	DevcmdParserLT            = 15
	DevcmdParserGT            = 16
	DevcmdParserSTRING        = 17
	DevcmdParserSINGLE_STRING = 18
	DevcmdParserNAME          = 19
	DevcmdParserNUMBER        = 20
	DevcmdParserPATH_CONTENT  = 21
	DevcmdParserDOT           = 22
	DevcmdParserCOMMA         = 23
	DevcmdParserSLASH         = 24
	DevcmdParserDASH          = 25
	DevcmdParserSTAR          = 26
	DevcmdParserPLUS          = 27
	DevcmdParserQUESTION      = 28
	DevcmdParserEXCLAIM       = 29
	DevcmdParserPERCENT       = 30
	DevcmdParserCARET         = 31
	DevcmdParserTILDE         = 32
	DevcmdParserUNDERSCORE    = 33
	DevcmdParserLBRACKET      = 34
	DevcmdParserRBRACKET      = 35
	DevcmdParserDOLLAR        = 36
	DevcmdParserHASH          = 37
	DevcmdParserDOUBLEQUOTE   = 38
	DevcmdParserCONTENT       = 39
	DevcmdParserCOMMENT       = 40
	DevcmdParserNEWLINE       = 41
	DevcmdParserWS            = 42
)

// DevcmdParser rules.
const (
	DevcmdParserRULE_program                 = 0
	DevcmdParserRULE_line                    = 1
	DevcmdParserRULE_variableDefinition      = 2
	DevcmdParserRULE_variableValue           = 3
	DevcmdParserRULE_commandDefinition       = 4
	DevcmdParserRULE_commandBody             = 5
	DevcmdParserRULE_decoratedCommand        = 6
	DevcmdParserRULE_functionDecorator       = 7
	DevcmdParserRULE_blockDecorator          = 8
	DevcmdParserRULE_simpleDecorator         = 9
	DevcmdParserRULE_decorator               = 10
	DevcmdParserRULE_decoratorContent        = 11
	DevcmdParserRULE_decoratorElement        = 12
	DevcmdParserRULE_nestedDecorator         = 13
	DevcmdParserRULE_decoratorTextElement    = 14
	DevcmdParserRULE_simpleCommand           = 15
	DevcmdParserRULE_decoratorCommand        = 16
	DevcmdParserRULE_blockCommand            = 17
	DevcmdParserRULE_blockStatements         = 18
	DevcmdParserRULE_nonEmptyBlockStatements = 19
	DevcmdParserRULE_blockStatement          = 20
	DevcmdParserRULE_continuationLine        = 21
	DevcmdParserRULE_commandText             = 22
	DevcmdParserRULE_commandTextElement      = 23
	DevcmdParserRULE_inlineDecorator         = 24
)

// IProgramContext is an interface to support dynamic dispatch.
type IProgramContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	EOF() antlr.TerminalNode
	AllLine() []ILineContext
	Line(i int) ILineContext

	// IsProgramContext differentiates from other interfaces.
	IsProgramContext()
}

type ProgramContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyProgramContext() *ProgramContext {
	var p = new(ProgramContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_program
	return p
}

func InitEmptyProgramContext(p *ProgramContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_program
}

func (*ProgramContext) IsProgramContext() {}

func NewProgramContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ProgramContext {
	var p = new(ProgramContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = DevcmdParserRULE_program

	return p
}

func (s *ProgramContext) GetParser() antlr.Parser { return s.parser }

func (s *ProgramContext) EOF() antlr.TerminalNode {
	return s.GetToken(DevcmdParserEOF, 0)
}

func (s *ProgramContext) AllLine() []ILineContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(ILineContext); ok {
			len++
		}
	}

	tst := make([]ILineContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(ILineContext); ok {
			tst[i] = t.(ILineContext)
			i++
		}
	}

	return tst
}

func (s *ProgramContext) Line(i int) ILineContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ILineContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(ILineContext)
}

func (s *ProgramContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ProgramContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ProgramContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.EnterProgram(s)
	}
}

func (s *ProgramContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.ExitProgram(s)
	}
}

func (s *ProgramContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case DevcmdParserVisitor:
		return t.VisitProgram(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *DevcmdParser) Program() (localctx IProgramContext) {
	localctx = NewProgramContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 0, DevcmdParserRULE_program)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(53)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&2199023779854) != 0 {
		{
			p.SetState(50)
			p.Line()
		}

		p.SetState(55)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(56)
		p.Match(DevcmdParserEOF)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ILineContext is an interface to support dynamic dispatch.
type ILineContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	VariableDefinition() IVariableDefinitionContext
	CommandDefinition() ICommandDefinitionContext
	NEWLINE() antlr.TerminalNode

	// IsLineContext differentiates from other interfaces.
	IsLineContext()
}

type LineContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyLineContext() *LineContext {
	var p = new(LineContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_line
	return p
}

func InitEmptyLineContext(p *LineContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_line
}

func (*LineContext) IsLineContext() {}

func NewLineContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *LineContext {
	var p = new(LineContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = DevcmdParserRULE_line

	return p
}

func (s *LineContext) GetParser() antlr.Parser { return s.parser }

func (s *LineContext) VariableDefinition() IVariableDefinitionContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IVariableDefinitionContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IVariableDefinitionContext)
}

func (s *LineContext) CommandDefinition() ICommandDefinitionContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ICommandDefinitionContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ICommandDefinitionContext)
}

func (s *LineContext) NEWLINE() antlr.TerminalNode {
	return s.GetToken(DevcmdParserNEWLINE, 0)
}

func (s *LineContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *LineContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *LineContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.EnterLine(s)
	}
}

func (s *LineContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.ExitLine(s)
	}
}

func (s *LineContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case DevcmdParserVisitor:
		return t.VisitLine(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *DevcmdParser) Line() (localctx ILineContext) {
	localctx = NewLineContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 2, DevcmdParserRULE_line)
	p.SetState(61)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case DevcmdParserDEF:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(58)
			p.VariableDefinition()
		}

	case DevcmdParserWATCH, DevcmdParserSTOP, DevcmdParserNAME:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(59)
			p.CommandDefinition()
		}

	case DevcmdParserNEWLINE:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(60)
			p.Match(DevcmdParserNEWLINE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	default:
		p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IVariableDefinitionContext is an interface to support dynamic dispatch.
type IVariableDefinitionContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	DEF() antlr.TerminalNode
	NAME() antlr.TerminalNode
	EQUALS() antlr.TerminalNode
	SEMICOLON() antlr.TerminalNode
	VariableValue() IVariableValueContext

	// IsVariableDefinitionContext differentiates from other interfaces.
	IsVariableDefinitionContext()
}

type VariableDefinitionContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyVariableDefinitionContext() *VariableDefinitionContext {
	var p = new(VariableDefinitionContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_variableDefinition
	return p
}

func InitEmptyVariableDefinitionContext(p *VariableDefinitionContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_variableDefinition
}

func (*VariableDefinitionContext) IsVariableDefinitionContext() {}

func NewVariableDefinitionContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *VariableDefinitionContext {
	var p = new(VariableDefinitionContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = DevcmdParserRULE_variableDefinition

	return p
}

func (s *VariableDefinitionContext) GetParser() antlr.Parser { return s.parser }

func (s *VariableDefinitionContext) DEF() antlr.TerminalNode {
	return s.GetToken(DevcmdParserDEF, 0)
}

func (s *VariableDefinitionContext) NAME() antlr.TerminalNode {
	return s.GetToken(DevcmdParserNAME, 0)
}

func (s *VariableDefinitionContext) EQUALS() antlr.TerminalNode {
	return s.GetToken(DevcmdParserEQUALS, 0)
}

func (s *VariableDefinitionContext) SEMICOLON() antlr.TerminalNode {
	return s.GetToken(DevcmdParserSEMICOLON, 0)
}

func (s *VariableDefinitionContext) VariableValue() IVariableValueContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IVariableValueContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IVariableValueContext)
}

func (s *VariableDefinitionContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *VariableDefinitionContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *VariableDefinitionContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.EnterVariableDefinition(s)
	}
}

func (s *VariableDefinitionContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.ExitVariableDefinition(s)
	}
}

func (s *VariableDefinitionContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case DevcmdParserVisitor:
		return t.VisitVariableDefinition(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *DevcmdParser) VariableDefinition() (localctx IVariableDefinitionContext) {
	localctx = NewVariableDefinitionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 4, DevcmdParserRULE_variableDefinition)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(63)
		p.Match(DevcmdParserDEF)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(64)
		p.Match(DevcmdParserNAME)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(65)
		p.Match(DevcmdParserEQUALS)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(67)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 2, p.GetParserRuleContext()) == 1 {
		{
			p.SetState(66)
			p.VariableValue()
		}

	} else if p.HasError() { // JIM
		goto errorExit
	}
	{
		p.SetState(69)
		p.Match(DevcmdParserSEMICOLON)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IVariableValueContext is an interface to support dynamic dispatch.
type IVariableValueContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	CommandText() ICommandTextContext

	// IsVariableValueContext differentiates from other interfaces.
	IsVariableValueContext()
}

type VariableValueContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyVariableValueContext() *VariableValueContext {
	var p = new(VariableValueContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_variableValue
	return p
}

func InitEmptyVariableValueContext(p *VariableValueContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_variableValue
}

func (*VariableValueContext) IsVariableValueContext() {}

func NewVariableValueContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *VariableValueContext {
	var p = new(VariableValueContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = DevcmdParserRULE_variableValue

	return p
}

func (s *VariableValueContext) GetParser() antlr.Parser { return s.parser }

func (s *VariableValueContext) CommandText() ICommandTextContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ICommandTextContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ICommandTextContext)
}

func (s *VariableValueContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *VariableValueContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *VariableValueContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.EnterVariableValue(s)
	}
}

func (s *VariableValueContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.ExitVariableValue(s)
	}
}

func (s *VariableValueContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case DevcmdParserVisitor:
		return t.VisitVariableValue(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *DevcmdParser) VariableValue() (localctx IVariableValueContext) {
	localctx = NewVariableValueContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 6, DevcmdParserRULE_variableValue)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(71)
		p.CommandText()
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ICommandDefinitionContext is an interface to support dynamic dispatch.
type ICommandDefinitionContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	NAME() antlr.TerminalNode
	COLON() antlr.TerminalNode
	CommandBody() ICommandBodyContext
	WATCH() antlr.TerminalNode
	STOP() antlr.TerminalNode

	// IsCommandDefinitionContext differentiates from other interfaces.
	IsCommandDefinitionContext()
}

type CommandDefinitionContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyCommandDefinitionContext() *CommandDefinitionContext {
	var p = new(CommandDefinitionContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_commandDefinition
	return p
}

func InitEmptyCommandDefinitionContext(p *CommandDefinitionContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_commandDefinition
}

func (*CommandDefinitionContext) IsCommandDefinitionContext() {}

func NewCommandDefinitionContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *CommandDefinitionContext {
	var p = new(CommandDefinitionContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = DevcmdParserRULE_commandDefinition

	return p
}

func (s *CommandDefinitionContext) GetParser() antlr.Parser { return s.parser }

func (s *CommandDefinitionContext) NAME() antlr.TerminalNode {
	return s.GetToken(DevcmdParserNAME, 0)
}

func (s *CommandDefinitionContext) COLON() antlr.TerminalNode {
	return s.GetToken(DevcmdParserCOLON, 0)
}

func (s *CommandDefinitionContext) CommandBody() ICommandBodyContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ICommandBodyContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ICommandBodyContext)
}

func (s *CommandDefinitionContext) WATCH() antlr.TerminalNode {
	return s.GetToken(DevcmdParserWATCH, 0)
}

func (s *CommandDefinitionContext) STOP() antlr.TerminalNode {
	return s.GetToken(DevcmdParserSTOP, 0)
}

func (s *CommandDefinitionContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *CommandDefinitionContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *CommandDefinitionContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.EnterCommandDefinition(s)
	}
}

func (s *CommandDefinitionContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.ExitCommandDefinition(s)
	}
}

func (s *CommandDefinitionContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case DevcmdParserVisitor:
		return t.VisitCommandDefinition(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *DevcmdParser) CommandDefinition() (localctx ICommandDefinitionContext) {
	localctx = NewCommandDefinitionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 8, DevcmdParserRULE_commandDefinition)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(74)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == DevcmdParserWATCH || _la == DevcmdParserSTOP {
		{
			p.SetState(73)
			_la = p.GetTokenStream().LA(1)

			if !(_la == DevcmdParserWATCH || _la == DevcmdParserSTOP) {
				p.GetErrorHandler().RecoverInline(p)
			} else {
				p.GetErrorHandler().ReportMatch(p)
				p.Consume()
			}
		}

	}
	{
		p.SetState(76)
		p.Match(DevcmdParserNAME)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(77)
		p.Match(DevcmdParserCOLON)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(78)
		p.CommandBody()
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ICommandBodyContext is an interface to support dynamic dispatch.
type ICommandBodyContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	DecoratedCommand() IDecoratedCommandContext
	BlockCommand() IBlockCommandContext
	SimpleCommand() ISimpleCommandContext

	// IsCommandBodyContext differentiates from other interfaces.
	IsCommandBodyContext()
}

type CommandBodyContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyCommandBodyContext() *CommandBodyContext {
	var p = new(CommandBodyContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_commandBody
	return p
}

func InitEmptyCommandBodyContext(p *CommandBodyContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_commandBody
}

func (*CommandBodyContext) IsCommandBodyContext() {}

func NewCommandBodyContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *CommandBodyContext {
	var p = new(CommandBodyContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = DevcmdParserRULE_commandBody

	return p
}

func (s *CommandBodyContext) GetParser() antlr.Parser { return s.parser }

func (s *CommandBodyContext) DecoratedCommand() IDecoratedCommandContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IDecoratedCommandContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IDecoratedCommandContext)
}

func (s *CommandBodyContext) BlockCommand() IBlockCommandContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IBlockCommandContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IBlockCommandContext)
}

func (s *CommandBodyContext) SimpleCommand() ISimpleCommandContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ISimpleCommandContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ISimpleCommandContext)
}

func (s *CommandBodyContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *CommandBodyContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *CommandBodyContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.EnterCommandBody(s)
	}
}

func (s *CommandBodyContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.ExitCommandBody(s)
	}
}

func (s *CommandBodyContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case DevcmdParserVisitor:
		return t.VisitCommandBody(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *DevcmdParser) CommandBody() (localctx ICommandBodyContext) {
	localctx = NewCommandBodyContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 10, DevcmdParserRULE_commandBody)
	p.SetState(83)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 4, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(80)
			p.DecoratedCommand()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(81)
			p.BlockCommand()
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(82)
			p.SimpleCommand()
		}

	case antlr.ATNInvalidAltNumber:
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IDecoratedCommandContext is an interface to support dynamic dispatch.
type IDecoratedCommandContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser
	// IsDecoratedCommandContext differentiates from other interfaces.
	IsDecoratedCommandContext()
}

type DecoratedCommandContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyDecoratedCommandContext() *DecoratedCommandContext {
	var p = new(DecoratedCommandContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_decoratedCommand
	return p
}

func InitEmptyDecoratedCommandContext(p *DecoratedCommandContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_decoratedCommand
}

func (*DecoratedCommandContext) IsDecoratedCommandContext() {}

func NewDecoratedCommandContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *DecoratedCommandContext {
	var p = new(DecoratedCommandContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = DevcmdParserRULE_decoratedCommand

	return p
}

func (s *DecoratedCommandContext) GetParser() antlr.Parser { return s.parser }

func (s *DecoratedCommandContext) CopyAll(ctx *DecoratedCommandContext) {
	s.CopyFrom(&ctx.BaseParserRuleContext)
}

func (s *DecoratedCommandContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *DecoratedCommandContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

type SimpleDecoratorLabelContext struct {
	DecoratedCommandContext
}

func NewSimpleDecoratorLabelContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *SimpleDecoratorLabelContext {
	var p = new(SimpleDecoratorLabelContext)

	InitEmptyDecoratedCommandContext(&p.DecoratedCommandContext)
	p.parser = parser
	p.CopyAll(ctx.(*DecoratedCommandContext))

	return p
}

func (s *SimpleDecoratorLabelContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *SimpleDecoratorLabelContext) SimpleDecorator() ISimpleDecoratorContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ISimpleDecoratorContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ISimpleDecoratorContext)
}

func (s *SimpleDecoratorLabelContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.EnterSimpleDecoratorLabel(s)
	}
}

func (s *SimpleDecoratorLabelContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.ExitSimpleDecoratorLabel(s)
	}
}

func (s *SimpleDecoratorLabelContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case DevcmdParserVisitor:
		return t.VisitSimpleDecoratorLabel(s)

	default:
		return t.VisitChildren(s)
	}
}

type BlockDecoratorLabelContext struct {
	DecoratedCommandContext
}

func NewBlockDecoratorLabelContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *BlockDecoratorLabelContext {
	var p = new(BlockDecoratorLabelContext)

	InitEmptyDecoratedCommandContext(&p.DecoratedCommandContext)
	p.parser = parser
	p.CopyAll(ctx.(*DecoratedCommandContext))

	return p
}

func (s *BlockDecoratorLabelContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *BlockDecoratorLabelContext) BlockDecorator() IBlockDecoratorContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IBlockDecoratorContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IBlockDecoratorContext)
}

func (s *BlockDecoratorLabelContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.EnterBlockDecoratorLabel(s)
	}
}

func (s *BlockDecoratorLabelContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.ExitBlockDecoratorLabel(s)
	}
}

func (s *BlockDecoratorLabelContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case DevcmdParserVisitor:
		return t.VisitBlockDecoratorLabel(s)

	default:
		return t.VisitChildren(s)
	}
}

type FunctionDecoratorLabelContext struct {
	DecoratedCommandContext
}

func NewFunctionDecoratorLabelContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *FunctionDecoratorLabelContext {
	var p = new(FunctionDecoratorLabelContext)

	InitEmptyDecoratedCommandContext(&p.DecoratedCommandContext)
	p.parser = parser
	p.CopyAll(ctx.(*DecoratedCommandContext))

	return p
}

func (s *FunctionDecoratorLabelContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *FunctionDecoratorLabelContext) FunctionDecorator() IFunctionDecoratorContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IFunctionDecoratorContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IFunctionDecoratorContext)
}

func (s *FunctionDecoratorLabelContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.EnterFunctionDecoratorLabel(s)
	}
}

func (s *FunctionDecoratorLabelContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.ExitFunctionDecoratorLabel(s)
	}
}

func (s *FunctionDecoratorLabelContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case DevcmdParserVisitor:
		return t.VisitFunctionDecoratorLabel(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *DevcmdParser) DecoratedCommand() (localctx IDecoratedCommandContext) {
	localctx = NewDecoratedCommandContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 12, DevcmdParserRULE_decoratedCommand)
	p.SetState(88)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 5, p.GetParserRuleContext()) {
	case 1:
		localctx = NewFunctionDecoratorLabelContext(p, localctx)
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(85)
			p.FunctionDecorator()
		}

	case 2:
		localctx = NewBlockDecoratorLabelContext(p, localctx)
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(86)
			p.BlockDecorator()
		}

	case 3:
		localctx = NewSimpleDecoratorLabelContext(p, localctx)
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(87)
			p.SimpleDecorator()
		}

	case antlr.ATNInvalidAltNumber:
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IFunctionDecoratorContext is an interface to support dynamic dispatch.
type IFunctionDecoratorContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AT() antlr.TerminalNode
	NAME() antlr.TerminalNode
	LPAREN() antlr.TerminalNode
	DecoratorContent() IDecoratorContentContext
	RPAREN() antlr.TerminalNode
	SEMICOLON() antlr.TerminalNode

	// IsFunctionDecoratorContext differentiates from other interfaces.
	IsFunctionDecoratorContext()
}

type FunctionDecoratorContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyFunctionDecoratorContext() *FunctionDecoratorContext {
	var p = new(FunctionDecoratorContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_functionDecorator
	return p
}

func InitEmptyFunctionDecoratorContext(p *FunctionDecoratorContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_functionDecorator
}

func (*FunctionDecoratorContext) IsFunctionDecoratorContext() {}

func NewFunctionDecoratorContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *FunctionDecoratorContext {
	var p = new(FunctionDecoratorContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = DevcmdParserRULE_functionDecorator

	return p
}

func (s *FunctionDecoratorContext) GetParser() antlr.Parser { return s.parser }

func (s *FunctionDecoratorContext) AT() antlr.TerminalNode {
	return s.GetToken(DevcmdParserAT, 0)
}

func (s *FunctionDecoratorContext) NAME() antlr.TerminalNode {
	return s.GetToken(DevcmdParserNAME, 0)
}

func (s *FunctionDecoratorContext) LPAREN() antlr.TerminalNode {
	return s.GetToken(DevcmdParserLPAREN, 0)
}

func (s *FunctionDecoratorContext) DecoratorContent() IDecoratorContentContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IDecoratorContentContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IDecoratorContentContext)
}

func (s *FunctionDecoratorContext) RPAREN() antlr.TerminalNode {
	return s.GetToken(DevcmdParserRPAREN, 0)
}

func (s *FunctionDecoratorContext) SEMICOLON() antlr.TerminalNode {
	return s.GetToken(DevcmdParserSEMICOLON, 0)
}

func (s *FunctionDecoratorContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *FunctionDecoratorContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *FunctionDecoratorContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.EnterFunctionDecorator(s)
	}
}

func (s *FunctionDecoratorContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.ExitFunctionDecorator(s)
	}
}

func (s *FunctionDecoratorContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case DevcmdParserVisitor:
		return t.VisitFunctionDecorator(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *DevcmdParser) FunctionDecorator() (localctx IFunctionDecoratorContext) {
	localctx = NewFunctionDecoratorContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 14, DevcmdParserRULE_functionDecorator)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(90)
		p.Match(DevcmdParserAT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(91)
		p.Match(DevcmdParserNAME)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(92)
		p.Match(DevcmdParserLPAREN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(93)
		p.DecoratorContent()
	}
	{
		p.SetState(94)
		p.Match(DevcmdParserRPAREN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(96)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 6, p.GetParserRuleContext()) == 1 {
		{
			p.SetState(95)
			p.Match(DevcmdParserSEMICOLON)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	} else if p.HasError() { // JIM
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IBlockDecoratorContext is an interface to support dynamic dispatch.
type IBlockDecoratorContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AT() antlr.TerminalNode
	Decorator() IDecoratorContext
	COLON() antlr.TerminalNode
	BlockCommand() IBlockCommandContext

	// IsBlockDecoratorContext differentiates from other interfaces.
	IsBlockDecoratorContext()
}

type BlockDecoratorContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyBlockDecoratorContext() *BlockDecoratorContext {
	var p = new(BlockDecoratorContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_blockDecorator
	return p
}

func InitEmptyBlockDecoratorContext(p *BlockDecoratorContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_blockDecorator
}

func (*BlockDecoratorContext) IsBlockDecoratorContext() {}

func NewBlockDecoratorContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *BlockDecoratorContext {
	var p = new(BlockDecoratorContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = DevcmdParserRULE_blockDecorator

	return p
}

func (s *BlockDecoratorContext) GetParser() antlr.Parser { return s.parser }

func (s *BlockDecoratorContext) AT() antlr.TerminalNode {
	return s.GetToken(DevcmdParserAT, 0)
}

func (s *BlockDecoratorContext) Decorator() IDecoratorContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IDecoratorContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IDecoratorContext)
}

func (s *BlockDecoratorContext) COLON() antlr.TerminalNode {
	return s.GetToken(DevcmdParserCOLON, 0)
}

func (s *BlockDecoratorContext) BlockCommand() IBlockCommandContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IBlockCommandContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IBlockCommandContext)
}

func (s *BlockDecoratorContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *BlockDecoratorContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *BlockDecoratorContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.EnterBlockDecorator(s)
	}
}

func (s *BlockDecoratorContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.ExitBlockDecorator(s)
	}
}

func (s *BlockDecoratorContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case DevcmdParserVisitor:
		return t.VisitBlockDecorator(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *DevcmdParser) BlockDecorator() (localctx IBlockDecoratorContext) {
	localctx = NewBlockDecoratorContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 16, DevcmdParserRULE_blockDecorator)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(98)
		p.Match(DevcmdParserAT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(99)
		p.Decorator()
	}
	{
		p.SetState(100)
		p.Match(DevcmdParserCOLON)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(101)
		p.BlockCommand()
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ISimpleDecoratorContext is an interface to support dynamic dispatch.
type ISimpleDecoratorContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AT() antlr.TerminalNode
	Decorator() IDecoratorContext
	COLON() antlr.TerminalNode
	DecoratorCommand() IDecoratorCommandContext

	// IsSimpleDecoratorContext differentiates from other interfaces.
	IsSimpleDecoratorContext()
}

type SimpleDecoratorContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptySimpleDecoratorContext() *SimpleDecoratorContext {
	var p = new(SimpleDecoratorContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_simpleDecorator
	return p
}

func InitEmptySimpleDecoratorContext(p *SimpleDecoratorContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_simpleDecorator
}

func (*SimpleDecoratorContext) IsSimpleDecoratorContext() {}

func NewSimpleDecoratorContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *SimpleDecoratorContext {
	var p = new(SimpleDecoratorContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = DevcmdParserRULE_simpleDecorator

	return p
}

func (s *SimpleDecoratorContext) GetParser() antlr.Parser { return s.parser }

func (s *SimpleDecoratorContext) AT() antlr.TerminalNode {
	return s.GetToken(DevcmdParserAT, 0)
}

func (s *SimpleDecoratorContext) Decorator() IDecoratorContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IDecoratorContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IDecoratorContext)
}

func (s *SimpleDecoratorContext) COLON() antlr.TerminalNode {
	return s.GetToken(DevcmdParserCOLON, 0)
}

func (s *SimpleDecoratorContext) DecoratorCommand() IDecoratorCommandContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IDecoratorCommandContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IDecoratorCommandContext)
}

func (s *SimpleDecoratorContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *SimpleDecoratorContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *SimpleDecoratorContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.EnterSimpleDecorator(s)
	}
}

func (s *SimpleDecoratorContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.ExitSimpleDecorator(s)
	}
}

func (s *SimpleDecoratorContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case DevcmdParserVisitor:
		return t.VisitSimpleDecorator(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *DevcmdParser) SimpleDecorator() (localctx ISimpleDecoratorContext) {
	localctx = NewSimpleDecoratorContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 18, DevcmdParserRULE_simpleDecorator)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(103)
		p.Match(DevcmdParserAT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(104)
		p.Decorator()
	}
	{
		p.SetState(105)
		p.Match(DevcmdParserCOLON)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(106)
		p.DecoratorCommand()
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IDecoratorContext is an interface to support dynamic dispatch.
type IDecoratorContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	NAME() antlr.TerminalNode

	// IsDecoratorContext differentiates from other interfaces.
	IsDecoratorContext()
}

type DecoratorContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyDecoratorContext() *DecoratorContext {
	var p = new(DecoratorContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_decorator
	return p
}

func InitEmptyDecoratorContext(p *DecoratorContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_decorator
}

func (*DecoratorContext) IsDecoratorContext() {}

func NewDecoratorContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *DecoratorContext {
	var p = new(DecoratorContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = DevcmdParserRULE_decorator

	return p
}

func (s *DecoratorContext) GetParser() antlr.Parser { return s.parser }

func (s *DecoratorContext) NAME() antlr.TerminalNode {
	return s.GetToken(DevcmdParserNAME, 0)
}

func (s *DecoratorContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *DecoratorContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *DecoratorContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.EnterDecorator(s)
	}
}

func (s *DecoratorContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.ExitDecorator(s)
	}
}

func (s *DecoratorContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case DevcmdParserVisitor:
		return t.VisitDecorator(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *DevcmdParser) Decorator() (localctx IDecoratorContext) {
	localctx = NewDecoratorContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 20, DevcmdParserRULE_decorator)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(108)
		p.Match(DevcmdParserNAME)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IDecoratorContentContext is an interface to support dynamic dispatch.
type IDecoratorContentContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllDecoratorElement() []IDecoratorElementContext
	DecoratorElement(i int) IDecoratorElementContext

	// IsDecoratorContentContext differentiates from other interfaces.
	IsDecoratorContentContext()
}

type DecoratorContentContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyDecoratorContentContext() *DecoratorContentContext {
	var p = new(DecoratorContentContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_decoratorContent
	return p
}

func InitEmptyDecoratorContentContext(p *DecoratorContentContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_decoratorContent
}

func (*DecoratorContentContext) IsDecoratorContentContext() {}

func NewDecoratorContentContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *DecoratorContentContext {
	var p = new(DecoratorContentContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = DevcmdParserRULE_decoratorContent

	return p
}

func (s *DecoratorContentContext) GetParser() antlr.Parser { return s.parser }

func (s *DecoratorContentContext) AllDecoratorElement() []IDecoratorElementContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IDecoratorElementContext); ok {
			len++
		}
	}

	tst := make([]IDecoratorElementContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IDecoratorElementContext); ok {
			tst[i] = t.(IDecoratorElementContext)
			i++
		}
	}

	return tst
}

func (s *DecoratorContentContext) DecoratorElement(i int) IDecoratorElementContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IDecoratorElementContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IDecoratorElementContext)
}

func (s *DecoratorContentContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *DecoratorContentContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *DecoratorContentContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.EnterDecoratorContent(s)
	}
}

func (s *DecoratorContentContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.ExitDecoratorContent(s)
	}
}

func (s *DecoratorContentContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case DevcmdParserVisitor:
		return t.VisitDecoratorContent(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *DevcmdParser) DecoratorContent() (localctx IDecoratorContentContext) {
	localctx = NewDecoratorContentContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 22, DevcmdParserRULE_decoratorContent)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(113)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&3298534881278) != 0 {
		{
			p.SetState(110)
			p.DecoratorElement()
		}

		p.SetState(115)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IDecoratorElementContext is an interface to support dynamic dispatch.
type IDecoratorElementContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	NestedDecorator() INestedDecoratorContext
	LPAREN() antlr.TerminalNode
	DecoratorContent() IDecoratorContentContext
	RPAREN() antlr.TerminalNode
	NEWLINE() antlr.TerminalNode
	DecoratorTextElement() IDecoratorTextElementContext

	// IsDecoratorElementContext differentiates from other interfaces.
	IsDecoratorElementContext()
}

type DecoratorElementContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyDecoratorElementContext() *DecoratorElementContext {
	var p = new(DecoratorElementContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_decoratorElement
	return p
}

func InitEmptyDecoratorElementContext(p *DecoratorElementContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_decoratorElement
}

func (*DecoratorElementContext) IsDecoratorElementContext() {}

func NewDecoratorElementContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *DecoratorElementContext {
	var p = new(DecoratorElementContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = DevcmdParserRULE_decoratorElement

	return p
}

func (s *DecoratorElementContext) GetParser() antlr.Parser { return s.parser }

func (s *DecoratorElementContext) NestedDecorator() INestedDecoratorContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(INestedDecoratorContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(INestedDecoratorContext)
}

func (s *DecoratorElementContext) LPAREN() antlr.TerminalNode {
	return s.GetToken(DevcmdParserLPAREN, 0)
}

func (s *DecoratorElementContext) DecoratorContent() IDecoratorContentContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IDecoratorContentContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IDecoratorContentContext)
}

func (s *DecoratorElementContext) RPAREN() antlr.TerminalNode {
	return s.GetToken(DevcmdParserRPAREN, 0)
}

func (s *DecoratorElementContext) NEWLINE() antlr.TerminalNode {
	return s.GetToken(DevcmdParserNEWLINE, 0)
}

func (s *DecoratorElementContext) DecoratorTextElement() IDecoratorTextElementContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IDecoratorTextElementContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IDecoratorTextElementContext)
}

func (s *DecoratorElementContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *DecoratorElementContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *DecoratorElementContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.EnterDecoratorElement(s)
	}
}

func (s *DecoratorElementContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.ExitDecoratorElement(s)
	}
}

func (s *DecoratorElementContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case DevcmdParserVisitor:
		return t.VisitDecoratorElement(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *DevcmdParser) DecoratorElement() (localctx IDecoratorElementContext) {
	localctx = NewDecoratorElementContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 24, DevcmdParserRULE_decoratorElement)
	p.SetState(123)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case DevcmdParserAT:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(116)
			p.NestedDecorator()
		}

	case DevcmdParserLPAREN:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(117)
			p.Match(DevcmdParserLPAREN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(118)
			p.DecoratorContent()
		}
		{
			p.SetState(119)
			p.Match(DevcmdParserRPAREN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case DevcmdParserNEWLINE:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(121)
			p.Match(DevcmdParserNEWLINE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case DevcmdParserDEF, DevcmdParserWATCH, DevcmdParserSTOP, DevcmdParserEQUALS, DevcmdParserCOLON, DevcmdParserSEMICOLON, DevcmdParserLBRACE, DevcmdParserRBRACE, DevcmdParserBACKSLASH, DevcmdParserAMPERSAND, DevcmdParserPIPE, DevcmdParserLT, DevcmdParserGT, DevcmdParserSTRING, DevcmdParserSINGLE_STRING, DevcmdParserNAME, DevcmdParserNUMBER, DevcmdParserPATH_CONTENT, DevcmdParserDOT, DevcmdParserCOMMA, DevcmdParserSLASH, DevcmdParserDASH, DevcmdParserSTAR, DevcmdParserPLUS, DevcmdParserQUESTION, DevcmdParserEXCLAIM, DevcmdParserPERCENT, DevcmdParserCARET, DevcmdParserTILDE, DevcmdParserUNDERSCORE, DevcmdParserLBRACKET, DevcmdParserRBRACKET, DevcmdParserDOLLAR, DevcmdParserHASH, DevcmdParserDOUBLEQUOTE, DevcmdParserCONTENT:
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(122)
			p.DecoratorTextElement()
		}

	default:
		p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// INestedDecoratorContext is an interface to support dynamic dispatch.
type INestedDecoratorContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AT() antlr.TerminalNode
	NAME() antlr.TerminalNode
	LPAREN() antlr.TerminalNode
	DecoratorContent() IDecoratorContentContext
	RPAREN() antlr.TerminalNode

	// IsNestedDecoratorContext differentiates from other interfaces.
	IsNestedDecoratorContext()
}

type NestedDecoratorContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyNestedDecoratorContext() *NestedDecoratorContext {
	var p = new(NestedDecoratorContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_nestedDecorator
	return p
}

func InitEmptyNestedDecoratorContext(p *NestedDecoratorContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_nestedDecorator
}

func (*NestedDecoratorContext) IsNestedDecoratorContext() {}

func NewNestedDecoratorContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *NestedDecoratorContext {
	var p = new(NestedDecoratorContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = DevcmdParserRULE_nestedDecorator

	return p
}

func (s *NestedDecoratorContext) GetParser() antlr.Parser { return s.parser }

func (s *NestedDecoratorContext) AT() antlr.TerminalNode {
	return s.GetToken(DevcmdParserAT, 0)
}

func (s *NestedDecoratorContext) NAME() antlr.TerminalNode {
	return s.GetToken(DevcmdParserNAME, 0)
}

func (s *NestedDecoratorContext) LPAREN() antlr.TerminalNode {
	return s.GetToken(DevcmdParserLPAREN, 0)
}

func (s *NestedDecoratorContext) DecoratorContent() IDecoratorContentContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IDecoratorContentContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IDecoratorContentContext)
}

func (s *NestedDecoratorContext) RPAREN() antlr.TerminalNode {
	return s.GetToken(DevcmdParserRPAREN, 0)
}

func (s *NestedDecoratorContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *NestedDecoratorContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *NestedDecoratorContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.EnterNestedDecorator(s)
	}
}

func (s *NestedDecoratorContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.ExitNestedDecorator(s)
	}
}

func (s *NestedDecoratorContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case DevcmdParserVisitor:
		return t.VisitNestedDecorator(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *DevcmdParser) NestedDecorator() (localctx INestedDecoratorContext) {
	localctx = NewNestedDecoratorContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 26, DevcmdParserRULE_nestedDecorator)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(125)
		p.Match(DevcmdParserAT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(126)
		p.Match(DevcmdParserNAME)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(127)
		p.Match(DevcmdParserLPAREN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(128)
		p.DecoratorContent()
	}
	{
		p.SetState(129)
		p.Match(DevcmdParserRPAREN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IDecoratorTextElementContext is an interface to support dynamic dispatch.
type IDecoratorTextElementContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	NAME() antlr.TerminalNode
	NUMBER() antlr.TerminalNode
	STRING() antlr.TerminalNode
	SINGLE_STRING() antlr.TerminalNode
	PATH_CONTENT() antlr.TerminalNode
	AMPERSAND() antlr.TerminalNode
	PIPE() antlr.TerminalNode
	LT() antlr.TerminalNode
	GT() antlr.TerminalNode
	COLON() antlr.TerminalNode
	EQUALS() antlr.TerminalNode
	BACKSLASH() antlr.TerminalNode
	DOT() antlr.TerminalNode
	COMMA() antlr.TerminalNode
	SLASH() antlr.TerminalNode
	DASH() antlr.TerminalNode
	STAR() antlr.TerminalNode
	PLUS() antlr.TerminalNode
	QUESTION() antlr.TerminalNode
	EXCLAIM() antlr.TerminalNode
	PERCENT() antlr.TerminalNode
	CARET() antlr.TerminalNode
	TILDE() antlr.TerminalNode
	UNDERSCORE() antlr.TerminalNode
	LBRACKET() antlr.TerminalNode
	RBRACKET() antlr.TerminalNode
	LBRACE() antlr.TerminalNode
	RBRACE() antlr.TerminalNode
	DOLLAR() antlr.TerminalNode
	HASH() antlr.TerminalNode
	DOUBLEQUOTE() antlr.TerminalNode
	SEMICOLON() antlr.TerminalNode
	WATCH() antlr.TerminalNode
	STOP() antlr.TerminalNode
	DEF() antlr.TerminalNode
	CONTENT() antlr.TerminalNode

	// IsDecoratorTextElementContext differentiates from other interfaces.
	IsDecoratorTextElementContext()
}

type DecoratorTextElementContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyDecoratorTextElementContext() *DecoratorTextElementContext {
	var p = new(DecoratorTextElementContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_decoratorTextElement
	return p
}

func InitEmptyDecoratorTextElementContext(p *DecoratorTextElementContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_decoratorTextElement
}

func (*DecoratorTextElementContext) IsDecoratorTextElementContext() {}

func NewDecoratorTextElementContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *DecoratorTextElementContext {
	var p = new(DecoratorTextElementContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = DevcmdParserRULE_decoratorTextElement

	return p
}

func (s *DecoratorTextElementContext) GetParser() antlr.Parser { return s.parser }

func (s *DecoratorTextElementContext) NAME() antlr.TerminalNode {
	return s.GetToken(DevcmdParserNAME, 0)
}

func (s *DecoratorTextElementContext) NUMBER() antlr.TerminalNode {
	return s.GetToken(DevcmdParserNUMBER, 0)
}

func (s *DecoratorTextElementContext) STRING() antlr.TerminalNode {
	return s.GetToken(DevcmdParserSTRING, 0)
}

func (s *DecoratorTextElementContext) SINGLE_STRING() antlr.TerminalNode {
	return s.GetToken(DevcmdParserSINGLE_STRING, 0)
}

func (s *DecoratorTextElementContext) PATH_CONTENT() antlr.TerminalNode {
	return s.GetToken(DevcmdParserPATH_CONTENT, 0)
}

func (s *DecoratorTextElementContext) AMPERSAND() antlr.TerminalNode {
	return s.GetToken(DevcmdParserAMPERSAND, 0)
}

func (s *DecoratorTextElementContext) PIPE() antlr.TerminalNode {
	return s.GetToken(DevcmdParserPIPE, 0)
}

func (s *DecoratorTextElementContext) LT() antlr.TerminalNode {
	return s.GetToken(DevcmdParserLT, 0)
}

func (s *DecoratorTextElementContext) GT() antlr.TerminalNode {
	return s.GetToken(DevcmdParserGT, 0)
}

func (s *DecoratorTextElementContext) COLON() antlr.TerminalNode {
	return s.GetToken(DevcmdParserCOLON, 0)
}

func (s *DecoratorTextElementContext) EQUALS() antlr.TerminalNode {
	return s.GetToken(DevcmdParserEQUALS, 0)
}

func (s *DecoratorTextElementContext) BACKSLASH() antlr.TerminalNode {
	return s.GetToken(DevcmdParserBACKSLASH, 0)
}

func (s *DecoratorTextElementContext) DOT() antlr.TerminalNode {
	return s.GetToken(DevcmdParserDOT, 0)
}

func (s *DecoratorTextElementContext) COMMA() antlr.TerminalNode {
	return s.GetToken(DevcmdParserCOMMA, 0)
}

func (s *DecoratorTextElementContext) SLASH() antlr.TerminalNode {
	return s.GetToken(DevcmdParserSLASH, 0)
}

func (s *DecoratorTextElementContext) DASH() antlr.TerminalNode {
	return s.GetToken(DevcmdParserDASH, 0)
}

func (s *DecoratorTextElementContext) STAR() antlr.TerminalNode {
	return s.GetToken(DevcmdParserSTAR, 0)
}

func (s *DecoratorTextElementContext) PLUS() antlr.TerminalNode {
	return s.GetToken(DevcmdParserPLUS, 0)
}

func (s *DecoratorTextElementContext) QUESTION() antlr.TerminalNode {
	return s.GetToken(DevcmdParserQUESTION, 0)
}

func (s *DecoratorTextElementContext) EXCLAIM() antlr.TerminalNode {
	return s.GetToken(DevcmdParserEXCLAIM, 0)
}

func (s *DecoratorTextElementContext) PERCENT() antlr.TerminalNode {
	return s.GetToken(DevcmdParserPERCENT, 0)
}

func (s *DecoratorTextElementContext) CARET() antlr.TerminalNode {
	return s.GetToken(DevcmdParserCARET, 0)
}

func (s *DecoratorTextElementContext) TILDE() antlr.TerminalNode {
	return s.GetToken(DevcmdParserTILDE, 0)
}

func (s *DecoratorTextElementContext) UNDERSCORE() antlr.TerminalNode {
	return s.GetToken(DevcmdParserUNDERSCORE, 0)
}

func (s *DecoratorTextElementContext) LBRACKET() antlr.TerminalNode {
	return s.GetToken(DevcmdParserLBRACKET, 0)
}

func (s *DecoratorTextElementContext) RBRACKET() antlr.TerminalNode {
	return s.GetToken(DevcmdParserRBRACKET, 0)
}

func (s *DecoratorTextElementContext) LBRACE() antlr.TerminalNode {
	return s.GetToken(DevcmdParserLBRACE, 0)
}

func (s *DecoratorTextElementContext) RBRACE() antlr.TerminalNode {
	return s.GetToken(DevcmdParserRBRACE, 0)
}

func (s *DecoratorTextElementContext) DOLLAR() antlr.TerminalNode {
	return s.GetToken(DevcmdParserDOLLAR, 0)
}

func (s *DecoratorTextElementContext) HASH() antlr.TerminalNode {
	return s.GetToken(DevcmdParserHASH, 0)
}

func (s *DecoratorTextElementContext) DOUBLEQUOTE() antlr.TerminalNode {
	return s.GetToken(DevcmdParserDOUBLEQUOTE, 0)
}

func (s *DecoratorTextElementContext) SEMICOLON() antlr.TerminalNode {
	return s.GetToken(DevcmdParserSEMICOLON, 0)
}

func (s *DecoratorTextElementContext) WATCH() antlr.TerminalNode {
	return s.GetToken(DevcmdParserWATCH, 0)
}

func (s *DecoratorTextElementContext) STOP() antlr.TerminalNode {
	return s.GetToken(DevcmdParserSTOP, 0)
}

func (s *DecoratorTextElementContext) DEF() antlr.TerminalNode {
	return s.GetToken(DevcmdParserDEF, 0)
}

func (s *DecoratorTextElementContext) CONTENT() antlr.TerminalNode {
	return s.GetToken(DevcmdParserCONTENT, 0)
}

func (s *DecoratorTextElementContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *DecoratorTextElementContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *DecoratorTextElementContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.EnterDecoratorTextElement(s)
	}
}

func (s *DecoratorTextElementContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.ExitDecoratorTextElement(s)
	}
}

func (s *DecoratorTextElementContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case DevcmdParserVisitor:
		return t.VisitDecoratorTextElement(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *DevcmdParser) DecoratorTextElement() (localctx IDecoratorTextElementContext) {
	localctx = NewDecoratorTextElementContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 28, DevcmdParserRULE_decoratorTextElement)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(131)
		_la = p.GetTokenStream().LA(1)

		if !((int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&1099511624686) != 0) {
			p.GetErrorHandler().RecoverInline(p)
		} else {
			p.GetErrorHandler().ReportMatch(p)
			p.Consume()
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ISimpleCommandContext is an interface to support dynamic dispatch.
type ISimpleCommandContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	CommandText() ICommandTextContext
	SEMICOLON() antlr.TerminalNode
	AllContinuationLine() []IContinuationLineContext
	ContinuationLine(i int) IContinuationLineContext

	// IsSimpleCommandContext differentiates from other interfaces.
	IsSimpleCommandContext()
}

type SimpleCommandContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptySimpleCommandContext() *SimpleCommandContext {
	var p = new(SimpleCommandContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_simpleCommand
	return p
}

func InitEmptySimpleCommandContext(p *SimpleCommandContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_simpleCommand
}

func (*SimpleCommandContext) IsSimpleCommandContext() {}

func NewSimpleCommandContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *SimpleCommandContext {
	var p = new(SimpleCommandContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = DevcmdParserRULE_simpleCommand

	return p
}

func (s *SimpleCommandContext) GetParser() antlr.Parser { return s.parser }

func (s *SimpleCommandContext) CommandText() ICommandTextContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ICommandTextContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ICommandTextContext)
}

func (s *SimpleCommandContext) SEMICOLON() antlr.TerminalNode {
	return s.GetToken(DevcmdParserSEMICOLON, 0)
}

func (s *SimpleCommandContext) AllContinuationLine() []IContinuationLineContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IContinuationLineContext); ok {
			len++
		}
	}

	tst := make([]IContinuationLineContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IContinuationLineContext); ok {
			tst[i] = t.(IContinuationLineContext)
			i++
		}
	}

	return tst
}

func (s *SimpleCommandContext) ContinuationLine(i int) IContinuationLineContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IContinuationLineContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IContinuationLineContext)
}

func (s *SimpleCommandContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *SimpleCommandContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *SimpleCommandContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.EnterSimpleCommand(s)
	}
}

func (s *SimpleCommandContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.ExitSimpleCommand(s)
	}
}

func (s *SimpleCommandContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case DevcmdParserVisitor:
		return t.VisitSimpleCommand(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *DevcmdParser) SimpleCommand() (localctx ISimpleCommandContext) {
	localctx = NewSimpleCommandContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 30, DevcmdParserRULE_simpleCommand)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(133)
		p.CommandText()
	}
	p.SetState(137)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == DevcmdParserBACKSLASH {
		{
			p.SetState(134)
			p.ContinuationLine()
		}

		p.SetState(139)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(140)
		p.Match(DevcmdParserSEMICOLON)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IDecoratorCommandContext is an interface to support dynamic dispatch.
type IDecoratorCommandContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	CommandText() ICommandTextContext
	AllContinuationLine() []IContinuationLineContext
	ContinuationLine(i int) IContinuationLineContext

	// IsDecoratorCommandContext differentiates from other interfaces.
	IsDecoratorCommandContext()
}

type DecoratorCommandContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyDecoratorCommandContext() *DecoratorCommandContext {
	var p = new(DecoratorCommandContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_decoratorCommand
	return p
}

func InitEmptyDecoratorCommandContext(p *DecoratorCommandContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_decoratorCommand
}

func (*DecoratorCommandContext) IsDecoratorCommandContext() {}

func NewDecoratorCommandContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *DecoratorCommandContext {
	var p = new(DecoratorCommandContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = DevcmdParserRULE_decoratorCommand

	return p
}

func (s *DecoratorCommandContext) GetParser() antlr.Parser { return s.parser }

func (s *DecoratorCommandContext) CommandText() ICommandTextContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ICommandTextContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ICommandTextContext)
}

func (s *DecoratorCommandContext) AllContinuationLine() []IContinuationLineContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IContinuationLineContext); ok {
			len++
		}
	}

	tst := make([]IContinuationLineContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IContinuationLineContext); ok {
			tst[i] = t.(IContinuationLineContext)
			i++
		}
	}

	return tst
}

func (s *DecoratorCommandContext) ContinuationLine(i int) IContinuationLineContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IContinuationLineContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IContinuationLineContext)
}

func (s *DecoratorCommandContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *DecoratorCommandContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *DecoratorCommandContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.EnterDecoratorCommand(s)
	}
}

func (s *DecoratorCommandContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.ExitDecoratorCommand(s)
	}
}

func (s *DecoratorCommandContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case DevcmdParserVisitor:
		return t.VisitDecoratorCommand(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *DevcmdParser) DecoratorCommand() (localctx IDecoratorCommandContext) {
	localctx = NewDecoratorCommandContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 32, DevcmdParserRULE_decoratorCommand)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(142)
		p.CommandText()
	}
	p.SetState(146)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == DevcmdParserBACKSLASH {
		{
			p.SetState(143)
			p.ContinuationLine()
		}

		p.SetState(148)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IBlockCommandContext is an interface to support dynamic dispatch.
type IBlockCommandContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	LBRACE() antlr.TerminalNode
	BlockStatements() IBlockStatementsContext
	RBRACE() antlr.TerminalNode
	NEWLINE() antlr.TerminalNode

	// IsBlockCommandContext differentiates from other interfaces.
	IsBlockCommandContext()
}

type BlockCommandContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyBlockCommandContext() *BlockCommandContext {
	var p = new(BlockCommandContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_blockCommand
	return p
}

func InitEmptyBlockCommandContext(p *BlockCommandContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_blockCommand
}

func (*BlockCommandContext) IsBlockCommandContext() {}

func NewBlockCommandContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *BlockCommandContext {
	var p = new(BlockCommandContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = DevcmdParserRULE_blockCommand

	return p
}

func (s *BlockCommandContext) GetParser() antlr.Parser { return s.parser }

func (s *BlockCommandContext) LBRACE() antlr.TerminalNode {
	return s.GetToken(DevcmdParserLBRACE, 0)
}

func (s *BlockCommandContext) BlockStatements() IBlockStatementsContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IBlockStatementsContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IBlockStatementsContext)
}

func (s *BlockCommandContext) RBRACE() antlr.TerminalNode {
	return s.GetToken(DevcmdParserRBRACE, 0)
}

func (s *BlockCommandContext) NEWLINE() antlr.TerminalNode {
	return s.GetToken(DevcmdParserNEWLINE, 0)
}

func (s *BlockCommandContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *BlockCommandContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *BlockCommandContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.EnterBlockCommand(s)
	}
}

func (s *BlockCommandContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.ExitBlockCommand(s)
	}
}

func (s *BlockCommandContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case DevcmdParserVisitor:
		return t.VisitBlockCommand(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *DevcmdParser) BlockCommand() (localctx IBlockCommandContext) {
	localctx = NewBlockCommandContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 34, DevcmdParserRULE_blockCommand)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(149)
		p.Match(DevcmdParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(151)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 11, p.GetParserRuleContext()) == 1 {
		{
			p.SetState(150)
			p.Match(DevcmdParserNEWLINE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	} else if p.HasError() { // JIM
		goto errorExit
	}
	{
		p.SetState(153)
		p.BlockStatements()
	}
	{
		p.SetState(154)
		p.Match(DevcmdParserRBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IBlockStatementsContext is an interface to support dynamic dispatch.
type IBlockStatementsContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	NonEmptyBlockStatements() INonEmptyBlockStatementsContext

	// IsBlockStatementsContext differentiates from other interfaces.
	IsBlockStatementsContext()
}

type BlockStatementsContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyBlockStatementsContext() *BlockStatementsContext {
	var p = new(BlockStatementsContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_blockStatements
	return p
}

func InitEmptyBlockStatementsContext(p *BlockStatementsContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_blockStatements
}

func (*BlockStatementsContext) IsBlockStatementsContext() {}

func NewBlockStatementsContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *BlockStatementsContext {
	var p = new(BlockStatementsContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = DevcmdParserRULE_blockStatements

	return p
}

func (s *BlockStatementsContext) GetParser() antlr.Parser { return s.parser }

func (s *BlockStatementsContext) NonEmptyBlockStatements() INonEmptyBlockStatementsContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(INonEmptyBlockStatementsContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(INonEmptyBlockStatementsContext)
}

func (s *BlockStatementsContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *BlockStatementsContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *BlockStatementsContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.EnterBlockStatements(s)
	}
}

func (s *BlockStatementsContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.ExitBlockStatements(s)
	}
}

func (s *BlockStatementsContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case DevcmdParserVisitor:
		return t.VisitBlockStatements(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *DevcmdParser) BlockStatements() (localctx IBlockStatementsContext) {
	localctx = NewBlockStatementsContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 36, DevcmdParserRULE_blockStatements)
	p.SetState(158)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 12, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(157)
			p.NonEmptyBlockStatements()
		}

	case antlr.ATNInvalidAltNumber:
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// INonEmptyBlockStatementsContext is an interface to support dynamic dispatch.
type INonEmptyBlockStatementsContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllBlockStatement() []IBlockStatementContext
	BlockStatement(i int) IBlockStatementContext
	AllSEMICOLON() []antlr.TerminalNode
	SEMICOLON(i int) antlr.TerminalNode
	AllNEWLINE() []antlr.TerminalNode
	NEWLINE(i int) antlr.TerminalNode

	// IsNonEmptyBlockStatementsContext differentiates from other interfaces.
	IsNonEmptyBlockStatementsContext()
}

type NonEmptyBlockStatementsContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyNonEmptyBlockStatementsContext() *NonEmptyBlockStatementsContext {
	var p = new(NonEmptyBlockStatementsContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_nonEmptyBlockStatements
	return p
}

func InitEmptyNonEmptyBlockStatementsContext(p *NonEmptyBlockStatementsContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_nonEmptyBlockStatements
}

func (*NonEmptyBlockStatementsContext) IsNonEmptyBlockStatementsContext() {}

func NewNonEmptyBlockStatementsContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *NonEmptyBlockStatementsContext {
	var p = new(NonEmptyBlockStatementsContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = DevcmdParserRULE_nonEmptyBlockStatements

	return p
}

func (s *NonEmptyBlockStatementsContext) GetParser() antlr.Parser { return s.parser }

func (s *NonEmptyBlockStatementsContext) AllBlockStatement() []IBlockStatementContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IBlockStatementContext); ok {
			len++
		}
	}

	tst := make([]IBlockStatementContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IBlockStatementContext); ok {
			tst[i] = t.(IBlockStatementContext)
			i++
		}
	}

	return tst
}

func (s *NonEmptyBlockStatementsContext) BlockStatement(i int) IBlockStatementContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IBlockStatementContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IBlockStatementContext)
}

func (s *NonEmptyBlockStatementsContext) AllSEMICOLON() []antlr.TerminalNode {
	return s.GetTokens(DevcmdParserSEMICOLON)
}

func (s *NonEmptyBlockStatementsContext) SEMICOLON(i int) antlr.TerminalNode {
	return s.GetToken(DevcmdParserSEMICOLON, i)
}

func (s *NonEmptyBlockStatementsContext) AllNEWLINE() []antlr.TerminalNode {
	return s.GetTokens(DevcmdParserNEWLINE)
}

func (s *NonEmptyBlockStatementsContext) NEWLINE(i int) antlr.TerminalNode {
	return s.GetToken(DevcmdParserNEWLINE, i)
}

func (s *NonEmptyBlockStatementsContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *NonEmptyBlockStatementsContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *NonEmptyBlockStatementsContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.EnterNonEmptyBlockStatements(s)
	}
}

func (s *NonEmptyBlockStatementsContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.ExitNonEmptyBlockStatements(s)
	}
}

func (s *NonEmptyBlockStatementsContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case DevcmdParserVisitor:
		return t.VisitNonEmptyBlockStatements(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *DevcmdParser) NonEmptyBlockStatements() (localctx INonEmptyBlockStatementsContext) {
	localctx = NewNonEmptyBlockStatementsContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 38, DevcmdParserRULE_nonEmptyBlockStatements)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(160)
		p.BlockStatement()
	}
	p.SetState(171)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 14, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(161)
				p.Match(DevcmdParserSEMICOLON)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}
			p.SetState(165)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 13, p.GetParserRuleContext())
			if p.HasError() {
				goto errorExit
			}
			for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
				if _alt == 1 {
					{
						p.SetState(162)
						p.Match(DevcmdParserNEWLINE)
						if p.HasError() {
							// Recognition error - abort rule
							goto errorExit
						}
					}

				}
				p.SetState(167)
				p.GetErrorHandler().Sync(p)
				if p.HasError() {
					goto errorExit
				}
				_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 13, p.GetParserRuleContext())
				if p.HasError() {
					goto errorExit
				}
			}
			{
				p.SetState(168)
				p.BlockStatement()
			}

		}
		p.SetState(173)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 14, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
	}
	p.SetState(175)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == DevcmdParserSEMICOLON {
		{
			p.SetState(174)
			p.Match(DevcmdParserSEMICOLON)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	}
	p.SetState(180)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == DevcmdParserNEWLINE {
		{
			p.SetState(177)
			p.Match(DevcmdParserNEWLINE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

		p.SetState(182)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IBlockStatementContext is an interface to support dynamic dispatch.
type IBlockStatementContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	DecoratedCommand() IDecoratedCommandContext
	CommandText() ICommandTextContext
	AllContinuationLine() []IContinuationLineContext
	ContinuationLine(i int) IContinuationLineContext

	// IsBlockStatementContext differentiates from other interfaces.
	IsBlockStatementContext()
}

type BlockStatementContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyBlockStatementContext() *BlockStatementContext {
	var p = new(BlockStatementContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_blockStatement
	return p
}

func InitEmptyBlockStatementContext(p *BlockStatementContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_blockStatement
}

func (*BlockStatementContext) IsBlockStatementContext() {}

func NewBlockStatementContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *BlockStatementContext {
	var p = new(BlockStatementContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = DevcmdParserRULE_blockStatement

	return p
}

func (s *BlockStatementContext) GetParser() antlr.Parser { return s.parser }

func (s *BlockStatementContext) DecoratedCommand() IDecoratedCommandContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IDecoratedCommandContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IDecoratedCommandContext)
}

func (s *BlockStatementContext) CommandText() ICommandTextContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ICommandTextContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ICommandTextContext)
}

func (s *BlockStatementContext) AllContinuationLine() []IContinuationLineContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IContinuationLineContext); ok {
			len++
		}
	}

	tst := make([]IContinuationLineContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IContinuationLineContext); ok {
			tst[i] = t.(IContinuationLineContext)
			i++
		}
	}

	return tst
}

func (s *BlockStatementContext) ContinuationLine(i int) IContinuationLineContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IContinuationLineContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IContinuationLineContext)
}

func (s *BlockStatementContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *BlockStatementContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *BlockStatementContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.EnterBlockStatement(s)
	}
}

func (s *BlockStatementContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.ExitBlockStatement(s)
	}
}

func (s *BlockStatementContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case DevcmdParserVisitor:
		return t.VisitBlockStatement(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *DevcmdParser) BlockStatement() (localctx IBlockStatementContext) {
	localctx = NewBlockStatementContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 40, DevcmdParserRULE_blockStatement)
	var _la int

	p.SetState(191)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 18, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(183)
			p.DecoratedCommand()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(184)
			p.CommandText()
		}
		p.SetState(188)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == DevcmdParserBACKSLASH {
			{
				p.SetState(185)
				p.ContinuationLine()
			}

			p.SetState(190)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

	case antlr.ATNInvalidAltNumber:
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IContinuationLineContext is an interface to support dynamic dispatch.
type IContinuationLineContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	BACKSLASH() antlr.TerminalNode
	NEWLINE() antlr.TerminalNode
	CommandText() ICommandTextContext

	// IsContinuationLineContext differentiates from other interfaces.
	IsContinuationLineContext()
}

type ContinuationLineContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyContinuationLineContext() *ContinuationLineContext {
	var p = new(ContinuationLineContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_continuationLine
	return p
}

func InitEmptyContinuationLineContext(p *ContinuationLineContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_continuationLine
}

func (*ContinuationLineContext) IsContinuationLineContext() {}

func NewContinuationLineContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ContinuationLineContext {
	var p = new(ContinuationLineContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = DevcmdParserRULE_continuationLine

	return p
}

func (s *ContinuationLineContext) GetParser() antlr.Parser { return s.parser }

func (s *ContinuationLineContext) BACKSLASH() antlr.TerminalNode {
	return s.GetToken(DevcmdParserBACKSLASH, 0)
}

func (s *ContinuationLineContext) NEWLINE() antlr.TerminalNode {
	return s.GetToken(DevcmdParserNEWLINE, 0)
}

func (s *ContinuationLineContext) CommandText() ICommandTextContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ICommandTextContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ICommandTextContext)
}

func (s *ContinuationLineContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ContinuationLineContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ContinuationLineContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.EnterContinuationLine(s)
	}
}

func (s *ContinuationLineContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.ExitContinuationLine(s)
	}
}

func (s *ContinuationLineContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case DevcmdParserVisitor:
		return t.VisitContinuationLine(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *DevcmdParser) ContinuationLine() (localctx IContinuationLineContext) {
	localctx = NewContinuationLineContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 42, DevcmdParserRULE_continuationLine)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(193)
		p.Match(DevcmdParserBACKSLASH)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(194)
		p.Match(DevcmdParserNEWLINE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(195)
		p.CommandText()
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ICommandTextContext is an interface to support dynamic dispatch.
type ICommandTextContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllCommandTextElement() []ICommandTextElementContext
	CommandTextElement(i int) ICommandTextElementContext

	// IsCommandTextContext differentiates from other interfaces.
	IsCommandTextContext()
}

type CommandTextContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyCommandTextContext() *CommandTextContext {
	var p = new(CommandTextContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_commandText
	return p
}

func InitEmptyCommandTextContext(p *CommandTextContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_commandText
}

func (*CommandTextContext) IsCommandTextContext() {}

func NewCommandTextContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *CommandTextContext {
	var p = new(CommandTextContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = DevcmdParserRULE_commandText

	return p
}

func (s *CommandTextContext) GetParser() antlr.Parser { return s.parser }

func (s *CommandTextContext) AllCommandTextElement() []ICommandTextElementContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(ICommandTextElementContext); ok {
			len++
		}
	}

	tst := make([]ICommandTextElementContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(ICommandTextElementContext); ok {
			tst[i] = t.(ICommandTextElementContext)
			i++
		}
	}

	return tst
}

func (s *CommandTextContext) CommandTextElement(i int) ICommandTextElementContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ICommandTextElementContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(ICommandTextElementContext)
}

func (s *CommandTextContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *CommandTextContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *CommandTextContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.EnterCommandText(s)
	}
}

func (s *CommandTextContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.ExitCommandText(s)
	}
}

func (s *CommandTextContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case DevcmdParserVisitor:
		return t.VisitCommandText(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *DevcmdParser) CommandText() (localctx ICommandTextContext) {
	localctx = NewCommandTextContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 44, DevcmdParserRULE_commandText)
	var _alt int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(200)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 19, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(197)
				p.CommandTextElement()
			}

		}
		p.SetState(202)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 19, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ICommandTextElementContext is an interface to support dynamic dispatch.
type ICommandTextElementContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	InlineDecorator() IInlineDecoratorContext
	NAME() antlr.TerminalNode
	NUMBER() antlr.TerminalNode
	STRING() antlr.TerminalNode
	SINGLE_STRING() antlr.TerminalNode
	PATH_CONTENT() antlr.TerminalNode
	LPAREN() antlr.TerminalNode
	RPAREN() antlr.TerminalNode
	LBRACE() antlr.TerminalNode
	RBRACE() antlr.TerminalNode
	LBRACKET() antlr.TerminalNode
	RBRACKET() antlr.TerminalNode
	AMPERSAND() antlr.TerminalNode
	PIPE() antlr.TerminalNode
	LT() antlr.TerminalNode
	GT() antlr.TerminalNode
	COLON() antlr.TerminalNode
	EQUALS() antlr.TerminalNode
	BACKSLASH() antlr.TerminalNode
	DOT() antlr.TerminalNode
	COMMA() antlr.TerminalNode
	SLASH() antlr.TerminalNode
	DASH() antlr.TerminalNode
	STAR() antlr.TerminalNode
	PLUS() antlr.TerminalNode
	QUESTION() antlr.TerminalNode
	EXCLAIM() antlr.TerminalNode
	PERCENT() antlr.TerminalNode
	CARET() antlr.TerminalNode
	TILDE() antlr.TerminalNode
	UNDERSCORE() antlr.TerminalNode
	DOLLAR() antlr.TerminalNode
	HASH() antlr.TerminalNode
	DOUBLEQUOTE() antlr.TerminalNode
	AT() antlr.TerminalNode
	WATCH() antlr.TerminalNode
	STOP() antlr.TerminalNode
	DEF() antlr.TerminalNode
	CONTENT() antlr.TerminalNode

	// IsCommandTextElementContext differentiates from other interfaces.
	IsCommandTextElementContext()
}

type CommandTextElementContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyCommandTextElementContext() *CommandTextElementContext {
	var p = new(CommandTextElementContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_commandTextElement
	return p
}

func InitEmptyCommandTextElementContext(p *CommandTextElementContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_commandTextElement
}

func (*CommandTextElementContext) IsCommandTextElementContext() {}

func NewCommandTextElementContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *CommandTextElementContext {
	var p = new(CommandTextElementContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = DevcmdParserRULE_commandTextElement

	return p
}

func (s *CommandTextElementContext) GetParser() antlr.Parser { return s.parser }

func (s *CommandTextElementContext) InlineDecorator() IInlineDecoratorContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IInlineDecoratorContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IInlineDecoratorContext)
}

func (s *CommandTextElementContext) NAME() antlr.TerminalNode {
	return s.GetToken(DevcmdParserNAME, 0)
}

func (s *CommandTextElementContext) NUMBER() antlr.TerminalNode {
	return s.GetToken(DevcmdParserNUMBER, 0)
}

func (s *CommandTextElementContext) STRING() antlr.TerminalNode {
	return s.GetToken(DevcmdParserSTRING, 0)
}

func (s *CommandTextElementContext) SINGLE_STRING() antlr.TerminalNode {
	return s.GetToken(DevcmdParserSINGLE_STRING, 0)
}

func (s *CommandTextElementContext) PATH_CONTENT() antlr.TerminalNode {
	return s.GetToken(DevcmdParserPATH_CONTENT, 0)
}

func (s *CommandTextElementContext) LPAREN() antlr.TerminalNode {
	return s.GetToken(DevcmdParserLPAREN, 0)
}

func (s *CommandTextElementContext) RPAREN() antlr.TerminalNode {
	return s.GetToken(DevcmdParserRPAREN, 0)
}

func (s *CommandTextElementContext) LBRACE() antlr.TerminalNode {
	return s.GetToken(DevcmdParserLBRACE, 0)
}

func (s *CommandTextElementContext) RBRACE() antlr.TerminalNode {
	return s.GetToken(DevcmdParserRBRACE, 0)
}

func (s *CommandTextElementContext) LBRACKET() antlr.TerminalNode {
	return s.GetToken(DevcmdParserLBRACKET, 0)
}

func (s *CommandTextElementContext) RBRACKET() antlr.TerminalNode {
	return s.GetToken(DevcmdParserRBRACKET, 0)
}

func (s *CommandTextElementContext) AMPERSAND() antlr.TerminalNode {
	return s.GetToken(DevcmdParserAMPERSAND, 0)
}

func (s *CommandTextElementContext) PIPE() antlr.TerminalNode {
	return s.GetToken(DevcmdParserPIPE, 0)
}

func (s *CommandTextElementContext) LT() antlr.TerminalNode {
	return s.GetToken(DevcmdParserLT, 0)
}

func (s *CommandTextElementContext) GT() antlr.TerminalNode {
	return s.GetToken(DevcmdParserGT, 0)
}

func (s *CommandTextElementContext) COLON() antlr.TerminalNode {
	return s.GetToken(DevcmdParserCOLON, 0)
}

func (s *CommandTextElementContext) EQUALS() antlr.TerminalNode {
	return s.GetToken(DevcmdParserEQUALS, 0)
}

func (s *CommandTextElementContext) BACKSLASH() antlr.TerminalNode {
	return s.GetToken(DevcmdParserBACKSLASH, 0)
}

func (s *CommandTextElementContext) DOT() antlr.TerminalNode {
	return s.GetToken(DevcmdParserDOT, 0)
}

func (s *CommandTextElementContext) COMMA() antlr.TerminalNode {
	return s.GetToken(DevcmdParserCOMMA, 0)
}

func (s *CommandTextElementContext) SLASH() antlr.TerminalNode {
	return s.GetToken(DevcmdParserSLASH, 0)
}

func (s *CommandTextElementContext) DASH() antlr.TerminalNode {
	return s.GetToken(DevcmdParserDASH, 0)
}

func (s *CommandTextElementContext) STAR() antlr.TerminalNode {
	return s.GetToken(DevcmdParserSTAR, 0)
}

func (s *CommandTextElementContext) PLUS() antlr.TerminalNode {
	return s.GetToken(DevcmdParserPLUS, 0)
}

func (s *CommandTextElementContext) QUESTION() antlr.TerminalNode {
	return s.GetToken(DevcmdParserQUESTION, 0)
}

func (s *CommandTextElementContext) EXCLAIM() antlr.TerminalNode {
	return s.GetToken(DevcmdParserEXCLAIM, 0)
}

func (s *CommandTextElementContext) PERCENT() antlr.TerminalNode {
	return s.GetToken(DevcmdParserPERCENT, 0)
}

func (s *CommandTextElementContext) CARET() antlr.TerminalNode {
	return s.GetToken(DevcmdParserCARET, 0)
}

func (s *CommandTextElementContext) TILDE() antlr.TerminalNode {
	return s.GetToken(DevcmdParserTILDE, 0)
}

func (s *CommandTextElementContext) UNDERSCORE() antlr.TerminalNode {
	return s.GetToken(DevcmdParserUNDERSCORE, 0)
}

func (s *CommandTextElementContext) DOLLAR() antlr.TerminalNode {
	return s.GetToken(DevcmdParserDOLLAR, 0)
}

func (s *CommandTextElementContext) HASH() antlr.TerminalNode {
	return s.GetToken(DevcmdParserHASH, 0)
}

func (s *CommandTextElementContext) DOUBLEQUOTE() antlr.TerminalNode {
	return s.GetToken(DevcmdParserDOUBLEQUOTE, 0)
}

func (s *CommandTextElementContext) AT() antlr.TerminalNode {
	return s.GetToken(DevcmdParserAT, 0)
}

func (s *CommandTextElementContext) WATCH() antlr.TerminalNode {
	return s.GetToken(DevcmdParserWATCH, 0)
}

func (s *CommandTextElementContext) STOP() antlr.TerminalNode {
	return s.GetToken(DevcmdParserSTOP, 0)
}

func (s *CommandTextElementContext) DEF() antlr.TerminalNode {
	return s.GetToken(DevcmdParserDEF, 0)
}

func (s *CommandTextElementContext) CONTENT() antlr.TerminalNode {
	return s.GetToken(DevcmdParserCONTENT, 0)
}

func (s *CommandTextElementContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *CommandTextElementContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *CommandTextElementContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.EnterCommandTextElement(s)
	}
}

func (s *CommandTextElementContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.ExitCommandTextElement(s)
	}
}

func (s *CommandTextElementContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case DevcmdParserVisitor:
		return t.VisitCommandTextElement(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *DevcmdParser) CommandTextElement() (localctx ICommandTextElementContext) {
	localctx = NewCommandTextElementContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 46, DevcmdParserRULE_commandTextElement)
	p.SetState(242)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 20, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(203)
			p.InlineDecorator()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(204)
			p.Match(DevcmdParserNAME)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(205)
			p.Match(DevcmdParserNUMBER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 4:
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(206)
			p.Match(DevcmdParserSTRING)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 5:
		p.EnterOuterAlt(localctx, 5)
		{
			p.SetState(207)
			p.Match(DevcmdParserSINGLE_STRING)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 6:
		p.EnterOuterAlt(localctx, 6)
		{
			p.SetState(208)
			p.Match(DevcmdParserPATH_CONTENT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 7:
		p.EnterOuterAlt(localctx, 7)
		{
			p.SetState(209)
			p.Match(DevcmdParserLPAREN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 8:
		p.EnterOuterAlt(localctx, 8)
		{
			p.SetState(210)
			p.Match(DevcmdParserRPAREN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 9:
		p.EnterOuterAlt(localctx, 9)
		{
			p.SetState(211)
			p.Match(DevcmdParserLBRACE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 10:
		p.EnterOuterAlt(localctx, 10)
		{
			p.SetState(212)
			p.Match(DevcmdParserRBRACE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 11:
		p.EnterOuterAlt(localctx, 11)
		{
			p.SetState(213)
			p.Match(DevcmdParserLBRACKET)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 12:
		p.EnterOuterAlt(localctx, 12)
		{
			p.SetState(214)
			p.Match(DevcmdParserRBRACKET)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 13:
		p.EnterOuterAlt(localctx, 13)
		{
			p.SetState(215)
			p.Match(DevcmdParserAMPERSAND)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 14:
		p.EnterOuterAlt(localctx, 14)
		{
			p.SetState(216)
			p.Match(DevcmdParserPIPE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 15:
		p.EnterOuterAlt(localctx, 15)
		{
			p.SetState(217)
			p.Match(DevcmdParserLT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 16:
		p.EnterOuterAlt(localctx, 16)
		{
			p.SetState(218)
			p.Match(DevcmdParserGT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 17:
		p.EnterOuterAlt(localctx, 17)
		{
			p.SetState(219)
			p.Match(DevcmdParserCOLON)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 18:
		p.EnterOuterAlt(localctx, 18)
		{
			p.SetState(220)
			p.Match(DevcmdParserEQUALS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 19:
		p.EnterOuterAlt(localctx, 19)
		{
			p.SetState(221)
			p.Match(DevcmdParserBACKSLASH)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 20:
		p.EnterOuterAlt(localctx, 20)
		{
			p.SetState(222)
			p.Match(DevcmdParserDOT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 21:
		p.EnterOuterAlt(localctx, 21)
		{
			p.SetState(223)
			p.Match(DevcmdParserCOMMA)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 22:
		p.EnterOuterAlt(localctx, 22)
		{
			p.SetState(224)
			p.Match(DevcmdParserSLASH)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 23:
		p.EnterOuterAlt(localctx, 23)
		{
			p.SetState(225)
			p.Match(DevcmdParserDASH)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 24:
		p.EnterOuterAlt(localctx, 24)
		{
			p.SetState(226)
			p.Match(DevcmdParserSTAR)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 25:
		p.EnterOuterAlt(localctx, 25)
		{
			p.SetState(227)
			p.Match(DevcmdParserPLUS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 26:
		p.EnterOuterAlt(localctx, 26)
		{
			p.SetState(228)
			p.Match(DevcmdParserQUESTION)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 27:
		p.EnterOuterAlt(localctx, 27)
		{
			p.SetState(229)
			p.Match(DevcmdParserEXCLAIM)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 28:
		p.EnterOuterAlt(localctx, 28)
		{
			p.SetState(230)
			p.Match(DevcmdParserPERCENT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 29:
		p.EnterOuterAlt(localctx, 29)
		{
			p.SetState(231)
			p.Match(DevcmdParserCARET)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 30:
		p.EnterOuterAlt(localctx, 30)
		{
			p.SetState(232)
			p.Match(DevcmdParserTILDE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 31:
		p.EnterOuterAlt(localctx, 31)
		{
			p.SetState(233)
			p.Match(DevcmdParserUNDERSCORE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 32:
		p.EnterOuterAlt(localctx, 32)
		{
			p.SetState(234)
			p.Match(DevcmdParserDOLLAR)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 33:
		p.EnterOuterAlt(localctx, 33)
		{
			p.SetState(235)
			p.Match(DevcmdParserHASH)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 34:
		p.EnterOuterAlt(localctx, 34)
		{
			p.SetState(236)
			p.Match(DevcmdParserDOUBLEQUOTE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 35:
		p.EnterOuterAlt(localctx, 35)
		{
			p.SetState(237)
			p.Match(DevcmdParserAT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 36:
		p.EnterOuterAlt(localctx, 36)
		{
			p.SetState(238)
			p.Match(DevcmdParserWATCH)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 37:
		p.EnterOuterAlt(localctx, 37)
		{
			p.SetState(239)
			p.Match(DevcmdParserSTOP)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 38:
		p.EnterOuterAlt(localctx, 38)
		{
			p.SetState(240)
			p.Match(DevcmdParserDEF)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 39:
		p.EnterOuterAlt(localctx, 39)
		{
			p.SetState(241)
			p.Match(DevcmdParserCONTENT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case antlr.ATNInvalidAltNumber:
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IInlineDecoratorContext is an interface to support dynamic dispatch.
type IInlineDecoratorContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AT() antlr.TerminalNode
	NAME() antlr.TerminalNode
	LPAREN() antlr.TerminalNode
	DecoratorContent() IDecoratorContentContext
	RPAREN() antlr.TerminalNode

	// IsInlineDecoratorContext differentiates from other interfaces.
	IsInlineDecoratorContext()
}

type InlineDecoratorContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyInlineDecoratorContext() *InlineDecoratorContext {
	var p = new(InlineDecoratorContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_inlineDecorator
	return p
}

func InitEmptyInlineDecoratorContext(p *InlineDecoratorContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_inlineDecorator
}

func (*InlineDecoratorContext) IsInlineDecoratorContext() {}

func NewInlineDecoratorContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *InlineDecoratorContext {
	var p = new(InlineDecoratorContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = DevcmdParserRULE_inlineDecorator

	return p
}

func (s *InlineDecoratorContext) GetParser() antlr.Parser { return s.parser }

func (s *InlineDecoratorContext) AT() antlr.TerminalNode {
	return s.GetToken(DevcmdParserAT, 0)
}

func (s *InlineDecoratorContext) NAME() antlr.TerminalNode {
	return s.GetToken(DevcmdParserNAME, 0)
}

func (s *InlineDecoratorContext) LPAREN() antlr.TerminalNode {
	return s.GetToken(DevcmdParserLPAREN, 0)
}

func (s *InlineDecoratorContext) DecoratorContent() IDecoratorContentContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IDecoratorContentContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IDecoratorContentContext)
}

func (s *InlineDecoratorContext) RPAREN() antlr.TerminalNode {
	return s.GetToken(DevcmdParserRPAREN, 0)
}

func (s *InlineDecoratorContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *InlineDecoratorContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *InlineDecoratorContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.EnterInlineDecorator(s)
	}
}

func (s *InlineDecoratorContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.ExitInlineDecorator(s)
	}
}

func (s *InlineDecoratorContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case DevcmdParserVisitor:
		return t.VisitInlineDecorator(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *DevcmdParser) InlineDecorator() (localctx IInlineDecoratorContext) {
	localctx = NewInlineDecoratorContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 48, DevcmdParserRULE_inlineDecorator)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(244)
		p.Match(DevcmdParserAT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(245)
		p.Match(DevcmdParserNAME)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(246)
		p.Match(DevcmdParserLPAREN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(247)
		p.DecoratorContent()
	}
	{
		p.SetState(248)
		p.Match(DevcmdParserRPAREN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}
