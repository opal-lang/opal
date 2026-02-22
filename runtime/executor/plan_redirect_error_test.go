package executor

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/opal-lang/opal/core/decorator"
	"github.com/opal-lang/opal/core/planfmt"
	_ "github.com/opal-lang/opal/runtime/decorators"
)

type planRedirectErrorSink struct {
	canRead    bool
	canWrite   bool
	canAppend  bool
	openErr    error
	readClose  error
	writeClose error
}

func (d *planRedirectErrorSink) Descriptor() decorator.Descriptor {
	return decorator.NewDescriptor("test.plan.redirect.sink").
		Summary("Test sink for plan redirect error semantics").
		Roles(decorator.RoleEndpoint).
		ParamString("command", "Sink identity suffix").
		Required().
		Done().
		Build()
}

func (d *planRedirectErrorSink) IOCaps() decorator.IOCaps {
	return decorator.IOCaps{Read: d.canRead, Write: d.canWrite, Append: d.canAppend}
}

func (d *planRedirectErrorSink) OpenRead(_ decorator.ExecContext, _ ...decorator.IOOpts) (io.ReadCloser, error) {
	if d.openErr != nil {
		return nil, d.openErr
	}
	return &planCloseControlledReader{reader: strings.NewReader("alpha\n"), closeErr: d.readClose}, nil
}

func (d *planRedirectErrorSink) OpenWrite(_ decorator.ExecContext, _ bool, _ ...decorator.IOOpts) (io.WriteCloser, error) {
	if d.openErr != nil {
		return nil, d.openErr
	}
	return &planCloseControlledWriter{closeErr: d.writeClose}, nil
}

func (d *planRedirectErrorSink) WithParams(params map[string]any) decorator.IO {
	canRead, hasRead := params["read"].(bool)
	canWrite, hasWrite := params["write"].(bool)
	canAppend, hasAppend := params["append"].(bool)
	failOpen, _ := params["fail_open"].(string)
	failClose, _ := params["fail_close"].(string)

	inst := &planRedirectErrorSink{}
	if hasRead {
		inst.canRead = canRead
	}
	if hasWrite {
		inst.canWrite = canWrite
	}
	if hasAppend {
		inst.canAppend = canAppend
	}

	if failOpen == "open" {
		inst.openErr = errors.New("open failed")
	}
	if failClose == "read" {
		inst.readClose = errors.New("read close failed")
	}
	if failClose == "write" {
		inst.writeClose = errors.New("write close failed")
	}

	return inst
}

type planCloseControlledReader struct {
	reader   io.Reader
	closeErr error
}

func (r *planCloseControlledReader) Read(p []byte) (int, error) {
	return r.reader.Read(p)
}

func (r *planCloseControlledReader) Close() error {
	return r.closeErr
}

type planCloseControlledWriter struct {
	closeErr error
}

func (w *planCloseControlledWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

func (w *planCloseControlledWriter) Close() error {
	return w.closeErr
}

var registerPlanRedirectErrorSinkOnce sync.Once

func registerPlanRedirectErrorSink(t *testing.T) {
	t.Helper()
	var registerErr error
	registerPlanRedirectErrorSinkOnce.Do(func() {
		registerErr = decorator.Register("test.plan.redirect.sink", &planRedirectErrorSink{canRead: true, canWrite: true, canAppend: true})
	})
	if registerErr != nil {
		t.Fatalf("register test.plan.redirect.sink: %v", registerErr)
	}
}

func TestPlanRedirectValidateFailureReturnsStructuredSinkError(t *testing.T) {
	registerPlanRedirectErrorSink(t)
	runPlanRedirectErrorCase(t, planRedirectErrorCase{
		mode:          planfmt.RedirectOverwrite,
		sourceCommand: "echo never-runs",
		sinkArgs: []planfmt.Arg{
			{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "validate"}},
			{Key: "write", Val: planfmt.Value{Kind: planfmt.ValueBool, Bool: false}},
			{Key: "append", Val: planfmt.Value{Kind: planfmt.ValueBool, Bool: true}},
			{Key: "read", Val: planfmt.Value{Kind: planfmt.ValueBool, Bool: true}},
		},
		transportID:      "transport:validate",
		expectedSinkID:   "@test.plan.redirect.sink(validate)",
		expectedOp:       "validate",
		expectedCauseMsg: "does not support overwrite (>)",
	})
}

