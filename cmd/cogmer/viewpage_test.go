package main

import (
	"strings"
	"testing"
)

// One page is the daemon's view and the static page that reads a cloud session's room, so the hooks
// that let the second replace the first's connections are in the page and nothing else differs
// (D-301). Where no cogmerView exists the page opens an EventSource and links to /room/ as before.
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
