package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const hostReminder = "<system-reminder>The user's timezone is Europe/Lisbon (UTC+00:00). Message sent at Wed 2026-11-25 13:01 local time.</system-reminder>\n"

// A Cowork cloud task puts the host's own reminder before every prompt. What the user typed is
// what is left (D-300).
func TestTheHostsReminderIsNotPartOfWhatTheUserTyped(t *testing.T) {
	for name, c := range map[string]struct{ in, want string }{
		"no reminder":           {"hello", "hello"},
		"a reminder":            {hostReminder + "hello", "hello"},
		"two reminders":         {hostReminder + hostReminder + "hello", "hello"},
		"a reminder over lines": {"<system-reminder>a\nb</system-reminder>\n\n/cogmer:self-status", "/cogmer:self-status"},
		"only a reminder":       {hostReminder, hostReminder},
		"one later in the text": {"hello " + hostReminder, "hello " + hostReminder},
		"an unclosed reminder":  {"<system-reminder>never closed hello", "<system-reminder>never closed hello"},
	} {
		if got := withoutHostReminders(c.in); got != c.want {
			t.Errorf("%s: %q, want %q", name, got, c.want)
		}
	}
	cmd, ok := parseTypedCommand(hostReminder+"/cogmer:room-invite bob", "/cogmer:")
	if !ok || cmd.name != "room-invite" || len(cmd.args) != 1 || cmd.args[0] != "bob" {
		t.Errorf("a command after the host's reminder was read as %+v, %v", cmd, ok)
	}
	if _, ok := parseTypedCommand(hostReminder+"tell me about /cogmer:room-invite", "/cogmer:"); ok {
		t.Error("a prompt that only mentions a command was taken for one")
	}
}

// A command the user typed is no turn of the room, and neither is what the model says about it, with
// the host's reminder before it or not. An ordinary prompt is recorded as typed, without the reminder.
func TestACommandAfterTheHostsReminderIsNoTurnOfTheRoom(t *testing.T) {
	d, room, post := promptRoom(t, false)
	stop := func(promptID, message string) {
		t.Helper()
		b, _ := json.Marshal(stopReq{Host: "claude", SessionID: "s", PromptID: promptID, LastAssistantMessage: message})
		r := httptest.NewRequest(http.MethodPost, "/hook/stop", bytes.NewReader(b))
		r.Header.Set(localGuardHeader, localGuardValue)
		w := httptest.NewRecorder()
		d.LocalRoutes().ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("/hook/stop: %d %s", w.Code, w.Body.String())
		}
	}
	events := func() []Event {
		t.Helper()
		store, err := d.storeFor(room.RoomID)
		if err != nil {
			t.Fatal(err)
		}
		evs, err := store.ListRoom(room.RoomID)
		if err != nil {
			t.Fatal(err)
		}
		return evs
	}

	post(promptReq{Host: "claude", SessionID: "s", PromptID: "c1", Prompt: hostReminder + "/cogmer:room-invite bob"})
	stop("c1", "Told bob.")
	if got := events(); len(got) != 0 {
		t.Fatalf("a command after the host's reminder left %d events in the room, first %q", len(got), got[0].Content)
	}

	post(promptReq{Host: "claude", SessionID: "s", PromptID: "p1", Prompt: hostReminder + "what changed?"})
	stop("p1", "Two things.")
	evs := events()
	if len(evs) != 2 || evs[0].Content != "what changed?" || strings.Contains(evs[0].Content, "system-reminder") {
		t.Errorf("an ordinary prompt was recorded as %+v, want what the user typed and the reply", evs)
	}
}

// A prompt held while the daemon starts is checked as the live one is.
func TestAHeldCommandAfterTheHostsReminderIsNotRecordedByTheDaemonEither(t *testing.T) {
	d, room, _ := promptRoom(t, false)
	if d.capturePrompt(promptReq{Host: "claude", SessionID: "s", PromptID: "h1", Prompt: hostReminder + "/cogmer:room-log"}) {
		t.Error("a held command was recorded")
	}
	if !d.capturePrompt(promptReq{Host: "claude", SessionID: "s", PromptID: "h2", Prompt: hostReminder + "held words"}) {
		t.Fatal("a held prompt was not recorded")
	}
	store, _ := d.storeFor(room.RoomID)
	evs, _ := store.ListRoom(room.RoomID)
	if len(evs) != 1 || evs[0].Content != "held words" {
		t.Errorf("the room holds %+v, want the held words alone", evs)
	}
}
