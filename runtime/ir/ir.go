package ir

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/aledsdavies/devcmd/core/decorators"
)

// ================================================================================================
// INTERMEDIATE REPRESENTATION (IR) - Unified between Interpreter and Generated modes
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

// ElementKind represents the type of a chain element (generic IR types)
type ElementKind string

const (
	ElementKindAction  ElementKind = "action"  // Decorator that executes as an action
	ElementKindShell   ElementKind = "shell"   // Raw shell command
	ElementKindBlock   ElementKind = "block"   // Block decorator (@workdir, @timeout, @parallel)
	ElementKindPattern ElementKind = "pattern" // Pattern decorator (@when, @try)
)

// DecoratorType represents the behavioral category of decorators (generic IR types)
type DecoratorType string

const (
	DecoratorTypeBlock   DecoratorType = "block"   // Wraps execution (timeout, workdir, etc.)
	DecoratorTypeAction  DecoratorType = "action"  // Executable command (log, cmd, etc.)
	DecoratorTypePattern DecoratorType = "pattern" // Conditional execution (when, try)
	DecoratorTypeValue   DecoratorType = "value"   // Inline value substitution (var, env)
)

// ChainElement represents one element in a command chain
type ChainElement struct {
	Kind ElementKind `json:"kind"` // Type of element (action, shell)

	// For action elements (decorators)
	Name string                      `json:"name,omitempty"` // decorator name (e.g., "cmd", "log", "my-custom")
	Args []decorators.DecoratorParam `json:"args,omitempty"` // named/positional parameters

	// For shell elements - structured content preserving value decorators
	Content *ElementContent `json:"content,omitempty"` // structured content with value decorators

	// For block decorators (temporary field until proper Wrapper transformation)
	InnerSteps []CommandStep `json:"inner_steps,omitempty"` // inner content for block decorators

	// Chain continuation
	OpNext ChainOp `json:"op_next"`          // operator to next element
	Target string  `json:"target,omitempty"` // target file for >> operator

	// Source location for error reporting (optional but recommended)
	Span *SourceSpan `json:"span,omitempty"` // file:line:col for precise error messages
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
	DecoratorName string                      `json:"decorator_name,omitempty"`
	DecoratorArgs []decorators.DecoratorParam `json:"decorator_args,omitempty"`

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
	Kind   string                 `json:"kind"`   // Decorator name (e.g., "timeout", "workdir", "my-custom")
	Params map[string]interface{} `json:"params"` // decorator parameters
	Inner  CommandSeq             `json:"inner"`  // wrapped command sequence
}

// Pattern represents a pattern decorator with conditional branches
type Pattern struct {
	Kind     string                 `json:"kind"`     // Decorator name (e.g., "when", "try", "my-switch")
	Params   map[string]interface{} `json:"params"`   // pattern parameters
	Branches map[string]CommandSeq  `json:"branches"` // conditional branches (branch name -> command sequence)
}

// Node represents any IR node type
type Node interface {
	NodeType() string
}

// Implement Node interface
func (c CommandSeq) NodeType() string { return "CommandSeq" }
func (w Wrapper) NodeType() string    { return "Wrapper" }
func (p Pattern) NodeType() string    { return "Pattern" }

// ================================================================================================
// ENVIRONMENT AND CONTEXT
// ================================================================================================

// EnvSnapshot represents a frozen environment snapshot for deterministic execution
type EnvSnapshot struct {
	Values      map[string]string `json:"values"`      // Immutable environment variables
	Fingerprint string            `json:"fingerprint"` // SHA256 of sorted KEY\x00VAL pairs
}

// EnvOptions configures environment snapshot creation
type EnvOptions struct {
	Manifest  []string // Keys referenced in IR (or nil=all)
	BlockList []string // Drop PWD, OLDPWD, SHLVL, RANDOM, PS*, TERM
	LockPath  string   // Optional path to persist/reuse env
}

