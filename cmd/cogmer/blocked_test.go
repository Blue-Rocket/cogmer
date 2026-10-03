package main

import (
	"errors"
	"strings"
	"testing"
)

// A blocked address is reported by what holds it. A cogmer daemon left running
// from another state directory was reported as "something that is not a cogmer
// daemon", which sent the reader looking for a different program.
func TestABlockedAddressSaysWhatHoldsIt(t *testing.T) {
	cause := errors.New("address already in use")
	const stop = "~/.cogmer/bin/cogmer"
	cases := []struct {
		name     string
		holders  []holder
		findErr  error
		detail   []string
		advice   []string
		notInAny []string
	}{
		{"another cogmer daemon", []holder{{PID: 41, Name: "cogmer"}}, nil,
			[]string{"held by another cogmer daemon", "pid 41"},
			[]string{"Stop it:  ~/.cogmer/bin/cogmer stop"},
			[]string{"not a cogmer daemon", "lsof"}},
		{"some other program", []holder{{PID: 7, Name: "nginx"}}, nil,
			[]string{"held by nginx", "pid 7", "not a cogmer daemon"},
			[]string{"leaves a process that is not a cogmer daemon alone"},
			nil},
		{"a holder with no name", []holder{{PID: 9}}, nil,
			[]string{"a process with no name", "pid 9"}, nil, nil},
		{"nothing can be found", nil, errors.New("lsof: not found"),
			[]string{"is in use", "could not say what holds it", "lsof: not found"},
			[]string{"Find what holds it"},
			[]string{"not a cogmer daemon"}},
		{"nothing is found and no error", nil, nil,
			[]string{"is in use", "could not say what holds it"},
			[]string{"Find what holds it"},
			[]string{"not a cogmer daemon"}},
		{"a cogmer daemon among others", []holder{{PID: 3, Name: "nginx"}, {PID: 4, Name: "cogmer"}}, nil,
			[]string{"another cogmer daemon", "pid 4"}, []string{"stop"}, nil},
	}
	for _, c := range cases {
		detail, advice := describeBlock("peer sync", "127.0.0.1:4783", cause, c.holders, c.findErr, stop)
		all := detail + "\n" + strings.Join(advice, "\n")
		for _, w := range c.detail {
			if !strings.Contains(detail, w) {
				t.Errorf("%s: detail lacks %q: %s", c.name, w, detail)
			}
		}
		for _, w := range c.advice {
			if !strings.Contains(all, w) {
				t.Errorf("%s: advice lacks %q: %s", c.name, w, all)
			}
		}
		for _, w := range c.notInAny {
			if strings.Contains(all, w) {
				t.Errorf("%s: says %q, which is not true here: %s", c.name, w, all)
			}
		}
		if !strings.Contains(all, "address already in use") {
			t.Errorf("%s: the cause is lost: %s", c.name, all)
		}
		if !strings.Contains(all, "COGMER_PEER_ADDR") {
			t.Errorf("%s: no way to move this daemon is offered: %s", c.name, all)
		}
	}
}
