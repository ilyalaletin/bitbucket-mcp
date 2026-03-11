package install

import (
	"os"
	"path/filepath"
)

type Agent struct {
	Name       string
	ConfigPath string
}

var userHomeDirFn = os.UserHomeDir // testable hook

func DetectAgents() ([]Agent, error) {
	homeDir, err := userHomeDirFn()
	if err != nil {
		return nil, err
	}

	var agents []Agent

	// Check Claude Code
	claudePath := filepath.Join(homeDir, ".claude.json")
	if _, err := os.Stat(claudePath); err == nil {
		agents = append(agents, Agent{
			Name:       "Claude Code",
			ConfigPath: claudePath,
		})
	}

	// Check Cursor
	cursorPath := filepath.Join(homeDir, ".cursor", "mcp.json")
	if _, err := os.Stat(cursorPath); err == nil {
		agents = append(agents, Agent{
			Name:       "Cursor",
			ConfigPath: cursorPath,
		})
	}

	return agents, nil
}
