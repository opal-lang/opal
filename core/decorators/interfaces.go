package decorators

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/aledsdavies/devcmd/core/plan"
)

// ================================================================================================
// EXECUTION TYPES (duplicated from runtime/execution/ir.go to avoid circular imports)
// ================================================================================================

// ChainOp represents shell operators connecting elements
type ChainOp string

const (
	ChainOpNone   ChainOp = ""   // No operator (first/last element)
	ChainOpAnd    ChainOp = "&&" // Execute next only if previous succeeded
	ChainOpOr     ChainOp = "||" // Execute next only if previous failed
	ChainOpPipe   ChainOp = "|"  // Pipe stdout to next stdin
	ChainOpAppend ChainOp = ">>" // Append stdout to file
)

// ElementKind represents the type of chain element
type ElementKind string

const (
	ElementKindAction ElementKind = "action" // Decorator that executes as an action
	ElementKindShell  ElementKind = "shell"  // Raw shell command
	ElementKindBlock  ElementKind = "block"  // Block decorator inside pattern branches
)

// ChainElement represents one element in a command chain
type ChainElement struct {
	Kind ElementKind `json:"kind"` // Type of element (action, shell)

	// For action elements (decorators)
	Name string           `json:"name,omitempty"` // decorator name
	Args []DecoratorParam `json:"args,omitempty"` // named/positional parameters

	// For shell elements
	Text string `json:"text,omitempty"` // raw shell command text

	// For block decorators (temporary field until proper Wrapper transformation)
	InnerSteps []CommandStep `json:"inner_steps,omitempty"` // inner content for block decorators

	// Chain continuation
	OpNext ChainOp `json:"op_next"`          // operator to next element
	Target string  `json:"target,omitempty"` // target file for >> operator
}

// CommandStep represents one command line (separated by newlines)
type CommandStep struct {
	Chain []ChainElement `json:"chain"` // elements connected by operators
}

// CommandSeq represents a sequence of command steps (newline-separated)
type CommandSeq struct {
	Steps []CommandStep `json:"steps"`
}

// ParallelMode represents different parallel execution modes
type ParallelMode string

const (
	ParallelModeFailFast      ParallelMode = "fail-fast"      // Stop scheduling on first failure (default)
	ParallelModeFailImmediate ParallelMode = "fail-immediate" // Cancel all on first failure
	ParallelModeAll           ParallelMode = "all"            // Run all to completion
)

// ================================================================================================
// FUNDAMENTAL DECORATOR INTERFACES
// ================================================================================================

// DecoratorBase provides common metadata for all decorators
type DecoratorBase interface {
	Name() string
	Description() string
	ParameterSchema() []ParameterSchema
	Examples() []Example
	ImportRequirements() ImportRequirement
}

// ArgType represents parameter types independent of AST
type ArgType string

const (
	ArgTypeString     ArgType = "string"
	ArgTypeBool       ArgType = "bool"
	ArgTypeInt        ArgType = "int"
	ArgTypeFloat      ArgType = "float"
	ArgTypeDuration   ArgType = "duration"   // Duration strings like "30s", "5m", "1h"
	ArgTypeIdentifier ArgType = "identifier" // Variable/command identifiers
	ArgTypeList       ArgType = "list"
	ArgTypeMap        ArgType = "map"
	ArgTypeAny        ArgType = "any"
)

// ParameterSchema describes a decorator parameter
type ParameterSchema struct {
	Name        string      // Parameter name
	Type        ArgType     // Parameter type (AST-independent)
	Required    bool        // Whether required
	Description string      // Human-readable description
	Default     interface{} // Default value if not provided
}

// Example provides usage examples
type Example struct {
	Code        string // Example code
	Description string // What it demonstrates
}

// ImportRequirement describes dependencies for code generation
type ImportRequirement struct {
	StandardLibrary []string          // Standard library imports
	ThirdParty      []string          // Third-party imports
	GoModules       map[string]string // Module dependencies (module -> version)
}

// PatternSchema describes pattern matching rules
type PatternSchema struct {
	AllowedPatterns     []string // Specific patterns allowed
	RequiredPatterns    []string // Patterns that must be present
	AllowsWildcard      bool     // Whether "*" wildcard is allowed
	AllowsAnyIdentifier bool     // Whether any identifier is allowed
	Description         string   // Human-readable description
}

