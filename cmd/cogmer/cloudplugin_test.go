package main

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// A cloud session's daemon takes its cloud value from the plugin files its account syncs
// into the session (D-279, a cloud session's daemon reads its cloud value from the plugin
// files it finds, as well as from the variable).

// writePluginValue puts value in a file named cloud.value in a folder named plugin under
// an account folder under root, as a session's host lays out a synced plugin.
func writePluginValue(t *testing.T, root, plugin, value string) {
	t.Helper()
	dir := filepath.Join(root, "acct_1", plugin)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, pluginValueFile), []byte(value), 0o600); err != nil {
		t.Fatal(err)
	}
}

// cloudValueFor adds a cloud key on a, through the daemon and not the page, and returns
// its value and the cloud key's identifier. The page adds one at a time (D-280, a user has
// one cloud key at a time), and a daemon that finds several is what these tests need.
func cloudValueFor(t *testing.T, a *machine, name string) (string, string) {
	t.Helper()
	value, key, err := a.d.addCloud(name)
	if err != nil {
		t.Fatal(err)
	}
	return value, key
}

func TestTheNewestPluginValueThatVerifiesIsTheOneTakenAndTheRestAreDiscarded(t *testing.T) {
	a, _ := twoMachines(t)
	older, _ := cloudValueFor(t, a, "older")
	newer, newerKey := cloudValueFor(t, a, "newer")
	newest, _ := cloudValueFor(t, a, "newest")
	name := cloudPluginName(a.d.id.User)
	root := t.TempDir()
	writePluginValue(t, root, name, older)
	writePluginValue(t, root, name+"~g2", newer)
	// A plugin that is not named as the page names its plugin is never opened, even when it
	// holds the newest value, and neither is a file deeper than a plugin's folder.
	writePluginValue(t, root, "copy-of-the-newest", newest)
	writePluginValue(t, root, name+"-copy", newest)
	if err := os.MkdirAll(filepath.Join(root, "acct_1", name+"~g9", "deeper"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "acct_1", name+"~g9", "deeper", pluginValueFile), []byte(newest), 0o600); err != nil {
		t.Fatal(err)
	}
	// A plugin that is named right and holds no value that reads is discarded, as is a
	// file too large to be one, and a value in a plugin named for another user's digits.
	writePluginValue(t, root, name+"~g3", "not a cloud value")
	big := filepath.Join(root, "acct_1", name+"~g4")
	if err := os.MkdirAll(big, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(big, pluginValueFile), bytes.Repeat([]byte("x"), pluginValueMax+1), 0o600); err != nil {
		t.Fatal(err)
	}
	writePluginValue(t, root, cloudPluginPrefix+"00000000", newest)

	best, discarded, mixed := newestPluginValue(root)
	if best == nil || best.setup.cloud.Machine != newerKey || mixed {
		t.Fatalf("the value taken is %v, want the newest of the plugins named as the page names them", best)
	}
	if discarded != 3 {
		t.Errorf("%d files were discarded, want the garbage, the oversized one and the one for other digits", discarded)
	}
	if got, _, _ := newestPluginValue(t.TempDir()); got != nil {
		t.Error("a folder with no plugin holds a value")
	}
	if got, _, _ := newestPluginValue(""); got != nil {
		t.Error("no folder holds a value")
	}
}

func TestTwoPluginValuesOfOneSerialAreToldApartByTheirKeysIdentifiers(t *testing.T) {
	a, b := twoMachines(t)
	va, ka := cloudValueFor(t, a, "from-a")
	vb, kb := cloudValueFor(t, b, "from-b")
	sa, _ := parseCloudValue(va)
	sb, _ := parseCloudValue(vb)
	if sa.cloud.Serial != sb.cloud.Serial {
		t.Skipf("the two machines issued serials %d and %d", sa.cloud.Serial, sb.cloud.Serial)
	}
	want := ka
	if kb < ka {
		want = kb
	}
	name := cloudPluginName(a.d.id.User)
	for _, order := range [][2]string{{va, vb}, {vb, va}} {
		root := t.TempDir()
		writePluginValue(t, root, name, order[0])
		writePluginValue(t, root, name+"~g2", order[1])
		if best, _, _ := newestPluginValue(root); best == nil || best.setup.cloud.Machine != want {
			t.Fatalf("with values of one serial the daemon took %v, want the key that sorts first", best)
		}
	}
}

