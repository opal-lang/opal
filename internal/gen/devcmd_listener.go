// Code generated from devcmd.g4 by ANTLR 4.13.2. DO NOT EDIT.

package gen // devcmd
import "github.com/antlr4-go/antlr/v4"

// devcmdListener is a complete listener for a parse tree produced by devcmdParser.
type devcmdListener interface {
	antlr.ParseTreeListener

	// EnterProgram is called when entering the program production.
	EnterProgram(c *ProgramContext)

	// EnterLine is called when entering the line production.
	EnterLine(c *LineContext)

	// EnterVariableDefinition is called when entering the variableDefinition production.
	EnterVariableDefinition(c *VariableDefinitionContext)

	// EnterCommandDefinition is called when entering the commandDefinition production.
	EnterCommandDefinition(c *CommandDefinitionContext)

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

	// ExitProgram is called when exiting the program production.
	ExitProgram(c *ProgramContext)

	// ExitLine is called when exiting the line production.
	ExitLine(c *LineContext)

	// ExitVariableDefinition is called when exiting the variableDefinition production.
	ExitVariableDefinition(c *VariableDefinitionContext)

	// ExitCommandDefinition is called when exiting the commandDefinition production.
	ExitCommandDefinition(c *CommandDefinitionContext)

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
}
