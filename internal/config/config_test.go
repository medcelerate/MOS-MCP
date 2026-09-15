package config

import (
	"path/filepath"
	"testing"

	"github.com/medcelerate/MOS-MCP/internal/mos"
)

func TestLoadDefaultsWhenMissing(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Role != RoleBoth {
		t.Fatalf("role = %q, want both", cfg.Role)
	}
	if cfg.Listen.LowerPort != 10540 {
		t.Fatalf("lower port = %d, want 10540", cfg.Listen.LowerPort)
	}
	if !cfg.HasProfile(2) || cfg.HasProfile(6) {
		t.Fatalf("unexpected profile set: %v", cfg.Profiles)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mos-mcp.yaml")
	cfg := Default()
	cfg.MosID = "custom.mos"
	cfg.Profiles = []int{0, 2}
	cfg.Peers = []mos.PeerConfig{{Name: "viz", Host: "10.0.0.5"}}
	if err := cfg.Save(path); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.MosID != "custom.mos" {
		t.Fatalf("mosID = %q", loaded.MosID)
	}
	if loaded.HasProfile(1) {
		t.Fatalf("profile 1 should be disabled: %v", loaded.Profiles)
	}
	if len(loaded.Peers) != 1 || loaded.Peers[0].Name != "viz" {
		t.Fatalf("peers not round-tripped: %+v", loaded.Peers)
	}
}

func TestValidate(t *testing.T) {
	cfg := Default()
	cfg.Role = "bogus"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for bad role")
	}
	cfg = Default()
	cfg.Peers = []mos.PeerConfig{{Name: "a", Host: "h"}, {Name: "a", Host: "h2"}}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for duplicate peer name")
	}
}

func TestMachInfoAdvertisesProfiles(t *testing.T) {
	cfg := Default()
	cfg.Profiles = []int{0, 1}
	mi := cfg.MachInfo()
	if mi.SupportedProfiles == nil || len(mi.SupportedProfiles.Profiles) != 2 {
		t.Fatalf("expected 2 advertised profiles, got %+v", mi.SupportedProfiles)
	}
}
