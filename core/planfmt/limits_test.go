package planfmt_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/opal-lang/opal/core/planfmt"
)

// TestDepthLimit verifies that deeply nested plans are rejected
func TestDepthLimit(t *testing.T) {
	// Create a very deep plan (1001 levels deep - exceeds 1000 limit)
	// Build deep chain of nested blocks
	plan := &planfmt.Plan{
		Target: "deep",
		Steps:  []planfmt.Step{{ID: 1}},
	}

	// Build deep nesting through blocks
	current := &plan.Steps[0]
	for i := 0; i < 1001; i++ {
		current.Tree = &planfmt.CommandNode{
			Decorator: "@shell",
			Block: []planfmt.Step{
				{
					ID:   uint64(i + 2),
					Tree: &planfmt.CommandNode{Decorator: "@noop"}, // Placeholder, will be overwritten in next iteration
				},
			},
		}
		cmdNode := current.Tree.(*planfmt.CommandNode)
		current = &cmdNode.Block[0]
	}

	// Write should succeed (no depth check on write)
	var buf bytes.Buffer
	_, err := planfmt.Write(&buf, plan)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Read should fail with depth limit error
	_, _, err = planfmt.Read(bytes.NewReader(buf.Bytes()))
	if err == nil {
		t.Fatal("Expected depth limit error, got nil")
	}

	if !strings.Contains(err.Error(), "max recursion depth") {
		t.Errorf("Expected 'max recursion depth' error, got: %v", err)
	}
}

// TestUnknownFlags verifies that unknown flags are rejected
func TestUnknownFlags(t *testing.T) {
	plan := &planfmt.Plan{Target: "test"}

	var buf bytes.Buffer
	_, err := planfmt.Write(&buf, plan)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	data := buf.Bytes()

	// Corrupt flags field (byte 6-7) to set unknown bit
	data[6] = 0x04 // Set bit 2 (unknown flag)

	// Read should reject unknown flags
	_, _, err = planfmt.Read(bytes.NewReader(data))
	if err == nil {
		t.Fatal("Expected unknown flags error, got nil")
	}

	if !strings.Contains(err.Error(), "unsupported flags") {
		t.Errorf("Expected 'unsupported flags' error, got: %v", err)
	}
}

// TestCompressedFlagRejected verifies that compressed flag is rejected (not implemented yet)
func TestCompressedFlagRejected(t *testing.T) {
	plan := &planfmt.Plan{Target: "test"}

	var buf bytes.Buffer
	_, err := planfmt.Write(&buf, plan)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	data := buf.Bytes()

	// Set FlagCompressed (bit 0)
	data[6] = 0x01

	// Read should reject compressed plans
	_, _, err = planfmt.Read(bytes.NewReader(data))
	if err == nil {
		t.Fatal("Expected compressed error, got nil")
	}

	if !strings.Contains(err.Error(), "compressed plans not yet supported") {
		t.Errorf("Expected 'compressed plans not yet supported' error, got: %v", err)
	}
}

// TestSignedFlagRejected verifies that signed flag is rejected (not implemented yet)
func TestSignedFlagRejected(t *testing.T) {
	plan := &planfmt.Plan{Target: "test"}

	var buf bytes.Buffer
	_, err := planfmt.Write(&buf, plan)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	data := buf.Bytes()

	// Set FlagSigned (bit 1)
	data[6] = 0x02

	// Read should reject signed plans
	_, _, err = planfmt.Read(bytes.NewReader(data))
	if err == nil {
		t.Fatal("Expected signed error, got nil")
	}

	if !strings.Contains(err.Error(), "signed plans not yet supported") {
		t.Errorf("Expected 'signed plans not yet supported' error, got: %v", err)
	}
}

// TestMaxBlockStepsLimit verifies reasonable limits on block step count
func TestMaxBlockStepsLimit(t *testing.T) {
	// uint16 max is 65535 - create a plan with that many block steps
	// This tests that we can handle the format limit
	plan := &planfmt.Plan{
		Target: "wide",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Tree: &planfmt.CommandNode{
					Decorator: "@parallel",
					Block:     make([]planfmt.Step, 65535),
				},
			},
		},
	}

	cmdNode := plan.Steps[0].Tree.(*planfmt.CommandNode)
	for i := 0; i < 65535; i++ {
		cmdNode.Block[i] = planfmt.Step{
			ID:   uint64(i + 2),
			Tree: &planfmt.CommandNode{Decorator: "@task"},
		}
	}

	// Should be able to write and read (though it's huge)
	var buf bytes.Buffer
	_, err := planfmt.Write(&buf, plan)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Read should succeed (within body size limit)
	_, _, err = planfmt.Read(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
}

// TestValueDepthLimit verifies that deeply nested arrays and maps are rejected
func TestValueDepthLimit(t *testing.T) {
	// Create a plan with deeply nested array (1001 levels - exceeds 1000 limit)
	// Build nested array: [[[[...1001 levels...]]]]
	var deepValue planfmt.Value
	deepValue.Kind = planfmt.ValueString
	deepValue.Str = "deep"

	for i := 0; i < 1001; i++ {
		deepValue = planfmt.Value{
			Kind:  planfmt.ValueArray,
			Array: []planfmt.Value{deepValue},
		}
	}

	plan := &planfmt.Plan{
		Target: "deep_values",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Tree: &planfmt.CommandNode{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "data", Val: deepValue},
					},
				},
			},
		},
	}

	// Write should succeed (no depth check on write)
	var buf bytes.Buffer
	_, err := planfmt.Write(&buf, plan)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Read should fail with depth limit error
	_, _, err = planfmt.Read(bytes.NewReader(buf.Bytes()))
	if err == nil {
		t.Fatal("Expected depth limit error, got nil")
	}

	if !strings.Contains(err.Error(), "max value recursion depth") {
		t.Errorf("Expected 'max value recursion depth' error, got: %v", err)
	}
}

// TestValueDepthLimitMaps verifies that deeply nested maps are rejected
func TestValueDepthLimitMaps(t *testing.T) {
	// Create a plan with deeply nested map (1001 levels - exceeds 1000 limit)
	// Build nested map: {"a": {"a": {"a": ...1001 levels...}}}
	var deepValue planfmt.Value
	deepValue.Kind = planfmt.ValueString
	deepValue.Str = "deep"

	for i := 0; i < 1001; i++ {
		deepValue = planfmt.Value{
			Kind: planfmt.ValueMap,
			Map:  map[string]planfmt.Value{"a": deepValue},
		}
	}

	plan := &planfmt.Plan{
		Target: "deep_values",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Tree: &planfmt.CommandNode{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "data", Val: deepValue},
					},
				},
			},
		},
	}

	// Write should succeed (no depth check on write)
	var buf bytes.Buffer
	_, err := planfmt.Write(&buf, plan)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Read should fail with depth limit error
	_, _, err = planfmt.Read(bytes.NewReader(buf.Bytes()))
	if err == nil {
		t.Fatal("Expected depth limit error, got nil")
	}

	if !strings.Contains(err.Error(), "max value recursion depth") {
		t.Errorf("Expected 'max value recursion depth' error, got: %v", err)
	}
}
