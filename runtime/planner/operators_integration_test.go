package planner_test

import (
	"context"
	"testing"

	"github.com/aledsdavies/opal/core/planfmt"
	_ "github.com/aledsdavies/opal/runtime/decorators"
	"github.com/aledsdavies/opal/runtime/executor"
	"github.com/aledsdavies/opal/runtime/lexer"
	"github.com/aledsdavies/opal/runtime/parser"
	"github.com/aledsdavies/opal/runtime/planner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOperatorsEndToEnd tests operators through the full pipeline
func TestOperatorsEndToEnd(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		target   string
		wantExit int
	}{
		{
			name:     "semicolon all succeed",
			source:   `fun test = echo "a"; echo "b"; echo "c"`,
			target:   "test",
			wantExit: 0,
		},
		{
			name:     "semicolon first fails rest run",
			source:   `fun test = exit 1; echo "still runs"; exit 0`,
			target:   "test",
			wantExit: 0,
		},
		{
			name:     "AND both succeed",
			source:   `fun test = echo "first" && echo "second"`,
			target:   "test",
			wantExit: 0,
		},
		{
			name:     "AND first fails second skipped",
			source:   `fun test = exit 1 && echo "should not run"`,
			target:   "test",
			wantExit: 1,
		},
		{
			name:     "OR first succeeds second skipped",
			source:   `fun test = echo "success" || echo "should not run"`,
			target:   "test",
			wantExit: 0,
		},
		{
			name:     "OR first fails second runs",
			source:   `fun test = exit 1 || echo "fallback"`,
			target:   "test",
			wantExit: 0,
		},
		{
			name:     "mixed AND then OR success",
			source:   `fun test = echo "a" && echo "b" || echo "fallback"`,
			target:   "test",
			wantExit: 0,
		},
		{
			name:     "mixed AND fails OR runs",
			source:   `fun test = exit 1 && echo "no" || echo "yes"`,
			target:   "test",
			wantExit: 0,
		},
		{
			name:     "pipe basic",
			source:   `fun test = echo "hello world" | grep "hello"`,
			target:   "test",
			wantExit: 0,
		},
		{
			name:     "pipe no match",
			source:   `fun test = echo "test" | grep "nomatch"`,
			target:   "test",
			wantExit: 1,
		},
		{
			name:     "pipe chained (3 commands)",
			source:   `fun test = printf "line1\nline2\nline3\n" | grep "line" | wc -l`,
			target:   "test",
			wantExit: 0,
		},
		{
			name:     "AND then pipe (precedence test)",
			source:   `fun test = echo "first" && echo "second" | grep "second"`,
			target:   "test",
			wantExit: 0,
		},
		{
			name:     "pipe then AND (precedence test)",
			source:   `fun test = echo "test" | grep "test" && echo "found"`,
			target:   "test",
			wantExit: 0,
		},
		{
			name:     "complex mix: pipe AND pipe OR",
			source:   `fun test = echo "a" | grep "a" && echo "b" | grep "b" || echo "fallback"`,
			target:   "test",
			wantExit: 0,
		},
		{
			name:     "pipe with AND",
			source:   `fun test = echo "test" | grep "test" && echo "found"`,
			target:   "test",
			wantExit: 0,
		},
		{
			name:     "pipe with OR",
			source:   `fun test = echo "test" | grep "nomatch" || echo "fallback"`,
			target:   "test",
			wantExit: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Lex
			lex := lexer.NewLexer()
			lex.Init([]byte(tt.source))
			tokens := lex.GetTokens()

			// Parse
			tree := parser.Parse([]byte(tt.source))
			require.Empty(t, tree.Errors, "parse errors")

			// Plan
			plan, err := planner.Plan(tree.Events, tokens, planner.Config{Target: tt.target})
			require.NoError(t, err, "plan error")

			// Execute
			steps := planfmt.ToSDKSteps(plan.Steps)
			result, err := executor.Execute(context.Background(), steps, executor.Config{}, nil)
			require.NoError(t, err, "execute error")

			// Verify
			assert.Equal(t, tt.wantExit, result.ExitCode, "exit code mismatch")
		})
	}
}
