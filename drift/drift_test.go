// Package drift gates toolsmith's own chassis against the copy of it that
// `toolsmith new` ships to every new tool. T2 says tools share the
// contract, not a library, so the chassis is copied per tool; T4 says the
// skeleton is a compiling Go module using toolname / TOOLNAME /
// toolnameerr as placeholders for that chassis.
// internal/verbs/new/instantiate.go's substitute is the source of truth
// for the substitution that instantiates it: toolname for toolsmith,
// TOOLNAME for TOOLSMITH, plus the module path and the two toolnameerr
// renames those two spellings already cover.
//
// Two gates, same shape, run by compareTrees:
//   - TestChassisMatchesSkeleton compares internal/ against
//     assets/_skeleton/internal/ — the Go chassis itself. Nothing checked
//     this and it drifted twice silently (Stage 5's manifest asset-walk
//     fix, commit 9cba3cd; Stage 6's claudecode comment fix, commit
//     64b8907) while every other gate stayed green.
//   - TestRootChassisMatchesSkeleton compares the chassis files outside
//     internal/ (Makefile, flake.nix, .pre-commit-config.yaml, and the
//     rest) against their assets/_skeleton counterparts.
//
// Both reverse the substitution on toolsmith's own files and require the
// result to be byte-identical to the skeleton's, except for an exact,
// reasoned exemption list. Two failure directions, both load-bearing:
//   - a real difference with no exemption covering it (drift slipped in
//     undetected, same as Stage 5 and Stage 6);
//   - an exemption with no real difference behind it anymore (the
//     exemption list rotted and is now hiding, not explaining).
//
// Enumeration. The skeleton side of each gate is a plain filesystem walk
// (loadTree): assets/_skeleton/internal/ for the first gate,
// assets/_skeleton/ minus internal/ for the second. The toolsmith side of
// the first gate walks internal/ the same way. The toolsmith side of the
// second gate instead reads `git -C <root> ls-files -z --cached` — not a
// filesystem walk, and not `--others --exclude-standard` — because
// untracked local clutter at the repo root (e.g. a nested agent worktree
// under an unignored .claude/) would otherwise fail the gate in a
// developer checkout; a git failure is t.Fatal, never t.Skip (audit.go's
// changelogTracked already runs git, so this is not a new dependency).
// flake.nix's checkPhase builds only subPackages = cmd/toolsmith, so this
// test and its git dependency never reach the nix sandbox. The cost: a new
// root file is invisible to this gate until it is staged; CI sees only
// committed files, so it always catches one. A tracked path that no longer
// exists on disk is skipped rather than erroring, and the list is deduped.
// Either way internal/ and assets/_skeleton/ are dropped from the
// toolsmith side before comparison, and a new file on either side fails in
// that direction: a new skeleton file with no toolsmith counterpart fails
// as "only in the skeleton", a new tracked toolsmith file no list covers
// fails as "only in toolsmith". Files that are already byte-identical
// after substitution need no list entry at all; they are compared
// automatically.
//
// Path mapping. Every toolsmith path is run through substitute (reversing
// the `new` verb's own substitution — internal/verbs/new/instantiate.go's
// substitute) to land in the skeleton's namespace, and — top-level paths
// only — through tmplRenames, which
// mirrors destPath in internal/verbs/new/instantiate.go: go.mod and
// go.sum are go.mod.tmpl and go.sum.tmpl in the skeleton, because a
// directory containing a real go.mod cannot be embedded (the
// skeletonPrefix comment in internal/verbs/new/instantiate.go explains
// why). Each key maps to at most one real path on each side — a
// collision (two real paths substituting to the same key) is t.Fatal, not
// a silent overwrite. Each tree keeps an origin map from every such key
// back to its real on-disk path, so failure messages and the printed
// `diff -u <(sed …) …` command name the files a reader can actually open.
//
// Three kinds of exemption:
//   - pathExemption: a file or directory that legitimately exists on only
//     one side (usually: only in toolsmith's tree). Removes it from the
//     toolsmith side.
//   - excludedPair: a file both trees carry that is deliberately not
//     compared at all — per-tool prose, or a whole-file placeholder.
//     Removes it from both sides, and counts as used only if both sides
//     have the file and the skeleton's copy still contains marker, so a
//     placeholder filled in by mistake reports a stale exclusion instead
//     of silently passing.
//   - textExemption: one known, reasoned textual difference inside a file
//     that otherwise matches.
//
// Placement: this package lives outside internal/ and outside assets/ on
// purpose. Anything under assets/ risks becoming a shipped asset (see
// internal/manifest/assets.go's isShippedAsset — a .go file inside a
// subdirectory of assets/ is payload, not infrastructure); anything under
// internal/ would be swept into the very internal/ tree the first gate
// walks and compares, and would need an exemption for itself just to
// exist. A package that is neither avoids both problems by construction,
// at the cost of living one level above the module's usual internal/
// boundary — judged an acceptable trade for a test that must never appear
// in its own comparison.
package drift

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// repoRoot resolves the toolsmith repo root from this file's own path
// (drift/drift_test.go, one level below the root) rather than from the
// working directory `go test` happens to be run from.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("drift: runtime.Caller could not resolve this test file's own path")
	}
	return filepath.Dir(filepath.Dir(file))
}

