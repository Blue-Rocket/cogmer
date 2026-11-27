#!/usr/bin/env python3
"""The harness for the measurements in docs/work/hosting-fitness.md. It runs cogmer daemons on this
box, each with its own COGMER_HOME and two ports from 20000 to 21999, one on loopback for hooks and
the page and one on Docker's bridge address for colleagues' machines, pairs them and
puts them in rooms, writes turns, samples each daemon from /proc and ss, and stops them. It never
names 4782 or 4783, runs only a development build, which refuses the account's real ~/.cogmer
(D-265), and checks after each start that the process it started holds both of its ports.

  harness.py build <dir>                   build the commit checked out here into <dir>/cogmer
  harness.py fleet <dir> <shape> <n> [--overlay]
                                           start n daemons, make identities, pair and form rooms in
                                           <shape>, then stop them and keep their state as <dir>/seed
  harness.py run <dir> <name> [--overlay] [--turns] [--sync-ms MS] [--warm S] [--sample S] [--du S]
                                           start the seed's daemons from a fresh copy, wait, sample,
                                           check every room still delivers, stop, and write
                                           <dir>/<name>/samples.jsonl and summary.json
  harness.py baseline <dir> [--sample S]   sample the box with no fleet
  harness.py restart <dir> <name> [--overlay] [--times K]
                                           start the seed's daemons, then stop and start all at once,
                                           K times, timing each start
  harness.py fill <dir> <name> [--days D,D,...] [--idle S]   (days 0.25,0.5,1,2 and 300s by default)
                                           start the seed's daemons and write turns into every room as
                                           fast as the hooks take them, up to each checkpoint's number
                                           of days of the reference user's turns (1440 a day in each
                                           room); at each checkpoint wait for delivery, read each
                                           daemon's store sizes and PSS, and sample S idle seconds
  harness.py wake <dir> <name> [--times K] [--down S] [--as-left]
                                           start the seed's daemons, or with --as-left the stores the
                                           last fill left, then K times: stop the first daemon, have a
                                           colleague write a turn in a room they share, wait S seconds
                                           (30 by default), start it, and time its answer, its peer
                                           port, the turn's arrival and a turn of its own reaching
                                           the colleague, with its PSS a minute after the start
  harness.py check <dir>                   the harness's own check: two daemons, one room, one turn,
                                           with the overlay off and then on
  harness.py summary <dir>/<name>          print a run's summary again

Shapes: solo (identities, no rooms), pair1 (two daemons, one room), pair3 (two daemons, three
rooms), group4 (groups of four, every two share a room), group3 (groups of three, the same).

Each turn is a 500-byte prompt and a 4000-byte reply, written through `cogmer hook prompt` and
`cogmer hook stop` as docs/work/host-observer/colleague.py does.
"""
import json, os, random, re, shutil, signal, socket, string, subprocess, sys, threading, time
import urllib.request

sys.dont_write_bytecode = True

REPO = os.path.abspath(os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "..", ".."))
PORT0, PORTS_END = 20000, 21999
# The peer address. `cogmer whoami` gives no pairing string for a loopback address (checkReachable in
# cmd/cogmer/main.go), so it is Docker's bridge address, which only this box and its containers reach.
# Linux carries traffic to an address of its own over lo.
PEER_HOST = os.environ.get("HF_PEER_HOST", "172.17.0.1")
# A connection between two daemons can leave from any of the box's own addresses, as Linux picks the
# source, so every address the box holds counts as local. Any other is the relay or the relay map.
OWN_ADDRS = set(re.findall(r"inet6? ([0-9a-f.:]+)/", subprocess.run(["ip", "-o", "addr"], capture_output=True,
                                                                   text=True).stdout))
FORBIDDEN_PORTS = {4782, 4783}
NOISE = re.compile(r"post-quantum|store now|server may need")
PROMPT_BYTES, REPLY_BYTES = 500, 4000
TICK = os.sysconf("SC_CLK_TCK")


def say(*a):
    print(*a, flush=True)


def die(msg):
    sys.exit("harness: " + msg)


# ---------- one daemon


