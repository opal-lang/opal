# Package definition for Sigil CLI (built from cli module with Go workspace support)
{ pkgs, lib, version, src ? ./.., vendorHashValue ? "sha256-qEvCXjTeYWf1Brh/j5WHse3QMC1iLJ1lDeiS7PrpOSU=" }:

pkgs.buildGoModule rec {
  pname = "sigil";
  inherit version;

  inherit src; # repo root that contains go.work
  modRoot = "cli"; # path to CLI module's go.mod
  subPackages = [ "." ]; # build the main package

  # Critical: Disable workspace mode for all build phases
  GOWORK = "off";

  # Build module graph with workspace mode disabled while preserving local replace directives.
  # The pinned flake source contains `core/` and `runtime/`, so the relative module links stay valid.
  overrideModAttrs = old: {
    GOWORK = "off";
  };

  # Vendor hash for CLI module dependencies
  vendorHash = vendorHashValue;

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
