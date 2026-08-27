/**
 * CLI and locale-plumbing gate for the language-neutral command surface.
 *
 * Runs without Antigravity installed.
 */

const assert = require('assert');
const { parseArgs, describeLanguage } = require('../bin/cli');
const { applyLanguageSwitch, normalizeLanguage } = require('../src/index');
const { DEFAULT_LOCALE, diffLocale, collectSourceStrings, describeLocale, loadLocale, validateLocale } = require('../src/locale');

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

console.log('CLI surface:');

check('apply is the default command and defaults to the reference locale', () => {
    assert.strictEqual(parseArgs([]).command, 'apply');
    assert.strictEqual(parseArgs([]).locale, DEFAULT_LOCALE);
});

check('generic verbs and language shorthands both resolve', () => {
    assert.strictEqual(parseArgs(['apply', '--locale', 'zh-CN']).locale, 'zh-CN');
    assert.strictEqual(parseArgs(['zh']).locale, 'zh-CN');
    assert.strictEqual(parseArgs(['chinese']).locale, 'zh-CN');
    assert.strictEqual(parseArgs(['restore']).command, 'restore');
    assert.strictEqual(parseArgs(['en']).command, 'en');
});

check('a shorthand contradicting --locale is rejected instead of guessing', () => {
    assert.throws(() => parseArgs(['zh', '--locale', 'ja']), /already means/);
    assert.deepStrictEqual(parseArgs(['zh', '--locale', 'zh-CN']).locale, 'zh-CN');
});

check('status names any locale, not just Chinese and English', () => {
    assert.match(describeLanguage('zh-CN', '简体中文'), /简体中文 \(zh-CN\)/);
    assert.match(describeLanguage('en', null), /official/);
    assert.match(describeLanguage('unknown', null), /Unknown/);
    // An archive that predates the locale header supplies no name, so the
    // bundled pack is consulted before falling back to the bare code.
    assert.match(describeLanguage('zh-CN', null), /简体中文/);
    assert.match(describeLanguage('ja', null), /日本語/);
    assert.strictEqual(describeLanguage('xx-YY', null), 'xx-YY');
});

check('a pre-rename state marker still reports its language', () => {
    assert.strictEqual(normalizeLanguage('zh'), 'zh-CN');
    assert.strictEqual(normalizeLanguage('zh-CN'), 'zh-CN');
    assert.strictEqual(normalizeLanguage('en'), 'en');
    assert.strictEqual(normalizeLanguage(undefined), 'unknown');
});

console.log('\nChromium language switch:');

const DEBUG_BLOCK = [
    "if (!electron_1.app.commandLine.hasSwitch('remote-debugging-port')) {",
    "    electron_1.app.commandLine.appendSwitch('remote-debugging-port', '0');",
    '}'
].join('\n');

check('the switch is created from the debug-port anchor', () => {
    const out = applyLanguageSwitch(DEBUG_BLOCK, { language: 'ja' });
    assert.match(out, /appendSwitch\('lang', 'ja'\)/);
    assert.ok(out.includes(DEBUG_BLOCK), 'debug-port block was clobbered');
});

check('switching language rewrites the switch instead of leaving the old one', () => {
    const first = applyLanguageSwitch(DEBUG_BLOCK, { language: 'zh-CN' });
    const second = applyLanguageSwitch(first, { language: 'ja' });
    assert.match(second, /appendSwitch\('lang', 'ja'\)/);
    assert.ok(!second.includes("'zh-CN'"), 'stale language switch survived');
    // One rewrite only: repeated runs must stay byte-stable.
    assert.strictEqual((second.match(/appendSwitch\('lang'/g) || []).length, 1);
    assert.strictEqual(applyLanguageSwitch(second, { language: 'ja' }), second);
});

check('chromiumLang overrides the locale code when supplied', () => {
    const out = applyLanguageSwitch(DEBUG_BLOCK, { language: 'pt-BR', chromiumLang: 'pt' });
    assert.match(out, /appendSwitch\('lang', 'pt'\)/);
});

check('an unusable language code fails before app.asar is touched', () => {
    assert.throws(() => applyLanguageSwitch(DEBUG_BLOCK, { language: "a'); evil(); ('" }), /unusable/);
});

console.log('\nLocale tooling:');

check('the reference pack is its own complete inventory', () => {
    const report = diffLocale(DEFAULT_LOCALE);
    assert.strictEqual(report.missing.text.length, 0);
    assert.strictEqual(report.untranslated.length, 0, 'reference pack has copied placeholders');
    assert.strictEqual(report.coverage, 1);
});

check('the source inventory is pure English, usable for a new language', () => {
    const source = collectSourceStrings();
    assert.ok(source.text.length > 100);
    const nonAscii = source.text.filter((key) => /[^\x00-\x7F]/.test(key));
    assert.deepStrictEqual(nonAscii, [], 'source keys must not contain translated text');
});

check('coverage gaps and copied placeholders are reported', () => {
    const report = diffLocale(DEFAULT_LOCALE);
    assert.ok(Array.isArray(report.missing.menu));
    assert.ok(Array.isArray(report.extra.patterns));
    assert.strictEqual(describeLocale('zh-CN'), loadLocale('zh-CN').name);
    assert.strictEqual(describeLocale('xx-YY'), null);
    assert.deepStrictEqual(diffLocale('es').untranslated, [], 'intentional Spanish cognates were misreported');
});

if (failures > 0) {
    console.error(`\n${failures} CLI check(s) failed.`);
    process.exit(1);
}
console.log('\nAll CLI checks passed.');
