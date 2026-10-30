# antigravity-i18n

適用于 [Google Antigravity](https://antigravity.google/) 桌面客戶端的多語言本地化工具。

[简体中文](./README.zh-CN.md) | [English](./README.md) | [繁體中文](./README.zh-Hant.md) | [日本語](./README.ja.md) | [한국어](./README.ko.md) | [Deutsch](./README.de.md) | [Español](./README.es.md) | [Français](./README.fr.md) | [Português (Brasil)](./README.pt-BR.md) | [Русский](./README.ru.md)

## 專案狀態與平台支援

本儲存庫是後續開發的主要儲存庫。目前 `main` 分支為 Go/Wails v3 版本，現階段**僅支援 Windows**。舊版跨平台 Node.js/npm 實作獨立保留在 [legacy-node 分支](https://github.com/wade19990814-hue/antigravity-i18n/tree/legacy-node)，原有 v1/v2 標籤與發行版本保持不變。

---

單個可執行檔案集成現代化圖形界面、純 Go 實現的 asar 讀寫引擎與完整語言包；無需安裝 Node.js 或任何運行庫，解壓即用，並支持隨時按字節原樣還原為官方原版。

---

## 核心特性

- **單檔案免安裝**：約 12 MB 獨立綠色程序，無需配置 Node.js、npm 或 Python 環境，開箱即用。
- **圖形與命令行雙支持**：預設啟動直觀易用的圖形界面，同時內嵌完整命令行子命令（`apply` / `restore` / `status` / `locales`），兼顧日常使用與腳本自動化。
- **字節級原樣還原**：首次修改前自動保存官方原始歸檔（`app.asar.clean-backup`）；執行還原后，`app.asar` 與官方發布的初始檔案 SHA-256 哈希完全一致。
- **非侵入式設計**：僅本地化外殼 UI、原生系統菜單與退出確認框；代碼編輯器（Monaco）、集成終端機（xterm）與 AI 對話區域完全保持原生，不觸碰任何使用者數據。
- **離線與高可靠**：全流程零網絡請求、零數據上傳；內置腳本語法合法性門禁與全流程事務控制，遇到異常自動安全回滾。

---

## 下載與使用

### 下載運行

前往專案的 [Releases](../../releases) 頁面，下載最新版本的 `antigravity-i18n.exe` 即可直接使用。

### 1. 圖形界面（推薦）

雙擊運行 `antigravity-i18n.exe`：
1. 程序會自動檢測 Antigravity 的安裝目錄（若未檢測到，可點擊“更改位置”手動選擇）。
2. 在語言列表中點擊選中目標語言（選中 **English** 即為還原官方原版）。
3. 點擊“應用”按鈕；工具在確認關閉程序后自動改寫資源並重啟 Antigravity。

### 2. 命令行模式

同一可執行檔案也可直接在終端機中調用：



```powershell
# 查看当前本地化状态与备份信息
antigravity-i18n.exe status

# 安装指定语言包（默认简体中文）
antigravity-i18n.exe apply --locale zh-CN

# 随时还原为官方英文原版
antigravity-i18n.exe restore

# 查看所有内置语言包及其覆盖率
antigravity-i18n.exe locales
```



常用參數：
- `--app-dir <path>`：手動指定 Antigravity 安裝路徑（亦可通過環境變量 `ANTIGRAVITY_APP_DIR` 設定）。
- `--locale <code>`：指定語言代碼，如 `zh-CN`、`zh-Hant`、`ja`、`ko`、`de`、`es`、`fr` 等。
- `--no-restart`：完成改寫后不自動重啟客戶端。
- `--force`：跳過 30 秒優雅退出等待，直接結束 Antigravity 進程。
- `--yes`：靜默確認進程關閉提示。

---

## 內置語言支持

所有語言包均已內嵌于二進制中，無需額外下載：

| 語言代碼 | 語言名稱 | 翻譯覆蓋度 | 說明 |
| :--- | :--- | :--- | :--- |
| `zh-CN` | 簡體中文 | 100% | 完整覆蓋介面外殼、原生功能表與結束彈窗 |
| `zh-Hant` | 繁體中文 | 100% | 完整覆蓋，標準繁體電腦術語 |
| `ja` | 日本語 | 100% | 完整覆蓋，專業 IDE 術語核對與對齊 |
| `ko` | 한국어 | 100% | 完整覆蓋，專業 IDE 術語核对与对齐 |
| `de` | Deutsch | 100% | 完整覆蓋，專業 IDE 術語核对与对齐 |
| `es` | Español | 100% | 完整覆蓋，專業 IDE 術語核对与对齐 |
| `fr` | Français | 100% | 完整覆蓋，專業 IDE 術語核对与对齐 |
| `pt-BR` | Português (Brasil) | 100% | 完整覆蓋，專業 IDE 術語核对与对齐 |
| `ru` | Русский | 100% | 完整覆蓋，專業 IDE 術語核对与对齐 |
| `en` | English | - | 官方原版英文（選中即執行還原） |

---

## 機制與常見問題

### 1. Antigravity 官方更新后失效？
Antigravity 自動更新會覆蓋 `app.asar`。官方更新完成后，重新打開本工具點擊一次“應用”即可重新生效；工具會自動基于新版檔案刷新干凈備份。

### 2. 語言切換與還原
不同語言之間可以直接點擊切換，工具會自動替換現有補丁，無需先手動執行還原。任何時候點擊“還原為官方英文”均可恢復至純凈狀態。

### 3. 備份檔案說明
- `app.asar.clean-backup`：官方純凈歸檔備份，是執行“還原”的核心依賴，**請勿手動刪除**。
- `app.asar.bak-*`：每次操作留下的時間戳快照。當確認當前軟體運行無誤后，可手動清理這些時間戳檔案以釋放磁盤空間。

---

## 源碼構建

如需自行編譯：

**依賴環境**：
- Go 1.26+
- [Wails v2](https://wails.io/) CLI



```powershell
# 克隆仓库
git clone https://github.com/wade19990814-hue/antigravity-i18n.git
cd antigravity-i18n

# 完整构建（同时产出 GUI 与 CLI）
.\build.ps1

# 运行自动化测试（包含真实 asar 还原闭环验证）
$env:ANTIGRAVITY_ASAR = "$env:LOCALAPPDATA\Programs\antigravity\resources\app.asar.clean-backup"
go test ./...
```



---

## 免責聲明

1. 本專案為開源社區維護的非官方本地化工具，與 Google 或 Antigravity 官方無關。
2. Antigravity 及相關商標版權歸原權利人所有。
3. 本工具僅供學習交流及個人使用，請在遵循相關軟體使用協議的前提下合理使用。

## 開源協議

[MIT License](./LICENSE)
