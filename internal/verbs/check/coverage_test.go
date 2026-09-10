package check

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// clauseIDPattern matches a bare clause ID string value ("C1.1"), used
// both to validate a string literal found in source and — as the
// leading-bullet form below — to find a clause's own heading line in
// CONTRACT.md.
var clauseIDPattern = regexp.MustCompile(`^C\d+\.\d+$`)

// scanAuditedClauseLiterals is TestAuditedClausesMatchSource's half of
// the gate: it finds every clause ID this package's own (non-test)
// source actually cites, by walking the AST — not grepping text — so
// that a clause ID mentioned only in a comment (audit.go's doc comment
// on Finding cites "C1.1", "C3.4" as prose examples) is never mistaken
// for a call site. Only two syntactic shapes count, matching how Audit
// actually reports (audit.go's find and checkManifestDoc):
//
//   - a.find("C1.2", ...)            — CallExpr, selector method "find"
//   - Finding{"C1.1", ...}           — CompositeLit, unkeyed first field
//   - Finding{Clause: "C1.1", ...}   — CompositeLit, keyed "Clause" field
func scanAuditedClauseLiterals(t *testing.T) []string {
	t.Helper()

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatalf("parsing package sources: %v", err)
	}

	found := map[string]bool{}
	record := func(lit *ast.BasicLit) {
		if lit == nil || lit.Kind != token.STRING {
			return
		}
		v, err := strconv.Unquote(lit.Value)
		if err != nil {
			return
		}
		if clauseIDPattern.MatchString(v) {
			found[v] = true
		}
	}

	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			ast.Inspect(file, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.CallExpr:
					if sel, ok := x.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "find" && len(x.Args) > 0 {
						if lit, ok := x.Args[0].(*ast.BasicLit); ok {
							record(lit)
						}
					}
				case *ast.CompositeLit:
					if ident, ok := x.Type.(*ast.Ident); ok && ident.Name == "Finding" {
						for i, elt := range x.Elts {
							switch e := elt.(type) {
							case *ast.KeyValueExpr:
								if key, ok := e.Key.(*ast.Ident); ok && key.Name == "Clause" {
									if lit, ok := e.Value.(*ast.BasicLit); ok {
										record(lit)
									}
								}
							case *ast.BasicLit:
								if i == 0 {
									record(e)
								}
							}
						}
					}
				}
				return true
			})
		}
	}

	ids := make([]string, 0, len(found))
	for id := range found {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// TestAuditedClausesMatchSource requires AuditedClauses() to name
// exactly the clause IDs this package's own source emits — no more, no
// less. It is what keeps the exported list from going stale against the
// code itself, independent of CONTRACT.md (that comparison is
// TestAuditedClausesMatchContract, below).
func TestAuditedClausesMatchSource(t *testing.T) {
	inSource := scanAuditedClauseLiterals(t)

	declared := append([]string(nil), AuditedClauses()...)
	sort.Strings(declared)

	onlyDeclared, onlySource := diffSets(declared, inSource)
	if len(onlyDeclared) > 0 || len(onlySource) > 0 {
		t.Fatalf("AuditedClauses() and the package's find/Finding call sites disagree:\n"+
			"  in AuditedClauses() but never emitted by any find/Finding call site: %v\n"+
			"  emitted by a find/Finding call site but missing from AuditedClauses(): %v",
			onlyDeclared, onlySource)
	}
}

// clauseBulletPattern matches CONTRACT.md's own top-level clause
// bullets ("- **C1.1** ..."), unindented. It deliberately does not match
// the indented sub-bullets a clause body may carry (C4.8's "  - **Splice
// targets**...") or an inline cross-reference like "(C5.1)" appearing
// mid-sentence in another clause's prose — neither starts a line with
// "- **C<digits>.<digits>**" at column zero.
var clauseBulletPattern = regexp.MustCompile(`^- \*\*(C\d+\.\d+)\*\*`)

// parseContractCheckMarks extracts the set of clause IDs whose body in
// CONTRACT.md contains a "[check]" marker anywhere — not just
// immediately after the clause number. A clause's body is everything
// from its own bullet line up to (but not including) the next top-level
// clause bullet or the next Markdown heading, so:
//
//   - a mark that sits mid-clause (C1.1's CGO_ENABLED sub-part, C6.5's
//     "GitHub Actions are SHA-pinned ... [check]" sentence buried after
//     the clause's own opening sentence) is still attributed to that
//     clause;
//   - the prose in the "## Conformance" section, which talks *about*
//     "[check]" markers in general, is never folded into C7.5's body —
//     the "## Clause history" heading between them ends C7.5's body
//     first.
func parseContractCheckMarks(data []byte) []string {
	marked := map[string]bool{}

	var curID string
	var body strings.Builder
	flush := func() {
		if curID != "" && strings.Contains(body.String(), "[check]") {
			marked[curID] = true
		}
		curID = ""
		body.Reset()
	}

	for _, line := range strings.Split(string(data), "\n") {
		if m := clauseBulletPattern.FindStringSubmatch(line); m != nil {
			flush()
			curID = m[1]
			body.WriteString(line)
			body.WriteByte('\n')
			continue
		}
		if strings.HasPrefix(line, "#") {
			flush()
			continue
		}
		if curID != "" {
			body.WriteString(line)
			body.WriteByte('\n')
		}
	}
	flush()

	ids := make([]string, 0, len(marked))
	for id := range marked {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// TestAuditedClausesMatchContract requires AuditedClauses() to name
// exactly the clause IDs CONTRACT.md marks [check] — the other half of
// the T24 gate. The two directions of drift mean different things and
// are reported separately: a mark with nothing behind it is the
// document overclaiming; an unmarked clause the checker actually audits
// is the document understating.
//
// CONTRACT.md is read repo-relative (three levels up from this
// package), not embedded — the binary does not ship the document (see
// AuditedClauses' doc comment).
func TestAuditedClausesMatchContract(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "CONTRACT.md"))
	if err != nil {
		t.Fatalf("reading CONTRACT.md: %v", err)
	}
	marked := parseContractCheckMarks(data)

	declared := append([]string(nil), AuditedClauses()...)
	sort.Strings(declared)

	onlyMarked, onlyDeclared := diffSets(marked, declared)
	if len(onlyMarked) > 0 || len(onlyDeclared) > 0 {
		t.Fatalf("CONTRACT.md's [check] marks and AuditedClauses() disagree:\n"+
			"  marked [check] in CONTRACT.md but never emitted by the checker (AuditedClauses()) — the document claims coverage that does not exist: %v\n"+
			"  emitted by the checker (AuditedClauses()) but not marked [check] in CONTRACT.md — the document understates what is verified: %v",
			onlyMarked, onlyDeclared)
	}
}

// diffSets returns the elements of a not in b (onlyA) and the elements
// of b not in a (onlyB). Both a and b must already be sorted; the
// results are sorted too.
func diffSets(a, b []string) (onlyA, onlyB []string) {
	inB := map[string]bool{}
	for _, v := range b {
		inB[v] = true
	}
	inA := map[string]bool{}
	for _, v := range a {
		inA[v] = true
	}
	for _, v := range a {
		if !inB[v] {
			onlyA = append(onlyA, v)
		}
	}
	for _, v := range b {
		if !inA[v] {
			onlyB = append(onlyB, v)
		}
	}
	return onlyA, onlyB
}
