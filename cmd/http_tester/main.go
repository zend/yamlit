package main

import (
	"flag"
	"fmt"
	"os"
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
		fmt.Fprintf(os.Stderr, "Usage: %s <file.yaml|directory|pattern> [-v] [-o report.json]\n", os.Args[0])
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
	allReports := make([]*runner.Report, 0, len(files))
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

		allReports = append(allReports, report)

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
		writeJSONReport(*outputFile, allReports)
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
	var files []string
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
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

func writeJSONReport(path string, reports []*runner.Report) {
	// TODO: implement JSON report output
	// For now, this is a placeholder
}
