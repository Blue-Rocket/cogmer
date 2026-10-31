#!/usr/bin/env python3
"""A throwaway MCP server for one question: where does a server that a plugin bundles run
in a Cowork session, on the user's computer or in the session's own machine?

It speaks MCP over stdio (one JSON message per line) and offers one tool, `ping`. When it
starts and each time it is called it appends a line to ~/cogmer-route-probe/server.log that
names the time, the host, the system, the process, its parent, the working directory, HOME,
the names (never the values) of the environment variables that begin CLAUDE or COGMER, and
the call's arguments. On start it also tries to write ~/cogmer-route-probe/write-test and to
start a detached `sleep 900`, and each call records whether that file and that process are
still there. It reads nothing else and sends nothing anywhere."""
import json, os, socket, subprocess, sys, time

DIR = os.path.expanduser("~/cogmer-route-probe")
LOG = os.path.join(DIR, "server.log")
WRITE_TEST = os.path.join(DIR, "write-test")
sleeper = None


def uname():
    u = os.uname()
    return "%s %s" % (u.sysname, u.machine)


def env_names():
    return ",".join(sorted(k for k in os.environ if k.startswith("CLAUDE") or k.startswith("COGMER"))) or "(none)"


def log(msg):
    try:
        os.makedirs(DIR, exist_ok=True)
        with open(LOG, "a") as f:
            f.write("%s | host=%s | %s | pid=%d ppid=%d | cwd=%s | HOME=%s | env=%s | %s\n" % (
                time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()), socket.gethostname(), uname(),
                os.getpid(), os.getppid(), os.getcwd(), os.environ.get("HOME", "(unset)"), env_names(), msg))
    except OSError:
        pass


def send(obj):
    sys.stdout.write(json.dumps(obj) + "\n")
    sys.stdout.flush()


def start_effects():
    """What the server tries on start: a file written outside its own directory, and a process
    that outlives it."""
    global sleeper
    try:
        os.makedirs(DIR, exist_ok=True)
        with open(WRITE_TEST, "w") as f:
            f.write("written by pid %d at %s\n" % (os.getpid(), time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())))
        wrote = "ok"
    except OSError as e:
        wrote = "failed: %s" % e
    try:
        sleeper = subprocess.Popen(["sleep", "900"], start_new_session=True, stdin=subprocess.DEVNULL,
                                   stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        started = "pid %d" % sleeper.pid
    except OSError as e:
        started = "failed: %s" % e
    log("server started | write-test: %s | detached sleep 900: %s" % (wrote, started))


def effects_now():
    alive = "none started"
    if sleeper is not None:
        alive = "pid %d %s" % (sleeper.pid, "alive" if sleeper.poll() is None else "gone")
    return "write-test exists: %s | detached sleep: %s" % (os.path.exists(WRITE_TEST), alive)


start_effects()
for line in sys.stdin:
    line = line.strip()
    if not line:
        continue
    try:
        m = json.loads(line)
    except Exception:
        continue
    method, mid = m.get("method"), m.get("id")
    if method == "initialize":
        send({"jsonrpc": "2.0", "id": mid, "result": {
            "protocolVersion": (m.get("params") or {}).get("protocolVersion", "2025-06-18"),
            "capabilities": {"tools": {}},
            "serverInfo": {"name": "cogmer-route-probe", "version": "0.1.0"}}})
    elif method == "tools/list":
        send({"jsonrpc": "2.0", "id": mid, "result": {"tools": [{
            "name": "ping",
            "description": "Reports that the route probe's server was reached. Takes optional `via` and `session` strings.",
            "inputSchema": {"type": "object", "properties": {"via": {"type": "string"}, "session": {"type": "string"}}}}]}})
    elif method == "tools/call":
        args = ((m.get("params") or {}).get("arguments")) or {}
        log("ping | args=%s | %s" % (json.dumps(args, sort_keys=True), effects_now()))
        text = json.dumps({"hookSpecificOutput": {
            "hookEventName": "UserPromptSubmit",
            "additionalContext": "route-probe server reached: %s" % socket.gethostname()}})
        send({"jsonrpc": "2.0", "id": mid, "result": {"content": [{"type": "text", "text": text}]}})
    elif mid is not None:
        send({"jsonrpc": "2.0", "id": mid, "error": {"code": -32601, "message": "method not found"}})
log("server stopped | %s" % effects_now())
