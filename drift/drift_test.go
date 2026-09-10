// Package drift is the drift gate between toolsmith's own chassis
// (internal/) and the copy of it that contrib/new-tool.sh ships to every
// new tool (assets/_skeleton/internal/). T2 says tools share the contract,
// not a library, so the chassis is copied per tool; T4 says the skeleton is
// a compiling Go module using toolname / TOOLNAME / toolnameerr as
// placeholders for that chassis. contrib/new-tool.sh is the source of truth
// for the substitution that instantiates it: toolname for toolsmith,
// TOOLNAME for TOOLSMITH, plus the module path and the two toolnameerr
// renames those two spellings already cover. Nothing checked that the two
// trees stayed a substitution apart, and they drifted twice silently
// (Stage 5's manifest asset-walk fix, commit 9cba3cd; Stage 6's claudecode
// comment fix, commit 64b8907) while every other gate stayed green. This
// test is the fix: it reverses the substitution on toolsmith's own
// internal/ tree and requires the result to be byte-identical to
// assets/_skeleton/internal/, except for an exact, reasoned exemption list.
//
// Two failure directions, both load-bearing:
//   - a real difference with no exemption covering it (drift slipped in
//     undetected, same as Stage 5 and Stage 6);
//   - an exemption with no real difference behind it anymore (the
//     exemption list rotted and is now hiding, not explaining).
//
// Placement: this package lives outside internal/ and outside assets/ on
// purpose. Anything under assets/ risks becoming a shipped asset (see
// internal/manifest/assets.go's isShippedAsset — a .go file inside a
// subdirectory of assets/ is payload, not infrastructure); anything under
// internal/ would be swept into the very internal/ tree this test walks
// and compares, and would need an exemption for itself just to exist. A
// package that is neither avoids both problems by construction, at the
// cost of living one level above the module's usual internal/ boundary —
// judged an acceptable trade for a test that must never appear in its own
// comparison.
package drift