// loadTree reads every regular file under root into a map keyed by its
// slash-separated path relative to root, unmodified.
func loadTree(t *testing.T, root string) map[string]string {
	t.Helper()
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		t.Fatalf("drift: %s is not a directory (or does not exist): %v", root, err)
	}
	out := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		out[filepath.ToSlash(rel)] = string(data)
		return nil
	})
	if err != nil {
		t.Fatalf("drift: walking %s: %v", root, err)
	}
	return out
}

// substitute applies the reverse of the `new` verb's own two placeholder
// spellings (internal/verbs/new/instantiate.go's substitute): toolname
// for toolsmith, TOOLNAME for TOOLSMITH. The verb also substitutes the
// module path first, but github.com/procrastivity/toolname contains
// "toolname" as a segment, so the plain lowercase pass already covers
// it; there is nothing left for a separate module-path pass to do. The
// verb's placeholder constants name exactly three spellings — the module
// path, toolname and TOOLNAME — and the module path folds into toolname,
// so a mixed-case "Toolsmith" is never substituted by the verb either —
// and toolsmith's tree only ever uses that spelling inside
// internal/verbs/new (exempted below as toolsmith-only), so it never
// reaches this comparison.
func substitute(s string) string {
	s = strings.ReplaceAll(s, "toolsmith", "toolname")
	s = strings.ReplaceAll(s, "TOOLSMITH", "TOOLNAME")
	return s
}

// tmplRenames applies, at the top level only, the two renames
// instantiate.go's destPath gives back when it writes the skeleton out as
// a real tree: go.mod and go.sum cannot ship inside assets/_skeleton under
// their real names (the skeletonPrefix comment in
// internal/verbs/new/instantiate.go explains why a directory containing a
// literal go.mod cannot be embedded), so the
// skeleton carries them as go.mod.tmpl / go.sum.tmpl instead.
var tmplRenames = map[string]string{
	"go.mod": "go.mod.tmpl",
	"go.sum": "go.sum.tmpl",
}

// skeletonKey maps a toolsmith-relative path to the key it is compared
// under: substitute, then tmplRenames if the whole (already-substituted)
// path is a top-level renamed name.
func skeletonKey(rel string) string {
	rel = substitute(rel)
	if renamed, ok := tmplRenames[rel]; ok {
		return renamed
	}
	return rel
}

// tree is one side of a drift comparison: files, keyed by the comparison
// path (the substituted, skeleton-shaped namespace both sides share), and
// origin, which maps each key back to the real path on disk (relative to
// the repo root) so failure messages and diff commands name a file a
// reader can open.
type tree struct {
	files  map[string]string
	origin map[string]string
}

