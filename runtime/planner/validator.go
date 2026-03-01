package planner

import (
	"fmt"
	"strings"

	"github.com/builtwithtofu/sigil/core/invariant"
	"github.com/builtwithtofu/sigil/core/planfmt"
)

func validateTransportID(id string) error {
	if id == "" {
		return fmt.Errorf("transport ID must not be empty")
	}

	if isReservedTransportID(id) {
		return fmt.Errorf("transport ID %q is reserved", id)
	}

	return nil
}

func validateTransportTable(transports []planfmt.Transport) error {
	seen := make(map[string]struct{}, len(transports))
	invariant.Check(seen != nil, "transport ID set must be initialized")

	for _, transport := range transports {
		if err := validateTransportID(transport.ID); err != nil {
			return err
		}

		if _, exists := seen[transport.ID]; exists {
			return fmt.Errorf("duplicate transport ID %q", transport.ID)
		}

		seen[transport.ID] = struct{}{}
	}

	return nil
}

func isReservedTransportID(id string) bool {
	switch id {
	case "local", "root":
		return true
	default:
		return false
	}
}

func detectCycle(transports []planfmt.Transport) error {
	parents := make(map[string]string, len(transports))
	for _, transport := range transports {
		parents[transport.ID] = transport.ParentID
	}

	const (
		unvisited = iota
		visiting
		visited
	)

	state := make(map[string]int, len(parents))
	stack := make([]string, 0, len(parents))
	stackIndex := make(map[string]int, len(parents))

	var visit func(id string) error
	visit = func(id string) error {
		switch state[id] {
		case visited:
			return nil
		case visiting:
			start := stackIndex[id]
			cycle := append([]string{}, stack[start:]...)
			cycle = append(cycle, id)
			return fmt.Errorf("transport cycle detected: %s", strings.Join(cycle, " -> "))
		}

		state[id] = visiting
		stackIndex[id] = len(stack)
		stack = append(stack, id)

		parentID := parents[id]
		if _, exists := parents[parentID]; exists {
			if err := visit(parentID); err != nil {
				return err
			}
		}

		stack = stack[:len(stack)-1]
		delete(stackIndex, id)
		state[id] = visited
		return nil
	}

	for _, transport := range transports {
		if err := visit(transport.ID); err != nil {
			return err
		}
	}

	return nil
}
