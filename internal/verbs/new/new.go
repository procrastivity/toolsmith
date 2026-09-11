// Package new implements the `toolsmith new <name>` verb: a Go port of
// contrib/new-tool.sh, retired at cutover (Stage 7 of toolsmith-binary).
// docs/binary/port-spec.md records the script's behavior — cite it, not
// the shell, for anything that isn't obvious from the code. It
// instantiates the shipped chassis skeleton as a new tool: create the
// repository and check git's identity in it (initGitRepo), copy the tree,
// rename the placeholder paths, rewrite the three placeholder spellings,
// restore the real module files, make the first commit (commitGit), and
// print the checklist of judgment steps the rename cannot do.
//
// The repository comes first, not last as in the oracle (port spec §4.2
// step 8), because only a check inside the new repository predicts whether
// its commit will succeed (initGitRepo). Parity retired at cutover
// (725e94e), so the order is free to change.
//
// The verb's real output is the produced directory tree, not stdout — its
// only stdout is the trailing checklist (port spec §5.2, §9.2).
// golden_test.go's TestGoldenNew pins both: it diffs the produced tree
// against a model built from the shipped skeleton (compareTrees) and pins
// the checklist bytes against testdata/golden/new/.
package new

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/procrastivity/toolsmith/internal/cliflags"
	"github.com/procrastivity/toolsmith/internal/iostreams"
	"github.com/procrastivity/toolsmith/internal/surface"
)

// Command constructs the `toolsmith new <name>` verb. streams is the writer
// pair threaded in at construction (C2.1).
//
// Exit codes follow CONTRACT.md C2.4, not the oracle's flat 1 (port spec §1
// judgment call 3, §9.3): Cobra's own path exits 2 for a bad flag, a
// missing value, a missing name and a second positional argument; the
// existing-target guard exits 3 as a refusal; validation.* and
// not-found.* failures (a bad name, the placeholder name, git missing or
// unidentified) exit 1; internal.* failures (internal.instantiate,
// internal.skeleton-missing, internal.git-failed) exit 4, an unexpected
// break rather than something the caller did. The oracle's single exit
// code is an artifact of one die() helper with two happy-path callers, and
// reproducing it would mean suppressing Cobra's usage path to preserve an
// accident nobody observed. This divergence is recorded as D3 in
// docs/binary/parity-divergences.md.
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
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := validate(args[0], targetDir, module, noGit)
			if err != nil {
				return err
			}

			skeleton, embedded, err := skeletonTree()
			if err != nil {
				return err
			}

			// gitStep runs one git phase and passes git's own output to
			// stderr (commitGit's comment says why not stdout).
			var gitOut bytes.Buffer
			gitStep := func(step func(params, *bytes.Buffer) error) error {
				stepErr := step(p, &gitOut)
				if _, err := streams.Err.Write(gitOut.Bytes()); err != nil {
					return err
				}
				gitOut.Reset()
				return stepErr
			}

			if p.doGit {
				if err := gitStep(initGitRepo); err != nil {
					return err
				}
			}

			if err := instantiate(p, skeleton, embedded); err != nil {
				return err
			}

			if p.doGit {
				if err := gitStep(commitGit); err != nil {
					return err
				}
			}

			// --json binds once at root, so every verb honors it rather
			// than silently ignoring it (C2.3). The JSON value replaces
			// the checklist, so stdout still carries one thing (C2.2).
			if cliflags.FromContext(cmd.Context()).JSON {
				b, err := json.Marshal(jsonResult{
					Name:   p.name,
					Dir:    p.targetDir,
					Module: p.module,
					Git:    p.doGit,
				})
				if err != nil {
					return err
				}
				_, err = fmt.Fprintln(streams.Out, string(b))
				return err
			}

			_, err = fmt.Fprint(streams.Out, checklist(p))
			return err
		},
	}

	// The oracle's --dir default is a sibling of the toolsmith repository.
	// A binary has no repository, so the default is resolved in
	// validate() instead of being spelled here, and the help text says
	// what it actually is. Recorded as D5 in
	// docs/binary/parity-divergences.md.
	cmd.Flags().StringVar(&targetDir, "dir", "", "where to create the tool (default: ./<name>)")
	cmd.Flags().StringVar(&module, "module", "", "Go module path (default: github.com/procrastivity/<name>)")
	cmd.Flags().BoolVar(&noGit, "no-git", false, "skip git init and the first commit")

	surface.Annotate(cmd, surface.Plumbing)
	return cmd
}

// jsonResult is new's --json success payload (C2.3): the instantiated
// tool's name, its target directory exactly as the caller wrote it, its
// module path, and whether a git repository was created and committed
// (false with --no-git).
type jsonResult struct {
	Name   string `json:"name"`
	Dir    string `json:"dir"`
	Module string `json:"module"`
	Git    bool   `json:"git"`
}

// checklist is the verb's entire stdout contract, byte for byte (port
// spec §4.2 step 9, transcribed in §5.2). Three values interpolate — the
// name, the target directory as the caller wrote it, and the module —
// and every other byte is literal, the blank line after the first line
// included.
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
  7. Run toolsmith check %s and clear any findings.
  8. Add the tool to toolsmith's TOOLS.md.
`, p.name, p.targetDir, p.module, p.name, p.targetDir, p.targetDir)
}
