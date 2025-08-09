# Development environment for devcmd project - smart derivation approach
{ pkgs, self ? null, gitRev ? "dev", system }:
let
  # Import our library to create the development CLI using fixed-output derivation
  devcmdLib = import ./lib.nix {
    inherit pkgs self gitRev system;
    lib = pkgs.lib;
  };

  # Create a shell script that generates the dev CLI on demand
  devCLI = devcmdLib.mkDevCLI {
    name = "devcmd-dev-cli";
    binaryName = "dev";
    commandsFile = ../commands.cli;
    version = "dev-${gitRev}";
  };
in
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
  ] ++ [
    self.packages.${system}.devcmd # Include the devcmd binary itself
    devCLI # Include the generated dev CLI
  ];

  shellHook = ''
    echo "🔧 Devcmd Development Environment"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    # Build dev CLI if it doesn't exist or if commands.cli is newer
    if [[ ! -f "./dev-compiled" ]] || [[ "commands.cli" -nt "./dev-compiled" ]]; then
      echo "🔨 Building dev CLI..."
      devcmd build --file commands.cli --binary dev -o ./dev-compiled
      echo "✅ dev CLI ready"
    else
      echo "✅ dev CLI ready"
    fi
    
    echo ""
    echo "Available commands:"
    echo "  devcmd - The devcmd CLI generator"
    echo "  dev    - Development commands for this project"
    echo ""
    echo "Run 'dev help' to see available development commands"
    exec ${pkgs.zsh}/bin/zsh
  '';
}
