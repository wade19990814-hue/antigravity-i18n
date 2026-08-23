# antigravity-zh

[English](./README.md)

一行命令为 [Google Antigravity](https://antigravity.google/) 桌面端开启简体中文界面，支持随时一键还原官方英文。

![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20Linux%20%7C%20macOS-blue?style=flat-square)
![License](https://img.shields.io/badge/license-MIT-green?style=flat-square)
![Node](https://img.shields.io/badge/node-%3E%3D16-brightgreen?style=flat-square)

---

## 特性

- **即用即走**：无需 clone 仓库或手动配置，终端敲一行 `npx` 自动定位路径并完成切换。
- **不碰代码与终端**：只汉化外壳 UI 与原生菜单，编辑器（Monaco）、终端（xterm）和大模型输入输出区域保持原样，不干扰正常开发。
- **逐字节还原**：首次运行自动备份原版 `app.asar`，执行 `en` 可完全恢复官方原版英文。
- **纯本地离线**：不发起任何网络请求，无埋点遥测，不读取账号凭据或密钥。
- **覆盖全面**：包含 600+ 静态界面词条、28+ 原生系统菜单，以及动态状态文本匹配（如 `Thought for 5s`）。

---

## 使用方法

系统需已安装 Node.js (≥16)。

### 常用命令

```bash
# 切换为简体中文
npx antigravity-zh zh

# 还原官方英文原版
npx antigravity-zh en

# 查看当前语言与备份状态
npx antigravity-zh status
```

### 源码运行

```bash
git clone https://github.com/wade19990814-hue/antigravity-zh.git
cd antigravity-zh
npm install
node bin/cli.js zh
```

### 参数说明

```text
命令:
  zh                切换为简体中文
  en                还原官方原版英文
  status            查看当前语言、安装路径与备份状态
  locales           列出内置语言包

选项:
  --app-dir <path>  手动指定 Antigravity 安装路径
  --locale <code>   指定语言包（默认 zh-CN）
  --no-restart      打补丁后不自动重启客户端
  --no-kill         仅在客户端已退出时继续，绝不结束进程
  --force           跳过等待，强制结束客户端进程
  -h, --help        显示帮助
  -v, --version     查看版本
```

---

## 注意事项

1. **运行前保存工作**：修改客户端需要先退出应用。工具会自动等待应用正常退出并保存状态，建议运行前先保存好未完成的代码或对话。
2. **官方版本更新**：Antigravity 自动更新后会覆盖补丁回到英文，更新完成后重新执行一次 `npx antigravity-zh zh` 即可。
3. **备份文件**：首次运行会在应用的 `resources` 目录下生成 `app.asar.clean-backup`；官方更新后会用当前未打补丁的归档刷新该备份。请勿手动删除。

---

## 参与贡献

如果发现漏翻或翻译不准确的地方，欢迎提交 PR。直接在 [`src/locales/zh-CN.json`](./src/locales/zh-CN.json) 的 `text` 字典里修改或添加对应词条即可：

```json
"Original Text": "中文翻译"
```

提交前请运行 `npm test` 确保测试通过。

---

## 免责声明

1. **本项目仅供个人学习、研究与交流使用，请勿用于任何商业用途。**
2. 本项目为独立的第三方开源工具，与 Google 及 Antigravity 官方团队无任何关联，Antigravity 相关商标与版权归其官方所有。
3. 使用本工具修改客户端属于用户个人行为，请在遵守相关服务条款的前提下使用。作者不对使用本工具可能导致的任何软件异常、数据丢失或其他后果承担责任。

---

## 许可证

[MIT](./LICENSE)
