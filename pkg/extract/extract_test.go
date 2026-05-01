package extract

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/zend/yamlit/pkg/step"
	"github.com/zend/yamlit/pkg/variable"
)

func TestExtractFromBody(t *testing.T) {
	body := `{"data":{"token":"abc123"}}`
	resp := &http.Response{
		Body:   io.NopCloser(strings.NewReader(body)),
		Header: make(http.Header),
	}

	vars := variable.NewPool()
	items := []step.ExtractItem{
		{Source: "body", Path: "data.token", VarName: "auth_token"},
	}

	Run(resp, items, vars)

	val, ok := vars.Get("auth_token")
	if !ok {
		t.Fatal("auth_token not found")
	}
	if val != "abc123" {
		t.Errorf("auth_token = %q, want %q", val, "abc123")
	}
}

func TestExtractFromBodyWithDollarPrefix(t *testing.T) {
	body := `{"data":{"token":"xyz789"}}`
	resp := &http.Response{
		Body:   io.NopCloser(strings.NewReader(body)),
		Header: make(http.Header),
	}

	vars := variable.NewPool()
	items := []step.ExtractItem{
		{Source: "body", Path: "$.data.token", VarName: "token"},
	}

	Run(resp, items, vars)

	val, ok := vars.Get("token")
	if !ok {
		t.Fatal("token not found")
	}
	if val != "xyz789" {
		t.Errorf("token = %q, want %q", val, "xyz789")
	}
}

func TestExtractFromHeader(t *testing.T) {
	header := make(http.Header)
	header.Set("X-Session-Id", "sess-999")
	resp := &http.Response{
		Body:   io.NopCloser(strings.NewReader("{}")),
		Header: header,
	}

	vars := variable.NewPool()
	items := []step.ExtractItem{
		{Source: "header", Path: "X-Session-Id", VarName: "session_id"},
	}

	Run(resp, items, vars)

	val, ok := vars.Get("session_id")
	if !ok {
		t.Fatal("session_id not found")
	}
	if val != "sess-999" {
		t.Errorf("session_id = %q, want %q", val, "sess-999")
	}
}

func TestExtractHeaderCaseInsensitive(t *testing.T) {
	header := make(http.Header)
	header.Set("X-Session-Id", "abc")
	resp := &http.Response{
		Body:   io.NopCloser(strings.NewReader("{}")),
		Header: header,
	}

	vars := variable.NewPool()
	items := []step.ExtractItem{
		{Source: "header", Path: "x-session-id", VarName: "sid"},
	}

	Run(resp, items, vars)

	val, ok := vars.Get("sid")
	if !ok {
		t.Fatal("sid not found (case insensitive)")
	}
	if val != "abc" {
		t.Errorf("sid = %q, want %q", val, "abc")
	}
}

func TestExtractOverwrite(t *testing.T) {
	vars := variable.NewPool()
	vars.Set("myname", "old")

	body := `{"value":"new"}`
	resp := &http.Response{
		Body:   io.NopCloser(strings.NewReader(body)),
		Header: make(http.Header),
	}

	items := []step.ExtractItem{
		{Source: "body", Path: "value", VarName: "myname"},
	}

	Run(resp, items, vars)

	val, _ := vars.Get("myname")
	if val != "new" {
		t.Errorf("expected overwrite: got %q, want %q", val, "new")
	}
}

func TestExtractNonexistentPath(t *testing.T) {
	body := `{"data":{}}`
	resp := &http.Response{
		Body:   io.NopCloser(strings.NewReader(body)),
		Header: make(http.Header),
	}

	vars := variable.NewPool()
	items := []step.ExtractItem{
		{Source: "body", Path: "data.nonexistent", VarName: "should_not_exist"},
	}

	Run(resp, items, vars)

	_, ok := vars.Get("should_not_exist")
	if ok {
		t.Error("variable should not be created for nonexistent path")
	}
}

func TestExtractNonexistentHeader(t *testing.T) {
	resp := &http.Response{
		Body:   io.NopCloser(strings.NewReader("{}")),
		Header: make(http.Header),
	}

	vars := variable.NewPool()
	items := []step.ExtractItem{
		{Source: "header", Path: "X-Nonexistent", VarName: "should_not_exist"},
	}

	Run(resp, items, vars)

	_, ok := vars.Get("should_not_exist")
	if ok {
		t.Error("variable should not be created for nonexistent header")
	}
}

func TestExtractInvalidSource(t *testing.T) {
	resp := &http.Response{
		Body:   io.NopCloser(strings.NewReader("{}")),
		Header: make(http.Header),
	}

	vars := variable.NewPool()
	items := []step.ExtractItem{
		{Source: "invalid", Path: "x", VarName: "v"},
	}

	// Should not panic
	Run(resp, items, vars)
}
