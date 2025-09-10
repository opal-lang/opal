package plan

import (
	"crypto/sha256"
	"fmt"
	"hash"
	"sort"
	"strings"
	"time"
)

// ExecutionPlan represents a detailed plan of what would be executed in dry run mode
type ExecutionPlan struct {
	Steps   []ExecutionStep        `json:"steps"`
	Edges   []PlanEdge             `json:"edges"` // Graph edges for visualization
	Context map[string]interface{} `json:"context"`
	Summary PlanSummary            `json:"summary"`
}

// ExecutionStep represents a single step in the execution plan
type ExecutionStep struct {
	ID          string            `json:"id"`
	Type        StepType          `json:"type"`
	Description string            `json:"description"`
	Command     string            `json:"command,omitempty"`
	Decorator   *DecoratorInfo    `json:"decorator,omitempty"`
	Children    []ExecutionStep   `json:"children,omitempty"`
	Condition   *ConditionInfo    `json:"condition,omitempty"`
	Timing      *TimingInfo       `json:"timing,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// PlanEdge represents a connection between execution steps for graph visualization
type PlanEdge struct {
	FromID string                 `json:"from_id"`
	ToID   string                 `json:"to_id"`
	Kind   EdgeKind               `json:"kind"`
	Label  string                 `json:"label,omitempty"`
	Props  map[string]interface{} `json:"properties,omitempty"`
}

// EdgeKind represents the type of connection between steps
type EdgeKind string

const (
	EdgeThen      EdgeKind = "then"       // Sequential execution between sibling steps
	EdgeOnSuccess EdgeKind = "on_success" // && operator
	EdgeOnFailure EdgeKind = "on_failure" // || operator
	EdgePipe      EdgeKind = "pipe"       // | operator
	EdgeAppend    EdgeKind = "append"     // >> operator
	EdgeBranch    EdgeKind = "branch"     // Conditional branch selection
	EdgeParallel  EdgeKind = "parallel"   // Parallel execution
)

// StepType defines the core semantic types of execution steps
// This is intentionally minimal to support plugin-friendly architecture
type StepType string

const (
	StepShell     StepType = "shell"     // Direct shell command execution
	StepDecorator StepType = "decorator" // Any decorator (timeout, parallel, retry, etc.)
	StepSequence  StepType = "sequence"  // Sequential command group
)

// DecoratorInfo contains information about a decorator
type DecoratorInfo struct {
	Name       string                 `json:"name"`
	Type       string                 `json:"type"` // "function", "block", "pattern"
	Parameters map[string]interface{} `json:"parameters"`
	Imports    []string               `json:"imports"`
}

// ConditionInfo describes conditional execution logic
type ConditionInfo struct {
	Variable      string          `json:"variable"`
	Branches      []BranchInfo    `json:"branches"`
	DefaultBranch *BranchInfo     `json:"default_branch,omitempty"`
	Evaluation    ConditionResult `json:"evaluation"`
}

// BranchInfo describes a conditional branch
type BranchInfo struct {
	Pattern     string `json:"pattern"`
	Description string `json:"description"`
	WillExecute bool   `json:"will_execute"`
}

// ConditionResult describes the result of condition evaluation
type ConditionResult struct {
	CurrentValue   string `json:"current_value"`
	SelectedBranch string `json:"selected_branch"`
	Reason         string `json:"reason"`
}

// TimingInfo contains timing-related execution details
type TimingInfo struct {
	Timeout          *time.Duration `json:"timeout,omitempty"`
	RetryAttempts    int            `json:"retry_attempts,omitempty"`
	RetryDelay       *time.Duration `json:"retry_delay,omitempty"`
	EstimatedTime    *time.Duration `json:"estimated_time,omitempty"`
	ConcurrencyLimit int            `json:"concurrency_limit,omitempty"`
}

// ExecutionMode represents different modes of command execution
type ExecutionMode string

const (
	ModeSequential    ExecutionMode = "sequential"     // Default sequential execution
	ModeParallel      ExecutionMode = "parallel"       // Concurrent execution
	ModeConditional   ExecutionMode = "conditional"    // Branch-based execution
	ModeErrorHandling ExecutionMode = "error_handling" // Try/catch/finally patterns
	ModeRetry         ExecutionMode = "retry"          // Retry with backoff
	ModeTimeout       ExecutionMode = "timeout"        // Time-bounded execution
	// Plugins can define their own execution modes
)

// PlanSummary provides a high-level overview of the execution plan
type PlanSummary struct {
	TotalSteps        int                   `json:"total_steps"`
	ShellCommands     int                   `json:"shell_commands"`
	ExecutionModes    map[ExecutionMode]int `json:"execution_modes"` // Mode -> count
	DecoratorsUsed    []string              `json:"decorators_used"` // Decorator names for reference
	EstimatedDuration *time.Duration        `json:"estimated_duration,omitempty"`
	RequiredImports   []string              `json:"required_imports"`

	// Legacy fields for backward compatibility - can be removed later
	ConditionalBranches int  `json:"conditional_branches,omitempty"`
	ParallelSections    int  `json:"parallel_sections,omitempty"`
	HasErrorHandling    bool `json:"has_error_handling,omitempty"`
}

// ANSI color codes
const (
	ColorReset  = "\033[0m"
	ColorBold   = "\033[1m"
	ColorDim    = "\033[2m"
	ColorBlue   = "\033[34m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorCyan   = "\033[36m"
	ColorGray   = "\033[90m"
)

// String returns a human-readable representation of the execution plan
func (ep *ExecutionPlan) String() string {
	var builder strings.Builder

	// Get the command name from context if available
	commandName := "command"
	if name, exists := ep.Context["command_name"]; exists {
		if nameStr, ok := name.(string); ok {
			commandName = nameStr
		}
	}

	// Command header with color
	builder.WriteString(fmt.Sprintf("%s%s%s:%s\n", ColorBold, ColorBlue, commandName, ColorReset))

	// Format each step with the new tree structure
	for i, step := range ep.Steps {
		isLast := i == len(ep.Steps)-1
		builder.WriteString(ep.formatStepAesthetic(step, "", isLast))
	}

	return builder.String()
}

// StringNoColor returns a human-readable representation of the execution plan without ANSI colors
func (ep *ExecutionPlan) StringNoColor() string {
	var builder strings.Builder

	// Get the command name from context if available
	commandName := "command"
	if name, exists := ep.Context["command_name"]; exists {
		if nameStr, ok := name.(string); ok {
			commandName = nameStr
		}
	}

	// Command header without color
	builder.WriteString(fmt.Sprintf("%s:\n", commandName))

	// Format each step with the new tree structure (no colors)
	for i, step := range ep.Steps {
		isLast := i == len(ep.Steps)-1
		builder.WriteString(ep.formatStepAestheticNoColor(step, "", isLast))
	}

	// Trim any trailing newline to match expected test output
	result := builder.String()
	return strings.TrimRight(result, "\n")
}

// formatStepAesthetic formats a step using the new aesthetic tree format
func (ep *ExecutionPlan) formatStepAesthetic(step ExecutionStep, prefix string, isLast bool) string {
	var builder strings.Builder

	// Choose the appropriate tree characters
	var connector, nextPrefix string
	if isLast {
		connector = "└─ "
		nextPrefix = prefix + "   "
	} else {
		connector = "├─ "
		nextPrefix = prefix + "│  "
	}

	// Format based on step type - now plugin-friendly!
	switch step.Type {
	case StepShell:
		// Clean shell command formatting
		cmd := step.Command
		if cmd == "" {
			cmd = step.Description
		}

		// Truncate very long commands and add ellipses
		if len(cmd) > 80 {
			cmd = cmd[:77] + "..."
		}

		builder.WriteString(fmt.Sprintf("%s%s%s\n",
			prefix, connector, cmd))

	case StepDecorator:
		// Generic decorator formatting using metadata - fully plugin-friendly!
		decoratorName := step.Metadata["decorator"]
		if decoratorName == "" {
			decoratorName = "unknown"
		}

		// Get color from decorator metadata (decorator decides its own color)
		color := step.Metadata["color"]
		if color == "" {
			color = ColorGray // neutral default
		}

		// Get info text from decorator metadata (decorator builds its own display info)
		info := step.Metadata["info"]
		if info == "" && step.Description != "" {
			// Fallback to description if no info provided
			info = step.Description
		} else {
			// Use the raw @decorator format with optional info
			display := fmt.Sprintf("@%s", decoratorName)
			if info != "" {
				display += " " + info
			}
			info = display
		}

		builder.WriteString(fmt.Sprintf("%s%s%s%s%s\n",
			prefix, connector, color, info, ColorReset))

	case StepSequence:
		// Sequential group formatting
		builder.WriteString(fmt.Sprintf("%s%s%s%s%s\n",
			prefix, connector, ColorGray, step.Description, ColorReset))

	default:
		// Fallback formatting
		builder.WriteString(fmt.Sprintf("%s%s%s%s%s\n",
			prefix, connector, ColorGray, step.Description, ColorReset))
	}

	// Recursive child formatting - safe now with parse-time recursion detection
	for i, child := range step.Children {
		isLastChild := i == len(step.Children)-1
		builder.WriteString(ep.formatStepAesthetic(child, nextPrefix, isLastChild))
	}

	return builder.String()
}

// formatStepAestheticNoColor formats a step using the new aesthetic tree format without colors
func (ep *ExecutionPlan) formatStepAestheticNoColor(step ExecutionStep, prefix string, isLast bool) string {
	var builder strings.Builder

	// Choose the appropriate tree characters
	var connector, nextPrefix string
	if isLast {
		connector = "└─ "
		nextPrefix = prefix + "   "
	} else {
		connector = "├─ "
		nextPrefix = prefix + "│  "
	}

	// Format based on step type (no colors)
	switch step.Type {
	case StepShell:
		// Clean shell command formatting
		cmd := step.Command
		if cmd == "" {
			cmd = step.Description
		}

		// Truncate very long commands and add ellipses
		if len(cmd) > 80 {
			cmd = cmd[:77] + "..."
		}

		builder.WriteString(fmt.Sprintf("%s%s%s\n",
			prefix, connector, cmd))

	case StepDecorator:
		// Generic decorator formatting (no colors) - fully plugin-friendly!
		decoratorName := step.Metadata["decorator"]
		if decoratorName == "" {
			decoratorName = "unknown"
		}

		// Get info text from decorator metadata (no colors)
		info := step.Metadata["info"]
		if info == "" && step.Description != "" {
			// Fallback to description if no info provided
			info = step.Description
		} else {
			// Use the raw @decorator format with optional info
			display := fmt.Sprintf("@%s", decoratorName)
			if info != "" {
				// Use info as-is (decorators should provide clean info for no-color)
				display += " " + info
			}
			info = display
		}

		builder.WriteString(fmt.Sprintf("%s%s%s\n",
			prefix, connector, info))

	case StepSequence:
		// Sequential group formatting
		builder.WriteString(fmt.Sprintf("%s%s%s\n",
			prefix, connector, step.Description))

	default:
		// Fallback formatting
		builder.WriteString(fmt.Sprintf("%s%s%s\n",
			prefix, connector, step.Description))
	}

	// Recursive child formatting - safe now with parse-time recursion detection
	for i, child := range step.Children {
		isLastChild := i == len(step.Children)-1
		builder.WriteString(ep.formatStepAestheticNoColor(child, nextPrefix, isLastChild))
	}

	return builder.String()
}

// AddStep adds a step to the execution plan
func (ep *ExecutionPlan) AddStep(step ExecutionStep) {
	ep.Steps = append(ep.Steps, step)
	ep.updateSummary()
}

// updateSummary recalculates the plan summary
func (ep *ExecutionPlan) updateSummary() {
	summary := PlanSummary{
		DecoratorsUsed:  make([]string, 0),
		RequiredImports: make([]string, 0),
	}

	decoratorSet := make(map[string]bool)
	importSet := make(map[string]bool)

	ep.countStepsRecursive(ep.Steps, &summary, decoratorSet, importSet)

	// Convert sets to slices
	for decorator := range decoratorSet {
		summary.DecoratorsUsed = append(summary.DecoratorsUsed, decorator)
	}
	for imp := range importSet {
		summary.RequiredImports = append(summary.RequiredImports, imp)
	}

	ep.Summary = summary
}

// countStepsRecursive recursively counts steps and collects metadata
func (ep *ExecutionPlan) countStepsRecursive(steps []ExecutionStep, summary *PlanSummary, decorators map[string]bool, imports map[string]bool) {
	for _, step := range steps {
		summary.TotalSteps++

		if step.Type == StepShell {
			summary.ShellCommands++
		}

		// Use execution mode approach - fully plugin-friendly!
		if step.Type == StepDecorator {
			// Decorators declare their execution mode via metadata
			if modeStr := step.Metadata["execution_mode"]; modeStr != "" {
				mode := ExecutionMode(modeStr)
				if summary.ExecutionModes == nil {
					summary.ExecutionModes = make(map[ExecutionMode]int)
				}
				summary.ExecutionModes[mode]++

				// Maintain legacy fields for backward compatibility
				switch mode {
				case ModeParallel:
					summary.ParallelSections++
				case ModeConditional:
					summary.ConditionalBranches++
				case ModeErrorHandling:
					summary.HasErrorHandling = true
				}
			}
		}

		if step.Decorator != nil {
			decorators[step.Decorator.Name] = true
			if step.Decorator.Imports != nil {
				for _, imp := range step.Decorator.Imports {
					imports[imp] = true
				}
			}
		}

		// Recursively process children
		ep.countStepsRecursive(step.Children, summary, decorators, imports)
	}
}

// NewExecutionPlan creates a new empty execution plan
func NewExecutionPlan() *ExecutionPlan {
	return &ExecutionPlan{
		Steps:   make([]ExecutionStep, 0),
		Edges:   make([]PlanEdge, 0),
		Context: make(map[string]interface{}),
		Summary: PlanSummary{
			DecoratorsUsed:  make([]string, 0),
			RequiredImports: make([]string, 0),
		},
	}
}

// AssignStableIDs assigns stable hierarchical IDs to all steps (call once after building tree)
func (ep *ExecutionPlan) AssignStableIDs() {
	var walkSteps func([]int, *ExecutionStep)
	walkSteps = func(path []int, step *ExecutionStep) {
		step.ID = pathToID(path)
		for i := range step.Children {
			walkSteps(append(path, i), &step.Children[i])
		}
	}

	for i := range ep.Steps {
		walkSteps([]int{i}, &ep.Steps[i])
	}
}

// AddEdge adds a graph edge to the execution plan
func (ep *ExecutionPlan) AddEdge(edge PlanEdge) {
	ep.Edges = append(ep.Edges, edge)
}

// GraphHash returns a deterministic hash of the plan structure for parity testing
func (ep *ExecutionPlan) GraphHash() string {
	h := sha256.New()

	// Hash steps in a deterministic order
	ep.hashStepsRecursive(h, ep.Steps)

	// Hash edges in sorted order
	edges := make([]PlanEdge, len(ep.Edges))
	copy(edges, ep.Edges)
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].FromID != edges[j].FromID {
			return edges[i].FromID < edges[j].FromID
		}
		if edges[i].ToID != edges[j].ToID {
			return edges[i].ToID < edges[j].ToID
		}
		return string(edges[i].Kind) < string(edges[j].Kind)
	})

	for _, edge := range edges {
		_, _ = fmt.Fprintf(h, "edge:%s->%s:%s:%s\n", edge.FromID, edge.ToID, edge.Kind, edge.Label)
	}

	// Hash context in sorted order (excluding non-deterministic fields)
	var keys []string
	for k := range ep.Context {
		if k != "timestamp" && k != "process_id" { // Exclude non-deterministic fields
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	for _, k := range keys {
		_, _ = fmt.Fprintf(h, "ctx:%s:%v\n", k, ep.Context[k])
	}

	return fmt.Sprintf("%x", h.Sum(nil))[:16] // Return first 16 hex chars
}

// hashStepsRecursive recursively hashes steps in a deterministic order
func (ep *ExecutionPlan) hashStepsRecursive(h hash.Hash, steps []ExecutionStep) {
	for _, step := range steps {
		_, _ = fmt.Fprintf(h, "step:%s:%s:%s:%s\n", step.ID, step.Type, step.Description, step.Command)

		if step.Decorator != nil {
			_, _ = fmt.Fprintf(h, "decorator:%s:%s\n", step.Decorator.Name, step.Decorator.Type)
		}

		if step.Condition != nil {
			_, _ = fmt.Fprintf(h, "condition:%s:%s:%s\n",
				step.Condition.Variable,
				step.Condition.Evaluation.CurrentValue,
				step.Condition.Evaluation.SelectedBranch)
		}

		// Hash children recursively
		ep.hashStepsRecursive(h, step.Children)
	}
}

// pathToID converts a path slice to a stable ID string
func pathToID(path []int) string {
	if len(path) == 0 {
		return "0"
	}

	var parts []string
	for _, p := range path {
		parts = append(parts, fmt.Sprintf("%d", p))
	}
	return strings.Join(parts, "/")
}

// ToDOT exports the plan as DOT format for Graphviz
func (ep *ExecutionPlan) ToDOT() string {
	var builder strings.Builder

	builder.WriteString("digraph ExecutionPlan {\n")
	builder.WriteString("  rankdir=TB;\n")
	builder.WriteString("  node [shape=box];\n\n")

	// Add nodes
	ep.addDOTNodesRecursive(&builder, ep.Steps, "")

	// Add edges
	for _, edge := range ep.Edges {
		style := ""
		switch edge.Kind {
		case EdgeOnSuccess:
			style = " [color=green, label=\"&&\"]"
		case EdgeOnFailure:
			style = " [color=red, label=\"||\"]"
		case EdgePipe:
			style = " [color=blue, label=\"|\"]"
		case EdgeAppend:
			style = " [color=orange, label=\">>\"]"
		case EdgeParallel:
			style = " [color=purple, label=\"parallel\"]"
		case EdgeBranch:
			style = " [color=cyan, label=\"branch\"]"
		default:
			style = " [label=\"then\"]"
		}

		builder.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\"%s;\n",
			edge.FromID, edge.ToID, style))
	}

	builder.WriteString("}\n")
	return builder.String()
}

// addDOTNodesRecursive recursively adds nodes to DOT output
func (ep *ExecutionPlan) addDOTNodesRecursive(builder *strings.Builder, steps []ExecutionStep, prefix string) {
	for _, step := range steps {
		label := step.Description
		if step.Command != "" && step.Command != step.Description {
			label = step.Command
		}

		// Escape quotes in label
		label = strings.ReplaceAll(label, "\"", "\\\"")
		if len(label) > 50 {
			label = label[:47] + "..."
		}

		_, _ = fmt.Fprintf(builder, "  \"%s\" [label=\"%s\"];\n", step.ID, label)

		// Process children
		ep.addDOTNodesRecursive(builder, step.Children, prefix+"  ")
	}
}
