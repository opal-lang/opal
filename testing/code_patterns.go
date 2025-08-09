package testing

import (
	"fmt"
	"strconv"
	"strings"
)

// CodePattern represents a semantic pattern to validate in generated code
type CodePattern interface {
	Matches(code string) bool
	Description() string
}

// RetryPattern validates that code implements retry logic
type RetryPattern struct {
	MaxAttempts int
	HasDelay    bool
}

func (r RetryPattern) Matches(code string) bool {
	// Check for key retry components
	hasMaxAttempts := strings.Contains(code, "maxAttempts := "+strconv.Itoa(r.MaxAttempts))
	hasAttemptLoop := strings.Contains(code, "for attempt :=") && strings.Contains(code, "<= maxAttempts")
	hasErrorHandling := strings.Contains(code, "lastErr") || strings.Contains(code, "err == nil")

	if r.HasDelay {
		hasDelayLogic := strings.Contains(code, "time.Sleep") && strings.Contains(code, "delay")
		return hasMaxAttempts && hasAttemptLoop && hasErrorHandling && hasDelayLogic
	}

	return hasMaxAttempts && hasAttemptLoop && hasErrorHandling
}

func (r RetryPattern) Description() string {
	if r.HasDelay {
		return fmt.Sprintf("retry pattern with %d attempts and delay", r.MaxAttempts)
	}
	return fmt.Sprintf("retry pattern with %d attempts", r.MaxAttempts)
}

// TimeoutPattern validates that code implements timeout logic
type TimeoutPattern struct {
	Duration   string
	HasContext bool
	HasSelect  bool
	HasChannel bool
}

func (t TimeoutPattern) Matches(code string) bool {
	// Check for key timeout components
	hasDurationParsing := strings.Contains(code, "time.ParseDuration") && strings.Contains(code, `"`+t.Duration+`"`)

	if t.HasContext {
		hasContextTimeout := strings.Contains(code, "context.WithTimeout")
		hasCancel := strings.Contains(code, "defer cancel()")

		if t.HasSelect {
			hasSelectStatement := strings.Contains(code, "select {") && strings.Contains(code, "case <-ctx.Done():")

			if t.HasChannel {
				hasChannel := strings.Contains(code, "make(chan") && strings.Contains(code, "done")
				return hasDurationParsing && hasContextTimeout && hasCancel && hasSelectStatement && hasChannel
			}

			return hasDurationParsing && hasContextTimeout && hasCancel && hasSelectStatement
		}

		return hasDurationParsing && hasContextTimeout && hasCancel
	}

	return hasDurationParsing
}

func (t TimeoutPattern) Description() string {
	features := []string{"timeout with " + t.Duration}
	if t.HasContext {
		features = append(features, "context cancellation")
	}
	if t.HasSelect {
		features = append(features, "select statement")
	}
	if t.HasChannel {
		features = append(features, "done channel")
	}
	return strings.Join(features, ", ")
}

// ParallelPattern validates that code implements parallel execution
type ParallelPattern struct {
	NumOperations      int
	HasWaitGroup       bool
	HasErrorCollection bool
}

func (p ParallelPattern) Matches(code string) bool {
	// Check for key parallel components
	hasGoRoutines := strings.Count(code, "go func()") >= p.NumOperations

	if p.HasWaitGroup {
		hasWaitGroup := strings.Contains(code, "sync.WaitGroup") &&
			strings.Contains(code, "wg.Add(") &&
			strings.Contains(code, "wg.Done()") &&
			strings.Contains(code, "wg.Wait()")

		if p.HasErrorCollection {
			hasErrorHandling := strings.Contains(code, "errorsChan") || strings.Contains(code, "errors")
			return hasGoRoutines && hasWaitGroup && hasErrorHandling
		}

		return hasGoRoutines && hasWaitGroup
	}

	return hasGoRoutines
}

func (p ParallelPattern) Description() string {
	features := []string{fmt.Sprintf("parallel execution of %d operations", p.NumOperations)}
	if p.HasWaitGroup {
		features = append(features, "WaitGroup synchronization")
	}
	if p.HasErrorCollection {
		features = append(features, "error collection")
	}
	return strings.Join(features, ", ")
}

// ConditionalPattern validates that code implements conditional logic
type ConditionalPattern struct {
	ConditionType string // "env", "when", etc.
	HasBranches   bool
	HasDefault    bool
}

func (c ConditionalPattern) Matches(code string) bool {
	// Check for key conditional components
	hasConditionCheck := false

	switch c.ConditionType {
	case "env":
		hasConditionCheck = strings.Contains(code, "os.Getenv") || strings.Contains(code, "envContext")
	case "when":
		hasConditionCheck = strings.Contains(code, "switch") || strings.Contains(code, "if")
	default:
		hasConditionCheck = strings.Contains(code, "if") || strings.Contains(code, "switch")
	}

	if c.HasBranches {
		hasBranchLogic := strings.Contains(code, "case") || strings.Contains(code, "else")

		if c.HasDefault {
			hasDefaultCase := strings.Contains(code, "default:") || strings.Contains(code, "else")
			return hasConditionCheck && hasBranchLogic && hasDefaultCase
		}

		return hasConditionCheck && hasBranchLogic
	}

	return hasConditionCheck
}

func (c ConditionalPattern) Description() string {
	features := []string{"conditional execution"}
	if c.ConditionType != "" {
		features[0] = c.ConditionType + " conditional execution"
	}
	if c.HasBranches {
		features = append(features, "multiple branches")
	}
	if c.HasDefault {
		features = append(features, "default case")
	}
	return strings.Join(features, ", ")
}

// VariablePattern validates that code properly handles variables
type VariablePattern struct {
	VariableName string
	IsEnvVar     bool
	HasDefault   bool
}

func (v VariablePattern) Matches(code string) bool {
	if v.IsEnvVar {
		// Environment variable access
		hasEnvAccess := strings.Contains(code, "os.Getenv") || strings.Contains(code, "envContext")

		if v.HasDefault {
			hasDefaultLogic := strings.Contains(code, "if val :=") || strings.Contains(code, "default")
			return hasEnvAccess && hasDefaultLogic
		}

		return hasEnvAccess
	} else {
		// Regular variable declaration
		hasVarDeclaration := strings.Contains(code, v.VariableName+" :=")
		return hasVarDeclaration
	}
}

func (v VariablePattern) Description() string {
	if v.IsEnvVar {
		if v.HasDefault {
			return fmt.Sprintf("environment variable %s with default", v.VariableName)
		}
		return fmt.Sprintf("environment variable %s", v.VariableName)
	}
	return fmt.Sprintf("variable %s declaration", v.VariableName)
}
