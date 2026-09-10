// TestOracleCheck proves, at capture time, that the goldens golden_test.go
// pins are what contrib/check-contract itself produces on the same
// fixtures — the reasoning documented in
// evidence/2026-09-10-parity-goldens.md. Deleted whole, by step-36, in the
// same commit that deletes contrib/check-contract and contrib/parity-check
// (Matter toolsmith-binary, Stage 7): every helper this file needs that
// golden_test.go itself has no other use for is defined here, on purpose,
// so that deletion leaves nothing orphaned.
package check_test

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
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
// set (this suite must never need contrib/check-contract on an ordinary
// `go test ./...`, since step-36 deletes it), then resolves rel and fails
// loudly if it is missing or not executable — a silent skip on a missing
// oracle would be indistinguishable from a passing proof.
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

func runOracleExit(t *testing.T, oracle string, env []string, args ...string) int {
	t.Helper()
	cmd := exec.Command(oracle, args...)
	cmd.Env = env
	err := cmd.Run()
	return exitCodeOf(t, err)
}

// TestOracleCheck runs contrib/check-contract on the exact same fixtures
// and env as TestGoldenCheck and requires its output to equal the port's
// golden — except the two cases port spec §9.1/§9.7 name as deliberate
// divergences (docs/binary/parity-divergences.md D1, D2), which must
// differ, and whose oracle output is logged rather than asserted.
func TestOracleCheck(t *testing.T) {
	oracle := oracleScript(t, "contrib/check-contract")

	for _, tc := range checkCases(t) {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			tc.setup(t, dir)
			target := dir
			if tc.targetRel != "" {
				target = filepath.Join(dir, tc.targetRel)
			}
			env, _ := envForCase(t, tc)

			cmd := exec.Command(oracle, target)
			cmd.Env = env
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			exit := exitCodeOf(t, cmd.Run())

			gotBlob := blob(normalizeRepo(t, stdout.String(), target), exit)
			wantBlob := readGolden(t, filepath.Join("check", tc.name+".stdout"))

			switch tc.name {
			case "probe-prettyjson", "probe-crlf":
				if gotBlob == wantBlob {
					t.Errorf("oracle output unexpectedly matches the port golden for %s; the documented divergence did not reproduce (docs/binary/parity-divergences.md)", tc.name)
				}
				t.Logf("oracle output for %s (expected to diverge from the port, port spec §9.1/§9.7):\n%s", tc.name, gotBlob)
			default:
				if gotBlob != wantBlob {
					t.Errorf("oracle blob differs from the port golden for %s:\n--- oracle ---\n%s\n--- golden (port) ---\n%s", tc.name, gotBlob, wantBlob)
				}
				gotStderr := normalizeRepo(t, stderr.String(), target)
				wantStderr := readGolden(t, filepath.Join("check", tc.name+".stderr"))
				if gotStderr != wantStderr {
					t.Errorf("oracle stderr differs from the port golden for %s:\n--- oracle ---\n%s\n--- golden (port) ---\n%s", tc.name, gotStderr, wantStderr)
				}
			}
		})
	}

	t.Run("cli", func(t *testing.T) {
		dir := t.TempDir()
		f := filepath.Join(dir, "not-a-dir")
		if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		env := hermeticEnv(t, nil)

		if exit := runOracleExit(t, oracle, env, f); exit != 2 {
			t.Errorf("file argument: oracle exit = %d, want 2", exit)
		}
		if exit := runOracleExit(t, oracle, env, "a", "b"); exit != 2 {
			t.Errorf("two positional arguments: oracle exit = %d, want 2", exit)
		}
		if exit := runOracleExit(t, oracle, env); exit != 2 {
			t.Errorf("zero arguments: oracle exit = %d, want 2 (port spec D4: the port diverges to exit 0)", exit)
		}
	})
}
