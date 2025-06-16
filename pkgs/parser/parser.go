package parser

import (
	"fmt"
	"strings"

	"github.com/aledsdavies/devcmd/internal/gen"
	"github.com/antlr4-go/antlr/v4"
)

//go:generate bash -c "cd ../../grammar && antlr -Dlanguage=Go -package gen -o ../internal/gen DevcmdLexer.g4 DevcmdParser.g4"

type ParseError struct {
	Line    int
	Column  int
	Message string
	Context string
	Debug   *DebugTrace
}

func (e *ParseError) Error() string {
	var builder strings.Builder

	if e.Context == "" {
		builder.WriteString(fmt.Sprintf("line %d: %s", e.Line, e.Message))
	} else {
		pointer := strings.Repeat(" ", e.Column) + "^"
		builder.WriteString(fmt.Sprintf("line %d: %s\n%s\n%s", e.Line, e.Message, e.Context, pointer))
	}

	// Add debug trace if available and enabled
	if e.Debug != nil && e.Debug.Enabled && e.Debug.HasTrace() {
		builder.WriteString(e.Debug.String())
	}

	return builder.String()
}

func NewParseError(line int, debug *DebugTrace, format string, args ...interface{}) *ParseError {
	return &ParseError{
		Line:    line,
		Message: fmt.Sprintf(format, args...),
		Debug:   debug,
	}
}

func NewDetailedParseError(line int, column int, context string, debug *DebugTrace, format string, args ...interface{}) *ParseError {
	return &ParseError{
		Line:    line,
		Column:  column,
		Context: context,
		Message: fmt.Sprintf(format, args...),
		Debug:   debug,
	}
}

type CommandRegistry struct {
	regularCommands map[string]int
	watchCommands   map[string]int
	stopCommands    map[string]int
	lines           []string
	debug           *DebugTrace
}

func NewCommandRegistry(lines []string, debug *DebugTrace) *CommandRegistry {
	return &CommandRegistry{
		regularCommands: make(map[string]int),
		watchCommands:   make(map[string]int),
		stopCommands:    make(map[string]int),
		lines:           lines,
		debug:           debug,
	}
}

func (cr *CommandRegistry) RegisterCommand(cmd Command) error {
	name := cmd.Name
	line := cmd.Line

	var lineContent string
	if line > 0 && line <= len(cr.lines) {
		lineContent = cr.lines[line-1]
	}

	namePos := strings.Index(lineContent, name)
	if namePos == -1 {
		namePos = 0
	}

	if cmd.IsWatch {
		if existingLine, exists := cr.watchCommands[name]; exists {
			return NewDetailedParseError(line, namePos, lineContent, cr.debug,
				"duplicate watch command '%s' (previously defined at line %d)",
				name, existingLine)
		}

		if existingLine, exists := cr.regularCommands[name]; exists {
			return NewDetailedParseError(line, namePos, lineContent, cr.debug,
				"watch command '%s' conflicts with regular command (defined at line %d)",
				name, existingLine)
		}

		cr.watchCommands[name] = line

	} else if cmd.IsStop {
		if existingLine, exists := cr.stopCommands[name]; exists {
			return NewDetailedParseError(line, namePos, lineContent, cr.debug,
				"duplicate stop command '%s' (previously defined at line %d)",
				name, existingLine)
		}

		if existingLine, exists := cr.regularCommands[name]; exists {
			return NewDetailedParseError(line, namePos, lineContent, cr.debug,
				"stop command '%s' conflicts with regular command (defined at line %d)",
				name, existingLine)
		}

		cr.stopCommands[name] = line

	} else {
		if existingLine, exists := cr.regularCommands[name]; exists {
			return NewDetailedParseError(line, namePos, lineContent, cr.debug,
				"duplicate command '%s' (previously defined at line %d)",
				name, existingLine)
		}

		if existingLine, exists := cr.watchCommands[name]; exists {
			return NewDetailedParseError(line, namePos, lineContent, cr.debug,
				"regular command '%s' conflicts with watch command (defined at line %d)",
				name, existingLine)
		}

		if existingLine, exists := cr.stopCommands[name]; exists {
			return NewDetailedParseError(line, namePos, lineContent, cr.debug,
				"regular command '%s' conflicts with stop command (defined at line %d)",
				name, existingLine)
		}

		cr.regularCommands[name] = line
	}

	return nil
}

