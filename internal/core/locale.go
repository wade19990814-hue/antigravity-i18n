package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// DefaultLocale is what apply uses when no locale is named.
const DefaultLocale = "zh-CN"

// LocaleCodeRe restricts locale codes to a BCP 47 shape.
var LocaleCodeRe = regexp.MustCompile(`^[A-Za-z]{2,3}(?:-[A-Za-z0-9]{2,8})*$`)

// writingDirections the injected engine knows how to apply to <html dir>.
var writingDirections = []string{"ltr", "rtl"}

// pluralCategories are CLDR's fixed set. A locale supplies only the ones its
// language uses, but 'other' is mandatory because it is the fallback that keeps
// a missing category from rendering as an empty string in the UI.
var pluralCategories = []string{"zero", "one", "two", "few", "many", "other"}

var exactPluralKeyRe = regexp.MustCompile(`^=\d+$`)

// rtlLanguages are written right to left, keyed by primary subtag. A locale for
// one of these that does not declare its direction would be fully translated
// yet still laid out left to right, a silent visual defect rather than an error.
var rtlLanguages = map[string]bool{
	"ar": true, "arc": true, "ckb": true, "dv": true, "fa": true, "he": true,
	"ku": true, "nqo": true, "ps": true, "sd": true, "ug": true, "ur": true, "yi": true,
}

// localeAliases are spellings that resolve onto a bundled pack instead of naming
// a file of their own: 'zh' and 'zh-Hans' (any casing) both mean the Simplified
// Chinese pack shipped as zh-CN. Matching itself is case-insensitive, so
// 'zh-hant' resolves onto the zh-Hant file without a separate alias.
//
// The extra entries cover the spellings people actually type: region subtags
// for the two Chinese scripts, the bare language for a pack shipped with a
// region (pt -> pt-BR), and the two codes that are commonly mistaken for their
// ISO 639-1 form (jp -> ja, kr -> ko). Rejecting these would be technically
// correct and practically annoying.
var localeAliases = map[string]string{
	"zh":      DefaultLocale,
	"zh-hans": DefaultLocale,
	"zh-tw":   "zh-Hant",
	"zh-hk":   "zh-Hant",
	"zh-mo":   "zh-Hant",
	"pt":      "pt-BR",
	"jp":      "ja",
	"kr":      "ko",
}

// CanonicalLocale folds a locale spelling onto the code it names, so 'ZH',
// 'zh_CN' and 'zh-CN' all compare equal. An unrecognized spelling is returned
// normalized rather than empty, which keeps it distinguishable from a real code
// without pretending it resolved to one.
//
// The alias table is applied here and not only in loadResolved, because the CLI
// compares two spellings to decide whether they conflict.
func CanonicalLocale(code string) string {
	normalized := strings.TrimSpace(code)
	if i := strings.IndexAny(normalized, ".@"); i >= 0 {
		normalized = normalized[:i]
	}
	normalized = strings.ToLower(strings.ReplaceAll(normalized, "_", "-"))
	if alias, ok := localeAliases[normalized]; ok {
		return strings.ToLower(alias)
	}
	return normalized
}

// PlaceholderToken names replaced in a fragment when it is built.
const (
	localeDataPlaceholder = "LOCALE_DATA_PLACEHOLDER"
	menuDataPlaceholder   = "MENU_DATA_PLACEHOLDER"
	dialogDataPlaceholder = "DIALOG_DATA_PLACEHOLDER"
)

