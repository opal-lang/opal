package parser

import (
	"fmt"
	"strings"

	"github.com/aledsdavies/devcmd/internal/gen"
	"github.com/antlr4-go/antlr/v4"
)

//go:generate bash -c "cd ../../grammar && antlr -Dlanguage=Go -package gen -o ../internal/gen devcmd.g4"

// ParseError represents an error that occurred during parsing
type ParseError struct {
	Line    int    // The line number where the error occurred
	Column  int    // The column number where the error occurred
	Message string // The error message
	Context string // The line of text where the error occurred
}

// Error formats the parse error as a string with visual context
func (e *ParseError) Error() string {
	if e.Context == "" {
		return fmt.Sprintf("line %d: %s", e.Line, e.Message)
	}

	// Create a visual error indicator with arrow pointing to error position
	pointer := strings.Repeat(" ", e.Column) + "^"

	return fmt.Sprintf("line %d: %s\n%s\n%s",
		e.Line,
		e.Message,
		e.Context,
		pointer)
}

// NewParseError creates a new ParseError without context
func NewParseError(line int, format string, args ...interface{}) *ParseError {
	return &ParseError{
		Line:    line,
		Message: fmt.Sprintf(format, args...),
	}
}

// NewDetailedParseError creates a ParseError with context information
func NewDetailedParseError(line int, column int, context string, format string, args ...interface{}) *ParseError {
	return &ParseError{
		Line:    line,
		Column:  column,
		Context: context,
		Message: fmt.Sprintf(format, args...),
	}
}

// CommandRegistry manages command names and prevents conflicts
type CommandRegistry struct {
	regularCommands map[string]int // name -> line number
	watchCommands   map[string]int // name -> line number
	stopCommands    map[string]int // name -> line number
	lines           []string       // source lines for error reporting
}

// NewCommandRegistry creates a new command registry
func NewCommandRegistry(lines []string) *CommandRegistry {
	return &CommandRegistry{
		regularCommands: make(map[string]int),
		watchCommands:   make(map[string]int),
		stopCommands:    make(map[string]int),
		lines:           lines,
	}
}

// RegisterCommand registers a command and checks for conflicts
func (cr *CommandRegistry) RegisterCommand(cmd Command) error {
	name := cmd.Name
	line := cmd.Line

	// Get the line content for error reporting
	var lineContent string
	if line > 0 && line <= len(cr.lines) {
		lineContent = cr.lines[line-1]
	}

	// Find the column position of the command name
	namePos := strings.Index(lineContent, name)
	if namePos == -1 {
		namePos = 0
	}

	if cmd.IsWatch {
		// Check for duplicate watch command
		if existingLine, exists := cr.watchCommands[name]; exists {
			return NewDetailedParseError(line, namePos, lineContent,
				"duplicate watch command '%s' (previously defined at line %d)",
				name, existingLine)
		}

		// Check for conflict with regular command
		if existingLine, exists := cr.regularCommands[name]; exists {
			return NewDetailedParseError(line, namePos, lineContent,
				"watch command '%s' conflicts with regular command (defined at line %d)",
				name, existingLine)
		}

		cr.watchCommands[name] = line

	} else if cmd.IsStop {
		// Check for duplicate stop command
		if existingLine, exists := cr.stopCommands[name]; exists {
			return NewDetailedParseError(line, namePos, lineContent,
				"duplicate stop command '%s' (previously defined at line %d)",
				name, existingLine)
		}

		// Check for conflict with regular command
		if existingLine, exists := cr.regularCommands[name]; exists {
			return NewDetailedParseError(line, namePos, lineContent,
				"stop command '%s' conflicts with regular command (defined at line %d)",
				name, existingLine)
		}

		cr.stopCommands[name] = line

	} else {
		// Regular command
		// Check for duplicate regular command
		if existingLine, exists := cr.regularCommands[name]; exists {
			return NewDetailedParseError(line, namePos, lineContent,
				"duplicate command '%s' (previously defined at line %d)",
				name, existingLine)
		}

		// Check for conflict with watch command
		if existingLine, exists := cr.watchCommands[name]; exists {
			return NewDetailedParseError(line, namePos, lineContent,
				"regular command '%s' conflicts with watch command (defined at line %d)",
				name, existingLine)
		}

		// Check for conflict with stop command
		if existingLine, exists := cr.stopCommands[name]; exists {
			return NewDetailedParseError(line, namePos, lineContent,
				"regular command '%s' conflicts with stop command (defined at line %d)",
				name, existingLine)
		}

		cr.regularCommands[name] = line
	}

	return nil
}

