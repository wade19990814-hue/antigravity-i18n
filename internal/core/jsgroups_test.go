package core

import (
	"os"
	"strings"
	"testing"
)

// TestJsCaptureGroups pins the count to what JavaScript itself reports for each
// construct. The validator uses the count to reject a template that references a
// capture group that does not exist, so counting a non-capturing group as one
// would let a broken {n} reference through as valid.
func TestJsCaptureGroups(t *testing.T) {
	cases := []struct {
		pattern string
		want    int
	}{
		{"(a)", 1},
		{"((a)(b))", 3},
		{"(?:a)", 0},
		{"(?:a)(b)", 1},
		{"(a)(?:b)(c)", 2},
		{"(?:", 0},
		{"(?=a)", 0},
		{"(?!a)", 0},
		{"(?<=a)b", 0},
		{"(?<!a)b", 0},
		{"[(]", 0},   // inside a character class
		{"\\(a)", 0}, // escaped parenthesis
		{"(?<name>a)", 1},
		{"(?<name>a)(?:b)", 1},
		{"(?<a>a)", 1},
		{"(?<=a)(?<!b)", 0},

		// The wrapping group the shipped locale patterns use is non-capturing.
		{`^Thought for (\d+(?:\.\d+)?)(s|m|h|ms)$`, 2},
		{`\b(\d*(?:11|12|13|14)) days?`, 1},
	}
	for _, c := range cases {
		if got := jsCaptureGroups(c.pattern); got != c.want {
			t.Errorf("jsCaptureGroups(%q) = %d, want %d", c.pattern, got, c.want)
		}
	}
}

// TestJsCaptureGroupsMatchShippedPatterns checks every pattern the binary ships
// against a counter written independently of the production scanner, so the two
// agreeing is evidence rather than the same code restated.
func TestJsCaptureGroupsMatchShippedPatterns(t *testing.T) {
	store, err := NewLocaleStore(os.DirFS("../.."), os.DirFS("../.."), "assets/locales", "assets/patches")
	if err != nil {
		t.Fatal(err)
	}
	compared := 0
	for _, code := range store.ListLocales() {
		loc, err := store.LoadLocale(code)
		if err != nil {
			t.Fatalf("loading %s: %v", code, err)
		}
		for i := range loc.Patterns {
			rule := &loc.Patterns[i]
			assertGroupsAgree(t, code, rule.ID, rule.Pattern)
			// A template may only reference groups the pattern actually has.
			want := referenceJsGroups(rule.Pattern)
			if rule.Template != "" {
				if err := validateTemplateGroups(code, rule.ID, rule.Template, want); err != nil {
					t.Errorf("%s/%s: %v", code, rule.ID, err)
				}
			}
			for category, template := range rule.Templates {
				if err := validateTemplateGroups(code, rule.ID, category+" "+template, want); err != nil {
					t.Errorf("%s/%s: %v", code, rule.ID, err)
				}
			}
		}
		for name, rules := range loc.ValueRules {
			for _, rule := range rules {
				assertGroupsAgree(t, code, name, rule.Pattern)
			}
		}
		compared++
	}
	if compared == 0 {
		t.Fatal("no locale packs were compared")
	}
}

// assertGroupsAgree fails when the production counter and the independent
// reference counter disagree about a pattern.
func assertGroupsAgree(t *testing.T, code, label, pattern string) {
	t.Helper()
	if pattern == "" {
		return
	}
	want := referenceJsGroups(pattern)
	got, err := countCaptureGroups(pattern)
	if err != nil {
		t.Fatalf("%s/%s %q: %v", code, label, pattern, err)
	}
	if got != want {
		t.Errorf("%s/%s %q: counted %d, JavaScript counts %d", code, label, pattern, got, want)
	}
}

// referenceJsGroups counts capturing groups the way JavaScript's regular
// expression engine does.
//
// The constructs in the "(?..." family are non-capturing except a named group,
// which has to be told apart from the two lookbehinds by the character that
// follows "("<".
func referenceJsGroups(pattern string) int {
	count := 0
	inClass := false
	for i := 0; i < len(pattern); i++ {
		c := pattern[i]
		if inClass {
			if c == ']' {
				inClass = false
			}
			continue
		}
		if c == '\\' {
			i++ // the next byte is escaped
			continue
		}
		switch c {
		case '[':
			inClass = true
			if i+1 < len(pattern) && pattern[i+1] == '^' {
				i++
			}
			if i+1 < len(pattern) && pattern[i+1] == ']' {
				i++ // a ']' in first position closes nothing
			}
		case '(':
			if opensCaptureGroup(pattern[i:]) {
				count++
			}
		}
	}
	return count
}

// opensCaptureGroup reports whether the group beginning at s captures in
// JavaScript.
func opensCaptureGroup(s string) bool {
	if !strings.HasPrefix(s, "(") {
		return false
	}
	if !strings.HasPrefix(s, "(?") {
		return len(s) > 1 // a plain "(x)" captures
	}
	name := strings.TrimPrefix(s, "(?<")
	if name == s {
		return false // (?: (?= (?! (?<= (?<! never capture
	}
	return !(len(name) > 0 && (name[0] == '=' || name[0] == '!'))
}
