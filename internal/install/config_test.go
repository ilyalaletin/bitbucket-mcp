package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestReadConfig_FileNotFound(t *testing.T) {
	config, err := ReadConfig("/nonexistent/path.json")
	if err != nil {
		t.Fatalf("expected no error for missing file, got %v", err)
	}
	if len(config) != 0 {
		t.Fatalf("expected empty config, got %v", config)
	}
}

func TestReadConfig_ValidJSON(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "config.json")
	data := `{"mcpServers":{"other":{"command":"other"}}}`
	os.WriteFile(tmpFile, []byte(data), 0644)

	config, err := ReadConfig(tmpFile)
	if err != nil {
		t.Fatalf("ReadConfig failed: %v", err)
	}

	servers, ok := config["mcpServers"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected mcpServers to be map, got %T", config["mcpServers"])
	}
	if _, ok := servers["other"]; !ok {
		t.Fatal("expected 'other' server in config")
	}
}

func TestReadConfig_InvalidJSON(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "config.json")
	os.WriteFile(tmpFile, []byte("{invalid json}"), 0644)

	_, err := ReadConfig(tmpFile)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestMergeEntry(t *testing.T) {
	config := make(map[string]interface{})
	result := MergeEntry(config, "https://bitbucket.example.com", "mytoken")

	servers, ok := result["mcpServers"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected mcpServers in result")
	}

	bitbucket, ok := servers["bitbucket"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected bitbucket entry")
	}

	if bitbucket["command"] != "bitbucket-mcp" {
		t.Fatalf("expected command='bitbucket-mcp', got %v", bitbucket["command"])
	}

	env, ok := bitbucket["env"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected env in bitbucket entry")
	}

	if env["BITBUCKET_URL"] != "https://bitbucket.example.com" {
		t.Fatalf("expected BITBUCKET_URL to be set")
	}
	if env["BITBUCKET_TOKEN"] != "mytoken" {
		t.Fatalf("expected BITBUCKET_TOKEN to be set")
	}
}

func TestWriteConfig(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "newconfig.json")
	config := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"bitbucket": map[string]interface{}{
				"command": "bitbucket-mcp",
			},
		},
	}

	err := WriteConfig(tmpFile, config)
	if err != nil {
		t.Fatalf("WriteConfig failed: %v", err)
	}

	// Verify file was created with correct permissions
	info, err := os.Stat(tmpFile)
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}

	mode := info.Mode()
	if (mode & 0077) != 0 {
		t.Fatalf("expected mode 0600, got %o", mode)
	}

	// Verify content is valid JSON
	data, _ := os.ReadFile(tmpFile)
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("invalid JSON written: %v", err)
	}
}
