package install

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

// ValidateCredentials validates Bitbucket URL and token by making a test API call.
// Makes GET {baseURL}/rest/api/1.0/application-properties with Authorization: Bearer {token}.
// Returns nil on 2xx (response body discarded).
// Returns fmt.Errorf("HTTP %d", statusCode) on non-2xx.
// Has a 10-second timeout.
func ValidateCredentials(baseURL, token string) error {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", baseURL+"/rest/api/1.0/application-properties", nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Discard response body
	io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return nil
}
