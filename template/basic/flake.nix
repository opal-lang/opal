{
  description = "My project with devcmd CLI";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    devcmd.url = "github:aledsdavies/devcmd";
  };

  outputs = { self, nixpkgs, devcmd }:
    let
      system = "x86_64-linux"; # or your system
      pkgs = nixpkgs.legacyPackages.${system};

      # Generate CLI from commands.devcmd
      projectCLI = devcmd.lib.mkDevCLI {
        inherit pkgs system;
        name = "myproject";
        commandsFile = ./commands.devcmd;
        # Optional: add preprocessing
        # preProcess = text: "# Auto-generated\n" + text;
      };

    in
    {
      # Make the CLI available as a package
      packages.${system} = {
        default = projectCLI;
        cli = projectCLI;
      };

      # Development shell with the CLI available
      devShells.${system}.default = devcmd.lib.mkDevShell {
        inherit pkgs;
        name = "myproject-dev";
        cli = projectCLI;
        extraPackages = with pkgs; [
          # Add your development tools here
          git
          curl
          # nodejs
          # go
          # python3
        ];
        shellHook = ''
          echo "Welcome to MyProject development environment!"
          echo "Available commands: myproject --help"
        '';
      };

      # Run with: nix run
      apps.${system}.default = {
        type = "app";
        program = "${projectCLI}/bin/myproject";
      };
    };
}
