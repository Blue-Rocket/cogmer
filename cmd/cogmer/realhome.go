package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
)

// A build made for development or for a test never opens the account's own state
// directory (D-265, a development or test build never opens the account's real state
// directory). The state directory holds the keys, the pairings and every room's history,
// and a program built from source can change it in ways the released program does not
// expect, such as migrating it. So the check does not depend on a test remembering to
// name a directory of its own, and it has no override: a program that must work on the
// real data is the one the installer fetched.

// realHomeForGuard is set only by a build that names a stand-in for the account's home
// with -ldflags "-X main.realHomeForGuard=...", so a test can show the refusal without
// the real directory being involved. Empty means the account's home as the operating
// system records it.
var realHomeForGuard string

// isDevelopmentBuild reports whether this program was built from source for development
// or for a test: it carries no release version, or a test built it, or it is a test.
func isDevelopmentBuild() bool { return developmentBuild(version, noBrowser, inTestBinary()) }

func developmentBuild(version, noBrowser string, inTest bool) bool {
	return version == "dev" || noBrowser != "" || inTest
}

// accountStateDir is the state directory under the account's home, as the operating
// system records the home and not as $HOME says, since a test sets $HOME. It is empty
// when the account has no home to name, as in an image with no passwd file.
func accountStateDir() string {
	home := realHomeForGuard
	if home == "" {
		u, err := user.Current()
		if err != nil || u.HomeDir == "" {
			return ""
		}
		home = u.HomeDir
	}
	return filepath.Join(home, stateDirName)
}

// refusedHome is the reason a development build may not use dir, or nil. dir is refused
// when it is the account's state directory or inside it, however it was reached: by
// default, by COGMER_HOME, or through a link.
func refusedHome(dir string) error { return refusedHomeIf(isDevelopmentBuild(), dir) }

func refusedHomeIf(development bool, dir string) error {
	if !development {
		return nil
	}
	real := accountStateDir()
	if real == "" || !sameOrInside(real, dir) {
		return nil
	}
	return fmt.Errorf("this is a development build (version %s), and it will not open %s, which holds this account's own cogmer keys, pairings and rooms. "+
		"Set COGMER_HOME to a directory made for this run, for example COGMER_HOME=$(mktemp -d). "+
		"There is no way to override this: the cogmer the plugin installed is the program to use on real data", version, dir)
}

// sameOrInside says whether child is parent or inside it, after resolving links in
// whatever part of each path exists, and ignoring case where the file system does.
func sameOrInside(parent, child string) bool {
	p, c := resolveExisting(parent), resolveExisting(child)
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		p, c = strings.ToLower(p), strings.ToLower(c)
	}
	rel, err := filepath.Rel(p, c)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)))
}

// resolveExisting makes p absolute and resolves links in the longest part of it that
// exists, so a link to the real directory is seen for what it is, and a directory that
// does not exist yet is still compared by its name.
func resolveExisting(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		return filepath.Clean(p)
	}
	rest := ""
	for cur := abs; ; {
		if r, err := filepath.EvalSymlinks(cur); err == nil {
			return filepath.Join(r, rest)
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return abs
		}
		rest = filepath.Join(filepath.Base(cur), rest)
		cur = parent
	}
}

// guardRealHome ends a development build that is about to use the real state directory,
// before anything there is read or written. A hook of a build somebody made from source
// says nothing and exits 0, as it does when cogmer is absent, and the bridge answers with
// the reason; a command, the daemon, and every subcommand of a build a test made say why
// and stop.
func guardRealHome(sub string) {
	if noDataEntryPoints[sub] {
		return
	}
	err := refusedHome(resolvedHomeDir())
	if err == nil {
		return
	}
	// A binary a test built refuses out loud for every subcommand, so a test that forgot to
	// name a directory fails where a silent hook would have let it pass.
	switch {
	case sub == "hook" && noBrowser == "":
		_, _ = io.Copy(io.Discard, os.Stdin)
		os.Exit(0)
	case sub == "bridge" && noBrowser == "":
		runRefusingBridge(err.Error())
		os.Exit(0)
	case sub == "daemon":
		log.Printf("not starting: %v", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, err)
	os.Exit(2)
}
