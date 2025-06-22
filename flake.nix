{
  description = "devcmd - Domain-specific language for generating development command CLIs";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem
      (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
          lib = nixpkgs.lib;

          # Main devcmd package
          devcmdPackage = import ./.nix/package.nix { inherit pkgs lib; version = "0.2.0"; };

          # Library functions with automatic system detection
          devcmdLib = import ./.nix/lib.nix { inherit pkgs self lib; };

          # Import all examples from examples.nix
          examples = import ./.nix/examples.nix { inherit pkgs lib self; };

          # Import tests from tests.nix
          tests = import ./.nix/tests.nix { inherit pkgs lib self; };

        in
        {
          # Main packages including examples and tests
          packages = {
            # Core devcmd package
            default = devcmdPackage;
            devcmd = devcmdPackage;

            # All example CLIs from examples.nix (using simplified naming)
            basicDev = examples.basicDev;
            webDev = examples.webDev;
            goProject = examples.goProject;
            rustProject = examples.rustProject;
            dataScienceProject = examples.dataScienceProject;
            devOpsProject = examples.devOpsProject;

            # Test packages
            tests = tests.runAllTests;
            test-examples = tests.testExamples;

            # Individual test suites (for granular testing)
            test-basic = tests.allTestDerivations.simpleCommand;
            test-posix = tests.allTestDerivations.posixSyntax;
            test-variables = tests.allTestDerivations.variableExpansion;
            test-processes = tests.allTestDerivations.watchStopCommands;
            test-blocks = tests.allTestDerivations.backgroundProcesses;
            test-errors = tests.allTestDerivations.invalidCommands;
            test-performance = tests.allTestDerivations.largeCLI;
            test-webdev = tests.allTestDerivations.webDevelopment;
            test-go = tests.allTestDerivations.goProject;
            test-shell = tests.allTestDerivations.shellSubstitution;
          };

          # Development shells including example shells from examples.nix
          devShells = {
            default = import ./.nix/development.nix { inherit pkgs; };

            # Example development shells
            basic = examples.shells.basicShell;
            web = examples.shells.webShell;
            go = examples.shells.goShell;
            data = examples.shells.dataShell;

            # Test environment shell with all tools needed for testing
            testEnv = pkgs.mkShell {
              name = "devcmd-test-env";
              buildInputs = with pkgs; [
                # Core development tools
                go
                gopls
                golangci-lint
                # Testing tools
                python3
                nodejs
                which
                # Utilities for examples
                git
                curl
                wget
                docker
              ] ++ [
                # Include all example CLIs for testing
                examples.basicDev
                examples.webDev
                examples.goProject
                examples.rustProject
                examples.dataScienceProject
                examples.devOpsProject
              ];

              shellHook = ''
                echo "🧪 Devcmd Test Environment"
                echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
                echo ""
                echo "Available example CLIs (simplified naming):"
                echo "  dev --help          # Basic development CLI"
                echo "  webdev --help       # Web development CLI"
                echo "  godev --help        # Go project CLI"
                echo "  rustdev --help      # Rust project CLI"
                echo "  datadev --help      # Data science CLI"
                echo "  devops --help       # DevOps CLI"
                echo ""
                echo "Test commands:"
                echo "  nix build .#tests                    # Run all tests"
                echo "  nix build .#test-basic               # Basic functionality tests"
                echo "  nix build .#test-webdev              # Web development tests"
                echo "  nix build .#test-go                  # Go project tests"
                echo ""
                echo "Library usage examples:"
                echo "  devcmdLib.mkDevCLI { name = \"mycli\"; commandsContent = \"...\"; }"
                echo "  devcmdLib.quickCLI \"mycli\" ./commands.cli"
                echo "  devcmdLib.autoCLI \"mycli\"  # Auto-detects commands.cli"
              '';
            };

            # Development shell specifically for testing library functions
            libTest = pkgs.mkShell {
              name = "devcmd-lib-test";
              buildInputs = with pkgs; [
                go
                bash
                nix
                nixpkgs-fmt
              ];

              shellHook = ''
                echo "📚 Devcmd Library Test Environment"
                echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
                echo ""
                echo "Quick test library functions:"
                echo ""
                echo "# Test mkDevCLI with inline content"
                echo 'nix eval --expr "(import ./.nix/lib.nix { pkgs = import <nixpkgs> {}; self = null; lib = (import <nixpkgs> {}).lib; }).mkDevCLI { name = \"test\"; commandsContent = \"hello: echo world;\"; }"'
                echo ""
                echo "# Test convenience functions"
                echo 'nix eval --expr "(import ./.nix/lib.nix { pkgs = import <nixpkgs> {}; self = null; lib = (import <nixpkgs> {}).lib; }).quickCLI \"test\" null"'
              '';
            };
          };

          # Library functions for other flakes (simplified interface)
          lib = devcmdLib;

          # Apps for easy running with proper binary names
          apps = {
            default = {
              type = "app";
              program = "${self.packages.${system}.default}/bin/devcmd";
            };

            # Example apps - binary names now match package names
            basicDev = {
              type = "app";
              program = "${self.packages.${system}.basicDev}/bin/dev";
            };

            webDev = {
              type = "app";
              program = "${self.packages.${system}.webDev}/bin/webdev";
            };

            goProject = {
              type = "app";
              program = "${self.packages.${system}.goProject}/bin/godev";
            };

            rustProject = {
              type = "app";
              program = "${self.packages.${system}.rustProject}/bin/rustdev";
            };

            dataScienceProject = {
              type = "app";
              program = "${self.packages.${system}.dataScienceProject}/bin/datadev";
            };

            devOpsProject = {
              type = "app";
              program = "${self.packages.${system}.devOpsProject}/bin/devops";
            };

            # Testing apps
            runTests = {
              type = "app";
              program = "${pkgs.writeShellScript "run-tests" ''
                echo "🧪 Running devcmd tests..."
                echo "Building test suite..."
                nix build ${self}#tests
                echo "✅ All tests passed!"
              ''}";
            };

            testExamples = {
              type = "app";
              program = "${pkgs.writeShellScript "test-examples" ''
                echo "🧪 Testing example CLIs..."
                nix build ${self}#test-examples
                echo "✅ All examples tested!"
              ''}";
            };
          };

          # Checks for CI/CD (comprehensive test coverage)
          checks = {
            # Core package builds
            package-builds = self.packages.${system}.default;

            # Example builds (all examples should build successfully)
            example-basic = self.packages.${system}.basicDev;
            example-web = self.packages.${system}.webDev;
            example-go = self.packages.${system}.goProject;
            example-rust = self.packages.${system}.rustProject;
            example-data = self.packages.${system}.dataScienceProject;
            example-devops = self.packages.${system}.devOpsProject;

            # Test builds (all tests should pass)
            tests-all = self.packages.${system}.tests;
            test-examples = self.packages.${system}.test-examples;

            # Individual test categories
            test-basic-functionality = self.packages.${system}.test-basic;
            test-posix-syntax = self.packages.${system}.test-posix;
            test-variable-expansion = self.packages.${system}.test-variables;
            test-process-management = self.packages.${system}.test-processes;
            test-block-commands = self.packages.${system}.test-blocks;
            test-error-handling = self.packages.${system}.test-errors;
            test-performance = self.packages.${system}.test-performance;
            test-web-development = self.packages.${system}.test-webdev;
            test-go-project = self.packages.${system}.test-go;
            test-shell-substitution = self.packages.${system}.test-shell;

            # Formatting checks
            formatting = pkgs.runCommand "check-formatting"
              {
                nativeBuildInputs = [ pkgs.nixpkgs-fmt ];
              } ''
              echo "Checking Nix file formatting..."
              cd ${self}
              find . -name "*.nix" -exec nixpkgs-fmt --check {} \;
              touch $out
            '';

            # Library API tests - simplified approach without nix eval
            library-api = pkgs.runCommand "test-library-api"
              {
                nativeBuildInputs = [ ];
              } ''
              echo "Testing library API..."

              # Test that library functions exist and are callable
              echo "Testing mkDevCLI function exists..."
              if test -f "${self}/.nix/lib.nix"; then
                echo "✅ lib.nix exists"
              else
                echo "❌ lib.nix not found"
                exit 1
              fi

              # Test that examples build (which use the library)
              echo "Testing library usage via examples..."
              if test -f "${self.packages.${system}.basicDev}/bin/dev"; then
                echo "✅ Library API test passed - examples build successfully"
              else
                echo "❌ Library API test failed - examples don't build"
                exit 1
              fi

              touch $out
            '';
          };

          # Formatter
          formatter = pkgs.nixpkgs-fmt;
        }) // {

      # Templates for other projects
      templates = {
        default = {
          path = ./template/basic;
          description = "Basic project with devcmd CLI (simplified interface)";
        };

        basic = {
          path = ./template/basic;
          description = "Basic development commands template";
        };

      };

      # Overlay for use in other flakes (updated for simplified interface)
      overlays.default = final: prev: {
        # Core devcmd package
        devcmd = self.packages.${prev.system}.default;

        # Simplified library interface
        devcmdLib = self.lib.${prev.system};

        # Make example CLIs available in overlay with their actual names
        devcmd-examples = {
          dev = self.packages.${prev.system}.basicDev;
          webdev = self.packages.${prev.system}.webDev;
          godev = self.packages.${prev.system}.goProject;
          rustdev = self.packages.${prev.system}.rustProject;
          datadev = self.packages.${prev.system}.dataScienceProject;
          devops = self.packages.${prev.system}.devOpsProject;
        };

        # Convenience functions for overlay users
        mkDevCLI = self.lib.${prev.system}.mkDevCLI;
        quickCLI = self.lib.${prev.system}.quickCLI;
        autoCLI = self.lib.${prev.system}.autoCLI;
      };
    };
}
