# Example devcmd configurations and generated CLIs
#
# Note: In devcmd syntax, shell command substitution must be escaped as \$()
# since devcmd reserves $() for its own variable references:
#   - $(VAR) = devcmd variable reference
#   - \$(command) = shell command substitution (escaped)
#   - \$VAR = shell variable reference (escaped)
{ pkgs, lib, self }:

let
  devcmdLib = import ./lib.nix { inherit pkgs self lib; };

in
rec {
  # Simple development commands
  basicDev = devcmdLib.mkDevCLI {
    name = "dev";
    commandsContent = ''
      # Basic development commands
      def SRC = ./src;
      def BUILD_DIR = ./build;

      build: {
        echo "Building project...";
        mkdir -p $(BUILD_DIR);
        @sh((cd $(SRC) && make) || echo "No Makefile found")
      }

      test: {
        echo "Running tests...";
        @sh((cd $(SRC) && make test) || go test ./... || npm test || echo "No tests found")
      }

      clean: {
        echo "Cleaning build artifacts...";
        rm -rf $(BUILD_DIR);
        @sh(find . -name "*.tmp" -delete);
        echo "Clean complete"
      }

      lint: {
        echo "Running linters...";
        @parallel: {
          @sh((which golangci-lint && golangci-lint run) || echo "No Go linter");
          @sh((which eslint && eslint .) || echo "No JS linter")
        };
        echo "Linting complete"
      }

      deps: {
        echo "Installing dependencies...";
        @parallel: {
          @sh((test -f go.mod && go mod download) || echo "No Go modules");
          @sh((test -f package.json && npm install) || echo "No NPM packages");
          @sh((test -f requirements.txt && pip install -r requirements.txt) || echo "No Python packages")
        };
        echo "Dependencies installed"
      }
    '';
  };

  # Web development with frontend/backend
  webDev = devcmdLib.mkDevCLI {
    name = "webdev";
    commandsContent = ''
      # Web development environment
      def FRONTEND_PORT = 3000;
      def BACKEND_PORT = 3001;
      def NODE_ENV = development;

      install: {
        echo "Installing all dependencies...";
        @parallel: {
          @sh((cd frontend && npm install) || echo "No frontend");
          @sh((cd backend && go mod download) || echo "No backend")
        };
        echo "Installation complete"
      }

      build: {
        echo "Building all components...";
        @parallel: {
          @sh((cd frontend && npm run build) || echo "No frontend build");
          @sh((cd backend && go build -o ../dist/api ./cmd/api) || echo "No backend build")
        };
        echo "Build complete"
      }

      watch dev: {
        echo "Starting development servers...";
        echo "Frontend: http://localhost:$(FRONTEND_PORT)";
        echo "Backend: http://localhost:$(BACKEND_PORT)";
        @parallel: {
          @sh(cd frontend && NODE_ENV=$(NODE_ENV) npm start);
          @sh(cd backend && go run ./cmd/api --port=$(BACKEND_PORT))
        }
      }

      stop dev: {
        echo "Stopping development servers...";
        @parallel: {
          @sh(pkill -f "npm start" || echo "Frontend not running");
          @sh(pkill -f "go run.*api" || echo "Backend not running")
        };
        echo "Servers stopped"
      }

      test: {
        echo "Running all tests...";
        @parallel: {
          @sh((cd frontend && npm test) || echo "No frontend tests");
          @sh((cd backend && go test -v ./...) || echo "No backend tests")
        };
        echo "Testing complete"
      }

      format: {
        echo "Formatting code...";
        @parallel: {
          @sh((cd frontend && npm run format) || echo "No frontend formatter");
          @sh((cd backend && go fmt ./...) || echo "No backend formatter")
        };
        echo "Formatting complete"
      }

      deploy: {
        echo "Deploying application...";
        webdev build;
        echo "Building Docker image...";
        @sh((which docker && docker build -t myapp:latest .) || echo "No Docker");
        echo "Deployment ready"
      }
    '';
  };

  # Go project with comprehensive tooling - demonstrates shell escaping patterns
  goProject = devcmdLib.mkDevCLI {
    name = "godev";
    commandsContent = ''
      # Go project development
      def MODULE = github.com/example/myproject;
      def BINARY = myproject;
      # Shell command substitution must be escaped as \$() since devcmd uses $() for variables
      def VERSION = \$(git describe --tags --always 2>/dev/null || echo "dev");
      def LDFLAGS = -s -w -X main.Version=$(VERSION) -X main.BuildTime=\$(date -u +%Y-%m-%dT%H:%M:%SZ);

      init: {
        echo "Initializing Go project...";
        go mod init $(MODULE);
        echo "module $(MODULE)" > go.mod;
        echo "go 1.21" >> go.mod;
        mkdir -p cmd/$(BINARY) pkg internal;
        echo "Project initialized"
      }

      deps: {
        echo "Managing dependencies...";
        go mod tidy;
        go mod download;
        go mod verify;
        echo "Dependencies updated"
      }

      build: {
        echo "Building $(BINARY) $(VERSION)...";
        CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY) ./cmd/$(BINARY);
        echo "Binary built: bin/$(BINARY)"
      }

      build-all: {
        echo "Building for multiple platforms...";
        @parallel: {
          GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY)-linux-amd64 ./cmd/$(BINARY);
          GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY)-darwin-amd64 ./cmd/$(BINARY);
          GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY)-windows-amd64.exe ./cmd/$(BINARY)
        };
        echo "Multi-platform build complete"
      }

      test: {
        echo "Running comprehensive tests...";
        @parallel: {
          go test -v ./...;
          go test -race ./...;
          go test -bench=. -benchmem ./...
        };
        echo "Testing complete"
      }

      cover: {
        echo "Generating coverage report...";
        go test -coverprofile=coverage.out ./...;
        go tool cover -html=coverage.out -o coverage.html;
        echo "Coverage report: coverage.html"
      }

      lint: {
        echo "Running linters...";
        @parallel: {
          @sh((which golangci-lint && golangci-lint run) || echo "golangci-lint not found");
          go fmt ./...;
          go vet ./...
        };
        echo "Linting complete"
      }

      # Use "watch tests" instead of "watch test" to avoid conflict with existing "test" command
      watch tests: {
        echo "Watching for changes and running tests...";
        @sh((which watchexec && watchexec -e go -- go test ./...) || echo "watchexec not found")
      }

      run: {
        echo "Running $(BINARY)...";
        go run ./cmd/$(BINARY)
      }

      debug: {
        echo "Running with debug info...";
        go run -race ./cmd/$(BINARY) --debug
      }

      profile: {
        echo "Building with profiling...";
        go build -o bin/$(BINARY)-profile ./cmd/$(BINARY);
        echo "Run with: ./bin/$(BINARY)-profile -cpuprofile=cpu.prof -memprofile=mem.prof"
      }

      release: {
        echo "Creating release $(VERSION)...";
        godev lint;
        godev test;
        godev build-all;
        echo "Release $(VERSION) ready"
      }
    '';
  };

  # Rust project development
  rustProject = devcmdLib.mkDevCLI {
    name = "rustdev";
    commandsContent = ''
      # Rust project development
      def CRATE_NAME = myproject;
      def TARGET_DIR = ./target;

      init: {
        echo "Initializing Rust project...";
        cargo init --name $(CRATE_NAME);
        echo "Project initialized"
      }

      build: {
        echo "Building project...";
        cargo build;
        echo "Build complete"
      }

      build-release: {
        echo "Building release...";
        cargo build --release;
        echo "Release build complete"
      }

      test: {
        echo "Running tests...";
        @parallel: {
          cargo test;
          cargo test --doc
        };
        echo "Testing complete"
      }

      check: {
        echo "Checking code...";
        @parallel: {
          cargo check;
          cargo clippy -- -D warnings;
          cargo fmt -- --check
        };
        echo "Check complete"
      }

      fix: {
        echo "Fixing code issues...";
        @parallel: {
          cargo fix --allow-dirty;
          cargo clippy --fix --allow-dirty;
          cargo fmt
        };
        echo "Fixes applied"
      }

      run: {
        echo "Running project...";
        cargo run
      }

      # Use "watch develop" for clarity and to distinguish from other dev-related commands
      watch develop: {
        echo "Watching for changes...";
        @sh((which cargo-watch && cargo watch -x run) || echo "cargo-watch not installed")
      }

      bench: {
        echo "Running benchmarks...";
        cargo bench;
        echo "Benchmarks complete"
      }

      doc: {
        echo "Building documentation...";
        cargo doc --open;
        echo "Documentation built"
      }

      clean: {
        echo "Cleaning build artifacts...";
        cargo clean;
        echo "Clean complete"
      }

      audit: {
        echo "Security audit...";
        @sh((which cargo-audit && cargo audit) || echo "cargo-audit not installed");
        echo "Audit complete"
      }
    '';
  };

  # Data science / Python project
  dataScienceProject = devcmdLib.mkDevCLI {
    name = "datadev";
    commandsContent = ''
      # Data science project development
      def PYTHON = python3;
      def VENV = ./venv;
      def JUPYTER_PORT = 8888;

      setup: {
        echo "Setting up Python environment...";
        $(PYTHON) -m venv $(VENV);
        $(VENV)/bin/pip install --upgrade pip;
        @sh((test -f requirements.txt && $(VENV)/bin/pip install -r requirements.txt) || echo "No requirements.txt");
        echo "Environment setup complete"
      }

      install: {
        echo "Installing packages...";
        $(VENV)/bin/pip install -r requirements.txt;
        @sh((test -f requirements-dev.txt && $(VENV)/bin/pip install -r requirements-dev.txt) || echo "No dev requirements");
        echo "Installation complete"
      }

      freeze: {
        echo "Freezing requirements...";
        $(VENV)/bin/pip freeze > requirements.txt;
        echo "Requirements frozen"
      }

      watch jupyter: {
        echo "Starting Jupyter Lab on port $(JUPYTER_PORT)...";
        $(VENV)/bin/jupyter lab --port=$(JUPYTER_PORT) --no-browser
      }

      stop jupyter: {
        echo "Stopping Jupyter...";
        @sh(pkill -f "jupyter" || echo "Jupyter not running")
      }

      test: {
        echo "Running tests...";
        $(VENV)/bin/pytest -v;
        echo "Testing complete"
      }

      lint: {
        echo "Linting code...";
        @parallel: {
          @sh($(VENV)/bin/flake8 . || echo "flake8 not installed");
          @sh($(VENV)/bin/black --check . || echo "black not installed")
        };
        echo "Linting complete"
      }

      format: {
        echo "Formatting code...";
        @parallel: {
          @sh($(VENV)/bin/black . || echo "black not installed");
          @sh($(VENV)/bin/isort . || echo "isort not installed")
        };
        echo "Formatting complete"
      }

      analyze: {
        echo "Running data analysis...";
        @sh($(VENV)/bin/python scripts/analyze.py || echo "No analysis script");
        echo "Analysis complete"
      }

      clean: {
        echo "Cleaning temporary files...";
        @parallel: {
          @sh(find . -name "*.pyc" -delete);
          @sh(find . -name "__pycache__" -type d -exec rm -rf {} + 2>/dev/null || true);
          @sh(find . -name ".pytest_cache" -type d -exec rm -rf {} + 2>/dev/null || true)
        };
        echo "Clean complete"
      }
    '';
  };

  # DevOps / Infrastructure project
  devOpsProject = devcmdLib.mkDevCLI {
    name = "devops";
    commandsContent = ''
      # DevOps and infrastructure management
      def ENVIRONMENT = development;
      def TERRAFORM_DIR = ./terraform;
      def ANSIBLE_DIR = ./ansible;
      def KUBE_NAMESPACE = myapp-$(ENVIRONMENT);

      plan: {
        echo "Planning infrastructure changes...";
        @sh((cd $(TERRAFORM_DIR) && terraform plan -var="environment=$(ENVIRONMENT)") || echo "No Terraform");
        echo "Plan complete"
      }

      apply: {
        echo "Applying infrastructure changes...";
        @sh((cd $(TERRAFORM_DIR) && terraform apply -var="environment=$(ENVIRONMENT)" -auto-approve) || echo "No Terraform");
        echo "Apply complete"
      }

      destroy: {
        echo "Destroying infrastructure...";
        echo "WARNING: This will destroy $(ENVIRONMENT) environment";
        @sh((cd $(TERRAFORM_DIR) && terraform destroy -var="environment=$(ENVIRONMENT)" -auto-approve) || echo "No Terraform")
      }

      provision: {
        echo "Provisioning servers...";
        @sh((cd $(ANSIBLE_DIR) && ansible-playbook -i inventory/$(ENVIRONMENT) site.yml) || echo "No Ansible");
        echo "Provisioning complete"
      }

      deploy: {
        echo "Deploying application to $(ENVIRONMENT)...";
        @sh((which kubectl && kubectl apply -f k8s/ -n $(KUBE_NAMESPACE)) || echo "No kubectl");
        echo "Deployment complete"
      }

      status: {
        echo "Checking infrastructure status...";
        @sh((which kubectl && kubectl get pods,svc,ing -n $(KUBE_NAMESPACE)) || echo "No kubectl");
        echo "Status check complete"
      }

      logs: {
        echo "Fetching application logs...";
        @sh((which kubectl && kubectl logs -f deployment/myapp -n $(KUBE_NAMESPACE)) || echo "No kubectl")
      }

      shell: {
        echo "Opening shell in application pod...";
        @sh((which kubectl && kubectl exec -it deployment/myapp -n $(KUBE_NAMESPACE) -- /bin/sh) || echo "No kubectl")
      }

      backup: {
        echo "Creating backup...";
        # Shell command substitution requires @sh() for complex operations with variable assignment
        @sh(DATE=\$(date +%Y%m%d-%H%M%S); echo "Backup timestamp: \$DATE");
        @sh((which kubectl && kubectl exec deployment/database -n $(KUBE_NAMESPACE) -- pg_dump myapp > backup-\$(date +%Y%m%d-%H%M%S).sql) || echo "No database")
      }

      monitor: {
        echo "Opening monitoring dashboard...";
        @sh((which kubectl && kubectl port-forward svc/grafana 3000:3000 -n monitoring) || echo "No monitoring")
      }

      lint: {
        echo "Linting infrastructure code...";
        @parallel: {
          @sh((cd $(TERRAFORM_DIR) && terraform fmt -check) || echo "No Terraform");
          @sh((cd $(ANSIBLE_DIR) && ansible-lint .) || echo "No Ansible")
        };
        echo "Linting complete"
      }
    '';
  };

  # All example CLIs
  examples = {
    inherit basicDev webDev goProject rustProject dataScienceProject devOpsProject;
  };

  # Development shells with example CLIs
  shells = {
    # Basic development shell
    basicShell = devcmdLib.mkDevShell {
      name = "basic-dev-shell";
      cli = basicDev;
      extraPackages = with pkgs; [ git curl wget ];
      shellHook = ''
        echo "Basic development environment loaded"
        echo "Available: dev build, dev test, dev clean, dev lint, dev deps"
      '';
    };

    # Web development shell
    webShell = devcmdLib.mkDevShell {
      name = "web-dev-shell";
      cli = webDev;
      extraPackages = with pkgs; [ nodejs python3 go git docker ];
      shellHook = ''
        echo "Web development environment loaded"
        echo "Available: webdev install, webdev dev start, webdev build, webdev deploy"
      '';
    };

    # Go development shell
    goShell = devcmdLib.mkDevShell {
      name = "go-dev-shell";
      cli = goProject;
      extraPackages = with pkgs; [ go gopls golangci-lint git ];
      shellHook = ''
        echo "Go development environment loaded"
        echo "Available: godev build, godev test, godev run, godev release"
      '';
    };

    # Data science shell
    dataShell = devcmdLib.mkDevShell {
      name = "data-science-shell";
      cli = dataScienceProject;
      extraPackages = with pkgs; [ python3 python3Packages.pip git ];
      shellHook = ''
        echo "Data science environment loaded"
        echo "Available: datadev setup, datadev jupyter start, datadev test, datadev analyze"
      '';
    };
  };

  # Test all examples
  testExamples = pkgs.runCommand "test-all-examples"
    {
      nativeBuildInputs = with pkgs; [ bash ] ++ (builtins.attrValues examples);
    } ''
    mkdir -p $out

    echo "Testing all example CLIs..."

    # Test each CLI's help command
    ${lib.concatMapStringsSep "\n" (name: cli: ''
      echo "Testing ${name}..."
      ${cli.meta.mainProgram or name} --help > $out/${name}-help.txt
      echo "✅ ${name} help works"
    '') (lib.mapAttrsToList (name: cli: cli) examples)}

    echo "🎉 All examples tested successfully!"
    date > $out/success
  '';
}
