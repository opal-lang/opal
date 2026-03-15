package decorators

import (
	// Register builtin plugin namespaces used by parser/planner/executor coexistence.
	_ "github.com/builtwithtofu/sigil/core/plugin/builtins"
	// Import isolated package to trigger init() registration of isolation decorators
	_ "github.com/builtwithtofu/sigil/runtime/decorators/isolated"
)