class Daemon:
    def __init__(self, run, i, overlay, sync_ms=None):
        self.run, self.i = run, i
        self.home = os.path.join(run, "homes", "d%02d" % i)
        self.addr = "127.0.0.1:%d" % (PORT0 + 2 * i)
        self.peer = "%s:%d" % (PEER_HOST, PORT0 + 2 * i + 1)
        self.name = "u%02d" % i
        self.overlay, self.sync_ms = overlay, sync_ms
        self.proc, self.pid = None, None
        for p in (PORT0 + 2 * i, PORT0 + 2 * i + 1):
            if p in FORBIDDEN_PORTS or not PORT0 <= p <= PORTS_END:
                die("port %d is outside the harness's range" % p)

    def env(self, session=None):
        e = {k: v for k, v in os.environ.items()
             if not k.startswith(("CLAUDE_CODE_", "CODEX_", "COGMER_", "CLAUDECODE"))}
        e.update(COGMER_HOME=self.home, COGMER_ADDR=self.addr, COGMER_PEER_ADDR=self.peer,
                 COGMER_HOST="claude", COGMER_PREFLIGHT="off")
        if not self.overlay:
            e["COGMER_TAILCAT"] = "off"
        if self.sync_ms:
            e["COGMER_SYNC_MS"] = str(self.sync_ms)
        if session:
            e["CLAUDE_CODE_SESSION_ID"] = session
        return e

    def bin(self):
        return os.path.join(self.run, "cogmer")

    def cli(self, *args, session=None, stdin=None, timeout=240):
        p = subprocess.run([self.bin(), *args], env=self.env(session), input=stdin, capture_output=True,
                           text=True, timeout=timeout)
        return p.returncode, "\n".join(l for l in (p.stdout + p.stderr).splitlines() if not NOISE.search(l))

    def call(self, path, body=None, timeout=30):
        data = json.dumps(body).encode() if body is not None else None
        req = urllib.request.Request("http://" + self.addr + path, data=data,
                                     headers={"Content-Type": "application/json", "X-Cogmer": "1"})
        return json.loads(urllib.request.urlopen(req, timeout=timeout).read())

    def up(self):
        try:
            urllib.request.urlopen("http://" + self.addr + "/healthz", timeout=2).read()
            return True
        except Exception:
            return False

    def start(self, wait=True):
        os.makedirs(self.home, exist_ok=True)
        log = open(os.path.join(self.home, "daemon.log"), "ab")
        self.proc = subprocess.Popen([self.bin(), "daemon"], env=self.env(), stdin=subprocess.DEVNULL,
                                     stdout=log, stderr=log, start_new_session=True)
        if wait:
            self.wait_up()

    def wait_up(self, within=60):
        end = time.time() + within
        while time.time() < end:
            if self.proc.poll() is not None:
                die("daemon %s exited with %s; see %s/daemon.log" % (self.name, self.proc.returncode, self.home))
            if self.up():
                holders = listeners()
                a, b = holders.get(PORT0 + 2 * self.i), holders.get(PORT0 + 2 * self.i + 1)
                if a != self.proc.pid or b != self.proc.pid:
                    self.stop()
                    die("daemon %s: ports held by %s and %s, not by the process started, %d"
                        % (self.name, a, b, self.proc.pid))
                self.pid = self.proc.pid
                return
            time.sleep(0.2)
        die("daemon %s did not answer at %s within %ds" % (self.name, self.addr, within))

    def stop(self):
        if self.proc and self.proc.poll() is None:
            self.proc.send_signal(signal.SIGTERM)
            try:
                self.proc.wait(timeout=20)
            except subprocess.TimeoutExpired:
                self.proc.kill()
                self.proc.wait()

    def make_identity(self):
        ph = self.call("/identity/phrase", {})
        if ph.get("error"):
            return
        typed = {str(pos): ph["words"][pos - 1] for pos in ph["confirm"]}
        res = self.call("/identity/create", {"createId": ph["createId"], "words": typed,
                                             "displayName": "User %02d" % self.i, "machineName": "m%02d" % self.i},
                        timeout=120)
        if not res.get("user"):
            die("no identity for %s: %s" % (self.name, res))

    def pairing_string(self):
        rc, out = self.cli("whoami")
        m = re.search(r"Send your colleague this[^\n]*\n\s*\n\s*(\S+)", out)
        if not m:
            die("no pairing string from %s:\n%s" % (self.name, out))
        return m.group(1)

    def log_count(self, text):
        try:
            return open(os.path.join(self.home, "daemon.log"), errors="replace").read().count(text)
        except OSError:
            return 0


def listeners():
    """Every listening TCP port in the harness's range, with the pid holding it."""
    out = subprocess.run(["ss", "-ltnpH"], capture_output=True, text=True).stdout
    held = {}
    for line in out.splitlines():
        f = line.split()
        if len(f) < 4:
            continue
        port = int(f[3].rsplit(":", 1)[1])
        m = re.search(r"pid=(\d+)", line)
        if PORT0 <= port <= PORTS_END:
            held[port] = int(m.group(1)) if m else -1
    return held


def ports_free(n):
    held = listeners()
    taken = [p for p in range(PORT0, PORT0 + 2 * n) if p in held]
    if taken:
        die("ports already taken: %s" % taken)


# ---------- pairing and rooms


def words(p, within=120):
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
    return None


def pair(a, b):
    sa, sb = a.pairing_string(), b.pairing_string()
    t0 = time.time()
    pa = subprocess.Popen([a.bin(), "pair", sb, b.name, "--terminal"], env=a.env(), stdin=subprocess.PIPE,
                          stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True, bufsize=1)
    pb = subprocess.Popen([b.bin(), "pair", sa, a.name, "--terminal"], env=b.env(), stdin=subprocess.PIPE,
                          stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True, bufsize=1)
    got = {}
    ts = [threading.Thread(target=lambda k, p: got.__setitem__(k, words(p)), args=(k, p)) for k, p in (("a", pa), ("b", pb))]
    for t in ts:
        t.start()
    for t in ts:
        t.join()
    same = bool(got.get("a")) and got.get("a") == got.get("b")
    outs = []
    for p in (pa, pb):
        try:
            p.stdin.write(("y" if same else "n") + "\n")
            p.stdin.close()
        except BrokenPipeError:
            pass
    for p in (pa, pb):
        outs.append(p.stdout.read())
        p.wait(timeout=120)
    if not same or not all("paired with" in o for o in outs):
        die("pairing %s with %s failed: words %r and %r\n%s\n%s" % (a.name, b.name, got.get("a"), got.get("b"), *outs))
    return time.time() - t0


def session(owner, other, k):
    return "hf-%s-%s-%d" % (owner.name, other.name, k)


def form_room(host, guest, k):
    rc, out = host.cli("create", session=session(host, guest, k))
    m = re.search(r"(?m)^created (\S+)", out)
    if not m:
        die("%s made no room:\n%s" % (host.name, out))
    room = m.group(1)
    rc, out = host.cli("invite", guest.name, session=session(host, guest, k))
    if "Told them" not in out:
        die("%s's invitation to %s was not sent:\n%s" % (host.name, guest.name, out))
    end = time.time() + 120
    while time.time() < end:
        rc, out = guest.cli("join", room, session=session(guest, host, k))
        if "joined " + room in out:
            return room
        time.sleep(2)
    die("%s could not join %s:\n%s" % (guest.name, room, out))


