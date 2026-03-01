// Package invariant provides contract assertions for Opal.
//
// This package implements Tiger Style safety principles: assertions are a force
// multiplier for discovering bugs. Use Precondition/Postcondition to express
// function contracts, and Invariant for internal consistency checks.
//
// All functions panic on violation - these are programming errors, not user errors.
package invariant

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
)

// Precondition checks an input contract at function entry.
// Panics with PRECONDITION VIOLATION if condition is false.
//
// Use this to validate function arguments and caller expectations.
//
// Example:
//
//	func Process(data []byte) error {
//	    invariant.Precondition(len(data) > 0, "data must not be empty")
//	    // ... work ...
//	}
func Precondition(condition bool, format string, args ...interface{}) {
	if !condition {
		fail("PRECONDITION", format, args...)
	}
}

// Postcondition checks an output contract before function return.
// Panics with POSTCONDITION VIOLATION if condition is false.
//
// Use this to validate function results and guarantees to caller.
//
// Example:
//
//	func Compute(x int) int {
//	    result := x * 2
//	    invariant.Postcondition(result > 0, "result must be positive")
//	    return result
//	}
func Postcondition(condition bool, format string, args ...interface{}) {
	if !condition {
		fail("POSTCONDITION", format, args...)
	}
}

// Invariant checks an internal invariant during function execution.
// Panics with INVARIANT VIOLATION if condition is false.
//
// Use this for loop progress checks, state consistency, and internal logic.
//
// Example:
//
//	prevPos := p.pos
//	for p.pos < len(p.events) {
//	    // ... process event ...
//	    invariant.Invariant(p.pos > prevPos, "position must advance")
//	    prevPos = p.pos
//	}
func Invariant(condition bool, format string, args ...interface{}) {
	if !condition {
		fail("INVARIANT", format, args...)
	}
}

// NotNil panics if value is nil.
// This is a precondition check for pointer/interface arguments.
//
// Example:
//
//	func Process(event *Event) {
//	    invariant.NotNil(event, "event")
//	    // ... work ...
//	}
func NotNil(value interface{}, name string) {
	if value == nil {
		fail("PRECONDITION", "%s must not be nil", name)
	}
	// Check for typed nil (e.g., (*T)(nil))
	// This uses reflection to detect nil pointers/interfaces
	if isNilValue(value) {
		fail("PRECONDITION", "%s must not be nil", name)
	}
}

// isNilValue checks if a value is a typed nil using reflection
func isNilValue(value interface{}) bool {
	if value == nil {
		return true
	}

	v := reflect.ValueOf(value)
	kind := v.Kind()

	// Check if the type can be nil
	switch kind {
	case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func:
		return v.IsNil()
	default:
		return false
	}
}

// InRange panics if value is outside [min, max].
// This is a precondition check for numeric arguments.
//
// Example:
//
//	func GetItem(index int) Item {
//	    invariant.InRange(index, 0, len(items)-1, "index")
//	    return items[index]
//	}
func InRange(value, minVal, maxVal int, name string) {
	if value < minVal || value > maxVal {
		fail("PRECONDITION", "%s must be in range [%d, %d], got %d",
			name, minVal, maxVal, value)
	}
}

// Positive panics if value <= 0.
// This is typically a postcondition check for generated IDs or counts.
//
// Example:
//
//	func GenerateID() uint64 {
//	    id := nextID()
//	    invariant.Positive(int(id), "id")
//	    return id
//	}
func Positive(value int, name string) {
	if value <= 0 {
		fail("POSTCONDITION", "%s must be positive, got %d", name, value)
	}
}

// ExpectNoError panics if error is not nil.
// This is a postcondition check for operations that should never fail.
//
// Example:
//
//	func ValidatePlan(plan *Plan) {
//	    err := plan.Validate()
//	    invariant.ExpectNoError(err, "plan validation")
//	}
func ExpectNoError(err error, msg string) {
	if err != nil {
		fail("POSTCONDITION", "%s must not fail: %v", msg, err)
	}
}

// ContextNotBackground panics if context is context.Background().
// This catches bugs where parent context should be passed but Background() is used instead.
//
// Use this to enforce proper context propagation for cancellation and timeouts.
// Only the root execution entry point (e.g., Execute()) should create a fresh context.
// All other functions MUST receive parent context as parameter.
//
// Example:
//
//	func executeRedirect(redirect *sdk.RedirectNode, ctx context.Context) int {
//	    invariant.ContextNotBackground(ctx, "executeRedirect")
//	    // ... use ctx for cancellation ...
//	}
//
// Why this matters:
//   - Prevents goroutine leaks when parent is cancelled
//   - Ensures timeouts propagate correctly
//   - Enables proper resource cleanup on cancellation
func ContextNotBackground(ctx context.Context, location string) {
	if ctx == nil {
		fail("PRECONDITION", "%s: context must not be nil", location)
	}
	// Import context package to compare
	// context.Background() returns the same singleton instance every time
	if ctx == context.Background() {
		fail("PRECONDITION", "%s: context must not be Background() - parent context required for cancellation", location)
	}
}

// fail panics with a formatted message including call stack context.
func fail(kind, format string, args ...interface{}) {
	// Capture call stack (skip fail() and wrapper function)
	pc := make([]uintptr, 10)
	n := runtime.Callers(3, pc)
	frames := runtime.CallersFrames(pc[:n])

	// Build violation message
	msg := fmt.Sprintf("%s VIOLATION: "+format, append([]interface{}{kind}, args...)...)

	// Add first frame for context (file:line where violation occurred)
	if frame, ok := frames.Next(); ok {
		msg += fmt.Sprintf("\n  at %s:%d", frame.File, frame.Line)
	}

	panic(msg)
}
