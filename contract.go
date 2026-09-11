// Package toolsmith embeds this repository's own CONTRACT.md so the
// compiled binary can serve it back. That is the whole reason it exists:
// `toolsmith check` audits a repo against the contract, but a session that
// has only the installed binary — no clone of this repository — had no way
// to read the document being audited against, since no verb printed it and
// it was never shipped as an asset. `toolsmith doc CONTRACT.md`
// (internal/verbs/doc) closes that gap.
//
// The contract is not tunable behavior (C5.1): it never goes through
// internal/asset's override -> default -> embedded resolution chain, and
// it is not listed among the manifest's assets (C3.3) — that list is for
// the shipped assets/ tree, and the contract is neither shipped as a file
// under assets/ nor a default a user is meant to override. It is served
// from exactly this one compiled-in copy, always.
//
// Rejected alternatives: moving CONTRACT.md into assets/ (on the order of
// fifty references across this repository and its docs name it at the
// repository root; relocating it would be its own migration, not a step
// of this one). A second copy living under assets/ behind a drift test
// (the kit's rule is one copy of the contract, not two a test keeps in
// sync).
package toolsmith

import _ "embed"

//go:embed CONTRACT.md
var contractMD []byte

// Contract returns this repository's CONTRACT.md exactly as compiled into
// the binary.
func Contract() []byte {
	return contractMD
}