func TestValuesOfTwoUsersAreNeverChosenBetweenAndSaidOnceInTheLog(t *testing.T) {
	var logs bytes.Buffer
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	a, _ := twoMachines(t)
	b := newMachine(t)
	phrase, err := NewPhrase()
	if err != nil {
		t.Fatal(err)
	}
	createUser(t, b, phrase, "Eli", "tablet")
	// The other user's value has the higher serial, which says nothing across users.
	va, _ := cloudValueFor(t, a, "mine")
	vb, _ := cloudValueFor(t, b, "theirs")
	if a.d.id.User == b.d.id.User {
		t.Fatal("the two machines are one user")
	}
	root := t.TempDir()
	writePluginValue(t, root, cloudPluginName(a.d.id.User), va)
	writePluginValue(t, root, cloudPluginName(b.d.id.User), vb)

	if best, _, mixed := newestPluginValue(root); best != nil || !mixed {
		t.Fatalf("values of two users gave %v, mixed %v", best, mixed)
	}
	c := newMachine(t)
	c.d.noCloudLoop = true
	c.d.pluginRoot = root
	c.d.startCloudFromPlugins()
	c.d.takeNewerPluginValue()
	c.d.takeNewerPluginValue()
	if c.d.cloud != nil || c.d.isCloudSession() {
		t.Error("a daemon given the values of two users took one")
	}
	if n := strings.Count(logs.String(), "name more than one user"); n != 1 {
		t.Errorf("the log says it %d times, want once", n)
	}
	words, _ := cloudValueSaid(t, root)
	if !strings.Contains(words, "name more than one user") && !strings.Contains(words, "more than one user") {
		t.Errorf("a command says %q", words)
	}
}

func TestAPluginNamedForAnotherUsersDigitsIsDiscarded(t *testing.T) {
	a, _ := twoMachines(t)
	b := newMachine(t)
	phrase, err := NewPhrase()
	if err != nil {
		t.Fatal(err)
	}
	createUser(t, b, phrase, "Eli", "tablet")
	value, _ := cloudValueFor(t, a, "mine")
	root := t.TempDir()
	writePluginValue(t, root, cloudPluginName(b.d.id.User), value)
	if best, discarded, _ := newestPluginValue(root); best != nil || discarded != 1 {
		t.Errorf("a value in a plugin named for another user gave %v, discarding %d", best, discarded)
	}
}

func TestADaemonTakesAPluginValueWhenItAppearsAndANewerOneWhenThatAppears(t *testing.T) {
	var logs bytes.Buffer
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	a, _ := twoMachines(t)
	older, olderKey := cloudValueFor(t, a, "older")
	newer, newerKey := cloudValueFor(t, a, "newer")
	name := cloudPluginName(a.d.id.User)
	root := t.TempDir()

	c := newMachine(t)
	c.d.noCloudLoop = true
	c.d.pluginRoot = root
	c.d.startCloudFromPlugins()
	c.d.takeNewerPluginValue()
	if c.d.cloud != nil || c.d.isCloudSession() {
		t.Fatal("a daemon with no value in any file became a cloud session's")
	}

	// The plugin syncs after the session began, and the next prompt takes its value.
	writePluginValue(t, root, name, older)
	c.d.takeNewerPluginValue()
	if c.d.cloud == nil || c.d.cloud.setup.cloud.Machine != olderKey || c.d.id.Chain.Session == nil {
		t.Fatalf("the daemon did not take the value that appeared: %+v", c.d.cloud)
	}
	if c.d.id.Chain.Session.CloudKey != olderKey {
		t.Errorf("the session key is certified by %s, not by the value's cloud key", c.d.id.Chain.Session.CloudKey)
	}

	// An upload replaces it with a newer one, which the next prompt takes.
	writePluginValue(t, root, name+"~g2", newer)
	c.d.takeNewerPluginValue()
	if c.d.cloud.setup.cloud.Machine != newerKey || c.d.id.Chain.Session.CloudKey != newerKey {
		t.Fatalf("the daemon kept the older value after a newer one appeared")
	}

	// A plugin removed, or an older value that appears, ends nothing: the daemon holds the
	// value it read, and only removing the cloud key ends its authority.
	if err := os.RemoveAll(filepath.Join(root, "acct_1")); err != nil {
		t.Fatal(err)
	}
	c.d.takeNewerPluginValue()
	writePluginValue(t, root, name+"~g3", older)
	c.d.takeNewerPluginValue()
	if c.d.cloud == nil || c.d.cloud.setup.cloud.Machine != newerKey {
		t.Error("the daemon dropped or lowered the value it held")
	}
	for _, f := range []string{older, newer, strings.TrimPrefix(older, cloudValuePrefix), strings.TrimPrefix(newer, cloudValuePrefix)} {
		if strings.Contains(logs.String(), f) {
			t.Fatal("the daemon's log holds a cloud value")
		}
	}
}