// loadSubstitutedToolsmithInternal loads toolsmith's own internal/ tree and
// applies the reverse substitution to both paths (so internal/toolsmitherr
// lands at toolnameerr/, matching the skeleton's directory and file
// renames) and contents.
func loadSubstitutedToolsmithInternal(t *testing.T, root string) tree {
	t.Helper()
	raw := loadTree(t, filepath.Join(root, "internal"))
	files := make(map[string]string, len(raw))
	origin := make(map[string]string, len(raw))
	for relPath, content := range raw {
		key := substitute(relPath)
		rel := path.Join("internal", relPath)
		if prev, dup := origin[key]; dup {
			t.Fatalf("drift: %s and %s both map to comparison key %s", prev, rel, key)
		}
		files[key] = substitute(content)
		origin[key] = rel
	}
	return tree{files: files, origin: origin}
}

// loadSkeletonInternal loads assets/_skeleton/internal/ as-is: it is
// already written in the skeleton's own toolname/TOOLNAME namespace, so no
// substitution applies and each key's origin is just its own path.
func loadSkeletonInternal(t *testing.T, root string) tree {
	t.Helper()
	raw := loadTree(t, filepath.Join(root, "assets", "_skeleton", "internal"))
	origin := make(map[string]string, len(raw))
	for p := range raw {
		origin[p] = path.Join("assets", "_skeleton", "internal", p)
	}
	return tree{files: raw, origin: origin}
}

// loadSkeletonRoot loads assets/_skeleton/ minus its internal/ subtree: the
// chassis files the new verb writes at the root of every tool it creates.
// Like loadSkeletonInternal, no substitution applies.
func loadSkeletonRoot(t *testing.T, root string) tree {
	t.Helper()
	raw := loadTree(t, filepath.Join(root, "assets", "_skeleton"))
	files := make(map[string]string, len(raw))
	origin := make(map[string]string, len(raw))
	for p, content := range raw {
		if p == "internal" || strings.HasPrefix(p, "internal/") {
			continue
		}
		files[p] = content
		origin[p] = path.Join("assets", "_skeleton", p)
	}
	return tree{files: files, origin: origin}
}

// loadSubstitutedToolsmithRoot loads toolsmith's own chassis files outside
// internal/, enumerated from `git ls-files --cached` rather than a
// filesystem walk — see the package comment for why. internal/ and
// assets/_skeleton/ are dropped (the first gate already covers internal/;
// assets/_skeleton/ is the skeleton itself, not a chassis file to compare
// against it), and a tracked path missing from disk is skipped rather than
// failing.
func loadSubstitutedToolsmithRoot(t *testing.T, root string) tree {
	t.Helper()
	out, err := exec.Command("git", "-C", root, "ls-files", "-z", "--cached").Output()
	if err != nil {
		t.Fatalf("drift: git ls-files --cached: %v", err)
	}

	files := make(map[string]string)
	origin := make(map[string]string)
	seen := make(map[string]bool)
	for _, rel := range strings.Split(strings.TrimSuffix(string(out), "\x00"), "\x00") {
		if rel == "" || seen[rel] {
			continue
		}
		seen[rel] = true
		rel = filepath.ToSlash(rel)
		if rel == "internal" || strings.HasPrefix(rel, "internal/") {
			continue
		}
		if rel == "assets/_skeleton" || strings.HasPrefix(rel, "assets/_skeleton/") {
			continue
		}

		full := filepath.Join(root, filepath.FromSlash(rel))
		info, statErr := os.Stat(full)
		if statErr != nil {
			if errors.Is(statErr, fs.ErrNotExist) {
				continue // tracked but missing from this checkout — nothing to compare
			}
			t.Fatalf("drift: stat %s: %v", full, statErr)
		}
		if info.IsDir() {
			continue // a gitlink — nothing to compare
		}
		data, readErr := os.ReadFile(full)
		if readErr != nil {
			t.Fatalf("drift: reading %s: %v", full, readErr)
		}

		key := skeletonKey(rel)
		if prev, dup := origin[key]; dup {
			t.Fatalf("drift: %s and %s both map to comparison key %s", prev, rel, key)
		}
		files[key] = substitute(string(data))
		origin[key] = rel
	}
	return tree{files: files, origin: origin}
}

