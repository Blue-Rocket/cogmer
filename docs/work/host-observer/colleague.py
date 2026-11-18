#!/usr/bin/env python3
"""A mock colleague for the host probes: a second cogmer on this Mac, with its own throwaway
state, user identity and machine key, that reaches the cogmer under test through the relay as a
colleague on another machine would, and whose turns are written by hand instead of by a session.

It never touches ~/.cogmer: every command runs with COGMER_HOME, COGMER_ADDR and
COGMER_PEER_ADDR set to the mock's own. The user identity is made with the identity page's own
requests, and the turns go through the real `hook prompt` and `hook stop` commands with
hand-written transcripts, as docs/work/windows-mac-relay-probe/probe.py did.

  colleague.py start              start the mock's daemon, make its identity, print its pairing string
  colleague.py pair <string> [name]   pair with the cogmer under test (real `pair --terminal`)
  colleague.py host <name>        make a room as the mock, invite <name>, print the line that joins it
  colleague.py join <room>        put the mock's session in a room it was invited to
  colleague.py say <text> [reply] a turn from the mock: a prompt and the reply to it
  colleague.py view               the events the mock's room holds, and what its next prompt is given
  colleague.py stop               stop the mock's daemon

Settings, as environment variables: COLLEAGUE_BIN (the cogmer program, default
~/host-probe-build/cogmer), COLLEAGUE_HOME (default ~/cogmer-colleague), COLLEAGUE_ADDR (default
127.0.0.1:4792) and COLLEAGUE_PEER_ADDR (default 127.0.0.1:4793).
"""
import json, os, re, subprocess, sys, time, urllib.request

BIN = os.path.expanduser(os.environ.get("COLLEAGUE_BIN", "~/host-probe-build/cogmer"))
HOME = os.path.expanduser(os.environ.get("COLLEAGUE_HOME", "~/cogmer-colleague"))
ADDR = os.environ.get("COLLEAGUE_ADDR", "127.0.0.1:4792")
PEER = os.environ.get("COLLEAGUE_PEER_ADDR", "127.0.0.1:4793")
SESSION = os.environ.get("COLLEAGUE_SESSION", "mock-colleague-session")
ROOM_FILE = os.path.join(HOME, "mock-room")
TRANSCRIPTS = os.path.join(HOME, "mock-transcripts")
NOISE = re.compile(r"post-quantum|store now|server may need")


def env():
    e = {k: v for k, v in os.environ.items()
         if not k.startswith(("CLAUDE_CODE_", "CODEX_", "COGMER_", "CLAUDECODE"))}
    e.update(COGMER_HOME=HOME, COGMER_ADDR=ADDR, COGMER_PEER_ADDR=PEER, COGMER_HOST="claude",
             COGMER_PREFLIGHT="off", CLAUDE_CODE_SESSION_ID=SESSION)
    return e


def cli(*args, stdin=None, timeout=240):
    p = subprocess.run([BIN, *args], env=env(), input=stdin, capture_output=True, text=True, timeout=timeout)
    out = "\n".join(l for l in (p.stdout + p.stderr).splitlines() if not NOISE.search(l))
    return p.returncode, out


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


def start():
    if not os.path.exists(BIN):
        sys.exit("no cogmer program at %s: run docs/work/host-observer/build.sh first, or set COLLEAGUE_BIN" % BIN)
    os.makedirs(HOME, exist_ok=True)
    if not up():
        log = open(os.path.join(HOME, "mock-daemon.log"), "ab")
        subprocess.Popen([BIN, "daemon"], env=env(), stdin=subprocess.DEVNULL, stdout=log, stderr=log,
                         start_new_session=True)
        for _ in range(40):
            if up():
                break
            time.sleep(0.5)
        else:
            sys.exit("the mock's daemon did not answer at %s; see %s" % (ADDR, log.name))
    ph = call("/identity/phrase", {})
    if ph.get("error"):
        print("identity: already made (%s)" % ph["error"][:70])
    else:
        typed = {str(pos): ph["words"][pos - 1] for pos in ph["confirm"]}
        res = call("/identity/create", {"createId": ph["createId"], "words": typed,
                                        "displayName": "Mock Colleague", "machineName": "mock-colleague"})
        assert res.get("user"), res
        print("identity: made, user %s" % res["user"])
    rc, out = cli("whoami")
    m = re.search(r"Send your colleague this[^\n]*\n\s*\n\s*(\S+)", out)
    if not m:
        sys.exit("no pairing string yet:\n" + out)
    print("\nThe mock's pairing string, to give the cogmer under test as /cogmer:peer-pair <string> mock:\n")
    print(m.group(1))


