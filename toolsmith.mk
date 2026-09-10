# toolsmith.mk — the make target that only toolsmith needs.
# drift/drift_test.go pins Makefile as a pure substitution of
# assets/_skeleton/Makefile; anything toolsmith-specific lives here
# instead, pulled in by Makefile's `include toolsmith.mk`.

.PHONY: smoke

# The skeleton must stay a working Go module: instantiate it into a scratch
# directory with the `new` verb, then build, vet, and test the result. A
# green smoke run means the next tool bootstraps green too.
#
# The skeleton instantiated here must be the one this working tree embeds,
# so the verb runs with XDG_CONFIG_HOME pointed at an empty directory: a
# $XDG_CONFIG_HOME/toolsmith override on this host would otherwise shadow
# it (C5.1). The variable is scoped to the verb alone, never to a `go`
# command, because go reads its own `go env -w` settings from there. The
# chain's shipped-default link resolves to ./share/toolsmith/assets beside
# bin/, which this repo does not have.
smoke: build
	rm -rf tmp/smoke tmp/smoke-xdg-config
	mkdir -p tmp/smoke-xdg-config
	XDG_CONFIG_HOME=$(CURDIR)/tmp/smoke-xdg-config bin/toolsmith new smoke --dir tmp/smoke --no-git
	cd tmp/smoke && CGO_ENABLED=0 go build ./... && go vet ./... && go test ./...

check: smoke
