#!/usr/bin/env python3
"""Does the mechanical parts of the host probes in docs/work/host-probes.md, so that what a
person does is only what needs the app: clicking in Claude Desktop or the ChatGPT app and sending
prompts. It moves the real state aside and back, installs the built cogmer, starts the daemon
under test and a mock colleague (colleague.py), pairs them, makes a room, and collects the
record. Run it from a plain terminal: it refuses while Claude Desktop, the ChatGPT app or Codex is running.

  probe.py before <1|2|3>   check nothing is running, stop the daemon, move ~/.cogmer aside,
                            install the built cogmer in a fresh one, clear the observer's log,
                            and for probe 1 take the cogmer entry out of Desktop's configuration
  probe.py setup            start the daemon, make its identity, start the mock colleague, pair
                            them with the words checked here, and make a room that invites the user
  probe.py say [text]       a turn from the mock colleague with a token in it, and what to ask
  probe.py check            what the daemon and the observer's log show now
  probe.py report <probe>   write docs/work/host-probe-<probe>-<UTC time>.txt from what is collected
  probe.py after            stop both daemons, keep the probe's state, put the real state and
                            Desktop's configuration back

Settings, as environment variables: PROBE_BUILD (the folder build.sh filled, default
~/host-probe-build), PROBE_ADDR and PROBE_PEER_ADDR (the daemon under test's addresses, which are
the program's defaults when unset), PROBE_SKIP_PROCESS_CHECK=1 (to test this script itself), and
COLLEAGUE_* as colleague.py reads them. HOME decides where the state is, so a scratch HOME runs
the whole flow without touching the real one.
"""
import datetime, glob, json, os, re, shutil, subprocess, sys, time, urllib.request

sys.dont_write_bytecode = True

HERE = os.path.dirname(os.path.abspath(__file__))
HOME = os.path.expanduser("~")
STATE = os.path.join(HOME, ".cogmer")
ASIDE = os.path.join(HOME, ".cogmer.before-probe")
OBS = os.path.join(HOME, ".cogmer-host-observer")
MARK = os.path.join(HOME, ".cogmer-probe-mark.json")
BUILD = os.path.expanduser(os.environ.get("PROBE_BUILD", "~/host-probe-build"))
ADDR = os.environ.get("PROBE_ADDR", "127.0.0.1:4782")
DESKTOP = os.path.join(HOME, "Library/Application Support/Claude/claude_desktop_config.json"
                       if sys.platform == "darwin" else ".config/Claude/claude_desktop_config.json")
# The Claude Code session that drives a run is allowed: its prompts reach the probe's daemon and are
# ignored there, since a session in no room hands cogmer nothing (D-284).
APPS = ("Claude", "claude-desktop", "ChatGPT", "codex")
NOISE = re.compile(r"post-quantum|store now|server may need")

os.environ.setdefault("COLLEAGUE_BIN", os.path.join(BUILD, "cogmer"))
sys.path.insert(0, HERE)
import colleague as mock  # noqa: E402  (reads its settings from the environment at import)


def say(*a):
    print(*a, flush=True)


def die(msg):
    sys.exit("probe: " + msg)


def env():
    """The daemon under test's environment: the real state directory, as a host's hook sees it."""
    e = {k: v for k, v in os.environ.items()
         if not k.startswith(("CLAUDE_CODE_", "CODEX_", "COGMER_", "CLAUDECODE"))}
    e.update(COGMER_HOST="claude", COGMER_PREFLIGHT="off")
    if "PROBE_ADDR" in os.environ:
        e["COGMER_ADDR"] = ADDR
    if "PROBE_PEER_ADDR" in os.environ:
        e["COGMER_PEER_ADDR"] = os.environ["PROBE_PEER_ADDR"]
    return e


def cogmer(*args, stdin=None, timeout=240):
    p = subprocess.run([os.path.join(STATE, "bin", "cogmer"), *args], env=env(), input=stdin,
                       capture_output=True, text=True, timeout=timeout)
    return p.returncode, "\n".join(l for l in (p.stdout + p.stderr).splitlines() if not NOISE.search(l))


def call(path, body=None):
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request("http://" + ADDR + path, data=data,
                                 headers={"Content-Type": "application/json", "X-Cogmer": "1"})
    return json.loads(urllib.request.urlopen(req, timeout=240).read())


