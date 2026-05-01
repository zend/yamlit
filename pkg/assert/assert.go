package assert

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"

	"github.com/zend/yamlit/pkg/step"
)

// Run executes all assertions against the HTTP response and returns results.
// Each assertion is evaluated independently; the caller decides AND/OR logic.
func Run(resp *http.Response, asserts []step.Assertion) []step.AssertResult {
	results := make([]step.AssertResult, 0, len(asserts))

	bodyBytes, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	bodyStr := string(bodyBytes)

	for _, a := range asserts {
		result := step.AssertResult{
			Type:     a.Type,
			Path:     a.Path,
			Expected: a.Expect,
			Passed:   true,
		}

		// Normalize JSONPath: strip leading "$." or "$" prefix if present
		// gjson uses "data.name" not "$.data.name"
		jsonPath := a.Path
		if strings.HasPrefix(jsonPath, "$.") {
			jsonPath = jsonPath[2:]
		} else if jsonPath == "$" {
			jsonPath = "@this"
		}

		switch a.Type {
		case "status_code":
			result.Actual = strconv.Itoa(resp.StatusCode)
			result.Passed = result.Actual == a.Expect

		case "jsonpath":
			actual := gjson.Get(bodyStr, jsonPath)
			result.Actual = actual.String()
			result.Passed = actual.Exists() && actual.String() == a.Expect
			if !actual.Exists() {
				result.Actual = "<path not found>"
			}

		case "body_match":
			result.Actual = bodyStr
			result.Passed = strings.Contains(bodyStr, a.Expect)

		case "body_equals":
			result.Actual = bodyStr
			result.Passed = strings.TrimSpace(bodyStr) == strings.TrimSpace(a.Expect)

		case "none":
			result.Passed = true

		default:
			result.Passed = false
			result.Actual = "unknown assertion type: " + a.Type
		}

		results = append(results, result)
	}

	return results
}
