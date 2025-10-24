package invariant_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/aledsdavies/opal/core/invariant"
)

// TestPreconditionPass verifies Precondition does not panic when condition is true
func TestPreconditionPass(t *testing.T) {
	// Should not panic
	x := 1
	invariant.Precondition(true, "this should pass")
	invariant.Precondition(x == 1, "math works")
	invariant.Precondition(len("hello") > 0, "string not empty")
}

// TestPreconditionFail verifies Precondition panics with correct message
func TestPreconditionFail(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for false precondition")
		}
		msg := fmt.Sprintf("%v", r)
		if !strings.Contains(msg, "PRECONDITION VIOLATION") {
			t.Errorf("expected PRECONDITION VIOLATION, got: %s", msg)
		}
		if !strings.Contains(msg, "data must not be empty") {
			t.Errorf("expected custom message, got: %s", msg)
		}
		if !strings.Contains(msg, "at ") {
			t.Errorf("expected stack trace context, got: %s", msg)
		}
	}()

	invariant.Precondition(false, "data must not be empty")
}

// TestPostconditionPass verifies Postcondition does not panic when condition is true
func TestPostconditionPass(t *testing.T) {
	// Should not panic
	invariant.Postcondition(true, "this should pass")
	invariant.Postcondition(2+2 == 4, "math works")
}

// TestPostconditionFail verifies Postcondition panics with correct message
func TestPostconditionFail(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for false postcondition")
		}
		msg := fmt.Sprintf("%v", r)
		if !strings.Contains(msg, "POSTCONDITION VIOLATION") {
			t.Errorf("expected POSTCONDITION VIOLATION, got: %s", msg)
		}
		if !strings.Contains(msg, "result must be positive") {
			t.Errorf("expected custom message, got: %s", msg)
		}
	}()

	invariant.Postcondition(false, "result must be positive")
}

// TestInvariantPass verifies Invariant does not panic when condition is true
func TestInvariantPass(t *testing.T) {
	// Should not panic
	invariant.Invariant(true, "this should pass")
	pos := 5
	prevPos := 4
	invariant.Invariant(pos > prevPos, "position advanced")
}

// TestInvariantFail verifies Invariant panics with correct message
func TestInvariantFail(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for false invariant")
		}
		msg := fmt.Sprintf("%v", r)
		if !strings.Contains(msg, "INVARIANT VIOLATION") {
			t.Errorf("expected INVARIANT VIOLATION, got: %s", msg)
		}
		if !strings.Contains(msg, "position must advance") {
			t.Errorf("expected custom message, got: %s", msg)
		}
	}()

	invariant.Invariant(false, "position must advance")
}

// TestNotNilPass verifies NotNil does not panic for non-nil values
func TestNotNilPass(t *testing.T) {
	// Should not panic
	str := "hello"
	invariant.NotNil(str, "str")

	ptr := &str
	invariant.NotNil(ptr, "ptr")

	slice := []int{1, 2, 3}
	invariant.NotNil(slice, "slice")
}

// TestNotNilFail verifies NotNil panics for nil values
func TestNotNilFail(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for nil value")
		}
		msg := fmt.Sprintf("%v", r)
		if !strings.Contains(msg, "PRECONDITION VIOLATION") {
			t.Errorf("expected PRECONDITION VIOLATION, got: %s", msg)
		}
		if !strings.Contains(msg, "event must not be nil") {
			t.Errorf("expected 'event must not be nil', got: %s", msg)
		}
	}()

	var ptr *string
	invariant.NotNil(ptr, "event")
}

// TestInRangePass verifies InRange does not panic for values in range
func TestInRangePass(t *testing.T) {
	// Should not panic
	invariant.InRange(5, 0, 10, "index")
	invariant.InRange(0, 0, 10, "index")  // min boundary
	invariant.InRange(10, 0, 10, "index") // max boundary
}

// TestInRangeFail verifies InRange panics for values outside range
func TestInRangeFail(t *testing.T) {
	tests := []struct {
		name  string
		value int
		min   int
		max   int
	}{
		{"below_min", -1, 0, 10},
		{"above_max", 11, 0, 10},
		{"far_below", -100, 0, 10},
		{"far_above", 100, 0, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if r == nil {
					t.Fatal("expected panic for out of range value")
				}
				msg := fmt.Sprintf("%v", r)
				if !strings.Contains(msg, "PRECONDITION VIOLATION") {
					t.Errorf("expected PRECONDITION VIOLATION, got: %s", msg)
				}
				if !strings.Contains(msg, "must be in range") {
					t.Errorf("expected range message, got: %s", msg)
				}
				if !strings.Contains(msg, fmt.Sprintf("got %d", tt.value)) {
					t.Errorf("expected actual value %d in message, got: %s", tt.value, msg)
				}
			}()

			invariant.InRange(tt.value, tt.min, tt.max, "index")
		})
	}
}

// TestPositivePass verifies Positive does not panic for positive values
func TestPositivePass(t *testing.T) {
	// Should not panic
	invariant.Positive(1, "id")
	invariant.Positive(42, "count")
	invariant.Positive(999999, "large_value")
}

