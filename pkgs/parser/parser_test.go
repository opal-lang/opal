package parser

import (
	"strings"
	"testing"
)

func TestBasicParsing(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantCommand string
		wantName    string
		wantErr     bool
	}{
		{
			name:        "simple command",
			input:       "build: echo hello;",
			wantCommand: "echo hello",
			wantName:    "build",
			wantErr:     false,
		},
		{
			name:        "command with arguments",
			input:       "test: go test -v ./...;",
			wantCommand: "go test -v ./...",
			wantName:    "test",
			wantErr:     false,
		},
		{
			name:        "command with special characters",
			input:       "run: echo 'Hello, World!';",
			wantCommand: "echo 'Hello, World!'",
			wantName:    "run",
			wantErr:     false,
		},
		{
			name:        "command with empty content",
			input:       "noop: ;",
			wantCommand: "",
			wantName:    "noop",
			wantErr:     false,
		},
		{
			name:        "command with trailing space",
			input:       "build: make all   ;",
			wantCommand: "make all",
			wantName:    "build",
			wantErr:     false,
		},
		// New edge cases for parentheses and POSIX syntax
		{
			name:        "command with parentheses - simple subshell",
			input:       "check: (echo test);",
			wantCommand: "(echo test)",
			wantName:    "check",
			wantErr:     false,
		},
		{
			name:        "command with parentheses - complex POSIX",
			input:       "validate: (echo \"Go not installed\" && exit 1);",
			wantCommand: "(echo \"Go not installed\" && exit 1)",
			wantName:    "validate",
			wantErr:     false,
		},
		{
			name:        "command with conditional and parentheses",
			input:       "setup: which go || (echo \"Go not installed\" && exit 1);",
			wantCommand: "which go || (echo \"Go not installed\" && exit 1)",
			wantName:    "setup",
			wantErr:     false,
		},
		{
			name:        "command with nested parentheses",
			input:       "complex: (cd src && (make clean || echo \"already clean\"));",
			wantCommand: "(cd src && (make clean || echo \"already clean\"))",
			wantName:    "complex",
			wantErr:     false,
		},
		// Test that 'watch' and 'stop' can appear in command text
		{
			name:        "command containing watch keyword",
			input:       "monitor: echo \"watching files\" && watch -n 1 ls;",
			wantCommand: "echo \"watching files\" && watch -n 1 ls",
			wantName:    "monitor",
			wantErr:     false,
		},
		{
			name:        "command containing stop keyword",
			input:       "halt: echo \"stopping service\" && systemctl stop nginx;",
			wantCommand: "echo \"stopping service\" && systemctl stop nginx",
			wantName:    "halt",
			wantErr:     false,
		},
		{
			name:        "command with both watch and stop in text",
			input:       "manage: watch -n 5 \"systemctl status app || systemctl stop app\";",
			wantCommand: "watch -n 5 \"systemctl status app || systemctl stop app\"",
			wantName:    "manage",
			wantErr:     false,
		},
		// Test POSIX shell constructs with braces
		{
			name:        "find command with braces",
			input:       "cleanup: find . -name \"*.tmp\" -exec rm {} \\;;",
			wantCommand: "find . -name \"*.tmp\" -exec rm {} \\;",
			wantName:    "cleanup",
			wantErr:     false,
		},
		{
			name:        "complex find with multiple braces",
			input:       "deep-clean: find . \\( -name \"*.log\" -o -name \"*.tmp\" \\) -exec rm {} \\;;",
			wantCommand: "find . \\( -name \"*.log\" -o -name \"*.tmp\" \\) -exec rm {} \\;",
			wantName:    "deep-clean",
			wantErr:     false,
		},
		{
			name:        "test command with braces",
			input:       "check-files: test -f {} && echo \"File exists\" || echo \"Missing\";",
			wantCommand: "test -f {} && echo \"File exists\" || echo \"Missing\"",
			wantName:    "check-files",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Parse(tt.input)

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Fatalf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			// Ensure we have exactly one command
			if len(result.Commands) != 1 {
				t.Fatalf("Expected 1 command, got %d", len(result.Commands))
			}

			// Check command properties
			cmd := result.Commands[0]
			if cmd.Name != tt.wantName {
				t.Errorf("Command name = %q, want %q", cmd.Name, tt.wantName)
			}

			if cmd.Command != tt.wantCommand {
				t.Errorf("Command text = %q, want %q", cmd.Command, tt.wantCommand)
			}
		})
	}
}

