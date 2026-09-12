#!/usr/bin/env node
/**
 * Language pack coverage report.
 *
 * The dictionary keys are the untranslated English UI strings, so the default
 * pack doubles as the source-of-truth inventory. This prints what a pack is
 * still missing, which is otherwise only discoverable by reading another
 * language's file by hand.
 *
 * Usage:
 *   node scripts/locale-report.js              report every bundled pack
 *   node scripts/locale-report.js ja            report one pack
 *   node scripts/locale-report.js ja --list     also list every missing key
 *   node scripts/locale-report.js --keys        print the source inventory
 */

const { DEFAULT_LOCALE, collectSourceStrings, diffLocale, listLocales, loadLocale } = require('../src/locale');

const args = process.argv.slice(2);
const wantList = args.includes('--list');
const wantKeys = args.includes('--keys');
const requested = args.filter((arg) => !arg.startsWith('-'));
const unknownFlags = args.filter((arg) => arg.startsWith('-') && !['--list', '--keys'].includes(arg));
if (unknownFlags.length > 0) {
    console.error(`Unknown option: ${unknownFlags.join(', ')}`);
    process.exit(1);
}

if (wantKeys) {
    const source = collectSourceStrings();
    for (const key of source.text) {
        console.log(key);
    }
    process.exit(0);
}

const targets = requested.length > 0 ? requested : listLocales();
if (targets.length === 0) {
    console.error('No bundled language packs found.');
    process.exit(1);
}
let incomplete = 0;

for (const code of targets) {
    let pack;
    try {
        pack = loadLocale(code);
    } catch (err) {
        console.error(`${code}: ${err.message}`);
        incomplete += 1;
        continue;
    }

    const report = diffLocale(code);
    const percent = (report.coverage * 100).toFixed(1);
    const flags = [];
    if (code === DEFAULT_LOCALE) flags.push('reference');
    if (pack.dir === 'rtl') flags.push('rtl');

    console.log(`\n${pack.name} (${code})${flags.length ? '  [' + flags.join(', ') + ']' : ''}`);
    console.log(`  text coverage     ${percent}%  (${report.missing.text.length} missing of ${report.missing.text.length + Object.keys(pack.text).length - report.extra.text.length})`);
    console.log(`  menu missing      ${report.missing.menu.length}`);
    console.log(`  dialogs missing   ${report.missing.dialogs.length}`);
    console.log(`  patterns missing  ${report.missing.patterns.length}`);
    if (report.untranslated.length > 0) {
        console.log(`  untranslated      ${report.untranslated.length} (value identical to the English key)`);
    }
    if (report.extra.text.length > 0) {
        console.log(`  not in reference  ${report.extra.text.length} (new strings, or stale after an app update)`);
    }

    if (wantList) {
        for (const [label, keys] of [['missing text', report.missing.text], ['missing menu', report.missing.menu], ['missing dialogs', report.missing.dialogs], ['missing patterns', report.missing.patterns], ['untranslated', report.untranslated], ['not in reference', report.extra.text]]) {
            if (keys.length === 0) continue;
            console.log(`\n  ${label}:`);
            for (const key of keys) {
                console.log(`    ${key}`);
            }
        }
    }

    if (Object.values(report.missing).some((keys) => keys.length > 0) || report.untranslated.length > 0) {
        incomplete += 1;
    }
}

console.log(`\n${targets.length} pack(s) reported, ${incomplete} incomplete.`);
if (!wantList && incomplete > 0) {
    console.log('Re-run with --list to see the exact keys.');
}
process.exitCode = incomplete > 0 ? 1 : 0;