func TestPlanRedirectOpenFailureReturnsStructuredSinkError(t *testing.T) {
	registerPlanRedirectErrorSink(t)
	runPlanRedirectErrorCase(t, planRedirectErrorCase{
		mode:          planfmt.RedirectOverwrite,
		sourceCommand: "echo open-fails",
		sinkArgs: []planfmt.Arg{
			{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "open"}},
			{Key: "write", Val: planfmt.Value{Kind: planfmt.ValueBool, Bool: true}},
			{Key: "append", Val: planfmt.Value{Kind: planfmt.ValueBool, Bool: true}},
			{Key: "read", Val: planfmt.Value{Kind: planfmt.ValueBool, Bool: true}},
			{Key: "fail_open", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "open"}},
		},
		transportID:      "transport:open",
		expectedSinkID:   "@test.plan.redirect.sink(open)",
		expectedOp:       "open",
		expectedCauseMsg: "open failed",
	})
}

func TestPlanRedirectInputCloseFailureReturnsStructuredSinkError(t *testing.T) {
	registerPlanRedirectErrorSink(t)
	runPlanRedirectErrorCase(t, planRedirectErrorCase{
		mode:          planfmt.RedirectInput,
		sourceCommand: "cat > /dev/null",
		sinkArgs: []planfmt.Arg{
			{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "input-close"}},
			{Key: "read", Val: planfmt.Value{Kind: planfmt.ValueBool, Bool: true}},
			{Key: "write", Val: planfmt.Value{Kind: planfmt.ValueBool, Bool: true}},
			{Key: "append", Val: planfmt.Value{Kind: planfmt.ValueBool, Bool: true}},
			{Key: "fail_close", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "read"}},
		},
		transportID:      "transport:input-close",
		expectedSinkID:   "@test.plan.redirect.sink(input-close)",
		expectedOp:       "close",
		expectedCauseMsg: "read close failed",
	})
}

func TestPlanRedirectOutputCloseFailureReturnsStructuredSinkError(t *testing.T) {
	registerPlanRedirectErrorSink(t)
	runPlanRedirectErrorCase(t, planRedirectErrorCase{
		mode:          planfmt.RedirectOverwrite,
		sourceCommand: "echo close-fails",
		sinkArgs: []planfmt.Arg{
			{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "output-close"}},
			{Key: "write", Val: planfmt.Value{Kind: planfmt.ValueBool, Bool: true}},
			{Key: "append", Val: planfmt.Value{Kind: planfmt.ValueBool, Bool: true}},
			{Key: "read", Val: planfmt.Value{Kind: planfmt.ValueBool, Bool: true}},
			{Key: "fail_close", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "write"}},
		},
		transportID:      "transport:output-close",
		expectedSinkID:   "@test.plan.redirect.sink(output-close)",
		expectedOp:       "close",
		expectedCauseMsg: "write close failed",
	})
}

type planRedirectErrorCase struct {
	mode             planfmt.RedirectMode
	sourceCommand    string
	sinkArgs         []planfmt.Arg
	transportID      string
	expectedSinkID   string
	expectedOp       string
	expectedCauseMsg string
}

func runPlanRedirectErrorCase(t *testing.T, tc planRedirectErrorCase) {
	t.Helper()

	plan := &planfmt.Plan{Target: "plan-redirect-errors", Steps: []planfmt.Step{{
		ID: 1,
		Tree: &planfmt.RedirectNode{
			Source: &planfmt.CommandNode{
				Decorator:   "@shell",
				TransportID: tc.transportID,
				Args:        []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: tc.sourceCommand}}},
			},
			Target: planfmt.CommandNode{Decorator: "@test.plan.redirect.sink", Args: tc.sinkArgs},
			Mode:   tc.mode,
		},
	}}}

	stderr := captureStderr(t, func() {
		result, err := ExecutePlan(context.Background(), plan, Config{}, testVault())
		if err != nil {
			t.Fatalf("execute failed: %v", err)
		}
		if diff := cmp.Diff(1, result.ExitCode); diff != "" {
			t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
		}
	})

	expected := "Error: sink " + tc.expectedSinkID + " " + tc.expectedOp + " failed on transport " + tc.transportID + ": " + tc.expectedCauseMsg
	if diff := cmp.Diff(true, strings.Contains(stderr, expected)); diff != "" {
		t.Fatalf("stderr mismatch (-want +got):\n%s\nstderr: %q", diff, stderr)
	}
}
