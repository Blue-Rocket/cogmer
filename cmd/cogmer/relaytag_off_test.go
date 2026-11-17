//go:build !ts_debug_websockets

package main

// builtWithWebSocketRelay says this test binary has Tailscale's WebSocket relay client
// compiled in, as a release does.
const builtWithWebSocketRelay = false
