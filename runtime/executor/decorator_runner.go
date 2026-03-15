package executor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/builtwithtofu/sigil/core/invariant"
	coreplugin "github.com/builtwithtofu/sigil/core/plugin"
	coreruntime "github.com/builtwithtofu/sigil/core/runtime"
	"github.com/builtwithtofu/sigil/core/sdk"
)

var displayIDPattern = regexp.MustCompile(`sigil:[A-Za-z0-9_-]{22}`)

func (e *executor) getStderr() io.Writer {
	if e.stderr == nil {
		return os.Stderr
	}
	return e.stderr
}

// executeCommandWithPipes executes a command with optional piped stdin/stdout.
func (e *executor) executeCommandWithPipes(execCtx sdk.ExecutionContext, cmd *sdk.CommandNode, stdin io.Reader, stdout io.Writer) int {
	invariant.NotNil(execCtx, "execCtx")
	commandExecCtx := withExecutionTransport(execCtx, cmd.TransportID)
	if isShellDecorator(cmd.Name) {
		return e.executeShellCommandWithPipes(commandExecCtx, cmd, stdin, stdout)
	}

	decoratorName := strings.TrimPrefix(cmd.Name, "@")
	if capability := isPluginOnlyCapability(decoratorName); capability != nil {
		params, ok := e.resolvedCommandParams(commandExecCtx, cmd)
		if !ok {
			return coreruntime.ExitFailure
		}
		var next coreruntime.ExecNode
		if len(cmd.Block) > 0 {
			next = &blockNode{execCtx: commandExecCtx, steps: cmd.Block}
		}
		switch typed := capability.(type) {
		case coreplugin.Wrapper:
			return e.executePluginWrapper(commandExecCtx, next, typed, params, stdin, stdout)
		case coreplugin.Transport:
			return e.executePluginTransport(commandExecCtx, cmd.Block, typed, params)
		default:
			_, _ = fmt.Fprintf(e.getStderr(), "Error: @%s is not executable\n", decoratorName)
			return coreruntime.ExitFailure
		}
	}

	_, _ = fmt.Fprintf(e.getStderr(), "Error: unknown decorator: %s\n", cmd.Name)
	return coreruntime.ExitFailure
}

func isShellDecorator(name string) bool {
	return strings.TrimPrefix(name, "@") == "shell"
}

func withExecutionTransport(execCtx sdk.ExecutionContext, transportID string) sdk.ExecutionContext {
	ec, ok := execCtx.(*executionContext)
	if !ok {
		return execCtx
	}
	if normalizedTransportID(transportID) == "local" && transportID == "" {
		return execCtx
	}

	requestedTransportID := normalizedTransportID(transportID)
	currentTransportID := normalizedTransportID(ec.transportID)
	if requestedTransportID == currentTransportID {
		return execCtx
	}

	return ec.withTransportID(requestedTransportID)
}

func executionTransportID(execCtx sdk.ExecutionContext) string {
	ec, ok := execCtx.(*executionContext)
	if !ok {
		return "local"
	}
	return normalizedTransportID(ec.transportID)
}

func (e *executor) executeShellCommandWithPipes(execCtx sdk.ExecutionContext, cmd *sdk.CommandNode, stdin io.Reader, stdout io.Writer) int {
	params, ok := e.resolvedCommandParams(execCtx, cmd)
	if !ok {
		return coreruntime.ExitFailure
	}

	return e.executeShellWithParams(execCtx, params, stdin, stdout)
}

func (e *executor) executeShellWithParams(execCtx sdk.ExecutionContext, params map[string]any, stdin io.Reader, stdout io.Writer) int {
	if params == nil {
		_, _ = fmt.Fprintln(e.getStderr(), "Error: @shell missing parameters")
		return coreruntime.ExitFailure
	}

	command, ok := params["command"].(string)
	if !ok || command == "" {
		_, _ = fmt.Fprintln(e.getStderr(), "Error: @shell requires a non-empty string command")
		return coreruntime.ExitFailure
	}

	if displayIDPattern.MatchString(command) {
		panic(fmt.Sprintf("INVARIANT VIOLATION: Command contains unresolved DisplayID: %s\n"+
			"DisplayIDs must be resolved to actual values before execution.\n"+
			"Format: sigil:<base64url-hash> (22 chars)\n"+
			"This indicates the executor is not resolving secrets from the plan.", command))
	}

	explicitShell := ""
	if shellArg, hasShellArg := params["shell"]; hasShellArg {
		shellStr, ok := shellArg.(string)
		if !ok {
			_, _ = fmt.Fprintln(e.getStderr(), "Error: @shell expects 'shell' to be a string when provided")
			return coreruntime.ExitFailure
		}
		explicitShell = shellStr
	}

	transportID := executionTransportID(execCtx)
	shellName, err := resolveShellName(explicitShell, execCtx.Environ())
	if err != nil {
		_, _ = fmt.Fprintf(e.getStderr(), "Error: %v\n", err)
		return coreruntime.ExitFailure
	}

	runCtx := execCtx.Context()
	if runCtx == nil {
		runCtx = context.Background()
	}

	if e.workers != nil && canUseShellWorker(shellName, stdin, stdout) && e.workers.shouldUseWorker(transportID, shellName) {
		exitCode, workerErr := e.workers.Run(runCtx, shellRunRequest{
			transportID: transportID,
			shellName:   shellName,
			command:     command,
			environ:     execCtx.Environ(),
			workdir:     execCtx.Workdir(),
			stdout:      stdout,
			stderr:      e.stderr,
		})
		if workerErr == nil {
			return exitCode
		}

		if errors.Is(workerErr, context.Canceled) || errors.Is(workerErr, context.DeadlineExceeded) {
			return coreruntime.ExitCanceled
		}

		if !canFallbackToSessionRun(workerErr) {
			_, _ = fmt.Fprintf(e.getStderr(), "Error: shell worker execution failed after command start: %v\n", workerErr)
			return coreruntime.ExitFailure
		}

		_, _ = fmt.Fprintf(e.getStderr(), "Warning: shell worker unavailable before command start, falling back to session run: %v\n", workerErr)
	}

	baseSession, sessionErr := e.sessions.SessionFor(transportID)
	if sessionErr != nil {
		_, _ = fmt.Fprintf(e.getStderr(), "Error creating session: %v\n", sessionErr)
		return coreruntime.ExitFailure
	}

	session := sessionForExecutionContext(baseSession, execCtx)

	argv, err := shellCommandArgs(shellName, command)
	if err != nil {
		_, _ = fmt.Fprintf(e.getStderr(), "Error: %v\n", err)
		return coreruntime.ExitFailure
	}

	if stdout == nil {
		stdout = os.Stdout
	}

	result, err := session.Run(runCtx, argv, coreruntime.RunOpts{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: e.stderr,
	})
	if err != nil {
		_, _ = fmt.Fprintf(e.getStderr(), "Error: %v\n", err)
	}

	return result.ExitCode
}