// TestPositiveFail verifies Positive panics for non-positive values
func TestPositiveFail(t *testing.T) {
	tests := []struct {
		name  string
		value int
	}{
		{"zero", 0},
		{"negative", -1},
		{"large_negative", -100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if r == nil {
					t.Fatal("expected panic for non-positive value")
				}
				msg := fmt.Sprintf("%v", r)
				if !strings.Contains(msg, "POSTCONDITION VIOLATION") {
					t.Errorf("expected POSTCONDITION VIOLATION, got: %s", msg)
				}
				if !strings.Contains(msg, "must be positive") {
					t.Errorf("expected 'must be positive', got: %s", msg)
				}
				if !strings.Contains(msg, fmt.Sprintf("got %d", tt.value)) {
					t.Errorf("expected actual value %d in message, got: %s", tt.value, msg)
				}
			}()

			invariant.Positive(tt.value, "step_id")
		})
	}
}

// TestFormattedMessages verifies formatted messages work correctly
func TestFormattedMessages(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic")
		}
		msg := fmt.Sprintf("%v", r)
		if !strings.Contains(msg, "position 42") {
			t.Errorf("expected formatted position, got: %s", msg)
		}
		if !strings.Contains(msg, "token EOF") {
			t.Errorf("expected formatted token, got: %s", msg)
		}
	}()

	pos := 42
	token := "EOF"
	invariant.Invariant(false, "stuck at position %d with token %s", pos, token)
}

// TestStackTraceContext verifies stack trace is included
func TestStackTraceContext(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic")
		}
		msg := fmt.Sprintf("%v", r)

		// Should include file:line context
		if !strings.Contains(msg, "at ") {
			t.Errorf("expected 'at' in stack trace, got: %s", msg)
		}
		if !strings.Contains(msg, "invariant_test.go:") {
			t.Errorf("expected file:line in stack trace, got: %s", msg)
		}
	}()

	invariant.Precondition(false, "test stack trace")
}

// Example usage in a function with contracts
func ExamplePrecondition() {
	processData := func(data []byte) {
		// INPUT CONTRACT
		invariant.Precondition(len(data) > 0, "data must not be empty")
		invariant.Precondition(len(data) < 1024, "data must be less than 1KB")

		// ... work ...
		fmt.Println("Processing", len(data), "bytes")
	}

	processData([]byte("hello"))
	// Output: Processing 5 bytes
}

// Example usage with loop invariant
func ExampleInvariant() {
	processEvents := func(events []string) {
		pos := 0
		prevPos := -1

		for pos < len(events) {
			// INVARIANT: position must advance
			invariant.Invariant(pos > prevPos, "position must advance")
			prevPos = pos

			fmt.Println("Event:", events[pos])
			pos++
		}
	}

	processEvents([]string{"start", "middle", "end"})
	// Output:
	// Event: start
	// Event: middle
	// Event: end
}

// Example usage with postcondition
func ExamplePostcondition() {
	generateID := func() int {
		id := 42 // Simulate ID generation

		// OUTPUT CONTRACT
		invariant.Postcondition(id > 0, "generated ID must be positive")

		return id
	}

	id := generateID()
	fmt.Println("Generated ID:", id)
	// Output: Generated ID: 42
}

// TestExpectNoErrorPass verifies ExpectNoError does not panic when error is nil
func TestExpectNoErrorPass(t *testing.T) {
	// Should not panic
	invariant.ExpectNoError(nil, "operation")
}

// TestExpectNoErrorFail verifies ExpectNoError panics when error is not nil
func TestExpectNoErrorFail(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for non-nil error")
		}
		msg := fmt.Sprintf("%v", r)
		if !strings.Contains(msg, "POSTCONDITION VIOLATION") {
			t.Errorf("expected POSTCONDITION VIOLATION, got: %s", msg)
		}
		if !strings.Contains(msg, "plan validation must not fail") {
			t.Errorf("expected context in message, got: %s", msg)
		}
	}()

	err := fmt.Errorf("validation failed")
	invariant.ExpectNoError(err, "plan validation")
}

// TestContextNotBackgroundPass verifies ContextNotBackground does not panic for valid contexts
func TestContextNotBackgroundPass(t *testing.T) {
	// Should not panic for contexts derived from Background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	invariant.ContextNotBackground(ctx, "test")

	// Should not panic for contexts with timeout
	ctxTimeout, cancelTimeout := context.WithTimeout(context.Background(), 0)
	defer cancelTimeout()
	invariant.ContextNotBackground(ctxTimeout, "test")
}

// TestContextNotBackgroundFailsOnBackground verifies ContextNotBackground panics for Background()
func TestContextNotBackgroundFailsOnBackground(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for context.Background()")
		}
		msg := fmt.Sprintf("%v", r)
		if !strings.Contains(msg, "PRECONDITION VIOLATION") {
			t.Errorf("expected PRECONDITION VIOLATION, got: %s", msg)
		}
		if !strings.Contains(msg, "context must not be Background()") {
			t.Errorf("expected Background() message, got: %s", msg)
		}
		if !strings.Contains(msg, "test location") {
			t.Errorf("expected location in message, got: %s", msg)
		}
	}()

	invariant.ContextNotBackground(context.Background(), "test location")
}

// TestContextNotBackgroundFailsOnNil verifies ContextNotBackground panics for nil context
func TestContextNotBackgroundFailsOnNil(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for nil context")
		}
		msg := fmt.Sprintf("%v", r)
		if !strings.Contains(msg, "context must not be nil") {
			t.Errorf("expected nil message, got: %s", msg)
		}
	}()

	invariant.ContextNotBackground(nil, "test location")
}
