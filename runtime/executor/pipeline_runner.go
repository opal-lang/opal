package executor

import (
	"io"
	"os"
	"sync"

	"github.com/opal-lang/opal/core/invariant"
	"github.com/opal-lang/opal/core/sdk"
)

// executePipelineIO executes a pipeline with optional stdin for the first command
// and optional stdout override for the last command.
func (e *executor) executePipelineIO(execCtx sdk.ExecutionContext, pipeline *sdk.PipelineNode, initialStdin io.Reader, finalStdout io.Writer) int {
	invariant.NotNil(execCtx, "execCtx")
	numCommands := len(pipeline.Commands)
	invariant.Precondition(numCommands > 0, "pipeline must have at least one command")

	if numCommands == 1 {
		return e.executeTreeNode(execCtx, pipeline.Commands[0], initialStdin, finalStdout)
	}

	pipeReaders := make([]*os.File, numCommands-1)
	pipeWriters := make([]*os.File, numCommands-1)
	for i := 0; i < numCommands-1; i++ {
		pr, pw, err := os.Pipe()
		if err != nil {
			for j := 0; j < i; j++ {
				_ = pipeReaders[j].Close()
				_ = pipeWriters[j].Close()
			}
			return 1
		}
		pipeReaders[i] = pr
		pipeWriters[i] = pw
	}

	exitCodes := make([]int, numCommands)
	pipeReaderCloseOnce := make([]sync.Once, numCommands-1)
	pipeWriterCloseOnce := make([]sync.Once, numCommands-1)

	var wg sync.WaitGroup
	wg.Add(numCommands)

	for i := 0; i < numCommands; i++ {
		cmdIndex := i
		node := pipeline.Commands[i]

		go func() {
			defer wg.Done()

			var stdin io.Reader
			if cmdIndex == 0 {
				stdin = initialStdin
			} else {
				stdin = pipeReaders[cmdIndex-1]
				defer func() {
					idx := cmdIndex - 1
					pipeReaderCloseOnce[idx].Do(func() {
						_ = pipeReaders[idx].Close()
					})
				}()
			}

			var stdout io.Writer
			if cmdIndex < numCommands-1 {
				stdout = pipeWriters[cmdIndex]
				defer func() {
					idx := cmdIndex
					pipeWriterCloseOnce[idx].Do(func() {
						_ = pipeWriters[idx].Close()
					})
				}()
			} else if finalStdout != nil {
				stdout = finalStdout
			} else {
				stdout = os.Stdout
			}

			exitCodes[cmdIndex] = e.executeTreeNode(execCtx, node, stdin, stdout)
		}()
	}

	wg.Wait()

	// TODO: Store PIPESTATUS in telemetry for debugging.
	return exitCodes[numCommands-1]
}
