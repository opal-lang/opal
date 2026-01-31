package planner

import (
	"strings"
	"testing"

	"github.com/opal-lang/opal/core/planfmt"
	"github.com/opal-lang/opal/runtime/lexer"
	"github.com/opal-lang/opal/runtime/parser"
)

// parseAndBuildIR is a test helper that parses source and builds IR.
func parseAndBuildIR(t *testing.T, source string) (*ExecutionGraph, []parser.Event, []lexer.Token) {
	t.Helper()

	// Parse source using the high-level API
	tree := parser.Parse([]byte(source))
	if len(tree.Errors) > 0 {
		t.Fatalf("parse errors: %v", tree.Errors)
	}

	// Build IR
	graph, err := BuildIR(tree.Events, tree.Tokens)
	if err != nil {
		t.Fatalf("build IR error: %v", err)
	}

	return graph, tree.Events, tree.Tokens
}

// TestPlanNew_SimpleCommand tests the new Plan function with a simple command.
func TestPlanNew_SimpleCommand(t *testing.T) {
	source := `echo "hello world"`

	graph, events, tokens := parseAndBuildIR(t, source)

	// Verify IR was built correctly
	if graph == nil {
		t.Fatal("expected non-nil graph")
	}
	if len(graph.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(graph.Statements))
	}

	// Create config
	config := Config{
		Target:    "",
		Telemetry: TelemetryOff,
		Debug:     DebugOff,
	}

	// Run new planner
	result, err := PlanNewWithObservability(events, tokens, config)
	if err != nil {
		t.Fatalf("PlanNew failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Plan == nil {
		t.Fatal("expected non-nil plan")
	}

	// Verify plan has one step
	if len(result.Plan.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(result.Plan.Steps))
	}

	// Verify step has CommandNode
	step := result.Plan.Steps[0]
	if step.ID != 1 {
		t.Errorf("expected step ID 1, got %d", step.ID)
	}

	cmdNode, ok := step.Tree.(*planfmt.CommandNode)
	if !ok {
		t.Fatalf("expected CommandNode, got %T", step.Tree)
	}
	if cmdNode.Decorator != "@shell" {
		t.Errorf("expected decorator @shell, got %s", cmdNode.Decorator)
	}

	// Verify command argument
	foundCommand := false
	for _, arg := range cmdNode.Args {
		if arg.Key == "command" {
			foundCommand = true
			if !strings.Contains(arg.Val.Str, "echo") {
				t.Errorf("expected command to contain 'echo', got %s", arg.Val.Str)
			}
		}
	}
	if !foundCommand {
		t.Error("expected command argument")
	}
}

// TestPlanNew_MultipleCommands tests the new Plan function with multiple commands.
func TestPlanNew_MultipleCommands(t *testing.T) {
	source := `echo "first"
echo "second"`

	_, events, tokens := parseAndBuildIR(t, source)

	config := Config{
		Target:    "",
		Telemetry: TelemetryOff,
		Debug:     DebugOff,
	}

	result, err := PlanNewWithObservability(events, tokens, config)
	if err != nil {
		t.Fatalf("PlanNew failed: %v", err)
	}

	// Verify plan has two steps
	if len(result.Plan.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(result.Plan.Steps))
	}

	// Verify step IDs are sequential
	if result.Plan.Steps[0].ID != 1 {
		t.Errorf("expected step 0 ID 1, got %d", result.Plan.Steps[0].ID)
	}
	if result.Plan.Steps[1].ID != 2 {
		t.Errorf("expected step 1 ID 2, got %d", result.Plan.Steps[1].ID)
	}
}

// TestPlanNew_FunctionMode tests the new Plan function in function/command mode.
func TestPlanNew_FunctionMode(t *testing.T) {
	source := `fun hello = echo "Hello, World!"
fun goodbye = echo "Goodbye!"`

	_, events, tokens := parseAndBuildIR(t, source)

	config := Config{
		Target:    "hello",
		Telemetry: TelemetryOff,
		Debug:     DebugOff,
	}

	result, err := PlanNewWithObservability(events, tokens, config)
	if err != nil {
		t.Fatalf("PlanNew failed: %v", err)
	}

	// Verify plan targets the correct function
	if result.Plan.Target != "hello" {
		t.Errorf("expected target 'hello', got %s", result.Plan.Target)
	}

	// Should have one step (the echo from hello function)
	if len(result.Plan.Steps) != 1 {
		t.Fatalf("expected 1 step for hello function, got %d", len(result.Plan.Steps))
	}
}

// TestPlanNew_Observability tests that telemetry and debug events are collected.
func TestPlanNew_Observability(t *testing.T) {
	source := `echo "test"`

	_, events, tokens := parseAndBuildIR(t, source)

	config := Config{
		Target:    "",
		Telemetry: TelemetryTiming,
		Debug:     DebugPaths,
	}

	result, err := PlanNewWithObservability(events, tokens, config)
	if err != nil {
		t.Fatalf("PlanNew failed: %v", err)
	}

	// Verify telemetry is collected
	if result.Telemetry == nil {
		t.Error("expected telemetry to be collected")
	} else {
		if result.Telemetry.EventCount == 0 {
			t.Error("expected non-zero event count")
		}
		if result.Telemetry.StepCount != 1 {
			t.Errorf("expected step count 1, got %d", result.Telemetry.StepCount)
		}
	}

	// Verify debug events are collected
	if len(result.DebugEvents) == 0 {
		t.Error("expected debug events to be collected")
	}

	// Verify plan time is recorded
	if result.PlanTime == 0 {
		t.Error("expected non-zero plan time")
	}
}

// TestPlanNew_PlanSalt tests that the plan has a salt for contract verification.
func TestPlanNew_PlanSalt(t *testing.T) {
	source := `echo "test"`

	_, events, tokens := parseAndBuildIR(t, source)

	config := Config{
		Target: "",
	}

	result, err := PlanNew(events, tokens, config)
	if err != nil {
		t.Fatalf("PlanNew failed: %v", err)
	}

	// Verify plan has salt
	if len(result.PlanSalt) == 0 {
		t.Error("expected plan salt to be set")
	}
}