func TestDefinitions(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantName  string
		wantValue string
		wantErr   bool
	}{
		{
			name:      "simple definition",
			input:     "def SRC = ./src;",
			wantName:  "SRC",
			wantValue: "./src",
			wantErr:   false,
		},
		{
			name:      "definition with complex value",
			input:     "def CMD = go test -v ./...;",
			wantName:  "CMD",
			wantValue: "go test -v ./...",
			wantErr:   false,
		},
		{
			name:      "definition with special chars",
			input:     "def PATH = /usr/local/bin:$PATH;",
			wantName:  "PATH",
			wantValue: "/usr/local/bin:$PATH",
			wantErr:   false,
		},
		{
			name:      "definition with quotes",
			input:     `def MSG = "Hello, World!";`,
			wantName:  "MSG",
			wantValue: `"Hello, World!"`,
			wantErr:   false,
		},
		{
			name:      "definition with empty value",
			input:     "def EMPTY = ;",
			wantName:  "EMPTY",
			wantValue: "",
			wantErr:   false,
		},
		{
			name:      "definition with integer",
			input:     "def PORT = 8080;",
			wantName:  "PORT",
			wantValue: "8080",
			wantErr:   false,
		},
		{
			name:      "definition with decimal",
			input:     "def VERSION = 1.5;",
			wantName:  "VERSION",
			wantValue: "1.5",
			wantErr:   false,
		},
		{
			name:      "definition with dot-leading decimal",
			input:     "def FACTOR = .75;",
			wantName:  "FACTOR",
			wantValue: ".75",
			wantErr:   false,
		},
		{
			name:      "definition with number in mixed value",
			input:     "def TIMEOUT = 30s;",
			wantName:  "TIMEOUT",
			wantValue: "30s",
			wantErr:   false,
		},
		// New edge cases for parentheses in definitions
		{
			name:      "definition with parentheses",
			input:     "def CHECK_CMD = (which go && echo \"found\");",
			wantName:  "CHECK_CMD",
			wantValue: "(which go && echo \"found\")",
			wantErr:   false,
		},
		{
			name:      "definition with watch/stop keywords",
			input:     "def MONITOR = watch -n 1 \"ps aux | grep myapp\";",
			wantName:  "MONITOR",
			wantValue: "watch -n 1 \"ps aux | grep myapp\"",
			wantErr:   false,
		},
		{
			name:      "definition with braces",
			input:     "def FIND_CMD = find . -name \"*.tmp\" -exec rm {} \\;;",
			wantName:  "FIND_CMD",
			wantValue: "find . -name \"*.tmp\" -exec rm {} \\;",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Parse(tt.input)

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Fatalf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			// Ensure we have exactly one definition
			if len(result.Definitions) != 1 {
				t.Fatalf("Expected 1 definition, got %d", len(result.Definitions))
			}

			// Check definition properties
			def := result.Definitions[0]
			if def.Name != tt.wantName {
				t.Errorf("Definition name = %q, want %q", def.Name, tt.wantName)
			}

			if def.Value != tt.wantValue {
				t.Errorf("Definition value = %q, want %q", def.Value, tt.wantValue)
			}
		})
	}
}

func TestBlockCommands(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantName       string
		wantBlockSize  int
		wantCommands   []string
		wantBackground []bool
		wantErr        bool
	}{
		{
			name:           "empty block",
			input:          "setup: { }",
			wantName:       "setup",
			wantBlockSize:  0,
			wantCommands:   []string{},
			wantBackground: []bool{},
			wantErr:        false,
		},
		{
			name:           "single statement block",
			input:          "setup: { npm install }",
			wantName:       "setup",
			wantBlockSize:  1,
			wantCommands:   []string{"npm install"},
			wantBackground: []bool{false},
			wantErr:        false,
		},
		{
			name:           "multiple statements",
			input:          "setup: { npm install; go mod tidy; echo done }",
			wantName:       "setup",
			wantBlockSize:  3,
			wantCommands:   []string{"npm install", "go mod tidy", "echo done"},
			wantBackground: []bool{false, false, false},
			wantErr:        false,
		},
		{
			name:           "multiline block",
			input:          "setup: {\n  npm install;\n  go mod tidy;\n  echo done\n}",
			wantName:       "setup",
			wantBlockSize:  3,
			wantCommands:   []string{"npm install", "go mod tidy", "echo done"},
			wantBackground: []bool{false, false, false},
			wantErr:        false,
		},
		{
			name:           "background processes",
			input:          "run-all: { server &; client &; db & }",
			wantName:       "run-all",
			wantBlockSize:  3,
			wantCommands:   []string{"server", "client", "db"},
			wantBackground: []bool{true, true, true},
			wantErr:        false,
		},
		{
			name:           "mixed background and foreground",
			input:          "run: { setup; server &; monitor }",
			wantName:       "run",
			wantBlockSize:  3,
			wantCommands:   []string{"setup", "server", "monitor"},
			wantBackground: []bool{false, true, false},
			wantErr:        false,
		},
		// New edge cases for parentheses in block commands
		{
			name:           "block with parentheses in commands",
			input:          "check: { (which go || echo \"not found\"); echo \"done\" }",
			wantName:       "check",
			wantBlockSize:  2,
			wantCommands:   []string{"(which go || echo \"not found\")", "echo \"done\""},
			wantBackground: []bool{false, false},
			wantErr:        false,
		},
		{
			name:           "block with background subshells",
			input:          "parallel: { (long-task1 && echo \"task1 done\") &; (long-task2 && echo \"task2 done\") & }",
			wantName:       "parallel",
			wantBlockSize:  2,
			wantCommands:   []string{"(long-task1 && echo \"task1 done\")", "(long-task2 && echo \"task2 done\")"},
			wantBackground: []bool{true, true},
			wantErr:        false,
		},
		{
			name:           "block with watch/stop keywords in command text",
			input:          "services: { watch -n 1 \"ps aux\" &; echo \"stop when ready\" }",
			wantName:       "services",
			wantBlockSize:  2,
			wantCommands:   []string{"watch -n 1 \"ps aux\"", "echo \"stop when ready\""},
			wantBackground: []bool{true, false},
			wantErr:        false,
		},
		// New edge cases for POSIX braces in block commands
		{
			name:           "block with find commands using braces",
			input:          "cleanup: { find . -name \"*.tmp\" -exec rm {} \\;; echo \"cleanup done\" }",
			wantName:       "cleanup",
			wantBlockSize:  2,
			wantCommands:   []string{"find . -name \"*.tmp\" -exec rm {} \\;", "echo \"cleanup done\""},
			wantBackground: []bool{false, false},
			wantErr:        false,
		},
		{
			name:           "block with complex braces and background",
			input:          "parallel-clean: { find /tmp -name \"*.log\" -exec rm {} \\; &; find /var -name \"*.tmp\" -exec rm {} \\; & }",
			wantName:       "parallel-clean",
			wantBlockSize:  2,
			wantCommands:   []string{"find /tmp -name \"*.log\" -exec rm {} \\;", "find /var -name \"*.tmp\" -exec rm {} \\;"},
			wantBackground: []bool{true, true},
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Parse(tt.input)

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Fatalf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			// Ensure we have exactly one command
			if len(result.Commands) != 1 {
				t.Fatalf("Expected 1 command, got %d", len(result.Commands))
			}

			// Check command properties
			cmd := result.Commands[0]
			if cmd.Name != tt.wantName {
				t.Errorf("Command name = %q, want %q", cmd.Name, tt.wantName)
			}

			if !cmd.IsBlock {
				t.Errorf("Expected IsBlock to be true")
			}

			if len(cmd.Block) != tt.wantBlockSize {
				t.Fatalf("Block size = %d, want %d", len(cmd.Block), tt.wantBlockSize)
			}

			// Check each statement in the block
			for i := 0; i < tt.wantBlockSize; i++ {
				if i >= len(cmd.Block) {
					t.Fatalf("Missing block statement %d", i)
				}

				stmt := cmd.Block[i]
				if stmt.Command != tt.wantCommands[i] {
					t.Errorf("Block[%d].Command = %q, want %q", i, stmt.Command, tt.wantCommands[i])
				}

				if stmt.Background != tt.wantBackground[i] {
					t.Errorf("Block[%d].Background = %v, want %v", i, stmt.Background, tt.wantBackground[i])
				}
			}
		})
	}
}

