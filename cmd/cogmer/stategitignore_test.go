package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The state directory carries a .gitignore that ignores everything (D-266, the state
// directory carries a .gitignore that ignores everything), so a bulk add in a repository
// that holds it skips the keys and the rooms.

func TestTheStateDirectoryCarriesAGitignoreThatIgnoresEverything(t *testing.T) {
	home := filepath.Join(t.TempDir(), "state")
	t.Setenv("COGMER_HOME", home)
	if err := ensureStateDir(); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(home, ".gitignore"))
	if err != nil {
		t.Fatalf("no .gitignore in a new state directory: %v", err)
	}
	if !strings.Contains(string(got), "\n*\n") {
		t.Errorf("the .gitignore does not ignore everything:\n%s", got)
	}
	if err := ensureStateDir(); err != nil {
		t.Fatal(err)
	}
	if again, _ := os.ReadFile(filepath.Join(home, ".gitignore")); string(again) != string(got) {
		t.Error("a second call changed the .gitignore")
	}
}

func TestADirectoryFromBeforeGetsOneAndOneAUserWroteIsLeftAlone(t *testing.T) {
	home := filepath.Join(t.TempDir(), "state")
	t.Setenv("COGMER_HOME", home)
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := ensureStateDir(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".gitignore")); err != nil {
		t.Errorf("a directory made without one did not get one: %v", err)
	}

	mine := filepath.Join(t.TempDir(), "state")
	t.Setenv("COGMER_HOME", mine)
	if err := os.MkdirAll(mine, 0o700); err != nil {
		t.Fatal(err)
	}
	const written = "my own rules\n"
	if err := os.WriteFile(filepath.Join(mine, ".gitignore"), []byte(written), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ensureStateDir(); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(filepath.Join(mine, ".gitignore")); string(got) != written {
		t.Errorf("a .gitignore the user wrote was replaced by %q", got)
	}
}

// What the file is for, checked with git: a bulk add skips the directory, and adding one of
// its files by name is refused.
func TestGitSkipsTheStateDirectoryInABulkAddAndRefusesAnExplicitOne(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git on this machine")
	}
	repo := t.TempDir()
	home := filepath.Join(repo, stateDirName)
	t.Setenv("COGMER_HOME", home)
	if err := ensureStateDir(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "identity.key"), []byte("a key\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "notes.txt"), []byte("kept\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git := func(args ...string) (string, error) {
		cmd := exec.Command("git", append([]string{"-C", repo,
			"-c", "user.name=t", "-c", "user.email=t@t", "-c", "core.excludesFile=/dev/null"}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
	if out, err := git("init", "-q"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	if out, err := git("add", "-A"); err != nil {
		t.Fatalf("git add -A: %v\n%s", err, out)
	}
	staged, _ := git("diff", "--cached", "--name-only")
	if strings.TrimSpace(staged) != "notes.txt" {
		t.Errorf("a bulk add staged %q, want only notes.txt", staged)
	}
	if out, err := git("add", filepath.Join(stateDirName, "identity.key")); err == nil {
		t.Errorf("adding a key by name was not refused: %s", out)
	}
}
