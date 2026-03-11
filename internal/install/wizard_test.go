package install

import (
	"bytes"
	"strings"
	"testing"
)

func TestRun_NoAgents(t *testing.T) {
	// Mock no agents found
	oldDetect := detectAgentsFn
	detectAgentsFn = func() ([]Agent, error) { return []Agent{}, nil }
	t.Cleanup(func() { detectAgentsFn = oldDetect })

	input := bytes.NewBufferString("")
	output := &bytes.Buffer{}

	code := Run(input, output)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	if !strings.Contains(output.String(), "No supported agents found") {
		t.Fatalf("expected manual template in output, got: %s", output.String())
	}
}

func TestRun_SingleAgent(t *testing.T) {
	// Mock single agent, valid credentials, no write errors
	oldDetect := detectAgentsFn
	detectAgentsFn = func() ([]Agent, error) {
		return []Agent{{Name: "Claude Code", ConfigPath: "/tmp/test.json"}}, nil
	}
	t.Cleanup(func() { detectAgentsFn = oldDetect })

	oldValidate := validateCredentialsFn
	validateCredentialsFn = func(url, token string) error { return nil }
	t.Cleanup(func() { validateCredentialsFn = oldValidate })

	oldRead := readConfigFn
	readConfigFn = func(path string) (map[string]interface{}, error) {
		return make(map[string]interface{}), nil
	}
	t.Cleanup(func() { readConfigFn = oldRead })

	oldWrite := writeConfigFn
	writeConfigFn = func(path string, config map[string]interface{}) error {
		return nil
	}
	t.Cleanup(func() { writeConfigFn = oldWrite })

	input := strings.NewReader("https://bitbucket.example.com\nmytoken\n")
	output := &bytes.Buffer{}

	code := Run(input, output)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	if !strings.Contains(output.String(), "Saved to") {
		t.Fatalf("expected success message, got: %s", output.String())
	}
}

func TestRun_ValidationFails(t *testing.T) {
	oldDetect := detectAgentsFn
	detectAgentsFn = func() ([]Agent, error) {
		return []Agent{{Name: "Claude Code", ConfigPath: "/tmp/test.json"}}, nil
	}
	t.Cleanup(func() { detectAgentsFn = oldDetect })

	oldValidate := validateCredentialsFn
	validateCredentialsFn = func(url, token string) error {
		return &validationError{statusCode: 401}
	}
	t.Cleanup(func() { validateCredentialsFn = oldValidate })

	input := strings.NewReader("https://bitbucket.example.com\nbadtoken\n")
	output := &bytes.Buffer{}

	code := Run(input, output)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}

	if !strings.Contains(output.String(), "validation failed") {
		t.Fatalf("expected validation error message, got: %s", output.String())
	}
}
