// Code generated from DevcmdParser.g4 by ANTLR 4.13.2. DO NOT EDIT.

package gen // DevcmdParser
import "github.com/antlr4-go/antlr/v4"

// A complete Visitor for a parse tree produced by DevcmdParser.
type DevcmdParserVisitor interface {
	antlr.ParseTreeVisitor

	// Visit a parse tree produced by DevcmdParser#program.
	VisitProgram(ctx *ProgramContext) interface{}

	// Visit a parse tree produced by DevcmdParser#line.
	VisitLine(ctx *LineContext) interface{}

	// Visit a parse tree produced by DevcmdParser#variableDefinition.
	VisitVariableDefinition(ctx *VariableDefinitionContext) interface{}

	// Visit a parse tree produced by DevcmdParser#variableValue.
	VisitVariableValue(ctx *VariableValueContext) interface{}

	// Visit a parse tree produced by DevcmdParser#commandDefinition.
	VisitCommandDefinition(ctx *CommandDefinitionContext) interface{}

	// Visit a parse tree produced by DevcmdParser#commandBody.
	VisitCommandBody(ctx *CommandBodyContext) interface{}

	// Visit a parse tree produced by DevcmdParser#functionDecorator.
	VisitFunctionDecorator(ctx *FunctionDecoratorContext) interface{}

	// Visit a parse tree produced by DevcmdParser#blockDecorator.
	VisitBlockDecorator(ctx *BlockDecoratorContext) interface{}

	// Visit a parse tree produced by DevcmdParser#simpleDecorator.
	VisitSimpleDecorator(ctx *SimpleDecoratorContext) interface{}

	// Visit a parse tree produced by DevcmdParser#decorator.
	VisitDecorator(ctx *DecoratorContext) interface{}

	// Visit a parse tree produced by DevcmdParser#decoratorContent.
	VisitDecoratorContent(ctx *DecoratorContentContext) interface{}

	// Visit a parse tree produced by DevcmdParser#decoratorElement.
	VisitDecoratorElement(ctx *DecoratorElementContext) interface{}

	// Visit a parse tree produced by DevcmdParser#simpleCommand.
	VisitSimpleCommand(ctx *SimpleCommandContext) interface{}

	// Visit a parse tree produced by DevcmdParser#decoratorCommand.
	VisitDecoratorCommand(ctx *DecoratorCommandContext) interface{}

	// Visit a parse tree produced by DevcmdParser#blockCommand.
	VisitBlockCommand(ctx *BlockCommandContext) interface{}

	// Visit a parse tree produced by DevcmdParser#blockStatements.
	VisitBlockStatements(ctx *BlockStatementsContext) interface{}

	// Visit a parse tree produced by DevcmdParser#nonEmptyBlockStatements.
	VisitNonEmptyBlockStatements(ctx *NonEmptyBlockStatementsContext) interface{}

	// Visit a parse tree produced by DevcmdParser#blockStatement.
	VisitBlockStatement(ctx *BlockStatementContext) interface{}

	// Visit a parse tree produced by DevcmdParser#continuationLine.
	VisitContinuationLine(ctx *ContinuationLineContext) interface{}

	// Visit a parse tree produced by DevcmdParser#commandText.
	VisitCommandText(ctx *CommandTextContext) interface{}

	// Visit a parse tree produced by DevcmdParser#commandTextElement.
	VisitCommandTextElement(ctx *CommandTextElementContext) interface{}
}
