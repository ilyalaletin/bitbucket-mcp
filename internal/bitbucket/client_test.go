package bitbucket

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient("https://bitbucket.example.com", "test-token")

	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.baseURL != "https://bitbucket.example.com" {
		t.Errorf("expected baseURL 'https://bitbucket.example.com', got %q", client.baseURL)
	}
}

func TestNewClient_TrimsTrailingSlash(t *testing.T) {
	client := NewClient("https://bitbucket.example.com/", "test-token")

	if client.baseURL != "https://bitbucket.example.com" {
		t.Errorf("expected trailing slash trimmed, got %q", client.baseURL)
	}
}

func TestDo_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Bearer auth header")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	resp, err := client.do(context.Background(), "GET", "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestDo_Retry_On5xx(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	resp, err := client.do(context.Background(), "GET", "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 after retries, got %d", resp.StatusCode)
	}
	if atomic.LoadInt32(&attempts) != 3 {
		t.Errorf("expected 3 attempts, got %d", atomic.LoadInt32(&attempts))
	}
}

func TestDo_NoRetry_On4xx(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	resp, err := client.do(context.Background(), "GET", "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if atomic.LoadInt32(&attempts) != 1 {
		t.Errorf("expected 1 attempt (no retry on 4xx), got %d", atomic.LoadInt32(&attempts))
	}
}

func TestDo_Retry_On429(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n < 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	resp, err := client.do(context.Background(), "GET", "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestGetJSON_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"name": "test"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token")

	var result struct {
		Name string `json:"name"`
	}
	err := client.getJSON(context.Background(), "/test", nil, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "test" {
		t.Errorf("expected name 'test', got %q", result.Name)
	}
}

func TestGetJSON_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"errors": [{"message": "not found"}]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token")

	var result map[string]any
	err := client.getJSON(context.Background(), "/test", nil, &result)
	if err == nil {
		t.Fatal("expected error for 404 response")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("expected status 404, got %d", apiErr.StatusCode)
	}
}
