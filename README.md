# antigravity-zh

[简体中文](./README.zh-CN.md)

One-command Simplified Chinese localization for the [Google Antigravity](https://antigravity.google/) desktop app, with instant rollback to the official English version.

![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20Linux%20%7C%20macOS-blue?style=flat-square)
![License](https://img.shields.io/badge/license-MIT-green?style=flat-square)
![Node](https://img.shields.io/badge/node-%3E%3D16-brightgreen?style=flat-square)

---

## Features

- **Zero-install**: Run directly with a single `npx` command. Automatically detects the installation path and restarts the app.
- **Non-invasive**: Translates only the shell UI and native menus. Code editors (Monaco), terminals (xterm), and LLM conversation areas remain completely untouched.
- **Byte-exact restore**: Backs up the original `app.asar` on first run. Run `en` at any time to restore the official pristine files.
- **Offline & private**: Zero network requests, zero telemetry, and no access to tokens, sessions, or credentials.
- **Comprehensive coverage**: Includes 600+ static UI labels, 28+ native menu items, and dynamic status text patterns (e.g. `Thought for 5s`).

---

## Usage

Requires Node.js (≥16).

### Quick start

```bash
# Switch to Simplified Chinese
npx antigravity-zh zh

# Restore official English
npx antigravity-zh en

# Check current status and backups
npx antigravity-zh status
```

### Run from source

```bash
git clone https://github.com/wade19990814-hue/antigravity-zh.git
cd antigravity-zh
npm install
node bin/cli.js zh
```

### Options

```text
Commands:
  zh                Switch to Simplified Chinese
  en                Restore official English
  status            Show language, installation path, and backup status
  locales           List bundled locales

Options:
  --app-dir <path>  Specify custom Antigravity installation path
  --locale <code>   Select locale (default: zh-CN)
  --no-restart      Do not restart the app after patching
  --no-kill         Require the app to be stopped; never terminate it
  --force           Skip graceful wait and force-kill process
  -h, --help        Show help
  -v, --version     Show version
```

---

## Notes

1. **Save your work**: The tool waits up to 20 seconds for the app to exit cleanly and save state. It is recommended to save unfinished work before running.
2. **Official updates**: Official updates overwrite `app.asar`. Simply run `npx antigravity-zh zh` again after updating.
3. **Backup files**: The initial run creates `app.asar.clean-backup` in the `resources` directory, and a later official update refreshes it from the current unpatched archive. Do not delete it manually.

---

## Contributing

Contributions are welcome! To fix or add translations, edit the `text` dictionary in [`src/locales/zh-CN.json`](./src/locales/zh-CN.json):

```json
"Original Text": "Chinese translation"
```

Please run `npm test` before submitting PRs.

---

## Disclaimer

1. **This project is for personal learning, study, and research purposes only. Please do not use it for commercial purposes.**
2. This is an independent open-source tool and is not affiliated with, endorsed by, or authorized by Google. Antigravity and related trademarks belong to their respective owners.
3. Modifying the client is at your own risk. The authors assume no responsibility for any unexpected issues, data loss, or other consequences.

---

## License

[MIT](./LICENSE)
