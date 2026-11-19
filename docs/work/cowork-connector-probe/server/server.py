#!/usr/bin/env python3
"""A throwaway remote MCP server for one question: can a Cowork cloud session's mcp_tool hook,
or its model, call a remote connector, and under what server name?

It speaks MCP over streamable HTTP at /mcp with JSON replies, needs no auth, and offers one
tool, `ping`, which takes any string arguments. It also accepts POST /egress, which the probe
plugin's command hook sends from the sandbox. Every request is appended as one JSON line to
the log named on the command line: time, method, path, the caller's address as the tunnel
reports it, the user-agent, the JSON-RPC method, and for a call its arguments. It logs no
other header and holds no secret."""
import json, sys, time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

LOG = sys.argv[1]
PORT = int(sys.argv[2]) if len(sys.argv) > 2 else 8790


def log(entry):
    entry["t"] = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())
    with open(LOG, "a") as f:
        f.write(json.dumps(entry, sort_keys=True) + "\n")


class H(BaseHTTPRequestHandler):
    def base(self):
        return {"method": self.command, "path": self.path,
                "from": self.headers.get("Cf-Connecting-Ip") or self.client_address[0],
                "ua": self.headers.get("User-Agent", "")}

    def reply(self, code, obj=None, extra=None):
        body = b"" if obj is None else json.dumps(obj).encode()
        self.send_response(code)
        if obj is not None:
            self.send_header("Content-Type", "application/json")
        for k, v in (extra or {}).items():
            self.send_header(k, v)
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_GET(self):
        log(self.base())
        self.reply(404 if self.path.startswith("/.well-known") else 405)

    def do_DELETE(self):
        log(self.base())
        self.reply(200)

    def do_POST(self):
        n = int(self.headers.get("Content-Length") or 0)
        raw = self.rfile.read(min(n, 65536))
        e = self.base()
        if self.path.startswith("/http-hook"):
            # An http hook's body is the hook's input, which holds the prompt or the reply, so
            # only its field names and event are logged.
            try:
                d = json.loads(raw)
                e["hook_event"] = d.get("hook_event_name")
                e["keys"] = sorted(d)
            except Exception:
                e["bad"] = True
            log(e)
            return self.reply(200, {})
        if self.path.startswith("/egress"):
            e["body"] = raw.decode("utf-8", "replace")[:2000]
            log(e)
            return self.reply(200, {"ok": True})
        if not self.path.startswith("/mcp"):
            log(e)
            return self.reply(404)
        try:
            m = json.loads(raw)
        except Exception:
            e["bad"] = True
            log(e)
            return self.reply(400)
        if isinstance(m, list):
            e["batch"] = True
            log(e)
            return self.reply(400)
        method, mid = m.get("method"), m.get("id")
        e["rpc"] = method
        params = m.get("params") or {}
        if method == "initialize":
            e["client"] = params.get("clientInfo")
            log(e)
            return self.reply(200, {"jsonrpc": "2.0", "id": mid, "result": {
                "protocolVersion": params.get("protocolVersion", "2025-06-18"),
                "capabilities": {"tools": {}},
                "serverInfo": {"name": "cprobe", "version": "0.1.0"}}},
                {"Mcp-Session-Id": "cprobe-%d" % int(time.time() * 1000)})
        if mid is None:
            log(e)
            return self.reply(202)
        if method == "tools/list":
            log(e)
            return self.reply(200, {"jsonrpc": "2.0", "id": mid, "result": {"tools": [{
                "name": "ping",
                "description": "Records that the connector probe was reached. Pass `via` naming who calls it.",
                "inputSchema": {"type": "object", "properties": {
                    "via": {"type": "string"}, "session": {"type": "string"}, "event": {"type": "string"}}}}]}})
        if method == "tools/call":
            e["tool"] = params.get("name")
            e["args"] = params.get("arguments")
            log(e)
            via = (params.get("arguments") or {}).get("via", "?")
            return self.reply(200, {"jsonrpc": "2.0", "id": mid, "result": {
                "content": [{"type": "text", "text": "connector probe reached via %s" % via}]}})
        if method == "ping":
            log(e)
            return self.reply(200, {"jsonrpc": "2.0", "id": mid, "result": {}})
        log(e)
        return self.reply(200, {"jsonrpc": "2.0", "id": mid, "error": {"code": -32601, "message": "method not found"}})

    def log_message(self, *a):
        pass


ThreadingHTTPServer(("127.0.0.1", PORT), H).serve_forever()
