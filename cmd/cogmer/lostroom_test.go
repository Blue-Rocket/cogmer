package main

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A room's events live in a database of their own, and this peer's position in its
// numbering lives in the membership store, so losing the first leaves a room that
// has simply gone quiet (D-029). The daemon says so when it opens a room, because
// nothing else would (D-169). These tests restart the daemon, which is when a room
// is opened afresh, over a room that has lost its database and over one that has
// not.

func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	old := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(old) })
	return &buf
}

// sendInto records n events this peer sent in a room, then stops the daemon.
func sendInto(t *testing.T, d *Daemon, room Room, n int) {
	t.Helper()
	store, err := d.storeFor(room.RoomID)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < n; i++ {
		if _, err := d.appendLocal(store, room, "sess-1", EventUserPrompt, "a turn", nil); err != nil {
			t.Fatal(err)
		}
	}
	d.closeStores()
}

// restarted is a new daemon over the same identity and membership, which opens
// every room afresh.
func restarted(t *testing.T, d *Daemon) *Daemon {
	t.Helper()
	d2 := &Daemon{id: d.id, members: d.members}
	t.Cleanup(d2.closeStores)
	return d2
}

func TestALostRoomIsReportedWhenItIsOpened(t *testing.T) {
	d, room := testDaemon(t)
	sendInto(t, d, room, 3)

	for _, suffix := range []string{"", "-wal", "-shm"} {
		err := os.Remove(filepath.Join(homeDir(), "rooms", room.RoomID+".db"+suffix))
		if err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(filepath.Join(homeDir(), "rooms", room.RoomID+".db")); err == nil {
		t.Fatal("the room's database is still there, so nothing was lost")
	}

	buf := captureLog(t)
	if _, err := restarted(t, d).storeFor(room.RoomID); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"ROOM STATE LOST", room.RoomName, "holds your events up to 0, but you issued 3", "New events resume above 3"} {
		if !strings.Contains(out, want) {
			t.Errorf("the notice lacks %q, so a user whose room went quiet is told nothing:\n%s", want, out)
		}
	}
}

func TestARoomHoldingWhatWasIssuedReportsNothing(t *testing.T) {
	d, room := testDaemon(t)
	sendInto(t, d, room, 3)

	buf := captureLog(t)
	if _, err := restarted(t, d).storeFor(room.RoomID); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "ROOM STATE LOST") {
		t.Errorf("a room that holds everything sent was reported lost:\n%s", buf.String())
	}
}
