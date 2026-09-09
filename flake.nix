{
  description = "toolsmith — the contract, playbook, and skeleton for procrastivity-style CLI tools";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.05";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
      in {
        # Phase A ships no package: toolsmith is docs + a skeleton + scripts.
        # The dev shell carries the full Go toolchain anyway, because the
        # skeleton is a compiling Go module and `make check` builds and
        # tests an instantiation of it. Phase B (the toolsmith binary)
        # adds a buildGoModule package here, shaped like skeleton/flake.nix.
        devShells.default = pkgs.mkShell {
          name = "toolsmith";
          packages = with pkgs; [
            go
            golangci-lint
            gofumpt
            git-cliff
            gnumake
            pre-commit
            shellcheck
          ];

          shellHook = ''
            echo "toolsmith dev shell — run 'make check' to lint + smoke-test the skeleton, 'make hooks' to install pre-commit."
          '';
        };
      });
}
