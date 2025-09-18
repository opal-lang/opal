package context

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"runtime"

	"github.com/aledsdavies/devcmd/core/decorators"
	"github.com/aledsdavies/devcmd/core/ir"
)

// ================================================================================================
// EXECUTION CONTEXT - Contains execution logic moved from core
// ================================================================================================

// Ctx carries state needed for command execution with execution methods
type Ctx struct {
	context.Context                    // Embedded Go context for cancellation
	Env             ir.EnvSnapshot     `json:"env"`      // Frozen environment snapshot
	Vars            map[string]string  `json:"vars"`     // CLI variables (@var) resolved to strings
	WorkDir         string             `json:"work_dir"` // Current working directory
	SysInfo         SystemInfoSnapshot `json:"sys_info"` // Frozen system information snapshot

	// IO streams
	Stdout io.Writer `json:"-"` // Standard output
	Stderr io.Writer `json:"-"` // Standard error
	Stdin  io.Reader `json:"-"` // Standard input

	// Execution flags
	DryRun bool `json:"dry_run"` // Plan mode - don't actually execute
	Debug  bool `json:"debug"`   // Debug mode - verbose output

	// UI configuration from standardized flags
	UI       *UIConfig          `json:"ui,omitempty"`        // UI behavior configuration
	UIConfig *UIConfig          `json:"ui_config,omitempty"` // Alias for backward compatibility
	Commands map[string]ir.Node `json:"commands,omitempty"`  // Available commands for @cmd decorator

	// Execution delegate for action decorators
	Executor ExecutionDelegate `json:"-"` // Delegate for executing actions within decorators
}

// UIConfig contains simplified UI behavior flags
type UIConfig struct {
	NoColor     bool `json:"no_color"`     // disable colored output
	AutoConfirm bool `json:"auto_confirm"` // auto-confirm all prompts (--yes)
}

// ExecutionDelegate provides action execution capability for decorator contexts
type ExecutionDelegate interface {
	ExecuteAction(ctx *Ctx, name string, args []decorators.Param) CommandResult
	ExecuteBlock(ctx *Ctx, name string, args []decorators.Param, innerSteps []ir.CommandStep) CommandResult
	ExecuteCommand(ctx *Ctx, commandName string) CommandResult
}

// CommandResult represents the result of executing a command or action
type CommandResult struct {
	Stdout   string `json:"stdout"`    // Standard output as string
	Stderr   string `json:"stderr"`    // Standard error as string
	ExitCode int    `json:"exit_code"` // Exit code (0 = success)
}

// Implement decorators.CommandResult interface
func (r CommandResult) GetStdout() string { return r.Stdout }
func (r CommandResult) GetStderr() string { return r.Stderr }
func (r CommandResult) GetExitCode() int  { return r.ExitCode }
func (r CommandResult) IsSuccess() bool   { return r.ExitCode == 0 }

// Additional convenience methods
func (r CommandResult) Success() bool { return r.IsSuccess() }
func (r CommandResult) Failed() bool  { return !r.IsSuccess() }

// ================================================================================================
// INTERFACE COMPLIANCE CHECKS
// ================================================================================================

// Ensure Ctx implements Context interface
var _ decorators.Context = (*Ctx)(nil)

// Ensure CommandResult implements decorators.CommandResult interface
var _ decorators.CommandResult = (*CommandResult)(nil)

// ================================================================================================
// EXECUTION CONTEXT INTERFACE IMPLEMENTATION
// ================================================================================================

// ExecShell executes a shell command and returns the result
func (ctx *Ctx) ExecShell(command string) decorators.CommandResult {
	if ctx.DryRun {
		return &CommandResult{
			Stdout:   fmt.Sprintf("[DRY RUN] Would execute: %s", command),
			Stderr:   "",
			ExitCode: 0,
		}
	}

	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = ctx.WorkDir

	// Set up environment
	cmd.Env = os.Environ()

	var stdout, stderr []byte
	var err error

	stdout, err = cmd.Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			stderr = exitError.Stderr
			return &CommandResult{
				Stdout:   string(stdout),
				Stderr:   string(stderr),
				ExitCode: exitError.ExitCode(),
			}
		}
		return &CommandResult{
			Stdout:   string(stdout),
			Stderr:   err.Error(),
			ExitCode: 1,
		}
	}

	return &CommandResult{
		Stdout:   string(stdout),
		Stderr:   "",
		ExitCode: 0,
	}
}

// GetEnv returns the value of an environment variable
func (ctx *Ctx) GetEnv(key string) (string, bool) {
	return ctx.Env.Get(key)
}

// SetEnv sets an environment variable (note: this only affects the context, not the actual process)
func (ctx *Ctx) SetEnv(key, value string) {
	if ctx.Env.Values == nil {
		ctx.Env.Values = make(map[string]string)
	}
	ctx.Env.Values[key] = value
}

// GetWorkingDir returns the current working directory
func (ctx *Ctx) GetWorkingDir() string {
	return ctx.WorkDir
}

// SetWorkingDir changes the working directory for subsequent operations
func (ctx *Ctx) SetWorkingDir(dir string) error {
	ctx.WorkDir = dir
	return nil
}

// GetVar returns a CLI variable value - immutable during execution
func (ctx *Ctx) GetVar(key string) (string, bool) {
	if ctx.Vars == nil {
		return "", false
	}
	value, exists := ctx.Vars[key]
	return value, exists
}

// SystemInfo returns system information interface
func (ctx *Ctx) SystemInfo() decorators.SystemInfo {
	return &ctx.SysInfo
}

// ================================================================================================
// SYSTEM INFO SNAPSHOT - Captured at initialization time
// ================================================================================================

