# Relay spike

Material for the open item "A relay change would break addresses users already hold" in `docs/open.md`,
deleted with it. It asks whether a relay cogmer runs could give the page that reads a cloud session's room a
small client in place of tailcat's 27 MB browser build.

- `cmd/relay` joins a browser's WebSocket to a daemon's: a daemon keeps a control connection at
  `/serve?key=K`, a browser dials `/dial?key=K`, the relay tells the daemon `dial <id>`, the daemon opens
  `/accept?id=<id>`, and the relay copies messages both ways and reads none of them.
- `cmd/responder` stands in for a daemon: it answers each dial with Noise_NK_25519_AESGCM_SHA256 under its
  static key and then speaks D-301's lines, serving one sample room of 40 turns, 24,582 bytes.
- `noise` is the handshake in Go with the standard library, with a test that runs both ends and refuses a
  first message made for another key.
- `page/client.js` is the browser's side with WebCrypto and no library, about 5 KB, and `page/index.html`
  times the page's start, the relay connection, the handshake and the first data.
  `node page/interop.test.mjs <relay> <key>` runs the client against a running relay and responder.

It is its own Go module, so the repository's `go test ./...` does not build it. Build it with
`go build ./cmd/...` here. On the droplet the relay runs on `127.0.0.1:8091` behind Caddy as
`relay.probe.cogmer.ai`, and the responder beside it, both from `~/relay-spike`, detached:

```sh
cd ~/relay-spike && setsid nohup ./relay -listen 127.0.0.1:8091 >> relay.log 2>&1 < /dev/null &
cd ~/relay-spike && setsid nohup ./responder -relay ws://127.0.0.1:8091 -key responder.key >> responder.log 2>&1 < /dev/null &
```

The page is `https://probe.cogmer.ai/relay/#key=<the responder's key>`, which `responder.log` prints.
The responder runs on the same machine as the relay, so a run here measures the client's cost and not the
distance between a relay and a daemon.
