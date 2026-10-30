package main

import (
	"os"
	"strings"
	"testing"

	"github.com/dop251/goja"
)

// The window's script is not covered by any compiler or by go vet, and a
// runtime error part way through it leaves the HTML untouched: the page still
// renders, showing whatever the markup hardcoded, so nothing looks broken from
// the outside. That is exactly how a mistyped variable hid behind a working
// window. This runs the real app.js against a DOM stub so such an error fails
// the build instead of silently freezing the interface.

// domStub builds just enough of a browser for app.js: element lookups, the
// event registration it does, timers, and the Wails binding object.
const domStub = `
var __els = {};
var __ids = __htmlIds;

function makeEl(id) {
  var el = {
    id: id,
    textContent: '',
    title: '',
    className: '',
    disabled: false,
    type: '',
    scrollTop: 0,
    scrollHeight: 0,
    onclick: null,
    _children: [],
    classList: {
      add: function () {}, remove: function () {}, toggle: function () {},
      contains: function () { return false; }
    },
    setAttribute: function () {},
    getAttribute: function () { return null; },
    appendChild: function (c) { this._children.push(c); },
    querySelectorAll: function () { return []; },
    _click: null,
    addEventListener: function (type, fn) { if (type === 'click') { this._click = fn; } }
  };
  // A real element's textContent includes its children's text, and the chip
  // label is built from two spans, so the stub has to concatenate them.
  var _text = '';
  Object.defineProperty(el, 'textContent', {
    get: function () {
      var out = _text;
      for (var i = 0; i < el._children.length; i++) { out += el._children[i].textContent; }
      return out;
    },
    set: function (v) { _text = v; }
  });
  // Clearing innerHTML is how the script resets the locale list, so the chip
  // tally has to follow it or repeated renders inflate the count.
  var _html = '';
  Object.defineProperty(el, 'innerHTML', {
    get: function () { return _html; },
    set: function (v) { _html = v; el._children = []; }
  });
  return el;
}

for (var i = 0; i < __ids.length; i++) { __els[__ids[i]] = makeEl(__ids[i]); }

var __timers = [];
var __ready = null;
var __status = null;
var __locales = null;
var __error = null;

var document = {
  title: '',
  body: makeEl('body'),
  documentElement: { scrollHeight: 0, clientHeight: 0 },
  getElementById: function (id) {
    if (!__els[id]) { throw new Error('app.js asked for #' + id + ' which index.html does not define'); }
    return __els[id];
  },
  querySelector: function () { return makeEl('q'); },
  querySelectorAll: function () { return []; },
  createElement: function (tag) { return makeEl('new-' + tag); },
  addEventListener: function () {}
};

var window = {
  document: document,
  addEventListener: function (type, fn) { if (type === 'DOMContentLoaded') { __ready = fn; } },
  runtime: {
    EventsOn: function () {},
    // The fallback path records that it was reached, so a test can assert the
    // button did something instead of silently ignoring the click.
    ClipboardSetText: function (text) {
      __clipboardCalls += 1;
      __clipboardText = text;
      return Promise.resolve(true);
    }
  }
};

// __navigatorClipboard lets a test remove navigator.clipboard, which is what an
// older WebView2 without a secure context looks like.
var __navigatorClipboard = null;
var navigator = {};
Object.defineProperty(navigator, 'clipboard', {
  get: function () { return __navigatorClipboard; }
});

var __clipboardText = null;
var __clipboardCalls = 0;

// __clicked and __firedChipClick let a test dispatch the listener the script
// registered on a chip, which is the path a real click takes.
var __clicked = null;
function __firedChipClick(el) { if (el && el._click) { el._click(); } }

function setTimeout(fn, ms) { __timers.push(fn); return __timers.length; }
function clearTimeout() {}

var __App = {
  GetVersion: function () { return Promise.resolve('1.2.3'); },
  GetStatus: function () { return Promise.resolve(__status); },
  GetLocales: function () { return Promise.resolve(__locales); },
  ApplyLocale: function (code, confirmed) {
    if (!confirmed) { return Promise.reject(new Error('confirmation required')); }
    return Promise.resolve({ LocaleName: code });
  },
  RestoreLocale: function (confirmed) {
    if (!confirmed) { return Promise.reject(new Error('confirmation required')); }
    return Promise.resolve({});
  },
  LaunchAntigravity: function () { return Promise.resolve(); },
  SelectAppDir: function () { return Promise.resolve(''); },
  SetAppDir: function () { return Promise.resolve(__status); }
};

// Assigned rather than declared: "go" is a reserved word and cannot appear in a
// variable name.
window.go = { main: { App: __App } };
`

