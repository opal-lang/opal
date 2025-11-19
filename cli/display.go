package main

import (
	"io"

	"github.com/opal-lang/opal/core/planfmt"
	"github.com/opal-lang/opal/core/planfmt/formatter"
)

// DisplayPlan renders a plan as a tree structure
// This is a thin wrapper around formatter.FormatTree for backwards compatibility
func DisplayPlan(w io.Writer, plan *planfmt.Plan, useColor bool) {
	formatter.FormatTree(w, plan, useColor)
}
