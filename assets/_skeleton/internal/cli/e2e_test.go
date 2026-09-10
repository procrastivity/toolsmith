// End-to-end tests for the chassis: exit codes, error shape, and the
// manifest/install/uninstall/doctor loop, exercised through the actual
// built binary rather than in-process, so Cobra's own argument-parsing
// path is covered along with the exitcode/toolnameerr machinery (C2.7).
package cli_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var binPath string

func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "toolname-e2e")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	binPath = filepath.Join(tmpDir, "toolname")

	build := exec.Command("go", "build", "-o", binPath, "github.com/procrastivity/toolname/cmd/toolname")
	if out, err := build.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "building toolname for e2e tests: %v\n%s", err, out)
		_ = os.RemoveAll(tmpDir)
		os.Exit(1)
	}

	code := m.Run()
	_ = os.RemoveAll(tmpDir)
	os.Exit(code)
}

type result struct {
	stdout   string
	stderr   string
	exitCode int
}

// hermeticEnv is the process environment plus the caller's overrides, with
// every harness's skills-dir seam defaulted to its own per-test temp dir
// when the caller does not set it — the suite must never read this host's
// real skill installs (C2.7; wip found this live: a stamped tree from an
// older build failed doctor inside tests that never mentioned skills).
// Add each new harness's seam var to this list when adding a target.
func hermeticEnv(t *testing.T, env []string) []string {
	t.Helper()
	out := append(os.Environ(), env...)
	for _, name := range []string{"TOOLNAME_CLAUDE_SKILLS_DIR"} {
		set := false
		for _, e := range env {
			if strings.HasPrefix(e, name+"=") {
				set = true
				break
			}
		}
		if !set {
			out = append(out, name+"="+t.TempDir())
		}
	}
	return out
}

func run(t *testing.T, env []string, args ...string) result {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	cmd.Env = hermeticEnv(t, env)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("running %v: %v", args, err)
		}
		exitCode = exitErr.ExitCode()
	}
	return result{stdout: stdout.String(), stderr: stderr.String(), exitCode: exitCode}
}

func TestUsageError_BadFlag(t *testing.T) {
	r := run(t, nil, "--bogus")
	if r.exitCode != 2 {
		t.Fatalf("exit code = %d, want 2 (usage error); stderr=%q", r.exitCode, r.stderr)
	}
	if r.stdout != "" {
		t.Fatalf("stdout = %q, want empty on failure", r.stdout)
	}
	if !strings.HasPrefix(r.stderr, "toolname: ") {
		t.Fatalf("stderr = %q, want it to start with %q", r.stderr, "toolname: ")
	}
}

func TestVersion_JSON(t *testing.T) {
	r := run(t, nil, "version", "--json")
	if r.exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", r.exitCode, r.stderr)
	}
	var payload struct {
		Version string `json:"version"`
		Commit  string `json:"commit"`
		Date    string `json:"date"`
	}
	if err := json.Unmarshal([]byte(r.stdout), &payload); err != nil {
		t.Fatalf("version --json stdout is not one JSON value: %v; stdout=%q", err, r.stdout)
	}
	if payload.Version == "" {
		t.Fatalf("version --json reported an empty version")
	}
}

func TestManifest_JSON_DeclaresContractAndDigest(t *testing.T) {
	r := run(t, nil, "manifest", "--json")
	if r.exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", r.exitCode, r.stderr)
	}
	var m struct {
		SchemaVersion  int    `json:"schemaVersion"`
		Contract       string `json:"contract"`
		ManifestDigest string `json:"manifest_digest"`
		Verbs          []struct {
			Name string `json:"name"`
			Kind string `json:"kind"`
		} `json:"verbs"`
	}
	if err := json.Unmarshal([]byte(r.stdout), &m); err != nil {
		t.Fatalf("manifest --json stdout is not one JSON value: %v; stdout=%q", err, r.stdout)
	}
	if m.Contract == "" {
		t.Fatalf("manifest declares no contract version (C3.6)")
	}
	if !strings.HasPrefix(m.ManifestDigest, "sha256:") {
		t.Fatalf("manifest_digest = %q, want a sha256: prefix (C3.4)", m.ManifestDigest)
	}
	if len(m.Verbs) == 0 {
		t.Fatalf("manifest lists no verbs")
	}
	for _, v := range m.Verbs {
		if v.Kind == "" {
			t.Fatalf("verb %q carries no surface kind (C3.2)", v.Name)
		}
	}
}

// TestInstallLoop drives the full install lifecycle against a hermetic
// skills dir: install writes a stamped tree, a re-install reports current,
// a hand-edit flips install to a refusal (exit 3) and doctor keeps
// advising, --force recovers, and uninstall removes exactly the tree.
func TestInstallLoop(t *testing.T) {
	skills := t.TempDir()
	env := []string{"TOOLNAME_CLAUDE_SKILLS_DIR=" + skills}

	if r := run(t, env, "install", "claude-code"); r.exitCode != 0 {
		t.Fatalf("install: exit=%d stderr=%q", r.exitCode, r.stderr)
	}
	skillDir := filepath.Join(skills, "toolname")
	for _, f := range []string{"SKILL.md", ".claude-plugin/plugin.json", ".toolname-manifest-stamp.json"} {
		if _, err := os.Stat(filepath.Join(skillDir, f)); err != nil {
			t.Fatalf("after install, %s: %v", f, err)
		}
	}

	if r := run(t, env, "install", "claude-code", "--json"); r.exitCode != 0 || !strings.Contains(r.stdout, `"status":"current"`) {
		t.Fatalf("re-install: exit=%d stdout=%q, want status current", r.exitCode, r.stdout)
	}

	if r := run(t, env, "doctor"); r.exitCode != 0 {
		t.Fatalf("doctor on a current install: exit=%d stdout=%q stderr=%q", r.exitCode, r.stdout, r.stderr)
	}

	skillMD := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillMD, []byte("hand-edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if r := run(t, env, "install", "claude-code"); r.exitCode != 3 {
		t.Fatalf("install over a hand-edited tree: exit=%d, want 3 (refusal); stderr=%q", r.exitCode, r.stderr)
	}
	if r := run(t, env, "install", "claude-code", "--force"); r.exitCode != 0 {
		t.Fatalf("install --force: exit=%d stderr=%q", r.exitCode, r.stderr)
	}

	if r := run(t, env, "uninstall", "claude-code"); r.exitCode != 0 {
		t.Fatalf("uninstall: exit=%d stderr=%q", r.exitCode, r.stderr)
	}
	if _, err := os.Stat(skillDir); !os.IsNotExist(err) {
		t.Fatalf("after uninstall, %s still exists", skillDir)
	}
}

func TestUnknownHarness_JSONEnvelope(t *testing.T) {
	r := run(t, nil, "install", "no-such-harness", "--json")
	if r.exitCode != 1 {
		t.Fatalf("exit code = %d, want 1 (validation); stderr=%q", r.exitCode, r.stderr)
	}
	if r.stdout != "" {
		t.Fatalf("stdout = %q, want empty on failure", r.stdout)
	}
	var envelope struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(r.stderr), &envelope); err != nil {
		t.Fatalf("stderr is not the {\"error\":...} envelope: %v; stderr=%q", err, r.stderr)
	}
	if envelope.Error.Code != "validation.unknown-harness" {
		t.Fatalf("error code = %q, want validation.unknown-harness", envelope.Error.Code)
	}
}
