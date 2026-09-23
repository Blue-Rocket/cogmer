package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

// docs/writing.md is enforced here. Documents are parsed as Markdown so that
// code spans and code blocks are skipped: the rules govern prose, and an
// identifier or a quoted template is not prose.

// writingNotYetRewritten lists the documents that have not been rewritten to
// docs/writing.md. It only shrinks. A listed document that passes fails the
// test, so that it comes off the list in the same commit that fixed it.
var writingNotYetRewritten = map[string]bool{
	"CLAUDE.md":                    true,
	"README.md":                    true,
	"Shared Claude Sessions.md":    true,
	"docs/decisions.md":            true,
	"docs/open.md":                 true,
	"docs/phase0-findings.md":      true,
	"docs/phase0a-findings.md":     true,
	"docs/phase2-experiment.md":    true,
	"docs/phase5-findings.md":      true,
	"docs/relied-on-behaviors.md":  true, // generated: rewrite behaviors.go
	"docs/spec-review.md":          true,
	"docs/what-leaves-findings.md": true,
	"plugin/README.md":             true,
}

// writingNeverChecked holds documents the word check cannot apply to.
var writingNeverChecked = map[string]string{
	"docs/writing.md": "it quotes every word and mark it bans",
}

// decisionFormatAfter is the last decision written before docs/writing.md.
// Every later decision must follow its template.
const decisionFormatAfter = 123

var (
	bannedWords = regexp.MustCompile(`(?i)\b(load-bearing|honest|honestly|not merely|leverage|leveraged|leverages|utilise|utilised|utilize|utilized|utilizes|robust|robustly|seamless|seamlessly|comprehensive|ensure|ensured|ensures|ensuring|nuanced|testament|tapestry|delve|delves|delving|crucial|crucially)\b`)
	// The guide permits some uses of these, so a use can carry an exception
	// marker, <!-- writing: <reason> -->, directly after the word.
	judgementWords = regexp.MustCompile(`(?i)\b(deliberately|exactly|precisely)\b`)
	writingMarker  = regexp.MustCompile(`(?s)^<!--\s*writing:(.*?)-->$`)
	thePoint       = regexp.MustCompile(`(?i)\bthe point\b`)
	// "the point at which" and "the point where" name a moment, not a purpose.
	thePointOfTime = regexp.MustCompile(`(?i)^\s+(at which|where)\b`)
	decoration     = regexp.MustCompile(`\b(Note|NOTE|Important|IMPORTANT):`)
	decisionTitle  = regexp.MustCompile(`^D-(\d{3}) — `)
)

// writingDocuments returns every document a person reads, relative to the
// repository root. Plugin commands are instructions to the model, not documents.
func writingDocuments(t *testing.T) []string {
	t.Helper()
	var docs []string
	for _, pattern := range []string{"*.md", "docs/*.md", "plugin/*.md"} {
		matches, err := filepath.Glob(filepath.Join("../..", pattern))
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range matches {
			rel, err := filepath.Rel("../..", m)
			if err != nil {
				t.Fatal(err)
			}
			docs = append(docs, filepath.ToSlash(rel))
		}
	}
	sort.Strings(docs)
	if len(docs) == 0 {
		t.Fatal("found no documents; the globs no longer match the repository")
	}
	return docs
}

func TestDocumentsFollowWritingGuide(t *testing.T) {
	docs := writingDocuments(t)
	present := map[string]bool{}
	for _, doc := range docs {
		present[doc] = true
		if _, skip := writingNeverChecked[doc]; skip {
			continue
		}
		src, err := os.ReadFile(filepath.Join("../..", doc))
		if err != nil {
			t.Fatal(err)
		}
		problems := writingProblems(doc, src)
		switch {
		case writingNotYetRewritten[doc] && len(problems) == 0:
			t.Errorf("%s now follows docs/writing.md: remove it from writingNotYetRewritten", doc)
		case !writingNotYetRewritten[doc]:
			for _, p := range problems {
				t.Errorf("%s:%s", doc, p)
			}
		}
	}
	for doc := range writingNotYetRewritten {
		if !present[doc] {
			t.Errorf("writingNotYetRewritten lists %s, which does not exist", doc)
		}
	}
	for doc := range writingNeverChecked {
		if !present[doc] {
			t.Errorf("writingNeverChecked lists %s, which does not exist", doc)
		}
	}
}

