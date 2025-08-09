package decorators

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aledsdavies/devcmd/core/ast"
	decoratortesting "github.com/aledsdavies/devcmd/testing"
)

func TestWorkdirDecorator_Basic(t *testing.T) {
	decorator := &WorkdirDecorator{}

	// Test basic workdir functionality with existing directory
	content := []ast.CommandContent{
		decoratortesting.Shell("pwd"),
		decoratortesting.Shell("echo 'in workdir'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "path", Value: &ast.StringLiteral{Value: "/tmp"}},
		}, content)

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		GeneratorSucceeds().
		GeneratorProducesValidGo().
		GeneratorExecutesCorrectly().
		GeneratorCodeContains("/tmp", "workdir").
		PlanSucceeds().
		PlanReturnsElement("workdir").
		CompletesWithin("1s").
		SupportsNesting().
		Validate()

	if len(errors) > 0 {
		t.Errorf("WorkdirDecorator basic test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestWorkdirDecorator_CreateIfNotExists(t *testing.T) {
	decorator := &WorkdirDecorator{}

	// Use a unique directory name to avoid conflicts
	testDir := filepath.Join(os.TempDir(), "devcmd_test_"+decoratortesting.RandomString(8))
	defer func() {
		if err := os.RemoveAll(testDir); err != nil {
			t.Logf("Warning: failed to clean up test dir %s: %v", testDir, err)
		}
	}() // Clean up after test

	content := []ast.CommandContent{
		decoratortesting.Shell("pwd"),
		decoratortesting.Shell("echo 'created directory'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "path", Value: &ast.StringLiteral{Value: testDir}},
			{Name: "createIfNotExists", Value: &ast.BooleanLiteral{Value: true}},
		}, content)

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		GeneratorSucceeds().
		GeneratorCodeContains("os.MkdirAll", testDir).
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("WorkdirDecorator create if not exists test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestWorkdirDecorator_NonExistentDirectory(t *testing.T) {
	decorator := &WorkdirDecorator{}

	// Use a directory that definitely doesn't exist
	nonExistentDir := "/nonexistent/path/that/should/not/exist"

	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'should not run'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "path", Value: &ast.StringLiteral{Value: nonExistentDir}},
		}, content)

	// Note: Interpreter will fail, but generator and plan should work
	errors := decoratortesting.Assert(result).
		GeneratorSucceeds().
		GeneratorCodeContains("os.Stat", nonExistentDir).
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("WorkdirDecorator nonexistent directory test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestWorkdirDecorator_RelativePath(t *testing.T) {
	decorator := &WorkdirDecorator{}

	// Test with relative path
	content := []ast.CommandContent{
		decoratortesting.Shell("pwd"),
		decoratortesting.Shell("echo 'relative path'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "path", Value: &ast.StringLiteral{Value: "../"}},
		}, content)

	// Relative paths with .. are now blocked for security (directory traversal)
	errors := decoratortesting.Assert(result).
		InterpreterFails("directory traversal").
		GeneratorFails("directory traversal").
		PlanFails("directory traversal").
		Validate()

	if len(errors) > 0 {
		t.Errorf("WorkdirDecorator relative path test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestWorkdirDecorator_DotPath(t *testing.T) {
	decorator := &WorkdirDecorator{}

	// Test with current directory
	content := []ast.CommandContent{
		decoratortesting.Shell("pwd"),
		decoratortesting.Shell("echo 'current directory'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "path", Value: &ast.StringLiteral{Value: "."}},
		}, content)

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		GeneratorSucceeds().
		GeneratorProducesValidGo().
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("WorkdirDecorator dot path test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestWorkdirDecorator_NestedCommands(t *testing.T) {
	decorator := &WorkdirDecorator{}

	// Test with multiple commands
	content := []ast.CommandContent{
		decoratortesting.Shell("pwd"),
		decoratortesting.Shell("ls -la"),
		decoratortesting.Shell("echo 'multiple commands'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "path", Value: &ast.StringLiteral{Value: "/tmp"}},
		}, content)

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		GeneratorSucceeds().
		GeneratorProducesValidGo().
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("WorkdirDecorator nested commands test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestWorkdirDecorator_MultipleShellCommands(t *testing.T) {
	decorator := &WorkdirDecorator{}

	// Test workdir with multiple shell commands to verify they all run in the correct directory
	content := []ast.CommandContent{
		decoratortesting.Shell("pwd"),
		decoratortesting.Shell("echo 'first command'"),
		decoratortesting.Shell("echo 'second command'"),
		decoratortesting.Shell("pwd"), // Should still be in workdir
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "path", Value: &ast.StringLiteral{Value: "/tmp"}},
		}, content)

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		GeneratorSucceeds().
		GeneratorProducesValidGo().
		GeneratorCodeContains("/tmp").
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("WorkdirDecorator multiple shell commands test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestWorkdirDecorator_InvalidParameters(t *testing.T) {
	decorator := &WorkdirDecorator{}

	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'test'"),
	}

	// Test missing path parameter
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{}, content)

	errors := decoratortesting.Assert(result).
		InterpreterFails("requires at least 1 parameter").
		GeneratorFails("requires at least 1 parameter").
		PlanFails("requires at least 1 parameter").
		Validate()

	if len(errors) > 0 {
		t.Errorf("WorkdirDecorator missing path test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestWorkdirDecorator_EmptyPath(t *testing.T) {
	decorator := &WorkdirDecorator{}

	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'test'"),
	}

	// Test empty path - the decorator accepts empty paths but fails at runtime
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "path", Value: &ast.StringLiteral{Value: ""}},
		}, content)

	// Empty path now fails during parameter validation (which is better!)
	errors := decoratortesting.Assert(result).
		GeneratorFails("path").
		PlanFails("path").
		Validate()

	if len(errors) > 0 {
		t.Errorf("WorkdirDecorator empty path test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestWorkdirDecorator_EmptyContent(t *testing.T) {
	decorator := &WorkdirDecorator{}

	// Test with no commands
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "path", Value: &ast.StringLiteral{Value: "/tmp"}},
		}, []ast.CommandContent{})

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		GeneratorSucceeds().
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("WorkdirDecorator empty content test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestWorkdirDecorator_DeepPath(t *testing.T) {
	decorator := &WorkdirDecorator{}

	// Create a deep path for testing
	testDir := filepath.Join(os.TempDir(), "devcmd_test", "deep", "nested", "path")
	defer func() {
		cleanupDir := filepath.Join(os.TempDir(), "devcmd_test")
		if err := os.RemoveAll(cleanupDir); err != nil {
			t.Logf("Warning: failed to clean up test dir %s: %v", cleanupDir, err)
		}
	}() // Clean up root

	content := []ast.CommandContent{
		decoratortesting.Shell("pwd"),
		decoratortesting.Shell("echo 'in deep path'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "path", Value: &ast.StringLiteral{Value: testDir}},
			{Name: "createIfNotExists", Value: &ast.BooleanLiteral{Value: true}},
		}, content)

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		GeneratorSucceeds().
		GeneratorCodeContains("os.MkdirAll").
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("WorkdirDecorator deep path test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestWorkdirDecorator_SpecialCharactersInPath(t *testing.T) {
	decorator := &WorkdirDecorator{}

	// Test path with spaces and special characters (but valid for filesystem)
	testDir := filepath.Join(os.TempDir(), "test dir with spaces")
	defer func() {
		if err := os.RemoveAll(testDir); err != nil {
			t.Logf("Warning: failed to remove test directory: %v", err)
		}
	}()

	content := []ast.CommandContent{
		decoratortesting.Shell("pwd"),
		decoratortesting.Shell("echo 'special chars'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "path", Value: &ast.StringLiteral{Value: testDir}},
			{Name: "createIfNotExists", Value: &ast.BooleanLiteral{Value: true}},
		}, content)

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		GeneratorSucceeds().
		GeneratorCodeContains("test dir with spaces").
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("WorkdirDecorator special characters test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestWorkdirDecorator_FileInsteadOfDirectory(t *testing.T) {
	decorator := &WorkdirDecorator{}

	// Create a temporary file instead of directory
	tempFile, err := os.CreateTemp("", "devcmd_test_file")
	if err != nil {
		t.Skip("Could not create temp file for test")
	}
	defer func() {
		if err := os.Remove(tempFile.Name()); err != nil {
			t.Logf("Warning: failed to remove temp file: %v", err)
		}
	}()
	if err := tempFile.Close(); err != nil {
		t.Logf("Warning: failed to close temp file: %v", err)
	}

	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'should not run'"),
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "path", Value: &ast.StringLiteral{Value: tempFile.Name()}},
		}, content)

	// Note: Interpreter will fail because it's a file not directory, but generator and plan should work
	errors := decoratortesting.Assert(result).
		GeneratorSucceeds().
		GeneratorProducesValidGo().
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("WorkdirDecorator file instead of directory test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestWorkdirDecorator_PerformanceCharacteristics(t *testing.T) {
	decorator := &WorkdirDecorator{}

	// Test that the decorator itself doesn't add significant overhead
	content := []ast.CommandContent{
		decoratortesting.Shell("echo 'performance test'"),
	}

	start := time.Now()
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "path", Value: &ast.StringLiteral{Value: "/tmp"}},
		}, content)
	generatorDuration := time.Since(start)

	errors := decoratortesting.Assert(result).
		GeneratorSucceeds().
		CompletesWithin("100ms"). // Should be very fast for generation
		Validate()

	// Additional check that generation is fast
	if generatorDuration > 100*time.Millisecond {
		errors = append(errors, "Workdir decorator generation is too slow")
	}

	if len(errors) > 0 {
		t.Errorf("WorkdirDecorator performance test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestWorkdirDecorator_MultipleCommandsWithDirectoryChanges(t *testing.T) {
	decorator := &WorkdirDecorator{}

	// Test that all commands run in the specified directory
	content := []ast.CommandContent{
		decoratortesting.Shell("pwd"),
		decoratortesting.Shell("cd / && pwd"), // This should still be in workdir context
		decoratortesting.Shell("pwd"),         // Should show workdir again
	}

	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestBlockDecorator([]ast.NamedParameter{
			{Name: "path", Value: &ast.StringLiteral{Value: "/tmp"}},
		}, content)

	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		GeneratorSucceeds().
		GeneratorProducesValidGo().
		GeneratorExecutesCorrectly().
		PlanSucceeds().
		Validate()

	if len(errors) > 0 {
		t.Errorf("WorkdirDecorator multiple commands with directory changes test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}
