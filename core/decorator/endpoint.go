package decorator

import "io"

// Endpoint is the interface for decorators that provide I/O capabilities.
// Endpoint decorators can act as sources or sinks for data.
// Examples: @file.read, @file.write, @http.get, @s3.put
type Endpoint interface {
	Decorator
	Open(ctx ExecContext, mode IOType) (io.ReadWriteCloser, error)
}

// IOType specifies the I/O mode for endpoint operations.
type IOType string

const (
	// IORead opens endpoint for reading (input source)
	IORead IOType = "read"

	// IOWrite opens endpoint for writing (output sink)
	IOWrite IOType = "write"

	// IODuplex opens endpoint for bidirectional I/O
	IODuplex IOType = "duplex"
)