// Locale is one language pack. Fields are decoded lazily: dictionary maps may be
// large, and validation needs to see them deterministically ordered so error
// messages are stable across runs.
type Locale struct {
	Language     string `json:"language"`
	Name         string `json:"name"`
	EnglishName  string `json:"englishName"`
	HTMLLang     string `json:"htmlLang"`
	ChromiumLang string `json:"chromiumLang"`
	Dir          string `json:"dir"`
	PluralLocale string `json:"pluralLocale"`

	Text                    map[string]string            `json:"text"`
	Menu                    map[string]string            `json:"menu"`
	Dialogs                 map[string]string            `json:"dialogs"`
	Punctuation             map[string]string            `json:"punctuation"`
	PunctuationSkipSuffixes []string                     `json:"punctuationSkipSuffixes"`
	ValueMaps               map[string]map[string]string `json:"valueMaps"`
	ValueRules              map[string][]ValueRule       `json:"valueRules"`
	Patterns                []PatternRule                `json:"patterns"`

	AllowSourceEqual []string `json:"allowSourceEqual"`
}

// ValueRule is a valueRules entry: a regex plus the template it rewrites into.
type ValueRule struct {
	Pattern  string `json:"pattern"`
	Flags    string `json:"flags"`
	Template string `json:"template"`
}

// PatternRule is one dynamic translation rule.
type PatternRule struct {
	ID          string            `json:"id"`
	Pattern     string            `json:"pattern"`
	Flags       string            `json:"flags"`
	Template    string            `json:"template"`
	Templates   map[string]string `json:"templates"`
	Sample      string            `json:"sample"`
	PluralGroup *int              `json:"pluralGroup"`
	Replace     map[string]string `json:"replace"`
	TrimGroups  []int             `json:"trimGroups"`
}

// LocaleStore loads the language packs embedded in the binary.
type LocaleStore struct {
	localesFS fs.FS
	patchesFS fs.FS
	localeDir string
	patchDir  string

	engineFragment []byte
	menuFragment   []byte
	dialogFragment []byte
}

// NewLocaleStore builds the store from the embedded assets.
func NewLocaleStore(localesFS, patchesFS fs.FS, localeDir, patchDir string) (*LocaleStore, error) {
	s := &LocaleStore{localesFS: localesFS, patchesFS: patchesFS, localeDir: localeDir, patchDir: patchDir}
	var err error
	if s.engineFragment, err = fs.ReadFile(patchesFS, patchDir+"/engine.jsfrag"); err != nil {
		return nil, fmt.Errorf("cannot read the translation engine template: %w", err)
	}
	if s.menuFragment, err = fs.ReadFile(patchesFS, patchDir+"/menu.jsfrag"); err != nil {
		return nil, fmt.Errorf("cannot read the menu template: %w", err)
	}
	if s.dialogFragment, err = fs.ReadFile(patchesFS, patchDir+"/dialog.jsfrag"); err != nil {
		return nil, fmt.Errorf("cannot read the dialog template: %w", err)
	}
	return s, nil
}

// ListLocales returns the locale codes shipped with this build, sorted.
func (s *LocaleStore) ListLocales() []string {
	entries, err := fs.ReadDir(s.localesFS, s.localeDir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		code := strings.TrimSuffix(name, ".json")
		if LocaleCodeRe.MatchString(code) {
			out = append(out, code)
		}
	}
	sort.Strings(out)
	return out
}

// Describe lists the bundled locales with their coverage, for status reporting.
func (s *LocaleStore) Describe() ([]LocaleInfo, error) {
	var out []LocaleInfo
	reference, err := s.loadResolved(DefaultLocale)
	if err != nil {
		return nil, err
	}
	for _, code := range s.ListLocales() {
		loc, err := s.LoadLocale(code)
		if err != nil {
			return nil, err
		}
		entries := 0
		covered := 0
		for k := range reference.Text {
			entries++
			if v, ok := loc.Text[k]; ok && v != "" {
				covered++
			}
		}
		coverage := 0
		if entries > 0 {
			coverage = covered * 100 / entries
		}
		out = append(out, LocaleInfo{
			Code:         code,
			Name:         loc.Name,
			EnglishName:  loc.EnglishName,
			Translations: len(loc.Text),
			Coverage:     coverage,
		})
	}
	return out, nil
}

// LoadLocale loads and validates a locale definition.
func (s *LocaleStore) LoadLocale(code string) (*Locale, error) {
	resolved, err := s.loadResolved(code)
	if err != nil {
		return nil, err
	}
	return resolved, nil
}

