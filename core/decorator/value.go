package decorator

// Value is the interface for decorators that produce values.
// Value decorators are pure functions that resolve at plan-time.
// Examples: @var, @env, @aws.secret
type Value interface {
	Decorator
	Resolve(ctx ValueEvalContext, call ValueCall) (any, error)
}

// ValueEvalContext provides the execution context for value resolution.
type ValueEvalContext struct {
	// Session is the ambient execution context (env, cwd, transport)
	Session Session

	// Vars contains plan-time variable bindings
	Vars map[string]any

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
