// Package checks is doctor's pluggable check registry (C4.7): doctor grows
// by registering a new Check here and in the doctor verb's call site, never
// by a second command or a second output path. Findings are flat
// {code, message} pairs with no severity levels; a code with the
// "advisory." prefix never fails the run.
package checks

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/procrastivity/toolsmith/internal/buildinfo"
	"github.com/procrastivity/toolsmith/internal/harness/registry"
	"github.com/procrastivity/toolsmith/internal/manifest"
)

// Finding is one reported condition: a stable dotted machine code and the
// human-readable message.
type Finding struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Check is one doctor check. Every check reports every instance of its
// condition present at the time of the run — never just the first (C4.7).
type Check func() ([]Finding, error)

// Run executes checks in order and concatenates their findings. A check
// error aborts the run — that is an I/O failure, not a finding.
func Run(cs ...Check) ([]Finding, error) {
	var findings []Finding
	for _, c := range cs {
		fs, err := c()
		if err != nil {
			return nil, err
		}
		findings = append(findings, fs...)
	}
	return findings, nil
}

// StaleHarnessCode is the advisory code for a generated harness artifact
// the current binary would render differently — drift the user resolves by
// re-running install, so it never fails doctor.
const StaleHarnessCode = "advisory.stale-harness-artifact"

// CheckStaleHarnessArtifacts compares what the current binary would
// generate for every registered harness (registry.All, in table order)
// against what the last install stamped on disk (C4.6's binary-vs-stamp
// comparison), concatenating each harness's findings. root is the
// *cobra.Command NewRootCommand is assembling, captured by reference — the
// same pattern the manifest and install verbs use — so the manifest this
// reads reflects every verb actually registered. A never-installed target
// carries no stamp and is not a finding — there is nothing to have drifted
// from.
func CheckStaleHarnessArtifacts(root *cobra.Command, build buildinfo.Info) ([]Finding, error) {
	var findings []Finding
	for _, h := range registry.All {
		fs, err := checkStaleHarnessArtifact(root, build, h.Name, h.InstallDir, h.Generate)
		if err != nil {
			return nil, err
		}
		findings = append(findings, fs...)
	}
	return findings, nil
}

func checkStaleHarnessArtifact(
	root *cobra.Command, build buildinfo.Info,
	harnessName string,
	installDir func() (string, error),
	generate func(manifest.Manifest) (map[string][]byte, error),
) ([]Finding, error) {
	dir, err := installDir()
	if err != nil {
		return nil, err
	}
	stamp, ok, err := manifest.ReadStamp(dir)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}

	m, err := manifest.Build(root, build)
	if err != nil {
		return nil, err
	}
	files, err := generate(m)
	if err != nil {
		return nil, err
	}
	want := manifest.ChecksumFiles(files)

	drifted := manifest.Drift(want, stamp)
	findings := make([]Finding, 0, len(drifted))
	for _, d := range drifted {
		findings = append(findings, Finding{
			Code: StaleHarnessCode,
			Message: fmt.Sprintf("found: %s harness artifact %s %s since last install; run `toolsmith install %s` to refresh it",
				harnessName, d.Path, d.Reason, harnessName),
		})
	}
	return findings, nil
}
