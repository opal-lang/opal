package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/aledsdavies/devcmd/pkgs/ast"
	"github.com/aledsdavies/devcmd/pkgs/lexer"
	"github.com/aledsdavies/devcmd/pkgs/stdlib"
)

// Parser implements a fast, spec-compliant recursive descent parser for the Devcmd language.
// It trusts the lexer to have correctly handled whitespace and tokenization, focusing
// purely on assembling the Abstract Syntax Tree (AST).
type Parser struct {
	input  string // The raw input string for accurate value slicing
	tokens []lexer.Token
	pos    int // current position in the token slice

	// errors is a slice of errors encountered during parsing.
	// This allows for better error reporting by collecting multiple errors.
	errors []string
}

// Parse tokenizes and parses the input string into a complete AST.
// It returns the Program node and any errors encountered.
func Parse(input string) (*ast.Program, error) {
	lex := lexer.New(input)
	p := &Parser{
		input:  input, // Store the raw input
		tokens: lex.TokenizeToSlice(),
	}
	program := p.parseProgram()

	if len(p.errors) > 0 {
		return nil, fmt.Errorf("parsing failed:\n- %s", strings.Join(p.errors, "\n- "))
	}
	return program, nil
}

// --- Main Parsing Logic ---

// parseProgram is the top-level entry point for parsing.
// It iterates through the tokens and parses all top-level statements.
// Program = { VariableDecl | VarGroup | CommandDecl }*
func (p *Parser) parseProgram() *ast.Program {
	program := &ast.Program{}

	for !p.isAtEnd() {
		p.skipWhitespaceAndComments()
		if p.isAtEnd() {
			break
		}

		switch p.current().Type {
		case lexer.VAR:
			if p.peek().Type == lexer.LPAREN {
				varGroup, err := p.parseVarGroup()
				if err != nil {
					p.addError(err)
					p.synchronize()
				} else {
					program.VarGroups = append(program.VarGroups, *varGroup)
				}
			} else {
				varDecl, err := p.parseVariableDecl()
				if err != nil {
					p.addError(err)
					p.synchronize()
				} else {
					program.Variables = append(program.Variables, *varDecl)
				}
			}
		case lexer.IDENTIFIER, lexer.WATCH, lexer.STOP:
			// A command can start with a name (IDENTIFIER), a keyword (WATCH/STOP),
			// or a decorator (@).
			cmd, err := p.parseCommandDecl()
			if err != nil {
				p.addError(err)
				p.synchronize()
			} else {
				program.Commands = append(program.Commands, *cmd)
			}
		default:
			p.addError(fmt.Errorf("unexpected token %s, expected a top-level declaration (var, command)", p.current().Type))
			p.synchronize()
		}
	}

	return program
}

// parseCommandDecl parses a full command declaration.
// CommandDecl = { Decorator }* [ "watch" | "stop" ] IDENTIFIER ":" CommandBody
func (p *Parser) parseCommandDecl() (*ast.CommandDecl, error) {
	startPos := p.current()

	// 1. Parse command type (watch, stop, or regular)
	cmdType := ast.Command
	var typeToken *lexer.Token
	if p.match(lexer.WATCH) {
		cmdType = ast.WatchCommand
		token := p.current()
		typeToken = &token
		p.advance()
	} else if p.match(lexer.STOP) {
		cmdType = ast.StopCommand
		token := p.current()
		typeToken = &token
		p.advance()
	}

	// 2. Parse command name
	nameToken, err := p.consume(lexer.IDENTIFIER, "expected command name")
	if err != nil {
		return nil, err
	}
	name := nameToken.Value

	// 3. Parse colon
	colonToken, err := p.consume(lexer.COLON, "expected ':' after command name")
	if err != nil {
		return nil, err
	}

	// 4. Parse command body (this will handle post-colon decorators and syntax sugar)
	body, err := p.parseCommandBody()
	if err != nil {
		return nil, err
	}

	return &ast.CommandDecl{
		Name:       name,
		Type:       cmdType,
		Body:       *body,
		Pos:        ast.Position{Line: startPos.Line, Column: startPos.Column},
		TypeToken:  typeToken,
		NameToken:  nameToken,
		ColonToken: colonToken,
	}, nil
}

