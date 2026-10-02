package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
	// Working material for an open item. It may use the fields a run is recorded
	// with, and	// nothing durable may cite it, because it is deleted with its item (W-19).
	workDocument = regexp.MustCompile(`^docs/work/[^/]+\.md$`)
	workCitation = regexp.MustCompile(`docs/work/`)
	// The field names the templates define, which are the only bold text
	// allowed outside an open.md item's opening sentence.
	decisionFields = map[string]bool{
		"Date:": true, "Status:": true, "Decision.": true, "Support.": true,
		"Areas:": true, "Rejected.": true, "Limits.": true, "Revisit when": true,
	}
	workFields = map[string]bool{"Run:": true, "Result:": true}

	// A support item names where its fact is kept: a section of the
	// specification, a decision, a behavior, a file, or a URL.
	supportSource = regexp.MustCompile("§\\d|\\bD-\\d{3}\\b|\\bB\\d{2}\\b|https?://|`[^` ]*(/|\\.(md|go|sh|json|html))[^` ]*`")
	// Words that describe the system as it stood when a decision was made.
	historyWords = regexp.MustCompile(`(?i)\b(stays|stay|is kept|are kept|continues to|continue to|already|used to|previously|formerly|no longer|replaced|replaces)\b`)
	// A withdrawn decision names the one that replaced it, whose **Rejected.**
	// holds the reason.
	withdrawnBy = regexp.MustCompile(`^\*\*Status:\*\* withdrawn \d{4}-\d{2}-\d{2}\.\s+Replaced\s+by\s+(D-\d{3})\s+\((?s:[^)]+)\)\.$`)
	// A decision placed in docs/writing.md names the rule or the section that holds it.
	movedTo = regexp.MustCompile("^\\*\\*Status:\\*\\* moved \\d{4}-\\d{2}-\\d{2} to `docs/writing\\.md`,\\s+(?:(W-\\d{2})\\s+\\((?s:[^)]+)\\)|\"((?s:[^\"]+))\")\\.$")
	// A decision that was a plan for the work is removed, and nothing replaces it.
	// The rule it names is assembled, so that the rule index check does not read it
	// as a rule this test enforces.
	removedPlan    = regexp.MustCompile(`^\*\*Status:\*\* removed \d{4}-\d{2}-\d{2}:\s+a\s+plan\s+for\s+the\s+work,\s+not\s+a\s+decision\s+about\s+the\s+system\s+\(` + "W-" + `53\)\.$`)
	decisionStatus = regexp.MustCompile(`^\*\*Date:\*\* \d{4}-\d{2}-\d{2} · \*\*Status:\*\* (?:active|not built)(?: · \*\*Areas:\*\* ([a-z-]+(?:, [a-z-]+)*))?$`)
	isoDate        = regexp.MustCompile(`\b\d{4}-\d{2}-\d{2}\b`)
	// What a pattern may not name: this project's decisions, sections of its
	// specification, its behaviours, or the project itself (W-56).
	projectName = regexp.MustCompile(`\bD-\d{3}\b|§\d|\bB\d{2}\b|(?i)\bcogmer\b`)
	// Open work a decision must not point to: the repository's list, and the
	// task URLs of the trackers a project is likely to use.
	openWork = regexp.MustCompile(`\bopen\.md\b|docs/work/|app\.clickup\.com/t/\S+|github\.com/[\w.-]+/[\w.-]+/issues/\d+|atlassian\.net/browse/\S+|linear\.app/\S+/issue/\S+`)
)

// behaviourDocument is generated from the behaviour registry, one heading per
// behaviour, and behaviourTitle is the heading MarkdownReport writes for each.
const behaviourDocument = "docs/relied-on-behaviors.md"

var behaviourTitle = regexp.MustCompile(`^B\d{2}: `)

// specSectionTitle is a numbered heading of the specification, which a § citation
// names, so it stays however short its section is.
var specSectionTitle = regexp.MustCompile(`^\d+[a-z]?(\.\d+)*\\?\.? `)

