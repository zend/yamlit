package step

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zend/yamlit/pkg/variable"
)

func TestExecuteGET(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok"}`)
	}))
	defer server.Close()

	exec := NewExecutor()
	vars := variable.NewPool()

	step := Step{
		Name:   "test-get",
		Method: "GET",
		URL:    server.URL,
	}

	result := exec.Execute(step, vars)
	if result.Error != nil {
		t.Fatalf("Execute failed: %v", result.Error)
	}
	if result.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", result.StatusCode)
	}
	if result.Body != `{"status":"ok"}` {
		t.Errorf("Body = %q, want %q", result.Body, `{"status":"ok"}`)
	}
}

func TestExecutePOSTWithJSONBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		ct := r.Header.Get("Content-Type")
		if ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id":1}`)
	}))
	defer server.Close()

	exec := NewExecutor()
	vars := variable.NewPool()

	step := Step{
		Name:   "test-post",
		Method: "POST",
		URL:    server.URL,
		Body: &Body{
			Type:    "json",
			Content: `{"name":"test"}`,
		},
	}

	result := exec.Execute(step, vars)
	if result.Error != nil {
		t.Fatalf("Execute failed: %v", result.Error)
	}
	if result.StatusCode != 201 {
		t.Errorf("StatusCode = %d, want 201", result.StatusCode)
	}
}

func TestExecuteWithParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("key") != "value" {
			t.Errorf("expected key=value, got %s", r.URL.RawQuery)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	exec := NewExecutor()
	vars := variable.NewPool()

	step := Step{
		Name:   "test-params",
		Method: "GET",
		URL:    server.URL,
		Params: map[string]string{"key": "value"},
	}

	result := exec.Execute(step, vars)
	if result.Error != nil {
		t.Fatalf("Execute failed: %v", result.Error)
	}
}

func TestExecuteWithVariableSubstitution(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer mytoken" {
			t.Errorf("Authorization = %q, want Bearer mytoken", r.Header.Get("Authorization"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	exec := NewExecutor()
	vars := variable.NewPool()
	vars.Set("token", "mytoken")

	step := Step{
		Name:   "test-vars",
		Method: "GET",
		URL:    server.URL,
		Headers: map[string]string{
			"Authorization": "Bearer ${token}",
		},
	}

	result := exec.Execute(step, vars)
	if result.Error != nil {
		t.Fatalf("Execute failed: %v", result.Error)
	}
}

func TestExecuteFORMBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ct := r.Header.Get("Content-Type")
		if ct != "application/x-www-form-urlencoded" {
			t.Errorf("Content-Type = %q, want application/x-www-form-urlencoded", ct)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	exec := NewExecutor()
	vars := variable.NewPool()

	step := Step{
		Name:   "test-form",
		Method: "POST",
		URL:    server.URL,
		Body: &Body{
			Type:    "form",
			Content: "user=test&pass=123",
		},
	}

	result := exec.Execute(step, vars)
	if result.Error != nil {
		t.Fatalf("Execute failed: %v", result.Error)
	}
}

func TestExecuteTextBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ct := r.Header.Get("Content-Type")
		if ct != "text/plain" {
			t.Errorf("Content-Type = %q, want text/plain", ct)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	exec := NewExecutor()
	vars := variable.NewPool()

	step := Step{
		Name:   "test-text",
		Method: "POST",
		URL:    server.URL,
		Body: &Body{
			Type:    "text",
			Content: "plain text body",
		},
	}

	result := exec.Execute(step, vars)
	if result.Error != nil {
		t.Fatalf("Execute failed: %v", result.Error)
	}
}

func TestExecuteTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Never respond — will timeout
		select {}
	}))
	defer server.Close()

	exec := NewExecutor()

	step := Step{
		Name:    "test-timeout",
		Method:  "GET",
		URL:     server.URL,
		Timeout: 1, // 1 nanosecond — will timeout
	}

	vars := variable.NewPool()
	result := exec.Execute(step, vars)
	if result.Error == nil {
		t.Error("expected timeout error, got nil")
	}
}

func TestExecuteWithCustomHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Custom") != "myvalue" {
			t.Errorf("X-Custom = %q, want myvalue", r.Header.Get("X-Custom"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	exec := NewExecutor()
	vars := variable.NewPool()

	step := Step{
		Name:   "test-headers",
		Method: "GET",
		URL:    server.URL,
		Headers: map[string]string{
			"X-Custom": "myvalue",
		},
	}

	result := exec.Execute(step, vars)
	if result.Error != nil {
		t.Fatalf("Execute failed: %v", result.Error)
	}
}

func TestExecuteWithExistingQueryParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("existing") != "true" {
			t.Errorf("expected existing=true, got %s", r.URL.RawQuery)
		}
		if r.URL.Query().Get("key") != "value" {
			t.Errorf("expected key=value, got %s", r.URL.RawQuery)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	exec := NewExecutor()
	vars := variable.NewPool()

	step := Step{
		Name:   "test-existing-params",
		Method: "GET",
		URL:    server.URL + "?existing=true",
		Params: map[string]string{"key": "value"},
	}

	result := exec.Execute(step, vars)
	if result.Error != nil {
		t.Fatalf("Execute failed: %v", result.Error)
	}
}