def edges(shape, n):
    """The rooms of a shape, as (host, guest, k), and the pairings in rounds of disjoint pairs."""
    if shape == "solo":
        return [], []
    if shape in ("pair1", "pair3"):
        if n % 2:
            die("%s needs an even number of daemons" % shape)
        per = 1 if shape == "pair1" else 3
        pairs = [(i, i + 1) for i in range(0, n, 2)]
        return [(a, b, k) for a, b in pairs for k in range(per)], [pairs]
    if shape == "group4":
        if n % 4:
            die("group4 needs a multiple of four daemons")
        rounds = [[], [], []]
        rooms = []
        for g in range(0, n, 4):
            a, b, c, d = g, g + 1, g + 2, g + 3
            for r, ps in enumerate((((a, b), (c, d)), ((a, c), (b, d)), ((a, d), (b, c)))):
                rounds[r] += list(ps)
            rooms += [(x, y, 0) for x, y in ((a, b), (c, d), (a, c), (b, d), (a, d), (b, c))]
        return rooms, rounds
    if shape == "group3":
        if n % 3:
            die("group3 needs a multiple of three daemons")
        rounds, rooms = [[], [], []], []
        for g in range(0, n, 3):
            a, b, c = g, g + 1, g + 2
            for r, p in enumerate(((a, b), (b, c), (a, c))):
                rounds[r].append(p)
            rooms += [(a, b, 0), (b, c, 0), (a, c, 0)]
        return rooms, rounds
    die("unknown shape %s" % shape)


# ---------- turns


def filler(n, token):
    body = token + " " + "".join(random.choice(string.ascii_lowercase + " ") for _ in range(n))
    return body[:n]


def write_turn(d, other, k, token):
    s = session(d, other, k)
    tdir = os.path.join(d.home, "transcripts")
    os.makedirs(tdir, exist_ok=True)
    transcript = os.path.join(tdir, s + ".jsonl")
    prompt, reply = filler(PROMPT_BYTES, token), filler(REPLY_BYTES, token + "-reply")
    pid = os.urandom(8).hex()
    payload = json.dumps({"session_id": s, "prompt": prompt, "prompt_id": pid, "cwd": d.home,
                          "transcript_path": transcript, "hook_event_name": "UserPromptSubmit"})
    d.cli("hook", "prompt", session=s, stdin=payload)
    open(transcript, "w").write(json.dumps({"type": "user", "promptSource": "typed", "isSidechain": False,
                                            "sessionId": s, "promptId": pid,
                                            "message": {"role": "user", "content": prompt}}) + "\n")
    stop = json.dumps({"session_id": s, "prompt_id": pid, "transcript_path": transcript, "cwd": d.home,
                       "last_assistant_message": reply, "hook_event_name": "Stop"})
    rc, out = d.cli("hook", "stop", session=s, stdin=stop)
    if "daemon unreachable" in out:
        raise RuntimeError(out)
    return time.time()


def wait_delivered(reader, room, token, within):
    end = time.time() + within
    while time.time() < end:
        try:
            evs = reader.call("/events?room=" + room).get("events") or []
            if token in json.dumps(evs):
                return time.time()
        except Exception:
            pass
        time.sleep(0.1)
    return None


def one_turn(fleet, room, results, interval_s):
    host, guest, k = fleet.daemons[room["host"]], fleet.daemons[room["guest"]], room["k"]
    writer, reader = (host, guest) if room["next"] == 0 else (guest, host)
    room["next"] ^= 1
    token = "tok%08x" % random.getrandbits(32)
    try:
        t0 = write_turn(writer, reader, k, token)
    except Exception as e:
        results.append({"room": room["id"], "error": str(e)[:200]})
        return
    t1 = wait_delivered(reader, room["id"], token, within=10 * interval_s + 30)
    results.append({"room": room["id"], "writer": writer.name, "t": t0,
                    "delivery_s": None if t1 is None else round(t1 - t0, 3)})


# ---------- the fleet


class Fleet:
    def __init__(self, run, n, overlay, sync_ms=None):
        self.run = run
        self.daemons = [Daemon(run, i, overlay, sync_ms) for i in range(n)]

    def start_all(self, parallel=False):
        ports_free(len(self.daemons))
        if parallel:
            for d in self.daemons:
                d.start(wait=False)
            for d in self.daemons:
                d.wait_up()
        else:
            for d in self.daemons:
                d.start()

    def stop_all(self):
        for d in self.daemons:
            d.stop()

    def relay_ready(self, within=120):
        end = time.time() + within
        while time.time() < end:
            if all(d.log_count("relay connection carried") for d in self.daemons):
                return True
            time.sleep(2)
        missing = [d.name for d in self.daemons if not d.log_count("relay connection carried")]
        die("with the overlay on, no relay connection for %s" % ", ".join(missing))


def cmd_build(run):
    os.makedirs(run, exist_ok=True)
    head = subprocess.run(["git", "-C", REPO, "rev-parse", "HEAD"], capture_output=True, text=True).stdout.strip()
    dirty = subprocess.run(["git", "-C", REPO, "status", "--porcelain", "--", "cmd", "go.mod", "go.sum"],
                           capture_output=True, text=True).stdout.strip()
    e = dict(os.environ, CGO_ENABLED="0")
    subprocess.run(["go", "build", "-tags", "ts_debug_websockets", "-ldflags", "-s -w", "-o",
                    os.path.join(run, "cogmer"), "./cmd/cogmer"], cwd=REPO, env=e, check=True)
    json.dump({"commit": head, "dirty": bool(dirty), "built": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())},
              open(os.path.join(run, "build.json"), "w"))
    say("built %s%s into %s/cogmer" % (head[:7], " with changes" if dirty else "", run))


