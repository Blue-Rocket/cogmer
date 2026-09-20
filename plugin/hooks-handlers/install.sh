#!/usr/bin/env bash
# Obtain the claude-team binary, once, in the background.
#
# Run detached from SessionStart, because §29 requires that starting must not delay
# a session: a session that begins before the binary exists simply has nothing to
# inject yet, and the next one has everything.
#
# Three rules learned from the ecosystem's only comparable bootstrap, and one of
# our own:
#
#   Install OUTSIDE the plugin directory. A plugin update must not throw away a
#   working binary and re-fetch it; the two change on different schedules.
#
#   Several sessions start at once on one machine routinely. Two concurrent
#   installs writing one path is a corrupt binary, so a lock decides which one
#   proceeds and the others simply leave.
#
#   A failure must not be retried every session. A machine with no network and no
#   Go would otherwise spend a download attempt on every session it ever starts.
#
#   And ours: NEVER run a binary that cannot be verified. A download whose hash is
#   not pinned in the plugin is deleted rather than executed. If nothing can be
#   verified, building from source is the only remaining path, and having no binary
#   is the correct outcome when neither works.
set -u
. "$(dirname "$0")/common.sh"

root="${CLAUDE_PLUGIN_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}"
home_dir="${CLAUDE_TEAM_HOME:-$HOME/.claude-team}"
bin_dir="$home_dir/bin"
target="$bin_dir/claude-team"
lock="$home_dir/.install.lock"
cooldown_seconds=3600

# Delegates to the shared logger, which creates the directory first -- otherwise
# the message reporting that a directory could not be made would itself need that
# directory to exist.
say() { ct_say install.log "$*"; }

# Defined BEFORE this, so that failing to create the install directory is not the
# one step that exits without saying anything. It is load-bearing -- nothing is
# installed without it -- and it used to be the quietest line in the file (D-084).
if ! mkdir -p "$bin_dir" 2>/dev/null; then
  say "cannot create $bin_dir; nothing can be installed"
  exit 0
fi

want="$(cat "$root/VERSION" 2>/dev/null || echo unknown)"

# Already the version this plugin expects.
if [ -x "$target" ] && [ "$("$target" version 2>/dev/null)" = "$want" ]; then
  start_daemon_if_needed
  exit 0
fi

# Recent failure: leave it alone until the cooldown expires. A person asking why
# is answered by the same mark, so the wait is explicable rather than mysterious.
read -r prior_state prior_age _ <<< "$(install_state)"
if [ "$prior_state" = failed ] && [ "$prior_age" -lt "$cooldown_seconds" ]; then
  exit 0
fi

# mkdir is atomic on every platform that matters, which is why it is the lock.
if ! mkdir "$lock" 2>/dev/null; then
  exit 0   # another session is doing this; not an error
fi
trap 'rmdir "$lock" 2>/dev/null' EXIT

fail() { say "FAILED: $*"; set_install_state failed "$*"; exit 0; }

case "$(uname -s)" in
  Darwin) os=darwin ;;
  Linux)  os=linux ;;
  MINGW*|MSYS*|CYGWIN*) os=windows ;;
  *) fail "unsupported operating system $(uname -s)" ;;
esac
case "$(uname -m)" in
  arm64|aarch64) arch=arm64 ;;
  x86_64|amd64)  arch=amd64 ;;
  *) fail "unsupported architecture $(uname -m)" ;;
esac

set_install_state installing "fetching v${want}"
asset="claude-team_${want}_${os}_${arch}"
[ "$os" = windows ] && asset="$asset.exe"
expected="$(grep -E "[[:space:]]${asset}\$" "$root/checksums.txt" 2>/dev/null | awk '{print $1}' | head -1)"

tmp="$(mktemp -d "$home_dir/.install.XXXXXX" 2>/dev/null)" || fail "no temporary directory"
trap 'rm -rf "$tmp"; rmdir "$lock" 2>/dev/null' EXIT

# Plain HTTP is acceptable here and the checksum is why: the binary is not secret,
# and integrity comes from the pin rather than from the transport. A tampered
# response is refused exactly as a tampered file would be. HTTPS would still be
# better and is what a public release host will give us.
if [ -n "$expected" ] && command -v curl > /dev/null 2>&1; then
  # Where assets live is data, not code, and it travels beside the hashes that
  # authorise them -- moving hosts is then one commit that changes both together.
  # An unreachable or wrong host costs a failed download; a hash that did not move
  # with it would cost a refusal nobody could explain.
  base="${CLAUDE_TEAM_RELEASE_URL:-$(grep -v '^#' "$root/release-url.txt" 2>/dev/null | head -1 | tr -d '[:space:]')}"
  url="${base}/v${want}/${asset}"
  say "downloading $url"
  if curl -fsSL --max-time 120 -o "$tmp/bin" "$url"; then
    got="$(shasum -a 256 "$tmp/bin" 2>/dev/null | awk '{print $1}')"
    [ -z "$got" ] && got="$(sha256sum "$tmp/bin" 2>/dev/null | awk '{print $1}')"
    if [ "$got" = "$expected" ]; then
      if chmod +x "$tmp/bin" && mv -f "$tmp/bin" "$target"; then
        say "installed $want from release"
        clear_install_state
        # The session that triggered this looked for a binary that was still
        # downloading and found none, so nothing is running yet. Start it here
        # rather than leaving the first session inert and the second one useful.
        start_daemon_if_needed
        exit 0
      fi
    fi
    # The hash is the authorisation. Without a match there is nothing to weigh.
    say "REFUSED: $asset hashed $got, expected $expected — deleted, not run"
  else
    say "download failed"
  fi
else
  say "no pinned checksum for $asset (or no curl); not downloading"
fi

# Building from source is the fallback, and is the only path until a release is
# published with its hashes recorded. It needs Go and a network for modules.
if command -v go > /dev/null 2>&1; then
  say "building from source"
  if (cd "$tmp" && GOBIN="$tmp" go install -ldflags "-X main.version=$want" \
        "github.com/bluerocket/claude-team/cmd/claude-team@v${want}" > "$tmp/build.log" 2>&1); then
    if [ -x "$tmp/claude-team" ] && mv -f "$tmp/claude-team" "$target"; then
      say "installed $want from source"
      clear_install_state
      start_daemon_if_needed
      exit 0
    fi
  fi
  say "build failed: $(tail -3 "$tmp/build.log" 2>/dev/null | tr '\n' ' ')"
fi

fail "no verified download and no usable Go toolchain"
