package sandbox

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExpandPath_Tilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot get home dir")
	}

	result, err := expandPath("~/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := filepath.Join(home, "test")
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestExpandPath_Relative(t *testing.T) {
	cwd, _ := os.Getwd()

	result, err := expandPath("./relative")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := filepath.Join(cwd, "relative")
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestExpandPath_Absolute(t *testing.T) {
	result, err := expandPath("/absolute/path")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "/absolute/path" {
		t.Errorf("got %q, want %q", result, "/absolute/path")
	}
}

func TestBuildEnv_CleanEnv(t *testing.T) {
	// Set test env vars
	os.Setenv("TEST_CUSTOM_VAR", "custom_value")
	os.Setenv("TEST_SECRET_KEY", "secret")
	defer os.Unsetenv("TEST_CUSTOM_VAR")
	defer os.Unsetenv("TEST_SECRET_KEY")

	cfg := Config{
		CleanEnv:     true,
		EnvAllowlist: []string{"TEST_CUSTOM_VAR"},
	}

	env := buildEnv(cfg)

	// Should contain allowlisted var
	found := false
	for _, e := range env {
		if e == "TEST_CUSTOM_VAR=custom_value" {
			found = true
		}
		if strings.HasPrefix(e, "TEST_SECRET_KEY=") {
			t.Error("should not contain TEST_SECRET_KEY")
		}
	}
	if !found {
		t.Error("should contain TEST_CUSTOM_VAR")
	}

	// Should contain PATH (essential var)
	foundPath := false
	for _, e := range env {
		if strings.HasPrefix(e, "PATH=") {
			foundPath = true
			break
		}
	}
	if !foundPath {
		t.Error("should contain PATH")
	}
}

func TestBuildEnv_Denylist(t *testing.T) {
	os.Setenv("TEST_AWS_SECRET", "secret123")
	os.Setenv("TEST_NORMAL_VAR", "normal")
	defer os.Unsetenv("TEST_AWS_SECRET")
	defer os.Unsetenv("TEST_NORMAL_VAR")

	cfg := Config{
		CleanEnv:    false,
		EnvDenylist: []string{"TEST_AWS_SECRET"},
	}

	env := buildEnv(cfg)

	for _, e := range env {
		if strings.HasPrefix(e, "TEST_AWS_SECRET=") {
			t.Error("should not contain denylisted var")
		}
	}

	foundNormal := false
	for _, e := range env {
		if e == "TEST_NORMAL_VAR=normal" {
			foundNormal = true
			break
		}
	}
	if !foundNormal {
		t.Error("should contain normal var")
	}
}

func TestValidatePaths_WorkdirMissing_LogsWarning(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	cfg := Config{Workdir: "/nonexistent/test/path/12345"}
	validatePaths(&cfg)

	logOutput := buf.String()
	if !strings.Contains(logOutput, "warning") {
		t.Error("should log warning")
	}
	if !strings.Contains(logOutput, "/nonexistent/test/path/12345") {
		t.Error("should contain path in warning")
	}
}

func TestValidatePaths_WorkdirExists_NoWarning(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	cfg := Config{Workdir: os.TempDir()}
	validatePaths(&cfg)

	if buf.Len() > 0 {
		t.Errorf("should not log anything, got: %s", buf.String())
	}
}

func TestPathInDenyRead(t *testing.T) {
	denyRead := []string{"/home/user/.ssh", "/home/user/.aws"}

	tests := []struct {
		path     string
		expected bool
	}{
		{"/home/user/.ssh", true},
		{"/home/user/.ssh/id_rsa", true},
		{"/home/user/.aws", true},
		{"/home/user/.aws/credentials", true},
		{"/home/user/project", false},
		{"/home/user/.sshkeys", false}, // Different dir, not subpath
	}

	for _, tt := range tests {
		result := pathInDenyRead(tt.path, denyRead)
		if result != tt.expected {
			t.Errorf("pathInDenyRead(%q) = %v, want %v", tt.path, result, tt.expected)
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Workdir == "" {
		t.Error("Workdir should not be empty")
	}

	if len(cfg.AllowWrite) == 0 {
		t.Error("AllowWrite should have defaults")
	}

	if len(cfg.DenyRead) == 0 {
		t.Error("DenyRead should have defaults")
	}

	if cfg.CleanEnv {
		t.Error("CleanEnv should be false by default")
	}

	// EnvDenylist should be empty by default (user configures as needed)
	if len(cfg.EnvDenylist) != 0 {
		t.Error("EnvDenylist should be empty by default")
	}
}
