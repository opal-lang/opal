package executor

import (
	"context"
	"testing"
	"time"
)

// TestExecutionContext_ArgString tests retrieving string arguments
func TestExecutionContext_ArgString(t *testing.T) {
	// Given: Arguments with string value
	args := map[string]interface{}{
		"command": "echo hello",
	}

	// When: Creating execution context
	ctx := newExecutionContext(args, nil, context.Background())

	// Then: Can retrieve argument
	got := ctx.ArgString("command")
	want := "echo hello"
	if got != want {
		t.Errorf("ArgString() = %q, want %q", got, want)
	}
}

// TestExecutionContext_ArgString_Missing tests missing argument returns empty string
func TestExecutionContext_ArgString_Missing(t *testing.T) {
	// Given: Empty arguments
	args := map[string]interface{}{}

	// When: Creating execution context
	ctx := newExecutionContext(args, nil, context.Background())

	// Then: Returns empty string for missing argument
	got := ctx.ArgString("missing")
	want := ""
	if got != want {
		t.Errorf("ArgString(missing) = %q, want %q", got, want)
	}
}

// TestExecutionContext_Context tests retrieving Go context
func TestExecutionContext_Context(t *testing.T) {
	// Given: Execution context with Go context
	args := map[string]interface{}{}
	goCtx := context.Background()

	// When: Creating execution context
	ctx := newExecutionContext(args, nil, goCtx)

	// Then: Can retrieve Go context
	if ctx.Context() != goCtx {
		t.Error("Context() did not return the provided context")
	}
}

// TestExecutionContext_WithContext tests context wrapping
func TestExecutionContext_WithContext(t *testing.T) {
	// Given: Execution context
	args := map[string]interface{}{}
	ctx := newExecutionContext(args, nil, context.Background())

	// When: Wrapping with new context
	newGoCtx := context.WithValue(context.Background(), "key", "value")
	wrapped := ctx.WithContext(newGoCtx)

	// Then: New context has the wrapped Go context
	if wrapped.Context() != newGoCtx {
		t.Error("WithContext() did not wrap the context")
	}

	// And: Original context unchanged
	if ctx.Context() == newGoCtx {
		t.Error("WithContext() modified original context")
	}
}

// TestExecutionContext_WithEnviron tests environment isolation
func TestExecutionContext_WithEnviron(t *testing.T) {
	// Given: Execution context with original environment
	args := map[string]interface{}{}
	ctx := newExecutionContext(args, nil, context.Background())
	originalEnv := ctx.Environ()

	// When: Creating new context with modified environment
	newEnv := map[string]string{
		"TEST_VAR": "test_value",
		"FOO":      "bar",
	}
	wrapped := ctx.WithEnviron(newEnv)

	// Then: New context has the new environment
	wrappedEnv := wrapped.Environ()
	if wrappedEnv["TEST_VAR"] != "test_value" {
		t.Errorf("WithEnviron() new context missing TEST_VAR")
	}
	if wrappedEnv["FOO"] != "bar" {
		t.Errorf("WithEnviron() new context missing FOO")
	}

	// And: Original context unchanged
	if _, exists := originalEnv["TEST_VAR"]; exists {
		t.Error("WithEnviron() modified original context environment")
	}

	// And: Modifying the input map doesn't affect the context
	newEnv["ADDED"] = "after"
	if _, exists := wrapped.Environ()["ADDED"]; exists {
		t.Error("WithEnviron() did not deep copy environment")
	}
}

// TestExecutionContext_WithWorkdir_Absolute tests absolute path handling
func TestExecutionContext_WithWorkdir_Absolute(t *testing.T) {
	// Given: Execution context with original workdir
	args := map[string]interface{}{}
	ctx := newExecutionContext(args, nil, context.Background())
	originalWd := ctx.Workdir()

	// When: Creating new context with absolute path
	wrapped := ctx.WithWorkdir("/tmp/test")

	// Then: New context has the absolute path
	if wrapped.Workdir() != "/tmp/test" {
		t.Errorf("WithWorkdir(/tmp/test) = %q, want /tmp/test", wrapped.Workdir())
	}

	// And: Original context unchanged
	if ctx.Workdir() != originalWd {
		t.Error("WithWorkdir() modified original context")
	}
}

