// Package config defines the bridge's configuration file format and the
// loading, validation and persistence logic shared by the CLI and the web UI.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/medcelerate/MOS-MCP/internal/mos"
	"github.com/medcelerate/MOS-MCP/internal/mos/messages"
	"gopkg.in/yaml.v3"
)

// Role values.
const (
	RoleClient = "client"
	RoleDevice = "device"
	RoleBoth   = "both"
)

// MCP transport values.
const (
	TransportStdio = "stdio"
	TransportHTTP  = "http"
	TransportBoth  = "both"
)

// Config is the full bridge configuration.
type Config struct {
	// MosID and NcsID identify this bridge on the MOS network.
	MosID string `yaml:"mosID"`
	NcsID string `yaml:"ncsID"`

	// Role selects client (dial out), device (listen), or both.
	Role string `yaml:"role"`

	// Profiles lists the MOS profiles to enable (0,1,2,3,4). MCP tools and
	// device replies are gated on this set.
	Profiles []int `yaml:"profiles"`

	// Peers are the remote MOS systems this bridge can dial (client role).
	Peers []mos.PeerConfig `yaml:"peers"`

	// Listen holds the TCP ports the device server binds (device role).
	Listen ListenConfig `yaml:"listen"`

	// Device describes this bridge for reqMachInfo responses.
	Device DeviceInfo `yaml:"device"`

	MCP MCPConfig `yaml:"mcp"`
	Web WebConfig `yaml:"web"`
	Log LogConfig `yaml:"log"`

	DialTimeoutSeconds int `yaml:"dialTimeoutSeconds"`
	InboxCapacity      int `yaml:"inboxCapacity"`
}

// ListenConfig holds the device-role bind ports.
type ListenConfig struct {
	LowerPort int `yaml:"lowerPort"`
	UpperPort int `yaml:"upperPort"`
	QueryPort int `yaml:"queryPort"`
}

// DeviceInfo populates the listMachInfo response.
type DeviceInfo struct {
	Manufacturer string `yaml:"manufacturer"`
	Model        string `yaml:"model"`
	HwRev        string `yaml:"hwRev"`
	SwRev        string `yaml:"swRev"`
}

// MCPConfig selects how MCP clients connect.
type MCPConfig struct {
	Transport string     `yaml:"transport"`
	HTTP      HTTPConfig `yaml:"http"`
}

// HTTPConfig is the bind address for the Streamable HTTP MCP endpoint.
type HTTPConfig struct {
	Addr string `yaml:"addr"`
}

// WebConfig controls the embedded admin console.
type WebConfig struct {
	Enabled bool   `yaml:"enabled"`
	Addr    string `yaml:"addr"`
}

// LogConfig controls logging.
type LogConfig struct {
	Level string `yaml:"level"`
}

// Default returns a configuration with sensible defaults filled in.
func Default() Config {
	return Config{
		MosID:    "mos-mcp.bridge",
		NcsID:    "mos-mcp.ncs",
		Role:     RoleBoth,
		Profiles: []int{0, 1, 2, 3, 4},
		Listen:   ListenConfig{LowerPort: 10540, UpperPort: 10541, QueryPort: 10542},
		Device: DeviceInfo{
			Manufacturer: "mos-mcp",
			Model:        "MOS-MCP Bridge",
			SwRev:        "0.1.0",
		},
		MCP:                MCPConfig{Transport: TransportStdio, HTTP: HTTPConfig{Addr: "127.0.0.1:8080"}},
		Web:                WebConfig{Enabled: true, Addr: "127.0.0.1:8088"},
		Log:                LogConfig{Level: "info"},
		DialTimeoutSeconds: 10,
		InboxCapacity:      200,
	}
}

