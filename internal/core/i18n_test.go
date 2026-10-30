package core

import (
	"os"
	"testing"
)

func newTestServiceForLocales(t *testing.T) *Service {
	t.Helper()
	assets := os.DirFS("../..")
	store, err := NewLocaleStore(assets, assets, "assets/locales", "assets/patches")
	if err != nil {
		t.Fatal(err)
	}
	return NewService(store)
}

// English is not a language pack but the official untranslated build, so the
// picker has to list it first and label it as needing no translations. Without
// it the GUI cannot express the one language every installation ships with.
func TestDescribeLocalesLeadsWithEnglish(t *testing.T) {
	svc := newTestServiceForLocales(t)
	locales, err := svc.DescribeLocales()
	if err != nil {
		t.Fatal(err)
	}
	if len(locales) < 10 {
		t.Fatalf("expected the English entry plus the bundled packs, got %d", len(locales))
	}
	en := locales[0]
	if en.Code != EnglishLocale {
		t.Fatalf("expected the first entry to be %q, got %q", EnglishLocale, en.Code)
	}
	if en.Name != "English" || en.EnglishName != "English" {
		t.Fatalf("the English entry is mislabelled: %+v", en)
	}
	if en.Translations != 0 || en.Coverage != 100 {
		t.Fatalf("English translates nothing yet is complete: %+v", en)
	}
}

// A progress sink that receives nothing would leave the GUI on a hardcoded
// sentence for the whole rewrite, so the stage name alone is asserted: what
// matters is that work is reported at all, not every detail string.
func TestApplyReportsProgress(t *testing.T) {
	svc := newTestService(t)
	var stages []string
	svc.Progress = func(stage, detail string) {
		stages = append(stages, stage)
	}

	if _, err := svc.ApplyLocale(Options{AppDir: installLayout(t, fixtureAsar(t)), Locale: "zh-CN", Restart: false}); err != nil {
		t.Fatal(err)
	}
	if len(stages) == 0 {
		t.Fatal("apply produced no progress at all")
	}
	for _, want := range []string{"正在校验语言包", "正在准备干净备份", "正在写回 app.asar"} {
		if !containsStage(stages, want) {
			t.Fatalf("apply never reported %q; got %v", want, stages)
		}
	}
}

func containsStage(stages []string, want string) bool {
	for _, s := range stages {
		if s == want {
			return true
		}
	}
	return false
}