// writingProblems returns each banned word or mark in the prose of src, as
// "line: problem".
func writingProblems(doc string, src []byte) []string {
	md := goldmark.New(goldmark.WithExtensions(extension.GFM))
	root := md.Parser().Parse(text.NewReader(src))

	// goldmark splits a sentence across several text nodes, so the prose is
	// joined into one string before matching, with the source offset of every
	// byte. A separator stands in for each block boundary and each piece of
	// code, so that no phrase is matched across them.
	var prose []byte
	var offsets []int
	allowed := map[int]bool{} // positions of em-dashes a decision heading may carry
	separate := func() {
		prose = append(prose, 0)
		offsets = append(offsets, -1)
	}
	type marker struct {
		at, offset int // position in prose, offset in src
		reason     string
		used       bool
	}
	var markers []*marker
	allowedDashes := 0
	ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n := n.(type) {
		case *ast.RawHTML:
			var html []byte
			for i := 0; i < n.Segments.Len(); i++ {
				seg := n.Segments.At(i)
				html = append(html, seg.Value(src)...)
			}
			if m := writingMarker.FindSubmatch(html); m != nil {
				markers = append(markers, &marker{
					at: len(prose), offset: n.Segments.At(0).Start, reason: strings.TrimSpace(string(m[1]))})
			}
			separate()
			return ast.WalkSkipChildren, nil
		case *ast.CodeSpan, *ast.CodeBlock, *ast.FencedCodeBlock, *ast.HTMLBlock:
			separate()
			return ast.WalkSkipChildren, nil
		case *ast.Heading:
			separate()
			allowedDashes = 0
			// A decision heading, "## D-NNN — <title>", may carry one em-dash.
			if doc == "docs/decisions.md" && n.Level == 2 && decisionTitle.MatchString(headingText(n, src)) {
				allowedDashes = 1
			}
		case *ast.Text:
			seg := n.Segment
			for i, b := range seg.Value(src) {
				if allowedDashes > 0 && bytes.HasPrefix(src[seg.Start+i:], []byte("—")) {
					allowed[len(prose)] = true
					allowedDashes--
				}
				prose = append(prose, b)
				offsets = append(offsets, seg.Start+i)
			}
			if n.SoftLineBreak() || n.HardLineBreak() {
				prose = append(prose, ' ')
				offsets = append(offsets, seg.Stop)
			}
		default:
			if n.Type() == ast.TypeBlock {
				separate()
			}
		}
		return ast.WalkContinue, nil
	})

	var problems []string
	reportOffset := func(offset int, format string, args ...any) {
		line := bytes.Count(src[:offset], []byte("\n")) + 1
		problems = append(problems, fmt.Sprintf("%d: %s", line, fmt.Sprintf(format, args...)))
	}
	report := func(at int, format string, args ...any) {
		reportOffset(offsets[at], format, args...)
	}
	s := string(prose)
	for i, r := range s {
		switch {
		case r == '—' && !allowed[i]:
			report(i, "em-dash")
		case isDecoration(r):
			report(i, "status decoration %q", r)
		}
	}
	for _, m := range bannedWords.FindAllStringIndex(s, -1) {
		report(m[0], "%q", s[m[0]:m[1]])
	}
	for _, m := range judgementWords.FindAllStringIndex(s, -1) {
		excused := false
		for _, mk := range markers {
			if mk.at >= m[1] && strings.TrimSpace(s[m[1]:mk.at]) == "" {
				mk.used, excused = true, true
			}
		}
		if !excused {
			report(m[0], "%q, which needs rephrasing or a marker giving the reason it stays", s[m[0]:m[1]])
		}
	}
	for _, mk := range markers {
		switch {
		case mk.reason == "":
			reportOffset(mk.offset, "exception marker gives no reason")
		case !mk.used:
			reportOffset(mk.offset, "exception marker does not follow a word it can excuse")
		}
	}
	for _, m := range thePoint.FindAllStringIndex(s, -1) {
		if !thePointOfTime.MatchString(s[m[1]:]) {
			report(m[0], "%q", s[m[0]:m[1]])
		}
	}
	for _, m := range decoration.FindAllStringIndex(s, -1) {
		report(m[0], "%q", s[m[0]:m[1]])
	}
	return problems
}

func headingText(h *ast.Heading, src []byte) string {
	var b strings.Builder
	for c := h.FirstChild(); c != nil; c = c.NextSibling() {
		if t, ok := c.(*ast.Text); ok {
			b.Write(t.Segment.Value(src))
		}
	}
	return b.String()
}

// isDecoration reports check marks, crosses and emoji.
func isDecoration(r rune) bool {
	switch {
	case r >= 0x2700 && r <= 0x27BF: // dingbats: ✓ ✔ ✗ ✘ ❌
		return true
	case r >= 0x2600 && r <= 0x26FF: // miscellaneous symbols: ⚠ ☑
		return true
	case r == 0x2705 || r == 0x2B50 || r == 0x2B55:
		return true
	case r >= 0x1F300 && r <= 0x1FAFF: // pictographs and emoji
		return true
	}
	return false
}

func TestIsDecorationSeparatesMarksFromTypography(t *testing.T) {
	for _, r := range "✓✔✅❌⚠🚀" {
		if !isDecoration(r) {
			t.Errorf("%q is decoration and was not reported", r)
		}
	}
	for _, r := range "→·§×≤é" {
		if isDecoration(r) {
			t.Errorf("%q is typography and was reported", r)
		}
	}
}