// GetWatchCommands returns all registered watch commands
func (cr *CommandRegistry) GetWatchCommands() map[string]int {
	return cr.watchCommands
}

// GetStopCommands returns all registered stop commands
func (cr *CommandRegistry) GetStopCommands() map[string]int {
	return cr.stopCommands
}

// GetRegularCommands returns all registered regular commands
func (cr *CommandRegistry) GetRegularCommands() map[string]int {
	return cr.regularCommands
}

// ValidateWatchStopPairs validates that watch commands have valid stop counterparts
func (cr *CommandRegistry) ValidateWatchStopPairs() error {
	// Note: We don't require every watch to have a stop since stop is optional
	// We also don't require every stop to have a watch since stop commands can be standalone
	// This method is kept for future validation rules if needed

	return nil
}

// Parse parses a command file content into a CommandFile structure
func Parse(content string) (*CommandFile, error) {
	// Ensure content has a trailing newline for consistent parsing
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	// Split the content into lines for error reporting
	lines := strings.Split(content, "\n")

	// Create input stream from the content
	input := antlr.NewInputStream(content)

	// Create lexer with error handling
	lexer := gen.NewdevcmdLexer(input)
	errorListener := &ErrorCollector{
		lines: lines,
	}
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(errorListener)

	// Create token stream and parser
	tokens := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := gen.NewdevcmdParser(tokens)
	parser.RemoveErrorListeners()
	parser.AddErrorListener(errorListener)

	// Parse the input
	tree := parser.Program()

	// Check for syntax errors
	if errorListener.HasErrors() {
		return nil, errorListener.Error()
	}

	// Create a CommandFile to store the parsing results
	commandFile := &CommandFile{
		Lines:       lines,
		Definitions: []Definition{},
		Commands:    []Command{},
	}

	// Use visitor to extract commands and definitions
	visitor := &DevcmdVisitor{
		commandFile: commandFile,
		tokenStream: tokens,
		inputStream: input,
	}
	visitor.Visit(tree)

	// Verify no duplicate definitions
	if err := validateDefinitions(commandFile.Definitions, lines); err != nil {
		return nil, err
	}

	// Verify command uniqueness and conflicts using the advanced registry
	if err := validateCommands(commandFile.Commands, lines); err != nil {
		return nil, err
	}

	// Perform semantic validation of the command file
	if err := Validate(commandFile); err != nil {
		return nil, err
	}

	return commandFile, nil
}

// validateDefinitions checks for duplicate variable definitions
func validateDefinitions(definitions []Definition, lines []string) error {
	defs := make(map[string]int)

	for _, def := range definitions {
		if line, exists := defs[def.Name]; exists {
			var defLine string
			if def.Line > 0 && def.Line <= len(lines) {
				defLine = lines[def.Line-1]
			}

			namePos := strings.Index(defLine, def.Name)
			if namePos == -1 {
				namePos = 0
			}

			return NewDetailedParseError(def.Line, namePos, defLine,
				"duplicate definition of '%s' (previously defined at line %d)",
				def.Name, line)
		}
		defs[def.Name] = def.Line
	}

	return nil
}

// validateCommands performs advanced command validation using the command registry
func validateCommands(commands []Command, lines []string) error {
	registry := NewCommandRegistry(lines)

	// Register all commands and check for conflicts
	for _, cmd := range commands {
		if err := registry.RegisterCommand(cmd); err != nil {
			return err
		}
	}

	// Validate watch/stop command relationships
	if err := registry.ValidateWatchStopPairs(); err != nil {
		return err
	}

	return nil
}

// ErrorCollector collects syntax errors during parsing
type ErrorCollector struct {
	antlr.DefaultErrorListener
	errors []SyntaxError
	lines  []string // Store the original source lines
}

// SyntaxError represents a syntax error with location information
type SyntaxError struct {
	Line    int
	Column  int
	Message string
}

