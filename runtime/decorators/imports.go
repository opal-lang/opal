package decorators

import (
	// Import isolated package to trigger init() registration of isolation decorators
	_ "github.com/opal-lang/opal/runtime/decorators/isolated"
)
