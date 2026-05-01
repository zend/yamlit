# yamlit — Design Document

**Date:** 2026-05-01
**Status:** Draft -> Validated

## Overview

yamlit is a lightweight YAML-driven HTTP API testing tool for individual developers. Written in Go, it ships as a single binary. Users write test scenarios as YAML files, where each step is an HTTP request with assertions, variable extraction, retry logic, and shell hooks.

---

## 1. YAML Format

Lightweight list-style format. A file is an array of steps.

```yaml
- name: login                          # 步骤标识名
  method: POST                         # HTTP 方法
  url: https://api.example.com/login   # 请求 URL（支持 ${var}）
  params:                              # URL 查询参数
    key1: value1
  headers:                             # 请求头
    Content-Type: application/json
  body:                                # 请求体
    type: json                         # json | form | text
    content: '{"user":"test"}'         # 支持 ${var}
  timeout: 30s                         # 单次请求超时
  retry_count: 2                       # 失败重试次数
  retry_interval: 1s                   # 重试间隔
  on_failure: stop                     # stop | continue（默认 stop）
  asserts:
    - type: status_code                # 状态码比对
      expect: 201
    - type: jsonpath                   # JSONPath 键值比对
      path: $.code
      expect: "0"
    - type: body_match                 # 子串匹配
      expect: "success"
    - type: body_equals                # 整体 JSON 精确比对
      expect: '{"code":0,"msg":"ok"}'
    - type: none                       # 无断言
  extract:                             # 变量提取
    - source: body                     # body | header
      path: $.data.token
      var_name: auth_token
  pre_script: "echo before"            # 前置 Shell 脚本
  post_script: "echo after"            # 后置 Shell 脚本
```

**Key rules:**
- `method` is not subject to variable substitution. All other string fields support `${var_name}`.
- Variables are file-scoped, shared across steps.
- Steps execute sequentially. Variable pool carries over (but does not cross files).
- Undefined `\${var}` is kept as-is (no error).

---

## 2. Go Project Structure

```
yamlit/
├── cmd/
│   └── yamlit/
│       └── main.go              # CLI entry point
├── pkg/
│   ├── parser/                  # YAML parsing & validation
│   │   └── parser.go
│   ├── runner/                  # Execution orchestrator
│   │   └── runner.go
│   ├── step/                    # Step definition & HTTP execution
│   │   └── step.go
│   ├── assert/                  # Assertion engine
│   │   └── assert.go
│   ├── extract/                 # Variable extraction from responses
│   │   └── extract.go
│   ├── variable/                # Variable pool & template replacement
│   │   └── variable.go
│   └── reporter/                # Terminal output
│       └── reporter.go
├── testdata/
│   ├── basic.yaml
│   └── advanced.yaml
├── go.mod
├── go.sum
└── README.md
```

### Component responsibilities

| Package | Responsibility |
|---|---|
| **parser** | Read YAML, deserialize to Go structs, validate required fields, method, body type |
| **variable** | Manage `map[string]string` pool; provide `Replace(input string) string` for `${var}` substitution |
| **step** | Convert parsed step to executable HTTP request; `Execute(vars *variable.Pool) (*StepResult, error)` |
| **assert** | Compare `*http.Response` against assertion config; supports 5 types (status_code, jsonpath, body_match, body_equals, none) |
| **extract** | Extract values from response body (gjson) or headers into variable pool |
| **runner** | Orchestrate: iterate steps, retry logic, pre/post scripts, collect results |
| **reporter** | Colorful terminal output per step + summary |

### Data flow

```
YAML → parser.Parse(file) → []Step
                              ↓
runner.Run(steps)            step.Execute(vars)
    ↓                           ↓ pre_script → HTTP request → asserts → extract
iterate steps                   ↓ post_script
    ↓                           ↓ returns StepResult
汇总 Report
    ↓
reporter.Print(report)
```

---

## 3. Execution Engine & Retry

```
for i, step := range steps {
    // 1. Variable substitution on step fields
    resolvedStep = variable.ReplaceAll(step, vars)

    // 2. Pre-script (if configured)
    preErr = runShell(resolvedStep.PreScript, timeout)
    if preErr != nil → step fails

    // 3. HTTP request + retry loop
    for attempt := 0; attempt <= retryCount; attempt++ {
        resp, err = sendHTTP(resolvedStep)       // with timeout
        if err != nil → retry (net error is retriable)

        failures = assert.Run(resp, resolvedStep.Asserts)
        if len(failures) == 0 → break retry loop
        if attempt < retryCount → sleep(retryInterval)
    }

    // 4. If retries exhausted → step failure
    // 5. Extract variables (only if asserts passed)
    extract.Run(resp, resolvedStep.Extract, vars)
    // 6. Post-script (runs regardless of success/failure)
    postErr = runShell(resolvedStep.PostScript, timeout)
    // 7. Collect result, print to terminal
}
```

