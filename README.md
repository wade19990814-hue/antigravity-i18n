# antigravity-i18n

[简体中文](./README.zh-CN.md)

Install a UI language pack into the [Google Antigravity](https://antigravity.google/) desktop app with one command, and restore the official build byte for byte at any time.

![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20Linux%20%7C%20macOS-blue?style=flat-square)
![License](https://img.shields.io/badge/license-MIT-green?style=flat-square)
![Node](https://img.shields.io/badge/node-%3E%3D16-brightgreen?style=flat-square)

---

## Features

- **Zero-install**: Run directly with a single `npx` command. Finds the installation path and restarts the app for you.
- **Any language**: Language data lives entirely in JSON packs under `src/locales/`. The translation engine holds no language of its own, so adding a language means adding a file. Simplified Chinese, Japanese, Korean, and Spanish ship in the box.
- **Non-invasive**: Translates only the shell UI and native menus. Code editors (Monaco), terminals (xterm), and conversation areas are left alone.
- **Byte-exact restore**: Backs up the original `app.asar` on first run. `restore` puts the official archive back unchanged.
- **Offline & private**: No network requests, no telemetry, and no access to tokens, sessions, or credentials.
- **Plural and direction aware**: Counted strings select CLDR plural forms via `Intl.PluralRules`, and right-to-left languages must declare their writing direction.

---

## Usage

Requires Node.js (≥16).

### Quick start

```bash
# Install a language pack (Simplified Chinese is the default)
npx antigravity-i18n apply --locale zh-CN

# Restore the official build
npx antigravity-i18n restore

# Check the active language and backups
npx antigravity-i18n status

# See what language packs are bundled
npx antigravity-i18n locales
```

`zh` is shorthand for `apply --locale zh-CN`, and `en` is shorthand for `restore`.

### Run from source

```bash
git clone https://github.com/wade19990814-hue/antigravity-i18n.git
cd antigravity-i18n
npm install
node bin/cli.js apply --locale zh-CN
```

### Options

```text
Commands:
  apply             Install a language pack (choose it with --locale)
  restore           Restore the official untranslated app
  status            Show the active language and app path
  locales           List the bundled language packs

Options:
  --app-dir <path>  Antigravity installation path
  --locale <code>   Language pack to install (default: zh-CN)
  --no-restart      Do not restart the app after patching
  --no-kill         Require the app to be stopped; never terminate it
  --force           Skip the graceful wait and terminate the app
  -h, --help        Show help
  -v, --version     Show version
```

---

## Notes

1. **Save your work**: The tool waits up to 20 seconds for the app to exit cleanly. Save unfinished work before running.
2. **Official updates**: An Antigravity update overwrites `app.asar`. Run `apply` again afterwards.
3. **Backup files**: The first run creates `app.asar.clean-backup` under `resources`, and a later official update refreshes it from the current unpatched archive. Do not delete it by hand.
4. **Switching languages**: Applying a different pack replaces the previous one directly; there is no need to `restore` in between.

---

## Contributing

Translations and new language packs are both welcome. The dictionary keys are the untranslated English UI strings, so the bundled pack doubles as the inventory a new language is written against:

```bash
# Print every English string a pack has to cover
node scripts/locale-report.js --keys

# Report coverage, gaps and untranslated placeholders
node scripts/locale-report.js
```

See [CONTRIBUTING.md](./CONTRIBUTING.md) for the pack format, and run `npm test` before submitting a PR.

---

## Disclaimer

1. **This project is for personal learning, study, and research purposes only. Please do not use it for commercial purposes.**
2. This is an independent open-source tool and is not affiliated with, endorsed by, or authorized by Google. Antigravity and related trademarks belong to their respective owners.
3. Modifying the client is at your own risk. The authors assume no responsibility for any unexpected issues, data loss, or other consequences.

---

## License

[MIT](./LICENSE)
