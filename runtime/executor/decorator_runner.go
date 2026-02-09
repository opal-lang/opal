package executor

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/opal-lang/opal/core/decorator"
	"github.com/opal-lang/opal/core/invariant"
	"github.com/opal-lang/opal/core/sdk"
)

var displayIDPattern = regexp.MustCompile(`opal:[A-Za-z0-9_-]{22}`)

// executeCommandWithPipes executes a command with optional piped stdin/stdout.
func (e *executor) executeCommandWithPipes(execCtx sdk.ExecutionContext, cmd *sdk.CommandNode, stdin io.Reader, stdout io.Writer) int {
	invariant.NotNil(execCtx, "execCtx")
	commandExecCtx := withExecutionTransport(execCtx, cmd.TransportID)

	decoratorName := strings.TrimPrefix(cmd.Name, "@")
	entry, exists := decorator.Global().Lookup(decoratorName)
	invariant.Invariant(exists, "unknown decorator: %s", cmd.Name)

	execDec, ok := entry.Impl.(decorator.Exec)
	invariant.Invariant(ok, "%s is not an execution decorator", cmd.Name)

	return e.executeDecorator(commandExecCtx, cmd, execDec, stdin, stdout)
}

func withExecutionTransport(execCtx sdk.ExecutionContext, transportID string) sdk.ExecutionContext {
	ec, ok := execCtx.(*executionContext)
	if !ok {
		return execCtx
	}

	return ec.withTransportID(normalizedTransportID(transportID))
}

// resolveDisplayIDs scans params for DisplayID strings and resolves them to actual values.
func (e *executor) resolveDisplayIDs(params map[string]any, decoratorName, transportID string) (map[string]any, error) {
	resolved := make(map[string]any)

	for key, val := range params {
		strVal, ok := val.(string)
		if !ok {
			resolved[key] = val
			continue
		}

		matches := displayIDPattern.FindAllString(strVal, -1)
		if len(matches) == 0 {
			resolved[key] = val
			continue
		}

		result := strVal
		for _, displayID := range matches {
			actualValue, err := e.vault.ResolveDisplayIDWithTransport(displayID, normalizedTransportID(transportID))
			if err != nil {
				return nil, fmt.Errorf("failed to resolve %s in %s.%s: %w", displayID, decoratorName, key, err)
			}
			result = strings.ReplaceAll(result, displayID, fmt.Sprint(actualValue))
		}

		resolved[key] = result
	}

	return resolved, nil
}

// executeDecorator executes a decorator via the Exec interface.
func (e *executor) executeDecorator(
	execCtx sdk.ExecutionContext,
	cmd *sdk.CommandNode,
	execDec decorator.Exec,
	stdin io.Reader,
	stdout io.Writer,
) int {
	params := make(map[string]any)
	for k, v := range cmd.Args {
		params[k] = v
	}

	if e.vault != nil {
		var err error
		params, err = e.resolveDisplayIDs(params, cmd.Name, cmd.TransportID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resolving secrets: %v\n", err)
			return 1
		}
	}

	var next decorator.ExecNode
	if len(cmd.Block) > 0 {
		next = &blockNode{execCtx: execCtx, steps: cmd.Block}
	}

	node := execDec.Wrap(next, params)
	if node == nil {
		if next == nil {
			return 0
		}
		node = next
	}

	baseSession, sessionErr := e.sessions.SessionFor(cmd.TransportID)
	if sessionErr != nil {
		fmt.Fprintf(os.Stderr, "Error creating session: %v\n", sessionErr)
		return 1
	}
	session := sessionForExecutionContext(baseSession, execCtx)

	if stdout == nil {
		stdout = os.Stdout
	}

	decoratorExecCtx := decorator.ExecContext{
		Context: execCtx.Context(),
		Session: session,
		Stdin:   stdin,
		Stdout:  stdout,
		Stderr:  os.Stderr,
		Trace:   nil,
	}

	result, err := node.Execute(decoratorExecCtx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}

	return result.ExitCode
}

func sessionForExecutionContext(base decorator.Session, execCtx sdk.ExecutionContext) decorator.Session {
	session := base

	if workdir := execCtx.Workdir(); workdir != "" && workdir != session.Cwd() {
		session = session.WithWorkdir(workdir)
	}

	if delta := envDelta(session.Env(), execCtx.Environ()); len(delta) > 0 {
		session = session.WithEnv(delta)
	}

	return session
}

func envDelta(base, target map[string]string) map[string]string {
	delta := make(map[string]string)
	for key, targetValue := range target {
		if baseValue, ok := base[key]; !ok || baseValue != targetValue {
			delta[key] = targetValue
		}
	}
	return delta
}
