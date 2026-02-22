package decorator

import "fmt"

const (
	TransportErrorCodeConnect          = "TRANSPORT_CONNECT_FAILED"
	TransportErrorCodeSession          = "TRANSPORT_SESSION_FAILED"
	TransportErrorCodeExecute          = "TRANSPORT_EXECUTE_FAILED"
	TransportErrorCodeIO               = "TRANSPORT_IO_FAILED"
	TransportErrorCodeContext          = "TRANSPORT_CONTEXT_CANCELLED"
	TransportErrorCodeValidationFailed = "TRANSPORT_VALIDATION_FAILED"
)

type TransportError struct {
	Code      string
	Message   string
	Retryable bool
	Cause     error
}

func (e TransportError) Error() string {
	return fmt.Sprintf("transport [%s]: %s", e.Code, e.Message)
}

func (e TransportError) DetailedError() string {
	if e.Cause == nil {
		return e.Error()
	}
	return fmt.Sprintf("transport [%s]: %s: %v", e.Code, e.Message, e.Cause)
}

func (e TransportError) Unwrap() error {
	return e.Cause
}