// loadResolved applies aliases and case folding, then reads and validates.
func (s *LocaleStore) loadResolved(code string) (*Locale, error) {
	available := s.ListLocales()
	// CanonicalLocale folds 'ZH', 'zh_CN.UTF-8' and 'zh' onto 'zh-cn', so the
	// comparison below only has to match one spelling per pack.
	resolved := ""
	if requested := CanonicalLocale(code); LocaleCodeRe.MatchString(requested) {
		for _, name := range available {
			if strings.EqualFold(name, requested) {
				resolved = name
				break
			}
		}
	}
	if resolved == "" {
		availableText := strings.Join(available, ", ")
		if availableText == "" {
			availableText = "none"
		}
		return nil, fmt.Errorf("unknown locale '%s'. available locales: %s", code, availableText)
	}

	raw, err := fs.ReadFile(s.localesFS, s.localeDir+"/"+resolved+".json")
	if err != nil {
		return nil, fmt.Errorf("locale '%s' is not available: %w", resolved, err)
	}
	var loc Locale
	dec := json.NewDecoder(bytes.NewReader(raw))
	if err := dec.Decode(&loc); err != nil {
		return nil, fmt.Errorf("locale '%s' is not valid JSON: %w", resolved, err)
	}
	if err := validateLocale(resolved, &loc); err != nil {
		return nil, err
	}
	return &loc, nil
}

func isPlainObject(v any) bool {
	if v == nil {
		return false
	}
	switch v.(type) {
	case map[string]any:
		return true
	}
	return false
}

// mapOptions tune validateStringMap for maps whose keys or values are allowed to
// be empty.
type mapOptions struct {
	allowEmptyKey   bool
	allowEmptyValue bool
}

func validateStringMap(code, label string, value map[string]string) error {
	return validateStringMapOpts(code, label, value, mapOptions{})
}

func validateStringMapOpts(code, label string, value map[string]string, opts mapOptions) error {
	// An empty map cannot be distinguished from a missing one in JSON, so a
	// nil map means "absent", which callers have already checked.
	if value == nil {
		return fmt.Errorf("locale '%s': '%s' must be an object", code, label)
	}
	for _, key := range sortedKeys(value) {
		if !opts.allowEmptyKey && key == "" {
			return fmt.Errorf("locale '%s': '%s' contains an empty key", code, label)
		}
		if key != strings.TrimSpace(key) {
			return fmt.Errorf("locale '%s': '%s' key '%s' has surrounding whitespace", code, label, key)
		}
		v := value[key]
		if !opts.allowEmptyValue && v == "" {
			return fmt.Errorf("locale '%s': '%s' value for '%s' must be a non-empty string", code, label, key)
		}
	}
	return nil
}

// countCaptureGroups counts capture groups in a pattern the way JavaScript
// does: a pattern with no groups still counts as one for the fallback branch.
// Go has no equivalent trick, so the count is derived from the syntax tree.
func countCaptureGroups(pattern string) (int, error) {
	// Counting needs no compilation: the count is structural, and an
	// unparseable pattern is caught by regexp.Compile on the line below.
	return jsCaptureGroups(pattern), nil
}

// validateTemplateGroups checks that every {n} placeholder in a template names a
// real capture group. A placeholder beyond the count would silently render as
// an empty string in the UI.
func validateTemplateGroups(code, label, template string, groupCount int) error {
	for _, m := range templatePlaceholderRe.FindAllStringSubmatch(template, -1) {
		idx, _ := strconv.Atoi(m[1])
		if idx < 1 || idx > groupCount {
			return fmt.Errorf("locale '%s': %s template references {%d} but the pattern has %d capture group(s)",
				code, label, idx, groupCount)
		}
	}
	return nil
}

var templatePlaceholderRe = regexp.MustCompile(`\{(\d+)\}`)