// NewEnvSnapshot creates a frozen environment snapshot
func NewEnvSnapshot(opts EnvOptions) (*EnvSnapshot, error) {
	// Implementation would capture and freeze environment
	// For now, simplified version
	env := make(map[string]string)
	for _, e := range os.Environ() {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			env[parts[0]] = parts[1]
		}
	}
	return &EnvSnapshot{
		Values:      env,
		Fingerprint: "placeholder", // Would compute actual hash
	}, nil
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

// ================================================================================================
// EXECUTION CONTEXT
// ================================================================================================

// UIConfig contains UI behavior configuration from standardized flags
type UIConfig struct {
	ColorMode   string `json:"color_mode"`   // auto, always, never
	Quiet       bool   `json:"quiet"`        // minimal output (errors only)
	Verbose     bool   `json:"verbose"`      // extra debugging output
	Interactive string `json:"interactive"`  // auto, always, never
	AutoConfirm bool   `json:"auto_confirm"` // auto-confirm all prompts (--yes)
	CI          bool   `json:"ci"`           // CI mode (optimized for CI environments)
}

// Ctx carries state needed for command execution (matches architecture doc)
type Ctx struct {
	Env     *EnvSnapshot      `json:"env"`      // Frozen environment snapshot
	Vars    map[string]string `json:"vars"`     // CLI variables (@var) resolved to strings
	WorkDir string            `json:"work_dir"` // Current working directory

	// IO streams
	Stdout io.Writer `json:"-"` // Standard output
	Stderr io.Writer `json:"-"` // Standard error
	Stdin  io.Reader `json:"-"` // Standard input

	// Execution flags
	DryRun bool `json:"dry_run"` // Plan mode - don't actually execute
	Debug  bool `json:"debug"`   // Debug mode - verbose output

	// System configuration
	NumCPU int `json:"num_cpu"` // Number of CPU cores for parallel execution

	// UI configuration from standardized flags
	UIConfig *UIConfig `json:"ui_config,omitempty"` // UI behavior configuration

	// Command registry for @cmd decorator support
	Commands map[string]Node `json:"-"` // Available commands by name
}

// Clone creates a copy of the context for isolated execution
func (ctx *Ctx) Clone() *Ctx {
	newVars := make(map[string]string, len(ctx.Vars))
	for k, v := range ctx.Vars {
		newVars[k] = v
	}

	// Deep copy UIConfig if it exists
	var uiConfig *UIConfig
	if ctx.UIConfig != nil {
		uiConfig = &UIConfig{
			ColorMode:   ctx.UIConfig.ColorMode,
			Quiet:       ctx.UIConfig.Quiet,
			Verbose:     ctx.UIConfig.Verbose,
			Interactive: ctx.UIConfig.Interactive,
			AutoConfirm: ctx.UIConfig.AutoConfirm,
			CI:          ctx.UIConfig.CI,
		}
	}

	return &Ctx{
		Env:      ctx.Env, // EnvSnapshot is immutable, safe to share
		Vars:     newVars,
		WorkDir:  ctx.WorkDir,
		Stdout:   ctx.Stdout,
		Stderr:   ctx.Stderr,
		Stdin:    ctx.Stdin,
		DryRun:   ctx.DryRun,
		Debug:    ctx.Debug,
		NumCPU:   ctx.NumCPU,
		UIConfig: uiConfig,
		Commands: ctx.Commands, // Commands are immutable, safe to share
	}
}

// WithWorkDir returns a new context with updated working directory
func (ctx *Ctx) WithWorkDir(workDir string) *Ctx {
	newCtx := ctx.Clone()
	newCtx.WorkDir = workDir
	return newCtx
}

