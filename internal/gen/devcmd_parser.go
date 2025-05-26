// Code generated from devcmd.g4 by ANTLR 4.13.2. DO NOT EDIT.

package gen // devcmd
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

type devcmdParser struct {
	*antlr.BaseParser
}

var DevcmdParserStaticData struct {
	once                   sync.Once
	serializedATN          []int32
	LiteralNames           []string
	SymbolicNames          []string
	RuleNames              []string
	PredictionContextCache *antlr.PredictionContextCache
	atn                    *antlr.ATN
	decisionToDFA          []*antlr.DFA
}

func devcmdParserInit() {
	staticData := &DevcmdParserStaticData
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
		"program", "line", "variableDefinition", "commandDefinition", "simpleCommand",
		"blockCommand", "blockStatements", "nonEmptyBlockStatements", "blockStatement",
		"continuationLine", "commandText",
	}
	staticData.PredictionContextCache = antlr.NewPredictionContextCache()
	staticData.serializedATN = []int32{
		4, 1, 21, 117, 2, 0, 7, 0, 2, 1, 7, 1, 2, 2, 7, 2, 2, 3, 7, 3, 2, 4, 7,
		4, 2, 5, 7, 5, 2, 6, 7, 6, 2, 7, 7, 7, 2, 8, 7, 8, 2, 9, 7, 9, 2, 10, 7,
		10, 1, 0, 5, 0, 24, 8, 0, 10, 0, 12, 0, 27, 9, 0, 1, 0, 1, 0, 1, 1, 1,
		1, 1, 1, 3, 1, 34, 8, 1, 1, 2, 1, 2, 1, 2, 1, 2, 3, 2, 40, 8, 2, 3, 2,
		42, 8, 2, 1, 2, 1, 2, 1, 2, 1, 3, 3, 3, 48, 8, 3, 1, 3, 1, 3, 1, 3, 1,
		3, 3, 3, 54, 8, 3, 1, 4, 1, 4, 5, 4, 58, 8, 4, 10, 4, 12, 4, 61, 9, 4,
		3, 4, 63, 8, 4, 1, 4, 1, 4, 1, 4, 1, 5, 1, 5, 3, 5, 70, 8, 5, 1, 5, 1,
		5, 1, 5, 3, 5, 75, 8, 5, 1, 6, 1, 6, 3, 6, 79, 8, 6, 1, 7, 1, 7, 1, 7,
		5, 7, 84, 8, 7, 10, 7, 12, 7, 87, 9, 7, 1, 7, 5, 7, 90, 8, 7, 10, 7, 12,
		7, 93, 9, 7, 1, 7, 3, 7, 96, 8, 7, 1, 7, 5, 7, 99, 8, 7, 10, 7, 12, 7,
		102, 9, 7, 1, 8, 1, 8, 3, 8, 106, 8, 8, 1, 9, 1, 9, 1, 9, 1, 9, 1, 10,
		4, 10, 113, 8, 10, 11, 10, 12, 10, 114, 1, 10, 0, 0, 11, 0, 2, 4, 6, 8,
		10, 12, 14, 16, 18, 20, 0, 3, 1, 1, 20, 20, 1, 0, 4, 5, 2, 0, 2, 10, 14,
		18, 123, 0, 25, 1, 0, 0, 0, 2, 33, 1, 0, 0, 0, 4, 35, 1, 0, 0, 0, 6, 47,
		1, 0, 0, 0, 8, 62, 1, 0, 0, 0, 10, 67, 1, 0, 0, 0, 12, 78, 1, 0, 0, 0,
		14, 80, 1, 0, 0, 0, 16, 103, 1, 0, 0, 0, 18, 107, 1, 0, 0, 0, 20, 112,
		1, 0, 0, 0, 22, 24, 3, 2, 1, 0, 23, 22, 1, 0, 0, 0, 24, 27, 1, 0, 0, 0,
		25, 23, 1, 0, 0, 0, 25, 26, 1, 0, 0, 0, 26, 28, 1, 0, 0, 0, 27, 25, 1,
		0, 0, 0, 28, 29, 5, 0, 0, 1, 29, 1, 1, 0, 0, 0, 30, 34, 3, 4, 2, 0, 31,
		34, 3, 6, 3, 0, 32, 34, 5, 20, 0, 0, 33, 30, 1, 0, 0, 0, 33, 31, 1, 0,
		0, 0, 33, 32, 1, 0, 0, 0, 34, 3, 1, 0, 0, 0, 35, 36, 5, 1, 0, 0, 36, 41,
		5, 16, 0, 0, 37, 39, 5, 2, 0, 0, 38, 40, 3, 20, 10, 0, 39, 38, 1, 0, 0,
		0, 39, 40, 1, 0, 0, 0, 40, 42, 1, 0, 0, 0, 41, 37, 1, 0, 0, 0, 41, 42,
		1, 0, 0, 0, 42, 43, 1, 0, 0, 0, 43, 44, 5, 13, 0, 0, 44, 45, 7, 0, 0, 0,
		45, 5, 1, 0, 0, 0, 46, 48, 7, 1, 0, 0, 47, 46, 1, 0, 0, 0, 47, 48, 1, 0,
		0, 0, 48, 49, 1, 0, 0, 0, 49, 50, 5, 16, 0, 0, 50, 53, 5, 3, 0, 0, 51,
		54, 3, 8, 4, 0, 52, 54, 3, 10, 5, 0, 53, 51, 1, 0, 0, 0, 53, 52, 1, 0,
		0, 0, 54, 7, 1, 0, 0, 0, 55, 59, 3, 20, 10, 0, 56, 58, 3, 18, 9, 0, 57,
		56, 1, 0, 0, 0, 58, 61, 1, 0, 0, 0, 59, 57, 1, 0, 0, 0, 59, 60, 1, 0, 0,
		0, 60, 63, 1, 0, 0, 0, 61, 59, 1, 0, 0, 0, 62, 55, 1, 0, 0, 0, 62, 63,
		1, 0, 0, 0, 63, 64, 1, 0, 0, 0, 64, 65, 5, 13, 0, 0, 65, 66, 7, 0, 0, 0,
		66, 9, 1, 0, 0, 0, 67, 69, 5, 11, 0, 0, 68, 70, 5, 20, 0, 0, 69, 68, 1,
		0, 0, 0, 69, 70, 1, 0, 0, 0, 70, 71, 1, 0, 0, 0, 71, 72, 3, 12, 6, 0, 72,
		74, 5, 12, 0, 0, 73, 75, 7, 0, 0, 0, 74, 73, 1, 0, 0, 0, 74, 75, 1, 0,
		0, 0, 75, 11, 1, 0, 0, 0, 76, 79, 1, 0, 0, 0, 77, 79, 3, 14, 7, 0, 78,
		76, 1, 0, 0, 0, 78, 77, 1, 0, 0, 0, 79, 13, 1, 0, 0, 0, 80, 91, 3, 16,
		8, 0, 81, 85, 5, 13, 0, 0, 82, 84, 5, 20, 0, 0, 83, 82, 1, 0, 0, 0, 84,
		87, 1, 0, 0, 0, 85, 83, 1, 0, 0, 0, 85, 86, 1, 0, 0, 0, 86, 88, 1, 0, 0,
		0, 87, 85, 1, 0, 0, 0, 88, 90, 3, 16, 8, 0, 89, 81, 1, 0, 0, 0, 90, 93,
		1, 0, 0, 0, 91, 89, 1, 0, 0, 0, 91, 92, 1, 0, 0, 0, 92, 95, 1, 0, 0, 0,
		93, 91, 1, 0, 0, 0, 94, 96, 5, 13, 0, 0, 95, 94, 1, 0, 0, 0, 95, 96, 1,
		0, 0, 0, 96, 100, 1, 0, 0, 0, 97, 99, 5, 20, 0, 0, 98, 97, 1, 0, 0, 0,
		99, 102, 1, 0, 0, 0, 100, 98, 1, 0, 0, 0, 100, 101, 1, 0, 0, 0, 101, 15,
		1, 0, 0, 0, 102, 100, 1, 0, 0, 0, 103, 105, 3, 20, 10, 0, 104, 106, 5,
		14, 0, 0, 105, 104, 1, 0, 0, 0, 105, 106, 1, 0, 0, 0, 106, 17, 1, 0, 0,
		0, 107, 108, 5, 15, 0, 0, 108, 109, 5, 20, 0, 0, 109, 110, 3, 20, 10, 0,
		110, 19, 1, 0, 0, 0, 111, 113, 7, 2, 0, 0, 112, 111, 1, 0, 0, 0, 113, 114,
		1, 0, 0, 0, 114, 112, 1, 0, 0, 0, 114, 115, 1, 0, 0, 0, 115, 21, 1, 0,
		0, 0, 17, 25, 33, 39, 41, 47, 53, 59, 62, 69, 74, 78, 85, 91, 95, 100,
		105, 114,
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

// devcmdParserInit initializes any static state used to implement devcmdParser. By default the
// static state used to implement the parser is lazily initialized during the first call to
// NewdevcmdParser(). You can call this function if you wish to initialize the static state ahead
// of time.
func DevcmdParserInit() {
	staticData := &DevcmdParserStaticData
	staticData.once.Do(devcmdParserInit)
}

// NewdevcmdParser produces a new parser instance for the optional input antlr.TokenStream.
func NewdevcmdParser(input antlr.TokenStream) *devcmdParser {
	DevcmdParserInit()
	this := new(devcmdParser)
	this.BaseParser = antlr.NewBaseParser(input)
	staticData := &DevcmdParserStaticData
	this.Interpreter = antlr.NewParserATNSimulator(this, staticData.atn, staticData.decisionToDFA, staticData.PredictionContextCache)
	this.RuleNames = staticData.RuleNames
	this.LiteralNames = staticData.LiteralNames
	this.SymbolicNames = staticData.SymbolicNames
	this.GrammarFileName = "devcmd.g4"

	return this
}

// devcmdParser tokens.
const (
	devcmdParserEOF                      = antlr.TokenEOF
	devcmdParserDEF                      = 1
	devcmdParserEQUALS                   = 2
	devcmdParserCOLON                    = 3
	devcmdParserWATCH                    = 4
	devcmdParserSTOP                     = 5
	devcmdParserOUR_VARIABLE_REFERENCE   = 6
	devcmdParserSHELL_VARIABLE_REFERENCE = 7
	devcmdParserESCAPED_SEMICOLON        = 8
	devcmdParserESCAPED_DOLLAR           = 9
	devcmdParserESCAPED_CHAR             = 10
	devcmdParserLBRACE                   = 11
	devcmdParserRBRACE                   = 12
	devcmdParserSEMICOLON                = 13
	devcmdParserAMPERSAND                = 14
	devcmdParserBACKSLASH                = 15
	devcmdParserNAME                     = 16
	devcmdParserNUMBER                   = 17
	devcmdParserCOMMAND_TEXT             = 18
	devcmdParserCOMMENT                  = 19
	devcmdParserNEWLINE                  = 20
	devcmdParserWS                       = 21
)

// devcmdParser rules.
const (
	devcmdParserRULE_program                 = 0
	devcmdParserRULE_line                    = 1
	devcmdParserRULE_variableDefinition      = 2
	devcmdParserRULE_commandDefinition       = 3
	devcmdParserRULE_simpleCommand           = 4
	devcmdParserRULE_blockCommand            = 5
	devcmdParserRULE_blockStatements         = 6
	devcmdParserRULE_nonEmptyBlockStatements = 7
	devcmdParserRULE_blockStatement          = 8
	devcmdParserRULE_continuationLine        = 9
	devcmdParserRULE_commandText             = 10
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
	p.RuleIndex = devcmdParserRULE_program
	return p
}

func InitEmptyProgramContext(p *ProgramContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = devcmdParserRULE_program
}

func (*ProgramContext) IsProgramContext() {}

func NewProgramContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ProgramContext {
	var p = new(ProgramContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = devcmdParserRULE_program

	return p
}

func (s *ProgramContext) GetParser() antlr.Parser { return s.parser }

func (s *ProgramContext) EOF() antlr.TerminalNode {
	return s.GetToken(devcmdParserEOF, 0)
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
	if listenerT, ok := listener.(devcmdListener); ok {
		listenerT.EnterProgram(s)
	}
}

func (s *ProgramContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(devcmdListener); ok {
		listenerT.ExitProgram(s)
	}
}

func (p *devcmdParser) Program() (localctx IProgramContext) {
	localctx = NewProgramContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 0, devcmdParserRULE_program)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(25)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&1114162) != 0 {
		{
			p.SetState(22)
			p.Line()
		}

		p.SetState(27)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(28)
		p.Match(devcmdParserEOF)
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
	p.RuleIndex = devcmdParserRULE_line
	return p
}

func InitEmptyLineContext(p *LineContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = devcmdParserRULE_line
}

func (*LineContext) IsLineContext() {}

func NewLineContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *LineContext {
	var p = new(LineContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = devcmdParserRULE_line

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
	return s.GetToken(devcmdParserNEWLINE, 0)
}

func (s *LineContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *LineContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *LineContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(devcmdListener); ok {
		listenerT.EnterLine(s)
	}
}

func (s *LineContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(devcmdListener); ok {
		listenerT.ExitLine(s)
	}
}

func (p *devcmdParser) Line() (localctx ILineContext) {
	localctx = NewLineContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 2, devcmdParserRULE_line)
	p.SetState(33)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case devcmdParserDEF:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(30)
			p.VariableDefinition()
		}

	case devcmdParserWATCH, devcmdParserSTOP, devcmdParserNAME:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(31)
			p.CommandDefinition()
		}

	case devcmdParserNEWLINE:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(32)
			p.Match(devcmdParserNEWLINE)
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
	SEMICOLON() antlr.TerminalNode
	NEWLINE() antlr.TerminalNode
	EOF() antlr.TerminalNode
	EQUALS() antlr.TerminalNode
	CommandText() ICommandTextContext

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
	p.RuleIndex = devcmdParserRULE_variableDefinition
	return p
}

