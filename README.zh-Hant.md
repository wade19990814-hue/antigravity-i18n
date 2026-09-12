# antigravity-i18n

[English](./README.md) | [简体中文](./README.zh-CN.md) | 繁體中文 | [日本語](./README.ja.md) | [한국어](./README.ko.md) | [Español](./README.es.md) | [Deutsch](./README.de.md) | [Français](./README.fr.md) | [Português do Brasil](./README.pt-BR.md) | [Русский](./README.ru.md)

一行指令為 [Google Antigravity](https://antigravity.google/) 桌面應用程式安裝介面語言包，並可隨時逐位元組還原官方版本。

![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20Linux%20%7C%20macOS-blue?style=flat-square)
![License](https://img.shields.io/badge/license-MIT-green?style=flat-square)
![Node](https://img.shields.io/badge/node-%3E%3D16-brightgreen?style=flat-square)

---

## 功能

- **免安裝**：直接使用一行 `npx` 指令執行，自動尋找安裝路徑並重新啟動應用程式。
- **多語言**：語言資料皆位於 `src/locales/` 的 JSON 語言包，翻譯引擎本身不含特定語言的內容。內建簡體中文、繁體中文、日本語、한국어、Español、Deutsch、Français、Português do Brasil、Русский。
- **非侵入式**：翻譯外層介面、原生選單及原生退出確認視窗，保留程式碼編輯器（Monaco）、終端機（xterm）與對話區域的內容。
- **逐位元組還原**：首次執行會備份原始 `app.asar`，`restore` 可將官方封存檔原樣還原。
- **離線與隱私**：不發出網路請求、不傳送遙測資料，也不存取權杖、工作階段或憑證。
- **複數與書寫方向**：含數量的字串透過 `Intl.PluralRules` 選擇 CLDR 複數形式；由右至左書寫的語言須宣告其書寫方向。

---

## 使用方式

需要 Node.js（≥16）。

### 快速開始

```bash
# 安裝繁體中文語言包
npx antigravity-i18n apply --locale zh-Hant
# 简体中文: npx antigravity-i18n apply --locale zh-CN
# 日本語: npx antigravity-i18n apply --locale ja
# 한국어: npx antigravity-i18n apply --locale ko
# Español: npx antigravity-i18n apply --locale es
# Deutsch: npx antigravity-i18n apply --locale de
# Français: npx antigravity-i18n apply --locale fr
# Português do Brasil: npx antigravity-i18n apply --locale pt-BR
# Русский: npx antigravity-i18n apply --locale ru

# 還原官方版本
npx antigravity-i18n restore

# 查看目前語言與備份狀態
npx antigravity-i18n status

# 列出內建語言包
npx antigravity-i18n locales
```

未指定 `--locale` 時預設使用簡體中文。`zh` 是 `apply --locale zh-CN` 的簡寫，`en` 是 `restore` 的簡寫。

### 從原始碼執行

```bash
git clone https://github.com/wade19990814-hue/antigravity-i18n.git
cd antigravity-i18n
npm install
node bin/cli.js apply --locale zh-Hant
```

### 參數說明

```text
Commands:
  apply             安裝語言包（使用 --locale 指定）
  restore           還原未翻譯的官方版本
  status            顯示目前語言、備份狀態與安裝路徑
  locales           列出內建語言包

Options:
  --app-dir <path>  指定 Antigravity 安裝路徑
  --locale <code>   要安裝的語言包（預設 zh-CN）
  --no-restart      套用修補後不自動重新啟動
  --no-kill         要求應用程式已關閉，絕不終止處理程序
  --force           跳過等待，直接強制關閉應用程式
  -h, --help        顯示說明
  -v, --version     顯示版本
```

---

## 注意事項

1. **先儲存工作**：工具會要求應用程式正常退出並等待最多 30 秒；逾時仍在執行時會強制關閉。`--force` 會跳過等待。執行前請先儲存未完成的內容。
2. **官方版本更新**：Antigravity 更新會覆寫 `app.asar`，更新完成後重新執行 `apply` 即可。如果套用或還原期間封存檔發生變更，操作會停止；請等待更新完成後再執行。
3. **備份檔案**：首次執行會在 `resources` 目錄產生 `app.asar.clean-backup`；官方更新後，會依目前未修補的封存檔自動更新備份，請勿手動刪除。每次 apply/restore 也會留下帶時間戳記的 `app.asar.bak-*` 快照。還原只需要 `app.asar.clean-backup`，確認目前安裝穩定後可刪除較舊的快照以釋放空間。`status` 會檢查封存檔結構、已有的完整性雜湊、修補標記與版本；損壞或版本不符的備份不會顯示為可還原。
4. **切換語言**：套用另一個語言包會直接取代目前的語言包，不需要先執行 `restore`。

---

## 參與貢獻

歡迎補充詞條、修正翻譯或新增語言包。詞表的鍵即為未翻譯的英文介面文字，可作為新增語言的對照清單：

```bash
# 列出靜態詞表的英文原文
node scripts/locale-report.js --keys

# 查看涵蓋率、缺少的詞條及未翻譯的占位內容
node scripts/locale-report.js
```

報告在語言無效、詞條缺漏或存在未宣告的同值占位內容時會回傳非零結束代碼，可用於自動檢查。語言包格式詳見 [CONTRIBUTING.md](./CONTRIBUTING.md)，提交 PR 前請執行 `npm test`。

---

## 免責聲明

1. 僅可在適用法律、契約及 Antigravity 使用條款允許的範圍內使用本專案；使用者應自行確認使用方式合規。
2. 本專案為獨立開源工具，與 Google 無隸屬、背書或授權關係。Antigravity 及相關商標歸各自權利人所有。
3. 修改用戶端存在風險，請自行判斷後使用。作者不對因此產生的異常、資料遺失或其他後果承擔責任。

---

## 授權

[MIT](./LICENSE)
