# antigravity-i18n

适用于 [Google Antigravity](https://antigravity.google/) 桌面客户端的多语言本地化工具。

**简体中文** | [English](./README.md) | [繁體中文](./README.zh-Hant.md) | [日本語](./README.ja.md) | [한국어](./README.ko.md) | [Deutsch](./README.de.md) | [Español](./README.es.md) | [Français](./README.fr.md) | [Português (Brasil)](./README.pt-BR.md) | [Русский](./README.ru.md)

## 项目状态与平台支持

本仓库是后续开发的主仓库。当前 `main` 分支为 Go/Wails v3 版本，现阶段**仅支持 Windows**。旧版跨平台 Node.js/npm 实现单独保留在 [legacy-node 分支](https://github.com/wade19990814-hue/antigravity-i18n/tree/legacy-node)，原有 v1/v2 标签和发行版保持不变。

---

单个可执行文件集成现代化图形界面、纯 Go 实现的 asar 读写引擎与完整语言包；无需安装 Node.js 或任何运行库，解压即用，并支持随时按字节原样还原为官方原版。

---

## 核心特性

- **单文件免安装**：约 12 MB 独立绿色程序，无需配置 Node.js、npm 或 Python 环境，开箱即用。
- **图形与命令行双支持**：默认启动直观易用的图形界面，同时内嵌完整命令行子命令（`apply` / `restore` / `status` / `locales`），兼顾日常使用与脚本自动化。
- **字节级原样还原**：首次修改前自动保存官方原始归档（`app.asar.clean-backup`）；执行还原后，`app.asar` 与官方发布的初始文件 SHA-256 哈希完全一致。
- **非侵入式设计**：仅本地化外壳 UI、原生系统菜单与退出确认框；代码编辑器（Monaco）、集成终端（xterm）与 AI 对话区域完全保持原生，不触碰任何用户数据。
- **离线与高可靠**：全流程零网络请求、零数据上传；内置脚本语法合法性门禁与全流程事务控制，遇到异常自动安全回滚。

---

## 下载与使用

### 下载运行

前往项目的 [Releases](../../releases) 页面，下载最新版本的 `antigravity-i18n.exe` 即可直接使用。

### 1. 图形界面（推荐）

双击运行 `antigravity-i18n.exe`：
1. 程序会自动检测 Antigravity 的安装目录（若未检测到，可点击“更改位置”手动选择）。
2. 在语言列表中点击选中目标语言（选中 **English** 即为还原官方原版）。
3. 点击“应用”按钮；工具在确认关闭程序后自动改写资源并重启 Antigravity。

### 2. 命令行模式

同一可执行文件也可直接在终端中调用：

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

常用参数：
- `--app-dir <path>`：手动指定 Antigravity 安装路径（亦可通过环境变量 `ANTIGRAVITY_APP_DIR` 设置）。
- `--locale <code>`：指定语言代码，如 `zh-CN`、`zh-Hant`、`ja`、`ko`、`de`、`es`、`fr` 等。
- `--no-restart`：完成改写后不自动重启客户端。
- `--force`：跳过 30 秒优雅退出等待，直接结束 Antigravity 进程。
- `--yes`：静默确认进程关闭提示。

---

## 内置语言支持

所有语言包均已内嵌于二进制中，无需额外下载：

| 语言代码 | 语言名称 | 翻译覆盖度 | 说明 |
| :--- | :--- | :--- | :--- |
| `zh-CN` | 简体中文 | 100% | 完整覆盖界面外壳、原生菜单与退出弹窗 |
| `zh-Hant` | 繁體中文 | 100% | 完整覆盖，标准繁体计算机术语 |
| `ja` | 日本語 | 100% | 完整覆盖，专业 IDE 术语核对与对齐 |
| `ko` | 한국어 | 100% | 完整覆盖，专业 IDE 术语核对与对齐 |
| `de` | Deutsch | 100% | 完整覆盖，专业 IDE 术语核对与对齐 |
| `es` | Español | 100% | 完整覆盖，专业 IDE 术语核对与对齐 |
| `fr` | Français | 100% | 完整覆盖，专业 IDE 术语核对与对齐 |
| `pt-BR` | Português (Brasil) | 100% | 完整覆盖，专业 IDE 术语核对与对齐 |
| `ru` | Русский | 100% | 完整覆盖，专业 IDE 术语核对与对齐 |
| `en` | English | - | 官方原版英文（选中即执行还原） |

---

## 机制与常见问题

### 1. Antigravity 官方更新后失效？
Antigravity 自动更新会覆盖 `app.asar`。官方更新完成后，重新打开本工具点击一次“应用”即可重新生效；工具会自动基于新版文件刷新干净备份。

### 2. 语言切换与还原
不同语言之间可以直接点击切换，工具会自动替换现有补丁，无需先手动执行还原。任何时候点击“还原为官方英文”均可恢复至纯净状态。

### 3. 备份文件说明
- `app.asar.clean-backup`：官方纯净归档备份，是执行“还原”的核心依赖，**请勿手动删除**。
- `app.asar.bak-*`：每次操作留下的时间戳快照。当确认当前软件运行无误后，可手动清理这些时间戳文件以释放磁盘空间。

---

## 源码构建

如需自行编译：

**依赖环境**：
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

## 免责声明

1. 本项目为开源社区维护的非官方本地化工具，与 Google 或 Antigravity 官方无关。
2. Antigravity 及相关商标版权归原权利人所有。
3. 本工具仅供学习交流及个人使用，请在遵循相关软件使用协议的前提下合理使用。

## 开源协议

[MIT License](./LICENSE)
