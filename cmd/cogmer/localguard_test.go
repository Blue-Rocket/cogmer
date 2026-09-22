package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// reached records whether the wrapped handler ran at all. A guard that returns the
// right status while still letting the handler mutate state would be worse than
// none, so every case asserts on this rather than only on the code.
func guarded(t *testing.T) (http.HandlerFunc, *bool) {
	t.Helper()
	reached := false
	return guardLocal(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	}), &reached
}

func call(t *testing.T, h http.HandlerFunc, method string, headers map[string]string) (int, bool) {
	t.Helper()
	req := httptest.NewRequest(method, "/verify/confirm", strings.NewReader(`{"peer":"x","matched":true}`))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec.Code, false
}

func TestLocalGuardRefusesAWebPage(t *testing.T) {
	t.Setenv("COGMER_ADDR", "127.0.0.1:4782")

	// The demonstrated attack: a page on another local port, simple request, so no
	// preflight. It cannot send our header, and it cannot hide its Origin.
	h, reached := guarded(t)
	code, _ := call(t, h, http.MethodPost, map[string]string{
		"Origin":       "http://127.0.0.1:4799",
		"Content-Type": "text/plain",
	})
	if code != http.StatusForbidden {
		t.Fatalf("hostile page got %d, want 403", code)
	}
	if *reached {
		t.Fatal("hostile request reached the handler; state would have changed")
	}
}

func TestLocalGuardRefusesAMissingHeader(t *testing.T) {
	t.Setenv("COGMER_ADDR", "127.0.0.1:4782")
	// No Origin at all -- the shape a non-browser attacker would try. Requiring
	// presence is what makes this fail closed rather than open.
	h, reached := guarded(t)
	code, _ := call(t, h, http.MethodPost, map[string]string{"Content-Type": "application/json"})
	if code != http.StatusForbidden {
		t.Fatalf("header-less request got %d, want 403", code)
	}
	if *reached {
		t.Fatal("header-less request reached the handler")
	}
}

func TestLocalGuardRefusesNonPost(t *testing.T) {
	t.Setenv("COGMER_ADDR", "127.0.0.1:4782")
	// A navigation or an <img src> carries no custom header anyway, but these must
	// not reach a handler that reads a body.
	for _, m := range []string{http.MethodGet, http.MethodHead, http.MethodPut} {
		h, reached := guarded(t)
		code, _ := call(t, h, m, map[string]string{localGuardHeader: localGuardValue})
		if code != http.StatusMethodNotAllowed {
			t.Fatalf("%s got %d, want 405", m, code)
		}
		if *reached {
			t.Fatalf("%s reached the handler", m)
		}
	}
}

func TestLocalGuardAdmitsOurOwnClients(t *testing.T) {
	t.Setenv("COGMER_ADDR", "127.0.0.1:4782")

	// The CLI and the hooks: our header, no Origin, because they are not browsers.
	h, reached := guarded(t)
	if code, _ := call(t, h, http.MethodPost, map[string]string{localGuardHeader: localGuardValue}); code != http.StatusOK {
		t.Fatalf("CLI-shaped request got %d, want 200", code)
	}
	if !*reached {
		t.Fatal("CLI-shaped request did not reach the handler")
	}

	// The view itself, under every spelling of loopback a person might type. A
	// guard that admitted only the spelling we happened to print would be a bug
	// nobody could diagnose from the page.
	for _, origin := range []string{
		"http://127.0.0.1:4782",
		"http://localhost:4782",
		"http://[::1]:4782",
	} {
		h, reached := guarded(t)
		code, _ := call(t, h, http.MethodPost, map[string]string{
			localGuardHeader: localGuardValue,
			"Origin":         origin,
		})
		if code != http.StatusOK {
			t.Fatalf("own view at %s got %d, want 200", origin, code)
		}
		if !*reached {
			t.Fatalf("own view at %s did not reach the handler", origin)
		}
	}
}

// A page that knows the header name still cannot use it: the browser preflights,
// and we answer nothing. This asserts the server half of that -- that the guard
// does not treat an OPTIONS as permission.
func TestLocalGuardDoesNotAnswerPreflight(t *testing.T) {
	t.Setenv("COGMER_ADDR", "127.0.0.1:4782")
	h, reached := guarded(t)
	req := httptest.NewRequest(http.MethodOptions, "/verify/confirm", nil)
	req.Header.Set("Origin", "http://127.0.0.1:4799")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "x-cogmer")
	rec := httptest.NewRecorder()
	h(rec, req)
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("preflight was granted CORS permission; the header check becomes bypassable")
	}
	if *reached {
		t.Fatal("preflight reached the handler")
	}
}
