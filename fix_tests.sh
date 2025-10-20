#!/bin/bash
# Fix all test files to use Tree instead of Commands

# List of test files to fix
FILES="
core/planfmt/formatter/tree_test.go
core/planfmt/formatter/text_test.go
core/planfmt/formatter/diff_test.go
cli/display_test.go
runtime/planner/planner_test.go
runtime/planner/tree_builder_test.go
runtime/planner/tree_builder.go
runtime/executor/executor_test.go
"

for file in $FILES; do
  if [ -f "$file" ]; then
    echo "Fixing $file..."
    # Replace planfmt.Command with Command (for planner package)
    if [[ "$file" == *"planner"* ]]; then
      sed -i 's/\[\]planfmt\.Command/[]Command/g' "$file"
      sed -i 's/planfmt\.Command{/Command{/g' "$file"
    fi
  fi
done

echo "Done!"
