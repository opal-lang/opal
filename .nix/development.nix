# Development environment for devcmd project - interpreter mode only
{ pkgs, self ? null, gitRev ? "dev", system }:

pkgs.mkShell {
  name = "devcmd-dev";

  buildInputs = with pkgs; [
    # Development tools
    go
    gopls
    golangci-lint
    git
    zsh
    nixpkgs-fmt
    gofumpt
  ] ++ (if self != null then [
    self.packages.${system}.devcmd # Include the devcmd binary itself
  ] else []);

  shellHook = ''
    echo "🔧 Devcmd Development Environment (Interpreter Mode)"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "Available tools:"
    echo "  devcmd     - The devcmd CLI generator (interpreter mode)"
    echo "  go         - Go compiler and tools"
    echo "  gofumpt    - Go formatter"
    echo "  golangci-lint - Go linter"
    echo "  nixpkgs-fmt   - Nix formatter"
    echo ""
    echo "Development commands (run manually):"
    echo "  go fmt ./...                    - Format Go code"
    echo "  gofumpt -w .                   - Format with gofumpt"
    echo "  golangci-lint run              - Run linter"
    echo "  go test -v ./...               - Run tests"
    echo "  cd cli && go build -o devcmd . - Build CLI"
    echo ""
    echo "Interpreter mode usage:"
    echo "  ./cli/devcmd run <command> -f commands.cli"
    echo ""
  '';
}
