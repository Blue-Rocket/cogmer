package main

import (
	"os/exec"
	"strings"
	"testing"
)

// Tests that compare versions run copies of one build, each stamped with the version the
// test asked for (D-293). A copy that reported the stamp, or the stamp's padding, would
// make every upgrade test compare the wrong strings and pass or fail for that reason alone.
func TestACopyOfTheBuildReportsTheVersionStampedIntoIt(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a binary")
	}
	dir := t.TempDir()
	for _, v := range []string{"0.0.1", "0.11.0-stamped", testBuildVersion} {
		out, err := exec.Command(buildVersion(t, dir, v), "version").Output()
		if err != nil {
			t.Fatalf("version %s: %v", v, err)
		}
		if got := strings.TrimSpace(string(out)); got != v {
			t.Errorf("a copy stamped %q reports %q", v, got)
		}
	}
}

func TestAVersionTheStampCannotHoldIsRefused(t *testing.T) {
	for _, v := range []string{strings.Repeat("9", len(stampedVersion)+1), "0.1" + versionPad} {
		if err := stamp("/nonexistent", t.TempDir()+"/cogmer", v); err == nil {
			t.Errorf("stamping %q was accepted", v)
		}
	}
}