// parseCommandBody parses the content after the command's colon.
// It handles the syntax sugar for simple vs. block commands.
// **FIXED**: Now properly implements syntax sugar equivalence as per spec.
// CommandBody = "{" CommandContent "}" | DecoratorSugar | CommandContent
func (p *Parser) parseCommandBody() (*ast.CommandBody, error) {
	startPos := p.current()

	// **FIXED**: Check for decorator syntax sugar: @decorator(args) { ... }
	// This should be equivalent to: { @decorator(args) { ... } }
	if p.match(lexer.AT) {
		// Save position in case we need to backtrack
		savedPos := p.pos

		// Try to parse a single decorator after the colon
		decorator, err := p.parseDecorator()
		if err != nil {
			return nil, err
		}

		// After decorators, we expect either:
		// 1. A block { ... } (syntax sugar - should be treated as IsBlock=true)
		// 2. Simple shell content (only valid for function decorators)

		if p.match(lexer.LBRACE) {
			// **SYNTAX SUGAR**: @decorator(args) { ... } becomes { @decorator(args) { ... } }
			openBrace, _ := p.consume(lexer.LBRACE, "") // already checked

			// Parse content differently based on decorator type
			switch d := decorator.(type) {
			case *ast.BlockDecorator:
				blockContent, err := p.parseBlockContent() // Parse multiple content items
				if err != nil {
					return nil, err
				}
				closeBrace, err := p.consume(lexer.RBRACE, "expected '}' to close command block")
				if err != nil {
					return nil, err
				}
				d.Content = blockContent
				return &ast.CommandBody{
					Content:    []ast.CommandContent{d},
					Pos:        ast.Position{Line: startPos.Line, Column: startPos.Column},
					OpenBrace:  &openBrace,
					CloseBrace: &closeBrace,
				}, nil
			case *ast.PatternDecorator:
				// For pattern decorators, parse pattern branches directly
				patterns, err := p.parsePatternBranchesInBlock()
				if err != nil {
					return nil, err
				}
				closeBrace, err := p.consume(lexer.RBRACE, "expected '}' to close pattern block")
				if err != nil {
					return nil, err
				}
				d.Patterns = patterns
				return &ast.CommandBody{
					Content:    []ast.CommandContent{d},
					Pos:        ast.Position{Line: startPos.Line, Column: startPos.Column},
					OpenBrace:  &openBrace,
					CloseBrace: &closeBrace,
				}, nil
			default:
				return nil, fmt.Errorf("unexpected decorator type in block context")
			}
		} else {
			// Decorator without braces - check if it's a function decorator
			if _, ok := decorator.(*ast.FunctionDecorator); !ok {
				// Block decorators must be followed by braces
				return nil, fmt.Errorf("expected '{' after block decorator(s) (at %d:%d, got %s)",
					p.current().Line, p.current().Column, p.current().Type)
			}

			// All function decorators - backtrack and parse as shell content
			p.pos = savedPos
			content, err := p.parseCommandContent(false)
			if err != nil {
				return nil, err
			}

			// **SYNTAX SUGAR NORMALIZATION**: Simple commands with only function decorators
			// should have the same AST structure as simple commands without decorators
			return &ast.CommandBody{
				Content: []ast.CommandContent{content},
				Pos:     ast.Position{Line: startPos.Line, Column: startPos.Column},
			}, nil
		}
	}

	// Explicit block: { ... }
	if p.match(lexer.LBRACE) {
		openBrace, _ := p.consume(lexer.LBRACE, "") // already checked
		contentItems, err := p.parseBlockContent()  // Parse multiple content items
		if err != nil {
			return nil, err
		}
		closeBrace, err := p.consume(lexer.RBRACE, "expected '}' to close command block")
		if err != nil {
			return nil, err
		}

		// **SYNTAX SUGAR NORMALIZATION**: All equivalent forms produce same AST structure
		// Both "build: npm run build" and "build: { npm run build }" are now identical
		if p.isSimpleShellContent(contentItems) {
			return &ast.CommandBody{
				Content: contentItems,
				Pos:     ast.Position{Line: startPos.Line, Column: startPos.Column},
				// Note: No brace tokens stored for simple commands (canonical form)
			}, nil
		}

		return &ast.CommandBody{
			Content:    contentItems, // Already a slice
			Pos:        ast.Position{Line: startPos.Line, Column: startPos.Column},
			OpenBrace:  &openBrace,
			CloseBrace: &closeBrace,
		}, nil
	}

	// Simple command (no braces, ends at newline)
	content, err := p.parseCommandContent(false) // Pass inBlock=false
	if err != nil {
		return nil, err
	}
	return &ast.CommandBody{
		Content: []ast.CommandContent{content},
		Pos:     ast.Position{Line: startPos.Line, Column: startPos.Column},
	}, nil
}

// isSimpleShellContent checks if content items represent simple shell content
// that should be normalized to canonical form (IsBlock=false)
func (p *Parser) isSimpleShellContent(contentItems []ast.CommandContent) bool {
	// Must be exactly one content item
	if len(contentItems) != 1 {
		return false
	}

	// Must be shell content without decorators
	if shell, ok := contentItems[0].(*ast.ShellContent); ok {
		// Check if it contains only text parts or function decorators (no block decorators)
		for _, part := range shell.Parts {
			if funcDecorator, ok := part.(*ast.FunctionDecorator); ok {
				// Function decorators are allowed in simple content
				if !stdlib.IsFunctionDecorator(funcDecorator.Name) {
					return false
				}
			}
		}
		return true
	}

	return false
}

