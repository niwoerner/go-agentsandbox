//go:build integration

package sandbox

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWriteToWorkdirAllowed(t *testing.T) {
	dir := t.TempDir()
	sb, err := New(Config{
		Workdir:    dir,
		AllowWrite: []string{dir},
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	testFile := filepath.Join(dir, "testfile")
	_, code, err := sb.Run(context.Background(), "touch "+testFile)
	if err != nil && code != 0 {
		t.Fatalf("Run() error: %v", err)
	}
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}

	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("file should have been created")
	}
}

func TestWriteOutsideWorkdirDenied(t *testing.T) {
	sb, err := New(Config{
		Workdir:    t.TempDir(),
		AllowWrite: []string{t.TempDir()},
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	_, code, _ := sb.Run(context.Background(), "touch /etc/testfile_sandbox_test")
	if code == 0 {
		t.Error("write outside workdir should fail")
		os.Remove("/etc/testfile_sandbox_test") // cleanup if somehow succeeded
	}
}

func TestReadProtectedDirDenied(t *testing.T) {
	dir := t.TempDir()
	sensitiveDir := filepath.Join(dir, "sensitive")
	if err := os.MkdirAll(sensitiveDir, 0755); err != nil {
		t.Fatal(err)
	}

	secretFile := filepath.Join(sensitiveDir, "secret")
	if err := os.WriteFile(secretFile, []byte("supersecret"), 0644); err != nil {
		t.Fatal(err)
	}

	sb, err := New(Config{
		Workdir:    dir,
		AllowWrite: []string{dir},
		DenyRead:   []string{sensitiveDir},
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	output, code, _ := sb.Run(context.Background(), "cat "+secretFile)
	if code == 0 {
		t.Error("read from DenyRead path should fail")
	}
	if strings.Contains(string(output), "supersecret") {
		t.Error("should not be able to read secret content")
	}
}

func TestNetworkAllowed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

	sb, err := New(Config{
		Workdir:    t.TempDir(),
		AllowWrite: []string{t.TempDir()},
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	// Try multiple network commands - at least one should work
	// This handles cases where curl isn't installed or network is flaky
	commands := []string{
		"ping -c 1 -W 2 127.0.0.1", // Ping localhost (always works if network stack available)
		"curl -s --max-time 5 -o /dev/null https://example.com",
		"wget -q --timeout=5 -O /dev/null https://example.com",
	}

	for _, cmd := range commands {
		_, code, _ := sb.Run(context.Background(), cmd)
		if code == 0 {
			return // Success - network is allowed
		}
	}

	// If we get here, try a simple socket test as last resort
	output, code, _ := sb.Run(context.Background(), "python3 -c \"import socket; socket.create_connection(('8.8.8.8', 53), timeout=2)\" 2>&1 || python -c \"import socket; socket.create_connection(('8.8.8.8', 53), timeout=2)\" 2>&1 || echo 'no python'")

	// Skip if no network tools available rather than fail
	if strings.Contains(string(output), "no python") && code != 0 {
		t.Skip("no network tools available to test network access")
	}

	if code != 0 {
		t.Errorf("network access should be allowed, got exit code %d, output: %s", code, output)
	}
}

func TestExitCodePreserved(t *testing.T) {
	sb, err := New(Config{
		Workdir:    t.TempDir(),
		AllowWrite: []string{t.TempDir()},
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	_, code, _ := sb.Run(context.Background(), "exit 42")
	if code != 42 {
		t.Errorf("expected exit code 42, got %d", code)
	}
}

func TestEnvDenylist(t *testing.T) {
	os.Setenv("TEST_SECRET_TOKEN", "supersecret123")
	defer os.Unsetenv("TEST_SECRET_TOKEN")

	sb, err := New(Config{
		Workdir:     t.TempDir(),
		AllowWrite:  []string{t.TempDir()},
		EnvDenylist: []string{"TEST_SECRET_TOKEN"},
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	output, _, _ := sb.Run(context.Background(), "env")
	if strings.Contains(string(output), "TEST_SECRET_TOKEN") {
		t.Error("denylisted env var should not be visible")
	}
	if strings.Contains(string(output), "supersecret123") {
		t.Error("denylisted env var value should not be visible")
	}
}

func TestCleanEnv(t *testing.T) {
	os.Setenv("TEST_RANDOM_VAR", "randomvalue")
	defer os.Unsetenv("TEST_RANDOM_VAR")

	sb, err := New(Config{
		Workdir:    t.TempDir(),
		AllowWrite: []string{t.TempDir()},
		CleanEnv:   true,
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	output, _, _ := sb.Run(context.Background(), "env")
	if strings.Contains(string(output), "TEST_RANDOM_VAR") {
		t.Error("random env var should not be visible with CleanEnv")
	}
	if !strings.Contains(string(output), "PATH=") {
		t.Error("PATH should still be present with CleanEnv")
	}
}

func TestContextCancellation(t *testing.T) {
	sb, err := New(Config{
		Workdir:    t.TempDir(),
		AllowWrite: []string{t.TempDir()},
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, _, err = sb.Run(ctx, "sleep 10")
	elapsed := time.Since(start)

	if err == nil {
		t.Error("should error on timeout")
	}

	if elapsed > 2*time.Second {
		t.Errorf("should have been cancelled quickly, took %v", elapsed)
	}

	if !errors.Is(err, context.DeadlineExceeded) && !strings.Contains(err.Error(), "killed") && !strings.Contains(err.Error(), "signal") {
		t.Logf("error type: %T, value: %v", err, err)
	}
}

func TestStdinPiping(t *testing.T) {
	sb, err := New(Config{
		Workdir:    t.TempDir(),
		AllowWrite: []string{t.TempDir()},
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	input := strings.NewReader("hello from stdin")
	output, code, err := sb.RunWithStdin(context.Background(), "cat", input)
	if err != nil && code != 0 {
		t.Fatalf("RunWithStdin() error: %v", err)
	}

	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}

	if string(output) != "hello from stdin" {
		t.Errorf("expected 'hello from stdin', got %q", string(output))
	}
}

func TestDryRun(t *testing.T) {
	sb, err := New(Config{
		Workdir:    t.TempDir(),
		AllowWrite: []string{t.TempDir()},
		DryRun:     true,
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	output, code, err := sb.Run(context.Background(), "echo test")
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	if code != 0 {
		t.Errorf("dry run should return exit code 0, got %d", code)
	}

	// Output should contain the command, not execute it
	if strings.Contains(string(output), "test\n") && !strings.Contains(string(output), "echo") {
		t.Error("dry run should show command, not output")
	}
}
