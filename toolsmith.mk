# toolsmith.mk — the make targets and shellcheck inputs that only
# toolsmith needs. drift/drift_test.go pins Makefile as a pure
# substitution of assets/_skeleton/Makefile; anything toolsmith-specific
# lives here instead, pulled in by Makefile's `include toolsmith.mk`.

# toolsmith's own shell scripts. The skeleton's own assets/_skeleton/.envrc
# and assets/_skeleton/contrib/* copies are left out: the drift gate pins
# them byte-identical to the root copies Makefile's SHELLCHECK_FILES already
# lists.
SHELLCHECK_FILES += contrib/new-tool.sh contrib/check-contract contrib/parity-check

.PHONY: smoke parity

# The skeleton must stay a working Go module: instantiate it into a scratch
# directory, then build, vet, and test the result. This is the same path
# `contrib/new-tool.sh` gives a real conversion, so a green smoke run means
# the next tool bootstraps green too.
smoke:
	rm -rf tmp/smoke
	contrib/new-tool.sh smoke --dir tmp/smoke --no-git
	cd tmp/smoke && CGO_ENABLED=0 go build ./... && go vet ./... && go test ./...

check: smoke

# The parity gate compares `toolsmith check` against its oracle,
# contrib/check-contract, over a pinned corpus of real repos plus
# generated probes (assets/playbook/parity-gate.md, docs/binary/port-spec.md
# §10). The corpus lives on this host only — two of the five repos sit on
# branches other than the one checked out — so this is deliberately not a
# dependency of `check` and never runs in CI. `build` is a prerequisite
# because the gate audits bin/toolsmith directly; the script itself
# regenerates tmp/smoke (`make smoke`) before using it.
parity: build
	contrib/parity-check
