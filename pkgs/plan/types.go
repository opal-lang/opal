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

// String returns a human-readable representation of the execution plan
func (ep *ExecutionPlan) String() string {
	var builder strings.Builder

	builder.WriteString("=== Execution Plan ===\n")
	builder.WriteString(fmt.Sprintf("Total Steps: %d\n", ep.Summary.TotalSteps))
	builder.WriteString(fmt.Sprintf("Shell Commands: %d\n", ep.Summary.ShellCommands))

	if len(ep.Summary.DecoratorsUsed) > 0 {
		builder.WriteString(fmt.Sprintf("Decorators: %s\n", strings.Join(ep.Summary.DecoratorsUsed, ", ")))
	}

	if ep.Summary.EstimatedDuration != nil {
		builder.WriteString(fmt.Sprintf("Estimated Duration: %v\n", *ep.Summary.EstimatedDuration))
	}

	builder.WriteString("\n=== Execution Steps ===\n")
	for i, step := range ep.Steps {
		isLast := i == len(ep.Steps)-1
		builder.WriteString(ep.formatStepWithTree(step, "", isLast, i+1))
	}

	return builder.String()
}

// formatStepWithTree formats a step using tree-like ASCII art
func (ep *ExecutionPlan) formatStepWithTree(step ExecutionStep, prefix string, isLast bool, stepNum int) string {
	var builder strings.Builder

	// Choose the appropriate tree characters
	var connector, nextPrefix string
	if prefix == "" {
		// Root level
		connector = ""
		nextPrefix = ""
	} else {
		if isLast {
			connector = "└── "
			nextPrefix = prefix + "    "
		} else {
			connector = "├── "
			nextPrefix = prefix + "│   "
		}
	}

	// Format the main step line
	stepLabel := fmt.Sprintf("[%s] %s", step.Type, step.Description)
	if step.Command != "" {
		stepLabel += fmt.Sprintf(": %s", step.Command)
	}

	builder.WriteString(fmt.Sprintf("%s%s%s\n", prefix, connector, stepLabel))

	// Add decorator information as a child node if present
	if step.Decorator != nil {
		decoratorInfo := fmt.Sprintf("@%s", step.Decorator.Name)
		if len(step.Decorator.Parameters) > 0 {
			var params []string
			for k, v := range step.Decorator.Parameters {
				params = append(params, fmt.Sprintf("%s=%v", k, v))
			}
			decoratorInfo += fmt.Sprintf("(%s)", strings.Join(params, ", "))
		}

		hasMoreDetails := step.Condition != nil || step.Timing != nil || len(step.Children) > 0
		detailConnector := "└── "
		if hasMoreDetails {
			detailConnector = "├── "
		}
		builder.WriteString(fmt.Sprintf("%s%s🔧 %s\n", nextPrefix, detailConnector, decoratorInfo))
	}

	// Add condition information
	if step.Condition != nil {
		conditionInfo := fmt.Sprintf("Condition: %s = %s → %s",
			step.Condition.Variable,
			step.Condition.Evaluation.CurrentValue,
			step.Condition.Evaluation.SelectedBranch)

		hasMoreDetails := step.Timing != nil || len(step.Children) > 0
		detailConnector := "└── "
		if hasMoreDetails {
			detailConnector = "├── "
		}
		builder.WriteString(fmt.Sprintf("%s%s🔀 %s\n", nextPrefix, detailConnector, conditionInfo))
	}

	// Add timing information
	if step.Timing != nil {
		var timingDetails []string

		if step.Timing.Timeout != nil {
			timingDetails = append(timingDetails, fmt.Sprintf("timeout=%v", *step.Timing.Timeout))
		}
		if step.Timing.RetryAttempts > 0 {
			retry := fmt.Sprintf("retry=%d", step.Timing.RetryAttempts)
			if step.Timing.RetryDelay != nil {
				retry += fmt.Sprintf(" delay=%v", *step.Timing.RetryDelay)
			}
			timingDetails = append(timingDetails, retry)
		}
		if step.Timing.ConcurrencyLimit > 0 {
			timingDetails = append(timingDetails, fmt.Sprintf("concurrency=%d", step.Timing.ConcurrencyLimit))
		}

		if len(timingDetails) > 0 {
			hasMoreDetails := len(step.Children) > 0
			detailConnector := "└── "
			if hasMoreDetails {
				detailConnector = "├── "
			}
			builder.WriteString(fmt.Sprintf("%s%s⏱️  %s\n", nextPrefix, detailConnector, strings.Join(timingDetails, ", ")))
		}
	}

	// Format child steps
	for i, child := range step.Children {
		isLastChild := i == len(step.Children)-1
		childStepNum := i + 1
		builder.WriteString(ep.formatStepWithTree(child, nextPrefix, isLastChild, childStepNum))
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
