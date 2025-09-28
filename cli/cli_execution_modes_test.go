package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCLIExecutionModes tests the 3 modes of CLI operation:
// 1. stdin mode (explicit with -f - or piped input)
// 2. file argument mode (user provides specific file)
// 3. default mode (looks for commands.cli)
func TestCLIExecutionModes(t *testing.T) {
	// Sample command content for testing
	sampleCommands := `test-cmd: echo "hello world"`

	t.Run("StdinMode", func(t *testing.T) {
		t.Run("ExplicitStdinFlag", func(t *testing.T) {
			// Test: opal -f - test-cmd

			// Mock stdin with command content
			oldStdin := os.Stdin
			r, w, err := os.Pipe()
			require.NoError(t, err)
			os.Stdin = r

			go func() {
				defer func() { _ = w.Close() }()
				_, err := w.Write([]byte(sampleCommands))
				assert.NoError(t, err)
			}()

			defer func() { os.Stdin = oldStdin }()

			// Test getInputReader
			reader, closeFunc, err := getInputReader("-")
			require.NoError(t, err)
			defer func() { _ = closeFunc() }()

			content, err := io.ReadAll(reader)
			require.NoError(t, err)

			assert.Equal(t, sampleCommands, string(content))
		})

		t.Run("PipedInputWithDefaultFile", func(t *testing.T) {
			// Test: echo "commands" | opal test-cmd
			// This should use stdin when data is piped and default file is used

			// Create a pipe to simulate piped input
			oldStdin := os.Stdin
			r, w, err := os.Pipe()
			require.NoError(t, err)
			os.Stdin = r

			go func() {
				defer func() { _ = w.Close() }()
				_, err := w.Write([]byte(sampleCommands))
				assert.NoError(t, err)
			}()

			defer func() { os.Stdin = oldStdin }()

			// Test getInputReader
			reader, closeFunc, err := getInputReader("commands.cli")
			require.NoError(t, err)
			defer func() { _ = closeFunc() }()

			content, err := io.ReadAll(reader)
			require.NoError(t, err)

			assert.Equal(t, sampleCommands, string(content))
		})

		t.Run("NoPipedData", func(t *testing.T) {
			// Test that when no data is piped, it doesn't try to read from stdin

			// Create temp directory for testing
			tempDir := t.TempDir()
			oldWd, err := os.Getwd()
			require.NoError(t, err)
			err = os.Chdir(tempDir)
			require.NoError(t, err)
			defer func() { _ = os.Chdir(oldWd) }()

			// No commands.cli file exists, and no piped data
			_, _, err = getInputReader("commands.cli")
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "commands.cli")
		})
	})

	t.Run("FileArgumentMode", func(t *testing.T) {
		t.Run("ExistingFile", func(t *testing.T) {
			// Test: opal -f mycommands.cli test-cmd
			tempDir := t.TempDir()
			testFile := filepath.Join(tempDir, "mycommands.cli")

			err := os.WriteFile(testFile, []byte(sampleCommands), 0o644)
			require.NoError(t, err)

			reader, closeFunc, err := getInputReader(testFile)
			require.NoError(t, err)
			defer func() { _ = closeFunc() }()

			content, err := io.ReadAll(reader)
			require.NoError(t, err)

			assert.Equal(t, sampleCommands, string(content))
		})

		t.Run("NonExistentFile", func(t *testing.T) {
			// Test error handling for non-existent file
			_, _, err := getInputReader("/does/not/exist.cli")
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "error opening file")
			assert.Contains(t, err.Error(), "/does/not/exist.cli")
		})

		t.Run("RelativePathFile", func(t *testing.T) {
			// Test: opal -f ./path/to/commands.cli test-cmd
			tempDir := t.TempDir()
			subDir := filepath.Join(tempDir, "path", "to")
			err := os.MkdirAll(subDir, 0o755)
			require.NoError(t, err)

			testFile := filepath.Join(subDir, "commands.cli")
			err = os.WriteFile(testFile, []byte(sampleCommands), 0o644)
			require.NoError(t, err)

			// Change to temp dir so relative path works
			oldWd, err := os.Getwd()
			require.NoError(t, err)
			err = os.Chdir(tempDir)
			require.NoError(t, err)
			defer func() { _ = os.Chdir(oldWd) }()

			reader, closeFunc, err := getInputReader("./path/to/commands.cli")
			require.NoError(t, err)
			defer func() { _ = closeFunc() }()

			content, err := io.ReadAll(reader)
			require.NoError(t, err)

			assert.Equal(t, sampleCommands, string(content))
		})
	})

	t.Run("DefaultMode", func(t *testing.T) {
		t.Run("CommandsCliExists", func(t *testing.T) {
			// Test: opal test-cmd (looks for commands.cli in current directory)
			tempDir := t.TempDir()
			commandsFile := filepath.Join(tempDir, "commands.cli")

			err := os.WriteFile(commandsFile, []byte(sampleCommands), 0o644)
			require.NoError(t, err)

			// Change to temp dir
			oldWd, err := os.Getwd()
			require.NoError(t, err)
			err = os.Chdir(tempDir)
			require.NoError(t, err)
			defer func() { _ = os.Chdir(oldWd) }()

			reader, closeFunc, err := getInputReader("commands.cli")
			require.NoError(t, err)
			defer func() { _ = closeFunc() }()

			content, err := io.ReadAll(reader)
			require.NoError(t, err)

			assert.Equal(t, sampleCommands, string(content))
		})

		t.Run("CommandsCliMissing", func(t *testing.T) {
			// Test error when commands.cli doesn't exist and no piped input
			tempDir := t.TempDir()

			// Change to temp dir (no commands.cli file)
			oldWd, err := os.Getwd()
			require.NoError(t, err)
			err = os.Chdir(tempDir)
			require.NoError(t, err)
			defer func() { _ = os.Chdir(oldWd) }()

			_, _, err = getInputReader("commands.cli")
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "commands.cli")
		})
	})
}

