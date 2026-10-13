#!/usr/bin/env python3
"""A throwaway MCP server for one question: can a Cowork session reach a program
running on the user's own machine?

It speaks MCP over stdio (one JSON message per line), offers one tool, `ping`, and
writes a line to ~/.cogmer-host-probe.log each time anything calls it, naming this
machine and the arguments. It reads nothing and sends nothing anywhere. If the log
gains a line when a Cowork session runs, the session reached this machine."""
import json, os, socket, sys, time

LOG = os.path.expanduser("~/.cogmer-host-probe.log")
MODE = os.path.expanduser("~/.cogmer-host-probe.mode")


def mode():
    """A failure to simulate, set by writing it to the mode file: "exit" makes the server
    quit when it starts and when it is called, and "hang" makes it take a call and never
    answer. An empty or missing file is normal behavior."""
    try:
        return open(MODE).read().strip()
    except OSError:
        return ""

def log(msg):
    with open(LOG, "a") as f:
        f.write("%s  %s  %s\n" % (time.strftime("%Y-%m-%dT%H:%M:%S%z"), socket.gethostname(), msg))

def send(obj):
    sys.stdout.write(json.dumps(obj) + "\n"); sys.stdout.flush()

log("server started pid=%d" % os.getpid())
if mode() == "exit":
    log("mode exit: quitting at start")
    sys.exit(1)
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
            "serverInfo": {"name": "cogmer-host-probe", "version": "0.0.1"}}})
    elif method == "tools/list":
        send({"jsonrpc": "2.0", "id": mid, "result": {"tools": [{
            "name": "ping",
            "description": "Reports that the probe on the user's own machine was reached. Takes an optional event name.",
            "inputSchema": {"type": "object", "properties": {"event": {"type": "string"}}}}]}})
    elif method == "tools/call":
        args = ((m.get("params") or {}).get("arguments")) or {}
        if mode() == "exit":
            log("mode exit: quitting on a call")
            os._exit(1)
        if mode() == "hang":
            log("mode hang: taking a call and not answering")
            time.sleep(600)
            continue
        shape = []
        for k, v in args.items():
            if k == "event":
                continue
            v = str(v)
            shape.append("%s: %d chars%s%s" % (k, len(v), ", UNRESOLVED" if "${" in v else "",
                                               (", first8=" + v[:8]) if k == "session" else ""))
        log("ping event=%s | %s" % (args.get("event", ""), "; ".join(shape)))
        fmt = args.get("format", "plain")
        text = "[cogmer-host-probe %s] reached %s at %s (event %s)" % (
            fmt, socket.gethostname(), time.strftime("%H:%M:%S"), args.get("event", "none"))
        if fmt == "json":
            # The structured form a hook may return, as Codex's hooks do.
            text = json.dumps({"hookSpecificOutput": {"hookEventName": args.get("event", ""),
                                                      "additionalContext": text}})
        send({"jsonrpc": "2.0", "id": mid, "result": {"content": [{"type": "text", "text": text}]}})
    elif mid is not None:
        send({"jsonrpc": "2.0", "id": mid, "error": {"code": -32601, "message": "method not found"}})
log("server stopped")
