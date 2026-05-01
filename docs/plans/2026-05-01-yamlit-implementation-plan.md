# yamlit Implementation Plan

> **REQUIRED SUB-SKILL:** Use the executing-plans skill to implement this plan task-by-task.

**Goal:** Implement a YAML-driven HTTP API testing tool as a single Go binary.

**Architecture:** Seven Go packages under `pkg/` (parser, runner, step, assert, extract, variable, reporter) with a thin CLI entry at `cmd/yamlit/`. YAML is parsed into typed structs, executed step-by-step with retry/assert/extract orchestration, and results are printed to terminal.

**Tech Stack:** Go 1.26, `gopkg.in/yaml.v3` (YAML), `github.com/tidwall/gjson` (JSONPath), `github.com/fatih/color` (terminal colors), standard library `net/http` and `os/exec`.

---

### Task 1: Define Core Data Types

**Files:**
- Create: `pkg/step/types.go`
- Create: `pkg/step/step_result.go`

**Step 1: Write types.go with YAML step definition structs**

```go
package step

import "time"

type Step struct {
    Name          string        `yaml:"name"`
    Method        string        `yaml:"method"`
    URL           string        `yaml:"url"`
    Params        map[string]string `yaml:"params,omitempty"`
    Headers       map[string]string `yaml:"headers,omitempty"`
    Body          *Body         `yaml:"body,omitempty"`
    Timeout       time.Duration `yaml:"timeout,omitempty"`
    RetryCount    int           `yaml:"retry_count,omitempty"`
    RetryInterval time.Duration `yaml:"retry_interval,omitempty"`
    OnFailure     string        `yaml:"on_failure,omitempty"`
    Asserts       []Assertion   `yaml:"asserts,omitempty"`
    Extract       []ExtractItem `yaml:"extract,omitempty"`
    PreScript     string        `yaml:"pre_script,omitempty"`
    PostScript    string        `yaml:"post_script,omitempty"`
}

type Body struct {
    Type    string `yaml:"type"`    // json | form | text
    Content string `yaml:"content"`
}

type Assertion struct {
    Type   string `yaml:"type"`    // status_code | jsonpath | body_match | body_equals | none
    Path   string `yaml:"path,omitempty"`
    Expect string `yaml:"expect"`
}

type ExtractItem struct {
    Source  string `yaml:"source"`   // body | header
    Path    string `yaml:"path"`
    VarName string `yaml:"var_name"`
}

// Default constants
const (
    OnFailureStop     = "stop"
    OnFailureContinue = "continue"
)
```

**Step 2: Write step_result.go for execution results**

```go
package step

import "time"

type StepResult struct {
    Name        string
    StepNumber  int
    TotalSteps  int
    Method      string
    URL         string
    StatusCode  int
    Duration    time.Duration
    Error       error          // network error or script error
    Failures    []AssertResult // assertion failures (empty if all passed)
    Body        string         // response body (for verbose / debug)
    Attempts    int            // how many retry attempts used
    PreScriptErr error
    PostScriptErr error
}

type AssertResult struct {
    Type     string // status_code | jsonpath | body_match | body_equals | none
    Path     string // for jsonpath
    Expected string
    Actual   string
    Passed   bool
}
```

**Step 3: Commit**

```bash
git add pkg/step/
git commit -m "feat: add core step data types"
```

---

### Task 2: Implement Variable Pool

**Files:**
- Create: `pkg/variable/pool.go`
- Create: `pkg/variable/pool_test.go`

**Step 1: Write the failing test**

```go
package variable

import "testing"

func TestReplace(t *testing.T) {
    p := NewPool()
    p.Set("name", "world")
    p.Set("count", "42")

    tests := []struct{
        input    string
        expected string
    }{
        {"hello ${name}", "hello world"},
        {"${name} ${name}", "world world"},
        {"count=${count}", "count=42"},
        {"${undefined}", "${undefined}"},
        {"no vars", "no vars"},
        {"${name}/${count}", "world/42"},
    }

    for _, tt := range tests {
        result := p.Replace(tt.input)
        if result != tt.expected {
            t.Errorf("Replace(%q) = %q, want %q", tt.input, result, tt.expected)
        }
    }
}

func TestReplaceAll(t *testing.T) {
    p := NewPool()
    p.Set("token", "abc123")

    step := &Step{
        URL: "https://api.example.com/${token}/data",
        Headers: map[string]string{"Authorization": "Bearer ${token}"},
    }

    // We'll implement ReplaceAll later
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/variable/ -v`
Expected: FAIL — package doesn't exist yet

