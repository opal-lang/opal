package planfmt

import (
	"fmt"
	"sort"
)

// Format limits (enforced by wire types):
// - Op/arg key/value strings: max 65,535 bytes (uint16 length prefix)
// - Args per step: max 65,535 (uint16 count)
// - Children per step: max 65,535 (uint16 count)
// - Recursion depth: max 1,000 levels (enforced by reader)
// - Header size: max 64 KB (enforced by reader)
// - Body size: max 32 MB (enforced by reader)
//
// Version compatibility:
// - Version field: uint16 encoded as major.minor (0x0001 = v1.0)
// - Breaking changes increment major, additions increment minor
// - Readers must reject versions with higher major number
// - Readers should accept higher minor versions (forward compatible)

// Plan is the in-memory representation of an execution plan.
// This is the stable contract between planner, executor, and formatters.
type Plan struct {
	Header PlanHeader
	Target string // Function/command being executed (e.g., "deploy")
	Steps  []Step // List of steps (newline-separated statements)
}

// PlanHeader contains metadata about the plan.
// Fields are designed for forward compatibility and versioning.
// Total size: 44 bytes (fixed)
type PlanHeader struct {
	SchemaID  [16]byte // UUID for this format schema version
	CreatedAt uint64   // Unix nanoseconds (UTC)
	Compiler  [16]byte // Build/commit fingerprint
	PlanKind  uint8    // 0=view, 1=contract, 2=executed
	_         [3]byte  // Reserved for future use (align to 8 bytes)
}

// Step represents a single step (newline-separated statement).
// A step can contain multiple commands chained with operators.
// Invariants:
// - ID must be unique within a plan
// - Commands order is semantically significant (operators depend on order)
// - Last command in step must have empty Operator
type Step struct {
	ID       uint64    // Unique identifier (stable across plan versions)
	Commands []Command // Commands in this step (operator-chained)
}

// Command represents a single decorator invocation within a step.
// Commands are chained with operators (&&, ||, |, ;) which are handled by bash.
// Invariants:
// - Args must be sorted by Key (for determinism)
// - Operator must be empty for last command in step
// - Block steps follow same rules as top-level steps
type Command struct {
	Decorator string // "@shell", "@retry", "@parallel", etc.
	Args      []Arg  // Sorted by Key for deterministic encoding
	Block     []Step // Nested steps (for decorators with blocks)
	Operator  string // "&&", "||", "|", ";" - how to chain to NEXT command (empty for last)
}

// Arg represents a typed argument to a decorator.
// Args are sorted by Key to ensure deterministic encoding.
type Arg struct {
	Key string
	Val Value
}

// Value is a union type for decorator arguments.
// Only one field should be set based on Kind.
type Value struct {
	Kind ValueKind

	// Union fields (only one valid per Kind)
	Str  string // For ValueString
	Int  int64  // For ValueInt
	Bool bool   // For ValueBool
	Ref  uint32 // For ValuePlaceholder (index into placeholder table)
}

// ValueKind identifies which field in Value is valid
type ValueKind uint8

const (
	ValueString      ValueKind = iota // Str field valid
	ValueInt                          // Int field valid
	ValueBool                         // Bool field valid
	ValuePlaceholder                  // Ref field valid (placeholder table index)
)

// Validate checks plan invariants
func (p *Plan) Validate() error {
	if len(p.Steps) == 0 {
		return nil // Empty plan is valid
	}

	// Check for duplicate step IDs
	seen := make(map[uint64]bool)
	for i := range p.Steps {
		if err := p.Steps[i].validate(seen); err != nil {
			return err
		}
	}
	return nil
}

// canonicalize ensures the plan is in canonical form for deterministic encoding.
// This sorts args by key within each command and recursively canonicalizes blocks.
// Must be called before writing to ensure deterministic output.
//
// Note: Command order is preserved (operator semantics depend on order).
// String comparison is bytewise (Go's native < operator).
func (p *Plan) canonicalize() {
	for i := range p.Steps {
		p.Steps[i].canonicalize()
	}
}

// validate checks step invariants recursively
func (s *Step) validate(seen map[uint64]bool) error {
	// Check ID uniqueness
	if seen[s.ID] {
		return fmt.Errorf("duplicate step ID: %d", s.ID)
	}
	seen[s.ID] = true

	// Validate each command
	for i, cmd := range s.Commands {
		// Check args are sorted
		for j := 1; j < len(cmd.Args); j++ {
			if cmd.Args[j-1].Key >= cmd.Args[j].Key {
				return fmt.Errorf("step %d command %d: args not sorted (key %q >= %q)",
					s.ID, i, cmd.Args[j-1].Key, cmd.Args[j].Key)
			}
		}

		// Last command must have empty operator
		if i == len(s.Commands)-1 && cmd.Operator != "" {
			return fmt.Errorf("step %d: last command has non-empty operator %q", s.ID, cmd.Operator)
		}

		// Non-last commands should have an operator
		if i < len(s.Commands)-1 && cmd.Operator == "" {
			return fmt.Errorf("step %d command %d: non-last command has empty operator", s.ID, i)
		}

		// Validate block steps recursively
		for j := range cmd.Block {
			if err := cmd.Block[j].validate(seen); err != nil {
				return err
			}
		}
	}

	return nil
}

// canonicalize sorts args and recursively canonicalizes blocks
func (s *Step) canonicalize() {
	// Canonicalize each command (preserve command order - operators depend on it)
	for i := range s.Commands {
		cmd := &s.Commands[i]

		// Sort args by key for deterministic encoding
		if len(cmd.Args) > 1 {
			sort.Slice(cmd.Args, func(j, k int) bool {
				return cmd.Args[j].Key < cmd.Args[k].Key
			})
		}

		// Recursively canonicalize block steps
		for j := range cmd.Block {
			cmd.Block[j].canonicalize()
		}
	}
}
