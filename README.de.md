# antigravity-i18n

[English](./README.md) | [简体中文](./README.zh-CN.md) | [日本語](./README.ja.md) | [한국어](./README.ko.md) | [Español](./README.es.md) | Deutsch | [Français](./README.fr.md) | [Português do Brasil](./README.pt-BR.md) | [Русский](./README.ru.md)

Installiere mit einem einzigen Befehl ein UI-Sprachpaket in der Desktop-App [Google Antigravity](https://antigravity.google/) und stelle jederzeit die offizielle Version bytegenau wieder her.

![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20Linux%20%7C%20macOS-blue?style=flat-square)
![License](https://img.shields.io/badge/license-MIT-green?style=flat-square)
![Node](https://img.shields.io/badge/node-%3E%3D16-brightgreen?style=flat-square)

---

## Funktionen

- **Keine dauerhafte Installation**: Direkt mit einem einzigen `npx`-Befehl ausführen. Installationspfad und Neustart der App werden automatisch erledigt.
- **Mehrsprachig**: Alle Sprachdaten liegen in JSON-Paketen unter `src/locales/`. Die Übersetzungs-Engine selbst enthält keine sprachspezifischen Daten. Chinesisch (vereinfacht), Chinesisch (traditionell), Japanisch, Koreanisch, Spanisch, Deutsch, Französisch, brasilianisches Portugiesisch und Russisch sind enthalten.
- **Nicht invasiv**: Nur die App-Oberfläche und native Menüs werden übersetzt. Code-Editor (Monaco), Terminal (xterm) und Unterhaltungsbereiche bleiben unverändert.
- **Bytegenaue Wiederherstellung**: Beim ersten Lauf wird die originale `app.asar` gesichert. `restore` stellt das offizielle Archiv unverändert wieder her.
- **Offline und privat**: Keine Netzwerkanfragen, keine Telemetrie und kein Zugriff auf Tokens, Sitzungen oder Zugangsdaten.
- **Pluralformen und Schreibrichtung**: Mengenabhängige Texte verwenden CLDR-Pluralformen über `Intl.PluralRules`; Sprachen mit Schreibrichtung von rechts nach links können ihre Richtung deklarieren.

---

## Verwendung

Erfordert Node.js 16 oder neuer.

### Schnellstart

```bash
# Deutsch anwenden
npx antigravity-i18n apply --locale de

# 简体中文: npx antigravity-i18n apply --locale zh-CN
# 繁體中文: npx antigravity-i18n apply --locale zh-Hant
# 日本語: npx antigravity-i18n apply --locale ja
# 한국어: npx antigravity-i18n apply --locale ko
# Español: npx antigravity-i18n apply --locale es
# Français: npx antigravity-i18n apply --locale fr
# Português do Brasil: npx antigravity-i18n apply --locale pt-BR
# Русский: npx antigravity-i18n apply --locale ru

# Offizielle Version wiederherstellen
npx antigravity-i18n restore

# Aktive Sprache und Sicherungen anzeigen
npx antigravity-i18n status

# Enthaltene Sprachpakete auflisten
npx antigravity-i18n locales
```

`zh` ist die Kurzform von `apply --locale zh-CN`, `en` die Kurzform von `restore`.

### Aus dem Quellcode ausführen

```bash
git clone https://github.com/wade19990814-hue/antigravity-i18n.git
cd antigravity-i18n
npm install
node bin/cli.js apply --locale de
```

### Optionen

```text
Commands:
  apply             Sprachpaket installieren (Auswahl mit --locale)
  restore           Offizielle unübersetzte Version wiederherstellen
  status            Aktive Sprache und App-Pfad anzeigen
  locales           Enthaltene Sprachpakete auflisten

Options:
  --app-dir <path>  Installationspfad von Antigravity
  --locale <code>   Zu installierendes Sprachpaket (Standard: zh-CN)
  --no-restart      Antigravity nach dem Patchen nicht neu starten
  --no-kill         Antigravity muss bereits beendet sein; Prozess niemals beenden
  --force           App sofort beenden, ohne auf einen sauberen Abschluss zu warten
  -h, --help        Hilfe anzeigen
  -v, --version     Version anzeigen
```

---

## Hinweise

1. **Arbeit speichern**: Das Werkzeug wartet bis zu 20 Sekunden, damit die App ihren Zustand speichern und sauber beendet werden kann. Speichere vorher alle offenen Arbeiten.
2. **Offizielle Updates**: Ein Antigravity-Update überschreibt `app.asar`. Führe danach `apply` erneut aus.
3. **Sicherungsdateien**: Beim ersten Lauf wird `app.asar.clean-backup` im Verzeichnis `resources` erstellt. Nach einem offiziellen Update wird sie automatisch aus dem aktuellen unveränderten Archiv erneuert. Lösche sie nicht manuell. Bei jedem apply/restore wird zusätzlich ein Zeitstempel-Snapshot `app.asar.bak-*` daneben abgelegt, und diese häufen sich mit der Zeit an. Für die Wiederherstellung genügt `app.asar.clean-backup`; sobald sich die aktuelle Installation bewährt hat, kannst du ältere Snapshots löschen, um Speicherplatz freizugeben.
4. **Sprache wechseln**: Ein anderes Sprachpaket ersetzt das aktive Paket direkt. Ein vorheriges `restore` ist nicht nötig.

---

## Mitwirken

Korrekturen und neue Sprachpakete sind willkommen. Die Wörterbuchschlüssel sind die unübersetzten englischen UI-Texte; das enthaltene Referenzpaket dient daher zugleich als Inventar für neue Sprachen.

```bash
# Alle zu übersetzenden englischen Texte ausgeben
node scripts/locale-report.js --keys

# Abdeckung, fehlende Einträge und unübersetzte Platzhalter anzeigen
node scripts/locale-report.js
```

Das Paketformat ist in [CONTRIBUTING.md](./CONTRIBUTING.md) beschrieben. Führe vor einem Pull Request `npm test` aus.

---

## Haftungsausschluss

1. Verwende dieses Projekt nur, soweit dies nach geltendem Recht, Verträgen und den Antigravity-Nutzungsbedingungen zulässig ist. Für die Einhaltung dieser Vorgaben bist du selbst verantwortlich.
2. Dieses unabhängige Open-Source-Werkzeug ist nicht mit Google verbunden und wird von Google weder unterstützt noch autorisiert. Antigravity und zugehörige Marken gehören ihren jeweiligen Rechteinhabern.
3. Änderungen am Client erfolgen auf eigenes Risiko. Die Autoren übernehmen keine Verantwortung für unerwartete Probleme, Datenverlust oder andere Folgen.

---

## Lizenz

[MIT](./LICENSE)
