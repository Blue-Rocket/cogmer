package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// The person is shown one line when the first install has finished (D-282, the person is shown
// one line when the install finishes and when cogmer needs repair). The installer writes its
// mark, the file install-notice holding the version, and the first hook that runs afterwards
// shows the line and removes the mark, so that it is shown once.

func installNoticePath() string { return filepath.Join(homeDir(), "install-notice") }

// installNoticeText is the line that says a first install has finished and what to run next.
func installNoticeText(version string) string {
	name := "cogmer"
	if version != "" {
		name += " " + version
	}
	return name + " is installed. Run /cogmer:self-status to get the string to send a colleague " +
		"and to check that colleagues can reach you."
}

// takeInstallNotice returns the line for a first install that has finished and has not been
// announced, and removes the mark, or "" when there is none.
func takeInstallNotice() string {
	raw, err := os.ReadFile(installNoticePath())
	if err != nil {
		return ""
	}
	_ = os.Remove(installNoticePath())
	version := strings.TrimSpace(strings.SplitN(string(raw), "\n", 2)[0])
	return installNoticeText(version)
}

// runNotice is `cogmer notice`, which the session-start hook runs when the installer left its
// mark: it prints the line, once.
func runNotice() {
	if line := takeInstallNotice(); line != "" {
		fmt.Println(line)
	}
}

// noticeShower is a host that shows the person a line from a hook, as Claude Code shows a hook's
// systemMessage on the screen (D-282). A host that is not one gets the note for the model alone.
type noticeShower interface {
	// NoticeHookOutput is what a prompt hook writes to its standard output so that the host
	// shows the person notice and gives the model context, when there is any.
	NoticeHookOutput(notice, context string) []byte
}

// NoticeHookOutput is the JSON Claude Code reads a hook's message and context from.
func (claudeHost) NoticeHookOutput(notice, context string) []byte {
	out := map[string]any{"systemMessage": notice}
	if context != "" {
		out["hookSpecificOutput"] = map[string]string{"hookEventName": "UserPromptSubmit", "additionalContext": context}
	}
	b, _ := json.Marshal(out)
	return append(b, '\n')
}

// noticeFor takes the line for a first install that has finished, for a host that shows one,
// and leaves the mark alone for one that does not, so that a later host that does is told.
func noticeFor(host Host) string {
	if _, ok := host.(noticeShower); !ok {
		return ""
	}
	return takeInstallNotice()
}

// promptHookOutput is what a prompt hook writes: the notice and the context in the form the
// host shows and reads when it has a notice to show, and otherwise the context alone.
func promptHookOutput(host Host, notice, context string) []byte {
	if s, ok := host.(noticeShower); ok && notice != "" {
		return s.NoticeHookOutput(notice, context)
	}
	if context != "" {
		return host.PromptHookOutput(context)
	}
	return nil
}