// TestExecutionContext_WithWorkdir_Relative tests relative path resolution
func TestExecutionContext_WithWorkdir_Relative(t *testing.T) {
	// Given: Execution context
	args := map[string]interface{}{}
	ctx := newExecutionContext(args, nil, context.Background())
	baseDir := ctx.Workdir()

	// When: Creating new context with relative path
	wrapped := ctx.WithWorkdir("subdir")

	// Then: Path is resolved relative to base
	expected := baseDir + "/subdir"
	if wrapped.Workdir() != expected {
		t.Errorf("WithWorkdir(subdir) = %q, want %q", wrapped.Workdir(), expected)
	}
}

// TestExecutionContext_WithWorkdir_ParentDir tests parent directory navigation
func TestExecutionContext_WithWorkdir_ParentDir(t *testing.T) {
	// Given: Execution context
	args := map[string]interface{}{}
	ctx := newExecutionContext(args, nil, context.Background())
	original := ctx.Workdir()

	// When: Going into subdir then back to parent
	wrapped := ctx.WithWorkdir("foo").WithWorkdir("..")

	// Then: Returns to original directory
	if wrapped.Workdir() != original {
		t.Errorf("WithWorkdir(foo).WithWorkdir(..) = %q, want %q", wrapped.Workdir(), original)
	}
}

// TestExecutionContext_WithWorkdir_Chained tests chained relative paths
func TestExecutionContext_WithWorkdir_Chained(t *testing.T) {
	// Given: Execution context
	args := map[string]interface{}{}
	ctx := newExecutionContext(args, nil, context.Background())
	baseDir := ctx.Workdir()

	// When: Chaining multiple relative paths
	wrapped := ctx.WithWorkdir("foo").WithWorkdir("bar")

	// Then: Paths are resolved sequentially
	expected := baseDir + "/foo/bar"
	if wrapped.Workdir() != expected {
		t.Errorf("WithWorkdir(foo).WithWorkdir(bar) = %q, want %q", wrapped.Workdir(), expected)
	}
}

// TestExecutionContext_WithWorkdir_DotSlash tests ./path handling
func TestExecutionContext_WithWorkdir_DotSlash(t *testing.T) {
	// Given: Execution context
	args := map[string]interface{}{}
	ctx := newExecutionContext(args, nil, context.Background())
	baseDir := ctx.Workdir()

	// When: Using ./subdir notation
	wrapped := ctx.WithWorkdir("./subdir")

	// Then: ./ is normalized away
	expected := baseDir + "/subdir"
	if wrapped.Workdir() != expected {
		t.Errorf("WithWorkdir(./subdir) = %q, want %q", wrapped.Workdir(), expected)
	}
}

// TestExecutionContext_WithWorkdir_ComplexPath tests complex relative paths
func TestExecutionContext_WithWorkdir_ComplexPath(t *testing.T) {
	// Given: Execution context
	args := map[string]interface{}{}
	ctx := newExecutionContext(args, nil, context.Background())
	baseDir := ctx.Workdir()

	// When: Using complex relative path with .. and .
	wrapped := ctx.WithWorkdir("foo/bar/../baz/./qux")

	// Then: Path is cleaned and normalized
	expected := baseDir + "/foo/baz/qux"
	if wrapped.Workdir() != expected {
		t.Errorf("WithWorkdir(foo/bar/../baz/./qux) = %q, want %q", wrapped.Workdir(), expected)
	}
}