func TestWatchStopCommands(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantName  string
		wantWatch bool
		wantStop  bool
		wantText  string
		wantBlock bool
		wantErr   bool
	}{
		{
			name:      "simple watch command",
			input:     "watch server: npm start;",
			wantName:  "server",
			wantWatch: true,
			wantStop:  false,
			wantText:  "npm start",
			wantBlock: false,
			wantErr:   false,
		},
		{
			name:      "simple stop command",
			input:     "stop server: pkill node;",
			wantName:  "server",
			wantWatch: false,
			wantStop:  true,
			wantText:  "pkill node",
			wantBlock: false,
			wantErr:   false,
		},
		{
			name:      "watch command with block",
			input:     "watch dev: {\nnpm start &;\ngo run main.go &\n}",
			wantName:  "dev",
			wantWatch: true,
			wantStop:  false,
			wantText:  "",
			wantBlock: true,
			wantErr:   false,
		},
		{
			name:      "stop command with block",
			input:     "stop dev: {\npkill node;\npkill go\n}",
			wantName:  "dev",
			wantWatch: false,
			wantStop:  true,
			wantText:  "",
			wantBlock: true,
			wantErr:   false,
		},
		// New edge cases for parentheses in watch/stop commands
		{
			name:      "watch command with parentheses",
			input:     "watch api: (cd api && npm start);",
			wantName:  "api",
			wantWatch: true,
			wantStop:  false,
			wantText:  "(cd api && npm start)",
			wantBlock: false,
			wantErr:   false,
		},
		{
			name:      "stop command with complex parentheses",
			input:     "stop services: (pkill -f \"node.*server\" || echo \"no node processes\");",
			wantName:  "services",
			wantWatch: false,
			wantStop:  true,
			wantText:  "(pkill -f \"node.*server\" || echo \"no node processes\")",
			wantBlock: false,
			wantErr:   false,
		},
		{
			name:      "watch block with parentheses and keywords",
			input:     "watch monitor: {\n(watch -n 1 \"ps aux\") &;\necho \"stop monitoring with Ctrl+C\"\n}",
			wantName:  "monitor",
			wantWatch: true,
			wantStop:  false,
			wantText:  "",
			wantBlock: true,
			wantErr:   false,
		},
		// New edge cases for POSIX braces in watch/stop commands
		{
			name:      "watch command with find and braces",
			input:     "watch cleanup: find . -name \"*.tmp\" -exec rm {} \\;;",
			wantName:  "cleanup",
			wantWatch: true,
			wantStop:  false,
			wantText:  "find . -name \"*.tmp\" -exec rm {} \\;",
			wantBlock: false,
			wantErr:   false,
		},
		{
			name:      "stop command with test and braces",
			input:     "stop validator: test -f {} && rm {} || echo \"file not found\";",
			wantName:  "validator",
			wantWatch: false,
			wantStop:  true,
			wantText:  "test -f {} && rm {} || echo \"file not found\"",
			wantBlock: false,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Parse(tt.input)

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Fatalf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			// Ensure we have exactly one command
			if len(result.Commands) != 1 {
				t.Fatalf("Expected 1 command, got %d", len(result.Commands))
			}

			// Check command properties
			cmd := result.Commands[0]
			if cmd.Name != tt.wantName {
				t.Errorf("Command name = %q, want %q", cmd.Name, tt.wantName)
			}

			if cmd.IsWatch != tt.wantWatch {
				t.Errorf("IsWatch = %v, want %v", cmd.IsWatch, tt.wantWatch)
			}

			if cmd.IsStop != tt.wantStop {
				t.Errorf("IsStop = %v, want %v", cmd.IsStop, tt.wantStop)
			}

			if cmd.IsBlock != tt.wantBlock {
				t.Errorf("IsBlock = %v, want %v", cmd.IsBlock, tt.wantBlock)
			}

			// For simple commands, check the command text
			if !tt.wantBlock && cmd.Command != tt.wantText {
				t.Errorf("Command text = %q, want %q", cmd.Command, tt.wantText)
			}
		})
	}
}

