# Dotenv Support Implementation Plan

> **REQUIRED SUB-SKILL:** Use the executing-plans skill to implement this plan task-by-task.

**Goal:** Add `.env` file support to yamlit, injecting `KEY=VALUE` pairs into the variable pool so YAML files can reference them via `${VAR}`.

**Architecture:** New `pkg/dotenv/` parses `.env` files into `map[string]string`. Existing `variable.Pool` gets two new methods (`LoadOSEnv`, `LoadDotenv`) and a third lookup layer in `Replace()`. Runner gets a `NewRunnerWithVars` constructor. CLI loads `.env` + OS env before dispatching to Runner.

**Tech Stack:** Go 1.25, standard library (`os`, `bufio`, `strings`), no new dependencies.

---

### Task 1: Create dotenv parser package

**Files:**
- Create: `pkg/dotenv/dotenv.go`
- Create: `pkg/dotenv/dotenv_test.go`

**Step 1: Write the failing test**

```go
package dotenv

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadBasic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	os.WriteFile(path, []byte("KEY=value\nFOO=bar\n"), 0644)

	vars, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if vars["KEY"] != "value" {
		t.Errorf("KEY = %q, want %q", vars["KEY"], "value")
	}
	if vars["FOO"] != "bar" {
		t.Errorf("FOO = %q, want %q", vars["FOO"], "bar")
	}
	if len(vars) != 2 {
		t.Errorf("got %d vars, want 2", len(vars))
	}
}

func TestLoadQuoted(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	os.WriteFile(path, []byte(`KEY="hello world"`+"\n"), 0644)

	vars, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if vars["KEY"] != "hello world" {
		t.Errorf("KEY = %q, want %q", vars["KEY"], "hello world")
	}
}

func TestLoadCommentsAndBlanks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := []byte("# this is a comment\n\nKEY=val\n# another comment\n\nFOO=bar\n")
	os.WriteFile(path, content, 0644)

	vars, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if vars["KEY"] != "val" || vars["FOO"] != "bar" || len(vars) != 2 {
		t.Errorf("unexpected vars: %v", vars)
	}
}

func TestLoadDuplicateKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	os.WriteFile(path, []byte("KEY=first\nKEY=second\n"), 0644)

	vars, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if vars["KEY"] != "second" {
		t.Errorf("KEY = %q, want %q", vars["KEY"], "second")
	}
}

func TestLoadEmptyValue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	os.WriteFile(path, []byte("KEY=\n"), 0644)

	vars, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if vars["KEY"] != "" {
		t.Errorf("KEY = %q, want empty", vars["KEY"])
	}
}

func TestLoadMalformedLine(t *testing.T) {
	// Lines without "=" are skipped
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	os.WriteFile(path, []byte("KEY=val\nbadline\nFOO=bar\n"), 0644)

	vars, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if vars["KEY"] != "val" || vars["FOO"] != "bar" || len(vars) != 2 {
		t.Errorf("unexpected vars: %v", vars)
	}
}

func TestLoadFileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/.env")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/dotenv/ -v`
Expected: FAIL — package doesn't exist yet

**Step 3: Write minimal implementation**

```go
package dotenv

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Load parses a .env file and returns key-value pairs.
// Returns an error only if the file cannot be read.
// Malformed lines (no "=") are silently skipped.
func Load(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("dotenv: %w", err)
	}
	defer f.Close()

	vars := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.IndexByte(line, '=')
		if idx == -1 {
			continue // malformed line, skip
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		if len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' {
			val = val[1 : len(val)-1]
		}
		vars[key] = val
	}
	return vars, scanner.Err()
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/dotenv/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/dotenv/
git commit -m "feat: add dotenv file parser"
```

---

### Task 2: Extend variable pool with dotenv + OS env support

**Files:**
- Modify: `pkg/variable/pool.go`
- Extend: `pkg/variable/pool_test.go`

**Step 1: Write the failing tests (add to pool_test.go)**