func TestADaemonHoldsItsUsersValueAndIgnoresAnotherUsersWhateverItsSerial(t *testing.T) {
	var logs bytes.Buffer
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	a, _ := twoMachines(t)
	b := newMachine(t)
	phrase, err := NewPhrase()
	if err != nil {
		t.Fatal(err)
	}
	createUser(t, b, phrase, "Eli", "tablet")
	mine, mineKey := cloudValueFor(t, a, "mine")
	theirs, _ := cloudValueFor(t, b, "theirs")
	for i := 0; i < 3; i++ { // so that their serial is the higher
		cloudValueFor(t, b, "more")
	}
	theirs, _ = cloudValueFor(t, b, "latest")

	root := t.TempDir()
	c := newMachine(t)
	c.d.noCloudLoop = true
	c.d.pluginRoot = root
	writePluginValue(t, root, cloudPluginName(a.d.id.User), mine)
	c.d.takeNewerPluginValue()
	if c.d.cloud == nil || c.d.cloud.setup.cloud.Machine != mineKey {
		t.Fatal("the daemon did not take its user's value")
	}
	if err := os.RemoveAll(filepath.Join(root, "acct_1")); err != nil {
		t.Fatal(err)
	}
	writePluginValue(t, root, cloudPluginName(b.d.id.User), theirs)
	c.d.takeNewerPluginValue()
	c.d.takeNewerPluginValue()
	if c.d.cloud.setup.cloud.Machine != mineKey || c.d.cloud.setup.user != a.d.id.User {
		t.Error("another user's value replaced the one the daemon held")
	}
	if n := strings.Count(logs.String(), "for another user, which is ignored"); n != 1 {
		t.Errorf("the log says it %d times, want once", n)
	}
}

func TestADaemonWithNoPluginRootLooksNowhere(t *testing.T) {
	a, _ := twoMachines(t)
	value, _ := cloudValueFor(t, a, "web")
	c := newMachine(t)
	c.d.noCloudLoop = true
	root := t.TempDir()
	writePluginValue(t, root, cloudPluginName(a.d.id.User), value)
	c.d.takeNewerPluginValue()
	if c.d.cloud != nil {
		t.Error("a daemon that was given no folder to look in took a value")
	}
	t.Setenv("COGMER_SYNCED_PLUGINS", root)
	if got := syncedPluginsRoot(); got != root {
		t.Errorf("the folder is %q, want the one the environment names", got)
	}
}

// cloudValueSaid is what a command says in a cloud session whose plugin files are under root.
func cloudValueSaid(t *testing.T, root string) (string, bool) {
	t.Helper()
	t.Setenv("CLAUDE_CODE_REMOTE", "true")
	t.Setenv(cloudKeyVar, "")
	t.Setenv("COGMER_SYNCED_PLUGINS", root)
	words, stopped := gateWords(t, &Identity{})
	return strings.Join(strings.Fields(words), " "), stopped
}

// The words a command gives in a cloud session that has no identity (D-281, a command in a
// cloud session with no value says the user must first set cogmer up on one of their own
// computers).

// gateWords runs the identity gate as a command does and returns what it printed and
// whether it stopped the command.
func gateWords(t *testing.T, id *Identity) (string, bool) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	real := os.Stdout
	os.Stdout = w
	stopped := needsUserIdentity(id)
	os.Stdout = real
	w.Close()
	var out bytes.Buffer
	if _, err := out.ReadFrom(r); err != nil {
		t.Fatal(err)
	}
	return out.String(), stopped
}

