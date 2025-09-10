package plan

import "fmt"

// ================================================================================================
// SIMPLE PLAN HELPERS - Basic building blocks for decorators
// ================================================================================================

// NewShellStep creates a basic shell execution step
func NewShellStep(command string) ExecutionStep {
	return ExecutionStep{
		Type:        StepShell,
		Description: command,
		Command:     command,
		Metadata:    make(map[string]string),
	}
}

// NewDecoratorStep creates a basic decorator step
func NewDecoratorStep(name string, stepType StepType) ExecutionStep {
	return ExecutionStep{
		Type:        stepType,
		Description: fmt.Sprintf("@%s", name),
		Metadata: map[string]string{
			"decorator": name,
		},
	}
}

// NewErrorStep creates a step for decorator errors
func NewErrorStep(decorator string, err error) ExecutionStep {
	return ExecutionStep{
		Type:        StepShell,
		Description: fmt.Sprintf("@%s(<error: %v>)", decorator, err),
		Command:     "",
		Metadata: map[string]string{
			"decorator": decorator,
			"error":     err.Error(),
		},
	}
}

// AddMetadata adds metadata to a step (mutates the step)
func AddMetadata(step *ExecutionStep, key, value string) {
	if step.Metadata == nil {
		step.Metadata = make(map[string]string)
	}
	step.Metadata[key] = value
}

// SetChildren sets the children of a step (mutates the step)
func SetChildren(step *ExecutionStep, children []ExecutionStep) {
	step.Children = children
}

// TruncateCommand truncates a command string for display
func TruncateCommand(cmd string, maxLen int) string {
	if len(cmd) <= maxLen {
		return cmd
	}
	if maxLen <= 3 {
		return "..."
	}
	return cmd[:maxLen-3] + "..."
}

// ================================================================================================
// TASK COUNTING UTILITIES - Common functionality for decorators
// ================================================================================================

// CountTasks recursively counts the number of executable tasks in a plan step
// This is useful for decorators like @parallel that need to know the task count
func CountTasks(step ExecutionStep) int {
	// If no children, this is a leaf node that counts as 1 task
	if len(step.Children) == 0 {
		return 1
	}

	// Count tasks in all children
	count := 0
	for _, child := range step.Children {
		count += CountTasks(child)
	}
	return count
}

// CountLeafSteps counts only leaf steps (steps with no children)
// This gives the number of actual executable commands
func CountLeafSteps(step ExecutionStep) int {
	if len(step.Children) == 0 {
		return 1
	}

	count := 0
	for _, child := range step.Children {
		count += CountLeafSteps(child)
	}
	return count
}

// CountShellCommands counts only shell command steps recursively
func CountShellCommands(step ExecutionStep) int {
	count := 0
	if step.Type == StepShell {
		count = 1
	}

	for _, child := range step.Children {
		count += CountShellCommands(child)
	}
	return count
}

// CountExecutableCommands counts actual executable commands (shell steps) for concurrency calculations
// This is what decorators like @parallel should use - it ignores wrapper/container steps
func CountExecutableCommands(step ExecutionStep) int {
	return CountShellCommands(step)
}

// CountDirectChildren counts only the direct children of a step
// This is what @parallel should use - it counts siblings that can run in parallel
func CountDirectChildren(step ExecutionStep) int {
	return len(step.Children)
}

// CountParallelTasks counts tasks that can be parallelized for block decorators
// For parallel execution, we want direct children, but if there's a wrapper,
// we look one level deeper to find the actual parallel tasks
func CountParallelTasks(step ExecutionStep) int {
	directChildren := len(step.Children)

	// If we have exactly 1 child and it's a wrapper/container (like "Inner commands"),
	// then the actual parallel tasks are its children
	if directChildren == 1 && step.Children[0].Description == "Inner commands" {
		return len(step.Children[0].Children)
	}

	// Otherwise, direct children are the parallel tasks
	return directChildren
}

// MinInt returns the minimum of multiple integers (utility for concurrency calculations)
func MinInt(values ...int) int {
	if len(values) == 0 {
		return 0
	}

	min := values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
	}
	return min
}
