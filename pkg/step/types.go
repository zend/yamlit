package step

import "time"

// Step represents a single HTTP API test step
type Step struct {
	Name          string            `yaml:"name"`
	Method        string            `yaml:"method"`
	URL           string            `yaml:"url"`
	Params        map[string]string `yaml:"params,omitempty"`
	Headers       map[string]string `yaml:"headers,omitempty"`
	Body          *Body             `yaml:"body,omitempty"`
	Timeout       time.Duration     `yaml:"timeout,omitempty"`
	RetryCount    int               `yaml:"retry_count,omitempty"`
	RetryInterval time.Duration     `yaml:"retry_interval,omitempty"`
	OnFailure     string            `yaml:"on_failure,omitempty"`
	Asserts       []Assertion       `yaml:"asserts,omitempty"`
	Extract       []ExtractItem     `yaml:"extract,omitempty"`
	PreScript     string            `yaml:"pre_script,omitempty"`
	PostScript    string            `yaml:"post_script,omitempty"`
}

// Body represents an HTTP request body
type Body struct {
	Type    string `yaml:"type"`    // json | form | text
	Content string `yaml:"content"`
}

// Assertion defines an assertion to check against the HTTP response
type Assertion struct {
	Type   string `yaml:"type"`   // status_code | jsonpath | body_match | body_equals | none
	Path   string `yaml:"path,omitempty"`
	Expect string `yaml:"expect"`
}

// ExtractItem defines a variable to extract from the HTTP response
type ExtractItem struct {
	Source  string `yaml:"source"`   // body | header
	Path    string `yaml:"path"`
	VarName string `yaml:"var_name"`
}

// OnFailure constants
const (
	OnFailureStop     = "stop"
	OnFailureContinue = "continue"
)