```go
func TestReplaceWithDotenv(t *testing.T) {
	p := NewPool()
	p.dotenv = map[string]string{"BASE_URL": "https://api.example.com"}

	result := p.Replace("${BASE_URL}/users")
	if result != "https://api.example.com/users" {
		t.Errorf("Replace = %q, want %q", result, "https://api.example.com/users")
	}
}

func TestReplaceWithOSEnv(t *testing.T) {
	t.Setenv("MY_VAR", "from_os")
	p := NewPool()
	p.LoadOSEnv()

	result := p.Replace("prefix-${MY_VAR}-suffix")
	if result != "prefix-from_os-suffix" {
		t.Errorf("Replace = %q, want %q", result, "prefix-from_os-suffix")
	}
}

func TestReplacePriority(t *testing.T) {
	// store > dotenv > os env
	t.Setenv("THE_VAR", "from_os")
	p := NewPool()
	p.dotenv = map[string]string{"THE_VAR": "from_dotenv"}
	p.Set("THE_VAR", "from_store")

	result := p.Replace("${THE_VAR}")
	if result != "from_store" {
		t.Errorf("Replace = %q, want %q", result, "from_store")
	}
}

func TestLoadDotenv(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/.env"
	os.WriteFile(path, []byte("DB_URL=postgres://localhost\n"), 0644)

	p := NewPool()
	if err := p.LoadDotenv(path); err != nil {
		t.Fatalf("LoadDotenv failed: %v", err)
	}

	result := p.Replace("${DB_URL}")
	if result != "postgres://localhost" {
		t.Errorf("Replace = %q, want %q", result, "postgres://localhost")
	}
}

func TestLoadDotenvFileNotFound(t *testing.T) {
	p := NewPool()
	err := p.LoadDotenv("/nonexistent/.env")
	if err != nil {
		t.Errorf("expected nil for missing file, got %v", err)
	}
}
```

**Step 2: Run test to verify they fail**

Run: `go test ./pkg/variable/ -v`
Expected: FAIL — dotenv field and methods don't exist yet

**Step 3: Update pool.go implementation**

Add `os` import to the import block. Add `dotenv` field. Add `LoadOSEnv()` and `LoadDotenv()` methods. Update `Replace()`.

```go
package variable

import (
	"os"
	"regexp"
)

var varRegex = regexp.MustCompile(`\$\{([^}]+)\}`)

// Pool manages a set of named variables and provides template substitution
type Pool struct {
	store  map[string]string
	dotenv map[string]string // .env and OS env vars
}

// NewPool creates a new empty variable pool
func NewPool() *Pool {
	return &Pool{
		store:  make(map[string]string),
		dotenv: make(map[string]string),
	}
}

// Set stores a variable value
func (p *Pool) Set(name, value string) {
	p.store[name] = value
}

// Get retrieves a variable value. Returns false if not found.
func (p *Pool) Get(name string) (string, bool) {
	v, ok := p.store[name]
	return v, ok
}

// LoadOSEnv injects all OS environment variables into the pool's dotenv layer.
func (p *Pool) LoadOSEnv() {
	for _, e := range os.Environ() {
		idx := indexByte(e, '=')
		if idx == -1 {
			continue
		}
		p.dotenv[e[:idx]] = e[idx+1:]
	}
}

// LoadDotenv parses a .env file and loads key-value pairs into the pool.
// If path is empty or the file does not exist, it is silently ignored.
func (p *Pool) LoadDotenv(path string) error {
	if path == "" {
		return nil
	}
	vars, err := dotenv.Load(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // silent skip
		}
		return err
	}
	for k, v := range vars {
		p.dotenv[k] = v
	}
	return nil
}

// Replace substitutes all ${var} templates in input with their values.
// Lookup order: store (step-extracted) > dotenv (.env + OS env) > os.Environ
// Undefined variables are left as-is.
func (p *Pool) Replace(input string) string {
	return varRegex.ReplaceAllStringFunc(input, func(match string) string {
		name := match[2 : len(match)-1]
		// 1. Step-extracted vars (highest priority)
		if val, ok := p.store[name]; ok {
			return val
		}
		// 2. .env / OS env vars (medium priority)
		if val, ok := p.dotenv[name]; ok {
			return val
		}
		// 3. OS environment (lowest priority)
		if val, ok := os.LookupEnv(name); ok {
			return val
		}
		return match
	})
}

// indexByte finds the first occurrence of c in s.
func indexByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}
```

