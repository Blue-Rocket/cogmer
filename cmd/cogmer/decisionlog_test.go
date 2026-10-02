package main

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// The decision log is the directory docs/decisions/, one file per decision.
// Every check of decisions reads it through these helpers, so that a change to
// the layout is made here.

// decisionFile is the path of a decision's file, relative to the repository root.
var decisionFile = regexp.MustCompile(`^docs/decisions/(D-\d{3})-([a-z0-9-]+)\.md$`)

// decisionFiles returns every decision file, relative to the repository root, in
// order of number.
func decisionFiles(t *testing.T) []string {
	t.Helper()
	matches, err := filepath.Glob("../../docs/decisions/D-*.md")
	if err != nil {
		t.Fatal(err)
	}
	var files []string
	for _, m := range matches {
		rel, err := filepath.Rel("../..", m)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, filepath.ToSlash(rel))
	}
	sort.Strings(files)
	if len(files) == 0 {
		t.Fatal("found no decision files; docs/decisions/ no longer holds them")
	}
	return files
}

// readDecisionLog returns every decision, in order of number, as one Markdown
// document, so that a check can read the whole log as it reads any document.
func readDecisionLog(t *testing.T) string {
	t.Helper()
	var b strings.Builder
	for _, f := range decisionFiles(t) {
		src, err := os.ReadFile(filepath.Join("../..", f))
		if err != nil {
			t.Fatal(err)
		}
		b.Write(src)
		b.WriteString("\n")
	}
	return b.String()
}

// decisionID returns the number a decision file's name carries.
func decisionID(doc string) (string, bool) {
	m := decisionFile.FindStringSubmatch(doc)
	if m == nil {
		return "", false
	}
	return m[1], true
}

// decisionAreas returns the areas docs/decisions/areas.md lists, one in each row
// of its table, so that the list is held in one place.
func decisionAreas(t *testing.T) map[string]bool {
	t.Helper()
	src, err := os.ReadFile("../../docs/decisions/areas.md")
	if err != nil {
		t.Fatal(err)
	}
	areas, problems := parseAreas(string(src))
	for _, p := range problems {
		t.Error(p)
	}
	if len(areas) == 0 {
		t.Fatal("docs/decisions/areas.md lists no areas")
	}
	return areas
}

var areaRow = regexp.MustCompile(`(?m)^\| ([a-z-]+) \|(.*)$`)

// parseAreas reads the table of areas, and reports a row that says nothing about
// what its area covers.
func parseAreas(src string) (map[string]bool, []string) {
	areas := map[string]bool{}
	var problems []string
	for _, m := range areaRow.FindAllStringSubmatch(src, -1) {
		areas[m[1]] = true
		if strings.Trim(m[2], "| \t") == "" {
			problems = append(problems, "docs/decisions/areas.md lists "+m[1]+" with no description of what it covers")
		}
	}
	return areas, problems
}

func TestAreasEachSayWhatTheyCover(t *testing.T) {
	cases := []struct {
		name     string
		src      string
		wantArea string
		problems int
	}{
		{"described", "| Area | What |\n|---|---|\n| rooms | what a room is |\n", "rooms", 0},
		{"no description", "| rooms | |\n", "rooms", 1},
		{"no second column", "| rooms |\n", "rooms", 1},
	}
	for _, c := range cases {
		areas, problems := parseAreas(c.src)
		if !areas[c.wantArea] || len(problems) != c.problems {
			t.Errorf("%s: got areas %v and problems %q", c.name, areas, problems)
		}
	}
}

var slugBreak = regexp.MustCompile(`[^a-z0-9]+`)

// decisionSlug is the part of a decision file's name that follows its number: the
// title in lower case, without apostrophes and backticks, with each run of other
// marks as one hyphen, cut to 60 characters at a hyphen.
func decisionSlug(title string) string {
	s := strings.ToLower(title)
	s = strings.NewReplacer("`", "", "'", "", "’", "").Replace(s)
	s = strings.Trim(slugBreak.ReplaceAllString(s, "-"), "-")
	if len(s) > 60 {
		s = s[:60]
		if i := strings.LastIndex(s, "-"); i >= 0 {
			s = s[:i]
		}
	}
	return s
}

// Each decision file is named for the number and the title its heading carries,
// and holds one decision.
func TestDecisionFilesAreNamedForTheirHeadings(t *testing.T) {
	heading := regexp.MustCompile(`(?m)^# (D-\d{3}) — (.+)$`)
	seen := map[string]string{}
	for _, f := range decisionFiles(t) {
		src, err := os.ReadFile(filepath.Join("../..", f))
		if err != nil {
			t.Fatal(err)
		}
		ids := heading.FindAllStringSubmatch(string(src), -1)
		if len(ids) != 1 {
			t.Errorf("%s holds %d decision headings, and a file holds one", f, len(ids))
			continue
		}
		id, ok := decisionID(f)
		if !ok {
			t.Errorf("%s is not named D-NNN-<slug>.md", f)
			continue
		}
		if ids[0][1] != id {
			t.Errorf("%s holds the heading of %s", f, ids[0][1])
		}
		if want := "docs/decisions/" + id + "-" + decisionSlug(ids[0][2]) + ".md"; f != want {
			t.Errorf("%s should be named %s, for its title", f, want)
		}
		if other, dup := seen[id]; dup {
			t.Errorf("%s and %s both hold %s", f, other, id)
		}
		seen[id] = f
	}
}
