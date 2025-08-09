package parser

import (
	"testing"
)

func TestCompleteParserIntegration(t *testing.T) {
	testCases := []TestCase{
		{
			Name: "complete devcmd file",
			Input: `# Complete devcmd example
var SRC = "./src"
var DIST = "./dist"
var PORT = 8080

var (
  NODE_ENV = "development"
  TIMEOUT = 30s
  DEBUG = true
)

build: go build -o @var(DIST)/app @var(SRC)/main.go

serve: @timeout(TIMEOUT) {
  cd @var(SRC) && go run main.go --port=@var(PORT)
}

watch dev: @parallel {
  cd @var(SRC) && NODE_ENV=@env("NODE_ENV") go run main.go
  cd frontend && npm start
}

stop dev: {
  pkill -f "go run"
  pkill -f "npm start"
}

deploy: @confirm("Deploy to production?") {
  cd @var(SRC) && go build -o @var(DIST)/app && rsync -av @var(DIST)/ server:/opt/app/
}`,
			Expected: Program(
				Var("SRC", Str("./src")),
				Var("DIST", Str("./dist")),
				Var("PORT", Num(8080)),
				// Grouped variables
				Var("NODE_ENV", Str("development")),
				Var("TIMEOUT", Dur("30s")),
				Var("DEBUG", Bool(true)),
				Cmd("build", Simple(
					Text("go build -o "),
					At("var", Id("DIST")),
					Text("/app "),
					At("var", Id("SRC")),
					Text("/main.go"),
				)),
				CmdBlock("serve",
					DecoratedShell(Decorator("timeout", Id("TIMEOUT")),
						Text("cd "),
						At("var", Id("SRC")),
						Text(" && go run main.go --port="),
						At("var", Id("PORT")),
					),
				),
				WatchBlock("dev",
					BlockDecorator("parallel",
						Shell(Text("cd "), At("var", Id("SRC")), Text(" && NODE_ENV="), At("env", Str("NODE_ENV")), Text(" go run main.go")),
						Shell(Text("cd frontend && npm start")),
					),
				),
				StopBlock("dev",
					Shell(`pkill -f "go run"`),
					Shell(`pkill -f "npm start"`),
				),
				CmdBlock("deploy",
					DecoratedShell(Decorator("confirm", Str("Deploy to production?")),
						Text("cd "),
						At("var", Id("SRC")),
						Text(" && go build -o "),
						At("var", Id("DIST")),
						Text("/app && rsync -av "),
						At("var", Id("DIST")),
						Text("/ server:/opt/app/"),
					),
				),
			),
		},
		{
			Name: "realistic development workflow",
			Input: `var (
  SRC = "./src"
  DIST = "./dist"
  PORT = 3000
  ENV = "development"
)

install: npm install

clean: rm -rf @var(DIST) node_modules

build: @timeout(2m) {
  echo "Building project..." && npm run build && echo "Build complete"
}

dev: @parallel {
  cd @var(SRC) && npm run dev
  echo "Server starting on port @var(PORT)"
}

test: @retry(3) {
  npm run test && npm run lint
}

watch test: @debounce(500ms) {
  npm run test:watch
}

deploy: @timeout(5m) {
  npm run build && rsync -av @var(DIST)/ server:/var/www/app/ && echo "Deployment complete"
}

stop test: pkill -f "npm.*test"`,
			Expected: Program(
				Var("SRC", Str("./src")),
				Var("DIST", Str("./dist")),
				Var("PORT", Num(3000)),
				Var("ENV", Str("development")),
				Cmd("install", "npm install"),
				Cmd("clean", Simple(
					Text("rm -rf "),
					At("var", Id("DIST")),
					Text(" node_modules"),
				)),
				CmdBlock("build",
					DecoratedShell(Decorator("timeout", Dur("2m")),
						Text("echo \"Building project...\" && npm run build && echo \"Build complete\""),
					),
				),
				CmdBlock("dev",
					BlockDecorator("parallel",
						Shell(Text("cd "), At("var", Id("SRC")), Text(" && npm run dev")),
						Shell(Text("echo \"Server starting on port "), At("var", Id("PORT")), Text("\"")),
					),
				),
				CmdBlock("test",
					DecoratedShell(Decorator("retry", Num(3)),
						Text("npm run test && npm run lint"),
					),
				),
				WatchBlock("test",
					DecoratedShell(Decorator("debounce", Dur("500ms")),
						Text("npm run test:watch"),
					),
				),
				CmdBlock("deploy",
					DecoratedShell(Decorator("timeout", Dur("5m")),
						Text("npm run build && rsync -av "),
						At("var", Id("DIST")),
						Text("/ server:/var/www/app/ && echo \"Deployment complete\""),
					),
				),
				Stop("test", "pkill -f \"npm.*test\""),
			),
		},
		{
			Name: "complex mixed content example",
			Input: `var API_URL = "https://api.example.com"
var TOKEN = "abc123"
var PROJECT = "myproject"

api-test: {
  echo "Testing API at @var(API_URL)" && curl -H "Authorization: Bearer @var(TOKEN)" @var(API_URL)/health && echo "API test complete"
}

backup: @confirm("Create backup?") {
  echo "Starting backup..." && DATE=$(date +%Y%m%d); echo "Backup date: $DATE" && rsync -av /data/ backup@server.com:/backups/@var(PROJECT)/ && echo "Backup complete"
}`,
			Expected: Program(
				Var("API_URL", Str("https://api.example.com")),
				Var("TOKEN", Str("abc123")),
				Var("PROJECT", Str("myproject")),
				CmdBlock("api-test",
					Shell("echo \"Testing API at ", At("var", Id("API_URL")), "\" && curl -H \"Authorization: Bearer ", At("var", Id("TOKEN")), "\" ", At("var", Id("API_URL")), "/health && echo \"API test complete\""),
				),
				CmdBlock("backup",
					DecoratedShell(Decorator("confirm", Str("Create backup?")),
						Text("echo \"Starting backup...\" && DATE=$(date +%Y%m%d); echo \"Backup date: $DATE\" && rsync -av /data/ backup@server.com:/backups/"),
						At("var", Id("PROJECT")),
						Text("/ && echo \"Backup complete\""),
					),
				),
			),
		},
		{
			Name: "simple commands with function decorators",
			Input: `var HOST = "localhost"
var PORT = 8080

ping: curl http://@var(HOST):@var(PORT)/health
status: echo "Server running at @var(HOST):@var(PORT)"
info: echo "Host: @var(HOST), Port: @var(PORT)"`,
			Expected: Program(
				Var("HOST", Str("localhost")),
				Var("PORT", Num(8080)),
				Cmd("ping", Simple(
					Text("curl http://"),
					At("var", Id("HOST")),
					Text(":"),
					At("var", Id("PORT")),
					Text("/health"),
				)),
				Cmd("status", Simple(
					Text("echo \"Server running at "),
					At("var", Id("HOST")),
					Text(":"),
					At("var", Id("PORT")),
					Text("\""),
				)),
				Cmd("info", Simple(
					Text("echo \"Host: "),
					At("var", Id("HOST")),
					Text(", Port: "),
					At("var", Id("PORT")),
					Text("\""),
				)),
			),
		},
		{
			Name: "single timeout decorator with retry logic",
			Input: `var RETRIES = 3
var TIMEOUT = 30s

complex: @timeout(TIMEOUT) {
  echo "Attempting operation..." && for i in $(seq 1 @var(RETRIES)); do ./run-operation.sh && break; done
}`,
			Expected: Program(
				Var("RETRIES", Num(3)),
				Var("TIMEOUT", Dur("30s")),
				CmdBlock("complex",
					DecoratedShell(Decorator("timeout", Id("TIMEOUT")),
						Text("echo \"Attempting operation...\" && for i in $(seq 1 "),
						At("var", Id("RETRIES")),
						Text("); do ./run-operation.sh && break; done"),
					),
				),
			),
		},
		{
			Name: "commands with valid content",
			Input: `var EMPTY = ""

empty: echo "empty command"
block: { echo "block command" }
decorated: @parallel { echo "decorated command" }`,
			Expected: Program(
				Var("EMPTY", Str("")),
				Cmd("empty", "echo \"empty command\""),
				CmdBlock("block",
					Shell("echo \"block command\""),
				),
				CmdBlock("decorated",
					DecoratedShell(Decorator("parallel"),
						Text("echo \"decorated command\""),
					),
				),
			),
		},
		{
			Name: "environment variables with function decorators",
			Input: `var KUBE_CONTEXT = "production"

deploy: kubectl config use-context @env("KUBE_CONTEXT") && kubectl apply -f k8s/
status: echo "Current context: @env("KUBE_CONTEXT"), Project: @var(KUBE_CONTEXT)"`,
			Expected: Program(
				Var("KUBE_CONTEXT", Str("production")),
				Cmd("deploy", Simple(
					Text("kubectl config use-context "),
					At("env", Str("KUBE_CONTEXT")),
					Text(" && kubectl apply -f k8s/"),
				)),
				Cmd("status", Simple(
					Text("echo \"Current context: "),
					At("env", Str("KUBE_CONTEXT")),
					Text(", Project: "),
					At("var", Id("KUBE_CONTEXT")),
					Text("\""),
				)),
			),
		},
		{
			Name: "mixed variable types",
			Input: `var (
  HOST = "localhost"
  PORT = 8080
  TIMEOUT = 30s
  DEBUG = true
  RETRIES = 3
)

serve: @timeout(TIMEOUT) {
  echo "Starting server on @var(HOST):@var(PORT) with debug=@var(DEBUG)"
  go run main.go --host=@var(HOST) --port=@var(PORT) --debug=@var(DEBUG)
}

test: @retry(RETRIES) {
  npm test
}`,
			Expected: Program(
				Var("HOST", Str("localhost")),
				Var("PORT", Num(8080)),
				Var("TIMEOUT", Dur("30s")),
				Var("DEBUG", Bool(true)),
				Var("RETRIES", Num(3)),
				CmdBlock("serve",
					BlockDecorator("timeout", Id("TIMEOUT"),
						Shell(Text("echo \"Starting server on "), At("var", Id("HOST")), Text(":"), At("var", Id("PORT")), Text(" with debug="), At("var", Id("DEBUG")), Text("\"")),
						Shell(Text("go run main.go --host="), At("var", Id("HOST")), Text(" --port="), At("var", Id("PORT")), Text(" --debug="), At("var", Id("DEBUG"))),
					),
				),
				CmdBlock("test",
					DecoratedShell(Decorator("retry", Id("RETRIES")),
						Text("npm test"),
					),
				),
			),
		},
		{
			Name: "comment between variable and command",
			Input: `var PYTHON = "python3"
# Data science project development
setup: echo "Setting up..."`,
			Expected: Program(
				Var("PYTHON", Str("python3")),
				Cmd("setup", Simple(
					Text("echo \"Setting up...\""),
				)),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}