// ================================================================================================
// EXECUTION CONTEXT (minimal for core interfaces)
// ================================================================================================

// EnvSnapshot represents a frozen environment snapshot for deterministic execution
type EnvSnapshot interface {
	Get(key string) (string, bool)
	GetAll() map[string]string // Access to all values for shell execution
}

// UIConfig contains standardized UI behavior flags
type UIConfig struct {
	ColorMode   string `json:"color_mode"`   // "auto", "always", "never"
	Quiet       bool   `json:"quiet"`        // minimal output (errors only)
	Verbose     bool   `json:"verbose"`      // extra debugging output
	Interactive string `json:"interactive"`  // "auto", "always", "never"
	AutoConfirm bool   `json:"auto_confirm"` // auto-confirm all prompts (--yes)
	CI          bool   `json:"ci"`           // CI mode (optimized for CI environments)
}

// ExecutionDelegate provides action execution capability for decorator contexts
type ExecutionDelegate interface {
	ExecuteAction(ctx *Ctx, name string, args []DecoratorParam) CommandResult
	ExecuteBlock(ctx *Ctx, name string, args []DecoratorParam, innerSteps []CommandStep) CommandResult
	ExecuteCommand(ctx *Ctx, commandName string) CommandResult
}

// Ctx carries minimal state needed for decorator execution
type Ctx struct {
	Env     EnvSnapshot       `json:"env"`      // Frozen environment snapshot
	Vars    map[string]string `json:"vars"`     // CLI variables (@var) resolved to strings
	WorkDir string            `json:"work_dir"` // Current working directory

	// IO streams
	Stdout io.Writer `json:"-"` // Standard output
	Stderr io.Writer `json:"-"` // Standard error
	Stdin  io.Reader `json:"-"` // Standard input

	// System information
	NumCPU int `json:"num_cpu"` // Number of CPU cores available

	// Execution flags
	DryRun bool `json:"dry_run"` // Plan mode - don't actually execute
	Debug  bool `json:"debug"`   // Debug mode - verbose output

	// UI configuration from standardized flags
	UI *UIConfig `json:"ui,omitempty"` // UI behavior configuration

	// Execution delegate for action decorators
	Executor ExecutionDelegate `json:"-"` // Delegate for executing actions within decorators
}

// Clone creates a copy of the context for isolated execution
func (ctx *Ctx) Clone() *Ctx {
	newVars := make(map[string]string, len(ctx.Vars))
	for k, v := range ctx.Vars {
		newVars[k] = v
	}

	// Deep copy UIConfig if it exists
	var ui *UIConfig
	if ctx.UI != nil {
		ui = &UIConfig{
			ColorMode:   ctx.UI.ColorMode,
			Quiet:       ctx.UI.Quiet,
			Verbose:     ctx.UI.Verbose,
			Interactive: ctx.UI.Interactive,
			AutoConfirm: ctx.UI.AutoConfirm,
			CI:          ctx.UI.CI,
		}
	}

	return &Ctx{
		Env:      ctx.Env, // EnvSnapshot is immutable, safe to share
		Vars:     newVars,
		WorkDir:  ctx.WorkDir,
		NumCPU:   ctx.NumCPU,
		Stdout:   ctx.Stdout,
		Stderr:   ctx.Stderr,
		Stdin:    ctx.Stdin,
		DryRun:   ctx.DryRun,
		Debug:    ctx.Debug,
		UI:       ui,
		Executor: ctx.Executor, // Share the execution delegate
	}
}

// WithWorkDir returns a new context with updated working directory
// This is the correct pattern - never use os.Chdir()
func (ctx *Ctx) WithWorkDir(workDir string) *Ctx {
	newCtx := ctx.Clone()
	newCtx.WorkDir = workDir
	return newCtx
}

// ================================================================================================
// CONTEXT HELPER METHODS - Simple execution primitives for decorators
// ================================================================================================

// Note: These are basic helpers. Full implementation will delegate to runtime/execution package.

// ExecSequential executes command steps sequentially (like shell scripts with set -e)
// Decorators like @timeout, @workdir use this for wrapped inner execution
func (ctx *Ctx) ExecSequential(steps []CommandStep) CommandResult {
	var lastResult CommandResult
	for _, step := range steps {
		lastResult = ctx.ExecStep(step)
		if lastResult.Failed() {
			return lastResult // Stop on first failure
		}
	}
	return lastResult
}

