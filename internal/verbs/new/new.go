// Package new implements the `toolsmith new <name>` verb: a Go port of
// contrib/new-tool.sh (docs/binary/port-spec.md is the normative spec this
// was ported against — cite it, not the shell, for anything that isn't
// obvious from the code). It instantiates the shipped chassis skeleton as a
// new tool: copy the tree, rename the placeholder paths, rewrite the three
// placeholder spellings, restore the real module files, optionally make the
// first commit, and print the checklist of judgment steps the rename cannot
// do.
//
// The verb's real output is the produced directory tree, not stdout — its
// only stdout is the trailing checklist (port spec §5.2, §9.2). That is why
// the parity contract for this verb is the tree plus the happy-path
// checklist bytes plus exit 0, and why the gate diffs trees.
package new

import (
	"bytes"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/procrastivity/toolsmith/internal/iostreams"
	"github.com/procrastivity/toolsmith/internal/surface"
)

// Command constructs the `toolsmith new <name>` verb. streams is the writer
// pair threaded in at construction (C2.1).
//
// Exit codes follow CONTRACT.md C2.4, not the oracle's flat 1 (port spec §1
// judgment call 3, §9.3): Cobra's own path exits 2 for a bad flag, a
// missing value, a missing name and a second positional argument; the
// existing-target guard exits 3 as a refusal; every other failure exits 1.
// The oracle's single exit code is an artifact of one die() helper with two
// happy-path callers, and reproducing it would mean suppressing Cobra's
// usage path to preserve an accident nobody observes. Failure-path exit
// codes are excluded from this verb's parity contract accordingly.
func Command(streams *iostreams.Streams) *cobra.Command {
	var (
		targetDir string
		module    string
		noGit     bool
	)

	cmd := &cobra.Command{
		Use:   "new <name>",
		Short: "instantiate the chassis skeleton as a new tool",
		Long: "instantiate the chassis skeleton as a new tool.\n\n" +
			"<name> becomes the binary name, the Go package names, the environment-variable prefix, and the paths: lowercase letters and digits, starting with a letter.",
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			p, err := validate(args[0], targetDir, module, noGit)
			if err != nil {
				return err
			}

			skeleton, embedded, err := skeletonTree()
			if err != nil {
				return err
			}

			if err := instantiate(p, skeleton, embedded); err != nil {
				return err
			}

			if p.doGit {
				var gitOut bytes.Buffer
				gitErr := initGit(p, &gitOut)
				if gitOut.Len() > 0 {
					if _, err := streams.Err.Write(gitOut.Bytes()); err != nil {
						return err
					}
				}
				if gitErr != nil {
					return gitErr
				}
			}

			_, err = fmt.Fprint(streams.Out, checklist(p))
			return err
		},
	}

	// The oracle's --dir default is a sibling of the toolsmith repository
	// (contrib/new-tool.sh:72). A binary has no repository, so the default
	// is resolved in validate() instead of being spelled here, and the
	// help text says what it actually is.
	cmd.Flags().StringVar(&targetDir, "dir", "", "where to create the tool (default: ./<name>)")
	cmd.Flags().StringVar(&module, "module", "", "Go module path (default: github.com/procrastivity/<name>)")
	cmd.Flags().BoolVar(&noGit, "no-git", false, "skip git init and the first commit")

	surface.Annotate(cmd, surface.Plumbing)
	return cmd
}

// checklist is the verb's entire stdout contract, byte for byte
// (contrib/new-tool.sh:117-136, transcribed in port spec §5.2). Three
// values interpolate — the name, the target directory as the caller wrote
// it, and the module — and every other byte is literal, the blank line
// after the first line included.
//
// Item 7 still names contrib/check-contract rather than `toolsmith check`.
// That is correct for now and deliberate: the oracle owns these bytes for
// the duration of the parity window, and the wording changes at cutover
// (Stage 7), when the oracle it names is deleted.
func checklist(p params) string {
	return fmt.Sprintf(`instantiated %s at %s (module %s)

Checklist — the judgment steps the rename cannot do:
  1. grep -rn 'TODO(%s)' — fill every marker: root Short, README,
     flake meta.description, the judgment and agent-guidance assets, the
     skill description.
  2. cd %s && CGO_ENABLED=0 go build ./... && go test ./...
     (should already pass; it did in the skeleton).
  3. nix build — it fails once and prints the real vendorHash; paste it
     into flake.nix.
  4. make hooks — installs both pre-commit stages.
  5. Decide the verb surface; register verbs in internal/cli/root.go,
     one package each, every constructor ending in surface.Annotate.
  6. For a migration (not a fresh tool): follow toolsmith's
     assets/playbook/migrate.md — port spec, parity gate, cutover.
  7. Run contrib/check-contract %s from the toolsmith repo and
     clear any findings.
  8. Add the tool to toolsmith's TOOLS.md.
`, p.name, p.targetDir, p.module, p.name, p.targetDir, p.targetDir)
}
