package harness

import (
	"fmt"

	"github.com/procrastivity/toolsmith/internal/toolsmitherr"
)

// Refusal codes, one per refusing state (C4.5, C4.6). A single shared
// refusal code used to cover UnownedConflict and Modified together, with
// only the message telling them apart; each refusing state now carries
// its own code so a caller scripting against codes, not message text, can
// tell them apart too.
const (
	CodeUnownedConflict = "refusal.unowned-harness-target"
	CodeModified        = "refusal.modified-harness-target"
	CodeIncompatible    = "refusal.incompatible-harness-target"
)

// Refusal reports, as a refusal error, whether s — dir's Status — refuses
// an install or an uninstall (C4.6, C4.7). UnownedConflict, Modified and
// Incompatible each refuse with their own code and a message naming that
// state's specific risk; every other state (Current, Missing, Stale) is
// not a refusal, and Refusal returns nil so the caller proceeds.
//
// remedy is the caller's closing sentence, appended verbatim: install
// names `--force` and what it would do to dir's content for this state
// (overwrite it, destroy it, or replace its stamp); uninstall, which has
// no --force, names removing the tree by hand instead. Refusal itself
// only ever states the risk — what is true about dir that makes s a
// refusal — since that fact does not depend on which verb is asking.
func Refusal(harnessName, dir string, s State, remedy string) error {
	switch s {
	case UnownedConflict:
		return toolsmitherr.New(CodeUnownedConflict,
			fmt.Sprintf("refused — %s holds content with no install stamp; it was not written by `toolsmith install %s` — %s", dir, harnessName, remedy))
	case Modified:
		return toolsmitherr.New(CodeModified,
			fmt.Sprintf("refused — %s no longer matches what `toolsmith install %s` last wrote — %s", dir, harnessName, remedy))
	case Incompatible:
		return toolsmitherr.New(CodeIncompatible,
			fmt.Sprintf("refused — %s carries a stamp `toolsmith install %s` cannot use (unparseable, or from a schema version this binary does not recognize) — %s", dir, harnessName, remedy))
	default:
		return nil
	}
}
