# Development environment for Opal project - interpreter mode only
{ pkgs, self ? null, gitRev ? "dev", system }:

pkgs.mkShell {
  name = "opal-dev";

  buildInputs = with pkgs; [
    # Development tools
    go
    gopls
    golangci-lint
    git
    zsh
    nixpkgs-fmt
    gofumpt
    openssh  # For SSH session testing
  ] ++ (if self != null then [
    self.packages.${system}.opal # Include the opal binary itself
  ] else []);

  shellHook = ''
    # Setup SSH for testing
    export SSH_TEST_DIR="$PWD/.ssh-test"
    mkdir -p "$SSH_TEST_DIR"
    
    # Generate SSH key if it doesn't exist
    if [ ! -f "$SSH_TEST_DIR/id_ed25519" ]; then
      ssh-keygen -t ed25519 -f "$SSH_TEST_DIR/id_ed25519" -N "" -C "opal-test" >/dev/null 2>&1
      cat "$SSH_TEST_DIR/id_ed25519.pub" >> ~/.ssh/authorized_keys 2>/dev/null || true
      echo "✓ Generated SSH test key"
    fi
    
    # Add test key to SSH agent if running
    if [ -n "$SSH_AUTH_SOCK" ]; then
      ssh-add "$SSH_TEST_DIR/id_ed25519" 2>/dev/null || true
    fi
    
    # Check if SSH to localhost works
    if ssh -o BatchMode=yes -o ConnectTimeout=1 -o StrictHostKeyChecking=no localhost whoami >/dev/null 2>&1; then
      export OPAL_SSH_TESTS_ENABLED=1
    fi
    
    # Only show welcome message in interactive shells
    if [ -t 0 ]; then
      echo "🔧 Opal Development Environment"
      echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
      echo ""
      echo "Available tools:"
      echo "  opal       - The Opal CLI (operations planning language)"
      echo "  go         - Go compiler and tools"
      echo "  gofumpt    - Go formatter"
      echo "  golangci-lint - Go linter"
      echo "  nixpkgs-fmt   - Nix formatter"
      echo ""
      
      if [ -n "$OPAL_SSH_TESTS_ENABLED" ]; then
        echo "✓ SSH testing enabled (localhost SSH working)"
      else
        echo "⚠ SSH testing disabled (localhost SSH not available)"
        echo "  To enable: Start SSH server and ensure key-based auth works"
      fi
      echo ""
      
      echo "Development commands (run manually):"
      echo "  go fmt ./...                    - Format Go code"
      echo "  gofumpt -w .                   - Format with gofumpt"
      echo "  golangci-lint run              - Run linter"
      echo "  go test -v ./...               - Run all tests (SSH if enabled)"
      echo "  go test -v -short ./...        - Run tests (skip SSH)"
      echo "  cd cli && go build -o opal .   - Build CLI"
      echo ""
      echo "Opal usage:"
      echo "  opal deploy --dry-run          - Show execution plan"
      echo "  opal deploy                    - Execute operation"
      echo ""
    fi
  '';
}