**Step 3: Write minimal implementation**

```go
package variable

import (
    "regexp"
    "strings"
)

var varRegex = regexp.MustCompile(`\$\{([^}]+)\}`)

type Pool struct {
    store map[string]string
}

func NewPool() *Pool {
    return &Pool{store: make(map[string]string)}
}

func (p *Pool) Set(name, value string) {
    p.store[name] = value
}

func (p *Pool) Get(name string) (string, bool) {
    v, ok := p.store[name]
    return v, ok
}

func (p *Pool) Replace(input string) string {
    return varRegex.ReplaceAllStringFunc(input, func(match string) string {
        name := match[2 : len(match)-1] // strip ${ and }
        if val, ok := p.store[name]; ok {
            return val
        }
        return match
    })
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/variable/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/variable/
git commit -m "feat: add variable pool with template replacement"
```

---

### Task 3: Implement Parser

**Files:**
- Create: `pkg/parser/parser.go`
- Create: `pkg/parser/parser_test.go`
- Create: `testdata/basic.yaml`

**Step 1: Write the failing test**

```go
package parser

import "testing"

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
    if len(s0.Asserts) != 2 {
        t.Errorf("step[0].Asserts len = %d, want 2", len(s0.Asserts))
    }
    if s0.Asserts[0].Type != "status_code" || s0.Asserts[0].Expect != "200" {
        t.Errorf("step[0].Asserts[0] mismatch")
    }
    if len(s0.Extract) != 1 {
        t.Errorf("step[0].Extract len = %d, want 1", len(s0.Extract))
    }
    if s0.RetryCount != 2 {
        t.Errorf("step[0].RetryCount = %d, want 2", s0.RetryCount)
    }
}

func TestParseInvalidMethod(t *testing.T) {
    // Create a temp file with invalid method
    // Expect validation error
}
```

**Step 2: Create testdata/basic.yaml**

```yaml
- name: login
  method: POST
  url: https://api.example.com/login
  headers:
    Content-Type: application/json
  body:
    type: json
    content: '{"user":"test","pass":"123"}'
  retry_count: 2
  retry_interval: 1s
  asserts:
    - type: status_code
      expect: 200
    - type: jsonpath
      path: $.code
      expect: "0"
  extract:
    - source: body
      path: $.data.token
      var_name: auth_token

- name: get_profile
  method: GET
  url: https://api.example.com/profile
  headers:
    Authorization: "Bearer ${auth_token}"
  asserts:
    - type: status_code
      expect: 200
    - type: body_match
      expect: "profile"
```

**Step 3: Write minimal implementation**

```go
package parser

import (
    "fmt"
    "os"
    "strings"

    "gopkg.in/yaml.v3"
    "github.com/zend/yamlit/pkg/step"
)

func ParseFile(path string) ([]step.Step, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("read file: %w", err)
    }

    return Parse(data)
}

func Parse(data []byte) ([]step.Step, error) {
    var steps []step.Step
    if err := yaml.Unmarshal(data, &steps); err != nil {
        return nil, fmt.Errorf("yaml parse: %w", err)
    }

    if len(steps) == 0 {
        return nil, fmt.Errorf("no steps defined")
    }

    for i, s := range steps {
        if s.Name == "" {
            return nil, fmt.Errorf("step %d: name is required", i+1)
        }
        if s.Method == "" {
            return nil, fmt.Errorf("step %d (%s): method is required", i+1, s.Name)
        }
        s.Method = strings.ToUpper(s.Method)
        validMethods := map[string]bool{
            "GET": true, "POST": true, "PUT": true, "DELETE": true,
            "PATCH": true, "HEAD": true, "OPTIONS": true,
        }
        if !validMethods[s.Method] {
            return nil, fmt.Errorf("step %d (%s): invalid method %q", i+1, s.Name, s.Method)
        }
        if s.URL == "" {
            return nil, fmt.Errorf("step %d (%s): url is required", i+1, s.Name)
        }
        if s.Body != nil && s.Body.Type != "" {
            validTypes := map[string]bool{"json": true, "form": true, "text": true}
            if !validTypes[s.Body.Type] {
                return nil, fmt.Errorf("step %d (%s): invalid body type %q", i+1, s.Name, s.Body.Type)
            }
        }
        if s.OnFailure == "" {
            s.OnFailure = step.OnFailureStop
        } else if s.OnFailure != step.OnFailureStop && s.OnFailure != step.OnFailureContinue {
            return nil, fmt.Errorf("step %d (%s): invalid on_failure %q", i+1, s.Name, s.OnFailure)
        }
        steps[i] = s
    }

    return steps, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/parser/ -v`
