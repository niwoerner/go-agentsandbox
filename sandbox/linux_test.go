//go:build linux

package sandbox

import (
	"slices"
	"strings"
	"testing"
)

func TestBuildArgs(t *testing.T) {
	cfg := Config{
		Workdir:    "/home/user/project",
		AllowWrite: []string{"/home/user/project", "/tmp"},
		DenyRead:   []string{"/home/user/.ssh", "/home/user/.aws"},
	}
	s := &linuxSandbox{cfg: cfg, bwrapBin: "/usr/bin/bwrap"}
	args := s.buildArgs("echo hello")

	// Network sharing
	if !slices.Contains(args, "--share-net") {
		t.Error("should contain --share-net")
	}

	// Read-only root
	if !containsSequence(args, "--ro-bind", "/", "/") {
		t.Error("should contain --ro-bind / /")
	}

	// Writable paths
	if !containsSequence(args, "--bind", "/home/user/project", "/home/user/project") {
		t.Error("should contain --bind for workdir")
	}
	if !containsSequence(args, "--bind", "/tmp", "/tmp") {
		t.Error("should contain --bind for /tmp")
	}

	// Hidden paths (tmpfs overlay)
	if !containsSequence(args, "--tmpfs", "/home/user/.ssh") {
		t.Error("should contain --tmpfs for .ssh")
	}
	if !containsSequence(args, "--tmpfs", "/home/user/.aws") {
		t.Error("should contain --tmpfs for .aws")
	}

	// Command at end
	if args[len(args)-1] != "echo hello" {
		t.Errorf("command should be at end, got %q", args[len(args)-1])
	}
}

func TestBuildArgs_PreservesOrder(t *testing.T) {
	cfg := Config{
		Workdir:    "/tmp",
		AllowWrite: []string{"/tmp"},
		DenyRead:   []string{"/home/user/.ssh"},
	}
	s := &linuxSandbox{cfg: cfg, bwrapBin: "/usr/bin/bwrap"}
	args := s.buildArgs("true")

	roBind := slices.Index(args, "--ro-bind")
	tmpfs := slices.Index(args, "--tmpfs")

	if roBind < 0 || tmpfs < 0 {
		t.Fatal("should contain both --ro-bind and --tmpfs")
	}

	if roBind >= tmpfs {
		t.Error("--ro-bind must come before --tmpfs")
	}
}

func TestBuildArgs_DenyReadTakesPrecedence(t *testing.T) {
	cfg := Config{
		Workdir:    "/tmp",
		AllowWrite: []string{"/home/user/.ssh"}, // Trying to allow write
		DenyRead:   []string{"/home/user/.ssh"}, // But DenyRead wins
	}
	s := &linuxSandbox{cfg: cfg, bwrapBin: "/usr/bin/bwrap"}
	args := s.buildArgs("true")

	// Should NOT have --bind for .ssh
	if containsSequence(args, "--bind", "/home/user/.ssh", "/home/user/.ssh") {
		t.Error("should not bind DenyRead path")
	}

	// Should have --tmpfs for .ssh
	if !containsSequence(args, "--tmpfs", "/home/user/.ssh") {
		t.Error("should tmpfs DenyRead path")
	}
}

func TestDryRunOutput_Linux(t *testing.T) {
	cfg := Config{
		Workdir:    "/tmp",
		AllowWrite: []string{"/tmp"},
		DryRun:     true,
	}
	s := &linuxSandbox{cfg: cfg, bwrapBin: "/usr/bin/bwrap"}
	args := s.buildArgs("echo hello")
	output := s.dryRunOutput(args)

	if !strings.Contains(output, "bwrap") {
		t.Error("dry run should show bwrap command")
	}
	if !strings.Contains(output, "--share-net") {
		t.Error("dry run should show args")
	}
}

// containsSequence checks if slice contains consecutive elements.
func containsSequence(slice []string, seq ...string) bool {
	if len(seq) == 0 {
		return true
	}
	for i := 0; i <= len(slice)-len(seq); i++ {
		match := true
		for j, s := range seq {
			if slice[i+j] != s {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
