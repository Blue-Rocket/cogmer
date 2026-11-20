package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"
)

// roomSummary is one room as the rooms pane lists it (D-289, every page of the view lists every
// room in a pane beside the room it shows). Everything in it is read from the
// snapshot the room's own page streams, so the list and the room never disagree.
type roomSummary struct {
	Name     string `json:"name"`
	ShowOnly bool   `json:"showOnly,omitempty"`
	// Sessions is how many sessions on this machine are in the room now.
	Sessions int `json:"sessions"`
	// LastAt is when the room's newest turn was written, in RFC 3339, empty in a room
	// with no turn. LastBy names who wrote it as the room page names them, and LastMine
	// says it was this user's.
	LastAt   string `json:"lastAt,omitempty"`
	LastBy   string `json:"lastBy,omitempty"`
	LastMine bool   `json:"lastMine,omitempty"`
	// Theirs counts the colleagues' turns in the room that count, which the page compares
	// with the count it saw when the reader last looked, to say how many are new.
	Theirs int `json:"theirs"`
	// Waiting counts the turns no session on this machine has been given (D-204).
	Waiting int `json:"waiting"`
	// Colleagues is how many colleagues the room shows, and Reachable how many of them
	// this machine can reach now.
	Colleagues int `json:"colleagues"`
	Reachable  int `json:"reachable"`
	// Removed says the room's list expels this user (D-241).
	Removed bool `json:"removed,omitempty"`
}

// roomSummaries lists every room on this machine, the most recently active first and
// rooms with no turn last, by name among equals.
func (d *Daemon) roomSummaries() ([]roomSummary, error) {
	rooms, err := d.members.Rooms()
	if err != nil {
		return nil, err
	}
	out := make([]roomSummary, 0, len(rooms))
	for _, r := range rooms {
		st, err := d.snapshot(r)
		if err != nil {
			return nil, err
		}
		out = append(out, summarize(r, st))
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].LastAt != out[j].LastAt {
			return out[i].LastAt > out[j].LastAt
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

func summarize(r Room, st uiState) roomSummary {
	s := roomSummary{Name: r.RoomName, ShowOnly: r.ShowOnly, Sessions: st.SessionsInRoom, Removed: st.Removed != nil}
	var last time.Time
	for _, e := range st.Events {
		if isGuestEventType(e.EventType) || e.Marked != "" {
			continue
		}
		if !e.Mine {
			s.Theirs++
		}
		if e.NotInSession {
			s.Waiting++
		}
		t, err := time.ParseInLocation("2006-01-02 15:04:05", e.At, time.Local)
		if err != nil || !t.After(last) {
			continue
		}
		last = t
		s.LastAt = t.Format(time.RFC3339)
		s.LastMine = e.Mine
		s.LastBy = shownAuthor(e)
	}
	for _, p := range st.Peers {
		if p.Own {
			continue
		}
		s.Colleagues++
		if p.Online {
			s.Reachable++
		}
	}
	return s
}

// shownAuthor names a turn's writer as the room page does: the label this user gave, then
// the name the writer gave themselves, then the name derived from the key (D-094).
func shownAuthor(e uiEvent) string {
	switch {
	case e.Mine:
		return "you"
	case e.Label != "":
		return e.Label
	case e.Display != "":
		return e.Display
	}
	return e.PeerName
}

// handleRooms answers with the list of rooms. It is a read of what each room's stream
// sends, so it is not guarded (D-087).
func (d *Daemon) handleRooms(w http.ResponseWriter, r *http.Request) {
	list, err := d.roomSummaries()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, list)
}

// handleRoomsStream pushes the list of rooms whenever any room changes, as /stream does
// for one room.
func (d *Daemon) handleRoomsStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := d.subscribe()
	defer d.unsubscribe(ch)
	send := func() bool {
		list, err := d.roomSummaries()
		if err != nil {
			return false
		}
		buf, err := json.Marshal(list)
		if err != nil {
			return false
		}
		fmt.Fprintf(w, "data: %s\n\n", buf)
		flusher.Flush()
		return true
	}
	if !send() {
		return
	}
	// Reachability changes without a room changing, so the list is also sent on the
	// heartbeat, which keeps intermediaries from closing an idle stream as well.
	beat := time.NewTicker(20 * time.Second)
	defer beat.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ch:
			if !send() {
				return
			}
		case <-beat.C:
			if !send() {
				return
			}
		}
	}
}
