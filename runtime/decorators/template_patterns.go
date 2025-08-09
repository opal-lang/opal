package decorators

import (
	"fmt"
	"time"
)

// Generic Go code patterns for resource management that any decorator can use
// These are building blocks, not decorator-specific templates

const (
	// ConcurrentExecutionPattern - Generic pattern for running multiple operations concurrently
	// Variables: .MaxConcurrency, .Operations[] (each has .Code)
	ConcurrentExecutionPattern = `{
	semaphore := make(chan struct{}, {{.MaxConcurrency}})
	var wg sync.WaitGroup
	errChan := make(chan error, {{len .Operations}})

	{{range $i, $op := .Operations}}
	wg.Add(1)
	go func() {
		defer wg.Done()
		
		// Acquire semaphore
		semaphore <- struct{}{}
		defer func() { <-semaphore }()

		// Execute operation with panic recovery
		result := func() (result CommandResult) {
			defer func() {
				if r := recover(); r != nil {
					result = CommandResult{Stdout: "", Stderr: fmt.Sprintf("panic in operation {{$i}}: %v", r), ExitCode: 1}
				}
			}()
			{{.Code}}
		}()
		if result.Failed() {
			errChan <- fmt.Errorf(result.Stderr)
			return
		}
		errChan <- nil
	}()
	{{end}}

	// Wait and collect errors
	go func() { wg.Wait(); close(errChan) }()
	var errors []string
	for err := range errChan {
		if err != nil {
			errors = append(errors, err.Error())
		}
	}
	if len(errors) > 0 {
		return CommandResult{Stdout: "", Stderr: strings.Join(errors, "; "), ExitCode: 1}
	}
	return CommandResult{Stdout: "", Stderr: "", ExitCode: 0}
}`

	// TimeoutPattern - Generic timeout wrapper for any operation
	// Variables: .DurationExpr (string) - idiomatic Go time expression like "30 * time.Second"
	TimeoutPattern = `{
	duration := {{.DurationExpr}} // Pre-validated duration

	timeoutCtx, cancel := context.WithTimeout(ctx.Context, duration)
	defer cancel()
	done := make(chan error, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- fmt.Errorf("panic during execution: %v", r)
			}
		}()
		
		// Check for cancellation
		select {
		case <-timeoutCtx.Done():
			done <- timeoutCtx.Err()
			return
		default:
		}

		result := func() (result CommandResult) {
			{{.Operation.Code}}
		}()
		if result.Failed() {
			done <- fmt.Errorf(result.Stderr)
			return
		}
		done <- nil
	}()

	select {
	case err := <-done:
		if err != nil {
			return CommandResult{Stdout: "", Stderr: err.Error(), ExitCode: 1}
		}
		return CommandResult{Stdout: "", Stderr: "", ExitCode: 0}
	case <-timeoutCtx.Done():
		return CommandResult{Stdout: "", Stderr: fmt.Sprintf("operation timed out after %s", duration), ExitCode: 124}
	}
}`

	// RetryPattern - Generic retry wrapper for any operation
	// Variables: .MaxAttempts, .DelayDuration (string), .Operation.Code
	RetryPattern = `{
	// Retry logic with {{.MaxAttempts}} attempts
	maxAttempts := {{.MaxAttempts}}
	delay := {{.DelayExpr}} // Pre-validated delay duration

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		result := func() (result CommandResult) {
			defer func() {
				if r := recover(); r != nil {
					result = CommandResult{Stdout: "", Stderr: fmt.Sprintf("panic in attempt %d: %v", attempt, r), ExitCode: 1}
				}
			}()
			{{.Operation.Code}}
		}()
		if result.Success() {
			break
		} else {
			lastErr = fmt.Errorf(result.Stderr)
			if attempt < maxAttempts {
				time.Sleep(delay)
			}
		}
	}
	if lastErr != nil {
		return CommandResult{Stdout: "", Stderr: fmt.Sprintf("all %d attempts failed, last error: %v", maxAttempts, lastErr), ExitCode: 1}
	}
	return CommandResult{Stdout: "", Stderr: "", ExitCode: 0}
}`

	// CancellableOperationPattern - Generic cancellable operation
	// Variables: .Operation.Code
	CancellableOperationPattern = `{
	cancellableCtx, cancel := context.WithCancel(ctx.Context)
	defer cancel()

	done := make(chan CommandResult, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- CommandResult{Stdout: "", Stderr: fmt.Sprintf("panic during execution: %v", r), ExitCode: 1}
			}
		}()

		select {
		case <-cancellableCtx.Done():
			done <- CommandResult{Stdout: "", Stderr: cancellableCtx.Err().Error(), ExitCode: 1}
			return
		default:
		}

		result := func() (result CommandResult) {
			{{.Operation.Code}}
		}()
		done <- result
	}()

	select {
	case result := <-done:
		return result
	case <-cancellableCtx.Done():
		return CommandResult{Stdout: "", Stderr: cancellableCtx.Err().Error(), ExitCode: 1}
	}
}`

	// SequentialExecutionPattern - Generic sequential execution with early termination
	// Variables: .Operations[] (each has .Code), .StopOnError (bool)
	SequentialExecutionPattern = `{
	{{range $i, $op := .Operations}}
	// Execute operation {{$i}}
	result{{$i}} := func() (result CommandResult) {
		defer func() {
			if r := recover(); r != nil {
				result = CommandResult{Stdout: "", Stderr: fmt.Sprintf("panic in operation {{$i}}: %v", r), ExitCode: 1}
			}
		}()
		
		{{.Code}}
	}()
	if result{{$i}}.Failed() {
		{{if $.StopOnError}}return result{{$i}}{{else}}// Continue on error{{end}}
	}
	{{end}}
	return CommandResult{Stdout: "", Stderr: "", ExitCode: 0}
}`

	// ConditionalExecutionPattern - Generic conditional execution
	// Variables: .Condition.Code, .ThenOperation.Code, .ElseOperation.Code (optional)
	ConditionalExecutionPattern = `{
	shouldExecute := func() bool {
		{{.Condition.Code}}
	}()

	if shouldExecute {
		result := func() (result CommandResult) {
			{{.ThenOperation.Code}}
		}()
		if result.Failed() {
			return result
		}
	}{{if .ElseOperation}} else {
		result := func() (result CommandResult) {
			{{.ElseOperation.Code}}
		}()
		if result.Failed() {
			return result
		}
	}{{end}}
	return CommandResult{Stdout: "", Stderr: "", ExitCode: 0}
}`

	// ResourceCleanupPattern - Generic resource cleanup with defer
	// Variables: .SetupCode, .Operation.Code, .CleanupCode
	ResourceCleanupPattern = `{
	// Setup resources
	{{.SetupCode}}
	
	// Ensure cleanup
	defer func() {
		{{.CleanupCode}}
	}()

	// Execute operation
	result := func() (result CommandResult) {
		{{.Operation.Code}}
	}()
	if result.Failed() {
		return result
	}
	return CommandResult{Stdout: "", Stderr: "", ExitCode: 0}
}`

	// ErrorCollectionPattern - Generic error collection from multiple operations
	// Variables: .Operations[] (each has .Code), .ContinueOnError (bool)
	ErrorCollectionPattern = `{
	var errors []error
	
	{{range $i, $op := .Operations}}
	// Execute operation {{$i}}
	result{{$i}} := func() (result CommandResult) {
		defer func() {
			if r := recover(); r != nil {
				result = CommandResult{Stdout: "", Stderr: fmt.Sprintf("panic in operation {{$i}}: %v", r), ExitCode: 1}
			}
		}()
		{{.Code}}
	}()
	if result{{$i}}.Failed() {
		errors = append(errors, fmt.Errorf(result{{$i}}.Stderr))
		{{if not $.ContinueOnError}}return CommandResult{Stdout: "", Stderr: result{{$i}}.Stderr, ExitCode: 1}{{end}}
	}
	{{end}}

	if len(errors) > 0 {
		if len(errors) == 1 {
			return CommandResult{Stdout: "", Stderr: errors[0].Error(), ExitCode: 1}
		}
		return CommandResult{Stdout: "", Stderr: fmt.Sprintf("multiple errors occurred: %v", errors), ExitCode: 1}
	}
	return CommandResult{Stdout: "", Stderr: "", ExitCode: 0}
}`

	// TryCatchFinallyPattern - Generic try-catch-finally pattern
	// Variables: .MainOperation.Code, .CatchOperation.Code (optional), .FinallyOperation.Code (optional), .HasCatch (bool), .HasFinally (bool)
	TryCatchFinallyPattern = `{
	var tryMainErr error
	var tryCatchErr error
	var tryFinallyErr error

	// Execute main block
	mainResult := {{.MainOperation.Code}}
	if mainResult.Failed() {
		tryMainErr = fmt.Errorf(mainResult.Stderr)
	}

	{{if .HasCatch}}
	// Execute catch block if main failed
	if tryMainErr != nil {
		catchResult := {{.CatchOperation.Code}}
		if catchResult.Failed() {
			tryCatchErr = fmt.Errorf(catchResult.Stderr)
			fmt.Fprintf(os.Stderr, "Catch block failed: %v\n", tryCatchErr)
		}
	}
	{{end}}

	{{if .HasFinally}}
	// Always execute finally block regardless of main/catch success
	finallyResult := {{.FinallyOperation.Code}}
	if finallyResult.Failed() {
		tryFinallyErr = fmt.Errorf(finallyResult.Stderr)
		fmt.Fprintf(os.Stderr, "Finally block failed: %v\n", tryFinallyErr)
	}
	{{end}}

	// Return the most significant error: main error takes precedence
	if tryMainErr != nil {
		return CommandResult{Stdout: "", Stderr: fmt.Sprintf("main block failed: %v", tryMainErr), ExitCode: 1}
	}
	if tryCatchErr != nil {
		return CommandResult{Stdout: "", Stderr: fmt.Sprintf("catch block failed: %v", tryCatchErr), ExitCode: 1}
	}
	if tryFinallyErr != nil {
		return CommandResult{Stdout: "", Stderr: fmt.Sprintf("finally block failed: %v", tryFinallyErr), ExitCode: 1}
	}
	return CommandResult{Stdout: "", Stderr: "", ExitCode: 0}
}`
)

