# Minimal library functions for generating CLI packages from devcmd files
{ pkgs, self, lib, gitRev }:

let
  # Simple content hash for deterministic caching
  mkContentHash = content:
    builtins.hashString "sha256" (toString content);

  # Helper function to safely read files
  tryReadFile = path:
    if builtins.pathExists (toString path) then
      builtins.readFile path
    else null;

in
rec {
  # Generate a CLI package from devcmd commands (for standalone binaries)
  mkDevCLI =
    {
      # Package name (also used as binary name - follows Nix conventions)
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
      # Use the same content resolution logic as mkDevCommands
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
            ./commands.cli # Primary default
            ./.commands.cli # Hidden
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
      contentHash = mkContentHash processedContent;

      # Cache-aware source naming
      commandsSrc = pkgs.writeText "${name}-commands-${contentHash}.cli" processedContent;

      # Get devcmd binary with better error handling
      devcmdBin =
        if self != null then self.packages.${pkgs.system}.default
        else throw "Self reference required for CLI generation. Cannot build '${name}' without devcmd parser.";

      # Simplified approach: Use devcmd build to handle everything
      cliDerivation = pkgs.writeShellScriptBin binaryName ''
        set -euo pipefail
        
        # Cache configuration
        CACHE_DIR="$HOME/.cache/devcmd/${name}"
        CONTENT_HASH="${contentHash}"
        BINARY_PATH="$CACHE_DIR/bin/${binaryName}"
        HASH_FILE="$CACHE_DIR/hash"
        
        # Check if cached binary exists and is current
        if [[ -f "$BINARY_PATH" && -f "$HASH_FILE" ]]; then
          if [[ "$(cat "$HASH_FILE" 2>/dev/null || echo "")" == "$CONTENT_HASH" ]]; then
            # Cache hit - use existing binary
            exec "$BINARY_PATH" "$@"
          fi
        fi
        
        # Cache miss - use devcmd build to generate and compile everything
        echo "🔨 Building ${name} CLI..."
        mkdir -p "$CACHE_DIR/bin"
        
        # Ensure Go toolchain is available
        if ! command -v go >/dev/null 2>&1; then
          echo "❌ Error: Go toolchain not found. Please ensure Go is installed."
          exit 1
        fi
        
        # Use devcmd build to handle generation, module management, and compilation
        ${devcmdBin}/bin/devcmd build \
          --file "${commandsSrc}" \
          --binary "${binaryName}" \
          --output "$BINARY_PATH"
        
        # Cache the result
        echo "$CONTENT_HASH" > "$HASH_FILE"
        echo "✅ ${name} CLI built successfully"
        
        # Execute the built binary
        exec "$BINARY_PATH" "$@"
      '';

    in
    cliDerivation;
}