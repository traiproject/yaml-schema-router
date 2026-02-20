{
  description = "Development environment for GitOps";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      nixpkgs,
      flake-utils,
      ...
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs {
          inherit system;
        };
      in
      {
        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go
            gopls
            delve
            golangci-lint
            gci
            gofumpt

            ttyd
            vhs

            nixfmt-tree
            nixfmt
            nixd
          ];
        };

        formatter = pkgs.nixfmt-rfc-style;
      }
    );
}
