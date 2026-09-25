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

// The structural rules in docs/writing.md: where bold may appear, where a header
// is needed, and what each template requires. No length limit is checked: a
// limit is met most cheaply by compressing an explanation into an allusion,
// which the guide forbids. The header rule is not a limit on content, since
// the fix it asks for is removing the header.

var (
	findingsDocument = regexp.MustCompile(`^docs/[^/]+-findings\.md$`)
	// Working material for an open item. It may use the findings fields, and
	// nothing durable may cite it, because it is deleted with its item (W-19).
	workDocument = regexp.MustCompile(`^docs/work/[^/]+\.md$`)
	workCitation = regexp.MustCompile(`docs/work/`)
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
	historyWords   = regexp.MustCompile(`(?i)\b(stays|stay|is kept|are kept|continues to|continue to|already|used to|previously|formerly|no longer|replaced|replaces)\b`)
	tombstoneWhy   = regexp.MustCompile("^\\*\\*Status:\\*\\* withdrawn \\d{4}-\\d{2}-\\d{2}\\. (?s:.*)Why:\\s+`(docs/[^`]+-findings\\.md)`,\\s+\"([^\"]+)\"\\.$")
	decisionStatus = regexp.MustCompile(`^\*\*Date:\*\* \d{4}-\d{2}-\d{2} · \*\*Status:\*\* (active|not built)$`)
	isoDate        = regexp.MustCompile(`\b\d{4}-\d{2}-\d{2}\b`)
	// What a pattern may not name: this project's decisions, sections of its
	// specification, its behaviours, or the project itself (W-56).
	projectName = regexp.MustCompile(`\bD-\d{3}\b|§\d|\bB\d{2}\b|(?i)\bcogmer\b`)
	// Open work a decision must not point to: the repository's list, and the
	// task URLs of the trackers a project is likely to use.
	openWork = regexp.MustCompile(`\bopen\.md\b|docs/work/|app\.clickup\.com/t/\S+|github\.com/[\w.-]+/[\w.-]+/issues/\d+|atlassian\.net/browse/\S+|linear\.app/\S+/issue/\S+`)
)

// specification is the one document that must carry no dates.
const specification = "Shared Claude Sessions.md"

// patternsDocument holds the patterns that apply beyond this project.
const patternsDocument = "docs/patterns.md"

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
	case findingsDocument.MatchString(doc), workDocument.MatchString(doc):
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
				report(firstOffset(n), "bold %q, which only a template field or an open.md item's opening sentence may be (W-14)", label)
			}
			return ast.WalkSkipChildren, nil
		case *ast.Text:
			if doc == specification {
				for _, m := range isoDate.FindAllIndex(n.Segment.Value(src), -1) {
					report(n.Segment.Start+m[0], "a date; the specification says what must be true, not when (W-81)")
				}
			}
		}
		return ast.WalkContinue, nil
	})

	// Nothing durable cites working material (W-19). The check reads each block's
	// Markdown as written, because such a citation is usually a path in backticks.
	if doc != "docs/open.md" && !workDocument.MatchString(doc) {
		ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
			if !entering {
				return ast.WalkContinue, nil
			}
			switch n.(type) {
			case *ast.FencedCodeBlock, *ast.CodeBlock, *ast.HTMLBlock:
				return ast.WalkSkipChildren, nil
			case *ast.Paragraph, *ast.TextBlock, *ast.Heading:
				if workCitation.MatchString(nodeSource(n, src)) {
					lo, _ := sourceRange(n)
					report(lo, "cites working material, which is deleted with its open item; cite a durable record instead (W-19)")
				}
				return ast.WalkSkipChildren, nil
			}
			return ast.WalkContinue, nil
		})
	}

	// A pattern names nothing in this repository (W-56), read from each block's
	// Markdown as written so a name in backticks counts.
	if doc == patternsDocument {
		ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
			if !entering {
				return ast.WalkContinue, nil
			}
			switch n.(type) {
			case *ast.Paragraph, *ast.TextBlock, *ast.Heading:
				for _, m := range projectName.FindAllString(nodeSource(n, src), -1) {
					lo, _ := sourceRange(n)
					report(lo, "a pattern names %q, part of this project; state it for any project (W-56)", m)
				}
				return ast.WalkSkipChildren, nil
			}
			return ast.WalkContinue, nil
		})
	}

	// A header over three lines or fewer is one the section does not need.
	// Headers a template defines are exempt: a decision's, a findings
	// document's sections, and each finding under "What we found".
	for c := root.FirstChild(); c != nil; c = c.NextSibling() {
		h, ok := c.(*ast.Heading)
		if !ok {
			continue
		}
		title := inlineText(h, src)
		documentTitle := h.Level == 1 && c == root.FirstChild()
		shaped := findingsDocument.MatchString(doc) || workDocument.MatchString(doc)
		if documentTitle || decisionTitle.MatchString(title) || (shaped && (h.Level == 3 || isFindingsSection(title))) {
			continue
		}
		first, last := 0, 0
		for s := c.NextSibling(); s != nil; s = s.NextSibling() {
			if _, heading := s.(*ast.Heading); heading {
				break
			}
			lo, hi := sourceRange(s)
			if lo < 0 {
				continue
			}
			if first == 0 {
				first = bytes.Count(src[:lo], []byte("\n")) + 1
			}
			last = bytes.Count(src[:hi-1], []byte("\n")) + 1
		}
		if first > 0 && last-first+1 <= 3 {
			report(firstOffset(h), "header %q over %d lines; a section this short needs no header (W-15)", title, last-first+1)
		}
	}

	if findingsDocument.MatchString(doc) {
		for f := range findingsFields {
			if !seenFields[f] {
				report(0, "no **%s** field; docs/writing.md gives the findings template (W-60)", f)
			}
		}
		for _, want := range findingsSections {
			found := false
			for _, s := range sections {
				found = found || s == want
			}
			if !found {
				report(0, "no \"## %s\" section; docs/writing.md gives the findings template (W-60)", want)
			}
		}
	}
	return problems
}

