package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

// The structural rules in docs/writing.md: where bold may appear and what each
// template requires. Length is not checked: a limit is met most cheaply by
// compressing an explanation into an allusion, which the guide forbids.

var (
	findingsDocument = regexp.MustCompile(`^docs/[^/]+-findings\.md$`)
	// The field names the templates define, which are the only bold text
	// allowed outside an open.md item's opening sentence.
	decisionFields = map[string]bool{
		"Date:": true, "Status:": true, "Decision.": true, "Support.": true,
		"Rejected.": true, "Limits.": true, "Revisit when": true,
	}
	findingsFields   = map[string]bool{"Run:": true, "Result:": true}
	findingsSections = []string{"What was run", "What we found", "What this does not show"}

	// A support item names where its fact is kept: a section of the
	// specification, a decision, a behavior, a file, or a URL.
	supportSource = regexp.MustCompile("§\\d|\\bD-\\d{3}\\b|\\bB\\d{2}\\b|https?://|`[^` ]*(/|\\.(md|go|sh|json|html))[^` ]*`")
	// Words that describe the system as it stood when a decision was made.
	historyWords = regexp.MustCompile(`(?i)\b(stays|stay|is kept|are kept|continues to|continue to|already|used to|previously|formerly|no longer|replaced|replaces)\b`)
	tombstoneWhy = regexp.MustCompile("^\\*\\*Status:\\*\\* withdrawn \\d{4}-\\d{2}-\\d{2}\\. (?s:.*)Why:\\s+`(docs/[^`]+-findings\\.md)`,\\s+\"([^\"]+)\"\\.$")
)

// structureProblems returns each structural rule src breaks, as "line: problem".
func structureProblems(doc string, src []byte) []string {
	root := goldmark.New(goldmark.WithExtensions(extension.GFM)).Parser().Parse(text.NewReader(src))
	var problems []string
	report := func(offset int, format string, args ...any) {
		line := bytes.Count(src[:offset], []byte("\n")) + 1
		problems = append(problems, fmt.Sprintf("%d: %s", line, fmt.Sprintf(format, args...)))
	}

	fields := map[string]bool{}
	switch {
	case doc == "docs/decisions.md":
		fields = decisionFields
	case findingsDocument.MatchString(doc):
		fields = findingsFields
	}
	seenFields := map[string]bool{}
	var sections []string

	ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n := n.(type) {
		case *ast.CodeSpan, *ast.CodeBlock, *ast.FencedCodeBlock, *ast.HTMLBlock, *ast.RawHTML:
			return ast.WalkSkipChildren, nil
		case *ast.Heading:
			if n.Level == 2 {
				sections = append(sections, inlineText(n, src))
			}
		case *ast.Emphasis:
			if n.Level != 2 {
				return ast.WalkContinue, nil
			}
			label := inlineText(n, src)
			switch {
			case fields[label]:
				seenFields[label] = true
			case doc == "docs/open.md" && opensTopLevelParagraph(n):
			default:
				report(firstOffset(n), "bold %q, which only a template field or an open.md item's opening sentence may be", label)
			}
			return ast.WalkSkipChildren, nil
		}
		return ast.WalkContinue, nil
	})

	if findingsDocument.MatchString(doc) {
		for f := range findingsFields {
			if !seenFields[f] {
				report(0, "no **%s** field; docs/writing.md gives the findings template", f)
			}
		}
		for _, want := range findingsSections {
			found := false
			for _, s := range sections {
				found = found || s == want
			}
			if !found {
				report(0, "no \"## %s\" section; docs/writing.md gives the findings template", want)
			}
		}
	}
	return problems
}

func opensTopLevelParagraph(n ast.Node) bool {
	p := n.Parent()
	if _, ok := p.(*ast.Paragraph); !ok || p.FirstChild() != n {
		return false
	}
	_, top := p.Parent().(*ast.Document)
	return top
}

// inlineText returns the text under n, with code spans kept as written.
func inlineText(n ast.Node, src []byte) string {
	var b strings.Builder
	ast.Walk(n, func(c ast.Node, entering bool) (ast.WalkStatus, error) {
		if t, ok := c.(*ast.Text); ok && entering {
			b.Write(t.Segment.Value(src))
			if t.SoftLineBreak() {
				b.WriteByte(' ')
			}
		}
		return ast.WalkContinue, nil
	})
	return strings.TrimSpace(b.String())
}

