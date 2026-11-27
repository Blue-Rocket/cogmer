// Command responder stands in for a daemon in the relay spike. It keeps a control connection to the relay,
// answers each dial with the Noise NK handshake under its static key, and then speaks D-301's lines: a token
// line, then for each request a path line, answered by "<status> <media type> <length>" and the body, each
// sent as one encrypted message. It serves one sample room.
package main

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/coder/websocket"
	"relayspike/noise"
)

func main() {
	relay := flag.String("relay", "ws://127.0.0.1:8091", "relay base address")
	keyFile := flag.String("key", "responder.key", "file holding the static private key")
	flag.Parse()
	static := loadKey(*keyFile)
	pub := base64.RawURLEncoding.EncodeToString(static.PublicKey().Bytes())
	log.Printf("responder key %s", pub)
	room := sampleRoom()
	for {
		run(*relay, pub, static, room)
		time.Sleep(2 * time.Second)
	}
}

func loadKey(path string) *ecdh.PrivateKey {
	if b, err := os.ReadFile(path); err == nil {
		k, err := ecdh.X25519().NewPrivateKey(b)
		if err == nil {
			return k
		}
	}
	k, _ := ecdh.X25519().GenerateKey(rand.Reader)
	if err := os.WriteFile(path, k.Bytes(), 0o600); err != nil {
		log.Fatal(err)
	}
	return k
}

func run(relay, pub string, static *ecdh.PrivateKey, room []byte) {
	ctx := context.Background()
	c, _, err := websocket.Dial(ctx, relay+"/serve?key="+pub, nil)
	if err != nil {
		log.Printf("serve: %v", err)
		return
	}
	defer c.CloseNow()
	for {
		_, msg, err := c.Read(ctx)
		if err != nil {
			log.Printf("serve ended: %v", err)
			return
		}
		if id, ok := strings.CutPrefix(string(msg), "dial "); ok {
			go session(relay, id, static, room)
		}
	}
}

func session(relay, id string, static *ecdh.PrivateKey, room []byte) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	c, _, err := websocket.Dial(ctx, relay+"/accept?id="+id, nil)
	if err != nil {
		log.Printf("accept: %v", err)
		return
	}
	defer c.CloseNow()
	c.SetReadLimit(1 << 22)
	_, m1, err := c.Read(ctx)
	if err != nil {
		return
	}
	m2, recv, send, err := noise.Responder(static, m1)
	if err != nil {
		log.Printf("handshake: %v", err)
		return
	}
	if err := c.Write(ctx, websocket.MessageBinary, m2); err != nil {
		return
	}
	read := func() (string, error) {
		_, m, err := c.Read(ctx)
		if err != nil {
			return "", err
		}
		p, err := recv.Open(m)
		return strings.TrimRight(string(p), "\n"), err
	}
	write := func(b []byte) error { return c.Write(ctx, websocket.MessageBinary, send.Seal(b)) }
	if tok, err := read(); err != nil || tok != "spike" {
		return
	}
	for {
		path, err := read()
		if err != nil {
			return
		}
		body, status := room, 200
		if path != "/room" {
			body, status = []byte("not found\n"), 404
		}
		write([]byte(fmt.Sprintf("%d application/json %d\n", status, len(body))))
		write(body)
	}
}

func sampleRoom() []byte {
	type turn struct {
		Who, Prompt, Response string
		At                    time.Time
	}
	var turns []turn
	for i := 0; i < 40; i++ {
		turns = append(turns, turn{
			Who:      []string{"Alice", "Ben"}[i%2],
			Prompt:   fmt.Sprintf("Turn %d: a prompt of ordinary length about the work in this room.", i+1),
			Response: strings.Repeat("A response of the length an assistant's answer usually has. ", 8),
			At:       time.Date(2026, 11, 27, 9, i, 0, 0, time.UTC),
		})
	}
	b, _ := json.Marshal(map[string]any{"room": "spike-room", "turns": turns})
	return b
}
