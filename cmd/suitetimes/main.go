// Command suitetimes reads `go test -json` on its standard input, prints what plain
// `go test` would have printed, and writes each test's time to the file it is given,
// slowest first. scripts/test-and-record.sh runs the suite through it, so every
// recorded run also says where the suite's time went.
//
//	go test -json ./... | go run ./cmd/suitetimes <times file>
//
// It exits 1 when any test or package failed, and 2 when it cannot write the file.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

type event struct {
	Action  string
	Package string
	Test    string
	Elapsed *float64
	Output  string
}

type timing struct {
	seconds float64
	name    string
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: go test -json ./... | suitetimes <times file>")
		os.Exit(2)
	}
	failed := false
	// A test's output is held until the test ends, and printed only when it failed,
	// as plain go test does.
	held := map[string][]string{}
	var times []timing
	in := bufio.NewScanner(os.Stdin)
	in.Buffer(make([]byte, 1024*1024), 64*1024*1024)
	for in.Scan() {
		line := in.Bytes()
		var e event
		if err := json.Unmarshal(line, &e); err != nil {
			// Build failures and other text go test prints outside its JSON.
			fmt.Println(string(line))
			continue
		}
		key := e.Package + " " + e.Test
		switch {
		case e.Action == "build-output":
			// The compiler's message for a package that did not build.
			fmt.Print(e.Output)
		case e.Action == "build-fail":
			failed = true
		case e.Test == "" && e.Action == "output":
			// Package lines: "ok", "FAIL", "?", and a package's own output.
			if !strings.HasPrefix(e.Output, "=== ") && !strings.HasPrefix(e.Output, "--- PASS") &&
				!strings.HasPrefix(e.Output, "PASS\n") {
				fmt.Print(e.Output)
			}
		case e.Test == "" && e.Action == "fail":
			failed = true
		case e.Action == "output":
			held[key] = append(held[key], e.Output)
		case e.Test != "" && (e.Action == "pass" || e.Action == "fail" || e.Action == "skip"):
			if e.Action == "fail" {
				failed = true
				fmt.Print(strings.Join(held[key], ""))
			}
			delete(held, key)
			if e.Elapsed != nil && e.Action != "skip" {
				times = append(times, timing{*e.Elapsed, e.Package + " " + e.Test})
			}
		}
	}
	if err := in.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "suitetimes:", err)
		failed = true
	}

	sort.Slice(times, func(i, j int) bool { return times[i].seconds > times[j].seconds })
	var b strings.Builder
	fmt.Fprintln(&b, "# seconds each test took, slowest first; a subtest's time is also in its parent's")
	for _, t := range times {
		fmt.Fprintf(&b, "%8.2f  %s\n", t.seconds, t.name)
	}
	if err := os.WriteFile(os.Args[1], []byte(b.String()), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "suitetimes:", err)
		os.Exit(2)
	}
	if failed {
		os.Exit(1)
	}
}
