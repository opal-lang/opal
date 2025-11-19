// Package planner converts parser events into execution plans.
//
// # Decorator Registry Requirement
//
// IMPORTANT: Callers must import the decorator registry for the parser to detect
// decorator patterns like @var.X, @env.X, etc. Without the registry, the parser
// won't emit NodeDecorator events and interpolation will be skipped.
//
// Import in your main package or test files:
//
//	import _ "github.com/aledsdavies/opal/runtime/decorators"
//
// # Contract Stability
//
// Plans are function-scoped by design. When planning a target function, only that
// function's events are processed - other functions in the file are skipped entirely.
// This means changing unrelated functions doesn't invalidate existing contracts.
//
// Example:
//
//	fun hello = echo "Hello"  // Contract hash: abc123
//	fun log = echo "Log"      // Contract hash: def456
//
// Changing 'log' doesn't invalidate 'hello' contract because:
//   - planTargetFunction() finds 'hello' and returns immediately
//   - planFunctionBody() uses depth tracking to process only 'hello' events
//   - Plan contains only 'hello' steps
//   - Hash is computed from Plan (not entire source file)
//
// Future: When @cmd.function() calls are added, dependency tracking will be needed.
// If 'hello' calls 'log', then 'hello' contract must include both functions.
package planner

import (
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	"github.com/aledsdavies/opal/core/decorator"
	"github.com/aledsdavies/opal/core/invariant"
	"github.com/aledsdavies/opal/core/planfmt"
	"github.com/aledsdavies/opal/core/sdk/secret"
	"github.com/aledsdavies/opal/runtime/lexer"
	"github.com/aledsdavies/opal/runtime/parser"
	"github.com/aledsdavies/opal/runtime/vault"
	"github.com/lithammer/fuzzysearch/fuzzy"
)

// Command represents a single decorator invocation during planning (internal type).
// Commands are collected from parser events and then converted to ExecutionNode tree.
// This is an intermediate representation - the final Step only contains the Tree.
type Command struct {
	Decorator      string         // "@shell", "@retry", "@parallel", etc.
	Args           []planfmt.Arg  // Decorator arguments
	Block          []planfmt.Step // Nested steps (for decorators with blocks)
	Operator       string         // "&&", "||", "|", ";" - how to chain to NEXT command (empty for last)
	RedirectMode   string         // ">", ">>" - redirect mode (empty if no redirect)
	RedirectTarget *Command       // For redirect operators, the target decorator (nil otherwise)
}

// CommandPartKind identifies the type of a command part
type CommandPartKind int

const (
	PartLiteral CommandPartKind = iota // Literal text (e.g., "echo ", "foo")
	PartVarRef                         // Variable reference with captured exprID
)

// CommandPart represents a piece of a command string
// Commands are tokenized into parts during Pass 2 to preserve temporal binding of variables to exprIDs.
type CommandPart struct {
	Kind   CommandPartKind // Type of this part
	Text   string          // For PartLiteral: the literal text
	ExprID string          // For PartVarRef: the captured exprID from Pass 2
}

// CommandIR is the intermediate representation of a command with captured exprIDs.
// Built during Pass 2, used during Pass 3 for interpolation.
// This preserves temporal binding: each @var.X usage captures the exprID that was in scope
// at the time the command was parsed, preventing shadowing bugs.
type CommandIR struct {
	Parts []CommandPart // Sequence of literals and variable references
}

// Config configures the planner
type Config struct {
	Target    string           // Command name (e.g., "hello") or "" for script mode
	IDFactory secret.IDFactory // Factory for generating deterministic secret IDs (optional, uses run-mode if nil)
	Vault     *vault.Vault     // Shared vault for variable storage and scrubbing (optional, creates new if nil)
	Telemetry TelemetryLevel   // Telemetry level (production-safe)
	Debug     DebugLevel       // Debug level (development only)
}

// TelemetryLevel controls telemetry collection (production-safe)
type TelemetryLevel int

const (
	TelemetryOff    TelemetryLevel = iota // Zero overhead (default)
	TelemetryBasic                        // Step counts only
	TelemetryTiming                       // Counts + timing per phase
)

// DebugLevel controls debug tracing (development only)
type DebugLevel int

const (
	DebugOff      DebugLevel = iota // No debug info (default)
	DebugPaths                      // Method call tracing (enter/exit)
	DebugDetailed                   // Event-level tracing (every step)
)

// PlanResult holds the plan and observability data
type PlanResult struct {
	Plan        *planfmt.Plan  // The execution plan
	PlanTime    time.Duration  // Planning time (always collected)
	Telemetry   *PlanTelemetry // Additional metrics (nil if TelemetryOff)
	DebugEvents []DebugEvent   // Debug events (nil if DebugOff)
}

// PlanTelemetry holds additional planner metrics (optional, production-safe)
type PlanTelemetry struct {
	EventCount int // Number of events processed
	StepCount  int // Number of steps created

	// Decorator resolution metrics (keyed by decorator name: "@var", "@env", "@aws.secret", etc.)
	DecoratorResolutions map[string]*DecoratorResolutionMetrics
}

// DecoratorResolutionMetrics tracks resolution statistics for a specific decorator type
type DecoratorResolutionMetrics struct {
	TotalCalls   int           // Total number of resolution calls
	BatchCalls   int           // Number of batch resolution calls (0 if no batching)
	BatchSizes   []int         // Size of each batch (empty if no batching)
	TotalTime    time.Duration // Total time spent resolving (if timing enabled)
	SkippedCalls int           // Calls skipped due to lazy evaluation
}

// DebugEvent holds debug tracing information (development only)
type DebugEvent struct {
	Timestamp time.Time
	Event     string // "enter_plan", "function_found", "step_created", etc.
	EventPos  int    // Current position in event stream
	Context   string // Additional context
}

