package planfmt_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/aledsdavies/opal/core/planfmt"
)

// TestDepthLimit verifies that deeply nested plans are rejected
func TestDepthLimit(t *testing.T) {
	// Create a very deep plan (1001 levels deep - exceeds 1000 limit)
	plan := &planfmt.Plan{
		Target: "deep",
	}

	// Build deep chain
	var current *planfmt.Step
	for i := 0; i < 1001; i++ {
		step := &planfmt.Step{
			ID:   uint64(i + 1),
			Kind: planfmt.KindDecorator,
			Op:   "shell",
		}
		if current != nil {
			current.Children = []*planfmt.Step{step}
		} else {
			plan.Root = step
		}
		current = step
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

// TestMaxChildrenLimit verifies reasonable limits on children count
func TestMaxChildrenLimit(t *testing.T) {
	// uint16 max is 65535 - create a plan with that many children
	// This tests that we can handle the format limit
	plan := &planfmt.Plan{
		Target: "wide",
		Root: &planfmt.Step{
			ID:       1,
			Kind:     planfmt.KindDecorator,
			Op:       "parallel",
			Children: make([]*planfmt.Step, 65535),
		},
	}

	for i := 0; i < 65535; i++ {
		plan.Root.Children[i] = &planfmt.Step{
			ID:   uint64(i + 2),
			Kind: planfmt.KindDecorator,
			Op:   "task",
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
