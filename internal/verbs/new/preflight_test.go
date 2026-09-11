// preflight_test.go drives new's git phases through RunE: a refusal
// leaves no target on disk (evidence/2026-09-11-toolsmith-conformance.md
// §7), --no-git skips git entirely, and the identity check answers for the
// new repository, not for the process's working directory. The error codes
// are pinned by instantiate_test.go's initGitRepo tests.
package new_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/procrastivity/toolsmith/internal/exitcode"
)

// gitEnv writes globalConfig to a temp file, points GIT_CONFIG_GLOBAL at
// it, sets GIT_CONFIG_NOSYSTEM=1, and unsets every environment variable
// that could otherwise supply an identity out of band — so globalConfig,
// plus whatever local repository config the caller's cwd happens to
// carry, is all that is left. t.Setenv cannot express "absent", only "set
// to this string", and an empty string is not the same thing to git:
// GIT_AUTHOR_NAME="" fails identity resolution with "empty ident name", a
// different failure than the unset case every caller here needs.
func gitEnv(t *testing.T, globalConfig string) {
	t.Helper()
	cfg := filepath.Join(t.TempDir(), "gitconfig")
	if err := os.WriteFile(cfg, []byte(globalConfig), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", cfg)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	for _, k := range []string{"GIT_AUTHOR_NAME", "GIT_AUTHOR_EMAIL", "GIT_COMMITTER_NAME", "GIT_COMMITTER_EMAIL", "EMAIL"} {
		old, had := os.LookupEnv(k)
		if err := os.Unsetenv(k); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if had {
				_ = os.Setenv(k, old)
			} else {
				_ = os.Unsetenv(k)
			}
		})
	}
}

// noIdentityEnv makes git's identity resolution deterministically
// negative, regardless of the host and regardless of cwd's own local
// config. A bare "unset everything" is not deterministic: git then falls
// back to a GECOS name and a hostname-derived email, which succeeds or
// fails depending on whether the hostname has a domain (on this host it
// does not, so git fails; a CI runner's hostname commonly does have one,
// so the same bare unset would succeed there —
// evidence/2026-09-11-toolsmith-conformance.md §7). Pointing
// GIT_CONFIG_GLOBAL at a file that sets user.useConfigOnly with no
// user.email disables that fallback outright (the probe behind ruling R2
// confirmed this). What remains is any local repository config at the
// target — new's own target is always freshly created, so unless a test
// deliberately checks the cwd-leak case, there is none.
func noIdentityEnv(t *testing.T) {
	t.Helper()
	gitEnv(t, "[user]\n\tuseConfigOnly = true\n")
}

