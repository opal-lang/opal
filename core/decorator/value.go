package decorator

// Value is the interface for decorators that produce values.
// Value decorators are pure functions that resolve at plan-time.
// Examples: @var, @env, @aws.secret
//
// Resolve handles one or more calls in a single operation.
// Single call: Resolve(ctx, call)
// Batch: Resolve(ctx, call1, call2, call3)
//
// Decorators optimize internally:
// - @env batches into one process call
// - @aws.secret batches into one API call
// - @var just loops (no external calls)
type Value interface {
	Decorator
	Resolve(ctx ValueEvalContext, calls ...ValueCall) ([]ResolveResult, error)
}

// ResolveResult represents the outcome of resolving a single call.
type ResolveResult struct {
	// Value is the raw resolved value (will be wrapped in DisplayID by planner)
	Value any

	// Origin is the decorator path for this value (e.g., "@env.API_KEY", "var.count")
	// Used for audit trails and DisplayID generation
	Origin string

	// Error is the per-call error (nil if successful)
	Error error
}

// ValueEvalContext provides the execution context for value resolution.
type ValueEvalContext struct {
	// Session is the ambient execution context (env, cwd, transport)
	Session Session

	// Vault is the scope-aware variable storage (primary source of truth)
	// All variable lookups go through Vault (no direct variable access)
	Vault any // *vault.Vault (any to avoid circular import)

	// IDFactory creates deterministic secret IDs
	IDFactory any // TODO: Replace with actual secret.IDFactory type in Phase 2

	// PlanHash is the deterministic plan hash for secret ID generation
	PlanHash []byte

	// StepPath is the current step path for secret provenance
	StepPath string

	// Trace is the telemetry span for observability
	Trace Span
}

// ValueCall represents a call to a value decorator.
type ValueCall struct {
	// Path is the decorator path (e.g., "env", "var", "aws.secret")
	Path string

	// Primary is the primary parameter (e.g., @env.HOME → "HOME")
	// Nil if decorator doesn't use primary param syntax
	Primary *string

	// Params contains named parameters
	Params map[string]any
}

// ResolvedValue is the result of value resolution with secret wrapping.
type ResolvedValue struct {
	// Value is the raw resolved value
	Value any

	// Handle is the secret handle for scrubbing (nil if not a secret)
	Handle any // TODO: Replace with actual *secret.Handle type in Phase 2

	// DisplayID is the deterministic secret ID (e.g., "opal:s:3J98t56A")
	DisplayID string
}
