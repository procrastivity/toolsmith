SHELL := bash

.PHONY: lint smoke check hooks

# Shell scripts are the only executable code in Phase A; lint them all.
lint:
	shellcheck contrib/new-tool.sh contrib/check-contract contrib/check-commit-msg skeleton/contrib/check-commit-msg skeleton/contrib/check-gofumpt

# The skeleton must stay a working Go module: instantiate it into a scratch
# directory, then build, vet, and test the result. This is the same path
# `contrib/new-tool.sh` gives a real conversion, so a green smoke run means
# the next tool bootstraps green too.
smoke:
	rm -rf tmp/smoke
	contrib/new-tool.sh smoke --dir tmp/smoke --no-git
	cd tmp/smoke && CGO_ENABLED=0 go build ./... && go vet ./... && go test ./...

check: lint smoke

# Both hook types on purpose: the commit-msg hook does not install with the
# default stage, and wip shipped with exactly that gap (backport/wip.md).
hooks:
	pre-commit install --hook-type pre-commit --hook-type commit-msg