func readFrontend(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile("frontend/dist/" + name)
	if err != nil {
		t.Fatalf("reading frontend/dist/%s: %v", name, err)
	}
	return string(data)
}

// htmlIDs extracts the ids the markup declares, so the stub only offers those
// and a reference to a missing element surfaces as an error.
func htmlIDs(html string) []string {
	var ids []string
	rest := html
	for {
		i := strings.Index(rest, `id="`)
		if i < 0 {
			break
		}
		rest = rest[i+len(`id="`):]
		j := strings.Index(rest, `"`)
		if j < 0 {
			break
		}
		ids = append(ids, rest[:j])
		rest = rest[j:]
	}
	return ids
}

// chipCount reports how many chips the locale list currently holds. The element
// is cleared and refilled on every render, so its children are the live count.
func chipCount(t *testing.T, vm *goja.Runtime) int {
	t.Helper()
	v, err := vm.RunString("__els['localeChips']._children.length")
	if err != nil {
		t.Fatalf("counting chips: %v", err)
	}
	return int(v.ToInteger())
}

// clickChip dispatches the click listener app.js registered on the chip whose
// label matches, returning the primary button text afterwards.
func clickChip(t *testing.T, vm *goja.Runtime, label string) string {
	t.Helper()
	script := `
		(function () {
			var chips = __els['localeChips']._children;
			for (var i = 0; i < chips.length; i++) {
				// The label is followed by the coverage figure, so match on the prefix.
				if (chips[i].textContent.indexOf('` + label + `') === 0) { __clicked = chips[i]; return true; }
			}
			return false;
		})();
	`
	v, err := vm.RunString(script)
	if err != nil {
		t.Fatalf("looking up chip %q: %v", label, err)
	}
	if !v.ToBoolean() {
		t.Fatalf("no chip labelled %q", label)
	}
	if _, err := vm.RunString("__firedChipClick(__clicked);"); err != nil {
		t.Fatalf("clicking chip %q: %v", label, err)
	}
	return elementText(t, vm, "btnPrimaryAction")
}

// installStatusJSON/installLocalesJSON seed the stub's data as JSON literals.
func runFrontend(t *testing.T, statusJSON, localesJSON string) *goja.Runtime {
	t.Helper()
	html := readFrontend(t, "index.html")
	appJS := readFrontend(t, "app.js")

	ids := htmlIDs(html)
	quoted := make([]string, 0, len(ids))
	for _, id := range ids {
		quoted = append(quoted, "'"+id+"'")
	}

	vm := goja.New()
	// The id list is defined first because the stub's element loop reads it.
	prelude := "var __htmlIds = [" + strings.Join(quoted, ",") + "];\n" +
		domStub +
		"__status = " + statusJSON + ";\n" +
		"__locales = " + localesJSON + ";\n"

	if _, err := vm.RunString(prelude); err != nil {
		t.Fatalf("DOM stub failed: %v", err)
	}
	if _, err := vm.RunString(appJS); err != nil {
		t.Fatalf("app.js threw while loading, so the page would show only its markup: %v", err)
	}
	return vm
}

// fireReady runs the DOMContentLoaded handler and the timers it schedules, so
// the same initialization path a real window takes is exercised here.
func fireReady(t *testing.T, vm *goja.Runtime) {
	t.Helper()
	if _, err := vm.RunString(`if (!__ready) { throw new Error('app.js never registered DOMContentLoaded'); } __ready();`); err != nil {
		t.Fatalf("DOMContentLoaded handler threw: %v", err)
	}
	// Drain the deferred initialization, then let promise jobs settle.
	for i := 0; i < 5; i++ {
		if _, err := vm.RunString(`
			var pending = __timers; __timers = [];
			for (var i = 0; i < pending.length; i++) { pending[i](); }
		`); err != nil {
			t.Fatalf("deferred initialization threw: %v", err)
		}
	}
}

// drainPromises lets queued microtasks run, so a .then handler registered by a
// click has finished before the test reads the DOM.
func drainPromises(t *testing.T, vm *goja.Runtime) {
	t.Helper()
	for i := 0; i < 5; i++ {
		if _, err := vm.RunString(`void 0`); err != nil {
			t.Fatalf("draining promises: %v", err)
		}
	}
}

func elementText(t *testing.T, vm *goja.Runtime, id string) string {
	t.Helper()
	v, err := vm.RunString(`__els[` + "'" + id + "'" + `] ? __els[` + "'" + id + "'" + `].textContent : '<missing>'`)
	if err != nil {
		t.Fatalf("reading #%s: %v", id, err)
	}
	return v.String()
}

const testStatusPatched = `{
  "AppDir": "C:\\Antigravity",
  "CurrentLanguage": "zh-CN",
  "LocaleName": "简体中文",
  "AppVersion": "2.23.0",
  "HasCleanBackup": true,
  "CleanBackupStatus": "valid",
  "StateIsStale": false
}`

