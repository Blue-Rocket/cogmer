package main

import (
	"fmt"
	"os/exec"
	"runtime"
)

// openInBrowser puts a page in front of the person at this machine.
//
// This is the only channel from here to a person's eyes that does not pass through
// the model (D-033, D-036), and it was measured to work from a process started
// detached by a hook: the browser comes to the front, and a fresh address opens a
// fresh tab (B23, D-082).
//
// It returns an error instead of failing quietly, because every caller has a
// fallback and choosing it requires knowing this did not work. A machine reached
// over SSH, a container, a server has no browser, and pairing has to remain possible
// there -- so the terminal ceremony is not dead code, it is the other branch.
func openInBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("could not open a browser (%w)", err)
	}
	return nil
}
