// Code generated from DevcmdParser.g4 by ANTLR 4.13.2. DO NOT EDIT.

package gen // DevcmdParser
import "github.com/antlr4-go/antlr/v4"

// DevcmdParserListener is a complete listener for a parse tree produced by DevcmdParser.
type DevcmdParserListener interface {
	antlr.ParseTreeListener

	// EnterProgram is called when entering the program production.
	EnterProgram(c *ProgramContext)

	// EnterLine is called when entering the line production.
	EnterLine(c *LineContext)

	// EnterVariableDefinition is called when entering the variableDefinition production.
	EnterVariableDefinition(c *VariableDefinitionContext)

	// EnterVariableValue is called when entering the variableValue production.
	EnterVariableValue(c *VariableValueContext)

	// EnterCommandDefinition is called when entering the commandDefinition production.
	EnterCommandDefinition(c *CommandDefinitionContext)

	// EnterCommandBody is called when entering the commandBody production.
	EnterCommandBody(c *CommandBodyContext)

	// EnterFunctionDecoratorLabel is called when entering the functionDecoratorLabel production.
	EnterFunctionDecoratorLabel(c *FunctionDecoratorLabelContext)

	// EnterBlockDecoratorLabel is called when entering the blockDecoratorLabel production.
	EnterBlockDecoratorLabel(c *BlockDecoratorLabelContext)

	// EnterFunctionDecorator is called when entering the functionDecorator production.
	EnterFunctionDecorator(c *FunctionDecoratorContext)

	// EnterBlockDecorator is called when entering the blockDecorator production.
	EnterBlockDecorator(c *BlockDecoratorContext)

	// EnterDecoratorContent is called when entering the decoratorContent production.
	EnterDecoratorContent(c *DecoratorContentContext)

	// EnterDecoratorElement is called when entering the decoratorElement production.
	EnterDecoratorElement(c *DecoratorElementContext)

	// EnterNestedDecorator is called when entering the nestedDecorator production.
	EnterNestedDecorator(c *NestedDecoratorContext)

	// EnterDecoratorTextElement is called when entering the decoratorTextElement production.
	EnterDecoratorTextElement(c *DecoratorTextElementContext)

	// EnterSimpleCommand is called when entering the simpleCommand production.
	EnterSimpleCommand(c *SimpleCommandContext)

	// EnterBlockCommand is called when entering the blockCommand production.
	EnterBlockCommand(c *BlockCommandContext)

	// EnterBlockStatements is called when entering the blockStatements production.
	EnterBlockStatements(c *BlockStatementsContext)

	// EnterNonEmptyBlockStatements is called when entering the nonEmptyBlockStatements production.
	EnterNonEmptyBlockStatements(c *NonEmptyBlockStatementsContext)

	// EnterBlockStatement is called when entering the blockStatement production.
	EnterBlockStatement(c *BlockStatementContext)

	// EnterContinuationLine is called when entering the continuationLine production.
	EnterContinuationLine(c *ContinuationLineContext)

	// EnterCommandText is called when entering the commandText production.
	EnterCommandText(c *CommandTextContext)

	// EnterCommandTextElement is called when entering the commandTextElement production.
	EnterCommandTextElement(c *CommandTextElementContext)

	// EnterInlineDecorator is called when entering the inlineDecorator production.
	EnterInlineDecorator(c *InlineDecoratorContext)

	// ExitProgram is called when exiting the program production.
	ExitProgram(c *ProgramContext)

	// ExitLine is called when exiting the line production.
	ExitLine(c *LineContext)

	// ExitVariableDefinition is called when exiting the variableDefinition production.
	ExitVariableDefinition(c *VariableDefinitionContext)

	// ExitVariableValue is called when exiting the variableValue production.
	ExitVariableValue(c *VariableValueContext)

	// ExitCommandDefinition is called when exiting the commandDefinition production.
	ExitCommandDefinition(c *CommandDefinitionContext)

	// ExitCommandBody is called when exiting the commandBody production.
	ExitCommandBody(c *CommandBodyContext)

	// ExitFunctionDecoratorLabel is called when exiting the functionDecoratorLabel production.
	ExitFunctionDecoratorLabel(c *FunctionDecoratorLabelContext)

	// ExitBlockDecoratorLabel is called when exiting the blockDecoratorLabel production.
	ExitBlockDecoratorLabel(c *BlockDecoratorLabelContext)

	// ExitFunctionDecorator is called when exiting the functionDecorator production.
	ExitFunctionDecorator(c *FunctionDecoratorContext)

	// ExitBlockDecorator is called when exiting the blockDecorator production.
	ExitBlockDecorator(c *BlockDecoratorContext)

	// ExitDecoratorContent is called when exiting the decoratorContent production.
	ExitDecoratorContent(c *DecoratorContentContext)

	// ExitDecoratorElement is called when exiting the decoratorElement production.
	ExitDecoratorElement(c *DecoratorElementContext)

	// ExitNestedDecorator is called when exiting the nestedDecorator production.
	ExitNestedDecorator(c *NestedDecoratorContext)

	// ExitDecoratorTextElement is called when exiting the decoratorTextElement production.
	ExitDecoratorTextElement(c *DecoratorTextElementContext)

	// ExitSimpleCommand is called when exiting the simpleCommand production.
	ExitSimpleCommand(c *SimpleCommandContext)

	// ExitBlockCommand is called when exiting the blockCommand production.
	ExitBlockCommand(c *BlockCommandContext)

	// ExitBlockStatements is called when exiting the blockStatements production.
	ExitBlockStatements(c *BlockStatementsContext)

	// ExitNonEmptyBlockStatements is called when exiting the nonEmptyBlockStatements production.
	ExitNonEmptyBlockStatements(c *NonEmptyBlockStatementsContext)

	// ExitBlockStatement is called when exiting the blockStatement production.
	ExitBlockStatement(c *BlockStatementContext)

	// ExitContinuationLine is called when exiting the continuationLine production.
	ExitContinuationLine(c *ContinuationLineContext)

	// ExitCommandText is called when exiting the commandText production.
	ExitCommandText(c *CommandTextContext)

	// ExitCommandTextElement is called when exiting the commandTextElement production.
	ExitCommandTextElement(c *CommandTextElementContext)

	// ExitInlineDecorator is called when exiting the inlineDecorator production.
	ExitInlineDecorator(c *InlineDecoratorContext)
}
