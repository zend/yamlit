package step

import (
	"io"
	"net/http"
	"strings"
	"time"
)

// StepResult holds the full result of executing a single step
type StepResult struct {
	Name         string
	StepNumber   int
	TotalSteps   int
	Method       string
	URL          string
	StatusCode   int
	Duration     time.Duration
	Error        error          // network error or script error or assertion failure
	Failures     []AssertResult // assertion failures (empty if all passed)
	Body         string         // response body (for verbose / debug)
	Attempts     int            // how many HTTP attempts used
	PreScriptErr error
	PostScriptErr error
}

// AssertResult holds the result of a single assertion check
type AssertResult struct {
	Type     string // status_code | jsonpath | body_match | body_equals | none
	Path     string // for jsonpath
	Expected string
	Actual   string
	Passed   bool
}

// ToHTTPResponse creates a synthetic *http.Response from StepResult
// for use by assert and extract packages
func (r *StepResult) ToHTTPResponse() *http.Response {
	return &http.Response{
		StatusCode: r.StatusCode,
		Body:       io.NopCloser(strings.NewReader(r.Body)),
		Header:     make(http.Header),
	}
}