// parseCommandContent parses the actual content of a command, which can be
// shell text, decorators, or pattern content.
// It is context-aware via the `inBlock` parameter.
func (p *Parser) parseCommandContent(inBlock bool) (ast.CommandContent, error) {
	// Check for pattern decorators (@when, @try)
	if p.isPatternDecorator() {
		return p.parsePatternContent()
	}

	// Check for block decorators
	if p.isBlockDecorator() {
		decorator, err := p.parseDecorator()
		if err != nil {
			return nil, err
		}

		// Handle different decorator types
		switch d := decorator.(type) {
		case *ast.BlockDecorator:
			// Parse the block content for block decorators
			if p.match(lexer.LBRACE) {
				p.advance() // consume '{'
				contentItems, err := p.parseBlockContent()
				if err != nil {
					return nil, err
				}
				_, err = p.consume(lexer.RBRACE, "expected '}' after block decorator content")
				if err != nil {
					return nil, err
				}
				d.Content = contentItems
			} else {
				return nil, fmt.Errorf("expected '{' after block decorator @%s", d.Name)
			}
			return d, nil
		case *ast.PatternDecorator:
			// Pattern decorators are handled separately
			return nil, fmt.Errorf("pattern decorators should be handled by parsePatternContent")
		default:
			return nil, fmt.Errorf("unexpected decorator type in block context")
		}
	}

	// Otherwise, it must be shell content.
	return p.parseShellContent(inBlock)
}

// parsePatternContent parses pattern-matching decorator content (@when, @try)
// This handles syntax like: @when(VAR) { pattern: command; pattern: command }
func (p *Parser) parsePatternContent() (*ast.PatternDecorator, error) {
	startPos := p.current()

	// Parse @ symbol
	atToken, err := p.consume(lexer.AT, "expected '@' to start pattern decorator")
	if err != nil {
		return nil, err
	}

	// Parse decorator name
	nameToken, err := p.consume(lexer.IDENTIFIER, "expected decorator name after '@'")
	if err != nil {
		return nil, err
	}
	decoratorName := nameToken.Value

	// Parse arguments if present
	var args []ast.Expression
	if p.match(lexer.LPAREN) {
		args, err = p.parseArgumentList()
		if err != nil {
			return nil, err
		}
		_, err = p.consume(lexer.RPAREN, "expected ')' after decorator arguments")
		if err != nil {
			return nil, err
		}
	}

	// Expect opening brace
	_, err = p.consume(lexer.LBRACE, "expected '{' after pattern decorator")
	if err != nil {
		return nil, err
	}

	// Parse pattern branches
	var patterns []ast.PatternBranch
	for !p.match(lexer.RBRACE) && !p.isAtEnd() {
		p.skipWhitespaceAndComments()
		if p.match(lexer.RBRACE) {
			break
		}

		branch, err := p.parsePatternBranch()
		if err != nil {
			return nil, err
		}
		patterns = append(patterns, *branch)
		p.skipWhitespaceAndComments()
	}

	// Expect closing brace
	_, err = p.consume(lexer.RBRACE, "expected '}' to close pattern block")
	if err != nil {
		return nil, err
	}

	return &ast.PatternDecorator{
		Name:      decoratorName,
		Args:      args,
		Patterns:  patterns,
		Pos:       ast.Position{Line: startPos.Line, Column: startPos.Column},
		AtToken:   atToken,
		NameToken: nameToken,
	}, nil
}

// parsePatternBranch parses a single pattern branch: pattern: command or pattern: { commands }
// **FIXED**: Now properly handles multiple commands per pattern branch
func (p *Parser) parsePatternBranch() (*ast.PatternBranch, error) {
	startPos := p.current()

	// Parse pattern (identifier or wildcard)
	var pattern ast.Pattern
	if p.match(lexer.ASTERISK) {
		token := p.current()
		p.advance()
		pattern = &ast.WildcardPattern{
			Pos:   ast.Position{Line: token.Line, Column: token.Column},
			Token: token,
		}
	} else if p.match(lexer.IDENTIFIER) {
		token := p.current()
		p.advance()
		pattern = &ast.IdentifierPattern{
			Name:  token.Value,
			Pos:   ast.Position{Line: token.Line, Column: token.Column},
			Token: token,
		}
	} else {
		return nil, fmt.Errorf("expected pattern identifier or '*', got %s", p.current().Type)
	}

	// Parse colon
	colonToken, err := p.consume(lexer.COLON, "expected ':' after pattern")
	if err != nil {
		return nil, err
	}

	// **FIXED**: Parse command content - handle both single commands and blocks
	var commands []ast.CommandContent

	// Check if pattern branch has explicit block syntax: pattern: { ... }
	if p.match(lexer.LBRACE) {
		p.advance() // consume '{'
		blockCommands, err := p.parseBlockContent()
		if err != nil {
			return nil, err
		}
		_, err = p.consume(lexer.RBRACE, "expected '}' to close pattern branch block")
		if err != nil {
			return nil, err
		}
		commands = blockCommands
	} else {
		// Single command without braces: pattern: command
		content, err := p.parseCommandContent(true) // Pattern branches are always in block context
		if err != nil {
			return nil, err
		}
		commands = []ast.CommandContent{content}
	}

	return &ast.PatternBranch{
		Pattern:    pattern,
		Commands:   commands, // Now properly supports multiple commands
		Pos:        ast.Position{Line: startPos.Line, Column: startPos.Column},
		ColonToken: colonToken,
	}, nil
}

