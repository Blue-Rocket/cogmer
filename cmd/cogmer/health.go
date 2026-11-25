package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// `health` is the in-session form of the doctor (D-298). It answers at once and spends no model
// turn: the three versions, whether a daemon answers, and what the behavior check recorded for this
// host's version. `health extended` starts the whole check, which drives real turns, in the
// background and answers at once, and a later `health` shows what it found.
const (
	// extendedLogPrefix begins the name of the log that holds the output of a background run of the
	// whole check, after a first line that says when it was started. Each host has a log of its own,
	// since one host's result says nothing about another's.
	extendedLogPrefix = "doctor-extended-"
	// extendedWithin is how long a run that has not finished is taken to be still running. The
	// check takes about 20 seconds, so a run past this has died.
	extendedWithin = 3 * time.Minute
	extendedStart  = "started "
)

// extendedProgram is the program the whole check runs in, as a variable so a test can stand in for it.
var extendedProgram = os.Executable

func extendedLogPath(h Host) string {
	return filepath.Join(homeDir(), extendedLogPrefix+h.Name()+".log")
}

// extendedRun is what the log of the whole check says.
type extendedRun struct {
	Started  time.Time
	Finished bool
	Findings []string // the FAIL lines and the closing count
}

// readExtendedRun reads the log, and reports false when no run was ever started.
func readExtendedRun(h Host) (extendedRun, bool) {
	b, err := os.ReadFile(extendedLogPath(h))
	if err != nil {
		return extendedRun{}, false
	}
	var run extendedRun
	for _, line := range strings.Split(string(b), "\n") {
		switch {
		case strings.HasPrefix(line, extendedStart):
			run.Started, _ = time.Parse(time.RFC3339, strings.TrimPrefix(line, extendedStart))
		case strings.Contains(line, " checked, ") && strings.HasSuffix(line, " failed"):
			run.Finished = true
			run.Findings = append(run.Findings, strings.TrimSpace(line))
		case strings.HasPrefix(strings.TrimSpace(line), "FAIL"):
			run.Findings = append(run.Findings, strings.TrimSpace(line))
		}
	}
	return run, true
}

// verificationKey is what the checks of the current host are recorded under, and the version
// that names it.
func verificationKey(h Host) (key, version string) {
	switch h.Name() {
	case "codex":
		v := CodexVersion()
		return "codex:" + v, v
	case "cowork":
		return coworkVerificationKey(), ClaudeVersion()
	default:
		v := ClaudeVersion()
		return v, v
	}
}

// failedBehaviors names each failed behavior with its title, in order.
func failedBehaviors(failures map[string]string) []string {
	ids := make([]string, 0, len(failures))
	for id := range failures {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var out []string
	for _, id := range ids {
		title := ""
		for _, b := range Behaviors {
			if b.ID == id {
				title = b.Title
			}
		}
		out = append(out, strings.TrimSpace(id+" "+title))
	}
	return out
}

// checksReport says what the behavior check recorded for the host this session runs in.
func checksReport(h Host) string {
	if inCloudContainer() {
		return "  The checks are not run in a cloud container.\n"
	}
	key, version := verificationKey(h)
	if version == "unknown" || version == "" {
		return fmt.Sprintf("  %s is not installed on this machine, so there is nothing to check.\n", h.Name())
	}
	v, ok := loadVerifications()[key]
	switch {
	case !ok:
		return fmt.Sprintf("  %s %s has not been checked. The first room formed on it checks it.\n", h.Name(), version)
	case len(v.Failures) == 1 && v.Failures[probeCouldNotRun] != "":
		return fmt.Sprintf("  The check could not run on %s %s: %s. It is tried again after an hour.\n",
			h.Name(), version, v.Failures[probeCouldNotRun])
	case len(v.Failures) > 0:
		return fmt.Sprintf("  %s %s failed %d behavior(s) that cogmer relies on, found %s:\n    %s\n",
			h.Name(), version, len(v.Failures), v.When, strings.Join(failedBehaviors(v.Failures), "\n    "))
	default:
		return fmt.Sprintf("  %s %s passed every check, found %s.\n", h.Name(), version, v.When)
	}
}

// extendedReport says what the whole check found, or that one is running.
func extendedReport(h Host, now time.Time) string {
	run, ok := readExtendedRun(h)
	switch {
	case !ok:
		return ""
	case run.Finished:
		return "\nThe whole check, started " + run.Started.Format(time.RFC3339) + ":\n  " + strings.Join(run.Findings, "\n  ") + "\n"
	case now.Sub(run.Started) < extendedWithin:
		return "\nThe whole check is running, started " + run.Started.Format(time.RFC3339) + ". Run this command again in a minute.\n"
	default:
		return "\nThe whole check started " + run.Started.Format(time.RFC3339) + " did not finish.\n"
	}
}

// healthReport is the instant report.
func healthReport(h Host, now time.Time) string {
	running, answering := daemonVersionAt(addr())
	var b strings.Builder
	b.WriteString(versionReport(pluginVersion(), version, running, answering))
	b.WriteString("\nWhat the behavior check recorded:\n")
	b.WriteString(checksReport(h))
	b.WriteString(extendedReport(h, now))
	if m, err := OpenMembership(); err == nil {
		defer m.Close()
		if id, err := LoadIdentity(); err == nil {
			if line := nextStepLine(m, id); line != "" {
				b.WriteString("\n" + line + "\n")
			}
		}
	}
	if !inCloudContainer() {
		b.WriteString("\nThe whole check drives real turns on your subscription and takes about 20 seconds. " +
			"Start it with `extended`, and run this command again to read what it found.\n")
	}
	return b.String()
}

// startExtendedRun starts the whole check detached, with its output in the log, and says what it
// did. A run that is still going is left alone.
func startExtendedRun(h Host, now time.Time) string {
	if inCloudContainer() {
		return "The whole check is not run in a cloud container.\n"
	}
	if run, ok := readExtendedRun(h); ok && !run.Finished && now.Sub(run.Started) < extendedWithin {
		return "The whole check is already running, started " + run.Started.Format(time.RFC3339) + ".\n"
	}
	exe, err := extendedProgram()
	if err != nil {
		return "The whole check could not be started: " + err.Error() + "\n"
	}
	if err := ensureStateDir(); err != nil {
		return "The whole check could not be started: " + err.Error() + "\n"
	}
	f, err := os.OpenFile(extendedLogPath(h), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return "The whole check could not be started: " + err.Error() + "\n"
	}
	defer f.Close()
	fmt.Fprintf(f, "%s%s\n", extendedStart, now.UTC().Format(time.RFC3339))
	cmd := exec.Command(exe, "doctor", "--host", h.Name())
	cmd.Stdin = nil
	cmd.Stdout, cmd.Stderr = f, f
	detach(cmd)
	if err := cmd.Start(); err != nil {
		return "The whole check could not be started: " + err.Error() + "\n"
	}
	_ = cmd.Process.Release()
	return "Started the whole check. It drives real turns and takes about 20 seconds.\n"
}

// runHealth reads `health [extended]`.
func runHealth(args []string) {
	h, err := currentHost()
	if err != nil || h == nil {
		h = claudeHost{}
	}
	now := time.Now()
	if len(args) > 0 {
		if args[0] != "extended" || len(args) > 1 {
			fmt.Printf("health takes no argument, or `extended`; it was given %q.\n", strings.Join(args, " "))
			return
		}
		fmt.Print(startExtendedRun(h, now), "\n")
		return
	}
	fmt.Print(healthReport(h, now))
}