// TestGetInputReaderEdgeCases tests edge cases and error conditions
func TestGetInputReaderEdgeCases(t *testing.T) {
	t.Run("EmptyStdin", func(t *testing.T) {
		// Test behavior with empty stdin
		oldStdin := os.Stdin
		r, w, err := os.Pipe()
		require.NoError(t, err)
		os.Stdin = r
		_ = w.Close() // Close write end immediately (empty input)

		defer func() { os.Stdin = oldStdin }()

		reader, closeFunc, err := getInputReader("-")
		require.NoError(t, err)
		defer func() { _ = closeFunc() }()

		content, err := io.ReadAll(reader)
		require.NoError(t, err)

		assert.Equal(t, "", string(content))
	})

	t.Run("LargeFile", func(t *testing.T) {
		// Test with a large file
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "large.cli")

		// Create a large command file
		var buf bytes.Buffer
		for i := 0; i < 1000; i++ {
			fmt.Fprintf(&buf, "cmd%d: echo \"command %d\"\n", i, i)
		}
		largeContent := buf.String()

		err := os.WriteFile(testFile, []byte(largeContent), 0o644)
		require.NoError(t, err)

		reader, closeFunc, err := getInputReader(testFile)
		require.NoError(t, err)
		defer func() { _ = closeFunc() }()

		content, err := io.ReadAll(reader)
		require.NoError(t, err)

		assert.Equal(t, largeContent, string(content))
	})

	t.Run("SpecialCharactersInPath", func(t *testing.T) {
		// Test with special characters in file path
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "my commands file.cli")

		err := os.WriteFile(testFile, []byte("test: echo hello"), 0o644)
		require.NoError(t, err)

		reader, closeFunc, err := getInputReader(testFile)
		require.NoError(t, err)
		defer func() { _ = closeFunc() }()

		content, err := io.ReadAll(reader)
		require.NoError(t, err)

		assert.Equal(t, "test: echo hello", string(content))
	})
}

// TestStdinDetection tests the logic for detecting piped input
func TestStdinDetection(t *testing.T) {
	t.Run("StdinStatError", func(t *testing.T) {
		// This test is hard to implement without mocking os.Stdin.Stat()
		// For now, we document that this edge case exists
		t.Skip("Cannot easily test os.Stdin.Stat() error without complex mocking")
	})

	t.Run("StdinModeDetection", func(t *testing.T) {
		// Test that the stdin detection logic works correctly
		// This validates the condition: (stat.Mode()&os.ModeCharDevice) == 0 && stat.Size() > 0

		// When using default file and stdin has piped data, should use stdin

		// Create a pipe to simulate piped input with actual data
		oldStdin := os.Stdin
		r, w, err := os.Pipe()
		require.NoError(t, err)
		os.Stdin = r

		testData := "piped-cmd: echo from pipe"
		go func() {
			defer func() { _ = w.Close() }()
			_, _ = w.Write([]byte(testData))
		}()

		defer func() { os.Stdin = oldStdin }()

		reader, closeFunc, err := getInputReader("commands.cli")
		require.NoError(t, err)
		defer func() { _ = closeFunc() }()

		// Should read from stdin, not try to open commands.cli
		content, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, testData, string(content))
	})
}

// TestFileCloseHandling tests that files are properly closed
func TestFileCloseHandling(t *testing.T) {
	t.Run("FileClosedProperly", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "test.cli")

		err := os.WriteFile(testFile, []byte("test: echo hello"), 0o644)
		require.NoError(t, err)

		reader, closeFunc, err := getInputReader(testFile)
		require.NoError(t, err)

		// Read some content
		content, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, "test: echo hello", string(content))

		// Close should work without error
		err = closeFunc()
		assert.NoError(t, err)
	})

	t.Run("StdinCloseIsNoOp", func(t *testing.T) {
		_, closeFunc, err := getInputReader("-")
		require.NoError(t, err)

		// Closing stdin should be a no-op and not error
		err = closeFunc()
		assert.NoError(t, err)
	})
}