// ExecParallel executes command steps in parallel with configurable failure mode
// Decorators like @parallel use this for concurrent execution
func (ctx *Ctx) ExecParallel(steps []CommandStep, mode ParallelMode) CommandResult {
	if len(steps) == 0 {
		return CommandResult{ExitCode: 0}
	}

	// For single step, just execute sequentially
	if len(steps) == 1 {
		return ctx.ExecStep(steps[0])
	}

	// Execute steps in parallel
	type stepResult struct {
		index  int
		result CommandResult
	}

	resultChan := make(chan stepResult, len(steps))

	// Start all steps in parallel
	for i, step := range steps {
		go func(index int, step CommandStep) {
			result := ctx.ExecStep(step)
			resultChan <- stepResult{index: index, result: result}
		}(i, step)
	}

	// Collect results
	results := make([]CommandResult, len(steps))
	var combinedStdout, combinedStderr string
	var finalExitCode int

	for i := 0; i < len(steps); i++ {
		stepRes := <-resultChan
		results[stepRes.index] = stepRes.result

		// Combine output in completion order
		if stepRes.result.Stdout != "" {
			combinedStdout += stepRes.result.Stdout
		}
		if stepRes.result.Stderr != "" {
			combinedStderr += stepRes.result.Stderr
		}

		// Handle failure modes
		if stepRes.result.ExitCode != 0 && finalExitCode == 0 {
			finalExitCode = stepRes.result.ExitCode

			// For fail-fast modes, we still wait for all running tasks
			// (immediate cancellation would require context.Context support)
		}
	}

	return CommandResult{
		Stdout:   combinedStdout,
		Stderr:   combinedStderr,
		ExitCode: finalExitCode,
	}
}

// ExecStep executes a single command step (chain of elements with operators)
// This is the basic building block that decorators can use
func (ctx *Ctx) ExecStep(step CommandStep) CommandResult {
	if len(step.Chain) == 0 {
		return CommandResult{ExitCode: 0}
	}

	// Execute chain with proper operator semantics
	var lastResult CommandResult
	var combinedOutput strings.Builder

	for i, element := range step.Chain {
		// Check if we should execute this element based on previous result and operator
		if i > 0 {
			prevOp := step.Chain[i-1].OpNext
			shouldSkip := ctx.shouldSkipElement(prevOp, lastResult)
			if shouldSkip {
				continue
			}
		}

		// Execute the element
		var result CommandResult
		switch element.Kind {
		case ElementKindShell:
			// Handle pipe operator - pass previous stdout as stdin
			if i > 0 && step.Chain[i-1].OpNext == ChainOpPipe {
				result = ctx.execShellWithInput(element.Text, strings.NewReader(lastResult.Stdout))
			} else {
				result = ctx.ExecShell(element.Text)
			}
		case ElementKindAction:
			// Delegate action execution to the runtime evaluator
			if ctx.Executor != nil {
				result = ctx.Executor.ExecuteAction(ctx, element.Name, element.Args)
			} else {
				result = CommandResult{
					Stderr:   "ExecutionDelegate not available for action execution",
					ExitCode: 1,
				}
			}
		case ElementKindBlock:
			// Delegate block execution to the runtime evaluator
			if ctx.Executor != nil {
				// Convert CommandStep to decorators.CommandStep
				var decoratorSteps []CommandStep
				for _, step := range element.InnerSteps {
					decoratorStep := CommandStep{
						Chain: make([]ChainElement, len(step.Chain)),
					}
					for i, elem := range step.Chain {
						decoratorStep.Chain[i] = ChainElement{
							Kind:       elem.Kind,
							Name:       elem.Name,
							Text:       elem.Text,
							Args:       elem.Args,
							InnerSteps: elem.InnerSteps, // Preserve inner steps
							OpNext:     elem.OpNext,
							Target:     elem.Target,
						}
					}
					decoratorSteps = append(decoratorSteps, decoratorStep)
				}
				result = ctx.Executor.ExecuteBlock(ctx, element.Name, element.Args, decoratorSteps)
			} else {
				result = CommandResult{
					Stderr:   "ExecutionDelegate not available for block execution",
					ExitCode: 1,
				}
			}
		default:
			result = CommandResult{
				Stderr:   "Unknown element kind: " + string(element.Kind),
				ExitCode: 1,
			}
		}

		// Handle append operator
		if element.OpNext == ChainOpAppend {
			if err := ctx.appendToFile(element.Target, result.Stdout); err != nil {
				return CommandResult{
					Stderr:   err.Error(),
					ExitCode: 1,
				}
			}
			// For append, return success unless append failed
			result = CommandResult{ExitCode: 0}
		}

		// Accumulate output based on operation type
		if i == 0 {
			// Always include first element's output
			if result.Stdout != "" {
				combinedOutput.WriteString(result.Stdout)
			}
		} else {
			prevOp := step.Chain[i-1].OpNext
			switch prevOp {
			case ChainOpPipe:
				// For pipes, only keep the final output (stdin->stdout flow)
				// Don't accumulate - result.Stdout will be the final output
			case ChainOpAnd, ChainOpOr, ChainOpNone:
				// For &&, ||, and no operator, accumulate all outputs
				if result.Stdout != "" {
					combinedOutput.WriteString(result.Stdout)
				}
			}
		}

		lastResult = result

		// Stop chain execution on failure for && or on success for ||
		if ctx.shouldTerminateChain(element.OpNext, result) {
			break
		}
	}

	// Return the result with combined output
	finalOutput := combinedOutput.String()

	// For operations ending with pipe, use the last result's stdout (the pipe output)
	hasPipeAtEnd := len(step.Chain) > 1 && step.Chain[len(step.Chain)-2].OpNext == ChainOpPipe
	if hasPipeAtEnd {
		finalOutput = lastResult.Stdout
	}

	return CommandResult{
		Stdout:   finalOutput,
		Stderr:   lastResult.Stderr,
		ExitCode: lastResult.ExitCode,
	}
}

