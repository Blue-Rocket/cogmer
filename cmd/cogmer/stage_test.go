package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// A release is installed and tested from the staging repository before it reaches
// anybody (D-292). The plugin a tester installs from there must download from the
// staging repository's releases, or the test downloads nothing and proves nothing, and
// everything else in it must be the release commit's, or the test is of something
// other than what is released. scripts/publish.sh reads where to publish from the
// commit, so the same check says each commit publishes where its plugin downloads.
func TestTheStagingCommitChangesOnlyWhereThePluginDownloads(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("no bash")
	}
	if err := exec.Command("git", "-C", "../..", "rev-parse", "--verify", "HEAD").Run(); err != nil {
		t.Skip("not a git checkout with a commit")
	}
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = "../.."
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.invalid",
			"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.invalid")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s: %v\n%s", strings.Join(args, " "), err, out)
		}
		return strings.TrimSpace(string(out))
	}

	staged := run("scripts/stage.sh", "--commit-only", "HEAD")
	if got := run("git", "rev-parse", staged+"^"); got != run("git", "rev-parse", "HEAD") {
		t.Errorf("the staging commit's parent is %s, not the release commit", got)
	}
	if changed := run("git", "diff", "--name-only", "HEAD", staged); changed != "plugin/release-url.txt" {
		t.Errorf("the staging commit changes %q; it must change plugin/release-url.txt alone", changed)
	}
	if got, want := run("scripts/publish.sh", "--where", staged), "Blue-Rocket/cogmer-staging "; !strings.HasPrefix(got, want) {
		t.Errorf("publish.sh would publish the staging commit to %q, not the staging repository", got)
	}
	if got, want := run("scripts/publish.sh", "--where", "HEAD"), "Blue-Rocket/cogmer "; !strings.HasPrefix(got, want) {
		t.Errorf("publish.sh would publish the release commit to %q, not Blue-Rocket/cogmer", got)
	}
}
