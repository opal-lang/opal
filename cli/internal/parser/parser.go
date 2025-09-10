package parser

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/aledsdavies/devcmd/cli/internal/lexer"
	"github.com/aledsdavies/devcmd/core/decorators"
	"github.com/aledsdavies/devcmd/core/types"
	"github.com/aledsdavies/devcmd/runtime/ast"
)

// Parser implements a fast, spec-compliant recursive descent parser for the Devcmd language.
// It trusts the lexer to have correctly handled whitespace and tokenization, focusing
// purely on assembling the Abstract Syntax Tree (AST).
type Parser struct {
	input  string // The raw input string for accurate value slicing
	tokens []types.Token
	pos    int // current position in the token slice

	// errors is a slice of errors encountered during parsing.
	// This allows for better error reporting by collecting multiple errors.
	errors []string

	// program is the AST being built during parsing (for variable type lookups)
	program *ast.Program

	// logger for debug output
	logger *slog.Logger
}

// Parse tokenizes and parses the input from an io.Reader into a complete AST.
// It returns the Program node and any errors encountered.
func Parse(reader io.Reader) (*ast.Program, error) {
	// Read the input to store for error reporting
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}
	input := string(data)

	// Create debug logger - check if DEVCMD_DEBUG_PARSER environment variable is set
	logLevel := slog.LevelInfo
	if os.Getenv("DEVCMD_DEBUG_PARSER") != "" {
		logLevel = slog.LevelDebug
	}

	// Custom parser-friendly handler
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Remove timestamp for cleaner parser output
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			// Simplify level display
			if a.Key == slog.LevelKey {
				return slog.Attr{}
			}
			return a
		},
	}))

	lex := lexer.New(strings.NewReader(input))
	p := &Parser{
		input:  input, // Store the raw input
		tokens: lex.TokenizeToSlice(),
		logger: logger,
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
	p.program = program // Store reference for variable type lookups

	for !p.isAtEnd() {
		p.skipWhitespaceAndComments()
		if p.isAtEnd() {
			break
		}

		switch p.current().Type {
		case types.VAR:
			if p.peek().Type == types.LPAREN {
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
		case types.IDENTIFIER, types.WATCH, types.STOP:
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
			p.addError(p.NewSyntaxError(fmt.Sprintf("unexpected token %s at top level, expected 'var' or command declaration", p.current().Type.String())))
			p.synchronize()
		}
	}

	return program
}

// parseCommandDecl parses a full command declaration.
// CommandDecl = { Decorator }* [ "watch" | "stop" ] IDENTIFIER ":" CommandBody
func (p *Parser) parseCommandDecl() (*ast.CommandDecl, error) {
	p.logger.Debug("→ parseCommandDecl", "token", p.current().Type.String())
	startPos := p.current()

	// 1. Parse command type (watch, stop, or regular)
	cmdType := ast.Command
	var typeToken *types.Token
	if p.match(types.WATCH) {
		p.logger.Debug("  found WATCH")
		cmdType = ast.WatchCommand
		token := p.current()
		typeToken = &token
		p.advance()
	} else if p.match(types.STOP) {
		p.logger.Debug("  found STOP")
		cmdType = ast.StopCommand
		token := p.current()
		typeToken = &token
		p.advance()
	}

	// 2. Parse command name
	p.logger.Debug("  parsing name", "token", p.current().Type.String())
	nameToken, err := p.consume(types.IDENTIFIER, "expected command name")
	if err != nil {
		p.logger.Debug("  ✗ name failed", "error", err)
		return nil, err
	}
	name := nameToken.Value
	p.logger.Debug("  ✓ name", "value", name)

	// 3. Parse colon
	p.logger.Debug("  parsing colon", "token", p.current().Type.String())
	colonToken, err := p.consume(types.COLON, "expected ':' after command name")
	if err != nil {
		p.logger.Debug("  ✗ colon failed", "error", err)
		return nil, err
	}

	// 4. Parse command body (this will handle post-colon decorators and syntax sugar)
	p.logger.Debug("  parsing body", "token", p.current().Type.String())
	body, err := p.parseCommandBody()
	if err != nil {
		p.logger.Debug("  ✗ body failed", "error", err)
		return nil, err
	}

	p.logger.Debug("← parseCommandDecl ✓", "name", name)
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
	p.logger.Debug("  → parseBody", "token", p.current().Type.String())
	startPos := p.current()

	// **FIXED**: Check for decorator syntax sugar: @decorator(args) { ... }
	// This should be equivalent to: { @decorator(args) { ... } }
	if p.match(types.AT) {
		p.logger.Debug("    decorator syntax sugar")
		// Save position in case we need to backtrack
		savedPos := p.pos

		// Try to parse a single decorator after the colon
		decorator, err := p.parseDecorator()
		if err != nil {
			p.logger.Debug("    ✗ decorator failed", "error", err)
			return nil, err
		}

		// After decorators, we expect either:
		// 1. A block { ... } (syntax sugar - should be treated as IsBlock=true)
		// 2. Simple shell content (only valid for function decorators)

		if p.match(types.LBRACE) {
			// **SYNTAX SUGAR**: @decorator(args) { ... } becomes { @decorator(args) { ... } }
			openBrace, _ := p.consume(types.LBRACE, "") // already checked

			// Parse content differently based on decorator type
			switch d := decorator.(type) {
			case *ast.BlockDecorator:
				blockContent, err := p.parseBlockContent() // Parse multiple content items
				if err != nil {
					return nil, err
				}
				closeBrace, err := p.consume(types.RBRACE, "expected '}' to close command block")
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
				closeBrace, err := p.consume(types.RBRACE, "expected '}' to close pattern block")
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
			// Decorator without braces - check if it's an action decorator
			switch decorator.(type) {
			case *ast.ActionDecorator:
				// Valid standalone decorators
			default:
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
	if p.match(types.LBRACE) {
		p.logger.Debug("    explicit block")
		openBrace, _ := p.consume(types.LBRACE, "") // already checked
		contentItems, err := p.parseBlockContent()  // Parse multiple content items
		if err != nil {
			p.logger.Debug("    ✗ block content failed", "error", err)
			return nil, err
		}
		closeBrace, err := p.consume(types.RBRACE, "expected '}' to close command block")
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
	p.logger.Debug("    simple command", "token", p.current().Type.String())
	content, err := p.parseCommandContent(false) // Pass inBlock=false
	if err != nil {
		p.logger.Debug("    ✗ simple content failed", "error", err)
		return nil, err
	}
	p.logger.Debug("  ← parseBody ✓")
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

	// Must be shell content without decorators (shell chains are not simple)
	if shell, ok := contentItems[0].(*ast.ShellContent); ok {
		// Check if it contains only text parts or value/action decorators (no block decorators)
		for _, part := range shell.Parts {
			switch part.(type) {
			case *ast.ValueDecorator, *ast.ActionDecorator:
				// Value and action decorators are allowed in simple content
			case *ast.TextPart:
				// Text parts are allowed
			default:
				// Block decorators and other types are not allowed in simple content
				return false
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
	p.logger.Debug("      → parseContent", "token", p.current().Type.String())
	// Check for pattern decorators (@when, @try)
	if p.isPatternDecorator() {
		p.logger.Debug("        pattern decorator")
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
			if p.match(types.LBRACE) {
				p.advance() // consume '{'
				contentItems, err := p.parseBlockContent()
				if err != nil {
					return nil, err
				}
				_, err = p.consume(types.RBRACE, "expected '}' after block decorator content")
				if err != nil {
					return nil, err
				}
				d.Content = contentItems
			} else {
				return nil, p.NewMissingTokenError("'{' after block decorator @" + d.Name)
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
	p.logger.Debug("        shell content")
	return p.parseShellContent(inBlock)
}

// parsePatternContent parses pattern-matching decorator content (@when, @try)
// This handles syntax like: @when(VAR) { pattern: command; pattern: command }
func (p *Parser) parsePatternContent() (*ast.PatternDecorator, error) {
	startPos := p.current()

	// Parse @ symbol
	atToken, err := p.consume(types.AT, "expected '@' to start pattern decorator")
	if err != nil {
		return nil, err
	}

	// Parse decorator name
	nameToken, err := p.consume(types.IDENTIFIER, "expected decorator name after '@'")
	if err != nil {
		return nil, err
	}
	decoratorName := nameToken.Value

	// Step 1: Check if decorator exists in registry and is a pattern decorator
	decorator, decoratorType, err := decorators.GetAny(decoratorName)
	if err != nil || decoratorType != decorators.PatternType {
		return nil, p.NewInvalidError("unknown pattern decorator @" + decoratorName)
	}

	// Step 2: Get parameter schema
	paramSchema := decorator.ParameterSchema()

	// Parse arguments if present
	var params []ast.NamedParameter
	if p.match(types.LPAREN) {
		p.advance() // consume '('
		params, err = p.parseParameterList(paramSchema)
		if err != nil {
			return nil, err
		}
		_, err = p.consume(types.RPAREN, "expected ')' after decorator arguments")
		if err != nil {
			return nil, err
		}
	}

	// Step 3: Validate parameters using decorator schema
	if err := p.validateDecoratorParameters(decorator, params, decoratorName); err != nil {
		return nil, err
	}

	// Expect opening brace
	_, err = p.consume(types.LBRACE, "expected '{' after pattern decorator")
	if err != nil {
		return nil, err
	}

	// Parse pattern branches
	var patterns []ast.PatternBranch
	for !p.match(types.RBRACE) && !p.isAtEnd() {
		p.skipWhitespaceAndComments()
		if p.match(types.RBRACE) {
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
	_, err = p.consume(types.RBRACE, "expected '}' to close pattern block")
	if err != nil {
		return nil, err
	}

	// Validate pattern branches using decorator schema
	if patternDecorator, ok := decorator.(decorators.PatternDecorator); ok {
		if err := p.validatePatternBranches(patternDecorator, patterns, decoratorName); err != nil {
			return nil, err
		}
	}

	return &ast.PatternDecorator{
		Name:      decoratorName,
		Args:      params,
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
	if p.match(types.IDENTIFIER) {
		token := p.current()
		p.advance()

		// Check if this is the "default" wildcard pattern
		if token.Value == "default" {
			pattern = &ast.WildcardPattern{
				Pos:   ast.Position{Line: token.Line, Column: token.Column},
				Token: token,
			}
		} else {
			pattern = &ast.IdentifierPattern{
				Name:  token.Value,
				Pos:   ast.Position{Line: token.Line, Column: token.Column},
				Token: token,
			}
		}
	} else {
		return nil, p.NewSyntaxError(fmt.Sprintf("expected pattern identifier, got %s", p.current().Type.String()))
	}

	// Parse colon
	colonToken, err := p.consume(types.COLON, "expected ':' after pattern")
	if err != nil {
		return nil, err
	}

	// **FIXED**: Parse command content - handle both single commands and blocks
	var commands []ast.CommandContent

	// Check if pattern branch has explicit block syntax: pattern: { ... }
	if p.match(types.LBRACE) {
		p.advance() // consume '{'
		blockCommands, err := p.parseBlockContent()
		if err != nil {
			return nil, err
		}
		_, err = p.consume(types.RBRACE, "expected '}' to close pattern branch block")
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

	for !p.match(types.RBRACE) && !p.isAtEnd() {
		p.skipWhitespaceAndComments()
		if p.match(types.RBRACE) {
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
		p.logger.Debug("      🔍 parseBlockContent: checking decorator", "pos", p.pos, "token", p.current().Type.String(), "value", p.current().Value)
		if p.isBlockDecorator() {
			p.logger.Debug("      ✅ parseBlockContent: found block decorator", "decorator", p.tokens[p.pos+1].Value)
			decorator, err := p.parseDecorator()
			if err != nil {
				return nil, err
			}

			// Handle different decorator types
			switch d := decorator.(type) {
			case *ast.BlockDecorator:
				// Parse the block content for block decorators
				if p.match(types.LBRACE) {
					p.logger.Debug("      📦 parseBlockContent: parsing block decorator content", "decorator", d.Name)
					p.advance() // consume '{'
					nestedContent, err := p.parseBlockContent()
					if err != nil {
						p.logger.Debug("      ❌ parseBlockContent: nested content failed", "decorator", d.Name, "error", err)
						return nil, err
					}
					_, err = p.consume(types.RBRACE, "expected '}' after block decorator content")
					if err != nil {
						p.logger.Debug("      ❌ parseBlockContent: missing closing brace", "decorator", d.Name, "error", err)
						return nil, err
					}
					p.logger.Debug("      ✅ parseBlockContent: block decorator complete", "decorator", d.Name, "contentCount", len(nestedContent))
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

		// **CRITICAL FIX**: Parse consecutive SHELL_TEXT tokens and decorator sequences as separate commands
		// This implements the spec requirement: "newlines create multiple commands everywhere"
		// NOTE: Check for @ tokens that are NOT block/pattern decorators (those are handled above)
		if p.match(types.SHELL_TEXT) || (p.match(types.AT) && !p.isBlockDecorator() && !p.isPatternDecorator()) {
			p.logger.Debug("      🛠️ parseBlockContent: parsing shell content", "token", p.current().Type.String(), "value", p.current().Value)
			shellContent, err := p.parseShellContent(true)
			if err != nil {
				return nil, err
			}

			// Only add non-empty shell content
			isEmpty := false
			if sc, ok := shellContent.(*ast.ShellContent); ok {
				isEmpty = len(sc.Parts) == 0
			} else if chain, ok := shellContent.(*ast.ShellChain); ok {
				isEmpty = len(chain.Elements) == 0
			}

			if !isEmpty {
				contentItems = append(contentItems, shellContent)
			}
			continue
		}

		// If we get here, we have an unexpected token
		break
	}

	return contentItems, nil
}

// parseShellContent parses a complete shell command from the new lexer token sequences
// Handles: SHELL_TEXT + AT + IDENTIFIER + LPAREN + params + RPAREN + SHELL_TEXT + ... + SHELL_END
// Also handles shell operators (&&, ||, |, >>) and creates ShellChain when operators are present
func (p *Parser) parseShellContent(inBlock bool) (ast.CommandContent, error) {
	p.logger.Debug("    → parseShell", "token", p.current().Type.String())
	startPos := p.current()

	// Check if this is a shell chain (has operators) by looking ahead
	if p.isShellChain() {
		p.logger.Debug("      shell chain")
		return p.parseShellChain()
	}

	// Parse simple shell content (no operators)
	var parts []ast.ShellPart
	p.logger.Debug("      simple shell")

	// Parse all parts of the shell command until SHELL_END
	for !p.match(types.SHELL_END) && !p.isAtEnd() && !p.match(types.RBRACE) {
		p.logger.Debug("      part", "token", p.current().Type.String())
		if p.match(types.SHELL_TEXT) {
			// Add shell text part
			parts = append(parts, &ast.TextPart{Text: p.current().Value})
			p.advance()
		} else if p.match(types.STRING_START) {
			// Handle strings within shell content - parse as string literal and add as text part
			p.logger.Debug("        string literal")
			stringLit, err := p.parseStringLiteral()
			if err != nil {
				return nil, err
			}
			// Use StringPart components directly in shell content since they now implement ShellPart
			for _, part := range stringLit.Parts {
				parts = append(parts, part)
			}
		} else if p.match(types.AT) {
			// Parse decorator in shell context - this can return ValueDecorator or ActionDecorator
			decorator, err := p.parseShellDecorator()
			if err != nil {
				return nil, err
			}
			parts = append(parts, decorator)
		} else {
			// Unexpected token - stop parsing
			p.logger.Debug("        unexpected, stopping", "token", p.current().Type.String())
			break
		}
	}

	// Consume SHELL_END if present
	if p.match(types.SHELL_END) {
		p.advance()
	}

	return &ast.ShellContent{
		Parts: parts,
		Pos:   ast.Position{Line: startPos.Line, Column: startPos.Column},
	}, nil
}

// isShellChain checks if the upcoming tokens represent a shell chain with operators
func (p *Parser) isShellChain() bool {
	// Look ahead to find shell operators
	pos := p.pos
	for pos < len(p.tokens) {
		token := p.tokens[pos]
		if token.Type == types.AND || token.Type == types.OR ||
			token.Type == types.PIPE || token.Type == types.APPEND {
			return true
		}
		if token.Type == types.SHELL_END || token.Type == types.RBRACE || token.Type == types.EOF {
			break
		}
		pos++
	}
	return false
}

// parseShellChain parses a shell chain with operators (&&, ||, |, >>)
func (p *Parser) parseShellChain() (*ast.ShellChain, error) {
	startPos := p.current()
	var elements []ast.ShellChainElement

	for {
		// Parse the current shell content element
		content, err := p.parseShellContentElement()
		if err != nil {
			return nil, err
		}

		element := ast.ShellChainElement{
			Content: content,
			Pos:     ast.Position{Line: startPos.Line, Column: startPos.Column},
		}

		// Check for operator after this element
		if p.match(types.AND) || p.match(types.OR) || p.match(types.PIPE) || p.match(types.APPEND) {
			operator := p.current().Value
			element.Operator = operator
			p.advance()

			// For >> operator, parse the target file if it's the next SHELL_TEXT
			if operator == ">>" && p.match(types.SHELL_TEXT) {
				element.Target = p.current().Value
				p.advance()
			}
		}

		elements = append(elements, element)

		// If no operator, we're done with the chain
		if element.Operator == "" {
			break
		}
	}

	// Consume SHELL_END if present
	if p.match(types.SHELL_END) {
		p.advance()
	}

	return &ast.ShellChain{
		Elements: elements,
		Pos:      ast.Position{Line: startPos.Line, Column: startPos.Column},
	}, nil
}

// parseShellContentElement parses a single shell content element (before an operator)
func (p *Parser) parseShellContentElement() (*ast.ShellContent, error) {
	startPos := p.current()
	var parts []ast.ShellPart

	// Parse parts until we hit an operator, SHELL_END, or block end
	for !p.match(types.SHELL_END) && !p.isAtEnd() && !p.match(types.RBRACE) &&
		!p.match(types.AND) && !p.match(types.OR) && !p.match(types.PIPE) && !p.match(types.APPEND) {

		if p.match(types.SHELL_TEXT) {
			// Add shell text part
			parts = append(parts, &ast.TextPart{Text: p.current().Value})
			p.advance()
		} else if p.match(types.AT) {
			// Parse decorator in shell context
			decorator, err := p.parseShellDecorator()
			if err != nil {
				return nil, err
			}
			parts = append(parts, decorator)
		} else if p.current().Type == types.STRING_START {
			// Handle strings within shell content - preserve as string parts
			stringLit, err := p.parseStringLiteral()
			if err != nil {
				return nil, err
			}
			// Add string parts directly to shell parts (they implement both interfaces)
			for _, part := range stringLit.Parts {
				parts = append(parts, part)
			}
		} else {
			// Unexpected token - stop parsing
			break
		}
	}

	return &ast.ShellContent{
		Parts: parts,
		Pos:   ast.Position{Line: startPos.Line, Column: startPos.Column},
	}, nil
}

// --- Expression and Literal Parsing ---

// --- Variable Parsing ---

// parseVariableDecl parses a variable declaration.
// **SPEC COMPLIANCE**: Now enforces that values must be string, number, duration, or boolean literals
func (p *Parser) parseVariableDecl() (*ast.VariableDecl, error) {
	startPos := p.current()
	_, err := p.consume(types.VAR, "expected 'var'")
	if err != nil {
		return nil, err
	}

	name, err := p.consume(types.IDENTIFIER, "expected variable name")
	if err != nil {
		return nil, err
	}
	_, err = p.consume(types.EQUALS, "expected '=' after variable name")
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
	case types.STRING_START:
		return p.parseStringLiteral()
	case types.NUMBER:
		p.advance()
		return &ast.NumberLiteral{Value: startToken.Value, Token: startToken}, nil
	case types.DURATION:
		p.advance()
		return &ast.DurationLiteral{Value: startToken.Value, Token: startToken}, nil
	case types.BOOLEAN:
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
	_, err := p.consume(types.VAR, "expected 'var'")
	if err != nil {
		return nil, err
	}
	openParen, err := p.consume(types.LPAREN, "expected '(' for var group")
	if err != nil {
		return nil, err
	}

	var variables []ast.VariableDecl
	for !p.match(types.RPAREN) && !p.isAtEnd() {
		p.skipWhitespaceAndComments()
		if p.match(types.RPAREN) {
			break
		}
		if p.current().Type != types.IDENTIFIER {
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

	closeParen, err := p.consume(types.RPAREN, "expected ')' to close var group")
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
	name, err := p.consume(types.IDENTIFIER, "expected variable name")
	if err != nil {
		return nil, err
	}
	_, err = p.consume(types.EQUALS, "expected '=' after variable name")
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

// parseShellDecorator parses a decorator in shell context and returns ShellPart (ValueDecorator or ActionDecorator)
func (p *Parser) parseShellDecorator() (ast.ShellPart, error) {
	// Reuse the same parsing logic as parseDecorator but return ShellPart
	startPos := p.current()
	atToken, _ := p.consume(types.AT, "expected '@'")

	// Get decorator name
	var nameToken types.Token
	var err error

	if p.current().Type == types.IDENTIFIER {
		nameToken, err = p.consume(types.IDENTIFIER, "expected decorator name")
	} else {
		return nil, p.NewSyntaxError("expected decorator name after '@'")
	}

	if err != nil {
		return nil, err
	}

	decoratorName := nameToken.Value
	if nameToken.Type != types.IDENTIFIER {
		decoratorName = strings.ToLower(nameToken.Value)
	}

	// Check if decorator exists in registry
	decorator, decoratorType, err := decorators.GetAny(decoratorName)
	if err != nil {
		return nil, p.NewInvalidError("unknown decorator @" + decoratorName)
	}

	// Get parameter schema from decorator
	paramSchema := decorator.ParameterSchema()

	// Parse parameters according to schema
	var params []ast.NamedParameter
	if p.match(types.LPAREN) {
		p.advance() // consume '('
		params, err = p.parseParameterList(paramSchema)
		if err != nil {
			return nil, err
		}
		_, err = p.consume(types.RPAREN, "expected ')' after decorator arguments")
		if err != nil {
			return nil, err
		}
	}

	// In shell context, ValueDecorator, ActionDecorator, and BlockDecorator are allowed per specification
	// See devcmd_specification.md examples: @parallel { ... }, @timeout(30s) { ... }
	switch decoratorType {
	case decorators.ValueType:
		return &ast.ValueDecorator{
			Name:      decoratorName,
			Args:      params,
			Pos:       ast.Position{Line: startPos.Line, Column: startPos.Column},
			AtToken:   atToken,
			NameToken: nameToken,
		}, nil
	case decorators.ActionType:
		return &ast.ActionDecorator{
			Name:      decoratorName,
			Args:      params,
			Pos:       ast.Position{Line: startPos.Line, Column: startPos.Column},
			AtToken:   atToken,
			NameToken: nameToken,
		}, nil
	case decorators.BlockType:
		// Block decorators are allowed in shell context per specification
		// Examples: setup: @parallel { ... }, server: @timeout(30s) { ... }
		return &ast.BlockDecorator{
			Name:      decoratorName,
			Args:      params,
			Content:   nil, // Will be filled by caller
			Pos:       ast.Position{Line: startPos.Line, Column: startPos.Column},
			AtToken:   atToken,
			NameToken: nameToken,
		}, nil
	default:
		return nil, fmt.Errorf("decorator @%s cannot be used in shell context (line %d:%d) - only value, action, and block decorators are allowed", decoratorName, startPos.Line, startPos.Column)
	}
}

// parseDecorator parses a single decorator and returns the appropriate AST node type
func (p *Parser) parseDecorator() (ast.CommandContent, error) {
	startPos := p.current()
	atToken, _ := p.consume(types.AT, "expected '@'")

	// Get decorator name
	var nameToken types.Token
	var err error

	if p.current().Type == types.IDENTIFIER {
		nameToken, err = p.consume(types.IDENTIFIER, "expected decorator name")
	} else {
		// Handle special cases where keywords appear as decorator names
		nameToken = p.current()
		if !p.isValidDecoratorName(nameToken) {
			return nil, p.NewSyntaxError(fmt.Sprintf("expected decorator name after '@', got %s", nameToken.Type.String()))
		}
		p.advance()
	}

	if err != nil {
		return nil, err
	}

	decoratorName := nameToken.Value
	if nameToken.Type != types.IDENTIFIER {
		decoratorName = strings.ToLower(nameToken.Value)
	}

	// Step 1: Check if decorator exists in registry
	decorator, decoratorType, err := decorators.GetAny(decoratorName)
	if err != nil {
		return nil, p.NewInvalidError("unknown decorator @" + decoratorName)
	}

	// Step 2: Get parameter schema from decorator
	paramSchema := decorator.ParameterSchema()

	// Step 3: Parse parameters according to schema
	var params []ast.NamedParameter
	if p.match(types.LPAREN) {
		p.advance() // consume '('
		params, err = p.parseParameterList(paramSchema)
		if err != nil {
			return nil, err
		}
		_, err = p.consume(types.RPAREN, "expected ')' after decorator arguments")
		if err != nil {
			return nil, err
		}
	}

	// Step 4: Validate parameters using decorator schema
	if err := p.validateDecoratorParameters(decorator, params, decoratorName); err != nil {
		return nil, err
	}

	// Step 5: Create appropriate AST node based on decorator type
	switch decoratorType {
	case decorators.ValueType:
		return nil, fmt.Errorf("value decorator @%s cannot be used as standalone command (line %d:%d) - value decorators can only be used inline within shell commands", decoratorName, startPos.Line, startPos.Column)
	case decorators.ActionType:
		return &ast.ActionDecorator{
			Name:      decoratorName,
			Args:      params,
			Pos:       ast.Position{Line: startPos.Line, Column: startPos.Column},
			AtToken:   atToken,
			NameToken: nameToken,
		}, nil
	case decorators.BlockType:
		return &ast.BlockDecorator{
			Name:      decoratorName,
			Args:      params,
			Content:   nil, // Will be filled in by caller
			Pos:       ast.Position{Line: startPos.Line, Column: startPos.Column},
			AtToken:   atToken,
			NameToken: nameToken,
		}, nil
	case decorators.PatternType:
		return &ast.PatternDecorator{
			Name:      decoratorName,
			Args:      params,
			Patterns:  nil, // Will be filled in by caller
			Pos:       ast.Position{Line: startPos.Line, Column: startPos.Column},
			AtToken:   atToken,
			NameToken: nameToken,
		}, nil
	default:
		return nil, p.NewInvalidError("unknown decorator type for @" + decoratorName)
	}
}

// parseParameterList parses a comma-separated list of named parameters using the decorator's schema
func (p *Parser) parseParameterList(paramSchema []decorators.ParameterSchema) ([]ast.NamedParameter, error) {
	var params []ast.NamedParameter
	if p.match(types.RPAREN) {
		return params, nil // No parameters
	}

	positionalIndex := 0

	for {
		param, err := p.parseParameter(paramSchema, &positionalIndex)
		if err != nil {
			return nil, err
		}
		params = append(params, param)

		if !p.match(types.COMMA) {
			break
		}
		p.advance() // consume ','
	}
	return params, nil
}

// parseParameter parses a single parameter (either named or positional) using the schema
func (p *Parser) parseParameter(paramSchema []decorators.ParameterSchema, positionalIndex *int) (ast.NamedParameter, error) {
	startPos := p.current()

	// Check if this is a named parameter (identifier = value)
	if p.current().Type == types.IDENTIFIER && p.peek().Type == types.EQUALS {
		// Named parameter
		nameToken, err := p.consume(types.IDENTIFIER, "expected parameter name")
		if err != nil {
			return ast.NamedParameter{}, err
		}
		equalsToken, err := p.consume(types.EQUALS, "expected '=' after parameter name")
		if err != nil {
			return ast.NamedParameter{}, err
		}

		// Find the parameter schema for this named parameter
		var foundSchema *decorators.ParameterSchema
		for i := range paramSchema {
			if paramSchema[i].Name == nameToken.Value {
				foundSchema = &paramSchema[i]
				break
			}
		}

		value, err := p.parseParameterValue(foundSchema, nameToken.Value)
		if err != nil {
			return ast.NamedParameter{}, err
		}

		return ast.NamedParameter{
			Name:        nameToken.Value,
			Value:       value,
			Pos:         ast.Position{Line: startPos.Line, Column: startPos.Column},
			NameToken:   &nameToken,
			EqualsToken: &equalsToken,
		}, nil
	} else {
		// Positional parameter
		var foundSchema *decorators.ParameterSchema
		var paramName string
		if *positionalIndex < len(paramSchema) {
			foundSchema = &paramSchema[*positionalIndex]
			paramName = paramSchema[*positionalIndex].Name
		} else {
			paramName = fmt.Sprintf("arg%d", *positionalIndex)
		}

		value, err := p.parseParameterValue(foundSchema, paramName)
		if err != nil {
			return ast.NamedParameter{}, err
		}
		*positionalIndex++

		return ast.NamedParameter{
			Name:  paramName,
			Value: value,
			Pos:   ast.Position{Line: startPos.Line, Column: startPos.Column},
			// NameToken and EqualsToken are nil for positional parameters
		}, nil
	}
}

// parseStringLiteral parses string literals with support for interpolation
func (p *Parser) parseStringLiteral() (*ast.StringLiteral, error) {
	if p.current().Type == types.STRING_START {
		// Double quotes/backticks: "text@var(X)more" → multiple parts (with interpolation)
		startTok := p.current()
		p.advance()

		var parts []ast.StringPart
		var allTokens []types.Token
		allTokens = append(allTokens, startTok)

		// Parse sequence: STRING_TEXT, decorators, STRING_TEXT, ..., STRING_END
		for p.current().Type != types.STRING_END && !p.isAtEnd() {
			if p.current().Type == types.STRING_TEXT {
				tok := p.current()
				parts = append(parts, &ast.TextStringPart{Text: tok.Value})
				allTokens = append(allTokens, tok)
				p.advance()
			} else if p.current().Type == types.AT {
				// Parse @var() or @env() within string
				decorator, err := p.parseValueDecorator()
				if err != nil {
					return nil, err
				}
				parts = append(parts, decorator)
				// Add decorator tokens to allTokens
				allTokens = append(allTokens, decorator.TokenRange().All...)
			} else {
				return nil, p.NewSyntaxError(fmt.Sprintf("unexpected token %s in string literal", p.current().Type.String()))
			}
		}

		endTok, err := p.consume(types.STRING_END, "expected closing quote")
		if err != nil {
			return nil, err
		}
		allTokens = append(allTokens, endTok)

		// Build raw string from start to end tokens
		raw := p.input[startTok.Span.Start.Offset:endTok.Span.End.Offset]

		return &ast.StringLiteral{
			Parts:       parts,
			Raw:         raw,
			StringToken: startTok,
			Tokens:      ast.TokenRange{All: allTokens, Start: startTok, End: endTok},
		}, nil
	}

	return nil, p.NewSyntaxError(fmt.Sprintf("expected string literal, got %s", p.current().Type.String()))
}

// parseValueDecorator parses a value decorator in string context (only @var, @env allowed)
func (p *Parser) parseValueDecorator() (*ast.ValueDecorator, error) {
	startPos := p.current()
	atToken, _ := p.consume(types.AT, "expected '@'")

	// Get decorator name
	var nameToken types.Token
	var err error

	if p.current().Type == types.IDENTIFIER {
		nameToken, err = p.consume(types.IDENTIFIER, "expected decorator name")
	} else {
		return nil, p.NewSyntaxError("expected decorator name after '@'")
	}

	if err != nil {
		return nil, err
	}

	decoratorName := nameToken.Value
	if nameToken.Type != types.IDENTIFIER {
		decoratorName = strings.ToLower(nameToken.Value)
	}

	// Check if decorator exists in registry and is a value decorator
	decorator, decoratorType, err := decorators.GetAny(decoratorName)
	if err != nil {
		return nil, p.NewInvalidError("unknown decorator @" + decoratorName)
	}

	// Only value decorators are allowed in string literals
	if decoratorType != decorators.ValueType {
		return nil, p.NewInvalidError(fmt.Sprintf("decorator @%s cannot be used in string literals - only value decorators like @var and @env are allowed", decoratorName))
	}

	// Get parameter schema from decorator
	paramSchema := decorator.ParameterSchema()

	// Parse parameters according to schema
	var params []ast.NamedParameter
	var openParen, closeParen *types.Token

	if p.match(types.LPAREN) {
		openParenTok := p.current()
		openParen = &openParenTok
		p.advance() // consume '('
		params, err = p.parseParameterList(paramSchema)
		if err != nil {
			return nil, err
		}
		closeParenTok, err := p.consume(types.RPAREN, "expected ')' after decorator arguments")
		if err != nil {
			return nil, err
		}
		closeParen = &closeParenTok
	}

	return &ast.ValueDecorator{
		Name:       decoratorName,
		Args:       params,
		Pos:        ast.Position{Line: startPos.Line, Column: startPos.Column},
		AtToken:    atToken,
		NameToken:  nameToken,
		OpenParen:  openParen,
		CloseParen: closeParen,
	}, nil
}

// parseValue parses a literal value (string, number, duration, boolean, identifier)
func (p *Parser) parseValue() (ast.Expression, error) {
	switch p.current().Type {
	case types.STRING_START:
		return p.parseStringLiteral()
	case types.NUMBER:
		tok := p.current()
		p.advance()
		return &ast.NumberLiteral{Value: tok.Value, Token: tok}, nil
	case types.DURATION:
		tok := p.current()
		p.advance()
		return &ast.DurationLiteral{Value: tok.Value, Token: tok}, nil
	case types.BOOLEAN:
		tok := p.current()
		p.advance()
		boolValue := tok.Value == "true"
		return &ast.BooleanLiteral{Value: boolValue, Raw: tok.Value, Token: tok}, nil
	case types.IDENTIFIER:
		tok := p.current()
		p.advance()
		return &ast.Identifier{Name: tok.Value, Token: tok}, nil
	default:
		return nil, p.NewSyntaxError(fmt.Sprintf("unexpected token %s, expected a value", p.current().Type.String()))
	}
}

// parseParameterValue parses a parameter value with type checking and enhanced error messages
func (p *Parser) parseParameterValue(schema *decorators.ParameterSchema, paramName string) (ast.Expression, error) {
	// If we have schema information, validate the type
	if schema != nil {
		return p.parseValueWithTypeCheck(p.convertArgTypeToExpressionType(schema.Type), paramName)
	}

	// Fallback to general value parsing if no schema
	return p.parseValue()
}

// parseValueWithTypeCheck parses a value and validates it against the expected type
func (p *Parser) parseValueWithTypeCheck(expectedType types.ExpressionType, paramName string) (ast.Expression, error) {
	currentToken := p.current()

	switch currentToken.Type {
	case types.STRING_START:
		if expectedType != types.StringType {
			return nil, p.NewTypeError(paramName, expectedType, p.current())
		}
		return p.parseStringLiteral()

	case types.NUMBER:
		if expectedType != types.NumberType {
			return nil, p.NewTypeError(paramName, expectedType, p.current())
		}
		tok := p.current()
		p.advance()
		return &ast.NumberLiteral{Value: tok.Value, Token: tok}, nil

	case types.DURATION:
		if expectedType != types.DurationType {
			return nil, p.NewTypeError(paramName, expectedType, p.current())
		}
		tok := p.current()
		p.advance()
		return &ast.DurationLiteral{Value: tok.Value, Token: tok}, nil

	case types.BOOLEAN:
		if expectedType != types.BooleanType {
			return nil, p.NewTypeError(paramName, expectedType, p.current())
		}
		tok := p.current()
		p.advance()
		boolValue := tok.Value == "true"
		return &ast.BooleanLiteral{Value: boolValue, Raw: tok.Value, Token: tok}, nil

	case types.IDENTIFIER:
		// Identifiers are valid for any type - they reference variables
		tok := p.current()
		p.advance()
		return &ast.Identifier{Name: tok.Value, Token: tok}, nil

	default:
		return nil, p.NewTypeError(paramName, expectedType, p.current())
	}
}

// Legacy error functions - these are now implemented in errors.go
// Kept for any remaining compatibility needs

// isValidDecoratorName checks if a token can be used as a decorator name
func (p *Parser) isValidDecoratorName(token types.Token) bool {
	switch token.Type {
	case types.IDENTIFIER:
		return true
	case types.VAR:
		// "var" can be used as a decorator name for @var()
		return true
	default:
		return false
	}
}

// parsePatternBranchesInBlock parses pattern branches directly from the token stream
func (p *Parser) parsePatternBranchesInBlock() ([]ast.PatternBranch, error) {
	var patterns []ast.PatternBranch

	for !p.match(types.RBRACE) && !p.isAtEnd() {
		p.skipWhitespaceAndComments()
		if p.match(types.RBRACE) {
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

func (p *Parser) advance() types.Token {
	if !p.isAtEnd() {
		p.pos++
	}
	return p.previous()
}

func (p *Parser) current() types.Token  { return p.tokens[p.pos] }
func (p *Parser) previous() types.Token { return p.tokens[p.pos-1] }
func (p *Parser) peek() types.Token     { return p.tokens[p.pos+1] }

func (p *Parser) isAtEnd() bool { return p.current().Type == types.EOF }

func (p *Parser) match(types ...types.TokenType) bool {
	for _, t := range types {
		if p.current().Type == t {
			return true
		}
	}
	return false
}

// formatError creates a detailed error message with source context
func (p *Parser) formatError(message string, token types.Token) error {
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

func (p *Parser) consume(t types.TokenType, message string) (types.Token, error) {
	if p.match(t) {
		tok := p.current()
		p.advance()
		return tok, nil
	}
	return types.Token{}, p.formatError(message, p.current())
}

func (p *Parser) skipWhitespaceAndComments() {
	// NEWLINE tokens no longer exist - they're handled as whitespace by lexer
	for p.match(types.COMMENT, types.MULTILINE_COMMENT) {
		p.advance()
	}
}

// isPatternDecorator checks if the current position starts a pattern decorator.
func (p *Parser) isPatternDecorator() bool {
	if p.current().Type != types.AT {
		return false
	}
	if p.pos+1 < len(p.tokens) {
		nextToken := p.tokens[p.pos+1]

		if nextToken.Type == types.IDENTIFIER {
			// Use the decorator registry to check for pattern decorators
			return decorators.IsPatternDecorator(nextToken.Value)
		}
	}
	return false
}

// isBlockDecorator checks if the current position starts a block decorator.
func (p *Parser) isBlockDecorator() bool {
	if p.current().Type != types.AT {
		p.logger.Debug("        ❌ isBlockDecorator: not AT", "token", p.current().Type.String())
		return false
	}
	if p.pos+1 < len(p.tokens) {
		nextToken := p.tokens[p.pos+1]
		var name string

		if nextToken.Type == types.IDENTIFIER {
			name = nextToken.Value
		} else {
			p.logger.Debug("        ❌ isBlockDecorator: next not IDENTIFIER", "next", nextToken.Type.String())
			return false
		}

		// Use the decorator registry to check for block decorators
		result := decorators.IsBlockDecorator(name)
		if result {
			p.logger.Debug("        ✅ isBlockDecorator: found", "decorator", name)
		} else {
			p.logger.Debug("        ❌ isBlockDecorator: not block", "name", name)
		}
		return result
	}
	p.logger.Debug("        ❌ isBlockDecorator: no next token")
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
		// Synchronization points for error recovery:

		// 1. Top-level keywords
		switch p.current().Type {
		case types.VAR, types.WATCH, types.STOP:
			return

		// 2. Commands: IDENTIFIER followed by COLON (command declaration)
		case types.IDENTIFIER:
			if !p.isAtEnd() && p.pos < len(p.tokens)-1 && p.tokens[p.pos+1].Type == types.COLON {
				return // Found "command-name:"
			}

		// 3. Block and Pattern decorators on potential new lines
		case types.AT:
			if !p.isAtEnd() && p.pos < len(p.tokens)-1 && p.tokens[p.pos+1].Type == types.IDENTIFIER {
				decoratorName := p.tokens[p.pos+1].Value
				// Check if this is a registered block or pattern decorator
				if _, decoratorType, err := decorators.GetAny(decoratorName); err == nil {
					switch decoratorType {
					case decorators.BlockType, decorators.PatternType:
						return // Found @parallel, @timeout, @when, etc.
					}
				}
			}

		// 4. Block boundaries - closing braces are good recovery points
		case types.RBRACE:
			p.advance() // consume the }
			return
		}

		p.advance()
	}
}

// validateDecoratorParameters validates parameters against the decorator's schema
func (p *Parser) validateDecoratorParameters(decorator decorators.DecoratorBase, params []ast.NamedParameter, decoratorName string) error {
	schema := decorator.ParameterSchema()

	// Check required parameters
	requiredParams := make(map[string]bool)
	for _, param := range schema {
		if param.Required {
			requiredParams[param.Name] = true
		}
	}

	// Check provided parameters
	providedParams := make(map[string]bool)
	for i, param := range params {
		var paramName string
		if param.Name != "" {
			// Named parameter
			paramName = param.Name
		} else {
			// Positional parameter - map to schema
			if i >= len(schema) {
				return fmt.Errorf("too many parameters for @%s decorator (expected %d, got %d)", decoratorName, len(schema), len(params))
			}
			paramName = schema[i].Name
		}

		// Check if parameter exists in schema
		found := false
		for _, schemaParam := range schema {
			if schemaParam.Name == paramName {
				found = true
				// Validate parameter type
				if err := p.validateParameterType(param.Value, p.convertArgTypeToExpressionType(schemaParam.Type), paramName, decoratorName); err != nil {
					return err
				}
				break
			}
		}
		if !found {
			return fmt.Errorf("unknown parameter '%s' for @%s decorator", paramName, decoratorName)
		}

		providedParams[paramName] = true
		delete(requiredParams, paramName)
	}

	// Check for missing required parameters
	for paramName := range requiredParams {
		return fmt.Errorf("missing required parameter '%s' for @%s decorator", paramName, decoratorName)
	}

	return nil
}

// validateParameterType checks if a parameter value matches the expected type
func (p *Parser) validateParameterType(value ast.Expression, expectedType ast.ExpressionType, paramName, decoratorName string) error {
	// Get the actual type of the provided value
	actualType := value.GetType()

	// Check if types match
	if actualType != expectedType {
		// Handle identifier references to variables
		if actualType == ast.IdentifierType {
			if ident, ok := value.(*ast.Identifier); ok {
				// Look up the variable to check its type
				varType, found := p.getVariableType(ident.Name)
				if !found {
					return fmt.Errorf("parameter '%s' for @%s decorator references undefined variable '%s'",
						paramName, decoratorName, ident.Name)
				}

				// Check if the variable's type matches the expected type
				if varType != expectedType {
					return fmt.Errorf("parameter '%s' for @%s decorator expects %s, but variable '%s' is %s",
						paramName, decoratorName, expectedType.String(), ident.Name, varType.String())
				}

				// Variable type matches - identifier is valid
				return nil
			}
		}

		return fmt.Errorf("parameter '%s' for @%s decorator expects %s, got %s",
			paramName, decoratorName, expectedType.String(), actualType.String())
	}

	return nil
}

// convertArgTypeToExpressionType converts decorators.ArgType to ast.ExpressionType for parser compatibility
func (p *Parser) convertArgTypeToExpressionType(argType decorators.ArgType) ast.ExpressionType {
	switch argType {
	case decorators.ArgTypeString:
		return ast.StringType
	case decorators.ArgTypeBool:
		return ast.BooleanType
	case decorators.ArgTypeInt, decorators.ArgTypeFloat:
		return ast.NumberType
	case decorators.ArgTypeDuration:
		return ast.DurationType
	case decorators.ArgTypeIdentifier:
		return ast.IdentifierType
	case decorators.ArgTypeList, decorators.ArgTypeMap, decorators.ArgTypeAny:
		// For complex types, use string as fallback for now
		return ast.StringType
	default:
		return ast.StringType
	}
}

// getVariableType looks up a variable's type from the program's variable declarations
func (p *Parser) getVariableType(varName string) (ast.ExpressionType, bool) {
	// Look in the current program being parsed
	if p.program != nil {
		// Check regular variables
		for _, variable := range p.program.Variables {
			if variable.Name == varName {
				return variable.Value.GetType(), true
			}
		}

		// Check variable groups
		for _, group := range p.program.VarGroups {
			for _, variable := range group.Variables {
				if variable.Name == varName {
					return variable.Value.GetType(), true
				}
			}
		}
	}

	return ast.StringType, false // Return any type since it wasn't found
}

// validatePatternBranches validates pattern branches against the decorator's pattern schema
func (p *Parser) validatePatternBranches(decorator decorators.PatternDecorator, patterns []ast.PatternBranch, decoratorName string) error {
	schema := decorator.PatternSchema()

	// Track which patterns are provided
	providedPatterns := make(map[string]bool)
	for _, patternBranch := range patterns {
		var patternName string

		// Handle different pattern types
		switch p := patternBranch.Pattern.(type) {
		case *ast.IdentifierPattern:
			patternName = p.Name
		case *ast.WildcardPattern:
			patternName = "default"
		default:
			return fmt.Errorf("unknown pattern type for @%s decorator", decoratorName)
		}

		// Check for wildcard
		if patternName == "default" {
			if !schema.AllowsWildcard {
				return fmt.Errorf("@%s decorator does not allow 'default' wildcard pattern", decoratorName)
			}
		} else {
			// Check if this specific pattern is allowed
			if !schema.AllowsAnyIdentifier && len(schema.AllowedPatterns) > 0 {
				allowed := false
				for _, allowedPattern := range schema.AllowedPatterns {
					if patternName == allowedPattern {
						allowed = true
						break
					}
				}
				if !allowed {
					return fmt.Errorf("unknown pattern '%s' for @%s decorator", patternName, decoratorName)
				}
			}
		}

		providedPatterns[patternName] = true
	}

	// Check for missing required patterns
	for _, requiredPattern := range schema.RequiredPatterns {
		if !providedPatterns[requiredPattern] {
			return fmt.Errorf("missing required pattern '%s' for @%s decorator", requiredPattern, decoratorName)
		}
	}

	return nil
}
