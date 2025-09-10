package parser

import (
	"testing"
)

func TestVariableDefinitions(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "simple variable",
			Input: `var SRC = "./src"`,
			Expected: Program(
				Var("SRC", Str("./src")),
			),
		},
		{
			Name:  "variable with complex value",
			Input: `var CMD = "go test -v ./..."`,
			Expected: Program(
				Var("CMD", Str("go test -v ./...")),
			),
		},
		{
			Name: "multiple variables",
			Input: `var SRC = "./src"
var BIN = "./bin"`,
			Expected: Program(
				Var("SRC", Str("./src")),
				Var("BIN", Str("./bin")),
			),
		},
		{
			Name:  "grouped variables",
			Input: "var (\n  SRC = \"./src\"\n  BIN = \"./bin\"\n)",
			Expected: Program(
				Var("SRC", Str("./src")),
				Var("BIN", Str("./bin")),
			),
		},
		{
			Name:  "variable with number value",
			Input: "var PORT = 8080",
			Expected: Program(
				Var("PORT", Num(8080)),
			),
		},
		{
			Name:  "variable with duration value",
			Input: "var TIMEOUT = 30s",
			Expected: Program(
				Var("TIMEOUT", Dur("30s")),
			),
		},
		{
			Name:  "variable with quoted string",
			Input: `var MESSAGE = "Hello, World!"`,
			Expected: Program(
				Var("MESSAGE", Str("Hello, World!")),
			),
		},
		{
			Name:  "variable with special characters",
			Input: `var API_URL = "https://api.example.com/v1"`,
			Expected: Program(
				Var("API_URL", Str("https://api.example.com/v1")),
			),
		},
		{
			Name:  "mixed variable types in group",
			Input: "var (\n  SRC = \"./src\"\n  PORT = 3000\n  TIMEOUT = 5m\n  DEBUG = true\n)",
			Expected: Program(
				Var("SRC", Str("./src")),
				Var("PORT", Num(3000)),
				Var("TIMEOUT", Dur("5m")),
				Var("DEBUG", Bool(true)),
			),
		},
		{
			Name:  "variable with environment-style name",
			Input: `var NODE_ENV = "production"`,
			Expected: Program(
				Var("NODE_ENV", Str("production")),
			),
		},
		{
			Name:  "variable with URL containing query params",
			Input: `var API_URL = "https://api.example.com/v1?key=abc123"`,
			Expected: Program(
				Var("API_URL", Str("https://api.example.com/v1?key=abc123")),
			),
		},
		{
			Name:  "variable with boolean value",
			Input: "var DEBUG = true",
			Expected: Program(
				Var("DEBUG", Bool(true)),
			),
		},
		{
			Name:  "variable with false boolean value",
			Input: "var PRODUCTION = false",
			Expected: Program(
				Var("PRODUCTION", Bool(false)),
			),
		},
		{
			Name:  "variable with path containing spaces",
			Input: `var PROJECT_PATH = "/path/with spaces/project"`,
			Expected: Program(
				Var("PROJECT_PATH", Str("/path/with spaces/project")),
			),
		},
		{
			Name:  "variable with empty string value",
			Input: `var EMPTY = ""`,
			Expected: Program(
				Var("EMPTY", Str("")),
			),
		},
		{
			Name:  "variable with numeric string",
			Input: `var VERSION = "1.2.3"`,
			Expected: Program(
				Var("VERSION", Str("1.2.3")),
			),
		},
		{
			Name:  "variable with complex file path",
			Input: `var CONFIG_FILE = "/etc/myapp/config.json"`,
			Expected: Program(
				Var("CONFIG_FILE", Str("/etc/myapp/config.json")),
			),
		},
		{
			Name:  "variable with URL containing port",
			Input: `var DATABASE_URL = "postgresql://user:pass@localhost:5432/dbname"`,
			Expected: Program(
				Var("DATABASE_URL", Str("postgresql://user:pass@localhost:5432/dbname")),
			),
		},
		{
			Name:  "variable with floating point duration",
			Input: "var TIMEOUT = 2.5s",
			Expected: Program(
				Var("TIMEOUT", Dur("2.5s")),
			),
		},
		{
			Name: "multiple variables with mixed types",
			Input: `var PORT = 3000
var HOST = "localhost"
var TIMEOUT = 30s
var DEBUG = true`,
			Expected: Program(
				Var("PORT", Num(3000)),
				Var("HOST", Str("localhost")),
				Var("TIMEOUT", Dur("30s")),
				Var("DEBUG", Bool(true)),
			),
		},
		{
			Name:  "variable with quoted identifier value",
			Input: `var MODE = "production"`,
			Expected: Program(
				Var("MODE", Str("production")),
			),
		},
		{
			Name:  "variable with underscores and URL",
			Input: `var API_BASE_URL = "https://api.example.com"`,
			Expected: Program(
				Var("API_BASE_URL", Str("https://api.example.com")),
			),
		},
		{
			Name:  "negative number variable",
			Input: "var OFFSET = -100",
			Expected: Program(
				Var("OFFSET", Num(-100)),
			),
		},
		{
			Name:  "floating point number variable",
			Input: "var RATIO = 3.14",
			Expected: Program(
				Var("RATIO", Num(3.14)),
			),
		},
		{
			Name:  "milliseconds duration",
			Input: "var DELAY = 500ms",
			Expected: Program(
				Var("DELAY", Dur("500ms")),
			),
		},
		{
			Name:  "hours duration",
			Input: "var CACHE_TTL = 24h",
			Expected: Program(
				Var("CACHE_TTL", Dur("24h")),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}

func TestVariableDecoratorArgumentRestrictions(t *testing.T) {
	// Test that @var() is not allowed in decorator arguments
	testCases := []TestCase{
		{
			Name: "reject @var() in decorator arguments",
			Input: `var TIMEOUT = 30s
test: @timeout(@var(TIMEOUT)) { npm test }`,
			WantErr:     true,
			ErrorSubstr: "parameter 'duration' expects duration, got AT",
		},
		{
			Name:        "reject @env() in decorator arguments",
			Input:       `test: @timeout(@env(DURATION)) { npm test }`,
			WantErr:     true,
			ErrorSubstr: "parameter 'duration' expects duration, got AT",
		},
		{
			Name: "allow direct variable references in decorator arguments",
			Input: `var TIMEOUT = 30s
test: @timeout(TIMEOUT) { npm test }`,
			Expected: Program(
				Var("TIMEOUT", Dur("30s")),
				CmdBlock("test",
					DecoratedShell(Decorator("timeout", Id("TIMEOUT")),
						Text("npm test"),
					),
				),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}

func TestVariableUsageInCommands(t *testing.T) {
	testCases := []TestCase{
		{
			Name: "variables with command usage - requires explicit block",
			Input: `var SRC = "./src"
var DEST = "./dist"
build: { cp -r @var(SRC)/* @var(DEST)/ }`,
			Expected: Program(
				Var("SRC", Str("./src")),
				Var("DEST", Str("./dist")),
				CmdBlock("build",
					Shell(Text("cp -r "), At("var", Id("SRC")), Text("/* "), At("var", Id("DEST")), Text("/")),
				),
			),
		},
		{
			Name: "grouped variables with usage - requires explicit block",
			Input: `var (
  PORT = 8080
  HOST = "localhost"
)
serve: { go run main.go --port=@var(PORT) --host=@var(HOST) }`,
			Expected: Program(
				Var("PORT", Num(8080)),
				Var("HOST", Str("localhost")),
				CmdBlock("serve",
					Shell(Text("go run main.go --port="), At("var", Id("PORT")), Text(" --host="), At("var", Id("HOST"))),
				),
			),
		},
		{
			Name: "variables in block commands",
			Input: `var SRC = "./src"
deploy: { cd @var(SRC); make clean; make install }`,
			Expected: Program(
				Var("SRC", Str("./src")),
				CmdBlock("deploy",
					Shell(Text("cd "), At("var", Id("SRC")), Text("; make clean; make install")),
				),
			),
		},
		{
			Name: "variables in decorator arguments",
			Input: `var TIMEOUT = 30s
test: @timeout(TIMEOUT) { npm test }`,
			Expected: Program(
				Var("TIMEOUT", Dur("30s")),
				CmdBlock("test",
					DecoratedShell(Decorator("timeout", Id("TIMEOUT")),
						Text("npm test"),
					),
				),
			),
		},
		{
			Name: "environment variable substitution with @env - simple command",
			Input: `var TIME = 5m
deploy: NODE_ENV=@env("NODE_ENV") npm run deploy`,
			Expected: Program(
				Var("TIME", Dur("5m")),
				Cmd("deploy", Simple(
					Text("NODE_ENV="),
					At("env", Str("NODE_ENV")),
					Text(" npm run deploy"),
				)),
			),
		},
		{
			Name: "variables in watch commands",
			Input: `var SRC = "./src"
watch build: @debounce(500ms) { echo "Building @var(SRC)" }`,
			Expected: Program(
				Var("SRC", Str("./src")),
				WatchBlock("build",
					BlockDecorator("debounce", Dur("500ms"),
						Shell(Text("echo "), StrPart("Building "), At("var", Id("SRC"))),
					),
				),
			),
		},
		{
			Name: "variables in stop commands - explicit block required",
			Input: `var PROCESS = "myapp"
stop server: { pkill -f @var(PROCESS) }`,
			Expected: Program(
				Var("PROCESS", Str("myapp")),
				StopBlock("server",
					Shell(Text("pkill -f "), At("var", Id("PROCESS"))),
				),
			),
		},
		{
			Name: "variables with file counting command - requires explicit block",
			Input: `var SRC = "./src"
build: { echo "Files: $(ls @var(SRC) | wc -l)" }`,
			Expected: Program(
				Var("SRC", Str("./src")),
				CmdBlock("build",
					Shell(Text("echo "), StrPart("Files: $(ls "), At("var", Id("SRC")), StrPart(" | wc -l)")),
				),
			),
		},
		{
			Name: "variables with nested shell content - requires explicit block",
			Input: `var HOST = "server.com"
var PORT = 22
connect: { ssh -p @var(PORT) user@@@var(HOST) }`,
			Expected: Program(
				Var("HOST", Str("server.com")),
				Var("PORT", Num(22)),
				CmdBlock("connect",
					Shell(Text("ssh -p "), At("var", Id("PORT")), Text(" user@@"), At("var", Id("HOST"))),
				),
			),
		},
		{
			Name: "variables in complex command chains - requires explicit block",
			Input: `var SRC = "./src"
var DEST = "./dist"
var ENV = "prod"
build: { cd @var(SRC) && npm run build:@var(ENV) && cp -r dist/* @var(DEST)/ }`,
			Expected: Program(
				Var("SRC", Str("./src")),
				Var("DEST", Str("./dist")),
				Var("ENV", Str("prod")),
				CmdBlock("build",
					Chain(
						Shell(Text("cd "), At("var", Id("SRC"))),
						And(Shell(Text(" npm run build:"), At("var", Id("ENV")))),
						And(Shell(Text(" cp -r dist/* "), At("var", Id("DEST")), Text("/"))),
					),
				),
			),
		},
		{
			Name: "variables in conditional expressions - requires explicit block",
			Input: `var ENV = "production"
check: { test "@var(ENV)" = "production" && echo "prod mode" || echo "dev mode" }`,
			Expected: Program(
				Var("ENV", Str("production")),
				CmdBlock("check",
					Chain(
						Shell(Text("test "), At("var", Id("ENV")), Text(" = "), StrPart("production")),
						And(Shell(Text(" echo "), StrPart("prod mode"))),
						Or(Shell(Text(" echo "), StrPart("dev mode"))),
					),
				),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}

func TestVariableEdgeCases(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "variable with special characters in name",
			Input: `var API_BASE_URL_V2 = "https://api.example.com/v2"`,
			Expected: Program(
				Var("API_BASE_URL_V2", Str("https://api.example.com/v2")),
			),
		},
		{
			Name:  "variable with mixed case",
			Input: `var NodeEnv = "development"`,
			Expected: Program(
				Var("NodeEnv", Str("development")),
			),
		},
		{
			Name:  "variable with numbers in name",
			Input: `var API_V2_URL = "https://api.example.com/v2"`,
			Expected: Program(
				Var("API_V2_URL", Str("https://api.example.com/v2")),
			),
		},
		{
			Name:  "variable with very long value",
			Input: `var LONG_VALUE = "this-is-a-very-long-value-that-spans-multiple-words-and-contains-hyphens"`,
			Expected: Program(
				Var("LONG_VALUE", Str("this-is-a-very-long-value-that-spans-multiple-words-and-contains-hyphens")),
			),
		},
		{
			Name:  "variable with value containing equals (quoted)",
			Input: `var QUERY = "name=value&other=data"`,
			Expected: Program(
				Var("QUERY", Str("name=value&other=data")),
			),
		},
		{
			Name:  `variable with quoted value containing spaces`,
			Input: `var MESSAGE = "Hello World from Devcmd"`,
			Expected: Program(
				Var("MESSAGE", Str("Hello World from Devcmd")),
			),
		},
		{
			Name: "variables with similar names",
			Input: `var API_URL = "https://api.com"
var API_URL_V2 = "https://api.com/v2"`,
			Expected: Program(
				Var("API_URL", Str("https://api.com")),
				Var("API_URL_V2", Str("https://api.com/v2")),
			),
		},
		{
			Name: "variable usage in quoted strings - requires explicit block",
			Input: `var NAME = "World"
greet: { echo "Hello @var(NAME)!" }`,
			Expected: Program(
				Var("NAME", Str("World")),
				CmdBlock("greet",
					Shell(Text("echo "), StrPart("Hello "), At("var", Id("NAME")), StrPart("!")),
				),
			),
		},
		{
			Name: "variable usage with shell operators - requires explicit block",
			Input: `var FILE = "data.txt"
process: { cat @var(FILE) | grep pattern | sort }`,
			Expected: Program(
				Var("FILE", Str("data.txt")),
				CmdBlock("process",
					Chain(
						Shell(Text("cat "), At("var", Id("FILE"))),
						Pipe(Shell(Text("grep pattern"))),
						Pipe(Shell(Text(" sort"))),
					),
				),
			),
		},
		{
			Name: "variable usage in file paths - requires explicit block",
			Input: `var HOME = "/home/user"
backup: { cp important.txt @var(HOME)/backup/ }`,
			Expected: Program(
				Var("HOME", Str("/home/user")),
				CmdBlock("backup",
					Shell(Text("cp important.txt "), At("var", Id("HOME")), Text("/backup/")),
				),
			),
		},
		{
			Name:  "zero value number",
			Input: "var COUNT = 0",
			Expected: Program(
				Var("COUNT", Num(0)),
			),
		},
		{
			Name:  "negative floating point",
			Input: "var OFFSET = -2.5",
			Expected: Program(
				Var("OFFSET", Num(-2.5)),
			),
		},
		{
			Name:  "boolean false in group",
			Input: "var (\n  ENABLED = true\n  DISABLED = false\n)",
			Expected: Program(
				Var("ENABLED", Bool(true)),
				Var("DISABLED", Bool(false)),
			),
		},
		{
			Name: "mixed duration units",
			Input: `var (
  SHORT = 100ms
  MEDIUM = 30s
  LONG = 5m
  VERY_LONG = 2h
)`,
			Expected: Program(
				Var("SHORT", Dur("100ms")),
				Var("MEDIUM", Dur("30s")),
				Var("LONG", Dur("5m")),
				Var("VERY_LONG", Dur("2h")),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}

func TestVariableTypeValidation(t *testing.T) {
	// Test that only the 4 supported types are allowed
	testCases := []TestCase{
		{
			Name:        "reject unquoted identifier",
			Input:       "var PATH = ./src",
			WantErr:     true,
			ErrorSubstr: "variable value must be a quoted string, number, duration, or boolean literal",
		},
		{
			Name:        "reject array syntax",
			Input:       "var ITEMS = [1, 2, 3]",
			WantErr:     true,
			ErrorSubstr: "variable value must be a quoted string, number, duration, or boolean literal",
		},
		{
			Name:        "reject object syntax",
			Input:       "var CONFIG = { key: value }",
			WantErr:     true,
			ErrorSubstr: "variable value must be a quoted string, number, duration, or boolean literal",
		},
		{
			Name:  "accept string literal",
			Input: `var PATH = "./src"`,
			Expected: Program(
				Var("PATH", Str("./src")),
			),
		},
		{
			Name:  "accept number literal",
			Input: "var COUNT = 42",
			Expected: Program(
				Var("COUNT", Num(42)),
			),
		},
		{
			Name:  "accept duration literal",
			Input: "var TIMEOUT = 30s",
			Expected: Program(
				Var("TIMEOUT", Dur("30s")),
			),
		},
		{
			Name:  "accept boolean literal",
			Input: "var ENABLED = true",
			Expected: Program(
				Var("ENABLED", Bool(true)),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}
