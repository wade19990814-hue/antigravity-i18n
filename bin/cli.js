#!/usr/bin/env node

const { switchToChinese, switchToEnglish, getStatus } = require('../src/index');
const { DEFAULT_LOCALE, listLocales, loadLocale } = require('../src/locale');

function printHelp() {
    console.log(`
Antigravity Localization CLI (antigravity-zh)
One-command language switcher for Antigravity Desktop App (Windows / Linux / macOS)

Usage:
  npx antigravity-zh <command> [options]
  node bin/cli.js <command> [options]

Commands:
  zh                Switch Antigravity UI to Simplified Chinese (简体中文)
  en                Restore official English version (还原英文官方原版)
  status            Show current localization status and app path
  locales           List the translation locales bundled with this package

Options:
  --app-dir <path>  Specify custom installation directory of Antigravity
  --locale <code>   Translation locale to apply (default: zh-CN; see the locales command)
  --no-restart      Do not restart Antigravity automatically after patching
  --no-kill         Do not stop running Antigravity processes (close it yourself first)
  --force           Skip the graceful-close wait and terminate Antigravity immediately
  -h, --help        Show this help message
  -v, --version     Show package version

Examples:
  npx antigravity-zh zh
  npx antigravity-zh en
  node bin/cli.js zh --app-dir "C:\\Users\\YourName\\AppData\\Local\\Programs\\antigravity"
`);
}

function parseArgs(args) {
    let command = 'zh';
    let appDir = null;
    let restart = true;
    let noKill = false;
    let force = false;
    let locale = DEFAULT_LOCALE;
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

    return { command, appDir, restart, noKill, force, locale };
}

function describeLanguage(language) {
    if (language === 'zh') return '简体中文 (zh-CN)';
    if (language === 'en') return 'English (Official)';
    return 'Unknown (could not read app.asar)';
}

async function main() {
    try {
        const { command, appDir, restart, noKill, force, locale } = parseArgs(process.argv.slice(2));
        if (command === 'locales') {
            console.log('\nBundled translation locales:');
            for (const code of listLocales()) {
                const data = loadLocale(code);
                const entries = Object.keys(data.text || {}).length;
                const patterns = (data.patterns || []).length;
                const isDefault = code === DEFAULT_LOCALE ? '  (default)' : '';
                console.log(`  ${code.padEnd(8)} ${data.name || ''} - ${entries} entries, ${patterns} dynamic rules${isDefault}`);
            }
            console.log('\nApply one with: npx antigravity-zh zh --locale <code>\n');
            return;
        }

        if (command === 'status') {
            const status = getStatus({ appDir });
            console.log('\n--- Antigravity Localization Status ---');
            console.log(`App Directory:    ${status.appDir}`);
            console.log(`ASAR File:        ${status.asarPath}`);
            console.log(`Current Language: ${describeLanguage(status.currentLanguage)}`);
            console.log(`Clean Backup:     ${status.hasCleanBackup ? 'Available (Ready for 1-click restore)' : 'Not yet created'}`);
            if (status.lastPatchedAt) {
                console.log(`Last Switched:    ${status.lastPatchedAt}`);
            }
            if (status.stateIsStale) {
                console.log('Note:             app.asar changed outside this tool (likely an Antigravity update).');
            }
            console.log('----------------------------------------\n');
            return;
        }

        if (command === 'zh' || command === 'cn' || command === 'chinese') {
            console.log('\n>>> Switching Antigravity to Chinese (简体中文)...');
            await switchToChinese({ appDir, restart, noKill, force, locale });
        } else if (command === 'en' || command === 'english' || command === 'restore') {
            console.log('\n>>> Restoring official English version...');
            switchToEnglish({ appDir, restart, noKill, force });
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

module.exports = { parseArgs };
