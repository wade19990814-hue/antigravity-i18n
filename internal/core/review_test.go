package core

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"
)

func TestReadArchiveRejectsInvalidEntries(t *testing.T) {
	for _, node := range []string{
		`null`,
		`{"size":-1,"offset":"0"}`,
		`{"size":1,"offset":"-1"}`,
		`{"size":9223372036854775807,"offset":"0"}`,
		`{"size":1,"offset":"9223372036854775807"}`,
		`{"size":2,"offset":"0"}`,
		`{"size":1}`,
	} {
		t.Run(node, func(t *testing.T) {
			header := pickleWrapJSON([]byte(fmt.Sprintf(`{"files":{"bad":%s}}`, node)))
			var lead [8]byte
			binary.LittleEndian.PutUint32(lead[:4], 4)
			binary.LittleEndian.PutUint32(lead[4:], uint32(len(header)))
			data := append(lead[:], header...)
			data = append(data, 'x')
			path := filepath.Join(t.TempDir(), "invalid.asar")
			if err := os.WriteFile(path, data, 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := ReadArchive(path); err == nil {
				t.Fatal("accepted an invalid entry")
			}
		})
	}
}

func TestWritePatchedStreamsFixture(t *testing.T) {
	src := fixtureAsar(t)
	dst := filepath.Join(t.TempDir(), "patched.asar")
	if err := WritePatched(src, dst, nil); err != nil {
		t.Fatal(err)
	}
	if fileHash(t, src) != fileHash(t, dst) {
		t.Fatal("an unchanged rewrite must be byte exact")
	}
	patch := Patch{Path: "dist/main.js", Data: []byte("// replacement\n")}
	if err := WritePatched(src, dst, []Patch{patch}); err != nil {
		t.Fatal(err)
	}
	before, err := ReadArchive(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, rel := range before.FileEntries() {
		want := mustReadFile(t, src, rel)
		if rel == patch.Path {
			want = patch.Data
		}
		if !bytes.Equal(mustReadFile(t, dst, rel), want) {
			t.Fatalf("incorrect content in %s", rel)
		}
	}
}

func TestWritePatchedWithSharedEmptyOffsets(t *testing.T) {
	header := pickleWrapJSON([]byte(`{"files":{"body":{"size":1,"offset":"0"},"empty-a":{"size":0,"offset":"0"},"empty-z":{"size":0,"offset":"0"}}}`))
	var lead [8]byte
	binary.LittleEndian.PutUint32(lead[:4], 4)
	binary.LittleEndian.PutUint32(lead[4:], uint32(len(header)))
	data := append(lead[:], header...)
	data = append(data, 'x')
	src := filepath.Join(t.TempDir(), "shared-offsets.asar")
	if err := os.WriteFile(src, data, 0o644); err != nil {
		t.Fatal(err)
	}
	arc, err := ReadArchive(src)
	if err != nil {
		t.Fatal(err)
	}
	// Empty entries must precede the body at the same offset, or a rewrite
	// moves their offsets and changes the header even when no patch is applied.
	if got := strings.Join(arc.FileEntries(), ","); got != "empty-a,empty-z,body" {
		t.Errorf("unstable shared-offset ordering: %s", got)
	}
	dst := filepath.Join(t.TempDir(), "rewritten.asar")
	if err := WritePatched(src, dst, nil); err != nil {
		t.Fatal(err)
	}
	if fileHash(t, src) != fileHash(t, dst) {
		t.Fatal("unchanged rewrite moved empty-file offsets")
	}
}

func TestWritePatchedRejectsSameFile(t *testing.T) {
	appDir := installLayout(t, fixtureAsar(t))
	path := asarOf(appDir)
	want := fileHash(t, path)
	if err := WritePatched(path, path, nil); err == nil {
		t.Fatal("accepted a rewrite over the source file")
	}
	if fileHash(t, path) != want {
		t.Fatal("source was modified")
	}
}

func TestHashReaderPropagatesReadFailure(t *testing.T) {
	want := errors.New("disk read failed")
	r := io.MultiReader(strings.NewReader("partial content"), iotest.ErrReader(want))
	if sum, err := hashReader(r); sum != "" || !errors.Is(err, want) {
		t.Fatalf("hashReader returned %q, %v; expected the read error", sum, err)
	}
}

func TestOperationsPreserveConcurrentArchiveUpdate(t *testing.T) {
	for _, restoring := range []bool{false, true} {
		t.Run(fmt.Sprintf("restore=%v", restoring), func(t *testing.T) {
			appDir := installLayout(t, fixtureAsar(t))
			path := asarOf(appDir)
			updated := filepath.Join(t.TempDir(), "updated.asar")
			previousReleaseFixture(t, updated)
			data, err := os.ReadFile(updated)
			if err != nil {
				t.Fatal(err)
			}
			svc := newTestService(t)
			if restoring {
				if _, err := svc.ApplyLocale(Options{AppDir: appDir}); err != nil {
					t.Fatal(err)
				}
			}
			svc.Progress = func(stage, _ string) {
				if stage == "正在写回 app.asar" {
					if err := os.WriteFile(path, data, 0o644); err != nil {
						t.Fatal(err)
					}
				}
			}
			if restoring {
				_, err = svc.RestoreOfficial(Options{AppDir: appDir})
			} else {
				_, err = svc.ApplyLocale(Options{AppDir: appDir})
			}
			if err == nil {
				t.Fatal("accepted an archive changed by another process")
			}
			if fileHash(t, path) != fileHash(t, updated) {
				t.Fatal("rollback overwrote the external archive update")
			}
		})
	}
}

func TestEveryLocaleAppliesAndRestores(t *testing.T) {
	appDir := installLayout(t, fixtureAsar(t))
	svc := newTestService(t)
	want := fileHash(t, asarOf(appDir))
	for _, code := range svc.store.ListLocales() {
		t.Run(code, func(t *testing.T) {
			if _, err := svc.ApplyLocale(Options{AppDir: appDir, Locale: code}); err != nil {
				t.Fatal(err)
			}
			if got, _, ok := readArchiveLocale(asarOf(appDir)); !ok || got != code {
				t.Fatalf("archive reports %q instead of %q", got, code)
			}
			if _, err := svc.RestoreOfficial(Options{AppDir: appDir}); err != nil {
				t.Fatal(err)
			}
			if fileHash(t, asarOf(appDir)) != want {
				t.Fatal("restoration changed the original bytes")
			}
		})
	}
}

func TestRelaunchFailurePreservesSuccessfulOperation(t *testing.T) {
	appDir := installLayout(t, fixtureAsar(t))
	want := fileHash(t, asarOf(appDir))
	svc := newTestService(t)
	svc.launchApp = func(*paths) error { return errors.New("start denied") }
	result, err := svc.ApplyLocale(Options{AppDir: appDir, Restart: true})
	if err != nil || result.Relaunched {
		t.Fatalf("apply result = %+v, %v", result, err)
	}
	if !isPatched(t, asarOf(appDir)) || len(svc.Notes()) == 0 {
		t.Fatal("a launch failure must preserve the patch and report a warning")
	}
	restored, err := svc.RestoreOfficial(Options{AppDir: appDir, Restart: true})
	if err != nil || restored.Relaunched || len(svc.Notes()) == 0 {
		t.Fatalf("restore result = %+v, %v", restored, err)
	}
	if fileHash(t, asarOf(appDir)) != want {
		t.Fatal("a launch failure must preserve the restored archive")
	}
}
