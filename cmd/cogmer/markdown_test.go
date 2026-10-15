package main

import (
	"regexp"
	"strings"
	"testing"
)

// What a reader sees of a turn is HTML the daemon wrote from text a colleague wrote
// (D-211), so each case here is a thing a colleague could type and what must come out.

func TestMarkdownRendersWhatAnAssistantWrites(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"heading", "# Title\n\ntext", "<h1>Title</h1>"},
		{"second heading", "## Part", "<h2>Part</h2>"},
		{"numbered list", "1. one\n2. two", "<ol>"},
		{"bullet list", "- a\n- b", "<ul>"},
		{"emphasis with underscores", "an _important_ point", "<em>important</em>"},
		{"emphasis with asterisks", "an *important* point", "<em>important</em>"},
		{"bold", "a **strong** point", "<strong>strong</strong>"},
		{"quotation", "> said so", "<blockquote>"},
		{"inline code", "run `go test`", "<code>go test</code>"},
		{"fenced code", "```go\nx := 1\n```", "<pre><code class=\"language-go\">x := 1\n</code></pre>"},
		{"table", "| a | b |\n|---|---|\n| 1 | 2 |", "<table>"},
		{"strikethrough", "~~gone~~", "<del>gone</del>"},
		{"rule", "a\n\n---\n\nb", "<hr"},
		{"a line break inside a paragraph", "one\ntwo", "one<br"},
		{"a link to a web address", "[the docs](https://example.com/a)", `<a href="https://example.com/a"`},
		{"a bare web address", "see https://example.com/a now", `<a href="https://example.com/a"`},
	}
	for _, c := range cases {
		if got := renderMarkdown(c.in); !strings.Contains(got, c.want) {
			t.Errorf("%s: %q renders as %q, which lacks %q", c.name, c.in, got, c.want)
		}
	}
}

// A link a reader follows opens away from the page and carries nothing of it.
func TestAFollowableLinkOpensAwayAndCarriesNoReferrer(t *testing.T) {
	got := renderMarkdown("[x](https://example.com)")
	for _, want := range []string{`target="_blank"`, `rel="noopener noreferrer"`} {
		if !strings.Contains(got, want) {
			t.Errorf("a link lacks %s: %s", want, got)
		}
	}
	if got := renderMarkdown("https://example.com"); !strings.Contains(got, `rel="noopener noreferrer"`) {
		t.Errorf("a bare address becomes a link that carries the reader's page: %s", got)
	}
}

// Text that differs from its address does not hide where the link goes.
func TestALinkShowsItsAddressWhenItsTextIsNotTheAddress(t *testing.T) {
	got := renderMarkdown("[your bank](https://example.net/login)")
	if !strings.Contains(got, "(https://example.net/login)") {
		t.Errorf("a link whose text is not its address does not show the address: %s", got)
	}
	if got := renderMarkdown("<https://example.net/login>"); strings.Contains(got, "(https://example.net/login)") {
		t.Errorf("an address shown as itself is shown twice: %s", got)
	}
}

// Only http and https are followed, whatever the scheme is spelled like.
func TestOnlyWebAddressesAreFollowed(t *testing.T) {
	for _, dest := range []string{
		"javascript:alert(1)", "JaVaScRiPt:alert(1)", "  javascript:alert(1)", "java\tscript:alert(1)",
		"data:text/html;base64,PHNjcmlwdD4=", "vbscript:x", "file:///etc/passwd", "ftp://example.com/x",
		"mailto:a@example.com", "//example.com/x", "/relative/path", "cmd/cogmer/main.go", "https:///nohost",
		"tel:123", "blob:https://example.com/x", "about:blank",
	} {
		for _, in := range []string{"[x](" + dest + ")", "[x](<" + dest + ">)", "![x](" + dest + ")", "<" + dest + ">"} {
			got := renderMarkdown(in)
			if strings.Contains(got, "<a ") || strings.Contains(got, "href=") || strings.Contains(got, "<img") || strings.Contains(got, "src=") {
				t.Errorf("%q renders something a browser can follow or fetch: %s", in, got)
			}
		}
	}
}

