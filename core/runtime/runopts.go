package runtime

import "github.com/builtwithtofu/sigil/core/decorator"

// RunOpts configures command execution.
type RunOpts = decorator.RunOpts

// Result is the outcome of command execution.
type Result = decorator.Result

const (
	ExitSuccess  = decorator.ExitSuccess
	ExitCanceled = decorator.ExitCanceled
	ExitFailure  = decorator.ExitFailure
)
