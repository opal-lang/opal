package planfmt_test

import (
	"bytes"
	"testing"

	"github.com/opal-lang/opal/core/planfmt"
)

// FuzzReadPlan tests that the reader never panics on arbitrary input
func FuzzReadPlan(f *testing.F) {
	// Seed corpus with valid plans
	addSeedCorpus(f)

	// Seed with truncated files at various offsets
	validPlan := &planfmt.Plan{
		Target: "test",
		Header: planfmt.PlanHeader{
			CreatedAt: 1234567890,
			PlanKind:  1,
		},
	}
	var buf bytes.Buffer
	_, _ = planfmt.Write(&buf, validPlan)
	validBytes := buf.Bytes()

	// Add truncated versions
	for i := 0; i < len(validBytes); i += 5 {
		f.Add(validBytes[:i])
	}

	// Add corrupted magic
	corruptedMagic := make([]byte, len(validBytes))
	copy(corruptedMagic, validBytes)
	corruptedMagic[0] = 'X'
	f.Add(corruptedMagic)

	// Add corrupted lengths
	corruptedLen := make([]byte, len(validBytes))
	copy(corruptedLen, validBytes)
	if len(corruptedLen) >= 12 {
		corruptedLen[8] = 0xFF // Corrupt HEADER_LEN
		corruptedLen[9] = 0xFF
		corruptedLen[10] = 0xFF
		corruptedLen[11] = 0xFF
	}
	f.Add(corruptedLen)

	f.Fuzz(func(t *testing.T, data []byte) {
		// Must not panic
		_, _, _ = planfmt.Read(bytes.NewReader(data))
	})
}

// addSeedCorpus adds valid plans to the fuzz corpus
func addSeedCorpus(f *testing.F) {
	f.Helper()
	seeds := []struct {
		name string
		plan *planfmt.Plan
	}{
		{
			name: "empty",
			plan: &planfmt.Plan{Target: ""},
		},
		{
			name: "minimal",
			plan: &planfmt.Plan{
				Target: "test",
				Header: planfmt.PlanHeader{CreatedAt: 1, PlanKind: 1},
			},
		},
		{
			name: "single_step",
			plan: &planfmt.Plan{
				Target: "build",
				Steps: []planfmt.Step{
					{
						ID: 1,
						Tree: &planfmt.CommandNode{
							Decorator: "@shell",
							Args: []planfmt.Arg{
								{Key: "cmd", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo test"}},
							},
						},
					},
				},
			},
		},
		{
			name: "nested_steps",
			plan: &planfmt.Plan{
				Target: "parallel",
				Steps: []planfmt.Step{
					{
						ID: 1,
						Tree: &planfmt.CommandNode{
							Decorator: "@parallel",
							Block: []planfmt.Step{
								{
									ID:   2,
									Tree: &planfmt.CommandNode{Decorator: "@task1"},
								},
								{
									ID:   3,
									Tree: &planfmt.CommandNode{Decorator: "@task2"},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "all_value_types",
			plan: &planfmt.Plan{
				Target: "types",
				Steps: []planfmt.Step{
					{
						ID: 1,
						Tree: &planfmt.CommandNode{
							Decorator: "@test",
							Args: []planfmt.Arg{
								{Key: "a_bool", Val: planfmt.Value{Kind: planfmt.ValueBool, Bool: true}},
								{Key: "b_int", Val: planfmt.Value{Kind: planfmt.ValueInt, Int: -42}},
								{Key: "c_str", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "hello"}},
								{Key: "d_ref", Val: planfmt.Value{Kind: planfmt.ValuePlaceholder, Ref: 0}},
							},
						},
					},
				},
			},
		},
	}

	for _, seed := range seeds {
		var buf bytes.Buffer
		_, err := planfmt.Write(&buf, seed.plan)
		if err != nil {
			continue // Skip invalid seeds
		}
		f.Add(buf.Bytes())
	}
}
