{
  description = "Wigo: eww and go widget panel and wallpaper bar for Wayland";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs { inherit system; };

        runtimeDeps = with pkgs; [
          pulseaudio
          cliphist
          wl-clipboard
          eww
          zenity
        ];
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "wigo";
          version = "0.1.0";
          src = ./.;

          vendorHash = "sha256-W1liTEluyPaW6K3JGIf3MSUmPn3BWumzAW1uQfBcVMQ=";

          subPackages = [ "cmd/wigo" ];

          nativeBuildInputs = [ pkgs.makeWrapper ];

          postInstall = ''
            wrapProgram $out/bin/wigo \
              --prefix PATH : ${pkgs.lib.makeBinPath runtimeDeps}
          '';
        };

        devShells.default = pkgs.mkShell {
          buildInputs =
            with pkgs;
            [
              go
            ]
            ++ runtimeDeps;

          shellHook = ''
            echo "--- Wigo Development Environment ---"
            go build -o wigo ./cmd/wigo
            export PATH="$PWD:$PATH"
            echo "Ready! 'wigo' binary has been built and added to your temporary PATH."
          '';
        };
      }
    );
}
