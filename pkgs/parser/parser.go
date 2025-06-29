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
		// Semantic validation: variable names should only use underscores, not hyphens
		if strings.Contains(def.Name, "-") {
			var defLine string
			if def.Line > 0 && def.Line <= len(lines) {
				defLine = lines[def.Line-1]
			}

			namePos := strings.Index(defLine, def.Name)
			if namePos == -1 {
				namePos = 0
			}

			return NewDetailedParseError(def.Line, namePos, defLine, debug,
				"syntax error - variable names cannot contain hyphens, use underscores instead")
		}

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
		// Semantic validation: command names shouldn't end with hyphen
		if strings.HasSuffix(cmd.Name, "-") {
			var cmdLine string
			if cmd.Line > 0 && cmd.Line <= len(lines) {
				cmdLine = lines[cmd.Line-1]
			}

			namePos := strings.Index(cmdLine, cmd.Name)
			if namePos == -1 {
				namePos = 0
			}

			return NewDetailedParseError(cmd.Line, namePos, cmdLine, debug,
				"syntax error - command names cannot end with hyphen")
		}

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

// Enhanced error message simplification with better context awareness
func simplifyErrorMessage(msg string) string {
	// Handle block command specific errors
	if strings.Contains(msg, "expecting") && strings.Contains(msg, "'}'") {
		if strings.Contains(msg, "SEMICOLON") {
			return "missing '}' - block commands should end with '}', not ';'"
		}
		return "missing '}' to close block command"
	}

	// Handle missing semicolons
	if strings.Contains(msg, "expecting") && strings.Contains(msg, "';'") {
		return "missing ';' at end of statement"
	}

	// Handle missing closing brace
	if strings.Contains(msg, "missing '}'") {
		return "missing '}' - check that all block commands are properly closed"
	}

	// Handle missing colon
	if strings.Contains(msg, "missing ':'") {
		return "syntax error - missing ':' after command name"
	}

	// Handle missing closing parenthesis
	if strings.Contains(msg, "missing ')'") && strings.Contains(msg, "'\\n'") {
		return "missing ')' - decorator arguments must be closed before line end"
	}

	// Handle expecting closing brace
	if strings.Contains(msg, "expecting") && strings.Contains(msg, "'}'") {
		return "expecting '}' to close block command"
	}

	// Handle no viable alternative errors
	if strings.Contains(msg, "no viable alternative") {
		if strings.Contains(msg, "@") {
			return "syntax error - invalid decorator syntax"
		}
		return "syntax error - unexpected input"
	}

	// Handle extraneous input
	if strings.Contains(msg, "extraneous input") {
		if strings.Contains(msg, "';'") {
			return "unexpected ';' - block commands don't use semicolons inside braces"
		}
		return "syntax error - unexpected input"
	}

	// Handle mismatched input with specific cases
	if strings.Contains(msg, "mismatched input") {
		// Handle numbers where identifiers expected
		if strings.Contains(msg, "expecting NAME") {
			return "syntax error - invalid identifier (cannot start with number or special character)"
		}

		if strings.Contains(msg, "expecting") {
			// Extract what was expected
			expectStart := strings.Index(msg, "expecting")
			if expectStart != -1 {
				expected := msg[expectStart:]
				if strings.Contains(expected, "'}'") {
					return "found ';' but expected '}' - use '}' to close block commands"
				}
			}
		}
		return "syntax error - unexpected token"
	}

	// Handle unexpected input (like starting with dash or number)
	if strings.Contains(msg, "unexpected input") {
		return "syntax error - unexpected input"
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
	case *gen.DecoratorContext:
		if v.debug != nil {
			v.debug.Log("Visiting decorator")
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
	// Use NAME token
	name := ctx.NAME().GetText()

	var value string
	// Handle both forms: def NAME = value; and def NAME = ;
	if varValue := ctx.VariableValue(); varValue != nil {
		if cmdText := varValue.CommandText(); cmdText != nil {
			value = v.getOriginalText(cmdText)
		}
	}
	// If no variableValue, value remains empty string

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
	// Use NAME token
	name := ctx.NAME().GetText()
	line := ctx.GetStart().GetLine()

	isWatch := ctx.WATCH() != nil
	isStop := ctx.STOP() != nil

	if v.debug != nil {
		v.debug.Log("Found command: %s (watch: %v, stop: %v)", name, isWatch, isStop)
	}

	commandBody := ctx.CommandBody()

	if decoratedCmd := commandBody.DecoratedCommand(); decoratedCmd != nil {
		// Handle decorated command at top level
		decoratedStmt := v.processDecoratedCommand(decoratedCmd.(*gen.DecoratedCommandContext))

		v.commandFile.Commands = append(v.commandFile.Commands, Command{
			Name:     name,
			Line:     line,
			IsWatch:  isWatch,
			IsStop:   isStop,
			IsBlock:  true,
			Block:    []BlockStatement{decoratedStmt},
			Elements: decoratedStmt.Elements,
		})

	} else if simpleCmd := commandBody.SimpleCommand(); simpleCmd != nil {
		cmd := v.processSimpleCommand(simpleCmd.(*gen.SimpleCommandContext))

		// For now, use the original text as one element if it doesn't contain valid decorators
		// The grammar should have already handled decorator parsing correctly
		var elements []CommandElement
		if cmd != "" {
			// Check if this contains any valid decorators by trying to parse
			tempElements := v.parseCommandString(cmd)
			hasDecorators := false
			for _, elem := range tempElements {
				if elem.IsDecorator() {
					hasDecorators = true
					break
				}
			}

			if hasDecorators {
				elements = tempElements
			} else {
				// No decorators found, treat as single text element
				elements = []CommandElement{NewTextElement(cmd)}
			}
		}

		v.commandFile.Commands = append(v.commandFile.Commands, Command{
			Name:     name,
			Command:  cmd,
			Line:     line,
			IsWatch:  isWatch,
			IsStop:   isStop,
			Elements: elements,
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
	var parts []string

	// Process main command text
	cmdText := v.getOriginalText(ctx.CommandText())
	cmdText = strings.TrimRight(cmdText, " \t")
	parts = append(parts, cmdText)

	// Process continuation lines - join with space
	for _, contLine := range ctx.AllContinuationLine() {
		contCtx := contLine.(*gen.ContinuationLineContext)
		contText := v.getOriginalText(contCtx.CommandText())
		contText = strings.TrimLeft(contText, " \t")
		parts = append(parts, contText)
	}

	return strings.Join(parts, " ")
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

		// Check for unified decorator first
		if decorator := stmtCtx.Decorator(); decorator != nil {
			if v.debug != nil {
				v.debug.Log("Block statement %d: unified decorator", i)
			}
			decorCtx := decorator.(*gen.DecoratorContext)
			decoratedStmt := v.processDecorator(decorCtx)
			statements = append(statements, decoratedStmt)
		} else {
			if v.debug != nil {
				v.debug.Log("Block statement %d: regular command", i)
			}

			var parts []string

			cmdText := v.getOriginalText(stmtCtx.CommandText())
			cmdText = strings.TrimSpace(cmdText)
			if cmdText != "" {
				parts = append(parts, cmdText)
			}

			for _, contLine := range stmtCtx.AllContinuationLine() {
				contCtx := contLine.(*gen.ContinuationLineContext)
				contText := v.getOriginalText(contCtx.CommandText())
				contText = strings.TrimLeft(contText, " \t")
				if contText != "" {
					parts = append(parts, contText)
				}
			}

			commandText := strings.Join(parts, " ")

			// Skip empty statements
			if commandText == "" {
				if v.debug != nil {
					v.debug.Log("Skipping empty statement %d", i)
				}
				continue
			}

			// Log what we're about to parse
			if v.debug != nil {
				v.debug.Log("Parsing command text: %s", commandText)
			}

			// Check if this contains any valid decorators
			var elements []CommandElement
			if commandText != "" {
				tempElements := v.parseCommandString(commandText)
				hasDecorators := false
				for _, elem := range tempElements {
					if elem.IsDecorator() {
						hasDecorators = true
						break
					}
				}

				if hasDecorators {
					elements = tempElements
				} else {
					// No decorators found, treat as single text element
					elements = []CommandElement{NewTextElement(commandText)}
				}
			}

			statements = append(statements, BlockStatement{
				Command:     commandText,
				IsDecorated: false,
				Elements:    elements,
			})
		}
	}

	return statements
}

func (v *DevcmdVisitor) processDecoratedCommand(ctx *gen.DecoratedCommandContext) BlockStatement {
	decorator := ctx.Decorator().(*gen.DecoratorContext)
	return v.processDecorator(decorator)
}

func (v *DevcmdVisitor) processDecorator(decorCtx *gen.DecoratorContext) BlockStatement {
	// Use NAME token
	decorator := decorCtx.NAME().GetText()

	// Check which form of decorator this is
	if decorCtx.LPAREN() != nil {
		// Function decorator: @name(...) or @name(...) { ... }
		var content string
		openParenToken := decorCtx.LPAREN()
		closeParenToken := decorCtx.RPAREN()

		if openParenToken != nil && closeParenToken != nil {
			openParenSymbol := openParenToken.GetSymbol()
			closeParenSymbol := closeParenToken.GetSymbol()

			if openParenSymbol != nil && closeParenSymbol != nil {
				contentStart := openParenSymbol.GetStop() + 1  // After the (
				contentStop := closeParenSymbol.GetStart() - 1 // Before the )

				if contentStop >= contentStart {
					content = v.inputStream.GetText(contentStart, contentStop)
				}
			}
		}

		// Check if decorator content has any nested decorators
		var elements []CommandElement
		if content != "" {
			tempElements := v.parseCommandString(content)
			hasDecorators := false
			for _, elem := range tempElements {
				if elem.IsDecorator() {
					hasDecorators = true
					break
				}
			}

			if hasDecorators {
				elements = tempElements
			} else {
				// No nested decorators, treat as single text element
				elements = []CommandElement{NewTextElement(content)}
			}
		}

		decoratorElem := &DecoratorElement{
			Name: decorator,
			Type: "function",
			Args: elements,
		}

		// Check if there's also a block command
		if blockCmd := decorCtx.BlockCommand(); blockCmd != nil {
			blockStatements := v.processBlockCommand(blockCmd.(*gen.BlockCommandContext))
			decoratorElem.Block = blockStatements

			if v.debug != nil {
				v.debug.Log("Function+Block decorator: %s(%s) with %d statements", decorator, content, len(blockStatements))
			}

			return BlockStatement{
				Elements:       []CommandElement{decoratorElem},
				IsDecorated:    true,
				Decorator:      decorator,
				DecoratorType:  "function",
				Command:        content,
				DecoratedBlock: blockStatements,
			}
		} else {
			if v.debug != nil {
				v.debug.Log("Function decorator: %s(%s)", decorator, content)
			}

			return BlockStatement{
				Elements:      []CommandElement{decoratorElem},
				IsDecorated:   true,
				Decorator:     decorator,
				DecoratorType: "function",
				Command:       content,
			}
		}
	} else {
		// Block decorator: @name { ... }
		blockCmd := decorCtx.BlockCommand().(*gen.BlockCommandContext)
		blockStatements := v.processBlockCommand(blockCmd)

		if v.debug != nil {
			v.debug.Log("Block decorator: %s with %d statements", decorator, len(blockStatements))
		}

		decoratorElem := &DecoratorElement{
			Name:  decorator,
			Type:  "block",
			Block: blockStatements,
		}

		return BlockStatement{
			Elements:       []CommandElement{decoratorElem},
			IsDecorated:    true,
			Decorator:      decorator,
			DecoratorType:  "block",
			DecoratedBlock: blockStatements,
		}
	}
}

// parseCommandString with context-aware @ symbol handling - only parse valid decorator patterns.
func (v *DevcmdVisitor) parseCommandString(text string) []CommandElement {
	var elements []CommandElement
	var textBuffer strings.Builder
	i := 0

	for i < len(text) {
		if text[i] == '@' {
			// Try to parse decorator starting here
			decorator, consumed := v.parseDecorator(text[i:])
			if decorator != nil {
				// Valid decorator - flush buffer and add decorator
				if textBuffer.Len() > 0 {
					elements = append(elements, NewTextElement(textBuffer.String()))
					textBuffer.Reset()
				}
				elements = append(elements, decorator)
				i += consumed

				if v.debug != nil {
					v.debug.Log("Successfully parsed decorator: %s, consumed: %d", decorator.Name, consumed)
				}
			} else {
				// Not a valid decorator - add @ to buffer
				textBuffer.WriteByte('@')
				i++
			}
		} else {
			// Regular character - add to buffer
			textBuffer.WriteByte(text[i])
			i++
		}
	}

	// Flush any remaining buffer
	if textBuffer.Len() > 0 {
		elements = append(elements, NewTextElement(textBuffer.String()))
	}

	if v.debug != nil {
		v.debug.Log("parseCommandString completed: %d elements", len(elements))
	}

	return elements
}

// parseDecorator parses decorator syntax from text starting with '@'.
// Only parses valid @name(...) patterns, not emails, SSH, or shell syntax.
func (v *DevcmdVisitor) parseDecorator(text string) (*DecoratorElement, int) {
	if len(text) < 2 || text[0] != '@' {
		return nil, 0
	}

	// Find decorator name - must start with letter and support hyphens/underscores
	nameStart := 1
	nameEnd := nameStart

	// Must start with letter
	if nameStart >= len(text) || !isLetter(text[nameStart]) {
		return nil, 0 // Invalid decorator name
	}

	for nameEnd < len(text) && (isLetter(text[nameEnd]) || isDigit(text[nameEnd]) || text[nameEnd] == '_' || text[nameEnd] == '-') {
		nameEnd++
	}

	if nameEnd == nameStart {
		return nil, 0 // No valid name
	}

	name := text[nameStart:nameEnd]

	// Check for opening parenthesis (function decorator)
	if nameEnd >= len(text) || text[nameEnd] != '(' {
		// Not a function decorator pattern
		return nil, 0
	}

	// Enhanced parentheses matching with proper quote awareness
	parenLevel := 1
	contentStart := nameEnd + 1
	i := contentStart
	inDoubleQuotes := false
	inSingleQuotes := false
	inBackticks := false

	for i < len(text) && parenLevel > 0 {
		ch := text[i]

		// Handle escape sequences - skip escaped characters entirely
		if ch == '\\' && i+1 < len(text) {
			i += 2 // Skip the backslash and the escaped character
			continue
		}

		// Handle quote state transitions (only when not inside other quotes)
		switch ch {
		case '"':
			if !inSingleQuotes && !inBackticks {
				inDoubleQuotes = !inDoubleQuotes
			}
		case '\'':
			if !inDoubleQuotes && !inBackticks {
				inSingleQuotes = !inSingleQuotes
			}
		case '`':
			if !inDoubleQuotes && !inSingleQuotes {
				inBackticks = !inBackticks
			}
		}

		// Only count decorator parentheses when not inside quotes
		// This allows shell syntax like $(cmd) inside quotes to be treated as raw text
		if !inDoubleQuotes && !inSingleQuotes && !inBackticks {
			switch ch {
			case '(':
				parenLevel++
			case ')':
				parenLevel--
			}
		}

		i++
	}

	if parenLevel != 0 {
		// Enhanced error logging for debugging
		if v.debug != nil {
			v.debug.LogError("Unmatched parentheses in decorator %s", name)
			v.debug.LogError("Remaining parentheses level: %d", parenLevel)
			v.debug.LogError("Quote states - double: %v, single: %v, backticks: %v",
				inDoubleQuotes, inSingleQuotes, inBackticks)

			// Show a sample of the problematic content
			maxLen := min(100, len(text)-contentStart)
			if maxLen > 0 {
				sample := text[contentStart : contentStart+maxLen]
				v.debug.LogError("Content sample: %s", sample)
			}
		}
		return nil, 0 // Unmatched parentheses
	}

	contentEnd := i - 1 // Before the closing )
	content := text[contentStart:contentEnd]

	// Parse the content recursively to handle nested @var() decorators
	var args []CommandElement
	if content != "" {
		args = v.parseCommandString(content)
	}

	decorator := &DecoratorElement{
		Name: name,
		Type: "function",
		Args: args,
	}

	if v.debug != nil {
		v.debug.Log("Successfully parsed decorator %s with content length %d", name, len(content))
	}

	return decorator, i // Total length including @name(...)
}

// Helper functions
func isLetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
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

// NewTextElement creates a TextElement
func NewTextElement(text string) *TextElement {
	return &TextElement{
		Text: text,
	}
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
