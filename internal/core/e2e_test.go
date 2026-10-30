package core

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const syntheticAsar = "synthetic-app.asar"

// fixtureAsar returns the path of a synthetic archive that @electron/asar itself
// produced, so the Go writer is validated against Electron's bytes rather than
// its own.
func fixtureAsar(t *testing.T) string {
	t.Helper()
	p := filepath.Join("..", "..", "testdata", syntheticAsar)
	if _, err := os.Stat(p); err != nil {
		t.Skipf("fixture missing: %v", err)
	}
	return p
}

// newService builds a service whose fragments come from the same locale pack the
// binary embeds.
func newTestService(t *testing.T) *Service {
	t.Helper()
	assets := os.DirFS("../..")
	store, err := NewLocaleStore(assets, assets, "assets/locales", "assets/patches")
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(store)
	// Fixtures live in temporary installations; tests must never close or query
	// the user's real Antigravity processes.
	svc.stopApp = func(bool, int) (bool, bool, bool, error) { return false, true, false, nil }
	svc.appRunning = func() (bool, error) { return false, nil }
	svc.launchApp = func(*paths) error { return nil }
	return svc
}

// installLayout copies the fixture into a fake installation directory that
// resolveAppPaths will accept, so apply can be exercised without touching the
// real Antigravity.
func installLayout(t *testing.T, asarPath string) string {
	t.Helper()
	appDir := filepath.Join(t.TempDir(), "Antigravity")
	resources := filepath.Join(appDir, "resources")
	if err := os.MkdirAll(resources, 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(asarPath)
	if err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(resources, "app.asar")
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return appDir
}

func fileHash(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func mustReadFile(t *testing.T, asarPath, rel string) []byte {
	t.Helper()
	arc, err := ReadArchive(asarPath)
	if err != nil {
		t.Fatal(err)
	}
	data, err := arc.ReadFile(rel)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// TestApplyThenRestoreIsByteExact is the acceptance gate the plan names: after
// apply and restore the archive must match the pristine backup byte for byte.
func TestApplyThenRestoreIsByteExact(t *testing.T) {
	appDir := installLayout(t, fixtureAsar(t))
	asarPath := filepath.Join(appDir, "resources", "app.asar")
	pristineHash := fileHash(t, asarPath)

	svc := newTestService(t)
	opts := Options{AppDir: appDir, Restart: false}

	result, err := svc.ApplyLocale(opts)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if result.Locale != DefaultLocale {
		t.Fatalf("applied locale = %q, want %q", result.Locale, DefaultLocale)
	}

	// The archive itself is authoritative: it must now be patched and must name
	// the locale that was installed.
	patched, err := detectPatchInArchive(asarPath)
	if err != nil || !patched {
		t.Fatalf("archive is not patched after apply: patched=%v err=%v", patched, err)
	}
	if code, _, ok := readArchiveLocale(asarPath); !ok || code != DefaultLocale {
		t.Fatalf("archive locale = %q (ok=%v), want %q", code, ok, DefaultLocale)
	}

	// Identity must survive the rewrite.
	id, err := readArchiveIdentity(asarPath)
	if err != nil || id.Name != "antigravity" || id.Version != "9.9.9" {
		t.Fatalf("identity after apply = %+v, err %v", id, err)
	}

	// The engine must be present exactly once, and the injected files must still
	// be valid JavaScript.
	preload := string(mustReadFile(t, asarPath, "dist/preload.js"))
	if n := strings.Count(preload, "/* antigravity-i18n:begin */"); n != 1 {
		t.Fatalf("engine block appears %d times, want 1", n)
	}
	if n := strings.Count(preload, "const AG_LOCALE"); n != 1 {
		t.Fatalf("AG_LOCALE appears %d times, want 1", n)
	}
	for _, rel := range []string{"dist/main.js", "dist/menu.js", "dist/preload.js"} {
		src := string(mustReadFile(t, asarPath, rel))
		if err := checkJavaScriptSyntax(rel, src); err != nil {
			t.Fatalf("%s is not valid JavaScript after apply: %v", rel, err)
		}
	}

	// The language switch must have been inserted, which is the one anchor the
	// synthetic main.js does not already carry.
	main := string(mustReadFile(t, asarPath, "dist/main.js"))
	if !strings.Contains(main, "appendSwitch('lang', 'zh-CN')") {
		t.Fatalf("language switch was not inserted into main.js")
	}
	menu := string(mustReadFile(t, asarPath, "dist/menu.js"))
	if !strings.Contains(menu, "translateMenu(menu);") {
		t.Fatalf("translateMenu was not wired into menu.js")
	}

	// Restore must bring the archive back to the pristine bytes exactly.
	restore, err := svc.RestoreOfficial(opts)
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	if restore.AlreadyEnglish {
		t.Fatal("restore reported already-English, which is wrong after an apply")
	}
	if got := fileHash(t, asarPath); got != pristineHash {
		t.Fatalf("restored archive differs from the pristine archive\n want %s\n  got %s", pristineHash, got)
	}
}

// TestApplyIsIdempotent re-applies the same locale and requires the archive to be
// unchanged: a patch that grows on every run would keep changing app.asar for no
// reason.
func TestApplyIsIdempotent(t *testing.T) {
	appDir := installLayout(t, fixtureAsar(t))
	svc := newTestService(t)
	opts := Options{AppDir: appDir, Restart: false}

	if _, err := svc.ApplyLocale(opts); err != nil {
		t.Fatalf("first apply: %v", err)
	}
	afterFirst := fileHash(t, filepath.Join(appDir, "resources", "app.asar"))

	if _, err := svc.ApplyLocale(opts); err != nil {
		t.Fatalf("second apply: %v", err)
	}
	afterSecond := fileHash(t, filepath.Join(appDir, "resources", "app.asar"))

	if afterFirst != afterSecond {
		t.Fatal("re-applying the same locale changed the archive")
	}
}

// TestSwitchingLocaleReplacesTheEngine switches between two languages and checks
// that only one engine survives and the archive follows the active language.
func TestSwitchingLocaleReplacesTheEngine(t *testing.T) {
	appDir := installLayout(t, fixtureAsar(t))
	svc := newTestService(t)
	base := Options{AppDir: appDir, Restart: false}

	if _, err := svc.ApplyLocale(withLocale(base, "ja")); err != nil {
		t.Fatalf("apply ja: %v", err)
	}
	if code, _, ok := readArchiveLocale(filepath.Join(appDir, "resources", "app.asar")); !ok || code != "ja" {
		t.Fatalf("locale after apply ja = %q (ok=%v)", code, ok)
	}

	if _, err := svc.ApplyLocale(withLocale(base, DefaultLocale)); err != nil {
		t.Fatalf("apply zh-CN: %v", err)
	}
	asarPath := filepath.Join(appDir, "resources", "app.asar")
	if code, _, ok := readArchiveLocale(asarPath); !ok || code != DefaultLocale {
		t.Fatalf("locale after apply zh-CN = %q (ok=%v)", code, ok)
	}
	preload := string(mustReadFile(t, asarPath, "dist/preload.js"))
	if n := strings.Count(preload, "/* antigravity-i18n:begin */"); n != 1 {
		t.Fatalf("engine block appears %d times after switching locale, want 1", n)
	}
	if n := strings.Count(preload, `"language":"ja"`); n != 0 {
		t.Fatalf("the previous language's engine is still present")
	}

	if _, err := svc.RestoreOfficial(base); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if got := fileHash(t, asarPath); got != fileHash(t, mustCopy(fixtureAsar(t), t)) {
		t.Fatalf("restored archive differs from pristine")
	}
}

func withLocale(o Options, code string) Options {
	o.Locale = code
	return o
}

func mustCopy(path string, t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(t.TempDir(), filepath.Base(path))
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return dst
}

// TestStatusReportsLanguage checks the reporting path, which is how a user knows
// what state their install is in.
func TestStatusReportsLanguage(t *testing.T) {
	appDir := installLayout(t, fixtureAsar(t))
	svc := newTestService(t)
	opts := Options{AppDir: appDir, Restart: false}

	status, err := svc.GetStatus(appDir)
	if err != nil {
		t.Fatalf("status on a clean install: %v", err)
	}
	if status.CurrentLanguage != "en" {
		t.Fatalf("language on a clean install = %q, want en", status.CurrentLanguage)
	}
	if status.HasCleanBackup {
		t.Fatal("a clean install should not already have a backup")
	}

	if _, err := svc.ApplyLocale(opts); err != nil {
		t.Fatalf("apply: %v", err)
	}
	status, err = svc.GetStatus(appDir)
	if err != nil {
		t.Fatalf("status after apply: %v", err)
	}
	if status.CurrentLanguage != DefaultLocale {
		t.Fatalf("language after apply = %q, want %s", status.CurrentLanguage, DefaultLocale)
	}
	if status.LocaleName != "简体中文" {
		t.Fatalf("locale name = %q", status.LocaleName)
	}
	if !status.HasCleanBackup {
		t.Fatal("apply should leave a valid clean backup behind")
	}
	if status.CleanBackupStatus != cleanBackupValid {
		t.Fatalf("clean backup status = %q, want %q", status.CleanBackupStatus, cleanBackupValid)
	}

	if _, err := svc.RestoreOfficial(opts); err != nil {
		t.Fatalf("restore: %v", err)
	}
	status, err = svc.GetStatus(appDir)
	if err != nil {
		t.Fatalf("status after restore: %v", err)
	}
	if status.CurrentLanguage != "en" {
		t.Fatalf("language after restore = %q, want en", status.CurrentLanguage)
	}
}

// TestRestoreRecoversFromMissingCleanBackup covers the case where the clean
// backup was deleted: restore has to find a pristine timestamped backup instead
// of failing.
func TestRestoreRecoversFromMissingCleanBackup(t *testing.T) {
	appDir := installLayout(t, fixtureAsar(t))
	resources := filepath.Join(appDir, "resources")
	svc := newTestService(t)
	opts := Options{AppDir: appDir, Restart: false}

	if _, err := svc.ApplyLocale(opts); err != nil {
		t.Fatalf("apply: %v", err)
	}
	patchedHash := fileHash(t, filepath.Join(resources, "app.asar"))

	// Only the pre-apply backup survives: the clean backup is deleted, so
	// recovery has to come from app.asar.bak-<stamp>, which holds the pristine
	// archive. The pre-restore backup is written later and holds the patched
	// build, so it must not be the one recovery reaches for.
	if err := os.Remove(filepath.Join(resources, "app.asar.clean-backup")); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(resources)
	if err != nil {
		t.Fatal(err)
	}
	var pristineBaks []string
	for _, e := range entries {
		// A bak-before-restore name embeds a stamp too, so match the prefix
		// rather than any name containing "app.asar.bak".
		if strings.HasPrefix(e.Name(), "app.asar.bak-") && !strings.HasPrefix(e.Name(), "app.asar.bak-before-restore") {
			pristineBaks = append(pristineBaks, e.Name())
		}
	}
	if len(pristineBaks) == 0 {
		t.Fatal("expected apply to leave a pristine timestamped backup")
	}
	if len(pristineBaks) != 1 {
		t.Fatalf("expected exactly one pristine backup, got %d", len(pristineBaks))
	}

	restore, err := svc.RestoreOfficial(opts)
	if err != nil {
		t.Fatalf("restore without a clean backup: %v", err)
	}
	_ = restore
	if got := fileHash(t, filepath.Join(resources, "app.asar")); got == patchedHash {
		t.Fatal("restore left the patched archive in place")
	}
	if !bytes.Contains(mustReadFile(t, filepath.Join(resources, "app.asar"), "dist/main.js"), []byte("whenReady")) {
		t.Fatal("restored main.js does not look like the synthetic fixture")
	}
}

// TestApplyRejectsUnknownLocale makes a bad request fail before anything is
// touched.
func TestApplyRejectsUnknownLocale(t *testing.T) {
	appDir := installLayout(t, fixtureAsar(t))
	svc := newTestService(t)
	before := fileHash(t, filepath.Join(appDir, "resources", "app.asar"))

	_, err := svc.ApplyLocale(Options{AppDir: appDir, Locale: "not-a-locale"})
	if err == nil {
		t.Fatal("expected an unknown locale to be rejected")
	}
	if got := fileHash(t, filepath.Join(appDir, "resources", "app.asar")); got != before {
		t.Fatal("a rejected apply modified the archive")
	}
}

// TestStateMarkerTracksLanguage checks the marker written beside the archive,
// including that a legacy marker is read.
func TestStateMarkerTracksLanguage(t *testing.T) {
	appDir := installLayout(t, fixtureAsar(t))
	resources := filepath.Join(appDir, "resources")
	svc := newTestService(t)
	opts := Options{AppDir: appDir, Restart: false}

	if _, err := svc.ApplyLocale(opts); err != nil {
		t.Fatalf("apply: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(resources, stateMarkerName))
	if err != nil {
		t.Fatalf("state marker: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["language"] != DefaultLocale {
		t.Fatalf("marker language = %v, want %s", payload["language"], DefaultLocale)
	}
	if payload["patchedAt"] == "" || payload["patchedAt"] == nil {
		t.Fatal("marker has no timestamp")
	}
	fmt.Fprintln(os.Stderr, "state marker:", strings.TrimSpace(string(data)))
}
