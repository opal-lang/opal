{
  description = "Minimal project with devcmd integration";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-24.11";
    devcmd.url = "github:aledsdavies/devcmd";
  };

  outputs = { self, nixpkgs, devcmd }:
    let
      system = builtins.currentSystem;
      pkgs = nixpkgs.legacyPackages.${system};
    in
    {
      devShells.${system}.default = pkgs.mkShell {
        # Add your own buildInputs here
        buildInputs = with pkgs; [
          # Example: git
        ];

        # Set up devcmd
        shellHook = (devcmd.lib.mkDevCommands {
          inherit pkgs system;
          commandsFile = ./commands;
        }).shellHook;
      };
    };
}