const testLocales = `[
  {"code":"en","name":"English","englishName":"English","translations":0,"coverage":100},
  {"code":"ja","name":"日本語","englishName":"Japanese","translations":644,"coverage":26},
  {"code":"zh-CN","name":"简体中文","englishName":"Simplified Chinese","translations":2406,"coverage":100}
]`

// A patched install has to name the language it is currently showing, and the
// button has to describe the selection rather than a fixed word.
func TestFrontendRendersStatusAndLocales(t *testing.T) {
	vm := runFrontend(t, testStatusPatched, testLocales)
	fireReady(t, vm)

	if got := elementText(t, vm, "statusLabel"); !strings.Contains(got, "zh-CN") {
		t.Errorf("status label did not report the installed language: %q", got)
	}
	if got := elementText(t, vm, "appPathDisplay"); !strings.Contains(got, "Antigravity") {
		t.Errorf("install path was not rendered: %q", got)
	}
	if got := elementText(t, vm, "appVersion"); !strings.Contains(got, "2.23.0") {
		t.Errorf("the Antigravity version was not rendered: %q", got)
	}

	// Every locale becomes a chip, including English.
	if n := chipCount(t, vm); n != 3 {
		t.Errorf("expected one chip per locale (3), got %d", n)
	}
	// Before any selection changes, the button names what is installed.
	if got := elementText(t, vm, "btnPrimaryAction"); !strings.Contains(got, "简体中文") {
		t.Errorf("primary button should name the selected locale, got %q", got)
	}
}

// Choosing another language has to move the button off the installed one, or it
// keeps advertising a language the click will not install.
func TestFrontendPrimaryButtonFollowsSelection(t *testing.T) {
	vm := runFrontend(t, testStatusPatched, testLocales)
	fireReady(t, vm)

	if got := clickChip(t, vm, "日本語"); !strings.Contains(got, "日本語") {
		t.Errorf("after choosing Japanese the button should name it, got %q", got)
	}
	// English is not a pack, so picking it while localized means restoring.
	if got := clickChip(t, vm, "English"); !strings.Contains(got, "English") {
		t.Errorf("choosing English should name it, got %q", got)
	}
}

// An unpatched install has nothing to re-apply, so it must offer to apply.
func TestFrontendOffersApplyWhenUnpatched(t *testing.T) {
	vm := runFrontend(t, `{
	  "AppDir": "C:\\Antigravity",
	  "CurrentLanguage": "en",
	  "LocaleName": "",
	  "AppVersion": "2.23.0",
	  "HasCleanBackup": true,
	  "CleanBackupStatus": "valid",
	  "StateIsStale": false
	}`, testLocales)
	fireReady(t, vm)

	got := elementText(t, vm, "btnPrimaryAction")
	if !strings.Contains(got, "应用") {
		t.Errorf("an unpatched install should offer to apply, got %q", got)
	}
	if got := elementText(t, vm, "statusLabel"); !strings.Contains(got, "英文") {
		t.Errorf("the status should read as English, got %q", got)
	}
}

// The version badge has to come from the backend. A number baked into the
// markup would keep showing a release that no longer matches the binary.
func TestFrontendVersionBadgeComesFromBackend(t *testing.T) {
	vm := runFrontend(t, testStatusPatched, testLocales)
	fireReady(t, vm)

	got := elementText(t, vm, "appVersionBadge")
	if !strings.Contains(got, "1.2.3") {
		t.Errorf("the badge should show the version reported by the backend, got %q", got)
	}
}

// With navigator.clipboard missing the copy button used to do nothing at all,
// which reads as a broken control. It now falls back to the Wails runtime call.
func TestFrontendCopyLogsFallsBackToRuntime(t *testing.T) {
	vm := runFrontend(t, testStatusPatched, testLocales)
	fireReady(t, vm)

	if _, err := vm.RunString(`
		__navigatorClipboard = null;
		__clipboardCalls = 0;
		__els['btnCopyLogs']._click();
	`); err != nil {
		t.Fatalf("clicking copy logs threw: %v", err)
	}
	drainPromises(t, vm)

	v, err := vm.RunString(`__clipboardCalls`)
	if err != nil {
		t.Fatalf("reading the clipboard call count: %v", err)
	}
	if v.ToInteger() != 1 {
		t.Fatalf("the runtime clipboard was called %d times, want 1", v.ToInteger())
	}
	if got := elementText(t, vm, "msgTitle"); !strings.Contains(got, "已复制") {
		t.Errorf("the user was not told the copy succeeded, got %q", got)
	}
}