def cmd_fleet(run, shape, n, overlay):
    if not os.path.exists(os.path.join(run, "cogmer")):
        die("run `build %s` first" % run)
    if os.path.exists(os.path.join(run, "seed")):
        die("%s/seed exists already" % run)
    shutil.rmtree(os.path.join(run, "homes"), ignore_errors=True)
    fleet = Fleet(run, n, overlay)
    rooms, rounds = edges(shape, n)
    fleet.start_all()
    try:
        for d in fleet.daemons:
            d.make_identity()
        if overlay:
            fleet.relay_ready()
        times = []
        for rnd in rounds:
            got, failed = [], []

            def one(a, b):
                try:
                    got.append(pair(fleet.daemons[a], fleet.daemons[b]))
                except BaseException as e:  # die() raises SystemExit, which a thread would swallow
                    failed.append(str(e))
            ts = [threading.Thread(target=one, args=p) for p in rnd]
            for t in ts:
                t.start()
            for t in ts:
                t.join()
            if failed:
                die("pairing failed:\n" + "\n".join(failed))
            times += got
        made = []
        for h, g, k in rooms:
            rid = form_room(fleet.daemons[h], fleet.daemons[g], k)
            made.append({"id": rid, "host": h, "guest": g, "k": k})
    finally:
        fleet.stop_all()
    os.rename(os.path.join(run, "homes"), os.path.join(run, "seed"))
    json.dump({"shape": shape, "n": n, "overlay_at_setup": overlay, "rooms": made,
               "pairing_s": [round(t, 1) for t in times]}, open(os.path.join(run, "fleet.json"), "w"), indent=1)
    say("fleet %s of %d: %d pairings (%s s each), %d rooms; state kept in %s/seed" % (
        shape, n, len(times), ", ".join("%.1f" % t for t in times), len(made), run))


# ---------- sampling


def read_proc(pid):
    try:
        st = open("/proc/%d/stat" % pid).read()
        f = st[st.rindex(")") + 2:].split()
        cpu = (int(f[11]) + int(f[12])) / TICK
        status = dict(l.split(":", 1) for l in open("/proc/%d/status" % pid).read().splitlines() if ":" in l)
        io = dict(l.split(": ") for l in open("/proc/%d/io" % pid).read().splitlines())
        return {"cpu_s": cpu, "rss_kb": int(status["VmRSS"].split()[0]), "threads": int(status["Threads"]),
                "disk_read": int(io["read_bytes"]), "disk_write": int(io["write_bytes"])}
    except (OSError, KeyError, ValueError):
        return None


def read_pss(pid):
    try:
        for l in open("/proc/%d/smaps_rollup" % pid):
            if l.startswith("Pss:"):
                return int(l.split()[1])
    except OSError:
        return None


def read_box():
    cpu = [int(x) for x in open("/proc/stat").readline().split()[1:]]
    mem = {l.split(":")[0]: int(l.split()[1]) for l in open("/proc/meminfo")}
    net = {}
    for l in open("/proc/net/dev").read().splitlines()[2:]:
        name, rest = l.split(":", 1)
        f = rest.split()
        net[name.strip()] = {"rx": int(f[0]), "tx": int(f[8])}
    return {"cpu_ticks": cpu, "steal": cpu[7] if len(cpu) > 7 else 0, "mem_available_kb": mem["MemAvailable"],
            "load1": float(open("/proc/loadavg").read().split()[0]), "net": net}


class Sockets:
    """Every TCP socket of the fleet seen so far, with the most bytes seen on it, so that a socket
    that closes still counts for what it carried while it was seen."""

    def __init__(self, pids, peer_ports):
        self.pids, self.peer_ports = set(pids), set(peer_ports)
        self.seen = {}

    def sample(self):
        out = subprocess.run(["ss", "-tinpH"], capture_output=True, text=True).stdout
        lines, cur = out.splitlines(), None
        states = {}
        for line in lines:
            if not line.startswith((" ", "\t")):
                f = line.split()
                m = re.search(r"pid=(\d+)", line)
                pid = int(m.group(1)) if m else None
                cur = None
                if pid in self.pids and len(f) >= 5:
                    cur = (pid, f[3], f[4])
                    states[pid] = states.get(pid, 0) + 1
            elif cur:
                sent = re.search(r"bytes_sent:(\d+)", line)
                recv = re.search(r"bytes_received:(\d+)", line)
                old = self.seen.get(cur, (0, 0))
                self.seen[cur] = (max(old[0], int(sent.group(1)) if sent else 0),
                                  max(old[1], int(recv.group(1)) if recv else 0))
        return states

    def totals(self):
        t = {}
        for (pid, local, remote), (s, r) in self.seen.items():
            host, port = remote.rsplit(":", 1)
            host = host.strip("[]").replace("::ffff:", "")
            kind = "loop" if host.startswith("127.") or host in OWN_ADDRS else "other"
            d = t.setdefault(pid, {"loop_sent": 0, "loop_recv": 0, "other_sent": 0, "other_recv": 0,
                                   "polls_made": 0, "relay_sockets": 0})
            d[kind + "_sent"] += s
            d[kind + "_recv"] += r
            if kind == "loop" and int(port) in self.peer_ports:
                d["polls_made"] += 1
            if kind == "other":
                d["relay_sockets"] += 1
        return t


def socket_states():
    out = subprocess.run(["ss", "-s"], capture_output=True, text=True).stdout
    m = re.search(r"TCP:\s+(\d+) \(estab (\d+), closed (\d+), orphaned (\d+), timewait (\d+)", out)
    return dict(zip(("tcp", "estab", "closed", "orphaned", "timewait"), map(int, m.groups()))) if m else {}


