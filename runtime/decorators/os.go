package decorators

import (
	"fmt"

	"github.com/builtwithtofu/sigil/core/decorator"
	"github.com/builtwithtofu/sigil/core/types"
)

type osGetDecorator struct{}

func (d *osGetDecorator) Descriptor() decorator.Descriptor {
	return decorator.NewDescriptor("os.Get").
		Summary("Get the current operating system name").
		Roles(decorator.RoleProvider).
		Returns(types.TypeString, "Current OS name").
		TransportScope(decorator.TransportScopeAny).
		Idempotent().
		Block(decorator.BlockForbidden).
		Build()
}

func (d *osGetDecorator) Resolve(ctx decorator.ValueEvalContext, calls ...decorator.ValueCall) ([]decorator.ResolveResult, error) {
	results := make([]decorator.ResolveResult, len(calls))
	platform := ""
	if ctx.Session != nil {
		platform = ctx.Session.Platform()
	}
	for i := range calls {
		results[i] = decorator.ResolveResult{
			Value:  platform,
			Origin: "@os.Get",
		}
	}
	return results, nil
}

type osLinuxDecorator struct{}

func (d *osLinuxDecorator) Descriptor() decorator.Descriptor {
	return decorator.NewDescriptor("os.Linux").
		Summary("Whether current OS is Linux").
		Roles(decorator.RoleProvider).
		Returns(types.TypeString, "\"true\" on Linux, \"false\" otherwise").
		TransportScope(decorator.TransportScopeAny).
		Idempotent().
		Block(decorator.BlockForbidden).
		Build()
}

func (d *osLinuxDecorator) Resolve(ctx decorator.ValueEvalContext, calls ...decorator.ValueCall) ([]decorator.ResolveResult, error) {
	results := make([]decorator.ResolveResult, len(calls))
	platform := ""
	if ctx.Session != nil {
		platform = ctx.Session.Platform()
	}
	value := "false"
	if platform == "linux" {
		value = "true"
	}

	for i := range calls {
		results[i] = decorator.ResolveResult{
			Value:  value,
			Origin: "@os.Linux",
		}
	}
	return results, nil
}

type osMacOSDecorator struct{}

func (d *osMacOSDecorator) Descriptor() decorator.Descriptor {
	return decorator.NewDescriptor("os.macOS").
		Summary("Whether current OS is macOS").
		Roles(decorator.RoleProvider).
		Returns(types.TypeString, "\"true\" on macOS, \"false\" otherwise").
		TransportScope(decorator.TransportScopeAny).
		Idempotent().
		Block(decorator.BlockForbidden).
		Build()
}

func (d *osMacOSDecorator) Resolve(ctx decorator.ValueEvalContext, calls ...decorator.ValueCall) ([]decorator.ResolveResult, error) {
	results := make([]decorator.ResolveResult, len(calls))
	platform := ""
	if ctx.Session != nil {
		platform = ctx.Session.Platform()
	}
	value := "false"
	if platform == "darwin" {
		value = "true"
	}

	for i := range calls {
		results[i] = decorator.ResolveResult{
			Value:  value,
			Origin: "@os.macOS",
		}
	}
	return results, nil
}

type osWindowsDecorator struct{}

func (d *osWindowsDecorator) Descriptor() decorator.Descriptor {
	return decorator.NewDescriptor("os.Windows").
		Summary("Whether current OS is Windows").
		Roles(decorator.RoleProvider).
		Returns(types.TypeString, "\"true\" on Windows, \"false\" otherwise").
		TransportScope(decorator.TransportScopeAny).
		Idempotent().
		Block(decorator.BlockForbidden).
		Build()
}

func (d *osWindowsDecorator) Resolve(ctx decorator.ValueEvalContext, calls ...decorator.ValueCall) ([]decorator.ResolveResult, error) {
	results := make([]decorator.ResolveResult, len(calls))
	platform := ""
	if ctx.Session != nil {
		platform = ctx.Session.Platform()
	}
	value := "false"
	if platform == "windows" {
		value = "true"
	}

	for i := range calls {
		results[i] = decorator.ResolveResult{
			Value:  value,
			Origin: "@os.Windows",
		}
	}
	return results, nil
}

func init() {
	if err := decorator.Register("os.Get", &osGetDecorator{}); err != nil {
		panic(fmt.Sprintf("failed to register @os.Get decorator: %v", err))
	}
	if err := decorator.Register("os.Linux", &osLinuxDecorator{}); err != nil {
		panic(fmt.Sprintf("failed to register @os.Linux decorator: %v", err))
	}
	if err := decorator.Register("os.macOS", &osMacOSDecorator{}); err != nil {
		panic(fmt.Sprintf("failed to register @os.macOS decorator: %v", err))
	}
	if err := decorator.Register("os.Windows", &osWindowsDecorator{}); err != nil {
		panic(fmt.Sprintf("failed to register @os.Windows decorator: %v", err))
	}
}
