package install

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestValidateCredentials_Success(t *testing.T) {
	// Mock Bitbucket API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/1.0/application-properties" {
			t.Errorf("expected /rest/api/1.0/application-properties, got %s", r.URL.Path)
		}
		auth := r.Header.Get("Authorization")
		if auth != "Bearer validtoken" {
			t.Errorf("expected 'Bearer validtoken', got %q", auth)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"version":"6.0.0"}`))
	}))
	defer server.Close()

	err := ValidateCredentials(server.URL, "validtoken")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidateCredentials_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	err := ValidateCredentials(server.URL, "badtoken")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != fmt.Sprintf("HTTP %d", http.StatusUnauthorized) {
		t.Fatalf("expected 'HTTP 401', got %q", err.Error())
	}
}

func TestValidateCredentials_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow server (will timeout)
		// Sleep longer than the 10-second timeout
		time.Sleep(15 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := ValidateCredentials(server.URL, "token")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
