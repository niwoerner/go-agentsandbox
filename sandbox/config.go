package sandbox

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// FileConfig represents the JSON config file structure.
type FileConfig struct {
	AllowWrite   []string `json:"allowWrite,omitempty"`
	DenyRead     []string `json:"denyRead,omitempty"`
	CleanEnv     *bool    `json:"cleanEnv,omitempty"`
	EnvAllowlist []string `json:"envAllowlist,omitempty"`
	EnvDenylist  []string `json:"envDenylist,omitempty"`
}

// DefaultConfigPath returns the default config file location.
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".agent", "sandbox", "config.json")
}

// LoadConfigFile loads and parses a config file.
// Returns nil if file doesn't exist (not an error).
func LoadConfigFile(path string) (*FileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var cfg FileConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// MergeConfig merges file config into base config.
// File config overrides base config; empty/omitted fields use base defaults.
func MergeConfig(base Config, file *FileConfig) Config {
	if file == nil {
		return base
	}

	// AllowWrite: non-empty overrides defaults
	if len(file.AllowWrite) > 0 {
		base.AllowWrite = file.AllowWrite
	}

	// DenyRead: non-empty overrides defaults
	if len(file.DenyRead) > 0 {
		base.DenyRead = file.DenyRead
	}

	// CleanEnv: explicit value overrides default
	if file.CleanEnv != nil {
		base.CleanEnv = *file.CleanEnv
	}

	// EnvAllowlist: non-empty overrides defaults
	if len(file.EnvAllowlist) > 0 {
		base.EnvAllowlist = file.EnvAllowlist
	}

	// EnvDenylist: non-empty overrides defaults
	if len(file.EnvDenylist) > 0 {
		base.EnvDenylist = file.EnvDenylist
	}

	return base
}

// IsWildcard checks if a path is the wildcard "*".
func IsWildcard(path string) bool {
	return path == "*"
}

// HasWildcard checks if a path list contains "*".
func HasWildcard(paths []string) bool {
	for _, p := range paths {
		if IsWildcard(p) {
			return true
		}
	}
	return false
}