def up():
    try:
        urllib.request.urlopen("http://" + ADDR + "/healthz", timeout=2).read()
        return True
    except Exception:
        return False


def running():
    """Host apps and Codex, as lines of `pid command`, excluding this script."""
    out = subprocess.run(["ps", "-axo", "pid=,comm="], capture_output=True, text=True).stdout
    found = []
    for line in out.splitlines():
        pid, _, comm = line.strip().partition(" ")
        name = os.path.basename(comm.strip())
        if name in APPS and int(pid) != os.getpid():
            found.append("%s %s" % (pid, comm.strip()))
    return found


def need_quiet():
    if os.environ.get("PROBE_SKIP_PROCESS_CHECK") == "1":
        return
    found = running()
    if found:
        die("quit these first (a daemon that a running session restarts would run on the wrong "
            "state), then run this again:\n  " + "\n  ".join(found))


def read_mark():
    try:
        return json.load(open(MARK))
    except OSError:
        die("run `before` first: there is no record that the real state was moved aside")


def stop_daemon():
    binary = os.path.join(STATE, "bin", "cogmer")
    if os.path.exists(binary):
        cogmer("stop")
    for _ in range(20):
        if not up():
            return
        time.sleep(0.5)
    die("a daemon still answers at %s after `cogmer stop`" % ADDR)


def describe(d):
    """Enough of a state directory to see whether it is the one that holds the user's rooms."""
    try:
        ident = json.load(open(os.path.join(d, "identity.json")))
        who = "%s on %s" % (ident.get("userDisplayName"), ident.get("machineId"))
    except (OSError, ValueError):
        who = "no identity"
    rooms = len(glob.glob(os.path.join(d, "rooms", "*.db")))
    return "%s, %d rooms" % (who, rooms)


def before(probe):
    if probe not in ("1", "2", "3"):
        die("before takes the probe's number, 1, 2 or 3")
    if os.path.exists(MARK):
        die("%s exists: `after` has not run since the last `before`" % MARK)
    if os.path.exists(ASIDE):
        die("%s exists already, so moving the state there would nest it" % ASIDE)
    binary = os.path.join(BUILD, "cogmer")
    if not os.path.exists(binary):
        die("no built cogmer at %s: run docs/work/host-observer/build.sh first" % binary)
    stop_daemon()
    need_quiet()
    mark = {"probe": probe, "at": datetime.datetime.now(datetime.timezone.utc).isoformat(), "desktop": None,
            "moved": os.path.exists(STATE)}
    if os.path.exists(STATE):
        say("moving aside %s: %s" % (STATE, describe(STATE)))
        os.rename(STATE, ASIDE)
    os.makedirs(os.path.join(STATE, "bin"))
    shutil.copy(binary, os.path.join(STATE, "bin", "cogmer"))
    os.chmod(os.path.join(STATE, "bin", "cogmer"), 0o755)
    shutil.rmtree(OBS, ignore_errors=True)
    if probe == "1" and os.path.exists(DESKTOP):
        cfg = json.load(open(DESKTOP))
        servers = cfg.get("mcpServers", {})
        gone = [k for k in ("cogmer", "cogmer-host-probe") if k in servers]
        if gone:
            backup = DESKTOP + ".before-probe"
            if os.path.exists(backup):
                die("%s exists already" % backup)
            shutil.copy(DESKTOP, backup)
            mark["desktop_removed"] = {k: servers[k] for k in gone}
            for k in gone:
                del servers[k]
            json.dump(cfg, open(DESKTOP, "w"), indent=2)
            mark["desktop"] = backup
            say("Desktop's configuration: took out %s, kept a copy at %s" % (", ".join(gone), backup))
        else:
            say("Desktop's configuration has no cogmer entry: nothing to take out")
    json.dump(mark, open(MARK, "w"))
    rc, out = cogmer("version")
    say("real state %s; fresh state in %s with cogmer %s" % (
        "moved to " + ASIDE if mark["moved"] else "was absent", STATE, out.strip()))


def start_daemon():
    if up():
        return
    log = open(os.path.join(STATE, "probe-daemon.log"), "ab")
    subprocess.Popen([os.path.join(STATE, "bin", "cogmer"), "daemon"], env=env(), stdin=subprocess.DEVNULL,
                     stdout=log, stderr=log, start_new_session=True)
    for _ in range(40):
        if up():
            return
        time.sleep(0.5)
    die("the daemon did not answer at %s; see %s" % (ADDR, log.name))