// TestExecutionContext_WithWorkdir_AbsoluteOverridesRelative tests absolute path overrides
func TestExecutionContext_WithWorkdir_AbsoluteOverridesRelative(t *testing.T) {
	// Given: Execution context with relative path set
	args := map[string]interface{}{}
	ctx := newExecutionContext(args, nil, context.Background())
	withRelative := ctx.WithWorkdir("foo/bar")

	// When: Setting absolute path
	wrapped := withRelative.WithWorkdir("/tmp/test")

	// Then: Absolute path replaces relative
	if wrapped.Workdir() != "/tmp/test" {
		t.Errorf("WithWorkdir(/tmp/test) after relative = %q, want /tmp/test", wrapped.Workdir())
	}
}

// TestExecutionContext_IsolationForParallel tests that contexts are properly isolated
// This is critical for @parallel decorator where each branch must be independent
func TestExecutionContext_IsolationForParallel(t *testing.T) {
	// Given: Base execution context
	args := map[string]interface{}{}
	baseCtx := newExecutionContext(args, nil, context.Background())

	// When: Creating two "parallel" contexts with different environments
	env1 := map[string]string{"BRANCH": "A", "VALUE": "1"}
	env2 := map[string]string{"BRANCH": "B", "VALUE": "2"}
	ctx1 := baseCtx.WithEnviron(env1).WithWorkdir("/tmp/branch-a")
	ctx2 := baseCtx.WithEnviron(env2).WithWorkdir("/tmp/branch-b")

	// Then: Each context has its own isolated state
	if ctx1.Environ()["BRANCH"] != "A" {
		t.Error("ctx1 does not have isolated environment")
	}
	if ctx2.Environ()["BRANCH"] != "B" {
		t.Error("ctx2 does not have isolated environment")
	}

	// And: Workdirs are isolated
	if ctx1.Workdir() != "/tmp/branch-a" {
		t.Error("ctx1 does not have isolated workdir")
	}
	if ctx2.Workdir() != "/tmp/branch-b" {
		t.Error("ctx2 does not have isolated workdir")
	}

	// And: Base context unchanged
	if _, exists := baseCtx.Environ()["BRANCH"]; exists {
		t.Error("base context was modified")
	}
}

// TestExecutionContext_ChainedWrapping tests multiple levels of context wrapping
func TestExecutionContext_ChainedWrapping(t *testing.T) {
	// Given: Base execution context
	args := map[string]interface{}{}
	baseCtx := newExecutionContext(args, nil, context.Background())

	// When: Chaining multiple context modifications
	ctx1 := baseCtx.WithWorkdir("/tmp")
	ctx2 := ctx1.WithEnviron(map[string]string{"VAR": "value"})
	ctx3 := ctx2.WithContext(context.WithValue(context.Background(), "key", "val"))

	// Then: Each level preserves previous modifications
	if ctx3.Workdir() != "/tmp" {
		t.Error("chained context lost workdir")
	}
	if ctx3.Environ()["VAR"] != "value" {
		t.Error("chained context lost environment")
	}
	if ctx3.Context().Value("key") != "val" {
		t.Error("chained context lost Go context value")
	}

	// And: Earlier contexts unchanged
	if ctx1.Environ()["VAR"] == "value" {
		t.Error("earlier context was modified")
	}
}

// TestExecutionContext_ArgInt tests integer argument retrieval
func TestExecutionContext_ArgInt(t *testing.T) {
	// Given: Arguments with int value
	args := map[string]interface{}{
		"times": int64(3),
	}

	// When: Creating execution context
	ctx := newExecutionContext(args, nil, context.Background())

	// Then: Can retrieve int argument
	if got := ctx.ArgInt("times"); got != 3 {
		t.Errorf("ArgInt(times) = %d, want 3", got)
	}

	// And: Missing argument returns 0
	if got := ctx.ArgInt("missing"); got != 0 {
		t.Errorf("ArgInt(missing) = %d, want 0", got)
	}
}

