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

          # Library functions
          devcmdLib = import ./.nix/lib.nix { inherit pkgs self system lib; };

          # Import all examples from examples.nix
          examples = import ./.nix/examples.nix { inherit pkgs lib self system; };

          # Import tests from tests.nix
          tests = import ./.nix/tests.nix { inherit pkgs lib self system; };

        in
        {
          # Main packages including examples and tests
          packages = {
            # Core devcmd package
            default = devcmdPackage;
            devcmd = devcmdPackage;

            # All example CLIs from examples.nix
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
            test-basic = tests.basicTests.simpleCommand or null;
            test-posix = tests.basicTests.posixSyntax or null;
            test-variables = tests.basicTests.variableExpansion or null;
            test-processes = tests.processManagementTests.watchStopCommands or null;
            test-blocks = tests.blockCommandTests.backgroundProcesses or null;
            test-errors = tests.errorHandlingTests.invalidCommands or null;
            test-performance = tests.performanceTests.largeCLI or null;
            test-webdev = tests.realWorldTests.webDevelopment or null;
            test-go = tests.realWorldTests.goProject or null;
          };

          # Development shells including example shells from examples.nix
          devShells = {
            default = import ./.nix/development.nix { inherit pkgs; };

            # Example development shells
            basic = examples.shells.basicShell;
            web = examples.shells.webShell;
            go = examples.shells.goShell;
            data = examples.shells.dataShell;

            # Test environment shell
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
                # Utilities
                git
                curl
                wget
              ] ++ [
                # Include all example CLIs for testing
                examples.basicDev
                examples.webDev
                examples.goProject
              ];

              shellHook = ''
                echo "🧪 Devcmd Test Environment"
                echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
                echo ""
                echo "Available example CLIs:"
                echo "  dev --help          # Basic development CLI"
                echo "  webdev --help       # Web development CLI"
                echo "  godev --help        # Go project CLI"
                echo ""
                echo "Run tests with: just test or just nix-test"
              '';
            };
          };

          # Library functions for other flakes
          lib = devcmdLib;

          # Apps for easy running
          apps = {
            default = {
              type = "app";
              program = "${self.packages.${system}.default}/bin/devcmd";
            };

            # Example apps
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
          };

          # Checks for CI/CD
          checks = {
            # Core package builds
            package-builds = self.packages.${system}.default;

            # Example builds
            example-basic = self.packages.${system}.basicDev;
            example-web = self.packages.${system}.webDev;
            example-go = self.packages.${system}.goProject;
            example-rust = self.packages.${system}.rustProject;
            example-data = self.packages.${system}.dataScienceProject;
            example-devops = self.packages.${system}.devOpsProject;

            # Test builds
            tests-build = self.packages.${system}.tests;
            test-examples-build = self.packages.${system}.test-examples;
          };

          # Formatter
          formatter = pkgs.nixpkgs-fmt;
        }) // {

      # Templates for other projects
      templates = {
        default = {
          path = ./template/basic;
          description = "Basic project with devcmd CLI";
        };

        basic = {
          path = ./template/basic;
          description = "Basic development commands template";
        };
      };

      # Overlay for use in other flakes
      overlays.default = final: prev: {
        devcmd = self.packages.${prev.system}.default;
        devcmdLib = self.lib.${prev.system};

        # Make example CLIs available in overlay
        devcmd-examples = {
          inherit (self.packages.${prev.system})
            basicDev webDev goProject rustProject
            dataScienceProject devOpsProject;
        };
      };
    };
}