Expected: PASS (add gopkg.in/yaml.v3 dependency)

**Step 5: Commit**

```bash
git add pkg/parser/ testdata/basic.yaml
git commit -m "feat: add YAML parser with validation"
```

---

### Task 4: Implement Assertion Engine

**Files:**
- Create: `pkg/assert/assert.go`
- Create: `pkg/assert/assert_test.go`

**Step 1: Write the failing test**

```go
package assert

import (
    "net/http"
    "strings"
    "testing"
    "github.com/zend/yamlit/pkg/step"
)

func TestStatusCodePass(t *testing.T) {
    resp := &http.Response{StatusCode: 200}
    results := Run(resp, []step.Assertion{
        {Type: "status_code", Expect: "200"},
    })
    if len(results) != 1 || !results[0].Passed {
        t.Errorf("expected pass, got %+v", results)
    }
}

func TestStatusCodeFail(t *testing.T) {
    resp := &http.Response{StatusCode: 500}
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

func TestBodyEquals(t *testing.T) {
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

func TestNone(t *testing.T) {
    resp := &http.Response{StatusCode: 500, Body: io.NopCloser(strings.NewReader(""))}
    results := Run(resp, []step.Assertion{{Type: "none"}})
    if len(results) != 1 || !results[0].Passed {
        t.Errorf("expected pass for none, got %+v", results)
    }
}

func TestMultipleAnd(t *testing.T) {
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
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/assert/ -v`
Expected: FAIL — package doesn't exist yet

**Step 3: Write minimal implementation**

```go
package assert

import (
    "io"
    "net/http"
    "strconv"
    "strings"

    "github.com/tidwall/gjson"
    "github.com/zend/yamlit/pkg/step"
)

func Run(resp *http.Response, asserts []step.Assertion) []step.AssertResult {
    results := make([]step.AssertResult, 0, len(asserts))

    bodyBytes, _ := io.ReadAll(resp.Body)
    resp.Body.Close()
    bodyStr := string(bodyBytes)

    for _, a := range asserts {
        result := step.AssertResult{
            Type:     a.Type,
            Path:     a.Path,
            Expected: a.Expect,
            Passed:   true,
        }

        switch a.Type {
        case "status_code":
            result.Actual = strconv.Itoa(resp.StatusCode)
            result.Passed = result.Actual == a.Expect

        case "jsonpath":
            actual := gjson.Get(bodyStr, a.Path)
            result.Actual = actual.String()
            result.Passed = actual.Exists() && actual.String() == a.Expect

        case "body_match":
            result.Actual = bodyStr
            result.Passed = strings.Contains(bodyStr, a.Expect)

        case "body_equals":
            result.Actual = bodyStr
            result.Passed = strings.TrimSpace(bodyStr) == strings.TrimSpace(a.Expect)

        case "none":
            result.Passed = true

        default:
            result.Passed = false
            result.Actual = "unknown assertion type: " + a.Type
        }

        results = append(results, result)
    }

    return results
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/assert/ -v`
Expected: PASS (add gjson dependency)

**Step 5: Commit**

```bash
git add pkg/assert/
git commit -m "feat: add assertion engine (status_code, jsonpath, body_match, body_equals, none)"
```

---

### Task 5: Implement Variable Extraction

**Files:**
- Create: `pkg/extract/extract.go`
- Create: `pkg/extract/extract_test.go`

**Step 1: Write the failing test**

