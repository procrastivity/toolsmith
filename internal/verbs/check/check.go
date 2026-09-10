// Package check implements the `toolsmith check [path]` verb: a Go port
// of contrib/check-contract (docs/binary/port-spec.md is the normative
// spec this was ported against — cite it, not the shell, for anything
// that isn't obvious from the code). It audits a tool repo against the
// mechanical ([check]-marked) clauses of CONTRACT.md and reports findings
// flat, exhaustively (T21) — every failing clause, never just the first.
// Exit 0 clean, 1 with findings, 2 usage.
package check

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/procrastivity/toolsmith/internal/exitcode"
	"github.com/procrastivity/toolsmith/internal/iostreams"
	"github.com/procrastivity/toolsmith/internal/surface"
)

// Command constructs the `toolsmith check [path]` verb. streams is the
// writer pair threaded in at construction (C2.1).
//
// path defaults to "." when omitted — port spec §1, judgment call 2:
// contrib/check-contract:13 requires exactly one positional argument, but
// the Brief's verb table lists `check [path]` as optional. The parity
// gate always passes an explicit path, so this never affects byte
// parity; it only affects the CLI contract (port spec §8.1/§9).
// Everything else about the oracle's CLI contract holds: a path that does
// not resolve to a directory exits 2.
func Command(streams *iostreams.Streams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check [path]",
		Short: "audit a tool repo against the mechanical ([check]) clauses of CONTRACT.md",
		Long: "audit a tool repo against the mechanical ([check]-marked) clauses of CONTRACT.md.\n\n" +
			"path defaults to the current directory. Findings are printed flat, one \"<clause>: <message>\" line each, exhaustively — every failing clause, never just the first.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			path := "."
			if len(args) == 1 {
				path = args[0]
			}

			info, err := os.Stat(path)
			if err != nil || !info.IsDir() {
				// contrib/check-contract:13 folds "wrong argument count"
				// and "the one argument you gave isn't a directory" into
				// one combined usage guard (port spec §5.1, §8.1); Cobra's
				// Args already rejects the wrong-count case above this
				// RunE, so what lands here is only the non-directory
				// case. Returning a plain (non-toolsmitherr,
				// non-exitcode.Silent) error routes through Execute's
				// Cobra-argument-parsing fallback, which exits 2 — the
				// exit code is under parity, the message text is not
				// (port spec §1, judgment call 2).
				return fmt.Errorf("%q is not a directory", path)
			}

			// contrib/check-contract:17 resolves the argument to an
			// absolute path with `repo="$(cd "$1" && pwd)"` before doing
			// anything else, because that resolved path is itself part
			// of the stdout/stderr payload (the clean-run line and the
			// count-summary line both name it). filepath.Abs matches it
			// for the ordinary, symlink-free case every corpus repo is.
			repo, err := filepath.Abs(path)
			if err != nil {
				return err
			}

			findings, notes := Audit(repo)

			// port spec §5.1: stream ownership is asymmetric by design.
			// The multiple-cmd/-entries note is the only stderr content
			// that can appear before findings are known, and per §9.4 it
			// can co-occur with a clean (exit 0) run — so it is written
			// unconditionally, ahead of the exit-path branch below.
			for _, n := range notes {
				if _, err := fmt.Fprintln(streams.Err, n); err != nil {
					return err
				}
			}
			for _, f := range findings {
				if _, err := fmt.Fprintf(streams.Out, "%s: %s\n", f.Clause, f.Message); err != nil {
					return err
				}
			}
			if len(findings) == 0 {
				_, err := fmt.Fprintf(streams.Out, "no findings — mechanical clauses hold for %s\n", repo)
				return err
			}
			if _, err := fmt.Fprintf(streams.Err, "%d finding(s) for %s\n", len(findings), repo); err != nil {
				return err
			}

			// port spec §9.5, citing CONTRACT.md C2.4: on the
			// findings-present path, the finding lines already written
			// to streams.Out above are the verb's payload, not an error
			// message — the oracle's own exit-1 stdout is exactly those
			// lines, no clean-run line. The standard toolsmitherr.Render
			// path would both empty stdout (C2.2) and write its own
			// "toolsmith: check: <message>" line, which is neither the
			// count-summary line already written to stderr above nor
			// byte-identical to it. Returning exitcode.Silent(1) exits 1
			// without invoking that renderer, so the bytes already
			// written stand.
			return exitcode.Silent(1)
		},
	}
	surface.Annotate(cmd, surface.Plumbing)
	return cmd
}
