package parser

import (
	"testing"
)

func TestCompleteParserIntegration(t *testing.T) {
	testCases := []TestCase{
		{
			Name: "complete opal file",
			Input: `# Complete opal example
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
					BlockDecorator("timeout", Id("TIMEOUT"),
						Chain(
							Shell(Text("cd "), At("var", Id("SRC"))),
							And(Shell(Text(" go run main.go --port="), At("var", Id("PORT")))),
						),
					),
				),
				WatchBlock("dev",
					BlockDecorator("parallel",
						Chain(
							Shell(Text("cd "), At("var", Id("SRC"))),
							And(Shell(Text(" NODE_ENV="), At("env", Str("NODE_ENV")), Text(" go run main.go"))),
						),
						Chain(
							Shell(Text("cd frontend")),
							And(Shell(Text(" npm start"))),
						),
					),
				),
				StopBlock("dev",
					Shell(Text("pkill -f "), StrPart("go run")),
					Shell(Text("pkill -f "), StrPart("npm start")),
				),
				CmdBlock("deploy",
					BlockDecorator("confirm", Str("Deploy to production?"),
						Chain(
							Shell(Text("cd "), At("var", Id("SRC"))),
							And(Shell(Text(" go build -o "), At("var", Id("DIST")), Text("/app"))),
							And(Shell(Text(" rsync -av "), At("var", Id("DIST")), Text("/ server:/opt/app/"))),
						),
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
					BlockDecorator("timeout", Dur("2m"),
						Chain(
							Shell(Text("echo "), StrPart("Building project...")),
							And(Shell(Text("npm run build"))),
							And(Shell(Text(" echo "), StrPart("Build complete"))),
						),
					),
				),
				CmdBlock("dev",
					BlockDecorator("parallel",
						Chain(
							Shell(Text("cd "), At("var", Id("SRC"))),
							And(Shell(Text(" npm run dev"))),
						),
						Shell(Text("echo "), StrPart("Server starting on port "), At("var", Id("PORT"))),
					),
				),
				CmdBlock("test",
					BlockDecorator("retry", Num(3),
						Chain(
							Shell(Text("npm run test")),
							And(Shell(Text(" npm run lint"))),
						),
					),
				),
				WatchBlock("test",
					BlockDecorator("debounce", Dur("500ms"),
						Shell(Text("npm run test:watch")),
					),
				),
				CmdBlock("deploy",
					BlockDecorator("timeout", Dur("5m"),
						Chain(
							Shell(Text("npm run build")),
							And(Shell(Text(" rsync -av "), At("var", Id("DIST")), Text("/ server:/var/www/app/"))),
							And(Shell(Text(" echo "), StrPart("Deployment complete"))),
						),
					),
				),
				Stop("test", Shell(Text("pkill -f "), StrPart("npm.*test"))),
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
					Chain(
						Shell(Text("echo "), StrPart("Testing API at "), At("var", Id("API_URL"))),
						And(Shell(Text(" curl -H "), StrPart("Authorization: Bearer "), At("var", Id("TOKEN")), Text(" "), At("var", Id("API_URL")), Text("/health"))),
						And(Shell(Text(" echo "), StrPart("API test complete"))),
					),
				),
				CmdBlock("backup",
					BlockDecorator("confirm", Str("Create backup?"),
						Chain(
							Shell(Text("echo "), StrPart("Starting backup...")),
							And(Shell(Text(" DATE=$(date +%Y%m%d); echo "), StrPart("Backup date: $DATE"))),
							And(Shell(Text(" rsync -av /data/ backup@server.com:/backups/"), At("var", Id("PROJECT")), Text("/"))),
							And(Shell(Text(" echo "), StrPart("Backup complete"))),
						),
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
					Text("echo "),
					StrPart("Server running at "),
					At("var", Id("HOST")),
					StrPart(":"),
					At("var", Id("PORT")),
				)),
				Cmd("info", Simple(
					Text("echo "),
					StrPart("Host: "),
					At("var", Id("HOST")),
					StrPart(", Port: "),
					At("var", Id("PORT")),
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
					BlockDecorator("timeout", Id("TIMEOUT"),
						Chain(
							Shell(Text("echo "), StrPart("Attempting operation...")),
							And(Shell(Text(" for i in $(seq 1 "), At("var", Id("RETRIES")), Text("); do ./run-operation.sh"))),
							And(Shell(Text(" break; done"))),
						),
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
				Cmd("empty", Shell(Text("echo "), StrPart("empty command"))),
				CmdBlock("block",
					Shell(Text("echo "), StrPart("block command")),
				),
				CmdBlock("decorated",
					BlockDecorator("parallel",
						Shell(Text("echo "), StrPart("decorated command")),
					),
				),
			),
		},
		{
			Name: "environment variables with function decorators",
			Input: `var KUBE_CONTEXT = "production"

deploy: kubectl config use-context @env('KUBE_CONTEXT') && kubectl apply -f k8s/
status: echo "Current context: @env('KUBE_CONTEXT'), Project: @var(KUBE_CONTEXT)"`,
			Expected: Program(
				Var("KUBE_CONTEXT", Str("production")),
				Cmd("deploy", Chain(
					Shell(Text("kubectl config use-context "), At("env", Str("KUBE_CONTEXT"))),
					And(Shell(Text(" kubectl apply -f k8s/"))),
				)),
				Cmd("status", Simple(
					Text("echo "),
					StrPart("Current context: "),
					At("env", Str("KUBE_CONTEXT")),
					StrPart(", Project: "),
					At("var", Id("KUBE_CONTEXT")),
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
						Shell(Text("echo "), StrPart("Starting server on "), At("var", Id("HOST")), StrPart(":"), At("var", Id("PORT")), StrPart(" with debug="), At("var", Id("DEBUG"))),
						Shell(Text("go run main.go --host="), At("var", Id("HOST")), Text(" --port="), At("var", Id("PORT")), Text(" --debug="), At("var", Id("DEBUG"))),
					),
				),
				CmdBlock("test",
					BlockDecorator("retry", Id("RETRIES"),
						Shell(Text("npm test")),
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
					Text("echo "),
					StrPart("Setting up..."),
				)),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}
