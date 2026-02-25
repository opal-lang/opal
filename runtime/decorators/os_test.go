package decorators

import (
	"runtime"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/opal-lang/opal/core/decorator"
)

func TestOSDecoratorsRegistered(t *testing.T) {
	if !decorator.Global().IsRegistered("os.Get") {
		t.Fatal("built-in decorator 'os.Get' should be registered")
	}
	if !decorator.Global().IsRegistered("os.Linux") {
		t.Fatal("built-in decorator 'os.Linux' should be registered")
	}
	if !decorator.Global().IsRegistered("os.macOS") {
		t.Fatal("built-in decorator 'os.macOS' should be registered")
	}
	if !decorator.Global().IsRegistered("os.Windows") {
		t.Fatal("built-in decorator 'os.Windows' should be registered")
	}
}

func TestOSGetDecoratorResolve(t *testing.T) {
	d := &osGetDecorator{}

	result := resolveSingle(t, d, decorator.ValueEvalContext{}, decorator.ValueCall{Path: "os.Get"})
	if result.Error != nil {
		t.Fatalf("result error: %v", result.Error)
	}

	if diff := cmp.Diff(runtime.GOOS, result.Value); diff != "" {
		t.Fatalf("value mismatch (-want +got):\n%s", diff)
	}
}

func TestOSLinuxDecoratorResolve(t *testing.T) {
	d := &osLinuxDecorator{}

	result := resolveSingle(t, d, decorator.ValueEvalContext{}, decorator.ValueCall{Path: "os.Linux"})
	if result.Error != nil {
		t.Fatalf("result error: %v", result.Error)
	}

	expected := "false"
	if runtime.GOOS == "linux" {
		expected = "true"
	}

	if diff := cmp.Diff(expected, result.Value); diff != "" {
		t.Fatalf("value mismatch (-want +got):\n%s", diff)
	}
}

func TestOSMacOSDecoratorResolve(t *testing.T) {
	d := &osMacOSDecorator{}

	result := resolveSingle(t, d, decorator.ValueEvalContext{}, decorator.ValueCall{Path: "os.macOS"})
	if result.Error != nil {
		t.Fatalf("result error: %v", result.Error)
	}

	expected := "false"
	if runtime.GOOS == "darwin" {
		expected = "true"
	}

	if diff := cmp.Diff(expected, result.Value); diff != "" {
		t.Fatalf("value mismatch (-want +got):\n%s", diff)
	}
}

func TestOSWindowsDecoratorResolve(t *testing.T) {
	d := &osWindowsDecorator{}

	result := resolveSingle(t, d, decorator.ValueEvalContext{}, decorator.ValueCall{Path: "os.Windows"})
	if result.Error != nil {
		t.Fatalf("result error: %v", result.Error)
	}

	expected := "false"
	if runtime.GOOS == "windows" {
		expected = "true"
	}

	if diff := cmp.Diff(expected, result.Value); diff != "" {
		t.Fatalf("value mismatch (-want +got):\n%s", diff)
	}
}
