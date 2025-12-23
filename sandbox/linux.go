//go:build linux

package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"syscall"
)

type linuxSandbox struct {
	cfg      Config
	bwrapBin string
}

func newLinux(cfg Config) (Sandbox, error) {
	bin, err := exec.LookPath("bwrap")
	if err != nil {
		return nil, fmt.Errorf("bubblewrap not found: install with 'apt install bubblewrap' or 'dnf install bubblewrap'")
	}

	s := &linuxSandbox{cfg: cfg, bwrapBin: bin}

	if err := s.testUserNamespace(); err != nil {
		return nil, fmt.Errorf("user namespaces disabled: run 'sudo sysctl kernel.unprivileged_userns_clone=1': %w", err)
	}

	return s, nil
}

func (s *linuxSandbox) Run(ctx context.Context, cmd string) ([]byte, int, error) {
	return s.RunWithStdin(ctx, cmd, nil)
}

func (s *linuxSandbox) RunWithStdin(ctx context.Context, cmd string, stdin io.Reader) ([]byte, int, error) {
	args := s.buildArgs(cmd)

	if s.cfg.DryRun {
		return []byte(s.dryRunOutput(args)), 0, nil
	}

	c := exec.Command(s.bwrapBin, args...)
	c.Env = buildEnv(s.cfg)
	c.Stdin = stdin
	// Create new process group so we can kill all children
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Use a buffer to capture combined output
	var buf bytes.Buffer
	c.Stdout = &buf
	c.Stderr = &buf

	if err := c.Start(); err != nil {
		return nil, 0, err
	}

	// Watch for context cancellation
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			if c.Process != nil {
				syscall.Kill(-c.Process.Pid, syscall.SIGKILL)
			}
		case <-done:
		}
	}()

	// Wait for process to finish
	waitErr := c.Wait()
	close(done)

	output := buf.Bytes()
	exitCode := 0
	if c.ProcessState != nil {
		exitCode = c.ProcessState.ExitCode()
	}

	// If context was cancelled, return context error
	if ctx.Err() != nil {
		return output, exitCode, ctx.Err()
	}
	return output, exitCode, waitErr
}

func (s *linuxSandbox) buildArgs(cmd string) []string {
	args := []string{
		"--share-net", // Allow network access
		"--die-with-parent",
	}

	// Read-only bind mount of root filesystem
	args = append(args, "--ro-bind", "/", "/")

	// Writable bind mounts (skip paths in DenyRead)
	for _, path := range s.cfg.AllowWrite {
		if pathInDenyRead(path, s.cfg.DenyRead) {
			continue
		}
		args = append(args, "--bind", path, path)
	}

	// Hide sensitive directories with tmpfs overlay
	// This must come after ro-bind to overlay the read-only mount
	for _, path := range s.cfg.DenyRead {
		args = append(args, "--tmpfs", path)
	}

	// Mount /dev and /proc for basic functionality
	args = append(args, "--dev", "/dev")
	args = append(args, "--proc", "/proc")

	// Set working directory
	args = append(args, "--chdir", s.cfg.Workdir)

	// Command to execute
	args = append(args, "sh", "-c", cmd)

	return args
}

func (s *linuxSandbox) testUserNamespace() error {
	c := exec.Command(s.bwrapBin, "--ro-bind", "/", "/", "/usr/bin/true")
	return c.Run()
}

func (s *linuxSandbox) dryRunOutput(args []string) string {
	return fmt.Sprintf("%s %s", s.bwrapBin, strings.Join(args, " "))
}