def make_identity():
    ph = call("/identity/phrase", {})
    if ph.get("error"):
        say("identity under test: already made")
        return
    typed = {str(pos): ph["words"][pos - 1] for pos in ph["confirm"]}
    res = call("/identity/create", {"createId": ph["createId"], "words": typed, "displayName": "Probe User",
                                    "machineName": "probe-mac"})
    assert res.get("user"), res
    say("identity under test: made, throwaway, its recovery phrase was never shown")


def pairing_string():
    rc, out = cogmer("whoami")
    m = re.search(r"Send your colleague this[^\n]*\n\s*\n\s*(\S+)", out)
    if not m:
        die("no pairing string from the daemon under test:\n" + out)
    return m.group(1)


def words(p, within=180):
    prev, end = "", time.time() + within
    while time.time() < end:
        line = p.stdout.readline()
        if not line:
            break
        line = line.rstrip("\n")
        if NOISE.search(line) or not line.strip():
            continue
        if line.startswith("say those two words aloud"):
            return prev.strip()
        prev = line
    die("no words appeared within %ds" % within)


def pair():
    mock_string = [l for l in subprocess.run([sys.executable, os.path.join(HERE, "colleague.py"), "start"],
                                             capture_output=True, text=True, env=os.environ).stdout.splitlines()
                   if l.strip()][-1]
    mine = pairing_string()
    a = subprocess.Popen([os.path.join(STATE, "bin", "cogmer"), "pair", mock_string, "mock", "--terminal"],
                         env=env(), stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
                         text=True, bufsize=1)
    b = subprocess.Popen([mock.BIN, "pair", mine, "probe-user", "--terminal"], env=mock.env(),
                         stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True,
                         bufsize=1)
    wa, wb = words(a), words(b)
    say("words: daemon under test %r, mock %r" % (wa, wb))
    same = bool(wa) and wa == wb
    for p in (a, b):
        p.stdin.write(("y" if same else "n") + "\n")
        p.stdin.close()
    for p in (a, b):
        out = p.stdout.read()
        if same and "paired with" not in out:
            die("pairing did not finish:\n" + out)
    if not same:
        die("the two sides showed different words")
    say("paired, with the words compared here")


def setup():
    read_mark()
    start_daemon()
    if os.environ.get("PROBE_NO_IDENTITY") != "1":
        make_identity()
    pair()
    mock.host("probe-user")
    r = open(mock.ROOM_FILE).read().strip()
    say("\nThe daemon under test is at %s and the mock colleague is in room %s." % (ADDR, r))


def room():
    try:
        return open(mock.ROOM_FILE).read().strip()
    except OSError:
        die("run `setup` first")


def say_turn(text):
    token = "kumquat-%04d" % (int(time.time()) % 10000)
    prompt = text or "the mock colleague says the code word is %s" % token
    mock.say(prompt, "noted")
    open(os.path.join(HOME, ".cogmer-probe-token"), "w").write(token)
    say("\nIn the session under test, ask: what did my colleague just say, word for word?")
    if not text:
        say("A session that was given the turn can quote %r." % token)


def events():
    try:
        return call("/events?room=" + room()).get("events") or []
    except Exception as e:  # the daemon may be down
        return [{"error": str(e)}]


def summarize_log():
    path = os.path.join(OBS, "log.txt")
    if not os.path.exists(path):
        return ["(no observer log: no hook of the observer has run on this machine)"]
    text = open(path).read()
    heads = re.findall(r"=== (\S+)\s+#(\d+)\s+host=(\S+)\s+event=(\S+)", text)
    counts = {}
    for _, _, host, ev in heads:
        counts[(host, ev)] = counts.get((host, ev), 0) + 1
    lines = ["observer events: " + ", ".join("%s/%s x%d" % (h, e, n) for (h, e), n in sorted(counts.items()))]
    for field, label in (("where: ", "where"), ("cogmer here: ", "cogmer here")):
        seen = sorted(set(re.findall(r"^  %s(.*)$" % re.escape(field), text, re.M)))
        lines += ["%s: %s" % (label, s[:200]) for s in seen[:4]]
    return lines


