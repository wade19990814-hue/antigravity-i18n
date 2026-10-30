package core

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// backupNoteThreshold is how many timestamped backups accumulate before the tool
// starts pointing out that they can be pruned.
const backupNoteThreshold = 5

// hashArchive returns the SHA-256 of a file, which is how changes to app.asar are
// detected across an operation. An automatic app update would otherwise swap the
// archive out from under a patch in progress.
func hashArchive(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	sum, err := hashReader(f)
	if err != nil {
		return "", fmt.Errorf("could not hash archive %s: %w", path, err)
	}
	return sum, nil
}

func hashReader(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// assertArchiveUnchanged fails when the archive no longer hashes to expected.
func assertArchiveUnchanged(path, expected string) error {
	current, err := hashArchive(path)
	if err != nil {
		return fmt.Errorf(
			"Antigravity archive changed during this operation, possibly because of an automatic update. " +
				"Wait for the update to finish and re-run")
	}
	if current != expected {
		return fmt.Errorf(
			"Antigravity archive changed during this operation, possibly because of an automatic update. " +
				"Wait for the update to finish and re-run")
	}
	return nil
}

// detectPatchState reports whether the archive is patched, or nil when the
// archive cannot be read at all.
func detectPatchState(asarPath string) *bool {
	patched, err := detectPatchInArchive(asarPath)
	if err != nil {
		return nil
	}
	return &patched
}

// archiveLocaleRe parses the locale header of an installed patch. The engine
// declares its own locale, so an installed archive is authoritative about which
// language is active instead of a state marker having to be trusted.
var archiveLocaleRe = regexp.MustCompile(`(?s)const AG_LOCALE = (\{.*?\});\s*const agText`)

// readArchiveLocale reads the locale out of a patched archive.
func readArchiveLocale(asarPath string) (code, name string, ok bool) {
	arc, err := ReadArchive(asarPath)
	if err != nil {
		return "", "", false
	}
	preload, err := arc.ReadFile("dist/preload.js")
	if err != nil {
		return "", "", false
	}
	m := archiveLocaleRe.FindSubmatch(preload)
	if m == nil {
		return "", "", false
	}
	var header struct {
		Language string `json:"language"`
		Name     string `json:"name"`
	}
	if err := json.Unmarshal(m[1], &header); err != nil {
		return "", "", false
	}
	if header.Language == "" {
		return "", "", false
	}
	return header.Language, header.Name, true
}

// normalizeLanguage maps a recorded language onto a locale code. Markers written
// before locales were tracked stored a bare 'zh', so an existing install keeps
// reporting correctly instead of showing up as unknown.
func normalizeLanguage(language string) string {
	if language == "" {
		return "unknown"
	}
	if language == "zh" {
		return DefaultLocale
	}
	return language
}

// getStatus reports the state of an installation.
func (s *Service) GetStatus(appDir string) (*Status, error) {
	p, err := resolveAppPaths(appDir)
	if err != nil {
		return nil, err
	}

	var expectedIdentity *Identity
	if arc, err := ReadArchive(p.AsarPath); err == nil {
		expectedIdentity = arc.Identity()
	}
	backupState := inspectCleanBackup(p.CleanBackupPath, expectedIdentity)

	state := readState(p)
	detected := detectPatchState(p.AsarPath)
	if detected == nil {
		return nil, fmt.Errorf("could not read Antigravity archive: %s", p.AsarPath)
	}

	var currentLanguage, localeName string
	if *detected {
		// A patched archive names its own locale. Archives written by a release
		// that predated locale files carry no header, and those were zh-CN only.
		if code, name, ok := readArchiveLocale(p.AsarPath); ok {
			currentLanguage = code
			localeName = name
		} else {
			currentLanguage = DefaultLocale
		}
	} else {
		currentLanguage = "en"
	}

	stale := false
	if state != nil && normalizeLanguage(state.Language) != currentLanguage {
		stale = true
	}

	appVersion := ""
	if expectedIdentity != nil {
		appVersion = expectedIdentity.Version
	}

	return &Status{
		AppDir:            p.AppDir,
		ResourcesDir:      p.ResourcesDir,
		AsarPath:          p.AsarPath,
		CleanBackupPath:   p.CleanBackupPath,
		StatePath:         p.StatePath,
		AppVersion:        appVersion,
		CurrentLanguage:   currentLanguage,
		LocaleName:        localeName,
		HasCleanBackup:    backupState == cleanBackupValid,
		CleanBackupStatus: backupState,
		LastPatchedAt:     statePatchedAt(state),
		StateIsStale:      stale,
	}, nil
}

func statePatchedAt(s *statePayload) *string {
	if s == nil || s.PatchedAt == "" {
		return nil
	}
	v := s.PatchedAt
	return &v
}

// clean backup states, ordered by how much rebuilding work each one implies.
const (
	cleanBackupValid       = "valid"
	cleanBackupMissing     = "missing"
	cleanBackupInvalid     = "invalid"
	cleanBackupPatched     = "patched"
	cleanBackupVersionDiff = "version-mismatch"
)

// inspectCleanBackup verifies a candidate clean archive rather than trusting its
// name. A backup taken after patching, or left over from a previous release,
// would restore the wrong thing.
func inspectCleanBackup(path string, expectedIdentity *Identity) string {
	if !fileExists(path) {
		return cleanBackupMissing
	}
	arc, err := ReadArchive(path)
	if err != nil {
		return cleanBackupInvalid
	}
	identity := arc.Identity()
	if identity == nil || identity.Name != "antigravity" {
		return cleanBackupInvalid
	}
	if expectedIdentity != nil && !SameRelease(expectedIdentity, identity) {
		return cleanBackupVersionDiff
	}
	patched := detectPatchState(path)
	if patched == nil {
		return cleanBackupInvalid
	}
	if *patched {
		return cleanBackupPatched
	}

	// Backups share app.asar.unpacked with the installation, so only packed
	// entries are checked; sidecar entries are not stored in the backup itself.
	payloadSize := int64(0)
	if info, err := os.Stat(path); err == nil {
		payloadSize = info.Size() - int64(8+arc.HeaderSize)
	}
	for _, rel := range arc.FileEntries() {
		node := arc.lookup(rel)
		if node == nil || node.Size == nil || node.Offset == nil {
			return cleanBackupInvalid
		}
		if node.Link != nil {
			continue
		}
		if node.Unpacked != nil && *node.Unpacked {
			continue
		}
		off, err := strconv.ParseInt(*node.Offset, 10, 64)
		if err != nil || off < 0 || off+*node.Size > payloadSize {
			return cleanBackupInvalid
		}
		data, err := arc.ReadFile(rel)
		if err != nil {
			return cleanBackupInvalid
		}
		if int64(len(data)) != *node.Size {
			return cleanBackupInvalid
		}
		if node.Integrity != nil {
			if node.Integrity.Algorithm != "SHA256" {
				return cleanBackupInvalid
			}
			sum := sha256.Sum256(data)
			if hex.EncodeToString(sum[:]) != node.Integrity.Hash {
				return cleanBackupInvalid
			}
		}
	}
	return cleanBackupValid
}

// backupStampRe extracts the yyyyMMdd-HHmmss stamp from a backup name.
var backupStampRe = regexp.MustCompile(`\d{8}-\d{6}`)

func backupStamp(name string) string {
	if m := backupStampRe.FindString(name); m != "" {
		return m
	}
	return ""
}

// findPristineArchive looks for an unpatched archive among the timestamped
// backups.
//
// Installs that were patched by an older version, or had their clean backup
// deleted, can still hold a pristine copy in app.asar.bak-*. Each candidate is
// verified rather than trusted by name, since some backups were themselves taken
// after patching.
func findPristineArchive(resourcesDir string, expectedIdentity *Identity) (string, error) {
	entries, err := os.ReadDir(resourcesDir)
	if err != nil {
		return "", nil
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasPrefix(e.Name(), "app.asar.bak-") {
			names = append(names, e.Name())
		}
	}
	// Backup names embed yyyyMMdd-HHmmss, but a restore also writes
	// bak-before-restore-<stamp> names, so ordering by the whole file name would
	// interleave the two schemes. Sort by the stamp itself, newest first: the
	// newest clean archive is preferred because an older one can belong to a
	// previous Antigravity release and would silently downgrade app.asar out of
	// sync with app.asar.unpacked and the native modules.
	sort.Slice(names, func(i, j int) bool {
		return backupStamp(names[j]) < backupStamp(names[i])
	})

	for _, name := range names {
		candidate := filepath.Join(resourcesDir, name)
		if inspectCleanBackup(candidate, expectedIdentity) == cleanBackupValid {
			return candidate, nil
		}
	}
	return "", nil
}

func backupStampOf(name string) string { return backupStamp(name) }

// formatTimestamp names a timestamped backup.
func formatTimestamp() string {
	now := time.Now()
	return fmt.Sprintf("%04d%02d%02d-%02d%02d%02d",
		now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())
}

// createUniqueBackupPath avoids colliding with an existing backup of the same
// name.
func createUniqueBackupPath(resourcesDir, fileName string) string {
	candidate := filepath.Join(resourcesDir, fileName)
	suffix := 1
	for fileExists(candidate) {
		candidate = filepath.Join(resourcesDir, fmt.Sprintf("%s-%d", fileName, suffix))
		suffix++
	}
	return candidate
}

// noteBackupAccumulation points out that timestamped snapshots grow with every
// run and can be pruned once the install proves itself. The clean backup is what
// restore actually needs.
func noteBackupAccumulation(s *Service, resourcesDir string) {
	entries, err := os.ReadDir(resourcesDir)
	if err != nil {
		return
	}
	count := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "app.asar.bak-") {
			count++
		}
	}
	if count > backupNoteThreshold {
		// Informational only: nothing is pruned automatically, because these are
		// the user's safety copies.
		s.notes = append(s.notes, fmt.Sprintf(
			"Found %d timestamped backups in %s. They accumulate on every apply and can be deleted "+
				"once you trust the install; app.asar.clean-backup is the one restore needs.", count, resourcesDir))
	}
}
