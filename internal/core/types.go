package core

// Identity names the application an asar archive belongs to.
type Identity struct {
	Name    string
	Version string
}

// SameRelease reports whether two identities describe the same build. A missing
// identity cannot contradict anything, so it matches.
func SameRelease(left, right *Identity) bool {
	if left == nil || right == nil {
		return true
	}
	return left.Name == right.Name && left.Version == right.Version
}

// Status is what the tool reports about an installation.
type Status struct {
	AppDir          string
	ResourcesDir    string
	AsarPath        string
	CleanBackupPath string
	StatePath       string

	// AppVersion is the Antigravity release the archive declares, shown in the
	// UI so the language pack and the app version can be matched up.
	AppVersion string

	// CurrentLanguage is the locale actually present in the archive, an
	// authoritative answer rather than a recorded one.
	CurrentLanguage   string
	LocaleName        string
	HasCleanBackup    bool
	CleanBackupStatus string
	LastPatchedAt     *string
	StateIsStale      bool
}

// ApplyResult describes what an apply run did.
type ApplyResult struct {
	Locale      string
	LocaleName  string
	BackupPath  string
	CleanBackup string
	WasRunning  bool
	Forced      bool
	Relaunched  bool
}

// RestoreResult describes what a restore run did.
type RestoreResult struct {
	AlreadyEnglish bool
	BackupPath     string
	WasRunning     bool
	Forced         bool
	Relaunched     bool
}

// LocaleInfo is one language pack the tool can install.
type LocaleInfo struct {
	Code         string `json:"code"`
	Name         string `json:"name"`
	EnglishName  string `json:"englishName"`
	Translations int    `json:"translations"`
	Coverage     int    `json:"coverage"`
}
