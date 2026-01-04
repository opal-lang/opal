package decorator

import "io"

// IOCaps declares what I/O operations a decorator supports.
// This is used by the parser to validate redirect usage at parse time.
type IOCaps struct {
	// Read supports < (input source)
	Read bool

	// Write supports > (output sink, overwrite/truncate)
	Write bool

	// Append supports >> (output sink, append)
	Append bool

	// Atomic indicates writes use temp file + rename (readers never see partial writes)
	Atomic bool
}

// IOOpts provides optional configuration for I/O operations.
type IOOpts struct {
	// Meta contains sink-specific metadata (e.g., S3 headers, HTTP headers)
	Meta map[string]any
}

// IO is the interface for decorators that support I/O redirection.
// Decorators implement this to be used as redirect sources (<) or sinks (>, >>).
//
// Examples:
//   - @shell("file.txt") - file I/O (internal, users write raw paths)
//   - @aws.s3.object("key") - S3 object (future)
//   - @http.post("url") - HTTP endpoint (future)
//
// The parser validates IOCaps at parse time to catch invalid redirects early:
//
//	echo "hello" > @aws.s3.object("key")  // OK if Write=true
//	echo "hello" >> @aws.s3.object("key") // Error if Append=false
//	cat < @aws.s3.object("key")           // OK if Read=true
type IO interface {
	Decorator

	// IOCaps returns what I/O operations this decorator supports.
	// Called at registration time to populate schema for validation.
	IOCaps() IOCaps

	// OpenRead opens for reading (< source).
	// Only called if IOCaps().Read is true.
	// The returned ReadCloser MUST be closed by the caller.
	OpenRead(ctx ExecContext, opts ...IOOpts) (io.ReadCloser, error)

	// OpenWrite opens for writing (> or >> sink).
	// Only called if IOCaps().Write or IOCaps().Append is true.
	// The append parameter indicates >> (true) vs > (false).
	// The returned WriteCloser MUST be closed by the caller.
	OpenWrite(ctx ExecContext, append bool, opts ...IOOpts) (io.WriteCloser, error)
}

// IOFactory is an optional interface for decorators that need to be
// instantiated with parameters before I/O operations.
//
// This is used for redirect targets where the decorator needs args from the plan:
//
//	echo "hello" > file.txt
//	                ^^^^^^^^ becomes @shell with params["command"]="file.txt"
//
// The planner calls WithParams to create a configured instance before calling
// OpenRead/OpenWrite.
type IOFactory interface {
	IO

	// WithParams creates a new IO instance configured with the given parameters.
	WithParams(params map[string]any) IO
}