func (cr *CommandRegistry) GetWatchCommands() map[string]int {
	return cr.watchCommands
}

func (cr *CommandRegistry) GetStopCommands() map[string]int {
	return cr.stopCommands
}

func (cr *CommandRegistry) GetRegularCommands() map[string]int {
	return cr.regularCommands
}

func (cr *CommandRegistry) ValidateWatchStopPairs() error {
	return nil
}

func Parse(content string, debug bool) (*CommandFile, error) {
	var debugTrace *DebugTrace
	if debug {
		debugTrace = &DebugTrace{Enabled: true}
	}

	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	lines := strings.Split(content, "\n")
	if debugTrace != nil {
		debugTrace.Log("Input lines: %d", len(lines))
	}

	input := antlr.NewInputStream(content)
	lexer := gen.NewDevcmdLexer(input)

	errorListener := &ErrorCollector{
		lines: lines,
		debug: debugTrace,
	}
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(errorListener)

	tokens := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := gen.NewDevcmdParser(tokens)
	parser.RemoveErrorListeners()
	parser.AddErrorListener(errorListener)

	if debugTrace != nil {
		debugTrace.Log("Starting parse")
	}
	tree := parser.Program()

	if errorListener.HasErrors() {
		if debugTrace != nil {
			debugTrace.LogError("Syntax errors found: %d", len(errorListener.errors))
		}
		return nil, errorListener.Error()
	}

	commandFile := &CommandFile{
		Lines:       lines,
		Definitions: []Definition{},
		Commands:    []Command{},
	}

	visitor := &DevcmdVisitor{
		commandFile: commandFile,
		tokenStream: tokens,
		inputStream: input,
		debug:       debugTrace,
	}
	visitor.Visit(tree)

	if debugTrace != nil {
		debugTrace.Log("Found %d definitions, %d commands", len(commandFile.Definitions), len(commandFile.Commands))
	}

	if err := validateDefinitions(commandFile.Definitions, lines, debugTrace); err != nil {
		if debugTrace != nil {
			debugTrace.LogError("Definition validation failed: %v", err)
		}
		return nil, err
	}

	if err := validateCommands(commandFile.Commands, lines, debugTrace); err != nil {
		if debugTrace != nil {
			debugTrace.LogError("Command validation failed: %v", err)
		}
		return nil, err
	}

	if err := ValidateWithDebug(commandFile, debugTrace); err != nil {
		if debugTrace != nil {
			debugTrace.LogError("Semantic validation failed: %v", err)
		}
		return nil, err
	}

	return commandFile, nil
}

func validateDefinitions(definitions []Definition, lines []string, debug *DebugTrace) error {
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

			return NewDetailedParseError(def.Line, namePos, defLine, debug,
				"duplicate definition of '%s' (previously defined at line %d)",
				def.Name, line)
		}
		defs[def.Name] = def.Line
	}

	return nil
}

func validateCommands(commands []Command, lines []string, debug *DebugTrace) error {
	registry := NewCommandRegistry(lines, debug)

	for _, cmd := range commands {
		if err := registry.RegisterCommand(cmd); err != nil {
			return err
		}
	}

	if err := registry.ValidateWatchStopPairs(); err != nil {
		return err
	}

	return nil
}

type ErrorCollector struct {
	antlr.DefaultErrorListener
	errors []SyntaxError
	lines  []string
	debug  *DebugTrace
}

type SyntaxError struct {
	Line    int
	Column  int
	Message string
}

