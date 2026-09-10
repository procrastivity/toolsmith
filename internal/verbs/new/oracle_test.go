// TestOracleNew and TestOracleNewCLI prove, at capture time, that the
// goldens golden_test.go pins are what contrib/new-tool.sh itself
// produces on the same cases — the reasoning documented in
// evidence/2026-09-10-parity-goldens.md. Deleted whole, by step-36, in the
// same commit that deletes contrib/new-tool.sh and contrib/parity-check
// (Matter toolsmith-binary, Stage 7): every helper this file needs that
// golden_test.go itself has no other use for is defined here, on purpose,
// so that deletion leaves nothing orphaned.
package new_test

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func exitCodeOf(t *testing.T, err error) int {
	t.Helper()
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	t.Fatalf("running subprocess: %v", err)
	return -1
}

func readGolden(t *testing.T, rel string) string {
	t.Helper()
	data, err := os.ReadFile(goldenPath(t, rel))
	if err != nil {
		t.Fatalf("reading golden %s: %v", rel, err)
	}
	return string(data)
}

// oracleScript skips the calling test unless TOOLSMITH_GOLDEN_ORACLE=1 is
// set (this suite must never need contrib/new-tool.sh on an ordinary
// `go test ./...`, since step-36 deletes it), then resolves rel and fails
// loudly if it is missing or not executable.
func oracleScript(t *testing.T, rel string) string {
	t.Helper()
	if os.Getenv("TOOLSMITH_GOLDEN_ORACLE") != "1" {
		t.Skip("set TOOLSMITH_GOLDEN_ORACLE=1 to run this oracle-comparison test")
	}
	path := filepath.Join(repoRootDir(), rel)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("oracle %s: %v", rel, err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Fatalf("oracle %s exists but is not executable", rel)
	}
	return path
}

// TestOracleNew runs contrib/new-tool.sh for every case in newCases,
// requires its produced tree to equal the same model expectedTree builds
// for the port, its checklist bytes to equal the port's golden, and — on
// the git case — its committed tree hash to equal the port's own. A tree
// hash is content-derived, so equal hashes are independent confirmation
// that both implementations committed exactly the same tree, beyond what
// the model comparison alone shows.
func TestOracleNew(t *testing.T) {
	oracle := oracleScript(t, "contrib/new-tool.sh")
	skeletonDir := filepath.Join(repoRootDir(), "assets", "_skeleton")

	for _, tc := range newCases {
		t.Run(tc.name, func(t *testing.T) {
			parent := t.TempDir()
			env := hermeticEnv(t, nil)

			cmd := exec.Command(oracle, tc.args...)
			cmd.Dir = parent
			cmd.Env = env
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			if err := cmd.Run(); err != nil {
				t.Fatalf("oracle new-tool.sh: %v\nstdout=%s\nstderr=%s", err, stdout.String(), stderr.String())
			}
			if stderr.Len() > 0 {
				t.Fatalf("oracle wrote to stderr on a clean run: %s", stderr.String())
			}

			target := filepath.Join(parent, tc.tool)
			tree := loadTree(t, target)
			compareTrees(t, expectedTree(t, skeletonDir, tc.tool, tc.module), tree)

			gotBlob := blob(stdout.String(), 0)
			wantBlob := readGolden(t, filepath.Join("new", tc.name+".stdout"))
			if gotBlob != wantBlob {
				t.Errorf("oracle checklist differs from the port golden for %s:\n--- oracle ---\n%s\n--- golden (port) ---\n%s", tc.name, gotBlob, wantBlob)
			}

			if tc.doGit {
				assertGitFacts(t, target, env, tc.tool)

				portParent := t.TempDir()
				portEnv := hermeticEnv(t, nil)
				_, _, exit := runBin(t, portParent, portEnv, append([]string{"new"}, tc.args...)...)
				if exit != 0 {
					t.Fatalf("port run for tree-hash comparison: exit=%d", exit)
				}
				oracleHash := strings.TrimSpace(gitOutput(t, target, env, "rev-parse", "HEAD^{tree}"))
				portHash := strings.TrimSpace(gitOutput(t, filepath.Join(portParent, tc.tool), portEnv, "rev-parse", "HEAD^{tree}"))
				if oracleHash != portHash {
					t.Errorf("committed tree hash differs: oracle=%s port=%s", oracleHash, portHash)
				}
			}
		})
	}
}

// TestOracleNewCLI requires every failure shape TestGoldenNewCLI pins the
// port's own codes for to exit 1 uniformly on the oracle side, including
// the existing-target refusal the port answers with exitcode.Refusal (3)
// instead (port spec §9.3, docs/binary/parity-divergences.md D3).
func TestOracleNewCLI(t *testing.T) {
	oracle := oracleScript(t, "contrib/new-tool.sh")

	cases := []struct {
		name string
		args []string
	}{
		{"unknown flag", []string{"--bogus", "acme"}},
		{"bad name shape", []string{"Acme"}},
		{"the placeholder name", []string{"toolname"}},
		{"no name", nil},
	}
	run := func(t *testing.T, dir string, args ...string) (stdout string, exit int) {
		t.Helper()
		env := hermeticEnv(t, nil)
		cmd := exec.Command(oracle, args...)
		cmd.Dir = dir
		cmd.Env = env
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = nil
		return out.String(), exitCodeOf(t, cmd.Run())
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			stdout, exit := run(t, t.TempDir(), c.args...)
			if exit != 1 {
				t.Errorf("exit = %d, want 1", exit)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want empty", stdout)
			}
		})
	}
	t.Run("existing target", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.Mkdir(filepath.Join(dir, "taken"), 0o755); err != nil {
			t.Fatal(err)
		}
		stdout, exit := run(t, dir, "acme", "--dir", "taken")
		if exit != 1 {
			t.Errorf("exit = %d, want 1 (oracle's flat die(), unlike the port's 3)", exit)
		}
		if stdout != "" {
			t.Errorf("stdout = %q, want empty", stdout)
		}
	})
}
