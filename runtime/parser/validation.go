package parser

import (
	"github.com/aledsdavies/opal/core/types"
	"github.com/aledsdavies/opal/runtime/lexer"
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
				if depth == 2 {
					switch nodeKind {
					case NodeShellCommand:
						return &ValidationError{
							Mode:    ModeCommand,
							Message: "command mode does not allow top-level shell commands",
							Context: "command mode is for definitions only (like just/make)",
						}
						// NodeVarDecl and NodeFunction are allowed
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
// Uses the TransportScope interface: decorators declare their scope (RootOnly, Agnostic, RemoteAware)
// and validation checks scope vs transportDepth.
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
						// Check transport scope
						scope := types.Global().GetTransportScope(decoratorName)

						// If decorator is root-only and we're inside a transport, error
						if scope == types.ScopeRootOnly && transportDepth > 0 {
							// Find which transport we're inside
							transportName := "remote transport"
							if len(decoratorStack) > 0 {
								transportName = "@" + decoratorStack[len(decoratorStack)-1]
							}

							return &ValidationError{
								Mode:    ModeScript,
								Message: "@" + decoratorName + " is root-only and cannot be used inside " + transportName,
								Line:    tok.Position.Line,
								Context: "use shell variables ($HOME, $USER, etc.) to access transport environment, or hoist: var X = @" + decoratorName + " at root then use @var.X",
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

					// Check if this decorator switches transport
					// For now, we check the schema's SwitchesTransport flag
					// TODO: Add TransportSwitcher interface for execution decorators
					schema, exists := types.Global().GetSchema(decoratorName)
					if exists && schema.SwitchesTransport {
						transportDepth++
					}
				}
			}

		case EventClose:
			nodeKind := NodeKind(evt.Data)
			if nodeKind == NodeDecorator && len(decoratorStack) > 0 {
				// Check if we're exiting a transport-switching decorator
				decoratorName := decoratorStack[len(decoratorStack)-1]
				schema, exists := types.Global().GetSchema(decoratorName)
				if exists && schema.SwitchesTransport {
					transportDepth--
				}

				// Pop decorator from stack
				decoratorStack = decoratorStack[:len(decoratorStack)-1]
			}
		}
	}

	return nil
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
