package runner

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mike/yaml-testing/pkg/step"
)

func TestRunSingleStepPass(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok"}`)
	}))
	defer server.Close()

	steps := []step.Step{
		{
			Name:   "test",
			Method: "GET",
			URL:    server.URL,
			Asserts: []step.Assertion{
				{Type: "status_code", Expect: "200"},
			},
		},
	}

	runner := NewRunner(steps, false)
	report := runner.Run()

	if report.Total != 1 {
		t.Errorf("Total = %d, want 1", report.Total)
	}
	if report.Passed != 1 {
		t.Errorf("Passed = %d, want 1", report.Passed)
	}
	if report.Failed != 0 {
		t.Errorf("Failed = %d, want 0", report.Failed)
	}
	if len(report.Steps) != 1 {
		t.Fatalf("len(Steps) = %d, want 1", len(report.Steps))
	}
	if report.Steps[0].Error != nil {
		t.Errorf("Step error: %v", report.Steps[0].Error)
	}
}

func TestRunAssertionFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error":"server error"}`)
	}))
	defer server.Close()

	steps := []step.Step{
		{
			Name:   "test",
			Method: "GET",
			URL:    server.URL,
			Asserts: []step.Assertion{
				{Type: "status_code", Expect: "200"},
			},
		},
	}

	runner := NewRunner(steps, false)
	report := runner.Run()

	if report.Passed != 0 {
		t.Errorf("Passed = %d, want 0", report.Passed)
	}
	if report.Failed != 1 {
		t.Errorf("Failed = %d, want 1", report.Failed)
	}
	if report.Steps[0].Error == nil {
		t.Error("expected error, got nil")
	}
}

func TestRunExtractAndSubstitute(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			// First call: login
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"token":"abc123"}`)
		} else {
			// Second call: should have token in header
			if r.Header.Get("Authorization") != "Bearer abc123" {
				t.Errorf("Authorization = %q, want Bearer abc123", r.Header.Get("Authorization"))
			}
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"status":"ok"}`)
		}
	}))
	defer server.Close()

	steps := []step.Step{
		{
			Name:   "login",
			Method: "GET",
			URL:    server.URL,
			Asserts: []step.Assertion{
				{Type: "status_code", Expect: "200"},
			},
			Extract: []step.ExtractItem{
				{Source: "body", Path: "token", VarName: "auth_token"},
			},
		},
		{
			Name:   "get_data",
			Method: "GET",
			URL:    server.URL,
			Headers: map[string]string{
				"Authorization": "Bearer ${auth_token}",
			},
			Asserts: []step.Assertion{
				{Type: "status_code", Expect: "200"},
			},
		},
	}

	runner := NewRunner(steps, false)
	report := runner.Run()

	if report.Total != 2 {
		t.Errorf("Total = %d, want 2", report.Total)
	}
	if report.Passed != 2 {
		t.Errorf("Passed = %d, want 2", report.Passed)
	}
	if report.Failed != 0 {
		t.Errorf("Failed = %d, want 0", report.Failed)
	}
}

func TestRunOnFailureStop(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	steps := []step.Step{
		{
			Name:       "first",
			Method:     "GET",
			URL:        server.URL,
			OnFailure:  "stop",
			Asserts:    []step.Assertion{{Type: "status_code", Expect: "200"}},
		},
		{
			Name:      "second",
			Method:    "GET",
			URL:       server.URL,
			OnFailure: "stop",
			Asserts:   []step.Assertion{{Type: "status_code", Expect: "200"}},
		},
	}

	runner := NewRunner(steps, false)
	report := runner.Run()

	if len(report.Steps) != 1 {
		t.Errorf("expected 1 step executed (stop), got %d", len(report.Steps))
	}
}

func TestRunOnFailureContinue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	steps := []step.Step{
		{
			Name:      "first",
			Method:    "GET",
			URL:       server.URL,
			OnFailure: "continue",
			Asserts:   []step.Assertion{{Type: "status_code", Expect: "200"}},
		},
		{
			Name:      "second",
			Method:    "GET",
			URL:       server.URL,
			OnFailure: "continue",
			Asserts:   []step.Assertion{{Type: "status_code", Expect: "200"}},
		},
	}

	runner := NewRunner(steps, false)
	report := runner.Run()

	if len(report.Steps) != 2 {
		t.Errorf("expected 2 steps executed (continue), got %d", len(report.Steps))
	}
	if report.Failed != 2 {
		t.Errorf("Failed = %d, want 2", report.Failed)
	}
}

func TestRunMultipleAssertsAllPass(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"code":0,"msg":"success"}`)
	}))
	defer server.Close()

	steps := []step.Step{
		{
			Name:   "multi-assert",
			Method: "GET",
			URL:    server.URL,
			Asserts: []step.Assertion{
				{Type: "status_code", Expect: "200"},
				{Type: "jsonpath", Path: "$.code", Expect: "0"},
				{Type: "body_match", Expect: "success"},
			},
		},
	}

	runner := NewRunner(steps, false)
	report := runner.Run()

	if report.Passed != 1 {
		t.Errorf("Passed = %d, want 1", report.Passed)
	}
}

func TestRunWithRetrySuccessOnSecondAttempt(t *testing.T) {
	attempt := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt++
		if attempt == 1 {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	steps := []step.Step{
		{
			Name:    "retry-test",
			Method:  "GET",
			URL:     server.URL,
			RetryCount:    2,
			RetryInterval: 1, // 1 nanosecond
			Asserts: []step.Assertion{{Type: "status_code", Expect: "200"}},
		},
	}

	runner := NewRunner(steps, false)
	report := runner.Run()

	if report.Passed != 1 {
		t.Errorf("Passed = %d, want 1", report.Passed)
	}
	if report.Steps[0].Attempts != 2 {
		t.Errorf("Attempts = %d, want 2", report.Steps[0].Attempts)
	}
}

func TestRunWithPreScriptAndPostScript(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	steps := []step.Step{
		{
			Name:       "script-test",
			Method:     "GET",
			URL:        server.URL,
			PreScript:  "echo before",
			PostScript: "echo after",
			Asserts:    []step.Assertion{{Type: "status_code", Expect: "200"}},
		},
	}

	runner := NewRunner(steps, false)
	report := runner.Run()

	if report.Passed != 1 {
		t.Errorf("Passed = %d, want 1", report.Passed)
	}
	if report.Steps[0].PreScriptErr != nil {
		t.Errorf("PreScriptErr: %v", report.Steps[0].PreScriptErr)
	}
	if report.Steps[0].PostScriptErr != nil {
		t.Errorf("PostScriptErr: %v", report.Steps[0].PostScriptErr)
	}
}

func TestRunBodyEqualsAssert(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"code":0}`)
	}))
	defer server.Close()

	steps := []step.Step{
		{
			Name:   "body-equals",
			Method: "GET",
			URL:    server.URL,
			Asserts: []step.Assertion{
				{Type: "body_equals", Expect: `{"code":0}`},
			},
		},
	}

	runner := NewRunner(steps, false)
	report := runner.Run()

	if report.Passed != 1 {
		t.Errorf("Passed = %d, want 1", report.Passed)
	}
}

func TestRunNoneAssert(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	steps := []step.Step{
		{
			Name:   "none-assert",
			Method: "GET",
			URL:    server.URL,
			Asserts: []step.Assertion{
				{Type: "none"},
			},
		},
	}

	runner := NewRunner(steps, false)
	report := runner.Run()

	// 'none' asserts always pass, even with 500
	if report.Passed != 1 {
		t.Errorf("Passed = %d, want 1 (none assert always passes)", report.Passed)
	}
}