func simplifyErrorMessage(msg string) string {
	if strings.Contains(msg, "expecting") && strings.Contains(msg, "';'") {
		return "missing ';'"
	}
	if strings.Contains(msg, "missing '}'") {
		return "missing '}'"
	}
	if strings.Contains(msg, "missing ':'") {
		return "missing ':'"
	}
	if strings.Contains(msg, "missing ')'") && strings.Contains(msg, "'\\n'") {
		return "missing ')' at '\\n'"
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

	return msg
}

func (e *ErrorCollector) SyntaxError(recognizer antlr.Recognizer, offendingSymbol interface{}, line, column int, msg string, ex antlr.RecognitionException) {
	simplified := simplifyErrorMessage(msg)
	if e.debug != nil {
		e.debug.LogError("Syntax error at %d:%d - original: %s, simplified: %s", line, column, msg, simplified)
	}

	e.errors = append(e.errors, SyntaxError{
		Line:    line,
		Column:  column,
		Message: simplified,
	})
}

func (e *ErrorCollector) HasErrors() bool {
	return len(e.errors) > 0
}

func (e *ErrorCollector) Error() error {
	if len(e.errors) == 0 {
		return nil
	}

	err := e.errors[0]

	var context string
	if err.Line > 0 && err.Line <= len(e.lines) {
		context = e.lines[err.Line-1]
	}

	if context != "" {
		return NewDetailedParseError(err.Line, err.Column, context, e.debug, "%s", err.Message)
	} else {
		return NewParseError(err.Line, e.debug, "syntax error at column %d: %s", err.Column, err.Message)
	}
}

type DevcmdVisitor struct {
	commandFile *CommandFile
	tokenStream antlr.TokenStream
	inputStream antlr.CharStream
	debug       *DebugTrace
}

func (v *DevcmdVisitor) Visit(tree antlr.ParseTree) {
	switch t := tree.(type) {
	case *gen.ProgramContext:
		if v.debug != nil {
			v.debug.Log("Visiting program")
		}
		v.visitProgram(t)
	case *gen.LineContext:
		if v.debug != nil {
			v.debug.Log("Visiting line")
		}
		v.visitLine(t)
	case *gen.VariableDefinitionContext:
		if v.debug != nil {
			v.debug.Log("Visiting variable definition")
		}
		v.visitVariableDefinition(t)
	case *gen.CommandDefinitionContext:
		if v.debug != nil {
			v.debug.Log("Visiting command definition")
		}
		v.visitCommandDefinition(t)
	case *gen.FunctionAnnotContext:
		if v.debug != nil {
			v.debug.Log("Visiting function annotation")
		}
	case *gen.BlockAnnotContext:
		if v.debug != nil {
			v.debug.Log("Visiting block annotation")
		}
	case *gen.SimpleAnnotContext:
		if v.debug != nil {
			v.debug.Log("Visiting simple annotation")
		}
	case antlr.TerminalNode:
		if v.debug != nil {
			v.debug.LogToken(t.GetText())
		}
	default:
		if v.debug != nil {
			v.debug.Log("Visiting unknown node type: %T", t)
		}
		for i := 0; i < tree.GetChildCount(); i++ {
			child := tree.GetChild(i)
			if parseTree, ok := child.(antlr.ParseTree); ok {
				v.Visit(parseTree)
			}
		}
	}
}

func (v *DevcmdVisitor) visitProgram(ctx *gen.ProgramContext) {
	for _, line := range ctx.AllLine() {
		v.Visit(line)
	}
}

func (v *DevcmdVisitor) visitLine(ctx *gen.LineContext) {
	if varDef := ctx.VariableDefinition(); varDef != nil {
		v.Visit(varDef)
	} else if cmdDef := ctx.CommandDefinition(); cmdDef != nil {
		v.Visit(cmdDef)
	}
}

func (v *DevcmdVisitor) visitVariableDefinition(ctx *gen.VariableDefinitionContext) {
	name := ctx.NAME().GetText()

	var value string
	if varValue := ctx.VariableValue(); varValue != nil {
		if cmdText := varValue.CommandText(); cmdText != nil {
			value = v.getOriginalText(cmdText)
		}
	}

	line := ctx.GetStart().GetLine()

	if v.debug != nil {
		v.debug.Log("Found definition: %s = %s", name, value)
	}
	v.commandFile.Definitions = append(v.commandFile.Definitions, Definition{
		Name:  name,
		Value: value,
		Line:  line,
	})
}

func (v *DevcmdVisitor) visitCommandDefinition(ctx *gen.CommandDefinitionContext) {
	name := ctx.NAME().GetText()
	line := ctx.GetStart().GetLine()

	isWatch := ctx.WATCH() != nil
	isStop := ctx.STOP() != nil

	if v.debug != nil {
		v.debug.Log("Found command: %s (watch: %v, stop: %v)", name, isWatch, isStop)
	}

	commandBody := ctx.CommandBody()

	if annotatedCmd := commandBody.AnnotatedCommand(); annotatedCmd != nil {
		// Handle annotated command at top level
		annotatedStmt := v.processAnnotatedCommand(annotatedCmd)

		v.commandFile.Commands = append(v.commandFile.Commands, Command{
			Name:    name,
			Line:    line,
			IsWatch: isWatch,
			IsStop:  isStop,
			IsBlock: true,
			Block:   []BlockStatement{annotatedStmt},
		})

	} else if simpleCmd := commandBody.SimpleCommand(); simpleCmd != nil {
		cmd := v.processSimpleCommand(simpleCmd.(*gen.SimpleCommandContext))

		v.commandFile.Commands = append(v.commandFile.Commands, Command{
			Name:    name,
			Command: cmd,
			Line:    line,
			IsWatch: isWatch,
			IsStop:  isStop,
		})

	} else if blockCmd := commandBody.BlockCommand(); blockCmd != nil {
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

func (v *DevcmdVisitor) processSimpleCommand(ctx *gen.SimpleCommandContext) string {
	cmdText := v.getOriginalText(ctx.CommandText())
	cmdText = strings.TrimRight(cmdText, " \t")

	var fullText strings.Builder
	fullText.WriteString(cmdText)

	for _, contLine := range ctx.AllContinuationLine() {
		contCtx := contLine.(*gen.ContinuationLineContext)
		contText := v.getOriginalText(contCtx.CommandText())
		contText = strings.TrimLeft(contText, " \t")
		fullText.WriteString(" ")
		fullText.WriteString(contText)
	}

	return fullText.String()
}

// Process annotation command (similar to simple command but without semicolon)
func (v *DevcmdVisitor) processAnnotationCommand(ctx *gen.AnnotationCommandContext) string {
	cmdText := v.getOriginalText(ctx.CommandText())
	cmdText = strings.TrimRight(cmdText, " \t")

	var fullText strings.Builder
	fullText.WriteString(cmdText)

	for _, contLine := range ctx.AllContinuationLine() {
		contCtx := contLine.(*gen.ContinuationLineContext)
		contText := v.getOriginalText(contCtx.CommandText())
		contText = strings.TrimLeft(contText, " \t")
		fullText.WriteString(" ")
		fullText.WriteString(contText)
	}

	return fullText.String()
}

func (v *DevcmdVisitor) processBlockCommand(ctx *gen.BlockCommandContext) []BlockStatement {
	var statements []BlockStatement

	blockStmts := ctx.BlockStatements()
	if blockStmts == nil {
		if v.debug != nil {
			v.debug.Log("Empty block")
		}
		return statements
	}

	nonEmptyStmts := blockStmts.(*gen.BlockStatementsContext).NonEmptyBlockStatements()
	if nonEmptyStmts == nil {
		if v.debug != nil {
			v.debug.Log("Block with no non-empty statements")
		}
		return statements
	}

	nonEmptyCtx := nonEmptyStmts.(*gen.NonEmptyBlockStatementsContext)
	allBlockStmts := nonEmptyCtx.AllBlockStatement()

	if v.debug != nil {
		v.debug.Log("Processing %d block statements", len(allBlockStmts))
	}

	for i, stmt := range allBlockStmts {
		stmtCtx := stmt.(*gen.BlockStatementContext)

		if annotatedCmd := stmtCtx.AnnotatedCommand(); annotatedCmd != nil {
			if v.debug != nil {
				v.debug.Log("Block statement %d: annotated command", i)
			}
			annotatedStmt := v.processAnnotatedCommand(annotatedCmd)
			statements = append(statements, annotatedStmt)
		} else {
			if v.debug != nil {
				v.debug.Log("Block statement %d: regular command", i)
			}
			cmdText := v.getOriginalText(stmtCtx.CommandText())
			cmdText = strings.TrimSpace(cmdText)

			// Skip empty statements (fixes block counting issues)
			if cmdText == "" {
				if v.debug != nil {
					v.debug.Log("Skipping empty statement %d", i)
				}
				continue
			}

			var fullText strings.Builder
			fullText.WriteString(cmdText)

			for _, contLine := range stmtCtx.AllContinuationLine() {
				contCtx := contLine.(*gen.ContinuationLineContext)
				contText := v.getOriginalText(contCtx.CommandText())
				contText = strings.TrimLeft(contText, " \t")
				fullText.WriteString(" ")
				fullText.WriteString(contText)
			}

			statements = append(statements, BlockStatement{
				Command:     fullText.String(),
				IsAnnotated: false,
			})
		}
	}

	return statements
}

func (v *DevcmdVisitor) processAnnotatedCommand(ctx antlr.ParserRuleContext) BlockStatement {
	switch annotCtx := ctx.(type) {
	case *gen.FunctionAnnotContext:
		// Extract annotation name from AT_NAME_LPAREN token
		atNameLParen := annotCtx.AT_NAME_LPAREN().GetText()
		// Remove @ and ( to get the annotation name
		annotation := strings.TrimSuffix(strings.TrimPrefix(atNameLParen, "@"), "(")

		// For function annotations like @sh(...), we need to get the exact text
		// between the parentheses, preserving all formatting
		var content string

		// The AT_NAME_LPAREN token includes the @name( part
		// The RPAREN token is the closing )
		// We need everything in between

		openParenToken := annotCtx.AT_NAME_LPAREN().GetSymbol()
		closeParenToken := annotCtx.RPAREN().GetSymbol()

		// Get positions
		contentStart := openParenToken.GetStop() + 1  // After the (
		contentStop := closeParenToken.GetStart() - 1 // Before the )

		if contentStop >= contentStart {
			// Get the raw text, which preserves spaces, newlines, backslashes, etc.
			content = v.inputStream.GetText(contentStart, contentStop)
		}

		if v.debug != nil {
			v.debug.Log("Function annotation: %s(%s)", annotation, content)
		}
		return BlockStatement{
			IsAnnotated:    true,
			Annotation:     annotation,
			AnnotationType: "function",
			Command:        content,
		}

	case *gen.BlockAnnotContext:
		annotation := annotCtx.Annotation().GetText()
		blockCmd := annotCtx.BlockCommand().(*gen.BlockCommandContext)
		blockStatements := v.processBlockCommand(blockCmd)
		if v.debug != nil {
			v.debug.Log("Block annotation: %s with %d statements", annotation, len(blockStatements))
		}
		return BlockStatement{
			IsAnnotated:    true,
			Annotation:     annotation,
			AnnotationType: "block",
			AnnotatedBlock: blockStatements,
		}

	case *gen.SimpleAnnotContext:
		annotation := annotCtx.Annotation().GetText()
		// Updated to use AnnotationCommand instead of SimpleCommand
		annotCmd := annotCtx.AnnotationCommand().(*gen.AnnotationCommandContext)
		commandText := v.processAnnotationCommand(annotCmd)
		if v.debug != nil {
			v.debug.Log("Simple annotation: %s:%s", annotation, commandText)
		}
		return BlockStatement{
			IsAnnotated:    true,
			Annotation:     annotation,
			AnnotationType: "simple",
			Command:        commandText,
		}

	default:
		if v.debug != nil {
			v.debug.LogError("Unknown annotation context type: %T", ctx)
		}
		return BlockStatement{
			IsAnnotated: false,
			Command:     "",
		}
	}
}

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
	text = strings.TrimLeft(text, " \t")

	return text
}
