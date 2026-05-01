package assert

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/mike/yaml-testing/pkg/step"
)

func TestStatusCodePass(t *testing.T) {
	resp := &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("{}"))}
	results := Run(resp, []step.Assertion{
		{Type: "status_code", Expect: "200"},
	})
	if len(results) != 1 || !results[0].Passed {
		t.Errorf("expected pass, got %+v", results)
	}
}

func TestStatusCodeFail(t *testing.T) {
	resp := &http.Response{StatusCode: 500, Body: io.NopCloser(strings.NewReader("{}"))}
	results := Run(resp, []step.Assertion{
		{Type: "status_code", Expect: "200"},
	})
	if len(results) != 1 || results[0].Passed {
		t.Errorf("expected fail, got %+v", results)
	}
	if results[0].Actual != "500" {
		t.Errorf("actual = %q, want 500", results[0].Actual)
	}
}

func TestJSONPathPass(t *testing.T) {
	body := `{"code":0,"data":{"name":"test"}}`
	resp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	results := Run(resp, []step.Assertion{
		{Type: "jsonpath", Path: "$.data.name", Expect: "test"},
	})
	if len(results) != 1 || !results[0].Passed {
		t.Errorf("expected pass, got %+v", results)
	}
}

func TestJSONPathFail(t *testing.T) {
	body := `{"code":0}`
	resp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	results := Run(resp, []step.Assertion{
		{Type: "jsonpath", Path: "$.nonexistent", Expect: "value"},
	})
	if len(results) != 1 || results[0].Passed {
		t.Errorf("expected fail, got %+v", results)
	}
}

func TestBodyMatchPass(t *testing.T) {
	body := `{"status":"ok","message":"everything is fine"}`
	resp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	results := Run(resp, []step.Assertion{
		{Type: "body_match", Expect: "everything"},
	})
	if len(results) != 1 || !results[0].Passed {
		t.Errorf("expected pass, got %+v", results)
	}
}

func TestBodyMatchFail(t *testing.T) {
	body := `{"status":"ok"}`
	resp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	results := Run(resp, []step.Assertion{
		{Type: "body_match", Expect: "nonexistent"},
	})
	if len(results) != 1 || results[0].Passed {
		t.Errorf("expected fail, got %+v", results)
	}
}

func TestBodyEqualsPass(t *testing.T) {
	body := `{"code":0}`
	resp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	results := Run(resp, []step.Assertion{
		{Type: "body_equals", Expect: `{"code":0}`},
	})
	if len(results) != 1 || !results[0].Passed {
		t.Errorf("expected pass, got %+v", results)
	}
}

func TestBodyEqualsFail(t *testing.T) {
	body := `{"code":1}`
	resp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	results := Run(resp, []step.Assertion{
		{Type: "body_equals", Expect: `{"code":0}`},
	})
	if len(results) != 1 || results[0].Passed {
		t.Errorf("expected fail, got %+v", results)
	}
}

func TestNoneAlwaysPasses(t *testing.T) {
	resp := &http.Response{StatusCode: 500, Body: io.NopCloser(strings.NewReader("error"))}
	results := Run(resp, []step.Assertion{{Type: "none"}})
	if len(results) != 1 || !results[0].Passed {
		t.Errorf("expected pass for none, got %+v", results)
	}
}

func TestMultipleAndAllPass(t *testing.T) {
	body := `{"code":0}`
	resp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	results := Run(resp, []step.Assertion{
		{Type: "status_code", Expect: "200"},
		{Type: "jsonpath", Path: "$.code", Expect: "0"},
	})
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for i, r := range results {
		if !r.Passed {
			t.Errorf("result[%d] failed: %+v", i, r)
		}
	}
}

func TestMultipleAndOneFails(t *testing.T) {
	body := `{"code":0}`
	resp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	results := Run(resp, []step.Assertion{
		{Type: "status_code", Expect: "200"},
		{Type: "jsonpath", Path: "$.code", Expect: "1"},
	})
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if !results[0].Passed {
		t.Errorf("result[0] should pass: %+v", results[0])
	}
	if results[1].Passed {
		t.Errorf("result[1] should fail: %+v", results[1])
	}
}

func TestBodyEqualsWithWhitespace(t *testing.T) {
	body := `  {"code":0}  `
	resp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	results := Run(resp, []step.Assertion{
		{Type: "body_equals", Expect: `{"code":0}`},
	})
	if len(results) != 1 || !results[0].Passed {
		t.Errorf("expected pass (trimmed), got %+v", results)
	}
}