// pathExemption marks a whole file or directory (named in the *substituted*
// toolsmith namespace) that legitimately exists only in toolsmith's tree
// and never in the skeleton's.
type pathExemption struct {
	prefix string // exact relative path, or a directory prefix ending in "/"
	reason string
}

func (pe pathExemption) matches(path string) bool {
	if strings.HasSuffix(pe.prefix, "/") {
		return strings.HasPrefix(path, pe.prefix)
	}
	return path == pe.prefix
}

// excludedPair marks a file both trees carry that is deliberately not
// compared at all — per-tool prose, or a whole-file placeholder the
// skeleton ships for a tool to fill in. It is removed from both sides, and
// counts as used only if both sides still have the file and the skeleton's
// copy still contains marker; a placeholder filled in by mistake (or a
// file deleted from one side) is then reported as a stale exclusion rather
// than silently passing.
type excludedPair struct {
	path   string
	marker string
	reason string
}

// textExemption marks one known, reasoned textual difference inside a file
// that otherwise exists on both sides. present must appear, verbatim and
// exactly once, in toolsmith's substituted content; replacement is what
// stands in its place in the skeleton's actual content. Applying
// strings.Replace(present -> replacement) to the toolsmith side is what
// "explains" the difference; if present is no longer found (or replacement
// is no longer in the skeleton file), the exemption is stale.
type textExemption struct {
	path        string
	present     string
	replacement string
	reason      string
}

// internalPathExemptions and internalTextExemptions together are the
// exact, reasoned exemption list for the internal/ <-> assets/_skeleton/
// internal/ chassis comparison. Every entry here was measured against the
// real trees, not assumed; anything found during that measurement that is
// not one of these entries is reported by compareTrees as an unexplained
// difference, never silently folded in here.
var internalPathExemptions = []pathExemption{
	{
		prefix: "verbs/check/",
		reason: "domain verb: the skeleton ships no domain verbs (T2); check is toolsmith's own conformance checker",
	},
	{
		prefix: "verbs/new/",
		reason: "domain verb: the skeleton ships no domain verbs (T2); new is toolsmith's own instantiator",
	},
	{
		prefix: "verbs/doc/",
		reason: "domain verb: the skeleton ships no domain verbs (T2); doc serves toolsmith's own CONTRACT.md, playbook and handoff-kit, none of which a generated tool ships an equivalent of",
	},
	{
		prefix: "cli/doc_e2e_test.go",
		reason: "e2e coverage for the toolsmith-only doc verb, kept in its own file (like verbs/check and verbs/new's own tests) so it never collides with e2e_test.go's shared chassis coverage",
	},
	{
		prefix: "asset/tree.go",
		reason: "Tree lets the new verb resolve the skeleton subtree it writes to disk, and DefaultTree lets the doc verb enumerate playbook/ and handoff-kit/ without the override link; both callers are toolsmith-only verbs the skeleton does not ship",
	},
	{
		prefix: "manifest/skeleton_payload_test.go",
		reason: "pins the payload the new verb writes to disk (assets/_skeleton/), which only toolsmith ships; a generated tool has no _skeleton/ of its own to pin",
	},
}

