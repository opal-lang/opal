//go:build !linux && !windows && !darwin

package isolation

import "github.com/builtwithtofu/sigil/core/decorator"

func init() {
	registerIsolator(
		func() decorator.IsolationContext {
			return &unsupportedIsolator{reason: "isolation is not supported on this platform"}
		},
		func() bool { return false },
	)
}
