package main

import (
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

// A cloud session's daemon takes its cloud value from the plugin files its account syncs
// into the session, as well as from the variable (D-279, a cloud session's daemon reads
// its cloud value from the plugin files it finds, as well as from the variable). The
// account's plugins are one folder each under an account folder, and every file named
// cloud.value in them is read, because the user can copy, rename and change plugins and
// no count of them can be assumed.

// pluginValueFile is the name of the file a plugin holds a cloud value in (D-278, the
// machines page offers the cloud value as a plugin for the user to upload once).
const pluginValueFile = "cloud.value"

// pluginValueMax is the largest file read as a value. A value is under 3KB, so a larger
// file is not one, and is not read.
const pluginValueMax = 64 << 10

// syncedPluginsRoot is the folder under which a cloud session's host puts the account's
// plugins. COGMER_SYNCED_PLUGINS names another, for a test.
func syncedPluginsRoot() string {
	if p := os.Getenv("COGMER_SYNCED_PLUGINS"); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude", "plugins", "synced")
}

// pluginValue is a cloud value read from a plugin's file, and what it reads as.
type pluginValue struct {
	raw   string
	setup *cloudSetup
}

// pluginFolderName is the name of a plugin folder that may hold a cloud value: the name the
// machines page gives its plugin, with the suffix that a later upload over the same name
// adds (D-279). A plugin of any other name is not opened, so that a plugin of another kind
// cannot supply a value.
var pluginFolderName = regexp.MustCompile(`^` + regexp.QuoteMeta(cloudPluginPrefix) + `([0-9a-f]{8})(~g[0-9]+)?$`)

// newerValue reports whether a is the value to take over b, for two values of one user: the
// one whose cloud certificate has the higher serial, and for equal serials the one whose
// cloud key's identifier sorts first, so that every session of the user takes the same one
// (D-279). A serial counts the certificates of one user, so it says nothing of values of two.
func newerValue(a, b *cloudSetup) bool {
	if a.cloud.Serial != b.cloud.Serial {
		return a.cloud.Serial > b.cloud.Serial
	}
	return a.cloud.Machine < b.cloud.Machine
}

// newestPluginValue reads the cloud.value of each folder one below an account folder under
// root that is named as the page names its plugin, discards each that is not a cloud value
// whose certificates verify or whose name does not end in the digits of the user it names, and
// returns the newest. When the values that remain name more than one user it returns none and
// says so, since which user the session should speak as is not decided by any value. It
// reports how many files it discarded, and never quotes a file.
func newestPluginValue(root string) (best *pluginValue, discarded int, mixed bool) {
	if root == "" {
		return nil, 0, false
	}
	paths, _ := filepath.Glob(filepath.Join(root, "*", cloudPluginPrefix+"*", pluginValueFile))
	sort.Strings(paths)
	var values []*pluginValue
	for _, p := range paths {
		m := pluginFolderName.FindStringSubmatch(filepath.Base(filepath.Dir(p)))
		if m == nil {
			continue
		}
		info, err := os.Lstat(p)
		if err != nil || !info.Mode().IsRegular() || info.Size() > pluginValueMax {
			discarded++
			continue
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			discarded++
			continue
		}
		setup, err := parseCloudValue(string(raw))
		if err != nil || m[1] != cloudPluginDigits(setup.user) {
			discarded++
			continue
		}
		values = append(values, &pluginValue{raw: string(raw), setup: setup})
	}
	for _, v := range values {
		if v.setup.user != values[0].setup.user {
			return nil, discarded, true
		}
	}
	for _, v := range values {
		if best == nil || newerValue(v.setup, best.setup) {
			best = v
		}
	}
	return best, discarded, false
}

// startCloudFromPlugins makes this daemon a cloud session's from the newest value in
// the plugin files, when there is one, and says in the log how many files it ignored.
func (d *Daemon) startCloudFromPlugins() {
	best, discarded, mixed := newestPluginValue(d.pluginRoot)
	if discarded > 0 {
		log.Printf("cloud: %d file(s) named %s under %s do not read as a cloud value for the user their plugin is named for and are ignored", discarded, pluginValueFile, d.pluginRoot)
	}
	if mixed {
		d.notePlugins("mixed", "cloud: the plugin files name more than one user, so none is taken")
		return
	}
	if best != nil {
		d.startCloudFrom(best.raw, "a plugin's "+pluginValueFile)
	}
}

// notePlugins writes a line to the log when its kind differs from the last one written,
// so that a prompt that looks again does not repeat it.
func (d *Daemon) notePlugins(kind, line string) {
	if d.pluginNoted == kind {
		return
	}
	d.pluginNoted = kind
	log.Print(line)
}

// takeNewerPluginValue looks at the plugin files again and starts a cloud session from
// the newest value when this daemon has none or a value of its own user older than it. It
// is called at each prompt, because a session that began before an upload runs its first
// hook on the files from before it (D-279). A value no longer in the files ends nothing: a
// daemon that has read one holds it, and removing the cloud key is what ends its authority.
// A value of another user never replaces the one it holds.
func (d *Daemon) takeNewerPluginValue() {
	if d.pluginRoot == "" {
		return
	}
	d.pluginMu.Lock()
	defer d.pluginMu.Unlock()
	best, _, mixed := newestPluginValue(d.pluginRoot)
	if mixed {
		d.notePlugins("mixed", "cloud: the plugin files name more than one user, so none is taken")
		return
	}
	if best == nil {
		return
	}
	if c := d.cloud; c != nil {
		if best.setup.user != c.setup.user {
			d.notePlugins("other:"+best.setup.user, "cloud: a plugin holds a cloud value for another user, which is ignored")
			return
		}
		if !newerValue(best.setup, c.setup) {
			return
		}
	}
	d.startCloudFrom(best.raw, "a plugin's "+pluginValueFile)
	if d.cloud != nil {
		d.ensureCloudLoop()
	}
}