**Failure types:**

| Type | Cause | Report tag |
|---|---|---|
| Network error | DNS/connection/TLS failure | `✗ NET_ERROR` |
| Timeout | Request exceeds `timeout` | `✗ TIMEOUT` |
| Assert failure | Status/JSONPath/text mismatch | `✗ ASSERT` |
| Script error | Pre/post script non-zero exit | `✗ SCRIPT` |
| Parse error | Invalid YAML, missing fields | Fatal, no report |

**`on_failure`:** `stop` (default) aborts the entire file run. `continue` marks step failed and proceeds.

---

## 4. Assertion Engine

```go
type Assertion struct {
    Type   string `yaml:"type"`              // status_code | jsonpath | body_match | body_equals | none
    Path   string `yaml:"path,omitempty"`
    Expect string `yaml:"expect"`
}
```

- **status_code** — `r.StatusCode` compared to `Expect` (string→int)
- **jsonpath** — Uses `tidwall/gjson` to extract value from body by JSONPath, string-compare to `Expect`
- **body_match** — Body contains `Expect` as substring
- **body_equals** — Exact string equality of body vs `Expect` (trimmed)
- **none** — Skip all assertions

Multiple assertions are AND: all must pass. First failure reports detail.

---

## 5. Variable Extraction

```go
type ExtractItem struct {
    Source  string `yaml:"source"`   // body | header
    Path    string `yaml:"path"`     // JSONPath (body) or header key (header)
    VarName string `yaml:"var_name"`
}
```

- `source: body` → gjson extraction from response body
- `source: header` → case-insensitive lookup in response headers
- Variables are extracted only if assertions pass
- Later values overwrite earlier ones (same variable name)

---

## 6. Shell Script Execution

```go
func runShell(script string, timeout time.Duration) error
```

- Uses `os/exec` with `sh -c "<script>"` (Linux/macOS)
- Synchronous with context timeout
- stdout/stderr merged, printed in verbose mode only
- Non-zero exit → error → step failure
- Post-scripts execute regardless of step outcome (for cleanup)

---

## 7. CLI Interface

```
./yamlit <file.yaml>              # single file
./yamlit <directory/>             # batch all .yaml/.yml files
./yamlit <pattern>                # wildcard, e.g. "tests/*.yaml"
./yamlit <input> -v               # verbose (show request/response bodies)
./yamlit <input> -o report.json   # optional JSON report file
```

Exit code: 0 (all pass), 1 (any failure).

---

## 8. Terminal Output

**Real-time:**
```
▶ [1/3] login ............................................ POST https://api.example.com/login
  ✓ 200 OK (238ms)

▶ [2/3] get_user_info .................................... GET https://api.example.com/user
  ✓ 200 OK (45ms)

▶ [3/3] create_order ..................................... POST https://api.example.com/orders
  ✗ 500 Internal Server Error (312ms)
    └─ 断言失败: $.code == "0"，实际值 "50001"
```

**Summary:**
```
════════════════════════════════════════════════
  总计: 3  |  ✓ 通过: 2  |  ✗ 失败: 1  |  耗时: 1.2s
  失败步骤: create_order
════════════════════════════════════════════════
```

**Colors:** ✓ green, ✗ red, URL/method cyan, step header white/gray, assertion detail red indented.

**Batch summary:**
```
▶ auth_test.yaml ......... 3/3 通过 (450ms)
▶ user_test.yaml ......... 2/3 失败 (1.2s)
  └─ 失败步骤: create_order
▶ order_test.yaml ........ 4/4 通过 (890ms)
══════════════════════════════════════
  文件: 3  |  ✓ 全通过: 2  |  ✗ 有失败: 1
══════════════════════════════════════
```

---

## 9. Libraries

| Library | Purpose |
|---|---|
| `gopkg.in/yaml.v3` | YAML deserialization |
| `github.com/tidwall/gjson` | JSONPath extraction (assertions & extract) |
| `github.com/fatih/color` | Terminal colored output |
| Standard `net/http` | HTTP client |
| Standard `os/exec` | Shell script execution |
