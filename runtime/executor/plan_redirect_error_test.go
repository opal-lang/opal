package executor

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/builtwithtofu/sigil/core/planfmt"
	"github.com/google/go-cmp/cmp"
)

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

func TestPlanRedirectValidateFailureReturnsStructuredSinkError(t *testing.T) {
	registerExecutorSessionTestPlugin()
	runPlanRedirectErrorCase(t, planRedirectErrorCase{
		mode:            planfmt.RedirectOverwrite,
		sourceCommand:   "echo never-runs",
		targetDecorator: "@test.plan.redirect.readonly",
		sinkArgs:        []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "validate"}}},
		transportID:     "transport:validate",
		expectedStderr:  "Error: sink @test.plan.redirect.readonly validate failed on transport transport:validate: does not support overwrite (>)",
	})
}

func TestPlanRedirectOpenFailureReturnsStructuredSinkError(t *testing.T) {
	registerExecutorSessionTestPlugin()
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
		transportID:    "transport:open",
		expectedStderr: "Error: sink @test.plan.redirect.sink open failed on transport transport:open: open failed",
	})
}

func TestPlanRedirectInputCloseFailureReturnsStructuredSinkError(t *testing.T) {
	registerExecutorSessionTestPlugin()
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
		transportID:    "transport:input-close",
		expectedStderr: "Error: sink @test.plan.redirect.sink close failed on transport transport:input-close: read close failed",
	})
}

func TestPlanRedirectOutputCloseFailureReturnsStructuredSinkError(t *testing.T) {
	registerExecutorSessionTestPlugin()
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
		transportID:    "transport:output-close",
		expectedStderr: "Error: sink @test.plan.redirect.sink close failed on transport transport:output-close: write close failed",
	})
}

type planRedirectErrorCase struct {
	mode            planfmt.RedirectMode
	sourceCommand   string
	sinkArgs        []planfmt.Arg
	targetDecorator string
	transportID     string
	expectedStderr  string
}

func runPlanRedirectErrorCase(t *testing.T, tc planRedirectErrorCase) {
	t.Helper()

	transports := []planfmt.Transport{{ID: "local", Decorator: "local", ParentID: ""}}
	if tc.transportID != "" && tc.transportID != "local" {
		transports = append(transports, planfmt.Transport{ID: tc.transportID, Decorator: "local", ParentID: ""})
	}

	plan := &planfmt.Plan{Target: "plan-redirect-errors", Transports: transports, Steps: []planfmt.Step{{
		ID: 1,
		Tree: &planfmt.RedirectNode{
			Source: &planfmt.CommandNode{
				Decorator:   "@shell",
				TransportID: tc.transportID,
				Args:        []planfmt.Arg{{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: tc.sourceCommand}}},
			},
			Target: planfmt.CommandNode{Decorator: targetDecorator(tc), Args: tc.sinkArgs},
			Mode:   tc.mode,
		},
	}}}

	stderrBuf := &bytes.Buffer{}
	result, err := ExecutePlan(context.Background(), plan, Config{Stderr: stderrBuf}, testVault())
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if diff := cmp.Diff(1, result.ExitCode); diff != "" {
		t.Fatalf("exit code mismatch (-want +got):\n%s", diff)
	}

	stderr := stderrBuf.String()
	if diff := cmp.Diff(true, strings.Contains(stderr, tc.expectedStderr)); diff != "" {
		t.Fatalf("stderr mismatch (-want +got):\n%s\nstderr: %q", diff, stderr)
	}
}

func targetDecorator(tc planRedirectErrorCase) string {
	if tc.targetDecorator != "" {
		return tc.targetDecorator
	}
	return "@test.plan.redirect.sink"
}