// An image loads from an address the colleague chose, so it is never an image.
func TestAnImageIsShownAsItsAddressAndNeverFetched(t *testing.T) {
	got := renderMarkdown("![a chart](https://tracker.example.com/pixel.png)")
	if strings.Contains(got, "<img") || strings.Contains(got, "src=") {
		t.Errorf("an image is left for the browser to fetch: %s", got)
	}
	if !strings.Contains(got, "[image: a chart] https://tracker.example.com/pixel.png") {
		t.Errorf("an image is not shown as its address: %s", got)
	}
	if got := renderMarkdown("![](https://tracker.example.com/pixel.png)"); !strings.Contains(got, "[image] https://tracker.example.com/pixel.png") {
		t.Errorf("an image with no description is not shown as its address: %s", got)
	}
}

// Markup in a turn is text in the page. Dropping it would lose what the colleague wrote,
// as "put a <div> there" would read "put a  there".
func TestMarkupInATurnIsShownAsTextAndNeverActs(t *testing.T) {
	cases := []struct{ name, in, shown string }{
		{"a script", "before <script>alert(1)</script> after", "&lt;script&gt;alert(1)&lt;/script&gt;"},
		{"an element in prose", "put a <div> there", "put a &lt;div&gt; there"},
		{"an image with a handler", `<img src=x onerror="alert(1)">`, `&lt;img src=x onerror=&quot;alert(1)&quot;&gt;`},
		{"a block of markup", "<div onclick=\"alert(1)\">\nhello\n</div>", "&lt;div onclick=&quot;alert(1)&quot;&gt;"},
		{"a comment", "a <!-- note --> b", "&lt;!-- note --&gt;"},
		{"a handler in a link's title", `[x](https://example.com "t\" onmouseover=\"alert(1)")`, ""},
		{"markup in link text", "[<b>x</b>](https://example.com)", "&lt;b&gt;x&lt;/b&gt;"},
		{"markup in code", "`<script>`", "<code>&lt;script&gt;</code>"},
		{"markup in a fence", "```\n<script>alert(1)</script>\n```", "&lt;script&gt;alert(1)&lt;/script&gt;"},
		{"an entity that spells a tag", "&lt;script&gt;", ""},
	}
	for _, c := range cases {
		got := renderMarkdown(c.in)
		if c.shown != "" && !strings.Contains(got, c.shown) {
			t.Errorf("%s: the markup is not shown as text; %q renders as %s", c.name, c.in, got)
		}
		if err := onlyMarkupWeWrite(got); err != "" {
			t.Errorf("%s: %q renders markup a colleague wrote: %s in %s", c.name, c.in, err, got)
		}
	}
}

// A quote or an angle bracket in an address cannot close the attribute it is written in.
func TestAnAddressCannotBreakOutOfItsAttribute(t *testing.T) {
	for _, in := range []string{
		`[x](https://example.com/"onmouseover="alert(1))`,
		`[x](<https://example.com/" onmouseover="alert(1)>)`,
		"[x](https://example.com/><script>alert(1)</script>)",
		"https://example.com/\"onmouseover=\"alert(1)",
	} {
		got := renderMarkdown(in)
		if err := onlyMarkupWeWrite(got); err != "" {
			t.Errorf("%q: %s in %s", in, err, got)
		}
	}
}

// Every tag and attribute in the output is one the renderer writes. The cases above name
// the attacks somebody thought of, and this holds for the ones nobody did.
func TestWhatIsRenderedHoldsNoMarkupBeyondWhatTheRendererWrites(t *testing.T) {
	inputs := []string{
		"<a href=\"javascript:x\">x</a>", "<iframe src=//evil></iframe>", "<style>*{}</style>",
		"<svg onload=x>", "<math><mi>x</mi></math>", "<form action=//evil><input></form>",
		"<<script>script>alert(1)<</script>/script>", "<scr<script>ipt>", "<object data=x>",
		"- <b onclick=x>item</b>", "> <meta http-equiv=refresh content=0>", "| <i onclick=x> |\n|---|\n| <u> |",
		"[a](https://x \"<script>\")", "![<script>](https://x)", "# <script>", "**<script>**",
		"<base href=//evil>", "<link rel=stylesheet href=//evil>", "&#x6A;avascript:alert(1)",
		"[x](&#x6A;avascript:alert(1))", "[x](java&#x73;cript:alert(1))", "\x00<script>", "<script\n>alert(1)</script\n>",
		"<details open ontoggle=x>", "<video src=//evil autoplay>", "<body onload=x>", "<a\nhref=x>",
	}
	for _, in := range inputs {
		if err := onlyMarkupWeWrite(renderMarkdown(in)); err != "" {
			t.Errorf("%q: %s in %s", in, err, renderMarkdown(in))
		}
	}
}

