// Code generated from DevcmdParser.g4 by ANTLR 4.13.2. DO NOT EDIT.

package gen // DevcmdParser
import "github.com/antlr4-go/antlr/v4"

// BaseDevcmdParserListener is a complete listener for a parse tree produced by DevcmdParser.
type BaseDevcmdParserListener struct{}

var _ DevcmdParserListener = &BaseDevcmdParserListener{}

// VisitTerminal is called when a terminal node is visited.
func (s *BaseDevcmdParserListener) VisitTerminal(node antlr.TerminalNode) {}

// VisitErrorNode is called when an error node is visited.
func (s *BaseDevcmdParserListener) VisitErrorNode(node antlr.ErrorNode) {}

// EnterEveryRule is called when any rule is entered.
func (s *BaseDevcmdParserListener) EnterEveryRule(ctx antlr.ParserRuleContext) {}

// ExitEveryRule is called when any rule is exited.
func (s *BaseDevcmdParserListener) ExitEveryRule(ctx antlr.ParserRuleContext) {}

// EnterProgram is called when production program is entered.
func (s *BaseDevcmdParserListener) EnterProgram(ctx *ProgramContext) {}

// ExitProgram is called when production program is exited.
func (s *BaseDevcmdParserListener) ExitProgram(ctx *ProgramContext) {}

// EnterLine is called when production line is entered.
func (s *BaseDevcmdParserListener) EnterLine(ctx *LineContext) {}

// ExitLine is called when production line is exited.
func (s *BaseDevcmdParserListener) ExitLine(ctx *LineContext) {}

// EnterVariableDefinition is called when production variableDefinition is entered.
func (s *BaseDevcmdParserListener) EnterVariableDefinition(ctx *VariableDefinitionContext) {}

// ExitVariableDefinition is called when production variableDefinition is exited.
func (s *BaseDevcmdParserListener) ExitVariableDefinition(ctx *VariableDefinitionContext) {}

// EnterVariableValue is called when production variableValue is entered.
func (s *BaseDevcmdParserListener) EnterVariableValue(ctx *VariableValueContext) {}

// ExitVariableValue is called when production variableValue is exited.
func (s *BaseDevcmdParserListener) ExitVariableValue(ctx *VariableValueContext) {}

// EnterCommandDefinition is called when production commandDefinition is entered.
func (s *BaseDevcmdParserListener) EnterCommandDefinition(ctx *CommandDefinitionContext) {}

// ExitCommandDefinition is called when production commandDefinition is exited.
func (s *BaseDevcmdParserListener) ExitCommandDefinition(ctx *CommandDefinitionContext) {}

// EnterCommandBody is called when production commandBody is entered.
func (s *BaseDevcmdParserListener) EnterCommandBody(ctx *CommandBodyContext) {}

// ExitCommandBody is called when production commandBody is exited.
func (s *BaseDevcmdParserListener) ExitCommandBody(ctx *CommandBodyContext) {}

// EnterFunctionDecoratorLabel is called when production functionDecoratorLabel is entered.
func (s *BaseDevcmdParserListener) EnterFunctionDecoratorLabel(ctx *FunctionDecoratorLabelContext) {}

// ExitFunctionDecoratorLabel is called when production functionDecoratorLabel is exited.
func (s *BaseDevcmdParserListener) ExitFunctionDecoratorLabel(ctx *FunctionDecoratorLabelContext) {}

// EnterBlockDecoratorLabel is called when production blockDecoratorLabel is entered.
func (s *BaseDevcmdParserListener) EnterBlockDecoratorLabel(ctx *BlockDecoratorLabelContext) {}

// ExitBlockDecoratorLabel is called when production blockDecoratorLabel is exited.
func (s *BaseDevcmdParserListener) ExitBlockDecoratorLabel(ctx *BlockDecoratorLabelContext) {}

// EnterFunctionDecorator is called when production functionDecorator is entered.
func (s *BaseDevcmdParserListener) EnterFunctionDecorator(ctx *FunctionDecoratorContext) {}

// ExitFunctionDecorator is called when production functionDecorator is exited.
func (s *BaseDevcmdParserListener) ExitFunctionDecorator(ctx *FunctionDecoratorContext) {}

// EnterBlockDecorator is called when production blockDecorator is entered.
func (s *BaseDevcmdParserListener) EnterBlockDecorator(ctx *BlockDecoratorContext) {}

// ExitBlockDecorator is called when production blockDecorator is exited.
func (s *BaseDevcmdParserListener) ExitBlockDecorator(ctx *BlockDecoratorContext) {}

