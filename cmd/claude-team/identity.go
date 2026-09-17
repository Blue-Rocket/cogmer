package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Identity is the stable per-machine peer identity required by spec §6.
type Identity struct {
	PeerID          string `json:"peerId"`
	UserID          string `json:"userId"`
	UserDisplayName string `json:"userDisplayName"`
	MachineID       string `json:"machineId"`
}

// Config separates room membership (shareable) from identity (personal), per §28.
type Config struct {
	Room string `json:"room"`
}

func homeDir() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(h, ".claude-team")
}

func randomID() string {
	b := make([]byte, 10)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// LoadIdentity reads ~/.claude-team/identity.json, creating a default on first run.
func LoadIdentity() (*Identity, error) {
	path := filepath.Join(homeDir(), "identity.json")
	data, err := os.ReadFile(path)
	if err == nil {
		var id Identity
		if err := json.Unmarshal(data, &id); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		return &id, nil
	}
	if !os.IsNotExist(err) {
		return nil, err
	}

	user := os.Getenv("USER")
	if user == "" {
		user = os.Getenv("USERNAME") // Windows
	}
	if user == "" {
		user = "developer"
	}
	host, _ := os.Hostname()
	if host == "" {
		host = "unknown-machine"
	}
	id := &Identity{
		PeerID:          "peer-" + randomID(),
		UserID:          strings.ToLower(user),
		UserDisplayName: strings.ToUpper(user[:1]) + user[1:],
		MachineID:       host,
	}
	if err := os.MkdirAll(homeDir(), 0o755); err != nil {
		return nil, err
	}
	buf, _ := json.MarshalIndent(id, "", "  ")
	if err := os.WriteFile(path, buf, 0o600); err != nil {
		return nil, err
	}
	return id, nil
}

// LoadConfig resolves the active room. CLAUDE_TEAM_ROOM wins so a single
// machine can drive several rooms during testing.
func LoadConfig() *Config {
	cfg := &Config{Room: "default"}
	path := filepath.Join(homeDir(), "config.json")
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, cfg)
	}
	if r := os.Getenv("CLAUDE_TEAM_ROOM"); r != "" {
		cfg.Room = r
	}
	if cfg.Room == "" {
		cfg.Room = "default"
	}
	return cfg
}
