#!/usr/bin/env bash
# Runs the whole suite on the tree that is staged, and records that tree as passed, so
# that the pre-push hook lets a commit of exactly that tree through without running the
# suite again. Stage every change first: the tree recorded is the staged one, and the
# suite runs on the working tree, so the two must be the same.
#
#   git add <files> && scripts/test-and-record.sh > <log> && git commit ...
#
# It exits non-zero when a test fails, when a tracked file differs from what is staged,
# or when a Go file the build would read is untracked, since the tree recorded would
# then not be the tree tested.
#
# Each run also writes every test's time, slowest first, to
# .git/suite-times/<tree>, and copies it to .git/suite-times/latest, so that where the
# suite spends its time is known from runs already made.
set -u
set -o pipefail
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
times="$(git rev-parse --git-dir)/suite-times"
mkdir -p "$times" || exit 2
go build -o "$times/.suitetimes" ./cmd/suitetimes || exit 2
go test -count=1 -json ./... | "$times/.suitetimes" "$times/$tree"
status=$?
cp -f "$times/$tree" "$times/latest" 2>/dev/null
[ "$status" -eq 0 ] || exit 1
if ! git diff --quiet || [ "$(git write-tree)" != "$tree" ]; then
  echo "test-and-record: the files changed while the suite ran, so nothing is recorded" >&2
  exit 1
fi
dir="$(git rev-parse --git-dir)/suite-passed"
mkdir -p "$dir" && touch "$dir/$tree"
echo "test-and-record: tree $tree passed"