// validatePluralTemplates checks a plural rule and returns every template it can
// render.
//
// Categories are checked against CLDR's fixed set so a typo such as 'mnay' fails
// here instead of silently falling through to 'other' at runtime, and 'other' is
// required because it is the engine's only fallback.
func validatePluralTemplates(code, label string, rule *PatternRule, groupCount int) ([]string, error) {
	if rule.Templates == nil {
		return nil, fmt.Errorf("locale '%s': %s 'templates' must be an object of plural forms", code, label)
	}
	entries := len(rule.Templates)
	if entries == 0 {
		return nil, fmt.Errorf("locale '%s': %s 'templates' is empty", code, label)
	}
	for _, category := range sortedMapKeys(rule.Templates) {
		known := false
		for _, c := range pluralCategories {
			if c == category {
				known = true
				break
			}
		}
		if !known && !exactPluralKeyRe.MatchString(category) {
			return nil, fmt.Errorf("locale '%s': %s has unknown plural form '%s'; use %s or =N",
				code, label, category, strings.Join(pluralCategories, ", "))
		}
		if rule.Templates[category] == "" {
			return nil, fmt.Errorf("locale '%s': %s plural form '%s' must be a non-empty string", code, label, category)
		}
	}
	if _, ok := rule.Templates["other"]; !ok {
		return nil, fmt.Errorf("locale '%s': %s must define the 'other' plural form as the required fallback", code, label)
	}
	// The counted group is what selects the plural form, so an out-of-range
	// index would make every string fall back to 'other'.
	pluralGroup := 1
	if rule.PluralGroup != nil {
		pluralGroup = *rule.PluralGroup
	}
	if pluralGroup < 1 || pluralGroup > groupCount {
		return nil, fmt.Errorf("locale '%s': %s has an invalid 'pluralGroup' %d; the pattern has %d capture group(s)",
			code, label, pluralGroup, groupCount)
	}

	templates := make([]string, 0, len(rule.Templates))
	for _, category := range sortedMapKeys(rule.Templates) {
		templates = append(templates, rule.Templates[category])
	}
	return templates, nil
}

// validateRegexFlags checks JavaScript flags separately from Go's regex parser.
// Value rules perform replacements and may use g/y; whole-string patterns may not.
func validateRegexFlags(code, label, flags string, allowStateful bool) error {
	seen := map[rune]bool{}
	for _, flag := range flags {
		if !strings.ContainsRune("dgimsuvy", flag) {
			return fmt.Errorf("locale '%s': %s has an invalid regex flag %q", code, label, flag)
		}
		if seen[flag] {
			return fmt.Errorf("locale '%s': %s has a duplicate regex flag %q", code, label, flag)
		}
		if !allowStateful && (flag == 'g' || flag == 'y') {
			return fmt.Errorf("locale '%s': %s cannot use stateful 'g' or 'y' flags", code, label)
		}
		seen[flag] = true
	}
	if seen['u'] && seen['v'] {
		return fmt.Errorf("locale '%s': %s cannot combine regex flags 'u' and 'v'", code, label)
	}
	return nil
}

