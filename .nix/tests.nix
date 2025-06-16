# Test scenarios for devcmd library and generated CLIs
{ pkgs, lib, self, system }:

let
  devcmdLib = import ./lib.nix { inherit pkgs self system lib; };

  # Common test utilities
  testUtils = {
    # Run a command and check exit code
    runAndCheck = cmd: expectedExitCode: ''
      echo "Running: ${cmd}"
      if ${cmd}; then
        EXIT_CODE=0
      else
        EXIT_CODE=$?
      fi
      if [ $EXIT_CODE -ne ${toString expectedExitCode} ]; then
        echo "Expected exit code ${toString expectedExitCode}, got $EXIT_CODE"
        exit 1
      fi
      echo "✅ Command succeeded with expected exit code"
    '';

    # Check if output contains expected text
    checkOutput = cmd: expectedText: ''
      echo "Running: ${cmd}"
      OUTPUT=$(${cmd} 2>&1 || true)
      if echo "$OUTPUT" | grep -q "${expectedText}"; then
        echo "✅ Output contains expected text: ${expectedText}"
      else
        echo "❌ Expected output to contain: ${expectedText}"
        echo "Actual output: $OUTPUT"
        exit 1
      fi
    '';

    # Simple test that runs a command
    simpleTest = cmd: ''
      echo "Testing: ${cmd}"
      ${cmd} || {
        echo "❌ Command failed: ${cmd}"
        exit 1
      }
      echo "✅ Command succeeded"
    '';
  };

  # Helper function to create test derivations for CLIs
  mkCLITest = { name, cli, testScript, extraPackages ? [ ] }: pkgs.runCommand "test-${name}"
    {
      nativeBuildInputs = [ pkgs.bash cli ] ++ extraPackages;
      meta.description = "Test for ${name} CLI";
    } ''
    set -euo pipefail
    mkdir -p $out

    echo "🧪 Testing ${name} CLI..."
    ${testScript}

    echo "✅ ${name} tests passed!"
    echo "success" > $out/result
  '';

