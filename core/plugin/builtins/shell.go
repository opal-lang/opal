package builtins

import (
	"fmt"
	"io"
	"time"

	"github.com/builtwithtofu/sigil/core/plugin"
	"github.com/builtwithtofu/sigil/core/types"
)

// ShellPlugin exposes shell execution and redirect capabilities.
type ShellPlugin struct{}

func (p *ShellPlugin) Identity() plugin.PluginIdentity {
	return plugin.PluginIdentity{Name: "shell", Version: "1.0.0", APIVersion: 1}
}

func (p *ShellPlugin) Capabilities() []plugin.Capability {
	return []plugin.Capability{ShellWrapperCapability{}}
}

// ShellWrapperCapability executes shell commands and supports redirect target I/O.
type ShellWrapperCapability struct{}

func (c ShellWrapperCapability) Path() string { return "shell" }

func (c ShellWrapperCapability) Schema() plugin.Schema {
	return plugin.Schema{
		Params: []plugin.Param{
			{Name: "command", Type: types.TypeString, Required: true},
			{Name: "shell", Type: types.TypeString, Enum: []string{"bash", "pwsh", "cmd"}},
		},
		Block: plugin.BlockForbidden,
	}
}

func (c ShellWrapperCapability) Wrap(next plugin.ExecNode, args plugin.ResolvedArgs) plugin.ExecNode {
	return shellNode{command: args.GetString("command"), shell: args.GetStringOptional("shell")}
}

func (c ShellWrapperCapability) RedirectCaps() plugin.RedirectCaps {
	return plugin.RedirectCaps{Read: true, Write: true, Append: true, Atomic: false}
}

func (c ShellWrapperCapability) OpenForRead(ctx plugin.ExecContext, args plugin.ResolvedArgs) (io.ReadCloser, error) {
	delegate := FileRedirectCapability{}
	return delegate.OpenForRead(ctx, shellRedirectArgs{args: args})
}

func (c ShellWrapperCapability) OpenForWrite(ctx plugin.ExecContext, args plugin.ResolvedArgs, appendMode bool) (io.WriteCloser, error) {
	delegate := FileRedirectCapability{}
	return delegate.OpenForWrite(ctx, shellRedirectArgs{args: args}, appendMode)
}

type shellNode struct {
	command string
	shell   string
}

func (n shellNode) Execute(ctx plugin.ExecContext) (plugin.Result, error) {
	if n.command == "" {
		return plugin.Result{ExitCode: plugin.ExitFailure}, fmt.Errorf("@shell requires command parameter")
	}

	shellName, err := resolvePluginShellName(n.shell, ctx.Session())
	if err != nil {
		return plugin.Result{ExitCode: plugin.ExitFailure}, err
	}

	argv, err := pluginShellCommandArgs(shellName, n.command)
	if err != nil {
		return plugin.Result{ExitCode: plugin.ExitFailure}, err
	}

	result, err := ctx.Session().Run(readCtx(ctx), argv, plugin.RunOpts{
		Stdin:  ctx.Stdin(),
		Stdout: ctx.Stdout(),
		Stderr: ctx.Stderr(),
		Dir:    ctx.Session().Snapshot().Workdir,
	})
	return result, err
}

func resolvePluginShellName(explicit string, session plugin.ParentTransport) (string, error) {
	if explicit != "" {
		return explicit, nil
	}

	if session != nil {
		envShell := session.Snapshot().Env["OPAL_SHELL"]
		if envShell != "" {
			if _, err := pluginShellCommandArgs(envShell, ""); err != nil {
				return "", fmt.Errorf("invalid OPAL_SHELL %q: expected one of bash, pwsh, cmd", envShell)
			}
			return envShell, nil
		}
	}

	return "bash", nil
}

func pluginShellCommandArgs(shellName, command string) ([]string, error) {
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

type shellRedirectArgs struct{ args plugin.ResolvedArgs }

func (a shellRedirectArgs) GetString(name string) string {
	if name == "path" {
		return a.args.GetString("command")
	}
	return a.args.GetString(name)
}

func (a shellRedirectArgs) GetStringOptional(name string) string {
	if name == "path" {
		return a.args.GetStringOptional("command")
	}
	return a.args.GetStringOptional(name)
}

func (a shellRedirectArgs) GetInt(name string) int {
	return a.args.GetInt(name)
}

func (a shellRedirectArgs) GetDuration(name string) time.Duration {
	return a.args.GetDuration(name)
}

func (a shellRedirectArgs) ResolveSecret(name string) (string, error) {
	return a.args.ResolveSecret(name)
}
