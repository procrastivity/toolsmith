{
  description = "ste9 — dev shell and package for the ste9 Go CLI";

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
        # release tag: `nix build` reports `171ee47`, `make build` at a tag
        # reports `v0.1.0`. Both name the same commit, and for
        # `nix build github:simensen/ste9/v0.1.0` the rev is the tag
        # resolved — a stricter identifier, not a looser one. Do not "fix"
        # this with a VERSION file: the tag is the single source of truth,
        # and a second copy would go stale in silence.
        version = self.shortRev or self.dirtyShortRev or "dev";

        ste9 = pkgs.buildGoModule {
          pname = "ste9";
          inherit version;
          src = ./.;
          vendorHash = "sha256-PJHasb5gVNWoqbwOh9/RCkzolUALakqFKBzqvGEOYQw=";

          env.CGO_ENABLED = 0;

          ldflags = [
            "-X main.version=${version}"
            "-X main.commit=${self.rev or self.dirtyRev or "unknown"}"
            # A fixed epoch, not an oversight: a real build date would make
            # the derivation unreproducible. `commit` above carries the
            # provenance the date would otherwise supply.
            "-X main.date=1970-01-01T00:00:00Z"
          ];

          subPackages = [ "cmd/ste9" ];

          postInstall = ''
            mkdir -p $out/share/ste9
            cp -r assets $out/share/ste9/assets
            # The .go files under assets/ only exist so assets/ (and
            # assets/spec/) can embed themselves as the binary's
            # last-resort fallback; they are source, not shipped assets,
            # and must not appear in the installed share tree.
            find $out/share/ste9/assets -name '*.go' -delete
          '';

          meta = {
            description = "ste9 — ASD-STE100 Issue 9 skills, register, and linter for coding-agent harnesses";
            license = pkgs.lib.licenses.mit;
            mainProgram = "ste9";
          };
        };
      in {
        packages.default = ste9;

        devShells.default = pkgs.mkShell {
          name = "ste9";
          packages = with pkgs; [
            go
            golangci-lint
            gofumpt
            git-cliff
            gnumake
            pre-commit
            # ste_lint.py is still the parity oracle (M2 retires it): both
            # internal/lint/lint_probe_test.go and scripts/lint_parity.sh
            # shell out to `python3` to run it.
            python3
          ];

          shellHook = ''
            echo "ste9 dev shell — run 'make check' to lint+test, 'make hooks' to install pre-commit."
          '';
        };
      });
}