// simplifyErrorMessage converts verbose ANTLR messages to user-friendly ones
func simplifyErrorMessage(msg string) string {
	// Common ANTLR error patterns and their simplified versions
	if strings.Contains(msg, "expecting") && strings.Contains(msg, "';'") {
		return "missing ';'"
	}
	if strings.Contains(msg, "missing '}'") {
		return "missing '}'"
	}
	if strings.Contains(msg, "missing ':'") {
		return "missing ':'"
	}
	if strings.Contains(msg, "expecting") && strings.Contains(msg, "'}'") {
		return "missing '}'"
	}
	if strings.Contains(msg, "no viable alternative") {
		return "syntax error"
	}
	if strings.Contains(msg, "extraneous input") {
		return "unexpected input"
	}

	// Return original message if no pattern matches
	return msg
}

// Update your existing SyntaxError method in ErrorCollector to use this:
func (e *ErrorCollector) SyntaxError(recognizer antlr.Recognizer, offendingSymbol interface{}, line, column int, msg string, ex antlr.RecognitionException) {
	e.errors = append(e.errors, SyntaxError{
		Line:    line,
		Column:  column,
		Message: simplifyErrorMessage(msg), // Use the simplified message
	})
}

// HasErrors returns true if syntax errors were found
func (e *ErrorCollector) HasErrors() bool {
	return len(e.errors) > 0
}

// Error returns a ParseError for the first syntax error
func (e *ErrorCollector) Error() error {
	if len(e.errors) == 0 {
		return nil
	}

	err := e.errors[0]

	// Get the line context if available
	var context string
	if err.Line > 0 && err.Line <= len(e.lines) {
		context = e.lines[err.Line-1]
	}

	if context != "" {
		return NewDetailedParseError(err.Line, err.Column, context, "%s", err.Message)
	} else {
		return NewParseError(err.Line, "syntax error at column %d: %s", err.Column, err.Message)
	}
}

// DevcmdVisitor implements the visitor pattern for traversing the parse tree
type DevcmdVisitor struct {
	commandFile *CommandFile
	tokenStream antlr.TokenStream
	inputStream antlr.CharStream
}

// Visit is the entry point for the visitor pattern
func (v *DevcmdVisitor) Visit(tree antlr.ParseTree) {
	switch t := tree.(type) {
	case *gen.ProgramContext:
		v.visitProgram(t)
	case *gen.LineContext:
		v.visitLine(t)
	case *gen.VariableDefinitionContext:
		v.visitVariableDefinition(t)
	case *gen.CommandDefinitionContext:
		v.visitCommandDefinition(t)
	case antlr.TerminalNode:
		// Skip terminal nodes silently
	default:
		// Visit children for other node types
		for i := 0; i < tree.GetChildCount(); i++ {
			child := tree.GetChild(i)
			// Type assertion to convert antlr.Tree to antlr.ParseTree
			if parseTree, ok := child.(antlr.ParseTree); ok {
				v.Visit(parseTree)
			}
		}
	}
}

// visitProgram processes the root program node
func (v *DevcmdVisitor) visitProgram(ctx *gen.ProgramContext) {
	for _, line := range ctx.AllLine() {
		v.Visit(line)
	}
}

// visitLine processes a line node
func (v *DevcmdVisitor) visitLine(ctx *gen.LineContext) {
	if varDef := ctx.VariableDefinition(); varDef != nil {
		v.Visit(varDef)
	} else if cmdDef := ctx.CommandDefinition(); cmdDef != nil {
		v.Visit(cmdDef)
	}
	// Skip NEWLINE-only lines
}

// visitVariableDefinition processes a variable definition
func (v *DevcmdVisitor) visitVariableDefinition(ctx *gen.VariableDefinitionContext) {
	name := ctx.NAME().GetText()
	cmdText := ctx.CommandText()
	value := v.getOriginalText(cmdText)
	line := ctx.GetStart().GetLine()

	v.commandFile.Definitions = append(v.commandFile.Definitions, Definition{
		Name:  name,
		Value: value,
		Line:  line,
	})
}

