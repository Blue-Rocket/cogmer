package main

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

func useHome(t *testing.T) string {
	t.Helper()
	home := filepath.Join(t.TempDir(), ".cogmer")
	t.Setenv("COGMER_HOME", home)
	return home
}

func writeVersionFile(t *testing.T, home, text string) {
	t.Helper()
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, dataVersionFile), []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}

// A directory with no version holds version 0, and a version this program knows, the
// same or lower, is read as itself (D-244, the data in the state directory carries
// one version).
func TestADataVersionAbsentEqualOrOlderIsRead(t *testing.T) {
	home := useHome(t)
	if v, err := storedDataVersion(); v != 0 || err != nil {
		t.Errorf("an absent version read as %d, %v; want 0", v, err)
	}
	for _, v := range []int{0, dataVersion} {
		writeVersionFile(t, home, strconv.Itoa(v)+"\n")
		if got, err := storedDataVersion(); got != v || err != nil {
			t.Errorf("version %d read as %d, %v", v, got, err)
		}
	}
}

// A version higher than this program knows, or one it cannot read, is data it must
// not touch.
func TestANewerOrUnreadableDataVersionIsRefused(t *testing.T) {
	home := useHome(t)
	for _, text := range []string{strconv.Itoa(dataVersion + 1), "two", "-1"} {
		writeVersionFile(t, home, text)
		_, err := storedDataVersion()
		var newer *NewerDataError
		if !errors.As(err, &newer) {
			t.Errorf("a data version of %q was not refused: %v", text, err)
		}
	}
}

// An older directory is brought up to this program's version: every room store under
// rooms/ is migrated, whether or not anything would have opened it, and only then is
// the version written.
func TestUpgradingMigratesEveryRoomStoreBeforeWritingTheVersion(t *testing.T) {
	if dataVersion == 0 {
		t.Skip("a build that knows no data version has nothing to upgrade to")
	}
	home := useHome(t)
	dir := filepath.Join(home, "rooms")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	old, err := sql.Open("sqlite", filepath.Join(dir, "room-old.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := old.Exec(eventsSchemaBeforeSigner); err != nil {
		t.Fatal(err)
	}
	old.Close()

	stored, err := storedDataVersion()
	if err != nil {
		t.Fatal(err)
	}
	if err := upgradeStateDirectory(stored); err != nil {
		t.Fatalf("upgrade: %v", err)
	}
	if v, err := storedDataVersion(); v != dataVersion || err != nil {
		t.Errorf("after an upgrade the version reads %d, %v; want %d", v, err, dataVersion)
	}
	db, err := sql.Open("sqlite", filepath.Join(dir, "room-old.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('events') WHERE name = 'signer'`).Scan(&n); err != nil || n != 1 {
		t.Errorf("a room store nothing opened was left unmigrated behind version %d: %v", dataVersion, err)
	}
	if leftovers, _ := filepath.Glob(filepath.Join(home, "."+dataVersionFile+"-*")); len(leftovers) != 0 {
		t.Errorf("writing the version left %v behind", leftovers)
	}
}

// A migration that fails leaves the version as it was, so the data never claims a
// version it has not reached.
func TestAFailedMigrationLeavesTheVersionUnwritten(t *testing.T) {
	if dataVersion == 0 {
		t.Skip("a build that knows no data version has nothing to upgrade to")
	}
	home := useHome(t)
	dir := filepath.Join(home, "rooms")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "room-broken.db"), bytes.Repeat([]byte("not a database "), 512), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := upgradeStateDirectory(0); err == nil {
		t.Fatal("a room store that cannot be opened was reported as migrated")
	}
	if _, err := os.Stat(filepath.Join(home, dataVersionFile)); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a failed migration wrote the version: %v", err)
	}
}

// Writing the version never lowers one a newer program recorded meanwhile.
func TestWritingTheVersionNeverLowersIt(t *testing.T) {
	home := useHome(t)
	writeVersionFile(t, home, strconv.Itoa(dataVersion+1))
	_ = writeDataVersion(dataVersion)
	raw, _ := os.ReadFile(filepath.Join(home, dataVersionFile))
	if strings.TrimSpace(string(raw)) != strconv.Itoa(dataVersion+1) {
		t.Errorf("the version was lowered to %q", raw)
	}
}