// CtxOptions configures execution context creation
type CtxOptions struct {
	EnvOptions EnvOptions        // Environment snapshot options
	Vars       map[string]string // CLI variables
	WorkDir    string            // Working directory (empty = current dir)
	Stdout     io.Writer         // Standard output (nil = os.Stdout)
	Stderr     io.Writer         // Standard error (nil = os.Stderr)
	Stdin      io.Reader         // Standard input (nil = os.Stdin)
	DryRun     bool              // Plan mode - don't actually execute
	Debug      bool              // Debug mode - verbose output
	NumCPU     int               // Number of CPU cores (0 = runtime.NumCPU())
	UIConfig   *UIConfig         // UI behavior configuration (optional)
	Commands   map[string]Node   // Available commands for @cmd decorator
}

// NewCtx creates a new execution context with proper defaults
func NewCtx(opts CtxOptions) (*Ctx, error) {
	// Create environment snapshot
	envSnapshot, err := NewEnvSnapshot(opts.EnvOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create environment snapshot: %w", err)
	}

	// Set defaults for unspecified values
	vars := opts.Vars
	if vars == nil {
		vars = make(map[string]string)
	}

	workDir := opts.WorkDir
	if workDir == "" {
		if wd, err := os.Getwd(); err == nil {
			workDir = wd
		} else {
			workDir = "." // Fallback
		}
	}

	stdout := opts.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}

	stderr := opts.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}

	stdin := opts.Stdin
	if stdin == nil {
		stdin = os.Stdin
	}

	numCPU := opts.NumCPU
	if numCPU <= 0 {
		numCPU = runtime.NumCPU()
	}

	return &Ctx{
		Env:      envSnapshot,
		Vars:     vars,
		WorkDir:  workDir,
		Stdout:   stdout,
		Stderr:   stderr,
		Stdin:    stdin,
		DryRun:   opts.DryRun,
		Debug:    opts.Debug,
		NumCPU:   numCPU,
		UIConfig: opts.UIConfig,
	}, nil
}

// Note: Keep ExecutionContext as alias for compatibility during migration
type ExecutionContext = Ctx

// ================================================================================================
// COMMAND RESULT - Shared between both execution modes
// ================================================================================================

// CommandResult represents the result of executing a command or action
type CommandResult struct {
	Stdout   string `json:"stdout"`    // Standard output as string
	Stderr   string `json:"stderr"`    // Standard error as string
	ExitCode int    `json:"exit_code"` // Exit code (0 = success)
}

// Success returns true if the command executed successfully (exit code 0)
func (r CommandResult) Success() bool {
	return r.ExitCode == 0
}

// Failed returns true if the command failed (non-zero exit code)
func (r CommandResult) Failed() bool {
	return r.ExitCode != 0
}

// Error returns an error representation when the command failed
func (r CommandResult) Error() error {
	if r.Success() {
		return nil
	}
	if r.Stderr != "" {
		return fmt.Errorf("exit code %d: %s", r.ExitCode, r.Stderr)
	}
	return fmt.Errorf("exit code %d", r.ExitCode)
}

// ================================================================================================
// UNIFIED DECORATOR CONTRACTS
// ================================================================================================

// ValueDecorator provides inline values for shell text substitution
// Signature: func(ctx) (string, error)
type ValueDecorator interface {
	Name() string
	ExpandValue(ctx *Ctx, params []decorators.DecoratorParam) (string, error)
}

// ActionDecorator represents runnable commands that return CommandResult
// Signature: func(ctx) CommandResult
type ActionDecorator interface {
	Name() string
	ExecuteAction(ctx *Ctx, params []decorators.DecoratorParam) CommandResult
}

// BlockDecorator wraps inner execution with behavior modifications
// Signature: func(ctx, run func(Context) CommandResult) CommandResult
type BlockDecorator interface {
	Name() string
	WrapExecution(ctx *Ctx, params []decorators.DecoratorParam, inner func(*Ctx) CommandResult) CommandResult
}

// PatternDecorator handles conditional execution based on pattern matching
// Signature: func(ctx, selectBranch func(Context) CommandSeq) CommandResult
type PatternDecorator interface {
	Name() string
	ExecutePattern(ctx *Ctx, params []decorators.DecoratorParam, branches map[string]CommandSeq) CommandResult
}

