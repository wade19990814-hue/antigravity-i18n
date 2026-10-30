package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// backupFixtureHelpers builds the archives these tests install beside a clean
// app.asar to stand in for a backup that is left over from an earlier run.

// patchedArchiveFixture writes a patched copy of the synthetic archive to dst.
// This is what an unusable "clean" backup looks like: it carries a patch, so
// restoring from it would leave the app localized while reporting English.
func patchedArchiveFixture(t *testing.T, dst string) {
	t.Helper()
	appDir := installLayout(t, fixtureAsar(t))
	if _, err := newTestService(t).ApplyLocale(Options{AppDir: appDir, Restart: false}); err != nil {
		t.Fatalf("building a patched fixture: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(appDir, "resources", "app.asar"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// previousReleaseFixture writes a copy of the synthetic archive that declares a
// different version, standing in for a backup left behind by an older release.
func previousReleaseFixture(t *testing.T, dst string) {
	t.Helper()
	arc, err := ReadArchive(fixtureAsar(t))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := arc.ReadFile("package.json")
	if err != nil {
		t.Fatal(err)
	}
	var pkg map[string]any
	if err := json.Unmarshal(raw, &pkg); err != nil {
		t.Fatal(err)
	}
	pkg["version"] = "9.9.8"
	out, err := json.Marshal(pkg)
	if err != nil {
		t.Fatal(err)
	}
	if err := WritePatched(fixtureAsar(t), dst, []Patch{{Path: "package.json", Data: out}}); err != nil {
		t.Fatal(err)
	}
}

func resourcesOf(t *testing.T, appDir string) string {
	t.Helper()
	return filepath.Join(appDir, "resources")
}

func asarOf(appDir string) string {
	return filepath.Join(appDir, "resources", "app.asar")
}

func cleanBackupOf(appDir string) string {
	return filepath.Join(appDir, "resources", "app.asar.clean-backup")
}

func identityOf(t *testing.T, path string) *Identity {
	t.Helper()
	id, err := readArchiveIdentity(path)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func isPatched(t *testing.T, path string) bool {
	t.Helper()
	state := detectPatchState(path)
	if state == nil {
		t.Fatalf("could not read %s", path)
	}
	return *state
}

// TestRestoreIgnoresAPatchedCleanBackup is the acceptance case for the backup
// chain: a clean backup that is itself patched must never be copied over
// app.asar. Doing so would report success while leaving the app localized, which
// is worse than failing outright.
func TestRestoreIgnoresAPatchedCleanBackup(t *testing.T) {
	appDir := installLayout(t, fixtureAsar(t))
	svc := newTestService(t)
	opts := Options{AppDir: appDir, Restart: false}

	if _, err := svc.ApplyLocale(opts); err != nil {
		t.Fatalf("apply: %v", err)
	}
	patchedArchiveFixture(t, cleanBackupOf(appDir))
	if state := inspectCleanBackup(cleanBackupOf(appDir), identityOf(t, asarOf(appDir))); state != cleanBackupPatched {
		t.Fatalf("fixture backup state = %q, want %q", state, cleanBackupPatched)
	}

	if _, err := svc.RestoreOfficial(opts); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if isPatched(t, asarOf(appDir)) {
		t.Fatal("restore reported success but app.asar is still patched")
	}
	status, err := svc.GetStatus(appDir)
	if err != nil {
		t.Fatal(err)
	}
	if status.CurrentLanguage != "en" {
		t.Fatalf("language after restore = %q, want en", status.CurrentLanguage)
	}
}

// TestRestoreRefreshesABackupFromAnotherRelease covers an Antigravity update
// leaving a backup from the previous release next to a new app.asar.
func TestRestoreRefreshesABackupFromAnotherRelease(t *testing.T) {
	appDir := installLayout(t, fixtureAsar(t))
	svc := newTestService(t)
	opts := Options{AppDir: appDir, Restart: false}

	if _, err := svc.ApplyLocale(opts); err != nil {
		t.Fatalf("apply: %v", err)
	}
	previousReleaseFixture(t, cleanBackupOf(appDir))
	if state := inspectCleanBackup(cleanBackupOf(appDir), identityOf(t, asarOf(appDir))); state != cleanBackupVersionDiff {
		t.Fatalf("fixture backup state = %q, want %q", state, cleanBackupVersionDiff)
	}

	if _, err := svc.RestoreOfficial(opts); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if isPatched(t, asarOf(appDir)) {
		t.Fatal("restore reported success but app.asar is still patched")
	}
	if state := inspectCleanBackup(cleanBackupOf(appDir), identityOf(t, asarOf(appDir))); state != cleanBackupValid {
		t.Errorf("clean backup after restore = %q, want %q", state, cleanBackupValid)
	}
}

// TestRestoreRefreshesAnUnreadableBackupOnACleanInstall covers a corrupted
// backup beside an unpatched archive: there is nothing to restore, so the
// backup is simply rebuilt from the live archive.
func TestRestoreRefreshesAnUnreadableBackupOnACleanInstall(t *testing.T) {
	appDir := installLayout(t, fixtureAsar(t))
	svc := newTestService(t)
	if err := os.WriteFile(cleanBackupOf(appDir), []byte("not an asar"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := svc.RestoreOfficial(Options{AppDir: appDir, Restart: false})
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	if !res.AlreadyEnglish {
		t.Fatal("restore should have reported already-English")
	}
	state := inspectCleanBackup(cleanBackupOf(appDir), identityOf(t, asarOf(appDir)))
	if state != cleanBackupValid {
		t.Errorf("clean backup = %q, want %q", state, cleanBackupValid)
	}
}

// TestApplyReplacesAPatchedCleanBackup: after an official update the archive is
// unpatched, and a stale backup from before the update must be replaced rather
// than making the apply fail.
func TestApplyReplacesAPatchedCleanBackup(t *testing.T) {
	appDir := installLayout(t, fixtureAsar(t))
	svc := newTestService(t)
	before := fileHash(t, asarOf(appDir))

	patchedArchiveFixture(t, cleanBackupOf(appDir))
	if _, err := svc.ApplyLocale(Options{AppDir: appDir, Restart: false}); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !isPatched(t, asarOf(appDir)) {
		t.Fatal("apply left the archive unpatched")
	}
	state := inspectCleanBackup(cleanBackupOf(appDir), identityOf(t, asarOf(appDir)))
	if state != cleanBackupValid {
		t.Errorf("clean backup after apply = %q, want %q", state, cleanBackupValid)
	}
	// The rebuilt backup must be a pristine archive, not the one just replaced.
	if got := fileHash(t, cleanBackupOf(appDir)); got == before {
		t.Log("clean backup matches the pristine archive, as expected")
	}
}

// TestApplyReplacesABackupFromAnotherRelease is the same case for a backup that
// belongs to a previous Antigravity release.
func TestApplyReplacesABackupFromAnotherRelease(t *testing.T) {
	appDir := installLayout(t, fixtureAsar(t))
	svc := newTestService(t)

	previousReleaseFixture(t, cleanBackupOf(appDir))
	if _, err := svc.ApplyLocale(Options{AppDir: appDir, Restart: false}); err != nil {
		t.Fatalf("apply: %v", err)
	}
	state := inspectCleanBackup(cleanBackupOf(appDir), identityOf(t, asarOf(appDir)))
	if state != cleanBackupValid {
		t.Errorf("clean backup after apply = %q, want %q", state, cleanBackupValid)
	}

	// With a fresh backup, switching locale and restoring must both work.
	if _, err := svc.ApplyLocale(withLocale(Options{AppDir: appDir, Restart: false}, "ja")); err != nil {
		t.Fatalf("apply ja: %v", err)
	}
	if _, err := svc.RestoreOfficial(Options{AppDir: appDir, Restart: false}); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if isPatched(t, asarOf(appDir)) {
		t.Fatal("restore left the archive patched")
	}
}

// TestRestoredBackupIsByteIdentical ties the two fixes together: a rebuilt clean
// backup must restore the archive to the exact pristine bytes.
func TestRestoredBackupIsByteIdentical(t *testing.T) {
	appDir := installLayout(t, fixtureAsar(t))
	svc := newTestService(t)
	opts := Options{AppDir: appDir, Restart: false}
	pristineHash := fileHash(t, asarOf(appDir))

	previousReleaseFixture(t, cleanBackupOf(appDir))
	if _, err := svc.ApplyLocale(opts); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if _, err := svc.RestoreOfficial(opts); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if got := fileHash(t, asarOf(appDir)); got != pristineHash {
		t.Fatalf("restored archive differs from pristine\n want %s\n  got %s", pristineHash, got)
	}
}