func TestVariableReferences(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantExpanded string
		wantErr      bool
	}{
		{
			name:         "simple variable reference",
			input:        "def SRC = ./src;\nbuild: cd $(SRC) && make;",
			wantExpanded: "cd ./src && make",
			wantErr:      false,
		},
		{
			name:         "multiple variable references",
			input:        "def SRC = ./src;\ndef BIN = ./bin;\nbuild: cp $(SRC)/main $(BIN)/app;",
			wantExpanded: "cp ./src/main ./bin/app",
			wantErr:      false,
		},
		{
			name:         "variable in block command",
			input:        "def SRC = ./src;\nsetup: { cd $(SRC); make all }",
			wantExpanded: "cd ./src", // Check just first statement
			wantErr:      false,
		},
		{
			name:         "escaped dollar sign",
			input:        "def PATH = /bin;\necho: echo \\$PATH is $(PATH);",
			wantExpanded: "echo $PATH is /bin",
			wantErr:      false,
		},
		{
			name:         "undefined variable",
			input:        "build: echo $(UNDEFINED);",
			wantExpanded: "",
			wantErr:      true, // Should fail during ExpandVariables
		},
		// New edge cases for parentheses with variables
		{
			name:         "variable with parentheses in value",
			input:        "def CHECK = (which go || echo \"not found\");\nvalidate: $(CHECK);",
			wantExpanded: "(which go || echo \"not found\")",
			wantErr:      false,
		},
		{
			name:         "variable in parentheses expression",
			input:        "def CMD = make clean;\nbuild: ($(CMD) && echo \"cleaned\") || echo \"failed\";",
			wantExpanded: "(make clean && echo \"cleaned\") || echo \"failed\"",
			wantErr:      false,
		},
		// New edge cases for POSIX braces with variables
		{
			name:         "variable with find braces",
			input:        "def PATTERN = \"*.tmp\";\ncleanup: find . -name $(PATTERN) -exec rm {} \\;;",
			wantExpanded: "find . -name \"*.tmp\" -exec rm {} \\;",
			wantErr:      false,
		},
		{
			name:         "variable with escaped characters",
			input:        "def MSG = \"Cost: \\$50\";\necho: echo $(MSG);",
			wantExpanded: "echo \"Cost: $50\"",
			wantErr:      false,
		},
		{
			name:         "variable with test braces",
			input:        "def FILE = config.json;\ncheck: test -f $(FILE) && echo \"found {}\" || echo \"not found\";",
			wantExpanded: "test -f config.json && echo \"found {}\" || echo \"not found\"",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			result, err := Parse(tt.input)
			if err != nil {
				if !tt.wantErr {
					t.Fatalf("Parse() error = %v", err)
				}
				return
			}

			// Try to expand variables
			err = result.ExpandVariables()
			if (err != nil) != tt.wantErr {
				t.Fatalf("ExpandVariables() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			// Check the expanded command
			if len(result.Commands) == 0 {
				t.Fatalf("No commands found")
			}

			cmd := result.Commands[0]
			var expandedText string

			if cmd.IsBlock {
				if len(cmd.Block) == 0 {
					t.Fatalf("No block statements found")
				}
				expandedText = cmd.Block[0].Command
			} else {
				expandedText = cmd.Command
			}

			if expandedText != tt.wantExpanded {
				t.Errorf("Expanded text = %q, want %q", expandedText, tt.wantExpanded)
			}
		})
	}
}

// Add these test functions to your parser_test.go file

