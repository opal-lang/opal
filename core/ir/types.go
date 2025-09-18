package ir

import (
	"github.com/aledsdavies/devcmd/core/decorators"
)

// ================================================================================================
// INTERMEDIATE REPRESENTATION (IR) - Structural Types Only
// ================================================================================================

// ChainOp represents operators that connect chain elements
type ChainOp string

const (
	ChainOpNone   ChainOp = ""   // No operator (first element or end of chain)
	ChainOpAnd    ChainOp = "&&" // Execute next only if previous succeeded
	ChainOpOr     ChainOp = "||" // Execute next only if previous failed
	ChainOpPipe   ChainOp = "|"  // Pipe stdout of previous to stdin of next
	ChainOpAppend ChainOp = ">>" // Append stdout of previous to file
)

// ElementKind represents the type of a chain element
type ElementKind string

const (
	ElementKindAction  ElementKind = "action"  // Decorator that executes as an action
	ElementKindShell   ElementKind = "shell"   // Raw shell command
	ElementKindBlock   ElementKind = "block"   // Block decorator (@workdir, @timeout, @parallel)
	ElementKindPattern ElementKind = "pattern" // Pattern decorator (@when, @try)
)

// ChainElement represents one element in a command chain
type ChainElement struct {
	Kind ElementKind `json:"kind"` // Type of element

	// For action elements (decorators)
	Name string             `json:"name,omitempty"` // decorator name
	Args []decorators.Param `json:"args,omitempty"` // parameters using core types

	// For shell elements - structured content preserving value decorators
	Content *ElementContent `json:"content,omitempty"`

	// For block decorators (until proper Wrapper transformation)
	InnerSteps []CommandStep `json:"inner_steps,omitempty"`

	// Chain continuation
	OpNext ChainOp `json:"op_next"`          // operator to next element
	Target string  `json:"target,omitempty"` // target file for >> operator

	// Source location for error reporting
	Span *SourceSpan `json:"span,omitempty"`
}

// ElementContent represents structured content for shell elements
type ElementContent struct {
	Parts []ContentPart `json:"parts"` // Structured parts preserving value decorators
}

// ContentPart represents a piece of shell content (literal or value decorator)
type ContentPart struct {
	Kind PartKind `json:"kind"`

	// For literal parts
	Text string `json:"text,omitempty"`

	// For value decorator parts (@var, @env, etc.)
	DecoratorName string             `json:"decorator_name,omitempty"`
	DecoratorArgs []decorators.Param `json:"decorator_args,omitempty"`

	// Source location
	Span *SourceSpan `json:"span,omitempty"`
}

// PartKind represents the type of content part
type PartKind string

const (
	PartKindLiteral   PartKind = "literal"   // Plain text
	PartKindDecorator PartKind = "decorator" // @var(), @env(), etc.
)

// SourceSpan represents a source location for error reporting
type SourceSpan struct {
	File   string `json:"file"`             // source file path
	Line   int    `json:"line"`             // line number (1-based)
	Column int    `json:"column"`           // column number (1-based)
	Length int    `json:"length,omitempty"` // length of span for highlighting
}

// CommandStep represents one command line (separated by newlines)
type CommandStep struct {
	Chain []ChainElement `json:"chain"`          // elements connected by operators
	Span  *SourceSpan    `json:"span,omitempty"` // source location of this step
}

// CommandSeq represents a sequence of command steps (newline-separated)
type CommandSeq struct {
	Steps []CommandStep `json:"steps"`
}

// Wrapper represents a block decorator that wraps inner execution
type Wrapper struct {
	Kind   string         `json:"kind"`   // Decorator name (e.g., "timeout", "workdir")
	Params map[string]any `json:"params"` // decorator parameters
	Inner  CommandSeq     `json:"inner"`  // wrapped command sequence
}

// Pattern represents a pattern decorator with conditional branches
type Pattern struct {
	Kind     string                `json:"kind"`     // Decorator name (e.g., "when", "try")
	Params   map[string]any        `json:"params"`   // pattern parameters
	Branches map[string]CommandSeq `json:"branches"` // conditional branches
}

// Node represents any IR node type
type Node interface {
	NodeType() string
}

// Implement Node interface
func (c CommandSeq) NodeType() string { return "CommandSeq" }
func (w Wrapper) NodeType() string    { return "Wrapper" }
func (p Pattern) NodeType() string    { return "Pattern" }

// Command represents a complete command definition with metadata
type Command struct {
	Name string `json:"name"` // command name
	Root Node   `json:"root"` // root IR node (CommandSeq, Wrapper, or Pattern)
}

func (c Command) NodeType() string { return "Command" }

// ================================================================================================
// ENVIRONMENT SNAPSHOT - Structural representation only
// ================================================================================================

// EnvSnapshot represents a frozen environment snapshot for deterministic execution
type EnvSnapshot struct {
	Values      map[string]string `json:"values"`      // Immutable environment variables
	Fingerprint string            `json:"fingerprint"` // SHA256 of sorted KEY\x00VAL pairs
}

// Get retrieves a value from the environment snapshot
func (e *EnvSnapshot) Get(key string) (string, bool) {
	value, exists := e.Values[key]
	return value, exists
}

// GetAll returns all environment values for shell execution
func (e *EnvSnapshot) GetAll() map[string]string {
	return e.Values
}
