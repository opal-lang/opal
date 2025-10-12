package parser

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