func TestDollarSyntaxHandling(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantExpanded string
		wantErr      bool
	}{
		{
			name:         "escaped shell command substitution - simple",
			input:        "date: echo \\$(date);",
			wantExpanded: "echo $(date)",
			wantErr:      false,
		},
		{
			name:         "escaped shell command substitution - complex",
			input:        "info: echo \"Current time: \\$(date '+%Y-%m-%d %H:%M:%S')\";",
			wantExpanded: "echo \"Current time: $(date '+%Y-%m-%d %H:%M:%S')\"",
			wantErr:      false,
		},
		{
			name:         "escaped devcmd variable reference",
			input:        "def SRC = ./src;\necho: echo \"Variable syntax: \\$(SRC)\";",
			wantExpanded: "echo \"Variable syntax: $(SRC)\"",
			wantErr:      false,
		},
		{
			name:         "mixed escaped and real variable references",
			input:        "def DIR = /tmp;\ncmd: echo \"Real: $(DIR), Escaped: \\$(whoami)\";",
			wantExpanded: "echo \"Real: /tmp, Escaped: $(whoami)\"",
			wantErr:      false,
		},
		{
			name:         "escaped shell variable vs devcmd variable",
			input:        "def PATH = mypath;\ncmd: echo \"Devcmd: $(PATH), Shell: \\$PATH\";",
			wantExpanded: "echo \"Devcmd: mypath, Shell: $PATH\"",
			wantErr:      false,
		},
		{
			name:         "nested shell command substitution - escaped",
			input:        "complex: echo \\$(echo \\$(date));",
			wantExpanded: "echo $(echo $(date))",
			wantErr:      false,
		},
		{
			name:         "shell command substitution with pipes",
			input:        "pipeline: echo \\$(ps aux | grep node | wc -l);",
			wantExpanded: "echo $(ps aux | grep node | wc -l)",
			wantErr:      false,
		},
		{
			name:         "arithmetic expansion - escaped",
			input:        "math: echo \\$((2 + 3));",
			wantExpanded: "echo $((2 + 3))",
			wantErr:      false,
		},
		{
			name:         "parameter expansion - escaped",
			input:        "param: echo \\${HOME}/bin;",
			wantExpanded: "echo ${HOME}/bin",
			wantErr:      false,
		},
		{
			name:         "complex mixed case",
			input:        "def SRC = ./src;\ncomplex: cd $(SRC) && echo \\$(pwd) && echo \\$USER;",
			wantExpanded: "cd ./src && echo $(pwd) && echo $USER",
			wantErr:      false,
		},
		{
			name:         "dockerfile-like syntax",
			input:        "def IMAGE = myapp;\nbuild: docker build -t $(IMAGE) . && echo \\$(docker images | grep $(IMAGE));",
			wantExpanded: "docker build -t myapp . && echo $(docker images | grep myapp)",
			wantErr:      false,
		},
		{
			name:         "escaped dollar in quotes",
			input:        "quote: echo \"Price: \\$10\" && echo \\$(date);",
			wantExpanded: "echo \"Price: $10\" && echo $(date)",
			wantErr:      false,
		},
		{
			name:         "multiple escapes in sequence",
			input:        "multi: echo \\$HOME \\$(whoami) \\$((1+1));",
			wantExpanded: "echo $HOME $(whoami) $((1+1))",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			result, err := Parse(tt.input)
			if err != nil {
				if !tt.wantErr {
					t.Fatalf("Parse() error = %v", err)
				}
				return
			}

			// Try to expand variables
			err = result.ExpandVariables()
			if (err != nil) != tt.wantErr {
				t.Fatalf("ExpandVariables() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			// Check the expanded command
			if len(result.Commands) == 0 {
				t.Fatalf("No commands found")
			}

			cmd := result.Commands[0]
			var expandedText string

			if cmd.IsBlock {
				if len(cmd.Block) == 0 {
					t.Fatalf("No block statements found")
				}
				expandedText = cmd.Block[0].Command
			} else {
				expandedText = cmd.Command
			}

			if expandedText != tt.wantExpanded {
				t.Errorf("Expanded text = %q, want %q", expandedText, tt.wantExpanded)
			}
		})
	}
}

func TestDollarSyntaxInBlocks(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantBlockSize int
		wantCommands  []string
		wantErr       bool
	}{
		{
			name:          "block with mixed dollar syntax",
			input:         "def PORT = 8080;\nsetup: { echo \"Starting on port $(PORT)\"; echo \\$(date); echo \"PID: \\$\\$\" }",
			wantBlockSize: 3,
			wantCommands:  []string{"echo \"Starting on port 8080\"", "echo $(date)", "echo \"PID: $$\""},
			wantErr:       false,
		},
		{
			name:          "watch block with shell command substitution",
			input:         "watch dev: { echo \"Started at \\$(date)\"; npm start &; echo \"PID: \\$!\" }",
			wantBlockSize: 3,
			wantCommands:  []string{"echo \"Started at $(date)\"", "npm start", "echo \"PID: $!\""},
			wantErr:       false,
		},
		{
			name:          "block with environment variable handling",
			input:         "def APP = myapp;\nenv: { export APP_NAME=$(APP); echo \\$APP_NAME; echo \\$(printenv APP_NAME) }",
			wantBlockSize: 3,
			wantCommands:  []string{"export APP_NAME=myapp", "echo $APP_NAME", "echo $(printenv APP_NAME)"},
			wantErr:       false,
		},
		{
			name:          "block with docker commands",
			input:         "def IMAGE = node:18;\ndocker: { docker run -d --name myapp $(IMAGE); echo \"Container ID: \\$(docker ps -q -f name=myapp)\" }",
			wantBlockSize: 2,
			wantCommands:  []string{"docker run -d --name myapp node:18", "echo \"Container ID: $(docker ps -q -f name=myapp)\""},
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			result, err := Parse(tt.input)
			if err != nil {
				if !tt.wantErr {
					t.Fatalf("Parse() error = %v", err)
				}
				return
			}

			// Expand variables
			err = result.ExpandVariables()
			if (err != nil) != tt.wantErr {
				t.Fatalf("ExpandVariables() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			// Find the command with a block
			var cmd *Command
			for i := range result.Commands {
				if result.Commands[i].IsBlock {
					cmd = &result.Commands[i]
					break
				}
			}

			if cmd == nil {
				t.Fatalf("No block command found")
			}

			if len(cmd.Block) != tt.wantBlockSize {
				t.Fatalf("Block size = %d, want %d", len(cmd.Block), tt.wantBlockSize)
			}

			// Check each statement in the block
			for i := 0; i < tt.wantBlockSize; i++ {
				if i >= len(cmd.Block) {
					t.Fatalf("Missing block statement %d", i)
				}

				stmt := cmd.Block[i]
				if stmt.Command != tt.wantCommands[i] {
					t.Errorf("Block[%d].Command = %q, want %q", i, stmt.Command, tt.wantCommands[i])
				}
			}
		})
	}
}

