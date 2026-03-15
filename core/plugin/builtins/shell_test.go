package builtins

import (
	"context"
	"io"
	"io/fs"
	"testing"

	"github.com/builtwithtofu/sigil/core/plugin"
	"github.com/google/go-cmp/cmp"
)

func TestShellWrapperCapabilityExecutesCommand(t *testing.T) {
	capability := ShellWrapperCapability{}
	session := &captureParentTransport{snapshot: plugin.SessionSnapshot{Workdir: "/tmp", Env: map[string]string{}}}
	node := capability.Wrap(nil, fakeArgs{strings: map[string]string{"command": "echo hello"}})

	result, err := node.Execute(execContextWithIO{ctx: context.Background(), session: session})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if diff := cmp.Diff(plugin.ExitSuccess, result.ExitCode); diff != "" {
		t.Fatalf("Execute() exit code mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]string{"bash", "-c", "echo hello"}, session.lastArgv); diff != "" {
		t.Fatalf("Execute() argv mismatch (-want +got):\n%s", diff)
	}
}

func TestShellRedirectTargetWriteAndRead(t *testing.T) {
	capability := ShellWrapperCapability{}
	session := &memoryParentTransport{snapshot: plugin.SessionSnapshot{Workdir: "/tmp"}, files: map[string][]byte{}}
	ctx := execContextWithIO{ctx: context.Background(), session: session}
	args := fakeArgs{strings: map[string]string{"command": "out.txt"}}

	writer, err := capability.OpenForWrite(ctx, args, false)
	if err != nil {
		t.Fatalf("OpenForWrite() error = %v", err)
	}
	if _, err := writer.Write([]byte("hello\n")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	reader, err := capability.OpenForRead(ctx, args)
	if err != nil {
		t.Fatalf("OpenForRead() error = %v", err)
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if diff := cmp.Diff("hello\n", string(data)); diff != "" {
		t.Fatalf("content mismatch (-want +got):\n%s", diff)
	}
}

type captureParentTransport struct {
	snapshot plugin.SessionSnapshot
	lastArgv []string
}

func (c *captureParentTransport) Run(ctx context.Context, argv []string, opts plugin.RunOpts) (plugin.Result, error) {
	c.lastArgv = append([]string(nil), argv...)
	return plugin.Result{ExitCode: plugin.ExitSuccess}, nil
}

func (c *captureParentTransport) Put(ctx context.Context, data []byte, path string, mode fs.FileMode) error {
	return nil
}

func (c *captureParentTransport) Get(ctx context.Context, path string) ([]byte, error) {
	return nil, nil
}

func (c *captureParentTransport) Snapshot() plugin.SessionSnapshot { return c.snapshot }

func (c *captureParentTransport) Close() error { return nil }

type execContextWithIO struct {
	ctx     context.Context
	session plugin.ParentTransport
}

func (c execContextWithIO) Context() context.Context        { return c.ctx }
func (c execContextWithIO) Session() plugin.ParentTransport { return c.session }
func (c execContextWithIO) Stdin() io.Reader                { return nil }
func (c execContextWithIO) Stdout() io.Writer               { return nil }
func (c execContextWithIO) Stderr() io.Writer               { return nil }
