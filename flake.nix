{
  description = "devcmd - Simple shell command DSL for Nix development environments";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs";
  };

  outputs = { self, nixpkgs, ... }:
    let
      # Standard library and helpers
      lib = nixpkgs.lib;
      systems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];
      forAllSystems = f: lib.genAttrs systems f;
      pkgsFor = system: import nixpkgs { inherit system; };
      version = "0.1.0";
    in
    {
      # Go parser binary package
      packages = forAllSystems (system:
        let pkgs = pkgsFor system;
        in {
          default = pkgs.buildGoModule {
            pname = "devcmd-parser";
            inherit version;
            src = ./.;
            vendorHash = null; # Skip vendoring for projects with no external dependencies
            subPackages = [ "cmd/devcmd-parser" ];
          };
        }
      );

      # Export the library functions
      lib = {
        mkDevCommands =
          { pkgs
          , system ? builtins.currentSystem
          , commandsFile ? null
          , commandsContent ? null
          , commands ? null  # Alias for commandsContent for backward compatibility
          , preProcess ? (text: text)
          , postProcess ? (text: text)
          , extraShellHook ? ""
          , templateFile ? null
          , debug ? false
          }:
          let
            # Helper function to read a file safely at evaluation time
            safeReadFile = path:
              if builtins.pathExists path
              then builtins.readFile path
              else null;

            # Get content from commandsFile if provided
            fileContent =
              if commandsFile != null
              then safeReadFile commandsFile
              else null;

            # Use either commandsContent or commands for inline content
            inlineContent =
              if commandsContent != null then commandsContent
              else if commands != null then commands
              else null;

            # Try to find a commands file in common locations
            autoDetectContent =
              let
                # Try to detect commands file in various common locations
                paths = [
                  # Absolute paths derived from caller's environment
                  "${builtins.toString ./.}/commands"
                  "${builtins.toString ./.}/commands.txt"
                  "${builtins.toString ./.}/commands.devcmd"
                  # Look in cwd-relative paths
                  ./commands
                  ./commands.txt
                  ./commands.devcmd
                ];
                existingPath = lib.findFirst (p: builtins.pathExists p) null paths;
              in
              if existingPath != null
              then builtins.readFile existingPath
              else null;

            # Determine what content to use (in order of priority)
            finalContent =
              if fileContent != null then fileContent
              else if inlineContent != null then inlineContent
              else if autoDetectContent != null then autoDetectContent
              else "# No commands defined";

            # Temporary file for commands content
            commandsSrc = pkgs.writeText "commands-content" finalContent;

            # Process text
            processedPath = pkgs.writeText "processed-commands"
              (preProcess finalContent);

            # Parse the commands
            parserBin = self.packages.${system}.default;

            # Safely handle template file paths
            templatePath =
              if templateFile != null && builtins.pathExists templateFile
              then toString templateFile
              else null;

            parserArgs =
              if templatePath != null
              then "--template ${templatePath}"
              else "";

            parsed =
              pkgs.runCommand "parsed-commands"
                { nativeBuildInputs = [ parserBin ]; }
                ''
                  ${parserBin}/bin/devcmd-parser ${parserArgs} ${processedPath} > $out || echo "" > $out
                '';

            # Generate shell code with appropriate messages
            generatedHook = postProcess (builtins.readFile parsed);

            # Determine source type for logging
            sourceType =
              if fileContent != null then "from file ${toString commandsFile}"
              else if inlineContent != null then "from inline content"
              else if autoDetectContent != null then "from auto-detected file"
              else "no commands found";

            # Debugging information
            debugInfo =
              if debug then ''
                echo "Debug: Commands source = ${sourceType}"
                echo "Debug: Current directory = ${builtins.toString ./.}"
                echo "Debug: Parser bin = ${toString parserBin}"
              '' else "";
          in
          {
            # The shellHook to inject into mkShell
            shellHook = ''
              ${debugInfo}
              echo "devcmd commands ${sourceType}"
              ${generatedHook}
              ${extraShellHook}
            '';

            # Exposed metadata for debugging
            inherit commandsSrc processedPath parsed;
            source = sourceType;
            raw = finalContent;
            generated = generatedHook;
          };
      };

      # Development shell for the project itself
      devShells = forAllSystems (system:
        let pkgs = pkgsFor system;
        in {
          default = pkgs.mkShell {
            buildInputs = with pkgs; [ go gopls go-tools ];
            shellHook = ''
              echo "devcmd development shell"
              echo "Build the parser with: go run ./cmd/devcmd-parser --help"
            '';
          };
        }
      );

      # Project template
      templates.default = {
        path = ./template;
        description = "Minimal project with devcmd integration";
      };
    };
}