func TestDollarSyntaxErrorCases(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantErrSubstr string
	}{
		{
			name:          "undefined variable with escaped syntax mix",
			input:         "test: echo $(UNDEFINED) \\$(date);",
			wantErrSubstr: "undefined variable",
		},
		{
			name:          "malformed escaped syntax should still parse",
			input:         "test: echo \\$(;", // Incomplete but should parse the escape
			wantErrSubstr: "",                 // Should not error on parsing, just produce the literal
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			result, err := Parse(tt.input)

			// Check if we should have a parsing error
			if tt.wantErrSubstr != "" {
				if err == nil {
					t.Fatalf("Expected parsing error containing %q, got nil", tt.wantErrSubstr)
				}
				if !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Errorf("Error = %q, want substring %q", err.Error(), tt.wantErrSubstr)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected parse error: %v", err)
			}

			// Try to expand variables - this is where semantic errors occur
			err = result.ExpandVariables()

			if tt.wantErrSubstr != "" {
				if err == nil {
					t.Fatalf("Expected error containing %q, got nil", tt.wantErrSubstr)
				}
				if !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Errorf("Error = %q, want substring %q", err.Error(), tt.wantErrSubstr)
				}
			} else if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
		})
	}
}

func TestDollarSyntaxWithContinuations(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantExpanded string
		wantErr      bool
	}{
		{
			name:         "escaped dollar with continuation",
			input:        "def DIR = /home;\ncmd: echo $(DIR) \\\n&& echo \\$(pwd);",
			wantExpanded: "echo /home && echo $(pwd)",
			wantErr:      false,
		},
		{
			name:         "complex shell substitution with continuation",
			input:        "complex: echo \\$(find . -name \"*.go\" \\\n| wc -l);",
			wantExpanded: "echo $(find . -name \"*.go\" | wc -l)",
			wantErr:      false,
		},
		{
			name:         "mixed syntax across continuation lines",
			input:        "def SRC = ./src;\nmulti: cd $(SRC) \\\n&& echo \\$(pwd) \\\n&& echo \"Done\";",
			wantExpanded: "cd ./src && echo $(pwd) && echo \"Done\"",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			result, err := Parse(tt.input)
			if err != nil {
				if !tt.wantErr {
					t.Fatalf("Parse() error = %v", err)
				}
				return
			}

			// Expand variables
			err = result.ExpandVariables()
			if (err != nil) != tt.wantErr {
				t.Fatalf("ExpandVariables() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			// Find the command (skip definitions)
			var cmd *Command
			for i := range result.Commands {
				if result.Commands[i].Name != "" && !strings.HasPrefix(result.Commands[i].Name, "def") {
					cmd = &result.Commands[i]
					break
				}
			}

			if cmd == nil {
				t.Fatalf("Command not found in result")
			}

			if cmd.Command != tt.wantExpanded {
				t.Errorf("Command text = %q, want %q", cmd.Command, tt.wantExpanded)
			}
		})
	}
}

func TestContinuationLines(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantCommand string
		wantErr     bool
	}{
		{
			name:        "simple continuation",
			input:       "build: echo hello \\\nworld;",
			wantCommand: "echo hello world",
			wantErr:     false,
		},
		{
			name:        "multiple continuations",
			input:       "build: echo hello \\\nworld \\\nuniverse;",
			wantCommand: "echo hello world universe",
			wantErr:     false,
		},
		{
			name:        "continuation with variables",
			input:       "def DIR = src;\nbuild: cd $(DIR) \\\n&& make;",
			wantCommand: "cd $(DIR) && make",
			wantErr:     false,
		},
		{
			name:        "continuation with indentation",
			input:       "build: echo hello \\\n    world;",
			wantCommand: "echo hello world",
			wantErr:     false,
		},
		// New edge cases for continuations with parentheses
		{
			name:        "continuation with parentheses",
			input:       "check: (which go \\\n|| echo \"not found\");",
			wantCommand: "(which go || echo \"not found\")",
			wantErr:     false,
		},
		{
			name:        "complex continuation with parentheses",
			input:       "setup: (cd src && \\\nmake clean) \\\n|| echo \"failed\";",
			wantCommand: "(cd src && make clean) || echo \"failed\"",
			wantErr:     false,
		},
		// New edge cases for continuations with POSIX braces
		{
			name:        "continuation with find braces",
			input:       "cleanup: find . -name \"*.tmp\" \\\n-exec rm {} \\;;",
			wantCommand: "find . -name \"*.tmp\" -exec rm {} \\;",
			wantErr:     false,
		},
		{
			name:        "complex continuation with braces",
			input:       "batch: find . \\( -name \"*.log\" \\\n-o -name \"*.tmp\" \\) \\\n-exec rm {} \\;;",
			wantCommand: "find . \\( -name \"*.log\" -o -name \"*.tmp\" \\) -exec rm {} \\;",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			result, err := Parse(tt.input)

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Fatalf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			// Find the actual command (might not be the first one in some tests)
			var cmd *Command
			for i := range result.Commands {
				if strings.Contains(result.Commands[i].Command, "echo") ||
					strings.Contains(result.Commands[i].Command, "cd") ||
					strings.Contains(result.Commands[i].Command, "find") ||
					strings.HasPrefix(result.Commands[i].Command, "(") {
					cmd = &result.Commands[i]
					break
				}
			}

			if cmd == nil {
				t.Fatalf("Command not found in result")
			}

			// Check the command text
			if cmd.Command != tt.wantCommand {
				t.Errorf("Command text = %q, want %q", cmd.Command, tt.wantCommand)
			}
		})
	}
}

func TestErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantErrSubstr string
	}{
		{
			name:          "duplicate command",
			input:         "build: echo hello;\nbuild: echo world;",
			wantErrSubstr: "duplicate command",
		},
		{
			name:          "duplicate definition",
			input:         "def VAR = value1;\ndef VAR = value2;",
			wantErrSubstr: "duplicate definition",
		},
		{
			name:          "syntax error in command",
			input:         "build echo hello;", // Missing colon
			wantErrSubstr: "missing ':'",       // Updated to match actual error
		},
		{
			name:          "unclosed block",
			input:         "build: { echo hello;",
			wantErrSubstr: "missing '}'", // Updated to match actual error
		},
		{
			name:          "bad variable expansion",
			input:         "build: echo $(missingVar);",
			wantErrSubstr: "undefined variable",
		},
		{
			name:          "missing semicolon in definition",
			input:         "def VAR = value\nbuild: echo hello;",
			wantErrSubstr: "missing ';'", // Updated to match actual error
		},
		{
			name:          "missing semicolon in simple command",
			input:         "build: echo hello\ntest: echo world;",
			wantErrSubstr: "missing ';'", // New test for semicolon requirement
		},
		// New edge cases for error handling with POSIX constructs
		{
			name:          "invalid find syntax with braces",
			input:         "cleanup: find . -name \"*.tmp\" -exec rm {;", // Missing closing brace
			wantErrSubstr: "missing ';'",                                 // Will likely be a syntax error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse and possibly expand variables
			result, err := Parse(tt.input)

			// If no syntax error, try expanding variables to catch semantic errors
			if err == nil && strings.Contains(tt.input, "$(") {
				err = result.ExpandVariables()
			}

			// Expect an error
			if err == nil {
				t.Fatalf("Expected error containing %q, got nil", tt.wantErrSubstr)
			}

			// Check that the error contains the expected substring
			if !strings.Contains(err.Error(), tt.wantErrSubstr) {
				t.Errorf("Error = %q, want substring %q", err.Error(), tt.wantErrSubstr)
			}
		})
	}
}