// firstOffset returns the source offset where n begins.
func firstOffset(n ast.Node) int {
	for c := n; c != nil; c = c.FirstChild() {
		if t, ok := c.(*ast.Text); ok {
			return t.Segment.Start
		}
		if c.Type() == ast.TypeBlock && c.Lines().Len() > 0 {
			return c.Lines().At(0).Start
		}
	}
	return 0
}

func TestStructureProblemsCatchesEachRule(t *testing.T) {
	cases := []struct {
		name string
		doc  string
		src  string
		want int
	}{
		{"bold in prose", "x.md", "Some **bold** text.\n", 1},
		{"bold label opening a paragraph", "x.md", "**Where it lives.** In a file.\n", 1},
		{"bold in a code span", "x.md", "Write `**x**` there.\n", 0},
		{"italic", "x.md", "Some *emphasis*.\n", 0},
		{"decision field", "docs/decisions.md", "**Date:** 2026-09-23 · **Status:** active\n\n**Decision.** X.\n", 0},
		{"decision field elsewhere", "x.md", "**Decision.** X.\n", 1},
		{"bold label in the log", "docs/decisions.md", "**Where it lives.** In a file.\n", 1},
		{"open.md item", "docs/open.md", "**The command fails.** It exits 1.\n", 0},
		{"open.md bold mid-paragraph", "docs/open.md", "It **fails**.\n", 1},
		{"open.md bold in a list", "docs/open.md", "- **Fails.** Yes.\n", 1},
		{"a long open.md item", "docs/open.md", "**An item.** " + strings.Repeat("Line.\n", 60), 0},
		{"complete findings", "docs/x-findings.md", "# T\n\n**Run:** a\n\n**Result:** b\n\n## What was run\n\nc\n\n## What we found\n\nd\n\n## What this does not show\n\ne\n", 0},
		{"findings with no Run", "docs/x-findings.md", "# T\n\n**Result:** b\n\n## What was run\n\nc\n\n## What we found\n\nd\n\n## What this does not show\n\ne\n", 1},
		{"findings with no limits", "docs/x-findings.md", "# T\n\n**Run:** a\n\n**Result:** b\n\n## What was run\n\nc\n\n## What we found\n\nd\n", 1},
	}
	for _, c := range cases {
		if got := structureProblems(c.doc, []byte(c.src)); len(got) != c.want {
			t.Errorf("%s: got %d problems %q, want %d", c.name, len(got), got, c.want)
		}
	}
}

// Every decision after decisionFormatAfter follows the decision template in
// docs/writing.md, and every tombstone, whatever its number, keeps only its
// status line and cites the finding that says why.
func TestLaterDecisionsFollowTemplate(t *testing.T) {
	src, err := os.ReadFile("../../docs/decisions.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range decisionProblems(string(src), findingsHeadings) {
		t.Error(p)
	}
}

// findingsHeadings returns the headings of a findings document, or false if it
// does not exist.
func findingsHeadings(rel string) (map[string]bool, bool) {
	src, err := os.ReadFile(filepath.Join("../..", rel))
	if err != nil {
		return nil, false
	}
	headings := map[string]bool{}
	for _, line := range strings.Split(string(src), "\n") {
		if strings.HasPrefix(line, "#") {
			headings[strings.TrimSpace(strings.TrimLeft(line, "#"))] = true
		}
	}
	return headings, true
}

func decisionProblems(log string, headingsOf func(string) (map[string]bool, bool)) []string {
	var problems []string
	report := func(id, format string, args ...any) {
		problems = append(problems, id+" "+fmt.Sprintf(format, args...))
	}
	for _, entry := range strings.Split(log, "\n## ")[1:] {
		title, body, _ := strings.Cut(entry, "\n")
		m := decisionTitle.FindStringSubmatch(title)
		if m == nil {
			continue
		}
		id := "D-" + m[1]
		body = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(body), "---"))

		if strings.HasPrefix(body, "**Status:** withdrawn") {
			why := tombstoneWhy.FindStringSubmatch(body)
			switch {
			case strings.Contains(body, "\n\n"):
				report(id, "is a tombstone and has more than its status line; the reason belongs in a finding")
			case why == nil:
				report(id, "is a tombstone without \"Why: `docs/<topic>-findings.md`, \\\"<section>\\\".\"")
			default:
				if headings, ok := headingsOf(why[1]); !ok {
					report(id, "cites %s, which does not exist", why[1])
				} else if section := strings.Join(strings.Fields(why[2]), " "); !headings[section] {
					report(id, "cites %q in %s, which has no such heading", section, why[1])
				}
			}
			continue
		}

		n, _ := strconv.Atoi(m[1])
		if n <= decisionFormatAfter {
			continue
		}
		for _, field := range []string{"**Date:**", "**Decision.**", "**Rejected.**", "**Revisit when**"} {
			if !strings.Contains(body, field) {
				report(id, "has no %s field; docs/writing.md gives the template", field)
			}
		}
		for _, item := range supportItems(body) {
			if !supportSource.MatchString(item) {
				report(id, "has a support item with no source: %q", firstWords(item))
			}
		}
		for _, w := range historyWords.FindAllString(body, -1) {
			report(id, "says %q, which describes the system before the decision", w)
		}
	}
	return problems
}

