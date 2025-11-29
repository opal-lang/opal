package decorators

import (
	"fmt"

	"github.com/opal-lang/opal/core/decorator"
)

// Register @test.transport decorator with the global registry.
// This is a mock transport for testing transport boundaries without real SSH/Docker.
func init() {
	if err := decorator.Register("test.transport", decorator.NewTestTransport("test")); err != nil {
		panic(fmt.Sprintf("failed to register @test.transport decorator: %v", err))
	}
}
