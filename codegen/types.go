package codegen

import (
	"fmt"
	"sync/atomic"
)

// simpleTempResult is a basic implementation of TempResult
type simpleTempResult struct {
	id   string
	code string
}

func (s *simpleTempResult) ID() string {
	return s.id
}

func (s *simpleTempResult) String() string {
	return s.code
}

// NewTempResult creates a new temporary result with a unique ID
func NewTempResult(code string) TempResult {
	return &simpleTempResult{
		id:   nextID(),
		code: code,
	}
}

// NewTempResultWithID creates a temporary result with a specific ID
func NewTempResultWithID(id, code string) TempResult {
	return &simpleTempResult{
		id:   id,
		code: code,
	}
}

// ID counter for unique temp result IDs
var idCounter int64

// nextID generates a unique identifier for temp results
func nextID() string {
	id := atomic.AddInt64(&idCounter, 1)
	return fmt.Sprintf("temp_%d", id)
}
