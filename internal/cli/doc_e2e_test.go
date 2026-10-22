// End-to-end tests for the `doc` verb (internal/verbs/doc): a domain verb,
// same class as `check` and `new`, that the skeleton does not ship — kept
// in its own file, alongside their own dedicated test files, so it never
// collides with e2e_test.go's shared chassis coverage that the skeleton's
// copy of this package must still match (see drift/drift_test.go).
package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/procrastivity/toolsmith/internal/buildinfo"
	toolcli "github.com/procrastivity/toolsmith/internal/cli"
	"github.com/procrastivity/toolsmith/internal/iostreams"
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

type docPayload struct {
	Path    string `json:"path"`
	Source  string `json:"source"`
	Content string `json:"content"`
}

func parseDocPayload(t *testing.T, r result) docPayload {
	t.Helper()
	var payload docPayload
	if err := json.Unmarshal([]byte(r.stdout), &payload); err != nil {
		t.Fatalf("doc stdout is not one JSON value: %v; stdout=%q", err, r.stdout)
	}
	return payload
}

func projectDocPath(root string) string {
	return filepath.Join(root, ".toolsmith", "assets", "playbook", "intake.md")
}

func writeProjectDoc(t *testing.T, root, content string) string {
	t.Helper()
	path := projectDocPath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func assertDocPayload(t *testing.T, r result, wantSource, wantContent string) {
	t.Helper()
	if r.exitCode != 0 || r.stderr != "" {
		t.Fatalf("doc --json: exit=%d stderr=%q, want a clean run", r.exitCode, r.stderr)
	}
	payload := parseDocPayload(t, r)
	if payload.Path != "playbook/intake.md" || payload.Source != wantSource || payload.Content != wantContent {
		t.Fatalf("doc payload = %+v, want path=playbook/intake.md source=%q content=%q", payload, wantSource, wantContent)
	}
}

func runInProcess(t *testing.T, args ...string) result {
	t.Helper()
	var stdout, stderr bytes.Buffer
	streams := &iostreams.Streams{Out: &stdout, Err: &stderr}
	root := toolcli.NewRootCommand(streams, buildinfo.Info{})
	root.SetArgs(args)
	exitCode := toolcli.Execute(root, streams)
	return result{stdout: stdout.String(), stderr: stderr.String(), exitCode: exitCode}
}

func TestDoc_ProjectRootSelectionAndProvenance(t *testing.T) {
	rootA := t.TempDir()
	rootB := t.TempDir()
	nested := filepath.Join(rootA, "nested")
	if err := os.Mkdir(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	pathA := writeProjectDoc(t, rootA, "project A\n")
	writeProjectDoc(t, rootB, "project B\n")

	xdg := t.TempDir()
	override := filepath.Join(xdg, "toolsmith", "playbook", "intake.md")
	if err := os.MkdirAll(filepath.Dir(override), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(override, []byte("XDG fallback\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := []string{"XDG_CONFIG_HOME=" + xdg}

	// A root-position flag selects the same absolute root from the test
	// process's cwd; verbose provenance retains the resolver's canonical path.
	fromRoot := run(t, env, "--project-root", rootA, "--json", "--verbose", "doc", "playbook/intake.md")
	if fromRoot.exitCode != 0 {
		t.Fatalf("doc with project root: exit=%d stdout=%q stderr=%q", fromRoot.exitCode, fromRoot.stdout, fromRoot.stderr)
	}
	payload := parseDocPayload(t, fromRoot)
	if payload.Source != "project" || payload.Content != "project A\n" {
		t.Fatalf("doc payload = %+v, want project source and project A content", payload)
	}
	canonicalPathA, err := filepath.EvalSymlinks(pathA)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(fromRoot.stderr, "doc: serving playbook/intake.md from project at "+canonicalPathA) {
		t.Fatalf("verbose stderr = %q, want the canonical project asset path %q", fromRoot.stderr, canonicalPathA)
	}

	// An inherited flag after the subcommand selects that same root from a
	// nested cwd. A relative root is also interpreted from that invocation cwd.
	fromNested := runFrom(t, nested, env, "doc", "--project-root", rootA, "--json", "playbook/intake.md")
	assertDocPayload(t, fromNested, "project", "project A\n")
	fromRelative := runFrom(t, nested, env, "--project-root", "..", "doc", "--json", "playbook/intake.md")
	assertDocPayload(t, fromRelative, "project", "project A\n")

	// Reusing the same built executable with a different explicit root selects
	// distinct content rather than retaining the previous project's choice.
	fromB := run(t, env, "--project-root", rootB, "doc", "--json", "playbook/intake.md")
	assertDocPayload(t, fromB, "project", "project B\n")

	for _, args := range [][]string{{"--help"}, {"doc", "--help"}} {
		help := run(t, nil, args...)
		if help.exitCode != 0 || !strings.Contains(help.stdout, "--project-root") || !strings.Contains(help.stdout, "runtime asset reads") {
			t.Errorf("%v help: exit=%d stdout=%q stderr=%q, want the documented project-root flag", args, help.exitCode, help.stdout, help.stderr)
		}
	}
}

func TestDoc_ProjectRootSelectionDoesNotLeakWithinProcess(t *testing.T) {
	rootA := t.TempDir()
	rootB := t.TempDir()
	writeProjectDoc(t, rootA, "project A\n")
	writeProjectDoc(t, rootB, "project B\n")
	xdg := t.TempDir()
	override := filepath.Join(xdg, "toolsmith", "playbook", "intake.md")
	if err := os.MkdirAll(filepath.Dir(override), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(override, []byte("XDG default\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", xdg)

	for _, selection := range []struct {
		root    string
		content string
	}{{root: rootA, content: "project A\n"}, {root: rootB, content: "project B\n"}} {
		r := runInProcess(t, "--project-root", selection.root, "doc", "--json", "playbook/intake.md")
		assertDocPayload(t, r, "project", selection.content)
	}
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(workingDir); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})
	if err := os.Chdir(rootA); err != nil {
		t.Fatal(err)
	}
	// Even from inside A, omission does not implicitly select the cwd as a
	// project root or retain B from the preceding invocation.
	withoutSelection := runInProcess(t, "doc", "--json", "playbook/intake.md")
	assertDocPayload(t, withoutSelection, "override", "XDG default\n")
}

func TestDoc_ProjectRootFallbackAndNonRuntimeCommands(t *testing.T) {
	root := t.TempDir()
	xdg := t.TempDir()
	override := filepath.Join(xdg, "toolsmith", "playbook", "intake.md")
	if err := os.MkdirAll(filepath.Dir(override), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(override, []byte("XDG fallback\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := []string{"XDG_CONFIG_HOME=" + xdg}

	missingProjectFile := run(t, env, "doc", "--project-root", root, "--json", "playbook/intake.md")
	assertDocPayload(t, missingProjectFile, "override", "XDG fallback\n")

	missingRoot := filepath.Join(t.TempDir(), "not-created")
	list := run(t, nil, "--project-root", missingRoot, "doc", "--json")
	if list.exitCode != 0 {
		t.Fatalf("doc list inspected invalid project root: exit=%d stderr=%q", list.exitCode, list.stderr)
	}
	var listed struct {
		Docs []string `json:"docs"`
	}
	if err := json.Unmarshal([]byte(list.stdout), &listed); err != nil {
		t.Fatalf("doc list stdout is not JSON: %v; stdout=%q", err, list.stdout)
	}
	found := false
	for _, name := range listed.Docs {
		if name == "playbook/intake.md" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("doc list = %v, want playbook/intake.md", listed.Docs)
	}

	unsafeRoot := t.TempDir()
	unsafePath := projectDocPath(unsafeRoot)
	if err := os.MkdirAll(filepath.Dir(unsafePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(unsafePath, 0o755); err != nil {
		t.Fatal(err)
	}
	contract := run(t, nil, "--project-root", unsafeRoot, "doc", "--json", "CONTRACT.md")
	if contract.exitCode != 0 {
		t.Fatalf("doc CONTRACT.md inspected unsafe project asset: exit=%d stderr=%q", contract.exitCode, contract.stderr)
	}
	contractPayload := parseDocPayload(t, contract)
	if contractPayload.Path != "CONTRACT.md" || contractPayload.Source != "embedded" || contractPayload.Content == "" {
		t.Fatalf("doc CONTRACT.md payload = %+v, want immutable embedded contract", contractPayload)
	}

	manifest := run(t, nil, "manifest", "--json")
	if manifest.exitCode != 0 || manifest.stderr != "" {
		t.Fatalf("manifest baseline: exit=%d stderr=%q", manifest.exitCode, manifest.stderr)
	}
	for name, selectedRoot := range map[string]string{"missing": missingRoot, "unsafe": unsafeRoot} {
		withRoot := run(t, nil, "--project-root", selectedRoot, "manifest", "--json")
		if withRoot.exitCode != 0 || withRoot.stderr != "" || withRoot.stdout != manifest.stdout {
			t.Errorf("manifest with %s project root: exit=%d stderr=%q; output changed=%t", name, withRoot.exitCode, withRoot.stderr, withRoot.stdout != manifest.stdout)
		}
	}
}

func TestDoc_ProjectRootRefusalsAreStructuredAndDoNotFallBack(t *testing.T) {
	xdg := t.TempDir()
	override := filepath.Join(xdg, "toolsmith", "playbook", "intake.md")
	if err := os.MkdirAll(filepath.Dir(override), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(override, []byte("must not fall back\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := []string{"XDG_CONFIG_HOME=" + xdg}

	missingRoot := filepath.Join(t.TempDir(), "not-created")
	nonregularRoot := t.TempDir()
	nonregularPath := projectDocPath(nonregularRoot)
	if err := os.MkdirAll(filepath.Dir(nonregularPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(nonregularPath, 0o755); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name       string
		root       string
		wantDetail string
	}{
		{name: "invalid root", root: missingRoot, wantDetail: "project root"},
		{name: "nonregular project asset", root: nonregularRoot, wantDetail: "not a regular file"},
	}

	if runtime.GOOS != "windows" {
		symlinkRoot := t.TempDir()
		symlinkPath := projectDocPath(symlinkRoot)
		if err := os.MkdirAll(filepath.Dir(symlinkPath), 0o755); err != nil {
			t.Fatal(err)
		}
		outside := filepath.Join(t.TempDir(), "outside.md")
		if err := os.WriteFile(outside, []byte("outside\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, symlinkPath); err != nil {
			t.Fatalf("create project-asset symlink: %v", err)
		}
		cases = append(cases, struct {
			name       string
			root       string
			wantDetail string
		}{name: "symlink project asset", root: symlinkRoot, wantDetail: "is a symlink"})
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, jsonMode := range []bool{false, true} {
				mode := "human"
				args := []string{"--project-root", tc.root}
				if jsonMode {
					mode = "json"
					args = append(args, "--json")
				}
				args = append(args, "doc", "playbook/intake.md")
				r := run(t, env, args...)
				if r.exitCode != 3 || r.stdout != "" {
					t.Fatalf("%s refusal: exit=%d stdout=%q stderr=%q; want structured refusal 3 with no content", mode, r.exitCode, r.stdout, r.stderr)
				}
				diagnostic := r.stderr
				if jsonMode {
					envelope := parseErrorEnvelope(t, r.stderr)
					if envelope.Error.Code != "refusal.project-asset" {
						t.Fatalf("%s error code = %q, want refusal.project-asset", mode, envelope.Error.Code)
					}
					diagnostic = envelope.Error.Message
				} else if !strings.HasPrefix(r.stderr, "toolsmith: doc: ") {
					t.Fatalf("%s diagnostic = %q, want the doc error renderer", mode, r.stderr)
				}
				if !strings.Contains(diagnostic, tc.wantDetail) || strings.Contains(diagnostic, "must not fall back") {
					t.Fatalf("%s diagnostic = %q, want filesystem detail %q and no fallback content", mode, diagnostic, tc.wantDetail)
				}
			}
		})
	}
}
