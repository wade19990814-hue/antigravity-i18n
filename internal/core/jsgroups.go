package core

import "strings"

// jsCaptureGroups counts capturing groups in a JavaScript regex body.
//
// JavaScript's own trick is to append an empty alternative and read the match
// length, which Go's RE2 cannot reproduce. So the body is scanned directly: a
// '(' opens a group unless it starts a non-capturing construct, and a '(' inside
// a character class or preceded by a backslash never opens one.
func jsCaptureGroups(pattern string) int {
	count := 0
	inClass := false

	for i := 0; i < len(pattern); i++ {
		c := pattern[i]

		if c == '\\' {
			i++ // skip the escaped byte
			continue
		}

		if inClass {
			if c == ']' {
				inClass = false
			}
			continue
		}

		switch c {
		case '[':
			inClass = true
			// A ']' in the first position is a literal, not the class close.
			if i+1 < len(pattern) && pattern[i+1] == '^' {
				i++
			}
			if i+1 < len(pattern) && pattern[i+1] == ']' {
				i++
			}
		case '(':
			if isCapturingGroup(pattern[i:]) {
				count++
			}
		}
	}
	return count
}

// isCapturingGroup reports whether the '(' at the start of s opens a capturing
// group. Named groups in the (?<name>...) form capture; the (?<= and (?<!
// lookbehinds do not, which is why the character following '?' decides.
func isCapturingGroup(s string) bool {
	if !strings.HasPrefix(s, "(") {
		return false
	}
	rest := s[1:]
	if rest == "" || rest[0] != '?' {
		// A plain group captures.
		return true
	}
	if len(rest) < 2 {
		// "(?" cannot open a group on its own.
		return false
	}
	if rest[1] != '<' {
		// (?:...), (?=...) and (?!...) never capture. Only the (?<name>...)
		// form does, so anything else after "(?" is non-capturing.
		return false
	}
	if len(rest) >= 3 && (rest[2] == '=' || rest[2] == '!') {
		// (?<=... and (?<!...) are lookbehinds, which do not capture.
		return false
	}
	// (?<name>...) is a named capturing group in JavaScript.
	return true
}
