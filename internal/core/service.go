package core

// Options controls an apply or restore run.
type Options struct {
	AppDir    string
	Locale    string
	Restart   bool
	Force     bool
	TimeoutMs int
}

// EnglishLocale is the official untranslated build. It is not a language pack:
// it is the state apply leaves behind, reached through restore.
const EnglishLocale = "en"

// Service is the entry point the CLI and the GUI both drive.
type Service struct {
	store *LocaleStore
	// Progress receives stage updates for a GUI.
	Progress func(stage, detail string)
	// notes carries warnings that should be shown to the user but must not fail
	// an otherwise successful operation.
	notes []string
	// Process hooks keep fixture tests independent of the installed application.
	stopApp    func(bool, int) (bool, bool, bool, error)
	appRunning func() (bool, error)
	launchApp  func(*paths) error
}

// NewService builds the service around the embedded locale pack.
func NewService(store *LocaleStore) *Service {
	return &Service{store: store, stopApp: stopAntigravity, appRunning: isAntigravityRunning, launchApp: launchAntigravity}
}

// Notes returns warnings accumulated by the last operation.
func (s *Service) Notes() []string { return s.notes }

// DescribeLocales lists the bundled language packs with their coverage.
//
// The official English build is listed as well, because choosing it from the
// GUI means restore rather than apply. Without it the picker cannot express the
// only language every installation actually ships with.
func (s *Service) DescribeLocales() ([]LocaleInfo, error) {
	locales, err := s.store.Describe()
	if err != nil {
		return nil, err
	}
	return append([]LocaleInfo{{
		Code:         EnglishLocale,
		Name:         "English",
		EnglishName:  "English",
		Translations: 0,
		Coverage:     100,
	}}, locales...), nil
}

// clearNotes drops warnings from previous operations.
func (s *Service) clearNotes() { s.notes = nil }

// progress reports a stage update when a sink is configured.
func (s *Service) progress(stage, detail string) {
	if s.Progress != nil {
		s.Progress(stage, detail)
	}
}
