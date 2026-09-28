// Package new implements the `toolsmith new [name]` verb. With a name it
// retains the fresh-target behavior ported from contrib/new-tool.sh,
// retired at cutover (Stage 7 of toolsmith-binary).
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
// With no name it writes into a mostly-bare current directory without
// changing Git state or replacing existing planning files.
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

// Command constructs the `toolsmith new [name]` verb. streams is the writer
// pair threaded in at construction (C2.1).
//
// Exit codes follow CONTRACT.md C2.4, not the oracle's flat 1 (port spec §1
// judgment call 3, §9.3): Cobra's own path exits 2 for a bad flag, a
// missing value, --dir without a name and a second positional argument; the
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
		nameFlag  string
		noGit     bool
	)

	cmd := &cobra.Command{
		Use:   "new [name]",
		Short: "instantiate the chassis skeleton as a new tool",
		Long: "instantiate the chassis skeleton as a new tool.\n\n" +
			"Names use lowercase ASCII letters and digits in hyphen-separated segments, starting with a letter. The public name becomes the binary, module default, config and share directories, and harness name. Go and Nix identifiers omit hyphens; environment prefixes replace them with underscores (docker-extras → dockerextras, DOCKER_EXTRAS). With no positional name, instantiate in the current mostly-bare directory, using its basename or --name. Existing README.md, .gitignore and LICENSE are preserved; other skeleton path conflicts are refused. In-place mode does not change Git history or the index.",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.RangeArgs(0, 1)(cmd, args); err != nil {
				return err
			}
			if len(args) == 0 && cmd.Flags().Changed("dir") {
				return fmt.Errorf("--dir requires a positional <name>; --name is only for in-place mode")
			}
			if len(args) != 0 && cmd.Flags().Changed("name") {
				return fmt.Errorf("--name cannot be combined with a positional <name>; use the positional name for a new target")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var p params
			var err error
			if len(args) == 0 {
				p, err = validateInPlace(nameFlag, module)
			} else {
				p, err = validate(args[0], targetDir, module, noGit)
			}
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

			if p.inPlace {
				if err := preflightInPlace(p, skeleton); err != nil {
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
	cmd.Flags().StringVar(&targetDir, "dir", "", "where to create the tool (default: ./<name>; omit <name> to use the current directory)")
	cmd.Flags().StringVar(&nameFlag, "name", "", "tool name for in-place mode (default: current directory basename; cannot combine with <name> or --dir)")
	cmd.Flags().StringVar(&module, "module", "", "Go module path (default: github.com/procrastivity/<name>)")
	cmd.Flags().BoolVar(&noGit, "no-git", false, "skip git init and the first commit")

	surface.Annotate(cmd, surface.Plumbing)
	return cmd
}

// jsonResult is new's --json success payload (C2.3): the instantiated
// tool's name, its target directory exactly as the caller wrote it, its
// module path, and whether a git repository was created and committed
// (false with --no-git or in-place mode).
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
	result := fmt.Sprintf(`instantiated %s at %s (module %s)

Checklist — the judgment steps the rename cannot do:
  1. grep -rn 'TODO(%s)' — fill every marker: root Short, README,
     flake meta.description, the judgment and agent-guidance assets, the
     skill description.
  2. cd %s && CGO_ENABLED=0 go build ./... && go test ./...
     (should already pass; it did in the skeleton).
  3. go mod vendor — commit vendor/ whenever a dependency changes;
     flake.nix ships vendorHash = null, so there is no hash to paste.
  4. make hooks — installs both pre-commit stages.
  5. Decide the verb surface; register verbs in internal/cli/root.go,
     one package each, every constructor ending in surface.Annotate.
  6. For a migration (not a fresh tool): run toolsmith doc
     playbook/migrate.md and follow it — port spec, parity gate, cutover.
  7. Run toolsmith check %s and clear any findings.
  8. Add the tool to toolsmith's TOOLS.md.
`, p.name, p.targetDir, p.module, p.name, p.targetDir, p.targetDir)
	if p.inPlace {
		result += "  9. Existing README.md, .gitignore and LICENSE were left untouched where present;\n" +
			"     reconcile them with the chassis. Review and commit the new files yourself.\n"
	}
	return result
}