// parseBlockContent parses multiple content items within a block
// **FIXED**: Now properly handles multiple consecutive SHELL_TEXT tokens as separate commands
func (p *Parser) parseBlockContent() ([]ast.CommandContent, error) {
	var contentItems []ast.CommandContent

	for !p.match(lexer.RBRACE) && !p.isAtEnd() {
		p.skipWhitespaceAndComments()
		if p.match(lexer.RBRACE) {
			break
		}

		// Check for pattern decorators (@when, @try)
		if p.isPatternDecorator() {
			pattern, err := p.parsePatternContent()
			if err != nil {
				return nil, err
			}
			contentItems = append(contentItems, pattern)
			continue
		}

		// Check for block decorators
		if p.isBlockDecorator() {
			decorator, err := p.parseDecorator()
			if err != nil {
				return nil, err
			}

			// Handle different decorator types
			switch d := decorator.(type) {
			case *ast.BlockDecorator:
				// Parse the block content for block decorators
				if p.match(lexer.LBRACE) {
					p.advance() // consume '{'
					nestedContent, err := p.parseBlockContent()
					if err != nil {
						return nil, err
					}
					_, err = p.consume(lexer.RBRACE, "expected '}' after block decorator content")
					if err != nil {
						return nil, err
					}
					d.Content = nestedContent
				} else {
					// Parse single shell content
					content, err := p.parseShellContent(true)
					if err != nil {
						return nil, err
					}
					d.Content = []ast.CommandContent{content}
				}
				contentItems = append(contentItems, d)
			case *ast.PatternDecorator:
				// Pattern decorators shouldn't appear here
				return nil, fmt.Errorf("pattern decorators should be handled separately")
			default:
				return nil, fmt.Errorf("unexpected decorator type in block context")
			}
			continue
		}

		// **CRITICAL FIX**: Parse consecutive SHELL_TEXT tokens as separate commands
		// This implements the spec requirement: "newlines create multiple commands everywhere"
		if p.match(lexer.SHELL_TEXT) {
			shellContent, err := p.parseShellContent(true)
			if err != nil {
				return nil, err
			}

			// Only add non-empty shell content
			if len(shellContent.Parts) > 0 {
				contentItems = append(contentItems, shellContent)
			}
			continue
		}

		// If we get here, we have an unexpected token
		break
	}

	return contentItems, nil
}

// parseShellContent parses a single shell content item (one SHELL_TEXT token)
// **UPDATED**: Now parses only one SHELL_TEXT token to create separate content items
func (p *Parser) parseShellContent(inBlock bool) (*ast.ShellContent, error) {
	startPos := p.current()
	var parts []ast.ShellPart

	// Parse only one SHELL_TEXT token at a time
	if p.match(lexer.SHELL_TEXT) {
		shellToken := p.current()
		p.advance()

		extractedParts, err := p.extractInlineDecorators(shellToken.Value)
		if err != nil {
			return nil, err
		}

		parts = append(parts, extractedParts...)
	}

	return &ast.ShellContent{
		Parts: parts,
		Pos:   ast.Position{Line: startPos.Line, Column: startPos.Column},
	}, nil
}

// extractInlineDecorators extracts function decorators from shell text using stdlib registry validation
func (p *Parser) extractInlineDecorators(shellText string) ([]ast.ShellPart, error) {
	var parts []ast.ShellPart
	textStart := 0

	for i := 0; i < len(shellText); {
		// Look for @ symbol
		atPos := strings.IndexByte(shellText[i:], '@')
		if atPos == -1 {
			// No more @ symbols, add remaining text if any
			if textStart < len(shellText) {
				parts = append(parts, &ast.TextPart{Text: shellText[textStart:]})
			}
			break
		}

		// Absolute position of @
		absAtPos := i + atPos

		// Try to extract decorator starting at @
		decorator, newPos, found := p.extractFunctionDecorator(shellText, absAtPos)
		if found {
			// Add any text before the decorator
			if absAtPos > textStart {
				parts = append(parts, &ast.TextPart{Text: shellText[textStart:absAtPos]})
			}
			// Add the decorator
			parts = append(parts, decorator)
			// Update positions
			i = newPos
			textStart = newPos
		} else {
			// Not a valid function decorator, continue scanning after this @
			i = absAtPos + 1
		}
	}

	return parts, nil
}

