# Replace Commands checks with Tree checks
s/len(plan\.Steps\[0\]\.Commands) != 1/plan.Steps[0].Tree == nil/g
s/len(plan\.Steps\[1\]\.Commands) != 1/plan.Steps[1].Tree == nil/g
s/plan\.Steps\[0\]\.Commands\[0\]\.Args\[0\]\.Val\.Str/getCommandArg(plan.Steps[0].Tree, "command")/g
s/plan\.Steps\[1\]\.Commands\[0\]\.Args\[0\]\.Val\.Str/getCommandArg(plan.Steps[1].Tree, "command")/g
