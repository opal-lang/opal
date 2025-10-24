package planfmt

// ExecutionNode represents a node in the operator precedence tree.
// This handles operator chaining within a single step.
//
// The tree structure captures operator precedence:
//
//	Precedence (high to low): | > redirect > && > || > ;
//
// Example: echo "a" | grep "a" > file.txt && echo "b" || echo "c"
//
//	Parsed as: (((echo "a" | grep "a") > file.txt) && echo "b") || echo "c"
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
// Commands can be CommandNode or RedirectNode (with invariants enforcing bash semantics).
type PipelineNode struct {
	Commands []ExecutionNode // All commands in the pipeline (CommandNode or RedirectNode only)
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

// RedirectMode specifies how to open the sink (overwrite or append).
type RedirectMode int

const (
	RedirectOverwrite RedirectMode = iota // > (truncate file)
	RedirectAppend                        // >> (append to file)
)

// RedirectNode redirects stdout from Source to Target decorator.
// Precedence: higher than &&, lower than |
//
// The target is ALWAYS a decorator that provides the sink:
//   - Static paths: @shell("output.txt") - shell opens the file
//   - Temp files: @file.temp() - decorator creates temp file
//   - S3 objects: @aws.s3.object("key") - decorator provides S3 writer
//   - HTTP: @http.post("url") - decorator provides HTTP writer
//
// Examples:
//
//	echo "hello" > output.txt
//	  → @shell("echo \"hello\"") > @shell("output.txt")
//
//	build > @file.temp()
//	  → @shell("build") > @file.temp()
//
//	logs >> @aws.s3.object("logs/app.log")
//	  → @shell("logs") >> @aws.s3.object("logs/app.log")
type RedirectNode struct {
	Source ExecutionNode // Command/pipeline producing output
	Target CommandNode   // Decorator providing the sink
	Mode   RedirectMode  // Overwrite or Append
}

func (*RedirectNode) isExecutionNode() {}
