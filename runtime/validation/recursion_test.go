package validation

import (
	"strings"
	"testing"

	"github.com/aledsdavies/devcmd/runtime/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	// Import builtin decorators to register them globally
	_ "github.com/aledsdavies/devcmd/runtime/decorators/builtin"
)

func TestValidateNoRecursion_SimpleRecursion(t *testing.T) {
	input := `build: @cmd(build)`

	program, err := parser.Parse(strings.NewReader(input))
	require.NoError(t, err, "parsing should succeed")

	err = ValidateNoRecursion(program)
	require.Error(t, err, "should detect recursion")

	var recursionErr *RecursionError
	require.ErrorAs(t, err, &recursionErr, "should be a RecursionError")
	assert.Equal(t, "build", recursionErr.Command)
	assert.Equal(t, []string{"build", "build"}, recursionErr.Cycle)
	assert.Contains(t, recursionErr.Message, "build -> build")
}

func TestValidateNoRecursion_IndirectRecursion(t *testing.T) {
	input := `
build: @cmd(test)
test: @cmd(deploy) 
deploy: @cmd(build)
`

	program, err := parser.Parse(strings.NewReader(input))
	require.NoError(t, err, "parsing should succeed")

	err = ValidateNoRecursion(program)
	require.Error(t, err, "should detect indirect recursion")

	var recursionErr *RecursionError
	require.ErrorAs(t, err, &recursionErr)
	assert.Contains(t, recursionErr.Cycle, "build")
	assert.Contains(t, recursionErr.Cycle, "test")
	assert.Contains(t, recursionErr.Cycle, "deploy")
}

func TestValidateNoRecursion_ComplexChain(t *testing.T) {
	input := `
a: @cmd(b)
b: @cmd(c)
c: @cmd(d)
d: @cmd(a)
`

	program, err := parser.Parse(strings.NewReader(input))
	require.NoError(t, err, "parsing should succeed")

	err = ValidateNoRecursion(program)
	require.Error(t, err, "should detect complex recursion")

	var recursionErr *RecursionError
	require.ErrorAs(t, err, &recursionErr)
	// Should detect the cycle
	assert.Len(t, recursionErr.Cycle, 5) // a->b->c->d->a (5 elements)
}

func TestValidateNoRecursion_NestedDecorator(t *testing.T) {
	input := `
build: @timeout(30s) {
    @cmd(build)
}
`

	program, err := parser.Parse(strings.NewReader(input))
	require.NoError(t, err, "parsing should succeed")

	err = ValidateNoRecursion(program)
	require.Error(t, err, "should detect recursion in nested decorator")

	var recursionErr *RecursionError
	require.ErrorAs(t, err, &recursionErr)
	assert.Equal(t, "build", recursionErr.Command)
}

func TestValidateNoRecursion_NoRecursion(t *testing.T) {
	input := `
build: go build -o bin/app
test: go test ./...
deploy: @cmd(build) && @cmd(test) && kubectl apply -f k8s/
`

	program, err := parser.Parse(strings.NewReader(input))
	require.NoError(t, err, "parsing should succeed")

	err = ValidateNoRecursion(program)
	assert.NoError(t, err, "should not detect recursion in valid commands")
}

func TestValidateNoRecursion_SelfReferenceInChain(t *testing.T) {
	input := `
build: echo "start" && @cmd(build) && echo "done"
`

	program, err := parser.Parse(strings.NewReader(input))
	require.NoError(t, err, "parsing should succeed")

	err = ValidateNoRecursion(program)
	require.Error(t, err, "should detect self-reference in chain")

	var recursionErr *RecursionError
	require.ErrorAs(t, err, &recursionErr)
	assert.Equal(t, "build", recursionErr.Command)
}

func TestValidateNoRecursion_MultipleCommands_OneRecursive(t *testing.T) {
	input := `
good: echo "this is fine"
bad: @cmd(bad)
other: @cmd(good) && echo "also fine"
`

	program, err := parser.Parse(strings.NewReader(input))
	require.NoError(t, err, "parsing should succeed")

	err = ValidateNoRecursion(program)
	require.Error(t, err, "should detect the recursive command")

	var recursionErr *RecursionError
	require.ErrorAs(t, err, &recursionErr)
	assert.Equal(t, "bad", recursionErr.Command)
}

func TestValidateNoRecursion_UnknownCommandReference(t *testing.T) {
	input := `
build: @cmd(unknown)
`

	program, err := parser.Parse(strings.NewReader(input))
	require.NoError(t, err, "parsing should succeed")

	// Unknown command references should not cause recursion errors
	// They will be caught as "command not found" errors elsewhere
	err = ValidateNoRecursion(program)
	assert.NoError(t, err, "unknown command references should not cause recursion errors")
}
