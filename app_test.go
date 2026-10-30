package main

import (
	"os"
	"path/filepath"
	"testing"

	"antigravity-i18n/internal/core"
)

func TestSetAppDirPreservesValidSelectionOnError(t *testing.T) {
	store, err := core.NewLocaleStore(assetFS("assets"), assetFS("assets"), "locales", "patches")
	if err != nil {
		t.Fatal(err)
	}
	appDir := t.TempDir()
	data, err := os.ReadFile("testdata/synthetic-app.asar")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "app.asar"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	app := NewApp(core.NewService(store), appDir)
	if _, err := app.SetAppDir(t.TempDir()); err == nil {
		t.Fatal("accepted a directory without an archive")
	}
	st, err := app.GetStatus()
	if err != nil || st.AppDir != appDir {
		t.Fatalf("lost the valid installation after rejecting a path: %+v, %v", st, err)
	}
}
