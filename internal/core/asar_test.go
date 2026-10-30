package core

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// envAsar points at a real app.asar so the round trip is exercised against the
// actual header Elect releasedar, not a synthetic one.
func envAsar(t *testing.T) string {
	t.Helper()
	p := os.Getenv("ANTIGRAVITY_ASAR")
	if p == "" {
		t.Skip("set ANTIGRAVITY_ASAR to a real app.asar to run this test")
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("ANTIGRAVITY_ASAR is not readable: %v", err)
	}
	return p
}

func TestReadArchiveRealHeader(t *testing.T) {
	src := envAsar(t)
	arc, err := ReadArchive(src)
	if err != nil {
		t.Fatalf("ReadArchive: %v", err)
	}
	files := arc.FileEntries()
	if len(files) < 10 {
		t.Fatalf("expected the archive to hold many files, got %d", len(files))
	}
	for _, rel := range []string{"package.json", "dist/main.js", "dist/menu.js", "dist/preload.js"} {
		if arc.lookup(rel) == nil {
			t.Fatalf("expected entry %s to be present", rel)
		}
	}
	id := arc.Identity()
	if id == nil {
		t.Fatal("expected package.json identity to be readable")
	}
	if id.Name != "antigravity" {
		t.Fatalf("identity name = %q, want antigravity", id.Name)
	}
}

// Every packed entry must be readable and match its declared size, which is
// what proves the offset arithmetic is right on a real archive.
func TestReadEveryEntry(t *testing.T) {
	src := envAsar(t)
	arc, err := ReadArchive(src)
	if err != nil {
		t.Fatalf("ReadArchive: %v", err)
	}
	for _, rel := range arc.FileEntries() {
		node := arc.lookup(rel)
		data, err := arc.ReadFile(rel)
		if err != nil {
			t.Fatalf("ReadFile(%s): %v", rel, err)
		}
		if int64(len(data)) != *node.Size {
			t.Fatalf("%s: read %d bytes, header says %d", rel, len(data), *node.Size)
		}
		if node.Integrity != nil && node.Integrity.Hash != "" {
			want := sha256Hex(data)
			if want != node.Integrity.Hash {
				t.Fatalf("%s: content hash does not match its integrity", rel)
			}
		}
	}
}

// The write path must reproduce the source exactly when nothing is patched:
// identical bytes, identical header, identical layout.
func TestWritePatchedIdentity(t *testing.T) {
	src := envAsar(t)
	dir := t.TempDir()
	dst := filepath.Join(dir, "app.asar")

	if err := WritePatched(src, dst, nil); err != nil {
		t.Fatalf("WritePatched: %v", err)
	}
	if !bytesEqual(src, dst) {
		t.Fatal("WritePatched with no patches changed the archive")
	}
}

// Patching three files must leave every other body byte for byte identical and
// only shift what follows them.
func TestWritePatchedKeepsOtherBodies(t *testing.T) {
	src := envAsar(t)
	dir := t.TempDir()
	dst := filepath.Join(dir, "app.asar")

	patches := []Patch{
		{Path: "dist/main.js", Data: []byte("// patched main\n")},
		{Path: "dist/menu.js", Data: []byte("// patched menu\n")},
		{Path: "dist/preload.js", Data: append([]byte("// patched preload\n"), make([]byte, 5000)...)},
	}
	if err := WritePatched(src, dst, patches); err != nil {
		t.Fatalf("WritePatched: %v", err)
	}

	before, err := ReadArchive(src)
	if err != nil {
		t.Fatal(err)
	}
	after, err := ReadArchive(dst)
	if err != nil {
		t.Fatalf("re-reading the patched archive: %v", err)
	}

	want := map[string][]byte{
		"dist/main.js":    []byte("// patched main\n"),
		"dist/menu.js":    []byte("// patched menu\n"),
		"dist/preload.js": patches[2].Data,
	}
	for _, rel := range after.FileEntries() {
		got, err := after.ReadFile(rel)
		if err != nil {
			t.Fatalf("ReadFile(%s) on patched archive: %v", rel, err)
		}
		if w, ok := want[rel]; ok {
			if !bytes.Equal(got, w) {
				t.Fatalf("%s: patched content not written", rel)
			}
			continue
		}
		orig, err := before.ReadFile(rel)
		if err != nil {
			t.Fatalf("ReadFile(%s) on source: %v", rel, err)
		}
		if !bytes.Equal(got, orig) {
			t.Fatalf("%s: untouched entry drifted", rel)
		}
	}

	// Integrity of a changed file must be recomputed and must verify.
	for _, rel := range []string{"dist/main.js", "dist/preload.js"} {
		node := after.lookup(rel)
		data, _ := after.ReadFile(rel)
		if node.Integrity == nil {
			t.Fatalf("%s: integrity missing after patch", rel)
		}
		if got := sha256Hex(data); got != node.Integrity.Hash {
			t.Fatalf("%s: integrity hash does not match new content", rel)
		}
		for i, blk := range node.Integrity.Blocks {
			start := i * int(node.Integrity.BlockSize)
			end := start + int(node.Integrity.BlockSize)
			if end > len(data) {
				end = len(data)
			}
			if sha256Hex(data[start:end]) != blk {
				t.Fatalf("%s: block %d hash mismatch", rel, i)
			}
		}
	}

	if id := after.Identity(); id == nil || id.Name != "antigravity" {
		t.Fatal("identity changed after patching")
	}
}

func bytesEqual(a, b string) bool {
	da, err := os.ReadFile(a)
	if err != nil {
		return false
	}
	db, err := os.ReadFile(b)
	if err != nil {
		return false
	}
	return bytes.Equal(da, db)
}
