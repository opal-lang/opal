# Development environment for devcmd project
# Dogfooding our own tool for development commands
{ pkgs, self ? null }:
let
  # Import our own library to create the development CLI
  devcmdLib = import ./lib.nix {
    inherit pkgs self;
    lib = pkgs.lib;
  };
  # Generate the development CLI from our commands.cli file
  devCLI =
    if self != null then
      devcmdLib.mkDevCLI
        {
          name = "dev";
          commandsFile = ../commands.cli;
          version = "latest";
          meta = {
            description = "Devcmd development CLI - dogfooding our own tool";
            longDescription = ''
              This CLI is generated from commands.cli using devcmd itself.
              It provides a streamlined development experience with all
              necessary commands for building, testing, and maintaining devcmd.
            '';
          };
        }
    else
      null;
in
pkgs.mkShell {
  name = "devcmd-dev";
  buildInputs = with pkgs; [
    # Core Go development
    go
    gopls
    golangci-lint
    # ANTLR for grammar generation
    antlr4
    openjdk17 # Required for ANTLR
    # Development tools
    git
    zsh
    # Code formatting
    nixpkgs-fmt
    gofumpt
  ] ++ pkgs.lib.optional (devCLI != null) devCLI;
  # Environment setup
  JAVA_HOME = "${pkgs.openjdk17}/lib/openjdk";
  shellHook = ''
    echo "🔧 Devcmd Development Environment"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    ${if devCLI != null then ''
        dev help
    '' else ''
      echo "⚠️  Development CLI not available (missing self reference)"
      echo "   To get the full experience: nix develop"
      echo ""
      echo "Manual commands:"
      echo "  go build -o devcmd ./cmd/devcmd"
      echo "  go test ./..."
    ''}
    exec ${pkgs.zsh}/bin/zsh
  '';
}
