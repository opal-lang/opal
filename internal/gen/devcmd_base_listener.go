// Code generated from devcmd.g4 by ANTLR 4.13.2. DO NOT EDIT.

package gen // devcmd
import "github.com/antlr4-go/antlr/v4"

// BasedevcmdListener is a complete listener for a parse tree produced by devcmdParser.
type BasedevcmdListener struct{}

var _ devcmdListener = &BasedevcmdListener{}

// VisitTerminal is called when a terminal node is visited.
func (s *BasedevcmdListener) VisitTerminal(node antlr.TerminalNode) {}

// VisitErrorNode is called when an error node is visited.
func (s *BasedevcmdListener) VisitErrorNode(node antlr.ErrorNode) {}

// EnterEveryRule is called when any rule is entered.
func (s *BasedevcmdListener) EnterEveryRule(ctx antlr.ParserRuleContext) {}

// ExitEveryRule is called when any rule is exited.
func (s *BasedevcmdListener) ExitEveryRule(ctx antlr.ParserRuleContext) {}

// EnterProgram is called when production program is entered.
func (s *BasedevcmdListener) EnterProgram(ctx *ProgramContext) {}

// ExitProgram is called when production program is exited.
func (s *BasedevcmdListener) ExitProgram(ctx *ProgramContext) {}

// EnterLine is called when production line is entered.
func (s *BasedevcmdListener) EnterLine(ctx *LineContext) {}

// ExitLine is called when production line is exited.
func (s *BasedevcmdListener) ExitLine(ctx *LineContext) {}

// EnterVariableDefinition is called when production variableDefinition is entered.
func (s *BasedevcmdListener) EnterVariableDefinition(ctx *VariableDefinitionContext) {}

// ExitVariableDefinition is called when production variableDefinition is exited.
func (s *BasedevcmdListener) ExitVariableDefinition(ctx *VariableDefinitionContext) {}

// EnterCommandDefinition is called when production commandDefinition is entered.
func (s *BasedevcmdListener) EnterCommandDefinition(ctx *CommandDefinitionContext) {}

// ExitCommandDefinition is called when production commandDefinition is exited.
func (s *BasedevcmdListener) ExitCommandDefinition(ctx *CommandDefinitionContext) {}

// EnterSimpleCommand is called when production simpleCommand is entered.
func (s *BasedevcmdListener) EnterSimpleCommand(ctx *SimpleCommandContext) {}

// ExitSimpleCommand is called when production simpleCommand is exited.
func (s *BasedevcmdListener) ExitSimpleCommand(ctx *SimpleCommandContext) {}

// EnterBlockCommand is called when production blockCommand is entered.
func (s *BasedevcmdListener) EnterBlockCommand(ctx *BlockCommandContext) {}

// ExitBlockCommand is called when production blockCommand is exited.
func (s *BasedevcmdListener) ExitBlockCommand(ctx *BlockCommandContext) {}

// EnterBlockStatements is called when production blockStatements is entered.
func (s *BasedevcmdListener) EnterBlockStatements(ctx *BlockStatementsContext) {}

// ExitBlockStatements is called when production blockStatements is exited.
func (s *BasedevcmdListener) ExitBlockStatements(ctx *BlockStatementsContext) {}

// EnterNonEmptyBlockStatements is called when production nonEmptyBlockStatements is entered.
func (s *BasedevcmdListener) EnterNonEmptyBlockStatements(ctx *NonEmptyBlockStatementsContext) {}

// ExitNonEmptyBlockStatements is called when production nonEmptyBlockStatements is exited.
func (s *BasedevcmdListener) ExitNonEmptyBlockStatements(ctx *NonEmptyBlockStatementsContext) {}

// EnterBlockStatement is called when production blockStatement is entered.
func (s *BasedevcmdListener) EnterBlockStatement(ctx *BlockStatementContext) {}

// ExitBlockStatement is called when production blockStatement is exited.
func (s *BasedevcmdListener) ExitBlockStatement(ctx *BlockStatementContext) {}

// EnterContinuationLine is called when production continuationLine is entered.
func (s *BasedevcmdListener) EnterContinuationLine(ctx *ContinuationLineContext) {}

// ExitContinuationLine is called when production continuationLine is exited.
func (s *BasedevcmdListener) ExitContinuationLine(ctx *ContinuationLineContext) {}

// EnterCommandText is called when production commandText is entered.
func (s *BasedevcmdListener) EnterCommandText(ctx *CommandTextContext) {}

// ExitCommandText is called when production commandText is exited.
func (s *BasedevcmdListener) ExitCommandText(ctx *CommandTextContext) {}
