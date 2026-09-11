// End-to-end tests for the `doc` verb (internal/verbs/doc): a domain verb,
// same class as `check` and `new`, that the skeleton does not ship — kept
// in its own file, alongside their own dedicated test files, so it never
// collides with e2e_test.go's shared chassis coverage that the skeleton's
// copy of this package must still match (see drift/drift_test.go).
package cli_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// repoRoot resolves the toolsmith repo root from this file's own path
// (internal/cli/doc_e2e_test.go, two levels below the root) rather than
// from the working directory `go test` happens to be run from.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not resolve this test file's own path")
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(file)))
}

// TestDoc_List asserts the doc set is exactly CONTRACT.md plus every file
// under playbook/ and handoff-kit/ — nothing else in the asset tree
// (_skeleton/, templates/, agent-guidance.md, config.default.yaml) is a
// doc.
func TestDoc_List(t *testing.T) {
	r := run(t, nil, "doc")
	if r.exitCode != 0 || r.stderr != "" {
		t.Fatalf("doc: exit=%d stderr=%q, want a clean run", r.exitCode, r.stderr)
	}
	lines := strings.Split(strings.TrimRight(r.stdout, "\n"), "\n")
	for _, want := range []string{"CONTRACT.md", "playbook/intake.md", "handoff-kit/sidecar-README.md"} {
		found := false
		for _, l := range lines {
			if l == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("doc list = %v, want it to contain %q", lines, want)
		}
	}
	for _, unwanted := range []string{"_skeleton/Makefile", "agent-guidance.md"} {
		for _, l := range lines {
			if l == unwanted {
				t.Errorf("doc list = %v, want it not to contain %q", lines, unwanted)
			}
		}
	}
}

// TestDoc_Print_ContractMD_ByteEqual asserts `doc CONTRACT.md` is the
// compiled-in root contract.go embed, byte for byte the same as the
// repo's own root CONTRACT.md, with no trailing newline added.
func TestDoc_Print_ContractMD_ByteEqual(t *testing.T) {
	want, err := os.ReadFile(filepath.Join(repoRoot(t), "CONTRACT.md"))
	if err != nil {
		t.Fatal(err)
	}

	r := run(t, nil, "doc", "CONTRACT.md")
	if r.exitCode != 0 || r.stderr != "" {
		t.Fatalf("doc CONTRACT.md: exit=%d stderr=%q, want a clean run", r.exitCode, r.stderr)
	}
	if r.stdout != string(want) {
		t.Fatalf("doc CONTRACT.md printed %d bytes, want it byte-equal to the repo's root CONTRACT.md (%d bytes)", len(r.stdout), len(want))
	}
}

// TestDoc_Print_PlaybookFile_ByteEqual asserts `doc playbook/intake.md`
// is byte-equal to assets/playbook/intake.md.
func TestDoc_Print_PlaybookFile_ByteEqual(t *testing.T) {
	want, err := os.ReadFile(filepath.Join(repoRoot(t), "assets", "playbook", "intake.md"))
	if err != nil {
		t.Fatal(err)
	}

	r := run(t, nil, "doc", "playbook/intake.md")
	if r.exitCode != 0 || r.stderr != "" {
		t.Fatalf("doc playbook/intake.md: exit=%d stderr=%q, want a clean run", r.exitCode, r.stderr)
	}
	if r.stdout != string(want) {
		t.Fatalf("doc playbook/intake.md printed %d bytes, want it byte-equal to assets/playbook/intake.md (%d bytes)", len(r.stdout), len(want))
	}
}