// A check that cannot fail reads as protection, so each rule is shown to fire,
// and shown to stay quiet inside code.
func TestWritingProblemsCatchesEachRule(t *testing.T) {
	cases := []struct {
		name string
		doc  string
		src  string
		want int
	}{
		{"em-dash in prose", "x.md", "One idea — and another.\n", 1},
		{"em-dash in a code span", "x.md", "Write `a — b` there.\n", 0},
		{"em-dash in a fenced block", "x.md", "```\n## D-001 — t\n```\n", 0},
		{"em-dash in an indented block", "x.md", "Text.\n\n    a — b\n", 0},
		{"decision heading", "docs/decisions.md", "## D-124 — A title\n", 0},
		{"second dash in a decision heading", "docs/decisions.md", "## D-124 — A — title\n", 1},
		{"decision heading outside the log", "x.md", "## D-124 — A title\n", 1},
		{"dash in a table cell", "x.md", "| a | b |\n|---|---|\n| c — d | e |\n", 1},
		{"banned word", "x.md", "It is load-bearing.\n", 1},
		{"banned word, any case", "x.md", "Honestly, no.\n", 1},
		{"banned word in a heading", "x.md", "# Exactly this\n", 1},
		{"banned word in a link", "x.md", "See [a robust thing](http://x).\n", 1},
		{"banned word in a code span", "x.md", "Run `ensure_dir`.\n", 0},
		{"word containing a banned word", "x.md", "Dishonesty and ensurer.\n", 0},
		{"the point as purpose", "x.md", "That is the point.\n", 1},
		{"phrase across a line wrap", "x.md", "It is not\nmerely that.\n", 1},
		{"phrase across a code span", "x.md", "Say not `x` merely.\n", 0},
		{"phrase across paragraphs", "x.md", "Say not\n\nmerely.\n", 0},
		{"the point as a moment", "x.md", "The point at which it fails, the point where it stops.\n", 0},
		{"status label", "x.md", "Note: this.\n", 1},
		{"check mark", "x.md", "Done ✓\n", 1},
		{"arrow", "x.md", "a → b\n", 0},
		{"judgement word", "x.md", "Chosen deliberately.\n", 1},
		{"judgement word with a marker", "x.md", "Chosen deliberately<!-- writing: not by accident --> here.\n", 0},
		{"marker after a space", "x.md", "It is exactly <!-- writing: a measurement --> 16s.\n", 0},
		{"marker with no reason", "x.md", "Chosen deliberately<!-- writing: --> here.\n", 1},
		{"marker away from its word", "x.md", "Chosen deliberately, then kept<!-- writing: why --> here.\n", 2},
		{"marker on a banned word", "x.md", "It is load-bearing<!-- writing: why -->.\n", 2},
		{"marker in a code span", "x.md", "Chosen deliberately`<!-- writing: why -->`.\n", 1},
		{"other comment", "x.md", "Chosen deliberately<!-- todo -->.\n", 1},
	}
	for _, c := range cases {
		got := writingProblems(c.doc, []byte(c.src))
		if len(got) != c.want {
			t.Errorf("%s: got %d problems %q, want %d", c.name, len(got), got, c.want)
		}
	}
}

func TestWritingProblemsReportsTheLine(t *testing.T) {
	got := writingProblems("x.md", []byte("Fine.\n\nAlso fine.\nNot — fine.\n"))
	if len(got) != 1 || !strings.HasPrefix(got[0], "4: ") {
		t.Errorf("got %q, want one problem on line 4", got)
	}
}

// Every decision after decisionFormatAfter follows the decision template in
// docs/writing.md. A tombstone, marked "**Status:** withdrawn", keeps only its
// number and title.
func TestLaterDecisionsFollowTemplate(t *testing.T) {
	src, err := os.ReadFile("../../docs/decisions.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range decisionTemplateProblems(string(src)) {
		t.Error(p)
	}
}

func decisionTemplateProblems(log string) []string {
	var problems []string
	for _, entry := range strings.Split(log, "\n## ")[1:] {
		title, body, _ := strings.Cut(entry, "\n")
		m := decisionTitle.FindStringSubmatch(title)
		if m == nil {
			continue
		}
		n, _ := strconv.Atoi(m[1])
		if n <= decisionFormatAfter || strings.Contains(body, "**Status:** withdrawn") {
			continue
		}
		for _, field := range []string{"**Date:**", "**Decision.**", "**Rejected.**", "**Revisit when**"} {
			if !strings.Contains(body, field) {
				problems = append(problems, fmt.Sprintf(
					"D-%s has no %s field; docs/writing.md gives the template", m[1], field))
			}
		}
	}
	return problems
}

func TestDecisionTemplateProblemsCatchesMissingFields(t *testing.T) {
	log := "# Log\n\n" +
		"## D-123 — Before the cutoff\n\nAnything.\n\n" +
		"## D-124 — Complete\n\n**Date:** d\n\n**Decision.** x\n\n**Rejected.** y\n\n**Revisit when** z.\n\n" +
		"## D-125 — Withdrawn\n\n**Status:** withdrawn 2026-09-23. Replaced by D-126.\n\n" +
		"## D-126 — No alternatives\n\n**Date:** d\n\n**Decision.** x\n\n**Revisit when** z.\n"
	got := decisionTemplateProblems(log)
	if len(got) != 1 || !strings.HasPrefix(got[0], "D-126 has no **Rejected.**") {
		t.Errorf("got %q, want one problem naming D-126's missing Rejected field", got)
	}
}
