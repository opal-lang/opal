package executor

import (
	"bytes"
	"context"
	"io"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/opal-lang/opal/core/sdk"
	_ "github.com/opal-lang/opal/runtime/decorators"
)

type captureSink struct {
	mu           sync.Mutex
	openCount    int
	transportIDs []string
	sessionIDs   []string
	output       bytes.Buffer
}

func (s *captureSink) Caps() sdk.SinkCaps {
	return sdk.SinkCaps{Overwrite: true, Append: true}
}

func (s *captureSink) Open(ctx sdk.ExecutionContext, mode sdk.RedirectMode, meta map[string]any) (io.WriteCloser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.openCount++
	if ec, ok := ctx.(*executionContext); ok {
		s.transportIDs = append(s.transportIDs, ec.transportID)
	}
	if transport, ok := ctx.Transport().(*sessionTransport); ok {
		s.sessionIDs = append(s.sessionIDs, transport.session.ID())
	}

	return nopWriteCloser{Writer: &s.output}, nil
}

func (s *captureSink) Identity() (kind, identifier string) {
	return "test.capture", "capture"
}

type nopWriteCloser struct {
	io.Writer
}

func (n nopWriteCloser) Close() error {
	return nil
}

func TestRedirectSinkUsesSourceTransportContext(t *testing.T) {
	sink := &captureSink{}
	steps := []sdk.Step{{
		ID: 1,
		Tree: &sdk.RedirectNode{
			Source: &sdk.CommandNode{
				Name:        "@shell",
				TransportID: "transport:A",
				Args:        map[string]any{"command": "echo routed"},
			},
			Sink: sink,
			Mode: sdk.RedirectOverwrite,
		},
	}}

	result, err := Execute(context.Background(), steps, Config{}, testVault())
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if diff := cmp.Diff(0, result.ExitCode); diff != "" {
		t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(1, sink.openCount); diff != "" {
		t.Fatalf("sink open count mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]string{"transport:A"}, sink.transportIDs); diff != "" {
		t.Fatalf("transport context mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]string{"transport:A"}, sink.sessionIDs); diff != "" {
		t.Fatalf("session routing mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("routed\n", sink.output.String()); diff != "" {
		t.Fatalf("sink output mismatch (-want +got):\n%s", diff)
	}
}
