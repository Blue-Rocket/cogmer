package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// What leaves the machine is told to users in plugin/README.md ("What leaves your
// machine"), and D-192 (the public relay is an accepted dependency) and D-115 (a
// centralized component must trace to a disclosed tradeoff) rest on it. A new way
// out is a change to what a user is told, so each test here fails when one appears,
// and the failure says to update the README and decide the new path before adding
// it to a list below.
//
// Several of these properties were true only because nothing had been written that
// would break them. They are held here by tests instead.

type site struct {
	file, what string
}

// outboundCalls lists, per file, how many times the code names each construct that
// opens a connection or starts a program. Every entry says where it goes.
var outboundCalls = map[site]struct {
	count int
	goes  string
}{
	{"browser.go", "exec.Command"}:       {3, "opens the pairing page the daemon serves on loopback"},
	{"doctor.go", "exec.Command"}:        {1, "asks the user's own claude for its version"},
	{"probe.go", "exec.Command"}:         {4, "runs the user's own claude and this binary on fixed test prompts, and launchctl"},
	{"stop_unix.go", "exec.Command"}:     {2, "lsof and ps, which read this machine's processes"},
	{"main.go", "http.Client"}:           {3, "the daemon on loopback, and a health probe of one address"},
	{"main.go", "http.Get"}:              {0, ""},
	{"transport.go", "http.Client"}:      {2, "a peer, over TLS pinned to its key, by TCP or the overlay"},
	{"sync.go", "http.Client"}:           {1, "the peer client it is handed, which is the one above"},
	{"verify.go", "http.Client"}:         {1, "the peer client it is handed, which is the one above"},
	{"localguard.go", "http.Client"}:     {1, "the daemon on loopback, with the guard header attached in one place"},
	{"localguard.go", "http.NewRequest"}: {1, "the same request"},
	{"transport.go", "http.Transport"}:   {1, "the same client"},
	{"transport.go", "net.Dialer"}:       {1, "a peer's TCP address"},
}

var outboundSelectors = map[string]map[string]bool{
	"http": {"Get": true, "Post": true, "PostForm": true, "Head": true, "NewRequest": true,
		"NewRequestWithContext": true, "DefaultClient": true, "Client": true, "Transport": true},
	"net": {"Dial": true, "DialTimeout": true, "Dialer": true, "DialTCP": true, "DialUDP": true,
		"ListenPacket": true, "ListenUDP": true, "LookupHost": true, "LookupIP": true, "LookupAddr": true,
		"LookupTXT": true, "Resolver": true},
	"tls":  {"Dial": true, "DialWithDialer": true, "Dialer": true},
	"exec": {"Command": true, "CommandContext": true},
}

func parseSources(t *testing.T) map[string]*ast.File {
	t.Helper()
	out := map[string]*ast.File{}
	matches, _ := filepath.Glob("*.go")
	for _, m := range matches {
		if strings.HasSuffix(m, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), m, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		out[m] = f
	}
	return out
}

func TestEveryWayOutOfTheCodeIsAccountedFor(t *testing.T) {
	found := map[site]int{}
	for name, f := range parseSources(t) {
		ast.Inspect(f, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok || !outboundSelectors[pkg.Name][sel.Sel.Name] {
				return true
			}
			found[site{name, pkg.Name + "." + sel.Sel.Name}]++
			return true
		})
	}
	for s, n := range found {
		want, ok := outboundCalls[s]
		if !ok || want.count != n {
			t.Errorf("%s names %s %d times and the list says %d. This is a way for data to leave the machine or to start a program. Decide where it goes, say so in plugin/README.md under \"What leaves your machine\" if a user's data is involved, and then change the list", s.file, s.what, n, want.count)
		}
	}
	for s, want := range outboundCalls {
		if want.count > 0 && found[s] == 0 {
			t.Errorf("the list expects %s in %s and the code does not hold it; remove the entry", s.what, s.file)
		}
	}
}

// Every dependency outside the standard library is code that can open connections
// of its own, as tailcat does. A new one is a new question about what leaves.
var knownDependencies = map[string]string{
	"github.com/tailscale/tailcat":  "the overlay: relay introduction, WireGuard, hole punching (D-068)",
	"tailscale.com/tailcfg":         "the relay region type tailcat takes",
	"tailscale.com/types/key":       "tailcat's key type",
	"tailscale.com/wgengine/filter": "tailcat's packet filter ports",
	"github.com/yuin/goldmark":      "renders text to HTML locally",
	"modernc.org/sqlite":            "the local stores",
	"golang.org/x/term":             "reads a terminal",
}

func TestEveryDependencyIsOneWeKnowTheReachOf(t *testing.T) {
	for name, f := range parseSources(t) {
		for _, im := range f.Imports {
			path, _ := strconv.Unquote(im.Path.Value)
			first := strings.SplitN(path, "/", 2)[0]
			if !strings.Contains(first, ".") {
				continue // the standard library
			}
			if _, ok := knownDependencies[path]; !ok {
				t.Errorf("%s imports %s, which nothing here says what it reaches. Find out whether it opens a connection, say so in plugin/README.md if a user's data could travel by it, and then add it to knownDependencies with the answer", name, path)
			}
		}
	}
}

// Only the overlay file may use the overlay. The relay, and the fetch of its list
// from tailcat.dev, are D-192 and are disclosed because of it.
func TestOnlyTheOverlayFileUsesTheOverlay(t *testing.T) {
	for name, f := range parseSources(t) {
		if name == "tailcat.go" {
			continue
		}
		for _, im := range f.Imports {
			if path, _ := strconv.Unquote(im.Path.Value); strings.HasPrefix(path, "github.com/tailscale/") || strings.HasPrefix(path, "tailscale.com") {
				t.Errorf("%s imports %s; the overlay is confined to tailcat.go (D-068)", name, path)
			}
		}
	}
}

