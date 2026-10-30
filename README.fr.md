# antigravity-i18n


Outil de localisation d'interface utilisateur pour le client de bureau Google Antigravity.


[简体中文](./README.zh-CN.md) | [English](./README.md) | [繁體中文](./README.zh-Hant.md) | [日本語](./README.ja.md) | [한국어](./README.ko.md) | [Deutsch](./README.de.md) | [Español](./README.es.md) | **Français** | [Português (Brasil)](./README.pt-BR.md) | [Русский](./README.ru.md)

## État du projet et plateformes prises en charge

Ce dépôt est le dépôt principal pour les développements à venir. La branche `main` actuelle contient l’application Go/Wails v3 et prend actuellement en charge **Windows uniquement**. L’ancienne implémentation multiplateforme Node.js/npm est conservée séparément dans la [branche legacy-node](https://github.com/wade19990814-hue/antigravity-i18n/tree/legacy-node) ; ses tags et versions v1/v2 restent inchangés.

---

Un exécutable autonome regroupant une interface graphique moderne, un moteur Go asar pur et des packs de langue complets. Fonctionne immédiatement sans aucune dépendance sur Node.js ou les bibliothèques d'exécution, et prend en charge la restauration exacte en octets vers la version officielle à tout moment.

---

## Principales fonctionnalités

- **Binaire autonome unique** : exécutable autonome d'environ 12 Mo – aucun environnement d'exécution Node.js, npm ou Python requis.
- **GUI & CLI Dual Mode** : Ouvre une fenêtre de bureau propre par défaut, tout en intégrant des sous-commandes CLI complètes (`apply` / `restore` / `status` / `locales`) pour les scripts et l'automatisation.
- **Restauration exacte en octets** : crée automatiquement une sauvegarde vierge (`app.asar.clean-backup`) avant toute modification. La restauration récupère une archive identique octet par octet (vérifiée SHA-256) à la version officielle.
- **Conception non invasive** : localise uniquement l'interface utilisateur du shell, les menus natifs et les boîtes de dialogue de fermeture. Les éditeurs de code (Monaco), les terminaux intégrés (xterm) et les flux de conversations IA restent strictement intacts.
- **Hors ligne et fiable** : zéro activité réseau, zéro collecte de données. Équipé d'une validation initiale de la syntaxe JavaScript et d'une restauration transactionnelle en cas d'erreurs inattendues.

---

## Téléchargement et utilisation

### Télécharger

Téléchargez le dernier « antigravity-i18n.exe » depuis la page des versions et exécutez-le directement.

### 1. Interface graphique (recommandée)

Double-cliquez sur « antigravity-i18n.exe » :
1. L'application détecte automatiquement le dossier d'installation d'Antigravity (ou cliquez sur « Modifier l'emplacement » pour le spécifier manuellement).
2. Cliquez sur votre langue préférée (la sélection de l'anglais restaure la version officielle).
3. Cliquez sur « Appliquer » ; l'outil confirme la fin du processus, met à jour les ressources et relance Antigravity.

### 2. Interface de ligne de commande

Le même binaire peut être invoqué depuis le terminal :

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

Options courantes :
- `--app-dir <path>` : spécifiez manuellement le répertoire d'installation d'Antigravity (ou définissez via `ANTIGRAVITY_APP_DIR`).
- `--locale <code>` : code de la langue cible (par exemple, `zh-CN`, `zh-Hant`, `ja`, `ko`, `de`, `es`, `fr`).
- `--no-restart` : Ne relancez pas Antigravity après l'application du correctif.
- `--force` : ignorez l'attente d'arrêt progressif de 30 secondes et terminez immédiatement les processus.
- `--yes` : supprime l'invite de confirmation d'arrêt.

---

## Langues groupées

Tous les modules linguistiques sont intégrés directement dans l'exécutable :

| Code | Langue | Couverture | Remarques |
| :--- | :--- | :--- | :--- |
| `zh-CN` | Simplified Chinese (简体中文) | 100% | Couverture complète de l'interface, des menus et des dialogues |
| `zh-Hant` | Traditional Chinese (繁體中文) | 100% | Couverture complète avec terminologie technique vérifiée |
| `ja` | Japanese (日本語) | 100% | Couverture complète avec terminologie technique vérifiée |
| `ko` | Korean (한국어) | 100% | Couverture complète avec terminologie technique vérifiée |
| `de` | German (Deutsch) | 100% | Couverture complète avec terminologie technique vérifiée |
| `es` | Spanish (Español) | 100% | Couverture complète avec terminologie technique vérifiée |
| `fr` | French (Français) | 100% | Couverture complète avec terminologie technique vérifiée |
| `pt-BR` | Portuguese (Brasil) | 100% | Couverture complète avec terminologie technique vérifiée |
| `ru` | Russian (Русский) | 100% | Couverture complète avec terminologie technique vérifiée |
| `en` | English | - | Version officielle intacte (sélectionnez pour restaurer) |

---

## FAQ et mécanique

### 1. Que se passe-t-il après une mise à jour officielle d'Antigravity ?
Les mises à jour officielles écraseront « app.asar ». Rouvrez simplement cet outil et cliquez à nouveau sur « Appliquer ». L'outil actualisera automatiquement sa sauvegarde propre à l'aide de la nouvelle version officielle.

### 2. Changement de langue et restauration
Vous pouvez basculer directement entre n’importe quelle langue ; l'outil remplace de manière transparente les correctifs actifs sans nécessiter de restauration manuelle au préalable. En cliquant sur « Restaurer l'anglais officiel », le client revient à tout moment à un état intact.

### 3. Gestion des fichiers de sauvegarde
- `app.asar.clean-backup` : la sauvegarde officielle vierge requise pour la restauration. Ne supprimez pas manuellement.
- `app.asar.bak-*` : instantanés horodatés historiques laissés à côté de l'archive. Une fois votre installation actuelle vérifiée stable, vous pouvez supprimer ces fichiers en toute sécurité pour libérer de l'espace disque.

---

## Construire à partir de la source

**Prérequis** :
- Allez 1.26+
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

## Clause de non-responsabilité

1. Il s'agit d'un projet non officiel maintenu par la communauté et n'est ni affilié, ni approuvé ni autorisé par Google ou Antigravity.
2. Google Antigravity et les marques associées appartiennent à leurs propriétaires respectifs.
3. Cet outil est fourni pour un usage personnel et éducatif. Veuillez vous assurer du respect des conditions de service du logiciel concerné.

## Licence

[Licence MIT](./LICENCE)
