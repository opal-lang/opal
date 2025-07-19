# Library functions for generating CLI packages from devcmd files
{ pkgs, self, lib }:

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
  # Generate shell commands from devcmd/cli files - main library function
  mkDevCommands =
    {
      # Content sources (in order of priority)
      commandsFile ? null      # Explicit path to .cli/.devcmd file
    , commandsContent ? null   # Inline content as string
    , commands ? null          # Alias for commandsContent (backward compatibility)

      # Processing options
    , preProcess ? (text: text)    # Function to transform input before parsing
    , postProcess ? (text: text)   # Function to transform generated shell code
    , templateFile ? null          # Custom Go template file path
    , extraShellHook ? ""          # Additional shell hook content
    , debug ? false               # Enable debug output
    }:

    let
      # Try to get content from explicit file
      fileContent =
        if commandsFile != null then
          tryReadFile commandsFile
        else null;

      # Use either commandsContent or commands for inline content
      inlineContent =
        if commandsContent != null then commandsContent
        else if commands != null then commands
        else null;

      # Auto-detect commands file in standard locations
      autoDetectContent =
        let
          candidatePaths = [
            ./commands.cli # Primary default filename
            ./commands # Legacy filename
            ./devcmd.cli # Alternative naming
            ./.devcmd # Hidden file variant
          ];

          findExistingFile = paths:
            if paths == [ ] then null
            else
              let
                head = builtins.head paths;
                tail = builtins.tail paths;
              in
              if builtins.pathExists (toString head) then tryReadFile head
              else findExistingFile tail;
        in
        findExistingFile candidatePaths;

      # Determine what content to use (in order of priority)
      finalContent =
        if fileContent != null then fileContent
        else if inlineContent != null then inlineContent
        else if autoDetectContent != null then autoDetectContent
        else throw "No commands content found for CLI generation. Expected one of: commands.cli, commands, devcmd.cli, or inline content.";

      # Process the content through preProcess function
      processedContent = preProcess finalContent;

      # Generate content hash for cache-friendly naming
      contentHash = mkContentHash processedContent;

      # Use content hash in filename for better caching
      commandsSrc = pkgs.writeText "commands-content-${contentHash}" processedContent;

      # Get devcmd parser binary (automatically uses pkgs.system)
      parserBin =
        if self != null then self.packages.${pkgs.system}.default
        else throw "Self reference required for CLI generation. Use explicit commandsContent if building standalone.";

      # Handle template file path safely
      templatePath =
        if templateFile != null && builtins.pathExists (toString templateFile)
        then toString templateFile
        else null;

      # Build parser arguments
      parserArgs = lib.optionalString (templatePath != null) "--template ${templatePath}";

      # Parse the commands and generate shell functions with cache-friendly naming
      parsedShellCode = pkgs.runCommand "parsed-commands-${contentHash}"
        {
          nativeBuildInputs = [ parserBin ];
          meta.description = "Generated shell functions from devcmd";

          # Cache-friendly attributes
          preferLocalBuild = true;
          allowSubstitutes = true;
        }
        ''
          echo "Parsing commands with devcmd..."
          ${parserBin}/bin/devcmd ${parserArgs} ${commandsSrc} > $out || {
            echo "# Error parsing commands" > $out
            echo 'echo "Error: Failed to parse commands"' >> $out
          }
        '';

      # Read the generated shell code and apply postProcess
      generatedHook =
        let rawGenerated = builtins.readFile parsedShellCode;
        in postProcess rawGenerated;

      # Determine source type for logging
      sourceType =
        if fileContent != null then "from file ${toString commandsFile}"
        else if inlineContent != null then "from inline content"
        else if autoDetectContent != null then "from auto-detected commands.cli"
        else "no commands found";

      # Debug information
      debugInfo = lib.optionalString debug ''
        echo "🔍 Debug: Commands source = ${sourceType}"
        echo "🔍 Debug: Content hash = ${contentHash}"
        echo "🔍 Debug: Parser bin = ${toString parserBin}"
        echo "🔍 Debug: Template = ${if templatePath != null then toString templatePath else "none"}"
        echo "🔍 Debug: System = ${pkgs.system}"
      '';

    in
    {
      # The shellHook to inject into mkShell
      shellHook = ''
        ${debugInfo}
        echo "🚀 devcmd commands loaded ${sourceType}"
        ${generatedHook}
        ${extraShellHook}
      '';

      # Exposed metadata for debugging and introspection
      inherit commandsSrc contentHash;
      source = sourceType;
      raw = finalContent;
      processed = processedContent;
      generated = generatedHook;
      parser = parsedShellCode;
      system = pkgs.system;
    };

  # Generate a CLI package from devcmd commands (for standalone binaries)
  mkDevCLI =
    {
      # Package name (also used as binary name - follows Nix conventions)
      name

      # Content sources (same as mkDevCommands)
    , commandsFile ? null
    , commandsContent ? null
    , commands ? null

      # Processing and build options
    , preProcess ? (text: text)
    , templateFile ? null
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
        else if commands != null then commands
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

      # Template arguments
      templateArgs = lib.optionalString (templateFile != null) "--template ${toString templateFile}";

      # Generate Go source with cache-friendly naming
      goSource = pkgs.runCommand "${name}-go-source-${contentHash}"
        {
          nativeBuildInputs = [ devcmdBin pkgs.go ];

          # Cache-friendly attributes
          preferLocalBuild = true;
          allowSubstitutes = true;
        } ''
        # Go needs a writable cache dir
        export HOME=$TMPDIR
        export GOCACHE=$TMPDIR/go-build

        mkdir -p "$GOCACHE" "$out"

        echo "Generating Go CLI from commands.cli..."
        ${devcmdBin}/bin/devcmd ${templateArgs} ${commandsSrc} > "$out/main.go"

        cat > "$out/go.mod" <<EOF
        module ${name}
        go 1.21
        EOF

        echo "Validating generated Go code..."
        ${pkgs.go}/bin/go mod tidy -C "$out"
        ${pkgs.go}/bin/go build -C "$out" -o /dev/null ./...
        echo "✅ Generated Go code is valid"
      '';

    in
    pkgs.buildGoModule {
      pname = name;
      inherit version;
      src = goSource;
      vendorHash = null;

      # Environment variables using correct syntax
      env.CGO_ENABLED = "0"; # Static binary for better caching

      # Build flags following CODE_GUIDELINES.md
      ldflags = [
        "-s"
        "-w"
        "-X main.Version=${version}"
        "-X main.GeneratedBy=devcmd"
        "-X main.BuildTime=1970-01-01T00:00:00Z" # Deterministic for caching
      ];

      meta = {
        description = "Generated CLI from devcmd: ${name}";
        license = lib.licenses.mit;
        platforms = lib.platforms.unix;
        mainProgram = name;
      } // meta;
    };

  # Create a development shell with generated CLI
  mkDevShell =
    { name ? "devcmd-shell"
    , cli ? null
    , extraPackages ? [ ]
    , shellHook ? ""
    }:

    pkgs.mkShell {
      inherit name;

      buildInputs = extraPackages ++ lib.optional (cli != null) cli;

      shellHook = ''
        echo "🚀 ${name} Development Shell"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

        ${lib.optionalString (cli != null) ''
          echo ""
          echo "Generated CLI available as: ${cli.meta.mainProgram or name}"
          echo "Run '${cli.meta.mainProgram or name} --help' to see available commands"
        ''}

        ${shellHook}
        echo ""
      '';
    };

  # Simplified convenience functions for common patterns

  # Quick CLI generation with minimal config
  quickCLI = name: commandsFile: mkDevCLI {
    inherit name commandsFile;
  };

  # Quick shell hook generation
  quickCommands = commandsFile: mkDevCommands {
    inherit commandsFile;
  };

  # Auto-detect and generate from local commands.cli (default behavior)
  autoDevCommands = args: mkDevCommands ({
    # Will auto-detect commands.cli by default
  } // args);

  # Auto-detect CLI with commands.cli as default filename
  autoCLI = name: mkDevCLI {
    inherit name;
    # Auto-detection will find commands.cli by default
  };

  # Utility functions for common patterns
  utils = {
    # Common pre-processors
    preProcessors = {
      # Add common definitions to the top of commands
      addCommonDefs = defs: content:
        (lib.concatMapStringsSep "\n" (def: "var ${def.name} = \"${def.value}\"") defs) + "\n\n" + content;

      # Strip comments from commands
      stripComments = content:
        lib.concatStringsSep "\n"
          (lib.filter (line: !lib.hasPrefix "#" (lib.trim line))
            (lib.splitString "\n" content));

      # Add project-specific variables
      addProjectVars = projectName: version: content:
        ''
          # Auto-generated project variables
          var PROJECT_NAME = "${projectName}"
          var PROJECT_VERSION = "${version}"
          var BUILD_TIME = "$(date -u +%Y-%m-%dT%H:%M:%SZ)"

        '' + content;
    };

    # Common post-processors
    postProcessors = {
      # Add extra shell functions
      addHelpers = helpers: shellCode:
        shellCode + "\n" + helpers;

      # Wrap commands with timing
      addTiming = shellCode:
        lib.replaceStrings
          [ "function " ]
          [ "function timed_" ]
          shellCode;

      # Add project banner
      addBanner = projectName: shellCode:
        ''
          echo "🚀 ${projectName} Development Environment"
          echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
          echo ""
        '' + shellCode;
    };

    # System detection helpers
    isLinux = lib.hasPrefix "x86_64-linux" pkgs.system || lib.hasPrefix "aarch64-linux" pkgs.system;
    isDarwin = lib.hasPrefix "x86_64-darwin" pkgs.system || lib.hasPrefix "aarch64-darwin" pkgs.system;

    # Platform-specific command variations
    platformCmd = linuxCmd: darwinCmd:
      if utils.isLinux then linuxCmd
      else if utils.isDarwin then darwinCmd
      else linuxCmd; # default to linux

    # Content hash utility for manual cache management
    getContentHash = mkContentHash;
  };
}
