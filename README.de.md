# antigravity-i18n


UI-Lokalisierungstool für den Google Antigravity-Desktop-Client.


[简体中文](./README.zh-CN.md) | [English](./README.md) | [繁體中文](./README.zh-Hant.md) | [日本語](./README.ja.md) | [한국어](./README.ko.md) | **Deutsch** | [Español](./README.es.md) | [Français](./README.fr.md) | [Português (Brasil)](./README.pt-BR.md) | [Русский](./README.ru.md)

## Projektstatus und Plattformunterstützung

Dieses Repository ist der Hauptort für die weitere Entwicklung. Der aktuelle Branch `main` enthält die Go/Wails-v3-Anwendung und unterstützt derzeit **nur Windows**. Die ältere plattformübergreifende Node.js/npm-Implementierung bleibt separat im [Branch legacy-node](https://github.com/wade19990814-hue/antigravity-i18n/tree/legacy-node) erhalten; ihre v1/v2-Tags und Releases bleiben unverändert.

---

Eine eigenständige ausführbare Datei, die eine moderne grafische Benutzeroberfläche, eine reine Go-Asar-Engine und vollständige Sprachpakete bündelt. Läuft sofort ohne Abhängigkeiten von Node.js oder Laufzeitbibliotheken und unterstützt jederzeit eine bytegenaue Wiederherstellung des offiziellen Builds.

---

## Hauptmerkmale

- **Einzelne eigenständige Binärdatei**: ~12 MB eigenständige ausführbare Datei – keine Node.js-, npm- oder Python-Laufzeit erforderlich.
- **GUI- und CLI-Dual-Modus**: Öffnet standardmäßig ein sauberes Desktop-Fenster und bettet gleichzeitig vollständige CLI-Unterbefehle („apply“ / „restore“ / „status“ / „locales“) für Skripterstellung und Automatisierung ein.
- **Bytegenaue Wiederherstellung**: Erstellt automatisch ein makelloses Backup („app.asar.clean-backup“), bevor Änderungen vorgenommen werden. Beim Wiederherstellen wird ein Archiv wiederhergestellt, das Byte für Byte mit der offiziellen Version identisch ist (SHA-256-verifiziert).
- **Nicht-invasives Design**: Lokalisiert nur die Shell-Benutzeroberfläche, native Menüs und Beenden-Dialoge. Code-Editoren (Monaco), integrierte Terminals (xterm) und KI-Konversationsströme bleiben strikt unberührt.
- **Offline und zuverlässig**: Keine Netzwerkaktivität, keine Datenerfassung. Ausgestattet mit einer JavaScript-Syntaxvalidierung im Voraus und einem Transaktions-Rollback bei unerwarteten Fehlern.

---

## Download und Nutzung

### Herunterladen

Laden Sie die neueste „antigravity-i18n.exe“ von der Seite „Releases“ herunter und führen Sie sie direkt aus.

### 1. Grafische Oberfläche (empfohlen)

Doppelklicken Sie auf „antigravity-i18n.exe“:
1. Die App erkennt den Antigravity-Installationsordner automatisch (oder klicken Sie auf „Speicherort ändern“, um ihn manuell anzugeben).
2. Klicken Sie auf Ihre bevorzugte Sprache (durch Auswahl von Englisch wird die offizielle Version wiederhergestellt).
3. Klicken Sie auf „Übernehmen“; Das Tool bestätigt die Beendigung des Prozesses, aktualisiert die Ressourcen und startet Antigravity neu.

### 2. Befehlszeilenschnittstelle

Die gleiche Binärdatei kann vom Terminal aus aufgerufen werden:

```powershell
# Check current localization status and backups
antigravity-i18n.exe status

# Install a language pack (Simplified Chinese by default)
antigravity-i18n.exe apply --locale zh-CN

# Restore the official untranslated English version
antigravity-i18n.exe restore

# List all bundled language packs with coverage
antigravity-i18n.exe locales
```

Häufige Optionen:
- `--app-dir <Pfad>`: Geben Sie das Antigravity-Installationsverzeichnis manuell an (oder legen Sie es über `ANTIGRAVITY_APP_DIR` fest).
- `--locale <code>`: Zielsprachencode (z. B. `zh-CN`, `zh-Hant`, `ja`, `ko`, `de`, `es`, `fr`).
- „--no-restart“: Antigravity nach dem Patchen nicht neu starten.
- „--force“: Überspringen Sie die 30-sekündige Wartezeit beim ordnungsgemäßen Herunterfahren und beenden Sie Prozesse sofort.
- „--yes“: Unterdrückt die Bestätigungsaufforderung zum Herunterfahren.

---

## Gebündelte Sprachen

Alle Sprachpakete sind direkt in die ausführbare Datei eingebettet:

| Code | Sprache | Abdeckung | Hinweise |
| :--- | :--- | :--- | :--- |
| `zh-CN` | Simplified Chinese (简体中文) | 100% | Vollständige Abdeckung der Benutzeroberfläche, Menüs und Dialoge |
| `zh-Hant` | Traditional Chinese (繁體中文) | 100% | Vollständige Abdeckung mit verifizierten Fachbegriffen |
| `ja` | Japanese (日本語) | 100% | Vollständige Abdeckung mit verifizierten Fachbegriffen |
| `ko` | Korean (한국어) | 100% | Vollständige Abdeckung mit verifizierten Fachbegriffen |
| `de` | German (Deutsch) | 100% | Vollständige Abdeckung mit verifizierten Fachbegriffen |
| `es` | Spanish (Español) | 100% | Vollständige Abdeckung mit verifizierten Fachbegriffen |
| `fr` | French (Français) | 100% | Vollständige Abdeckung mit verifizierten Fachbegriffen |
| `pt-BR` | Portuguese (Brasil) | 100% | Vollständige Abdeckung mit verifizierten Fachbegriffen |
| `ru` | Russian (Русский) | 100% | Vollständige Abdeckung mit verifizierten Fachbegriffen |
| `en` | English | - | Offizieller Original-Build (zum Wiederherstellen auswählen) |

---

## FAQ & Mechanik

### 1. Was passiert nach einem offiziellen Antigravity-Update?
Offizielle Updates überschreiben „app.asar“. Öffnen Sie einfach dieses Tool erneut und klicken Sie erneut auf „Übernehmen“. Das Tool aktualisiert sein sauberes Backup automatisch mit der neuen offiziellen Version.

### 2. Sprachumschaltung und Wiederherstellung
Sie können direkt zwischen jeder Sprache wechseln; Das Tool ersetzt aktive Patches nahtlos, ohne dass zunächst eine manuelle Wiederherstellung erforderlich ist. Durch Klicken auf „Offizielles Englisch wiederherstellen“ kann der Client jederzeit in einen unberührten Zustand zurückversetzt werden.

### 3. Verwaltung der Sicherungsdateien
- „app.asar.clean-backup“: Das makellose offizielle Backup, das für die Wiederherstellung erforderlich ist. Nicht manuell löschen.
- `app.asar.bak-*`: Historische Snapshots mit Zeitstempel links neben dem Archiv. Sobald Ihre aktuelle Installation als stabil bestätigt wurde, können Sie diese Dateien bedenkenlos löschen, um Speicherplatz freizugeben.

---

## Bauen aus der Quelle

**Voraussetzungen**:
- Gehen Sie 1,26+
- [Wails v2](https://wails.io/) CLI

```powershell
# Clone the repository
git clone https://github.com/wade19990814-hue/antigravity-i18n.git
cd antigravity-i18n

# Build both GUI and CLI binaries
.\build.ps1

# Run tests (including end-to-end asar verification)
$env:ANTIGRAVITY_ASAR = "$env:LOCALAPPDATA\Programs\antigravity\resources\app.asar.clean-backup"
go test ./...
```

---

## Haftungsausschluss

1. Dies ist ein inoffizielles, von der Community gepflegtes Projekt und steht in keiner Verbindung zu Google oder Antigravity, wird von diesen nicht unterstützt oder autorisiert.
2. Google Antigravity und zugehörige Marken gehören ihren jeweiligen Eigentümern.
3. Dieses Tool wird für den persönlichen und pädagogischen Gebrauch bereitgestellt. Bitte stellen Sie sicher, dass Sie die entsprechenden Software-Nutzungsbedingungen einhalten.

## Lizenz

[MIT-Lizenz](./LICENSE)
