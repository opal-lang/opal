package planner

import (
	"context"
	"strings"
	"time"

	"github.com/opal-lang/opal/core/planfmt"
	"github.com/opal-lang/opal/core/sdk/secret"
	"github.com/opal-lang/opal/runtime/lexer"
	"github.com/opal-lang/opal/runtime/parser"
	"github.com/opal-lang/opal/runtime/vault"
)

// Config configures planner behavior.
type Config struct {
	Target    string           // Command name (e.g. "hello") or "" for script mode.
	Args      []FunctionArg    // Optional target function arguments (positional + named).
	Context   context.Context  // Optional planning context for cancellation/deadlines.
	IDFactory secret.IDFactory // Optional deterministic placeholder factory.
	Vault     *vault.Vault     // Optional shared vault for value storage/scrubbing.
	PlanSalt  []byte           // Optional deterministic salt (32 bytes) for contract verification.
	Telemetry TelemetryLevel   // Telemetry level (production-safe).
	Debug     DebugLevel       // Debug level (development only).
}

// FunctionArg represents one target function argument.
// Empty Name means positional argument.
type FunctionArg struct {
	Name  string
	Value any
}

// TelemetryLevel controls telemetry collection.
type TelemetryLevel int

const (
	TelemetryOff TelemetryLevel = iota
	TelemetryBasic
	TelemetryTiming
)

// DebugLevel controls debug tracing.
type DebugLevel int

const (
	DebugOff DebugLevel = iota
	DebugPaths
	DebugDetailed
)

// PlanResult holds a generated plan and optional observability data.
type PlanResult struct {
	Plan        *planfmt.Plan
	PlanTime    time.Duration
	Telemetry   *PlanTelemetry
	DebugEvents []DebugEvent
}

// PlanTelemetry holds planner metrics.
type PlanTelemetry struct {
	EventCount int
	StepCount  int

	DecoratorResolutions map[string]*DecoratorResolutionMetrics
}

// DecoratorResolutionMetrics tracks per-decorator resolution metrics.
type DecoratorResolutionMetrics struct {
	TotalCalls   int
	BatchCalls   int
	BatchSizes   []int
	TotalTime    time.Duration
	SkippedCalls int
}

// DebugEvent captures planner trace/debug events.
type DebugEvent struct {
	Timestamp time.Time
	Event     string
	EventPos  int
	Context   string
}

// PlanError represents a planning error with contextual hints.
type PlanError struct {
	Message     string
	Context     string
	EventPos    int
	TotalEvents int
	Suggestion  string
	Example     string
}

func (e *PlanError) Error() string {
	var b strings.Builder
	b.WriteString(e.Message)
	if e.Suggestion != "" {
		b.WriteString("\n")
		b.WriteString(e.Suggestion)
	}
	if e.Example != "" {
		b.WriteString("\n")
		b.WriteString(e.Example)
	}
	return b.String()
}

// Plan is the canonical planner entrypoint.
func Plan(events []parser.Event, tokens []lexer.Token, config Config) (*planfmt.Plan, error) {
	return planCanonical(events, tokens, config)
}

// PlanWithObservability is the canonical observability entrypoint.
func PlanWithObservability(events []parser.Event, tokens []lexer.Token, config Config) (*PlanResult, error) {
	return planCanonicalWithObservability(events, tokens, config)
}