func isFindingsSection(title string) bool {
	for _, s := range findingsSections {
		if s == title {
			return true
		}
	}
	return false
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
		{"a short finding under its template heading", "docs/x-findings.md", "# T\n\n**Run:** a\n\n**Result:** b\n\n## What was run\n\nc\n\n## What we found\n\n### It failed\n\nd\n\n## What this does not show\n\ne\n", 0},
		{"a document title over a short introduction", "x.md", "# T\n\nOne line.\n", 0},
		{"header over one line", "x.md", "Intro.\n\n## T\n\nOne line.\n", 1},
		{"header over four lines", "x.md", "Intro.\n\n## T\n\na\nb\nc\nd\n", 0},
		{"header over a subsection", "x.md", "Intro.\n\n## T\n\n### U\n\na\nb\nc\nd\n", 0},
		{"header over a short list", "x.md", "Intro.\n\n## T\n\n- a\n- b\n", 1},
		{"a second level-1 header over one line", "x.md", "# T\n\na\nb\nc\nd\n\n# U\n\nOne line.\n", 1},
		{"decision heading", "docs/decisions.md", "## D-124 — T\n\n**Date:** 2026-09-23 · **Status:** active\n", 0},
		{"date in the specification", specification, "The daemon started on 2026-09-22 and\nis still running today, over\nseveral lines, with no header.\n", 1},
		{"date in a code span in the specification", specification, "Write `2026-09-22` there.\n", 0},
		{"date elsewhere", "x.md", "Run on 2026-09-22.\n", 0},
		{"a durable record citing working material", "docs/x-findings.md", "# T\n\n**Run:** a\n\n**Result:** b, from `docs/work/split.md`.\n\n## What was run\n\nc\n\n## What we found\n\nd\n\n## What this does not show\n\ne\n", 1},
		{"open.md citing working material", "docs/open.md", "**Split the log.** See `docs/work/split.md`.\n", 0},
		{"working material with the findings fields", "docs/work/split.md", "# T\n\n**Run:** a\n\n**Result:** b\n", 0},
		{"working material citing itself", "docs/work/split.md", "Also in `docs/work/other.md`.\n", 0},
		{"a general pattern", patternsDocument, "P-01. Key on stable identifiers. Names collide.\n", 0},
		{"a pattern citing a decision", patternsDocument, "P-01. Key on stable identifiers (D-017).\n", 1},
		{"a pattern naming the project", patternsDocument, "P-01. Cogmer keys on stable identifiers.\n", 1},
		{"a pattern citing a section in backticks", patternsDocument, "P-01. Fail open, as `§3.1` says.\n", 1},
		{"a decision elsewhere is fine", "x.md", "As D-017 decides.\n", 0},
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

// decisionProblems reads the log as Markdown, so that a field name inside a code
// block or quoted in a sentence is not mistaken for the field (D-124, the writing
// test reads documents through goldmark). An entry runs from its heading to the
// next heading of level 2 or above.
func decisionProblems(log string, headingsOf func(string) (map[string]bool, bool)) []string {
	src := []byte(log)
	root := goldmark.New(goldmark.WithExtensions(extension.GFM)).Parser().Parse(text.NewReader(src))
	var problems []string
	report := func(id, format string, args ...any) {
		problems = append(problems, id+" "+fmt.Sprintf(format, args...))
	}

	type entry struct {
		id     string
		number int
		body   []ast.Node
	}
	var entries []*entry
	var current *entry
	for c := root.FirstChild(); c != nil; c = c.NextSibling() {
		if h, ok := c.(*ast.Heading); ok && h.Level <= 2 {
			current = nil
			if m := decisionTitle.FindStringSubmatch(inlineText(h, src)); m != nil && h.Level == 2 {
				n, _ := strconv.Atoi(m[1])
				current = &entry{id: "D-" + m[1], number: n}
				entries = append(entries, current)
			}
			continue
		}
		if _, rule := c.(*ast.ThematicBreak); rule || current == nil {
			continue
		}
		current.body = append(current.body, c)
	}

	for _, e := range entries {
		if len(e.body) > 0 && strings.HasPrefix(nodeSource(e.body[0], src), "**Status:** withdrawn") {
			why := tombstoneWhy.FindStringSubmatch(nodeSource(e.body[0], src))
			switch {
			case len(e.body) > 1:
				report(e.id, "is a tombstone and has more than its status line; the reason belongs in a finding (W-37)")
			case why == nil:
				report(e.id, "is a tombstone without \"Why: `docs/<topic>-findings.md`, \\\"<section>\\\".\" (W-37)")
			default:
				if headings, ok := headingsOf(why[1]); !ok {
					report(e.id, "cites %s, which does not exist (W-37)", why[1])
				} else if section := strings.Join(strings.Fields(why[2]), " "); !headings[section] {
					report(e.id, "cites %q in %s, which has no such heading (W-37)", section, why[1])
				}
			}
			continue
		}
		if e.number <= decisionFormatAfter {
			continue
		}

		if len(e.body) == 0 || !decisionStatus.MatchString(nodeSource(e.body[0], src)) {
			report(e.id, "has no **Date:** line whose status is active or not built (W-30)")
		}
		next := 0
		for i, c := range e.body {
			label, ok := boldOpener(c, src)
			if !ok {
				continue
			}
			at := -1
			for j, f := range decisionOrder {
				if f.name == label {
					at = j
				}
			}
			if at < 0 {
				continue // structureProblems reports any other bold
			}
			if at < next {
				report(e.id, "has **%s** out of order (W-30)", label)
				continue
			}
			for _, f := range decisionOrder[next:at] {
				if f.required {
					report(e.id, "has no **%s** field (%s)", f.name, fieldRule(f.name))
				}
			}
			next = at + 1
			if label != "Support." {
				continue
			}
			var list *ast.List
			if i+1 < len(e.body) {
				list, _ = e.body[i+1].(*ast.List)
			}
			if list == nil {
				report(e.id, "has **Support.** with no list of facts under it (W-30)")
				continue
			}
			for item := list.FirstChild(); item != nil; item = item.NextSibling() {
				if s := nodeSource(item, src); !supportSource.MatchString(s) {
					report(e.id, "has a support item with no source: %q (W-34)", firstWords(s))
				}
			}
		}
		for _, f := range decisionOrder[next:] {
			if f.required {
				report(e.id, "has no **%s** field (%s)", f.name, fieldRule(f.name))
			}
		}
		for _, c := range e.body {
			for _, w := range historyWords.FindAllString(proseText(c, src), -1) {
				report(e.id, "says %q, which describes the system before the decision (W-33)", w)
			}
			for _, w := range openWork.FindAllString(nodeSource(c, src), -1) {
				report(e.id, "points to open work, %q, which goes stale when it is resolved; state the decision's scope instead (W-41)", w)
			}
		}
	}
	return problems
}

// decisionOrder is the order of a decision's fields after its **Date:** line.
var decisionOrder = []struct {
	name     string
	required bool
}{
	{"Decision.", true},
	{"Support.", true},
	{"Rejected.", true},
	{"Limits.", false},
	{"Revisit when", true},
}

// fieldRule names the rule a missing field breaks: W-36 for **Rejected.**, which
// every entry needs, and W-30 for the rest of the template.
func fieldRule(name string) string {
	if name == "Rejected." {
		return "W-36"
	}
	return "W-30"
}

// boldOpener returns the bold text a paragraph opens with.
func boldOpener(n ast.Node, src []byte) (string, bool) {
	p, ok := n.(*ast.Paragraph)
	if !ok {
		return "", false
	}
	e, ok := p.FirstChild().(*ast.Emphasis)
	if !ok || e.Level != 2 {
		return "", false
	}
	return inlineText(e, src), true
}

// nodeSource returns the Markdown source n covers, as written.
func nodeSource(n ast.Node, src []byte) string {
	lo, hi := sourceRange(n)
	if lo < 0 {
		return ""
	}
	return strings.TrimSpace(string(src[lo:hi]))
}

// sourceRange returns the first and last source offsets under n, or -1, -1.
func sourceRange(n ast.Node) (int, int) {
	lo, hi := -1, -1
	ast.Walk(n, func(c ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		note := func(start, stop int) {
			if lo < 0 || start < lo {
				lo = start
			}
			if stop > hi {
				hi = stop
			}
		}
		if c.Type() == ast.TypeBlock {
			for i := 0; i < c.Lines().Len(); i++ {
				note(c.Lines().At(i).Start, c.Lines().At(i).Stop)
			}
		}
		if t, ok := c.(*ast.Text); ok {
			note(t.Segment.Start, t.Segment.Stop)
		}
		return ast.WalkContinue, nil
	})
	return lo, hi
}

// proseText returns the text under n with code spans and code blocks left out.
func proseText(n ast.Node, src []byte) string {
	var b strings.Builder
	ast.Walk(n, func(c ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch c := c.(type) {
		case *ast.CodeSpan, *ast.CodeBlock, *ast.FencedCodeBlock, *ast.HTMLBlock, *ast.RawHTML:
			return ast.WalkSkipChildren, nil
		case *ast.Text:
			b.Write(c.Segment.Value(src))
			b.WriteByte(' ')
		}
		return ast.WalkContinue, nil
	})
	return b.String()
}

// relianceNotRewritten lists the behaviours whose Reliance predates the
// behaviour-registry template in docs/writing.md. It only shrinks.
var relianceNotRewritten = map[string]bool{
	"B01": true, "B02": true, "B03": true, "B04": true, "B05": true, "B06": true,
	"B07": true, "B08": true, "B09": true, "B10": true, "B11": true, "B12": true,
	"B13": true, "B14": true, "B15": true, "B16": true, "B17": true, "B18": true,
	"B19": true, "B20": true, "B21": true, "B22": true, "B23": true,
}

// A behaviour's Reliance starts with what breaks, for somebody debugging at 2am.
func TestBehaviorRelianceStartsWithWhatBreaks(t *testing.T) {
	registered := map[string]bool{}
	for _, b := range Behaviors {
		registered[b.ID] = true
		follows := strings.HasPrefix(b.Reliance, "If this changes")
		switch {
		case relianceNotRewritten[b.ID] && follows:
			t.Errorf("%s's Reliance now follows W-71: remove it from relianceNotRewritten", b.ID)
		case !relianceNotRewritten[b.ID] && !follows:
			t.Errorf("%s's Reliance must start \"If this changes\": %q (W-71)", b.ID, firstWords(b.Reliance))
		}
	}
	for id := range relianceNotRewritten {
		if !registered[id] {
			t.Errorf("relianceNotRewritten lists %s, which is not in the registry", id)
		}
	}
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
	date := "**Date:** 2026-09-23 · **Status:** active\n\n"
	support := "**Support.**\n- A fact. `docs/x-findings.md`, \"Why it went\".\n- Another, over\n  two lines. D-001 (Go).\n\n"
	complete := date + "**Decision.** x\n\n" + support + "**Rejected.** y\n\n**Revisit when** z.\n"
	tombstone := "**Status:** withdrawn 2026-09-20. Replaced by D-126 (words). Why:\n`docs/x-findings.md`, \"Why it went\".\n"
	cases := []struct {
		name  string
		entry string
		want  []string
	}{
		{"before the cutoff", "## D-123 — Old\n\nAnything, already.\n", nil},
		{"complete", "## D-124 — T\n\n" + complete, nil},
		{"complete, with Limits", "## D-124 — T\n\n" + strings.Replace(complete, "**Revisit when**", "**Limits.** Some.\n\n**Revisit when**", 1), nil},
		{"not built", "## D-124 — T\n\n" + strings.Replace(complete, "active", "not built", 1), nil},
		{"missing Rejected", "## D-124 — T\n\n" + strings.Replace(complete, "**Rejected.** y\n\n", "", 1), []string{"D-124 has no **Rejected.**"}},
		{"missing Support", "## D-124 — T\n\n" + strings.Replace(complete, support, "", 1), []string{"D-124 has no **Support.**"}},
		{"Support with no list", "## D-124 — T\n\n" + strings.Replace(complete, support, "**Support.** It is so.\n\n", 1), []string{"D-124 has **Support.** with no list"}},
		{"superseded status", "## D-124 — T\n\n" + strings.Replace(complete, "active", "superseded by D-126", 1), []string{"D-124 has no **Date:** line whose status"}},
		{"a date that is not one", "## D-124 — T\n\n" + strings.Replace(complete, "2026-09-23", "d", 1), []string{"D-124 has no **Date:** line whose status"}},
		{"out of order", "## D-124 — T\n\n" + strings.Replace(complete, "**Revisit when** z.", "**Revisit when** z.\n\n**Decision.** Again.", 1), []string{"D-124 has **Decision.** out of order"}},
		{"a field only in a code block", "## D-124 — T\n\n" + strings.Replace(complete, "**Rejected.** y\n\n", "```\n**Rejected.** y\n```\n\n", 1), []string{"D-124 has no **Rejected.**"}},
		{"support with no source", "## D-124 — T\n\n" + strings.Replace(complete, "D-001 (Go).", "It is so.", 1), []string{"D-124 has a support item with no source"}},
		{"history word", "## D-124 — T\n\n" + strings.Replace(complete, "**Decision.** x", "**Decision.** It stays as it is.", 1), []string{"D-124 says \"stays\""}},
		{"history word in code", "## D-124 — T\n\n" + strings.Replace(complete, "**Decision.** x", "**Decision.** Run `stays`.", 1), nil},
		{"Limits pointing at open.md", "## D-124 — T\n\n" + strings.Replace(complete, "**Revisit when**", "**Limits.** Three questions are open in `docs/open.md`.\n\n**Revisit when**", 1), []string{"D-124 points to open work"}},
		{"Limits pointing at a tracker task", "## D-124 — T\n\n" + strings.Replace(complete, "**Revisit when**", "**Limits.** Tracked in https://app.clickup.com/t/86abc123.\n\n**Revisit when**", 1), []string{"D-124 points to open work"}},
		{"Limits pointing at working material", "## D-124 — T\n\n" + strings.Replace(complete, "**Revisit when**", "**Limits.** Detail in `docs/work/split.md`.\n\n**Revisit when**", 1), []string{"D-124 points to open work"}},
		{"Limits stating scope", "## D-124 — T\n\n" + strings.Replace(complete, "**Revisit when**", "**Limits.** It does not decide whether stop restarts the daemon.\n\n**Revisit when**", 1), nil},
		{"open work before the cutoff", "## D-123 — Old\n\nLeft open in `docs/open.md`.\n", nil},
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