// validateLocale rejects malformed locale data before it is injected into the
// app.
//
// A bad regex or a template referencing a missing capture group would only
// surface as a broken UI after app.asar had already been rewritten, so every
// pattern is compiled and checked here instead.
func validateLocale(code string, loc *Locale) error {
	if loc == nil || len(loc.Text) == 0 {
		return fmt.Errorf("locale '%s' is missing a 'text' dictionary", code)
	}
	// The 'language' field becomes <html lang> and the CLI's reported code, so a
	// copy-pasted locale file whose field still names the source language would
	// silently mislabel the app.
	if loc.Language != code {
		return fmt.Errorf("locale '%s': 'language' is '%s' but the file is named '%s.json'; they must match",
			code, loc.Language, code)
	}
	if strings.TrimSpace(loc.Name) == "" {
		return fmt.Errorf("locale '%s': 'name' must be a non-empty string", code)
	}
	if loc.EnglishName != "" && strings.TrimSpace(loc.EnglishName) == "" {
		return fmt.Errorf("locale '%s': 'englishName' must be a non-empty string", code)
	}

	if loc.Dir != "" && !containsString(writingDirections, loc.Dir) {
		return fmt.Errorf("locale '%s': 'dir' must be one of %s", code, strings.Join(writingDirections, ", "))
	}
	primarySubtag := strings.ToLower(strings.SplitN(code, "-", 2)[0])
	if rtlLanguages[primarySubtag] && loc.Dir != "rtl" {
		return fmt.Errorf("locale '%s': '%s' is written right to left, so 'dir' must be set to 'rtl'", code, primarySubtag)
	}

	// htmlLang becomes <html lang> and chromiumLang becomes Chromium's --lang
	// switch; both are injected into the app, so malformed values are rejected
	// here instead of failing later at apply time.
	if loc.HTMLLang != "" && !LocaleCodeRe.MatchString(loc.HTMLLang) {
		return fmt.Errorf("locale '%s': 'htmlLang' must be a BCP 47 code", code)
	}
	if loc.ChromiumLang != "" && !chromiumLangRe.MatchString(loc.ChromiumLang) {
		return fmt.Errorf("locale '%s': 'chromiumLang' must only contain letters, digits and hyphens", code)
	}
	// pluralLocale lets a regional file borrow another language's plural rules
	// when its own code is not a language Intl.PluralRules recognises.
	if loc.PluralLocale != "" && !LocaleCodeRe.MatchString(loc.PluralLocale) {
		return fmt.Errorf("locale '%s': 'pluralLocale' must be a BCP 47 code", code)
	}

	if err := validateStringMap(code, "text", loc.Text); err != nil {
		return err
	}
	// The engine re-processes text it just wrote, so a translation that equals
	// another source key is looked up again. Walk each chain: it is safe when it
	// leaves the dictionary or reaches a fixed point, and an error when it
	// revisits a key, which would translate in a cycle forever.
	for _, startKey := range sortedKeys(loc.Text) {
		visited := map[string]bool{startKey: true}
		value := loc.Text[startKey]
		for {
			next, ok := loc.Text[value]
			if !ok {
				break
			}
			if next == value {
				break
			}
			if visited[value] {
				return fmt.Errorf(
					"locale '%s': translation of '%s' re-enters the dictionary through '%s', "+
						"so the injected engine would re-translate its own output in a cycle",
					code, startKey, value)
			}
			visited[value] = true
			value = next
		}
	}

	if loc.AllowSourceEqual != nil {
		seen := map[string]bool{}
		for _, key := range loc.AllowSourceEqual {
			if key == "" {
				return fmt.Errorf("locale '%s': 'allowSourceEqual' must be an array of non-empty strings", code)
			}
			if seen[key] {
				return fmt.Errorf("locale '%s': 'allowSourceEqual' cannot contain duplicates", code)
			}
			seen[key] = true
			if _, ok := loc.Text[key]; !ok {
				return fmt.Errorf("locale '%s': 'allowSourceEqual' references unknown text key '%s'", code, key)
			}
			if loc.Text[key] != key {
				return fmt.Errorf("locale '%s': 'allowSourceEqual' key '%s' must have a value exactly equal to its source", code, key)
			}
		}
	}

	if loc.Menu != nil {
		if err := validateStringMap(code, "menu", loc.Menu); err != nil {
			return err
		}
	}
	if loc.Dialogs != nil {
		if err := validateStringMap(code, "dialogs", loc.Dialogs); err != nil {
			return err
		}
	}
	if loc.Punctuation != nil {
		if err := validateStringMapOpts(code, "punctuation", loc.Punctuation, mapOptions{allowEmptyKey: true, allowEmptyValue: true}); err != nil {
			return err
		}
	}
	if loc.ValueMaps != nil {
		for _, name := range sortedMapKeys(loc.ValueMaps) {
			if err := validateStringMapOpts(code, "valueMaps."+name, loc.ValueMaps[name], mapOptions{allowEmptyKey: true, allowEmptyValue: true}); err != nil {
				return err
			}
		}
	}

	if loc.ValueRules != nil {
		for _, name := range sortedMapKeys(loc.ValueRules) {
			rules := loc.ValueRules[name]
			for index, rule := range rules {
				label := fmt.Sprintf("value rule '%s' #%d", name, index+1)
				if rule.Pattern == "" || rule.Template == "" {
					return fmt.Errorf("locale '%s': %s needs both 'pattern' and 'template' strings", code, label)
				}
				if err := validateRegexFlags(code, label, rule.Flags, true); err != nil {
					return err
				}
				if _, err := regexp.Compile(rule.Pattern); err != nil {
					return fmt.Errorf("locale '%s': %s has an invalid regex: %v", code, label, err)
				}
				groupCount, err := countCaptureGroups(rule.Pattern)
				if err != nil {
					return fmt.Errorf("locale '%s': %s has an invalid regex: %v", code, label, err)
				}
				if err := validateTemplateGroups(code, label, rule.Template, groupCount); err != nil {
					return err
				}
			}
		}
	}

	seenIDs := map[string]bool{}
	for index := range loc.Patterns {
		rule := &loc.Patterns[index]
		label := ""
		if rule.ID != "" {
			label = fmt.Sprintf("pattern '%s'", rule.ID)
		} else {
			label = fmt.Sprintf("pattern #%d", index+1)
		}
		if rule.Pattern == "" {
			return fmt.Errorf("locale '%s': %s needs a 'pattern' string", code, label)
		}
		// A rule carries either one template or a plural category map, never
		// both: two sources of output would silently disagree.
		hasTemplate := rule.Template != ""
		hasTemplates := rule.Templates != nil
		if hasTemplate == hasTemplates {
			return fmt.Errorf("locale '%s': %s needs exactly one of 'template' or 'templates'", code, label)
		}
		if rule.Sample == "" {
			return fmt.Errorf("locale '%s': %s needs a non-empty 'sample' string", code, label)
		}
		if !strings.HasPrefix(rule.Pattern, "^") || !strings.HasSuffix(rule.Pattern, "$") {
			return fmt.Errorf("locale '%s': %s must be anchored with ^ and $", code, label)
		}
		if err := validateRegexFlags(code, label, rule.Flags, false); err != nil {
			return err
		}
		if rule.ID != "" {
			if seenIDs[rule.ID] {
				return fmt.Errorf("locale '%s': duplicate pattern id '%s'", code, rule.ID)
			}
			seenIDs[rule.ID] = true
		}

		compiled, err := regexp.Compile(rule.Pattern)
		if err != nil {
			return fmt.Errorf("locale '%s': %s has an invalid regex: %v", code, label, err)
		}
		groupCount, err := countCaptureGroups(rule.Pattern)
		if err != nil {
			return fmt.Errorf("locale '%s': %s has an invalid regex: %v", code, label, err)
		}
		templates := []string{rule.Template}
		if hasTemplates {
			templates, err = validatePluralTemplates(code, label, rule, groupCount)
			if err != nil {
				return err
			}
		}
		for _, template := range templates {
			if err := validateTemplateGroups(code, label, template, groupCount); err != nil {
				return err
			}
		}
		if !compiled.MatchString(rule.Sample) {
			return fmt.Errorf("locale '%s': %s does not match its own sample", code, label)
		}

		if rule.Replace != nil {
			for _, capture := range sortedMapKeys(rule.Replace) {
				idx, err := strconv.Atoi(capture)
				if err != nil || idx < 1 || idx > groupCount {
					return fmt.Errorf("locale '%s': %s has invalid replace capture '%s'", code, label, capture)
				}
				name := rule.Replace[capture]
				known := false
				if loc.ValueMaps != nil {
					if _, ok := loc.ValueMaps[name]; ok {
						known = true
					}
				}
				if !known && loc.ValueRules != nil {
					if _, ok := loc.ValueRules[name]; ok {
						known = true
					}
				}
				if !known {
					return fmt.Errorf("locale '%s': %s references unknown value map '%s'", code, label, name)
				}
			}
		}
		if rule.TrimGroups != nil {
			for _, g := range rule.TrimGroups {
				if g < 1 || g > groupCount {
					return fmt.Errorf("locale '%s': %s has an invalid 'trimGroups' capture index", code, label)
				}
			}
		}
	}
	return nil
}

