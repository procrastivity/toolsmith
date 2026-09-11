package doc_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/procrastivity/toolsmith/internal/cliflags"
	"github.com/procrastivity/toolsmith/internal/exitcode"
	"github.com/procrastivity/toolsmith/internal/iostreams"
	"github.com/procrastivity/toolsmith/internal/toolsmitherr"
	docverb "github.com/procrastivity/toolsmith/internal/verbs/doc"
)

// run drives doc's Command in-process, the way the other verb packages'
// own tests do (internal/cli/e2e_test.go covers the same paths again
// through the real built binary). --json and -v bind here exactly as
// internal/cli.NewRootCommand binds them at root, since doc's own
// constructor — like every verb's — assumes the chassis already bound
// them once (C2.3) and never redeclares them itself.
func run(t *testing.T, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	var out, errBuf bytes.Buffer
	cmd := docverb.Command(&iostreams.Streams{Out: &out, Err: &errBuf})
	cmd.PersistentFlags().Bool("json", false, "")
	cmd.PersistentFlags().BoolP("verbose", "v", false, "")
	cmd.PersistentPreRunE = func(c *cobra.Command, _ []string) error {
		jsonOut, err := c.Flags().GetBool("json")
		if err != nil {
			return err
		}
		verbose, err := c.Flags().GetBool("verbose")
		if err != nil {
			return err
		}
		c.SetContext(cliflags.WithFlags(c.Context(), cliflags.Flags{JSON: jsonOut, Verbose: verbose}))
		return nil
	}
	cmd.SetArgs(append([]string{}, args...)) // never nil: Cobra reads os.Args on nil
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	runErr := cmd.Execute()
	if runErr == nil {
		return out.String(), errBuf.String(), 0
	}
	var terr *toolsmitherr.Error
	if errors.As(runErr, &terr) {
		// internal/cli.Execute renders the structured error; this test
		// drives doc's Command standalone, without that chassis loop, so
		// it replicates the render step here (check_test.go and
		// new_test.go's own run() helpers skip this only because neither
		// asserts on the rendered envelope the way TestNotFound_Cases
		// does).
		jsonOut, _ := cmd.Flags().GetBool("json")
		toolsmitherr.Render(&errBuf, "doc", terr, jsonOut)
		return out.String(), errBuf.String(), exitcode.FromError(terr)
	}
	return out.String(), errBuf.String(), exitcode.Usage
}

// repoRoot resolves the toolsmith repo root relative to this test file's
// own package directory (internal/verbs/doc), not the working directory
// `go test` happens to run from.
func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(wd)))
}

func TestList_ContainsExpectedNames_ExcludesNonDocs(t *testing.T) {
	stdout, stderr, code := run(t)
	if code != 0 || stderr != "" {
		t.Fatalf("exit=%d stderr=%q, want a clean run", code, stderr)
	}
	lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
	want := map[string]bool{"CONTRACT.md": false, "playbook/intake.md": false, "handoff-kit/sidecar-README.md": false}
	for _, l := range lines {
		if _, ok := want[l]; ok {
			want[l] = true
		}
		if l == "_skeleton/Makefile" || l == "agent-guidance.md" {
			t.Fatalf("list included non-doc name %q", l)
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("list missing %q; got %v", name, lines)
		}
	}
	if !sort.StringsAreSorted(lines) {
		t.Errorf("list is not sorted lexically: %v", lines)
	}
}