// DurationToGoExpr converts a time.Duration to idiomatic Go time expression
func DurationToGoExpr(d time.Duration) string {
	// Handle common durations idiomatically
	switch {
	case d == 0:
		return "0"
	case d < time.Microsecond:
		return fmt.Sprintf("%d * time.Nanosecond", d.Nanoseconds())
	case d < time.Millisecond && d%time.Microsecond == 0:
		return fmt.Sprintf("%d * time.Microsecond", d.Microseconds())
	case d < time.Second && d%time.Millisecond == 0:
		return fmt.Sprintf("%d * time.Millisecond", d.Milliseconds())
	case d < time.Minute && d%time.Second == 0:
		return fmt.Sprintf("%d * time.Second", int64(d.Seconds()))
	case d < time.Hour && d%time.Minute == 0:
		return fmt.Sprintf("%d * time.Minute", int64(d.Minutes()))
	case d%time.Hour == 0:
		return fmt.Sprintf("%d * time.Hour", int64(d.Hours()))
	default:
		// For complex durations, use nanoseconds but with proper type
		return fmt.Sprintf("time.Duration(%d)", d.Nanoseconds())
	}
}

// Common import groups that patterns can reference
var (
	CoreImports        = []string{"fmt"}
	ConcurrencyImports = []string{"sync"}
	TimeImports        = []string{"time"}
	ContextImports     = []string{"context"}
	FileSystemImports  = []string{"os"}
	StringImports      = []string{"strings"}
)