def check():
    read_mark()
    say("daemon under test: " + ("answers at %s" % ADDR if up() else "does not answer at %s" % ADDR))
    bound = glob.glob(os.path.join(STATE, "bound-sessions", "*"))
    say("bound sessions on this machine: %d" % len(bound))
    evs = events()
    mine = [e for e in evs if e.get("userDisplayName") == "Probe User"]
    theirs = [e for e in evs if e.get("userDisplayName") == "Mock Colleague"]
    say("room %s: %d events, %d from the session under test (%s), %d from the mock" % (
        room(), len(evs), len(mine), ", ".join(sorted({e.get("eventType", "?") for e in mine})) or "none", len(theirs)))
    for line in summarize_log():
        say(line)


def sh(*a):
    return subprocess.run(a, capture_output=True, text=True).stdout.strip()


def report(probe):
    mark = read_mark()
    now = datetime.datetime.now(datetime.timezone.utc)
    path = os.path.join(HERE, "..", "host-probe-%s-%s.txt" % (probe, now.strftime("%Y%m%dT%H%M%SZ")))
    evs = events()
    lines = ["Host probe %s on %s, collected by docs/work/host-observer/probe.py" % (probe, now.strftime("%Y-%m-%dT%H:%M:%SZ")),
             "", "Machine and versions",
             "  macOS %s on %s" % (sh("sw_vers", "-productVersion"), sh("uname", "-m")),
             "  cogmer %s, build commit and hashes: see the build.sh output kept with this record" % cogmer("version")[1].strip(),
             "  Claude Code %s" % (sh("claude", "--version") or "(not on PATH)"),
             "", "The room (%s), as the daemon under test holds it" % room()]
    for e in evs:
        lines.append("  %s %-20s %-14s %s" % (e.get("timestamp", ""), e.get("eventType", e.get("error", "")),
                                                e.get("userDisplayName", ""), (e.get("content", "")
                                                                               if e.get("userDisplayName") == "Mock Colleague"
                                                                               and e.get("eventType") != "GUEST_INVITE"
                                                                               else "")[:80]))
    lines += ["", "The observer's log, in full", ""]
    logp = os.path.join(OBS, "log.txt")
    lines.append(open(logp).read() if os.path.exists(logp) else "(none)")
    lines += ["", "Steps as clicked, and what each showed (to fill in by hand)", "", "  ", "",
              "What the run does not show (to fill in by hand)", "", "  "]
    open(path, "w").write("\n".join(lines) + "\n")
    say("wrote %s\nFill in the two hand sections, then commit it with a message that says when, on what "
        "machine and versions, and what was seen." % os.path.normpath(path))


def after():
    mark = read_mark()
    need_quiet()
    mock.cli("stop")
    stop_daemon()
    dest = os.path.join(HOME, ".cogmer.probe-%s" % datetime.datetime.now().strftime("%Y%m%d"))
    if os.path.exists(dest):
        die("%s exists already" % dest)
    os.rename(STATE, dest)
    if mark["moved"]:
        say("restoring %s: %s" % (ASIDE, describe(ASIDE)))
        os.rename(ASIDE, STATE)
    if mark.get("desktop"):
        # Desktop writes this file itself while it runs, so the entries taken out are added back to
        # the file as it is now, and the copy is kept beside it.
        cfg = json.load(open(DESKTOP))
        cfg.setdefault("mcpServers", {}).update(mark.get("desktop_removed", {}))
        json.dump(cfg, open(DESKTOP, "w"), indent=2)
        say("Desktop's configuration: put back %s; the copy from before is at %s" % (
            ", ".join(mark.get("desktop_removed", {})) or "nothing", mark["desktop"]))
    os.remove(MARK)
    if os.path.exists(os.path.join(HOME, ".cogmer-probe-token")):
        os.remove(os.path.join(HOME, ".cogmer-probe-token"))
    say("the probe's state is kept at %s" % dest)
    say("the real state is %s" % ("back at " + STATE if mark["moved"] else "absent, as it was before"))
    say("still by hand: remove the observer and the probe's cogmer plugin from the app, and reinstall "
        "the cogmer plugin the Mac used before")


def main():
    a = sys.argv[1:]
    cmd = a[0] if a else ""
    if cmd == "before" and len(a) == 2:
        before(a[1])
    elif cmd == "setup":
        setup()
    elif cmd == "say":
        say_turn(a[1] if len(a) > 1 else "")
    elif cmd == "check":
        check()
    elif cmd == "report" and len(a) == 2:
        report(a[1])
    elif cmd == "after":
        after()
    else:
        sys.exit(__doc__)


main()
