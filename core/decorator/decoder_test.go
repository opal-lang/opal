package decorator

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNormalizeArgsPrimaryAndPositional(t *testing.T) {
	schema := NewDescriptor("env").
		PrimaryParamString("property", "Environment variable name").
		Done().
		ParamString("default", "Default value").
		Done().
		ParamInt("retries", "Retry count").
		Done().
		Build().Schema

	primary := "HOME"
	raw := map[string]any{
		"arg1":    "/tmp",
		"retries": 3,
	}

	canonical, warnings, err := NormalizeArgs(schema, &primary, raw)
	if err != nil {
		t.Fatalf("NormalizeArgs() error = %v", err)
	}

	if len(warnings) != 0 {
		t.Fatalf("warnings length = %d, want 0", len(warnings))
	}

	want := map[string]any{
		"property": "HOME",
		"default":  "/tmp",
		"retries":  3,
	}

	if diff := cmp.Diff(want, canonical); diff != "" {
		t.Errorf("canonical args mismatch (-want +got):\n%s", diff)
	}
}

func TestNormalizeArgsMixedPositionalAndNamed(t *testing.T) {
	schema := NewDescriptor("retry").
		ParamInt("times", "Retry attempts").
		Done().
		ParamDuration("delay", "Retry delay").
		Done().
		ParamEnum("backoff", "Backoff mode").
		Values("exponential", "linear").
		Done().
		Build().Schema

	raw := map[string]any{
		"arg1":  3,
		"delay": "5s",
	}

	canonical, warnings, err := NormalizeArgs(schema, nil, raw)
	if err != nil {
		t.Fatalf("NormalizeArgs() error = %v", err)
	}

	if len(warnings) != 0 {
		t.Fatalf("warnings length = %d, want 0", len(warnings))
	}

	want := map[string]any{
		"times": 3,
		"delay": "5s",
	}
	if diff := cmp.Diff(want, canonical); diff != "" {
		t.Errorf("canonical args mismatch (-want +got):\n%s", diff)
	}
}

func TestNormalizeArgsMixedNamedThenPositional(t *testing.T) {
	schema := NewDescriptor("retry").
		ParamInt("times", "Retry attempts").
		Done().
		ParamDuration("delay", "Retry delay").
		Done().
		Build().Schema

	// Mirrors parser behavior for @retry(times=3, 5s): positional binds to next unfilled param.
	raw := map[string]any{
		"times": 3,
		"arg1":  "5s",
	}

	canonical, warnings, err := NormalizeArgs(schema, nil, raw)
	if err != nil {
		t.Fatalf("NormalizeArgs() error = %v", err)
	}

	if len(warnings) != 0 {
		t.Fatalf("warnings length = %d, want 0", len(warnings))
	}

	want := map[string]any{
		"times": 3,
		"delay": "5s",
	}
	if diff := cmp.Diff(want, canonical); diff != "" {
		t.Errorf("canonical args mismatch (-want +got):\n%s", diff)
	}
}

func TestNormalizeArgsRequiredParametersShiftLeft(t *testing.T) {
	schema := NewDescriptor("shift").
		ParamString("a", "Optional value").
		Done().
		ParamInt("b", "Required value").
		Required().
		Done().
		ParamString("c", "Optional trailing value").
		Done().
		Build().Schema

	raw := map[string]any{"arg1": int64(3)}

	canonical, warnings, err := NormalizeArgs(schema, nil, raw)
	if err != nil {
		t.Fatalf("NormalizeArgs() error = %v", err)
	}

	if len(warnings) != 0 {
		t.Fatalf("warnings length = %d, want 0", len(warnings))
	}

	want := map[string]any{"b": int64(3)}
	if diff := cmp.Diff(want, canonical); diff != "" {
		t.Errorf("canonical args mismatch (-want +got):\n%s", diff)
	}
}

