package main

import (
	"os"
	"strings"
	"testing"
)

// One page is the daemon's view and the static page that reads a cloud session's room, so the hooks
// that let the second replace the first's connections are in the page and nothing else differs
// (D-302). Where no cogmerView exists the page opens an EventSource and links to /room/ as before.
func TestTheViewPageHasTheHooksTheStaticPageFillsAndKeepsItsOwnWays(t *testing.T) {
	for what, want := range map[string]string{
		"the place the build adds the script": "<!-- cogmer:view-hooks -->",
		"the view object":                     "window.cogmerView || null",
		"the room":                            "const ROOM = VIEW ? VIEW.room : (location.pathname.startsWith('/room/')",
		"the stream of a room":                "VIEW ? VIEW.stream('/stream?room=' + encodeURIComponent(ROOM)) : new EventSource('/stream?room='",
		"the stream of the rooms":             "VIEW ? VIEW.stream('/rooms/stream') : new EventSource('/rooms/stream')",
		"the link to a room":                  "VIEW ? VIEW.roomHref(r.name) : '/room/'+encodeURIComponent(r.name)",
	} {
		if n := strings.Count(string(uiPage), want); n != 1 {
			t.Errorf("%s: the page holds %q %d times, want once", what, want, n)
		}
	}
}

// The static page carries a security policy of its own, since the daemon cannot send one for a page it
// does not serve (D-212, the room page's security policy allows only the daemon's own images and
// requests). A mistake in the renderer must not make the browser fetch an address a colleague chose:
// the page loads no image but one made from bytes it fetched through the daemon, runs no code from
// elsewhere and contacts nothing but its own files, the relay map and a relay.
func TestTheStaticPagesPolicyLoadsNoImageFromAnAddressAndContactsOnlyTheRelay(t *testing.T) {
	b, err := os.ReadFile("../../web/view/policy.txt")
	if err != nil {
		t.Fatal(err)
	}
	policy := strings.TrimSpace(string(b))
	for _, want := range []string{
		"default-src 'none'",
		"img-src blob:",
		"connect-src 'self' https://tailcat.dev wss:",
		"base-uri 'none'",
		"form-action 'none'",
		"'wasm-unsafe-eval'",
	} {
		if !strings.Contains(policy, want) {
			t.Errorf("the page's policy lacks %q: %s", want, policy)
		}
	}
	for _, banned := range []string{"'unsafe-eval'", "img-src 'self'", "img-src *", "http:", "https: ", "connect-src *", "script-src *", "data:"} {
		if strings.Contains(policy, banned) {
			t.Errorf("the page's policy holds %q: %s", banned, policy)
		}
	}
	if strings.Contains(policy, "\n") {
		t.Error("the policy is more than one line, which a meta tag cannot hold")
	}
	// The daemon's own page keeps loading no image from an address either.
	if !strings.Contains(roomPagePolicy, "default-src 'none'") || strings.Contains(roomPagePolicy, "img-src *") {
		t.Errorf("the daemon's page policy changed: %s", roomPagePolicy)
	}
}