// Load reads a YAML config from path, applies defaults to unset fields and then
// environment overrides. A missing file yields the default configuration.
func Load(path string) (*Config, error) {
	cfg := Default()
	if path != "" {
		data, err := os.ReadFile(path)
		switch {
		case err == nil:
			if err := yaml.Unmarshal(data, &cfg); err != nil {
				return nil, fmt.Errorf("parse config %s: %w", path, err)
			}
		case os.IsNotExist(err):
			// Keep defaults.
		default:
			return nil, fmt.Errorf("read config %s: %w", path, err)
		}
	}
	cfg.applyDefaults()
	cfg.applyEnv()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// applyDefaults backfills zero values that a partial file may leave unset.
func (c *Config) applyDefaults() {
	d := Default()
	if c.MosID == "" {
		c.MosID = d.MosID
	}
	if c.NcsID == "" {
		c.NcsID = d.NcsID
	}
	if c.Role == "" {
		c.Role = d.Role
	}
	if len(c.Profiles) == 0 {
		c.Profiles = d.Profiles
	}
	if c.Listen.LowerPort == 0 {
		c.Listen.LowerPort = d.Listen.LowerPort
	}
	if c.Listen.UpperPort == 0 {
		c.Listen.UpperPort = d.Listen.UpperPort
	}
	if c.Listen.QueryPort == 0 {
		c.Listen.QueryPort = d.Listen.QueryPort
	}
	if c.MCP.Transport == "" {
		c.MCP.Transport = d.MCP.Transport
	}
	if c.MCP.HTTP.Addr == "" {
		c.MCP.HTTP.Addr = d.MCP.HTTP.Addr
	}
	if c.Web.Addr == "" {
		c.Web.Addr = d.Web.Addr
	}
	if c.Log.Level == "" {
		c.Log.Level = d.Log.Level
	}
	if c.DialTimeoutSeconds == 0 {
		c.DialTimeoutSeconds = d.DialTimeoutSeconds
	}
	if c.InboxCapacity == 0 {
		c.InboxCapacity = d.InboxCapacity
	}
	if c.Device.Model == "" {
		c.Device.Model = d.Device.Model
	}
}

// applyEnv overrides selected fields from MOSMCP_* environment variables.
func (c *Config) applyEnv() {
	if v := os.Getenv("MOSMCP_MOSID"); v != "" {
		c.MosID = v
	}
	if v := os.Getenv("MOSMCP_NCSID"); v != "" {
		c.NcsID = v
	}
	if v := os.Getenv("MOSMCP_ROLE"); v != "" {
		c.Role = v
	}
	if v := os.Getenv("MOSMCP_MCP_TRANSPORT"); v != "" {
		c.MCP.Transport = v
	}
	if v := os.Getenv("MOSMCP_MCP_HTTP_ADDR"); v != "" {
		c.MCP.HTTP.Addr = v
	}
	if v := os.Getenv("MOSMCP_WEB_ADDR"); v != "" {
		c.Web.Addr = v
	}
	if v := os.Getenv("MOSMCP_WEB_ENABLED"); v != "" {
		c.Web.Enabled, _ = strconv.ParseBool(v)
	}
	if v := os.Getenv("MOSMCP_LOG_LEVEL"); v != "" {
		c.Log.Level = v
	}
}

// Validate checks the configuration for internal consistency.
func (c *Config) Validate() error {
	switch c.Role {
	case RoleClient, RoleDevice, RoleBoth:
	default:
		return fmt.Errorf("invalid role %q (want client|device|both)", c.Role)
	}
	switch c.MCP.Transport {
	case TransportStdio, TransportHTTP, TransportBoth:
	default:
		return fmt.Errorf("invalid mcp.transport %q (want stdio|http|both)", c.MCP.Transport)
	}
	for _, p := range c.Profiles {
		if p < 0 || p > 7 {
			return fmt.Errorf("invalid profile %d", p)
		}
	}
	seen := map[string]bool{}
	for _, peer := range c.Peers {
		if peer.Name == "" {
			return fmt.Errorf("peer with empty name")
		}
		if seen[peer.Name] {
			return fmt.Errorf("duplicate peer name %q", peer.Name)
		}
		seen[peer.Name] = true
	}
	return nil
}

// Save writes the config to path as YAML using an atomic temp-file rename.
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".mos-mcp-*.yaml")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, path)
}

// HasProfile reports whether MOS profile n is enabled.
func (c *Config) HasProfile(n int) bool {
	for _, p := range c.Profiles {
		if p == n {
			return true
		}
	}
	return false
}

// Identity returns the mosID/ncsID pair.
func (c *Config) Identity() messages.Identity {
	return messages.Identity{MosID: c.MosID, NcsID: c.NcsID}
}

// ListenPorts returns the distinct device-role listen ports.
func (c *Config) ListenPorts() []int {
	return []int{c.Listen.LowerPort, c.Listen.UpperPort, c.Listen.QueryPort}
}

// MachInfo builds the listMachInfo payload advertised to peers.
func (c *Config) MachInfo() *messages.ListMachInfo {
	sp := &messages.SupportedProfiles{DeviceType: "MOS"}
	for _, p := range c.Profiles {
		sp.Profiles = append(sp.Profiles, messages.MosProfile{Number: p, Supported: "YES"})
	}
	return &messages.ListMachInfo{
		Manufacturer:      c.Device.Manufacturer,
		Model:             c.Device.Model,
		HwRev:             c.Device.HwRev,
		SwRev:             c.Device.SwRev,
		ID:                c.MosID,
		Time:              messages.NowEBU(),
		MosRev:            "2.8.5",
		SupportedProfiles: sp,
	}
}

// EnabledProfilesString renders the enabled profiles as e.g. "0,1,2".
func (c *Config) EnabledProfilesString() string {
	parts := make([]string, 0, len(c.Profiles))
	for _, p := range c.Profiles {
		parts = append(parts, strconv.Itoa(p))
	}
	return strings.Join(parts, ",")
}
