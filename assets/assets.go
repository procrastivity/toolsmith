// Package assets embeds the shipped default asset tree as the last-resort
// fallback the asset-resolution chain reaches for when neither a user
// override ($XDG_CONFIG_HOME/toolsmith/) nor an installed share tree
// (<prefix>/share/toolsmith/assets/) has the requested file — e.g. a bare
// `go run` with nothing installed. Assets are inputs, never state (C5.3):
// this package only ever reads the tree baked in at build time.
package assets

import "embed"

// FS holds every shipped asset file: the chassis skeleton, the migration
// playbook, and the handoff-kit sidecar templates, alongside toolsmith's
// own agent-guidance, config default, and claude-code judgment template.
//
// Every pattern below carries the "all:" prefix, not just the ones naming
// a tree: a bare `//go:embed X` silently omits dotfiles and
// dot-directories, and the skeleton alone carries eight such paths
// (.envrc, .gitignore, .golangci.yml, .pre-commit-config.yaml, and
// .github/ with both of its workflows) — without "all:" those would build
// clean and ship broken.
//
// _skeleton, never skeleton: a directory containing a file named exactly
// go.mod cannot be embedded ("cannot embed directory X: in different
// module" — the "all:" prefix does not help). Renaming go.mod away is not
// sufficient by itself either: once it's gone, the tree's .go files
// become ordinary packages of *this* module, and `go build ./...`,
// `go vet ./...`, and `go test ./...` break on 23 unresolved imports the
// skeleton was never meant to resolve here. The "_" prefix is what fixes
// that half — the go tool skips underscore-prefixed directories when
// expanding "./..." — and go.mod.tmpl / go.sum.tmpl (renamed back to
// go.mod / go.sum by contrib/new-tool.sh as it writes a tree out) is what
// fixes the embed half. Do not "fix" either one by reverting the name or
// the suffix; a future reader who does will reintroduce both failures.
//
//go:embed all:_skeleton
//go:embed all:playbook
//go:embed all:handoff-kit
//go:embed all:templates
//go:embed all:agent-guidance.md
//go:embed all:config.default.yaml
var FS embed.FS