// EnterDecoratorContent is called when production decoratorContent is entered.
func (s *BaseDevcmdParserListener) EnterDecoratorContent(ctx *DecoratorContentContext) {}

// ExitDecoratorContent is called when production decoratorContent is exited.
func (s *BaseDevcmdParserListener) ExitDecoratorContent(ctx *DecoratorContentContext) {}

// EnterDecoratorElement is called when production decoratorElement is entered.
func (s *BaseDevcmdParserListener) EnterDecoratorElement(ctx *DecoratorElementContext) {}

// ExitDecoratorElement is called when production decoratorElement is exited.
func (s *BaseDevcmdParserListener) ExitDecoratorElement(ctx *DecoratorElementContext) {}

// EnterNestedDecorator is called when production nestedDecorator is entered.
func (s *BaseDevcmdParserListener) EnterNestedDecorator(ctx *NestedDecoratorContext) {}

// ExitNestedDecorator is called when production nestedDecorator is exited.
func (s *BaseDevcmdParserListener) ExitNestedDecorator(ctx *NestedDecoratorContext) {}

// EnterDecoratorTextElement is called when production decoratorTextElement is entered.
func (s *BaseDevcmdParserListener) EnterDecoratorTextElement(ctx *DecoratorTextElementContext) {}

// ExitDecoratorTextElement is called when production decoratorTextElement is exited.
func (s *BaseDevcmdParserListener) ExitDecoratorTextElement(ctx *DecoratorTextElementContext) {}

// EnterSimpleCommand is called when production simpleCommand is entered.
func (s *BaseDevcmdParserListener) EnterSimpleCommand(ctx *SimpleCommandContext) {}

// ExitSimpleCommand is called when production simpleCommand is exited.
func (s *BaseDevcmdParserListener) ExitSimpleCommand(ctx *SimpleCommandContext) {}

// EnterBlockCommand is called when production blockCommand is entered.
func (s *BaseDevcmdParserListener) EnterBlockCommand(ctx *BlockCommandContext) {}

// ExitBlockCommand is called when production blockCommand is exited.
func (s *BaseDevcmdParserListener) ExitBlockCommand(ctx *BlockCommandContext) {}

// EnterBlockStatements is called when production blockStatements is entered.
func (s *BaseDevcmdParserListener) EnterBlockStatements(ctx *BlockStatementsContext) {}

// ExitBlockStatements is called when production blockStatements is exited.
func (s *BaseDevcmdParserListener) ExitBlockStatements(ctx *BlockStatementsContext) {}

// EnterNonEmptyBlockStatements is called when production nonEmptyBlockStatements is entered.
func (s *BaseDevcmdParserListener) EnterNonEmptyBlockStatements(ctx *NonEmptyBlockStatementsContext) {
}

// ExitNonEmptyBlockStatements is called when production nonEmptyBlockStatements is exited.
func (s *BaseDevcmdParserListener) ExitNonEmptyBlockStatements(ctx *NonEmptyBlockStatementsContext) {}

// EnterBlockStatement is called when production blockStatement is entered.
func (s *BaseDevcmdParserListener) EnterBlockStatement(ctx *BlockStatementContext) {}

// ExitBlockStatement is called when production blockStatement is exited.
func (s *BaseDevcmdParserListener) ExitBlockStatement(ctx *BlockStatementContext) {}

// EnterContinuationLine is called when production continuationLine is entered.
func (s *BaseDevcmdParserListener) EnterContinuationLine(ctx *ContinuationLineContext) {}

// ExitContinuationLine is called when production continuationLine is exited.
func (s *BaseDevcmdParserListener) ExitContinuationLine(ctx *ContinuationLineContext) {}

// EnterCommandText is called when production commandText is entered.
func (s *BaseDevcmdParserListener) EnterCommandText(ctx *CommandTextContext) {}

// ExitCommandText is called when production commandText is exited.
func (s *BaseDevcmdParserListener) ExitCommandText(ctx *CommandTextContext) {}

// EnterCommandTextElement is called when production commandTextElement is entered.
func (s *BaseDevcmdParserListener) EnterCommandTextElement(ctx *CommandTextElementContext) {}

// ExitCommandTextElement is called when production commandTextElement is exited.
func (s *BaseDevcmdParserListener) ExitCommandTextElement(ctx *CommandTextElementContext) {}

// EnterInlineDecorator is called when production inlineDecorator is entered.
func (s *BaseDevcmdParserListener) EnterInlineDecorator(ctx *InlineDecoratorContext) {}

// ExitInlineDecorator is called when production inlineDecorator is exited.
func (s *BaseDevcmdParserListener) ExitInlineDecorator(ctx *InlineDecoratorContext) {}