// specification is the one document that must carry no dates.
const specification = "Shared Claude Sessions.md"

// patternsDocument holds the patterns that apply beyond this project.
const patternsDocument = "docs/patterns.md"

// valuesOrPatterns is how a citation of either document would name it (W-59).
var valuesOrPatterns = regexp.MustCompile(`\b(values|patterns)\.md\b`)

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
	case decisionFile.MatchString(doc):
		fields = decisionFields
	case workDocument.MatchString(doc):
		fields = workFields
	}

	ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n := n.(type) {
		case *ast.CodeSpan, *ast.CodeBlock, *ast.FencedCodeBlock, *ast.HTMLBlock, *ast.RawHTML:
			return ast.WalkSkipChildren, nil
		case *ast.Emphasis:
			if n.Level != 2 {
				return ast.WalkContinue, nil
			}
			label := inlineText(n, src)
			switch {
			case fields[label]:
			case doc == "docs/open.md" && opensTopLevelParagraph(n):
			case doc == "CLAUDE.md" && opensParagraphOrItem(n):
			default:
				report(firstOffset(n), "bold %q, which only a template field, or the opening sentence of an item in open.md or CLAUDE.md, may be (W-14)", label)
			}
			return ast.WalkSkipChildren, nil
		case *ast.Text:
			if doc == specification {
				for _, m := range isoDate.FindAllIndex(n.Segment.Value(src), -1) {
					report(n.Segment.Start+m[0], "a date; the specification says what must be true, not when (W-81)")
				}
				for _, m := range decisionCitation.FindAllIndex(n.Segment.Value(src), -1) {
					report(n.Segment.Start+m[0], "cites a decision; a requirement states what must be true without the log (W-84)")
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

	// Nothing cites a value or a pattern (W-59); CLAUDE.md imports both, and the
	// two documents may describe themselves.
	if doc != "CLAUDE.md" && doc != patternsDocument && doc != "docs/values.md" {
		ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
			if !entering {
				return ast.WalkContinue, nil
			}
			switch n.(type) {
			case *ast.FencedCodeBlock, *ast.CodeBlock, *ast.HTMLBlock:
				return ast.WalkSkipChildren, nil
			case *ast.Paragraph, *ast.TextBlock, *ast.Heading:
				if valuesOrPatterns.MatchString(nodeSource(n, src)) {
					lo, _ := sourceRange(n)
					report(lo, "cites the values or patterns document, which nothing cites; CLAUDE.md imports both (W-59)")
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
	// Headers a template defines are exempt: a decision's, and each finding in
	// working material.
	for c := root.FirstChild(); c != nil; c = c.NextSibling() {
		h, ok := c.(*ast.Heading)
		if !ok {
			continue
		}
		title := inlineText(h, src)
		documentTitle := h.Level == 1 && c == root.FirstChild()
		behaviourEntry := doc == behaviourDocument && behaviourTitle.MatchString(title)
		specSection := doc == specification && specSectionTitle.MatchString(title)
		if documentTitle || decisionTitle.MatchString(title) || behaviourEntry || specSection || (workDocument.MatchString(doc) && h.Level == 3) {
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

// opensParagraphOrItem reports whether n opens a top-level paragraph or a list
// item, the only bold CLAUDE.md may hold.
func opensParagraphOrItem(n ast.Node) bool {
	p := n.Parent()
	switch p.(type) {
	case *ast.Paragraph, *ast.TextBlock:
	default:
		return false
	}
	if p.FirstChild() != n {
		return false
	}
	switch p.Parent().(type) {
	case *ast.Document, *ast.ListItem:
		return true
	}
	return false
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
		{"decision field", "docs/decisions/D-124-t.md", "**Date:** 2026-09-23 · **Status:** active\n\n**Decision.** X.\n", 0},
		{"decision field elsewhere", "x.md", "**Decision.** X.\n", 1},
		{"bold label in the log", "docs/decisions/D-124-t.md", "**Where it lives.** In a file.\n", 1},
		{"open.md item", "docs/open.md", "**The command fails.** It exits 1.\n", 0},
		{"open.md bold mid-paragraph", "docs/open.md", "It **fails**.\n", 1},
		{"open.md bold in a list", "docs/open.md", "- **Fails.** Yes.\n", 1},
		{"CLAUDE.md opening a paragraph", "CLAUDE.md", "**Read the code first.** Always.\n", 0},
		{"CLAUDE.md opening a list item", "CLAUDE.md", "- **Read the code first.** Always.\n- **Cite only what exists.** Check.\n", 0},
		{"CLAUDE.md bold mid-sentence", "CLAUDE.md", "Say **user** for someone using it.\n", 1},
		{"CLAUDE.md bold mid-item", "CLAUDE.md", "- Say **user** for someone using it.\n", 1},
		{"a long open.md item", "docs/open.md", "**An item.** " + strings.Repeat("Line.\n", 60), 0},
		{"a document title over a short introduction", "x.md", "# T\n\nOne line.\n", 0},
		{"header over one line", "x.md", "Intro.\n\n## T\n\nOne line.\n", 1},
		{"header over four lines", "x.md", "Intro.\n\n## T\n\na\nb\nc\nd\n", 0},
		{"header over a subsection", "x.md", "Intro.\n\n## T\n\n### U\n\na\nb\nc\nd\n", 0},
		{"header over a short list", "x.md", "Intro.\n\n## T\n\n- a\n- b\n", 1},
		{"a second level-1 header over one line", "x.md", "# T\n\na\nb\nc\nd\n\n# U\n\nOne line.\n", 1},
		{"decision heading", "docs/decisions/D-124-t.md", "# D-124 — T\n\n**Date:** 2026-09-23 · **Status:** active\n", 0},
		{"numbered specification heading", specification, "Intro.\n\n## 3.3 No authoritative peer\n\nOne line.\n", 0},
		{"numbered top-level specification heading", specification, "Intro.\n\n# 16\\. Propagation\n\nOne line.\n", 0},
		{"unnumbered specification heading", specification, "Intro.\n\n## No authoritative peer\n\nOne line.\n", 1},
		{"behaviour heading", behaviourDocument, "Intro.\n\n### B01: A behaviour\n\nIf this changes, it breaks.\n", 0},
		{"behaviour heading elsewhere", "x.md", "Intro.\n\n### B01: A behaviour\n\nIf this changes, it breaks.\n", 1},
		{"date in the specification", specification, "The daemon started on 2026-09-22 and\nis still running today, over\nseveral lines, with no header.\n", 1},
		{"date in a code span in the specification", specification, "Write `2026-09-22` there.\n", 0},
		{"date elsewhere", "x.md", "Run on 2026-09-22.\n", 0},
		{"a decision cited in the specification", specification, "Names collide, so a room is keyed on its id (D-017).\n", 1},
		{"a decision in a code span in the specification", specification, "Write `D-017` there.\n", 0},
		{"a durable record citing working material", "x.md", "Taken from `docs/work/split.md`.\n", 1},
		{"open.md citing working material", "docs/open.md", "**Split the log.** See `docs/work/split.md`.\n", 0},
		{"working material with the findings fields", "docs/work/split.md", "# T\n\n**Run:** a\n\n**Result:** b\n", 0},
		{"working material citing itself", "docs/work/split.md", "Also in `docs/work/other.md`.\n", 0},
		{"a general pattern", patternsDocument, "P-01. Key on stable identifiers. Names collide.\n", 0},
		{"a pattern citing a decision", patternsDocument, "P-01. Key on stable identifiers (D-017).\n", 1},
		{"a pattern naming the project", patternsDocument, "P-01. Cogmer keys on stable identifiers.\n", 1},
		{"a pattern citing a section in backticks", patternsDocument, "P-01. Fail open, as `§3.1` says.\n", 1},
		{"a decision elsewhere is fine", "x.md", "As D-017 decides.\n", 0},
		{"a document citing a pattern", "x.md", "As `docs/patterns.md` says.\n", 1},
		{"an open item citing a value", "docs/open.md", "**Fix it.** As `docs/values.md` says.\n", 1},
		{"CLAUDE.md importing both", "CLAUDE.md", "Read them.\n\n@docs/values.md\n\n@docs/patterns.md\n", 0},
		{"the values document naming itself", "docs/values.md", "# Values\n\nThis file, values.md, holds them.\n", 0},
	}
	for _, c := range cases {
		if got := structureProblems(c.doc, []byte(c.src)); len(got) != c.want {
			t.Errorf("%s: got %d problems %q, want %d", c.name, len(got), got, c.want)
		}
	}
}

// Every decision not in decisionsNotRewritten follows the decision template in
// docs/writing.md, and every tombstone, whatever its number, keeps only its
// status line. A withdrawn one names the decision whose **Rejected.** says why.
func TestLaterDecisionsFollowTemplate(t *testing.T) {
	for _, p := range decisionProblems(readDecisionLog(t), guideHeadings, decisionAreas(t)) {
		t.Error(p)
	}
}

// guideHeadings returns the headings of docs/writing.md, and the ID of each rule,
// which a moved tombstone may name.
func guideHeadings(rel string) (map[string]bool, bool) {
	src, err := os.ReadFile(filepath.Join("../..", rel))
	if err != nil {
		return nil, false
	}
	headings := map[string]bool{}
	for _, line := range strings.Split(string(src), "\n") {
		if strings.HasPrefix(line, "#") {
			headings[strings.TrimSpace(strings.TrimLeft(line, "#"))] = true
		}
		if m := ruleOpening.FindStringSubmatch(line); m != nil {
			headings[m[1]] = true
		}
	}
	return headings, true
}

var ruleOpening = regexp.MustCompile(`^(W-\d{2})\. `)

// decisionProblems reads the log as Markdown, so that a field name inside a code
// block or quoted in a sentence is not mistaken for the field (docs/writing.md,
// "Enforcement"). An entry runs from its heading to the
// next heading of level 2 or above.
func decisionProblems(log string, headingsOf func(string) (map[string]bool, bool), areas map[string]bool) []string {
	src := []byte(log)
	root := goldmark.New(goldmark.WithExtensions(extension.GFM)).Parser().Parse(text.NewReader(src))
	var problems []string
	report := func(id, format string, args ...any) {
		problems = append(problems, id+" "+fmt.Sprintf(format, args...))
	}

	type entry struct {
		id   string
		body []ast.Node
	}
	var entries []*entry
	var current *entry
	for c := root.FirstChild(); c != nil; c = c.NextSibling() {
		if h, ok := c.(*ast.Heading); ok && h.Level <= 1 {
			current = nil
			if m := decisionTitle.FindStringSubmatch(inlineText(h, src)); m != nil && h.Level == 1 {
				current = &entry{id: "D-" + m[1]}
				entries = append(entries, current)
			}
			continue
		}
		if _, rule := c.(*ast.ThematicBreak); rule || current == nil {
			continue
		}
		current.body = append(current.body, c)
	}

	hasRejected := func(entries []*entry, id string) bool {
		for _, e := range entries {
			if e.id != id {
				continue
			}
			for _, c := range e.body {
				if label, ok := boldOpener(c, src); ok && label == "Rejected." {
					return true
				}
			}
		}
		return false
	}
	for _, e := range entries {
		if len(e.body) > 0 && strings.HasPrefix(nodeSource(e.body[0], src), "**Status:** removed") {
			switch {
			case len(e.body) > 1:
				report(e.id, "is a tombstone and has more than its status line (W-37)")
			case !removedPlan.MatchString(nodeSource(e.body[0], src)):
				report(e.id, "is a removed tombstone without \"removed YYYY-MM-DD: a plan for the work, not a decision about the system ("+"W-"+"53).\" (W-37)")
			}
			continue
		}
		if len(e.body) > 0 && strings.HasPrefix(nodeSource(e.body[0], src), "**Status:** moved") {
			moved := movedTo.FindStringSubmatch(nodeSource(e.body[0], src))
			guide, _ := headingsOf("docs/writing.md")
			switch {
			case len(e.body) > 1:
				report(e.id, "is a tombstone and has more than its status line (W-37)")
			case moved == nil:
				report(e.id, "is a moved tombstone without \"to `docs/writing.md`, W-NN (<words>).\" or a quoted heading (W-37)")
			case moved[1] != "" && !guide[moved[1]]:
				report(e.id, "moved to %s, which docs/writing.md does not hold (W-37)", moved[1])
			case moved[2] != "" && !guide[strings.Join(strings.Fields(moved[2]), " ")]:
				report(e.id, "moved to %q, which is not a heading of docs/writing.md (W-37)", moved[2])
			}
			continue
		}
		if len(e.body) > 0 && strings.HasPrefix(nodeSource(e.body[0], src), "**Status:** withdrawn") {
			by := withdrawnBy.FindStringSubmatch(nodeSource(e.body[0], src))
			switch {
			case len(e.body) > 1:
				report(e.id, "is a tombstone and has more than its status line; the reason belongs in the replacing decision's **Rejected.** (W-37)")
			case by == nil:
				report(e.id, "is a tombstone whose status line is not \"withdrawn YYYY-MM-DD. Replaced by D-MMM (<words>).\" (W-37)")
			case !hasRejected(entries, by[1]):
				report(e.id, "is replaced by %s, which has no **Rejected.** saying why (W-37)", by[1])
			}
			continue
		}
		before := len(problems)

		if len(e.body) == 0 || !decisionStatus.MatchString(nodeSource(e.body[0], src)) {
			report(e.id, "has no **Date:** line whose status is active or not built (W-30)")
		} else if m := decisionStatus.FindStringSubmatch(nodeSource(e.body[0], src)); m[1] == "" {
			report(e.id, "has no **Areas:** on its **Date:** line (W-30)")
		} else {
			for _, a := range strings.Split(m[1], ", ") {
				if !areas[a] {
					report(e.id, "names the area %q, which docs/decisions/README.md does not list (W-30)", a)
				}
			}
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
					report(e.id, "has no **%s** field (W-30)", f.name)
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
				report(e.id, "has no **%s** field (W-30)", f.name)
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
		if decisionsNotRewritten[e.id] {
			if len(problems) == before {
				report(e.id, "now follows the decision template: remove it from decisionsNotRewritten (W-30)")
			} else {
				problems = problems[:before]
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
	{"Rejected.", false},
	{"Limits.", false},
	{"Revisit when", false},
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

// A behaviour's Reliance starts with what breaks, for somebody debugging at 2am.
func TestBehaviorRelianceStartsWithWhatBreaks(t *testing.T) {
	for _, b := range Behaviors {
		if !strings.HasPrefix(b.Reliance, "If this changes") {
			t.Errorf("%s's Reliance must start \"If this changes\": %q (W-71)", b.ID, firstWords(b.Reliance))
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
	// Built at run time, so that the rule index check does not read these
	// samples as rules this test enforces.
	rule := "W-"
	headings := func(rel string) (map[string]bool, bool) {
		if rel == "docs/writing.md" {
			return map[string]bool{rule + "42": true, "Enforcement": true}, true
		}
		return nil, false
	}
	date := "**Date:** 2026-09-23 · **Status:** active · **Areas:** rooms, sync\n\n"
	areas := map[string]bool{"rooms": true, "sync": true}
	support := "**Support.**\n- A fact. `84a0751:docs/x-findings.md`, \"Why it went\".\n- Another, over\n  two lines. D-001 (Go).\n\n"
	complete := date + "**Decision.** x\n\n" + support + "**Rejected.** y\n\n**Revisit when** z.\n"
	tombstone := "**Status:** withdrawn 2026-09-20. Replaced by D-126\n(words).\n"
	// The decision that replaced the tombstone's, which holds the reason.
	replacement := "\n\n# D-126 — R\n\n" + complete
	cases := []struct {
		name  string
		entry string
		want  []string
	}{
		{"not yet rewritten", "# D-123 — Old\n\nAnything, already.\n", nil},
		{"rewritten but still listed", "# D-123 — T\n\n" + complete, []string{"D-123 now follows the decision template"}},
		{"complete", "# D-124 — T\n\n" + complete, nil},
		{"complete, with Limits", "# D-124 — T\n\n" + strings.Replace(complete, "**Revisit when**", "**Limits.** Some.\n\n**Revisit when**", 1), nil},
		{"no Areas", "# D-124 — T\n\n" + strings.Replace(complete, " · **Areas:** rooms, sync", "", 1), []string{"D-124 has no **Areas:** on its **Date:** line"}},
		{"an area the README does not list", "# D-124 — T\n\n" + strings.Replace(complete, "rooms, sync", "rooms, elsewhere", 1), []string{"D-124 names the area \"elsewhere\""}},
		{"not built", "# D-124 — T\n\n" + strings.Replace(complete, "active", "not built", 1), nil},
		{"no Revisit when", "# D-124 — T\n\n" + strings.Replace(complete, "\n\n**Revisit when** z.", "", 1), nil},
		{"no Rejected", "# D-124 — T\n\n" + strings.Replace(complete, "**Rejected.** y\n\n", "", 1), nil},
		{"missing Support", "# D-124 — T\n\n" + strings.Replace(complete, support, "", 1), []string{"D-124 has no **Support.**"}},
		{"Support with no list", "# D-124 — T\n\n" + strings.Replace(complete, support, "**Support.** It is so.\n\n", 1), []string{"D-124 has **Support.** with no list"}},
		{"superseded status", "# D-124 — T\n\n" + strings.Replace(complete, "active", "superseded by D-126", 1), []string{"D-124 has no **Date:** line whose status"}},
		{"a date that is not one", "# D-124 — T\n\n" + strings.Replace(complete, "2026-09-23", "d", 1), []string{"D-124 has no **Date:** line whose status"}},
		{"out of order", "# D-124 — T\n\n" + strings.Replace(complete, "**Revisit when** z.", "**Revisit when** z.\n\n**Decision.** Again.", 1), []string{"D-124 has **Decision.** out of order"}},
		{"a field only in a code block", "# D-124 — T\n\n" + strings.Replace(complete, "**Decision.** x\n\n", "```\n**Decision.** x\n```\n\n", 1), []string{"D-124 has no **Decision.**"}},
		{"support with no source", "# D-124 — T\n\n" + strings.Replace(complete, "D-001 (Go).", "It is so.", 1), []string{"D-124 has a support item with no source"}},
		{"history word", "# D-124 — T\n\n" + strings.Replace(complete, "**Decision.** x", "**Decision.** It stays as it is.", 1), []string{"D-124 says \"stays\""}},
		{"history word in code", "# D-124 — T\n\n" + strings.Replace(complete, "**Decision.** x", "**Decision.** Run `stays`.", 1), nil},
		{"Limits pointing at open.md", "# D-124 — T\n\n" + strings.Replace(complete, "**Revisit when**", "**Limits.** Three questions are open in `docs/open.md`.\n\n**Revisit when**", 1), []string{"D-124 points to open work"}},
		{"Limits pointing at a tracker task", "# D-124 — T\n\n" + strings.Replace(complete, "**Revisit when**", "**Limits.** Tracked in https://app.clickup.com/t/86abc123.\n\n**Revisit when**", 1), []string{"D-124 points to open work"}},
		{"Limits pointing at working material", "# D-124 — T\n\n" + strings.Replace(complete, "**Revisit when**", "**Limits.** Detail in `docs/work/split.md`.\n\n**Revisit when**", 1), []string{"D-124 points to open work"}},
		{"Limits stating scope", "# D-124 — T\n\n" + strings.Replace(complete, "**Revisit when**", "**Limits.** It does not decide whether stop restarts the daemon.\n\n**Revisit when**", 1), nil},
		{"open work in an entry not yet rewritten", "# D-123 — Old\n\nLeft open in `docs/open.md`.\n", nil},
		{"tombstone", "# D-076 — T\n\n" + tombstone + replacement, nil},
		{"tombstone with more", "# D-076 — T\n\n" + tombstone + "\nMore history.\n" + replacement, []string{"D-076 is a tombstone and has more"}},
		{"tombstone with a Why", "# D-076 — T\n\n**Status:** withdrawn 2026-09-20. Replaced by D-126 (words). Why: `05f89c4`.\n" + replacement, []string{"D-076 is a tombstone whose status line"}},
		{"replacement with no Rejected", "# D-076 — T\n\n" + tombstone + strings.Replace(replacement, "**Rejected.** y\n\n", "", 1), []string{"D-076 is replaced by D-126, which has no **Rejected.**"}},
		{"replacement missing", "# D-076 — T\n\n" + tombstone, []string{"D-076 is replaced by D-126, which has no **Rejected.**"}},
		{"moved to a rule", "# D-128 — T\n\n**Status:** moved 2026-09-25 to `docs/writing.md`, " + rule + "42\n(evidence is cited).\n", nil},
		{"moved to a section", "# D-124 — T\n\n**Status:** moved 2026-09-25 to `docs/writing.md`, \"Enforcement\".\n", nil},
		{"moved to a missing rule", "# D-124 — T\n\n**Status:** moved 2026-09-25 to `docs/writing.md`, " + rule + "99 (x).\n", []string{"D-124 moved to " + rule + "99"}},
		{"moved to a missing section", "# D-124 — T\n\n**Status:** moved 2026-09-25 to `docs/writing.md`, \"Elsewhere\".\n", []string{"D-124 moved to \"Elsewhere\""}},
		{"removed as a plan", "# D-002 — T\n\n**Status:** removed 2026-09-29: a plan for the work, not a decision\nabout the system (" + rule + "53).\n", nil},
		{"removed without its reason", "# D-002 — T\n\n**Status:** removed 2026-09-29.\n", []string{"D-002 is a removed tombstone without"}},
		{"removed with more", "# D-002 — T\n\n**Status:** removed 2026-09-29: a plan for the work, not a decision about the system (" + rule + "53).\n\nMore.\n", []string{"D-002 is a tombstone and has more"}},
		{"moved with more", "# D-124 — T\n\n**Status:** moved 2026-09-25 to `docs/writing.md`, \"Enforcement\".\n\nMore.\n", []string{"D-124 is a tombstone and has more"}},
	}
	for _, c := range cases {
		got := decisionProblems("# Log\n\n"+c.entry, headings, areas)
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

// explanationCitations returns each citation in an explanation, as "line: citation".
// Code spans count: a citation in backticks goes stale the same way.
func explanationCitations(src []byte) []string {
	var found []string
	for i, line := range strings.Split(string(src), "\n") {
		for _, re := range []*regexp.Regexp{decisionCitation, sectionCitation, behaviorCitation} {
			for _, m := range re.FindAllString(line, -1) {
				found = append(found, fmt.Sprintf("%d: %s", i+1, m))
			}
		}
	}
	return found
}

// An explanation cites nothing, so nothing in it has to be kept in step with the
// entry it cites (W-65).
func TestExplanationsCiteNothing(t *testing.T) {
	matches, err := filepath.Glob("../../docs/explanations/*.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range matches {
		src, err := os.ReadFile(m)
		if err != nil {
			t.Fatal(err)
		}
		rel, _ := filepath.Rel("../..", m)
		for _, c := range explanationCitations(src) {
			t.Errorf("%s:%s: an explanation cites nothing; say what the cited entry holds instead (W-65)", filepath.ToSlash(rel), c)
		}
	}
}

func TestExplanationCitationsCatchesEachKind(t *testing.T) {
	src := []byte("The header check (D-087) holds.\nSee §25 and `B09`.\nNothing here.\n")
	got := explanationCitations(src)
	want := []string{"1: D-087", "2: §25", "2: B09"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("explanationCitations = %q, want %q", got, want)
	}
}
