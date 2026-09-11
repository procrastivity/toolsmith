// Package check implements the `toolsmith check [path]` verb: a Go port
// of contrib/check-contract, retired at cutover (Stage 7 of
// toolsmith-binary). docs/binary/port-spec.md records the script's
// behavior — cite it, not the shell, for anything that isn't obvious
// from the code. It audits a tool repo against the mechanical
// ([check]-marked) clauses of CONTRACT.md and reports findings flat,
// exhaustively (T21) — every failing clause, never just the first.
// Exit 0 clean, 1 with findings, 2 usage.
package check

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"

	"github.com/procrastivity/toolsmith/internal/cliflags"
	"github.com/procrastivity/toolsmith/internal/iostreams"
	"github.com/procrastivity/toolsmith/internal/surface"
	"github.com/procrastivity/toolsmith/internal/toolsmitherr"
)

// jsonFinding mirrors doctor's checks.Finding shape (C2.3: every verb
// accepts --json; T21: the checker's payload matches a tool's doctor) —
// defined locally rather than importing internal/checks, which is
// doctor's own check registry, not a general finding type.
type jsonFinding struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// jsonOutput is check --json's one stdout value. Audited names the
// clause IDs the checker itself claims to cover (T24) — present on both
// the clean and findings-present runs, and distinct from any tool's own
// declared conformance.
type jsonOutput struct {
	Findings []jsonFinding `json:"findings"`
	Audited  []string      `json:"audited"`
}

// Command constructs the `toolsmith check [path]` verb. streams is the
// writer pair threaded in at construction (C2.1).
//
// path defaults to "." when omitted — port spec §1, judgment call 2: the
// oracle requires exactly one positional argument (port spec §8.1), but
// the Brief's verb table lists `check [path]` as optional.
// TestCheck_DefaultPath (check_test.go) pins this behavior directly.
// Everything else about the oracle's CLI contract holds: a path that does
// not resolve to a directory exits 2.
func Command(streams *iostreams.Streams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check [path]",
		Short: "audit a tool repo against the mechanical ([check]) clauses of CONTRACT.md",
		Long: "audit a tool repo against the mechanical ([check]-marked) clauses of CONTRACT.md.\n\n" +
			"path defaults to the current directory. Findings are printed flat, one \"<clause>: <message>\" line each, exhaustively — every failing clause, never just the first.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := cliflags.FromContext(cmd.Context())
			path := "."
			if len(args) == 1 {
				path = args[0]
			}

			info, err := os.Stat(path)
			if err != nil || !info.IsDir() {
				// The oracle folds "wrong argument count" and "the one
				// argument you gave isn't a directory" into one combined
				// usage guard (port spec §5.1, §8.1); Cobra's Args
				// already rejects the wrong-count case above this RunE,
				// so what lands here is only the non-directory case.
				// Returning a plain (non-toolsmitherr,
				// non-exitcode.Silent) error routes through Execute's
				// Cobra-argument-parsing fallback, which exits 2,
				// matching the oracle's usage exit code (port spec §1,
				// judgment call 2). TestCheck_NotADirectory and
				// TestGoldenCheckCLI pin the exit code; nothing pins the
				// message text.
				return fmt.Errorf("%q is not a directory", path)
			}

			// The oracle resolves the argument to an absolute path with
			// `repo="$(cd "$1" && pwd)"` before doing anything else
			// (port spec §2.1), because that resolved path is itself
			// part of the stdout/stderr payload (the clean-run line and
			// the count-summary line both name it). filepath.Abs matches
			// it for the ordinary, symlink-free case every corpus repo
			// is.
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

			if flags.JSON {
				out := jsonOutput{
					Findings: make([]jsonFinding, 0, len(findings)),
					// AuditedClauses() is already sorted; copy and sort
					// again so this stays true even if that changes.
					Audited: append([]string(nil), AuditedClauses()...),
				}
				sort.Strings(out.Audited)
				for _, f := range findings {
					out.Findings = append(out.Findings, jsonFinding{Code: f.Clause, Message: f.Message})
				}
				b, err := json.Marshal(out)
				if err != nil {
					return err
				}
				if _, err := fmt.Fprintln(streams.Out, string(b)); err != nil {
					return err
				}
				return findingsVerdict(findings, repo)
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
			return findingsVerdict(findings, repo)
		},
	}
	surface.Annotate(cmd, surface.Plumbing)
	return cmd
}

// findingsVerdict returns nil on a clean run, else the verdict about the
// findings already written to stdout above (C2.5): Render writes it to
// stderr as one line and the exit is 1, as for doctor. Rejected:
// exitcode.Silent, which served the parity oracle retired at cutover
// (725e94e) and would keep the verdict out of --json.
func findingsVerdict(findings []Finding, repo string) error {
	if len(findings) == 0 {
		return nil
	}
	return toolsmitherr.New("check.findings-present", fmt.Sprintf("%d finding(s) for %s", len(findings), repo))
}
