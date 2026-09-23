package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"
)

// Two tools for the writing-review skill (.claude/skills/writing-review). Both
// are skipped unless their environment variable is set, and neither fails the
// suite in normal use.

// TestReviewOneDocument runs every deterministic writing check on the
// documents named in WRITING_REVIEW, whether or not they are exempt, so a
// document can be checked while it is being rewritten:
//
//	WRITING_REVIEW=docs/open.md go test ./cmd/cogmer -run TestReviewOneDocument -count=1
func TestReviewOneDocument(t *testing.T) {
	docs := os.Getenv("WRITING_REVIEW")
	if docs == "" {
		t.Skip("set WRITING_REVIEW to a comma-separated list of documents")
	}
	for _, doc := range strings.Split(docs, ",") {
		doc = strings.TrimSpace(doc)
		src, err := os.ReadFile(filepath.Join("../..", doc))
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range append(writingProblems(doc, src), structureProblems(doc, src)...) {
			t.Errorf("%s:%s", doc, p)
		}
	}
}

// A reviewFinding is one problem the review reports. Quote is copied from the
// document's source, so that it can be checked.
type reviewFinding struct {
	File       string `json:"file"`
	Quote      string `json:"quote"`
	Rule       string `json:"rule"`
	Problem    string `json:"problem"`
	Suggestion string `json:"suggestion,omitempty"`
	Line       int    `json:"line,omitempty"`
}

// TestReviewFindingsQuoteTheirDocuments reads the findings in the JSON file
// named by WRITING_FINDINGS and keeps only those whose quote appears in the
// named file, adding its line. It writes the kept findings to the same path
// with ".verified" appended, and fails listing every finding it dropped:
//
//	WRITING_FINDINGS=/tmp/findings.json go test ./cmd/cogmer -run TestReviewFindingsQuoteTheirDocuments -count=1 -v
func TestReviewFindingsQuoteTheirDocuments(t *testing.T) {
	path := os.Getenv("WRITING_FINDINGS")
	if path == "" {
		t.Skip("set WRITING_FINDINGS to a JSON file of review findings")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var findings []reviewFinding
	if err := json.Unmarshal(data, &findings); err != nil {
		t.Fatalf("%s is not a JSON array of findings: %v", path, err)
	}
	var kept []reviewFinding
	for _, f := range findings {
		// A document is named from the repository root; a commit message saved
		// elsewhere is named by its absolute path.
		name := f.File
		if !filepath.IsAbs(name) {
			name = filepath.Join("../..", name)
		}
		src, err := os.ReadFile(name)
		if err != nil {
			t.Errorf("dropped: %s does not exist (quote %q)", f.File, f.Quote)
			continue
		}
		line := quoteLine(string(src), f.Quote)
		if line == 0 {
			t.Errorf("dropped: %s does not contain %q", f.File, f.Quote)
			continue
		}
		f.Line = line
		kept = append(kept, f)
	}
	out, err := json.MarshalIndent(kept, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path+".verified", out, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("kept %d of %d findings in %s.verified", len(kept), len(findings), path)
}

// quoteLine returns the line of src on which quote starts, or 0 if src does
// not contain it. Runs of whitespace match each other, so a quote may cross a
// line wrap; nothing else is loosened.
func quoteLine(src, quote string) int {
	q := collapseSpace(quote)
	if q.text == "" {
		return 0
	}
	s := collapseSpace(src)
	i := strings.Index(s.text, q.text)
	if i < 0 {
		return 0
	}
	return strings.Count(src[:s.offsets[i]], "\n") + 1
}

type collapsed struct {
	text    string
	offsets []int // offset in the original of each byte of text
}

func collapseSpace(s string) collapsed {
	var b strings.Builder
	var offsets []int
	space := false
	for i, r := range strings.TrimSpace(s) {
		if unicode.IsSpace(r) {
			space = true
			continue
		}
		if space && b.Len() > 0 {
			b.WriteByte(' ')
			offsets = append(offsets, i)
		}
		space = false
		start := b.Len()
		b.WriteRune(r)
		for range b.Len() - start {
			offsets = append(offsets, i)
		}
	}
	// TrimSpace moved the start; shift offsets back into the original.
	shift := len(s) - len(strings.TrimLeftFunc(s, unicode.IsSpace))
	for i := range offsets {
		offsets[i] += shift
	}
	return collapsed{b.String(), offsets}
}

func TestQuoteLineFindsQuotesAcrossWraps(t *testing.T) {
	src := "# Title\n\nThe daemon reads the\nconfig at startup.\n\nIt holds   two ports.\n"
	cases := []struct {
		quote string
		want  int
	}{
		{"The daemon reads the config", 3},
		{"the\nconfig at startup", 3},
		{"config at startup.", 4},
		{"holds two ports", 6},
		{"The daemon reads a config", 0},
		{"the daemon reads the config", 0},
		{"   ", 0},
	}
	for _, c := range cases {
		if got := quoteLine(src, c.quote); got != c.want {
			t.Errorf("quoteLine(%q) = %d, want %d", c.quote, got, c.want)
		}
	}
	if got := quoteLine("\n\n  Leading space.\n", "Leading space."); got != 3 {
		t.Errorf("a document starting with whitespace put the quote on line %d, want 3", got)
	}
}
