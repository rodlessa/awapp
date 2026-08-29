{
  description = "awapp — dependency-free terminal weather visualizer";
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  outputs = { self, nixpkgs }:
    let
      systems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];
      forAllSystems = nixpkgs.lib.genAttrs systems;
    in
    {
      packages = forAllSystems (system:
        let pkgs = import nixpkgs { inherit system; };
        in {
          default = pkgs.stdenv.mkDerivation {
            pname = "awapp";
            version = "1.0.5";
            src = self;
            nativeBuildInputs = [ pkgs.go ];
            buildPhase = ''
              go build -trimpath -ldflags "-s -w -X main.version=${"v1.0.5"}" -o awapp .
            '';
            installPhase = ''
              install -Dm755 awapp $out/bin/awapp
              install -Dm644 packaging/man/awapp.1 $out/share/man/man1/awapp.1
            '';
            meta = with pkgs.lib; {
              description = "Dependency-free terminal weather visualizer with ANSI animation";
              license = licenses.mit;
              platforms = platforms.unix;
            };
          };
        });
    };
}
