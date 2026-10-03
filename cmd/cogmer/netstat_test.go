package main

import (
	"reflect"
	"testing"
)

const netstatSample = `
Active Connections

  Proto  Local Address          Foreign Address        State           PID
  TCP    0.0.0.0:135            0.0.0.0:0              LISTENING       1032
  TCP    127.0.0.1:4783         0.0.0.0:0              LISTENING       5120
  TCP    127.0.0.1:47830        0.0.0.0:0              LISTENING       777
  TCP    [::]:4783              [::]:0                 LISTENING       5120
  TCP    [::1]:4783             [::]:0                 ABHÖREN         6001
  TCP    127.0.0.1:50000        127.0.0.1:4783         ESTABLISHED     8888
  UDP    0.0.0.0:4783           *:*                                    9999
`

func TestNetstatListenersAreFoundByTheirForeignAddress(t *testing.T) {
	if got, want := parseNetstatListeners(netstatSample, "4783"), []int{5120, 6001}; !reflect.DeepEqual(got, want) {
		t.Errorf("listeners on 4783 = %v, want %v: a port number inside another port (47830), an established connection and a UDP socket must not count, and a machine set to German prints another word for the state", got, want)
	}
	if got := parseNetstatListeners(netstatSample, "9"); len(got) != 0 {
		t.Errorf("a port that nothing listens on matched %v", got)
	}
	if got := parseNetstatListeners("", "4783"); len(got) != 0 {
		t.Errorf("no output matched %v", got)
	}
}

func TestTasklistNamesTheProcessOrNothing(t *testing.T) {
	cases := []struct{ out, want string }{
		{`"cogmer.exe","5120","Console","1","24,512 K"` + "\r\n", "cogmer.exe"},
		{`"claude.exe","77","Console","1","1 K"`, "claude.exe"},
		{"INFO: No tasks are running which match the specified criteria.\r\n", ""},
		{"INFORMATION : aucune tâche en cours ne correspond aux critères spécifiés.", ""},
		{"", ""},
	}
	for _, c := range cases {
		if got := parseTasklistName(c.out); got != c.want {
			t.Errorf("parseTasklistName(%q) = %q, want %q", c.out, got, c.want)
		}
	}
}

func TestACogmerImageIsRecognizedOnEveryPlatform(t *testing.T) {
	for name, want := range map[string]bool{"cogmer": true, "cogmer.exe": true, "COGMER.EXE": true, "cogmer.test": false, "not-cogmer": false, "": false} {
		if got := isCogmerImage(name); got != want {
			t.Errorf("isCogmerImage(%q) = %v, want %v", name, got, want)
		}
	}
}
