package main

import (
	"strings"
	"testing"
)

func TestVersionReportSaysWhatResolvesADisagreement(t *testing.T) {
	cases := []struct {
		name                   string
		plugin, binary, daemon string
		answering              bool
		want, notWant          string
	}{
		{"all agree", "0.7.4", "0.7.4", "0.7.4", true, "daemon  0.7.4", "A new session"},
		{"binary behind the plugin", "0.7.4", "0.7.3", "0.7.3", true, "fetches it", "replaces it"},
		{"daemon behind the binary", "0.7.4", "0.7.4", "0.7.3", true, "replaces it", "fetches it"},
		{"daemon from before versions", "0.7.4", "0.7.4", "", true, "from before daemons reported a version", "fetches it"},
		{"no daemon", "0.7.4", "0.7.4", "", false, "starts one", "replaces it"},
		{"run from a terminal", "", "0.7.4", "0.7.4", true, "binary  0.7.4", "plugin"},
		{"a dev binary chosen by hand", "0.7.4", "dev", "dev", true, "binary  dev", "fetches it"},
		// An old copy of the plugin, such as one a desktop app installed, beside a newer program.
		{"plugin behind the binary", "0.7.4", "0.10.2", "0.10.2", true, "remove the older copy", "fetches it"},
		{"plugin behind, numbers not letters", "0.9.1", "0.10.0", "0.10.0", true, "older than the binary", "fetches it"},
		{"binary behind, numbers not letters", "0.10.0", "0.9.1", "0.9.1", true, "fetches it", "remove the older copy"},
	}
	for _, c := range cases {
		got := versionReport(c.plugin, c.binary, c.daemon, c.answering)
		if !strings.Contains(got, c.want) || strings.Contains(got, c.notWant) {
			t.Errorf("%s: report\n%s\nshould contain %q and not %q", c.name, got, c.want, c.notWant)
		}
	}
}
