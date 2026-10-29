#!/usr/bin/env python3
"""Pair a Mac test daemon with a Windows test daemon through the public relay, make a
room, and hand a turn each way. Both daemons use throwaway state; nothing touches a
real ~/.cogmer. The pairing is the real `cogmer pair --terminal` command on each side,
answered `y` only when the two words are the same."""
import json, os, queue, re, subprocess, sys, threading, time, urllib.request

S = os.path.dirname(os.path.abspath(__file__))
MAC_BIN = os.path.join(S, "cogmer")
MAC_ENV = dict(os.environ, COGMER_HOME=os.path.join(S, "machome"), COGMER_ADDR="127.0.0.1:47811",
               COGMER_PEER_ADDR="127.0.0.1:47812", COGMER_HOST="claude", COGMER_PREFLIGHT="off")
WIN_HOME = r"C:\Users\cogmer-test.DAVIDFOOTE5CBC"
NOISE = re.compile(r"post-quantum|store now|server may need")


def say(*a):
    print(time.strftime("%H:%M:%S"), *a, flush=True)


def ssh_cmd(cmd):
    return ["ssh", "-o", "BatchMode=yes", "cogmer-windows", cmd]


class Machine:
    def __init__(self, name):
        self.name = name

    # -- the page's requests
    def post(self, path, body):
        raise NotImplementedError

    def get(self, path):
        raise NotImplementedError

    def cli(self, session, *args, stdin=None, timeout=240):
        raise NotImplementedError

    def popen_cli(self, session, *args):
        raise NotImplementedError

    def write_file(self, path, text):
        raise NotImplementedError


class Mac(Machine):
    def post(self, path, body):
        req = urllib.request.Request("http://127.0.0.1:47811" + path, data=json.dumps(body).encode(),
                                     headers={"Content-Type": "application/json", "X-Cogmer": "1"})
        return json.loads(urllib.request.urlopen(req, timeout=240).read())

    def get(self, path):
        return json.loads(urllib.request.urlopen("http://127.0.0.1:47811" + path, timeout=30).read())

    def _env(self, session):
        e = dict(MAC_ENV)
        if session:
            e["CLAUDE_CODE_SESSION_ID"] = session
        return e

    def cli(self, session, *args, stdin=None, timeout=240):
        p = subprocess.run([MAC_BIN, *args], env=self._env(session), input=stdin, capture_output=True,
                           text=True, timeout=timeout)
        return p.returncode, p.stdout + p.stderr

    def popen_cli(self, session, *args):
        return subprocess.Popen([MAC_BIN, *args], env=self._env(session), stdin=subprocess.PIPE,
                                stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True, bufsize=1)

    def write_file(self, path, text):
        os.makedirs(os.path.dirname(path), exist_ok=True)
        open(path, "w").write(text)


class Win(Machine):
    def _run(self, cmd, stdin=None, timeout=240):
        p = subprocess.run(ssh_cmd(cmd), input=stdin, capture_output=True, text=True, timeout=timeout)
        out = "\n".join(l for l in (p.stdout + p.stderr).splitlines() if not NOISE.search(l))
        return p.returncode, out

    def post(self, path, body):
        rc, out = self._run("ctest\\cd.cmd post " + path, stdin=json.dumps(body))
        return json.loads(out)

    def get(self, path):
        rc, out = self._run("ctest\\cd.cmd get " + path)
        return json.loads(out)

    def _line(self, session, args):
        pre = 'set "CLAUDE_CODE_SESSION_ID=%s" & ' % session if session else ""
        return pre + "ctest\\cg.cmd " + " ".join(args)

    def cli(self, session, *args, stdin=None, timeout=240):
        return self._run(self._line(session, args), stdin=stdin, timeout=timeout)

    def popen_cli(self, session, *args):
        return subprocess.Popen(ssh_cmd(self._line(session, args)), stdin=subprocess.PIPE,
                                stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True, bufsize=1)

    def write_file(self, path, text):
        rc, out = self._run("ctest\\cd.cmd write " + path, stdin=text)
        assert rc == 0, out


mac, win = Mac("mac"), Win("win")
WIN_TRANSCRIPTS = WIN_HOME + r"\ctest\transcripts"
MAC_TRANSCRIPTS = os.path.join(S, "transcripts")


def create_user(m, display, machine_name):
    ph = m.post("/identity/phrase", {})
    if ph.get("error"):
        return "already made (" + ph["error"][:60] + ")"
    typed = {str(pos): ph["words"][pos - 1] for pos in ph["confirm"]}
    res = m.post("/identity/create", {"createId": ph["createId"], "words": typed, "displayName": display,
                                       "machineName": machine_name})
    assert res.get("user"), res
    return res["user"]


PAIRING = re.compile(r"Send your colleague this[^\n]*\n\s*\n\s*(\S+)")


def pairing_string(m):
    rc, out = m.cli("", "whoami")
    match = PAIRING.search(out)
    assert match, "no pairing string from %s:\n%s" % (m.name, out)
    return match.group(1), out


