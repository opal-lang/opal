package parser

import (
	"testing"
)

func TestVarDecorators(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "simple @var() reference - gets syntax sugar in simple command",
			Input: "build: cd @var(SRC)",
			Expected: Program(
				Cmd("build", Simple(
					Text("cd "),
					At("var", Id("SRC")),
				)),
			),
		},
		{
			Name:  "multiple @var() references - gets syntax sugar in simple command",
			Input: "deploy: docker build -t @var(IMAGE):@var(TAG)",
			Expected: Program(
				Cmd("deploy", Simple(
					Text("docker build -t "),
					At("var", Id("IMAGE")),
					Text(":"),
					At("var", Id("TAG")),
				)),
			),
		},
		{
			Name:  "@var() in double quoted string - should still be parsed as decorators",
			Input: "echo: echo \"Building @var(PROJECT) version @var(VERSION)\"",
			Expected: Program(
				Cmd("echo", Shell(
					Text("echo "),
					StrPart("Building "),
					At("var", Id("PROJECT")),
					StrPart(" version "),
					At("var", Id("VERSION")),
				)),
			),
		},
		{
			Name:  "mixed @var() and shell variables - gets syntax sugar in simple command",
			Input: "info: echo \"Project: @var(NAME), User: $USER\"",
			Expected: Program(
				Cmd("info", Shell(
					Text("echo "),
					StrPart("Project: "),
					At("var", Id("NAME")),
					StrPart(", User: $USER"),
				)),
			),
		},
		{
			Name:  "@var() in file paths - gets syntax sugar in simple command",
			Input: "copy: cp @var(SRC)/*.go @var(DEST)/",
			Expected: Program(
				Cmd("copy", Simple(
					Text("cp "),
					At("var", Id("SRC")),
					Text("/*.go "),
					At("var", Id("DEST")),
					Text("/"),
				)),
			),
		},
		{
			Name:  "@var() in command arguments - gets syntax sugar in simple command",
			Input: "serve: go run main.go --port=@var(PORT) --host=@var(HOST)",
			Expected: Program(
				Cmd("serve", Simple(
					Text("go run main.go --port="),
					At("var", Id("PORT")),
					Text(" --host="),
					At("var", Id("HOST")),
				)),
			),
		},
		{
			Name:  "@var() with special characters in value - gets syntax sugar in simple command",
			Input: "url: curl \"@var(API_URL)/users?filter=@var(FILTER)\"",
			Expected: Program(
				Cmd("url", Shell(
					Text("curl "),
					At("var", Id("API_URL")),
					StrPart("/users?filter="),
					At("var", Id("FILTER")),
				)),
			),
		},
		{
			Name:  "@var() in conditional expressions - gets syntax sugar in simple command",
			Input: "check: test \"@var(ENV)\" = \"production\" && echo prod || echo dev",
			Expected: Program(
				Cmd("check", Chain(
					Shell(
						Text("test "),
						At("var", Id("ENV")),
						Text(" = "),
						StrPart("production"),
					),
					And(Shell(Text("echo prod"))),
					Or(Shell(Text(" echo dev"))),
				)),
			),
		},
		{
			Name:  "@var() in loops - gets syntax sugar in simple command",
			Input: "process: for file in @var(SRC)/*.txt; do process $file; done",
			Expected: Program(
				Cmd("process", Simple(
					Text("for file in "),
					At("var", Id("SRC")),
					Text("/*.txt; do process $file; done"),
				)),
			),
		},
		{
			Name:  "string with escaped quotes and @var - gets syntax sugar in simple command",
			Input: "msg: echo \"He said \\\"Hello @var(NAME)\\\" to everyone\"",
			Expected: Program(
				Cmd("msg", Shell(
					Text("echo "),
					StrPart("He said \\"),
					Text("Hello "),
					At("var", Id("NAME")),
					Text("\\"),
					StrPart(" to everyone"),
				)),
			),
		},
		{
			Name:  "@var() in explicit block",
			Input: "build: { cd @var(SRC); make @var(TARGET) }",
			Expected: Program(
				CmdBlock("build",
					Shell("cd ", At("var", Id("SRC")), "; make ", At("var", Id("TARGET"))),
				),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}

func TestEnvDecorators(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "simple @env() reference - gets syntax sugar in simple command",
			Input: "deploy: kubectl config use-context @env(\"KUBE_CONTEXT\")",
			Expected: Program(
				Cmd("deploy", Simple(
					Text("kubectl config use-context "),
					At("env", Str("KUBE_CONTEXT")),
				)),
			),
		},
		{
			Name:  "multiple @env() references - gets syntax sugar in simple command",
			Input: "status: echo Context: @env(KUBE_CONTEXT), Project: @env(PROJECT_ID)",
			Expected: Program(
				Cmd("status", Simple(
					Text("echo Context: "),
					At("env", Id("KUBE_CONTEXT")),
					Text(", Project: "),
					At("env", Id("PROJECT_ID")),
				)),
			),
		},
		{
			Name:  "@env() in explicit block",
			Input: "deploy: { kubectl config use-context @env(\"KUBE_CONTEXT\"); kubectl apply -f k8s/ }",
			Expected: Program(
				CmdBlock("deploy",
					Shell("kubectl config use-context ", At("env", Str("KUBE_CONTEXT")), "; kubectl apply -f k8s/"),
				),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}

func TestBlockDecorators(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "valid @timeout block decorator",
			Input: "deploy: @timeout(30s) { echo deploying }",
			Expected: Program(
				CmdBlock("deploy",
					DecoratedShell(Decorator("timeout", Dur("30s")),
						Text("echo deploying"),
					),
				),
			),
		},
		{
			Name:  "valid @confirm decorator",
			Input: "dangerous: @confirm(\"Are you sure?\") { rm -rf /tmp/* }",
			Expected: Program(
				CmdBlock("dangerous",
					DecoratedShell(Decorator("confirm", Str("Are you sure?")),
						Text("rm -rf /tmp/*"),
					),
				),
			),
		},
		{
			Name:  "valid @debounce decorator",
			Input: "watch-changes: @debounce(500ms) { npm run build }",
			Expected: Program(
				CmdBlock("watch-changes",
					DecoratedShell(Decorator("debounce", Dur("500ms")),
						Text("npm run build"),
					),
				),
			),
		},
		{
			Name:  "valid @cwd decorator",
			Input: "build-lib: @cwd(\"./lib\") { make all }",
			Expected: Program(
				CmdBlock("build-lib",
					DecoratedShell(Decorator("cwd", Str("./lib")),
						Text("make all"),
					),
				),
			),
		},
		{
			Name:  "valid @parallel block decorator with multiple statements",
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
			Name:  "valid @retry block decorator with multiple statements",
			Input: "flaky-test: @retry(3) { npm test; echo 'done' }",
			Expected: Program(
				CmdBlock("flaky-test",
					DecoratedShell(Decorator("retry", Num(3)),
						Text("npm test; echo "),
						StrPart("done"),
					),
				),
			),
		},
		{
			Name:  "valid @watch-files block decorator with multiple statements",
			Input: "monitor: @watch-files(\"*.js\") { echo 'checking'; sleep 1 }",
			Expected: Program(
				CmdBlock("monitor",
					DecoratedShell(Decorator("watch-files", Str("*.js")),
						Text("echo "),
						StrPart("checking"),
						Text("; sleep 1"),
					),
				),
			),
		},
		{
			Name:  "empty block with decorators",
			Input: "parallel-empty: @parallel { }",
			Expected: Program(
				CmdBlock("parallel-empty",
					BlockDecorator("parallel"),
				),
			),
		},
		{
			Name: "multiple commands in parallel block - each gets decorated",
			Input: `services: @parallel {
  npm run api
  npm run worker
}`,
			Expected: Program(
				CmdBlock("services",
					BlockDecorator("parallel", "npm run api", "npm run worker"),
				),
			),
		},
		{
			Name: "multiple commands in timeout block - each gets decorated",
			Input: `deploy: @timeout(5m) {
  npm run build
  npm test
  kubectl apply -f k8s/
}`,
			Expected: Program(
				CmdBlock("deploy",
					BlockDecorator("timeout", Dur("5m"), "npm run build", "npm test", "kubectl apply -f k8s/"),
				),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}

func TestPatternDecorators(t *testing.T) {
	testCases := []TestCase{
		{
			Name: "@when pattern decorator with simple branches",
			Input: `deploy: @when("ENV") {
  production: kubectl apply -f k8s/prod/
  staging: kubectl apply -f k8s/staging/
  default: echo "Unknown environment"
}`,
			Expected: Program(
				Cmd("deploy",
					PatternDecoratorWithBranches("when", Str("ENV"),
						Branch("production", Shell("kubectl apply -f k8s/prod/")),
						Branch("staging", Shell("kubectl apply -f k8s/staging/")),
						Branch("default", Shell(Text("echo "), StrPart("Unknown environment"))),
					),
				),
			),
		},
		{
			Name: "@when pattern decorator with multiple commands per branch",
			Input: `deploy: @when("ENV") {
  production: {
    kubectl config use-context prod
    kubectl apply -f k8s/prod/
    kubectl rollout status deployment/app
  }
  staging: kubectl apply -f k8s/staging/
}`,
			Expected: Program(
				Cmd("deploy",
					PatternDecoratorWithBranches("when", Str("ENV"),
						Branch("production",
							Shell("kubectl config use-context prod"),
							Shell("kubectl apply -f k8s/prod/"),
							Shell("kubectl rollout status deployment/app"),
						),
						Branch("staging", Shell("kubectl apply -f k8s/staging/")),
					),
				),
			),
		},
		{
			Name: "@try pattern decorator with error handling",
			Input: `backup: @try {
  main: {
    pg_dump mydb > backup.sql
    aws s3 cp backup.sql s3://backups/
  }
  error: {
    echo "Backup failed"
    rm -f backup.sql
  }
  finally: echo "Backup process completed"
}`,
			Expected: Program(
				Cmd("backup",
					PatternDecoratorWithBranches("try", nil,
						Branch("main",
							Shell("pg_dump mydb > backup.sql"),
							Shell("aws s3 cp backup.sql s3://backups/"),
						),
						Branch("error",
							Shell(Text("echo "), StrPart("Backup failed")),
							Shell("rm -f backup.sql"),
						),
						Branch("finally", Shell(Text("echo "), StrPart("Backup process completed"))),
					),
				),
			),
		},
		{
			Name: "@when with @var references in commands",
			Input: `deploy: @when("MODE") {
  production: echo "Deploying @var(APP) to production"
  staging: echo "Deploying @var(APP) to staging"
}`,
			Expected: Program(
				Cmd("deploy",
					PatternDecoratorWithBranches("when", Str("MODE"),
						Branch("production", Shell(Text("echo "), StrPart("Deploying "), At("var", Id("APP")), StrPart(" to production"))),
						Branch("staging", Shell(Text("echo "), StrPart("Deploying "), At("var", Id("APP")), StrPart(" to staging"))),
					),
				),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}

func TestNamedParameterSupport(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "retry with positional parameter",
			Input: "test: @retry(3) { echo \"task\" }",
			Expected: Program(
				CmdBlock("test",
					BlockDecorator("retry", Num(3),
						Shell(Text("echo "), StrPart("task")),
					),
				),
			),
		},
		{
			Name:  "retry with positional parameter",
			Input: "test: @retry(attempts=3) { echo \"task\" }",
			Expected: Program(
				CmdBlock("test",
					BlockDecorator("retry", Named("attempts", Num(3)), Shell(Text("echo "), StrPart("task"))),
				),
			),
		},
		{
			Name:  "retry with named parameter",
			Input: "test: @retry(attempts=3) { echo \"task\" }",
			Expected: Program(
				CmdBlock("test",
					BlockDecorator("retry", Named("attempts", Num(3)), Shell(Text("echo "), StrPart("task"))),
				),
			),
		},
		{
			Name:  "retry with mixed parameters",
			Input: "test: @retry(attempts=3, delay=1s) { echo \"task\" }",
			Expected: Program(
				CmdBlock("test",
					BlockDecorator("retry", Named("attempts", Num(3)), Named("delay", Dur("1s")), Shell(Text("echo "), StrPart("task"))),
				),
			),
		},
		{
			Name:  "timeout with named parameter",
			Input: "test: @timeout(duration=30s) { echo \"task\" }",
			Expected: Program(
				CmdBlock("test",
					BlockDecorator("timeout", Named("duration", Dur("30s")), Shell(Text("echo "), StrPart("task"))),
				),
			),
		},
		{
			Name:  "parallel with named parameters",
			Input: "test: @parallel(mode=\"fail-fast\") { echo \"task\" }",
			Expected: Program(
				CmdBlock("test",
					BlockDecorator("parallel", Named("mode", Str("fail-fast")), Shell(Text("echo "), StrPart("task"))),
				),
			),
		},
		{
			Name:  "retry with mixed parameters (positional + named)",
			Input: "test: @retry(3, delay=1s) { echo \"task\" }",
			Expected: Program(
				CmdBlock("test",
					BlockDecorator("retry", Num(3), Named("delay", Dur("1s")),
						Shell(Text("echo "), StrPart("task")),
					),
				),
			),
		},
		{
			Name:  "timeout with named parameter (duplicate)",
			Input: "test: @timeout(duration=30s) { echo \"task\" }",
			Expected: Program(
				CmdBlock("test",
					BlockDecorator("timeout", Named("duration", Dur("30s")),
						Shell(Text("echo "), StrPart("task")),
					),
				),
			),
		},
		{
			Name:  "parallel with concurrency and mode",
			Input: "test: @parallel(concurrency=2, mode=\"all\") { echo \"task1\"; echo \"task2\" }",
			Expected: Program(
				CmdBlock("test",
					BlockDecorator("parallel", Named("concurrency", Num(2)), Named("mode", Str("all")),
						Shell(
							Text("echo "), StrPart("task1"),
							Text("; echo "), StrPart("task2"),
						),
					),
				),
			),
		},
		{
			Name:  "timeout with named parameter duplicate",
			Input: "test: @timeout(duration=30s) { echo \"task\" }",
			Expected: Program(
				CmdBlock("test",
					BlockDecorator("timeout", Named("duration", Dur("30s")),
						Shell(Text("echo "), StrPart("task")),
					),
				),
			),
		},
		{
			Name:  "parallel with concurrency and mode parameters",
			Input: "test: @parallel(concurrency=2, mode=\"all\") { echo \"task1\"; echo \"task2\" }",
			Expected: Program(
				CmdBlock("test",
					BlockDecorator("parallel", Named("concurrency", Num(2)), Named("mode", Str("all")),
						Shell(Text("echo "), StrPart("task1"), Text("; echo "), StrPart("task2")),
					),
				),
			),
		},
		{
			Name: "when with string parameter",
			Input: `test: @when("ENV") {
  production: echo "prod"
  default: echo "dev"
}`,
			Expected: Program(
				Cmd("test",
					PatternDecoratorWithBranches("when", Str("ENV"),
						Branch("production", Shell(Text("echo "), StrPart("prod"))),
						Branch("default", Shell(Text("echo "), StrPart("dev"))),
					),
				),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}

func TestShellSubstitution(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "shell command substitution - native shell feature",
			Input: "build: echo \"Current date: $(date)\"",
			Expected: Program(
				Cmd("build", Shell(
					Text("echo "),
					StrPart("Current date: $(date)"),
				)),
			),
		},
		{
			Name:  "shell substitution with @var",
			Input: "deploy: echo \"Building in $(pwd) for @var(ENV)\"",
			Expected: Program(
				Cmd("deploy", Shell(
					Text("echo "),
					StrPart("Building in $(pwd) for "),
					At("var", Id("ENV")),
				)),
			),
		},
		{
			Name:  "complex shell substitution",
			Input: "info: echo \"Files: $(ls | wc -l), User: $(whoami)\"",
			Expected: Program(
				Cmd("info", Shell(
					Text("echo "),
					StrPart("Files: $(ls | wc -l), User: $(whoami)"),
				)),
			),
		},
		{
			Name:  "shell substitution in block",
			Input: "backup: { DATE=$(date +%Y%m%d); echo \"Backup date: $DATE\" }",
			Expected: Program(
				CmdBlock("backup",
					Shell("DATE=$(date +%Y%m%d); echo ", StrPart("Backup date: $DATE")),
				),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}

func TestNestedDecorators(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "block decorator with @var inside",
			Input: "deploy: @timeout(30s) { echo \"Deploying @var(APP)\" }",
			Expected: Program(
				CmdBlock("deploy",
					BlockDecorator("timeout", Dur("30s"),
						Shell(
							Text("echo "),
							StrPart("Deploying "),
							At("var", Id("APP")),
						),
					),
				),
			),
		},
		{
			Name: "EXACT commands.cli reproduction with variables - now works!",
			Input: `var PROJECT = "opal"

setup: {
    @log("🔧 Setting up @var(PROJECT) development environment...")
    @log("📦 Downloading Go dependencies for all modules...")
    @parallel {
        @cmd(core-deps)
        @cmd(codegen-deps)
        @cmd(runtime-deps)
        @cmd(testing-deps)
        @cmd(cli-deps)
    }
    go work sync
    @log("✅ Setup complete! Run 'dev ci' to verify everything works.")
}`,
			// This now parses successfully due to lexer string interpolation fixes!
			// The key issue was @var(PROJECT) in @log string breaking @parallel parsing
			Expected: Program(
				Var("PROJECT", Str("opal")),
				CmdBlock("setup",
					// Now correctly parsing all content due to lexer string interpolation fixes
					Shell(
						At("log", Str("🔧 Setting up @var(PROJECT) development environment...")),
						At("log", Str("📦 Downloading Go dependencies for all modules...")),
					),
					BlockDecorator("parallel",
						Shell(At("cmd", Id("core-deps"))),
						Shell(At("cmd", Id("codegen-deps"))),
						Shell(At("cmd", Id("runtime-deps"))),
						Shell(At("cmd", Id("testing-deps"))),
						Shell(At("cmd", Id("cli-deps"))),
					),
					Shell(Text("go work sync")),
					Shell(At("log", Str("✅ Setup complete! Run 'dev ci' to verify everything works."))),
				),
			),
		},
		{
			Name:  "parallel with mixed content",
			Input: "multi: @parallel { echo start; echo end }",
			Expected: Program(
				CmdBlock("multi",
					BlockDecorator("parallel",
						Shell(Text("echo start; echo end")),
					),
				),
			),
		},
		{
			Name:  "decorator with simple argument",
			Input: "setup: @cwd(\"/usr/bin\") { which tool }",
			Expected: Program(
				CmdBlock("setup",
					BlockDecorator("cwd", Str("/usr/bin"),
						Shell(Text("which tool")),
					),
				),
			),
		},
		{
			Name:  "single timeout decorator",
			Input: "build: @timeout(30s) { npm test }",
			Expected: Program(
				CmdBlock("build",
					BlockDecorator("timeout", Dur("30s"),
						Shell(Text("npm test")),
					),
				),
			),
		},
		{
			Name:  "decorator with variable as argument",
			Input: "build: @cwd(\"BUILD_DIR\") { make clean && make all }",
			Expected: Program(
				CmdBlock("build",
					BlockDecorator("cwd", Str("BUILD_DIR"),
						Chain(
							Shell(Text("make clean")),
							And(Shell(Text(" make all"))),
						),
					),
				),
			),
		},
		{
			Name:  "single timeout decorator with complex command",
			Input: "complex: @timeout(30s) { npm run integration-tests && npm run e2e }",
			Expected: Program(
				CmdBlock("complex",
					BlockDecorator("timeout", Dur("30s"),
						Chain(
							Shell(Text("npm run integration-tests")),
							And(Shell(Text(" npm run e2e"))),
						),
					),
				),
			),
		},
		{
			Name: "multiple commands with decorator - each gets decorated",
			Input: `build: @timeout(2m) {
  echo "Starting build"
  npm run build
  echo "Build complete"
}`,
			Expected: Program(
				CmdBlock("build",
					BlockDecorator("timeout", Dur("2m"),
						Shell(Text("echo "), StrPart("Starting build")),
						Shell(Text("npm run build")),
						Shell(Text("echo "), StrPart("Build complete")),
					),
				),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}

func TestNewDecoratorParameterTypes(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "timeout with positional duration parameter",
			Input: "test: @timeout(30s) { npm test }",
			Expected: Program(
				CmdBlock("test",
					BlockDecorator("timeout", Dur("30s"),
						Shell(Text("npm test")),
					),
				),
			),
		},
		{
			Name:  "retry with positional parameters",
			Input: "test: @retry(3, 1s) { npm test }",
			Expected: Program(
				CmdBlock("test",
					BlockDecorator("retry", Num(3), Dur("1s"),
						Shell(Text("npm test")),
					),
				),
			),
		},
		{
			Name:  "retry with single attempts parameter",
			Input: "test: @retry(5) { npm test }",
			Expected: Program(
				CmdBlock("test",
					BlockDecorator("retry", Num(5),
						Shell(Text("npm test")),
					),
				),
			),
		},
		{
			Name:  "parallel with concurrency parameter",
			Input: "test: @parallel(concurrency=2) { npm test }",
			Expected: Program(
				CmdBlock("test",
					DecoratedShell(Decorator("parallel", Named("concurrency", Num(2))),
						Text("npm test"),
					),
				),
			),
		},
		{
			Name:  "parallel with concurrency and mode",
			Input: "test: @parallel(concurrency=2, mode=\"all\") { npm test }",
			Expected: Program(
				CmdBlock("test",
					DecoratedShell(Decorator("parallel", Named("concurrency", Num(2)), Named("mode", Str("all"))),
						Text("npm test"),
					),
				),
			),
		},
		{
			Name:    "FULL FILE BUG: Parse temp_commands_test.cli - now works!",
			Input:   "",    // Will read from file in test
			WantErr: false, // Fixed by lexer string interpolation improvements
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}

func TestDecoratorVariations(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "decorator with no arguments",
			Input: "sync: @parallel { task1; task2 }",
			Expected: Program(
				CmdBlock("sync",
					DecoratedShell(Decorator("parallel"),
						Text("task1; task2"),
					),
				),
			),
		},
		{
			Name:  "decorator with single string argument",
			Input: "ask: @confirm(\"Are you sure?\") { rm -rf /tmp/* }",
			Expected: Program(
				CmdBlock("ask",
					DecoratedShell(Decorator("confirm", Str("Are you sure?")),
						Text("rm -rf /tmp/*"),
					),
				),
			),
		},
		{
			Name:  "decorator with duration argument",
			Input: "slow: @timeout(5m) { sleep 300 }",
			Expected: Program(
				CmdBlock("slow",
					DecoratedShell(Decorator("timeout", Dur("5m")),
						Text("sleep 300"),
					),
				),
			),
		},
		{
			Name:  "decorator with number argument",
			Input: "retry-task: @retry(3) { flaky-command }",
			Expected: Program(
				CmdBlock("retry-task",
					DecoratedShell(Decorator("retry", Num(3)),
						Text("flaky-command"),
					),
				),
			),
		},
		{
			Name:  "decorator with single argument",
			Input: "watch-files: @debounce(500ms) { npm run build }",
			Expected: Program(
				CmdBlock("watch-files",
					DecoratedShell(Decorator("debounce", Dur("500ms")),
						Text("npm run build"),
					),
				),
			),
		},
		{
			Name:  "decorator with variable argument",
			Input: "deploy: @cwd(\"BUILD_DIR\") { make install }",
			Expected: Program(
				CmdBlock("deploy",
					DecoratedShell(Decorator("cwd", Str("BUILD_DIR")),
						Text("make install"),
					),
				),
			),
		},
		{
			Name:  "decorator with variable pattern argument",
			Input: "advanced: @watch-files(\"PATTERN\") { rebuild }",
			Expected: Program(
				CmdBlock("advanced",
					DecoratedShell(Decorator("watch-files", Str("PATTERN")),
						Text("rebuild"),
					),
				),
			),
		},
		{
			Name:  "decorator with boolean argument",
			Input: "deploy: @confirm(defaultYes=true) { ./deploy.sh }",
			Expected: Program(
				CmdBlock("deploy",
					DecoratedShell(Decorator("confirm", Named("defaultYes", Bool(true))),
						Text("./deploy.sh"),
					),
				),
			),
		},
		{
			Name:  "decorator with negative number",
			Input: "adjust: @offset(-5) { process }",
			Expected: Program(
				CmdBlock("adjust",
					DecoratedShell(Decorator("offset", Num(-5)),
						Text("process"),
					),
				),
			),
		},
		{
			Name:  "decorator with decimal number",
			Input: "scale: @factor(1.5) { scale-service }",
			Expected: Program(
				CmdBlock("scale",
					DecoratedShell(Decorator("factor", Num(1.5)),
						Text("scale-service"),
					),
				),
			),
		},
		{
			Name:  "decorator with no arguments but parentheses",
			Input: "test: @parallel { task1; task2 }",
			Expected: Program(
				CmdBlock("test",
					DecoratedShell(Decorator("parallel"),
						Text("task1; task2"),
					),
				),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}

func TestNestedPatternDecorators(t *testing.T) {
	testCases := []TestCase{
		{
			Name: "pattern decorator inside block decorator",
			Input: `test: @retry(attempts=2) {
				@when("ENV") {
					production: echo "prod task"
					default: echo "default task"
				}
			}`,
			// This should parse correctly since lexer handles it properly
			Expected: Program(
				CmdBlock("test",
					BlockDecorator("retry", Named("attempts", Num(2)),
						PatternDecoratorWithBranches("when", Str("ENV"),
							Branch("production", Shell(Text("echo "), StrPart("prod task"))),
							Branch("default", Shell(Text("echo "), StrPart("default task"))),
						),
					),
				),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}

func TestVarVsEnvDecorators(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "@var decorator should parse as FunctionDecorator",
			Input: `test: echo "@var(PORT)"`,
			Expected: Program(
				Cmd("test", Simple(
					Text("echo "),
					At("var", Id("PORT")),
				)),
			),
		},
		{
			Name:  "@env decorator should parse as FunctionDecorator (using quoted string)",
			Input: `test: echo "@env(HOME)"`,
			Expected: Program(
				Cmd("test", Simple(
					Text("echo "),
					At("env", Id("HOME")),
				)),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}

func TestParallelCommandsCliIssue(t *testing.T) {
	testCases := []TestCase{
		{
			Name: "parallel with cmd decorators - reproducing commands.cli issue",
			Input: `setup: @parallel {
				@cmd(core-deps)
				@cmd(runtime-deps)
			}`,
			Expected: Program(
				CmdBlock("setup",
					BlockDecorator("parallel",
						Shell(At("cmd", Id("core-deps"))),
						Shell(At("cmd", Id("runtime-deps"))),
					),
				),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}