var internalTextExemptions = []textExemption{
	{
		path:        "cli/root.go",
		present:     "\tcheckverb \"github.com/procrastivity/toolname/internal/verbs/check\"\n",
		replacement: "",
		reason:      "imports the toolsmith-only check verb to register it; support code for a domain verb the skeleton does not ship",
	},
	{
		path:        "cli/root.go",
		present:     "\tnewverb \"github.com/procrastivity/toolname/internal/verbs/new\"\n",
		replacement: "",
		reason:      "imports the toolsmith-only new verb to register it; support code for a domain verb the skeleton does not ship",
	},
	{
		path:        "cli/root.go",
		present:     "\troot.AddCommand(checkverb.Command(streams))\n",
		replacement: "",
		reason:      "registers the toolsmith-only check verb on the root command",
	},
	{
		path:        "cli/root.go",
		present:     "\troot.AddCommand(newverb.Command(streams))\n",
		replacement: "",
		reason:      "registers the toolsmith-only new verb on the root command",
	},
	{
		path:        "cli/root.go",
		present:     "\tdocverb \"github.com/procrastivity/toolname/internal/verbs/doc\"\n",
		replacement: "",
		reason:      "imports the toolsmith-only doc verb to register it; support code for a domain verb the skeleton does not ship",
	},
	{
		path:        "cli/root.go",
		present:     "\troot.AddCommand(docverb.Command(streams))\n",
		replacement: "",
		reason:      "registers the toolsmith-only doc verb on the root command",
	},
	{
		path:        "cli/root.go",
		present:     "\t\tShort: \"toolname — instantiate the chassis, audit a tool against the contract, carry the migration playbook\",\n",
		replacement: "\t\tShort: \"toolname — TODO: one line on what this tool is\",\n",
		reason:      "placeholder: toolsmith new's own checklist names \"root Short\" as a marker to fill; same class as skillDescription",
	},
	{
		path:        "harness/registry/registry.go",
		present:     "toolname's assets/playbook/new-harness-target.md",
		replacement: "toolsmith's assets/playbook/new-harness-target.md",
		reason:      "names toolsmith the project, whose playbook every new tool must be pointed at; the reverse substitution cannot tell \"toolsmith, this tool\" from \"toolsmith, the project\", same class as the Contract constant",
	},
	{
		path:        "manifest/manifest.go",
		present:     "// Contract names the toolname contract version this tool conforms to\n",
		replacement: "// Contract names the toolsmith contract version this tool conforms to\n",
		reason:      "a tool conforms to toolsmith's contract, not its own; the doc comment states the same fact the Contract constant below carries, so the substitution must leave both alone",
	},
	{
		path:        "manifest/manifest.go",
		present:     "const Contract = \"toolname/v1\"\n",
		replacement: "const Contract = \"toolsmith/v1\"\n",
		reason:      "the skeleton keeps Contract = \"toolsmith/v1\" because a tool conforms to toolsmith's contract, not its own; the substitution would otherwise wrongly turn it into toolname/v1",
	},
}

// rootPathExemptions, rootExcludedPairs and rootTextExemptions together are
// the exact, reasoned exemption list for the second gate: toolsmith's
// chassis files outside internal/ against their assets/_skeleton/
// counterparts. Same discipline as the internal/ list — measured against
// the real trees, never assumed.
var rootPathExemptions = []pathExemption{
	{prefix: "CONTRACT.md", reason: "the contract itself; a tool conforms to it and carries only the Contract constant"},
	{prefix: "contract.go", reason: "embeds CONTRACT.md so the doc verb can serve it; the contract is not tunable behavior (C5.1) so it is never shipped as an asset, and a generated tool has no CONTRACT.md of its own to embed"},
	{prefix: "DECISIONS.md", reason: "toolsmith's decision register (T-numbers); a tool records its own decisions elsewhere (C7.1)"},
	{prefix: "TOOLS.md", reason: "the fleet register; exists once, in toolsmith"},
	{prefix: "assets/playbook/", reason: "the migration playbook toolsmith ships; a generated tool ships no playbook"},
	{prefix: "assets/handoff-kit/", reason: "the sidecar templates toolsmith ships; a generated tool ships none"},
	{prefix: "backport/", reason: "punch lists for existing tools; toolsmith's own register"},
	{prefix: "docs/binary/", reason: "the toolsmith-binary Matter's port spec and divergences (C7.2); a tool writes its own"},
	{prefix: "docs/contract-v1-2-reconcile/", reason: "the contract-v1-2-reconcile Matter's decisions (C7.1); a tool writes its own"},
	{prefix: "evidence/", reason: "toolsmith's own verification records (C7.3); the skeleton ships no evidence"},
	{prefix: "drift/", reason: "this gate; it compares toolsmith against the skeleton and has no meaning inside a generated tool"},
	{prefix: "flake.lock", reason: "per-repo nix lock; the skeleton ships none so each tool resolves nixpkgs when it is created instead of inheriting toolsmith's pin"},
	{prefix: "toolname.mk", reason: "toolsmith.mk (substituted key): the make target only toolsmith needs (smoke), split out so Makefile stays a pure substitution"},
}

