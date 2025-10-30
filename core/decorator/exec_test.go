package decorator

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestExecContextWithIOReaderStdin verifies that ExecContext.Stdin accepts io.Reader
func TestExecContextWithIOReaderStdin(t *testing.T) {
	data := []byte("test data")
	stdin := bytes.NewReader(data)

	ctx := ExecContext{
		Stdin: stdin, // Should compile with io.Reader
	}

	// Verify we can read from stdin
	buf := make([]byte, len(data))
	n, err := ctx.Stdin.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.Equal(t, data, buf)
}

// TestExecContextHasStderrField verifies that ExecContext has Stderr field
func TestExecContextHasStderrField(t *testing.T) {
	var buf bytes.Buffer
	ctx := ExecContext{
		Stderr: &buf, // Should compile
	}

	// Verify we can write to stderr
	_, err := ctx.Stderr.Write([]byte("error"))
	assert.NoError(t, err)
	assert.Equal(t, "error", buf.String())
}
