package core

import (
	"fmt"
	"regexp"
	"strings"
)

// PATCH_MARKERS are strings that appear in the archive only after a patch has
// been applied. A scan of the whole archive for any of them is how the tool
// decides whether the install is localised.
var patchMarkers = []string{
	"installLocaleDialogs",
	"installLocalePatch",
	"installZhCNPatch",
	"const zhCNText = new Map([",
	"function translateMenu(menu)",
}

// blockSentinels wrap every injected block. Fragments carry them, so a re-run
// can excise the previous block exactly instead of guessing its bounds. The
// former polygravity and antigravity-zh pairs are still recognised, so installs
// patched before either rename are replaced cleanly instead of accumulating a
// second block.
var blockSentinels = [][2]string{
	{"/* antigravity-i18n:begin */", "/* antigravity-i18n:end */"},
	{"/* polygravity:begin */", "/* polygravity:end */"},
	{"/* antigravity-zh:begin */", "/* antigravity-zh:end */"},
}

// legacyEngineStartMarkers locate engine blocks injected before sentinels
// existed. Matching on a code identifier cannot see the fragment's leading
// comment, so these are only used when no sentinel is present.
var legacyEngineStartMarkers = []string{"const AG_LOCALE = ", "const zhCNText = new Map(["}

const legacyMenuStartMarker = "function translateMenu(menu)"

// updaterAnchor is where the engine block ends in a legacy preload, and the
// anchor a fresh preload is split at: the engine must run before the preload
// script exposes its bridges.
const updaterAnchor = "const updaterAPI = {"

// split holds the source divided at the injected block's position, with
// surrounding whitespace normalised.
type split struct {
	head     string
	tail     string
	replaced bool
}

// splitAtInjectedBlock removes every sentinel-delimited block from source and
// reports where the replacement belongs.
//
// The first block's position is where the replacement goes, so output does not
// depend on the order the sentinel pairs are swept in, and a file that already
// accumulated duplicates is repaired rather than merely not made worse. Legacy
// blocks were located by a code marker, so their extent has to be supplied:
// legacyEndMarker bounds the block, and its absence means it ran to EOF.
//
// Whitespace around the removed block is collapsed to a single newline so that
// re-injecting is byte-stable: without this, every run leaves one more blank
// line and app.asar keeps changing even though the patch is identical.
func splitAtInjectedBlock(source string, legacy *legacySpec) split {
	cleaned := source
	head := -1

	for _, pair := range blockSentinels {
		beginMarker, endMarker := pair[0], pair[1]
		for {
			begin := strings.Index(cleaned, beginMarker)
			if begin < 0 {
				break
			}
			end := strings.Index(cleaned[begin:], endMarker)
			if end < 0 {
				break
			}
			end += begin
			before := cleaned[:begin]
			after := cleaned[end+len(endMarker):]
			// The earliest block across all sentinel pairs decides where the
			// replacement lands, so output does not depend on sweep order.
			if head < 0 || len(before) < head {
				head = len(before)
			}
			cleaned = before + after
		}
	}
	if head >= 0 {
		return split{
			head:     trimTrailingSpace(source[:head]),
			tail:     trimLeadingSpace(cleaned[head:]),
			replaced: true,
		}
	}

	if legacy != nil {
		for _, marker := range legacy.startMarkers {
			index := strings.Index(cleaned, marker)
			if index < 0 {
				continue
			}
			cutTo := len(cleaned)
			if legacy.endMarker != "" {
				if endIndex := strings.Index(cleaned[index:], legacy.endMarker); endIndex > 0 {
					cutTo = index + endIndex
				}
			}
			return split{
				head:     trimTrailingSpace(cleaned[:index]),
				tail:     trimLeadingSpace(cleaned[cutTo:]),
				replaced: true,
			}
		}
	}
	return split{head: source, tail: "", replaced: false}
}

// legacySpec describes a pre-sentinel injected block.
type legacySpec struct {
	startMarkers []string
	endMarker    string
}

// injectFragment inserts a fragment into a file, replacing any block a previous
// run injected.
//
// When nothing was previously injected, fallbackAnchor marks the insertion
// point; without one the fragment is appended at EOF.
func injectFragment(source, fragment string, legacy *legacySpec, fallbackAnchor string) string {
	sp := splitAtInjectedBlock(source, legacy)
	if sp.replaced {
		if sp.tail != "" {
			return fmt.Sprintf("%s\n\n%s\n\n%s", sp.head, fragment, sp.tail)
		}
		return fmt.Sprintf("%s\n\n%s\n", sp.head, fragment)
	}

	if fallbackAnchor != "" {
		if anchorAt := strings.Index(source, fallbackAnchor); anchorAt >= 0 {
			head := trimTrailingSpace(source[:anchorAt])
			tail := source[anchorAt:]
			return fmt.Sprintf("%s\n\n%s\n\n%s", head, fragment, tail)
		}
	}
	return fmt.Sprintf("%s\n\n%s\n", trimTrailingSpace(source), fragment)
}

// trimTrailingSpace removes trailing whitespace the way the JS version's
// /\s+$/ does, so re-running produces identical bytes.
func trimTrailingSpace(s string) string {
	return strings.TrimRight(s, " \t\r\n\v\f")
}

// trimLeadingSpace removes leading whitespace the way /^\s+/ does.
func trimLeadingSpace(s string) string {
	return strings.TrimLeft(s, " \t\r\n\v\f")
}