```go
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
        Body: io.NopCloser(strings.NewReader(body)),
        Header: make(http.Header),
    }

    vars := variable.NewPool()
    items := []step.ExtractItem{
        {Source: "body", Path: "$.data.token", VarName: "auth_token"},
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

func TestExtractOverwrite(t *testing.T) {
    vars := variable.NewPool()
    vars.Set("old", "value")

    body := `{"new":"value2"}`
    resp := &http.Response{
        Body:   io.NopCloser(strings.NewReader(body)),
        Header: make(http.Header),
    }

    // Extract into existing variable
    items := []step.ExtractItem{
        {Source: "body", Path: "$.new", VarName: "old"},
    }

    Run(resp, items, vars)

    val, _ := vars.Get("old")
    if val != "value2" {
        t.Errorf("expected overwrite: got %q, want %q", val, "value2")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/extract/ -v`
Expected: FAIL — package doesn't exist yet

**Step 3: Write minimal implementation**

```go
package extract

import (
    "io"
    "net/http"
    "strings"

    "github.com/tidwall/gjson"
    "github.com/zend/yamlit/pkg/step"
    "github.com/zend/yamlit/pkg/variable"
)

func Run(resp *http.Response, items []step.ExtractItem, vars *variable.Pool) {
    bodyBytes, _ := io.ReadAll(resp.Body)
    resp.Body.Close()
    bodyStr := string(bodyBytes)

    for _, item := range items {
        switch item.Source {
        case "body":
            result := gjson.Get(bodyStr, item.Path)
            if result.Exists() {
                vars.Set(item.VarName, result.String())
            }
        case "header":
            val := resp.Header.Get(item.Path)
            if val != "" {
                vars.Set(item.VarName, val)
            }
        }
    }
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/extract/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/extract/
git commit -m "feat: add variable extraction from response body and headers"
```

---

### Task 6: Implement Step Executor (HTTP Request)

**Files:**
- Create: `pkg/step/executor.go`
- Create: `pkg/step/executor_test.go`

**Step 1: Write the implementation (tests need real HTTP server)**

```go
package step

import (
    "bytes"
    "fmt"
    "io"
    "net/http"
    "strings"
    "time"
)

type Executor struct {
    client *http.Client
}

func NewExecutor() *Executor {
    return &Executor{
        client: &http.Client{},
    }
}

func (e *Executor) Execute(step Step, vars *Pool) *StepResult {
    result := &StepResult{
        Name:       step.Name,
        Method:     step.Method,
        URL:        step.URL,
        StepNumber: 0,
        Duration:   0,
    }

    // Apply variable substitution
    url := vars.Replace(step.URL)
    headers := make(map[string]string)
    for k, v := range step.Headers {
        headers[k] = vars.Replace(v)
    }
    params := make(map[string]string)
    for k, v := range step.Params {
        params[k] = vars.Replace(v)
    }

    // Build request body
    var bodyReader io.Reader
    if step.Body != nil {
        content := vars.Replace(step.Body.Content)
        switch step.Body.Type {
        case "json":
            bodyReader = strings.NewReader(content)
            if headers["Content-Type"] == "" {
                headers["Content-Type"] = "application/json"
            }
        case "form":
            // Parse key=value&key=value or just send content as-is
            bodyReader = strings.NewReader(content)
            if headers["Content-Type"] == "" {
                headers["Content-Type"] = "application/x-www-form-urlencoded"
            }
        case "text":
            bodyReader = strings.NewReader(content)
            if headers["Content-Type"] == "" {
                headers["Content-Type"] = "text/plain"
            }
        }
    }

    // Add query params
    if len(params) > 0 {
        q := url
        sep := "?"
        if strings.Contains(url, "?") {
            sep = "&"
        }
        for k, v := range params {
            q = q + sep + k + "=" + v
            sep = "&"
        }
        url = q
    }

    // Create request
    req, err := http.NewRequest(step.Method, url, bodyReader)
    if err != nil {
        result.Error = fmt.Errorf("create request: %w", err)
        return result
    }

    for k, v := range headers {
        req.Header.Set(k, v)
    }

    // Set timeout
    timeout := step.Timeout
    if timeout == 0 {
        timeout = 30 * time.Second
    }
    e.client.Timeout = timeout

    // Execute
    start := time.Now()
    resp, err := e.client.Do(req)
    result.Duration = time.Since(start)

    if err != nil {
        result.Error = fmt.Errorf("http request: %w", err)
        return result
    }
    defer resp.Body.Close()

    result.StatusCode = resp.StatusCode

    bodyBytes, _ := io.ReadAll(resp.Body)
    result.Body = string(bodyBytes)

    return result
}
```

