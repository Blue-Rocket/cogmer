package main

import (
	"errors"
	"fmt"
	"io"
	"os"
)

// --- the commands' side of the user identity ---

// identityPageURL is the address of the identity page, at a section of it.
func identityPageURL(section string) string {
	u := "http://" + addr() + "/identity"
	if section != "" {
		u += "#" + section
	}
	return u
}

// userIdentityGateOff turns the gate below off, for a test of a command's behavior on
// a machine with no user identity, as COGMER_PREFLIGHT=off turns the behavior check
// off for CI.
func userIdentityGateOff() bool { return os.Getenv("COGMER_IDENTITY_PAGE") == "off" }

// needsUserIdentity is the gate a command that needs the user identity passes first:
// printing a first pairing string, pairing, and creating or joining a room. Until the
// identity exists the command says that it is made in the daemon's page, opens the
// page as pairing opens the view (D-088), and does nothing else, because the phrase is
// shown and typed only there and a person is there only once they have acted (D-230, a
// user identity is made in the daemon's page the first time the user needs one). It
// reports whether the command must stop, and the command then exits 0, as every
// command that cannot proceed does.
func needsUserIdentity(id *Identity) bool {
	if id.HasUser() || userIdentityGateOff() {
		return false
	}
	writeIdentityNeeded(os.Stdout)
	return true
}

func writeIdentityNeeded(w io.Writer) {
	fmt.Fprintln(w, "Nothing was done yet: this needs your cogmer identity, which you make once, in a page")
	fmt.Fprintln(w, "in your browser rather than here, because it shows your recovery phrase and nothing")
	fmt.Fprintln(w, "typed here should ever hold it.")
	fmt.Fprintln(w)
	openPage(w, identityPageURL(""))
	fmt.Fprintln(w)
	fmt.Fprintln(w, "The page asks first whether you have used cogmer on another machine. When it is")
	fmt.Fprintln(w, "done, run this command again.")
}

// openPage opens a page of the daemon in the browser, or says why it could not. The
// address holds nothing secret, so it is printed either way.
func openPage(w io.Writer, url string) {
	if !daemonAlreadyServing(addr()) {
		fmt.Fprint(w, commandText("The cogmer daemon is not running, and the page needs it. It starts with an agent\n"+
			"session, and /cogmer:self-restart starts it without one.\n"))
		return
	}
	if err := openInBrowser(url); err != nil {
		fmt.Fprintf(w, "%v. Open this address in a browser on this machine:\n  %s\n", err, url)
		return
	}
	fmt.Fprintf(w, "Opened the page in your browser:\n  %s\n", url)
}

// runMachines opens the page where the user's machines are added, renamed and
// removed, and where the user recovers (D-231, a user's machines are managed from one
// command that opens the view). A machine with no user identity is shown the page's
// first question instead.
func runMachines() {
	id, err := LoadIdentity()
	if err != nil {
		fmt.Printf("identity: %v\n", err)
		return
	}
	if !id.HasUser() {
		writeIdentityNeeded(os.Stdout)
		return
	}
	fmt.Println("Your machines are managed in a page in your browser, where adding one shows two words")
	fmt.Println("to compare and nothing passes through this session.")
	fmt.Println()
	openPage(os.Stdout, identityPageURL("machines"))
}

// writeOwnMachines lists the user's machines for self-status, with whether each can be
// reached from this machine (D-231). Reach comes from the daemon; without one the
// machines are listed from this machine's records with reach unknown.
func writeOwnMachines(w io.Writer, m *Membership, id *Identity) {
	if !id.HasUser() {
		return
	}
	var st identityState
	if err := postLocal("/identity/state", struct{}{}, &st); err != nil || !st.HasUser {
		certs, current := m.displayCerts(id.User)
		st.Machines = nil
		for k, c := range certs {
			st.Machines = append(st.Machines, machineView{Machine: k, Name: c.Name, Serial: c.Serial, Self: k == id.PeerID, Current: current[k]})
		}
		sortMachineViews(st.Machines)
	}
	if len(st.Machines) == 0 {
		return
	}
	fmt.Fprintln(w, "\nYour machines:")
	for _, mv := range st.Machines {
		fmt.Fprintf(w, "  %-20s #%-3d %s\n", mv.Name, mv.Serial, machineReachText(mv))
	}
	fmt.Fprint(w, commandText("Add, rename or remove one with /cogmer:self-machines.\n"))
}

func machineReachText(mv machineView) string {
	switch {
	case mv.Self:
		return "this machine"
	case !mv.Current:
		return "suspended until it is added again"
	case mv.Reach == "not reachable" && mv.Trying != "":
		return "not reachable from here (tried for " + mv.Trying + ")"
	case mv.Reach == "":
		return "reach not known; the daemon did not answer"
	case mv.Reach == "not tried":
		return "not tried yet; no address for it here"
	case mv.Reach == reachUnknown:
		return "being checked"
	}
	return mv.Reach
}

func sortMachineViews(v []machineView) {
	for i := range v {
		for j := i + 1; j < len(v); j++ {
			if v[j].Serial < v[i].Serial || (v[j].Serial == v[i].Serial && v[j].Machine < v[i].Machine) {
				v[i], v[j] = v[j], v[i]
			}
		}
	}
}

// joinFromList records a room named on the user's room list and not yet here, so a
// session on another of the user's machines joins it by name (D-228). It reports
// whether the list held the room.
func joinFromList(m *Membership, id *Identity, name string, showOnly bool) (bool, error) {
	if !id.HasUser() {
		return false, nil
	}
	if _, err := m.FindRoom(name); !errors.Is(err, errNoSuchRoom) {
		return false, nil
	}
	roomID, v, ok := m.RoomOnList(id.User, name)
	if !ok {
		return false, nil
	}
	return true, m.JoinFromList(id, roomID, v, showOnly)
}
