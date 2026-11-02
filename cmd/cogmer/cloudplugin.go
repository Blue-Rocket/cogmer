package main

import (
	"log"
	"os"
	"path/filepath"
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

// newerValue reports whether a is the value to take over b: the one whose cloud
// certificate has the higher serial, and for equal serials the one whose cloud key's
// identifier sorts first, so that every session of the user takes the same one (D-279).
func newerValue(a, b *cloudSetup) bool {
	if a.cloud.Serial != b.cloud.Serial {
		return a.cloud.Serial > b.cloud.Serial
	}
	return a.cloud.Machine < b.cloud.Machine
}

// newestPluginValue reads every cloud.value one folder below an account folder under
// root, discards each that is not a cloud value whose certificates verify, and returns the
// one to act on. It reports how many files it discarded, and never quotes a file.
func newestPluginValue(root string) (best *pluginValue, discarded int) {
	if root == "" {
		return nil, 0
	}
	paths, _ := filepath.Glob(filepath.Join(root, "*", "*", pluginValueFile))
	sort.Strings(paths)
	for _, p := range paths {
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
		if err != nil {
			discarded++
			continue
		}
		if v := (&pluginValue{raw: string(raw), setup: setup}); best == nil || newerValue(v.setup, best.setup) {
			best = v
		}
	}
	return best, discarded
}

// startCloudFromPlugins makes this daemon a cloud session's from the newest value in
// the plugin files, when there is one, and says in the log how many files it ignored.
func (d *Daemon) startCloudFromPlugins() {
	best, discarded := newestPluginValue(d.pluginRoot)
	if discarded > 0 {
		log.Printf("cloud: %d file(s) named %s under %s do not read as a cloud value and are ignored", discarded, pluginValueFile, d.pluginRoot)
	}
	if best != nil {
		d.startCloudFrom(best.raw, "a plugin's "+pluginValueFile)
	}
}

// takeNewerPluginValue looks at the plugin files again and starts a cloud session from
// the newest value when this daemon has none or a value older than it. It is called at
// each prompt, because a session that began before an upload runs its first hook on the
// files from before it (D-279). A value no longer in the files ends nothing: a daemon
// that has read one holds it, and removing the cloud key is what ends its authority.
func (d *Daemon) takeNewerPluginValue() {
	if d.pluginRoot == "" {
		return
	}
	d.pluginMu.Lock()
	defer d.pluginMu.Unlock()
	best, _ := newestPluginValue(d.pluginRoot)
	if best == nil {
		return
	}
	if c := d.cloud; c != nil && !newerValue(best.setup, c.setup) {
		return
	}
	d.startCloudFrom(best.raw, "a plugin's "+pluginValueFile)
	if d.cloud != nil {
		d.ensureCloudLoop()
	}
}
