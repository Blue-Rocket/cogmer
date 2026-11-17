// Command containerdriver runs inside a container of the container tests, where it
// does what a person's browser and a host's files would do on that machine.
//
// The daemon's local routes listen on loopback and require the header our own code
// sends, so a page act such as creating a user identity or confirming two words can
// only be made from inside the container. This program makes the same request the
// page makes, and prints the reply for the test to read. It knows no route of its
// own, so one build drives every version of the daemon.
//
// It is copied into an image built FROM scratch, which has no shell, so it also
// writes the files a test needs in the container, such as a session's transcript.
//
//	containerdriver post <path>   POST stdin as JSON to the daemon, print the reply
//	containerdriver get <path>    GET from the daemon, print the reply
//	containerdriver write <file>  write stdin to a file, making its directory
package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func main() {
	if len(os.Args) != 3 {
		fail("usage: containerdriver post|get <path> | write <file>")
	}
	switch os.Args[1] {
	case "post":
		body, err := io.ReadAll(os.Stdin)
		if err != nil {
			fail("reading stdin: %v", err)
		}
		request(http.MethodPost, os.Args[2], body)
	case "get":
		request(http.MethodGet, os.Args[2], nil)
	case "write":
		body, err := io.ReadAll(os.Stdin)
		if err != nil {
			fail("reading stdin: %v", err)
		}
		if err := os.MkdirAll(filepath.Dir(os.Args[2]), 0o700); err != nil {
			fail("%v", err)
		}
		if err := os.WriteFile(os.Args[2], body, 0o600); err != nil {
			fail("%v", err)
		}
	default:
		fail("unknown act %q", os.Args[1])
	}
}

// request sends what the page's script sends: JSON, and the header a web page from
// another origin cannot send.
func request(method, path string, body []byte) {
	address := os.Getenv("COGMER_ADDR")
	if address == "" {
		address = "127.0.0.1:4782"
	}
	req, err := http.NewRequest(method, "http://"+address+path, bytes.NewReader(body))
	if err != nil {
		fail("%v", err)
	}
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Cogmer", "1")
	}
	// Adding a machine waits up to three minutes for the other page's answer.
	client := &http.Client{Timeout: 4 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		fail("%v", err)
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		fail("%s: %s", resp.Status, bytes.TrimSpace(out))
	}
	_, _ = os.Stdout.Write(out)
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
