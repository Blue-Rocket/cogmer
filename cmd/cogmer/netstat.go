package main

import (
	"encoding/csv"
	"strconv"
	"strings"
)

// Windows has no lsof, so the process holding a port is found from `netstat -ano`
// and `tasklist`. The parsing lives here, without a build tag, so that it is tested
// on every machine; only stop_windows.go runs the programs.

// parseNetstatListeners returns the pids of processes listening on a TCP port, from
// the output of `netstat -ano -p tcp`.
//
// A listening socket is recognized by its foreign address, 0.0.0.0:0 or [::]:0,
// and not by the word LISTENING, because Windows prints the state in the language
// the machine is set to.
func parseNetstatListeners(out, port string) []int {
	var pids []int
	seen := map[int]bool{}
	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(line)
		if len(f) < 5 || !strings.EqualFold(f[0], "TCP") {
			continue
		}
		local, foreign := f[1], f[2]
		if foreign != "0.0.0.0:0" && foreign != "[::]:0" && foreign != "*:*" {
			continue
		}
		i := strings.LastIndex(local, ":")
		if i < 0 || local[i+1:] != port {
			continue
		}
		pid, err := strconv.Atoi(f[len(f)-1])
		if err != nil || seen[pid] {
			continue
		}
		seen[pid] = true
		pids = append(pids, pid)
	}
	return pids
}

// parseTasklistName returns the image name from `tasklist /FO CSV /NH` for one
// process, or "" when no process matched. A machine set to another language prints
// a sentence in place of the row, which is not CSV with a pid in its second field.
func parseTasklistName(out string) string {
	r := csv.NewReader(strings.NewReader(strings.TrimSpace(out)))
	r.FieldsPerRecord = -1
	rec, err := r.Read()
	if err != nil || len(rec) < 2 {
		return ""
	}
	if _, err := strconv.Atoi(strings.TrimSpace(rec[1])); err != nil {
		return ""
	}
	return strings.TrimSpace(rec[0])
}

// isCogmerImage reports whether a process image is this program: cogmer on a
// Unix, and cogmer.exe on Windows, whatever the case.
func isCogmerImage(name string) bool {
	return strings.EqualFold(strings.TrimSuffix(strings.ToLower(name), ".exe"), "cogmer")
}