var rootExcludedPairs = []excludedPair{
	{path: "README.md", marker: "TODO(toolname)", reason: "per-tool prose (C7.5); the skeleton's copy is a TODO scaffold and toolsmith's is the project front page, with no chassis behavior in either"},
	{path: "assets/agent-guidance.md", marker: "TODO(toolname)", reason: "whole-file placeholder: the skeleton's content is the instruction for what to write"},
	{path: "assets/templates/skills/claude-code/judgment.md", marker: "TODO(toolname)", reason: "whole-file placeholder: the per-harness judgment prose each tool writes for itself"},
	{path: "assets/templates/skills/claude-code/description.txt", marker: "TODO(toolname)", reason: "whole-file placeholder: the SKILL.md frontmatter description (C4.4), this harness's own one-line trigger sentence each tool writes for itself"},
}

var rootTextExemptions = []textExemption{
	{path: "Makefile", present: "\ninclude toolname.mk\n", replacement: "", reason: "pulls in toolsmith.mk, the toolsmith-only targets; the one line that lets the rest of the Makefile stay shared"},
	{path: ".gitignore", present: "# wip's per-clone render tree. Ignored by decision, not by\n# .git/info/exclude (C6.8): wip's doctor refuses to render into a\n# tracked .wip/, and the Matters themselves live in wip, not here.\n/.wip/\n\n", replacement: "", reason: "toolsmith uses wip (H8); not every tool does, and C6.8 asks each tool to decide its own posture"},
	{path: ".gitignore", present: "# Instantiation smoke-test output\n/tmp/\n\n", replacement: "", reason: "names the directory toolsmith's smoke target writes; a generated tool has no smoke target"},
	{path: "flake.nix", present: "          # The first `nix build` fails and prints the real hash — paste it\n          # here. Re-do this whenever go.mod changes.\n          vendorHash = \"sha256-komX1AmHt2NoF1x6xsNa2RFkfVzOXfYEMPhT0zwMxjw=\";\n", replacement: "          # TODO(toolname): the first `nix build` fails and prints the real\n          # hash — paste it here. Re-do this whenever go.mod changes.\n          vendorHash = pkgs.lib.fakeHash;\n", reason: "placeholder (new-tool checklist step 3); when go.mod changes, update this hash in the same commit as flake.nix"},
	{path: "flake.nix", present: "            description = \"toolname — the conventions repo for procrastivity-style tooling: it instantiates the chassis, audits a tool against the contract, and carries the migration playbook\";\n", replacement: "            description = \"toolname — TODO: one line on what this tool is\";\n", reason: "placeholder: the new-tool checklist names flake meta.description as a marker to fill"},
}

// TestChassisMatchesSkeleton is the first drift gate: internal/ against
// assets/_skeleton/internal/. See the package comment.
func TestChassisMatchesSkeleton(t *testing.T) {
	root := repoRoot(t)
	compareTrees(t, gateSpec{
		name:      "internal/ chassis drift gate",
		toolsmith: loadSubstitutedToolsmithInternal(t, root),
		skeleton:  loadSkeletonInternal(t, root),
		paths:     internalPathExemptions,
		texts:     internalTextExemptions,
	})
}