// buildPreloadFragment injects runtime locale data into the locale-neutral engine
// template.
func (s *LocaleStore) buildPreloadFragment(loc *Locale) (string, error) {
	// The engine receives runtime locale data only. Native maps belong to the
	// main process, while allowSourceEqual is metadata used only by reports.
	// Both are dropped by marshalling the fields the engine reads.
	renderer := struct {
		Language                string                       `json:"language,omitempty"`
		Name                    string                       `json:"name,omitempty"`
		EnglishName             string                       `json:"englishName,omitempty"`
		HTMLLang                string                       `json:"htmlLang,omitempty"`
		ChromiumLang            string                       `json:"chromiumLang,omitempty"`
		Dir                     string                       `json:"dir,omitempty"`
		PluralLocale            string                       `json:"pluralLocale,omitempty"`
		Text                    map[string]string            `json:"text,omitempty"`
		Punctuation             map[string]string            `json:"punctuation,omitempty"`
		PunctuationSkipSuffixes []string                     `json:"punctuationSkipSuffixes,omitempty"`
		ValueMaps               map[string]map[string]string `json:"valueMaps,omitempty"`
		ValueRules              map[string][]ValueRule       `json:"valueRules,omitempty"`
		Patterns                []PatternRule                `json:"patterns,omitempty"`
	}{
		Language:                loc.Language,
		Name:                    loc.Name,
		EnglishName:             loc.EnglishName,
		HTMLLang:                loc.HTMLLang,
		ChromiumLang:            loc.ChromiumLang,
		Dir:                     loc.Dir,
		PluralLocale:            loc.PluralLocale,
		Text:                    loc.Text,
		Punctuation:             loc.Punctuation,
		PunctuationSkipSuffixes: loc.PunctuationSkipSuffixes,
		ValueMaps:               loc.ValueMaps,
		ValueRules:              loc.ValueRules,
		Patterns:                loc.Patterns,
	}
	data, err := json.Marshal(renderer)
	if err != nil {
		return "", fmt.Errorf("cannot serialise locale data: %w", err)
	}
	// Replace once: the template may legitimately contain the placeholder text
	// inside its own comments, and replacing all of them would corrupt those.
	out, n := replaceFirst(string(s.engineFragment), localeDataPlaceholder, string(data))
	if n == 0 {
		return "", fmt.Errorf("the translation engine template has no %s", localeDataPlaceholder)
	}
	return out, nil
}

func (s *LocaleStore) buildMenuFragment(loc *Locale) (string, error) {
	data, err := json.Marshal(loc.Menu)
	if err != nil {
		return "", err
	}
	out, n := replaceFirst(string(s.menuFragment), menuDataPlaceholder, string(data))
	if n == 0 {
		return "", fmt.Errorf("the menu template has no %s", menuDataPlaceholder)
	}
	return out, nil
}

func (s *LocaleStore) buildDialogFragment(loc *Locale) (string, error) {
	data, err := json.Marshal(loc.Dialogs)
	if err != nil {
		return "", err
	}
	out, n := replaceFirst(string(s.dialogFragment), dialogDataPlaceholder, string(data))
	if n == 0 {
		return "", fmt.Errorf("the dialog template has no %s", dialogDataPlaceholder)
	}
	return out, nil
}

func containsString(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedMapKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
