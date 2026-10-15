package main

import (
	"bytes"
	"net/url"
	"strings"
	"sync"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// A turn in the view is Markdown written by someone else, and it is untrusted text
// (D-211). The daemon renders it, so a test can read the result, and the page puts
// the result in the feed as it is. Everything that could act in the reader's browser
// is decided here: no markup a colleague wrote reaches the page as markup, a link is
// followed only when it is an http or https address, and nothing is fetched on the
// reader's behalf.
var markdownRenderer = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithParserOptions(parser.WithASTTransformers(util.Prioritized(safeNodes{}, 100))),
	// A line break inside a paragraph is kept. People type a turn as lines, and the
	// page showed each of them.
	goldmark.WithRendererOptions(html.WithHardWraps()),
)

// renderMarkdown returns the HTML for one turn's text.
func renderMarkdown(src string) string {
	var out bytes.Buffer
	if err := markdownRenderer.Convert([]byte(src), &out); err != nil {
		return "<p>" + escapeHTMLText(src) + "</p>"
	}
	return out.String()
}

func escapeHTMLText(s string) string {
	return string(util.EscapeHTML([]byte(s)))
}

// safeNodes rewrites the parts of a parsed turn that goldmark would otherwise leave to
// its own defaults. Those defaults drop raw HTML silently, which loses text such as
// "put a <div> there", and they refuse a short list of address schemes where only two
// are wanted.
type safeNodes struct{}

func (safeNodes) Transform(doc *ast.Document, reader text.Reader, _ parser.Context) {
	source := reader.Source()
	var found []ast.Node
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			switch n.(type) {
			case *ast.RawHTML, *ast.HTMLBlock, *ast.Link, *ast.AutoLink, *ast.Image:
				found = append(found, n)
			}
		}
		return ast.WalkContinue, nil
	})
	for _, n := range found {
		parent := n.Parent()
		if parent == nil {
			continue
		}
		switch n := n.(type) {
		case *ast.RawHTML:
			// Shown as the characters the colleague typed.
			var raw []byte
			for i := 0; i < n.Segments.Len(); i++ {
				seg := n.Segments.At(i)
				raw = append(raw, seg.Value(source)...)
			}
			parent.ReplaceChild(parent, n, literal(raw))
		case *ast.HTMLBlock:
			// Shown as a block of code, which is how a reader expects markup to look.
			block := ast.NewCodeBlock()
			lines := n.Lines()
			if n.HasClosure() {
				lines.Append(n.ClosureLine)
			}
			block.SetLines(lines)
			parent.ReplaceChild(parent, n, block)
		case *ast.Image:
			// An image would load from an address the colleague chose, which tells them
			// when the reader looked. It is shown as its address and never fetched.
			alt := string(nodeText(n, source))
			label := "[image] "
			if alt != "" {
				label = "[image: " + alt + "] "
			}
			parent.ReplaceChild(parent, n, literal([]byte(label+string(n.Destination))))
		case *ast.Link:
			dest := string(n.Destination)
			if followable(dest) {
				openAway(n)
				if string(nodeText(n, source)) != dest && dest != "" {
					parent.InsertAfter(parent, n, literal([]byte(" ("+dest+")")))
				}
				continue
			}
			// Any other address is shown beside the text and cannot be followed.
			for c := n.FirstChild(); c != nil; {
				next := c.NextSibling()
				parent.InsertBefore(parent, n, c)
				c = next
			}
			if dest != "" {
				parent.InsertBefore(parent, n, literal([]byte(" ("+dest+")")))
			}
			parent.RemoveChild(parent, n)
		case *ast.AutoLink:
			dest := string(n.URL(source))
			if n.AutoLinkType == ast.AutoLinkURL && followable(dest) {
				openAway(n)
				continue
			}
			parent.ReplaceChild(parent, n, literal(n.Label(source)))
		}
	}
}

// literal is text that is written out escaped, exactly as it is.
func literal(b []byte) *ast.String {
	s := ast.NewString(b)
	s.SetRaw(true)
	return s
}

// openAway marks a link to open in a new tab and to tell the page it leads to nothing
// about the reader's.
func openAway(n ast.Node) {
	n.SetAttributeString("target", []byte("_blank"))
	n.SetAttributeString("rel", []byte("noopener noreferrer"))
}

// followable is an address a reader may follow: http or https, with a host. Anything
// else, such as javascript:, data: or file:, is refused by what it is not and not by
// a list of what it is.
func followable(dest string) bool {
	u, err := url.Parse(dest)
	if err != nil || u.Host == "" {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		return true
	}
	return false
}

// nodeText is the plain text under a node.
func nodeText(n ast.Node, source []byte) []byte {
	var out []byte
	_ = ast.Walk(n, func(c ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch c := c.(type) {
		case *ast.Text:
			out = append(out, c.Segment.Value(source)...)
		case *ast.String:
			out = append(out, c.Value...)
		}
		return ast.WalkContinue, nil
	})
	return out
}

// renderedTurnsKept bounds the turns whose rendering is remembered. A room's turns do not
// change once written, and the stream sends the whole room at every arrival, so without
// this a room of a thousand turns is rendered a thousand times for each one that comes.
const renderedTurnsKept = 4096

// renderedTurns remembers each turn's rendering by its event, and renders again if the
// text under an event is ever different.
type renderedTurns struct {
	mu sync.Mutex
	by map[string]renderedTurn
}

type renderedTurn struct{ text, html string }

func (r *renderedTurns) get(eventID, content string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if t, ok := r.by[eventID]; ok && t.text == content {
		return t.html
	}
	html := renderMarkdown(content)
	if r.by == nil || len(r.by) >= renderedTurnsKept {
		r.by = map[string]renderedTurn{}
	}
	r.by[eventID] = renderedTurn{content, html}
	return html
}
