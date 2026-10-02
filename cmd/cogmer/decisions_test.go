package main

import (
	"regexp"
	"testing"
)

// The decision log ties decisions to the checks that would invalidate them
// ("Revisit when B04 fires"). If a behavior is renamed or removed, those
// pointers rot silently and the log starts citing checks that no longer run.
func TestDecisionLogReferencesRealBehaviors(t *testing.T) {
	data := readDecisionLog(t)
	known := map[string]bool{}
	for _, b := range Behaviors {
		known[b.ID] = true
	}
	found := regexp.MustCompile(`\bB\d{2}\b`).FindAllString(data, -1)
	if len(found) == 0 {
		t.Fatal("decision log cites no behaviors; the revisit triggers are missing")
	}
	for _, id := range found {
		if !known[id] {
			t.Errorf("decision log cites %s, which is not in the registry", id)
		}
	}
}
