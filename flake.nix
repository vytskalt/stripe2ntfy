{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    gomod2nix.url = "github:nix-community/gomod2nix";
  };

  outputs = { nixpkgs, gomod2nix, ... }: let
    forAllSystems = function:
      nixpkgs.lib.genAttrs [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ] (system:
        function (import nixpkgs {
          inherit system;
          overlays = [
            (import "${gomod2nix}/overlay.nix")
          ];
        }));

    createPackage = pkgs: pkgs.buildGoApplication {
      pname = "stripe2ntfy";
      version = "0.1.0";
      pwd = ./.;
      src = ./.;
      modules = ./gomod2nix.toml;
      CGO_ENABLED = "0";
      ldflags = [ "-s" "-w" ];
      flags = [ "-trimpath" ];

      nativeBuildInputs = [ pkgs.removeReferencesTo ];
      postInstall = ''
        remove-references-to -t ${pkgs.tzdata} $out/bin/stripe2ntfy
        remove-references-to -t ${pkgs.mailcap} $out/bin/stripe2ntfy
        remove-references-to -t ${pkgs.iana-etc} $out/bin/stripe2ntfy
      '';
    };

    createImage = suffix: package: pkgs: pkgs.dockerTools.buildLayeredImage {
      name = "ghcr.io/vytskalt/stripe2ntfy${suffix}";
      tag = package.version;
      config = {
        Cmd = [ "${package}/bin/stripe2ntfy" ];
        Env = [
          "GIT_SSL_CAINFO=${pkgs.cacert}/etc/ssl/certs/ca-bundle.crt"
          "SSL_CERT_FILE=${pkgs.cacert}/etc/ssl/certs/ca-bundle.crt"
        ];
        ExposedPorts = {
          "8000/tcp" = {};
        };
      };
    };
  in {
    packages = forAllSystems (pkgs: let
      pkgsAarch64 = pkgs.pkgsCross.aarch64-multiplatform;
    in rec {
      package = createPackage pkgs;
      image = createImage "" package pkgs;

      package-aarch64 = createPackage pkgsAarch64;
      image-aarch64 = createImage "-aarch64" package-aarch64 pkgsAarch64;
    });

    devShells = forAllSystems (pkgs: {
      default = pkgs.mkShell {
        buildInputs = [
          pkgs.go
          pkgs.gomod2nix
        ];
      };
    });
  };
}
