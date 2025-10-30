package decorator

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestRunOptsWithIOReaderStdin verifies that RunOpts.Stdin accepts io.Reader
func TestRunOptsWithIOReaderStdin(t *testing.T) {
	data := []byte("test data")
	stdin := bytes.NewReader(data)

	opts := RunOpts{
		Stdin: stdin, // Should compile with io.Reader
	}

	// Verify we can read from stdin
	buf := make([]byte, len(data))
	n, err := opts.Stdin.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.Equal(t, data, buf)
}

// TestRunOptsHasStderrField verifies that RunOpts has Stderr field
func TestRunOptsHasStderrField(t *testing.T) {
	var buf bytes.Buffer
	opts := RunOpts{
		Stderr: &buf, // Should compile
	}

	// Verify we can write to stderr
	_, err := opts.Stderr.Write([]byte("error"))
	assert.NoError(t, err)
	assert.Equal(t, "error", buf.String())
}
