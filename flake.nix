{
  description = "CLI tool for interacting with MCP servers";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
  };

  outputs = { self, nixpkgs }:
    let
      supportedSystems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];
      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;
    in {
      packages = forAllSystems (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
          version = self.shortRev or self.dirtyShortRev or "dev";
        in {
          default = pkgs.buildGoModule {
            pname = "mcpli";
            version = version;
            src = ./.;
            vendorHash = "sha256-TcpuAFrIQhlGBVcQ45FiyIdoa8LXJ43nAzZzOIvu/g8=";
            ldflags = [ "-s" "-w" "-X github.com/juanibiapina/mcpli/internal/version.Version=${version}" ];
          };
        }
      );
    };
}
