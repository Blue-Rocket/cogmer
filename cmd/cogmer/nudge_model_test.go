package main

import (
	"fmt"
	"testing"
)

// The search below models polling in rounds, and asks, of every arrangement it can form,
// whether a turn written on a machine that answers reaches every machine that answers
// within a bounded number of rounds (D-306, a machine nudges the machines it shares a room
// with when it writes an event). Which machines are polled is decided by the daemon's own
// nudgedBy, setAside and setFailing, one daemon for each modelled machine, so the model
// cannot disagree with the code about which machine is set aside.
//
// What it forms:
//   - a colleague with one to three machines, the reader's user with one or two, and
//     sometimes a third user with one machine, all in one room;
//   - an order in which the colleague's machines nudged before the turn, which decides
//     which of them each machine has set aside;
//   - the machine that writes the turn, any machine of any user;
//   - which machines, other than the writer, do not answer;
//   - which nudges of the turn are lost for good, as when the writer cannot reach a
//     machine that can reach it. A nudge that is lost and later arrives is sent again
//     until it does, which the model takes as arriving one round late.
//
// A poll of a machine that answers takes every event it holds, and a poll of one that
// does not answer fails, as polls do over a network that carries them.

// strandBound is how many rounds a turn may take to reach every machine that answers.
const strandBound = 6

type modelMachine struct {
	name, user string
	up         bool
	d          *Daemon
	holds      bool // holds the turn
}

func newModelMachine(name, user string) *modelMachine {
	return &modelMachine{name: name, user: user, up: true,
		d: &Daemon{id: &Identity{PeerID: name, User: user, Chain: &Chain{}}}}
}

// modelRound has every machine that answers poll every other machine it does not set
// aside, with what each held at the start of the round.
func modelRound(ms []*modelMachine) {
	held := map[string]bool{}
	for _, m := range ms {
		held[m.name] = m.holds
	}
	for _, m := range ms {
		if !m.up {
			continue
		}
		for _, o := range ms {
			if o == m || m.d.setAside(o.name) {
				continue
			}
			if !o.up {
				m.d.pollFailed(o.name)
				continue
			}
			m.d.pollAnswered(o.name)
			if held[o.name] {
				m.holds = true
			}
		}
	}
}

func TestNoArrangementStrandsATurnOnAMachineThatAnswers(t *testing.T) {
	checked, most := 0, 3
	if testing.Short() {
		most = 2 // a colleague with three machines takes most of the four seconds
	}
	for colleague := 1; colleague <= most; colleague++ {
		for readers := 1; readers <= 2; readers++ {
			for _, third := range []bool{false, true} {
				names := func() []*modelMachine {
					var ms []*modelMachine
					for i := 0; i < colleague; i++ {
						ms = append(ms, newModelMachine(fmt.Sprintf("ana-%d", i), "ana"))
					}
					for i := 0; i < readers; i++ {
						ms = append(ms, newModelMachine(fmt.Sprintf("ben-%d", i), "ben"))
					}
					if third {
						ms = append(ms, newModelMachine("cal-0", "cal"))
					}
					return ms
				}
				n := len(names())
				for _, order := range nudgeOrders(colleague) {
					for writer := 0; writer < n; writer++ {
						for down := 0; down < 1<<n; down++ {
							if down&(1<<writer) != 0 {
								continue
							}
							for lost := 0; lost < 1<<n; lost++ {
								if lost&(1<<writer) != 0 {
									continue
								}
								checked++
								if why := strands(names(), order, writer, down, lost); why != "" {
									t.Fatalf("a colleague with %d machines, a reader's user with %d, third user %v, colleague's nudges in order %v, writer %d, down %b, nudges lost %b: %s",
										colleague, readers, third, order, writer, down, lost, why)
								}
							}
						}
					}
				}
			}
		}
	}
	t.Logf("checked %d arrangements", checked)
}

// strands plays one arrangement and says which machine that answers lacks the turn after
// strandBound rounds, or nothing.
func strands(ms []*modelMachine, order []int, writer, down, lost int) string {
	// Every machine heard the colleague's machines nudge in this order, while all answered.
	for _, i := range order {
		for _, m := range ms {
			if m != ms[i] {
				m.d.nudgedBy(ms[i].user, ms[i].name)
			}
		}
	}
	for i, m := range ms {
		m.up = down&(1<<i) == 0
	}
	w := ms[writer]
	w.holds = true
	var late []*modelMachine
	for i, m := range ms {
		if m == w || !m.up || lost&(1<<i) != 0 {
			continue
		}
		late = append(late, m)
	}
	for round := 0; round < strandBound; round++ {
		if round == 1 { // a nudge sent again arrives one round late
			for _, m := range late {
				m.d.nudgedBy(w.user, w.name)
			}
		}
		modelRound(ms)
	}
	for _, m := range ms {
		if m.up && !m.holds {
			return fmt.Sprintf("%s answers and lacks the turn written on %s after %d rounds", m.name, w.name, strandBound)
		}
	}
	return ""
}

// nudgeOrders is every order in which some of a colleague's machines nudged, including
// none.
func nudgeOrders(n int) [][]int {
	out := [][]int{nil}
	var grow func(prefix []int, used int)
	grow = func(prefix []int, used int) {
		for i := 0; i < n; i++ {
			if used&(1<<i) != 0 {
				continue
			}
			next := append(append([]int(nil), prefix...), i)
			out = append(out, next)
			grow(next, used|1<<i)
		}
	}
	grow(nil, 0)
	return out
}
