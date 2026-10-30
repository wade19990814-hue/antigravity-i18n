package main

import (
	"strings"
	"testing"
)

// parseArgs is tested rather than runCLI: the parser is where the shorthand and
// conflict rules live, and running a command would stop a real Antigravity
// instance and rewrite a real archive.
func TestParseArgsShorthands(t *testing.T) {
	for _, name := range []string{"zh", "cn", "chinese", "ZH", "Chinese", "CHINESE"} {
		parsed, err := parseArgs([]string{name, "--app-dir", `C:\AG`, "--no-restart", "--yes"})
		if err != nil {
			t.Errorf("%q was rejected: %v", name, err)
			continue
		}
		if parsed.cmd != "apply" {
			t.Errorf("%q produced command %q, want apply", name, parsed.cmd)
		}
		if parsed.opts.Locale != "zh-CN" {
			t.Errorf("%q produced locale %q, want zh-CN", name, parsed.opts.Locale)
		}
		if !parsed.confirm || parsed.opts.Restart {
			t.Errorf("%q did not carry the flags through: %+v", name, parsed)
		}
	}
	for _, name := range []string{"en", "english", "official", "EN"} {
		parsed, err := parseArgs([]string{name, "--app-dir", `C:\AG`, "--yes"})
		if err != nil {
			t.Errorf("%q was rejected: %v", name, err)
			continue
		}
		if parsed.cmd != "restore" {
			t.Errorf("%q produced command %q, want restore", name, parsed.cmd)
		}
	}
}

// A shorthand already names its locale, so a different one alongside it is
// ambiguous and must be refused rather than silently resolved.
func TestParseArgsShorthandLocaleConflicts(t *testing.T) {
	for _, tc := range []struct {
		args []string
	}{
		{[]string{"zh", "--locale", "ja"}},
		{[]string{"zh", "--locale=ja"}},
		{[]string{"en", "--locale", "ja"}},
		{[]string{"english", "--locale", "zh-CN"}},
	} {
		if _, err := parseArgs(tc.args); err == nil {
			t.Errorf("%v should have been refused", tc.args)
		}
	}
	// The same locale spelled differently is not a conflict.
	for _, args := range [][]string{
		{"zh", "--locale", "zh-CN"},
		{"zh", "--locale", "zh_CN"},
		{"zh", "--locale", "ZH"},
		{"en", "--locale", "en"},
	} {
		if _, err := parseArgs(args); err != nil {
			t.Errorf("%v was refused but should not be: %v", args, err)
		}
	}
}

func TestParseArgsBasics(t *testing.T) {
	parsed, err := parseArgs([]string{"apply", "--locale", "ja", "--app-dir", `C:\AG`, "--timeout", "5000", "--force"})
	if err != nil {
		t.Fatal(err)
	}
	if parsed.cmd != "apply" || parsed.opts.Locale != "ja" || parsed.opts.AppDir != `C:\AG` ||
		parsed.opts.TimeoutMs != 5000 || !parsed.opts.Force {
		t.Fatalf("parsed %+v", parsed)
	}
	// A flag before the command name is still handled.
	if p, err := parseArgs([]string{"-v"}); err != nil || p.cmd != "version" {
		t.Fatalf("-v parsed as %+v, %v", p, err)
	}
	if p, err := parseArgs([]string{"--help"}); err != nil || p.cmd != "help" {
		t.Fatalf("--help parsed as %+v, %v", p, err)
	}
	// Unknown commands and options are still errors.
	for _, args := range [][]string{{"nonsense"}, {"apply", "--bogus"}, {"apply", "extra"}} {
		if _, err := parseArgs(args); err == nil {
			t.Errorf("%v should have been refused", args)
		}
	}
}

func TestParseArgsTimeout(t *testing.T) {
	for _, command := range []string{"apply", "restore"} {
		for _, value := range []string{"-5000", "0", "invalid", "999999999999999999999999", "1", "5000"} {
			for _, equals := range []bool{false, true} {
				args := []string{command, "--timeout", value}
				if equals {
					args = []string{command, "--timeout=" + value}
				}
				parsed, err := parseArgs(args)
				if value == "1" || value == "5000" {
					if err != nil || parsed.opts.TimeoutMs <= 0 {
						t.Errorf("%v: got %+v, %v", args, parsed, err)
					}
				} else if err == nil || !strings.Contains(err.Error(), "--timeout") {
					t.Errorf("%v: expected a timeout error, got %v", args, err)
				}
			}
		}
	}
}

// The help text is the only place the shorthands are documented, so it has to
// mention them.
func TestUsageDocumentsShorthands(t *testing.T) {
	for _, want := range []string{"zh", "cn", "chinese", "en", "official", "Shorthands"} {
		if !strings.Contains(usage, want) {
			t.Errorf("usage does not mention %q", want)
		}
	}
}