// ================================================================================================
// EXECUTION PRIMITIVES - Shared helpers used in both modes
// ================================================================================================

// ExecShell executes a shell command and returns the result
func ExecShell(ctx *Ctx, cmd string) CommandResult {
	if ctx.DryRun {
		return CommandResult{
			Stdout:   fmt.Sprintf("[DRY-RUN] %s", cmd),
			ExitCode: 0,
		}
	}

	// Create command with shell
	execCmd := exec.Command("sh", "-c", cmd)

	// Set working directory
	if ctx.WorkDir != "" {
		execCmd.Dir = ctx.WorkDir
	}

	// Apply frozen environment
	if ctx.Env != nil {
		envMap := ctx.Env.GetAll()
		execCmd.Env = make([]string, 0, len(envMap))
		for k, v := range envMap {
			execCmd.Env = append(execCmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	// Stream output to user AND capture for result
	var stdout, stderr bytes.Buffer
	if ctx.Stdout != nil {
		execCmd.Stdout = io.MultiWriter(&stdout, ctx.Stdout)
	} else {
		execCmd.Stdout = &stdout
	}
	if ctx.Stderr != nil {
		execCmd.Stderr = io.MultiWriter(&stderr, ctx.Stderr)
	} else {
		execCmd.Stderr = &stderr
	}

	// Connect stdin if available
	if ctx.Stdin != nil {
		execCmd.Stdin = ctx.Stdin
	}

	// Execute command
	err := execCmd.Run()

	// Determine exit code
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			// Command failed to start or other error
			exitCode = 127
		}
	}

	// Log debug output if enabled
	if ctx.Debug {
		fmt.Fprintf(ctx.Stderr, "[DEBUG] Command: %s\n", cmd)
		fmt.Fprintf(ctx.Stderr, "[DEBUG] Exit Code: %d\n", exitCode)
		if stdout.Len() > 0 {
			fmt.Fprintf(ctx.Stderr, "[DEBUG] Stdout: %s\n", stdout.String())
		}
		if stderr.Len() > 0 {
			fmt.Fprintf(ctx.Stderr, "[DEBUG] Stderr: %s\n", stderr.String())
		}
	}

	return CommandResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
	}
}

// ExecShellWithInput executes a shell command with stdin input
func ExecShellWithInput(ctx *Ctx, cmd, input string) CommandResult {
	if ctx.DryRun {
		return CommandResult{
			Stdout:   fmt.Sprintf("[DRY-RUN] %s (piped input)", cmd),
			ExitCode: 0,
		}
	}

	// Create command with shell
	execCmd := exec.Command("sh", "-c", cmd)

	// Set working directory
	if ctx.WorkDir != "" {
		execCmd.Dir = ctx.WorkDir
	}

	// Apply frozen environment
	if ctx.Env != nil {
		envMap := ctx.Env.GetAll()
		execCmd.Env = make([]string, 0, len(envMap))
		for k, v := range envMap {
			execCmd.Env = append(execCmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	// Set up stdin pipe
	execCmd.Stdin = strings.NewReader(input)

	// Stream output to user AND capture for result
	var stdout, stderr bytes.Buffer
	if ctx.Stdout != nil {
		execCmd.Stdout = io.MultiWriter(&stdout, ctx.Stdout)
	} else {
		execCmd.Stdout = &stdout
	}
	if ctx.Stderr != nil {
		execCmd.Stderr = io.MultiWriter(&stderr, ctx.Stderr)
	} else {
		execCmd.Stderr = &stderr
	}

	// Execute command
	err := execCmd.Run()

	// Determine exit code
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			// Command failed to start or other error
			exitCode = 127
		}
	}

	// Log debug output if enabled
	if ctx.Debug {
		fmt.Fprintf(ctx.Stderr, "[DEBUG] Command (with input): %s\n", cmd)
		fmt.Fprintf(ctx.Stderr, "[DEBUG] Input Length: %d bytes\n", len(input))
		fmt.Fprintf(ctx.Stderr, "[DEBUG] Exit Code: %d\n", exitCode)
	}

	return CommandResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
	}
}

