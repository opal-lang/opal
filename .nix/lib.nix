# Library functions for generating CLI packages using shell scripts
{ pkgs, self, lib, gitRev, system }:

let
  # Helper function to safely read files
  tryReadFile = path:
    if builtins.pathExists (toString path) then
      builtins.readFile path
    else null;

in
rec {
  # Generate a pre-compiled CLI binary
  mkDevCLI =
    {
      # Package name 
      name

      # Binary name (defaults to "dev" if not specified)
    , binaryName ? "dev"

      # Content sources
    , commandsFile ? null
    , commandsContent ? null

      # Processing and build options
    , preProcess ? (text: text)
    , version ? "generated"
    , meta ? { }
    }:

    let
      # Content resolution logic
      fileContent =
        if commandsFile != null then tryReadFile commandsFile
        else null;

      inlineContent =
        if commandsContent != null then commandsContent
        else null;

      # Auto-detect with commands.cli as default
      autoDetectContent =
        let
          candidates = [
            ../commands.cli # Look in parent directory (project root)
            ./commands.cli # Look in current directory
            ./.commands.cli # Hidden variant
          ];

          findFirst = paths:
            if paths == [ ] then null
            else
              let candidate = builtins.head paths;
              in
              if builtins.pathExists (toString candidate) then tryReadFile candidate
              else findFirst (builtins.tail paths);
        in
        findFirst candidates;

      finalContent =
        if fileContent != null then fileContent
        else if inlineContent != null then inlineContent
        else if autoDetectContent != null then autoDetectContent
        else throw "No commands content found for CLI '${name}'. Expected commands.cli file or explicit content.";

      processedContent = preProcess finalContent;

      # Get devcmd binary
      devcmdBin =
        if self != null then self.packages.${system}.devcmd or self.packages.${system}.default
        else throw "Self reference required for CLI generation. Cannot build '${name}' without devcmd parser.";

      # Create a shell script that checks for existing binary first
      cliScript = pkgs.writeShellScriptBin binaryName ''
        #!/usr/bin/env bash
        set -euo pipefail
        
        # Check if compiled binary exists
        BINARY_PATH="./${binaryName}-compiled"
        if [[ -f "$BINARY_PATH" ]]; then
          exec "$BINARY_PATH" "$@"
        fi
        
        echo "❌ ${binaryName} binary not found. Please run 'nix develop' to rebuild."
        exit 1
      '';

    in
    cliScript;
}