// visitCommandDefinition processes a command definition
func (v *DevcmdVisitor) visitCommandDefinition(ctx *gen.CommandDefinitionContext) {
	name := ctx.NAME().GetText()
	line := ctx.GetStart().GetLine()

	// Check modifiers
	isWatch := ctx.WATCH() != nil
	isStop := ctx.STOP() != nil

	if simpleCmd := ctx.SimpleCommand(); simpleCmd != nil {
		// Process simple command
		cmd := v.processSimpleCommand(simpleCmd.(*gen.SimpleCommandContext))

		v.commandFile.Commands = append(v.commandFile.Commands, Command{
			Name:    name,
			Command: cmd,
			Line:    line,
			IsWatch: isWatch,
			IsStop:  isStop,
		})
	} else if blockCmd := ctx.BlockCommand(); blockCmd != nil {
		// Process block command
		blockStatements := v.processBlockCommand(blockCmd.(*gen.BlockCommandContext))

		v.commandFile.Commands = append(v.commandFile.Commands, Command{
			Name:    name,
			Line:    line,
			IsWatch: isWatch,
			IsStop:  isStop,
			IsBlock: true,
			Block:   blockStatements,
		})
	}
}

// processSimpleCommand extracts text from a simple command
func (v *DevcmdVisitor) processSimpleCommand(ctx *gen.SimpleCommandContext) string {
	// Get main text
	cmdText := v.getOriginalText(ctx.CommandText())
	cmdText = strings.TrimRight(cmdText, " \t") // keep tail blanks only for continuations

	// Process continuations
	var fullText strings.Builder
	fullText.WriteString(cmdText)

	for _, contLine := range ctx.AllContinuationLine() {
		contCtx := contLine.(*gen.ContinuationLineContext)
		contText := v.getOriginalText(contCtx.CommandText())
		contText = strings.TrimLeft(contText, " \t") // strip leading blanks only
		fullText.WriteString(" ")                    // Add a single space for continuation
		fullText.WriteString(contText)
	}

	return fullText.String()
}

// processBlockCommand extracts statements from a block command
func (v *DevcmdVisitor) processBlockCommand(ctx *gen.BlockCommandContext) []BlockStatement {
	var statements []BlockStatement

	blockStmts := ctx.BlockStatements()
	if blockStmts == nil {
		return statements
	}

	nonEmptyStmts := blockStmts.(*gen.BlockStatementsContext).NonEmptyBlockStatements()
	if nonEmptyStmts == nil {
		return statements // Empty block
	}

	// Process each statement
	nonEmptyCtx := nonEmptyStmts.(*gen.NonEmptyBlockStatementsContext)
	allBlockStmts := nonEmptyCtx.AllBlockStatement()

	for _, stmt := range allBlockStmts {
		stmtCtx := stmt.(*gen.BlockStatementContext)

		// Get the command text and check for background indicator
		command, isBackground := v.getCommandTextWithBackground(stmtCtx)

		statements = append(statements, BlockStatement{
			Command:    command,
			Background: isBackground,
		})
	}

	return statements
}

// getCommandTextWithBackground extracts command text and determines if it's a background command
// Uses a more robust approach that handles grammar parsing issues
func (v *DevcmdVisitor) getCommandTextWithBackground(ctx *gen.BlockStatementContext) (string, bool) {
	// Get the original text for the entire block statement
	start := ctx.GetStart().GetStart()
	stop := ctx.GetStop().GetStop()

	if start < 0 || stop < 0 || start > stop {
		return "", false
	}

	// Extract the full text of the statement
	fullText := v.inputStream.GetText(start, stop)
	fullText = strings.TrimSpace(fullText)

	// Check if the statement ends with &
	isBackground := strings.HasSuffix(fullText, "&")

	// If it's a background command, remove the & and any trailing whitespace
	command := fullText
	if isBackground {
		command = strings.TrimSuffix(command, "&")
		command = strings.TrimRight(command, " \t")
	}

	return command, isBackground
}

// getOriginalText extracts the original source text for a rule context
// This function handles BACKSLASH SEMICOLON sequences specially for command text
func (v *DevcmdVisitor) getOriginalText(ctx antlr.ParserRuleContext) string {
	if ctx == nil {
		return ""
	}

	start := ctx.GetStart().GetStart()
	stop := ctx.GetStop().GetStop()

	if start < 0 || stop < 0 || start > stop {
		return ""
	}

	text := v.inputStream.GetText(start, stop)

	// For command text contexts, handle special cases
	if _, ok := ctx.(*gen.CommandTextContext); ok {
		// Handle the conversion from \\; (input) to \; (output)
		// The grammar should now parse \\; as BACKSLASH SEMICOLON sequence
		text = strings.ReplaceAll(text, "\\\\;", "\\;")
		text = strings.TrimLeft(text, " \t")
	} else {
		text = strings.TrimLeft(text, " \t")
	}

	return text
}
