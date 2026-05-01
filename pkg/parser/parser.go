package parser

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/zend/yamlit/pkg/step"
)

// validMethods is the set of supported HTTP methods
var validMethods = map[string]bool{
	"GET": true, "POST": true, "PUT": true, "DELETE": true,
	"PATCH": true, "HEAD": true, "OPTIONS": true,
}

// validBodyTypes is the set of supported body content types
var validBodyTypes = map[string]bool{
	"json": true, "form": true, "text": true,
}

// ParseFile reads and parses a YAML file into a slice of Steps
func ParseFile(path string) ([]step.Step, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	return Parse(data)
}

// Parse parses YAML bytes into a slice of Steps
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
		if !validMethods[s.Method] {
			return nil, fmt.Errorf("step %d (%s): invalid method %q", i+1, s.Name, s.Method)
		}
		if s.URL == "" {
			return nil, fmt.Errorf("step %d (%s): url is required", i+1, s.Name)
		}
		if s.Body != nil && s.Body.Type != "" {
			if !validBodyTypes[s.Body.Type] {
				return nil, fmt.Errorf("step %d (%s): invalid body type %q (must be json, form, or text)", i+1, s.Name, s.Body.Type)
			}
		}
		if s.OnFailure == "" {
			s.OnFailure = step.OnFailureStop
		} else if s.OnFailure != step.OnFailureStop && s.OnFailure != step.OnFailureContinue {
			return nil, fmt.Errorf("step %d (%s): invalid on_failure %q (must be stop or continue)", i+1, s.Name, s.OnFailure)
		}
		steps[i] = s
	}

	return steps, nil
}
