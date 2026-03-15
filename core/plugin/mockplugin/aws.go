package mockplugin

import (
	"context"
	"fmt"
	"io/fs"

	"github.com/builtwithtofu/sigil/core/plugin"
	"github.com/builtwithtofu/sigil/core/types"
)

// AWSPlugin is a mock external-style plugin used to validate plugin contracts.
type AWSPlugin struct{}

func (p AWSPlugin) Identity() plugin.PluginIdentity {
	return plugin.PluginIdentity{Name: "aws", Version: "1.0.0", APIVersion: 1}
}

func (p AWSPlugin) Capabilities() []plugin.Capability {
	return []plugin.Capability{AWSSecretsCapability{}, AWSInstanceConnectCapability{}}
}

type AWSSecretsCapability struct{}

func (c AWSSecretsCapability) Kind() plugin.CapabilityKind { return plugin.KindValue }
func (c AWSSecretsCapability) Path() string                { return "aws.secrets" }

func (c AWSSecretsCapability) Schema() plugin.Schema {
	return plugin.Schema{
		Primary: plugin.Param{Name: "name", Type: types.TypeString, Required: true},
		Returns: types.TypeString,
	}
}

func (c AWSSecretsCapability) Resolve(ctx plugin.ValueContext, args plugin.ResolvedArgs) (string, error) {
	return fmt.Sprintf("mock-secret-%s", args.GetString("name")), nil
}

type AWSInstanceConnectCapability struct{}

func (c AWSInstanceConnectCapability) Kind() plugin.CapabilityKind { return plugin.KindTransport }
func (c AWSInstanceConnectCapability) Path() string                { return "aws.instance.connect" }

func (c AWSInstanceConnectCapability) Schema() plugin.Schema {
	return plugin.Schema{
		Params: []plugin.Param{
			{Name: "instance", Type: types.TypeString, Required: true},
			{Name: "connectionType", Type: types.TypeString, Default: "ssm", Enum: []string{"ssm", "ssh"}},
		},
		Block:   plugin.BlockRequired,
		Secrets: []string{"credentials", "sshKey"},
	}
}

func (c AWSInstanceConnectCapability) Open(ctx context.Context, parent plugin.ParentTransport, args plugin.ResolvedArgs) (plugin.OpenedTransport, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	creds, err := args.ResolveSecret("credentials")
	if err != nil {
		return nil, err
	}

	env := map[string]string{}
	if parent != nil {
		for key, value := range parent.Snapshot().Env {
			env[key] = value
		}
	}

	return &mockSession{
		env:      env,
		workdir:  workdirFromParent(parent),
		platform: platformFromParent(parent),
		instance: args.GetString("instance"),
		connType: args.GetString("connectionType"),
		creds:    creds,
	}, nil
}

type mockSession struct {
	env      map[string]string
	workdir  string
	platform string
	instance string
	connType string
	creds    string
}

func (s *mockSession) Run(ctx context.Context, argv []string, opts plugin.RunOpts) (plugin.Result, error) {
	return plugin.Result{ExitCode: plugin.ExitSuccess}, nil
}
func (s *mockSession) Put(ctx context.Context, data []byte, path string, mode fs.FileMode) error {
	return nil
}
func (s *mockSession) Get(ctx context.Context, path string) ([]byte, error) { return nil, nil }
func (s *mockSession) Snapshot() plugin.SessionSnapshot {
	copyEnv := make(map[string]string, len(s.env))
	for key, value := range s.env {
		copyEnv[key] = value
	}
	return plugin.SessionSnapshot{Env: copyEnv, Workdir: s.workdir, Platform: s.platform}
}
func (s *mockSession) WithSnapshot(snapshot plugin.SessionSnapshot) plugin.OpenedTransport {
	clone := *s
	clone.env = make(map[string]string, len(snapshot.Env))
	for key, value := range snapshot.Env {
		clone.env[key] = value
	}
	clone.workdir = snapshot.Workdir
	clone.platform = snapshot.Platform
	return &clone
}
func (s *mockSession) Close() error { return nil }

func workdirFromParent(parent plugin.ParentTransport) string {
	if parent == nil {
		return ""
	}
	return parent.Snapshot().Workdir
}

func platformFromParent(parent plugin.ParentTransport) string {
	if parent == nil {
		return "linux"
	}
	platform := parent.Snapshot().Platform
	if platform == "" {
		return "linux"
	}
	return platform
}
