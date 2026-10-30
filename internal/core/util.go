package core

import (
	"errors"
	"io"
	"io/fs"
	"os"
)

// explainWriteError turns the one failure users cannot diagnose from the raw
// text into an actionable message. Windows reports a write into a protected
// directory as the bare string "Access is denied.", which says nothing about
// what to do; an Antigravity installed under Program Files needs an elevated
// run, and the default %LOCALAPPDATA% install never does.
func explainWriteError(err error) error {
	if err == nil {
		return nil
	}
	if !errors.Is(err, fs.ErrPermission) {
		return err
	}
	return &permissionError{Cause: err}
}

// permissionError adds guidance without discarding the original text, so the
// real path and Win32 wording stay available for a bug report.
type permissionError struct {
	Cause error
}

func (e *permissionError) Error() string {
	return e.Cause.Error() + "\n\n" +
		"Windows refused the write. Antigravity is installed in a protected location " +
		"(for example Program Files), so run this tool as administrator and try again. " +
		"An installation under %LOCALAPPDATA% does not need this."
}

func (e *permissionError) Unwrap() error { return e.Cause }

func openFile(path string) (*os.File, error) {
	return os.Open(path)
}

func containsAny(haystack []byte, needles []string) bool {
	if len(haystack) == 0 {
		return false
	}
	for _, n := range needles {
		if indexBytes(haystack, []byte(n)) >= 0 {
			return true
		}
	}
	return false
}

func indexBytes(haystack, needle []byte) int {
	n := len(needle)
	if n == 0 || len(haystack) < n {
		return -1
	}
	first := needle[0]
	for i := 0; i+n <= len(haystack); i++ {
		if haystack[i] != first {
			continue
		}
		if string(haystack[i:i+n]) == string(needle) {
			return i
		}
	}
	return -1
}

// replaceFirst replaces the first occurrence of old with new, returning the
// result and how many replacements were made.
func replaceFirst(s, old, new string) (string, int) {
	if old == "" {
		return s, 0
	}
	i := indexBytes([]byte(s), []byte(old))
	if i < 0 {
		return s, 0
	}
	return s[:i] + new + s[i+len(old):], 1
}

var _ io.Reader = (*os.File)(nil)
