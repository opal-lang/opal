package parser

import (
	"testing"
)

func TestBasicCommands(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "simple command",
			Input: "build: echo hello",
			Expected: Program(
				Cmd("build", "echo hello"),
			),
		},
		{
			Name:  "command with special characters",
			Input: "run: echo 'Hello, World!'",
			Expected: Program(
				Cmd("run", "echo 'Hello, World!'"),
			),
		},
		{
			Name:  "empty command",
			Input: "noop:",
			Expected: Program(
				Cmd("noop", ""),
			),
		},
		{
			Name:  "command with parentheses",
			Input: "check: (echo test)",
			Expected: Program(
				Cmd("check", "(echo test)"),
			),
		},
		{
			Name:  "command with pipes",
			Input: "process: echo hello | grep hello",
			Expected: Program(
				Cmd("process", "echo hello | grep hello"),
			),
		},
		{
			Name:  "command with redirection",
			Input: "save: echo hello > output.txt",
			Expected: Program(
				Cmd("save", "echo hello > output.txt"),
			),
		},
		{
			Name:  "command with background process",
			Input: "background: sleep 10 &",
			Expected: Program(
				Cmd("background", "sleep 10 &"),
			),
		},
		{
			Name:  "command with logical operators",
			Input: "conditional: test -f file.txt && echo exists || echo missing",
			Expected: Program(
				Cmd("conditional", "test -f file.txt && echo exists || echo missing"),
			),
		},
		{
			Name:  "command with environment variables",
			Input: "env-test: NODE_ENV=production npm start",
			Expected: Program(
				Cmd("env-test", "NODE_ENV=production npm start"),
			),
		},
		{
			Name:  "command with complex shell syntax",
			Input: "complex: for i in 1 2 3; do echo $i; done",
			Expected: Program(
				Cmd("complex", "for i in 1 2 3; do echo $i; done"),
			),
		},
		{
			Name:  "command with tabs and mixed whitespace",
			Input: "build:\t\techo\t\"building\" \t&& \tmake",
			Expected: Program(
				Cmd("build", "echo\t\"building\" \t&& \tmake"),
			),
		},
		{
			Name:  "command name with underscores and hyphens",
			Input: "test_command-name: echo hello",
			Expected: Program(
				Cmd("test_command-name", "echo hello"),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}

func TestWatchStopCommands(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "simple watch command",
			Input: "watch server: npm start",
			Expected: Program(
				Watch("server", "npm start"),
			),
		},
		{
			Name:  "simple stop command",
			Input: "stop server: pkill node",
			Expected: Program(
				Stop("server", "pkill node"),
			),
		},
		{
			Name:  "watch command with @var() - gets syntax sugar",
			Input: "watch server: go run @var(MAIN_FILE) --port=@var(PORT)",
			Expected: Program(
				Watch("server", Simple(
					Text("go run "),
					At("var", Id("MAIN_FILE")),
					Text(" --port="),
					At("var", Id("PORT")),
				)),
			),
		},
		{
			Name:  "watch block command",
			Input: "watch dev: { npm start; go run main.go }",
			Expected: Program(
				WatchBlock("dev",
					Shell("npm start; go run main.go"),
				),
			),
		},
		{
			Name:  "watch with timeout decorator",
			Input: "watch build: @timeout(60s) { npm run watch:build }",
			Expected: Program(
				WatchBlock("build",
					DecoratedShell(Decorator("timeout", Dur("60s")),
						Text("npm run watch:build"),
					),
				),
			),
		},
		{
			Name:  "watch with parallel decorator",
			Input: "watch services: @parallel { npm run api; npm run worker; npm run scheduler }",
			Expected: Program(
				WatchBlock("services",
					DecoratedShell(Decorator("parallel"),
						Text("npm run api; npm run worker; npm run scheduler"),
					),
				),
			),
		},
		{
			Name:  "stop with cleanup block",
			Input: "stop services: { pkill -f node; docker stop $(docker ps -q); echo cleaned }",
			Expected: Program(
				StopBlock("services",
					Shell("pkill -f node; docker stop $(docker ps -q); echo cleaned"),
				),
			),
		},
		{
			Name:  "watch and stop with same name should be allowed",
			Input: "watch server: npm start\nstop server: pkill node",
			Expected: Program(
				Watch("server", "npm start"),
				Stop("server", "pkill node"),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}

func TestBlockCommands(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "empty block",
			Input: "setup: { }",
			Expected: Program(
				CmdBlock("setup"),
			),
		},
		{
			Name:  "single statement block",
			Input: "setup: { npm install }",
			Expected: Program(
				CmdBlock("setup",
					Shell("npm install"),
				),
			),
		},
		{
			Name:  "multiple statements",
			Input: "setup: { npm install; go mod tidy; echo done }",
			Expected: Program(
				CmdBlock("setup",
					Shell("npm install; go mod tidy; echo done"),
				),
			),
		},
		{
			Name:  "block with @var() references",
			Input: "build: { cd @var(SRC); make @var(TARGET) }",
			Expected: Program(
				CmdBlock("build",
					Shell("cd ", At("var", Id("SRC")), "; make ", At("var", Id("TARGET"))),
				),
			),
		},
		{
			Name:  "block with complex shell statements using alternative syntax",
			Input: "test: { echo start; for i in 1 2 3; do echo $i; done; echo end }",
			Expected: Program(
				CmdBlock("test",
					Shell("echo start; for i in 1 2 3; do echo $i; done; echo end"),
				),
			),
		},
		{
			Name:  "block with conditional statements",
			Input: "conditional: { test -f file.txt && echo exists || echo missing; echo checked }",
			Expected: Program(
				CmdBlock("conditional",
					Shell("test -f file.txt && echo exists || echo missing; echo checked"),
				),
			),
		},
		{
			Name:  "block with background processes",
			Input: "background: { server &; client &; wait }",
			Expected: Program(
				CmdBlock("background",
					Shell("server &; client &; wait"),
				),
			),
		},
		{
			Name:  "block with mixed @var() and shell text",
			Input: "deploy: { echo \"Deploying @var(APP_NAME) to @var(ENVIRONMENT)\"; kubectl apply -f @var(CONFIG_FILE) }",
			Expected: Program(
				CmdBlock("deploy",
					Shell("echo \"Deploying ", At("var", Id("APP_NAME")), " to ", At("var", Id("ENVIRONMENT")), "\"; kubectl apply -f ", At("var", Id("CONFIG_FILE"))),
				),
			),
		},
		{
			Name:  "block with decorator",
			Input: "services: @parallel { server; client }",
			Expected: Program(
				CmdBlock("services",
					DecoratedShell(Decorator("parallel"),
						Text("server; client"),
					),
				),
			),
		},
		{
			Name:  "block with timeout decorator",
			Input: "deploy: @timeout(5m) { npm run build; npm run deploy }",
			Expected: Program(
				CmdBlock("deploy",
					DecoratedShell(Decorator("timeout", Dur("5m")),
						Text("npm run build; npm run deploy"),
					),
				),
			),
		},
		{
			Name:  "block with retry decorator",
			Input: "flaky-task: @retry(3) { npm test }",
			Expected: Program(
				CmdBlock("flaky-task",
					DecoratedShell(Decorator("retry", Num(3)),
						Text("npm test"),
					),
				),
			),
		},
		{
			Name:  "timeout decorator with complex command",
			Input: "complex: @timeout(30s) { npm run integration-tests && npm run e2e }",
			Expected: Program(
				CmdBlock("complex",
					DecoratedShell(Decorator("timeout", Dur("30s")),
						Text("npm run integration-tests && npm run e2e"),
					),
				),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}

func TestCommandsWithVariables(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "simple variable usage - gets syntax sugar",
			Input: "build: echo @var(MESSAGE)",
			Expected: Program(
				Cmd("build", Simple(
					Text("echo "),
					At("var", Id("MESSAGE")),
				)),
			),
		},
		{
			Name:  "multiple variables in command - gets syntax sugar",
			Input: "deploy: docker run --name @var(CONTAINER) -p @var(PORT):@var(PORT) @var(IMAGE)",
			Expected: Program(
				Cmd("deploy", Simple(
					Text("docker run --name "),
					At("var", Id("CONTAINER")),
					Text(" -p "),
					At("var", Id("PORT")),
					Text(":"),
					At("var", Id("PORT")),
					Text(" "),
					At("var", Id("IMAGE")),
				)),
			),
		},
		{
			Name:  "variable in quoted string - gets syntax sugar",
			Input: "msg: echo \"Hello @var(NAME), welcome to @var(APP)!\"",
			Expected: Program(
				Cmd("msg", Simple(
					Text("echo \"Hello "),
					At("var", Id("NAME")),
					Text(", welcome to "),
					At("var", Id("APP")),
					Text("!\""),
				)),
			),
		},
		{
			Name:  "variable with file paths - gets syntax sugar",
			Input: "copy: cp @var(SRC)/* @var(DEST)/",
			Expected: Program(
				Cmd("copy", Simple(
					Text("cp "),
					At("var", Id("SRC")),
					Text("/* "),
					At("var", Id("DEST")),
					Text("/"),
				)),
			),
		},
		{
			Name:  "variable in complex shell command - gets syntax sugar",
			Input: "check: test -f @var(CONFIG_FILE) && echo \"Config exists\" || echo \"Missing config\"",
			Expected: Program(
				Cmd("check", Simple(
					Text("test -f "),
					At("var", Id("CONFIG_FILE")),
					Text(" && echo \"Config exists\" || echo \"Missing config\""),
				)),
			),
		},
		{
			Name:  "variable with email-like text - gets syntax sugar",
			Input: "notify: echo \"Build @var(STATUS)\" | mail admin@company.com",
			Expected: Program(
				Cmd("notify", Simple(
					Text("echo \"Build "),
					At("var", Id("STATUS")),
					Text("\" | mail admin@company.com"),
				)),
			),
		},
		{
			Name:  "variable in environment setting - gets syntax sugar",
			Input: "serve: NODE_ENV=@var(ENV) npm start",
			Expected: Program(
				Cmd("serve", Simple(
					Text("NODE_ENV="),
					At("var", Id("ENV")),
					Text(" npm start"),
				)),
			),
		},
		{
			Name:  "variable in URL - gets syntax sugar",
			Input: "api-call: curl https://api.example.com/@var(ENDPOINT)?token=@var(TOKEN)",
			Expected: Program(
				Cmd("api-call", Simple(
					Text("curl https://api.example.com/"),
					At("var", Id("ENDPOINT")),
					Text("?token="),
					At("var", Id("TOKEN")),
				)),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}

// TestRealWorldFormatCommand tests parsing of the failing format command from commands.cli
func TestRealWorldFormatCommand(t *testing.T) {
	testCase := TestCase{
		Name: "Real world format command with parallel decorator",
		Input: `# Format all code
format: {
    echo "📝 Formatting all code..."
    echo "Formatting Go code..."
    @parallel {
        if command -v gofumpt >/dev/null 2>&1; then gofumpt -w .; else go fmt ./...; fi
        if command -v nixpkgs-fmt >/dev/null 2>&1; then find . -name '*.nix' -exec nixpkgs-fmt {} +; else echo "⚠️  nixpkgs-fmt not available"; fi
    }
    echo "✅ Code formatted!"
}`,
		Expected: Program(
			CmdBlock("format",
				Shell("echo \"📝 Formatting all code...\""),
				Shell("echo \"Formatting Go code...\""),
				BlockDecorator("parallel",
					"if command -v gofumpt >/dev/null 2>&1; then gofumpt -w .; else go fmt ./...; fi",
					"if command -v nixpkgs-fmt >/dev/null 2>&1; then find . -name '*.nix' -exec nixpkgs-fmt {} +; else echo \"⚠️  nixpkgs-fmt not available\"; fi",
				),
				Shell("echo \"✅ Code formatted!\""),
			),
		),
	}

	RunTestCase(t, testCase)
}
