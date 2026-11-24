#!/usr/bin/env bash
# Runs the suite on the tree that is staged, and records that tree as passed. Stage
# every change first: the tree recorded is the staged one, and the suite runs on the
# working tree, so the two must be the same.
#
#   git add <files> && scripts/test-and-record.sh > <log> && git commit ...
#   scripts/test-and-record.sh --full > <log>
#
# By default it runs the suite with -short, which leaves out the tests that build
# programs and run them, and records the tree in .git/suite-passed-short, which is what
# a commit waits on. With --full it runs the whole suite and records the tree in
# .git/suite-passed as well, and .githooks/pre-push lets a commit of exactly that tree
# through without running the whole suite again. A push therefore always has the whole
# suite behind it, and a commit has every test that runs in a few seconds.
#
# It exits non-zero when a test fails, when a tracked file differs from what is staged,
# or when a Go file the build would read is untracked, since the tree recorded would
# then not be the tree tested.
#
# Each run also writes every test's time, slowest first, to
# .git/suite-times/<tree>[-short], and copies it to .git/suite-times/latest, so that
# where the suite spends its time is known from runs already made.
set -u
set -o pipefail
full=""
[ "${1:-}" = "--full" ] && full=1
cd "$(git rev-parse --show-toplevel)" || exit 2
if ! git diff --quiet; then
  echo "test-and-record: a tracked file differs from what is staged; stage it, or set it aside, first" >&2
  exit 2
fi
stray=$(git ls-files --others --exclude-standard -- '*.go' 'go.mod' 'go.sum' '*.html' 'cmd/**' 'plugin/**')
if [ -n "$stray" ]; then
  echo "test-and-record: untracked files the build or tests may read: $stray" >&2
  exit 2
fi
tree=$(git write-tree) || exit 2
gitdir="$(git rev-parse --git-dir)"
times="$gitdir/suite-times"
mkdir -p "$times" || exit 2
go build -o "$times/.suitetimes" ./cmd/suitetimes || exit 2
if [ -n "$full" ]; then
  name="$tree"; flags=""
else
  name="$tree-short"; flags="-short"
fi
go test -count=1 $flags -json ./... | "$times/.suitetimes" "$times/$name"
status=$?
cp -f "$times/$name" "$times/latest" 2>/dev/null
[ "$status" -eq 0 ] || exit 1
if ! git diff --quiet || [ "$(git write-tree)" != "$tree" ]; then
  echo "test-and-record: the files changed while the suite ran, so nothing is recorded" >&2
  exit 1
fi
mkdir -p "$gitdir/suite-passed-short" && touch "$gitdir/suite-passed-short/$tree"
if [ -n "$full" ]; then
  mkdir -p "$gitdir/suite-passed" && touch "$gitdir/suite-passed/$tree"
  echo "test-and-record: tree $tree passed the whole suite"
else
  echo "test-and-record: tree $tree passed the short suite; a push runs the whole suite unless --full passed on it"
fi
