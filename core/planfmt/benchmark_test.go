package planfmt_test

import (
	"bytes"
	"testing"

	"github.com/aledsdavies/opal/core/planfmt"
)

// Benchmark helpers to generate plans of various sizes
func generatePlan(stepCount int) *planfmt.Plan {
	plan := &planfmt.Plan{
		Target: "benchmark",
		Header: planfmt.PlanHeader{
			CreatedAt: 1234567890,
			PlanKind:  1,
		},
	}

	if stepCount == 0 {
		return plan
	}

	// Create a flat list of steps (simpler for benchmarking)
	var nextID uint64 = 1
	plan.Steps = make([]planfmt.Step, stepCount)
	for i := 0; i < stepCount; i++ {
		plan.Steps[i] = planfmt.Step{
			ID: nextID,
			Tree: &planfmt.CommandNode{
				Decorator: "@shell",
				Args: []planfmt.Arg{
					{Key: "cmd", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo test"}},
					{Key: "timeout", Val: planfmt.Value{Kind: planfmt.ValueInt, Int: 30}},
				},
			},
		}
		nextID++
	}

	return plan
}

// BenchmarkWrite measures serialization performance
func BenchmarkWrite(b *testing.B) {
	sizes := []struct {
		name  string
		steps int
	}{
		{"Empty", 0},
		{"Small_10", 10},
		{"Medium_100", 100},
		{"Large_1000", 1000},
		{"Huge_10000", 10000},
	}

	for _, size := range sizes {
		b.Run(size.name, func(b *testing.B) {
			plan := generatePlan(size.steps)
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				var buf bytes.Buffer
				_, err := planfmt.Write(&buf, plan)
				if err != nil {
					b.Fatalf("Write failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkRead measures deserialization performance
func BenchmarkRead(b *testing.B) {
	sizes := []struct {
		name  string
		steps int
	}{
		{"Empty", 0},
		{"Small_10", 10},
		{"Medium_100", 100},
		{"Large_1000", 1000},
		{"Huge_10000", 10000},
	}

	for _, size := range sizes {
		b.Run(size.name, func(b *testing.B) {
			plan := generatePlan(size.steps)
			var buf bytes.Buffer
			_, err := planfmt.Write(&buf, plan)
			if err != nil {
				b.Fatalf("Write failed: %v", err)
			}
			data := buf.Bytes()

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, _, err := planfmt.Read(bytes.NewReader(data))
				if err != nil {
					b.Fatalf("Read failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkRoundTrip measures full write+read cycle
func BenchmarkRoundTrip(b *testing.B) {
	sizes := []struct {
		name  string
		steps int
	}{
		{"Empty", 0},
		{"Small_10", 10},
		{"Medium_100", 100},
		{"Large_1000", 1000},
		{"Huge_10000", 10000},
	}

	for _, size := range sizes {
		b.Run(size.name, func(b *testing.B) {
			plan := generatePlan(size.steps)
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				var buf bytes.Buffer
				_, err := planfmt.Write(&buf, plan)
				if err != nil {
					b.Fatalf("Write failed: %v", err)
				}

				_, _, err = planfmt.Read(bytes.NewReader(buf.Bytes()))
				if err != nil {
					b.Fatalf("Read failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkWriteThroughput measures bytes/sec throughput
func BenchmarkWriteThroughput(b *testing.B) {
	plan := generatePlan(1000) // 1000 steps

	// First, measure size
	var buf bytes.Buffer
	_, err := planfmt.Write(&buf, plan)
	if err != nil {
		b.Fatalf("Write failed: %v", err)
	}
	planSize := int64(buf.Len())

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(planSize)

	for i := 0; i < b.N; i++ {
		buf.Reset()
		_, err := planfmt.Write(&buf, plan)
		if err != nil {
			b.Fatalf("Write failed: %v", err)
		}
	}
}

// BenchmarkReadThroughput measures bytes/sec throughput
func BenchmarkReadThroughput(b *testing.B) {
	plan := generatePlan(1000) // 1000 steps
	var buf bytes.Buffer
	_, err := planfmt.Write(&buf, plan)
	if err != nil {
		b.Fatalf("Write failed: %v", err)
	}
	data := buf.Bytes()
	planSize := int64(len(data))

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(planSize)

	for i := 0; i < b.N; i++ {
		_, _, err := planfmt.Read(bytes.NewReader(data))
		if err != nil {
			b.Fatalf("Read failed: %v", err)
		}
	}
}

// BenchmarkCanonicalizeOverhead measures cost of arg sorting
func BenchmarkCanonicalizeOverhead(b *testing.B) {
	// Create plan with unsorted args
	plan := &planfmt.Plan{
		Target: "test",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Tree: &planfmt.CommandNode{
					Decorator: "@test",
					Args: []planfmt.Arg{
						{Key: "z_last", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "value"}},
						{Key: "m_middle", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "value"}},
						{Key: "a_first", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "value"}},
					},
				},
			},
		},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Note: canonicalize is now internal, called by Write
		// This benchmark now measures Write overhead
		var buf bytes.Buffer
		_, _ = planfmt.Write(&buf, plan)
	}
}

// BenchmarkMemoryFootprint measures memory usage for large plans
func BenchmarkMemoryFootprint(b *testing.B) {
	sizes := []struct {
		name  string
		steps int
	}{
		{"N_100", 100},
		{"N_1000", 1000},
		{"N_10000", 10000},
	}

	for _, size := range sizes {
		b.Run(size.name, func(b *testing.B) {
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				plan := generatePlan(size.steps)
				var buf bytes.Buffer
				_, _ = planfmt.Write(&buf, plan)
				_, _, _ = planfmt.Read(bytes.NewReader(buf.Bytes()))
			}
		})
	}
}

// BenchmarkWideTree measures performance with many nested steps (wide fanout)
// Tests for O(N²) behavior in block handling
func BenchmarkWideTree(b *testing.B) {
	// Create plan with 1 step containing 1000 nested block steps
	plan := &planfmt.Plan{
		Target: "wide",
		Header: planfmt.PlanHeader{
			CreatedAt: 1234567890,
			PlanKind:  1,
		},
		Steps: []planfmt.Step{
			{
				ID: 1,
				Tree: &planfmt.CommandNode{
					Decorator: "@parallel",
					Block:     make([]planfmt.Step, 1000),
				},
			},
		},
	}

	parallelNode := plan.Steps[0].Tree.(*planfmt.CommandNode)
	for i := 0; i < 1000; i++ {
		parallelNode.Block[i] = planfmt.Step{
			ID: uint64(i + 2),
			Tree: &planfmt.CommandNode{
				Decorator: "@shell",
				Args: []planfmt.Arg{
					{Key: "cmd", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo test"}},
				},
			},
		}
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_, err := planfmt.Write(&buf, plan)
		if err != nil {
			b.Fatalf("Write failed: %v", err)
		}
	}
}

// BenchmarkArgsHeavy measures performance with many args per step
// Tests for O(N²) behavior in arg sorting/copying
func BenchmarkArgsHeavy(b *testing.B) {
	// Create plan with 100 steps, each with 64 args
	var nextID uint64 = 1
	plan := &planfmt.Plan{
		Target: "args_heavy",
		Header: planfmt.PlanHeader{
			CreatedAt: 1234567890,
			PlanKind:  1,
		},
	}

	// Build 100 steps with 64 args each
	plan.Steps = make([]planfmt.Step, 100)
	for i := 0; i < 100; i++ {
		args := make([]planfmt.Arg, 64)
		for j := 0; j < 64; j++ {
			// Use different keys to force sorting
			args[j] = planfmt.Arg{
				Key: string(rune('a'+(j%26))) + string(rune('0'+(j/26))),
				Val: planfmt.Value{Kind: planfmt.ValueString, Str: "value"},
			}
		}
		plan.Steps[i] = planfmt.Step{
			ID: nextID,
			Tree: &planfmt.CommandNode{
				Decorator: "@test",
				Args:      args,
			},
		}
		nextID++
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_, err := planfmt.Write(&buf, plan)
		if err != nil {
			b.Fatalf("Write failed: %v", err)
		}
	}
}
