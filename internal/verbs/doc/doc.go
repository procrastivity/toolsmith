// Package doc implements the `toolsmith doc [path]` verb: it lists, or
// prints, the docs this binary carries — the contract it is audited
// against, the migration playbook, and the handoff-kit templates — so a
// session with only the installed binary (no clone of this repository) can
// still read them. No other verb prints CONTRACT.md, and CONTRACT.md is
// not shipped as an asset at all (root contract.go's doc comment explains
// why); the playbook and handoff-kit ship under assets/, same as every
// other tunable asset, and are enumerated from that tree rather than a
// second, hand-kept list.
//
// The doc set is exactly three things: CONTRACT.md, every file under
// playbook/, and every file under handoff-kit/. Nothing else in the asset
// tree is a doc — _skeleton/, templates/, agent-guidance.md and
// config.default.yaml all answer not-found.
package doc

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"sort"

	"github.com/spf13/cobra"

	"github.com/procrastivity/toolsmith"
	"github.com/procrastivity/toolsmith/internal/asset"
	"github.com/procrastivity/toolsmith/internal/cliflags"
	"github.com/procrastivity/toolsmith/internal/iostreams"
	"github.com/procrastivity/toolsmith/internal/surface"
	"github.com/procrastivity/toolsmith/internal/toolsmitherr"
)

// treePrefixes are the asset subtrees, beyond CONTRACT.md, whose files are
// docs. Order fixes the order docNames builds its (later sorted) list in;
// it has no effect on behavior.
var treePrefixes = []string{"playbook", "handoff-kit"}

// Command constructs the `toolsmith doc [path]` verb. streams is the
// writer pair threaded in at construction (C2.1).
func Command(streams *iostreams.Streams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doc [path]",
		Short: "list the contract, playbook and handoff-kit this binary carries, or print one",
		Long: "list the contract, playbook and handoff-kit this binary carries, or print one.\n\n" +
			"With no argument, list doc names, sorted lexically, one per line. With one argument, it must exactly equal a listed name; the file's bytes print to stdout verbatim, with no added newline.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := cliflags.FromContext(cmd.Context())

			names, err := docNames()
			if err != nil {
				return err
			}

			if len(args) == 0 {
				return list(streams, flags, names)
			}
			return show(streams, flags, names, args[0])
		},
	}
	surface.Annotate(cmd, surface.Plumbing)
	return cmd
}

// docNames returns every doc name this binary carries, sorted lexically:
// "CONTRACT.md" plus every file under playbook/ and handoff-kit/, each
// named by its path relative to the served tree (no "assets/" prefix).
// playbook/ and handoff-kit/ are enumerated from the default -> embedded
// link of the asset chain only (asset.DefaultTree), never the user-override
// link — an override shadows an existing name at content-resolution time
// (show, via asset.Resolve) but never adds a new one to this list.
func docNames() ([]string, error) {
	names := []string{"CONTRACT.md"}
	for _, prefix := range treePrefixes {
		sub, err := treeNames(prefix)
		if err != nil {
			return nil, err
		}
		names = append(names, sub...)
	}
	sort.Strings(names)
	return names, nil
}

// treeNames lists every regular file under the default/embedded copy of
// the prefix subtree (playbook or handoff-kit), each name joined back with
// prefix so it reads as a doc name ("playbook/intake.md").
func treeNames(prefix string) ([]string, error) {
	tree, _, err := asset.DefaultTree(prefix)
	if err != nil {
		return nil, fmt.Errorf("doc: resolving %s tree: %w", prefix, err)
	}

	var names []string
	err = fs.WalkDir(tree, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		names = append(names, path.Join(prefix, p))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("doc: walking %s tree: %w", prefix, err)
	}
	return names, nil
}

// listPayload is doc's --json shape with no argument.
type listPayload struct {
	Docs []string `json:"docs"`
}

// list prints every name in names, sorted, one per line — or, under
// --json, the same names as {"docs":[...]}.
func list(streams *iostreams.Streams, flags cliflags.Flags, names []string) error {
	if flags.JSON {
		b, err := json.Marshal(listPayload{Docs: names})
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(streams.Out, string(b))
		return err
	}
	for _, n := range names {
		if _, err := fmt.Fprintln(streams.Out, n); err != nil {
			return err
		}
	}
	return nil
}

// showPayload is doc's --json shape with one argument: name is the doc
// name exactly as given (and listed), source is one of the asset package's
// Source strings ("embedded" for CONTRACT.md, which never resolves through
// the override/default chain), and content is the file's bytes.
type showPayload struct {
	Path    string `json:"path"`
	Source  string `json:"source"`
	Content string `json:"content"`
}

// show validates name against names and either prints its content or
// raises not-found.doc. name must exactly equal a listed name — no
// normalization, no path cleaning — so an unknown name, a name outside the
// doc set (e.g. "_skeleton/Makefile", "agent-guidance.md"), a bare
// "playbook", or a traversal like "../CONTRACT.md" all fall through to the
// same not-found case.
func show(streams *iostreams.Streams, flags cliflags.Flags, names []string, name string) error {
	if !contains(names, name) {
		return toolsmitherr.New("not-found.doc", fmt.Sprintf("no doc named %q; run 'toolsmith doc' to list them", name))
	}

	data, source, diskPath, err := resolveDoc(name)
	if err != nil {
		return err
	}

	if flags.Verbose {
		line := fmt.Sprintf("doc: serving %s from %s", name, source)
		if diskPath != "" {
			line += " at " + diskPath
		}
		if _, err := fmt.Fprintln(streams.Err, line); err != nil {
			return err
		}
	}

	if flags.JSON {
		b, err := json.Marshal(showPayload{Path: name, Source: source.String(), Content: string(data)})
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(streams.Out, string(b))
		return err
	}

	_, err = streams.Out.Write(data)
	return err
}

// resolveDoc returns name's content, the link of the chain that served it,
// and the disk path when there is one (empty for an embedded source).
// CONTRACT.md always comes from the root embed (contract.go) — never
// through the override/default/embedded asset chain, per C5.1's own scope
// (the contract is not tunable behavior) — so an override file named
// CONTRACT.md is ignored. Every other doc name resolves through
// asset.Resolve, which shadows by name (C5.2): an override at
// $XDG_CONFIG_HOME/toolsmith/playbook/x.md replaces the shipped default at
// that same name.
func resolveDoc(name string) (data []byte, source asset.Source, diskPath string, err error) {
	if name == "CONTRACT.md" {
		return toolsmith.Contract(), asset.SourceEmbedded, "", nil
	}
	resolved, err := asset.Resolve(name)
	if err != nil {
		return nil, 0, "", err
	}
	return resolved.Bytes(), resolved.Source, resolved.Path, nil
}

func contains(names []string, name string) bool {
	for _, n := range names {
		if n == name {
			return true
		}
	}
	return false
}
