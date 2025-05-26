# Development environment for devcmd project
{ pkgs }:

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
    just
    git
    zsh

    # Optional: code formatting
    nixpkgs-fmt
    gofumpt
  ];

  # Environment setup
  JAVA_HOME = "${pkgs.openjdk17}/lib/openjdk";

  shellHook = ''
      echo "🔧 Devcmd Development Environment"
      echo ""
      echo "Available commands:"
      just
      echo ""
      echo "Tools: Go $(go version | cut -d' ' -f3), Just, ANTLR"
      echo ""


    # Make zsh available
    export SHELL = ${pkgs.zsh}/bin/zsh
    # Uncomment to auto-switch to zsh
    exec ${pkgs.zsh}/bin/zsh
  '';
}
