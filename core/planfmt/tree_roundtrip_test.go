package planfmt_test

import (
	"bytes"
	"testing"

	"github.com/aledsdavies/opal/core/planfmt"
	"github.com/google/go-cmp/cmp"
)

func TestTreeRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		plan *planfmt.Plan
	}{
		{
			name: "plan with simple command tree",
			plan: &planfmt.Plan{
				Target: "test",
				Steps: []planfmt.Step{
					{
						ID: 1,
						Tree: &planfmt.CommandNode{
							Decorator: "@shell",
							Args: []planfmt.Arg{
								{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo hello"}},
							},
						},
					},
				},
			},
		},
		{
			name: "plan with AND tree",
			plan: &planfmt.Plan{
				Target: "test",
				Steps: []planfmt.Step{
					{
						ID: 1,
						Tree: &planfmt.AndNode{
							Left: &planfmt.CommandNode{
								Decorator: "@shell",
								Args: []planfmt.Arg{
									{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo a"}},
								},
							},
							Right: &planfmt.CommandNode{
								Decorator: "@shell",
								Args: []planfmt.Arg{
									{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo b"}},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "plan with pipeline tree",
			plan: &planfmt.Plan{
				Target: "test",
				Steps: []planfmt.Step{
					{
						ID: 1,
						Tree: &planfmt.PipelineNode{
							Commands: []planfmt.ExecutionNode{
								&planfmt.CommandNode{
									Decorator: "@shell",
									Args: []planfmt.Arg{
										{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo test"}},
									},
								},
								&planfmt.CommandNode{
									Decorator: "@shell",
									Args: []planfmt.Arg{
										{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "grep test"}},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "plan with complex tree (pipe > AND > OR)",
			plan: &planfmt.Plan{
				Target: "test",
				Steps: []planfmt.Step{
					{
						ID: 1,
						Tree: &planfmt.OrNode{
							Left: &planfmt.AndNode{
								Left: &planfmt.PipelineNode{
									Commands: []planfmt.ExecutionNode{
										&planfmt.CommandNode{
											Decorator: "@shell",
											Args: []planfmt.Arg{
												{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo a"}},
											},
										},
										&planfmt.CommandNode{
											Decorator: "@shell",
											Args: []planfmt.Arg{
												{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "grep a"}},
											},
										},
									},
								},
								Right: &planfmt.PipelineNode{
									Commands: []planfmt.ExecutionNode{
										&planfmt.CommandNode{
											Decorator: "@shell",
											Args: []planfmt.Arg{
												{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo b"}},
											},
										},
										&planfmt.CommandNode{
											Decorator: "@shell",
											Args: []planfmt.Arg{
												{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "grep b"}},
											},
										},
									},
								},
							},
							Right: &planfmt.CommandNode{
								Decorator: "@shell",
								Args: []planfmt.Arg{
									{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo fallback"}},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "plan with sequence tree (semicolon operator)",
			plan: &planfmt.Plan{
				Target: "test",
				Steps: []planfmt.Step{
					{
						ID: 1,
						Tree: &planfmt.SequenceNode{
							Nodes: []planfmt.ExecutionNode{
								&planfmt.CommandNode{
									Decorator: "@shell",
									Args: []planfmt.Arg{
										{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo a"}},
									},
								},
								&planfmt.CommandNode{
									Decorator: "@shell",
									Args: []planfmt.Arg{
										{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo b"}},
									},
								},
								&planfmt.CommandNode{
									Decorator: "@shell",
									Args: []planfmt.Arg{
										{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo c"}},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write plan to bytes
			var buf1 bytes.Buffer
			hash1, err := planfmt.Write(&buf1, tt.plan)
			if err != nil {
				t.Fatalf("first write failed: %v", err)
			}

			// Save the bytes before reading (Read consumes the buffer)
			bytes1 := buf1.Bytes()
			bytesReader := bytes.NewReader(bytes1)

			// Read plan back
			plan2, readHash, err := planfmt.Read(bytesReader)
			if err != nil {
				t.Fatalf("read failed: %v", err)
			}

			// Read hash should match write hash
			if hash1 != readHash {
				t.Errorf("hash mismatch (write vs read):\n  write: %x\n  read:  %x", hash1, readHash)
			}

			// Write again to verify determinism
			var buf2 bytes.Buffer
			hash2, err := planfmt.Write(&buf2, plan2)
			if err != nil {
				t.Fatalf("second write failed: %v", err)
			}

			// Hashes must match (deterministic)
			if hash1 != hash2 {
				t.Errorf("hash mismatch:\n  first:  %x\n  second: %x", hash1, hash2)
			}

			// Bytes must match exactly (lossless)
			bytes2 := buf2.Bytes()
			if !bytes.Equal(bytes1, bytes2) {
				t.Errorf("bytes mismatch (not lossless)\n  first:  %d bytes\n  second: %d bytes", len(bytes1), len(bytes2))
			}

			// Structure must match (deep equality)
			if diff := cmp.Diff(tt.plan, plan2); diff != "" {
				t.Errorf("plan mismatch (-want +got):\n%s", diff)
			}

			// Verify Tree is present
			if plan2.Steps[0].Tree == nil {
				t.Errorf("Tree should be present after deserialization")
			}
		})
	}
}

// TestTreeOnlySerialized verifies that only the Tree is serialized.
func TestTreeOnlySerialized(t *testing.T) {
	plan := &planfmt.Plan{
		Target: "test",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Tree: &planfmt.CommandNode{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo hello"}},
					},
				},
			},
		},
	}

	// Write plan
	var buf bytes.Buffer
	_, err := planfmt.Write(&buf, plan)
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}

	// Read plan back
	plan2, _, err := planfmt.Read(&buf)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	// Verify Tree was serialized
	if plan2.Steps[0].Tree == nil {
		t.Fatalf("Tree should be present")
	}

	// Verify Tree has correct data
	cmdNode, ok := plan2.Steps[0].Tree.(*planfmt.CommandNode)
	if !ok {
		t.Fatalf("expected CommandNode, got %T", plan2.Steps[0].Tree)
	}

	if cmdNode.Decorator != "@shell" {
		t.Errorf("expected decorator '@shell', got %q", cmdNode.Decorator)
	}

	if len(cmdNode.Args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(cmdNode.Args))
	}

	if cmdNode.Args[0].Val.Str != "echo hello" {
		t.Errorf("expected command 'echo hello', got %q", cmdNode.Args[0].Val.Str)
	}
}