// supportItems returns the items listed under **Support.**, each joined onto
// one line.
func supportItems(body string) []string {
	_, rest, ok := strings.Cut(body, "**Support.**")
	if !ok {
		return nil
	}
	var items []string
	for _, line := range strings.Split(rest, "\n") {
		switch {
		case strings.HasPrefix(line, "**"):
			return items
		case strings.HasPrefix(line, "- "):
			items = append(items, strings.TrimPrefix(line, "- "))
		case strings.HasPrefix(line, "  ") && len(items) > 0:
			items[len(items)-1] += " " + strings.TrimSpace(line)
		}
	}
	return items
}

func firstWords(s string) string {
	if f := strings.Fields(s); len(f) > 8 {
		return strings.Join(f[:8], " ") + " …"
	}
	return s
}

func TestDecisionProblemsCatchesEachRule(t *testing.T) {
	headings := func(rel string) (map[string]bool, bool) {
		if rel == "docs/x-findings.md" {
			return map[string]bool{"Why it went": true}, true
		}
		return nil, false
	}
	complete := "**Date:** d\n\n**Decision.** x\n\n**Support.**\n- A fact. `docs/x-findings.md`, \"Why it went\".\n- Another, over\n  two lines. D-001 (Go).\n\n**Rejected.** y\n\n**Revisit when** z.\n"
	tombstone := "**Status:** withdrawn 2026-09-20. Replaced by D-126 (words). Why:\n`docs/x-findings.md`, \"Why it went\".\n"
	cases := []struct {
		name  string
		entry string
		want  []string
	}{
		{"before the cutoff", "## D-123 — Old\n\nAnything, already.\n", nil},
		{"complete", "## D-124 — T\n\n" + complete, nil},
		{"missing Rejected", "## D-124 — T\n\n" + strings.Replace(complete, "**Rejected.** y\n\n", "", 1), []string{"D-124 has no **Rejected.**"}},
		{"support with no source", "## D-124 — T\n\n" + strings.Replace(complete, "D-001 (Go).", "It is so.", 1), []string{"D-124 has a support item with no source"}},
		{"history word", "## D-124 — T\n\n" + strings.Replace(complete, "**Decision.** x", "**Decision.** It stays as it is.", 1), []string{"D-124 says \"stays\""}},
		{"tombstone", "## D-076 — T\n\n" + tombstone + "\n---\n", nil},
		{"tombstone with more", "## D-076 — T\n\n" + tombstone + "\nMore history.\n", []string{"D-076 is a tombstone and has more"}},
		{"tombstone with no reason", "## D-076 — T\n\n**Status:** withdrawn 2026-09-20. Replaced by D-077 (words).\n", []string{"D-076 is a tombstone without"}},
		{"tombstone citing a missing finding", "## D-076 — T\n\n" + strings.Replace(tombstone, "x-findings", "y-findings", 1), []string{"D-076 cites docs/y-findings.md, which does not exist"}},
		{"tombstone citing a missing heading", "## D-076 — T\n\n" + strings.Replace(tombstone, "Why it went", "Elsewhere", 1), []string{"D-076 cites \"Elsewhere\""}},
	}
	for _, c := range cases {
		got := decisionProblems("# Log\n\n"+c.entry, headings)
		if len(got) != len(c.want) {
			t.Errorf("%s: got %q, want %d problems", c.name, got, len(c.want))
			continue
		}
		for i, w := range c.want {
			if !strings.HasPrefix(got[i], w) {
				t.Errorf("%s: got %q, want it to start %q", c.name, got[i], w)
			}
		}
	}
}
