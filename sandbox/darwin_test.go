//go:build darwin

package sandbox

import (
	"strings"
	"testing"
)

func TestGenerateProfile(t *testing.T) {
	cfg := Config{
		Workdir:    "/home/user/project",
		AllowWrite: []string{"/home/user/project", "/tmp"},
		DenyRead:   []string{"/home/user/.ssh"},
	}
	s := &darwinSandbox{cfg: cfg}
	profile := s.generateProfile()

	checks := []string{
		"(version 1)",
		"(allow default)",
		"(allow network*)",
		"(deny file-write*)",
		`(allow file-write* (subpath "/home/user/project"))`,
		`(allow file-write* (subpath "/tmp"))`,
		`(deny file-read* (subpath "/home/user/.ssh"))`,
	}

	for _, check := range checks {
		if !strings.Contains(profile, check) {
			t.Errorf("profile should contain %q\nGot:\n%s", check, profile)
		}
	}
}

func TestGenerateProfile_DenyReadTakesPrecedence(t *testing.T) {
	cfg := Config{
		Workdir:    "/tmp",
		AllowWrite: []string{"/home/user/.ssh"}, // Trying to allow write to sensitive dir
		DenyRead:   []string{"/home/user/.ssh"}, // But DenyRead should win
	}
	s := &darwinSandbox{cfg: cfg}
	profile := s.generateProfile()

	// Should NOT have allow file-write for .ssh
	if strings.Contains(profile, `(allow file-write* (subpath "/home/user/.ssh"))`) {
		t.Error("should not allow write to DenyRead path")
	}

	// Should have deny file-read for .ssh
	if !strings.Contains(profile, `(deny file-read* (subpath "/home/user/.ssh"))`) {
		t.Error("should deny read from DenyRead path")
	}
}

func TestDryRunOutput_Darwin(t *testing.T) {
	cfg := Config{
		Workdir:    "/tmp",
		AllowWrite: []string{"/tmp"},
		DryRun:     true,
	}
	s := &darwinSandbox{cfg: cfg}
	s.profile = s.generateProfile()

	output := s.dryRunOutput("echo hello")

	if !strings.Contains(output, "sandbox-exec") {
		t.Error("dry run should show sandbox-exec command")
	}
	if !strings.Contains(output, "echo hello") {
		t.Error("dry run should show the command")
	}
}