class Pairing:
    def __init__(self, m, theirs, name):
        self.m = m
        self.p = m.popen_cli("", "pair", theirs, name, "--terminal")
        self.q = queue.Queue()
        self.text = []
        threading.Thread(target=self._read, daemon=True).start()

    def _read(self):
        for line in self.p.stdout:
            line = line.rstrip("\n")
            if NOISE.search(line) or not line.strip():
                continue
            self.text.append(line)
            self.q.put(line)
        self.q.put(None)

    def words(self, within=180):
        prev, end = "", time.time() + within
        while time.time() < end:
            try:
                line = self.q.get(timeout=2)
            except queue.Empty:
                continue
            if line is None:
                raise RuntimeError("%s: pairing ended without words:\n%s" % (self.m.name, "\n".join(self.text)))
            if line.startswith("say those two words aloud"):
                return prev.strip()
            if line.strip():
                prev = line
        raise RuntimeError("%s: no words within %ds:\n%s" % (self.m.name, within, "\n".join(self.text)))

    def answer(self, a):
        self.p.stdin.write(a + "\n")
        self.p.stdin.close()

    def finish(self, within=120):
        try:
            self.p.wait(timeout=within)
        except subprocess.TimeoutExpired:
            self.p.kill()
        return "\n".join(self.text)


def turn(m, session, prompt, reply, transcripts, home_for_cwd):
    sep = "\\" if m is win else "/"
    transcript = transcripts + sep + session + ".jsonl"
    prompt_id = os.urandom(8).hex()
    payload = json.dumps({"session_id": session, "prompt": prompt, "prompt_id": prompt_id, "cwd": home_for_cwd,
                          "transcript_path": transcript, "hook_event_name": "UserPromptSubmit"})
    rc, given = m.cli("", "hook", "prompt", stdin=payload)
    lines = [json.dumps({"type": "user", "promptSource": "typed", "isSidechain": False, "sessionId": session,
                         "promptId": prompt_id, "message": {"role": "user", "content": prompt}})]
    if given.strip():
        lines.append(json.dumps({"type": "attachment", "attachment": {
            "type": "hook_success", "hookName": "UserPromptSubmit", "hookEvent": "UserPromptSubmit",
            "content": given.strip()}}))
    m.write_file(transcript, "\n".join(lines) + "\n")
    stop = json.dumps({"session_id": session, "prompt_id": prompt_id, "transcript_path": transcript,
                       "cwd": home_for_cwd, "last_assistant_message": reply, "hook_event_name": "Stop"})
    rc2, out2 = m.cli("", "hook", "stop", stdin=stop)
    assert "daemon unreachable" not in out2, out2
    return given


def view_has(m, room, text):
    return any(e.get("content") == text for e in m.get("/events?room=" + room).get("events", []))


def eventually(what, within, cond):
    end = time.time() + within
    while time.time() < end:
        if cond():
            return time.time()
        time.sleep(1)
    raise RuntimeError("%s did not happen within %ds" % (what, within))


def main():
    say("users")
    say("mac user:", create_user(mac, "ProbeMac", "probe-mac"))
    say("win user:", create_user(win, "ProbeWin", "probe-win"))

    sm, out_m = pairing_string(mac)
    sw, out_w = pairing_string(win)
    for who, out in (("mac", out_m), ("win", out_w)):
        say("%s self-status says:" % who, [l for l in out.splitlines() if l.startswith("Checked") or "outside" in l])
    say("mac string length", len(sm), "win string length", len(sw))

    say("pairing (real `pair --terminal` on both machines)")
    pm = Pairing(mac, sw, "probewin")
    pw = Pairing(win, sm, "probemac")
    wm, ww = pm.words(), pw.words()
    say("words shown: mac=%r win=%r" % (wm, ww))
    if not wm or wm != ww:
        pm.answer("n"); pw.answer("n")
        raise SystemExit("the two words differ")
    pm.answer("y"); pw.answer("y")
    for p in (pm, pw):
        out = p.finish()
        say("%s pair output ends:" % p.m.name, out.splitlines()[-1] if out else "(none)")
        assert "paired with" in out, out

    say("room")
    rc, out = mac.cli("session-mac-1", "create")
    room = re.search(r"(?m)^created (\S+)", out).group(1)
    say("created", room)
    rc, out = mac.cli("session-mac-1", "invite", "probewin")
    say("invite:", out.strip().splitlines()[-1] if out.strip() else "(none)")
    assert "Told them" in out, out
    rc, out = win.cli("session-win-1", "join", room)
    say("join:", out.strip().splitlines()[-1] if out.strip() else "(none)")
    assert "joined " + room in out, out

    say("turn mac -> win")
    t0 = time.time()
    turn(mac, "session-mac-1", "mac says the parser is done", "noted", MAC_TRANSCRIPTS, S)
    t1 = eventually("the mac turn reaching win", 90, lambda: view_has(win, room, "mac says the parser is done"))
    say("mac turn in win's view after %.1fs" % (t1 - t0))
    given = turn(win, "session-win-1", "what is new from mac?", "nothing", WIN_TRANSCRIPTS, WIN_HOME)
    say("win's next prompt was given the mac turn:", "mac says the parser is done" in given)

    say("turn win -> mac")
    t0 = time.time()
    turn(win, "session-win-1", "win says the installer is done", "noted", WIN_TRANSCRIPTS, WIN_HOME)
    t1 = eventually("the win turn reaching mac", 90, lambda: view_has(mac, room, "win says the installer is done"))
    say("win turn in mac's view after %.1fs" % (t1 - t0))
    given = turn(mac, "session-mac-1", "what is new from win?", "nothing", MAC_TRANSCRIPTS, S)
    say("mac's next prompt was given the win turn:", "win says the installer is done" in given)
    say("done")


main()
