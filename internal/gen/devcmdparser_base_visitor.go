// Code generated from DevcmdParser.g4 by ANTLR 4.13.2. DO NOT EDIT.

package gen // DevcmdParser
import "github.com/antlr4-go/antlr/v4"

type BaseDevcmdParserVisitor struct {
	*antlr.BaseParseTreeVisitor
}

func (v *BaseDevcmdParserVisitor) VisitProgram(ctx *ProgramContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseDevcmdParserVisitor) VisitLine(ctx *LineContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseDevcmdParserVisitor) VisitVariableDefinition(ctx *VariableDefinitionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseDevcmdParserVisitor) VisitVariableValue(ctx *VariableValueContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseDevcmdParserVisitor) VisitCommandDefinition(ctx *CommandDefinitionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseDevcmdParserVisitor) VisitCommandBody(ctx *CommandBodyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseDevcmdParserVisitor) VisitFunctionDecorator(ctx *FunctionDecoratorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseDevcmdParserVisitor) VisitBlockDecorator(ctx *BlockDecoratorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseDevcmdParserVisitor) VisitSimpleDecorator(ctx *SimpleDecoratorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseDevcmdParserVisitor) VisitDecorator(ctx *DecoratorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseDevcmdParserVisitor) VisitDecoratorContent(ctx *DecoratorContentContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseDevcmdParserVisitor) VisitDecoratorElement(ctx *DecoratorElementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseDevcmdParserVisitor) VisitSimpleCommand(ctx *SimpleCommandContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseDevcmdParserVisitor) VisitDecoratorCommand(ctx *DecoratorCommandContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseDevcmdParserVisitor) VisitBlockCommand(ctx *BlockCommandContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseDevcmdParserVisitor) VisitBlockStatements(ctx *BlockStatementsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseDevcmdParserVisitor) VisitNonEmptyBlockStatements(ctx *NonEmptyBlockStatementsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseDevcmdParserVisitor) VisitBlockStatement(ctx *BlockStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseDevcmdParserVisitor) VisitContinuationLine(ctx *ContinuationLineContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseDevcmdParserVisitor) VisitCommandText(ctx *CommandTextContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseDevcmdParserVisitor) VisitCommandTextElement(ctx *CommandTextElementContext) interface{} {
	return v.VisitChildren(ctx)
}
