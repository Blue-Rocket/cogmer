package main

import (
	"bytes"
	"net"
	"net/http"
)

// --- keeping a web page out of the local API (D-087) ---
//
// The daemon serves the room view over HTTP on loopback. Loopback keeps other
// machines out; it does nothing about a page open in this machine's browser, which
// can reach 127.0.0.1 like any other address. That matters because the local API
// can mark a peer verified -- the check D-054 makes every other guarantee depend on
// -- and can publish into a room, which reaches teammates' context windows.
//
// Demonstrated before this existed: a page served from another port marked an
// unverified peer verified with one POST. No ceremony, no two words, no call. Its
// Content-Type was text/plain, which is a "simple request" and so is sent without
// the browser asking permission first.
//
// The rule is REQUIRE, not refuse: a request must positively present a header only
// our own code sends. A browser cannot send a custom header to another origin
// without a preflight, and we answer preflights with nothing, so the real request
// is never sent. Measured: the OPTIONS arrived carrying
// `Access-Control-Request-Headers: x-cogmer`, and no POST followed.
//
// Requiring presence rather than refusing a bad value is what makes this
// fail-closed. Anything that cannot present the header is refused, whatever it is,
// so no argument is needed about which headers a browser can be made to omit.
//
// NOT Referer, which was tested and is useless here. A page suppresses its own
// referrer with one <meta> tag -- measured: Referer absent, Origin still present --
// and we must treat an absent Referer as allowed because the CLI and hooks send
// none. The check would pass exactly when it needed to fail. A server redirect does
// not help either: the referrer is the initiating document and survives the
// redirect unchanged, which was also measured. Do not add it back.
//
// This does not defend against a malicious PROGRAM running as this user; such a
// program can set any header, and could edit membership.db directly in any case.
// The threat closed here is a web page.

const (
	localGuardHeader = "X-Cogmer"
	localGuardValue  = "1"
)

// guardLocal wraps a state-changing local route.
func guardLocal(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// POST only. A navigation, an <img src> or a <link> carries no custom
		// header and could never pass the check below -- but refusing the method
		// outright means those never reach a handler that reads a body at all.
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// An Origin from anywhere but our own view is a page, and a page has no
		// business here. Absent is normal: the CLI and the hooks are not browsers.
		if origin := r.Header.Get("Origin"); origin != "" && !isOwnOrigin(origin) {
			http.Error(w, "cross-origin request refused", http.StatusForbidden)
			return
		}
		// The load-bearing check.
		if r.Header.Get(localGuardHeader) != localGuardValue {
			http.Error(w, "this endpoint is not reachable from a web page", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

// isOwnOrigin reports whether an Origin header names this daemon's own view.
//
// Both spellings of loopback count, and so does IPv6: a person who types
// localhost:4782 is looking at the same page as one who types 127.0.0.1:4782, and
// a view that worked only for the spelling we happened to print would be a bug
// nobody could diagnose from the page.
func isOwnOrigin(origin string) bool {
	host, port, err := net.SplitHostPort(addr())
	if err != nil {
		return false
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		return false // not loopback-only; refuse rather than guess
	}
	for _, h := range loopbackSpellings(host) {
		if origin == "http://"+net.JoinHostPort(h, port) {
			return true
		}
	}
	return false
}

func loopbackSpellings(host string) []string {
	switch host {
	case "127.0.0.1", "localhost", "::1", "[::1]":
		return []string{"127.0.0.1", "localhost", "::1"}
	}
	return []string{host}
}

// localPost is how OUR code talks to OUR daemon. Both the CLI and the hooks go
// through here, so the header is attached in one place rather than remembered in
// two.
func localPost(client *http.Client, url string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(localGuardHeader, localGuardValue)
	return client.Do(req)
}
