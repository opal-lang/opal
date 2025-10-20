package planfmt

// ExecutionNode represents a node in the operator precedence tree.
// This handles operator chaining within a single step.
//
// The tree structure captures operator precedence:
//
//	Precedence (high to low): | > && > || > ;
//
// Example: echo "a" | grep "a" && echo "b" || echo "c"
//
//	Parsed as: ((echo "a" | grep "a") && echo "b") || echo "c"
type ExecutionNode interface {
	isExecutionNode()
}

// CommandNode is a leaf node - represents a single decorator invocation.
type CommandNode struct {
	Decorator string // "@shell", "@retry", "@parallel", etc.
	Args      []Arg  // Decorator arguments (sorted by Key)
	Block     []Step // Nested steps (for decorators with blocks)
}

func (*CommandNode) isExecutionNode() {}

// PipelineNode executes a chain of piped commands (cmd1 | cmd2 | cmd3).
// All commands run concurrently with stdout→stdin streaming.
type PipelineNode struct {
	Commands []CommandNode // All commands in the pipeline
}

func (*PipelineNode) isExecutionNode() {}

// AndNode executes left, then right only if left succeeded (exit 0).
// Implements bash && operator semantics.
type AndNode struct {
	Left  ExecutionNode
	Right ExecutionNode
}

func (*AndNode) isExecutionNode() {}

// OrNode executes left, then right only if left failed (exit != 0).
// Implements bash || operator semantics.
type OrNode struct {
	Left  ExecutionNode
	Right ExecutionNode
}

func (*OrNode) isExecutionNode() {}

// SequenceNode executes all nodes sequentially (semicolon operator).
// Always executes all nodes regardless of exit codes.
// Returns exit code of last node.
type SequenceNode struct {
	Nodes []ExecutionNode
}

func (*SequenceNode) isExecutionNode() {}
