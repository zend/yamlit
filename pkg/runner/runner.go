package runner

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/zend/yamlit/pkg/assert"
	"github.com/zend/yamlit/pkg/extract"
	"github.com/zend/yamlit/pkg/step"
	"github.com/zend/yamlit/pkg/variable"
)

// Runner orchestrates the execution of a sequence of test steps
type Runner struct {
	steps    []step.Step
	vars     *variable.Pool
	executor *step.Executor
	verbose  bool
}

// Report holds the overall execution results
type Report struct {
	Steps   []*step.StepResult
	Total   int
	Passed  int
	Failed  int
	Elapsed time.Duration
}

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

// NewRunner creates a new runner for the given steps
func NewRunner(steps []step.Step, verbose bool) *Runner {
	return &Runner{
		steps:    steps,
		vars:     variable.NewPool(),
		executor: step.NewExecutor(),
		verbose:  verbose,
	}
}

// Run executes all steps sequentially and returns the report
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

		// HTTP request with retry loop
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
				result.StatusCode = lastResult.StatusCode
				result.Duration = lastResult.Duration
				break
			}

			result.StatusCode = lastResult.StatusCode
			result.Duration = lastResult.Duration
			result.Body = lastResult.Body

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
				// Extract variables only on assertion success
				if len(s.Extract) > 0 {
					extract.Run(lastResult.ToHTTPResponse(), s.Extract, r.vars)
				}
				result.Failures = nil
				break
			}

			// Assertion failed — retry
			if attempt < retryCount {
				time.Sleep(retryInterval)
			} else {
				result.Failures = lastFailures
				result.Error = fmt.Errorf("assertion failed")
			}
		}

		if lastResult != nil && lastResult.Error != nil && result.Error == nil {
			result.Error = lastResult.Error
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