**Step 2: Write tests using httptest**

```go
package step

import (
    "fmt"
    "net/http"
    "net/http/httptest"
    "testing"
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
    vars := NewPool()

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
}

func TestExecutePOSTWithBody(t *testing.T) {
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
    vars := NewPool()

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
    vars := NewPool()

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
```

**Step 3: Run tests to verify they pass**

Run: `go test ./pkg/step/ -v`
Expected: PASS

**Step 4: Commit**

```bash
git add pkg/step/executor.go pkg/step/executor_test.go
git commit -m "feat: add HTTP request executor with body type support and variable substitution"
```

---

### Task 7: Implement Runner (Orchestrator)

**Files:**
- Create: `pkg/runner/runner.go`

**Step 1: Write the implementation**

```go
package runner

import (
    "context"
    "fmt"
    "os/exec"
    "strings"
    "time"

    "github.com/zend/yamlit/pkg/assert"
    "github.com/zend/yamlit/pkg/extract"
    "github.com/zend/yamlit/pkg/step"
    "github.com/zend/yamlit/pkg/variable"
)

type Runner struct {
    steps    []step.Step
    vars     *variable.Pool
    executor *step.Executor
    verbose  bool
}

type Report struct {
    Steps   []*step.StepResult
    Total   int
    Passed  int
    Failed  int
    Elapsed time.Duration
}

func NewRunner(steps []step.Step, verbose bool) *Runner {
    return &Runner{
        steps:    steps,
        vars:     variable.NewPool(),
        executor: step.NewExecutor(),
        verbose:  verbose,
    }
}

func (r *Runner) Run() *Report {
    report := &Report{
        Steps: make([]*step.StepResult, 0, len(r.steps)),
        Total: len(r.steps),
    }

    start := time.Now()

    for i, s := range r.steps {
        result := &step.StepResult{
            Name:       s.Name,
            StepNumber: i + 1,
            TotalSteps: len(r.steps),
            Method:     s.Method,
            URL:        s.URL,
        }

        // Apply variable substitution to step fields
        resolvedName := r.vars.Replace(s.Name)
        resolvedURL := r.vars.Replace(s.URL)
        result.Name = resolvedName
        result.URL = resolvedURL

        // Pre-script
        if s.PreScript != "" {
            script := r.vars.Replace(s.PreScript)
            if err := runShell(script, 30*time.Second); err != nil {
                result.PreScriptErr = err
                result.Error = fmt.Errorf("pre-script failed: %w", err)
                report.Steps = append(report.Steps, result)
                report.Failed++
                if s.OnFailure == step.OnFailureStop {
                    break
                }
                continue
            }
        }

        // HTTP request with retry
        retryCount := s.RetryCount
        if retryCount < 0 {
            retryCount = 0
        }
        retryInterval := s.RetryInterval
        if retryInterval <= 0 {
            retryInterval = 1 * time.Second
        }

        var lastResult *step.StepResult
        var lastFailures []step.AssertResult

        for attempt := 0; attempt <= retryCount; attempt++ {
            lastResult = r.executor.Execute(s, r.vars)
            result.Attempts = attempt + 1

            if lastResult.Error != nil {
                // Network error — retry
                if attempt < retryCount {
                    time.Sleep(retryInterval)
                    continue
                }
                result.Error = lastResult.Error
                break
            }

            // Run assertions
            failures := assert.Run(lastResult.ToHTTPResponse(), s.Asserts)
            lastFailures = failures

            allPassed := true
            for _, f := range failures {
                if !f.Passed {
                    allPassed = false
                    break
                }
            }

            if allPassed {
                // Extract variables only on success
                if len(s.Extract) > 0 {
                    extract.Run(lastResult.ToHTTPResponse(), s.Extract, r.vars)
                }
                lastResult.Failures = nil
                break
            }

            // Assertion failed — retry if possible
            if attempt < retryCount {
                time.Sleep(retryInterval)
            } else {
                lastResult.Failures = lastFailures
            }
        }

        // Copy fields from last execution result
        if lastResult != nil {
            result.StatusCode = lastResult.StatusCode
            result.Duration = lastResult.Duration
            result.Body = lastResult.Body
            if lastResult.Error != nil {
                result.Error = lastResult.Error
            }
            if len(lastResult.Failures) > 0 {
                result.Failures = lastResult.Failures
                result.Error = fmt.Errorf("assertion failed")
            }
        }

        // Post-script (always runs)
        if s.PostScript != "" {
            script := r.vars.Replace(s.PostScript)
            if err := runShell(script, 30*time.Second); err != nil {
                result.PostScriptErr = err
            }
        }

        report.Steps = append(report.Steps, result)
        if result.Error != nil {
            report.Failed++
        } else {
            report.Passed++
        }

        // Check on_failure
        if result.Error != nil && s.OnFailure == step.OnFailureStop {
            break
        }
    }

    report.Elapsed = time.Since(start)
    return report
}

func runShell(script string, timeout time.Duration) error {
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()

    cmd := exec.CommandContext(ctx, "sh", "-c", script)
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("script error: %w\noutput: %s", err, string(output))
    }
    return nil
}
```

