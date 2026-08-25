# polygravity

[English](./README.md)

一行命令为 [Google Antigravity](https://antigravity.google/) 桌面端安装界面语言包，并支持随时按字节还原官方版本。

![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20Linux%20%7C%20macOS-blue?style=flat-square)
![License](https://img.shields.io/badge/license-MIT-green?style=flat-square)
![Node](https://img.shields.io/badge/node-%3E%3D16-brightgreen?style=flat-square)

---

## 特性

- **免安装**：一条 `npx` 命令直接运行，自动识别安装路径并重启应用。
- **多语言**：语言数据全部位于 `src/locales/` 下的 JSON 语言包中，翻译引擎本身不含任何语言内容，新增语言即新增一个文件。简体中文已内置。
- **非侵入**：仅翻译外壳界面与原生菜单，代码编辑器（Monaco）、终端（xterm）与对话区域完全不动。
- **字节级还原**：首次运行会备份原始 `app.asar`，`restore` 可将官方归档原样放回。
- **离线且私有**：无网络请求、无遥测，不接触 token、会话或任何凭据。
- **复数与书写方向**：带计数的字符串通过 `Intl.PluralRules` 选择 CLDR 复数形式；从右向左书写的语言必须声明书写方向。

---

## 使用

需要 Node.js（≥16）。

### 快速开始

```bash
# 安装语言包（默认简体中文）
npx polygravity apply --locale zh-CN

# 还原官方版本
npx polygravity restore

# 查看当前语言与备份状态
npx polygravity status

# 查看内置的语言包
npx polygravity locales
```

`zh` 是 `apply --locale zh-CN` 的简写，`en` 是 `restore` 的简写。

### 从源码运行

```bash
git clone https://github.com/wade19990814-hue/polygravity.git
cd polygravity
npm install
node bin/cli.js apply --locale zh-CN
```

### 参数说明

```text
Commands:
  apply             安装语言包（用 --locale 指定）
  restore           还原未翻译的官方版本
  status            显示当前语言与安装路径
  locales           列出内置语言包

Options:
  --app-dir <path>  指定 Antigravity 安装路径
  --locale <code>   要安装的语言包（默认 zh-CN）
  --no-restart      打补丁后不自动重启
  --no-kill         要求应用已关闭，绝不终止进程
  --force           跳过优雅等待，直接终止应用
  -h, --help        显示帮助
  -v, --version     显示版本
```

---

## 注意事项

1. **先保存工作**：工具最多等待 20 秒让应用自行退出并保存状态，运行前请先保存未完成的内容。
2. **官方版本更新**：Antigravity 自动更新后会覆盖 `app.asar`，更新完成后重新执行一次 `apply` 即可。
3. **备份文件**：首次运行会在 `resources` 目录生成 `app.asar.clean-backup`；官方更新后它会依据当前未打补丁的归档自动刷新，请勿手动删除。
4. **切换语言**：安装另一个语言包会直接替换原有的，中间不需要先 `restore`。

---

## 参与贡献

欢迎补充词条，也欢迎提交新的语言包。词表的键就是未翻译的英文界面文本，因此内置语言包同时也是新语言的对照清单：

```bash
# 打印语言包需要覆盖的全部英文原文
node scripts/locale-report.js --keys

# 查看覆盖率、缺失词条与未翻译的占位内容
node scripts/locale-report.js
```

语言包格式详见 [CONTRIBUTING.md](./CONTRIBUTING.md)，提交 PR 前请运行 `npm test`。

---

## 免责声明

1. **本项目仅供个人学习、研究与技术交流使用，请勿用于商业用途。**
2. 本项目为独立开源工具，与 Google 无隶属、背书或授权关系。Antigravity 及相关商标归其各自所有者所有。
3. 修改客户端存在风险，请自行判断后使用。作者不对由此产生的异常、数据丢失或其他后果承担责任。

---

## 许可

[MIT](./LICENSE)
