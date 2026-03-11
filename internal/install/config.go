package install

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ReadConfig reads a JSON MCP config file.
// If the file doesn't exist, it returns an empty map.
// If the file contains invalid JSON, it returns an error.
func ReadConfig(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]interface{}), nil
		}
		return nil, fmt.Errorf("cannot read %s: %v", path, err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("cannot parse %s: %v", path, err)
	}

	return config, nil
}

// MergeEntry merges the bitbucket MCP server entry into the config.
// If mcpServers doesn't exist or isn't a map, it creates one.
// Sets the bitbucket entry with command and env vars.
func MergeEntry(config map[string]interface{}, url, token string) map[string]interface{} {
	var servers map[string]interface{}

	if s, ok := config["mcpServers"].(map[string]interface{}); ok {
		servers = s
	} else {
		servers = make(map[string]interface{})
		config["mcpServers"] = servers
	}

	servers["bitbucket"] = map[string]interface{}{
		"command": "bitbucket-mcp",
		"env": map[string]interface{}{
			"BITBUCKET_URL":   url,
			"BITBUCKET_TOKEN": token,
		},
	}

	return config
}

// WriteConfig marshals config to indented JSON and writes it to path.
// Creates parent directory if needed.
// File is written with mode 0600 to protect the token.
func WriteConfig(path string, config map[string]interface{}) error {
	// Create parent directory
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("cannot create directory %s: %v", dir, err)
	}

	// Marshal with 2-space indent
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("cannot marshal JSON: %v", err)
	}

	// Write with 0600 mode
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("cannot write %s: %v", path, err)
	}

	return nil
}