// TestRootChassisMatchesSkeleton is the second drift gate: toolsmith's
// chassis files outside internal/ against their assets/_skeleton/
// counterparts. See the package comment.
func TestRootChassisMatchesSkeleton(t *testing.T) {
	root := repoRoot(t)
	compareTrees(t, gateSpec{
		name:      "root chassis drift gate",
		toolsmith: loadSubstitutedToolsmithRoot(t, root),
		skeleton:  loadSkeletonRoot(t, root),
		paths:     rootPathExemptions,
		excluded:  rootExcludedPairs,
		texts:     rootTextExemptions,
	})
}

// gateSpec is one drift comparison: two trees, sharing one comparison
// namespace, and the exemptions that explain every known difference
// between them.
type gateSpec struct {
	name      string
	toolsmith tree
	skeleton  tree
	paths     []pathExemption
	excluded  []excludedPair
	texts     []textExemption
}

// staleLocation names where a stale exemption's key points: the real
// toolsmith path behind it, when the key survives as one (an exact file
// path still has an origin entry even after its file was removed from the
// comparison; a directory prefix never had one), else the gate's own name
// so the message still says which gate is talking.
func (g gateSpec) staleLocation(key string) string {
	if orig, ok := g.toolsmith.origin[key]; ok {
		return orig
	}
	return g.name
}

