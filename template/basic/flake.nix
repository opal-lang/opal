{
  description = "My project with devcmd CLI";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    devcmd.url = "github:aledsdavies/devcmd";
  };

  outputs = { self, nixpkgs, devcmd }:
    let
      system = "x86_64-linux"; # Change to your system
      pkgs = nixpkgs.legacyPackages.${system};

      # Generate CLI from commands.cli
      projectCLI = devcmd.lib.mkDevCLI {
        name = "myproject";
        commandsFile = ./commands.cli;
      };

    in
    {
      # Make the CLI available as a package
      packages.${system}.default = projectCLI;

      # Development shell with the CLI available
      devShells.${system}.default = pkgs.mkShell {
        buildInputs = [ projectCLI ];
        shellHook = ''
          echo "🚀 Welcome to MyProject!"
          echo "Try: myproject --help"
        '';
      };

      # Run with: nix run
      apps.${system}.default = {
        type = "app";
        program = "${projectCLI}/bin/myproject";
      };
    };
}