func InitEmptyVariableDefinitionContext(p *VariableDefinitionContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = devcmdParserRULE_variableDefinition
}

func (*VariableDefinitionContext) IsVariableDefinitionContext() {}

func NewVariableDefinitionContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *VariableDefinitionContext {
	var p = new(VariableDefinitionContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = devcmdParserRULE_variableDefinition

	return p
}

func (s *VariableDefinitionContext) GetParser() antlr.Parser { return s.parser }

func (s *VariableDefinitionContext) DEF() antlr.TerminalNode {
	return s.GetToken(devcmdParserDEF, 0)
}

func (s *VariableDefinitionContext) NAME() antlr.TerminalNode {
	return s.GetToken(devcmdParserNAME, 0)
}

func (s *VariableDefinitionContext) SEMICOLON() antlr.TerminalNode {
	return s.GetToken(devcmdParserSEMICOLON, 0)
}

func (s *VariableDefinitionContext) NEWLINE() antlr.TerminalNode {
	return s.GetToken(devcmdParserNEWLINE, 0)
}

func (s *VariableDefinitionContext) EOF() antlr.TerminalNode {
	return s.GetToken(devcmdParserEOF, 0)
}

func (s *VariableDefinitionContext) EQUALS() antlr.TerminalNode {
	return s.GetToken(devcmdParserEQUALS, 0)
}

func (s *VariableDefinitionContext) CommandText() ICommandTextContext {
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

func (s *VariableDefinitionContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *VariableDefinitionContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *VariableDefinitionContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(devcmdListener); ok {
		listenerT.EnterVariableDefinition(s)
	}
}

func (s *VariableDefinitionContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(devcmdListener); ok {
		listenerT.ExitVariableDefinition(s)
	}
}

func (p *devcmdParser) VariableDefinition() (localctx IVariableDefinitionContext) {
	localctx = NewVariableDefinitionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 4, devcmdParserRULE_variableDefinition)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(35)
		p.Match(devcmdParserDEF)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(36)
		p.Match(devcmdParserNAME)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(41)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == devcmdParserEQUALS {
		{
			p.SetState(37)
			p.Match(devcmdParserEQUALS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		p.SetState(39)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&509948) != 0 {
			{
				p.SetState(38)
				p.CommandText()
			}

		}

	}
	{
		p.SetState(43)
		p.Match(devcmdParserSEMICOLON)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(44)
		_la = p.GetTokenStream().LA(1)

		if !(_la == devcmdParserEOF || _la == devcmdParserNEWLINE) {
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

// ICommandDefinitionContext is an interface to support dynamic dispatch.
type ICommandDefinitionContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	NAME() antlr.TerminalNode
	COLON() antlr.TerminalNode
	SimpleCommand() ISimpleCommandContext
	BlockCommand() IBlockCommandContext
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
	p.RuleIndex = devcmdParserRULE_commandDefinition
	return p
}

func InitEmptyCommandDefinitionContext(p *CommandDefinitionContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = devcmdParserRULE_commandDefinition
}

func (*CommandDefinitionContext) IsCommandDefinitionContext() {}

func NewCommandDefinitionContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *CommandDefinitionContext {
	var p = new(CommandDefinitionContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = devcmdParserRULE_commandDefinition

	return p
}

func (s *CommandDefinitionContext) GetParser() antlr.Parser { return s.parser }

func (s *CommandDefinitionContext) NAME() antlr.TerminalNode {
	return s.GetToken(devcmdParserNAME, 0)
}

func (s *CommandDefinitionContext) COLON() antlr.TerminalNode {
	return s.GetToken(devcmdParserCOLON, 0)
}

func (s *CommandDefinitionContext) SimpleCommand() ISimpleCommandContext {
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

func (s *CommandDefinitionContext) BlockCommand() IBlockCommandContext {
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

func (s *CommandDefinitionContext) WATCH() antlr.TerminalNode {
	return s.GetToken(devcmdParserWATCH, 0)
}

func (s *CommandDefinitionContext) STOP() antlr.TerminalNode {
	return s.GetToken(devcmdParserSTOP, 0)
}

func (s *CommandDefinitionContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *CommandDefinitionContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *CommandDefinitionContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(devcmdListener); ok {
		listenerT.EnterCommandDefinition(s)
	}
}

func (s *CommandDefinitionContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(devcmdListener); ok {
		listenerT.ExitCommandDefinition(s)
	}
}

func (p *devcmdParser) CommandDefinition() (localctx ICommandDefinitionContext) {
	localctx = NewCommandDefinitionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 6, devcmdParserRULE_commandDefinition)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(47)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == devcmdParserWATCH || _la == devcmdParserSTOP {
		{
			p.SetState(46)
			_la = p.GetTokenStream().LA(1)

			if !(_la == devcmdParserWATCH || _la == devcmdParserSTOP) {
				p.GetErrorHandler().RecoverInline(p)
			} else {
				p.GetErrorHandler().ReportMatch(p)
				p.Consume()
			}
		}

	}
	{
		p.SetState(49)
		p.Match(devcmdParserNAME)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(50)
		p.Match(devcmdParserCOLON)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(53)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case devcmdParserEQUALS, devcmdParserCOLON, devcmdParserWATCH, devcmdParserSTOP, devcmdParserOUR_VARIABLE_REFERENCE, devcmdParserSHELL_VARIABLE_REFERENCE, devcmdParserESCAPED_SEMICOLON, devcmdParserESCAPED_DOLLAR, devcmdParserESCAPED_CHAR, devcmdParserSEMICOLON, devcmdParserAMPERSAND, devcmdParserBACKSLASH, devcmdParserNAME, devcmdParserNUMBER, devcmdParserCOMMAND_TEXT:
		{
			p.SetState(51)
			p.SimpleCommand()
		}

	case devcmdParserLBRACE:
		{
			p.SetState(52)
			p.BlockCommand()
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

// ISimpleCommandContext is an interface to support dynamic dispatch.
type ISimpleCommandContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	SEMICOLON() antlr.TerminalNode
	NEWLINE() antlr.TerminalNode
	EOF() antlr.TerminalNode
	CommandText() ICommandTextContext
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
	p.RuleIndex = devcmdParserRULE_simpleCommand
	return p
}

func InitEmptySimpleCommandContext(p *SimpleCommandContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = devcmdParserRULE_simpleCommand
}

func (*SimpleCommandContext) IsSimpleCommandContext() {}

func NewSimpleCommandContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *SimpleCommandContext {
	var p = new(SimpleCommandContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = devcmdParserRULE_simpleCommand

	return p
}

func (s *SimpleCommandContext) GetParser() antlr.Parser { return s.parser }

func (s *SimpleCommandContext) SEMICOLON() antlr.TerminalNode {
	return s.GetToken(devcmdParserSEMICOLON, 0)
}

func (s *SimpleCommandContext) NEWLINE() antlr.TerminalNode {
	return s.GetToken(devcmdParserNEWLINE, 0)
}

func (s *SimpleCommandContext) EOF() antlr.TerminalNode {
	return s.GetToken(devcmdParserEOF, 0)
}

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
	if listenerT, ok := listener.(devcmdListener); ok {
		listenerT.EnterSimpleCommand(s)
	}
}

func (s *SimpleCommandContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(devcmdListener); ok {
		listenerT.ExitSimpleCommand(s)
	}
}

func (p *devcmdParser) SimpleCommand() (localctx ISimpleCommandContext) {
	localctx = NewSimpleCommandContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 8, devcmdParserRULE_simpleCommand)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(62)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&509948) != 0 {
		{
			p.SetState(55)
			p.CommandText()
		}
		p.SetState(59)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == devcmdParserBACKSLASH {
			{
				p.SetState(56)
				p.ContinuationLine()
			}

			p.SetState(61)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

	}
	{
		p.SetState(64)
		p.Match(devcmdParserSEMICOLON)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(65)
		_la = p.GetTokenStream().LA(1)

		if !(_la == devcmdParserEOF || _la == devcmdParserNEWLINE) {
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

// IBlockCommandContext is an interface to support dynamic dispatch.
type IBlockCommandContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	LBRACE() antlr.TerminalNode
	BlockStatements() IBlockStatementsContext
	RBRACE() antlr.TerminalNode
	AllNEWLINE() []antlr.TerminalNode
	NEWLINE(i int) antlr.TerminalNode
	EOF() antlr.TerminalNode

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
	p.RuleIndex = devcmdParserRULE_blockCommand
	return p
}

func InitEmptyBlockCommandContext(p *BlockCommandContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = devcmdParserRULE_blockCommand
}

func (*BlockCommandContext) IsBlockCommandContext() {}

func NewBlockCommandContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *BlockCommandContext {
	var p = new(BlockCommandContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = devcmdParserRULE_blockCommand

	return p
}

func (s *BlockCommandContext) GetParser() antlr.Parser { return s.parser }

func (s *BlockCommandContext) LBRACE() antlr.TerminalNode {
	return s.GetToken(devcmdParserLBRACE, 0)
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
	return s.GetToken(devcmdParserRBRACE, 0)
}

func (s *BlockCommandContext) AllNEWLINE() []antlr.TerminalNode {
	return s.GetTokens(devcmdParserNEWLINE)
}

func (s *BlockCommandContext) NEWLINE(i int) antlr.TerminalNode {
	return s.GetToken(devcmdParserNEWLINE, i)
}

func (s *BlockCommandContext) EOF() antlr.TerminalNode {
	return s.GetToken(devcmdParserEOF, 0)
}

func (s *BlockCommandContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *BlockCommandContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *BlockCommandContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(devcmdListener); ok {
		listenerT.EnterBlockCommand(s)
	}
}

func (s *BlockCommandContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(devcmdListener); ok {
		listenerT.ExitBlockCommand(s)
	}
}

func (p *devcmdParser) BlockCommand() (localctx IBlockCommandContext) {
	localctx = NewBlockCommandContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 10, devcmdParserRULE_blockCommand)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(67)
		p.Match(devcmdParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(69)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == devcmdParserNEWLINE {
		{
			p.SetState(68)
			p.Match(devcmdParserNEWLINE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	}
	{
		p.SetState(71)
		p.BlockStatements()
	}
	{
		p.SetState(72)
		p.Match(devcmdParserRBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(74)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 9, p.GetParserRuleContext()) == 1 {
		{
			p.SetState(73)
			_la = p.GetTokenStream().LA(1)

			if !(_la == devcmdParserEOF || _la == devcmdParserNEWLINE) {
				p.GetErrorHandler().RecoverInline(p)
			} else {
				p.GetErrorHandler().ReportMatch(p)
				p.Consume()
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
	p.RuleIndex = devcmdParserRULE_blockStatements
	return p
}

func InitEmptyBlockStatementsContext(p *BlockStatementsContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = devcmdParserRULE_blockStatements
}

func (*BlockStatementsContext) IsBlockStatementsContext() {}

func NewBlockStatementsContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *BlockStatementsContext {
	var p = new(BlockStatementsContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = devcmdParserRULE_blockStatements

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
	if listenerT, ok := listener.(devcmdListener); ok {
		listenerT.EnterBlockStatements(s)
	}
}

func (s *BlockStatementsContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(devcmdListener); ok {
		listenerT.ExitBlockStatements(s)
	}
}

func (p *devcmdParser) BlockStatements() (localctx IBlockStatementsContext) {
	localctx = NewBlockStatementsContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 12, devcmdParserRULE_blockStatements)
	p.SetState(78)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case devcmdParserRBRACE:
		p.EnterOuterAlt(localctx, 1)

	case devcmdParserEQUALS, devcmdParserCOLON, devcmdParserWATCH, devcmdParserSTOP, devcmdParserOUR_VARIABLE_REFERENCE, devcmdParserSHELL_VARIABLE_REFERENCE, devcmdParserESCAPED_SEMICOLON, devcmdParserESCAPED_DOLLAR, devcmdParserESCAPED_CHAR, devcmdParserAMPERSAND, devcmdParserBACKSLASH, devcmdParserNAME, devcmdParserNUMBER, devcmdParserCOMMAND_TEXT:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(77)
			p.NonEmptyBlockStatements()
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
	p.RuleIndex = devcmdParserRULE_nonEmptyBlockStatements
	return p
}

func InitEmptyNonEmptyBlockStatementsContext(p *NonEmptyBlockStatementsContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = devcmdParserRULE_nonEmptyBlockStatements
}

func (*NonEmptyBlockStatementsContext) IsNonEmptyBlockStatementsContext() {}

func NewNonEmptyBlockStatementsContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *NonEmptyBlockStatementsContext {
	var p = new(NonEmptyBlockStatementsContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = devcmdParserRULE_nonEmptyBlockStatements

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
	return s.GetTokens(devcmdParserSEMICOLON)
}

func (s *NonEmptyBlockStatementsContext) SEMICOLON(i int) antlr.TerminalNode {
	return s.GetToken(devcmdParserSEMICOLON, i)
}

func (s *NonEmptyBlockStatementsContext) AllNEWLINE() []antlr.TerminalNode {
	return s.GetTokens(devcmdParserNEWLINE)
}

func (s *NonEmptyBlockStatementsContext) NEWLINE(i int) antlr.TerminalNode {
	return s.GetToken(devcmdParserNEWLINE, i)
}

func (s *NonEmptyBlockStatementsContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *NonEmptyBlockStatementsContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *NonEmptyBlockStatementsContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(devcmdListener); ok {
		listenerT.EnterNonEmptyBlockStatements(s)
	}
}

func (s *NonEmptyBlockStatementsContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(devcmdListener); ok {
		listenerT.ExitNonEmptyBlockStatements(s)
	}
}

func (p *devcmdParser) NonEmptyBlockStatements() (localctx INonEmptyBlockStatementsContext) {
	localctx = NewNonEmptyBlockStatementsContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 14, devcmdParserRULE_nonEmptyBlockStatements)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(80)
		p.BlockStatement()
	}
	p.SetState(91)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 12, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(81)
				p.Match(devcmdParserSEMICOLON)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}
			p.SetState(85)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)

			for _la == devcmdParserNEWLINE {
				{
					p.SetState(82)
					p.Match(devcmdParserNEWLINE)
					if p.HasError() {
						// Recognition error - abort rule
						goto errorExit
					}
				}

				p.SetState(87)
				p.GetErrorHandler().Sync(p)
				if p.HasError() {
					goto errorExit
				}
				_la = p.GetTokenStream().LA(1)
			}
			{
				p.SetState(88)
				p.BlockStatement()
			}

		}
		p.SetState(93)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 12, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
	}
	p.SetState(95)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == devcmdParserSEMICOLON {
		{
			p.SetState(94)
			p.Match(devcmdParserSEMICOLON)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	}
	p.SetState(100)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == devcmdParserNEWLINE {
		{
			p.SetState(97)
			p.Match(devcmdParserNEWLINE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

		p.SetState(102)
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
	CommandText() ICommandTextContext
	AMPERSAND() antlr.TerminalNode

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
	p.RuleIndex = devcmdParserRULE_blockStatement
	return p
}

func InitEmptyBlockStatementContext(p *BlockStatementContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = devcmdParserRULE_blockStatement
}

func (*BlockStatementContext) IsBlockStatementContext() {}

func NewBlockStatementContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *BlockStatementContext {
	var p = new(BlockStatementContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = devcmdParserRULE_blockStatement

	return p
}

func (s *BlockStatementContext) GetParser() antlr.Parser { return s.parser }

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

func (s *BlockStatementContext) AMPERSAND() antlr.TerminalNode {
	return s.GetToken(devcmdParserAMPERSAND, 0)
}

func (s *BlockStatementContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *BlockStatementContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *BlockStatementContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(devcmdListener); ok {
		listenerT.EnterBlockStatement(s)
	}
}

func (s *BlockStatementContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(devcmdListener); ok {
		listenerT.ExitBlockStatement(s)
	}
}

func (p *devcmdParser) BlockStatement() (localctx IBlockStatementContext) {
	localctx = NewBlockStatementContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 16, devcmdParserRULE_blockStatement)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(103)
		p.CommandText()
	}
	p.SetState(105)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == devcmdParserAMPERSAND {
		{
			p.SetState(104)
			p.Match(devcmdParserAMPERSAND)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
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
	p.RuleIndex = devcmdParserRULE_continuationLine
	return p
}

func InitEmptyContinuationLineContext(p *ContinuationLineContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = devcmdParserRULE_continuationLine
}

func (*ContinuationLineContext) IsContinuationLineContext() {}

func NewContinuationLineContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ContinuationLineContext {
	var p = new(ContinuationLineContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = devcmdParserRULE_continuationLine

	return p
}

func (s *ContinuationLineContext) GetParser() antlr.Parser { return s.parser }

func (s *ContinuationLineContext) BACKSLASH() antlr.TerminalNode {
	return s.GetToken(devcmdParserBACKSLASH, 0)
}

func (s *ContinuationLineContext) NEWLINE() antlr.TerminalNode {
	return s.GetToken(devcmdParserNEWLINE, 0)
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
	if listenerT, ok := listener.(devcmdListener); ok {
		listenerT.EnterContinuationLine(s)
	}
}

func (s *ContinuationLineContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(devcmdListener); ok {
		listenerT.ExitContinuationLine(s)
	}
}

func (p *devcmdParser) ContinuationLine() (localctx IContinuationLineContext) {
	localctx = NewContinuationLineContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 18, devcmdParserRULE_continuationLine)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(107)
		p.Match(devcmdParserBACKSLASH)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(108)
		p.Match(devcmdParserNEWLINE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(109)
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
	AllESCAPED_CHAR() []antlr.TerminalNode
	ESCAPED_CHAR(i int) antlr.TerminalNode
	AllOUR_VARIABLE_REFERENCE() []antlr.TerminalNode
	OUR_VARIABLE_REFERENCE(i int) antlr.TerminalNode
	AllSHELL_VARIABLE_REFERENCE() []antlr.TerminalNode
	SHELL_VARIABLE_REFERENCE(i int) antlr.TerminalNode
	AllESCAPED_SEMICOLON() []antlr.TerminalNode
	ESCAPED_SEMICOLON(i int) antlr.TerminalNode
	AllESCAPED_DOLLAR() []antlr.TerminalNode
	ESCAPED_DOLLAR(i int) antlr.TerminalNode
	AllAMPERSAND() []antlr.TerminalNode
	AMPERSAND(i int) antlr.TerminalNode
	AllCOLON() []antlr.TerminalNode
	COLON(i int) antlr.TerminalNode
	AllEQUALS() []antlr.TerminalNode
	EQUALS(i int) antlr.TerminalNode
	AllNUMBER() []antlr.TerminalNode
	NUMBER(i int) antlr.TerminalNode
	AllSTOP() []antlr.TerminalNode
	STOP(i int) antlr.TerminalNode
	AllWATCH() []antlr.TerminalNode
	WATCH(i int) antlr.TerminalNode
	AllNAME() []antlr.TerminalNode
	NAME(i int) antlr.TerminalNode
	AllBACKSLASH() []antlr.TerminalNode
	BACKSLASH(i int) antlr.TerminalNode
	AllCOMMAND_TEXT() []antlr.TerminalNode
	COMMAND_TEXT(i int) antlr.TerminalNode

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
	p.RuleIndex = devcmdParserRULE_commandText
	return p
}

func InitEmptyCommandTextContext(p *CommandTextContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = devcmdParserRULE_commandText
}

func (*CommandTextContext) IsCommandTextContext() {}

func NewCommandTextContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *CommandTextContext {
	var p = new(CommandTextContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = devcmdParserRULE_commandText

	return p
}

func (s *CommandTextContext) GetParser() antlr.Parser { return s.parser }

func (s *CommandTextContext) AllESCAPED_CHAR() []antlr.TerminalNode {
	return s.GetTokens(devcmdParserESCAPED_CHAR)
}

func (s *CommandTextContext) ESCAPED_CHAR(i int) antlr.TerminalNode {
	return s.GetToken(devcmdParserESCAPED_CHAR, i)
}

func (s *CommandTextContext) AllOUR_VARIABLE_REFERENCE() []antlr.TerminalNode {
	return s.GetTokens(devcmdParserOUR_VARIABLE_REFERENCE)
}

func (s *CommandTextContext) OUR_VARIABLE_REFERENCE(i int) antlr.TerminalNode {
	return s.GetToken(devcmdParserOUR_VARIABLE_REFERENCE, i)
}

func (s *CommandTextContext) AllSHELL_VARIABLE_REFERENCE() []antlr.TerminalNode {
	return s.GetTokens(devcmdParserSHELL_VARIABLE_REFERENCE)
}

func (s *CommandTextContext) SHELL_VARIABLE_REFERENCE(i int) antlr.TerminalNode {
	return s.GetToken(devcmdParserSHELL_VARIABLE_REFERENCE, i)
}

func (s *CommandTextContext) AllESCAPED_SEMICOLON() []antlr.TerminalNode {
	return s.GetTokens(devcmdParserESCAPED_SEMICOLON)
}

func (s *CommandTextContext) ESCAPED_SEMICOLON(i int) antlr.TerminalNode {
	return s.GetToken(devcmdParserESCAPED_SEMICOLON, i)
}

func (s *CommandTextContext) AllESCAPED_DOLLAR() []antlr.TerminalNode {
	return s.GetTokens(devcmdParserESCAPED_DOLLAR)
}

func (s *CommandTextContext) ESCAPED_DOLLAR(i int) antlr.TerminalNode {
	return s.GetToken(devcmdParserESCAPED_DOLLAR, i)
}

func (s *CommandTextContext) AllAMPERSAND() []antlr.TerminalNode {
	return s.GetTokens(devcmdParserAMPERSAND)
}

func (s *CommandTextContext) AMPERSAND(i int) antlr.TerminalNode {
	return s.GetToken(devcmdParserAMPERSAND, i)
}

func (s *CommandTextContext) AllCOLON() []antlr.TerminalNode {
	return s.GetTokens(devcmdParserCOLON)
}

func (s *CommandTextContext) COLON(i int) antlr.TerminalNode {
	return s.GetToken(devcmdParserCOLON, i)
}

func (s *CommandTextContext) AllEQUALS() []antlr.TerminalNode {
	return s.GetTokens(devcmdParserEQUALS)
}

func (s *CommandTextContext) EQUALS(i int) antlr.TerminalNode {
	return s.GetToken(devcmdParserEQUALS, i)
}

func (s *CommandTextContext) AllNUMBER() []antlr.TerminalNode {
	return s.GetTokens(devcmdParserNUMBER)
}

func (s *CommandTextContext) NUMBER(i int) antlr.TerminalNode {
	return s.GetToken(devcmdParserNUMBER, i)
}

func (s *CommandTextContext) AllSTOP() []antlr.TerminalNode {
	return s.GetTokens(devcmdParserSTOP)
}

func (s *CommandTextContext) STOP(i int) antlr.TerminalNode {
	return s.GetToken(devcmdParserSTOP, i)
}

func (s *CommandTextContext) AllWATCH() []antlr.TerminalNode {
	return s.GetTokens(devcmdParserWATCH)
}

func (s *CommandTextContext) WATCH(i int) antlr.TerminalNode {
	return s.GetToken(devcmdParserWATCH, i)
}

func (s *CommandTextContext) AllNAME() []antlr.TerminalNode {
	return s.GetTokens(devcmdParserNAME)
}

func (s *CommandTextContext) NAME(i int) antlr.TerminalNode {
	return s.GetToken(devcmdParserNAME, i)
}

func (s *CommandTextContext) AllBACKSLASH() []antlr.TerminalNode {
	return s.GetTokens(devcmdParserBACKSLASH)
}

func (s *CommandTextContext) BACKSLASH(i int) antlr.TerminalNode {
	return s.GetToken(devcmdParserBACKSLASH, i)
}

func (s *CommandTextContext) AllCOMMAND_TEXT() []antlr.TerminalNode {
	return s.GetTokens(devcmdParserCOMMAND_TEXT)
}

func (s *CommandTextContext) COMMAND_TEXT(i int) antlr.TerminalNode {
	return s.GetToken(devcmdParserCOMMAND_TEXT, i)
}

func (s *CommandTextContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *CommandTextContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *CommandTextContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(devcmdListener); ok {
		listenerT.EnterCommandText(s)
	}
}

func (s *CommandTextContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(devcmdListener); ok {
		listenerT.ExitCommandText(s)
	}
}

func (p *devcmdParser) CommandText() (localctx ICommandTextContext) {
	localctx = NewCommandTextContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 20, devcmdParserRULE_commandText)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(112)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = 1
	for ok := true; ok; ok = _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		switch _alt {
		case 1:
			{
				p.SetState(111)
				_la = p.GetTokenStream().LA(1)

				if !((int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&509948) != 0) {
					p.GetErrorHandler().RecoverInline(p)
				} else {
					p.GetErrorHandler().ReportMatch(p)
					p.Consume()
				}
			}

		default:
			p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
			goto errorExit
		}

		p.SetState(114)
		p.GetErrorHandler().Sync(p)
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 16, p.GetParserRuleContext())
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
