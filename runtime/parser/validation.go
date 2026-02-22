package parser

import (
	"fmt"

	"github.com/opal-lang/opal/core/decorator"
	"github.com/opal-lang/opal/runtime/lexer"
)

// ExecutionMode represents the context in which Opal code is executed
type ExecutionMode int

const (
	ModeScript  ExecutionMode = iota // Script mode: full language (vars, functions, shell commands, execution)
	ModeCommand                      // Command mode: definitions only (like just/make - no top-level execution)
)

// ValidationError represents a mode-specific validation error
type ValidationError struct {
	Mode    ExecutionMode
	Message string
	Line    int
	Context string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// Validate checks if the parse tree is valid for the given execution mode
func (t *ParseTree) Validate(mode ExecutionMode) error {
	// Universal validations (apply to all modes)
	if err := validateEnvInRemoteTransport(t); err != nil {
		return err
	}

	// Mode-specific validations
	switch mode {
	case ModeScript:
		return validateScriptMode(t)
	case ModeCommand:
		return validateCommandMode(t)
	default:
		return &ValidationError{
			Mode:    mode,
			Message: "unknown execution mode",
		}
	}
}

// validateScriptMode validates script mode (everything is allowed)
func validateScriptMode(t *ParseTree) error {
	// Script mode allows:
	// - Variable declarations
	// - Function definitions
	// - Shell commands
	// - Control flow (when implemented)
	// - Decorators (when implemented)

	// No restrictions in script mode
	return nil
}

// validateCommandMode validates command mode (command definitions only)
func validateCommandMode(t *ParseTree) error {
	// Command mode allows:
	// - Function definitions (commands)
	// - Variable declarations (for parameterization)
	//
	// Command mode rejects:
	// - Top-level shell commands (no execution, only definitions)
	// - Decorator calls (no execution, only definitions)

	inSource := false
	depth := 0

	for _, evt := range t.Events {
		if evt.Kind == EventOpen {
			nodeKind := NodeKind(evt.Data)

			if nodeKind == NodeSource {
				inSource = true
				depth++
				continue
			}

			// Track depth to identify top-level nodes
			if inSource {
				depth++

				// Top-level is depth 2 (Source -> Statement)
				// NodeVarDecl and NodeFunction are allowed.
				if depth == 2 {
					switch nodeKind {
					case NodeShellCommand:
						return &ValidationError{
							Mode:    ModeCommand,
							Message: "command mode does not allow top-level shell commands",
							Context: "command mode is for definitions only (like just/make)",
						}
					case NodeFunctionCall:
						return &ValidationError{
							Mode:    ModeCommand,
							Message: "command mode does not allow top-level function calls",
							Context: "command mode is for definitions only (like just/make)",
						}
					}
				}
			}
		} else if evt.Kind == EventClose {
			if inSource {
				depth--
				if depth == 0 {
					inSource = false
				}
			}
		}
	}

	return nil
}

// validateEnvInRemoteTransport validates that root-only value decorators (like @env)
// are not used inside transport-switching decorators.
//
// This prevents confusing behavior where @env.HOME inside @ssh.connect would resolve
// to the local environment instead of the remote environment.
//
// Uses the decorator registry: decorators with RoleBoundary role are transport-switching,
// and value decorators with TransportScopeLocal capability are root-only.
func validateEnvInRemoteTransport(t *ParseTree) error {
	// Track transport depth (increments when entering transport-switching decorators)
	transportDepth := 0

	// Track decorator stack for error messages
	var decoratorStack []string

	for i, evt := range t.Events {
		switch evt.Kind {
		case EventToken:
			// Check if this is a value decorator token (@)
			if int(evt.Data) < len(t.Tokens) {
				tok := t.Tokens[evt.Data]
				if tok.Type == lexer.AT {
					// Look ahead to get decorator name
					decoratorName := extractDecoratorNameFromTokens(t, i)
					if decoratorName != "" {
						// Check if this value decorator is restricted to local/RootOnly scope
						// and we're inside a transport boundary
						if transportDepth > 0 && isRootOnlyValueDecorator(decoratorName) {
							// Find which transport we're inside
							transportName := "remote transport"
							if len(decoratorStack) > 0 {
								transportName = "@" + decoratorStack[len(decoratorStack)-1]
							}

							return &ValidationError{
								Mode:    ModeScript,
								Message: "@" + decoratorName + " is root-only and cannot be used inside " + transportName,
								Line:    tok.Position.Line,
								Context: fmt.Sprintf(
									"use shell variables ($HOME, $USER, etc.) to access remote environment, "+
										"or hoist to root: var X = @%s at top-level, then use @var.X inside %s",
									decoratorName, transportName,
								),
							}
						}
					}
				}
			}

		case EventOpen:
			nodeKind := NodeKind(evt.Data)
			if nodeKind == NodeDecorator {
				// Extract decorator name
				decoratorName := extractDecoratorNameFromEvents(t, i)
				if decoratorName != "" {
					decoratorStack = append(decoratorStack, decoratorName)

					// Check if this decorator switches transport (has RoleBoundary role)
					if isTransportSwitchingDecorator(decoratorName) {
						transportDepth++
					}
				}
			}

		case EventClose:
			nodeKind := NodeKind(evt.Data)
			if nodeKind == NodeDecorator && len(decoratorStack) > 0 {
				// Check if we're exiting a transport-switching decorator
				decoratorName := decoratorStack[len(decoratorStack)-1]
				if isTransportSwitchingDecorator(decoratorName) {
					transportDepth--
				}

				// Pop decorator from stack
				decoratorStack = decoratorStack[:len(decoratorStack)-1]
			}
		}
	}

	return nil
}

// isTransportSwitchingDecorator checks if a decorator switches transport context.
// Returns true if the decorator has the RoleBoundary role (implements Transport interface).
func isTransportSwitchingDecorator(path string) bool {
	entry, ok := decorator.Global().Lookup(path)
	if !ok {
		return false
	}
	for _, role := range entry.Roles {
		if role == decorator.RoleBoundary {
			return true
		}
	}
	return false
}

// isRootOnlyValueDecorator checks if a value decorator is restricted to local/RootOnly scope.
// Returns true if the decorator is a value provider with TransportScopeLocal capability.
// Tries progressively shorter paths to handle property accessors like @test.env.KEY
func isRootOnlyValueDecorator(path string) bool {
	// Try progressively shorter paths to find the decorator
	// For @test.env.KEY, try test.env.KEY, then test.env, then test
	for {
		entry, ok := decorator.Global().Lookup(path)
		if ok {
			// Check if it's a value provider
			hasValueRole := false
			for _, role := range entry.Roles {
				if role == decorator.RoleProvider {
					hasValueRole = true
					break
				}
			}
			if !hasValueRole {
				return false
			}
			// Check if it's restricted to local scope
			return entry.Impl.Descriptor().Capabilities.TransportScope == decorator.TransportScopeLocal
		}

		// Try shorter path by removing last component
		lastDot := -1
		for i := len(path) - 1; i >= 0; i-- {
			if path[i] == '.' {
				lastDot = i
				break
			}
		}
		if lastDot == -1 {
			break // No more components to remove
		}
		path = path[:lastDot]
	}

	return false
}

// extractDecoratorNameFromTokens extracts decorator name starting from @ token
func extractDecoratorNameFromTokens(t *ParseTree, atTokenIdx int) string {
	var parts []string

	// Start from the token after @
	for i := atTokenIdx + 1; i < len(t.Events) && i < atTokenIdx+10; i++ {
		evt := t.Events[i]
		if evt.Kind != EventToken {
			break
		}

		tokIdx := evt.Data
		if int(tokIdx) >= len(t.Tokens) {
			break
		}

		tok := t.Tokens[tokIdx]

		switch tok.Type {
		case lexer.IDENTIFIER:
			parts = append(parts, string(tok.Text))
		case lexer.DOT:
			// Continue building dotted name
			continue
		default:
			// End of decorator name
			if len(parts) > 0 {
				return joinParts(parts)
			}
			return ""
		}
	}

	if len(parts) > 0 {
		return joinParts(parts)
	}
	return ""
}

// extractDecoratorNameFromEvents extracts the decorator name from parse events
// starting at the given EventOpen position for a NodeDecorator
func extractDecoratorNameFromEvents(t *ParseTree, startIdx int) string {
	// Look for tokens after the @ symbol to build decorator name
	var parts []string

	for i := startIdx + 1; i < len(t.Events) && i < startIdx+20; i++ {
		evt := t.Events[i]
		if evt.Kind != EventToken {
			continue
		}

		tokIdx := evt.Data
		if int(tokIdx) >= len(t.Tokens) {
			continue
		}

		tok := t.Tokens[tokIdx]

		switch tok.Type {
		case lexer.AT:
			// Start of decorator
			continue
		case lexer.IDENTIFIER:
			parts = append(parts, string(tok.Text))
		case lexer.DOT:
			// Continue building dotted name
			continue
		case lexer.LPAREN, lexer.LBRACE:
			// End of decorator name
			if len(parts) > 0 {
				return joinParts(parts)
			}
			return ""
		default:
			// Unknown token, stop
			if len(parts) > 0 {
				return joinParts(parts)
			}
			return ""
		}
	}

	if len(parts) > 0 {
		return joinParts(parts)
	}
	return ""
}

// joinParts joins decorator name parts with dots
func joinParts(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += "." + parts[i]
	}
	return result
}