// compareTrees runs one drift gate: apply path exemptions, then excluded
// pairs, then text exemptions, then report whatever's left as unexplained
// differences, then report any exemption that went unused as stale. Both
// reports run (and both can fire in the same test) so a single run never
// hides one direction behind the other.
func compareTrees(t *testing.T, g gateSpec) {
	t.Helper()

	toolsmithFiles := g.toolsmith.files
	skeletonFiles := g.skeleton.files

	pathUsed := make([]bool, len(g.paths))
	for i, pe := range g.paths {
		for path := range toolsmithFiles {
			if pe.matches(path) {
				delete(toolsmithFiles, path)
				pathUsed[i] = true
			}
		}
	}

	excludedUsed := make([]bool, len(g.excluded))
	for i, ep := range g.excluded {
		_, tOK := toolsmithFiles[ep.path]
		skelContent, sOK := skeletonFiles[ep.path]
		if tOK && sOK && strings.Contains(skelContent, ep.marker) {
			delete(toolsmithFiles, ep.path)
			delete(skeletonFiles, ep.path)
			excludedUsed[i] = true
		}
	}

	textUsed := make([]bool, len(g.texts))
	for i, te := range g.texts {
		content, ok := toolsmithFiles[te.path]
		if !ok {
			continue // no such file left to apply this exemption to
		}
		if strings.Count(content, te.present) != 1 {
			continue // pattern gone (or now ambiguous) on the toolsmith side
		}
		skelContent, skelOK := skeletonFiles[te.path]
		if !skelOK || !strings.Contains(skelContent, te.replacement) {
			continue // the skeleton side no longer carries what this exemption expects
		}
		toolsmithFiles[te.path] = strings.Replace(content, te.present, te.replacement, 1)
		textUsed[i] = true
	}

	allPaths := make(map[string]struct{}, len(toolsmithFiles)+len(skeletonFiles))
	for p := range toolsmithFiles {
		allPaths[p] = struct{}{}
	}
	for p := range skeletonFiles {
		allPaths[p] = struct{}{}
	}
	paths := make([]string, 0, len(allPaths))
	for p := range allPaths {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	var unexplained []string
	for _, p := range paths {
		tc, tOK := toolsmithFiles[p]
		sc, sOK := skeletonFiles[p]
		tPath := g.toolsmith.origin[p]
		sPath := g.skeleton.origin[p]
		switch {
		case tOK && !sOK:
			unexplained = append(unexplained, fmt.Sprintf(
				"%s: only in toolsmith after substitution; no exemption covers it\n--- toolsmith (substituted), %d bytes ---\n%s",
				tPath, len(tc), tc))
		case !tOK && sOK:
			unexplained = append(unexplained, fmt.Sprintf(
				"%s: only in the skeleton; no exemption covers it\n--- skeleton, %d bytes ---\n%s",
				sPath, len(sc), sc))
		case tOK && sOK && tc != sc:
			unexplained = append(unexplained, fmt.Sprintf(
				"%s vs %s: content differs after substitution and every known exemption; no exemption covers the remainder\n%s",
				tPath, sPath, divergenceReport(tPath, sPath, tc, sc)))
		}
	}

	if len(unexplained) > 0 {
		t.Errorf("%s: %d unexempted difference(s) [direction: a difference with no exemption]:\n\n%s",
			g.name, len(unexplained), strings.Join(unexplained, "\n\n"))
	}

	var stale []string
	for i, pe := range g.paths {
		if !pathUsed[i] {
			stale = append(stale, fmt.Sprintf("%s: path exemption %q (%s): no file matches this anymore",
				g.staleLocation(pe.prefix), pe.prefix, pe.reason))
		}
	}
	for i, ep := range g.excluded {
		if !excludedUsed[i] {
			stale = append(stale, fmt.Sprintf("excluded pair %q (%s): one side no longer carries this file, or the skeleton's copy lost marker %q", ep.path, ep.reason, ep.marker))
		}
	}
	for i, te := range g.texts {
		if !textUsed[i] {
			stale = append(stale, fmt.Sprintf("%s: text exemption in %s (%s): present=%q / replacement=%q no longer both hold",
				g.staleLocation(te.path), te.path, te.reason, te.present, te.replacement))
		}
	}
	if len(stale) > 0 {
		t.Errorf("%s: %d stale exemption(s) — no matching real difference behind them anymore [direction: an exemption with no difference]:\n\n%s",
			g.name, len(stale), strings.Join(stale, "\n\n"))
	}
}

// divergenceReport renders a short, actionable report of where toolsmith's
// substituted content (a, at toolsmithPath) and the skeleton's actual
// content (b, at skeletonPath) first diverge: the line number, a few lines
// of context from each side, and a copy-pasteable command that prints the
// full aligned diff. A naive line-indexed comparison has no alignment — a
// single inserted or deleted line shifts every later line number, so one
// real edit turns into dozens of false-looking pairs that nobody can act
// on. This gives up on rendering the whole diff inline and instead points
// the reader at the one command that will.
func divergenceReport(toolsmithPath, skeletonPath, a, b string) string {
	al := strings.Split(a, "\n")
	bl := strings.Split(b, "\n")

	shorter := len(al)
	if len(bl) < shorter {
		shorter = len(bl)
	}
	first := shorter
	for i := 0; i < shorter; i++ {
		if al[i] != bl[i] {
			first = i
			break
		}
	}

	const contextLines = 3
	cmd := fmt.Sprintf(
		"diff -u <(sed -e 's/toolsmith/toolname/g; s/TOOLSMITH/TOOLNAME/g' %s) %s",
		toolsmithPath, skeletonPath)

	return fmt.Sprintf(
		"  first divergence at line %d (of %d/%d lines):\n    toolsmith (substituted):\n%s\n    skeleton:\n%s\n  full aligned diff:\n    %s",
		first+1, len(al), len(bl),
		indentLines(sliceLines(al, first, first+contextLines), "      "),
		indentLines(sliceLines(bl, first, first+contextLines), "      "),
		cmd)
}

// sliceLines returns lines[start:end], clamped to lines' bounds.
func sliceLines(lines []string, start, end int) []string {
	if start < 0 {
		start = 0
	}
	if start > len(lines) {
		start = len(lines)
	}
	if end > len(lines) {
		end = len(lines)
	}
	if end < start {
		end = start
	}
	return lines[start:end]
}

// indentLines prefixes every line for readability inside a larger message;
// an empty slice renders as "(end of file)" so a divergence caused by one
// side simply running out of lines is not reported as blank context.
func indentLines(lines []string, prefix string) string {
	if len(lines) == 0 {
		return prefix + "(end of file)"
	}
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = prefix + l
	}
	return strings.Join(out, "\n")
}