// langSwitchRe matches the Chromium language switch block. It is rewritten in
// place: pointing --lang at the locale is what makes native dialogs, spell
// checking and date formatting follow the chosen language. Writing it inside a
// hasSwitch guard keeps an explicit command-line --lang from the user winning.
var langSwitchRe = regexp.MustCompile(`if \(!electron_1\.app\.commandLine\.hasSwitch\('lang'\)\) \{\s*electron_1\.app\.commandLine\.appendSwitch\('lang', '[^']*'\);\s*\}`)

// debugPortRe matches the remote debugging port block. Antigravity 2.28 no
// longer ships its own language switch, so this stable block is where the
// switch is inserted when the older language block is absent.
var debugPortRe = regexp.MustCompile(`if\s*\(!electron_1\.app\.commandLine\.hasSwitch\('remote-debugging-port'\)\)\s*\{\s*electron_1\.app\.commandLine\.appendSwitch\('remote-debugging-port',\s*'0'\);\s*\}`)

// applyLanguageSwitch points Chromium's --lang switch at the locale being
// installed.
//
// An earlier version only inserted the switch when absent, which meant that
// switching from one language to another left the previous code in place and
// Chromium kept formatting dates and spell checking in the old language. The
// block is therefore rewritten every run rather than merely created once.
func applyLanguageSwitch(source string, locale *Locale) (string, error) {
	chromiumLang := locale.ChromiumLang
	if chromiumLang == "" {
		chromiumLang = locale.Language
	}
	if !chromiumLangRe.MatchString(chromiumLang) {
		return "", fmt.Errorf("locale '%s' has an unusable Chromium language code", locale.Language)
	}
	block := strings.Join([]string{
		"if (!electron_1.app.commandLine.hasSwitch('lang')) {",
		fmt.Sprintf("    electron_1.app.commandLine.appendSwitch('lang', '%s');", chromiumLang),
		"}",
	}, "\n")

	if langSwitchRe.MatchString(source) {
		return langSwitchRe.ReplaceAllStringFunc(source, func(string) string { return block }), nil
	}
	if loc := debugPortRe.FindStringIndex(source); loc != nil {
		return source[:loc[1]] + "\n" + block + source[loc[1]:], nil
	}
	return source, nil
}

// chromiumLangRe restricts the language code to characters safe to place in
// generated source.
var chromiumLangRe = regexp.MustCompile(`^[A-Za-z0-9-]+$`)

// patchMain rewrites main.js: the language switch plus the native dialog patch.
func patchMain(source string, locale *Locale, dialogFragment string) (string, error) {
	out, err := applyLanguageSwitch(source, locale)
	if err != nil {
		return "", err
	}
	if !strings.Contains(out, "appendSwitch('lang'") {
		return "", fmt.Errorf(
			"could not apply the language switch to main.js: the expected code pattern was not found. " +
				"Antigravity was likely updated to a build this patch does not recognise; " +
				"report this so the anchor can be updated")
	}
	out = injectFragment(out, dialogFragment, nil, "if (!electron_1.app.commandLine.hasSwitch('lang')) {")
	return out, nil
}

// reapplyMenuRe anchors the call that re-applies the translated menu.
var reapplyMenuRe = regexp.MustCompile(`(\s*)//\s*Re-apply the menu so the change takes effect\.`)

// patchMenu rewrites menu.js: the native menu translation plus the call that
// wires it in.
func patchMenu(source string, menuFragment string) (string, error) {
	legacy := &legacySpec{startMarkers: []string{legacyMenuStartMarker}}
	out := injectFragment(source, menuFragment, legacy, "")

	// Wire the translated menu in. The replacement uses ${1} rather than $1
	// because Go reads a $1 followed by a letter as a named group reference,
	// which would silently swallow the group's text.
	if !strings.Contains(out, "translateMenu(menu);") {
		out = reapplyMenuRe.ReplaceAllString(out, "${1}translateMenu(menu);\n${1}// Re-apply the menu so the change takes effect.")
	}
	if !strings.Contains(out, "translateMenu(menu);") {
		return "", fmt.Errorf(
			"could not wire translateMenu into menu.js: the \"Re-apply the menu\" anchor was not found. " +
				"Antigravity was likely updated to a build this patch does not recognise; " +
				"report this so the anchor can be updated")
	}
	return out, nil
}

// patchPreload injects the DOM translation engine into preload.js.
//
// A previously injected block is replaced, so re-running the patch or switching
// locale never stacks two translation engines. A legacy engine block ended
// where updaterAPI began.
func patchPreload(source string, engineFragment string) string {
	legacy := &legacySpec{startMarkers: legacyEngineStartMarkers, endMarker: updaterAnchor}
	return injectFragment(source, engineFragment, legacy, updaterAnchor)
}

// detectPatchInArchive reports whether the archive carries a localization
// patch, or nil when the archive cannot be read.
//
// Reading a fixed prefix of app.asar is unreliable because asar does not
// guarantee where a given file lands inside the archive, so the injected
// preload can sit well past any fixed offset. The whole archive is streamed
// with an overlap so a marker split across a chunk boundary is still found.
func detectPatchInArchive(path string) (bool, error) {
	const chunkSize = 1 << 20
	longest := 0
	for _, m := range patchMarkers {
		if len(m) > longest {
			longest = len(m)
		}
	}

	f, err := openFile(path)
	if err != nil {
		return false, nil
	}
	defer f.Close()

	buf := make([]byte, chunkSize)
	carry := make([]byte, 0, longest)
	for {
		n, err := f.Read(buf)
		if n <= 0 {
			break
		}
		window := append(append([]byte{}, carry...), buf[:n]...)
		if containsAny(window, patchMarkers) {
			return true, nil
		}
		if len(window) > longest {
			carry = append(carry[:0], window[len(window)-longest:]...)
		} else {
			carry = append(carry[:0], window...)
		}
		if err != nil {
			break
		}
	}
	return false, nil
}