import (
	"fmt"
	"io/fs"
	"os"
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

// substitute applies contrib/new-tool.sh's own two placeholder spellings in
// reverse: toolname for toolsmith, TOOLNAME for TOOLSMITH. new-tool.sh also
// substitutes the module path first, but github.com/procrastivity/toolname
// contains "toolname" as a segment, so the plain lowercase pass already
// covers it; there is nothing left for a separate module-path pass to do.
// The script uses exactly these two spellings and no others (its own
// comment: "the skeleton uses exactly three placeholder spellings —
// toolname..., TOOLNAME..., and nothing else" once the module path is
// folded in), so a third, mixed-case "Toolsmith" is never substituted by
// new-tool.sh either — and toolsmith's tree only ever uses that spelling
// inside internal/verbs/new (exempted below as toolsmith-only), so it never
// reaches this comparison.
func substitute(s string) string {
	s = strings.ReplaceAll(s, "toolsmith", "toolname")
	s = strings.ReplaceAll(s, "TOOLSMITH", "TOOLNAME")
	return s
}

// loadSubstitutedToolsmithInternal loads toolsmith's own internal/ tree and
// applies the reverse substitution to both paths (so internal/toolsmitherr
// lands at toolname/toolnameerr, matching the skeleton's directory and file
// renames) and contents.
func loadSubstitutedToolsmithInternal(t *testing.T, root string) map[string]string {
	t.Helper()
	raw := loadTree(t, filepath.Join(root, "internal"))
	out := make(map[string]string, len(raw))
	for relPath, content := range raw {
		out[substitute(relPath)] = substitute(content)
	}
	return out
}

// pathExemption marks a whole file or directory (named in the *substituted*
// toolsmith namespace) that legitimately exists only in toolsmith's
// internal/ and never in the skeleton's.
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

// pathExemptions and textExemptions together are the exact, reasoned
// exemption list for the toolsmith <-> assets/_skeleton chassis
// comparison. Every entry here was measured against the real trees, not
// assumed; anything found during that measurement that is not one of these
// entries is reported by the calling gate as an unexplained difference,
// never silently folded in here.
var pathExemptions = []pathExemption{
	{
		prefix: "verbs/check/",
		reason: "domain verb: the skeleton ships no domain verbs (T2); check is toolsmith's own conformance checker",
	},
	{
		prefix: "verbs/new/",
		reason: "domain verb: the skeleton ships no domain verbs (T2); new is toolsmith's own instantiator",
	},
	{
		prefix: "asset/tree.go",
		reason: "exists only to let the new verb resolve the skeleton subtree it writes to disk (its own doc comment points at internal/verbs/new); no caller outside that verb",
	},
	{
		prefix: "manifest/skeleton_payload_test.go",
		reason: "pins the payload the new verb writes to disk (assets/_skeleton/), which only toolsmith ships; a generated tool has no _skeleton/ of its own to pin",
	},
}

var textExemptions = []textExemption{
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
		present:     "\t\tShort: \"toolname — instantiate the chassis, audit a tool against the contract, carry the migration playbook\",\n",
		replacement: "\t\tShort: \"toolname — TODO: one line on what this tool is\",\n",
		reason:      "placeholder: new-tool.sh's own checklist names \"root Short\" as a marker to fill; same class as skillDescription",
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
	{
		path:        "harness/claudecode/claudecode.go",
		present:     "// pair is a tension with the clause rather than a conformance to it \u2014\n// see the Stage 6 finding on the toolname-binary Matter.\n",
		replacement: "// pair is a tension with the clause rather than a conformance to it.\n",
		reason:      "toolsmith's copy carries one extra sentence in the package comment citing the toolsmith-binary Matter; a generated tool must not cite toolsmith's own Matter register",
	},
	{
		path:        "harness/claudecode/claudecode.go",
		present:     "const skillDescription = \"Instantiate the toolname chassis for a new procrastivity-style CLI tool, or audit an existing tool repo against the toolname contract.\"\n",
		replacement: "// TODO(toolname): replace with one line saying when to reach for this\n// tool \u2014 Claude reads it to decide when to load the skill.\nconst skillDescription = \"Drive toolname through its plumbing verb surface.\"\n",
		reason:      "the skeleton keeps a TODO(toolname) marker and a placeholder skillDescription; toolsmith's copy carries its own real one-line description instead",
	},
}

// TestChassisMatchesSkeleton is the drift gate. See the package comment.
func TestChassisMatchesSkeleton(t *testing.T) {
	root := repoRoot(t)

	toolsmithTree := loadSubstitutedToolsmithInternal(t, root)
	skeletonTree := loadTree(t, filepath.Join(root, "assets", "_skeleton", "internal"))

	pathExemptionUsed := make([]bool, len(pathExemptions))
	for i, pe := range pathExemptions {
		for path := range toolsmithTree {
			if pe.matches(path) {
				delete(toolsmithTree, path)
				pathExemptionUsed[i] = true
			}
		}
	}

	textExemptionUsed := make([]bool, len(textExemptions))
	for i, te := range textExemptions {
		content, ok := toolsmithTree[te.path]
		if !ok {
			continue // no such file left to apply this exemption to
		}
		if strings.Count(content, te.present) != 1 {
			continue // pattern gone (or now ambiguous) on the toolsmith side
		}
		skelContent, skelOK := skeletonTree[te.path]
		if !skelOK || !strings.Contains(skelContent, te.replacement) {
			continue // the skeleton side no longer carries what this exemption expects
		}
		toolsmithTree[te.path] = strings.Replace(content, te.present, te.replacement, 1)
		textExemptionUsed[i] = true
	}

	allPaths := make(map[string]struct{}, len(toolsmithTree)+len(skeletonTree))
	for p := range toolsmithTree {
		allPaths[p] = struct{}{}
	}
	for p := range skeletonTree {
		allPaths[p] = struct{}{}
	}
	paths := make([]string, 0, len(allPaths))
	for p := range allPaths {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	var unexplained []string
	for _, p := range paths {
		tc, tOK := toolsmithTree[p]
		sc, sOK := skeletonTree[p]
		switch {
		case tOK && !sOK:
			unexplained = append(unexplained, fmt.Sprintf(
				"internal/%s: only in toolsmith after substitution; no exemption covers it\n--- toolsmith (substituted), %d bytes ---\n%s",
				p, len(tc), tc))
		case !tOK && sOK:
			unexplained = append(unexplained, fmt.Sprintf(
				"assets/_skeleton/internal/%s: only in the skeleton; no exemption covers it\n--- skeleton, %d bytes ---\n%s",
				p, len(sc), sc))
		case tOK && sOK && tc != sc:
			unexplained = append(unexplained, fmt.Sprintf(
				"internal/%s vs assets/_skeleton/internal/%s: content differs after substitution and every known exemption; no exemption covers the remainder\n%s",
				p, p, divergenceReport(p, tc, sc)))
		}
	}

	if len(unexplained) > 0 {
		t.Errorf("drift gate: %d unexempted difference(s) between toolsmith's internal/ (substituted) and assets/_skeleton/internal/ [direction: a difference with no exemption]:\n\n%s",
			len(unexplained), strings.Join(unexplained, "\n\n"))
	}

	var stale []string
	for i, pe := range pathExemptions {
		if !pathExemptionUsed[i] {
			stale = append(stale, fmt.Sprintf("path exemption %q (%s): no file under internal/ matches this anymore", pe.prefix, pe.reason))
		}
	}
	for i, te := range textExemptions {
		if !textExemptionUsed[i] {
			stale = append(stale, fmt.Sprintf("text exemption in internal/%s (%s): present=%q / replacement=%q no longer both hold", te.path, te.reason, te.present, te.replacement))
		}
	}
	if len(stale) > 0 {
		t.Errorf("drift gate: %d stale exemption(s) — no matching real difference behind them anymore [direction: an exemption with no difference]:\n\n%s",
			len(stale), strings.Join(stale, "\n\n"))
	}
}

// divergenceReport renders a short, actionable report of where toolsmith's
// substituted content (a) and the skeleton's actual content (b) for relPath
// first diverge: the line number, a few lines of context from each side,
// and a copy-pasteable command that prints the full aligned diff. A naive
// line-indexed comparison (the previous version of this function) has no
// alignment — a single inserted or deleted line shifts every later line
// number, so one real edit turns into dozens of false-looking pairs that
// nobody can act on. This gives up on rendering the whole diff inline and
// instead points the reader at the one command that will.
func divergenceReport(relPath, a, b string) string {
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
		"diff -u <(sed -e 's/toolsmith/toolname/g; s/TOOLSMITH/TOOLNAME/g' internal/%s) assets/_skeleton/internal/%s",
		relPath, relPath)

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
