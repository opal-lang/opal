package executor

import "fmt"

type SinkError struct {
	SinkID      string
	Operation   string
	TransportID string
	Cause       error
}

func (e SinkError) Error() string {
	if e.Cause == nil {
		return fmt.Sprintf("sink %s %s failed on transport %s", e.SinkID, e.Operation, e.TransportID)
	}
	return fmt.Sprintf("sink %s %s failed on transport %s: %v", e.SinkID, e.Operation, e.TransportID, e.Cause)
}