// ExecShell executes a shell command using the same logic as runtime/execution
func (ctx *Ctx) ExecShell(cmd string) CommandResult {
	if ctx.DryRun {
		return CommandResult{
			Stdout:   "[DRY-RUN] " + cmd,
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
	} else {
		// Use current environment if no frozen env
		execCmd.Env = os.Environ()
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
		_, _ = fmt.Fprintf(ctx.Stderr, "[DEBUG] Command: %s\n", cmd)
		_, _ = fmt.Fprintf(ctx.Stderr, "[DEBUG] Exit Code: %d\n", exitCode)
		if stdout.Len() > 0 {
			_, _ = fmt.Fprintf(ctx.Stderr, "[DEBUG] Stdout: %s\n", stdout.String())
		}
		if stderr.Len() > 0 {
			_, _ = fmt.Fprintf(ctx.Stderr, "[DEBUG] Stderr: %s\n", stderr.String())
		}
	}

	return CommandResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
	}
}

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

// ================================================================================================
// EXPANSION SYSTEM - Hint-based decorator expansion
// ================================================================================================

// Decorators use standard ExecutionStep.Metadata to provide expansion hints.
// The plan generator reads these hints and builds the expanded structure.
//
// Standard expansion metadata keys:
//   - "expansion_type": Type of expansion needed ("command_reference", "template_include", etc.)
//   - "command_name": For command_reference expansion
//   - "template_path": For template_include expansion
//   - "module_name": For module_import expansion
//   - "include_path": For file_include expansion
//
// Example @cmd decorator usage:
//   Metadata: map[string]string{
//       "expansion_type": "command_reference",
//       "command_name":   cmdName,
//   }
//
// The generator will:
// 1. See expansion_type in metadata
// 2. Use appropriate resolver to fetch the referenced resource
// 3. Generate ExecutionSteps from the resource
// 4. Set those as the decorator step's Children

// ================================================================================================
// CORE DECORATOR INTERFACES
// ================================================================================================

// ValueDecorator - Inline value substitution
type ValueDecorator interface {
	DecoratorBase
	// Runtime execution
	Render(ctx *Ctx, args []DecoratorParam) (string, error)
	// Plan generation
	Describe(ctx *Ctx, args []DecoratorParam) plan.ExecutionStep
}

// ActionDecorator - Executable commands that return CommandResult
type ActionDecorator interface {
	DecoratorBase
	// Runtime execution
	Run(ctx *Ctx, args []DecoratorParam) CommandResult
	// Plan generation
	Describe(ctx *Ctx, args []DecoratorParam) plan.ExecutionStep
}

