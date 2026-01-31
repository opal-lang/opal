package planner

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/opal-lang/opal/core/decorator"
	"github.com/opal-lang/opal/core/invariant"
	"github.com/opal-lang/opal/core/planfmt"
	"github.com/opal-lang/opal/runtime/lexer"
	"github.com/opal-lang/opal/runtime/parser"
	"github.com/opal-lang/opal/runtime/vault"
)

// PlanNew consumes parser events and generates an execution plan using the new pipeline.
// This is the new implementation that uses BuildIR → Resolver → Emitter.
//
// Pipeline:
//  1. BuildIR: Convert parser events → ExecutionGraph
//  2. Resolve: Wave-based resolution → ResolveResult with pruned tree
//  3. Emit: Generate planfmt.Plan with structure preservation
func PlanNew(events []parser.Event, tokens []lexer.Token, config Config) (*planfmt.Plan, error) {
	result, err := PlanNewWithObservability(events, tokens, config)
	if err != nil {
		return nil, err
	}
	return result.Plan, nil
}

// PlanNewWithObservability returns plan with telemetry and debug events using the new pipeline.
func PlanNewWithObservability(events []parser.Event, tokens []lexer.Token, config Config) (*PlanResult, error) {
	var telemetry *PlanTelemetry
	var debugEvents []DebugEvent

	// Always track planning time
	startTime := time.Now()

	// Initialize telemetry if enabled
	if config.Telemetry >= TelemetryBasic {
		telemetry = &PlanTelemetry{
			DecoratorResolutions: make(map[string]*DecoratorResolutionMetrics),
		}
	}

	// Initialize debug events if enabled
	if config.Debug >= DebugPaths {
		debugEvents = make([]DebugEvent, 0, 100)
	}

	// Record initial debug event
	if config.Debug >= DebugPaths {
		debugEvents = append(debugEvents, DebugEvent{
			Timestamp: time.Now(),
			Event:     "enter_plan",
			Context:   fmt.Sprintf("target=%s", config.Target),
		})
	}

	// Use provided Vault or create new one with random planKey
	var vlt *vault.Vault
	if config.Vault != nil {
		// Use shared vault from caller (e.g., CLI for scrubbing integration)
		vlt = config.Vault
	} else {
		// Create new vault with random planKey for HMAC-based SiteIDs
		planKey := make([]byte, 32)
		_, err := rand.Read(planKey)
		invariant.ExpectNoError(err, "failed to generate Vault planKey")
		vlt = vault.NewWithPlanKey(planKey)
	}

	// Step 1: BuildIR - Convert parser events to ExecutionGraph
	if config.Debug >= DebugPaths {
		debugEvents = append(debugEvents, DebugEvent{
			Timestamp: time.Now(),
			Event:     "build_ir_start",
			Context:   fmt.Sprintf("events=%d", len(events)),
		})
	}

	graph, err := BuildIR(events, tokens)
	if err != nil {
		return nil, &PlanError{
			Message:     fmt.Sprintf("failed to build IR: %v", err),
			Context:     "building execution graph",
			TotalEvents: len(events),
		}
	}

	if config.Debug >= DebugPaths {
		debugEvents = append(debugEvents, DebugEvent{
			Timestamp: time.Now(),
			Event:     "build_ir_complete",
			Context:   fmt.Sprintf("statements=%d functions=%d", len(graph.Statements), len(graph.Functions)),
		})
	}

	// Step 2: Resolve - Wave-based resolution with pruned tree output
	if config.Debug >= DebugPaths {
		debugEvents = append(debugEvents, DebugEvent{
			Timestamp: time.Now(),
			Event:     "resolve_start",
			Context:   "",
		})
	}

	session := decorator.NewLocalSession()
	resolveConfig := ResolveConfig{
		TargetFunction: config.Target,
		Context:        nil, // TODO: pass context from config if available
	}

	resolveResult, err := Resolve(graph, vlt, session, resolveConfig)
	if err != nil {
		return nil, &PlanError{
			Message:     fmt.Sprintf("failed to resolve: %v", err),
			Context:     "resolving expressions",
			TotalEvents: len(events),
		}
	}

	if config.Debug >= DebugPaths {
		debugEvents = append(debugEvents, DebugEvent{
			Timestamp: time.Now(),
			Event:     "resolve_complete",
			Context:   fmt.Sprintf("statements=%d", len(resolveResult.Statements)),
		})
	}

	// Step 3: Emit - Generate plan with structure preservation
	if config.Debug >= DebugPaths {
		debugEvents = append(debugEvents, DebugEvent{
			Timestamp: time.Now(),
			Event:     "emit_start",
			Context:   "",
		})
	}

	// Determine which scopes to use for emission
	var scopes *ScopeStack
	if resolveConfig.TargetFunction != "" && graph.Functions[resolveConfig.TargetFunction] != nil {
		// Command mode: use function's scopes
		fn := graph.Functions[resolveConfig.TargetFunction]
		if fn.Scopes != nil {
			scopes = fn.Scopes
		} else {
			scopes = graph.Scopes
		}
	} else {
		// Script mode: use graph's scopes
		scopes = graph.Scopes
	}

	emitter := NewEmitter(resolveResult, vlt, scopes, config.Target)
	plan, err := emitter.Emit()
	if err != nil {
		return nil, &PlanError{
			Message:     fmt.Sprintf("failed to emit plan: %v", err),
			Context:     "emitting plan",
			TotalEvents: len(events),
		}
	}

	if config.Debug >= DebugPaths {
		debugEvents = append(debugEvents, DebugEvent{
			Timestamp: time.Now(),
			Event:     "emit_complete",
			Context:   fmt.Sprintf("steps=%d secretUses=%d", len(plan.Steps), len(plan.SecretUses)),
		})
	}

	// Set plan metadata
	plan.Target = config.Target
	vaultKey := vlt.GetPlanKey()
	if vaultKey != nil {
		plan.PlanSalt = vaultKey
	}

	// Validate the plan
	if err := plan.Validate(); err != nil {
		return nil, &PlanError{
			Message:     fmt.Sprintf("plan validation failed: %v", err),
			Context:     "validating plan",
			TotalEvents: len(events),
		}
	}

	// Finalize telemetry
	planTime := time.Since(startTime)
	if telemetry != nil {
		telemetry.EventCount = len(events)
		telemetry.StepCount = len(plan.Steps)
	}

	// Record final debug event
	if config.Debug >= DebugPaths {
		debugEvents = append(debugEvents, DebugEvent{
			Timestamp: time.Now(),
			Event:     "exit_plan",
			Context:   fmt.Sprintf("steps=%d secretUses=%d", len(plan.Steps), len(plan.SecretUses)),
		})
	}

	return &PlanResult{
		Plan:        plan,
		PlanTime:    planTime,
		Telemetry:   telemetry,
		DebugEvents: debugEvents,
	}, nil
}
