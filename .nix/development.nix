# Development environment for Sigil project - interpreter mode only
{ pkgs, self ? null, gitRev ? "dev", system, bootstrapVersion }:

let
  sigil = pkgs.writeShellScriptBin "sigil" ''
    if [ "$#" -ge 1 ] && [ "$1" = "version" ]; then
      if [ "$#" -ge 2 ] && [ "$2" = "--json" ]; then
        printf '{"version":"${bootstrapVersion}"}\n'
      else
        printf 'sigil ${bootstrapVersion}\n'
      fi
      exit 0
    fi

    if [ "$#" -eq 0 ] || [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
      cat <<'EOF'
Bootstrap Sigil wrapper for this branch.

Use:
  sigil version
  sigil version --json
  sigil-dev <command>

Until this work is merged, `sigil-dev` is the dogfooding target and `sigil`
is the stable recovery entrypoint inside `nix develop`.
EOF
      exit 0
    fi

    printf 'sigil bootstrap wrapper: use sigil-dev for command execution on this branch\n' >&2
    exit 1
  '';

  sigilDev = pkgs.writeShellScriptBin "sigil-dev" ''
    repo_root=$(${pkgs.git}/bin/git rev-parse --show-toplevel 2>/dev/null || pwd)
    use_repo_root=1
    prev=""
    for arg in "$@"; do
      case "$prev" in
        -f|--file|--plan)
          use_repo_root=0
          ;;
      esac

      case "$arg" in
        --file=*|--plan=*|-f=*)
          use_repo_root=0
          prev=""
          continue
          ;;
        -f|--file|--plan)
          prev="$arg"
          continue
          ;;
      esac

      prev=""
    done

    if [ "$use_repo_root" -eq 1 ]; then
      cd "$repo_root"
      exec ${pkgs.go_1_25}/bin/go run -ldflags "-X main.Version=${bootstrapVersion}" "$repo_root/cli" -f "$repo_root/commands.sgl" "$@"
    fi

    exec ${pkgs.go_1_25}/bin/go run -ldflags "-X main.Version=${bootstrapVersion}" "$repo_root/cli" "$@"
  '';
in
pkgs.mkShell {
  name = "sigil-dev";

  buildInputs = with pkgs; [
    # Go toolchain (locked version)
    go_1_25
    gopls
    
    # Linting & formatting
    golangci-lint
    gofumpt
    # gci  # Disabled: broken in nixpkgs with Go 1.25 (tokeninternal incompatibility)
    
    # Testing
    gotestsum  # Better test output
    
    # Build tools
    git
    jq  # JSON processing for fuzz workflows
    
    # Nix tooling
    nixpkgs-fmt
    
    # Project-specific
    openssh  # For SSH session testing
    zsh
    sigil
    sigilDev
  ];

  shellHook = ''
    # Lock Go toolchain to prevent implicit downloads
    export GOTOOLCHAIN=local
    
    # Setup SSH for testing
    export SSH_TEST_DIR="$PWD/.ssh-test"
    mkdir -p "$SSH_TEST_DIR"
    
    # Generate SSH key if it doesn't exist
    if [ ! -f "$SSH_TEST_DIR/id_ed25519" ]; then
      ssh-keygen -t ed25519 -f "$SSH_TEST_DIR/id_ed25519" -N "" -C "sigil-test" >/dev/null 2>&1
      cat "$SSH_TEST_DIR/id_ed25519.pub" >> ~/.ssh/authorized_keys 2>/dev/null || true
      echo "✓ Generated SSH test key"
    fi
    
    # Add test key to SSH agent if running
    if [ -n "$SSH_AUTH_SOCK" ]; then
      ssh-add "$SSH_TEST_DIR/id_ed25519" 2>/dev/null || true
    fi
    
    # Check if SSH to localhost works
    if ssh -o BatchMode=yes -o ConnectTimeout=1 -o StrictHostKeyChecking=no localhost whoami >/dev/null 2>&1; then
      export SIGIL_SSH_TESTS_ENABLED=1
    fi
    
    # Only show welcome message in interactive shells
    if [ -t 0 ]; then
      echo "🔧 Sigil Development Environment"
      echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
      echo ""
       echo "Available tools:"
       echo "  sigil      - Bootstrap wrapper (stable recovery path)"
       echo "  sigil-dev  - Current branch Sigil runner"
       echo "  go         - Go compiler and tools"
       echo "  gofumpt    - Go formatter"
       echo "  golangci-lint - Go linter"
      echo "  nixpkgs-fmt   - Nix formatter"
      echo ""
      
      if [ -n "$SIGIL_SSH_TESTS_ENABLED" ]; then
        echo "✓ SSH testing enabled (localhost SSH working)"
      else
        echo "⚠ SSH testing disabled (localhost SSH not available)"
        echo "  To enable: Start SSH server and ensure key-based auth works"
      fi
      echo ""
      
       echo "Development commands (run manually):"
       echo "  sigil version                   - Show bootstrap version"
       echo "  sigil-dev info                  - Run commands.sgl with current source"
       echo "  go fmt ./...                    - Format Go code"
       echo "  gofumpt -w .                   - Format with gofumpt"
      echo "  golangci-lint run              - Run linter"
      echo "  go test -v ./...               - Run all tests (SSH if enabled)"
      echo "  go test -v -short ./...        - Run tests (skip SSH)"
      echo "  cd cli && go build -o sigil .   - Build CLI binary"
      echo ""
    fi
  '';
}