func TestCompleteFile(t *testing.T) {
	input := `
# Development commands
def SRC = ./src;
def BIN = ./bin;

# Build commands
build: cd $(SRC) && make all;

# Run commands
watch server: {
  cd $(SRC);
  ./server --port=8080 &;
  ./worker --queue=jobs &
}

stop server: pkill -f "server|worker";

# Complex commands with parentheses and keywords
check-deps: (which go && echo "Go found") || (echo "Go missing" && exit 1);

monitor: {
  watch -n 1 "ps aux | grep server";
  echo "Use stop server to halt processes"
}

# POSIX shell commands with braces
cleanup: find . -name "*.tmp" -exec rm {} \;;

batch-clean: {
  find /tmp -name "*.log" -exec rm {} \; &;
  find /var -name "*.tmp" -exec rm {} \; &;
  wait
}
`

	result, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Verify definitions
	if len(result.Definitions) != 2 {
		t.Errorf("Expected 2 definitions, got %d", len(result.Definitions))
	} else {
		defNames := map[string]string{
			result.Definitions[0].Name: result.Definitions[0].Value,
			result.Definitions[1].Name: result.Definitions[1].Value,
		}

		if defNames["SRC"] != "./src" {
			t.Errorf("Definition SRC = %q, want %q", defNames["SRC"], "./src")
		}

		if defNames["BIN"] != "./bin" {
			t.Errorf("Definition BIN = %q, want %q", defNames["BIN"], "./bin")
		}
	}

	// Verify commands - we expect 7 commands: build, watch server, stop server, check-deps, monitor, cleanup, batch-clean
	if len(result.Commands) != 7 {
		t.Errorf("Expected 7 commands, got %d", len(result.Commands))
	} else {
		// Find commands by type since we can have both watch and stop with same name
		var buildCmd *Command
		var watchServerCmd *Command
		var stopServerCmd *Command
		var checkDepsCmd *Command
		var monitorCmd *Command
		var cleanupCmd *Command
		var batchCleanCmd *Command

		for i := range result.Commands {
			cmd := &result.Commands[i]
			switch {
			case cmd.Name == "build" && !cmd.IsWatch && !cmd.IsStop:
				buildCmd = cmd
			case cmd.Name == "server" && cmd.IsWatch:
				watchServerCmd = cmd
			case cmd.Name == "server" && cmd.IsStop:
				stopServerCmd = cmd
			case cmd.Name == "check-deps":
				checkDepsCmd = cmd
			case cmd.Name == "monitor":
				monitorCmd = cmd
			case cmd.Name == "cleanup":
				cleanupCmd = cmd
			case cmd.Name == "batch-clean":
				batchCleanCmd = cmd
			}
		}

		// Check build command
		if buildCmd == nil {
			t.Errorf("Missing 'build' command")
		} else if buildCmd.Command != "cd $(SRC) && make all" {
			t.Errorf("build command = %q, want %q", buildCmd.Command, "cd $(SRC) && make all")
		}

		// Check watch server command
		if watchServerCmd == nil {
			t.Errorf("Missing 'watch server' command")
		} else {
			if !watchServerCmd.IsWatch {
				t.Errorf("Expected server command to be a watch command")
			}

			if !watchServerCmd.IsBlock {
				t.Errorf("Expected server command to be a block command")
			}

			if len(watchServerCmd.Block) != 3 {
				t.Errorf("Expected 3 block statements in server command, got %d", len(watchServerCmd.Block))
			} else {
				// Check for background statements
				backgroundCount := 0
				for _, stmt := range watchServerCmd.Block {
					if stmt.Background {
						backgroundCount++
					}
				}

				if backgroundCount != 2 {
					t.Errorf("Expected 2 background statements, got %d", backgroundCount)
				}
			}
		}

		// Check stop server command
		if stopServerCmd == nil {
			t.Errorf("Missing 'stop server' command")
		} else {
			if !stopServerCmd.IsStop {
				t.Errorf("Expected stop server command to be a stop command")
			}

			if stopServerCmd.IsBlock {
				t.Errorf("Expected stop server command to be a simple command, not a block")
			}
		}

		// Check check-deps command (contains parentheses)
		if checkDepsCmd == nil {
			t.Errorf("Missing 'check-deps' command")
		} else {
			expectedCmd := "(which go && echo \"Go found\") || (echo \"Go missing\" && exit 1)"
			if checkDepsCmd.Command != expectedCmd {
				t.Errorf("check-deps command = %q, want %q", checkDepsCmd.Command, expectedCmd)
			}
		}

		// Check monitor command (contains watch/stop keywords in text)
		if monitorCmd == nil {
			t.Errorf("Missing 'monitor' command")
		} else {
			if !monitorCmd.IsBlock {
				t.Errorf("Expected monitor command to be a block command")
			}

			if len(monitorCmd.Block) != 2 {
				t.Errorf("Expected 2 block statements in monitor command, got %d", len(monitorCmd.Block))
			} else {
				// First statement should contain 'watch' keyword
				firstStmt := monitorCmd.Block[0].Command
				if !strings.Contains(firstStmt, "watch -n 1") {
					t.Errorf("Expected first statement to contain 'watch -n 1', got: %q", firstStmt)
				}

				// Second statement should contain 'stop' keyword
				secondStmt := monitorCmd.Block[1].Command
				if !strings.Contains(secondStmt, "stop server") {
					t.Errorf("Expected second statement to contain 'stop server', got: %q", secondStmt)
				}
			}
		}

		// Check cleanup command (contains POSIX braces)
		if cleanupCmd == nil {
			t.Errorf("Missing 'cleanup' command")
		} else {
			expectedCmd := "find . -name \"*.tmp\" -exec rm {} \\;"
			if cleanupCmd.Command != expectedCmd {
				t.Errorf("cleanup command = %q, want %q", cleanupCmd.Command, expectedCmd)
			}
		}

		// Check batch-clean command (contains POSIX braces in block)
		if batchCleanCmd == nil {
			t.Errorf("Missing 'batch-clean' command")
		} else {
			if !batchCleanCmd.IsBlock {
				t.Errorf("Expected batch-clean command to be a block command")
			}

			if len(batchCleanCmd.Block) != 3 {
				t.Errorf("Expected 3 block statements in batch-clean command, got %d", len(batchCleanCmd.Block))
			} else {
				// First two statements should be background find commands
				for i := 0; i < 2; i++ {
					stmt := batchCleanCmd.Block[i]
					if !stmt.Background {
						t.Errorf("Expected statement %d to be background", i)
					}
					if !strings.Contains(stmt.Command, "find") {
						t.Errorf("Expected statement %d to contain 'find', got: %q", i, stmt.Command)
					}
					if !strings.Contains(stmt.Command, "-exec rm {} \\;") {
						t.Errorf("Expected statement %d to contain '-exec rm {} \\;', got: %q", i, stmt.Command)
					}
				}

				// Third statement should be wait command
				waitStmt := batchCleanCmd.Block[2]
				if waitStmt.Background {
					t.Errorf("Expected wait statement to be foreground")
				}
				if waitStmt.Command != "wait" {
					t.Errorf("Expected wait command, got: %q", waitStmt.Command)
				}
			}
		}

		// Verify variable expansion
		err = result.ExpandVariables()
		if err != nil {
			t.Fatalf("ExpandVariables() error = %v", err)
		}

		// Check that variables were expanded in the build command
		if buildCmd != nil && buildCmd.Command != "cd ./src && make all" {
			t.Errorf("Expanded build command = %q, want %q", buildCmd.Command, "cd ./src && make all")
		}
	}
}
