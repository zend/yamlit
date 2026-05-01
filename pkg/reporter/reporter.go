package reporter

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/fatih/color"

	"github.com/mike/yaml-testing/pkg/runner"
	"github.com/mike/yaml-testing/pkg/step"
)

// Exported color variables for use by CLI
var (
	PassColor = color.New(color.FgGreen)
	FailColor = color.New(color.FgRed)
)

var (
	colorStep   = color.New(color.FgHiWhite)
	colorURL    = color.New(color.FgCyan)
	colorDim    = color.New(color.FgHiBlack)
	colorBold   = color.New(color.Bold)
	colorYellow = color.New(color.FgYellow)
)

// PrintReport prints the full report for a single runner run
func PrintReport(report *runner.Report) {
	for _, result := range report.Steps {
		printStep(result)
	}
	printSummary(report)
}

// PrintFileResult prints a one-line summary for a file in batch mode
func PrintFileResult(name string, report *runner.Report) {
	if report.Failed > 0 {
		FailColor.Printf("▶ %s ......... %d/%d ✗ 失败 (%s)\n",
			name, report.Passed, report.Total, FormatDuration(report.Elapsed))

		failedSteps := make([]string, 0)
		for _, r := range report.Steps {
			if r.Error != nil {
				failedSteps = append(failedSteps, r.Name)
			}
		}
		FailColor.Printf("  └─ 失败步骤: %s\n", strings.Join(failedSteps, ", "))
	} else {
		PassColor.Printf("▶ %s ......... %d/%d ✓ 通过 (%s)\n",
			name, report.Passed, report.Total, FormatDuration(report.Elapsed))
	}
}

// FormatDuration formats a duration for display
func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return d.Round(time.Second).String()
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
		PassColor.Printf("✓ %d OK (%s)", result.StatusCode, FormatDuration(result.Duration))
	} else {
		FailColor.Printf("✗ %d %s (%s)", result.StatusCode, errorTag(result), FormatDuration(result.Duration))
	}
	fmt.Println()

	// Assertion failures
	if len(result.Failures) > 0 {
		for _, f := range result.Failures {
			if !f.Passed {
				FailColor.Printf("    └─ %s", failureDescription(f))
				fmt.Println()
			}
		}
	}

	// Script errors
	if result.PreScriptErr != nil {
		FailColor.Printf("    └─ pre-script: %v", result.PreScriptErr)
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
		FailColor.Printf("  总计: %d  |  ✓ 通过: %d  |  ✗ 失败: %d  |  耗时: %s\n",
			report.Total, report.Passed, report.Failed, FormatDuration(report.Elapsed))
	} else {
		PassColor.Printf("  总计: %d  |  ✓ 通过: %d  |  ✗ 失败: %d  |  耗时: %s\n",
			report.Total, report.Passed, report.Failed, FormatDuration(report.Elapsed))
	}

	// List failed steps
	failedNames := make([]string, 0)
	for _, r := range report.Steps {
		if r.Error != nil {
			failedNames = append(failedNames, r.Name)
		}
	}
	if len(failedNames) > 0 {
		FailColor.Printf("  失败步骤: %s\n", strings.Join(failedNames, ", "))
	}

	fmt.Println(sep)
	fmt.Println()
}

func errorTag(result *step.StepResult) string {
	if result.Error == nil {
		return "OK"
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
