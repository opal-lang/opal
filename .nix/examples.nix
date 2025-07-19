# Example devcmd configurations and generated CLIs
#
# With @var() syntax, there's no conflict with shell variables:
#   - @var(NAME) = devcmd variable reference
#   - $(command) = shell command substitution (no escaping needed)
#   - $VAR = shell variable reference (no escaping needed)
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
      var SRC = "./src"
      var BUILD_DIR = "./build"

      build: {
        echo "Building project..."
        mkdir -p @var(BUILD_DIR)
        (cd @var(SRC) && make) || echo "No Makefile found"
      }

      test: {
        echo "Running tests..."
        (cd @var(SRC) && make test) || go test ./... || npm test || echo "No tests found"
      }

      clean: {
        echo "Cleaning build artifacts..."
        rm -rf @var(BUILD_DIR)
        find . -name "*.tmp" -delete
        echo "Clean complete"
      }

      lint: {
        echo "Running linters..."
        @parallel {
          (which golangci-lint && golangci-lint run) || echo "No Go linter"
          (which eslint && eslint .) || echo "No JS linter"
        }
        echo "Linting complete"
      }

      deps: {
        echo "Installing dependencies..."
        @parallel {
          (test -f go.mod && go mod download) || echo "No Go modules"
          (test -f package.json && npm install) || echo "No NPM packages"
          (test -f requirements.txt && pip install -r requirements.txt) || echo "No Python packages"
        }
        echo "Dependencies installed"
      }
    '';
  };

  # Web development with frontend/backend
  webDev = devcmdLib.mkDevCLI {
    name = "webdev";
    commandsContent = ''
      # Web development environment
      var FRONTEND_PORT = "3000"
      var BACKEND_PORT = "3001"
      var NODE_ENV = "development"

      install: {
        echo "Installing all dependencies..."
        @parallel {
          (cd frontend && npm install) || echo "No frontend"
          (cd backend && go mod download) || echo "No backend"
        }
        echo "Installation complete"
      }

      build: {
        echo "Building all components..."
        @parallel {
          (cd frontend && npm run build) || echo "No frontend build"
          (cd backend && go build -o ../dist/api ./cmd/api) || echo "No backend build"
        }
        echo "Build complete"
      }

      watch dev: {
        echo "Starting development servers..."
        echo "Frontend: http://localhost:@var(FRONTEND_PORT)"
        echo "Backend: http://localhost:@var(BACKEND_PORT)"
        @parallel {
          cd frontend && NODE_ENV=@var(NODE_ENV) npm start
          cd backend && go run ./cmd/api --port=@var(BACKEND_PORT)
        }
      }

      stop dev: {
        echo "Stopping development servers..."
        @parallel {
          pkill -f "npm start" || echo "Frontend not running"
          pkill -f "go run.*api" || echo "Backend not running"
        }
        echo "Servers stopped"
      }

      test: {
        echo "Running all tests..."
        @parallel {
          (cd frontend && npm test) || echo "No frontend tests"
          (cd backend && go test -v ./...) || echo "No backend tests"
        }
        echo "Testing complete"
      }

      format: {
        echo "Formatting code..."
        @parallel {
          (cd frontend && npm run format) || echo "No frontend formatter"
          (cd backend && go fmt ./...) || echo "No backend formatter"
        }
        echo "Formatting complete"
      }

      deploy: {
        echo "Deploying application..."
        webdev build
        echo "Building Docker image..."
        (which docker && docker build -t myapp:latest .) || echo "No Docker"
        echo "Deployment ready"
      }
    '';
  };

  # Go project with comprehensive tooling - demonstrates shell command substitution
  goProject = devcmdLib.mkDevCLI {
    name = "godev";
    commandsContent = ''
      # Go project development
      var MODULE = "github.com/example/myproject"
      var BINARY = "myproject"
      # Shell command substitution uses regular $() syntax
      var VERSION = "$(git describe --tags --always 2>/dev/null || echo 'dev')"
      var LDFLAGS = "-s -w -X main.Version=@var(VERSION) -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)"

      init: {
        echo "Initializing Go project..."
        go mod init @var(MODULE)
        echo "module @var(MODULE)" > go.mod
        echo "go 1.21" >> go.mod
        mkdir -p cmd/@var(BINARY) pkg internal
        echo "Project initialized"
      }

      deps: {
        echo "Managing dependencies..."
        go mod tidy
        go mod download
        go mod verify
        echo "Dependencies updated"
      }

      build: {
        echo "Building @var(BINARY) @var(VERSION)..."
        CGO_ENABLED=0 go build -ldflags="@var(LDFLAGS)" -o bin/@var(BINARY) ./cmd/@var(BINARY)
        echo "Binary built: bin/@var(BINARY)"
      }

      build-all: {
        echo "Building for multiple platforms..."
        @parallel {
          GOOS=linux GOARCH=amd64 go build -ldflags="@var(LDFLAGS)" -o bin/@var(BINARY)-linux-amd64 ./cmd/@var(BINARY)
          GOOS=darwin GOARCH=amd64 go build -ldflags="@var(LDFLAGS)" -o bin/@var(BINARY)-darwin-amd64 ./cmd/@var(BINARY)
          GOOS=windows GOARCH=amd64 go build -ldflags="@var(LDFLAGS)" -o bin/@var(BINARY)-windows-amd64.exe ./cmd/@var(BINARY)
        }
        echo "Multi-platform build complete"
      }

      test: {
        echo "Running comprehensive tests..."
        @parallel {
          go test -v ./...
          go test -race ./...
          go test -bench=. -benchmem ./...
        }
        echo "Testing complete"
      }

      cover: {
        echo "Generating coverage report..."
        go test -coverprofile=coverage.out ./...
        go tool cover -html=coverage.out -o coverage.html
        echo "Coverage report: coverage.html"
      }

      lint: {
        echo "Running linters..."
        @parallel {
          (which golangci-lint && golangci-lint run) || echo "golangci-lint not found"
          go fmt ./...
          go vet ./...
        }
        echo "Linting complete"
      }

      # Use "watch tests" instead of "watch test" to avoid conflict with existing "test" command
      watch tests: {
        echo "Watching for changes and running tests..."
        (which watchexec && watchexec -e go -- go test ./...) || echo "watchexec not found"
      }

      run: {
        echo "Running @var(BINARY)..."
        go run ./cmd/@var(BINARY)
      }

      debug: {
        echo "Running with debug info..."
        go run -race ./cmd/@var(BINARY) --debug
      }

      profile: {
        echo "Building with profiling..."
        go build -o bin/@var(BINARY)-profile ./cmd/@var(BINARY)
        echo "Run with: ./bin/@var(BINARY)-profile -cpuprofile=cpu.prof -memprofile=mem.prof"
      }

      release: {
        echo "Creating release @var(VERSION)..."
        godev lint
        godev test
        godev build-all
        echo "Release @var(VERSION) ready"
      }
    '';
  };

  # Rust project development
  rustProject = devcmdLib.mkDevCLI {
    name = "rustdev";
    commandsContent = ''
      # Rust project development
      var CRATE_NAME = "myproject"
      var TARGET_DIR = "./target"

      init: {
        echo "Initializing Rust project..."
        cargo init --name @var(CRATE_NAME)
        echo "Project initialized"
      }

      build: {
        echo "Building project..."
        cargo build
        echo "Build complete"
      }

      build-release: {
        echo "Building release..."
        cargo build --release
        echo "Release build complete"
      }

      test: {
        echo "Running tests..."
        @parallel {
          cargo test
          cargo test --doc
        }
        echo "Testing complete"
      }

      check: {
        echo "Checking code..."
        @parallel {
          cargo check
          cargo clippy -- -D warnings
          cargo fmt -- --check
        }
        echo "Check complete"
      }

      fix: {
        echo "Fixing code issues..."
        @parallel {
          cargo fix --allow-dirty
          cargo clippy --fix --allow-dirty
          cargo fmt
        }
        echo "Fixes applied"
      }

      run: {
        echo "Running project..."
        cargo run
      }

      # Use "watch develop" for clarity and to distinguish from other dev-related commands
      watch develop: {
        echo "Watching for changes..."
        (which cargo-watch && cargo watch -x run || echo "cargo-watch not installed")
      }

      bench: {
        echo "Running benchmarks..."
        cargo bench
        echo "Benchmarks complete"
      }

      doc: {
        echo "Building documentation..."
        cargo doc --open
        echo "Documentation built"
      }

      clean: {
        echo "Cleaning build artifacts..."
        cargo clean
        echo "Clean complete"
      }

      audit: {
        echo "Security audit..."
        (which cargo-audit && cargo audit || echo "cargo-audit not installed")
        echo "Audit complete"
      }
    '';
  };

  # Data science / Python project
  dataScienceProject = devcmdLib.mkDevCLI {
    name = "datadev";
    commandsContent = ''
      # Data science project development
      var PYTHON = "python3"
      var VENV = "./venv"
      var JUPYTER_PORT = "8888"

      setup: {
        echo "Setting up Python environment..."
        @var(PYTHON) -m venv @var(VENV)
        @var(VENV)/bin/pip install --upgrade pip
        (test -f requirements.txt && @var(VENV)/bin/pip install -r requirements.txt) || echo "No requirements.txt"
        echo "Environment setup complete"
      }

      install: {
        echo "Installing packages..."
        @var(VENV)/bin/pip install -r requirements.txt
        (test -f requirements-dev.txt && @var(VENV)/bin/pip install -r requirements-dev.txt) || echo "No dev requirements"
        echo "Installation complete"
      }

      freeze: {
        echo "Freezing requirements..."
        @var(VENV)/bin/pip freeze > requirements.txt
        echo "Requirements frozen"
      }

      watch jupyter: {
        echo "Starting Jupyter Lab on port @var(JUPYTER_PORT)..."
        @var(VENV)/bin/jupyter lab --port=@var(JUPYTER_PORT) --no-browser
      }

      stop jupyter: {
        echo "Stopping Jupyter..."
        pkill -f "jupyter" || echo "Jupyter not running"
      }

      test: {
        echo "Running tests..."
        @var(VENV)/bin/pytest -v
        echo "Testing complete"
      }

      lint: {
        echo "Linting code..."
        @parallel {
          (@var(VENV)/bin/flake8 . || echo "flake8 not installed")
          (@var(VENV)/bin/black --check . || echo "black not installed")
        }
        echo "Linting complete"
      }

      format: {
        echo "Formatting code..."
        @parallel {
          (@var(VENV)/bin/black . || echo "black not installed")
          (@var(VENV)/bin/isort . || echo "isort not installed")
        }
        echo "Formatting complete"
      }

      analyze: {
        echo "Running data analysis..."
        (@var(VENV)/bin/python scripts/analyze.py || echo "No analysis script")
        echo "Analysis complete"
      }

      clean: {
        echo "Cleaning temporary files..."
        @parallel {
          find . -name "*.pyc" -delete
          find . -name "__pycache__" -type d -exec rm -rf {} + 2>/dev/null || true
          find . -name ".pytest_cache" -type d -exec rm -rf {} + 2>/dev/null || true
        }
        echo "Clean complete"
      }
    '';
  };

  # DevOps / Infrastructure project
  devOpsProject = devcmdLib.mkDevCLI {
    name = "devops";
    commandsContent = ''
      # DevOps and infrastructure management
      var ENVIRONMENT = "development"
      var TERRAFORM_DIR = "./terraform"
      var ANSIBLE_DIR = "./ansible"
      var KUBE_NAMESPACE = "myapp-@var(ENVIRONMENT)"

      plan: {
        echo "Planning infrastructure changes..."
        (cd @var(TERRAFORM_DIR) && terraform plan -var="environment=@var(ENVIRONMENT)") || echo "No Terraform"
        echo "Plan complete"
      }

      apply: {
        echo "Applying infrastructure changes..."
        (cd @var(TERRAFORM_DIR) && terraform apply -var="environment=@var(ENVIRONMENT)" -auto-approve) || echo "No Terraform"
        echo "Apply complete"
      }

      destroy: {
        echo "Destroying infrastructure..."
        echo "WARNING: This will destroy @var(ENVIRONMENT) environment"
        (cd @var(TERRAFORM_DIR) && terraform destroy -var="environment=@var(ENVIRONMENT)" -auto-approve) || echo "No Terraform"
      }

      provision: {
        echo "Provisioning servers..."
        (cd @var(ANSIBLE_DIR) && ansible-playbook -i inventory/@var(ENVIRONMENT) site.yml) || echo "No Ansible"
        echo "Provisioning complete"
      }

      deploy: {
        echo "Deploying application to @var(ENVIRONMENT)..."
        (which kubectl && kubectl apply -f k8s/ -n @var(KUBE_NAMESPACE)) || echo "No kubectl"
        echo "Deployment complete"
      }

      status: {
        echo "Checking infrastructure status..."
        (which kubectl && kubectl get pods,svc,ing -n @var(KUBE_NAMESPACE)) || echo "No kubectl"
        echo "Status check complete"
      }

      logs: {
        echo "Fetching application logs..."
        (which kubectl && kubectl logs -f deployment/myapp -n @var(KUBE_NAMESPACE)) || echo "No kubectl"
      }

      shell: {
        echo "Opening shell in application pod..."
        (which kubectl && kubectl exec -it deployment/myapp -n @var(KUBE_NAMESPACE) -- /bin/sh) || echo "No kubectl"
      }

      backup: {
        echo "Creating backup..."
        # Shell command substitution uses regular $() syntax
        DATE=$(date +%Y%m%d-%H%M%S); echo "Backup timestamp: $DATE"
        (which kubectl && kubectl exec deployment/database -n @var(KUBE_NAMESPACE) -- pg_dump myapp > backup-$(date +%Y%m%d-%H%M%S).sql) || echo "No database"
      }

      monitor: {
        echo "Opening monitoring dashboard..."
        (which kubectl && kubectl port-forward svc/grafana 3000:3000 -n monitoring) || echo "No monitoring"
      }

      lint: {
        echo "Linting infrastructure code..."
        @parallel {
          (cd @var(TERRAFORM_DIR) && terraform fmt -check) || echo "No Terraform"
          (cd @var(ANSIBLE_DIR) && ansible-lint .) || echo "No Ansible"
        }
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
      shellHook = '';
        echo "Web development environment loaded"
        echo "Available: webdev install, webdev dev start, webdev build, webdev deploy"
      '';
    };

    # Go development shell
    goShell = devcmdLib.mkDevShell {
      name = "go-dev-shell";
      cli = goProject;
      extraPackages = with pkgs; [ go gopls golangci-lint git ];
      shellHook = '';
        echo "Go development environment loaded"
        echo "Available: godev build, godev test, godev run, godev release"
      '';
    };

    # Data science shell
    dataShell = devcmdLib.mkDevShell {
      name = "data-science-shell";
      cli = dataScienceProject;
      extraPackages = with pkgs; [ python3 python3Packages.pip git ];
      shellHook = '';
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
