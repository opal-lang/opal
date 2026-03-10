# Package definition for Sigil CLI (built from cli module with Go workspace support)
{ pkgs, lib, version }:

pkgs.buildGoModule rec {
  pname = "sigil";
  inherit version;

  src = ./..; # repo root that contains go.work
  modRoot = "cli"; # path to CLI module's go.mod
  subPackages = [ "." ]; # build the main package

  # Critical: Disable workspace mode for all build phases
  GOWORK = "off";

  # Override vendor phase to ensure clean vendoring without workspace/replace paths
  overrideModAttrs = old: {
    GOWORK = "off";
    # Clean up go.mod to remove replace directives that would create store path references
    postPatch = ''
      # Remove replace directives that point to local paths
      sed -i '/^replace.*=> \.\./d' go.mod
    '';
  };

  # Vendor hash for CLI module dependencies
  vendorHash = "sha256-LyTou8FZs0Y8TPyVpFKh2PKpKgptgyf0DY9k6tswWCA=";

  # Build with version info
  ldflags = [
    "-s"
    "-w"
    "-X main.Version=${version}"
    "-X main.BuildTime=1970-01-01T00:00:00Z"
  ];

  # Rename binary from 'cli' to 'sigil'
  postInstall = ''
    mv $out/bin/cli $out/bin/sigil
  '';

  doCheck = false; # Skip tests during build for now

  meta = with lib; {
    description = "Sigil - The Operations Planning Language";
    homepage = "https://github.com/builtwithtofu/sigil";
    license = licenses.mit;
    maintainers = [ maintainers.aledsdavies ];
    platforms = platforms.unix;
    mainProgram = "sigil";
  };
}
