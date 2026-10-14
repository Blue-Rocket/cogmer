package main

import (
	"sync"
	"time"
)

// hostCalls remembers when each host last made a hook call of this daemon. A Cowork
// session reaches the daemon only through a name that Anthropic chooses (D-205), and a
// name that stops matching stops every call without anything failing, so the age of the
// last call is the one place the loss shows.
type hostCalls struct {
	mu   sync.Mutex
	last map[string]time.Time
}

func (h *hostCalls) note(host string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.last == nil {
		h.last = map[string]time.Time{}
	}
	h.last[host] = time.Now()
}

// ages is how long ago each host last called, in the words roughDuration uses.
func (h *hostCalls) ages() map[string]string {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := map[string]string{}
	for host, at := range h.last {
		out[host] = roughDuration(time.Since(at))
	}
	return out
}