// extractFunctionDecorator extracts a function decorator starting at position i using stdlib validation
// Returns the decorator, new position, and whether a decorator was found
func (p *Parser) extractFunctionDecorator(shellText string, i int) (*ast.FunctionDecorator, int, bool) {
	if i >= len(shellText) || shellText[i] != '@' {
		return nil, i, false
	}

	// Look for decorator name after @
	start := i + 1 // Skip @
	nameStart := start

	// First character must be a letter
	if start >= len(shellText) || !isLetter(rune(shellText[start])) {
		return nil, i, false
	}
	start++

	// Rest can be letters, digits, underscore, or hyphen
	for start < len(shellText) && (isLetter(rune(shellText[start])) || isDigit(rune(shellText[start])) || shellText[start] == '_' || shellText[start] == '-') {
		start++
	}

	decoratorName := shellText[nameStart:start]

	// **CRITICAL FIX**: Use stdlib registry to validate function decorators
	if !stdlib.IsFunctionDecorator(decoratorName) {
		return nil, i, false
	}

	// Look for opening parenthesis
	if start >= len(shellText) || shellText[start] != '(' {
		// Function decorators require parentheses
		return nil, i, false
	}

	// Find matching closing parenthesis
	start++ // Skip opening (
	parenCount := 1
	argStart := start

	for start < len(shellText) && parenCount > 0 {
		switch shellText[start] {
		case '(':
			parenCount++
		case ')':
			parenCount--
		}
		start++
	}

	if parenCount != 0 {
		// Unmatched parentheses
		return nil, i, false
	}

	// Extract argument text (between parentheses)
	argEnd := start - 1 // Position of closing ')'
	argText := shellText[argStart:argEnd]

	// Parse the argument
	var args []ast.Expression
	if strings.TrimSpace(argText) != "" {
		trimmed := strings.TrimSpace(argText)

		// Handle quoted strings
		if (strings.HasPrefix(trimmed, `"`) && strings.HasSuffix(trimmed, `"`)) ||
			(strings.HasPrefix(trimmed, `'`) && strings.HasSuffix(trimmed, `'`)) ||
			(strings.HasPrefix(trimmed, "`") && strings.HasSuffix(trimmed, "`")) {
			// String literal - remove quotes
			unquoted := trimmed[1 : len(trimmed)-1]
			args = append(args, &ast.StringLiteral{Value: unquoted})
		} else {
			// Identifier
			args = append(args, &ast.Identifier{Name: trimmed})
		}
	}

	decorator := &ast.FunctionDecorator{
		Name: decoratorName,
		Args: args,
		Pos:  ast.Position{Line: 1, Column: i + 1}, // Approximate position
	}

	return decorator, start, true
}