def sample_loop(path, daemons, seconds, every=10, pss_every=60, du_every=None, stop_flag=None):
    pids = {d.pid: d.name for d in daemons}
    socks = Sockets(pids, range(PORT0 + 1, PORTS_END + 1, 2))
    out = open(path, "a")
    t_end, last_pss, last_du = time.time() + seconds, 0, 0
    while time.time() < t_end and not (stop_flag and stop_flag.is_set()):
        t = time.time()
        rec = {"t": t, "box": read_box(), "sockets": socket_states(), "self_cpu_s": sum(os.times()[:4]),
               "daemons": {}}
        counts = socks.sample()
        for d in daemons:
            p = read_proc(d.pid)
            if p is None:
                rec["daemons"][d.name] = {"gone": True}
                continue
            p["sockets"] = counts.get(d.pid, 0)
            rec["daemons"][d.name] = p
        if t - last_pss >= pss_every:
            last_pss = t
            for d in daemons:
                if d.name in rec["daemons"] and not rec["daemons"][d.name].get("gone"):
                    rec["daemons"][d.name]["pss_kb"] = read_pss(d.pid)
        if du_every and t - last_du >= du_every:
            last_du = t
            for d in daemons:
                du = subprocess.run(["du", "-sk", d.home], capture_output=True, text=True).stdout.split()
                rec["daemons"][d.name]["home_kb"] = int(du[0]) if du else None
        tot = socks.totals()
        for d in daemons:
            if d.name in rec["daemons"]:
                rec["daemons"][d.name]["net"] = tot.get(d.pid, {})
        out.write(json.dumps(rec) + "\n")
        out.flush()
        time.sleep(max(0, every - (time.time() - t)))
    out.close()