**Add ToHTTPResponse method to StepResult:**

In executor_test.go, also add a helper method to make StepResult testable. Actually, let's add it to step_result.go:

```go
// ToHTTPResponse creates a synthetic *http.Response from StepResult
// for use by assert and extract packages
func (r *StepResult) ToHTTPResponse() *http.Response {
    return &http.Response{
        StatusCode: r.StatusCode,
        Body:       io.NopCloser(strings.NewReader(r.Body)),
        Header:     make(http.Header),
    }
}
```

**Step 2: Run tests**

Run: `go build ./...`
Expected: success

**Step 3: Commit**

```bash
git add pkg/runner/ pkg/step/step_result.go
git commit -m "feat: add runner orchestrator with retry, pre/post scripts, on_failure"
```

---

### Task 8: Implement Terminal Reporter

**Files:**
- Create: `pkg/reporter/reporter.go`

**Step 1: Write the implementation**

```go
package reporter

import (
    "fmt"
    "strings"
    "time"
    "unicode/utf8"

    "github.com/fatih/color"
    "github.com/zend/yamlit/pkg/runner"
    "github.com/zend/yamlit/pkg/step"
)

var (
    colorStep     = color.New(color.FgHiWhite)
    colorPass     = color.New(color.FgGreen)
    colorFail     = color.New(color.FgRed)
    colorURL      = color.New(color.FgCyan)
    colorDim      = color.New(color.FgHiBlack)
    colorBold     = color.New(color.Bold)
    colorYellow   = color.New(color.FgYellow)
)

func PrintReport(report *runner.Report) {
    for _, result := range report.Steps {
        printStep(result)
    }
    printSummary(report)
}

func printStep(result *step.StepResult) {
    // Step header: ▶ [1/3] get_user
    stepHeader := fmt.Sprintf("▶ [%d/%d] %s", result.StepNumber, result.TotalSteps, result.Name)
    colorStep.Print(stepHeader)

    // Padding dots
    lineWidth := 60
    currentWidth := utf8.RuneCountInString(stepHeader)
    if currentWidth < lineWidth {
        fmt.Print(strings.Repeat(".", lineWidth-currentWidth))
    }

    // URL and method
    colorURL.Printf(" %s %s", result.Method, result.URL)
    fmt.Println()

    // Result line
    fmt.Print("  ")
    if result.Error == nil {
        colorPass.Printf("✓ %d OK (%s)", result.StatusCode, formatDuration(result.Duration))
    } else {
        colorFail.Printf("✗ %d %s (%s)", result.StatusCode, errorTag(result), formatDuration(result.Duration))
    }
    fmt.Println()

    // Assertion failures
    if len(result.Failures) > 0 {
        for _, f := range result.Failures {
            if !f.Passed {
                colorFail.Printf("    └─ %s", failureDescription(f))
                fmt.Println()
            }
        }
    }

    // Script errors
    if result.PreScriptErr != nil {
        colorFail.Printf("    └─ pre-script: %v", result.PreScriptErr)
        fmt.Println()
    }
    if result.PostScriptErr != nil {
        colorYellow.Printf("    └─ post-script: %v", result.PostScriptErr)
        fmt.Println()
    }

    fmt.Println()
}

func printSummary(report *runner.Report) {
    sep := strings.Repeat("═", 50)
    fmt.Println(sep)

    if report.Failed > 0 {
        colorFail.Printf("  总计: %d  |  ✓ 通过: %d  |  ✗ 失败: %d  |  耗时: %s\n",
            report.Total, report.Passed, report.Failed, formatDuration(report.Elapsed))
    } else {
        colorPass.Printf("  总计: %d  |  ✓ 通过: %d  |  ✗ 失败: %d  |  耗时: %s\n",
            report.Total, report.Passed, report.Failed, formatDuration(report.Elapsed))
    }

    // List failed steps
    failedNames := make([]string, 0)
    for _, r := range report.Steps {
        if r.Error != nil {
            failedNames = append(failedNames, r.Name)
        }
    }
    if len(failedNames) > 0 {
        colorFail.Printf("  失败步骤: %s\n", strings.Join(failedNames, ", "))
    }

    fmt.Println(sep)
    fmt.Println()
}

func formatDuration(d time.Duration) string {
    if d < time.Second {
        return fmt.Sprintf("%dms", d.Milliseconds())
    }
    if d < time.Minute {
        return fmt.Sprintf("%.1fs", d.Seconds())
    }
    return d.Round(time.Second).String()
}

func errorTag(result *step.StepResult) string {
    if result.Error == nil {
        return ""
    }
    errMsg := result.Error.Error()
    if strings.Contains(errMsg, "pre-script") || strings.Contains(errMsg, "post-script") {
        return "SCRIPT"
    }
    if strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "context deadline") {
        return "TIMEOUT"
    }
    if strings.Contains(errMsg, "assertion") {
        return "ASSERT"
    }
    return "NET_ERROR"
}

func failureDescription(f step.AssertResult) string {
    switch f.Type {
    case "status_code":
        return fmt.Sprintf("状态码: 期望 %s，实际 %s", f.Expected, f.Actual)
    case "jsonpath":
        return fmt.Sprintf("JSONPath %s: 期望 %s，实际 %s", f.Path, f.Expected, f.Actual)
    case "body_match":
        return fmt.Sprintf("未找到匹配文本: %s", f.Expected)
    case "body_equals":
        return fmt.Sprintf("响应体不匹配: 期望 %s", f.Expected)
    default:
        return fmt.Sprintf("%s: 期望 %s，实际 %s", f.Type, f.Expected, f.Actual)
    }
}
```

