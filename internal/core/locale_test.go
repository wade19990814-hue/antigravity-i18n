package core

import (
	"os"
	"strings"
	"testing"
)

func TestLocaleRegexFlags(t *testing.T) {
	for _, kind := range []string{"value rule", "pattern"} {
		for _, flags := range []string{"", "g", "y", "gi", "dimsu", "v", "x", "gg", "ii", "uv", "I", "i "} {
			t.Run(kind+"/"+flags, func(t *testing.T) {
				loc := &Locale{Language: "ja", Name: "Japanese", Text: map[string]string{"Hello": "こんにちは"}}
				valid := flags == "" || flags == "g" || flags == "y" || flags == "gi" || flags == "dimsu" || flags == "v"
				if kind == "value rule" {
					loc.ValueRules = map[string][]ValueRule{"duration": {{Pattern: `(\d+)d`, Template: "{1}日", Flags: flags}}}
				} else {
					loc.Patterns = []PatternRule{{Pattern: `^(\d+)d$`, Template: "{1}日", Sample: "2d", Flags: flags}}
					valid = valid && !strings.ContainsAny(flags, "gy")
				}
				err := validateLocale("ja", loc)
				if valid && err != nil {
					t.Fatalf("valid flags %q rejected: %v", flags, err)
				}
				if !valid && (err == nil || !strings.Contains(err.Error(), kind) || !strings.Contains(err.Error(), "flag")) {
					t.Fatalf("invalid flags %q: expected a contextual error, got %v", flags, err)
				}
			})
		}
	}
}

// TestAllLocalesValidate loads and validates every shipped locale through the
// same path the installer uses, so a malformed pack fails the build rather than
// a user's apply.
func TestAllLocalesValidate(t *testing.T) {
	assets := os.DirFS("../..")
	store, err := NewLocaleStore(assets, assets, "assets/locales", "assets/patches")
	if err != nil {
		t.Fatal(err)
	}
	list := store.ListLocales()
	if len(list) != 9 {
		t.Fatalf("expected 9 locales, got %d: %v", len(list), list)
	}
	for _, code := range list {
		loc, err := store.LoadLocale(code)
		if err != nil {
			t.Fatalf("locale %s: %v", code, err)
		}
		if len(loc.Text) == 0 {
			t.Fatalf("locale %s has no text", code)
		}
		t.Logf("OK %-8s name=%-14s entries=%d patterns=%d menu=%d dialogs=%d",
			code, loc.Name, len(loc.Text), len(loc.Patterns), len(loc.Menu), len(loc.Dialogs))
	}
}

// Fragments must come out as valid JavaScript: they are injected into the app's
// own scripts, and a syntax error there stops Antigravity from starting.
func TestFragmentsBuild(t *testing.T) {
	assets := os.DirFS("../..")
	store, err := NewLocaleStore(assets, assets, "assets/locales", "assets/patches")
	if err != nil {
		t.Fatal(err)
	}
	for _, code := range store.ListLocales() {
		loc, err := store.LoadLocale(code)
		if err != nil {
			t.Fatal(err)
		}
		engine, err := store.buildPreloadFragment(loc)
		if err != nil {
			t.Fatalf("%s: %v", code, err)
		}
		if len(engine) < 1000 {
			t.Fatalf("%s: engine fragment suspiciously small: %d", code, len(engine))
		}
		if err := checkJavaScriptSyntax("dist/preload.js", engine); err != nil {
			t.Fatalf("%s engine: %v", code, err)
		}
		menu, err := store.buildMenuFragment(loc)
		if err != nil {
			t.Fatalf("%s: %v", code, err)
		}
		if err := checkJavaScriptSyntax("dist/menu.js", menu); err != nil {
			t.Fatalf("%s menu: %v", code, err)
		}
		dialog, err := store.buildDialogFragment(loc)
		if err != nil {
			t.Fatalf("%s: %v", code, err)
		}
		if err := checkJavaScriptSyntax("dist/main.js", dialog); err != nil {
			t.Fatalf("%s dialog: %v", code, err)
		}
		t.Logf("OK %-8s engine=%d menu=%d dialog=%d", code, len(engine), len(menu), len(dialog))
	}
}

// TestLocaleSpellingsResolve covers the spellings people actually type. A
// rejected code is not a safety problem, but it is the first thing a new user
// hits, and the aliases the Node CLI accepted must keep working.
func TestLocaleSpellingsResolve(t *testing.T) {
	assets := os.DirFS("../..")
	store, err := NewLocaleStore(assets, assets, "assets/locales", "assets/patches")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ input, want string }{
		{"zh-CN", "zh-CN"},
		{"zh", "zh-CN"},
		{"ZH", "zh-CN"},
		{"zh-Hans", "zh-CN"},
		{"zh_CN", "zh-CN"},
		{"zh_CN.UTF-8", "zh-CN"},
		{"zh-Hant", "zh-Hant"},
		{"zh-TW", "zh-Hant"},
		{"zh_HK", "zh-Hant"},
		{"pt", "pt-BR"},
		{"pt-br", "pt-BR"},
		{"jp", "ja"},
		{"kr", "ko"},
		{"  ja  ", "ja"},
	} {
		loc, err := store.LoadLocale(tc.input)
		if err != nil {
			t.Errorf("LoadLocale(%q): %v", tc.input, err)
			continue
		}
		if loc.Language != tc.want {
			t.Errorf("LoadLocale(%q) resolved to %q, want %q", tc.input, loc.Language, tc.want)
		}
	}
	// A genuinely unknown code must still fail rather than fall back silently.
	if _, err := store.LoadLocale("not-a-locale"); err == nil {
		t.Fatal("accepted an unknown locale")
	}
}
