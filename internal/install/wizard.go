package install

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// testable hooks
var (
	detectAgentsFn        = DetectAgents
	validateCredentialsFn = ValidateCredentials
	readConfigFn          = ReadConfig
	writeConfigFn         = WriteConfig
)

// validationError is used in tests to represent validation errors.
type validationError struct {
	statusCode int
}

func (e *validationError) Error() string {
	return fmt.Sprintf("HTTP %d", e.statusCode)
}

// Run orchestrates the full wizard flow.
// Reads input line-by-line from `in`, writes all output to `out`.
// Returns 0 on success, 1 on error.
func Run(in io.Reader, out io.Writer) int {
	printf := func(format string, args ...interface{}) {
		fmt.Fprintf(out, format, args...)
	}

	reader := bufio.NewReader(in)
	readLine := func() (string, error) {
		line, err := reader.ReadString('\n')
		return strings.TrimRight(line, "\n"), err
	}

	// Step 1: Detect agents
	agents, err := detectAgentsFn()
	if err != nil {
		printf("Error: cannot determine home directory: %v\n", err)
		return 1
	}

	// Step 2: No agents found
	if len(agents) == 0 {
		printf("No supported agents found (Claude Code, Cursor).\n\n")
		printf("To configure manually, add the following to your agent's MCP config:\n\n")
		printf("{\n  \"mcpServers\": {\n    \"bitbucket\": {\n      \"command\": \"bitbucket-mcp\",\n      \"env\": {\n        \"BITBUCKET_URL\": \"<YOUR_BITBUCKET_URL>\",\n        \"BITBUCKET_TOKEN\": \"<YOUR_TOKEN>\"\n      }\n    }\n  }\n}\n")
		return 0
	}

	// Step 3: Agent selection
	var selectedAgents []Agent
	if len(agents) == 1 {
		selectedAgents = agents
	} else {
		for {
			for i, agent := range agents {
				printf("[%d] %s (%s)\n", i+1, agent.Name, agent.ConfigPath)
			}
			printf("Install into (1, 2, or \"all\") [all]: ")

			input, err := readLine()
			if err != nil && err != io.EOF {
				printf("Error: %v\n", err)
				return 1
			}
			if err == io.EOF {
				printf("Error: unexpected end of input\n")
				return 1
			}

			input = strings.ToLower(strings.TrimSpace(input))
			if input == "" || input == "all" {
				selectedAgents = agents
				break
			}

			// Try parsing as number
			var idx int
			if _, err := fmt.Sscanf(input, "%d", &idx); err == nil && idx > 0 && idx <= len(agents) {
				selectedAgents = []Agent{agents[idx-1]}
				break
			}

			printf("Invalid selection. Try again.\n")
		}
	}

	// Step 4: URL prompt
	var url string
	for {
		printf("Bitbucket Server URL: ")
		input, err := readLine()
		if err == io.EOF {
			printf("Error: unexpected end of input\n")
			return 1
		}
		if err != nil {
			printf("Error: %v\n", err)
			return 1
		}

		url = strings.TrimRight(strings.TrimSpace(input), "/")
		if url == "" {
			printf("URL cannot be empty. Try again.\n")
			continue
		}

		// Step 5: URL pre-validation
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			printf("Error: URL must start with http:// or https://\n")
			continue
		}

		break
	}

	// Step 6: Token instructions
	printf("\nTo generate a Personal Access Token:\n")
	printf("  1. Go to %s/plugins/servlet/access-tokens/manage\n", url)
	printf("  2. Click \"Create token\"\n")
	printf("  3. Give it a name, select \"Read\" scope\n")
	printf("  4. Copy the generated token\n\n")

	// Step 7: Token prompt
	var token string
	for {
		printf("Token: ")
		input, err := readLine()
		if err == io.EOF {
			printf("Error: unexpected end of input\n")
			return 1
		}
		if err != nil {
			printf("Error: %v\n", err)
			return 1
		}

		token = strings.TrimSpace(input)
		if token == "" {
			printf("Token cannot be empty. Try again.\n")
			continue
		}
		break
	}

	// Step 8: Validation
	if err := validateCredentialsFn(url, token); err != nil {
		printf("Error: validation failed (HTTP %v)\n", extractStatus(err))
		return 1
	}

	// Step 9: Write config
	var written []string
	for _, agent := range selectedAgents {
		config, err := readConfigFn(agent.ConfigPath)
		if err != nil {
			printf("Error writing %s: %v\nNote: config was already written to: %v\n", agent.ConfigPath, err, written)
			return 1
		}

		config = MergeEntry(config, url, token)

		if err := writeConfigFn(agent.ConfigPath, config); err != nil {
			printf("Error writing %s: %v\nNote: config was already written to: %v\n", agent.ConfigPath, err, written)
			return 1
		}

		written = append(written, agent.ConfigPath)
	}

	// Step 10: Security reminder
	printf("Note: your token is stored in plaintext in the config file.\n\n")

	// Step 11: Done message
	printf("Saved to:\n")
	for _, path := range written {
		printf("  - %s\n", path)
	}
	printf("\nRestart Claude Code / Cursor to activate the bitbucket-mcp server.\n")

	return 0
}

// extractStatus extracts the HTTP status from a validation error.
func extractStatus(err error) string {
	return err.Error() // Already formatted as "HTTP NNN" from ValidateCredentials
}
