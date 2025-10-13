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

	// Create a tree of steps with monotonic IDs
	var nextID uint64 = 1
	plan.Root = generateStepTree(&nextID, stepCount, 0)
	return plan
}

func generateStepTree(nextID *uint64, remaining int, depth int) *planfmt.Step {
	if remaining <= 0 {
		return nil
	}

	// Allocate unique ID
	id := *nextID
	*nextID++

	step := &planfmt.Step{
		ID:   id,
		Kind: planfmt.KindDecorator,
		Op:   "shell",
		Args: []planfmt.Arg{
			{Key: "cmd", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo test"}},
			{Key: "timeout", Val: planfmt.Value{Kind: planfmt.ValueInt, Int: 30}},
		},
	}

	remaining--

	// Create children (binary tree for balanced structure)
	if remaining > 0 && depth < 10 {
		childCount := 2
		if remaining == 1 {
			childCount = 1
		}

		step.Children = make([]*planfmt.Step, 0, childCount)
		for i := 0; i < childCount && remaining > 0; i++ {
			childRemaining := remaining / childCount
			if i == 0 {
				childRemaining += remaining % childCount
			}
			child := generateStepTree(nextID, childRemaining, depth+1)
			if child != nil {
				step.Children = append(step.Children, child)
				remaining -= childRemaining
			}
		}
	}

	return step
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
		Root: &planfmt.Step{
			ID:   1,
			Kind: planfmt.KindDecorator,
			Op:   "test",
			Args: []planfmt.Arg{
				{Key: "z_last", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "value"}},
				{Key: "m_middle", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "value"}},
				{Key: "a_first", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "value"}},
			},
		},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		plan.Canonicalize()
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

// BenchmarkWideTree measures performance with many children (wide fanout)
// Tests for O(N²) behavior in child handling
func BenchmarkWideTree(b *testing.B) {
	// Create plan with 1 root + 1000 direct children (wide, not deep)
	plan := &planfmt.Plan{
		Target: "wide",
		Header: planfmt.PlanHeader{
			CreatedAt: 1234567890,
			PlanKind:  1,
		},
		Root: &planfmt.Step{
			ID:       1,
			Kind:     planfmt.KindDecorator,
			Op:       "parallel",
			Children: make([]*planfmt.Step, 1000),
		},
	}

	for i := 0; i < 1000; i++ {
		plan.Root.Children[i] = &planfmt.Step{
			ID:   uint64(i + 2),
			Kind: planfmt.KindDecorator,
			Op:   "shell",
			Args: []planfmt.Arg{
				{Key: "cmd", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo test"}},
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
	steps := make([]*planfmt.Step, 100)
	for i := 0; i < 100; i++ {
		args := make([]planfmt.Arg, 64)
		for j := 0; j < 64; j++ {
			// Use different keys to force sorting
			args[j] = planfmt.Arg{
				Key: string(rune('a'+(j%26))) + string(rune('0'+(j/26))),
				Val: planfmt.Value{Kind: planfmt.ValueString, Str: "value"},
			}
		}
		steps[i] = &planfmt.Step{
			ID:   nextID,
			Kind: planfmt.KindDecorator,
			Op:   "test",
			Args: args,
		}
		nextID++
	}

	// Link as linear chain
	plan.Root = steps[0]
	for i := 0; i < 99; i++ {
		steps[i].Children = []*planfmt.Step{steps[i+1]}
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
