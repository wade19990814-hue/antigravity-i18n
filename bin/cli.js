#!/usr/bin/env node

const { applyLocale, restoreOfficial, getStatus } = require('../src/index');
const { DEFAULT_LOCALE, describeLocale, listLocales, loadLocale } = require('../src/locale');

// Language-specific shorthands kept from the pre-rename CLI so existing scripts
// and documented one-liners keep working. Each maps onto the generic verbs.
const APPLY_ALIASES = new Map([
    ['apply', null],
    ['zh', 'zh-CN'],
    ['cn', 'zh-CN'],
    ['chinese', 'zh-CN']
]);
const RESTORE_ALIASES = new Set(['restore', 'en', 'english', 'official']);

function printHelp() {
    console.log(`
antigravity-i18n - Antigravity Desktop App localization CLI
Applies a bundled language pack to Antigravity and restores the official build.

Usage:
  npx antigravity-i18n <command> [options]
  node bin/cli.js <command> [options]

Commands:
  apply             Install a language pack (choose it with --locale)
  restore           Restore the official untranslated app
  status            Show the active language and app path
  locales           List the bundled language packs

Options:
  --app-dir <path>  Installation directory of Antigravity
  --locale <code>   Language pack to install (default: ${DEFAULT_LOCALE})
  --no-restart      Do not restart Antigravity after patching
  --no-kill         Require Antigravity to be closed already; never terminate it
  --force           Terminate Antigravity immediately instead of waiting
  -h, --help        Show this help message
  -v, --version     Show package version

The app gets 30 seconds to exit before it is force-closed. Save your work first.

Examples:
  npx antigravity-i18n apply --locale zh-CN
  npx antigravity-i18n restore
  npx antigravity-i18n locales
  node bin/cli.js apply --app-dir "C:\\Users\\YourName\\AppData\\Local\\Programs\\antigravity"

Shorthands: 'zh' applies zh-CN and 'en' restores the official build.
`);
}

function parseArgs(args) {
    let command = 'apply';
    let appDir = null;
    let restart = true;
    let noKill = false;
    let force = false;
    let locale = null;
    const positionals = [];

    const readValue = (index, option) => {
        const value = args[index + 1];
        if (!value || value.startsWith('-')) {
            throw new Error(`${option} requires a value.`);
        }
        return value;
    };

    for (let i = 0; i < args.length; i++) {
        const arg = args[i];
        if (arg === '-h' || arg === '--help' || arg === 'help') {
            printHelp();
            process.exit(0);
        } else if (arg === '-v' || arg === '--version') {
            const pkg = require('../package.json');
            console.log(`v${pkg.version}`);
            process.exit(0);
        } else if (arg === '--app-dir') {
            appDir = readValue(i, arg);
            i += 1;
        } else if (arg === '--locale') {
            locale = readValue(i, arg);
            i += 1;
        } else if (arg === '--no-restart') {
            restart = false;
        } else if (arg === '--no-kill') {
            noKill = true;
        } else if (arg === '--force') {
            force = true;
        } else if (arg.startsWith('-')) {
            throw new Error(`Unknown option: ${arg}`);
        } else {
            positionals.push(arg);
        }
    }

    if (positionals.length > 1) {
        throw new Error(`Unexpected arguments: ${positionals.slice(1).join(' ')}`);
    }
    if (positionals.length === 1) {
        command = positionals[0].toLowerCase();
    }
    if (force && noKill) {
        throw new Error('--force cannot be combined with --no-kill.');
    }

    // A shorthand names its own locale, so combining it with a different
    // --locale would be ambiguous about which language to install.
    const aliasLocale = APPLY_ALIASES.get(command);
    if (aliasLocale && locale && locale !== aliasLocale) {
        throw new Error(`Command '${command}' already means --locale ${aliasLocale}; drop one of them.`);
    }

    return {
        command,
        appDir,
        restart,
        noKill,
        force,
        locale: locale || aliasLocale || DEFAULT_LOCALE
    };
}

/**
 * Render the detected language for the status output.
 *
 * @param {string} language Locale code, 'en', or 'unknown'.
 * @param {string|null} localeName Display name read from the patched archive.
 */
function describeLanguage(language, localeName) {
    if (!language || language === 'unknown') {
        return 'Unknown (could not read app.asar)';
    }
    if (language === 'en') {
        return 'English (official, unpatched)';
    }
    // The archive names its own language; a bundled pack can supply the label
    // for an install whose archive predates the locale header.
    const name = localeName || describeLocale(language);
    return name ? `${name} (${language})` : language;
}

function describeBackup(status) {
    return {
        valid: 'Verified for the current app version (Ready for 1-click restore)',
        missing: 'Not yet created',
        invalid: 'Invalid or unreadable; a verified backup is required',
        patched: 'Contains a localization patch; cannot restore from this copy',
        'version-mismatch': 'Does not match the current app version; cannot restore from this copy'
    }[status] || 'Unknown (not verified)';
}

async function main() {
    try {
        const { command, appDir, restart, noKill, force, locale } = parseArgs(process.argv.slice(2));

        if (command === 'locales') {
            console.log('\nBundled language packs:');
            for (const code of listLocales()) {
                const data = loadLocale(code);
                const entries = Object.keys(data.text || {}).length;
                const patterns = (data.patterns || []).length;
                const isDefault = code === DEFAULT_LOCALE ? '  (default)' : '';
                console.log(`  ${code.padEnd(8)} ${data.name || ''} - ${entries} entries, ${patterns} dynamic rules${isDefault}`);
            }
            console.log('\nApply one with: npx antigravity-i18n apply --locale <code>\n');
            return;
        }

        if (command === 'status') {
            const status = getStatus({ appDir });
            console.log('\n--- Antigravity Localization Status ---');
            console.log(`App Directory:    ${status.appDir}`);
            console.log(`ASAR File:        ${status.asarPath}`);
            console.log(`Current Language: ${describeLanguage(status.currentLanguage, status.localeName)}`);
            console.log(`Clean Backup:     ${describeBackup(status.cleanBackupStatus)}`);
            if (status.lastPatchedAt) {
                console.log(`Last Switched:    ${status.lastPatchedAt}`);
            }
            if (status.stateIsStale) {
                console.log('Note:             app.asar changed outside this tool (likely an Antigravity update).');
            }
            console.log('----------------------------------------\n');
            return;
        }

        if (APPLY_ALIASES.has(command)) {
            const pack = loadLocale(locale);
            console.log(`\n>>> Applying ${pack.name} (${pack.language}) to Antigravity...`);
            await applyLocale({ appDir, restart, noKill, force, locale });
        } else if (RESTORE_ALIASES.has(command)) {
            console.log('\n>>> Restoring the official Antigravity build...');
            restoreOfficial({ appDir, restart, noKill, force });
        } else {
            console.error(`\nUnknown command: "${command}"`);
            printHelp();
            process.exit(1);
        }
    } catch (err) {
        console.error(`\n[Error] ${err.message}`);
        process.exit(1);
    }
}

if (require.main === module) {
    main();
}

module.exports = { parseArgs, describeLanguage, describeBackup };
