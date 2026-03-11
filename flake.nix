{
  description = "Sigil - The Operations Planning Language";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    bootstrap-sigil = {
      url = "git+file:///home/adavies/Projects/sigil?rev=1395a3bad7c8a011c06087d9d0b757dba978d934";
      inputs.nixpkgs.follows = "nixpkgs";
      inputs.flake-utils.follows = "flake-utils";
    };
  };

  outputs = { self, nixpkgs, flake-utils, bootstrap-sigil }:
    flake-utils.lib.eachDefaultSystem
      (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
          lib = nixpkgs.lib;
          version = builtins.replaceStrings ["\n"] [""] (builtins.readFile ./VERSION);
          bootstrapVersion = builtins.replaceStrings ["\n"] [""] (builtins.readFile (bootstrap-sigil + "/VERSION"));

          # Get git revision for generated CLIs (fallback for dirty trees)
          gitRev = self.rev or "dev-${toString self.lastModified}";

          # Main sigil package
          sigilPackage = import ./.nix/package.nix { inherit pkgs lib version; };

          # Stable bootstrap package for recovery workflows inside nix develop
          bootstrapSigilPackage = import ./.nix/package.nix {
            inherit pkgs lib;
            version = bootstrapVersion;
            src = bootstrap-sigil;
            vendorHashValue = "sha256-T/egqlhcIaSO6Lmhx3i9S/LxA0m1wtSs8QcYppyInP4=";
          };

          # Library functions with automatic system detection
          sigilLib = import ./.nix/lib.nix { inherit pkgs self lib gitRev system; };


        in
        {
          packages = {
            # Core sigil package
            default = sigilPackage;
            sigil = sigilPackage;
          };

          devShells = {
            # Main development shell with stable bootstrap sigil and branch build wrapper
            default = import ./.nix/development.nix {
              inherit pkgs self gitRev system;
              bootstrapSigil = bootstrapSigilPackage;
              branchSigil = sigilPackage;
            };
          };

          # Library functions for other flakes (simplified interface)
          lib = sigilLib;

          apps = {
            default = {
              type = "app";
              program = "${self.packages.${system}.default}/bin/sigil";
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
        # Core sigil package
        sigil = self.packages.${prev.system}.default;

        # Library interface
        sigilLib = self.lib.${prev.system};

        # Core function for overlay users
        mkDevCLI = self.lib.${prev.system}.mkDevCLI;
      };
    };
}
