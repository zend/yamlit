package extract

import (
	"io"
	"net/http"
	"strings"

	"github.com/tidwall/gjson"

	"github.com/zend/yamlit/pkg/step"
	"github.com/zend/yamlit/pkg/variable"
)

// Run extracts variables from the HTTP response according to the given items.
// Variables are stored in the provided pool.
func Run(resp *http.Response, items []step.ExtractItem, vars *variable.Pool) {
	bodyBytes, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	bodyStr := string(bodyBytes)

	for _, item := range items {
		switch item.Source {
		case "body":
			// Normalize JSONPath: strip leading "$." prefix if present
			path := item.Path
			if strings.HasPrefix(path, "$.") {
				path = path[2:]
			} else if path == "$" {
				path = "@this"
			}

			result := gjson.Get(bodyStr, path)
			if result.Exists() {
				vars.Set(item.VarName, result.String())
			}

		case "header":
			val := resp.Header.Get(item.Path)
			if val != "" {
				vars.Set(item.VarName, val)
			}
		}
	}
}