**Step 2: Run tests**

Run: `go build ./...`
Expected: success

**Step 3: Commit**

```bash
git add pkg/reporter/
git commit -m "feat: add terminal reporter with color output"
```

---

### Task 9: Implement CLI Entry Point

**Files:**
- Create: `cmd/yamlit/main.go`

**Step 1: Write the implementation**

```go
package main

import (
    "flag"
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "github.com/zend/yamlit/pkg/parser"
    "github.com/zend/yamlit/pkg/reporter"
    "github.com/zend/yamlit/pkg/runner"
    "github.com/zend/yamlit/pkg/variable"
)

func main() {
    verbose := flag.Bool("v", false, "verbose mode: print request/response bodies")
    outputFile := flag.String("o", "", "output JSON report to file")
    flag.Parse()

    args := flag.Args()
    if len(args) < 1 {
        fmt.Fprintf(os.Stderr, "Usage: %s <file.yaml|directory|pattern> [-v] [-o report.json]\n", os.Args[0])
        os.Exit(1)
    }

    input := args[0]

    // Collect files
    var files []string
    info, err := os.Stat(input)
    if err != nil {
        // Try as glob pattern
        matches, err := filepath.Glob(input)
        if err != nil || len(matches) == 0 {
            fmt.Fprintf(os.Stderr, "error: %s: %v\n", input, err)
            os.Exit(1)
        }
        files = matches
    } else if info.IsDir() {
        // Walk directory
        err = filepath.Walk(input, func(path string, info os.FileInfo, err error) error {
            if err != nil {
                return err
            }
            if !info.IsDir() && (strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")) {
                files = append(files, path)
            }
            return nil
        })
        if err != nil {
            fmt.Fprintf(os.Stderr, "error walking directory: %v\n", err)
            os.Exit(1)
        }
    } else {
        files = []string{input}
    }

    if len(files) == 0 {
        fmt.Fprintln(os.Stderr, "no YAML files found")
        os.Exit(1)
    }

    // Process each file
    totalFiles := len(files)
    totalPassed := 0
    totalFailed := 0
    failedFiles := make(map[string][]string) // filename -> failed step names

    for _, file := range files {
        steps, err := parser.ParseFile(file)
        if err != nil {
            fmt.Fprintf(os.Stderr, "✗ %s: parse error: %v\n", filepath.Base(file), err)
            totalFailed++
            continue
        }

        r := runner.NewRunner(steps, *verbose)
        report := r.Run()

        if *verbose {
            // Print step details
            reporter.PrintReport(report)
        } else {
            // Print one-liner per file
            status := "✓"
            color := reporter.PassColor
            if report.Failed > 0 {
                status = "✗"
                color = reporter.FailColor
            }
            color.Printf("▶ %s ......... %d/%d %s (%s)\n",
                filepath.Base(file), report.Passed, report.Total, status, reporter.FormatDuration(report.Elapsed))

            if report.Failed > 0 {
                failedSteps := make([]string, 0)
                for _, r := range report.Steps {
                    if r.Error != nil {
                        failedSteps = append(failedSteps, r.Name)
                    }
                }
                reporter.FailColor.Printf("  └─ 失败步骤: %s\n", strings.Join(failedSteps, ", "))
            }
        }

        if report.Failed > 0 {
            totalFailed++
            failedFileSteps := make([]string, 0)
            for _, r := range report.Steps {
                if r.Error != nil {
                    failedFileSteps = append(failedFileSteps, r.Name)
                }
            }
            failedFiles[filepath.Base(file)] = failedFileSteps
        } else {
            totalPassed++
        }
    }

    // Batch summary
    if len(files) > 1 || totalFiles > 0 {
        sep := strings.Repeat("═", 50)
        fmt.Println(sep)

        if totalFailed > 0 {
            reporter.FailColor.Printf("  文件: %d  |  ✓ 全通过: %d  |  ✗ 有失败: %d\n",
                totalFiles, totalPassed, totalFailed)
            for file, steps := range failedFiles {
                reporter.FailColor.Printf("  ✗ %s: %s\n", file, strings.Join(steps, ", "))
            }
        } else {
            reporter.PassColor.Printf("  文件: %d  |  ✓ 全部通过\n", totalFiles)
        }
        fmt.Println(sep)
    }

    if totalFailed > 0 {
        os.Exit(1)
    }
}
```