// SystemInfoSnapshot implements decorators.SystemInfo with values captured at creation time
type SystemInfoSnapshot struct {
	NumCPU   int    `json:"num_cpu"`
	MemoryMB int64  `json:"memory_mb"`
	Hostname string `json:"hostname"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	TempDir  string `json:"temp_dir"`
	HomeDir  string `json:"home_dir"`
	UserName string `json:"user_name"`
}

// CaptureSystemInfo creates a snapshot of current system information
func CaptureSystemInfo() SystemInfoSnapshot {
	var sysInfo SystemInfoSnapshot

	// Capture hardware info
	sysInfo.NumCPU = runtime.NumCPU()
	sysInfo.MemoryMB = 8192 // TODO: Implement actual memory detection

	// Capture system identification
	if hostname, err := os.Hostname(); err == nil {
		sysInfo.Hostname = hostname
	} else {
		sysInfo.Hostname = "unknown"
	}
	sysInfo.OS = runtime.GOOS
	sysInfo.Arch = runtime.GOARCH

	// Capture runtime information
	sysInfo.TempDir = os.TempDir()
	if home, err := os.UserHomeDir(); err == nil {
		sysInfo.HomeDir = home
	}
	if user, err := user.Current(); err == nil {
		sysInfo.UserName = user.Username
	} else {
		sysInfo.UserName = "unknown"
	}

	return sysInfo
}

// Interface implementation methods
func (s *SystemInfoSnapshot) GetNumCPU() int      { return s.NumCPU }
func (s *SystemInfoSnapshot) GetMemoryMB() int64  { return s.MemoryMB }
func (s *SystemInfoSnapshot) GetHostname() string { return s.Hostname }
func (s *SystemInfoSnapshot) GetOS() string       { return s.OS }
func (s *SystemInfoSnapshot) GetArch() string     { return s.Arch }
func (s *SystemInfoSnapshot) GetTempDir() string  { return s.TempDir }
func (s *SystemInfoSnapshot) GetHomeDir() string  { return s.HomeDir }
func (s *SystemInfoSnapshot) GetUserName() string { return s.UserName }

// Prompt asks the user for input (implementation depends on UI configuration)
func (ctx *Ctx) Prompt(message string) (string, error) {
	if ctx.UI != nil && ctx.UI.AutoConfirm {
		return "", fmt.Errorf("prompt not available in auto-confirm mode")
	}

	_, _ = fmt.Fprintf(ctx.Stdout, "%s: ", message)
	// For now, return empty string - would need actual input handling
	return "", fmt.Errorf("interactive prompts not yet implemented")
}

// Confirm asks the user for yes/no confirmation
func (ctx *Ctx) Confirm(message string) (bool, error) {
	if ctx.UI != nil && ctx.UI.AutoConfirm {
		return true, nil
	}

	_, _ = fmt.Fprintf(ctx.Stdout, "%s (y/N): ", message)
	// For now, return false - would need actual input handling
	return false, fmt.Errorf("interactive confirmation not yet implemented")
}

// Log outputs a log message at the specified level
func (ctx *Ctx) Log(level decorators.LogLevel, message string) {
	prefix := ""
	switch level {
	case decorators.LogLevelDebug:
		if ctx.Debug {
			prefix = "[DEBUG] "
		} else {
			return
		}
	case decorators.LogLevelInfo:
		prefix = "[INFO] "
	case decorators.LogLevelWarn:
		prefix = "[WARN] "
	case decorators.LogLevelError:
		prefix = "[ERROR] "
	}

	_, _ = fmt.Fprintf(ctx.Stderr, "%s%s\n", prefix, message)
}

// Printf outputs a formatted message
func (ctx *Ctx) Printf(format string, args ...any) {
	_, _ = fmt.Fprintf(ctx.Stdout, format, args...)
}

// ================================================================================================
// COMMAND RESULT INTERFACE IMPLEMENTATION - Already implemented above
// ================================================================================================

// ================================================================================================
// CONTEXT METHODS
// ================================================================================================

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
			NoColor:     ctx.UI.NoColor,
			AutoConfirm: ctx.UI.AutoConfirm,
		}
	}

	return &Ctx{
		Env:      ctx.Env, // EnvSnapshot is immutable, safe to share
		Vars:     newVars,
		WorkDir:  ctx.WorkDir,
		SysInfo:  ctx.SysInfo, // SystemInfoSnapshot is immutable, safe to share
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
// CONTEXT CREATION HELPERS
// ================================================================================================

// CtxOptions contains options for creating a new execution context
type CtxOptions struct {
	WorkDir    string
	EnvOptions EnvOptions
	DryRun     bool
	Debug      bool
	UIConfig   *UIConfig
	Executor   ExecutionDelegate
	Vars       map[string]string  // CLI variables
	Commands   map[string]ir.Node // Available commands for @cmd decorator
}

// EnvOptions contains environment configuration
type EnvOptions struct {
	BlockList []string // Environment variables to exclude
}

// NewCtx creates a new execution context with the given options
func NewCtx(opts CtxOptions) (*Ctx, error) {
	// Create environment snapshot
	env := ir.EnvSnapshot{} // This would be implemented in core/ir if needed

	// Initialize vars map
	vars := opts.Vars
	if vars == nil {
		vars = make(map[string]string)
	}

	return &Ctx{
		Env:      env,
		Vars:     vars,
		WorkDir:  opts.WorkDir,
		SysInfo:  CaptureSystemInfo(), // Capture system info at context creation
		DryRun:   opts.DryRun,
		Debug:    opts.Debug,
		UI:       opts.UIConfig,
		UIConfig: opts.UIConfig, // Set both fields for compatibility
		Commands: opts.Commands, // FIX: Add missing Commands assignment
		Executor: opts.Executor,
	}, nil
}
