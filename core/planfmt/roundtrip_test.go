package planfmt_test

import (
	"bytes"
	"testing"

	"github.com/aledsdavies/opal/core/planfmt"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// TestRoundTrip verifies that plan → write → read → write produces identical bytes
func TestRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		plan *planfmt.Plan
	}{
		{
			name: "empty plan",
			plan: &planfmt.Plan{
				Target: "",
			},
		},
		{
			name: "plan with header",
			plan: &planfmt.Plan{
				Target: "deploy",
				Header: planfmt.PlanHeader{
					SchemaID:  [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
					CreatedAt: 1234567890,
					Compiler:  [16]byte{16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1},
					PlanKind:  1,
				},
			},
		},
		{
			name: "plan with single step",
			plan: &planfmt.Plan{
				Target: "build",
				Header: planfmt.PlanHeader{
					CreatedAt: 9876543210,
					PlanKind:  1,
				},
				Root: &planfmt.Step{
					ID:   1,
					Kind: planfmt.KindDecorator,
					Op:   "shell",
					Args: []planfmt.Arg{
						{Key: "cmd", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo hello"}},
					},
				},
			},
		},
		{
			name: "plan with nested steps",
			plan: &planfmt.Plan{
				Target: "test",
				Root: &planfmt.Step{
					ID:   1,
					Kind: planfmt.KindDecorator,
					Op:   "parallel",
					Children: []*planfmt.Step{
						{
							ID:   2,
							Kind: planfmt.KindDecorator,
							Op:   "shell",
							Args: []planfmt.Arg{
								{Key: "cmd", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "task1"}},
							},
						},
						{
							ID:   3,
							Kind: planfmt.KindDecorator,
							Op:   "shell",
							Args: []planfmt.Arg{
								{Key: "cmd", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "task2"}},
							},
						},
					},
				},
			},
		},
		{
			name: "plan with all value types",
			plan: &planfmt.Plan{
				Target: "complex",
				Root: &planfmt.Step{
					ID:   1,
					Kind: planfmt.KindDecorator,
					Op:   "test",
					Args: []planfmt.Arg{
						{Key: "bool_val", Val: planfmt.Value{Kind: planfmt.ValueBool, Bool: true}},
						{Key: "int_val", Val: planfmt.Value{Kind: planfmt.ValueInt, Int: 42}},
						{Key: "str_val", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "hello"}},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First write
			var buf1 bytes.Buffer
			hash1, err := planfmt.Write(&buf1, tt.plan)
			if err != nil {
				t.Fatalf("First write failed: %v", err)
			}

			// Read back
			plan2, hash2, err := planfmt.Read(bytes.NewReader(buf1.Bytes()))
			if err != nil {
				t.Fatalf("Read failed: %v", err)
			}

			// Second write
			var buf2 bytes.Buffer
			hash3, err := planfmt.Write(&buf2, plan2)
			if err != nil {
				t.Fatalf("Second write failed: %v", err)
			}

			// Verify bytes are identical (idempotent writer)
			bytes1 := buf1.Bytes()
			bytes2 := buf2.Bytes()

			if !bytes.Equal(bytes1, bytes2) {
				t.Errorf("Round-trip not idempotent:\n"+
					"  first write:  %d bytes\n"+
					"  second write: %d bytes\n"+
					"  difference: %d bytes",
					len(bytes1), len(bytes2), len(bytes2)-len(bytes1))

				// Show detailed diff for debugging
				if diff := cmp.Diff(bytes1, bytes2); diff != "" {
					t.Errorf("Byte diff (-first +second):\n%s", diff)
				}
			}

			// Verify hashes match (when implemented)
			_ = hash1
			_ = hash2
			_ = hash3
		})
	}
}

// TestRoundTripPreservesSemantics verifies that read plan is semantically identical
func TestRoundTripPreservesSemantics(t *testing.T) {
	original := &planfmt.Plan{
		Target: "deploy",
		Header: planfmt.PlanHeader{
			SchemaID:  [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
			CreatedAt: 1234567890,
			Compiler:  [16]byte{16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1},
			PlanKind:  1,
		},
		Root: &planfmt.Step{
			ID:   1,
			Kind: planfmt.KindDecorator,
			Op:   "parallel",
			Args: []planfmt.Arg{
				{Key: "max_concurrent", Val: planfmt.Value{Kind: planfmt.ValueInt, Int: 4}},
			},
			Children: []*planfmt.Step{
				{
					ID:   2,
					Kind: planfmt.KindDecorator,
					Op:   "shell",
					Args: []planfmt.Arg{
						{Key: "cmd", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo task1"}},
						{Key: "timeout", Val: planfmt.Value{Kind: planfmt.ValueInt, Int: 30}},
					},
				},
				{
					ID:   3,
					Kind: planfmt.KindDecorator,
					Op:   "shell",
					Args: []planfmt.Arg{
						{Key: "cmd", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo task2"}},
					},
				},
			},
		},
	}

	// Write
	var buf bytes.Buffer
	_, err := planfmt.Write(&buf, original)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Read
	decoded, _, err := planfmt.Read(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	// Compare semantics (use cmp for deep equality, ignore unexported fields)
	opts := cmpopts.IgnoreUnexported(planfmt.PlanHeader{})
	if diff := cmp.Diff(original, decoded, opts); diff != "" {
		t.Errorf("Semantic mismatch after round-trip (-original +decoded):\n%s", diff)
	}
}
