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

        # A flake sees rev/shortRev/dirtyRev and lastModified — never tags.
        # So the Nix path stamps the commit where the make path stamps the
        # release tag (C1.7): `nix build` reports `abc1234`, `make build`
        # at a tag reports `v0.1.0`. Both name the same commit, and for
        # `nix build github:OWNER/toolsmith/v0.1.0` the rev is the tag
        # resolved — a stricter identifier, not a looser one. Do not "fix"
        # this with a VERSION file: the tag is the single source of truth,
        # and a second copy would go stale in silence.
        version = self.shortRev or self.dirtyShortRev or "dev";

        toolsmith = pkgs.buildGoModule {
          pname = "toolsmith";
          inherit version;
          src = ./.;
          # The first `nix build` fails and prints the real hash — paste it
          # here. Re-do this whenever go.mod changes.
          vendorHash = "sha256-komX1AmHt2NoF1x6xsNa2RFkfVzOXfYEMPhT0zwMxjw=";

          env.CGO_ENABLED = 0;

          ldflags = [
            "-X main.version=${version}"
            "-X main.commit=${self.rev or self.dirtyRev or "unknown"}"
            # A fixed epoch, not an oversight: a real build date would make
            # the derivation unreproducible. `commit` above carries the
            # provenance the date would otherwise supply.
            "-X main.date=1970-01-01T00:00:00Z"
          ];

          subPackages = [ "cmd/toolsmith" ];

          postInstall = ''
            mkdir -p $out/share/toolsmith
            cp -r assets $out/share/toolsmith/assets
            # assets.go only exists so `assets/` can embed itself as the
            # binary's last-resort fallback; it is source, not a shipped
            # asset, and must not appear in the installed share tree (C1.6).
            rm -f $out/share/toolsmith/assets/assets.go
          '';

          meta = {
            description = "toolsmith — the conventions repo for procrastivity-style tooling: it instantiates the chassis, audits a tool against the contract, and carries the migration playbook";
            license = pkgs.lib.licenses.mit;
            mainProgram = "toolsmith";
          };
        };
      in {
        packages.default = toolsmith;

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

          # stderr, not stdout: `nix develop --command toolsmith manifest
          # --json | jq` has to work, and anything this hook prints to
          # stdout lands in front of the document (C2.1 in spirit — the
          # tool owns its stdout, and so must its environment).
          shellHook = ''
            echo "toolsmith dev shell — run 'make check' to lint, test, and smoke-test the skeleton, 'make hooks' to install pre-commit." >&2
          '';
        };
      });
}
