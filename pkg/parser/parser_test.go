package parser

import (
	"testing"
)

func TestParseBasic(t *testing.T) {
	steps, err := ParseFile("../../testdata/basic.yaml")
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if len(steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(steps))
	}

	s0 := steps[0]
	if s0.Name != "login" {
		t.Errorf("step[0].Name = %q, want %q", s0.Name, "login")
	}
	if s0.Method != "POST" {
		t.Errorf("step[0].Method = %q, want %q", s0.Method, "POST")
	}
	if s0.URL != "https://api.example.com/login" {
		t.Errorf("step[0].URL = %q, want %q", s0.URL, "https://api.example.com/login")
	}
	if s0.Body == nil || s0.Body.Type != "json" {
		t.Errorf("step[0].Body.Type = %q, want %q", s0.Body.Type, "json")
	}
	if s0.Body.Content != `{"user":"test","pass":"123"}` {
		t.Errorf("step[0].Body.Content = %q", s0.Body.Content)
	}
	if len(s0.Asserts) != 2 {
		t.Fatalf("step[0].Asserts len = %d, want 2", len(s0.Asserts))
	}
	if s0.Asserts[0].Type != "status_code" || s0.Asserts[0].Expect != "200" {
		t.Errorf("step[0].Asserts[0] mismatch: %+v", s0.Asserts[0])
	}
	if s0.Asserts[1].Type != "jsonpath" || s0.Asserts[1].Path != "$.code" || s0.Asserts[1].Expect != "0" {
		t.Errorf("step[0].Asserts[1] mismatch: %+v", s0.Asserts[1])
	}
	if len(s0.Extract) != 1 {
		t.Fatalf("step[0].Extract len = %d, want 1", len(s0.Extract))
	}
	if s0.Extract[0].Source != "body" || s0.Extract[0].Path != "$.data.token" || s0.Extract[0].VarName != "auth_token" {
		t.Errorf("step[0].Extract[0] mismatch: %+v", s0.Extract[0])
	}
	if s0.RetryCount != 2 {
		t.Errorf("step[0].RetryCount = %d, want 2", s0.RetryCount)
	}
	if s0.Timeout != 0 {
		t.Errorf("step[0].Timeout = %v, want 0", s0.Timeout)
	}
}

func TestParseInline(t *testing.T) {
	yaml := `
- name: test
  method: GET
  url: https://example.com
  asserts:
    - type: status_code
      expect: 200
`
	steps, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	if steps[0].Name != "test" {
		t.Errorf("Name = %q", steps[0].Name)
	}
}

func TestParseInvalidMethod(t *testing.T) {
	yaml := `
- name: bad
  method: INVALID
  url: https://example.com
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for invalid method")
	}
}

func TestParseNoName(t *testing.T) {
	yaml := `
- method: GET
  url: https://example.com
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestParseNoMethod(t *testing.T) {
	yaml := `
- name: test
  url: https://example.com
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for missing method")
	}
}

func TestParseNoURL(t *testing.T) {
	yaml := `
- name: test
  method: GET
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for missing url")
	}
}

func TestParseInvalidBodyType(t *testing.T) {
	yaml := `
- name: test
  method: POST
  url: https://example.com
  body:
    type: xml
    content: "<a></a>"
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for invalid body type")
	}
}

func TestParseInvalidOnFailure(t *testing.T) {
	yaml := `
- name: test
  method: GET
  url: https://example.com
  on_failure: invalid
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for invalid on_failure")
	}
}

func TestParseEmptySteps(t *testing.T) {
	yaml := `[]`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for empty steps")
	}
}

func TestParseMethodCaseInsensitive(t *testing.T) {
	yaml := `
- name: test
  method: post
  url: https://example.com
`
	steps, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if steps[0].Method != "POST" {
		t.Errorf("Method = %q, want POST", steps[0].Method)
	}
}

func TestParseDefaultOnFailure(t *testing.T) {
	yaml := `
- name: test
  method: GET
  url: https://example.com
`
	steps, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if steps[0].OnFailure != "stop" {
		t.Errorf("OnFailure = %q, want 'stop'", steps[0].OnFailure)
	}
}
