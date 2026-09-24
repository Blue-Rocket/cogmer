package main

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// The rule index in docs/writing.md and the checks that cite its IDs must agree,
// so that a failing check always leads to a rule and a rule marked checked is
// always checked.

var (
	ruleIndexRow = regexp.MustCompile(`^\| (W-\d{2}) \| .+ \| (checked|judgement) \|$`)
	// An ID inside a Go string literal is one a check reports.
	citedRule = regexp.MustCompile(`"[^"\n]*\b(W-\d{2})\b[^"\n]*"`)
)

// ruleCheckFiles hold every check that reports a rule of docs/writing.md.
var ruleCheckFiles = []string{"writing_test.go", "structure_test.go", "citations_test.go"}

func TestWritingRulesAreIndexed(t *testing.T) {
	guide, err := os.ReadFile("../../docs/writing.md")
	if err != nil {
		t.Fatal(err)
	}
	indexed := map[string]string{} // ID to enforcement
	for _, line := range strings.Split(string(guide), "\n") {
		if m := ruleIndexRow.FindStringSubmatch(line); m != nil {
			if _, dup := indexed[m[1]]; dup {
				t.Errorf("docs/writing.md indexes %s twice", m[1])
			}
			indexed[m[1]] = m[2]
		}
	}
	if len(indexed) == 0 {
		t.Fatal("found no rule index in docs/writing.md")
	}
	for id := range indexed {
		if !strings.Contains(string(guide), "\n"+id+". ") {
			t.Errorf("docs/writing.md indexes %s but has no paragraph starting %q", id, id+".")
		}
	}

	cited := map[string]string{} // ID to the file that reports it
	for _, f := range ruleCheckFiles {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range citedRule.FindAllStringSubmatch(string(src), -1) {
			cited[m[1]] = f
		}
	}
	for id, f := range cited {
		switch indexed[id] {
		case "":
			t.Errorf("%s reports %s, which docs/writing.md does not index", f, id)
		case "judgement":
			t.Errorf("%s reports %s, which docs/writing.md marks judgement; mark it checked", f, id)
		}
	}
	for id, enforcement := range indexed {
		if enforcement == "checked" && cited[id] == "" {
			t.Errorf("docs/writing.md marks %s checked, but no check reports it", id)
		}
	}
}
