package sdk

import (
	"context"
	"time"
)

// Step represents a unit of work to execute (runtime execution model).
// This is separate from planfmt.Step (binary serialization format).
//
// Knowledge domain: How to execute work
// NOT: How to serialize/deserialize plans
type Step struct {
	ID       uint64    // Unique identifier (from plan)
	Commands []Command // Commands in this step (operator-chained)
}

// Command represents a decorator invocation (runtime execution model).
// This is separate from planfmt.Command (binary serialization format).
//
// Knowledge domain: How to invoke decorators
// NOT: How to encode/decode binary format
type Command struct {
	Name     string                 // Decorator name: "shell", "retry", "parallel"
	Args     map[string]interface{} // Decorator arguments (typed values)
	Block    []Step                 // Nested steps (for decorators with blocks)
	Operator string                 // "&&", "||", "|", ";" - how to chain to NEXT command
}

// ExecutionContext provides execution environment for decorators.
// This is the interface decorators receive - it abstracts away the executor implementation.
//
// Design: Decorators depend on this interface (in core/sdk), runtime implements it.
// This avoids circular dependencies: core/sdk ← runtime/executor implements.
//
// Security: Decorators have NO direct I/O access. All output flows through
// the executor which automatically scrubs secrets.
type ExecutionContext interface {
	// ExecuteBlock executes nested steps within this context.
	// This enables recursive composition: @retry { @timeout { @shell {...} } }
	//
	// The executor calls back into itself to execute the block, allowing
	// decorators to wrap execution without knowing executor internals.
	ExecuteBlock(steps []Step) (exitCode int, err error)

	// Context returns the Go context for cancellation and deadlines.
	// Decorators should pass this to long-running operations.
	Context() context.Context

	// Argument accessors - typed access to decorator parameters
	// Returns zero value if argument doesn't exist or has wrong type
	ArgString(key string) string
	ArgInt(key string) int64
	ArgBool(key string) bool
	ArgDuration(key string) time.Duration

	// Args returns a snapshot of all arguments for logging/debugging.
	// Modifications to the returned map do NOT affect the context.
	Args() map[string]interface{}

	// Environment and working directory (immutable snapshots)
	// These are captured at context creation time to ensure isolation.
	// Changes to os.Getwd() or os.Setenv() do NOT affect this context.
	Environ() map[string]string
	Workdir() string

	// Context wrapping - returns NEW context with modifications
	// Original context is unchanged (immutable, copy-on-write)
	//
	// This enables decorators to modify execution environment:
	//   @aws.auth(profile="prod") {
	//       // This block runs with prod auth in environment
	//   }
	WithContext(ctx context.Context) ExecutionContext
	WithEnviron(env map[string]string) ExecutionContext
	WithWorkdir(dir string) ExecutionContext
}

// ExecutionHandler is the function signature for execution decorators.
// Decorators receive:
// - ctx: Execution context with args, environment, and ExecuteBlock callback
// - block: Child steps to execute (empty slice for leaf decorators)
//
// Block is optional - many decorators don't need it:
// - Leaf decorators: @shell("echo hi"), @file.write(...) - block is empty
// - Control flow: @retry(3) {...}, @parallel {...} - block has steps
//
// Returns:
// - exitCode: 0 for success, non-zero for failure
// - err: Error if decorator itself failed (not the command it ran)
//
// Error precedence (normative):
// 1. err != nil → Failure (exit code informational)
// 2. err == nil + exitCode == 0 → Success
// 3. err == nil + exitCode != 0 → Failure
type ExecutionHandler func(ctx ExecutionContext, block []Step) (exitCode int, err error)

// ValueHandler is the function signature for value decorators.
// Value decorators return data with no side effects - used at PLAN TIME.
//
// Key distinction:
// - Value decorators: Resolve values during planning (@env.HOME, @aws.secret.db_password)
// - Execution decorators: Perform/modify tasks during execution (@shell, @retry, @parallel)
//
// Value decorators can be interpolated in strings:
//
//	echo "Home: @env.HOME"  ← resolved at plan time
//
// Execution decorators cannot:
//
//	echo "@shell('ls')"  ← stays literal, not executed
//
// Examples: @env.HOME, @var.name, @aws.secret.db_password, @git.commit_hash
//
// Returns:
// - value: The resolved value (string, int, bool, etc.)
// - err: Error if resolution failed
type ValueHandler func(ctx ExecutionContext) (value interface{}, err error)
