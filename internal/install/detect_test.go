package install

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectAgents(t *testing.T) {
	// Create temp home dir
	tmpHome := t.TempDir()

	// Create Claude Code config
	claudePath := filepath.Join(tmpHome, ".claude.json")
	os.WriteFile(claudePath, []byte(`{"mcpServers":{}}`), 0644)

	// Mock os.UserHomeDir
	oldUserHomeDir := userHomeDirFn
	userHomeDirFn = func() (string, error) { return tmpHome, nil }
	t.Cleanup(func() { userHomeDirFn = oldUserHomeDir })

	agents, err := DetectAgents()
	if err != nil {
		t.Fatalf("DetectAgents failed: %v", err)
	}

	if len(agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(agents))
	}
	if agents[0].Name != "Claude Code" {
		t.Fatalf("expected 'Claude Code', got %q", agents[0].Name)
	}
	if agents[0].ConfigPath != claudePath {
		t.Fatalf("expected %q, got %q", claudePath, agents[0].ConfigPath)
	}
}

func TestDetectAgents_NoAgents(t *testing.T) {
	tmpHome := t.TempDir()
	oldUserHomeDir := userHomeDirFn
	userHomeDirFn = func() (string, error) { return tmpHome, nil }
	t.Cleanup(func() { userHomeDirFn = oldUserHomeDir })

	agents, err := DetectAgents()
	if err != nil {
		t.Fatalf("DetectAgents failed: %v", err)
	}
	if len(agents) != 0 {
		t.Fatalf("expected 0 agents, got %d", len(agents))
	}
}

func TestDetectAgents_HomeDirError(t *testing.T) {
	oldUserHomeDir := userHomeDirFn
	userHomeDirFn = func() (string, error) { return "", os.ErrNotExist }
	t.Cleanup(func() { userHomeDirFn = oldUserHomeDir })

	_, err := DetectAgents()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