// The pages the daemon serves hold no address of any other machine and load
// nothing from outside. A page that fetched a font or a script would tell that host
// every time somebody looked at their room.
func TestTheEmbeddedPagesReachOnlyTheDaemon(t *testing.T) {
	external := regexp.MustCompile(`(?i)(https?:)?//[a-z0-9.-]+\.[a-z]{2,}|@import|<link\b|\bsrc\s*=\s*["']?(https?:)?//|url\(\s*["']?(https?:)?//|\bWebSocket\b|sendBeacon|XMLHttpRequest`)
	for _, name := range []string{"ui.html", "pair.html"} {
		b, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if loc := external.FindString(string(b)); loc != "" {
			t.Errorf("%s holds %q, which reaches something other than the daemon that served it", name, loc)
		}
		for _, call := range regexp.MustCompile(`(?:fetch|EventSource)\(\s*([^,)]+)`).FindAllStringSubmatch(string(b), -1) {
			arg := strings.TrimSpace(call[1])
			if strings.HasPrefix(arg, "'/") || strings.HasPrefix(arg, `"/`) || arg == "path" {
				continue
			}
			t.Errorf("%s calls %s with %s, which is not a path on the daemon", name, call[0], arg)
		}
	}
}

// The plugin's scripts run outside the Go code. Each line that fetches something
// is listed, with where it goes.
var scriptFetches = map[string]struct {
	count int
	goes  string
}{
	"plugin/hooks-handlers/common.sh":        {1, "the daemon's health address on loopback"},
	"plugin/hooks-handlers/session-start.sh": {1, "the daemon's health address on loopback"},
	"plugin/hooks-handlers/install.sh":       {2, "the release host, for the binary, and Go's module download when building from source (D-121)"},
}

func TestEveryFetchInThePluginScriptsIsAccountedFor(t *testing.T) {
	fetch := regexp.MustCompile(`\bcurl\b[^\n]*(http|\$url|\$\{?base)|\bwget\b|\bgo (install|get|build|mod download)\b`)
	found := map[string]int{}
	var scripts []string
	filepath.Walk("../../plugin", func(p string, info os.FileInfo, err error) error {
		if err == nil && strings.HasSuffix(p, ".sh") {
			scripts = append(scripts, p)
		}
		return nil
	})
	sort.Strings(scripts)
	for _, p := range scripts {
		b, _ := os.ReadFile(p)
		rel := strings.TrimPrefix(filepath.ToSlash(p), "../../")
		for _, line := range strings.Split(string(b), "\n") {
			l := strings.TrimSpace(line)
			if strings.HasPrefix(l, "#") || strings.HasPrefix(l, "say ") || strings.HasPrefix(l, "ct_say ") || strings.HasPrefix(l, "echo ") {
				continue
			}
			if fetch.MatchString(l) {
				found[rel]++
			}
		}
	}
	for p, n := range found {
		if want := scriptFetches[p]; want.count != n {
			t.Errorf("%s has %d lines that fetch something and the list says %d. Say where it goes in the list, and in plugin/README.md if a user's data could travel by it", p, n, want.count)
		}
	}
	for p, want := range scriptFetches {
		if found[p] == 0 {
			t.Errorf("the list expects %d fetches in %s and the script has none", want.count, p)
		}
	}
}

// What a peer receives is exactly these fields. The machine-level list of the peers
// this machine knows is in none of them, and was absent only because no payload
// happened to include it. A new field is a new thing a colleague's machine learns
// about this one, so it has to be looked at.
func TestWhatAPeerReceivesIsTheseFieldsAndNoOthers(t *testing.T) {
	want := map[string][]string{
		"syncRequest":   {"Protocol", "RoomID", "Have", "PeerID", "Timestamp", "Nonce", "Signature", "Endpoint"},
		"syncResponse":  {"Protocol", "RoomID", "Events"},
		"wireEvent":     {"EventID", "PeerID", "PeerSequence", "RoomID", "Timestamp", "UserID", "UserDisplayName", "MachineID", "OriginSessionID", "EventType", "Content", "Metadata", "Signature", "SigVersion"},
		"offerRequest":  {"Protocol", "PeerID", "RoomID", "RoomName", "Endpoint", "Timestamp", "Nonce", "Signature"},
		"verifyMessage": {"Step", "PeerID", "Payload", "Signature"},
	}
	fields := map[string][]string{}
	for _, f := range parseSources(t) {
		ast.Inspect(f, func(n ast.Node) bool {
			ts, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}
			if _, tracked := want[ts.Name.Name]; !tracked {
				return true
			}
			if st, ok := ts.Type.(*ast.StructType); ok {
				for _, fl := range st.Fields.List {
					for _, nm := range fl.Names {
						fields[ts.Name.Name] = append(fields[ts.Name.Name], nm.Name)
					}
				}
			}
			return true
		})
	}
	for typ, names := range want {
		got := fields[typ]
		if len(got) == 0 {
			t.Errorf("%s was not found, so what it sends is not checked", typ)
			continue
		}
		if strings.Join(got, ",") != strings.Join(names, ",") {
			t.Errorf("%s carries %v and the list says %v. A peer's machine receives these fields. Say in plugin/README.md what a new one reveals if it reveals anything about this machine, and then change the list", typ, got, names)
		}
	}
}