def summarize(path, turns=None):
    recs = [json.loads(l) for l in open(path)]
    if len(recs) < 2:
        return {"error": "fewer than two samples"}
    a, b = recs[0], recs[-1]
    hours = (b["t"] - a["t"]) / 3600
    per = {}
    for name in b["daemons"]:
        x, y = a["daemons"].get(name, {}), b["daemons"][name]
        if y.get("gone") or not x:
            per[name] = {"gone": True}
            continue
        psss = [r["daemons"][name].get("pss_kb") for r in recs if r["daemons"].get(name, {}).get("pss_kb")]
        nx, ny = x.get("net", {}), y.get("net", {})
        wins, third = [], len(recs) // 3
        for w in range(3):
            r0, r1 = recs[w * third], recs[min(len(recs) - 1, (w + 1) * third)]
            dt = r1["t"] - r0["t"]
            if dt > 0 and name in r0["daemons"] and name in r1["daemons"]:
                wins.append((r1["daemons"][name]["cpu_s"] - r0["daemons"][name]["cpu_s"]) / dt * 3600)
        per[name] = {
            "cpu_s_per_h": round((y["cpu_s"] - x["cpu_s"]) / hours, 2),
            "cpu_s_per_h_windows": [round(w, 2) for w in wins],
            "pss_mb": round(sum(psss) / len(psss) / 1024, 2) if psss else None,
            "rss_mb_last": round(y["rss_kb"] / 1024, 2),
            "threads_last": y["threads"],
            "sockets_mean": round(sum(r["daemons"][name].get("sockets", 0) for r in recs) / len(recs), 1),
            "disk_write_per_h": round((y["disk_write"] - x["disk_write"]) / hours),
            **{k + "_per_h": round((ny.get(k, 0) - nx.get(k, 0)) / hours)
               for k in ("loop_sent", "loop_recv", "other_sent", "other_recv", "polls_made")},
            "relay_sockets_seen": ny.get("relay_sockets", 0),
        }
        homes = [r["daemons"][name].get("home_kb") for r in recs if r["daemons"].get(name, {}).get("home_kb")]
        if homes:
            per[name]["home_kb_first_last"] = [homes[0], homes[-1]]
    box_cpu = [p - q for p, q in zip(b["box"]["cpu_ticks"], a["box"]["cpu_ticks"])]
    s = {"seconds": round(b["t"] - a["t"]), "samples": len(recs), "daemons": per,
         "box": {"busy_share": round(1 - (box_cpu[3] + box_cpu[4]) / max(1, sum(box_cpu)), 4),
                 "steal_share": round((box_cpu[7] if len(box_cpu) > 7 else 0) / max(1, sum(box_cpu)), 4),
                 "mem_available_mb_first_last": [a["box"]["mem_available_kb"] // 1024,
                                                 b["box"]["mem_available_kb"] // 1024],
                 "net_bytes": {k: {"rx": b["box"]["net"][k]["rx"] - a["box"]["net"][k]["rx"],
                                   "tx": b["box"]["net"][k]["tx"] - a["box"]["net"][k]["tx"]}
                               for k in b["box"]["net"] if k in a["box"]["net"]},
                 "timewait_last": b["sockets"].get("timewait")},
         "sampler_cpu_s": round(b["self_cpu_s"] - a["self_cpu_s"], 2)}
    live = [v for v in per.values() if not v.get("gone")]
    if live:
        def mean(k):
            vals = [v[k] for v in live if v.get(k) is not None]
            return round(sum(vals) / len(vals), 2) if vals else None
        s["mean_daemon"] = {k: mean(k) for k in ("cpu_s_per_h", "pss_mb", "sockets_mean", "disk_write_per_h",
                                                 "loop_sent_per_h", "loop_recv_per_h", "other_sent_per_h",
                                                 "other_recv_per_h", "polls_made_per_h")}
        s["windows_disagree"] = any(max(v["cpu_s_per_h_windows"]) > 1.1 * min(v["cpu_s_per_h_windows"])
                                    for v in live if len(v["cpu_s_per_h_windows"]) == 3
                                    and min(v["cpu_s_per_h_windows"]) > 0)
    if turns is not None:
        ds = sorted(t["delivery_s"] for t in turns if t.get("delivery_s") is not None)
        s["turns"] = {"written": len(turns), "delivered": len(ds),
                      "undelivered": sum(1 for t in turns if t.get("delivery_s") is None),
                      "median_s": ds[len(ds) // 2] if ds else None,
                      "p95_s": ds[int(len(ds) * 0.95)] if ds else None}
    return s


# ---------- runs


def load_seed(run, overlay, sync_ms):
    meta = json.load(open(os.path.join(run, "fleet.json")))
    # A pairing records the address the other side gave, which is the overlay's address only when the
    # overlay was on, so a seed made with it off would sync over loopback in a run with it on.
    if meta["overlay_at_setup"] != overlay:
        die("the seed was made with the overlay %s; make a fleet with the overlay %s for this run" % (
            "on" if meta["overlay_at_setup"] else "off", "on" if overlay else "off"))
    homes = os.path.join(run, "homes")
    shutil.rmtree(homes, ignore_errors=True)
    shutil.copytree(os.path.join(run, "seed"), homes, symlinks=True)
    fleet = Fleet(run, meta["n"], overlay, sync_ms)
    rooms = [dict(r, next=0) for r in meta["rooms"]]
    return fleet, rooms, meta


def final_check(fleet, rooms, interval_s):
    results = []
    ts = [threading.Thread(target=one_turn, args=(fleet, r, results, interval_s)) for r in rooms]
    for t in ts:
        t.start()
    for t in ts:
        t.join()
    return results


def cmd_run(run, name, overlay, turns, sync_ms, warm, sample, du_every):
    outdir = os.path.join(run, name)
    if os.path.exists(outdir):
        die("%s exists already" % outdir)
    os.makedirs(outdir)
    fleet, rooms, meta = load_seed(run, overlay, sync_ms)
    interval_s = (sync_ms or 1000) / 1000
    info = {"name": name, "shape": meta["shape"], "n": meta["n"], "overlay": overlay, "turns": turns,
            "sync_ms": sync_ms or 1000, "warm_s": warm, "sample_s": sample,
            "build": json.load(open(os.path.join(run, "build.json"))),
            "started": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())}
    turn_log, stop_turns = [], threading.Event()

    def turn_loop():
        # Each room writes one turn a minute, the rooms spread over the minute.
        start = time.time()
        offsets = {r["id"]: 60.0 * i / max(1, len(rooms)) for i, r in enumerate(rooms)}
        nxt = {r["id"]: start + offsets[r["id"]] for r in rooms}
        while not stop_turns.is_set():
            now = time.time()
            for r in rooms:
                if now >= nxt[r["id"]]:
                    nxt[r["id"]] += 60
                    threading.Thread(target=one_turn, args=(fleet, r, turn_log, interval_s), daemon=True).start()
            stop_turns.wait(0.5)

    fleet.start_all()
    try:
        if overlay:
            fleet.relay_ready()
        unreach0 = {d.name: d.log_count("is unreachable") for d in fleet.daemons}
        if turns and rooms:
            threading.Thread(target=turn_loop, daemon=True).start()
        say("%s: %d daemons up, waiting %ds, then sampling for %ds" % (name, len(fleet.daemons), warm, sample))
        time.sleep(warm)
        warm_turns = len(turn_log)
        sample_loop(os.path.join(outdir, "samples.jsonl"), fleet.daemons, sample, du_every=du_every)
        stop_turns.set()
        time.sleep(10 * interval_s + 35)
        sampled_turns = turn_log[warm_turns:]
        check = final_check(fleet, rooms, interval_s) if rooms else []
        unreach = {d.name: d.log_count("is unreachable") - unreach0[d.name] for d in fleet.daemons}
        exited = [d.name for d in fleet.daemons if d.proc.poll() is not None]
    finally:
        fleet.stop_all()
    s = summarize(os.path.join(outdir, "samples.jsonl"), sampled_turns if turns else None)
    s["run"] = info
    s["final_check"] = {"rooms": len(check), "delivered": sum(1 for c in check if c.get("delivery_s") is not None),
                        "max_s": max([c["delivery_s"] for c in check if c.get("delivery_s") is not None], default=None)}
    s["unreachable_lines"] = unreach
    s["exited"] = exited
    s["valid"] = (not exited and not any(unreach.values()) and s["final_check"]["delivered"] == len(check)
                  and (not turns or s.get("turns", {}).get("undelivered", 0) == 0))
    json.dump(turn_log, open(os.path.join(outdir, "turns.json"), "w"))
    json.dump(s, open(os.path.join(outdir, "summary.json"), "w"), indent=1)
    say(json.dumps(s, indent=1))


def store_sizes(home):
    """Bytes of each kind of file under a COGMER_HOME: room databases, their logs, membership and the rest."""
    out = {"rooms_db": 0, "rooms_wal": 0, "membership": 0, "other": 0}
    for root, _, files in os.walk(home):
        if os.path.basename(root) == "transcripts":
            continue  # the harness's own transcript files, which a daemon does not write
        for f in files:
            try:
                n = os.path.getsize(os.path.join(root, f))
            except OSError:
                continue
            if os.path.basename(root) == "rooms":
                out["rooms_wal" if f.endswith(("-wal", "-shm")) else "rooms_db"] += n
            elif f.startswith("membership.db"):
                out["membership"] += n
            else:
                out["other"] += n
    out["total"] = sum(out.values())
    return out


def cmd_fill(run, name, days, idle):
    outdir = os.path.join(run, name)
    if os.path.exists(outdir):
        die("%s exists already" % outdir)
    os.makedirs(outdir)
    fleet, rooms, meta = load_seed(run, False, None)
    if not rooms:
        die("the seed has no rooms")
    info = {"name": name, "shape": meta["shape"], "n": meta["n"], "days": days, "idle_s": idle,
            "per_room_per_day": 1440, "build": json.load(open(os.path.join(run, "build.json"))),
            "started": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())}
    written = {r["id"]: 0 for r in rooms}
    errors, checkpoints = [], []
    seed_sizes = {d.name: store_sizes(d.home) for d in fleet.daemons}

    def fill_room(r, target, times):
        host, guest, k = fleet.daemons[r["host"]], fleet.daemons[r["guest"]], r["k"]
        while written[r["id"]] < target:
            writer, other = (host, guest) if r["next"] == 0 else (guest, host)
            r["next"] ^= 1
            t0 = time.time()
            try:
                write_turn(writer, other, k, "fill%08x" % random.getrandbits(32))
            except Exception as e:
                errors.append({"room": r["id"], "error": str(e)[:200]})
                if len(errors) > 20:
                    return
            times.append(time.time() - t0)
            written[r["id"]] += 1

    fleet.start_all()
    try:
        for day in days:
            target = int(1440 * day)
            say("%s: writing to %d turns in each room" % (name, target))
            t_start, times = time.time(), []
            ts = [threading.Thread(target=fill_room, args=(r, target, times)) for r in rooms]
            for t in ts:
                t.start()
            for t in ts:
                t.join()
            if len(errors) > 20:
                die("more than 20 turns failed: %s" % errors[:3])
            write_s = time.time() - t_start
            check = final_check(fleet, rooms, 1)
            for r in rooms:
                written[r["id"]] += 1
            time.sleep(60)
            cp_dir = os.path.join(outdir, "day%g" % day)
            os.makedirs(cp_dir)
            sample_loop(os.path.join(cp_dir, "samples.jsonl"), fleet.daemons, idle)
            idle_summary = summarize(os.path.join(cp_dir, "samples.jsonl"))
            times.sort()
            cp = {"days": day, "turns_per_room": written[rooms[0]["id"]],
                  "turns_per_daemon": {d.name: sum(written[r["id"]] for r in rooms
                                                   if fleet.daemons[r["host"]] is d or fleet.daemons[r["guest"]] is d)
                                       for d in fleet.daemons},
                  "write_s": round(write_s, 1),
                  "turn_write_s": {"median": round(times[len(times) // 2], 3) if times else None,
                                   "p95": round(times[int(len(times) * 0.95)], 3) if times else None},
                  "delivered": sum(1 for c in check if c.get("delivery_s") is not None), "rooms": len(check),
                  "delivery_max_s": max([c["delivery_s"] for c in check if c.get("delivery_s") is not None],
                                        default=None),
                  "sizes": {d.name: store_sizes(d.home) for d in fleet.daemons},
                  "idle": {k: idle_summary["mean_daemon"][k] for k in ("cpu_s_per_h", "pss_mb", "loop_sent_per_h",
                                                                         "disk_write_per_h")},
                  "idle_windows_disagree": idle_summary["windows_disagree"],
                  "exited": [d.name for d in fleet.daemons if d.proc.poll() is not None]}
            checkpoints.append(cp)
            say(json.dumps(cp))
            json.dump({"run": info, "seed_sizes": seed_sizes, "errors": errors, "checkpoints": checkpoints},
                      open(os.path.join(outdir, "fill.json"), "w"), indent=1)
            if cp["exited"]:
                die("daemons exited: %s" % cp["exited"])
    finally:
        fleet.stop_all()


def cmd_baseline(run, sample):
    outdir = os.path.join(run, "baseline-%s" % time.strftime("%Y%m%dT%H%M%SZ", time.gmtime()))
    os.makedirs(outdir)
    sample_loop(os.path.join(outdir, "samples.jsonl"), [], sample)
    s = summarize(os.path.join(outdir, "samples.jsonl"))
    json.dump(s, open(os.path.join(outdir, "summary.json"), "w"), indent=1)
    say(json.dumps(s, indent=1))


def cmd_restart(run, name, overlay, times):
    outdir = os.path.join(run, name)
    if os.path.exists(outdir):
        die("%s exists already" % outdir)
    os.makedirs(outdir)
    fleet, rooms, meta = load_seed(run, overlay, None)
    fleet.start_all()
    results = []
    try:
        if overlay:
            fleet.relay_ready()
        time.sleep(60)
        for k in range(times):
            fleet.stop_all()
            before = read_box()["mem_available_kb"]
            t0 = time.time()
            stop = threading.Event()
            box_path = os.path.join(outdir, "box-%d.jsonl" % k)
            sampler = threading.Thread(target=lambda: box_fast(box_path, stop))
            sampler.start()
            for d in fleet.daemons:
                d.start(wait=False)
            up_at = {}
            while len(up_at) < len(fleet.daemons) and time.time() - t0 < 300:
                for d in fleet.daemons:
                    if d.name not in up_at and d.up():
                        up_at[d.name] = time.time() - t0
                time.sleep(0.05)
            for d in fleet.daemons:
                d.wait_up()
            if overlay:
                fleet.relay_ready()
            check = final_check(fleet, rooms, 1.0)
            stop.set()
            sampler.join()
            box = [json.loads(l) for l in open(box_path)]
            results.append({"all_up_s": round(max(up_at.values()), 2) if up_at else None,
                            "first_turn_delivered_max_s": max([c["delivery_s"] for c in check
                                                               if c.get("delivery_s") is not None], default=None),
                            "undelivered": sum(1 for c in check if c.get("delivery_s") is None),
                            "mem_available_drop_mb": (before - min(r["mem_available_kb"] for r in box)) // 1024,
                            "peak_busy_share": max(r["busy_share"] for r in box) if box else None})
            say("start %d: %s" % (k + 1, results[-1]))
            time.sleep(60)
    finally:
        fleet.stop_all()
    json.dump({"n": meta["n"], "overlay": overlay, "starts": results}, open(os.path.join(outdir, "summary.json"), "w"),
              indent=1)


def cmd_wake(run, name, times, down, as_left):
    outdir = os.path.join(run, name)
    if os.path.exists(outdir):
        die("%s exists already" % outdir)
    if as_left:
        meta = json.load(open(os.path.join(run, "fleet.json")))
        if meta["overlay_at_setup"]:
            die("the seed was made with the overlay on, and wake runs with it off")
        if not os.path.isdir(os.path.join(run, "homes")):
            die("no stores left in %s/homes" % run)
        fleet = Fleet(run, meta["n"], False)
        rooms = [dict(r, next=0) for r in meta["rooms"]]
    else:
        fleet, rooms, meta = load_seed(run, False, None)
    shared = [r for r in rooms if 0 in (r["host"], r["guest"])]
    if not shared:
        die("the first daemon shares no room")
    os.makedirs(outdir)
    target = fleet.daemons[0]
    peer_port = PORT0 + 2 * target.i + 1
    results = []
    fleet.start_all()
    try:
        time.sleep(60)
        for k in range(times):
            r = shared[k % len(shared)]
            other = fleet.daemons[r["guest"] if r["host"] == 0 else r["host"]]
            target.stop()
            stopped_disk = store_sizes(target.home)["total"]
            token = "wake%08x" % random.getrandbits(32)
            write_turn(other, target, r["k"], token)
            time.sleep(down)
            t0 = time.time()
            target.start(wait=False)
            answered = port_s = None
            while time.time() - t0 < 120 and (answered is None or port_s is None):
                if target.proc.poll() is not None:
                    die("daemon %s exited on start %d; see %s/daemon.log" % (target.name, k + 1, target.home))
                now = time.time() - t0
                if port_s is None and listeners().get(peer_port) == target.proc.pid:
                    port_s = now
                if answered is None and target.up():
                    answered = now
                time.sleep(0.02)
            target.wait_up()
            got = wait_delivered(target, r["id"], token, within=120)
            back = "back%08x" % random.getrandbits(32)
            t1 = write_turn(target, other, r["k"], back)
            reached = wait_delivered(other, r["id"], back, within=120)
            time.sleep(max(0, 60 - (time.time() - t0)))
            pss = read_pss(target.pid)
            results.append({"answers_s": None if answered is None else round(answered, 3),
                            "peer_port_s": None if port_s is None else round(port_s, 3),
                            "missed_turn_s": None if got is None else round(got - t0, 3),
                            "own_turn_to_colleague_s": None if reached is None else round(reached - t1, 3),
                            "pss_mb_at_60s": None if pss is None else round(pss / 1024, 1),
                            "stopped_disk_mb": round(stopped_disk / 2 ** 20, 2)})
            say("wake %d: %s" % (k + 1, results[-1]))
            json.dump({"n": meta["n"], "down_s": down, "as_left": as_left,
                       "build": json.load(open(os.path.join(run, "build.json"))), "wakes": results},
                      open(os.path.join(outdir, "summary.json"), "w"), indent=1)
    finally:
        fleet.stop_all()


def box_fast(path, stop):
    out = open(path, "w")
    prev = read_box()
    while not stop.is_set():
        time.sleep(1)
        cur = read_box()
        d = [p - q for p, q in zip(cur["cpu_ticks"], prev["cpu_ticks"])]
        out.write(json.dumps({"t": time.time(), "mem_available_kb": cur["mem_available_kb"],
                              "busy_share": round(1 - (d[3] + d[4]) / max(1, sum(d)), 3)}) + "\n")
        prev = cur
    out.close()


def cmd_check(run):
    for overlay in (False, True):
        sub = os.path.join(run, "check-overlay-%s" % ("on" if overlay else "off"))
        if os.path.exists(sub):
            die("%s exists already" % sub)
        os.makedirs(sub)
        shutil.copy(os.path.join(run, "cogmer"), os.path.join(sub, "cogmer"))
        shutil.copy(os.path.join(run, "build.json"), os.path.join(sub, "build.json"))
        cmd_fleet(sub, "pair1", 2, overlay)
        fleet, rooms, meta = load_seed(sub, overlay, None)
        fleet.start_all()
        try:
            if overlay:
                fleet.relay_ready()
            res = final_check(fleet, rooms, 1.0) + final_check(fleet, rooms, 1.0)
        finally:
            fleet.stop_all()
        ok = all(r.get("delivery_s") is not None for r in res)
        say("check with the overlay %s: %s, a turn each way, delivered in %s s; pairing took %s s" % (
            "on" if overlay else "off", "passed" if ok else "FAILED", [r.get("delivery_s") for r in res],
            meta["pairing_s"]))
        if not ok:
            die("the harness check failed: %s" % res)


def main():
    a = sys.argv[1:]
    flags = {x for x in a if x.startswith("--")}

    def opt(name, default):
        if name in a:
            return int(a[a.index(name) + 1])
        return default

    def optstr(name, default):
        if name in a:
            return a[a.index(name) + 1]
        return default

    pos = [x for i, x in enumerate(a) if not x.startswith("--") and (i == 0 or a[i - 1] not in
                                                                      ("--sync-ms", "--warm", "--sample", "--du", "--times",
                                                                       "--days", "--idle", "--down"))]
    if not pos:
        sys.exit(__doc__)
    cmd, rest = pos[0], [os.path.abspath(os.path.expanduser(pos[1]))] + pos[2:] if len(pos) > 1 else []
    if cmd == "build" and len(rest) == 1:
        cmd_build(rest[0])
    elif cmd == "fleet" and len(rest) == 3:
        cmd_fleet(rest[0], rest[1], int(rest[2]), "--overlay" in flags)
    elif cmd == "run" and len(rest) == 2:
        cmd_run(rest[0], rest[1], "--overlay" in flags, "--turns" in flags, opt("--sync-ms", None),
                opt("--warm", 300), opt("--sample", 900), opt("--du", None))
    elif cmd == "baseline" and len(rest) == 1:
        cmd_baseline(rest[0], opt("--sample", 900))
    elif cmd == "restart" and len(rest) == 2:
        cmd_restart(rest[0], rest[1], "--overlay" in flags, opt("--times", 3))
    elif cmd == "fill" and len(rest) == 2:
        cmd_fill(rest[0], rest[1], [float(x) for x in optstr("--days", "0.25,0.5,1,2").split(",")], opt("--idle", 300))
    elif cmd == "wake" and len(rest) == 2:
        cmd_wake(rest[0], rest[1], opt("--times", 10), opt("--down", 30), "--as-left" in flags)
    elif cmd == "check" and len(rest) == 1:
        cmd_check(rest[0])
    elif cmd == "summary" and len(rest) == 1:
        say(open(os.path.join(rest[0], "summary.json")).read())
    else:
        sys.exit(__doc__)


main()
