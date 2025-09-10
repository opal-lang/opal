package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// ================================================================================================
// SHARED TYPES - Used by both interpreter and generated modes
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

// Args represents decorator arguments (simplified from ast.NamedParameter)
type Args map[string]interface{}

// GetString returns a string argument with fallback
func (a Args) GetString(key, fallback string) string {
	if val, exists := a[key]; exists {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return fallback
}

// GetBool returns a bool argument with fallback
func (a Args) GetBool(key string, fallback bool) bool {
	if val, exists := a[key]; exists {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return fallback
}

// GetDuration returns a duration argument with fallback
func (a Args) GetDuration(key string, fallback time.Duration) time.Duration {
	if val, exists := a[key]; exists {
		if d, ok := val.(time.Duration); ok {
			return d
		}
		if str, ok := val.(string); ok {
			if parsed, err := time.ParseDuration(str); err == nil {
				return parsed
			}
		}
	}
	return fallback
}

// Ctx represents execution context
type Ctx struct {
	Env     map[string]string // Environment variables
	Vars    map[string]string // CLI variables
	WorkDir string            // Current working directory
	DryRun  bool              // Plan mode - don't actually execute
	Debug   bool              // Debug mode - verbose output
}

// NewCtx creates a new execution context
func NewCtx() *Ctx {
	env := make(map[string]string)
	for _, e := range os.Environ() {
		if pair := strings.SplitN(e, "=", 2); len(pair) == 2 {
			env[pair[0]] = pair[1]
		}
	}

	workDir, _ := os.Getwd()

	return &Ctx{
		Env:     env,
		Vars:    make(map[string]string),
		WorkDir: workDir,
		DryRun:  false,
		Debug:   false,
	}
}

// Clone creates a copy of the context
func (ctx *Ctx) Clone() *Ctx {
	newEnv := make(map[string]string, len(ctx.Env))
	for k, v := range ctx.Env {
		newEnv[k] = v
	}

	newVars := make(map[string]string, len(ctx.Vars))
	for k, v := range ctx.Vars {
		newVars[k] = v
	}

	return &Ctx{
		Env:     newEnv,
		Vars:    newVars,
		WorkDir: ctx.WorkDir,
		DryRun:  ctx.DryRun,
		Debug:   ctx.Debug,
	}
}

// WithWorkDir returns a new context with updated working directory
func (ctx *Ctx) WithWorkDir(workDir string) *Ctx {
	newCtx := ctx.Clone()
	newCtx.WorkDir = workDir
	return newCtx
}

// We'll use the existing core/plan types instead of defining our own
// Import the plan package for ExecutionStep, ExecutionPlan etc.

// ================================================================================================
// EXECUTION PRIMITIVES - Shared helpers
// ================================================================================================

// ExecShell executes a shell command and returns the result
func ExecShell(ctx *Ctx, cmd string) CommandResult {
	if ctx.Debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] Executing: %s\n", cmd)
	}

	if ctx.DryRun {
		return CommandResult{
			Stdout:   fmt.Sprintf("[DRY-RUN] %s", cmd),
			ExitCode: 0,
		}
	}

	execCmd := exec.Command("sh", "-c", cmd)
	execCmd.Dir = ctx.WorkDir

	// Set environment variables
	execCmd.Env = os.Environ()
	for key, value := range ctx.Env {
		execCmd.Env = append(execCmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Capture output
	output, err := execCmd.CombinedOutput()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return CommandResult{
				Stdout:   string(output),
				Stderr:   string(output),
				ExitCode: exitError.ExitCode(),
			}
		}
		return CommandResult{
			Stderr:   err.Error(),
			ExitCode: 1,
		}
	}

	return CommandResult{
		Stdout:   string(output),
		ExitCode: 0,
	}
}

// ExecShellWithInput executes a shell command with stdin input
func ExecShellWithInput(ctx *Ctx, cmd, input string) CommandResult {
	if ctx.Debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] Executing with input: %s\n", cmd)
	}

	if ctx.DryRun {
		return CommandResult{
			Stdout:   fmt.Sprintf("[DRY-RUN] %s < %s", cmd, input),
			ExitCode: 0,
		}
	}

	execCmd := exec.Command("sh", "-c", cmd)
	execCmd.Dir = ctx.WorkDir
	execCmd.Stdin = strings.NewReader(input)

	// Set environment variables
	execCmd.Env = os.Environ()
	for key, value := range ctx.Env {
		execCmd.Env = append(execCmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Capture output
	output, err := execCmd.CombinedOutput()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return CommandResult{
				Stdout:   string(output),
				Stderr:   string(output),
				ExitCode: exitError.ExitCode(),
			}
		}
		return CommandResult{
			Stderr:   err.Error(),
			ExitCode: 1,
		}
	}

	return CommandResult{
		Stdout:   string(output),
		ExitCode: 0,
	}
}
