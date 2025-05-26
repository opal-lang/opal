# Library functions for generating CLI packages from devcmd files
{ pkgs, self, system, lib }:

rec {
  # Generate shell commands from devcmd/cli files - main library function
  mkDevCommands =
    {
      # Required
      pkgs
    , system ? builtins.currentSystem

      # Content sources (in order of priority)
    , commandsFile ? null      # Explicit path to .cli/.devcmd file
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
            # New .cli extension (preferred)
            ./commands.cli
            ./commands
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

      # Process the content through preProcess function
      processedContent = preProcess finalContent;

      # Write processed content to store for the parser
      commandsSrc = pkgs.writeText "commands-content" processedContent;

      # Get devcmd parser binary
      parserBin = self.packages.${system}.default;

      # Handle template file path safely
      templatePath =
        if templateFile != null && builtins.pathExists templateFile
        then toString templateFile
        else null;

      # Build parser arguments
      parserArgs = lib.optionalString (templatePath != null) "--template ${templatePath}";

      # Parse the commands and generate shell functions
      parsedShellCode = pkgs.runCommand "parsed-commands"
        {
          nativeBuildInputs = [ parserBin ];
          meta.description = "Generated shell functions from devcmd";
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
        else if autoDetectContent != null then "from auto-detected file"
        else "no commands found";

      # Debug information (fixed or operator)
      debugInfo = lib.optionalString debug ''
        echo "🔍 Debug: Commands source = ${sourceType}"
        echo "🔍 Debug: Parser bin = ${toString parserBin}"
        echo "🔍 Debug: Template = ${if templatePath != null then toString templatePath else "none"}"
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
      inherit commandsSrc;
      source = sourceType;
      raw = finalContent;
      processed = processedContent;
      generated = generatedHook;
      parser = parsedShellCode;
    };

  # Generate a CLI package from devcmd commands (for standalone binaries)
  mkDevCLI =
    {
      # Package name for the generated CLI
      name ? "devcmd-cli"

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
      safeReadFile = path:
        if builtins.pathExists path
        then builtins.readFile path
        else null;

      fileContent =
        if commandsFile != null then safeReadFile commandsFile
        else null;

      inlineContent =
        if commandsContent != null then commandsContent
        else if commands != null then commands
        else null;

      autoDetectContent =
        let
          paths = [ ./commands.cli ./commands ];
          existingPath = lib.findFirst (p: builtins.pathExists p) null paths;
        in
        if existingPath != null then builtins.readFile existingPath else null;

      finalContent =
        if fileContent != null then fileContent
        else if inlineContent != null then inlineContent
        else if autoDetectContent != null then autoDetectContent
        else throw "No commands content found for CLI generation";

      processedContent = preProcess finalContent;
      commandsSrc = pkgs.writeText "${name}-commands.cli" processedContent;

      # Get devcmd binary
      devcmdBin = self.packages.${system}.default;

      # Template arguments
      templateArgs = lib.optionalString (templateFile != null) "--template ${toString templateFile}";

      # Generate Go source
      goSource = pkgs.runCommand "${name}-go-source"
        {
          nativeBuildInputs = [ devcmdBin pkgs.go ];
        } ''
            # --------------------------------------------
            # Go needs a writable cache dir; /homeless-shelter is read-only.
            export HOME=$TMPDIR                 # satisfies other tools
            export GOCACHE=$TMPDIR/go-build     # tell Go where to cache
            # --------------------------------------------

            mkdir -p "$GOCACHE" "$out"

            echo "Generating Go CLI from devcmd file..."
            ${devcmdBin}/bin/devcmd ${templateArgs} ${commandsSrc} > "$out/main.go"

            cat > "$out/go.mod" <<EOF
            module ${name}
            go 1.21
            EOF

            echo "Validating generated Go code..."
            ${pkgs.go}/bin/go mod tidy  -C "$out"
            ${pkgs.go}/bin/go build -C "$out" -o /dev/null ./...
            echo "✅ Generated Go code is valid"
      '';

    in
    pkgs.buildGoModule {
      pname = name;
      inherit version;
      src = goSource;
      vendorHash = null;

      # Build flags following CODE_GUIDELINES.md
      ldflags = [
        "-s"
        "-w"
        "-X main.Version=${version}"
        "-X main.GeneratedBy=devcmd"
        "-X main.BuildTime=1970-01-01T00:00:00Z"
      ];

      # Rename binary to match package name
      postInstall = ''
        if [ "$pname" != "$(basename $out/bin/*)" ]; then
          mv $out/bin/* $out/bin/${name}
        fi
      '';

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

  # Utility functions for common patterns
  utils = {
    # Common pre-processors
    preProcessors = {
      # Add common definitions to the top of commands
      addCommonDefs = defs: content:
        (lib.concatMapStringsSep "\n" (def: "def ${def.name} = ${def.value};") defs) + "\n\n" + content;

      # Strip comments from commands
      stripComments = content:
        lib.concatStringsSep "\n"
          (lib.filter (line: !lib.hasPrefix "#" (lib.trim line))
            (lib.splitString "\n" content));
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
    };
  };
}
