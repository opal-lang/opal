package parser

import (
	"testing"
)

func TestComplexShellCommands(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "simple shell command substitution",
			Input: `test-simple: echo "$(date)"`,
			Expected: Program(
				Cmd("test-simple", Shell(
					Text("echo "),
					StrPart("$(date)"),
				)),
			),
		},
		{
			Name:  "shell command with test and command substitution",
			Input: `test-condition: if [ "$(echo test)" = "test" ]; then echo ok; fi`,
			Expected: Program(
				Cmd("test-condition", Shell(
					Text("if [ "),
					StrPart("$(echo test)"),
					Text(" = "),
					StrPart("test"),
					Text(" ]; then echo ok; fi"),
				)),
			),
		},
		{
			Name: "command with @var and shell substitution",
			Input: `var SRC = "./src"
test-mixed: cd @var(SRC) && echo "files: $(ls | wc -l)"`,
			Expected: Program(
				Var("SRC", Str("./src")),
				Cmd("test-mixed", Chain(
					Shell(
						Text("cd "),
						At("var", Id("SRC")),
					),
					And(Shell(
						Text(" echo "),
						StrPart("files: $(ls | wc -l)"),
					)),
				)),
			),
		},
		{
			Name:  "simplified version of failing command",
			Input: `test-format: if [ "$(gofumpt -l . | wc -l)" -gt 0 ]; then echo "issues"; fi`,
			Expected: Program(
				Cmd("test-format", Shell(
					Text("if [ "),
					StrPart("$(gofumpt -l . | wc -l)"),
					Text(" -gt 0 ]; then echo "),
					StrPart("issues"),
					Text("; fi"),
				)),
			),
		},
		{
			Name: "backup command with shell substitution and @var",
			Input: `var KUBE_NAMESPACE = "default"
backup: {
        echo "Creating backup..."
        DATE=$(date +%Y%m%d-%H%M%S); echo "Backup timestamp: $DATE"
        (which kubectl && kubectl exec deployment/database -n @var(KUBE_NAMESPACE) -- pg_dump myapp > backup-$(date +%Y%m%d-%H%M%S).sql) || echo "No database"
      }`,
			Expected: Program(
				Var("KUBE_NAMESPACE", Str("default")),
				CmdBlock("backup",
					Shell(
						Text("echo "),
						StrPart("Creating backup..."),
					),
					Shell(Text("DATE=$(date +%Y%m%d-%H%M%S); echo "), StrPart("Backup timestamp: $DATE")),
					Chain(
						Shell(Text("(which kubectl")),
						And(Shell(
							Text(" kubectl exec deployment/database -n "),
							At("var", Id("KUBE_NAMESPACE")),
							Text("-- pg_dump myapp > backup-$(date +%Y%m%d-%H%M%S).sql)"),
						)),
						Or(Shell(
							Text(" echo "),
							StrPart("No database"),
						)),
					),
				),
			),
		},
		{
			Name: "exact command from real commands.cli file",
			Input: `test-quick: {
    echo "⚡ Running quick checks..."
    echo "🔍 Checking Go formatting..."
    if command -v gofumpt >/dev/null 2>&1; then if [ "$(gofumpt -l . | wc -l)" -gt 0 ]; then echo "❌ Go formatting issues:"; gofumpt -l .; exit 1; fi; else if [ "$(gofmt -l . | wc -l)" -gt 0 ]; then echo "❌ Go formatting issues:"; gofmt -l .; exit 1; fi; fi
    echo "🔍 Checking Nix formatting..."
    if command -v nixpkgs-fmt >/dev/null 2>&1; then nixpkgs-fmt --check . || (echo "❌ Run 'dev format' to fix"; exit 1); else echo "⚠️  nixpkgs-fmt not available, skipping Nix format check"; fi
    dev lint
    echo "✅ Quick checks passed!"
}`,
			Expected: Program(
				CmdBlock("test-quick",
					// Each line in the block becomes a separate shell command
					Shell(
						Text("echo "),
						StrPart("⚡ Running quick checks..."),
					),
					Shell(
						Text("echo "),
						StrPart("🔍 Checking Go formatting..."),
					),
					Shell(
						Text("if command -v gofumpt >/dev/null 2>&1; then if [ "),
						StrPart("$(gofumpt -l . | wc -l)"),
						Text(" -gt 0 ]; then echo "),
						StrPart("❌ Go formatting issues:"),
						Text("; gofumpt -l .; exit 1; fi; else if [ "),
						StrPart("$(gofmt -l . | wc -l)"),
						Text(" -gt 0 ]; then echo "),
						StrPart("❌ Go formatting issues:"),
						Text("; gofmt -l .; exit 1; fi; fi"),
					),
					Shell(
						Text("echo "),
						StrPart("🔍 Checking Nix formatting..."),
					),
					Chain(
						Shell(Text("if command -v nixpkgs-fmt >/dev/null 2>&1; then nixpkgs-fmt --check .")),
						Or(Shell(
							Text(" (echo "),
							StrPart("❌ Run 'dev format' to fix"),
							Text("; exit 1); else echo "),
							StrPart("⚠️  nixpkgs-fmt not available, skipping Nix format check"),
							Text("; fi"),
						)),
					),
					Shell(Text("dev lint")),
					Shell(
						Text("echo "),
						StrPart("✅ Quick checks passed!"),
					),
				),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}

func TestVarInShellCommands(t *testing.T) {
	testCases := []TestCase{
		{
			Name: "simple @var in shell command",
			Input: `var DIR = "/home/user"
test-var: cd @var(DIR)`,
			Expected: Program(
				Var("DIR", Str("/home/user")),
				Cmd("test-var", Simple(
					Text("cd "),
					At("var", Id("DIR")),
				)),
			),
		},
		{
			Name: "@var with shell command substitution",
			Input: `var DIR = "/project"
test-var-cmd: cd @var(DIR) && echo "$(pwd)"`,
			Expected: Program(
				Var("DIR", Str("/project")),
				Cmd("test-var-cmd", Chain(
					Shell(
						Text("cd "),
						At("var", Id("DIR")),
					),
					And(Shell(
						Text(" echo "),
						StrPart("$(pwd)"),
					)),
				)),
			),
		},
		{
			Name: "multiple @var with complex shell",
			Input: `var SRC = "./src"
test-multi-var: if [ -d @var(SRC) ] && [ "$(ls @var(SRC) | wc -l)" -gt 0 ]; then echo "Source dir has files"; fi`,
			Expected: Program(
				Var("SRC", Str("./src")),
				Cmd("test-multi-var", Chain(
					Shell(
						Text("if [ -d "),
						At("var", Id("SRC")),
						Text("]"),
					),
					And(Shell(
						Text(" [ "),
						StrPart("$(ls "),
						At("var", Id("SRC")),
						StrPart(" | wc -l)"),
						Text(" -gt 0 ]; then echo "),
						StrPart("Source dir has files"),
						Text("; fi"),
					)),
				)),
			),
		},
		{
			Name: "@var in shell array",
			Input: `var (
    FILE1 = "file1.txt"
    FILE2 = "file2.txt"
    FILE3 = "file3.txt"
)
array-var: FILES=(@var(FILE1) @var(FILE2) @var(FILE3)); echo ${FILES[@]}`,
			Expected: Program(
				Var("FILE1", Str("file1.txt")),
				Var("FILE2", Str("file2.txt")),
				Var("FILE3", Str("file3.txt")),
				Cmd("array-var", Simple(
					Text("FILES=("),
					At("var", Id("FILE1")),
					Text(" "),
					At("var", Id("FILE2")),
					Text(" "),
					At("var", Id("FILE3")),
					Text("); echo ${FILES[@]}"),
				)),
			),
		},
		{
			Name: "@var in shell case statement",
			Input: `var ENV = "production"
case-var: case @var(ENV) in prod) echo production;; dev) echo development;; esac`,
			Expected: Program(
				Var("ENV", Str("production")),
				Cmd("case-var", Simple(
					Text("case "),
					At("var", Id("ENV")),
					Text(" in prod) echo production;; dev) echo development;; esac"),
				)),
			),
		},
		{
			Name: "@var in shell parameter expansion",
			Input: `var (
    VAR = "myvalue"
    DEFAULT = "fallback"
)
param-expansion: echo ${@var(VAR):-@var(DEFAULT)}`,
			Expected: Program(
				Var("VAR", Str("myvalue")),
				Var("DEFAULT", Str("fallback")),
				Cmd("param-expansion", Simple(
					Text("echo ${"),
					At("var", Id("VAR")),
					Text(":-"),
					At("var", Id("DEFAULT")),
					Text("}"),
				)),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}

func TestLineContinuationEdgeCases(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "continuation at end of line with no following content",
			Input: "test: echo hello \\\n",
			Expected: Program(
				Cmd("test", Simple(Text("echo hello"))),
			),
		},
		{
			Name:  "continuation with only whitespace on next line",
			Input: "test: echo hello \\\n   ",
			Expected: Program(
				Cmd("test", Simple(Text("echo hello"))),
			),
		},
		{
			Name:  "continuation with tab characters",
			Input: "test: echo hello \\\n\tworld",
			Expected: Program(
				Cmd("test", Simple(Text("echo hello world"))),
			),
		},
		{
			Name:  "continuation in middle of quoted string",
			Input: `test: echo "hello \` + "\n" + `world"`,
			Expected: Program(
				Cmd("test", Simple(
					Text("echo "),
					StrPart("hello \\\nworld"),
				)),
			),
		},
		{
			Name:  "continuation in single quotes (should be literal)",
			Input: "test: echo 'hello \\\nworld'",
			Expected: Program(
				Cmd("test", Simple(
					Text("echo "),
					StrPart("hello \\\nworld"),
				)),
			),
		},
		{
			Name: "continuation with @var across lines",
			Input: `var NAME = "testuser"
test: echo \
@var(NAME) \
is here`,
			Expected: Program(
				Var("NAME", Str("testuser")),
				Cmd("test", Simple(
					Text("echo "),
					At("var", Id("NAME")),
					Text(" is here"),
				)),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}

func TestContinuationLines(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "simple continuation",
			Input: "build: echo hello \\\nworld",
			Expected: Program(
				Cmd("build", Simple(Text("echo hello world"))),
			),
		},
		{
			Name: "continuation with @var()",
			Input: `var DIR = "/project"
build: cd @var(DIR) \
&& make`,
			Expected: Program(
				Var("DIR", Str("/project")),
				Cmd("build", Chain(
					Shell(Text("cd "), At("var", Id("DIR"))),
					And(Shell(Text(" make"))),
				)),
			),
		},
		{
			Name:  "multiple line continuations",
			Input: "complex: echo start \\\n&& echo middle \\\n&& echo end",
			Expected: Program(
				Cmd("complex", Chain(
					Shell(Text("echo start")),
					And(Shell(Text("echo middle"))),
					And(Shell(Text(" echo end"))),
				)),
			),
		},
		{
			Name:  "continuation in block",
			Input: "block: { echo hello \\\nworld; echo done }",
			Expected: Program(
				CmdBlock("block",
					Shell("echo hello world; echo done"),
				),
			),
		},
		{
			Name: "continuation with mixed content",
			Input: `var (
    IMAGE = "myapp:latest"
    PORT = 8080
)
mixed: docker run \
@var(IMAGE) \
--port=@var(PORT)`,
			Expected: Program(
				Var("IMAGE", Str("myapp:latest")),
				Var("PORT", Num(8080)),
				Cmd("mixed", Simple(
					Text("docker run "),
					At("var", Id("IMAGE")),
					Text(" --port="),
					At("var", Id("PORT")),
				)),
			),
		},
		{
			Name:  "continuation with trailing spaces",
			Input: "trailing: echo hello \\\n   world",
			Expected: Program(
				Cmd("trailing", Simple(Text("echo hello world"))),
			),
		},
		{
			Name:  "continuation breaking long docker command",
			Input: "docker: docker run \\\n--name myapp \\\n--port 8080:8080 \\\n--env NODE_ENV=production \\\nmyimage:latest",
			Expected: Program(
				Cmd("docker", Simple(Text("docker run --name myapp --port 8080:8080 --env NODE_ENV=production myimage:latest"))),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}
