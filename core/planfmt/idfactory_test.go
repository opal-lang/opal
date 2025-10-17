package planfmt

import (
	"testing"

	"github.com/aledsdavies/opal/core/sdk/secret"
	"github.com/stretchr/testify/assert"
)

func TestNewPlanIDFactory(t *testing.T) {
	// Create a simple plan
	plan := &Plan{
		Target: "test",
		Steps: []Step{
			{
				ID: 1,
				Commands: []Command{
					{
						Decorator: "@shell",
						Args: []Arg{
							{Key: "command", Val: Value{Kind: ValueString, Str: "echo hello"}},
						},
					},
				},
			},
		},
	}

	factory1, err := NewPlanIDFactory(plan)
	assert.NoError(t, err)
	assert.NotNil(t, factory1)

	factory2, err := NewPlanIDFactory(plan)
	assert.NoError(t, err)
	assert.NotNil(t, factory2)

	// Same plan should produce same DisplayIDs
	ctx := secret.IDContext{
		PlanHash:  []byte("test-hash"),
		StepPath:  "test.step[0]",
		Decorator: "@test",
		KeyName:   "TEST",
		Kind:      "s",
	}

	value := []byte("test-value")

	id1 := factory1.Make(ctx, value)
	id2 := factory2.Make(ctx, value)

	assert.Equal(t, id1, id2, "Same plan should produce deterministic DisplayIDs")
}

func TestNewPlanIDFactoryDifferentPlans(t *testing.T) {
	// Create two different plans
	plan1 := &Plan{
		Target: "test1",
		Steps: []Step{
			{
				ID: 1,
				Commands: []Command{
					{
						Decorator: "@shell",
						Args: []Arg{
							{Key: "command", Val: Value{Kind: ValueString, Str: "echo hello"}},
						},
					},
				},
			},
		},
	}

	plan2 := &Plan{
		Target: "test2",
		Steps: []Step{
			{
				ID: 1,
				Commands: []Command{
					{
						Decorator: "@shell",
						Args: []Arg{
							{Key: "command", Val: Value{Kind: ValueString, Str: "echo goodbye"}},
						},
					},
				},
			},
		},
	}

	factory1, err := NewPlanIDFactory(plan1)
	assert.NoError(t, err)

	factory2, err := NewPlanIDFactory(plan2)
	assert.NoError(t, err)

	// Different plans should produce different DisplayIDs for same value
	ctx := secret.IDContext{
		PlanHash:  []byte("test-hash"),
		StepPath:  "test.step[0]",
		Decorator: "@test",
		KeyName:   "TEST",
		Kind:      "s",
	}

	value := []byte("test-value")

	id1 := factory1.Make(ctx, value)
	id2 := factory2.Make(ctx, value)

	assert.NotEqual(t, id1, id2, "Different plans should produce different DisplayIDs")
}

func TestNewRunIDFactory(t *testing.T) {
	factory1, err := NewRunIDFactory()
	assert.NoError(t, err)
	assert.NotNil(t, factory1)

	factory2, err := NewRunIDFactory()
	assert.NoError(t, err)
	assert.NotNil(t, factory2)

	// Different runs should produce different DisplayIDs
	ctx := secret.IDContext{
		PlanHash:  []byte("test-hash"),
		StepPath:  "test.step[0]",
		Decorator: "@test",
		KeyName:   "TEST",
		Kind:      "s",
	}

	value := []byte("test-value")

	id1 := factory1.Make(ctx, value)
	id2 := factory2.Make(ctx, value)

	assert.NotEqual(t, id1, id2, "Different run keys should produce different DisplayIDs")
}