func TestNormalizeArgsTooManyPositional(t *testing.T) {
	schema := NewDescriptor("var").
		PrimaryParamString("name", "Variable name").
		Done().
		Build().Schema

	primary := "HOME"
	raw := map[string]any{
		"arg1": "extra",
	}

	_, _, err := NormalizeArgs(schema, &primary, raw)
	if err == nil {
		t.Fatal("NormalizeArgs() error = nil, want error")
	}

	want := "too many positional arguments"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestNormalizeArgsRejectsPositionalGapWithoutNamedReservations(t *testing.T) {
	schema := NewDescriptor("retry").
		ParamInt("times", "Retry attempts").
		Done().
		ParamDuration("delay", "Retry delay").
		Done().
		Build().Schema

	_, _, err := NormalizeArgs(schema, nil, map[string]any{"arg2": "5s"})
	if err == nil {
		t.Fatal("NormalizeArgs() error = nil, want error")
	}

	want := "invalid positional argument index: missing arg1"
	if diff := cmp.Diff(want, err.Error()); diff != "" {
		t.Errorf("error mismatch (-want +got):\n%s", diff)
	}
}

func TestNormalizeArgsAllowsPositionalGapCoveredByNamedArg(t *testing.T) {
	schema := NewDescriptor("retry").
		ParamInt("times", "Retry attempts").
		Done().
		ParamDuration("delay", "Retry delay").
		Done().
		Build().Schema

	raw := map[string]any{
		"times": 3,
		"arg2":  "5s",
	}

	canonical, warnings, err := NormalizeArgs(schema, nil, raw)
	if err != nil {
		t.Fatalf("NormalizeArgs() error = %v", err)
	}

	if len(warnings) != 0 {
		t.Fatalf("warnings length = %d, want 0", len(warnings))
	}

	want := map[string]any{
		"times": 3,
		"delay": "5s",
	}
	if diff := cmp.Diff(want, canonical); diff != "" {
		t.Errorf("canonical args mismatch (-want +got):\n%s", diff)
	}
}

func TestNormalizeArgsDeprecatedParameterName(t *testing.T) {
	schema := NewDescriptor("parallel").
		ParamInt("maxConcurrency", "Deprecated parameter").
		Deprecation(DeprecationInfo{ReplacedBy: "max_workers"}).
		Done().
		ParamInt("max_workers", "Replacement parameter").
		Done().
		Build().Schema

	raw := map[string]any{"maxConcurrency": 5}

	canonical, warnings, err := NormalizeArgs(schema, nil, raw)
	if err != nil {
		t.Fatalf("NormalizeArgs() error = %v", err)
	}

	if len(warnings) != 1 {
		t.Fatalf("warnings length = %d, want 1", len(warnings))
	}

	if warnings[0].Parameter != "maxConcurrency" {
		t.Errorf("warning parameter = %q, want %q", warnings[0].Parameter, "maxConcurrency")
	}

	want := map[string]any{"max_workers": 5}
	if diff := cmp.Diff(want, canonical); diff != "" {
		t.Errorf("canonical args mismatch (-want +got):\n%s", diff)
	}
}

func TestNormalizeArgsUnknownParameter(t *testing.T) {
	schema := NewDescriptor("retry").
		ParamInt("times", "Retry attempts").
		Done().
		Build().Schema

	_, _, err := NormalizeArgs(schema, nil, map[string]any{"unknown": 1})
	if err == nil {
		t.Fatal("NormalizeArgs() error = nil, want error")
	}

	want := "unknown parameter \"unknown\""
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestValidateArgsStrictTypes(t *testing.T) {
	schema := NewDescriptor("retry").
		ParamInt("times", "Retry attempts").
		Required().
		Min(1).
		Max(5).
		Done().
		Build().Schema

	valid := map[string]any{"times": 3}
	if _, err := ValidateArgs(schema, valid); err != nil {
		t.Fatalf("ValidateArgs(valid) error = %v", err)
	}

	invalid := map[string]any{"times": "3"}
	_, err := ValidateArgs(schema, invalid)
	if err == nil {
		t.Fatal("ValidateArgs(invalid) error = nil, want error")
	}

	want := "parameter \"times\" expects integer"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestValidateArgsEnumDeprecatedValue(t *testing.T) {
	schema := NewDescriptor("parallel").
		ParamEnum("onFailure", "Failure policy").
		Values("fail_fast", "wait_all").
		Deprecated("fail-fast", "fail_fast").
		Done().
		Build().Schema

	canonical := map[string]any{"onFailure": "fail-fast"}
	warnings, err := ValidateArgs(schema, canonical)
	if err != nil {
		t.Fatalf("ValidateArgs() error = %v", err)
	}

	if len(warnings) != 1 {
		t.Fatalf("warnings length = %d, want 1", len(warnings))
	}

	if warnings[0].Parameter != "onFailure" {
		t.Errorf("warning parameter = %q, want %q", warnings[0].Parameter, "onFailure")
	}

	if canonical["onFailure"] != "fail_fast" {
		t.Errorf("canonical[onFailure] = %v, want %v", canonical["onFailure"], "fail_fast")
	}
}

func TestValidateArgsDuration(t *testing.T) {
	schema := NewDescriptor("timeout").
		ParamDuration("duration", "Timeout").
		Required().
		Done().
		Build().Schema

	valid := map[string]any{"duration": "5m"}
	if _, err := ValidateArgs(schema, valid); err != nil {
		t.Fatalf("ValidateArgs(valid) error = %v", err)
	}

	invalid := map[string]any{"duration": "not_a_duration"}
	_, err := ValidateArgs(schema, invalid)
	if err == nil {
		t.Fatal("ValidateArgs(invalid) error = nil, want error")
	}

	want := "parameter \"duration\" expects duration"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestDecodeIntoStruct(t *testing.T) {
	type retryConfig struct {
		Times   int    `decorator:"times"`
		Backoff string `decorator:"backoff"`
	}

	schema := NewDescriptor("retry").
		ParamInt("times", "Retry attempts").
		Default(3).
		Done().
		ParamEnum("backoff", "Backoff mode").
		Values("exponential", "linear").
		Default("exponential").
		Done().
		Build().Schema

	cfg, warnings, err := DecodeInto[retryConfig](schema, nil, map[string]any{})
	if err != nil {
		t.Fatalf("DecodeInto() error = %v", err)
	}

	if len(warnings) != 0 {
		t.Fatalf("warnings length = %d, want 0", len(warnings))
	}

	want := retryConfig{Times: 3, Backoff: "exponential"}
	if diff := cmp.Diff(want, cfg); diff != "" {
		t.Errorf("decoded config mismatch (-want +got):\n%s", diff)
	}
}

func TestDecodeIntoStructMissingField(t *testing.T) {
	type badConfig struct {
		SomethingElse string `decorator:"other"`
	}

	schema := NewDescriptor("test").
		ParamString("name", "Name").
		Required().
		Done().
		Build().Schema

	_, _, err := DecodeInto[badConfig](schema, nil, map[string]any{"name": "alice"})
	if err == nil {
		t.Fatal("DecodeInto() error = nil, want error")
	}

	want := "no struct field mapped for parameter \"name\""
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}