// Helper functions for character classification
func isLetter(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func isDigit(ch rune) bool {
	return ch >= '0' && ch <= '9'
}

// --- Expression and Literal Parsing ---

// parseExpression parses any valid expression (literals, identifiers, function decorators).
// This is used for parsing decorator arguments, where an identifier can be complex.
func (p *Parser) parseExpression() (ast.Expression, error) {
	switch p.current().Type {
	case lexer.STRING:
		tok := p.current()
		p.advance()
		return &ast.StringLiteral{Value: tok.Value, Raw: tok.Raw, StringToken: tok}, nil
	case lexer.NUMBER:
		tok := p.current()
		p.advance()
		return &ast.NumberLiteral{Value: tok.Value, Token: tok}, nil
	case lexer.DURATION:
		tok := p.current()
		p.advance()
		return &ast.DurationLiteral{Value: tok.Value, Token: tok}, nil
	case lexer.BOOLEAN:
		tok := p.current()
		p.advance()
		return &ast.BooleanLiteral{Value: tok.Value == "true", Token: tok}, nil
	case lexer.IDENTIFIER:
		// For decorator arguments, an "identifier" can be a complex value.
		// This function consumes tokens until a separator is found.
		return p.parseDecoratorArgument()
	case lexer.AT:
		// **SPEC COMPLIANCE**: REJECT function decorators in decorator arguments
		return nil, fmt.Errorf("function decorators (@var, @env, etc.) are not allowed as decorator arguments. Use direct variable names instead (e.g., @timeout(DURATION) not @timeout(@var(DURATION)))")
	}
	return nil, fmt.Errorf("unexpected token %s, expected an expression (literal or identifier)", p.current().Type)
}

// parseDecoratorArgument handles complex decorator arguments.
// **UPDATED**: This version is now robust and handles nested parentheses correctly,
// ensuring it consumes the entire intended argument without overrunning.
func (p *Parser) parseDecoratorArgument() (ast.Expression, error) {
	startToken := p.current()
	startOffset := startToken.Span.Start.Offset

	// We need to find the end of the argument, which is either a comma or a closing parenthesis
	// at the same parenthesis level.
	parenDepth := 0
	searchPos := p.pos

	for searchPos < len(p.tokens) {
		tok := p.tokens[searchPos]
		if (tok.Type == lexer.COMMA || tok.Type == lexer.RPAREN) && parenDepth == 0 {
			break
		}
		switch tok.Type {
		case lexer.LPAREN:
			parenDepth++
		case lexer.RPAREN:
			parenDepth--
		}
		searchPos++
	}

	// The argument ends at the start of the terminator token, or the end of the last token if at EOF.
	var endOffset int
	if searchPos < len(p.tokens) {
		endOffset = p.tokens[searchPos].Span.Start.Offset
		// Trim trailing space
		for endOffset > startOffset && strings.ContainsRune(" \t", rune(p.input[endOffset-1])) {
			endOffset--
		}
	} else {
		endOffset = p.tokens[len(p.tokens)-1].Span.End.Offset // EOF
	}

	value := p.input[startOffset:endOffset]

	// Advance parser position past the consumed tokens for the argument.
	p.pos = searchPos

	return &ast.Identifier{
		Name:  value,
		Token: lexer.Token{Value: value, Line: startToken.Line, Column: startToken.Column},
	}, nil
}

// --- Variable Parsing ---

// parseVariableDecl parses a variable declaration.
// **SPEC COMPLIANCE**: Now enforces that values must be string, number, duration, or boolean literals
func (p *Parser) parseVariableDecl() (*ast.VariableDecl, error) {
	startPos := p.current()
	_, err := p.consume(lexer.VAR, "expected 'var'")
	if err != nil {
		return nil, err
	}

	name, err := p.consume(lexer.IDENTIFIER, "expected variable name")
	if err != nil {
		return nil, err
	}
	_, err = p.consume(lexer.EQUALS, "expected '=' after variable name")
	if err != nil {
		return nil, err
	}

	// Parse variable value - must be a literal (string, number, duration, or boolean)
	value, err := p.parseVariableValue()
	if err != nil {
		return nil, err
	}

	return &ast.VariableDecl{
		Name:      name.Value,
		Value:     value,
		Pos:       ast.Position{Line: startPos.Line, Column: startPos.Column},
		NameToken: name,
	}, nil
}

// parseVariableValue parses variable values, now restricted to literals only.
// **SPEC COMPLIANCE**: Only allows the 4 supported types: string, number, duration, boolean
func (p *Parser) parseVariableValue() (ast.Expression, error) {
	startToken := p.current()

	// Handle standard literals only - no unquoted strings allowed
	switch startToken.Type {
	case lexer.STRING:
		p.advance()
		return &ast.StringLiteral{Value: startToken.Value, Raw: startToken.Raw, StringToken: startToken}, nil
	case lexer.NUMBER:
		p.advance()
		return &ast.NumberLiteral{Value: startToken.Value, Token: startToken}, nil
	case lexer.DURATION:
		p.advance()
		return &ast.DurationLiteral{Value: startToken.Value, Token: startToken}, nil
	case lexer.BOOLEAN:
		p.advance()
		return &ast.BooleanLiteral{Value: startToken.Value == "true", Token: startToken}, nil
	default:
		// **SPEC COMPLIANCE**: No longer allow arbitrary unquoted strings
		return nil, fmt.Errorf("variable value must be a quoted string, number, duration, or boolean literal at line %d, col %d (got %s)",
			startToken.Line, startToken.Column, startToken.Type)
	}
}

func (p *Parser) parseVarGroup() (*ast.VarGroup, error) {
	startPos := p.current()
	_, err := p.consume(lexer.VAR, "expected 'var'")
	if err != nil {
		return nil, err
	}
	openParen, err := p.consume(lexer.LPAREN, "expected '(' for var group")
	if err != nil {
		return nil, err
	}

	var variables []ast.VariableDecl
	for !p.match(lexer.RPAREN) && !p.isAtEnd() {
		p.skipWhitespaceAndComments()
		if p.match(lexer.RPAREN) {
			break
		}
		if p.current().Type != lexer.IDENTIFIER {
			p.addError(fmt.Errorf("expected variable name inside var group, got %s", p.current().Type))
			p.synchronize()
			continue
		}

		varDecl, err := p.parseGroupedVariableDecl()
		if err != nil {
			return nil, err // Be strict inside var groups
		}
		variables = append(variables, *varDecl)
		p.skipWhitespaceAndComments()
	}

	closeParen, err := p.consume(lexer.RPAREN, "expected ')' to close var group")
	if err != nil {
		return nil, err
	}

	return &ast.VarGroup{
		Variables:  variables,
		Pos:        ast.Position{Line: startPos.Line, Column: startPos.Column},
		OpenParen:  openParen,
		CloseParen: closeParen,
	}, nil
}

// parseGroupedVariableDecl is a helper for parsing `NAME = VALUE` lines within a `var (...)` block.
func (p *Parser) parseGroupedVariableDecl() (*ast.VariableDecl, error) {
	name, err := p.consume(lexer.IDENTIFIER, "expected variable name")
	if err != nil {
		return nil, err
	}
	_, err = p.consume(lexer.EQUALS, "expected '=' after variable name")
	if err != nil {
		return nil, err
	}

	// Use the same restricted value parsing logic.
	value, err := p.parseVariableValue()
	if err != nil {
		return nil, err
	}

	return &ast.VariableDecl{
		Name:      name.Value,
		Value:     value,
		Pos:       ast.Position{Line: name.Line, Column: name.Column},
		NameToken: name,
	}, nil
}

// --- Decorator Parsing ---

// parseDecorator parses a single decorator and returns the appropriate AST node type
func (p *Parser) parseDecorator() (ast.CommandContent, error) {
	startPos := p.current()
	atToken, _ := p.consume(lexer.AT, "expected '@'")

	// Get decorator name
	var nameToken lexer.Token
	var err error

	if p.current().Type == lexer.IDENTIFIER {
		nameToken, err = p.consume(lexer.IDENTIFIER, "expected decorator name")
	} else {
		// Handle special cases where keywords appear as decorator names
		nameToken = p.current()
		if !p.isValidDecoratorName(nameToken) {
			return nil, fmt.Errorf("expected decorator name, got %s", nameToken.Type)
		}
		p.advance()
	}

	if err != nil {
		return nil, err
	}

	decoratorName := nameToken.Value
	if nameToken.Type != lexer.IDENTIFIER {
		decoratorName = strings.ToLower(nameToken.Value)
	}

	// Parse arguments if present
	var args []ast.Expression
	if p.match(lexer.LPAREN) {
		p.advance() // consume '('
		args, err = p.parseArgumentList()
		if err != nil {
			return nil, err
		}
		_, err = p.consume(lexer.RPAREN, "expected ')' after decorator arguments")
		if err != nil {
			return nil, err
		}
	}

	// Determine decorator type from registry and create appropriate AST node
	if stdlib.IsBlockDecorator(decoratorName) {
		return &ast.BlockDecorator{
			Name:      decoratorName,
			Args:      args,
			Content:   nil, // Will be filled in by caller
			Pos:       ast.Position{Line: startPos.Line, Column: startPos.Column},
			AtToken:   atToken,
			NameToken: nameToken,
		}, nil
	} else if stdlib.IsPatternDecorator(decoratorName) {
		return &ast.PatternDecorator{
			Name:      decoratorName,
			Args:      args,
			Patterns:  nil, // Will be filled in by caller
			Pos:       ast.Position{Line: startPos.Line, Column: startPos.Column},
			AtToken:   atToken,
			NameToken: nameToken,
		}, nil
	} else if stdlib.IsFunctionDecorator(decoratorName) {
		return &ast.FunctionDecorator{
			Name:      decoratorName,
			Args:      args,
			Pos:       ast.Position{Line: startPos.Line, Column: startPos.Column},
			AtToken:   atToken,
			NameToken: nameToken,
		}, nil
	} else {
		return nil, fmt.Errorf("unknown decorator: @%s", decoratorName)
	}
}

// parseBlockDecorator removed - block decorators are now created directly in parseBlockDecoratorContent

// parsePatternDecorator removed - pattern decorators are now created directly in parsePatternContent

// isValidDecoratorName checks if a token can be used as a decorator name
func (p *Parser) isValidDecoratorName(token lexer.Token) bool {
	switch token.Type {
	case lexer.IDENTIFIER:
		return true
	case lexer.VAR:
		// "var" can be used as a decorator name for @var()
		return true
	case lexer.WHEN, lexer.TRY:
		// Pattern decorator keywords
		return true
	default:
		return false
	}
}

// parseArgumentList parses a comma-separated list of expressions.
func (p *Parser) parseArgumentList() ([]ast.Expression, error) {
	var args []ast.Expression
	if p.match(lexer.RPAREN) {
		return args, nil // No arguments
	}

	for {
		arg, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
		if !p.match(lexer.COMMA) {
			break
		}
		p.advance() // consume ','
	}
	return args, nil
}

// parsePatternBranchesInBlock parses pattern branches directly from the token stream
func (p *Parser) parsePatternBranchesInBlock() ([]ast.PatternBranch, error) {
	var patterns []ast.PatternBranch

	for !p.match(lexer.RBRACE) && !p.isAtEnd() {
		p.skipWhitespaceAndComments()
		if p.match(lexer.RBRACE) {
			break
		}

		branch, err := p.parsePatternBranch()
		if err != nil {
			return nil, err
		}
		patterns = append(patterns, *branch)

		p.skipWhitespaceAndComments()
	}

	return patterns, nil
}

// --- Utility and Helper Methods ---

func (p *Parser) advance() lexer.Token {
	if !p.isAtEnd() {
		p.pos++
	}
	return p.previous()
}

func (p *Parser) current() lexer.Token  { return p.tokens[p.pos] }
func (p *Parser) previous() lexer.Token { return p.tokens[p.pos-1] }
func (p *Parser) peek() lexer.Token     { return p.tokens[p.pos+1] }

func (p *Parser) isAtEnd() bool { return p.current().Type == lexer.EOF }

func (p *Parser) match(types ...lexer.TokenType) bool {
	for _, t := range types {
		if p.current().Type == t {
			return true
		}
	}
	return false
}

// formatError creates a detailed error message with source context
func (p *Parser) formatError(message string, token lexer.Token) error {
	lines := strings.Split(p.input, "\n")
	lineNum := token.Line
	colNum := token.Column

	var errorMsg strings.Builder
	errorMsg.WriteString(fmt.Sprintf("parsing failed:\n- %s\n\n", message))

	// Show context around the error
	startLine := max(1, lineNum-1)
	endLine := min(len(lines), lineNum+1)

	maxLineNumWidth := len(strconv.Itoa(endLine))

	for i := startLine; i <= endLine; i++ {
		lineContent := ""
		if i <= len(lines) {
			lineContent = lines[i-1] // lines are 0-indexed, but line numbers are 1-indexed
		}

		lineNumStr := fmt.Sprintf("%*d", maxLineNumWidth, i)

		if i == lineNum {
			// This is the error line - highlight it
			errorMsg.WriteString(fmt.Sprintf(" --> %s | %s\n", lineNumStr, lineContent))

			// Add pointer to the exact column
			padding := strings.Repeat(" ", maxLineNumWidth+3+colNum-1) // account for " --> " and column position
			errorMsg.WriteString(fmt.Sprintf("     %s | %s^\n", strings.Repeat(" ", maxLineNumWidth), padding))
			errorMsg.WriteString(fmt.Sprintf("     %s | %s%s\n", strings.Repeat(" ", maxLineNumWidth), padding, "unexpected "+token.Type.String()))
		} else {
			// Context line
			errorMsg.WriteString(fmt.Sprintf("     %s | %s\n", lineNumStr, lineContent))
		}
	}

	return fmt.Errorf("%s", errorMsg.String())
}

// max returns the larger of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (p *Parser) consume(t lexer.TokenType, message string) (lexer.Token, error) {
	if p.match(t) {
		tok := p.current()
		p.advance()
		return tok, nil
	}
	return lexer.Token{}, p.formatError(message, p.current())
}

