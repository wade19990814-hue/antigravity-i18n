/**
 * Locale definition test gate.
 *
 * Runs without Antigravity installed: it validates every shipped locale, builds
 * its fragments, and exercises the translation engine on representative inputs.
 */

const assert = require('assert');
const { buildMenuFragment, buildPreloadFragment, listLocales, loadLocale, validateLocale } = require('../src/locale');

let failures = 0;

function check(name, fn) {
    try {
        fn();
        console.log(`  ok   ${name}`);
    } catch (err) {
        failures += 1;
        console.error(`  FAIL ${name}: ${err.message}`);
    }
}

/**
 * Evaluate the non-DOM half of a generated preload fragment.
 *
 * @param {string} fragment
 * @returns {{ hasTranslation: Function, translateString: Function }}
 */
function loadEngine(fragment) {
    const marker = fragment.indexOf('// DOM traversal layer');
    const core = marker > 0 ? fragment.slice(0, fragment.lastIndexOf('// ---', marker)) : fragment;
    const module_ = { exports: {} };
    const factory = new Function('module', `${core}\nmodule.exports = { hasTranslation, translateString };`);
    factory(module_);
    return module_.exports;
}

console.log('Locale definitions:');
const locales = listLocales();
assert.ok(locales.length > 0, 'no locales found under src/locales');

for (const code of locales) {
    const locale = loadLocale(code);

    check(`${code}: passes validation`, () => validateLocale(code, locale));

    check(`${code}: declares required metadata`, () => {
        assert.ok(locale.language, 'missing language');
        assert.ok(locale.name, 'missing display name');
        assert.ok(Object.keys(locale.text).length > 0, 'empty text dictionary');
    });

    check(`${code}: every dictionary entry is a clean non-empty string`, () => {
        for (const [key, value] of Object.entries(locale.text)) {
            assert.strictEqual(typeof value, 'string', `value for '${key}' is not a string`);
            assert.ok(value.length > 0, `value for '${key}' is empty`);
            assert.strictEqual(key, key.trim(), `key '${key}' has surrounding whitespace`);
        }
    });

    check(`${code}: fragments build with no leftover placeholders`, () => {
        const preload = buildPreloadFragment(locale);
        const menu = buildMenuFragment(locale);
        assert.ok(!preload.includes('LOCALE_DATA_PLACEHOLDER'), 'preload placeholder not replaced');
        assert.ok(!preload.includes('allowSourceEqual'), 'report-only allowlist leaked into runtime data');
        assert.ok(!menu.includes('MENU_DATA_PLACEHOLDER'), 'menu placeholder not replaced');
        assert.ok(preload.includes('installLocalePatch'), 'preload missing install entry point');
    });

    check(`${code}: engine translates dictionary and dynamic patterns`, () => {
        const { hasTranslation, translateString } = loadEngine(buildPreloadFragment(locale));

        const [sampleKey, sampleValue] = Object.entries(locale.text)[0];
        assert.strictEqual(translateString(sampleKey), sampleValue, 'dictionary lookup failed');
        assert.strictEqual(hasTranslation(sampleKey), true, 'hasTranslation disagrees with dictionary');

        const unknown = '__definitely not a ui string__';
        assert.strictEqual(translateString(unknown), unknown, 'unknown text was modified');
        assert.strictEqual(hasTranslation(unknown), false, 'unknown text reported as translatable');

        assert.strictEqual(translateString(''), '', 'empty string altered');
        assert.strictEqual(translateString('   '), '   ', 'whitespace-only string altered');
        assert.strictEqual(translateString(null), null, 'non-string input altered');

        // hasTranslation and translateString must never disagree: anything
        // reported translatable has to actually change.
        for (const rule of locale.patterns || []) {
            if (!rule.sample) continue;
            assert.strictEqual(hasTranslation(rule.sample), true, `pattern '${rule.id}' does not match its own sample`);
            assert.notStrictEqual(translateString(rule.sample), rule.sample, `pattern '${rule.id}' matched but produced no change`);
        }
    });
}

if (failures > 0) {
    console.error(`\n${failures} locale check(s) failed.`);
    process.exit(1);
}
console.log(`\nAll locale checks passed (${locales.length} locale(s)).`);
