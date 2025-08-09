# Package definition for devcmd CLI (built from cli module with Go workspace support)
{ pkgs, lib, version ? "dev" }:

pkgs.buildGoModule rec {
  pname = "devcmd";
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
  vendorHash = "sha256-gs1Ls/pkkTAJ9ANKo4XU+JpyefmR0SDLnqC3Fi0emTU=";

  # Build with version info
  ldflags = [
    "-s"
    "-w"
    "-X main.Version=${version}"
    "-X main.BuildTime=1970-01-01T00:00:00Z"
  ];

  # Rename binary from 'cli' to 'devcmd'
  postInstall = ''
    mv $out/bin/cli $out/bin/devcmd
  '';

  doCheck = false; # Skip tests during build for now

  meta = with lib; {
    description = "Domain-specific language for generating development command CLIs";
    homepage = "https://github.com/aledsdavies/devcmd";
    license = licenses.mit;
    maintainers = [ maintainers.aledsdavies ];
    platforms = platforms.unix;
    mainProgram = "devcmd";
  };
}