var (
	tagInOutput  = regexp.MustCompile(`<(/?)([a-zA-Z][a-zA-Z0-9]*)((?:\s+[^<>]*?)?)\s*/?>`)
	attrInOutput = regexp.MustCompile(`\s([a-zA-Z-]+)(?:="([^"]*)")?`)
)

// the tags the renderer writes and the attributes each may carry
var rendered = map[string][]string{
	"p": nil, "br": nil, "hr": nil, "strong": nil, "em": nil, "del": nil, "code": {"class"},
	"pre": nil, "blockquote": nil, "ul": nil, "ol": {"start"}, "li": nil,
	"h1": nil, "h2": nil, "h3": nil, "h4": nil, "h5": nil, "h6": nil,
	"table": nil, "thead": nil, "tbody": nil, "tr": nil, "th": {"align"}, "td": {"align"},
	"a": {"href", "target", "rel", "title"}, "input": {"checked", "disabled", "type"},
}

// onlyMarkupWeWrite returns what is wrong with html: a tag or an attribute the renderer
// does not write, a handler, or an address that is not http or https. An angle bracket
// that is not part of a tag shows as &lt; and so cannot start one.
func onlyMarkupWeWrite(html string) string {
	stripped := regexp.MustCompile(`<!--.*?-->`).ReplaceAllString(html, "")
	if stripped != html {
		return "a comment"
	}
	rest := tagInOutput.ReplaceAllStringFunc(html, func(tag string) string {
		m := tagInOutput.FindStringSubmatch(tag)
		allowed, ok := rendered[strings.ToLower(m[2])]
		if !ok {
			return "\x01tag " + m[2] + "\x01"
		}
		for _, a := range attrInOutput.FindAllStringSubmatch(m[3], -1) {
			name := strings.ToLower(a[1])
			if strings.HasPrefix(name, "on") || !contains(allowed, name) {
				return "\x01attribute " + name + " of " + m[2] + "\x01"
			}
			if name == "href" && !followable(unescapeAttr(a[2])) {
				return "\x01address " + a[2] + "\x01"
			}
		}
		return ""
	})
	if i := strings.Index(rest, "\x01"); i >= 0 {
		return rest[i+1 : i+1+strings.Index(rest[i+1:], "\x01")]
	}
	if strings.ContainsAny(rest, "<>") {
		return "an angle bracket outside any tag"
	}
	return ""
}

func unescapeAttr(s string) string {
	return strings.NewReplacer("&amp;", "&", "&quot;", `"`, "&lt;", "<", "&gt;", ">", "&#39;", "'").Replace(s)
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// The checker is only worth having if it can fail.
func TestTheMarkupCheckerSeesWhatItIsFor(t *testing.T) {
	for _, bad := range []string{
		"<script>x</script>", `<a href="javascript:x">x</a>`, `<a href="https://x" onclick="y">x</a>`,
		`<img src="https://x">`, "<p>a <!-- b --> c</p>", `<a href="https://x" style="y">x</a>`, "<p>1 < 2</p>",
	} {
		if onlyMarkupWeWrite(bad) == "" {
			t.Errorf("the checker passes %s", bad)
		}
	}
	for _, good := range []string{`<p>a <a href="https://x" target="_blank" rel="noopener noreferrer">b</a></p>`, "<p>1 &lt; 2</p>"} {
		if err := onlyMarkupWeWrite(good); err != "" {
			t.Errorf("the checker refuses %s: %s", good, err)
		}
	}
}

func TestATurnIsRenderedOnceWhileItsTextIsTheSame(t *testing.T) {
	var r renderedTurns
	first := r.get("e1", "**a**")
	r.mu.Lock()
	r.by["e1"] = renderedTurn{"**a**", "from the memory"}
	r.mu.Unlock()
	if got := r.get("e1", "**a**"); got != "from the memory" {
		t.Errorf("a turn already rendered is rendered again: %s", got)
	}
	if got := r.get("e1", "**b**"); got == "from the memory" || got == first {
		t.Errorf("a turn whose text changed is served from the memory: %s", got)
	}
}

func TestTheMemoryOfRenderedTurnsIsBounded(t *testing.T) {
	var r renderedTurns
	for i := 0; i < renderedTurnsKept*3; i++ {
		r.get(strings.Repeat("x", i%7)+string(rune('a'+i%26))+string(rune(i)), "t")
		if len(r.by) > renderedTurnsKept {
			t.Fatalf("the memory holds %d turns after %d", len(r.by), i+1)
		}
	}
}
