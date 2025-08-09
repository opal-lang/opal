package decorators

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aledsdavies/devcmd/core/ast"
	"github.com/aledsdavies/devcmd/runtime/execution"
)

// ExecutionFunction represents a function that can be executed with error handling
type ExecutionFunction func() error

// ConcurrentExecutor provides utilities for safe parallel execution
type ConcurrentExecutor struct {
	concurrency int
}

// NewConcurrentExecutor creates a new concurrent executor
func NewConcurrentExecutor(concurrency int) *ConcurrentExecutor {
	return &ConcurrentExecutor{
		concurrency: concurrency,
	}
}

// Execute runs functions concurrently with proper context cancellation
func (ce *ConcurrentExecutor) Execute(functions []ExecutionFunction) error {
	if len(functions) == 0 {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // This is the correct pattern - cancel on exit

	errChan := make(chan error, len(functions))
	var wg sync.WaitGroup

	// Start all functions
	for _, fn := range functions {
		wg.Add(1)
		go func(f ExecutionFunction) {
			defer wg.Done()

			// Check cancellation before work
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			default:
			}

			if err := f(); err != nil {
				errChan <- err
			} else {
				errChan <- nil
			}
		}(fn)
	}

	// Wait for completion
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Collect errors
	var errors []error
	for err := range errChan {
		if err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("concurrent execution failed: %v", errors)
	}
	return nil
}

// Cleanup is a no-op since context handles cancellation
func (ce *ConcurrentExecutor) Cleanup() {
	// Context cancellation in defer handles cleanup
}

// TimeoutExecutor provides utilities for timeout-based execution
type TimeoutExecutor struct {
	timeout time.Duration
}

// NewTimeoutExecutor creates a new timeout executor
func NewTimeoutExecutor(timeout time.Duration) *TimeoutExecutor {
	return &TimeoutExecutor{
		timeout: timeout,
	}
}

// Execute runs a function with timeout
func (te *TimeoutExecutor) Execute(fn ExecutionFunction) error {
	ctx, cancel := context.WithTimeout(context.Background(), te.timeout)
	defer cancel() // This is the correct pattern

	done := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- fmt.Errorf("panic: %v", r)
			}
		}()
		done <- fn()
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("operation timed out after %v", te.timeout)
	}
}

// Cleanup is a no-op since context handles cancellation
func (te *TimeoutExecutor) Cleanup() {
	// Context cancellation in defer handles cleanup
}

// RetryExecutor provides utilities for retry-based execution
type RetryExecutor struct {
	maxAttempts int
	delay       time.Duration
}

// NewRetryExecutor creates a new retry executor
func NewRetryExecutor(maxAttempts int, delay time.Duration) *RetryExecutor {
	return &RetryExecutor{
		maxAttempts: maxAttempts,
		delay:       delay,
	}
}

// Execute runs a function with retry logic
func (re *RetryExecutor) Execute(fn ExecutionFunction) error {
	var lastErr error
	for attempt := 1; attempt <= re.maxAttempts; attempt++ {
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
			if attempt < re.maxAttempts {
				time.Sleep(re.delay)
			}
		}
	}
	return fmt.Errorf("all %d attempts failed, last error: %w", re.maxAttempts, lastErr)
}

// Cleanup is a no-op
func (re *RetryExecutor) Cleanup() {
	// No resources to clean up
}

// CommandExecutor provides utilities for executing AST CommandContent with proper type handling
type CommandExecutor struct{}

// NewCommandExecutor creates a new command executor
func NewCommandExecutor() *CommandExecutor {
	return &CommandExecutor{}
}

// ExecuteCommandWithInterpreter executes a single command using interpreter context
func (ce *CommandExecutor) ExecuteCommandWithInterpreter(ctx execution.InterpreterContext, cmd ast.CommandContent) error {
	switch c := cmd.(type) {
	case *ast.ShellContent:
		result := ctx.ExecuteShell(c)
		return result.Error
	case *ast.BlockDecorator:
		blockDecorator, err := GetBlock(c.Name)
		if err != nil {
			return fmt.Errorf("block decorator @%s not found: %w", c.Name, err)
		}
		result := blockDecorator.ExecuteInterpreter(ctx, c.Args, c.Content)
		return result.Error
	default:
		return fmt.Errorf("unsupported command content type: %T", cmd)
	}
}

// ExecuteCommandsWithInterpreter executes multiple commands sequentially
func (ce *CommandExecutor) ExecuteCommandsWithInterpreter(ctx execution.InterpreterContext, commands []ast.CommandContent) error {
	for _, cmd := range commands {
		if err := ce.ExecuteCommandWithInterpreter(ctx, cmd); err != nil {
			return err
		}
	}
	return nil
}

// Cleanup is a no-op
func (ce *CommandExecutor) Cleanup() {
	// No resources to clean up
}