func TestACloudSessionWithNoValueSaysItSharesNothingAndSendsTheUserToTheirOwnComputer(t *testing.T) {
	t.Setenv("CLAUDE_CODE_REMOTE", "true")
	t.Setenv(cloudKeyVar, "")
	t.Setenv("COGMER_SYNCED_PLUGINS", t.TempDir())
	words, stopped := gateWords(t, &Identity{})
	if !stopped {
		t.Fatal("a command went on with no identity")
	}
	flat := strings.Join(strings.Fields(words), " ")
	for _, want := range []string{"shares nothing", "no cloud value was found", cloudKeyVar, pluginValueFile, "own computers", "self-machines", "Install in Claude Code"} {
		if !strings.Contains(flat, want) {
			t.Errorf("the words do not say %q: %s", want, words)
		}
	}
	for _, forbidden := range []string{"recovery phrase", "phrase", "127.0.0.1", "http://", "browser", "Opened the page"} {
		if strings.Contains(strings.ToLower(words), strings.ToLower(forbidden)) {
			t.Errorf("the words mention %q: %s", forbidden, words)
		}
	}
}

func TestACloudSessionSaysWhetherAValueItFoundReadsAndNeverQuotesIt(t *testing.T) {
	a, _ := twoMachines(t)
	value, _ := cloudValueFor(t, a, "web")
	t.Setenv("CLAUDE_CODE_REMOTE", "true")
	t.Setenv(cloudKeyVar, "")

	root := t.TempDir()
	t.Setenv("COGMER_SYNCED_PLUGINS", root)
	writePluginValue(t, root, cloudPluginName(a.d.id.User), value)
	words, stopped := gateWords(t, &Identity{})
	if flat := strings.Join(strings.Fields(words), " "); !stopped || !strings.Contains(flat, "it reads") || !strings.Contains(flat, pluginValueFile) || !strings.Contains(flat, "next prompt") {
		t.Errorf("a value in a plugin file that reads was said as: %s", words)
	}

	if err := os.RemoveAll(filepath.Join(root, "acct_1")); err != nil {
		t.Fatal(err)
	}
	writePluginValue(t, root, cloudPluginName(a.d.id.User), "garbage that is no cloud value")
	words, _ = gateWords(t, &Identity{})
	if flat := strings.Join(strings.Fields(words), " "); !strings.Contains(flat, "does not read") || !strings.Contains(flat, "self-machines") {
		t.Errorf("a plugin file that does not read was said as: %s", words)
	}

	t.Setenv(cloudKeyVar, "cogmer-cloud-v1:notbase64!!")
	words, _ = gateWords(t, &Identity{})
	if flat := strings.Join(strings.Fields(words), " "); !strings.Contains(flat, cloudKeyVar) || !strings.Contains(flat, "does not read") || strings.Contains(words, "notbase64") {
		t.Errorf("a variable that does not read was said as: %s", words)
	}
	for _, w := range []string{value, strings.TrimPrefix(value, cloudValuePrefix)} {
		if strings.Contains(words, w) {
			t.Fatal("the words quote the value")
		}
	}
}

// The machines page offers the cloud value as a plugin (D-278, the machines page offers the
// cloud value as a plugin for the user to upload once) and a user has one cloud key at a
// time (D-280, a user has one cloud key at a time).

