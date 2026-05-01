package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mike/yaml-testing/pkg/parser"
	"github.com/mike/yaml-testing/pkg/reporter"
	"github.com/mike/yaml-testing/pkg/runner"
)

func main() {
	verbose := flag.Bool("v", false, "verbose mode: print detailed step results")
	outputFile := flag.String("o", "", "output JSON report to file")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s [-v] [-o report.json] <file.yaml|directory|pattern>\n", os.Args[0])
		os.Exit(1)
	}

	input := args[0]
	files := resolveFiles(input)

	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "no YAML files found: %s\n", input)
		os.Exit(1)
	}

	totalFiles := len(files)
	totalPassed := 0
	totalFailed := 0
	batchMode := totalFiles > 1
	fileReports := make(map[string]*runner.Report, len(files))
	failedFileNames := make([]string, 0)

	for _, file := range files {
		steps, err := parser.ParseFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "✗ %s: parse error: %v\n", filepath.Base(file), err)
			totalFailed++
			failedFileNames = append(failedFileNames, filepath.Base(file))
			continue
		}

		r := runner.NewRunner(steps, *verbose)
		report := r.Run()

		if *verbose || !batchMode {
			reporter.PrintReport(report)
		} else {
			reporter.PrintFileResult(filepath.Base(file), report)
		}

		fileReports[filepath.Base(file)] = report

		if report.Failed > 0 {
			totalFailed++
			failedFileNames = append(failedFileNames, filepath.Base(file))
		} else {
			totalPassed++
		}
	}

	// Batch mode summary
	if batchMode {
		sep := strings.Repeat("═", 50)
		fmt.Println(sep)
		if totalFailed > 0 {
			reporter.FailColor.Printf("  文件: %d  |  ✓ 全通过: %d  |  ✗ 有失败: %d\n",
				totalFiles, totalPassed, totalFailed)
			reporter.FailColor.Printf("  失败文件: %s\n", strings.Join(failedFileNames, ", "))
		} else {
			reporter.PassColor.Printf("  文件: %d  |  ✓ 全部通过\n", totalFiles)
		}
		fmt.Println(sep)
	}

	// Write JSON report if requested
	if *outputFile != "" {
		writeJSONReport(*outputFile, fileReports, *verbose, batchMode)
	}

	if totalFailed > 0 {
		os.Exit(1)
	}
}

func resolveFiles(input string) []string {
	info, err := os.Stat(input)
	if err == nil {
		if info.IsDir() {
			return findYAMLFiles(input)
		}
		return []string{input}
	}

	// Try as glob pattern
	matches, err := filepath.Glob(input)
	if err != nil || len(matches) == 0 {
		return nil
	}

	// Filter to yaml/yml files only
	var files []string
	for _, m := range matches {
		ext := strings.ToLower(filepath.Ext(m))
		if ext == ".yaml" || ext == ".yml" {
			files = append(files, m)
		}
	}
	return files
}

func findYAMLFiles(dir string) []string {
	// Use git ls-files to respect .gitignore
	cmd := exec.Command("git", "-C", dir, "ls-files", "--cached", "--others", "--exclude-standard")
	output, err := cmd.Output()
	if err == nil {
		var files []string
		for _, line := range strings.Split(string(output), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			path := filepath.Join(dir, line)
			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".yaml" || ext == ".yml" {
				files = append(files, path)
			}
		}
		return files
	}

	// Fallback: walk directory, skip hidden dirs
	var files []string
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".yaml" || ext == ".yml" {
			files = append(files, path)
		}
		return nil
	})
	return files
}

// jsonStepResult is a serializable representation of a step result
// jsonReport is a serializable representation of a single file report
type jsonReport struct {
	File    string           `json:"file,omitempty"`
	Total   int              `json:"total"`
	Passed  int              `json:"passed"`
	Failed  int              `json:"failed"`
	Elapsed string           `json:"elapsed"`
	Steps   []jsonStepResult `json:"steps"`
}

type jsonStepResult struct {
	Name         string            `json:"name"`
	Number       int               `json:"number"`
	Method       string            `json:"method"`
	URL          string            `json:"url"`
	StatusCode   int               `json:"status_code"`
	Duration     string            `json:"duration"`
	Passed       bool              `json:"passed"`
	Error        string            `json:"error,omitempty"`
	Failures     []jsonAssertResult `json:"failures,omitempty"`
	PreScriptErr string            `json:"pre_script_error,omitempty"`
	PostScriptErr string           `json:"post_script_error,omitempty"`
}

type jsonAssertResult struct {
	Type     string `json:"type"`
	Path     string `json:"path,omitempty"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Passed   bool   `json:"passed"`
}

func writeJSONReport(path string, fileReports map[string]*runner.Report, verbose bool, batchMode bool) {
	var reports []jsonReport

	for filename, report := range fileReports {
		jr := jsonReport{
			File:    filename,
			Total:   report.Total,
			Passed:  report.Passed,
			Failed:  report.Failed,
			Elapsed: reporter.FormatDuration(report.Elapsed),
			Steps:   make([]jsonStepResult, 0, len(report.Steps)),
		}

		for _, s := range report.Steps {
			js := jsonStepResult{
				Name:       s.Name,
				Number:     s.StepNumber,
				Method:     s.Method,
				URL:        s.URL,
				StatusCode: s.StatusCode,
				Duration:   reporter.FormatDuration(s.Duration),
				Passed:     s.Error == nil,
			}
			if s.Error != nil {
				js.Error = s.Error.Error()
			}
			if s.PreScriptErr != nil {
				js.PreScriptErr = s.PreScriptErr.Error()
			}
			if s.PostScriptErr != nil {
				js.PostScriptErr = s.PostScriptErr.Error()
			}
			if len(s.Failures) > 0 {
				for _, f := range s.Failures {
					if !f.Passed {
						js.Failures = append(js.Failures, jsonAssertResult{
							Type:     f.Type,
							Path:     f.Path,
							Expected: f.Expected,
							Actual:   f.Actual,
							Passed:   f.Passed,
						})
					}
				}
			}
			jr.Steps = append(jr.Steps, js)
		}

		reports = append(reports, jr)
	}

	data, err := json.MarshalIndent(reports, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error marshaling JSON report: %v\n", err)
		return
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "error writing JSON report: %v\n", err)
		return
	}

	if verbose || !batchMode {
		reporter.PassColor.Printf("✓ JSON report written to %s\n", path)
	}
}