func TestList_JSON(t *testing.T) {
	stdout, stderr, code := run(t, "--json")
	if code != 0 || stderr != "" {
		t.Fatalf("exit=%d stderr=%q, want a clean run", code, stderr)
	}
	var payload struct {
		Docs []string `json:"docs"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("doc --json stdout is not one JSON value: %v; stdout=%q", err, stdout)
	}
	found := false
	for _, d := range payload.Docs {
		if d == "CONTRACT.md" {
			found = true
		}
	}
	if !found {
		t.Fatalf("docs = %v, want CONTRACT.md", payload.Docs)
	}
}

func TestPrint_ContractMD_ByteEqual(t *testing.T) {
	want, err := os.ReadFile(filepath.Join(repoRoot(t), "CONTRACT.md"))
	if err != nil {
		t.Fatal(err)
	}

	stdout, stderr, code := run(t, "CONTRACT.md")
	if code != 0 || stderr != "" {
		t.Fatalf("exit=%d stderr=%q, want a clean run", code, stderr)
	}
	if stdout != string(want) {
		t.Fatalf("printed CONTRACT.md differs from the repo's root copy")
	}
}

func TestPrint_PlaybookFile_ByteEqual(t *testing.T) {
	want, err := os.ReadFile(filepath.Join(repoRoot(t), "assets", "playbook", "intake.md"))
	if err != nil {
		t.Fatal(err)
	}

	stdout, stderr, code := run(t, "playbook/intake.md")
	if code != 0 || stderr != "" {
		t.Fatalf("exit=%d stderr=%q, want a clean run", code, stderr)
	}
	if stdout != string(want) {
		t.Fatalf("printed playbook/intake.md differs from assets/playbook/intake.md")
	}
}

func TestPrint_JSON(t *testing.T) {
	stdout, stderr, code := run(t, "--json", "CONTRACT.md")
	if code != 0 || stderr != "" {
		t.Fatalf("exit=%d stderr=%q, want a clean run", code, stderr)
	}
	var payload struct {
		Path    string `json:"path"`
		Source  string `json:"source"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("doc --json stdout is not one JSON value: %v; stdout=%q", err, stdout)
	}
	if payload.Path != "CONTRACT.md" {
		t.Errorf("path = %q, want CONTRACT.md", payload.Path)
	}
	if payload.Source != "embedded" {
		t.Errorf("source = %q, want embedded", payload.Source)
	}
	if !strings.Contains(payload.Content, "toolsmith contract") {
		t.Errorf("content missing expected contract text; got %d bytes", len(payload.Content))
	}
}

func TestOverride_PlaybookFile_ShadowsByName(t *testing.T) {
	xdg := t.TempDir()
	override := filepath.Join(xdg, "toolsmith", "playbook", "intake.md")
	if err := os.MkdirAll(filepath.Dir(override), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(override, []byte("overridden content\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", xdg)

	stdout, stderr, code := run(t, "--json", "playbook/intake.md")
	if code != 0 || stderr != "" {
		t.Fatalf("exit=%d stderr=%q, want a clean run", code, stderr)
	}
	var payload struct {
		Source  string `json:"source"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("doc --json stdout is not one JSON value: %v; stdout=%q", err, stdout)
	}
	if payload.Source != "override" {
		t.Errorf("source = %q, want override", payload.Source)
	}
	if payload.Content != "overridden content\n" {
		t.Errorf("content = %q, want the override's own content", payload.Content)
	}
}

func TestOverride_ContractMD_Ignored(t *testing.T) {
	xdg := t.TempDir()
	override := filepath.Join(xdg, "toolsmith", "CONTRACT.md")
	if err := os.MkdirAll(filepath.Dir(override), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(override, []byte("not the real contract\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", xdg)

	stdout, stderr, code := run(t, "--json", "CONTRACT.md")
	if code != 0 || stderr != "" {
		t.Fatalf("exit=%d stderr=%q, want a clean run", code, stderr)
	}
	var payload struct {
		Source  string `json:"source"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("doc --json stdout is not one JSON value: %v; stdout=%q", err, stdout)
	}
	if payload.Source != "embedded" {
		t.Errorf("source = %q, want embedded (the override must be ignored)", payload.Source)
	}
	if strings.Contains(payload.Content, "not the real contract") {
		t.Errorf("content came from the override; CONTRACT.md must always come from the root embed")
	}
}

func TestNotFound_Cases(t *testing.T) {
	for _, name := range []string{"nope", "_skeleton/Makefile", "agent-guidance.md", "../CONTRACT.md", "playbook/", "playbook", "/CONTRACT.md"} {
		t.Run(name, func(t *testing.T) {
			stdout, stderr, code := run(t, "--json", name)
			if code != 1 {
				t.Fatalf("exit=%d, want 1; stdout=%q stderr=%q", code, stdout, stderr)
			}
			if stdout != "" {
				t.Fatalf("stdout = %q, want empty on failure", stdout)
			}
			var envelope struct {
				Error struct {
					Code    string `json:"code"`
					Message string `json:"message"`
				} `json:"error"`
			}
			if err := json.Unmarshal([]byte(stderr), &envelope); err != nil {
				t.Fatalf("stderr is not the error envelope: %v; stderr=%q", err, stderr)
			}
			if envelope.Error.Code != "not-found.doc" {
				t.Fatalf("error code = %q, want not-found.doc", envelope.Error.Code)
			}
		})
	}
}

func TestTwoArgs_UsageError(t *testing.T) {
	_, _, code := run(t, "CONTRACT.md", "playbook/intake.md")
	if code != 2 {
		t.Fatalf("exit=%d, want 2 (usage)", code)
	}
}