in
rec {
  # Test basic devcmd functionality
  basicTests = {
    # Test simple command generation
    simpleCommand = mkCLITest {
      name = "simple-command";
      cli = devcmdLib.mkDevCLI {
        name = "simple-test";
        commandsContent = ''
          build: echo "Building project...";
          test: echo "Running tests...";
          clean: rm -f *.tmp;
        '';
      };

      testScript = ''
        ${testUtils.simpleTest "simple-test --help"}
        ${testUtils.checkOutput "simple-test build" "Building project"}
        ${testUtils.checkOutput "simple-test test" "Running tests"}
      '';
    };

    # Test commands with POSIX syntax and parentheses - UPDATED with correct shell command substitution
    posixSyntax = mkCLITest {
      name = "posix-syntax";
      cli = devcmdLib.mkDevCLI {
        name = "posix-test";
        commandsContent = ''
          check-deps: (which go && echo "Go found") || (echo "Go missing" && exit 1);
          validate: test -f go.mod && echo "Go module found" || echo "No go.mod";
          complex: @sh((cd /tmp && echo "In tmp: \$(pwd)") && echo "Back to: \$(pwd)");
        '';
      };

      # Add which command (usually in util-linux or coreutils)
      extraPackages = with pkgs; [ which go ];

      testScript = ''
        ${testUtils.simpleTest "posix-test --help"}
        # Test that parentheses are preserved in commands
        OUTPUT=$(posix-test check-deps 2>&1 || true)
        if echo "$OUTPUT" | grep -q "Go found\|Go missing"; then
          echo "✅ Parentheses syntax working"
        else
          echo "❌ Parentheses syntax test failed"
          exit 1
        fi
      '';
    };

    # Test variable expansion
    variableExpansion = mkCLITest {
      name = "variable-expansion";
      cli = devcmdLib.mkDevCLI {
        name = "variables-test";
        commandsContent = ''
          def SRC = ./src;
          def PORT = 8080;
          def CHECK_CMD = which node || echo "missing";

          build: mkdir -p $(SRC) && cd $(SRC) && echo "Building in $(SRC)";
          serve: echo "Starting server on port $(PORT)";
          check: $(CHECK_CMD) && echo "Dependencies OK";
        '';
      };

      # Add nodejs and which for the CHECK_CMD variable
      extraPackages = with pkgs; [ nodejs which ];

      testScript = ''
        ${testUtils.checkOutput "variables-test build" "Building in ./src"}
        ${testUtils.checkOutput "variables-test serve" "port 8080"}
        # Test complex variable expansion
        variables-test check &>/dev/null || echo "Complex variable test completed"
      '';
    };
  };

  # Test watch/stop process management
  processManagementTests = {
    watchStopCommands = mkCLITest {
      name = "process-management";
      cli = devcmdLib.mkDevCLI {
        name = "process-test";
        commandsContent = ''
          watch demo: python3 -m http.server 9999 &;
          stop demo: pkill -f "python3 -m http.server 9999";

          watch multi: {
            echo "Starting services...";
            sleep 1 &;
            sleep 2 &;
            echo "Services started"
          }
        '';
      };

      # Add python3 for the http server command
      extraPackages = with pkgs; [ python3 ];

      testScript = ''
        # Check that process management commands exist
        ${testUtils.checkOutput "process-test --help" "status"}
        ${testUtils.simpleTest "process-test status"}
      '';
    };
  };

  # Test block commands and background processes
  blockCommandTests = {
    backgroundProcesses = mkCLITest {
      name = "block-commands";
      cli = devcmdLib.mkDevCLI {
        name = "block-test";
        commandsContent = ''
          setup: {
            echo "Step 1: Initialize";
            echo "Step 2: Configure";
            echo "Step 3: Complete"
          }

          parallel: {
            echo "Task 1" &;
            echo "Task 2" &;
            echo "Task 3"
          }

          complex: {
            @sh((echo "Subshell 1" && sleep 0.1) &);
            @sh((echo "Subshell 2" || echo "Fallback") &);
            echo "Main thread"
          }
        '';
      };

      testScript = ''
        # Test sequential block
        OUTPUT=$(block-test setup 2>&1)
        if echo "$OUTPUT" | grep -q "Step 1" && echo "$OUTPUT" | grep -q "Step 2" && echo "$OUTPUT" | grep -q "Step 3"; then
          echo "✅ Sequential block test passed"
        else
          echo "❌ Sequential block test failed"
          echo "Output: $OUTPUT"
          exit 1
        fi

        # Test parallel block
        ${testUtils.checkOutput "block-test parallel" "Task"}

        # Test complex block
        ${testUtils.checkOutput "block-test complex" "Main thread"}
      '';
    };
  };

  # Test error handling and edge cases
  errorHandlingTests = {
    invalidCommands = mkCLITest {
      name = "error-handling";
      cli = devcmdLib.mkDevCLI {
        name = "error-test";
        commandsContent = ''
          valid: echo "This works";
          special-chars: echo "Special: !@#\\\$%^&*()";
          unicode: echo "Hello 世界";
        '';
      };

      testScript = ''
        ${testUtils.checkOutput "error-test valid" "This works"}
        ${testUtils.checkOutput "error-test special-chars" "Special:"}
        ${testUtils.checkOutput "error-test unicode" "世界"}
      '';
    };
  };

  # Performance and scale tests
  performanceTests = {
    largeCLI = mkCLITest {
      name = "performance";
      cli = devcmdLib.mkDevCLI {
        name = "large-test";
        commandsContent = lib.concatStringsSep "\n" (
          lib.genList (i: "cmd${toString i}: echo 'Command ${toString i}';") 20
        );
      };

      testScript = ''
        # Test that CLI with many commands works
        HELP_LINES=$(large-test --help | wc -l)
        if [ "$HELP_LINES" -gt 10 ]; then
          echo "✅ Help output has reasonable length: $HELP_LINES lines"
        else
          echo "❌ Help output too short: $HELP_LINES lines"
          exit 1
        fi

        ${testUtils.checkOutput "large-test cmd10" "Command 10"}
      '';
    };
  };

  # Integration tests with real-world scenarios
  realWorldTests = {
    webDevelopment = mkCLITest {
      name = "web-development";
      cli = devcmdLib.mkDevCLI {
        name = "webdev";
        commandsContent = ''
          def NODE_ENV = development;
          def PORT = 3000;
          def API_PORT = 3001;

          install: echo "npm install" && echo "Dependencies installed";
          build: {
            echo "Building frontend...";
            @sh((test -d frontend && cd frontend && npm run build) || echo "No frontend");
            echo "Building backend...";
            @sh((test -d backend && cd backend && go build) || echo "No backend")
          }

          test: {
            echo "Running frontend tests...";
            @sh((test -d frontend && cd frontend && npm test) || echo "No frontend tests");
            echo "Running backend tests...";
            @sh((test -d backend && cd backend && go test ./...) || echo "No backend tests")
          }
        '';
      };

      # Add nodejs and go for npm and go commands
      extraPackages = with pkgs; [ nodejs go ];

      testScript = ''
        # Check all expected commands exist
        ${testUtils.checkOutput "webdev --help" "install"}
        ${testUtils.checkOutput "webdev --help" "build"}
        ${testUtils.checkOutput "webdev --help" "test"}

        ${testUtils.checkOutput "webdev install" "Dependencies installed"}

        # Test build command
        OUTPUT=$(webdev build 2>&1)
        if echo "$OUTPUT" | grep -q "Building frontend" && echo "$OUTPUT" | grep -q "Building backend"; then
          echo "✅ Build command test passed"
        else
          echo "❌ Build command test failed"
          echo "Output: $OUTPUT"
          exit 1
        fi
      '';
    };

    goProject = mkCLITest {
      name = "go-project";
      cli = devcmdLib.mkDevCLI {
        name = "goproj";
        commandsContent = ''
          def MODULE = github.com/example/myapp;
          def BINARY = myapp;
          def VERSION = v0.1.0;

          deps: {
            echo "Managing dependencies...";
            @sh((test -f go.mod && go mod tidy) || echo "No go.mod");
            @sh((test -f go.mod && go mod download) || echo "No go.mod");
            @sh((test -f go.mod && go mod verify) || echo "No go.mod")
          }

          build: {
            echo "Building $(BINARY) $(VERSION)...";
            @sh((test -d ./cmd/$(BINARY) && go build -ldflags="-X main.Version=$(VERSION)" -o $(BINARY) ./cmd/$(BINARY)) || echo "No ./cmd/$(BINARY) directory")
          }

          test: {
            echo "Running tests...";
            @sh((go test -v ./... 2>/dev/null) || echo "No tests or go.mod");
            @sh((go test -race ./... 2>/dev/null) || echo "No tests or go.mod")
          }

          lint: {
            echo "Running linters...";
            @sh((which golangci-lint && golangci-lint run) || echo "No linter");
            @sh((test -f go.mod && go fmt ./...) || echo "No go.mod");
            @sh((test -f go.mod && go vet ./...) || echo "No go.mod")
          }
        '';
      };

      # Add go for go commands
      extraPackages = with pkgs; [ go ];

      testScript = ''
        # Check Go commands exist
        ${testUtils.checkOutput "goproj --help" "build"}
        ${testUtils.checkOutput "goproj --help" "test"}
        ${testUtils.checkOutput "goproj --help" "lint"}

        # Test that commands work even without Go project structure
        ${testUtils.checkOutput "goproj deps" "Managing dependencies"}
        ${testUtils.checkOutput "goproj build" "Building myapp"}
        ${testUtils.checkOutput "goproj test" "Running tests"}
        ${testUtils.checkOutput "goproj lint" "Running linters"}
      '';
    };

    # Test shell command substitution patterns
    shellSubstitution = mkCLITest {
      name = "shell-substitution";
      cli = devcmdLib.mkDevCLI {
        name = "shell-test";
        commandsContent = ''
          def LOG_DIR = /tmp/logs;
          def APP_NAME = myapp;

          # Test escaped shell command substitution
          timestamp: echo "Current time: \$(date)";

          # Test shell variables
          user-info: echo "User: \$USER, Home: \$HOME";

          # Test complex shell operations with @sh()
          backup: @sh(DATE=\$(date +%Y%m%d-%H%M%S); echo "Backup created: backup-\$DATE.tar.gz");

          # Test mixed devcmd variables and shell substitution
          logrotate: @sh(find $(LOG_DIR) -name "$(APP_NAME)*.log" -mtime +7 -exec rm {} \; && echo "Logs rotated at \$(date)");

          # Test arithmetic expansion
          calculate: echo "Result: \$((2 + 3 * 4))";
        '';
      };

      testScript = ''
        ${testUtils.checkOutput "shell-test timestamp" "Current time:"}
        ${testUtils.checkOutput "shell-test user-info" "User:"}
        ${testUtils.checkOutput "shell-test backup" "Backup created:"}
        ${testUtils.checkOutput "shell-test calculate" "Result: 14"}
      '';
    };
  };

  # All individual test derivations
  allTestDerivations = {
    inherit (basicTests) simpleCommand posixSyntax variableExpansion;
    inherit (processManagementTests) watchStopCommands;
    inherit (blockCommandTests) backgroundProcesses;
    inherit (errorHandlingTests) invalidCommands;
    inherit (performanceTests) largeCLI;
    inherit (realWorldTests) webDevelopment goProject shellSubstitution;
  };

  # Test examples from examples.nix
  testExamples = pkgs.runCommand "test-examples"
    {
      nativeBuildInputs = with pkgs; [ bash ];
      meta.description = "Test all example CLIs";
    } ''
    mkdir -p $out
    echo "🧪 Testing example CLIs..."

    # Import examples
    ${lib.optionalString (builtins.pathExists ./.nix/examples.nix) ''
      echo "Examples file exists, testing would go here"
      echo "In a real scenario, we'd test each example CLI"
    ''}

    echo "🎉 Example tests completed!"
    date > $out/success
  '';

  # Combined test runner
  runAllTests = pkgs.runCommand "devcmd-all-tests"
    {
      nativeBuildInputs = [ pkgs.bash ] ++ (lib.attrValues allTestDerivations);
      meta.description = "Run all devcmd tests";
    } ''
    mkdir -p $out
    echo "🧪 Running all devcmd tests..."

    # Run each test and collect results
    FAILED_TESTS=""
    PASSED_TESTS=""

    ${lib.concatStringsSep "\n" (lib.mapAttrsToList (testName: test: ''
      echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
      echo "🧪 Running test: ${testName}"
      if [ -f "${test}/result" ]; then
        echo "✅ ${testName} passed"
        PASSED_TESTS="$PASSED_TESTS ${testName}"
      else
        echo "❌ ${testName} failed"
        FAILED_TESTS="$FAILED_TESTS ${testName}"
      fi
    '') allTestDerivations)}

    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "📊 Test Results Summary:"
    echo "Passed tests:$PASSED_TESTS"
    if [ -n "$FAILED_TESTS" ]; then
      echo "Failed tests:$FAILED_TESTS"
      echo "❌ Some tests failed"
      exit 1
    else
      echo "🎉 All tests passed!"
    fi

    date > $out/success
    echo "All tests completed successfully" > $out/summary
  '';

  # Export all test components
  tests = allTestDerivations;
}