// TestExecutionContext_ArgBool tests boolean argument retrieval
func TestExecutionContext_ArgBool(t *testing.T) {
	// Given: Arguments with bool value
	args := map[string]interface{}{
		"enabled": true,
	}

	// When: Creating execution context
	ctx := newExecutionContext(args, nil, context.Background())

	// Then: Can retrieve bool argument
	if got := ctx.ArgBool("enabled"); got != true {
		t.Errorf("ArgBool(enabled) = %v, want true", got)
	}

	// And: Missing argument returns false
	if got := ctx.ArgBool("missing"); got != false {
		t.Errorf("ArgBool(missing) = %v, want false", got)
	}
}

// TestExecutionContext_Args tests snapshot of all arguments
func TestExecutionContext_Args(t *testing.T) {
	// Given: Arguments with multiple types
	args := map[string]interface{}{
		"name":    "test",
		"count":   int64(42),
		"enabled": true,
	}

	// When: Creating execution context and getting args snapshot
	ctx := newExecutionContext(args, nil, context.Background())
	snapshot := ctx.Args()

	// Then: Snapshot contains all arguments
	if snapshot["name"] != "test" {
		t.Errorf("Args()[name] = %v, want test", snapshot["name"])
	}
	if snapshot["count"] != int64(42) {
		t.Errorf("Args()[count] = %v, want 42", snapshot["count"])
	}
	if snapshot["enabled"] != true {
		t.Errorf("Args()[enabled] = %v, want true", snapshot["enabled"])
	}
}

// TestExecutionContext_ContextCancellation tests that child contexts inherit cancellation
func TestExecutionContext_ContextCancellation(t *testing.T) {
	// Given: Base context with cancellation
	args := map[string]interface{}{}
	parentGoCtx, cancel := context.WithCancel(context.Background())
	baseCtx := newExecutionContext(args, nil, parentGoCtx)

	// When: Creating child context (simulating @timeout decorator)
	childGoCtx, _ := context.WithTimeout(baseCtx.Context(), 1*time.Hour)
	childCtx := baseCtx.WithContext(childGoCtx)

	// And: Cancelling parent
	cancel()

	// Then: Child context is also cancelled
	select {
	case <-childCtx.Context().Done():
		// Expected - child context cancelled when parent cancelled
	case <-time.After(100 * time.Millisecond):
		t.Error("Child context not cancelled when parent cancelled")
	}
}

// TestExecutionContext_ContextTimeout tests timeout propagation
func TestExecutionContext_ContextTimeout(t *testing.T) {
	// Given: Base context
	args := map[string]interface{}{}
	baseCtx := newExecutionContext(args, nil, context.Background())

	// When: Creating child context with short timeout (simulating @timeout decorator)
	childGoCtx, cancel := context.WithTimeout(baseCtx.Context(), 10*time.Millisecond)
	defer cancel()
	childCtx := baseCtx.WithContext(childGoCtx)

	// Then: Child context times out
	select {
	case <-childCtx.Context().Done():
		// Expected - context timed out
		if childCtx.Context().Err() != context.DeadlineExceeded {
			t.Errorf("Expected DeadlineExceeded, got %v", childCtx.Context().Err())
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Context did not timeout")
	}
}

// TestExecutionContext_NestedTimeouts tests that shortest timeout wins
func TestExecutionContext_NestedTimeouts(t *testing.T) {
	// Given: Base context with long timeout
	args := map[string]interface{}{}
	baseGoCtx, cancel1 := context.WithTimeout(context.Background(), 1*time.Hour)
	defer cancel1()
	baseCtx := newExecutionContext(args, nil, baseGoCtx)

	// When: Creating child context with shorter timeout
	childGoCtx, cancel2 := context.WithTimeout(baseCtx.Context(), 10*time.Millisecond)
	defer cancel2()
	childCtx := baseCtx.WithContext(childGoCtx)

	// Then: Child context times out first (shortest deadline wins)
	select {
	case <-childCtx.Context().Done():
		// Expected - child timeout is shorter
		if childCtx.Context().Err() != context.DeadlineExceeded {
			t.Errorf("Expected DeadlineExceeded, got %v", childCtx.Context().Err())
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Context did not timeout")
	}
}
