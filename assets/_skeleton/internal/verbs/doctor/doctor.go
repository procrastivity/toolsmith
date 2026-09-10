// Package doctor implements the `toolname doctor` verb: it runs every
// registered check (internal/checks) and reports findings flat, with no
// severity levels (C4.7). A tool grows doctor by registering checks, never
// by a second command or a second output path.
package doctor

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/procrastivity/toolname/internal/buildinfo"
	"github.com/procrastivity/toolname/internal/checks"
	"github.com/procrastivity/toolname/internal/cliflags"
	"github.com/procrastivity/toolname/internal/iostreams"
	"github.com/procrastivity/toolname/internal/surface"
	"github.com/procrastivity/toolname/internal/toolnameerr"
)

// Command constructs the `toolname doctor` verb. root is the
// *cobra.Command NewRootCommand is assembling, captured by reference — the
// same pattern manifest/install use — so the stale-harness-artifact check
// reads the manifest every verb ultimately registered on it.
func Command(streams *iostreams.Streams, build buildinfo.Info, root *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "diagnose this host's toolname installation",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			flags := cliflags.FromContext(cmd.Context())

			findings, err := checks.Run(
				func() ([]checks.Finding, error) {
					return checks.CheckStaleHarnessArtifacts(root, build)
				},
			)
			if err != nil {
				return err
			}

			if flags.JSON {
				out := findings
				if out == nil {
					out = []checks.Finding{}
				}
				b, err := json.Marshal(struct {
					Findings []checks.Finding `json:"findings"`
				}{Findings: out})
				if err != nil {
					return err
				}
				if _, err := fmt.Fprintln(streams.Out, string(b)); err != nil {
					return err
				}
				return findingsError(findings)
			}

			if len(findings) == 0 {
				_, err := fmt.Fprintln(streams.Out, "no issues found")
				return err
			}
			for _, f := range findings {
				if _, err := fmt.Fprintf(streams.Out, "%s: %s\n", f.Code, f.Message); err != nil {
					return err
				}
			}
			return findingsError(findings)
		},
	}
	surface.Annotate(cmd, surface.Plumbing)
	return cmd
}

// findingsError signals doctor's exit posture — 0 with no failing
// findings, 1 with one or more failing findings. Advisory-prefixed codes
// remain in the same flat output list but never make doctor fail (C4.7).
func findingsError(findings []checks.Finding) error {
	failing := 0
	for _, finding := range findings {
		if !strings.HasPrefix(finding.Code, "advisory.") {
			failing++
		}
	}
	if failing == 0 {
		return nil
	}
	return toolnameerr.New("doctor.findings-present", fmt.Sprintf("%d finding(s) reported; see above", failing))
}
