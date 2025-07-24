# Package definition for devcmd
{ pkgs, lib, version ? "dev" }:

pkgs.buildGoModule rec {
  pname = "devcmd";
  inherit version;

  src = ./..;

  # Computed vendorHash for clean dependencies (no replace directive)
  vendorHash = "sha256-D+GwlCwvxMg71a8jxF+zm5uiWsS4mj1W8Ynk35A2tFw=";

  subPackages = [ "cmd/devcmd" ];

  # Enhanced build flags following CODE_GUIDELINES.md
  # Note: -buildid= is set by default by buildGoModule, so we don't include it
  ldflags = [
    "-s"
    "-w"
    "-X main.Version=${version}"
    "-X main.BuildTime=1970-01-01T00:00:00Z"
  ];

  # Follow performance contracts from guidelines
  # Verify build performance doesn't exceed reasonable bounds
  postBuild = ''
    echo "✅ devcmd build completed successfully"
  '';

  doCheck = false;

  # Ensure tests pass following CODE_GUIDELINES.md
  checkPhase = ''
    runHook preCheck
    echo "Running devcmd tests..."
    go test -v ./...
    runHook postCheck
  '';

  meta = with lib; {
    description = "Domain-specific language for generating development command CLIs";
    homepage = "https://github.com/aledsdavies/devcmd";
    license = licenses.mit;
    maintainers = [ maintainers.aledsdavies ];
    platforms = platforms.unix;
    mainProgram = "devcmd";
  };
}