// TestNewRefusesBeforeWriting_GitAbsent: with git unreachable, new exits 1
// (not-found.git) and writes nothing, with empty stdout.
func TestNewRefusesBeforeWriting_GitAbsent(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	t.Setenv("PATH", t.TempDir())

	stdout, _, exit := run(t, "acme", "--dir", "acme")
	if exit != exitcode.UserFail {
		t.Errorf("exit = %d, want %d (not-found.git)", exit, exitcode.UserFail)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
	if _, err := os.Stat(filepath.Join(dir, "acme")); err == nil {
		t.Error("acme exists after a git-absent preflight refusal; nothing should be written")
	}
}

// TestNewRefusesBeforeWriting_IdentityAbsent: git is present but has no
// identity in the new repository, so new exits 1 (not-found.git-identity)
// and the target it created is gone afterward.
func TestNewRefusesBeforeWriting_IdentityAbsent(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	noIdentityEnv(t)

	stdout, _, exit := run(t, "acme", "--dir", "acme")
	if exit != exitcode.UserFail {
		t.Errorf("exit = %d, want %d (not-found.git-identity)", exit, exitcode.UserFail)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
	if _, err := os.Stat(filepath.Join(dir, "acme")); err == nil {
		t.Error("acme exists after a git-identity refusal; initGitRepo must remove what it created")
	}
}

// TestNewIdentityCheck_IgnoresCwdRepo guards the false pass: the working
// directory is a repository whose local config sets an identity, and the
// target is elsewhere. That local identity never reaches the new
// repository, so new must refuse rather than fail at the commit.
func TestNewIdentityCheck_IgnoresCwdRepo(t *testing.T) {
	outer := t.TempDir()
	gitOutput(t, outer, nil, "init", "-q")
	gitOutput(t, outer, nil, "config", "user.name", "Outer Local")
	gitOutput(t, outer, nil, "config", "user.email", "outer@example.invalid")
	chdir(t, outer)

	noIdentityEnv(t)

	sibling := t.TempDir()
	target := filepath.Join(sibling, "acme")

	stdout, _, exit := run(t, "acme", "--dir", target)
	if exit != exitcode.UserFail {
		t.Errorf("exit = %d, want %d (not-found.git-identity)", exit, exitcode.UserFail)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
	if _, err := os.Stat(target); err == nil {
		t.Error("target exists; the outer repo's local identity at cwd must not leak into it")
	}
}

// TestNewIdentityCheck_IncludeIf guards the false refusal: the only
// identity comes from an `includeIf "gitdir:..."` that matches the target
// and not the working directory, so new must succeed and commit under it.
// EvalSymlinks resolves the temp root because git matches gitdir against
// the real path (macOS's /tmp is a symlink).
func TestNewIdentityCheck_IncludeIf(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	chdir(t, t.TempDir()) // cwd is deliberately outside root and outside any gitdir match

	identity := filepath.Join(root, "identity.gitconfig")
	if err := os.WriteFile(identity, []byte("[user]\n\tname = Included Name\n\temail = included@example.invalid\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	global := fmt.Sprintf("[user]\n\tuseConfigOnly = true\n[includeIf \"gitdir:%s/\"]\n\tpath = %s\n", root, identity)
	gitEnv(t, global)

	target := filepath.Join(root, "acme")
	stdout, stderr, exit := run(t, "acme", "--dir", target)
	if exit != 0 {
		t.Fatalf("exit = %d stderr = %q, want a clean run", exit, stderr)
	}
	if stdout == "" {
		t.Error("want the checklist on stdout")
	}

	subject := strings.TrimSpace(gitOutput(t, target, nil, "log", "-1", "--format=%an <%ae>"))
	want := "Included Name <included@example.invalid>"
	if subject != want {
		t.Errorf("commit identity = %q, want %q (the includeIf-scoped identity)", subject, want)
	}
}

// TestNewNoGitSkipsPreflight: with --no-git, new succeeds when git is
// absent and when it has no identity.
func TestNewNoGitSkipsPreflight(t *testing.T) {
	t.Run("git absent", func(t *testing.T) {
		dir := t.TempDir()
		chdir(t, dir)
		t.Setenv("PATH", t.TempDir())

		stdout, stderr, exit := run(t, "acme", "--dir", "acme", "--no-git")
		if exit != 0 {
			t.Fatalf("exit = %d stderr = %q, want a clean run", exit, stderr)
		}
		if stdout == "" {
			t.Error("want the checklist on stdout")
		}
		if _, err := os.Stat(filepath.Join(dir, "acme")); err != nil {
			t.Errorf("acme missing after a successful --no-git instantiation: %v", err)
		}
	})

	t.Run("identity absent", func(t *testing.T) {
		dir := t.TempDir()
		chdir(t, dir)
		noIdentityEnv(t)

		stdout, stderr, exit := run(t, "acme", "--dir", "acme", "--no-git")
		if exit != 0 {
			t.Fatalf("exit = %d stderr = %q, want a clean run", exit, stderr)
		}
		if stdout == "" {
			t.Error("want the checklist on stdout")
		}
		if _, err := os.Stat(filepath.Join(dir, "acme")); err != nil {
			t.Errorf("acme missing after a successful --no-git instantiation: %v", err)
		}
	})
}
