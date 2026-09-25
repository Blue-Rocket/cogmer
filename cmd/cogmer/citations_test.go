package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// Every D-NNN, §N and BNN cited anywhere in the repository names something that
// exists. A number allocated while writing and never followed by its entry reads
// as a source, so a citation is checked, not trusted.

var (
	decisionCitation  = regexp.MustCompile(`\bD-\d{3}\b`)
	sectionCitation   = regexp.MustCompile(`§(\d+[a-z]?(?:\.\d+)*)`)
	behaviorCitation  = regexp.MustCompile(`\bB\d{2}\b`)
	decisionHeading   = regexp.MustCompile(`(?m)^## (D-\d{3}) — `)
	specHeading       = regexp.MustCompile(`^#+ (\d+[a-z]?(?:\.\d+)*)\\?\.?(?:\s|$)`)
	specNumberedPoint = regexp.MustCompile(`^(\d+)\. `)
)

var documentCheckTests = map[string]bool{
	"cmd/cogmer/writing_test.go":   true,
	"cmd/cogmer/citations_test.go": true,
	"cmd/cogmer/structure_test.go": true,
}

// citationSources returns every file that may cite a decision, a section or
// a behavior, relative to the repository root. The test files for the writing
// and citation checks are left out, because their cases cite things that do not
// exist on purpose.
func citationSources(t *testing.T) []string {
	t.Helper()
	var files []string
	add := func(rel string) {
		if !documentCheckTests[rel] {
			files = append(files, rel)
		}
	}
	for _, pattern := range []string{"*.md", "docs/*.md", "docs/work/*.md", "cmd/cogmer/*.go", "scripts/*"} {
		matches, err := filepath.Glob(filepath.Join("../..", pattern))
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range matches {
			rel, _ := filepath.Rel("../..", m)
			add(filepath.ToSlash(rel))
		}
	}
	err := filepath.WalkDir("../../plugin", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if ext := filepath.Ext(path); !d.IsDir() && (ext == ".md" || ext == ".sh") {
			rel, _ := filepath.Rel("../..", path)
			add(filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(files)
	return files
}

// specSections returns every section the specification defines: its numbered
// headings, and the numbered points directly under a top-level section, which
// is how §36.10 names the tenth step of §36.
func specSections(spec string) map[string]bool {
	sections := map[string]bool{}
	top := ""
	inFence := false
	for _, line := range strings.Split(spec, "\n") {
		if strings.HasPrefix(line, "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if m := specHeading.FindStringSubmatch(line); m != nil {
			sections[m[1]] = true
			if !strings.Contains(m[1], ".") {
				top = m[1]
			}
			continue
		}
		if m := specNumberedPoint.FindStringSubmatch(line); m != nil && top != "" {
			sections[top+"."+m[1]] = true
		}
	}
	return sections
}

func TestCitationsNameThingsThatExist(t *testing.T) {
	read := func(rel string) string {
		data, err := os.ReadFile(filepath.Join("../..", rel))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	decisions := map[string]bool{}
	for _, m := range decisionHeading.FindAllStringSubmatch(read("docs/decisions.md"), -1) {
		decisions[m[1]] = true
	}
	sections := specSections(read("Shared Claude Sessions.md"))
	behaviors := map[string]bool{}
	for _, b := range Behaviors {
		behaviors[b.ID] = true
	}
	if len(decisions) == 0 || len(sections) == 0 || len(behaviors) == 0 {
		t.Fatalf("found %d decisions, %d sections and %d behaviors; a pattern no longer matches",
			len(decisions), len(sections), len(behaviors))
	}

	for _, rel := range citationSources(t) {
		for _, p := range citationProblems(read(rel), decisions, sections, behaviors) {
			t.Errorf("%s:%s", rel, p)
		}
	}
}

func citationProblems(src string, decisions, sections, behaviors map[string]bool) []string {
	var problems []string
	for i, line := range strings.Split(src, "\n") {
		for _, d := range decisionCitation.FindAllString(line, -1) {
			if !decisions[d] {
				problems = append(problems, fmt.Sprintf("%d: cites %s, which has no entry in docs/decisions.md (W-09)", i+1, d))
			}
		}
		for _, m := range sectionCitation.FindAllStringSubmatch(line, -1) {
			if !sections[m[1]] {
				problems = append(problems, fmt.Sprintf("%d: cites §%s, which is not a section of the specification (W-10)", i+1, m[1]))
			}
		}
		for _, b := range behaviorCitation.FindAllString(line, -1) {
			if !behaviors[b] {
				problems = append(problems, fmt.Sprintf("%d: cites %s, which is not in the behavior registry (W-09)", i+1, b))
			}
		}
	}
	return problems
}

func TestCitationProblemsCatchesEachKind(t *testing.T) {
	decisions := map[string]bool{"D-001": true}
	behaviors := map[string]bool{"B01": true}
	sections := specSections("# 3\\. Principles\n\n## 3.1 First\n\n# 36\\. Task\n\n1. One\n2. Two\n\n```\n# 99\\. Fenced\n```\n")
	for _, want := range []string{"3", "3.1", "36", "36.1", "36.2"} {
		if !sections[want] {
			t.Errorf("specSections missed §%s", want)
		}
	}
	if sections["99"] {
		t.Error("specSections read a heading inside a code block")
	}
	cases := []struct {
		src  string
		want int
	}{
		{"See D-001 (Go) and §3.1 (do no harm) and B01.", 0},
		{"See D-002.", 1},
		{"See §3.2.", 1},
		{"See §36.2 and §36.3.", 1},
		{"See B02.", 1},
		{"D-0012 and XB01 and ED-001 are not citations.", 0},
	}
	for _, c := range cases {
		if got := citationProblems(c.src, decisions, sections, behaviors); len(got) != c.want {
			t.Errorf("%q: got %d problems %q, want %d", c.src, len(got), got, c.want)
		}
	}
}
