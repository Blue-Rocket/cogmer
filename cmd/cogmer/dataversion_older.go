//go:build olderdataversion

package main

// dataVersion, in a build tagged olderdataversion, is 0, the version of data no
// program has versioned. The container test builds such a program to stand for one
// released before the data it finds, and no release is built with the tag.
const dataVersion = 0