// PatternImports maps each pattern to its required standard library imports
var PatternImports = map[string][]string{
	"ConcurrentExecutionPattern":  CombineImports(CoreImports, StringImports, ConcurrencyImports),
	"TimeoutPattern":              CombineImports(ContextImports, CoreImports, TimeImports),
	"RetryPattern":                CombineImports(CoreImports, TimeImports),
	"CancellableOperationPattern": CombineImports(ContextImports, CoreImports),
	"SequentialExecutionPattern":  CombineImports(CoreImports),
	"ConditionalExecutionPattern": CombineImports(CoreImports),
	"ResourceCleanupPattern":      CombineImports(CoreImports),
	"ErrorCollectionPattern":      CombineImports(CoreImports),
	"TryCatchFinallyPattern":      CombineImports(CoreImports, FileSystemImports),
}

// CombineImports merges multiple import slices and deduplicates them
func CombineImports(importGroups ...[]string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, group := range importGroups {
		for _, imp := range group {
			if !seen[imp] {
				seen[imp] = true
				result = append(result, imp)
			}
		}
	}

	return result
}

// StandardImportRequirement creates an ImportRequirement with standard library imports only
func StandardImportRequirement(importGroups ...[]string) ImportRequirement {
	return ImportRequirement{
		StandardLibrary: CombineImports(importGroups...),
		ThirdParty:      []string{},
		GoModules:       map[string]string{},
	}
}

// RequiresCore is a helper for decorators that need basic fmt functionality
func RequiresCore() ImportRequirement {
	return StandardImportRequirement(CoreImports)
}

// RequiresConcurrency is a helper for decorators that need concurrency primitives
func RequiresConcurrency() ImportRequirement {
	return StandardImportRequirement(CoreImports, ConcurrencyImports)
}

// RequiresTime is a helper for decorators that need time operations
func RequiresTime() ImportRequirement {
	return StandardImportRequirement(CoreImports, TimeImports)
}

// RequiresContext is a helper for decorators that need context operations
func RequiresContext() ImportRequirement {
	return StandardImportRequirement(CoreImports, ContextImports)
}

// RequiresFileSystem is a helper for decorators that need file system operations
func RequiresFileSystem() ImportRequirement {
	return StandardImportRequirement(CoreImports, FileSystemImports)
}

// RequiresResourceCleanup is a helper for decorators that use ResourceCleanupPattern
func RequiresResourceCleanup() ImportRequirement {
	return StandardImportRequirement(CoreImports)
}

// RequiresTryCatchFinally is a helper for decorators that use TryCatchFinallyPattern
func RequiresTryCatchFinally() ImportRequirement {
	return StandardImportRequirement(CoreImports, FileSystemImports)
}
