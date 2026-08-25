# 贡献指南

欢迎补充词条、修正译法、适配新版本界面，或提交新的语言包。

## 本地开发

1. **Fork 本仓库** 并克隆到本地：
   ```bash
   git clone https://github.com/wade19990814-hue/antigravity-i18n.git
   cd antigravity-i18n
   ```

2. **运行测试**：
   ```bash
   npm test
   ```

3. **本地验证**：
   在外部终端运行本地 CLI 测试语言切换：
   ```bash
   node bin/cli.js apply --locale zh-CN
   node bin/cli.js restore
   ```

---

## 新增一种语言

翻译引擎（[`src/patches/engine.jsfrag`](./src/patches/engine.jsfrag)）不含任何语言内容，
所有语言数据都在 `src/locales/<code>.json` 里，因此新增语言不需要改动任何代码。

1. **导出需要翻译的英文原文**。词表的键就是未翻译的界面文本：
   ```bash
   node scripts/locale-report.js --keys > strings.txt
   ```

2. **新建 `src/locales/<code>.json`**，文件名必须与内部的 `language` 字段一致（例如 `ja.json` 对应 `"language": "ja"`）。
   字段含义与约束全部记录在 [`src/locales/locale.schema.json`](./src/locales/locale.schema.json)，
   支持 JSON Schema 的编辑器会据此提供补全与实时校验。

3. **查看覆盖情况**：
   ```bash
   node scripts/locale-report.js <code> --list
   ```
   报告会列出缺失的词条，以及"值与英文键完全相同"的占位条目——后者能通过校验，但并不是真正的翻译。

4. **运行 `npm test`**。校验会在改写 `app.asar` 之前拦截结构性问题，因此一个错误的语言包不会先把应用改坏再报错。

### 语言包的必填与常用字段

| 字段 | 说明 |
| --- | --- |
| `language` | BCP 47 语言代码，必须与文件名一致 |
| `name` | 用该语言自身书写的显示名称，例如 `简体中文` |
| `text` | 静态词表：英文原文 → 译文 |
| `dir` | 书写方向 `ltr` / `rtl`。**从右向左的语言必须设为 `rtl`**，否则校验会直接拒绝 |
| `chromiumLang` | 传给 Chromium `--lang` 的值，影响原生对话框、拼写检查与日期格式，默认取 `language` |
| `pluralLocale` | 当区域代码无法被 `Intl.PluralRules` 解析时，指定借用哪种语言的复数规则 |

---

## 词条规范

### 1. 静态词条
在 `text` 字段中添加键值对：
```json
"English Text": "译文"
```
* **去首尾空格**：DOM 遍历时会对文本做 `.trim()`，键**请勿带有首尾多余空格**。
* **键必须是英文原文**：从已翻译的界面里采集词条会把译文混进键里，导致该条目在其他语言中永远无法命中。
* **专业术语保留**：专有名词（如 `skills`、`MCP`、`Monaco` 等）通常保留原词，不建议生硬机翻。
* **译文可以包含 `$`**：替换按字面插入，`$&`、`` $` ``、`$1` 等不会被当作替换模式解释。

### 2. 带有超链接/拆分 DOM 的长句
如果一个长句中间包含 `<a>` 链接（例如 `Google Chrome`），React 会将其拆分为多个独立的 TextNode。请将前半句和后半句**分别**作为一个独立的键加入 `text`。

### 3. 动态参数/正则规则
对于带有时间、数字、邮箱等变量的字符串，在 `patterns` 数组中新增一条规则即可：

```json
{
  "id": "tasksRunning",
  "pattern": "^(\\d+) tasks? running$",
  "flags": "i",
  "template": "{1} 个任务正在运行",
  "sample": "7 tasks running"
}
```

* `template` 中用 `{1}`、`{2}` 引用正则的捕获组。
* 若某个捕获组需要按值翻译（如时间单位 `s` → `秒`），在 `replace` 中指向 `valueMaps` 里的一张表：
  `"replace": { "2": "durationUnit" }`。
* 若需要在捕获组内部做多次替换（如 `2 days, 3 hours`），使用 `valueRules`。
* `sample` 是必填的自测样例：`npm test` 会验证该规则确实匹配它并产生了变化，
  避免出现"判定可翻译但实际没翻译"的不一致。

### 4. 复数形式

中文、日文等没有复数一致关系的语言用单个 `template` 即可。俄语、波兰语、阿拉伯语等需要多种形式时，
把 `template` 换成 `templates`，按 CLDR 复数类别（`zero` / `one` / `two` / `few` / `many` / `other`）分别给出：

```json
{
  "id": "tasksRunning",
  "pattern": "^(\\d+) tasks? running$",
  "flags": "i",
  "templates": {
    "one": "{1} задача выполняется",
    "few": "{1} задачи выполняются",
    "many": "{1} задач выполняется",
    "other": "{1} задачи выполняются"
  },
  "sample": "7 tasks running"
}
```

* 类别由 `Intl.PluralRules` 依据该语言选择，因此只需给出这门语言实际用到的形式。
* `other` 是必填的兜底形式；缺少它会被校验拒绝，以免某个类别渲染成空字符串。
* 计数默认取第 1 个捕获组，如需改用其他捕获组请设置 `pluralGroup`。
* 需要针对具体数值特殊表述时，可用 `"=0"`、`"=1"` 这类精确键，它们优先于复数类别。
* `template` 与 `templates` 互斥：同时给出两者会被拒绝，避免两个输出来源相互矛盾。

---

## 提交 Pull Request

1. 创建你的特性分支：
   ```bash
   git checkout -b feat/add-japanese-pack
   ```
2. 提交你的修改并推送到远程仓库：
   ```bash
   git commit -m "feat: add Japanese language pack"
   git push origin feat/add-japanese-pack
   ```
3. 在 GitHub 上发起 Pull Request，简要说明新增或修正的界面与词条内容。