// snapshot is every file under dir, by path, with a hash of its bytes.
func snapshot(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(b)
		rel, _ := filepath.Rel(dir, p)
		out[rel] = hex.EncodeToString(sum[:])
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func sameSnapshot(t *testing.T, what string, before, after map[string]string) {
	t.Helper()
	var diff []string
	for p, h := range after {
		if before[p] != h {
			diff = append(diff, p)
		}
	}
	for p := range before {
		if _, ok := after[p]; !ok {
			diff = append(diff, p+" (removed)")
		}
	}
	sort.Strings(diff)
	if len(diff) > 0 {
		t.Errorf("%s changed the state directory: %v", what, diff)
	}
}

// On data newer than it knows, the program touches nothing: a hook is silent and
// quick, a command says the plugin needs updating, and the daemon does not start.
func TestAProgramOnNewerDataTouchesNothing(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the binary")
	}
	bin := buildVersion(t, t.TempDir(), "0.0.1-newer-data")
	home := filepath.Join(t.TempDir(), ".cogmer")
	if err := os.MkdirAll(filepath.Join(home, "rooms"), 0o700); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		dataVersionFile:   strconv.Itoa(dataVersion+1) + "\n",
		"identity.json":   `{"peerId":"x","future":"a field this program does not know"}`,
		"membership.db":   "written by a newer cogmer",
		"rooms/room-a.db": "written by a newer cogmer",
		"identity.key":    "a newer key format",
	} {
		if err := os.WriteFile(filepath.Join(home, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	before := snapshot(t, home)
	env := append(os.Environ(), "COGMER_HOME="+home, "COGMER_ADDR="+freeAddr(t), "COGMER_PEER_ADDR="+freeAddr(t),
		"COGMER_TAILCAT=off", "COGMER_HOST=claude", "CLAUDE_CODE_SESSION_ID=s-1")

	for _, kind := range []string{"prompt", "stop"} {
		cmd := exec.Command(bin, "hook", kind)
		cmd.Env = env
		cmd.Stdin = strings.NewReader(`{"session_id":"s-1","prompt":"hello","transcript_path":"/nonexistent"}`)
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		start := time.Now()
		err := cmd.Run()
		if err != nil || stdout.Len() != 0 || stderr.Len() != 0 {
			t.Errorf("the %s hook on newer data was not silent: %v, stdout %q, stderr %q", kind, err, stdout.String(), stderr.String())
		}
		if took := time.Since(start); took > 2*time.Second {
			t.Errorf("the %s hook on newer data took %s", kind, took)
		}
	}

	cmd := exec.Command(bin, "whoami")
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "written by a newer cogmer") || !strings.Contains(string(out), "Updating the cogmer plugin") {
		t.Errorf("a command on newer data did not say the plugin needs updating: %v\n%s", err, out)
	}

	daemon := exec.Command(bin, "daemon")
	daemon.Env = env
	done := make(chan []byte, 1)
	go func() { b, _ := daemon.CombinedOutput(); done <- b }()
	select {
	case b := <-done:
		if !strings.Contains(string(b), "not starting") {
			t.Errorf("the daemon on newer data did not say why it stopped:\n%s", b)
		}
	case <-time.After(10 * time.Second):
		_ = daemon.Process.Kill()
		t.Fatal("the daemon on newer data started")
	}

	sameSnapshot(t, "a program on newer data", before, snapshot(t, home))
}

// A bridge on newer data relays nothing, answers a typed command with the refusal, and
// answers every other call with nothing.
func TestABridgeOnNewerDataRelaysNothing(t *testing.T) {
	d := &fakeDaemon{}
	var out bytes.Buffer
	b := &bridge{out: &out, post: d.post, deadline: time.Second, grace: time.Second, exit: func(int) {}, refusal: "update the plugin"}
	call := func(id int, tool, prompt string) string {
		out.Reset()
		params, _ := json.Marshal(map[string]any{"name": tool, "arguments": map[string]any{"session_id": "s", "prompt_id": "p", "prompt": prompt}})
		b.call(bridgeMessage{ID: json.RawMessage(strconv.Itoa(id)), Method: "tools/call", Params: params})
		return out.String()
	}
	if got := call(1, "prompt", "/cogmer:room-status"); !strings.Contains(got, "update the plugin") {
		t.Errorf("a typed command was not given the refusal: %s", got)
	}
	for i, tc := range []struct{ tool, prompt string }{{"prompt", "an ordinary prompt"}, {"stop", ""}} {
		if got := call(i+2, tc.tool, tc.prompt); strings.Contains(got, "update the plugin") || !strings.Contains(got, `"text":""`) {
			t.Errorf("the %s call was not answered with nothing: %s", tc.tool, got)
		}
	}
	if calls := d.recorded(); len(calls) != 0 {
		t.Errorf("a bridge on newer data relayed %d call(s) to a daemon", len(calls))
	}
}