// StdinAware actions can receive piped input
type StdinAware interface {
	RunWithInput(ctx *Ctx, args []DecoratorParam, input io.Reader) CommandResult
}

// BlockDecorator - Execution wrappers that modify inner command behavior
// Wraps a single inner sequence; no branching (e.g. @parallel, @timeout, @retry, @workdir)
type BlockDecorator interface {
	DecoratorBase
	// Runtime execution - wraps inner command sequence
	WrapCommands(ctx *Ctx, args []DecoratorParam, inner CommandSeq) CommandResult
	// Plan generation
	Describe(ctx *Ctx, args []DecoratorParam, inner plan.ExecutionStep) plan.ExecutionStep
}

// PatternDecorator - Conditional execution with branch selection
// Chooses between named branches (e.g. @when, @try)
type PatternDecorator interface {
	DecoratorBase
	PatternSchema() PatternSchema
	// Runtime execution - selects and executes one branch
	SelectBranch(ctx *Ctx, args []DecoratorParam, branches map[string]CommandSeq) CommandResult
	// Plan generation
	Describe(ctx *Ctx, args []DecoratorParam, branches map[string]plan.ExecutionStep) plan.ExecutionStep
}

// ================================================================================================
// OPTIONAL ENHANCED INTERFACES - For decorators needing finer control
// ================================================================================================

// StepParallel provides access to individual steps for decorators like @parallel
// Decorators implementing BlockDecorator can optionally also implement this for step-level control
type StepParallel interface {
	// Execute steps in parallel with individual control
	WrapStepsParallel(ctx *Ctx, args []DecoratorParam, steps []CommandStep) CommandResult
}

// ================================================================================================
// SHELL OPERATOR HELPER METHODS - Following docs/engine-architecture.md semantics
// ================================================================================================

// shouldSkipElement determines if an element should be skipped based on the operator and previous result
func (ctx *Ctx) shouldSkipElement(op ChainOp, prevResult CommandResult) bool {
	switch op {
	case ChainOpAnd:
		// cmd1 && cmd2 - cmd2 runs only if cmd1 succeeds
		return prevResult.ExitCode != 0
	case ChainOpOr:
		// cmd1 || cmd2 - cmd2 runs only if cmd1 fails
		return prevResult.ExitCode == 0
	case ChainOpPipe, ChainOpAppend, ChainOpNone:
		// Always execute for pipes, append, and no operator
		return false
	default:
		return false
	}
}

// shouldTerminateChain determines if chain execution should stop based on operator and result
func (ctx *Ctx) shouldTerminateChain(op ChainOp, result CommandResult) bool {
	switch op {
	case ChainOpAnd:
		// Stop on failure for &&
		return result.ExitCode != 0
	case ChainOpOr:
		// Stop on success for ||
		return result.ExitCode == 0
	case ChainOpPipe, ChainOpAppend, ChainOpNone:
		// Continue for pipes, append, and no operator
		return false
	default:
		return false
	}
}

// execShellWithInput executes a shell command with provided stdin (for pipe operator)
func (ctx *Ctx) execShellWithInput(cmd string, stdin io.Reader) CommandResult {
	if ctx.DryRun {
		return CommandResult{
			Stdout:   "[DRY-RUN] " + cmd,
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
	} else {
		// Use current environment if no frozen env
		execCmd.Env = os.Environ()
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
	execCmd.Stdin = stdin

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
		_, _ = fmt.Fprintf(ctx.Stderr, "[DEBUG] Command: %s\n", cmd)
		_, _ = fmt.Fprintf(ctx.Stderr, "[DEBUG] Exit Code: %d\n", exitCode)
		if stdout.Len() > 0 {
			_, _ = fmt.Fprintf(ctx.Stderr, "[DEBUG] Stdout: %s\n", stdout.String())
		}
		if stderr.Len() > 0 {
			_, _ = fmt.Fprintf(ctx.Stderr, "[DEBUG] Stderr: %s\n", stderr.String())
		}
	}

	return CommandResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
	}
}

// appendToFile appends content to a file (for >> operator)
func (ctx *Ctx) appendToFile(filename, content string) error {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open file %s for append: %w", filename, err)
	}
	defer func() { _ = file.Close() }()

	_, err = file.WriteString(content)
	if err != nil {
		return fmt.Errorf("failed to write to file %s: %w", filename, err)
	}

	return nil
}