func TestTheCloudAddResponseOffersAPluginThatHoldsOnlyItsManifestAndTheValue(t *testing.T) {
	var logs bytes.Buffer
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	a, _ := twoMachines(t)
	var res cloudAddResponse
	a.local(t, "/machines/cloud/add", cloudAddRequest{Name: "web"}, &res)
	if res.Error != "" || res.Plugin == "" {
		t.Fatalf("the page offered no plugin: %+v", res)
	}
	setup, err := parseCloudValue(res.Value)
	if err != nil {
		t.Fatal(err)
	}
	name := cloudPluginName(a.d.id.User)
	if !strings.HasPrefix(name, "cogmer-cloud-") || len(name) != len("cogmer-cloud-")+8 || res.PluginFile != name+".zip" {
		t.Errorf("the plugin is named %q and its file %q", name, res.PluginFile)
	}
	raw, err := base64.StdEncoding.DecodeString(res.Plugin)
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		var b bytes.Buffer
		_, _ = b.ReadFrom(rc)
		rc.Close()
		files[f.Name] = b.Bytes()
		if f.Modified.Year() < 2026 {
			t.Errorf("%s is dated %v, not when the plugin was made", f.Name, f.Modified)
		}
		if f.Name == pluginValueFile && f.Mode().Perm() != 0o600 {
			t.Errorf("%s has mode %o, want 600", f.Name, f.Mode().Perm())
		}
	}
	if len(files) != 2 || files[pluginValueFile] == nil || files[".claude-plugin/plugin.json"] == nil {
		t.Fatalf("the plugin holds %d files, want its manifest and %s alone", len(files), pluginValueFile)
	}
	if string(files[pluginValueFile]) != res.Value {
		t.Error("the plugin's file is not the value the page showed")
	}
	var manifest map[string]any
	if err := json.Unmarshal(files[".claude-plugin/plugin.json"], &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest["name"] != name || manifest["version"] != "1.0."+strconv.FormatUint(setup.cloud.Serial, 10) {
		t.Errorf("the manifest names %v at version %v", manifest["name"], manifest["version"])
	}
	for _, component := range []string{"hooks", "commands", "skills", "agents", "mcpServers"} {
		if _, has := manifest[component]; has {
			t.Errorf("the manifest declares %s", component)
		}
	}
	// The value reaches nobody but the page: no log holds it, and the machine keeps no key.
	for _, f := range []string{res.Value, strings.TrimPrefix(res.Value, cloudValuePrefix), res.Plugin} {
		if strings.Contains(logs.String(), f) {
			t.Fatal("the daemon's log holds the cloud value or the plugin")
		}
	}
	if files := filesHolding(t, a.home, secretForms(setup.key)); len(files) > 0 {
		t.Errorf("the machine keeps the cloud key in %v", files)
	}
}

func TestAUserWhoHasACurrentCloudKeyAddsNoSecondUntilTheFirstIsRemoved(t *testing.T) {
	a, _ := twoMachines(t)
	first := addCloudEnv(t, a, "first")
	setup, _ := parseCloudValue(first)

	var res cloudAddResponse
	a.local(t, "/machines/cloud/add", cloudAddRequest{Name: "second"}, &res)
	if res.Value != "" || res.Plugin != "" || !strings.Contains(res.Error, "already have a cloud environment, first") ||
		!strings.Contains(res.Error, "Remove it first") {
		t.Fatalf("a second cloud key was not refused as it should be: %+v", res)
	}
	cloudKeys := func() int {
		certs, _ := a.d.members.displayCerts(a.d.id.User)
		n := 0
		for _, c := range certs {
			if c.Kind == KindCloud {
				n++
			}
		}
		return n
	}
	if cloudKeys() != 1 {
		t.Fatalf("the user has %d cloud keys after a refusal", cloudKeys())
	}

	var removed identityResult
	a.local(t, "/machines/remove", removeRequest{Machine: setup.cloud.Machine, Stolen: true}, &removed)
	if removed.Status != "removed" {
		t.Fatalf("removing the cloud key: %+v", removed)
	}
	if second := addCloudEnv(t, a, "second"); second == first {
		t.Error("the second value is the first")
	}
}

func TestThePageTellsTheUserWhatKeepingTheValueInAPluginRevealsBeforeOfferingIt(t *testing.T) {
	page := strings.Join(strings.Fields(identityPageSrc), " ")
	for _, said := range []string{
		"Anthropic stores the file",
		"relay addresses of your machines, and they hold no secret",
		"administrators can see that the plugin exists",
		"in every version an upload over the same name keeps",
		"Every cloud session of your account that syncs your plugins receives the value",
		"a value left in any copy of the plugin is used until its cloud key is removed",
		"remove the plugin in your plugin settings, then upload the new one",
		"Keep the plugin's name as it is: a session reads only a plugin with that name, and a renamed one may not be read",
		"Share it and publish it nowhere",
	} {
		if !strings.Contains(page, said) {
			t.Errorf("the page does not say %q", said)
		}
	}
	if strings.Index(page, "Anthropic stores the file") > strings.Index(page, "Download ${esc(res.pluginFile)}")+400 {
		t.Error("the page's disclosure comes long after its download button")
	}
}