// TestDoc_Override_PlaybookFile_ShadowsByName asserts a user override
// under a temp XDG_CONFIG_HOME shadows a playbook doc by name and reports
// source "override" under --json.
func TestDoc_Override_PlaybookFile_ShadowsByName(t *testing.T) {
	xdg := t.TempDir()
	override := filepath.Join(xdg, "toolsmith", "playbook", "intake.md")
	if err := os.MkdirAll(filepath.Dir(override), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(override, []byte("overridden content\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	r := run(t, []string{"XDG_CONFIG_HOME=" + xdg}, "doc", "--json", "playbook/intake.md")
	if r.exitCode != 0 || r.stderr != "" {
		t.Fatalf("doc --json playbook/intake.md: exit=%d stderr=%q, want a clean run", r.exitCode, r.stderr)
	}
	var payload struct {
		Source  string `json:"source"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(r.stdout), &payload); err != nil {
		t.Fatalf("doc --json stdout is not one JSON value: %v; stdout=%q", err, r.stdout)
	}
	if payload.Source != "override" {
		t.Fatalf("source = %q, want override", payload.Source)
	}
	if payload.Content != "overridden content\n" {
		t.Fatalf("content = %q, want the override's own content", payload.Content)
	}
}

// TestDoc_Override_ContractMD_Ignored asserts an override file named
// CONTRACT.md is ignored: the contract is not tunable behavior (C5.1) and
// always comes from the root embed.
func TestDoc_Override_ContractMD_Ignored(t *testing.T) {
	xdg := t.TempDir()
	override := filepath.Join(xdg, "toolsmith", "CONTRACT.md")
	if err := os.MkdirAll(filepath.Dir(override), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(override, []byte("not the real contract\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	want, err := os.ReadFile(filepath.Join(repoRoot(t), "CONTRACT.md"))
	if err != nil {
		t.Fatal(err)
	}

	r := run(t, []string{"XDG_CONFIG_HOME=" + xdg}, "doc", "CONTRACT.md")
	if r.exitCode != 0 || r.stderr != "" {
		t.Fatalf("doc CONTRACT.md: exit=%d stderr=%q, want a clean run", r.exitCode, r.stderr)
	}
	if r.stdout != string(want) {
		t.Fatalf("doc CONTRACT.md under an XDG_CONFIG_HOME override printed something other than the root CONTRACT.md")
	}
}

// TestDoc_NotFound_Cases drives the not-found path across the cases the
// design calls out by name: an unknown name, a real asset-tree file that
// is not a doc, and a traversal or bare-directory name that never equals
// a listed name exactly. Each exits 1 with empty stdout and the
// "toolsmith: doc: ..." stderr line.
func TestDoc_NotFound_Cases(t *testing.T) {
	for _, name := range []string{"nope", "_skeleton/Makefile", "../CONTRACT.md", "playbook/", "playbook"} {
		t.Run(name, func(t *testing.T) {
			r := run(t, nil, "doc", name)
			if r.exitCode != 1 {
				t.Fatalf("doc %s: exit=%d, want 1; stdout=%q stderr=%q", name, r.exitCode, r.stdout, r.stderr)
			}
			if r.stdout != "" {
				t.Fatalf("doc %s: stdout = %q, want empty on failure", name, r.stdout)
			}
			if !strings.HasPrefix(r.stderr, "toolsmith: doc: ") {
				t.Fatalf("doc %s: stderr = %q, want it to start with %q", name, r.stderr, "toolsmith: doc: ")
			}
		})
	}
}

// TestDoc_JSON_Shapes drives the list, print and error-envelope --json
// shapes through the real binary.
func TestDoc_JSON_Shapes(t *testing.T) {
	list := run(t, nil, "doc", "--json")
	if list.exitCode != 0 {
		t.Fatalf("doc --json: exit=%d stderr=%q, want 0", list.exitCode, list.stderr)
	}
	var listPayload struct {
		Docs []string `json:"docs"`
	}
	if err := json.Unmarshal([]byte(list.stdout), &listPayload); err != nil {
		t.Fatalf("doc --json stdout is not one JSON value: %v; stdout=%q", err, list.stdout)
	}
	if len(listPayload.Docs) == 0 {
		t.Fatalf("doc --json listed no docs")
	}

	printed := run(t, nil, "doc", "--json", "CONTRACT.md")
	if printed.exitCode != 0 {
		t.Fatalf("doc --json CONTRACT.md: exit=%d stderr=%q, want 0", printed.exitCode, printed.stderr)
	}
	var printPayload struct {
		Path    string `json:"path"`
		Source  string `json:"source"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(printed.stdout), &printPayload); err != nil {
		t.Fatalf("doc --json CONTRACT.md stdout is not one JSON value: %v; stdout=%q", err, printed.stdout)
	}
	if printPayload.Path != "CONTRACT.md" || printPayload.Source != "embedded" || printPayload.Content == "" {
		t.Fatalf("doc --json CONTRACT.md payload = %+v, want path=CONTRACT.md source=embedded and non-empty content", printPayload)
	}

	notFound := run(t, nil, "doc", "--json", "nope")
	if notFound.exitCode != 1 {
		t.Fatalf("doc --json nope: exit=%d, want 1", notFound.exitCode)
	}
	if notFound.stdout != "" {
		t.Fatalf("doc --json nope: stdout = %q, want empty on failure", notFound.stdout)
	}
	envelope := parseErrorEnvelope(t, notFound.stderr)
	if envelope.Error.Code != "not-found.doc" {
		t.Fatalf("error code = %q, want not-found.doc", envelope.Error.Code)
	}
}

// TestDoc_TwoArgs_UsageError asserts more than one positional argument is
// a usage error, exit 2, following how new and check produce it.
func TestDoc_TwoArgs_UsageError(t *testing.T) {
	r := run(t, nil, "doc", "CONTRACT.md", "playbook/intake.md")
	if r.exitCode != 2 {
		t.Fatalf("doc with two args: exit=%d, want 2 (usage); stderr=%q", r.exitCode, r.stderr)
	}
}