func canUseShellWorker(shellName string, stdin io.Reader, stdout io.Writer) bool {
	if shellName != "bash" {
		return false
	}
	if stdin != nil {
		return false
	}
	return !isStreamPipeWriter(stdout)
}

func isStreamPipeWriter(stdout io.Writer) bool {
	if stdout == nil || stdout == os.Stdout || stdout == os.Stderr {
		return false
	}

	file, ok := stdout.(*os.File)
	if !ok {
		return false
	}

	info, err := file.Stat()
	if err != nil {
		return false
	}

	return info.Mode()&os.ModeNamedPipe != 0
}

func canFallbackToSessionRun(workerErr error) bool {
	var runErr *workerRunError
	if !errors.As(workerErr, &runErr) {
		return false
	}
	return !runErr.commandStarted
}

func resolveShellName(explicit string, env map[string]string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}

	if envShell := env["OPAL_SHELL"]; envShell != "" {
		if _, err := shellCommandArgs(envShell, ""); err != nil {
			return "", fmt.Errorf("invalid OPAL_SHELL %q: expected one of bash, pwsh, cmd", envShell)
		}
		return envShell, nil
	}

	return "bash", nil
}

func shellCommandArgs(shellName, command string) ([]string, error) {
	switch shellName {
	case "bash":
		return []string{"bash", "-c", command}, nil
	case "pwsh":
		return []string{"pwsh", "-NoProfile", "-NonInteractive", "-Command", command}, nil
	case "cmd":
		return []string{"cmd", "/C", command}, nil
	default:
		return nil, fmt.Errorf("unsupported shell %q: expected one of bash, pwsh, cmd", shellName)
	}
}

func (e *executor) resolveDisplayIDsExcept(params map[string]any, decoratorName, transportID string, skip map[string]struct{}) (map[string]any, error) {
	resolved := make(map[string]any)

	for key, val := range params {
		if _, ok := skip[key]; ok {
			resolved[key] = val
			continue
		}

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

func (e *executor) resolvedCommandParams(execCtx sdk.ExecutionContext, cmd *sdk.CommandNode) (map[string]any, bool) {
	return e.resolveCommandParams(execCtx, cmd.Name, cmd.Args)
}

func (e *executor) resolveCommandParams(execCtx sdk.ExecutionContext, decoratorName string, raw map[string]any) (map[string]any, bool) {
	params := make(map[string]any, len(raw))
	for k, v := range raw {
		params[k] = v
	}

	if e.vault == nil {
		return params, true
	}

	var skip map[string]struct{}
	if capability := isPluginOnlyCapability(strings.TrimPrefix(decoratorName, "@")); capability != nil {
		for _, name := range capability.Schema().Secrets {
			if skip == nil {
				skip = make(map[string]struct{})
			}
			skip[name] = struct{}{}
		}
	}

	resolved, err := e.resolveDisplayIDsExcept(params, decoratorName, executionTransportID(execCtx), skip)
	if err != nil {
		_, _ = fmt.Fprintf(e.getStderr(), "Error resolving secrets: %v\n", err)
		return nil, false
	}

	return resolved, true
}

func sessionForExecutionContext(base coreruntime.Session, execCtx sdk.ExecutionContext) coreruntime.Session {
	session := base
	ec, typed := execCtx.(*executionContext)

	workdir := execCtx.Workdir()
	baseWorkdir := ""
	if typed {
		baseWorkdir = ec.baseWorkdir
	}
	if baseWorkdir == "" {
		baseWorkdir = session.Cwd()
	}
	if workdir != "" && workdir != baseWorkdir {
		session = session.WithWorkdir(workdir)
	}

	baseEnv := map[string]string(nil)
	if typed {
		baseEnv = ec.baseEnviron
	}
	if baseEnv == nil {
		baseEnv = session.Env()
	}
	if delta := envDelta(baseEnv, execCtx.Environ()); len(delta) > 0 {
		session = session.WithEnv(delta)
	}

	return session
}

func envDelta(base, target map[string]string) map[string]string {
	var delta map[string]string
	for key, targetValue := range target {
		if baseValue, ok := base[key]; !ok || baseValue != targetValue {
			if delta == nil {
				delta = make(map[string]string)
			}
			delta[key] = targetValue
		}
	}
	return delta
}
