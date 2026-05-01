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