**Step 4: Run test to verify they pass**

Run: `go test ./pkg/variable/ -v`
Expected: PASS (all 6 tests including 3 new ones)

**Step 5: Commit**

```bash
git add pkg/variable/
git commit -m "feat: extend variable pool with dotenv and OS env support"
```

---

### Task 3: Add NewRunnerWithVars to Runner

**Files:**
- Modify: `pkg/runner/runner.go`
- Extend: `pkg/runner/runner_test.go`

**Step 1: Write the failing test**

Add to runner_test.go:

```go
func TestRunWithExternalVars(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok"}`)
	}))
	defer server.Close()

	vars := variable.NewPool()
	vars.dotenv = map[string]string{"TEST_URL": server.URL}

	steps := []step.Step{
		{
			Name:   "external-url",
			Method: "GET",
			URL:    "${TEST_URL}",
			Asserts: []step.Assertion{
				{Type: "status_code", Expect: "200"},
			},
		},
	}

	r := NewRunnerWithVars(steps, false, vars)
	report := r.Run()

	if report.Failed > 0 {
		t.Errorf("expected all pass, got %d failures", report.Failed)
	}
	if report.Steps[0].Error != nil {
		t.Errorf("step error: %v", report.Steps[0].Error)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/runner/ -v -run TestRunWithExternalVars`
Expected: FAIL — NewRunnerWithVars not defined

**Step 3: Add NewRunnerWithVars to runner.go**

```go
// NewRunnerWithVars creates a runner that uses an external variable pool.
// This allows sharing .env and OS env vars across multiple YAML file runs.
func NewRunnerWithVars(steps []step.Step, verbose bool, vars *variable.Pool) *Runner {
	return &Runner{
		steps:    steps,
		vars:     vars,
		executor: step.NewExecutor(),
		verbose:  verbose,
	}
}
```

Also add import for `net/http/httptest` and `fmt` if not already in test file.

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/runner/ -v -run TestRunWithExternalVars`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/runner/
git commit -m "feat: add NewRunnerWithVars constructor"
```

---

### Task 4: Update CLI to load .env and OS env

**Files:**
- Modify: `cmd/yamlit/main.go`

**Step 1: Update main.go to initialize shared vars pool**

After flag parsing and before the file loop, add:

```go
// Initialize shared variable pool with OS env and .env file
vars := variable.NewPool()
vars.LoadOSEnv()
vars.LoadDotenv(".env")
```

Replace `runner.NewRunner(steps, *verbose)` with `runner.NewRunnerWithVars(steps, *verbose, vars)`.

**Step 2: Build and verify compilation**

Run: `go build -o yamlit ./cmd/yamlit/`
Expected: success

**Step 3: Manual integration test**

```bash
# Create .env
echo "BASE_URL=https://httpbin.org" > /tmp/testenv/.env

# Create test file
cat > /tmp/testenv/test.yaml << 'EOF'
- name: test-dotenv
  method: GET
  url: ${BASE_URL}/get
  asserts:
    - type: status_code
      expect: 200
EOF

# Run from that directory
cd /tmp/testenv && /path/to/yamlit test.yaml
```

Expected: Step passes, resolves ${BASE_URL} from .env

**Step 4: Commit**

```bash
git add cmd/yamlit/main.go
git commit -m "feat: load .env and OS env vars in CLI"
```

---

### Task 5: Run full test suite and verify

**Step 1: Run all tests**

Run: `go test ./... -v`
Expected: all pass (7 packages, all tests green)

**Step 2: Run go vet**

Run: `go vet ./...`
Expected: no issues

**Step 3: Build binary**

Run: `go build -o yamlit ./cmd/yamlit/`
Expected: success

**Step 4: Final commit**

```bash
git add .
git commit -m "chore: final cleanup and verification"
```