func (p *Parser) skipWhitespaceAndComments() {
	// NEWLINE tokens no longer exist - they're handled as whitespace by lexer
	for p.match(lexer.COMMENT, lexer.MULTILINE_COMMENT) {
		p.advance()
	}
}

// isPatternDecorator checks if the current position starts a pattern decorator.
func (p *Parser) isPatternDecorator() bool {
	if p.current().Type != lexer.AT {
		return false
	}
	if p.pos+1 < len(p.tokens) {
		nextToken := p.tokens[p.pos+1]
		var name string

		switch nextToken.Type {
		case lexer.IDENTIFIER:
			name = nextToken.Value
		case lexer.WHEN:
			name = "when"
		case lexer.TRY:
			name = "try"
		default:
			return false
		}

		return stdlib.IsPatternDecorator(name)
	}
	return false
}

// isBlockDecorator checks if the current position starts a block decorator.
func (p *Parser) isBlockDecorator() bool {
	if p.current().Type != lexer.AT {
		return false
	}
	if p.pos+1 < len(p.tokens) {
		nextToken := p.tokens[p.pos+1]
		var name string

		if nextToken.Type == lexer.IDENTIFIER {
			name = nextToken.Value
		} else {
			return false
		}

		return stdlib.IsBlockDecorator(name)
	}
	return false
}

// addError records an error and allows parsing to continue.
func (p *Parser) addError(err error) {
	p.errors = append(p.errors, err.Error())
}

// synchronize advances the parser until it finds a probable statement boundary,
// allowing it to recover from an error and report more than one error per file.
func (p *Parser) synchronize() {
	p.advance()
	for !p.isAtEnd() {
		// NEWLINE tokens no longer exist - removed synchronization point
		// A new top-level keyword is also a good place.
		switch p.current().Type {
		case lexer.VAR, lexer.WATCH, lexer.STOP:
			return
		}
		p.advance()
	}
}
