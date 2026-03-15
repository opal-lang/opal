package runtime

import "github.com/builtwithtofu/sigil/core/decorator"

// ExecNode represents an executable node in the runtime execution tree.
type ExecNode = decorator.ExecNode

// BranchExecutor executes independent branches for block-aware wrappers.
type BranchExecutor = decorator.BranchExecutor

// ExecContext provides runtime execution context.
type ExecContext = decorator.ExecContext
