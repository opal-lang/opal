{
  description = "Opal - The Operations Planning Language";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem
      (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
          lib = nixpkgs.lib;

          # Get git revision for generated CLIs (fallback for dirty trees)
          gitRev = self.rev or "dev-${toString self.lastModified}";

          # Main opal package
          opalPackage = import ./.nix/package.nix { inherit pkgs lib; version = "0.2.0"; };

          # Library functions with automatic system detection
          opalLib = import ./.nix/lib.nix { inherit pkgs self lib gitRev system; };


        in
        {
          packages = {
            # Core opal package
            default = opalPackage;
            opal = opalPackage;
          };

          devShells = {
            # Main development shell with generated CLI
            default = import ./.nix/development.nix { inherit pkgs self gitRev system; };
          };

          # Library functions for other flakes (simplified interface)
          lib = opalLib;

          apps = {
            default = {
              type = "app";
              program = "${self.packages.${system}.default}/bin/opal";
            };
          };

          checks = {
            # Core package builds
            package-builds = self.packages.${system}.default;

            # Formatting checks
            formatting = pkgs.runCommand "check-formatting"
              {
                nativeBuildInputs = [ pkgs.nixpkgs-fmt ];
              } ''
              echo "Checking Nix file formatting..."
              cd ${self}
              find . -name "*.nix" -exec nixpkgs-fmt --check {} \;
              touch $out
            '';
          };

          # Formatter
          formatter = pkgs.nixpkgs-fmt;
        }) // {

      # Overlay for use in other flakes
      overlays.default = final: prev: {
        # Core opal package
        opal = self.packages.${prev.system}.default;

        # Library interface
        opalLib = self.lib.${prev.system};

        # Core function for overlay users
        mkDevCLI = self.lib.${prev.system}.mkDevCLI;
      };
    };
}
