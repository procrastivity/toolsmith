// json_test.go covers `new --json` (CONTRACT.md C2.3): on
// success stdout carries exactly one JSON value instead of the checklist.
// It drives the compiled binary through runBin/hermeticEnv (golden_test.go)
// rather than new_test.go's in-process run helper, because --json is a
// root persistent flag (internal/cli/root.go) that in-process helper never
// binds; the compiled binary is the only harness here that reproduces
// root's PersistentPreRunE wiring the flag into context.
package new_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNewJSON_NoGit asserts the exact --no-git success payload, that it
// parses, and that the target tree was actually produced.
func TestNewJSON_NoGit(t *testing.T) {
	parent := t.TempDir()
	env := hermeticEnv(t, nil)
	stdout, stderr, exit := runBin(t, parent, env, "new", "demo", "--dir", "demo", "--no-git", "--json")
	if exit != 0 {
		t.Fatalf("exit=%d stderr=%q, want a clean run", exit, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr=%q, want empty on a clean run", stderr)
	}

	want := `{"name":"demo","dir":"demo","module":"github.com/procrastivity/demo","git":false}` + "\n"
	if stdout != want {
		t.Fatalf("stdout = %q, want %q", stdout, want)
	}

	var payload struct {
		Name   string `json:"name"`
		Dir    string `json:"dir"`
		Module string `json:"module"`
		Git    bool   `json:"git"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSuffix(stdout, "\n")), &payload); err != nil {
		t.Fatalf("new --json stdout is not one JSON value: %v; stdout=%q", err, stdout)
	}
	if payload.Name != "demo" || payload.Dir != "demo" || payload.Module != "github.com/procrastivity/demo" || payload.Git {
		t.Errorf("decoded payload = %+v, want {demo demo github.com/procrastivity/demo false}", payload)
	}

	target := filepath.Join(parent, "demo")
	for _, rel := range []string{"go.mod", "go.sum", "cmd/demo/main.go", "internal/demoerr/demoerr.go"} {
		if _, err := os.Stat(filepath.Join(target, rel)); err != nil {
			t.Errorf("expected %s in the instantiated tree: %v", rel, err)
		}
	}
	if _, err := os.Stat(filepath.Join(target, ".git")); err == nil {
		t.Errorf(".git exists under %s despite --no-git", target)
	}
}

// TestNewJSON_WithGit asserts "git":true on success and that a real
// commit exists, using hermeticEnv's explicit identity (GIT_CONFIG_GLOBAL,
// GIT_CONFIG_NOSYSTEM, and all four GIT_AUTHOR_*/GIT_COMMITTER_* vars) —
// never the host's own git identity.
func TestNewJSON_WithGit(t *testing.T) {
	parent := t.TempDir()
	env := hermeticEnv(t, nil)
	stdout, stderr, exit := runBin(t, parent, env, "new", "gitdemo", "--dir", "gitdemo", "--json")
	if exit != 0 {
		t.Fatalf("exit=%d stderr=%q, want a clean run", exit, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr=%q, want empty on a clean run", stderr)
	}

	want := `{"name":"gitdemo","dir":"gitdemo","module":"github.com/procrastivity/gitdemo","git":true}` + "\n"
	if stdout != want {
		t.Fatalf("stdout = %q, want %q", stdout, want)
	}

	var payload struct {
		Name   string `json:"name"`
		Dir    string `json:"dir"`
		Module string `json:"module"`
		Git    bool   `json:"git"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSuffix(stdout, "\n")), &payload); err != nil {
		t.Fatalf("new --json stdout is not one JSON value: %v; stdout=%q", err, stdout)
	}
	if !payload.Git {
		t.Errorf("payload.Git = false, want true")
	}

	target := filepath.Join(parent, "gitdemo")
	assertGitFacts(t, target, env, "gitdemo")
}
