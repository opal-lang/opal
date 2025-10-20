package planfmt

import (
	"bytes"
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
	Header  PlanHeader
	Target  string   // Function/command being executed (e.g., "deploy")
	Steps   []Step   // List of steps (newline-separated statements)
	Secrets []Secret // Secrets to scrub from output (value decorators)
}

// Secret represents a resolved value that must be scrubbed from output.
// ALL value decorators produce secrets - even @env.HOME or @git.commit_hash
// could leak sensitive system information. Scrub everything by default.
//
// Two-track identity:
// - DisplayID: Opaque random ID shown to users (no length leak, no correlation)
// - RuntimeValue: Actual secret value (runtime only, never serialized)
type Secret struct {
	Key          string // Variable name (e.g., "db_password", "HOME", "commit_hash")
	RuntimeValue string // Actual resolved value (runtime only, never serialized)
	DisplayID    string // Opaque ID for display: opal:secret:3J98t56A
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
//
// IMPORTANT: Always set Tree when creating a Step!
// Binary format only serializes Tree (Commands field is ignored).
//
// Example:
//
//	step := Step{
//	    ID: 1,
//	    Tree: &CommandNode{
//	        Decorator: "@shell",
//	        Args: []Arg{{Key: "command", Val: Value{Kind: ValueString, Str: "echo hello"}}},
//	    },
//	}
//
// Binary Serialization:
// - Writer: Only Tree is serialized (Commands ignored, Tree must not be nil)
// - Reader: Only Tree is deserialized
//
// Invariants:
// - ID must be unique within a plan
// - Tree must not be nil (enforced by writer precondition)
type Step struct {
	ID   uint64        // Unique identifier (stable across plan versions)
	Tree ExecutionNode // Operator precedence tree (REQUIRED - must not be nil)
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

	// Validate tree (if present)
	if s.Tree != nil {
		return validateNode(s.Tree, s.ID, seen)
	}

	return nil
}

// validateNode recursively validates tree nodes
func validateNode(node ExecutionNode, stepID uint64, seen map[uint64]bool) error {
	switch n := node.(type) {
	case *CommandNode:
		// Check args are sorted
		for j := 1; j < len(n.Args); j++ {
			if n.Args[j-1].Key >= n.Args[j].Key {
				return fmt.Errorf("step %d: args not sorted (key %q >= %q)",
					stepID, n.Args[j-1].Key, n.Args[j].Key)
			}
		}
		// Validate block steps recursively
		for j := range n.Block {
			if err := n.Block[j].validate(seen); err != nil {
				return err
			}
		}

	case *PipelineNode:
		for i := range n.Commands {
			if err := validateNode(&n.Commands[i], stepID, seen); err != nil {
				return err
			}
		}

	case *AndNode:
		if err := validateNode(n.Left, stepID, seen); err != nil {
			return err
		}
		if err := validateNode(n.Right, stepID, seen); err != nil {
			return err
		}

	case *OrNode:
		if err := validateNode(n.Left, stepID, seen); err != nil {
			return err
		}
		if err := validateNode(n.Right, stepID, seen); err != nil {
			return err
		}

	case *SequenceNode:
		for i := range n.Nodes {
			if err := validateNode(n.Nodes[i], stepID, seen); err != nil {
				return err
			}
		}
	}

	return nil
}

// canonicalize sorts args in the execution tree
func (s *Step) canonicalize() {
	if s.Tree != nil {
		canonicalizeNode(s.Tree)
	}
}

// canonicalizeNode recursively sorts args in all tree nodes
func canonicalizeNode(node ExecutionNode) {
	switch n := node.(type) {
	case *CommandNode:
		// Sort args by key for deterministic encoding
		if len(n.Args) > 1 {
			sort.Slice(n.Args, func(i, j int) bool {
				return n.Args[i].Key < n.Args[j].Key
			})
		}
		// Recursively canonicalize block steps
		for i := range n.Block {
			n.Block[i].canonicalize()
		}

	case *PipelineNode:
		for i := range n.Commands {
			canonicalizeNode(&n.Commands[i])
		}

	case *AndNode:
		canonicalizeNode(n.Left)
		canonicalizeNode(n.Right)

	case *OrNode:
		canonicalizeNode(n.Left)
		canonicalizeNode(n.Right)

	case *SequenceNode:
		for i := range n.Nodes {
			canonicalizeNode(n.Nodes[i])
		}
	}
}

// Digest computes an unkeyed BLAKE2b-256 hash of the canonical plan bytes
// Used for: integrity checks, cache keys, deduplication
// This is about plan structure, NOT secret values
// Returns hex-encoded hash: "blake2b:a3f8b2c1d4e5f6a7..."
func (p *Plan) Digest() (string, error) {
	// Serialize plan to canonical bytes
	var buf bytes.Buffer
	hash, err := Write(&buf, p)
	if err != nil {
		return "", fmt.Errorf("failed to serialize plan for digest: %w", err)
	}

	// Return hex-encoded hash with algorithm prefix
	return fmt.Sprintf("blake2b:%x", hash), nil
}
