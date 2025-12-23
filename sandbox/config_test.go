package sandbox

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfigPath(t *testing.T) {
	path := DefaultConfigPath()
	if path == "" {
		t.Skip("cannot get home dir")
	}

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".agent", "sandbox", "config.json")
	if path != expected {
		t.Errorf("got %q, want %q", path, expected)
	}
}

func TestLoadConfigFile_NotExist(t *testing.T) {
	cfg, err := LoadConfigFile("/nonexistent/path/config.json")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if cfg != nil {
		t.Error("expected nil config for non-existent file")
	}
}

func TestLoadConfigFile_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	content := `{
		"allowWrite": ["/custom/write"],
		"denyRead": ["~/.custom"],
		"cleanEnv": true,
		"envDenylist": ["SECRET_KEY"]
	}`

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfigFile(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.AllowWrite) != 1 || cfg.AllowWrite[0] != "/custom/write" {
		t.Errorf("AllowWrite = %v, want [/custom/write]", cfg.AllowWrite)
	}

	if len(cfg.DenyRead) != 1 || cfg.DenyRead[0] != "~/.custom" {
		t.Errorf("DenyRead = %v, want [~/.custom]", cfg.DenyRead)
	}

	if cfg.CleanEnv == nil || !*cfg.CleanEnv {
		t.Error("CleanEnv should be true")
	}

	if len(cfg.EnvDenylist) != 1 || cfg.EnvDenylist[0] != "SECRET_KEY" {
		t.Errorf("EnvDenylist = %v, want [SECRET_KEY]", cfg.EnvDenylist)
	}
}

func TestLoadConfigFile_Invalid(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	if err := os.WriteFile(configPath, []byte("invalid json"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadConfigFile(configPath)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestLoadConfigFile_EmptyArrays(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	content := `{
		"allowWrite": [],
		"denyRead": []
	}`

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfigFile(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Empty arrays should be loaded as empty (will use defaults when merged)
	if len(cfg.AllowWrite) != 0 {
		t.Errorf("AllowWrite = %v, want []", cfg.AllowWrite)
	}
	if len(cfg.DenyRead) != 0 {
		t.Errorf("DenyRead = %v, want []", cfg.DenyRead)
	}
}

func TestMergeConfig_NilFile(t *testing.T) {
	base := Config{
		AllowWrite: []string{"/base"},
		DenyRead:   []string{"~/.ssh"},
	}

	result := MergeConfig(base, nil)

	if len(result.AllowWrite) != 1 || result.AllowWrite[0] != "/base" {
		t.Errorf("AllowWrite = %v, want [/base]", result.AllowWrite)
	}
}

func TestMergeConfig_OverridesValues(t *testing.T) {
	base := Config{
		AllowWrite: []string{"/base", "/tmp"},
		DenyRead:   []string{"~/.ssh", "~/.aws"},
		CleanEnv:   false,
	}

	cleanEnv := true
	file := &FileConfig{
		AllowWrite: []string{"/custom"},
		DenyRead:   []string{"~/.custom"},
		CleanEnv:   &cleanEnv,
	}

	result := MergeConfig(base, file)

	// File values should replace base
	if len(result.AllowWrite) != 1 || result.AllowWrite[0] != "/custom" {
		t.Errorf("AllowWrite = %v, want [/custom]", result.AllowWrite)
	}

	if len(result.DenyRead) != 1 || result.DenyRead[0] != "~/.custom" {
		t.Errorf("DenyRead = %v, want [~/.custom]", result.DenyRead)
	}

	if !result.CleanEnv {
		t.Error("CleanEnv should be true")
	}
}

func TestMergeConfig_EmptyArraysUseDefaults(t *testing.T) {
	base := Config{
		AllowWrite: []string{"/base"},
		DenyRead:   []string{"~/.ssh"},
	}

	file := &FileConfig{
		AllowWrite: []string{}, // Empty = use defaults
		DenyRead:   []string{}, // Empty = use defaults
	}

	result := MergeConfig(base, file)

	// Empty arrays should NOT override - base values kept
	if len(result.AllowWrite) != 1 || result.AllowWrite[0] != "/base" {
		t.Errorf("AllowWrite = %v, want [/base]", result.AllowWrite)
	}

	if len(result.DenyRead) != 1 || result.DenyRead[0] != "~/.ssh" {
		t.Errorf("DenyRead = %v, want [~/.ssh]", result.DenyRead)
	}
}

func TestIsWildcard(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"*", true},
		{"/path", false},
		{"/*", false},
		{"", false},
	}

	for _, tt := range tests {
		result := IsWildcard(tt.path)
		if result != tt.expected {
			t.Errorf("IsWildcard(%q) = %v, want %v", tt.path, result, tt.expected)
		}
	}
}

func TestHasWildcard(t *testing.T) {
	tests := []struct {
		paths    []string
		expected bool
	}{
		{[]string{"*"}, true},
		{[]string{"/path", "*"}, true},
		{[]string{"/path", "/other"}, false},
		{[]string{}, false},
		{nil, false},
	}

	for _, tt := range tests {
		result := HasWildcard(tt.paths)
		if result != tt.expected {
			t.Errorf("HasWildcard(%v) = %v, want %v", tt.paths, result, tt.expected)
		}
	}
}

func TestLoadConfigFile_Wildcard(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	content := `{
		"allowWrite": ["*"],
		"denyRead": ["*"]
	}`

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfigFile(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !HasWildcard(cfg.AllowWrite) {
		t.Error("AllowWrite should contain wildcard")
	}

	if !HasWildcard(cfg.DenyRead) {
		t.Error("DenyRead should contain wildcard")
	}
}

func TestDefaultConfigWithPath_Empty(t *testing.T) {
	cfg := DefaultConfigWithPath("")

	// Should return hardcoded defaults
	if cfg.Workdir == "" {
		t.Error("Workdir should not be empty")
	}

	if len(cfg.AllowWrite) == 0 {
		t.Error("AllowWrite should have defaults")
	}

	if len(cfg.DenyRead) == 0 {
		t.Error("DenyRead should have defaults")
	}
}

func TestDefaultConfigWithPath_CustomFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	content := `{
		"denyRead": ["~/.custom-secret"]
	}`

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := DefaultConfigWithPath(configPath)

	// DenyRead should be from file
	if len(cfg.DenyRead) != 1 || cfg.DenyRead[0] != "~/.custom-secret" {
		t.Errorf("DenyRead = %v, want [~/.custom-secret]", cfg.DenyRead)
	}

	// AllowWrite should still be defaults (not specified in file)
	if len(cfg.AllowWrite) == 0 {
		t.Error("AllowWrite should have defaults")
	}
}
