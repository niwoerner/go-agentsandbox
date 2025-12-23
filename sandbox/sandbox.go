package sandbox

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Config defines sandbox configuration.
type Config struct {
	// Filesystem
	Workdir    string   // Working directory (default: cwd)
	AllowWrite []string // Writable paths (default: workdir, /tmp)
	DenyRead   []string // Protected paths (default: ~/.ssh, ~/.aws, etc.)

	// Environment
	CleanEnv     bool     // If true, start with empty env (default: false)
	EnvAllowlist []string // When CleanEnv=true, only pass these vars
	EnvDenylist  []string // When CleanEnv=false, remove these vars

	// Execution
	DryRun bool // If true, return command string instead of executing
}

// Sandbox executes commands in a restricted environment.
type Sandbox interface {
	Run(ctx context.Context, command string) (output []byte, exitCode int, err error)
	RunWithStdin(ctx context.Context, command string, stdin io.Reader) (output []byte, exitCode int, err error)
}

// hardcodedDefaults returns the built-in default configuration.
func hardcodedDefaults() Config {
	cwd, _ := os.Getwd()
	return Config{
		Workdir:    cwd,
		AllowWrite: []string{cwd, "/tmp"},
		DenyRead:   []string{"~/.ssh", "~/.aws", "~/.gnupg", "~/.kube", "~/.docker", "~/.config/gh"},
		CleanEnv:   false,
	}
}

// DefaultConfig returns config merged from hardcoded defaults and config file.
// Config file at ~/.agent/sandbox/config.json is loaded if it exists.
// Use DefaultConfigWithPath to specify a custom config file path.
func DefaultConfig() Config {
	return DefaultConfigWithPath(DefaultConfigPath())
}

// DefaultConfigWithPath returns config merged from hardcoded defaults and specified config file.
// If configPath is empty or file doesn't exist, returns hardcoded defaults only.
func DefaultConfigWithPath(configPath string) Config {
	base := hardcodedDefaults()

	if configPath == "" {
		return base
	}

	fileCfg, err := LoadConfigFile(configPath)
	if err != nil {
		// Log error but continue with defaults
		log.Printf("warning: failed to load config file %q: %v", configPath, err)
		return base
	}

	return MergeConfig(base, fileCfg)
}

// New creates a platform-specific sandbox.
// Returns error if backend unavailable or invalid paths.
// Logs warning if workdir doesn't exist.
func New(cfg Config) (Sandbox, error) {
	// Expand and validate paths
	var err error
	cfg.Workdir, err = expandPath(cfg.Workdir)
	if err != nil {
		return nil, fmt.Errorf("invalid workdir: %w", err)
	}

	for i, p := range cfg.AllowWrite {
		cfg.AllowWrite[i], err = expandPath(p)
		if err != nil {
			return nil, fmt.Errorf("invalid AllowWrite path %q: %w", p, err)
		}
	}

	for i, p := range cfg.DenyRead {
		cfg.DenyRead[i], err = expandPath(p)
		if err != nil {
			// DenyRead paths might not exist (e.g., ~/.aws on systems without AWS CLI)
			// Just skip expansion errors for non-existent paths
			expanded, _ := expandPathNoResolve(p)
			cfg.DenyRead[i] = expanded
		}
	}

	validatePaths(&cfg)

	switch runtime.GOOS {
	case "darwin":
		return newDarwin(cfg)
	case "linux":
		return newLinux(cfg)
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// expandPath resolves ~ and relative paths to absolute paths with symlink resolution.
func expandPath(p string) (string, error) {
	p, err := expandPathNoResolve(p)
	if err != nil {
		return "", err
	}

	// Resolve symlinks to prevent bypasses
	resolved, err := filepath.EvalSymlinks(p)
	if err != nil {
		if os.IsNotExist(err) {
			return p, nil
		}
		return "", err
	}
	return resolved, nil
}

// expandPathNoResolve expands ~ and relative paths without resolving symlinks.
func expandPathNoResolve(p string) (string, error) {
	if strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot expand ~: %w", err)
		}
		p = filepath.Join(home, p[2:])
	}

	return filepath.Abs(p)
}

// validatePaths checks paths and logs warnings.
func validatePaths(cfg *Config) {
	if _, err := os.Stat(cfg.Workdir); err != nil {
		log.Printf("warning: workdir %q does not exist", cfg.Workdir)
	}
}

// buildEnv constructs environment variables based on config.
func buildEnv(cfg Config) []string {
	if cfg.CleanEnv {
		env := []string{}

		// Add allowlisted vars
		for _, key := range cfg.EnvAllowlist {
			if val, ok := os.LookupEnv(key); ok {
				env = append(env, key+"="+val)
			}
		}

		// Always include essential vars
		for _, key := range []string{"PATH", "HOME", "USER", "TERM"} {
			if val, ok := os.LookupEnv(key); ok {
				// Don't duplicate if already in allowlist
				found := false
				for _, e := range env {
					if strings.HasPrefix(e, key+"=") {
						found = true
						break
					}
				}
				if !found {
					env = append(env, key+"="+val)
				}
			}
		}
		return env
	}

	// Start with current env, remove denylisted vars
	denySet := make(map[string]bool)
	for _, key := range cfg.EnvDenylist {
		denySet[key] = true
	}

	env := []string{}
	for _, e := range os.Environ() {
		key := strings.SplitN(e, "=", 2)[0]
		if !denySet[key] {
			env = append(env, e)
		}
	}
	return env
}

// pathInDenyRead checks if a path should be denied based on DenyRead config.
// DenyRead always takes precedence over AllowWrite.
func pathInDenyRead(path string, denyRead []string) bool {
	for _, denied := range denyRead {
		if path == denied || strings.HasPrefix(path, denied+string(filepath.Separator)) {
			return true
		}
	}
	return false
}
