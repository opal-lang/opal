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
  ] ++ (if self != null then [
    self.packages.${system}.opal # Include the opal binary itself
  ] else []);

  shellHook = ''
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
    echo "Development commands (run manually):"
    echo "  go fmt ./...                    - Format Go code"
    echo "  gofumpt -w .                   - Format with gofumpt"
    echo "  golangci-lint run              - Run linter"
    echo "  go test -v ./...               - Run tests"
    echo "  cd cli && go build -o opal .   - Build CLI"
    echo ""
    echo "Opal usage:"
    echo "  opal deploy --dry-run          - Show execution plan"
    echo "  opal deploy                    - Execute operation"
    echo ""
  '';
}
