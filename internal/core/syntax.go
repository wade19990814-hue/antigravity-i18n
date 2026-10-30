package core

import (
	"fmt"
	"path/filepath"

	"github.com/dop251/goja/parser"
)

// checkJavaScriptSyntax parses a generated fragment before it is written into
// the archive. A syntax error would leave the app unable to start, so it is
// caught here rather than at the next launch.
func checkJavaScriptSyntax(path, source string) error {
	_, err := parser.ParseFile(nil, path, source, 0)
	if err != nil {
		return fmt.Errorf("JavaScript syntax check failed for %s: %w", filepath.Base(path), err)
	}
	return nil
}
