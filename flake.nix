{
  description = "Wigo – eww widget panel and wallpaper bar for Wayland";

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
    {
      homeModules.wigo = import ./nix/wigo.nix;
    }
    // flake-utils.lib.eachDefaultSystem (
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
          version = "0.1.1";
          src = ./.;

          vendorHash = "sha256-pkSrmi7XyNBfXBoQ9T/DGoTprnSml8hwr0T+yf03f2E=";

          subPackages = [ "cmd/wigo" ];

          nativeBuildInputs = [ pkgs.makeWrapper ];
          buildFlagsArray = [ "-mod=mod" ];
          postInstall = ''
            wrapProgram $out/bin/wigo \
              --prefix PATH : ${pkgs.lib.makeBinPath runtimeDeps}

            echo "Running wigo update --init post-install to setup the environment..."
            $out/bin/wigo update --init
          '';
        };
      }
    );
}