// AppendToFile appends content to a file (used by >> operator)
func AppendToFile(filename, content string) error {
	// Open file in append mode, create if it doesn't exist
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open file %s for append: %w", filename, err)
	}
	defer file.Close()

	// Write content
	_, err = file.WriteString(content)
	if err != nil {
		return fmt.Errorf("failed to append to file %s: %w", filename, err)
	}

	return nil
}

// ================================================================================================
// STRUCTURED CONTENT RESOLUTION
// ================================================================================================

// Resolve converts ElementContent to executable string by resolving value decorators
func (ec *ElementContent) Resolve(ctx *decorators.Ctx, registry *decorators.Registry) (string, error) {
	if ec == nil {
		return "", nil
	}

	var result strings.Builder

	for _, part := range ec.Parts {
		switch part.Kind {
		case PartKindLiteral:
			result.WriteString(part.Text)

		case PartKindDecorator:
			decorator := registry.Values[part.DecoratorName]
			if decorator == nil {
				return "", fmt.Errorf("unknown value decorator: %s", part.DecoratorName)
			}

			value, err := decorator.Render(ctx, part.DecoratorArgs)
			if err != nil {
				return "", fmt.Errorf("expanding @%s: %w", part.DecoratorName, err)
			}

			result.WriteString(value)

		default:
			return "", fmt.Errorf("unknown content part kind: %s", part.Kind)
		}
	}

	return result.String(), nil
}

// GetResolvedText returns the resolved text for this ChainElement
func (ce *ChainElement) GetResolvedText(ctx *decorators.Ctx, registry *decorators.Registry) (string, error) {
	if ce.Content == nil {
		return "", fmt.Errorf("shell element missing structured content")
	}
	return ce.Content.Resolve(ctx, registry)
}

// PlanDescription returns a description suitable for plan output
func (ce *ChainElement) PlanDescription(ctx *decorators.Ctx, registry *decorators.Registry) string {
	if ce.Content == nil {
		// This should never happen in clean greenfield code
		return fmt.Sprintf("<ERROR: shell element missing structured content>")
	}

	var parts []string
	for _, part := range ce.Content.Parts {
		switch part.Kind {
		case PartKindLiteral:
			parts = append(parts, part.Text)
		case PartKindDecorator:
			if ctx != nil && registry != nil {
				// Show resolved value in plan
				decorator := registry.Values[part.DecoratorName]
				if decorator != nil {
					if value, err := decorator.Render(ctx, part.DecoratorArgs); err == nil {
						parts = append(parts, fmt.Sprintf("@%s(%s) → %q", part.DecoratorName, argsToString(part.DecoratorArgs), value))
						continue
					}
				}
			}
			// Fallback: show raw decorator
			parts = append(parts, fmt.Sprintf("@%s(%s)", part.DecoratorName, argsToString(part.DecoratorArgs)))
		}
	}

	return strings.Join(parts, "")
}

// Helper function to convert decorator args to string for display
func argsToString(args []decorators.DecoratorParam) string {
	if len(args) == 0 {
		return ""
	}

	var parts []string
	for _, arg := range args {
		if arg.Name != "" {
			parts = append(parts, fmt.Sprintf("%s=%v", arg.Name, arg.Value))
		} else {
			parts = append(parts, fmt.Sprintf("%v", arg.Value))
		}
	}
	return strings.Join(parts, ", ")
}

// NewShellElement creates a ChainElement with structured content from simple text
func NewShellElement(text string) ChainElement {
	return ChainElement{
		Kind: ElementKindShell,
		Content: &ElementContent{
			Parts: []ContentPart{
				{Kind: PartKindLiteral, Text: text},
			},
		},
	}
}