def pair(theirs, name):
    p = subprocess.Popen([BIN, "pair", theirs, name, "--terminal"], env=env(), stdin=subprocess.PIPE,
                         stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True, bufsize=1)
    prev = ""
    for line in p.stdout:
        line = line.rstrip("\n")
        if NOISE.search(line) or not line.strip():
            continue
        print(line, flush=True)
        if line.startswith("say those two words aloud"):
            print("\nThe mock shows: %r" % prev.strip())
            ans = input("Does the cogmer under test show the same two words? [y/N] ").strip().lower()
            p.stdin.write(("y" if ans == "y" else "n") + "\n")
            p.stdin.close()
        elif line.strip():
            prev = line
    p.wait(timeout=120)


def host(name):
    rc, out = cli("create")
    m = re.search(r"(?m)^created (\S+)", out)
    if not m:
        sys.exit("no room made:\n" + out)
    r = m.group(1)
    open(ROOM_FILE, "w").write(r)
    rc, out = cli("invite", name)
    if "Told them" not in out:
        sys.exit("the invitation was not sent:\n" + out)
    print("room %s made, and %s told. In the session under test, run:\n\n  /cogmer:room-join %s" % (r, name, r))


def join(room):
    rc, out = cli("join", room)
    print(out)
    if "joined " + room in out:
        open(ROOM_FILE, "w").write(room)


def room():
    try:
        return open(ROOM_FILE).read().strip()
    except OSError:
        sys.exit("the mock has not joined a room: run `join <room>` first")


def say(prompt, reply):
    os.makedirs(TRANSCRIPTS, exist_ok=True)
    transcript = os.path.join(TRANSCRIPTS, SESSION + ".jsonl")
    prompt_id = os.urandom(8).hex()
    payload = json.dumps({"session_id": SESSION, "prompt": prompt, "prompt_id": prompt_id, "cwd": HOME,
                          "transcript_path": transcript, "hook_event_name": "UserPromptSubmit"})
    rc, given = cli("hook", "prompt", stdin=payload)
    lines = [json.dumps({"type": "user", "promptSource": "typed", "isSidechain": False, "sessionId": SESSION,
                         "promptId": prompt_id, "message": {"role": "user", "content": prompt}})]
    if given.strip():
        lines.append(json.dumps({"type": "attachment", "attachment": {
            "type": "hook_success", "hookName": "UserPromptSubmit", "hookEvent": "UserPromptSubmit",
            "content": given.strip()}}))
    open(transcript, "w").write("\n".join(lines) + "\n")
    stop = json.dumps({"session_id": SESSION, "prompt_id": prompt_id, "transcript_path": transcript,
                       "cwd": HOME, "last_assistant_message": reply, "hook_event_name": "Stop"})
    rc2, out2 = cli("hook", "stop", stdin=stop)
    if "daemon unreachable" in out2:
        sys.exit(out2)
    print("sent as the mock: prompt %r, reply %r" % (prompt, reply))
    print("that prompt was given: %s" % (given.strip() or "(nothing)"))
    return given


def view():
    r = room()
    events = call("/events?room=" + r).get("events", [])
    print("room %s holds %d events:" % (r, len(events)))
    for e in events:
        print("  ", json.dumps({k: (v if len(str(v)) < 120 else str(v)[:117] + "...") for k, v in e.items()}))
    payload = json.dumps({"session_id": SESSION, "prompt": "(view only)", "prompt_id": os.urandom(8).hex(),
                          "cwd": HOME, "transcript_path": os.path.join(TRANSCRIPTS, "none.jsonl"),
                          "hook_event_name": "UserPromptSubmit"})
    rc, given = cli("hook", "prompt", stdin=payload)
    print("\nthe mock's next prompt would be given:\n" + (given.strip() or "(nothing)"))


def main():
    a = sys.argv[1:]
    cmd = a[0] if a else ""
    if cmd == "start":
        start()
    elif cmd == "pair" and len(a) >= 2:
        pair(a[1], a[2] if len(a) > 2 else "probe")
    elif cmd == "host" and len(a) == 2:
        host(a[1])
    elif cmd == "join" and len(a) == 2:
        join(a[1])
    elif cmd == "say" and len(a) in (2, 3):
        say(a[1], a[2] if len(a) == 3 else "noted")
    elif cmd == "view":
        view()
    elif cmd == "stop":
        print(cli("stop")[1])
    else:
        sys.exit(__doc__)


if __name__ == "__main__":
    main()
