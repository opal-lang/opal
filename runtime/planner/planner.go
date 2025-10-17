// Package planner converts parser events into execution plans.
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
	"fmt"
	"strings"
	"time"

	"github.com/aledsdavies/opal/core/invariant"
	"github.com/aledsdavies/opal/core/planfmt"
	"github.com/aledsdavies/opal/core/sdk/secret"
	"github.com/aledsdavies/opal/runtime/lexer"
	"github.com/aledsdavies/opal/runtime/parser"
	"github.com/lithammer/fuzzysearch/fuzzy"
)

// Config configures the planner
type Config struct {
	Target    string           // Command name (e.g., "hello") or "" for script mode
	IDFactory secret.IDFactory // Factory for generating deterministic secret IDs (optional, uses run-mode if nil)
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
		telemetry = &PlanTelemetry{}
	}

	// Initialize debug events if enabled
	if config.Debug >= DebugPaths {
		debugEvents = make([]DebugEvent, 0, 100)
	}

	p := &planner{
		events:      events,
		tokens:      tokens,
		config:      config,
		pos:         0,
		stepID:      1,
		telemetry:   telemetry,
		debugEvents: debugEvents,
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

// planner holds state during planning
type planner struct {
	events []parser.Event
	tokens []lexer.Token
	config Config

	pos    int    // Current position in event stream
	stepID uint64 // Next step ID to assign

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

// plan is the main planning entry point
func (p *planner) plan() (*planfmt.Plan, error) {
	if p.config.Debug >= DebugPaths {
		p.recordDebugEvent("enter_plan", "target="+p.config.Target)
	}

	plan := &planfmt.Plan{
		Header: planfmt.PlanHeader{
			PlanKind: 0, // View plan
		},
		Target: p.config.Target,
		Steps:  []planfmt.Step{},
	}

	// Command mode: find target function
	if p.config.Target != "" {
		steps, err := p.planTargetFunction()
		if err != nil {
			return nil, err
		}
		plan.Steps = steps
	} else {
		// Script mode: plan all top-level commands
		steps, err := p.planSource()
		if err != nil {
			return nil, err
		}
		plan.Steps = steps
	}

	// POSTCONDITION: plan must be valid
	err := plan.Validate()
	invariant.ExpectNoError(err, "plan validation")

	if p.config.Debug >= DebugPaths {
		p.recordDebugEvent("exit_plan", fmt.Sprintf("steps=%d", len(plan.Steps)))
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
			// Found a step boundary - plan the entire step
			step, err := p.planStep()
			if err != nil {
				return nil, err
			}
			steps = append(steps, step)
			// planStep already advanced p.pos past EventStepExit, so continue
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
			// Found a top-level step boundary (depth 1 = inside Source, not inside Function)
			step, err := p.planStep()
			if err != nil {
				return nil, err
			}
			steps = append(steps, step)
			// planStep already advanced p.pos past EventStepExit, so continue
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

	var commands []planfmt.Command

	// Collect all shell commands until EventStepExit
	for p.pos < len(p.events) {
		evt := p.events[p.pos]

		if evt.Kind == parser.EventStepExit {
			// End of step
			p.pos++ // Move past EventStepExit
			break
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
		return planfmt.Step{}, &PlanError{
			Message:     "step has no commands",
			Context:     "planning step",
			EventPos:    p.pos,
			TotalEvents: len(p.events),
		}
	}

	step := planfmt.Step{
		ID:       p.nextStepID(),
		Commands: commands,
	}

	if p.config.Debug >= DebugDetailed {
		p.recordDebugEvent("step_created", fmt.Sprintf("id=%d commands=%d", step.ID, len(commands)))
	}

	return step, nil
}

// planCommand plans a single command within a step (shell command + optional operator)
func (p *planner) planCommand() (planfmt.Command, error) {
	if p.config.Debug >= DebugDetailed {
		p.recordDebugEvent("enter_planCommand", fmt.Sprintf("pos=%d", p.pos))
	}

	startPos := p.pos
	p.pos++ // Move past OPEN ShellCommand

	// Collect all tokens in the shell command
	var commandTokens []string
	depth := 1

	for p.pos < len(p.events) && depth > 0 {
		evt := p.events[p.pos]

		if evt.Kind == parser.EventOpen {
			depth++
		} else if evt.Kind == parser.EventClose {
			depth--
			if depth == 0 {
				// Move past the CLOSE ShellCommand event
				p.pos++
				break
			}
		} else if evt.Kind == parser.EventToken {
			tokenIdx := evt.Data
			tokenText := string(p.tokens[tokenIdx].Text)
			commandTokens = append(commandTokens, tokenText)
		}

		p.pos++
	}

	// Build command string
	command := ""
	for i, tok := range commandTokens {
		if i > 0 {
			command += " "
		}
		command += tok
	}

	// POSTCONDITION: command must not be empty
	invariant.Postcondition(command != "", "shell command must not be empty")

	// Check for operator after this command (&&, ||, |, ;)
	operator := ""
	if p.pos < len(p.events) {
		evt := p.events[p.pos]
		if evt.Kind == parser.EventToken {
			tokenIdx := evt.Data
			tokenType := p.tokens[tokenIdx].Type
			// Check if it's an operator (use token type, not text)
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

	cmd := planfmt.Command{
		Decorator: "@shell",
		Args: []planfmt.Arg{
			{
				Key: "command",
				Val: planfmt.Value{
					Kind: planfmt.ValueString,
					Str:  command,
				},
			},
		},
		Operator: operator,
	}

	if p.config.Debug >= DebugDetailed {
		p.recordDebugEvent("command_created", fmt.Sprintf("decorator=@shell command=%q operator=%q", command, operator))
	}

	// POSTCONDITION: position must advance
	invariant.Postcondition(p.pos > startPos, "position must advance in planCommand")

	return cmd, nil
}