// PlanError represents a planning error with rich context
type PlanError struct {
	Message     string // Clear, specific error message
	Context     string // What we were planning
	EventPos    int    // Position in event stream
	TotalEvents int    // Total events for context
	Suggestion  string // How to fix it
	Example     string // Valid example
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

// Plan consumes parser events and generates an execution plan.
func Plan(events []parser.Event, tokens []lexer.Token, config Config) (*planfmt.Plan, error) {
	result, err := PlanWithObservability(events, tokens, config)
	if err != nil {
		return nil, err
	}
	return result.Plan, nil
}

// PlanWithObservability returns plan with telemetry and debug events
func PlanWithObservability(events []parser.Event, tokens []lexer.Token, config Config) (*PlanResult, error) {
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

	// Initialize ID factory for deterministic placeholder generation
	idFactory := config.IDFactory
	if idFactory == nil {
		// Default: use plan mode with random 32-byte key
		key := make([]byte, 32)
		_, err := rand.Read(key)
		invariant.ExpectNoError(err, "failed to generate IDFactory key")
		idFactory = secret.NewIDFactory(secret.ModePlan, key)
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

	p := &planner{
		events:        events,
		tokens:        tokens,
		config:        config,
		pos:           0,
		stepID:        1,
		vault:         vlt,                         // Scope-aware variable storage (shared or new)
		session:       decorator.NewLocalSession(), // Session for decorator resolution
		idFactory:     idFactory,                   // For placeholder generation
		commandIRs:    make(map[uint64]*CommandIR), // CommandIR storage (Pass 1 → Pass 3)
		nextCommandID: 1,
		telemetry:     telemetry,
		debugEvents:   debugEvents,
	}

	plan, err := p.plan()
	if err != nil {
		return nil, err
	}

	// Finalize telemetry
	planTime := time.Since(startTime)
	if telemetry != nil {
		telemetry.EventCount = len(events)
		telemetry.StepCount = len(plan.Steps)
	}

	return &PlanResult{
		Plan:        plan,
		PlanTime:    planTime,
		Telemetry:   telemetry,
		DebugEvents: p.debugEvents,
	}, nil
}

// decoratorBlockContext tracks active decorator blocks for scope management.
// Execution decorators create isolated scopes where variables don't leak out.
type decoratorBlockContext struct {
	name string // Decorator name for debugging
}

// planner holds state during planning
type planner struct {
	events []parser.Event
	tokens []lexer.Token
	config Config

	pos    int    // Current position in event stream
	stepID uint64 // Next step ID to assign

	// Variable scoping with transport boundary guards
	vault   *vault.Vault      // Scope-aware variable storage
	session decorator.Session // Session for decorator resolution (LocalSession by default)

	// Decorator block scope tracking
	// Execution decorators (@retry, @timeout, @parallel, etc.) create isolated scopes
	// where variables declared inside don't leak to outer scope
	decoratorStack []decoratorBlockContext

	// Placeholder generation (for deterministic IDs in plan mode)
	idFactory secret.IDFactory // For generating deterministic placeholder IDs

	// CommandIR storage (Pass 1 → Pass 3)
	// Maps unique command IDs to their tokenized representation with captured exprIDs
	// Each command gets a unique ID during Pass 1 to handle shadowing correctly
	commandIRs    map[uint64]*CommandIR
	nextCommandID uint64

	// Observability
	telemetry   *PlanTelemetry
	debugEvents []DebugEvent
}

// recordDebugEvent records debug events when debug tracing is enabled
func (p *planner) recordDebugEvent(event, context string) {
	if p.config.Debug == DebugOff || p.debugEvents == nil {
		return
	}
	p.debugEvents = append(p.debugEvents, DebugEvent{
		Timestamp: time.Now(),
		Event:     event,
		EventPos:  p.pos,
		Context:   context,
	})
}

// recordDecoratorResolution records a single decorator resolution
func (p *planner) recordDecoratorResolution(decoratorName string) {
	if p.telemetry == nil {
		return
	}
	metrics := p.getOrCreateMetrics(decoratorName)
	metrics.TotalCalls++
}

// getOrCreateMetrics gets or creates metrics for a decorator
func (p *planner) getOrCreateMetrics(decoratorName string) *DecoratorResolutionMetrics {
	if p.telemetry.DecoratorResolutions[decoratorName] == nil {
		p.telemetry.DecoratorResolutions[decoratorName] = &DecoratorResolutionMetrics{
			BatchSizes: []int{},
		}
	}
	return p.telemetry.DecoratorResolutions[decoratorName]
}

// checkDecoratorBlock peeks ahead to determine if current position is a decorator with a block.
// Returns (hasBlock, decoratorName) without advancing position.
// Used to decide whether to enter isolated scope for execution decorators.
func (p *planner) checkDecoratorBlock() (bool, string) {
	if p.pos >= len(p.events) {
		return false, ""
	}

	evt := p.events[p.pos]
	if evt.Kind != parser.EventOpen || parser.NodeKind(evt.Data) != parser.NodeDecorator {
		return false, ""
	}

	savedPos := p.pos
	defer func() { p.pos = savedPos }()

	p.pos++

	// Skip @ token
	if p.pos < len(p.events) && p.events[p.pos].Kind == parser.EventToken {
		p.pos++
	}

	// Extract decorator name
	decoratorName := ""
	if p.pos < len(p.events) && p.events[p.pos].Kind == parser.EventToken {
		tokenIdx := p.events[p.pos].Data
		decoratorName = "@" + string(p.tokens[tokenIdx].Text)
		p.pos++
	}

	// Search for NodeBlock within decorator
	depth := 0
	for p.pos < len(p.events) {
		prevPos := p.pos
		evt := p.events[p.pos]

		switch evt.Kind {
		case parser.EventOpen:
			if parser.NodeKind(evt.Data) == parser.NodeBlock {
				return true, decoratorName
			}
			depth++
		case parser.EventClose:
			depth--
			if depth < 0 {
				return false, ""
			}
		}

		p.pos++
		invariant.Invariant(p.pos > prevPos, "checkDecoratorBlock stuck at pos %d", prevPos)
	}

	return false, ""
}

// processDecoratorBlock handles a decorator block by creating a Step for the decorator itself.
// Assumes p.pos is at STEP_ENTER and the step contains a decorator block.
// Returns a Step containing the decorator CommandNode with its block steps.
func (p *planner) processDecoratorBlock(decoratorName string) (planfmt.Step, error) {
	p.pos++ // Move past STEP_ENTER

	// Skip to decorator
	for p.pos < len(p.events) && p.events[p.pos].Kind != parser.EventOpen {
		p.pos++
	}

	// We're now at OPEN Decorator
	p.pos++ // Move past OPEN Decorator

	// Skip @ and decorator name tokens
	for p.pos < len(p.events) && p.events[p.pos].Kind == parser.EventToken {
		p.pos++
	}

	// Parse decorator arguments (ParamList)
	var args []planfmt.Arg
	if p.pos < len(p.events) && p.events[p.pos].Kind == parser.EventOpen &&
		parser.NodeKind(p.events[p.pos].Data) == parser.NodeParamList {
		var err error
		args, err = p.parseParamList()
		if err != nil {
			return planfmt.Step{}, err
		}
	}

	// Enter scope for variable isolation
	p.vault.Push(decoratorName)
	p.decoratorStack = append(p.decoratorStack, decoratorBlockContext{
		name: decoratorName,
	})

	// Ensure scope is popped on all exit paths (normal return or error)
	closed := false
	defer func() {
		if !closed {
			p.vault.Pop()
			p.decoratorStack = p.decoratorStack[:len(p.decoratorStack)-1]
		}
	}()

	if p.config.Debug >= DebugDetailed {
		p.recordDebugEvent("decorator_block_enter",
			fmt.Sprintf("name=%s", decoratorName))
	}

	// Find the block
	for p.pos < len(p.events) {
		prevPos := p.pos
		evt := p.events[p.pos]
		if evt.Kind == parser.EventOpen && parser.NodeKind(evt.Data) == parser.NodeBlock {
			p.pos++ // Move past OPEN Block
			break
		}
		p.pos++
		invariant.Invariant(p.pos > prevPos, "processDecoratorBlock stuck finding block at pos %d", prevPos)
	}

	// Collect block steps into local slice
	var blockSteps []planfmt.Step

	// Process nested steps
	for p.pos < len(p.events) {
		prevPos := p.pos
		evt := p.events[p.pos]

		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodeBlock {
			// Exit block
			p.pos++

			// Pop scope
			p.vault.Pop()
			p.decoratorStack = p.decoratorStack[:len(p.decoratorStack)-1]
			closed = true // Mark as closed so defer becomes no-op

			if p.config.Debug >= DebugDetailed {
				p.recordDebugEvent("decorator_block_exit", fmt.Sprintf("name=%s", decoratorName))
			}

			// Skip past CLOSE Decorator and STEP_EXIT
			for p.pos < len(p.events) {
				evt := p.events[p.pos]
				p.pos++
				if evt.Kind == parser.EventStepExit {
					break
				}
			}

			// Create CommandNode for the decorator
			decoratorCmd := &planfmt.CommandNode{
				Decorator: decoratorName,
				Args:      args,
				Block:     blockSteps,
			}

			// Create Step
			step := planfmt.Step{
				ID:   p.nextStepID(),
				Tree: decoratorCmd,
			}

			return step, nil
		}

		if evt.Kind == parser.EventStepEnter {
			// Check for nested decorator block
			savedPos := p.pos
			p.pos++

			hasNestedBlock := false
			nestedName := ""

			if p.pos < len(p.events) {
				nextEvt := p.events[p.pos]
				if nextEvt.Kind == parser.EventOpen && parser.NodeKind(nextEvt.Data) == parser.NodeDecorator {
					hasNestedBlock, nestedName = p.checkDecoratorBlock()
				}
			}

			p.pos = savedPos

			if hasNestedBlock {
				// Recursively handle nested decorator block
				nestedStep, err := p.processDecoratorBlock(nestedName)
				if err != nil {
					return planfmt.Step{}, err
				}
				if nestedStep.ID != 0 {
					blockSteps = append(blockSteps, nestedStep)
				}
				continue
			}

			// Normal step
			step, err := p.planStep()
			if err != nil {
				return planfmt.Step{}, err
			}
			if step.ID != 0 {
				blockSteps = append(blockSteps, step)
			}
			continue
		}

		p.pos++
		invariant.Invariant(p.pos > prevPos, "processDecoratorBlock stuck processing steps at pos %d", prevPos)
	}

	return planfmt.Step{}, fmt.Errorf("decorator block not closed properly")
}

// parseParamList parses decorator parameters from the event stream.
// Expects to be positioned at OPEN ParamList, leaves position after CLOSE ParamList.
func (p *planner) parseParamList() ([]planfmt.Arg, error) {
	var args []planfmt.Arg

	// PRECONDITION: Must be at OPEN ParamList
	invariant.Precondition(p.pos < len(p.events) &&
		p.events[p.pos].Kind == parser.EventOpen &&
		parser.NodeKind(p.events[p.pos].Data) == parser.NodeParamList,
		"parseParamList must start at OPEN ParamList")

	p.pos++ // Move past OPEN ParamList

	// Skip opening paren token
	if p.pos < len(p.events) && p.events[p.pos].Kind == parser.EventToken {
		p.pos++
	}

	// Parse each parameter
	for p.pos < len(p.events) {
		prevPos := p.pos
		evt := p.events[p.pos]

		// Check for CLOSE ParamList
		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodeParamList {
			p.pos++ // Move past CLOSE ParamList
			break
		}

		// Parse individual parameter
		if evt.Kind == parser.EventOpen && parser.NodeKind(evt.Data) == parser.NodeParam {
			arg, err := p.parseParam()
			if err != nil {
				return nil, err
			}
			args = append(args, arg)
			continue
		}

		// Skip tokens (commas, whitespace, etc.)
		if evt.Kind == parser.EventToken {
			p.pos++
			continue
		}

		p.pos++
		invariant.Invariant(p.pos > prevPos, "parseParamList stuck at pos %d", prevPos)
	}

	return args, nil
}

// parseParam parses a single parameter (key=value).
// Expects to be positioned at OPEN Param, leaves position after CLOSE Param.
func (p *planner) parseParam() (planfmt.Arg, error) {
	// PRECONDITION: Must be at OPEN Param
	invariant.Precondition(p.pos < len(p.events) &&
		p.events[p.pos].Kind == parser.EventOpen &&
		parser.NodeKind(p.events[p.pos].Data) == parser.NodeParam,
		"parseParam must start at OPEN Param")

	p.pos++ // Move past OPEN Param

	// Parse parameter name
	var paramName string
	if p.pos < len(p.events) && p.events[p.pos].Kind == parser.EventToken {
		tokenIdx := p.events[p.pos].Data
		paramName = string(p.tokens[tokenIdx].Text)
		p.pos++
	}

	// Skip = token
	if p.pos < len(p.events) && p.events[p.pos].Kind == parser.EventToken {
		p.pos++
	}

	// Parse parameter value
	var paramValue planfmt.Value
	if p.pos < len(p.events) && p.events[p.pos].Kind == parser.EventToken {
		tokenIdx := p.events[p.pos].Data
		token := p.tokens[tokenIdx]
		tokenText := string(token.Text)

		// Determine value type from token
		switch token.Type {
		case lexer.INTEGER:
			// Parse integer
			var intVal int64
			if _, err := fmt.Sscanf(tokenText, "%d", &intVal); err != nil {
				return planfmt.Arg{}, fmt.Errorf("failed to parse integer parameter %q: %w", paramName, err)
			}
			paramValue = planfmt.Value{
				Kind: planfmt.ValueInt,
				Int:  intVal,
			}
		case lexer.STRING:
			// String value (remove quotes)
			str := tokenText
			if len(str) >= 2 && str[0] == '"' && str[len(str)-1] == '"' {
				str = str[1 : len(str)-1]
			}
			paramValue = planfmt.Value{
				Kind: planfmt.ValueString,
				Str:  str,
			}
		case lexer.BOOLEAN:
			// Boolean value
			paramValue = planfmt.Value{
				Kind: planfmt.ValueBool,
				Bool: tokenText == "true",
			}
		case lexer.DURATION:
			// Duration value (stored as string)
			paramValue = planfmt.Value{
				Kind: planfmt.ValueString,
				Str:  tokenText,
			}
		default:
			// Default to string
			paramValue = planfmt.Value{
				Kind: planfmt.ValueString,
				Str:  tokenText,
			}
		}
		p.pos++
	}

	// Skip to CLOSE Param
	for p.pos < len(p.events) {
		prevPos := p.pos
		evt := p.events[p.pos]
		if evt.Kind == parser.EventClose && parser.NodeKind(evt.Data) == parser.NodeParam {
			p.pos++ // Move past CLOSE Param
			break
		}
		p.pos++
		invariant.Invariant(p.pos > prevPos, "parseParam stuck at pos %d", prevPos)
	}

	return planfmt.Arg{
		Key: paramName,
		Val: paramValue,
	}, nil
}

// plan is the main planning entry point
func (p *planner) plan() (*planfmt.Plan, error) {
	if p.config.Debug >= DebugPaths {
		p.recordDebugEvent("enter_plan", "target="+p.config.Target)
	}

	// Contract verification requires same PlanSalt for deterministic DisplayIDs
	plan := planfmt.NewPlan()

	// Preserve NewPlan's random salt unless vault provides one for verification
	vaultKey := p.vault.GetPlanKey()
	if vaultKey != nil {
		invariant.Invariant(len(vaultKey) == 32,
			"vault plan key must be 32 bytes, got %d", len(vaultKey))
		plan.PlanSalt = vaultKey
	}
	plan.Header.PlanKind = 0 // View plan
	plan.Target = p.config.Target
	plan.Steps = []planfmt.Step{}

	// Pass 1: Scan - build complete execution graph
	if p.config.Target != "" {
		steps, err := p.planTargetFunction()
		if err != nil {
			return nil, err
		}
		plan.Steps = steps
	} else {
		steps, err := p.planSource()
		if err != nil {
			return nil, err
		}
		plan.Steps = steps
	}

	// Pass 2: Resolve - mark all touched expressions as resolved
	// CommandIRs were already built during Pass 1 (in planCommand)
	// Enables batching efficiency: decorators can batch API calls (e.g., multiple @aws.secret)
	p.vault.ResolveAllTouched()

	// Pass 3: Interpolate - replace all @var.X with DisplayIDs using CommandIRs
	if err := p.interpolateAllCommands(plan.Steps); err != nil {
		return nil, err
	}

	// Prune untouched expressions (declared but never used)
	// Saves API calls and reduces secrets in plan
	p.vault.PruneUntouched()

	// Build SecretUses from Vault (authorization list for contract verification)
	// This populates the plan with DisplayID → SiteID mappings for each variable usage.
	// Same DisplayID can appear multiple times (different usage sites).
	vaultUses := p.vault.BuildSecretUses()
	for _, use := range vaultUses {
		plan.SecretUses = append(plan.SecretUses, planfmt.SecretUse{
			DisplayID: use.DisplayID,
			SiteID:    use.SiteID,
			Site:      use.Site,
		})
	}

	// POSTCONDITION: plan must be valid
	err := plan.Validate()
	invariant.ExpectNoError(err, "plan validation")

	if p.config.Debug >= DebugPaths {
		p.recordDebugEvent("exit_plan", fmt.Sprintf("steps=%d secretUses=%d", len(plan.Steps), len(plan.SecretUses)))
	}

	return plan, nil
}

// planTargetFunction finds the target function and returns immediately after planning it.
// Other functions in the file are never processed (contract stability).
func (p *planner) planTargetFunction() ([]planfmt.Step, error) {
	if p.config.Debug >= DebugPaths {
		p.recordDebugEvent("enter_planTargetFunction", "target="+p.config.Target)
	}

	// Collect all available function names for "did you mean" suggestions
	var availableFunctions []string

	// Walk events to find the target function
	for p.pos < len(p.events) {
		prevPos := p.pos
		evt := p.events[p.pos]

		if evt.Kind == parser.EventOpen && parser.NodeKind(evt.Data) == parser.NodeFunction {
			// Found a function, check if it's our target
			// Event structure: OPEN Function, TOKEN(fun), TOKEN(name), TOKEN(=), ...
			funcNamePos := p.pos + 2 // Skip OPEN Function and TOKEN(fun)
			if funcNamePos < len(p.events) && p.events[funcNamePos].Kind == parser.EventToken {
				funcNameTokenIdx := p.events[funcNamePos].Data
				funcName := string(p.tokens[funcNameTokenIdx].Text)

				// Collect for suggestions
				availableFunctions = append(availableFunctions, funcName)

				if funcName == p.config.Target {
					if p.config.Debug >= DebugDetailed {
						p.recordDebugEvent("function_found", fmt.Sprintf("name=%s pos=%d", funcName, p.pos))
					}

					// Plan the function body
					return p.planFunctionBody()
				}
			}
		}

		p.pos++

		// INVARIANT: position must advance (no infinite loops)
		invariant.Invariant(p.pos > prevPos, "position must advance in planTargetFunction (stuck at pos %d)", prevPos)
	}

	// Build error with "did you mean" suggestion
	suggestion := fmt.Sprintf("Define the function with: fun %s = <command>", p.config.Target)
	example := fmt.Sprintf("fun %s = echo \"Hello\"", p.config.Target)

	if len(availableFunctions) > 0 {
		closest := findClosestMatch(p.config.Target, availableFunctions)
		if closest != "" {
			suggestion = fmt.Sprintf("Did you mean '%s'?", closest)
			example = fmt.Sprintf("Available commands: %s", strings.Join(availableFunctions, ", "))
		}
	}

	return nil, &PlanError{
		Message:     fmt.Sprintf("command not found: %s", p.config.Target),
		Context:     "searching for target function",
		EventPos:    p.pos,
		TotalEvents: len(p.events),
		Suggestion:  suggestion,
		Example:     example,
	}
}

// planFunctionBody plans the body of a function using depth tracking.
// Stops when depth reaches 0 (exited function), ensuring only target function events are processed.
func (p *planner) planFunctionBody() ([]planfmt.Step, error) {
	if p.config.Debug >= DebugPaths {
		p.recordDebugEvent("enter_planFunctionBody", fmt.Sprintf("pos=%d", p.pos))
	}

	// Skip to function body (past OPEN Function, name token, '=' token)
	depth := 1
	p.pos++ // Move past OPEN Function

	var steps []planfmt.Step

	for p.pos < len(p.events) && depth > 0 {
		prevPos := p.pos
		evt := p.events[p.pos]

		if evt.Kind == parser.EventStepEnter {
			// Check if this step contains a decorator block
			savedPos := p.pos
			p.pos++

			hasDecoratorBlock := false
			decoratorName := ""

			if p.pos < len(p.events) {
				nextEvt := p.events[p.pos]
				if nextEvt.Kind == parser.EventOpen && parser.NodeKind(nextEvt.Data) == parser.NodeDecorator {
					hasDecoratorBlock, decoratorName = p.checkDecoratorBlock()
				}
			}

			p.pos = savedPos

			if hasDecoratorBlock {
				step, err := p.processDecoratorBlock(decoratorName)
				if err != nil {
					return nil, err
				}
				if step.ID != 0 {
					steps = append(steps, step)
				}
				continue
			}

			// Normal step
			step, err := p.planStep()
			if err != nil {
				return nil, err
			}
			if step.ID != 0 {
				steps = append(steps, step)
			}
			continue
		} else if evt.Kind == parser.EventOpen {
			depth++
		} else if evt.Kind == parser.EventClose {
			depth--
		}

		p.pos++

		// INVARIANT: position must advance
		invariant.Invariant(p.pos > prevPos, "position must advance in planFunctionBody (stuck at pos %d)", prevPos)
	}

	if len(steps) == 0 {
		return nil, &PlanError{
			Message:     "no commands found in function body",
			Context:     fmt.Sprintf("planning function %s", p.config.Target),
			EventPos:    p.pos,
			TotalEvents: len(p.events),
			Suggestion:  "Add at least one command to the function body",
			Example:     "fun hello = echo \"Hello, World!\"",
		}
	}

	if p.config.Debug >= DebugPaths {
		p.recordDebugEvent("exit_planFunctionBody", fmt.Sprintf("steps=%d", len(steps)))
	}

	return steps, nil
}

// planSource plans all top-level commands in script mode
func (p *planner) planSource() ([]planfmt.Step, error) {
	if p.config.Debug >= DebugPaths {
		p.recordDebugEvent("enter_planSource", "script mode")
	}

	var steps []planfmt.Step

	// Walk events looking for top-level step boundaries (EventStepEnter)
	// Skip step boundaries inside functions (depth > 1)
	depth := 0
	for p.pos < len(p.events) {
		prevPos := p.pos
		evt := p.events[p.pos]

		if evt.Kind == parser.EventOpen {
			depth++
		} else if evt.Kind == parser.EventClose {
			depth--
		} else if evt.Kind == parser.EventStepEnter && depth == 1 {
			// Top-level step - check if it contains a decorator block
			savedPos := p.pos
			p.pos++

			hasDecoratorBlock := false
			decoratorName := ""

			if p.pos < len(p.events) {
				nextEvt := p.events[p.pos]
				if nextEvt.Kind == parser.EventOpen && parser.NodeKind(nextEvt.Data) == parser.NodeDecorator {
					hasDecoratorBlock, decoratorName = p.checkDecoratorBlock()
				}
			}

			p.pos = savedPos

			if hasDecoratorBlock {
				step, err := p.processDecoratorBlock(decoratorName)
				if err != nil {
					return nil, err
				}
				if step.ID != 0 {
					steps = append(steps, step)
				}
				continue
			}

			// Normal step
			step, err := p.planStep()
			if err != nil {
				return nil, err
			}
			if step.ID != 0 {
				steps = append(steps, step)
			}
			continue
		}

		p.pos++

		// INVARIANT: position must advance
		invariant.Invariant(p.pos > prevPos, "position must advance in planSource (stuck at pos %d)", prevPos)
	}

	if p.config.Debug >= DebugPaths {
		p.recordDebugEvent("exit_planSource", fmt.Sprintf("steps=%d", len(steps)))
	}

	return steps, nil
}

// nextStepID returns the next step ID and increments the counter
func (p *planner) nextStepID() uint64 {
	id := p.stepID
	p.stepID++
	return id
}

// findClosestMatch finds the closest string match using fuzzy matching
func findClosestMatch(target string, candidates []string) string {
	if len(candidates) == 0 {
		return ""
	}

	// Use fuzzy ranking to find best match
	ranks := fuzzy.RankFindFold(target, candidates)
	if len(ranks) > 0 {
		// Return the best match (lowest distance)
		return ranks[0].Target
	}

	return ""
}

// planStep plans a single step (from EventStepEnter to EventStepExit)
// A step contains one or more shell commands connected by operators
func (p *planner) planStep() (planfmt.Step, error) {
	if p.config.Debug >= DebugDetailed {
		p.recordDebugEvent("enter_planStep", fmt.Sprintf("pos=%d", p.pos))
	}

	// We're at EventStepEnter, move past it
	p.pos++

	// Track step in Vault for site path generation (authorization)
	// Note: Variables are declared at root scope (accessible across all steps)
	// but site paths include step segment for authorization granularity
	stepName := fmt.Sprintf("step-%d", p.stepID)
	p.vault.ResetCounts() // Reset decorator indices for new step
	p.vault.Push(stepName)
	defer p.vault.Pop()

	var commands []Command

	// Collect all shell commands and var declarations until EventStepExit
	for p.pos < len(p.events) {
		evt := p.events[p.pos]

		if evt.Kind == parser.EventStepExit {
			// End of step
			p.pos++ // Move past EventStepExit
			break
		}

		if evt.Kind == parser.EventOpen && parser.NodeKind(evt.Data) == parser.NodeVarDecl {
			// Found a variable declaration
			err := p.planVarDecl()
			if err != nil {
				return planfmt.Step{}, err
			}
			// planVarDecl already advanced p.pos, continue
			continue
		}

		if evt.Kind == parser.EventOpen && parser.NodeKind(evt.Data) == parser.NodeShellCommand {
			// Found a shell command
			cmd, err := p.planCommand()
			if err != nil {
				return planfmt.Step{}, err
			}
			commands = append(commands, cmd)
			// planCommand already advanced p.pos, continue
			continue
		}

		p.pos++
	}

	if len(commands) == 0 {
		// Step has only var declarations (no commands)
		// Return empty step with ID=0 as sentinel (caller should skip adding to plan)
		if p.config.Debug >= DebugDetailed {
			p.recordDebugEvent("step_skipped", "step has only var declarations")
		}
		return planfmt.Step{ID: 0}, nil
	}

	step := planfmt.Step{
		ID:   p.nextStepID(),
		Tree: buildStepTree(commands),
	}

	if p.config.Debug >= DebugDetailed {
		p.recordDebugEvent("step_created", fmt.Sprintf("id=%d commands=%d", step.ID, len(commands)))
	}

	return step, nil
}

// planVarDecl processes a variable declaration and stores the value
// Event structure:
//   - Simple form: OPEN VarDecl, TOKEN(var), TOKEN(name), TOKEN(=), OPEN Literal, TOKEN(value), CLOSE Literal, CLOSE VarDecl
//   - Block form: TOKEN(var), TOKEN((), [OPEN VarDecl, TOKEN(name), TOKEN(=), ...], TOKEN())
func (p *planner) planVarDecl() error {
	if p.config.Debug >= DebugDetailed {
		p.recordDebugEvent("enter_planVarDecl", fmt.Sprintf("pos=%d", p.pos))
	}

	startPos := p.pos
	p.pos++ // Move past OPEN VarDecl

	// PRECONDITION: Must have at least 2 tokens (name, =) + literal
	invariant.Invariant(p.pos+2 < len(p.events), "planVarDecl: insufficient events at pos %d", startPos)

	// Check if next token is 'var' keyword (simple form) or identifier (block form)
	if p.events[p.pos].Kind == parser.EventToken {
		tokenIdx := p.events[p.pos].Data
		tokenText := string(p.tokens[tokenIdx].Text)

		// If it's 'var', skip it (simple form)
		if tokenText == "var" {
			p.pos++
		}
		// Otherwise it's the variable name (block form), don't skip
	}

	// Get variable name
	if p.events[p.pos].Kind != parser.EventToken {
		return &PlanError{
			Message:     "expected variable name",
			Context:     "parsing variable declaration",
			EventPos:    p.pos,
			TotalEvents: len(p.events),
		}
	}
	nameTokenIdx := p.events[p.pos].Data
	varName := string(p.tokens[nameTokenIdx].Text)
	p.pos++

	// Skip TOKEN(=)
	p.pos++

	// Parse the value expression (supports literals, objects, arrays, decorators)
	value, err := p.parseVarValue(varName)
	if err != nil {
		return err
	}

	// Variable scope excludes step segments because steps are not scopes
	rawExpr := fmt.Sprintf("literal:%v", value)
	exprID := p.vault.DeclareVariable(varName, rawExpr)

	// Store value for deferred resolution to enable batching efficiency
	// Preserves original type (string, int, bool, map, slice)
	p.vault.StoreUnresolvedValue(exprID, value)

	// Record telemetry
	p.recordDecoratorResolution("@var")

	// Record debug event
	if p.config.Debug >= DebugDetailed {
		displayID := p.vault.GetDisplayID(exprID)
		p.recordDebugEvent("var_declared", fmt.Sprintf("name=%s value=%v exprID=%s displayID=%s",
			varName, value, exprID, displayID))
	}

	return nil
}

// planCommand plans a single command within a step (shell command + optional operator)
func (p *planner) planCommand() (Command, error) {
	if p.config.Debug >= DebugDetailed {
		p.recordDebugEvent("enter_planCommand", fmt.Sprintf("pos=%d", p.pos))
	}

	startPos := p.pos
	p.pos++ // Move past OPEN ShellCommand

	// Collect all token indices in the shell command
	// Also detect NodeDecorator for variable interpolation
	var tokenIndices []uint32
	var hasDecorator bool
	depth := 1

	for p.pos < len(p.events) && depth > 0 {
		evt := p.events[p.pos]

		if evt.Kind == parser.EventOpen {
			depth++
			// Check if this is a NodeDecorator (variable interpolation)
			if parser.NodeKind(evt.Data) == parser.NodeDecorator {
				hasDecorator = true
			}
		} else if evt.Kind == parser.EventClose {
			depth--
			if depth == 0 {
				// Move past the CLOSE ShellCommand event
				p.pos++
				break
			}
		} else if evt.Kind == parser.EventToken {
			tokenIndices = append(tokenIndices, evt.Data)
		}

		p.pos++
	}

	// Build command string using HasSpaceBefore to preserve original spacing
	command := ""
	for i, tokenIdx := range tokenIndices {
		token := p.tokens[tokenIdx]

		// Add space if this token had space before it (except for first token)
		if i > 0 && token.HasSpaceBefore {
			command += " "
		}

		// Get token text - handle operators with empty Text
		tokenText := getTokenText(token)
		command += tokenText
	}

	// POSTCONDITION: command must not be empty
	invariant.Postcondition(command != "", "shell command must not be empty")

	// THREE-PASS MODEL:
	// Pass 1: Build CommandIR NOW (capture exprIDs at this point in time)
	// Pass 2: Resolve all touched expressions
	// Pass 3: Interpolate using CommandIR (uses captured exprIDs)

	// Assign unique ID to this command instance
	commandID := p.nextCommandID
	p.nextCommandID++

	// Build CommandIR if command has decorators (captures exprIDs NOW)
	if hasDecorator {
		ir, err := p.buildCommandIR(command)
		if err != nil {
			return Command{}, err
		}
		// Store CommandIR with unique ID for Pass 3
		p.commandIRs[commandID] = ir
	}

	// Store raw command string for now (will be interpolated in Pass 3)
	// Also store command ID so Pass 3 can look up the CommandIR
	commandValue := planfmt.Value{
		Kind: planfmt.ValueString,
		Str:  command,
	}

	// Check for redirect operator after this command (> or >>)
	var redirectTarget *Command
	redirectMode := "" // ">" or ">>" - stored separately from chaining operator
	operator := ""     // Chaining operator: "&&", "||", "|", ";"

	if p.pos < len(p.events) {
		evt := p.events[p.pos]

		// Check for NodeRedirect
		if evt.Kind == parser.EventOpen && parser.NodeKind(evt.Data) == parser.NodeRedirect {
			p.pos++ // Move past OPEN NodeRedirect

			// Next should be the redirect operator token (> or >>)
			if p.pos < len(p.events) && p.events[p.pos].Kind == parser.EventToken {
				tokenIdx := p.events[p.pos].Data
				tokenType := p.tokens[tokenIdx].Type

				switch tokenType {
				case lexer.GT:
					redirectMode = ">"
				case lexer.APPEND:
					redirectMode = ">>"
				}
				p.pos++ // Consume the operator token
			}

			// Next should be OPEN NodeRedirectTarget
			if p.pos < len(p.events) && p.events[p.pos].Kind == parser.EventOpen &&
				parser.NodeKind(p.events[p.pos].Data) == parser.NodeRedirectTarget {
				p.pos++ // Move past OPEN NodeRedirectTarget

				// Collect tokens for redirect target
				var targetTokens []uint32
				targetDepth := 1

				for p.pos < len(p.events) && targetDepth > 0 {
					evt := p.events[p.pos]

					if evt.Kind == parser.EventOpen {
						targetDepth++
					} else if evt.Kind == parser.EventClose {
						targetDepth--
						if targetDepth == 0 {
							p.pos++ // Move past CLOSE NodeRedirectTarget
							break
						}
					} else if evt.Kind == parser.EventToken {
						targetTokens = append(targetTokens, evt.Data)
					}

					p.pos++
				}

				// Build target command string
				targetCmd := ""
				for i, tokenIdx := range targetTokens {
					token := p.tokens[tokenIdx]

					if i > 0 && token.HasSpaceBefore {
						targetCmd += " "
					}

					tokenText := getTokenText(token)
					targetCmd += tokenText
				}

				// Create redirect target command
				if targetCmd != "" {
					redirectTarget = &Command{
						Decorator: "@shell",
						Args: []planfmt.Arg{
							{
								Key: "command",
								Val: planfmt.Value{
									Kind: planfmt.ValueString,
									Str:  targetCmd,
								},
							},
						},
					}
				}
			}

			// Move past CLOSE NodeRedirect
			if p.pos < len(p.events) && p.events[p.pos].Kind == parser.EventClose {
				p.pos++
			}

			// CRITICAL FIX: After processing redirect, continue checking for chaining operators
			// This allows: echo a > out && echo b (both redirect AND chaining)
			if p.pos < len(p.events) && p.events[p.pos].Kind == parser.EventToken {
				tokenIdx := p.events[p.pos].Data
				tokenType := p.tokens[tokenIdx].Type

				switch tokenType {
				case lexer.AND_AND:
					operator = "&&"
					p.pos++ // Consume the operator
				case lexer.OR_OR:
					operator = "||"
					p.pos++ // Consume the operator
				case lexer.PIPE:
					operator = "|"
					p.pos++ // Consume the operator
				case lexer.SEMICOLON:
					operator = ";"
					p.pos++ // Consume the operator
				}
			}
		} else if evt.Kind == parser.EventToken {
			// Check for chaining operators (&&, ||, |, ;) when no redirect
			tokenIdx := evt.Data
			tokenType := p.tokens[tokenIdx].Type

			switch tokenType {
			case lexer.AND_AND:
				operator = "&&"
				p.pos++ // Consume the operator
			case lexer.OR_OR:
				operator = "||"
				p.pos++ // Consume the operator
			case lexer.PIPE:
				operator = "|"
				p.pos++ // Consume the operator
			case lexer.SEMICOLON:
				operator = ";"
				p.pos++ // Consume the operator
			}
		}
	}

	// Build Args with command and internal commandID (must be sorted alphabetically)
	args := []planfmt.Arg{}

	// Add internal __commandID for Pass 3 interpolation lookup (comes first alphabetically)
	if hasDecorator {
		args = append(args, planfmt.Arg{
			Key: "__commandID",
			Val: planfmt.Value{
				Kind: planfmt.ValueInt,
				Int:  int64(commandID),
			},
		})
	}

	// Add command argument
	args = append(args, planfmt.Arg{
		Key: "command",
		Val: commandValue,
	})

	cmd := Command{
		Decorator:      "@shell",
		Args:           args,
		Operator:       operator,
		RedirectMode:   redirectMode,
		RedirectTarget: redirectTarget,
	}

	if p.config.Debug >= DebugDetailed {
		p.recordDebugEvent("command_created", fmt.Sprintf("decorator=@shell command=%q operator=%q", command, operator))
	}

	// POSTCONDITION: position must advance
	invariant.Postcondition(p.pos > startPos, "position must advance in planCommand")

	return cmd, nil
}

// isAlphaNumeric checks if a byte is alphanumeric
func isAlphaNumeric(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')
}

// getTokenText returns the string representation of a token.
// For tokens with Text (identifiers, strings, numbers), returns the text.
// For operator/punctuation tokens with empty Text, reconstructs from token type.
func getTokenText(token lexer.Token) string {
	if len(token.Text) > 0 {
		return string(token.Text)
	}

	// Handle punctuation/operator tokens that have empty Text
	switch token.Type {
	case lexer.DECREMENT:
		return "--"
	case lexer.INCREMENT:
		return "++"
	case lexer.PLUS:
		return "+"
	case lexer.MINUS:
		return "-"
	case lexer.MULTIPLY:
		return "*"
	case lexer.DIVIDE:
		return "/"
	case lexer.MODULO:
		return "%"
	case lexer.EQUALS:
		return "="
	case lexer.EQ_EQ:
		return "=="
	case lexer.NOT_EQ:
		return "!="
	case lexer.LT:
		return "<"
	case lexer.LT_EQ:
		return "<="
	case lexer.GT:
		return ">"
	case lexer.GT_EQ:
		return ">="
	case lexer.AND_AND:
		return "&&"
	case lexer.OR_OR:
		return "||"
	case lexer.PIPE:
		return "|"
	case lexer.NOT:
		return "!"
	case lexer.COLON:
		return ":"
	case lexer.COMMA:
		return ","
	case lexer.SEMICOLON:
		return ";"
	case lexer.LPAREN:
		return "("
	case lexer.RPAREN:
		return ")"
	case lexer.LBRACE:
		return "{"
	case lexer.RBRACE:
		return "}"
	case lexer.LSQUARE:
		return "["
	case lexer.RSQUARE:
		return "]"
	case lexer.AT:
		return "@"
	case lexer.DOT:
		return "."
	case lexer.DOTDOTDOT:
		return "..."
	case lexer.ARROW:
		return "->"
	case lexer.APPEND:
		return ">>"
	case lexer.PLUS_ASSIGN:
		return "+="
	case lexer.MINUS_ASSIGN:
		return "-="
	case lexer.MULTIPLY_ASSIGN:
		return "*="
	case lexer.DIVIDE_ASSIGN:
		return "/="
	case lexer.MODULO_ASSIGN:
		return "%="
	default:
		return ""
	}
}

// parseVarValue parses a variable value expression (literal, object, or array)
func (p *planner) parseVarValue(varName string) (any, error) {
	if p.pos >= len(p.events) {
		return nil, &PlanError{
			Message:     "unexpected end of input",
			Context:     fmt.Sprintf("parsing variable '%s' value", varName),
			EventPos:    p.pos,
			TotalEvents: len(p.events),
		}
	}

	evt := p.events[p.pos]
	if evt.Kind != parser.EventOpen {
		return nil, &PlanError{
			Message:     "expected expression",
			Context:     fmt.Sprintf("parsing variable '%s' value", varName),
			EventPos:    p.pos,
			TotalEvents: len(p.events),
		}
	}

	nodeKind := parser.NodeKind(evt.Data)

	switch nodeKind {
	case parser.NodeLiteral:
		return p.parseLiteralValue(varName)
	case parser.NodeObjectLiteral:
		return p.parseObjectLiteral(varName)
	case parser.NodeArrayLiteral:
		return p.parseArrayLiteral(varName)
	case parser.NodeDecorator:
		return p.parseDecoratorValue(varName)
	default:
		return nil, &PlanError{
			Message:     fmt.Sprintf("unsupported expression type for variable value: %v", nodeKind),
			Context:     fmt.Sprintf("parsing variable '%s'", varName),
			EventPos:    p.pos,
			TotalEvents: len(p.events),
			Suggestion:  "Use a literal, object {}, or array []",
			Example:     `var config = {timeout: "5m", retries: 3}`,
		}
	}
}

// parseLiteralValue parses a simple literal value
func (p *planner) parseLiteralValue(varName string) (any, error) {
	p.pos++ // Move past OPEN Literal

	// Get literal value
	if p.pos >= len(p.events) || p.events[p.pos].Kind != parser.EventToken {
		return nil, &PlanError{
			Message:     "expected literal value",
			Context:     fmt.Sprintf("parsing variable '%s'", varName),
			EventPos:    p.pos,
			TotalEvents: len(p.events),
		}
	}

	valueTokenIdx := p.events[p.pos].Data
	valueToken := p.tokens[valueTokenIdx]

	// Parse literal value based on token type
	var value any
	switch valueToken.Type {
	case lexer.STRING:
		// Remove quotes from string literal
		value = strings.Trim(string(valueToken.Text), `"'`)
	case lexer.INTEGER, lexer.FLOAT, lexer.SCIENTIFIC:
		// Store as string for now (proper number parsing can be added later)
		value = string(valueToken.Text)
	case lexer.BOOLEAN:
		// Boolean literal
		value = string(valueToken.Text)
	case lexer.IDENTIFIER:
		// Handle identifiers (could be true/false if not recognized as BOOLEAN)
		value = string(valueToken.Text)
	default:
		value = string(valueToken.Text)
	}

	p.pos++ // Move past TOKEN(value)

	// Skip CLOSE Literal
	if p.pos < len(p.events) && p.events[p.pos].Kind == parser.EventClose {
		p.pos++
	}

	// Skip CLOSE VarDecl
	if p.pos < len(p.events) && p.events[p.pos].Kind == parser.EventClose {
		p.pos++
	}

	return value, nil
}

// parseObjectLiteral parses an object literal {key: value, ...}
func (p *planner) parseObjectLiteral(varName string) (any, error) {
	p.pos++ // Move past OPEN ObjectLiteral

	obj := make(map[string]any)
	depth := 1

	for p.pos < len(p.events) && depth > 0 {
		evt := p.events[p.pos]

		if evt.Kind == parser.EventOpen {
			nodeKind := parser.NodeKind(evt.Data)
			if nodeKind == parser.NodeObjectField {
				// Parse field: key: value
				p.pos++ // Move past OPEN ObjectField

				// Get key (should be TOKEN)
				if p.pos >= len(p.events) || p.events[p.pos].Kind != parser.EventToken {
					return nil, &PlanError{
						Message:     "expected object field key",
						Context:     fmt.Sprintf("parsing variable '%s' object", varName),
						EventPos:    p.pos,
						TotalEvents: len(p.events),
					}
				}
				keyTokenIdx := p.events[p.pos].Data
				key := string(p.tokens[keyTokenIdx].Text)
				p.pos++ // Move past key token

				// Skip colon token
				if p.pos < len(p.events) && p.events[p.pos].Kind == parser.EventToken {
					p.pos++
				}

				// Parse value (recursive - could be literal, object, or array)
				fieldValue, err := p.parseVarValue(fmt.Sprintf("%s.%s", varName, key))
				if err != nil {
					return nil, err
				}

				obj[key] = fieldValue

				// Skip CLOSE ObjectField
				if p.pos < len(p.events) && p.events[p.pos].Kind == parser.EventClose {
					p.pos++
				}
			} else {
				depth++
				p.pos++
			}
		} else if evt.Kind == parser.EventClose {
			depth--
			if depth == 0 {
				p.pos++ // Move past CLOSE ObjectLiteral
				break
			}
			p.pos++
		} else {
			p.pos++
		}
	}

	// Skip CLOSE VarDecl
	if p.pos < len(p.events) && p.events[p.pos].Kind == parser.EventClose {
		p.pos++
	}

	return obj, nil
}

// parseArrayLiteral parses an array literal [expr, expr, ...]
func (p *planner) parseArrayLiteral(varName string) (any, error) {
	p.pos++ // Move past OPEN ArrayLiteral

	arr := make([]any, 0)
	depth := 1
	elementIndex := 0

	for p.pos < len(p.events) && depth > 0 {
		evt := p.events[p.pos]

		if evt.Kind == parser.EventOpen {
			nodeKind := parser.NodeKind(evt.Data)

			// Check if this is an element expression
			if nodeKind == parser.NodeLiteral || nodeKind == parser.NodeObjectLiteral || nodeKind == parser.NodeArrayLiteral {
				// Parse element (recursive)
				elementValue, err := p.parseVarValue(fmt.Sprintf("%s[%d]", varName, elementIndex))
				if err != nil {
					return nil, err
				}
				arr = append(arr, elementValue)
				elementIndex++
			} else {
				depth++
				p.pos++
			}
		} else if evt.Kind == parser.EventClose {
			depth--
			if depth == 0 {
				p.pos++ // Move past CLOSE ArrayLiteral
				break
			}
			p.pos++
		} else {
			// Skip commas and other tokens
			p.pos++
		}
	}

	// Skip CLOSE VarDecl
	if p.pos < len(p.events) && p.events[p.pos].Kind == parser.EventClose {
		p.pos++
	}

	return arr, nil
}

// parseDecoratorValue resolves a decorator and returns its value.
// This is used for variable declarations like: var HOME = @env.HOME
func (p *planner) parseDecoratorValue(varName string) (any, error) {
	startPos := p.pos
	p.pos++ // Move past OPEN Decorator

	// Extract decorator name and property from tokens
	// Expected structure: TOKEN(@), TOKEN(decorator), TOKEN(.), TOKEN(property)
	var decoratorParts []string
	var primary *string

	for p.pos < len(p.events) {
		evt := p.events[p.pos]

		if evt.Kind == parser.EventClose {
			// End of decorator
			break
		}

		if evt.Kind != parser.EventToken {
			p.pos++
			continue
		}

		tokIdx := evt.Data
		if int(tokIdx) >= len(p.tokens) {
			return nil, &PlanError{
				Message:     "invalid token index in decorator",
				Context:     fmt.Sprintf("parsing variable '%s'", varName),
				EventPos:    p.pos,
				TotalEvents: len(p.events),
			}
		}

		tok := p.tokens[tokIdx]

		switch tok.Type {
		case lexer.AT:
			// Skip @ symbol
			p.pos++
		case lexer.IDENTIFIER:
			// Collect all identifiers separated by dots
			// The last identifier becomes the primary parameter if there's more than one segment
			// Examples:
			//   @env → path="env", primary=nil
			//   @env.HOME → path="env", primary="HOME"
			//   @aws.ssm.param → path="aws.ssm", primary="param"
			decoratorParts = append(decoratorParts, string(tok.Text))
			p.pos++
		case lexer.DOT:
			// Separator between decorator and property
			p.pos++
		case lexer.LPAREN:
			// Decorator has parameters - not yet supported
			return nil, &PlanError{
				Message: fmt.Sprintf("decorator @%s has parameters, which are not yet supported in variable declarations",
					strings.Join(decoratorParts, ".")),
				Context:     fmt.Sprintf("parsing variable '%s'", varName),
				EventPos:    p.pos,
				TotalEvents: len(p.events),
			}
		default:
			// Unknown token - should not happen in well-formed decorator
			return nil, &PlanError{
				Message:     fmt.Sprintf("unexpected token %s in decorator", tok.Type),
				Context:     fmt.Sprintf("parsing variable '%s'", varName),
				EventPos:    p.pos,
				TotalEvents: len(p.events),
			}
		}
	}

	// Consume EventClose
	if p.pos < len(p.events) && p.events[p.pos].Kind == parser.EventClose {
		p.pos++
	}

	if len(decoratorParts) == 0 {
		return nil, &PlanError{
			Message:     "empty decorator name",
			Context:     fmt.Sprintf("parsing variable '%s'", varName),
			EventPos:    startPos,
			TotalEvents: len(p.events),
		}
	}

	// Find the decorator by trying progressively shorter paths (most specific first).
	// Like URL routing: try longest match first, then progressively shorter.
	//
	// For @aws.s3.bucket.object.tag, try:
	//   1. "aws.s3.bucket.object.tag" (most specific, no primary)
	//   2. "aws.s3.bucket.object" with primary="tag"
	//   3. "aws.s3.bucket" with primary="object" (ERROR: 2 remaining segments)
	//   4. "aws.s3" with primary="bucket" (ERROR: 3 remaining segments)
	//   5. "aws" with primary="s3" (ERROR: 4 remaining segments)
	//
	// For @env.HOME, try:
	//   1. "env.HOME" (full path, no primary)
	//   2. "env" with primary="HOME" ✓
	//
	// Only ONE segment after the decorator path is allowed as primary parameter.
	var decoratorName string
	for splitPoint := len(decoratorParts); splitPoint > 0; splitPoint-- {
		candidatePath := strings.Join(decoratorParts[:splitPoint], ".")
		_, found := decorator.Global().Lookup(candidatePath)
		if found {
			remainingSegments := len(decoratorParts) - splitPoint
			if remainingSegments > 1 {
				// Too many segments after decorator name
				return nil, &PlanError{
					Message: fmt.Sprintf("decorator @%s: found registered decorator %q but %d segments remain (%s); only 1 primary parameter allowed",
						strings.Join(decoratorParts, "."), candidatePath, remainingSegments,
						strings.Join(decoratorParts[splitPoint:], ".")),
					Context:     fmt.Sprintf("parsing variable '%s'", varName),
					EventPos:    startPos,
					TotalEvents: len(p.events),
				}
			}
			decoratorName = candidatePath
			if remainingSegments == 1 {
				// Exactly one segment remains - use as primary parameter
				lastPart := decoratorParts[splitPoint]
				primary = &lastPart
			}
			break
		}
	}

	if decoratorName == "" {
		return nil, &PlanError{
			Message:     fmt.Sprintf("decorator @%s not found in registry", strings.Join(decoratorParts, ".")),
			Context:     fmt.Sprintf("parsing variable '%s'", varName),
			EventPos:    startPos,
			TotalEvents: len(p.events),
		}
	}

	// Build ValueCall for decorator resolution
	call := decorator.ValueCall{
		Path:    decoratorName,
		Primary: primary,
		Params:  make(map[string]any),
	}

	// Create evaluation context
	ctx := decorator.ValueEvalContext{
		Session: p.session,
		Vault:   p.vault, // Scope-aware variable storage
	}

	// Get transport scope from current session to enforce transport-scope guards
	currentScope := p.session.TransportScope()

	// Resolve decorator using global registry
	result, err := decorator.ResolveValue(ctx, call, currentScope)
	if err != nil {
		return nil, &PlanError{
			Message:     fmt.Sprintf("failed to resolve @%s: %v", decoratorName, err),
			Context:     fmt.Sprintf("parsing variable '%s'", varName),
			EventPos:    startPos,
			TotalEvents: len(p.events),
		}
	}

	return result.Value, nil
}

// buildCommandIR tokenizes a command string into CommandIR with captured exprIDs.
//
// CommandIR solves the variable shadowing problem by capturing exprIDs at parse time.
// Without this, Pass 3 would do fresh variable lookups and find the wrong values:
//
//	var COUNT = "5"
//	echo "@var.COUNT"   # Should use "5"
//	var COUNT = "10"
//	echo "@var.COUNT"   # Should use "10"
//
// If Pass 3 did fresh lookups, both echoes would find COUNT="10" (latest declaration).
// CommandIR preserves temporal binding: each @var.COUNT captures the exprID that was
// in scope when that specific command was parsed.
//
// This is Pass 2 - builds IR, captures exprIDs, validates (hoisting check), marks touched.
func (p *planner) buildCommandIR(command string) (*CommandIR, error) {
	if p.config.Debug >= DebugDetailed {
		p.recordDebugEvent("buildCommandIR", fmt.Sprintf("command=%s", command))
	}

	ir := &CommandIR{
		Parts: make([]CommandPart, 0),
	}

	// Parse command into parts (literals and variable references)
	i := 0
	for i < len(command) {
		// Find next @var.
		idx := strings.Index(command[i:], "@var.")
		if idx == -1 {
			// No more @var patterns - rest is literal
			if i < len(command) {
				ir.Parts = append(ir.Parts, CommandPart{
					Kind: PartLiteral,
					Text: command[i:],
				})
			}
			break
		}

		// Absolute position in command
		pos := i + idx

		// Add literal part before @var (if any)
		if pos > i {
			ir.Parts = append(ir.Parts, CommandPart{
				Kind: PartLiteral,
				Text: command[i:pos],
			})
		}

		// Extract variable name after @var.
		varStart := pos + 5 // len("@var.")
		varEnd := varStart
		for varEnd < len(command) && (isAlphaNumeric(command[varEnd]) || command[varEnd] == '_') {
			varEnd++
		}

		if varEnd == varStart {
			return nil, &PlanError{
				Message: "invalid variable name in decorator",
				Context: fmt.Sprintf("parsing @var at position %d", pos),
			}
		}

		varName := command[varStart:varEnd]

		// Lookup variable in Vault (captures exprID at this point in time)
		// This is where hoisting validation happens - if variable not declared yet, error
		exprID, err := p.vault.LookupVariable(varName)
		if err != nil {
			return nil, fmt.Errorf("variable %q not found: %w", varName, err)
		}

		// DEBUG: Log captured exprID
		if p.config.Debug >= DebugDetailed {
			p.recordDebugEvent("buildCommandIR", fmt.Sprintf("captured exprID=%s for var=%s in command=%s", exprID, varName, command))
		}

		// Record reference (authorize this site)
		if err := p.vault.RecordReference(exprID, "command"); err != nil {
			return nil, err
		}

		// Mark as touched (in execution path)
		p.vault.MarkTouched(exprID)

		// Add variable reference part with CAPTURED exprID
		ir.Parts = append(ir.Parts, CommandPart{
			Kind:   PartVarRef,
			ExprID: exprID, // Temporal binding preserved!
		})

		// Move past this @var
		i = varEnd
	}

	return ir, nil
}

// interpolateCommandIR converts CommandIR to final string with DisplayIDs.
//
// Uses captured exprIDs from Pass 2 instead of doing fresh variable lookups.
// This preserves temporal binding and prevents shadowing bugs (see buildCommandIR).
//
// This is Pass 3 - final interpolation after all expressions are resolved.
func (p *planner) interpolateCommandIR(ir *CommandIR) string {
	var result strings.Builder
	for _, part := range ir.Parts {
		switch part.Kind {
		case PartLiteral:
			result.WriteString(part.Text)
		case PartVarRef:
			// Use captured exprID - NO lookup needed!
			displayID := p.vault.GetDisplayID(part.ExprID)
			result.WriteString(displayID)
		}
	}
	return result.String()
}

// interpolateAllCommands walks all steps and interpolates commands (Pass 3).
// Replaces all @var.X patterns with DisplayIDs after resolution completes.
func (p *planner) interpolateAllCommands(steps []planfmt.Step) error {
	for i := range steps {
		if err := p.interpolateStepTree(&steps[i].Tree); err != nil {
			return err
		}
	}
	return nil
}

// interpolateStepTree recursively walks an execution tree and interpolates all command strings.
//
//nolint:gocritic // node must be pointer - we modify n.Args[].Val.Str
func (p *planner) interpolateStepTree(node *planfmt.ExecutionNode) error {
	if node == nil || *node == nil {
		return nil
	}

	switch n := (*node).(type) {
	case *planfmt.CommandNode:
		// Interpolate the command argument using CommandIR
		var commandID uint64
		commandArgIdx := -1

		// Find command arg and commandID
		for i := range n.Args {
			if n.Args[i].Key == "command" {
				commandArgIdx = i
			}
			if n.Args[i].Key == "__commandID" && n.Args[i].Val.Kind == planfmt.ValueInt {
				commandID = uint64(n.Args[i].Val.Int)
			}
		}

		// If we have a commandID, look up CommandIR and interpolate
		if commandID > 0 && commandArgIdx >= 0 {
			ir, ok := p.commandIRs[commandID]
			if ok {
				// Interpolate using CommandIR (uses captured exprIDs, no lookup)
				interpolated := p.interpolateCommandIR(ir)
				n.Args[commandArgIdx].Val.Str = interpolated
			}
		}
		// Recursively interpolate nested blocks
		for i := range n.Block {
			if err := p.interpolateStepTree(&n.Block[i].Tree); err != nil {
				return err
			}
		}

	case *planfmt.PipelineNode:
		for i := range n.Commands {
			if err := p.interpolateStepTree(&n.Commands[i]); err != nil {
				return err
			}
		}

	case *planfmt.AndNode:
		if err := p.interpolateStepTree(&n.Left); err != nil {
			return err
		}
		if err := p.interpolateStepTree(&n.Right); err != nil {
			return err
		}

	case *planfmt.OrNode:
		if err := p.interpolateStepTree(&n.Left); err != nil {
			return err
		}
		if err := p.interpolateStepTree(&n.Right); err != nil {
			return err
		}

	case *planfmt.SequenceNode:
		for i := range n.Nodes {
			if err := p.interpolateStepTree(&n.Nodes[i]); err != nil {
				return err
			}
		}

	case *planfmt.RedirectNode:
		if err := p.interpolateStepTree(&n.Source); err != nil {
			return err
		}
		// Target is a CommandNode, convert to ExecutionNode
		var targetNode planfmt.ExecutionNode = &n.Target
		if err := p.interpolateStepTree(&targetNode); err != nil {
			return err
		}
	}

	return nil
}
