package step

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mike/yaml-testing/pkg/variable"
)

// Executor executes a single step's HTTP request
type Executor struct {
	client *http.Client
}

// NewExecutor creates a new HTTP step executor
func NewExecutor() *Executor {
	return &Executor{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Execute performs the HTTP request defined by the step and returns the result.
// Variable substitution must be done by the caller before calling Execute.
func (e *Executor) Execute(step Step, vars *variable.Pool) *StepResult {
	result := &StepResult{
		Name:   step.Name,
		Method: step.Method,
		URL:    step.URL,
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

	// Add query params to URL
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
		result.Error = err
		return result
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Set timeout via context (not mutating shared client)
	timeout := step.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req = req.WithContext(ctx)

	// Execute
	start := time.Now()
	resp, err := e.client.Do(req)
	result.Duration = time.Since(start)

	if err != nil {
		result.Error = err
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode

	bodyBytes, _ := io.ReadAll(resp.Body)
	result.Body = string(bodyBytes)

	return result
}
