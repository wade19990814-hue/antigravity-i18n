# antigravity-i18n

[English](./README.md) | [简体中文](./README.zh-CN.md) | 日本語 | [한국어](./README.ko.md) | [Español](./README.es.md) | [Deutsch](./README.de.md) | [Français](./README.fr.md) | [Português do Brasil](./README.pt-BR.md) | [Русский](./README.ru.md)

1つのコマンドで [Google Antigravity](https://antigravity.google/) デスクトップアプリにUI言語パックを適用し、いつでも公式版へバイト単位で復元できます。

![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20Linux%20%7C%20macOS-blue?style=flat-square)
![License](https://img.shields.io/badge/license-MIT-green?style=flat-square)
![Node](https://img.shields.io/badge/node-%3E%3D16-brightgreen?style=flat-square)

---

## 特長

- **インストール不要**：1つの `npx` コマンドで直接実行でき、インストール先の検出とアプリの再起動も自動で行います。
- **多言語対応**：言語データはすべて `src/locales/` のJSONパックに分離されています。翻訳エンジン自体は特定の言語に依存しません。簡体字中国語、繁体字中国語、日本語、韓国語、スペイン語、ドイツ語、フランス語、ブラジルポルトガル語、ロシア語を内蔵しています。
- **非侵襲的**：シェルUIとネイティブメニューだけを翻訳します。コードエディター（Monaco）、ターミナル（xterm）、会話領域には触れません。
- **バイト単位の復元**：初回実行時に元の `app.asar` をバックアップし、`restore` で公式アーカイブを変更前の状態へ戻します。
- **オフラインかつプライベート**：ネットワーク通信やテレメトリーはなく、トークン、セッション、認証情報にはアクセスしません。
- **複数形と書字方向に対応**：数を含む文字列は `Intl.PluralRules` でCLDRの複数形を選択し、右から左へ書く言語は書字方向を宣言できます。

---

## 使い方

Node.js 16以上が必要です。

### クイックスタート

```bash
# 日本語を適用
npx antigravity-i18n apply --locale ja

# 簡体字中国語: npx antigravity-i18n apply --locale zh-CN
# 繁體中文: npx antigravity-i18n apply --locale zh-Hant
# 한국어: npx antigravity-i18n apply --locale ko
# Español: npx antigravity-i18n apply --locale es
# Deutsch: npx antigravity-i18n apply --locale de
# Français: npx antigravity-i18n apply --locale fr
# Português do Brasil: npx antigravity-i18n apply --locale pt-BR
# Русский: npx antigravity-i18n apply --locale ru

# 公式版へ復元
npx antigravity-i18n restore

# 現在の言語とバックアップ状態を確認
npx antigravity-i18n status

# 内蔵言語パックを一覧表示
npx antigravity-i18n locales
```

`zh` は `apply --locale zh-CN`、`en` は `restore` の短縮形です。

### ソースから実行

```bash
git clone https://github.com/wade19990814-hue/antigravity-i18n.git
cd antigravity-i18n
npm install
node bin/cli.js apply --locale ja
```

### オプション

```text
Commands:
  apply             言語パックを適用（--locale で選択）
  restore           未翻訳の公式版へ復元
  status            現在の言語とアプリのパスを表示
  locales           内蔵言語パックを一覧表示

Options:
  --app-dir <path>  Antigravity のインストール先
  --locale <code>   適用する言語パック（既定値: zh-CN）
  --no-restart      パッチ適用後にアプリを再起動しない
  --no-kill         アプリが終了済みであることを要求し、プロセスを終了しない
  --force           正常終了を待たず、直ちにアプリを終了する
  -h, --help        ヘルプを表示
  -v, --version     バージョンを表示
```

---

## 注意事項

1. **作業を保存してください**：アプリが状態を保存して正常終了できるよう最大20秒待機します。実行前に未保存の作業を保存してください。
2. **公式アップデート**：Antigravityの更新により `app.asar` は上書きされます。更新後にもう一度 `apply` を実行してください。
3. **バックアップファイル**：初回実行時に `resources` 内へ `app.asar.clean-backup` を作成します。公式アップデート後は、現在の未変更アーカイブから自動更新されます。手動で削除しないでください。
4. **言語の切り替え**：別の言語パックを適用すると、現在のパックを直接置き換えます。先に `restore` を実行する必要はありません。

---

## コントリビューション

訳語の修正や新しい言語パックの追加を歓迎します。辞書のキーは未翻訳の英語UI文字列であり、内蔵パックが新しい言語の原文一覧としても機能します。

```bash
# 翻訳対象の英語文字列をすべて表示
node scripts/locale-report.js --keys

# カバレッジ、欠落、未翻訳のプレースホルダーを確認
node scripts/locale-report.js
```

パック形式は [CONTRIBUTING.md](./CONTRIBUTING.md) を参照し、Pull Requestを送る前に `npm test` を実行してください。

---

## 免責事項

1. 適用される法令、契約、およびAntigravityの利用規約で認められる範囲でのみ本プロジェクトを使用してください。利用者は自身の使用が適法かつ適切であることを確認する責任を負います。
2. 本プロジェクトは独立したオープンソースツールであり、Googleとの提携、承認、または許可を受けたものではありません。Antigravityおよび関連する商標は各権利者に帰属します。
3. クライアントの変更は自己責任で行ってください。作者は、予期しない問題、データ損失、その他の結果について責任を負いません。

---

## ライセンス

[MIT](./LICENSE)
