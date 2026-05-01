package reporter

import (
	"testing"
	"time"

	"github.com/zend/yamlit/pkg/runner"
	"github.com/zend/yamlit/pkg/step"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input    time.Duration
		expected string
	}{
		{500 * time.Millisecond, "500ms"},
		{1500 * time.Millisecond, "1.5s"},
		{90 * time.Second, "1m30s"},
		{0, "0ms"},
	}

	for _, tt := range tests {
		result := FormatDuration(tt.input)
		if result != tt.expected {
			t.Errorf("FormatDuration(%v) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestPrintReportNoCrash(t *testing.T) {
	report := &runner.Report{
		Steps: []*step.StepResult{
			{
				Name:       "test-step",
				StepNumber: 1,
				TotalSteps: 1,
				Method:     "GET",
				URL:        "http://example.com",
				StatusCode: 200,
				Duration:   100 * time.Millisecond,
			},
		},
		Total:   1,
		Passed:  1,
		Failed:  0,
		Elapsed: 100 * time.Millisecond,
	}

	// Just ensure no panic
	PrintReport(report)
}

func TestPrintReportWithFailures(t *testing.T) {
	report := &runner.Report{
		Steps: []*step.StepResult{
			{
				Name:       "fail-step",
				StepNumber: 1,
				TotalSteps: 1,
				Method:     "GET",
				URL:        "http://example.com",
				StatusCode: 500,
				Duration:   50 * time.Millisecond,
				Error:      assertError(),
				Failures: []step.AssertResult{
					{Type: "status_code", Expected: "200", Actual: "500", Passed: false},
				},
			},
		},
		Total:  1,
		Passed: 0,
		Failed: 1,
	}

	// Just ensure no panic
	PrintReport(report)
}

func assertError() error {
	return &assertErrorType{"assertion failed"}
}

type assertErrorType struct{ msg string }

func (e *assertErrorType) Error() string { return e.msg }
