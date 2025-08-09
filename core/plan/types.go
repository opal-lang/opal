package plan

import (
	"fmt"
	"strings"
	"time"
)

// ExecutionPlan represents a detailed plan of what would be executed in dry run mode
type ExecutionPlan struct {
	Steps   []ExecutionStep        `json:"steps"`
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

// StepType defines the type of execution step
type StepType string

const (
	StepShell       StepType = "shell"       // Direct shell command execution
	StepTimeout     StepType = "timeout"     // Commands with timeout wrapper
	StepParallel    StepType = "parallel"    // Concurrent command execution
	StepRetry       StepType = "retry"       // Commands with retry logic
	StepConditional StepType = "conditional" // Pattern-based conditional execution
	StepTryCatch    StepType = "try_catch"   // Error handling with fallbacks
	StepSequence    StepType = "sequence"    // Sequential command group
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

// PlanSummary provides a high-level overview of the execution plan
type PlanSummary struct {
	TotalSteps          int            `json:"total_steps"`
	ShellCommands       int            `json:"shell_commands"`
	DecoratorsUsed      []string       `json:"decorators_used"`
	EstimatedDuration   *time.Duration `json:"estimated_duration,omitempty"`
	RequiredImports     []string       `json:"required_imports"`
	ConditionalBranches int            `json:"conditional_branches"`
	ParallelSections    int            `json:"parallel_sections"`
	HasErrorHandling    bool           `json:"has_error_handling"`
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

	return builder.String()
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

	// Format based on step type
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

	case StepParallel:
		// Format parallel decorator with concurrency info
		concurrency := ""
		count := len(step.Children)
		if step.Timing != nil && step.Timing.ConcurrencyLimit > 0 {
			count = step.Timing.ConcurrencyLimit
		}
		concurrency = fmt.Sprintf("%s{%s%d%s concurrent%s}%s",
			ColorGray, ColorYellow, count, ColorGray, ColorGray, ColorReset)

		builder.WriteString(fmt.Sprintf("%s%s%s@parallel%s %s\n",
			prefix, connector, ColorYellow, ColorReset, concurrency))

	case StepTimeout:
		// Format timeout decorator with duration info
		duration := ""
		if step.Timing != nil && step.Timing.Timeout != nil {
			duration = fmt.Sprintf("%s{%s%s timeout%s}%s",
				ColorGray, ColorYellow, step.Timing.Timeout.String(), ColorGray, ColorReset)
		}

		builder.WriteString(fmt.Sprintf("%s%s%s@timeout%s %s\n",
			prefix, connector, ColorCyan, ColorReset, duration))

	case StepRetry:
		// Format retry decorator with attempt info
		attempts := ""
		if step.Timing != nil && step.Timing.RetryAttempts > 0 {
			attempts = fmt.Sprintf("%s{%s%d%s attempts",
				ColorGray, ColorYellow, step.Timing.RetryAttempts, ColorGray)
			if step.Timing.RetryDelay != nil {
				attempts += fmt.Sprintf(", %s%s%s delay", ColorYellow, step.Timing.RetryDelay.String(), ColorGray)
			}
			attempts += fmt.Sprintf("}%s", ColorReset)
		}

		builder.WriteString(fmt.Sprintf("%s%s%s@retry%s %s\n",
			prefix, connector, ColorYellow, ColorReset, attempts))

	case StepConditional:
		// Format conditional decorator with evaluation info
		evalInfo := ""
		if step.Condition != nil {
			evalInfo = fmt.Sprintf("%s{%s%s%s = %s%s%s → %s%s%s}%s",
				ColorGray,
				ColorYellow, step.Condition.Variable, ColorGray,
				ColorYellow, step.Condition.Evaluation.CurrentValue, ColorGray,
				ColorYellow, step.Condition.Evaluation.SelectedBranch, ColorGray,
				ColorReset)
		}

		builder.WriteString(fmt.Sprintf("%s%s%s@when%s %s\n",
			prefix, connector, ColorCyan, ColorReset, evalInfo))

	default:
		// Generic step formatting
		builder.WriteString(fmt.Sprintf("%s%s%s%s%s\n",
			prefix, connector, ColorGray, step.Description, ColorReset))
	}

	// Format child steps recursively
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

	case StepParallel:
		// Format parallel decorator with concurrency info (no colors)
		count := len(step.Children)
		if step.Timing != nil && step.Timing.ConcurrencyLimit > 0 {
			count = step.Timing.ConcurrencyLimit
		}
		concurrency := fmt.Sprintf("{%d concurrent}", count)

		builder.WriteString(fmt.Sprintf("%s%s@parallel %s\n",
			prefix, connector, concurrency))

	case StepTimeout:
		// Format timeout decorator with duration info (no colors)
		duration := ""
		if step.Timing != nil && step.Timing.Timeout != nil {
			duration = fmt.Sprintf("{%s timeout}", step.Timing.Timeout.String())
		}

		builder.WriteString(fmt.Sprintf("%s%s@timeout %s\n",
			prefix, connector, duration))

	case StepRetry:
		// Format retry decorator with attempt info (no colors)
		attempts := ""
		if step.Timing != nil && step.Timing.RetryAttempts > 0 {
			attempts = fmt.Sprintf("{%d attempts", step.Timing.RetryAttempts)
			if step.Timing.RetryDelay != nil {
				attempts += fmt.Sprintf(", %s delay", step.Timing.RetryDelay.String())
			}
			attempts += "}"
		}

		builder.WriteString(fmt.Sprintf("%s%s@retry %s\n",
			prefix, connector, attempts))

	case StepConditional:
		// Format conditional decorator with evaluation info (no colors)
		evalInfo := ""
		if step.Condition != nil {
			evalInfo = fmt.Sprintf("{%s = %s → %s}",
				step.Condition.Variable,
				step.Condition.Evaluation.CurrentValue,
				step.Condition.Evaluation.SelectedBranch)
		}

		builder.WriteString(fmt.Sprintf("%s%s@when %s\n",
			prefix, connector, evalInfo))

	default:
		// Generic step formatting
		builder.WriteString(fmt.Sprintf("%s%s%s\n",
			prefix, connector, step.Description))
	}

	// Format child steps recursively
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

		if step.Type == StepParallel {
			summary.ParallelSections++
		}

		if step.Type == StepConditional {
			summary.ConditionalBranches++
		}

		if step.Type == StepTryCatch {
			summary.HasErrorHandling = true
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
		Context: make(map[string]interface{}),
		Summary: PlanSummary{
			DecoratorsUsed:  make([]string, 0),
			RequiredImports: make([]string, 0),
		},
	}
}
