//go:build !docker && !windows

package main

import (
	"os"
	"testing"
)

// The binaries that tests build are shared, so the process removes them when it is done.
func TestMain(m *testing.M) {
	code := m.Run()
	removeSharedBuilds()
	os.Exit(code)
}