**Note:** Some reporter symbols need to be exported for main.go to use. Update reporter.go to export Color vars.

**Step 2: Update reporter.go with exported symbols**

```go
var PassColor = color.New(color.FgGreen)
var FailColor = color.New(color.FgRed)
func FormatDuration(d time.Duration) string { return formatDuration(d) }
```

**Step 3: Run go build**

Run: `go build -o yamlit ./cmd/yamlit/`
Expected: success

**Step 4: Run end-to-end with testdata/basic.yaml**

Run: `./yamlit testdata/basic.yaml -v`
(Will fail to connect to api.example.com — but should show the execution flow)

**Step 5: Commit**

```bash
git add cmd/yamlit/ pkg/reporter/
git commit -m "feat: add CLI entry point with file, directory, and glob support"
```

---

### Task 10: Run All Tests and Verify

**Step 1: Run all unit tests**

Run: `go test ./... -v`
Expected: all pass

**Step 2: Run go vet**

Run: `go vet ./...`
Expected: no issues

**Step 3: Build binary**

Run: `go build -o yamlit ./cmd/yamlit/`
Expected: success

**Step 4: Final commit**

```bash
git add .
git commit -m "chore: final cleanup and build verification"
```

---

### Task 11: Update Root README

**Files:**
- Modify: `README.md`

**Step 1: Write documentation**

Update the root README.md with:
- Installation instructions (go install / build from source)
- Full YAML format reference
- CLI usage examples
- Variable substitution docs
- Assertion types reference

**Step 2: Commit**

```bash
git add README.md
git commit -m "docs: update README with full usage documentation"
```
