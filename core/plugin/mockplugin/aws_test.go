package mockplugin

import (
	"context"
	"errors"
	"io/fs"
	"testing"
	"time"

	"github.com/builtwithtofu/sigil/core/plugin"
	"github.com/google/go-cmp/cmp"
)

type fakeValueContext struct {
	session plugin.ParentTransport
}

func (f fakeValueContext) Context() context.Context        { return context.Background() }
func (f fakeValueContext) Session() plugin.ParentTransport { return f.session }

type fakeArgs struct {
	strings   map[string]string
	ints      map[string]int
	durations map[string]time.Duration
	secrets   map[string]string
	secretErr error
}

func (f fakeArgs) GetString(name string) string          { return f.strings[name] }
func (f fakeArgs) GetStringOptional(name string) string  { return f.strings[name] }
func (f fakeArgs) GetInt(name string) int                { return f.ints[name] }
func (f fakeArgs) GetDuration(name string) time.Duration { return f.durations[name] }
func (f fakeArgs) ResolveSecret(name string) (string, error) {
	if f.secretErr != nil {
		return "", f.secretErr
	}
	return f.secrets[name], nil
}

type fakeSession struct {
	env map[string]string
	cwd string
}

func (f *fakeSession) Run(ctx context.Context, argv []string, opts plugin.RunOpts) (plugin.Result, error) {
	return plugin.Result{}, nil
}
func (f *fakeSession) Put(ctx context.Context, data []byte, path string, mode fs.FileMode) error {
	return nil
}
func (f *fakeSession) Get(ctx context.Context, path string) ([]byte, error) { return nil, nil }
func (f *fakeSession) Snapshot() plugin.SessionSnapshot {
	workdir := f.cwd
	if workdir == "" {
		workdir = "/tmp"
	}
	return plugin.SessionSnapshot{Env: f.env, Workdir: workdir, Platform: "linux"}
}
func (f *fakeSession) Close() error { return nil }

func TestAWSSecretsResolve(t *testing.T) {
	capability := AWSSecretsCapability{}
	ctx := fakeValueContext{session: &fakeSession{env: map[string]string{}}}
	args := fakeArgs{strings: map[string]string{"name": "test"}}

	got, err := capability.Resolve(ctx, args)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if diff := cmp.Diff("mock-secret-test", got); diff != "" {
		t.Fatalf("Resolve() mismatch (-want +got):\n%s", diff)
	}
}

func TestAWSSecretsSchemaUsesPrimaryParameter(t *testing.T) {
	capability := AWSSecretsCapability{}
	schema := capability.Schema()
	if diff := cmp.Diff(plugin.Param{Name: "name", Type: schema.Primary.Type, Required: true}, schema.Primary); diff != "" {
		t.Fatalf("Schema() primary mismatch (-want +got):\n%s", diff)
	}
}

func TestAWSInstanceConnectOpenInheritsParentEnv(t *testing.T) {
	capability := AWSInstanceConnectCapability{}
	parent := &fakeSession{env: map[string]string{"HOME": "/home/tester"}}
	args := fakeArgs{
		strings: map[string]string{"instance": "i-123", "connectionType": "ssh"},
		secrets: map[string]string{"credentials": "creds"},
	}

	session, err := capability.Open(context.Background(), parent, args)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	if diff := cmp.Diff("/home/tester", session.Snapshot().Env["HOME"]); diff != "" {
		t.Fatalf("Open() env mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("linux", session.Snapshot().Platform); diff != "" {
		t.Fatalf("Open() platform mismatch (-want +got):\n%s", diff)
	}
}

func TestAWSInstanceConnectOpenResolvesDeclaredSecret(t *testing.T) {
	capability := AWSInstanceConnectCapability{}
	parent := &fakeSession{env: map[string]string{}}
	args := fakeArgs{
		strings: map[string]string{"instance": "i-123", "connectionType": "ssm"},
		secrets: map[string]string{"credentials": "creds"},
	}

	session, err := capability.Open(context.Background(), parent, args)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	awsSession := session.(*mockSession)
	if diff := cmp.Diff("creds", awsSession.creds); diff != "" {
		t.Fatalf("Open() creds mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("ssm", awsSession.connType); diff != "" {
		t.Fatalf("Open() connectionType mismatch (-want +got):\n%s", diff)
	}
}

func TestAWSInstanceConnectOpenRejectsCancelledContext(t *testing.T) {
	capability := AWSInstanceConnectCapability{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := capability.Open(ctx, &fakeSession{env: map[string]string{}}, fakeArgs{})
	if err == nil {
		t.Fatal("Open() error = nil, want cancellation error")
	}
}

func TestAWSInstanceConnectOpenPropagatesSecretError(t *testing.T) {
	capability := AWSInstanceConnectCapability{}
	_, err := capability.Open(context.Background(), &fakeSession{env: map[string]string{}}, fakeArgs{
		strings:   map[string]string{"instance": "i-123", "connectionType": "ssh"},
		secretErr: errors.New("undeclared secret"),
	})
	if err == nil {
		t.Fatal("Open() error = nil, want secret error")
	}
	if diff := cmp.Diff("undeclared secret", err.Error()); diff != "" {
		t.Fatalf("Open() error mismatch (-want +got):\n%s", diff)
	}
}
