//go:build darwin

package sandbox

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type darwinSandbox struct {
	cfg     Config
	profile string //sandbox-exec profiler
}

func newDarwin(cfg Config) (Sandbox, error) {
	s := &darwinSandbox{cfg: cfg}
	s.profile = s.generateProfile()

	if err := s.validateProfile(); err != nil {
		return nil, fmt.Errorf("invalid sandbox profile: %w", err)
	}

	return s, nil
}

func (s *darwinSandbox) Run(ctx context.Context, cmd string) ([]byte, int, error) {
	return s.RunWithStdin(ctx, cmd, nil)
}

func (s *darwinSandbox) RunWithStdin(ctx context.Context, cmd string, stdin io.Reader) ([]byte, int, error) {
	if s.cfg.DryRun {
		return []byte(s.dryRunOutput(cmd)), 0, nil
	}

	c := exec.CommandContext(ctx, "sandbox-exec", "-p", s.profile, "sh", "-c", cmd)
	c.Env = buildEnv(s.cfg)
	c.Stdin = stdin
	output, err := c.CombinedOutput()

	exitCode := 0
	if c.ProcessState != nil {
		exitCode = c.ProcessState.ExitCode()
	}

	return output, exitCode, err
}

func (s *darwinSandbox) generateProfile() string {
	var sb strings.Builder

	sb.WriteString("(version 1)\n")
	sb.WriteString("(allow default)\n")
	sb.WriteString("(allow network*)\n")

	// Handle write permissions
	if HasWildcard(s.cfg.AllowWrite) {
		// Wildcard: allow all writes (don't add deny rule)
	} else {
		// Deny all file writes by default
		sb.WriteString("(deny file-write*)\n")

		// Allow writes to specific paths
		for _, path := range s.cfg.AllowWrite {
			// Skip if path is in DenyRead (DenyRead takes precedence)
			if pathInDenyRead(path, s.cfg.DenyRead) {
				continue
			}
			sb.WriteString(fmt.Sprintf("(allow file-write* (subpath %q))\n", path))
		}
	}

	// Handle read restrictions
	if HasWildcard(s.cfg.DenyRead) {
		// Wildcard: deny all reads (except essential system paths for execution)
		sb.WriteString("(deny file-read*)\n")
		// Must allow reads from essential paths for command execution
		sb.WriteString("(allow file-read* (subpath \"/usr\"))\n")
		sb.WriteString("(allow file-read* (subpath \"/bin\"))\n")
		sb.WriteString("(allow file-read* (subpath \"/sbin\"))\n")
		sb.WriteString("(allow file-read* (subpath \"/var\"))\n")
		sb.WriteString("(allow file-read* (subpath \"/private\"))\n")
		sb.WriteString("(allow file-read* (subpath \"/dev\"))\n")
		sb.WriteString("(allow file-read* (subpath \"/System\"))\n")
		sb.WriteString("(allow file-read* (subpath \"/Library\"))\n")
	} else {
		// Deny reads from specific sensitive paths
		for _, path := range s.cfg.DenyRead {
			sb.WriteString(fmt.Sprintf("(deny file-read* (subpath %q))\n", path))
		}
	}

	return sb.String()
}

func (s *darwinSandbox) validateProfile() error {
	// Run a no-op command to validate the profile syntax
	c := exec.Command("sandbox-exec", "-p", s.profile, "/usr/bin/true")
	if err := c.Run(); err != nil {
		return fmt.Errorf("profile validation failed: %w", err)
	}
	return nil
}

func (s *darwinSandbox) dryRunOutput(cmd string) string {
	return fmt.Sprintf("sandbox-exec -p '%s' sh -c '%s'", s.profile, cmd)
}
