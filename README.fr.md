# antigravity-i18n

[English](./README.md) | [简体中文](./README.zh-CN.md) | [日本語](./README.ja.md) | [한국어](./README.ko.md) | [Español](./README.es.md) | [Deutsch](./README.de.md) | Français | [Português do Brasil](./README.pt-BR.md) | [Русский](./README.ru.md)

Installez en une seule commande un paquet de langue pour l'interface de l'application de bureau [Google Antigravity](https://antigravity.google/), puis restaurez à tout moment la version officielle à l'octet près.

![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20Linux%20%7C%20macOS-blue?style=flat-square)
![License](https://img.shields.io/badge/license-MIT-green?style=flat-square)
![Node](https://img.shields.io/badge/node-%3E%3D16-brightgreen?style=flat-square)

---

## Fonctionnalités

- **Sans installation permanente** : exécution directe avec une seule commande `npx`. Le chemin d'installation est détecté et l'application est redémarrée automatiquement.
- **Multilingue** : toutes les données linguistiques sont stockées dans des paquets JSON sous `src/locales/`. Le moteur de traduction ne contient aucune donnée propre à une langue. Le chinois simplifié, le chinois traditionnel, le japonais, le coréen, l'espagnol, l'allemand, le français, le portugais brésilien et le russe sont inclus.
- **Non intrusif** : seuls l'interface générale et les menus natifs sont traduits. L'éditeur de code (Monaco), le terminal (xterm) et les zones de conversation restent intacts.
- **Restauration exacte** : le fichier `app.asar` d'origine est sauvegardé lors de la première exécution. `restore` remet en place l'archive officielle sans aucune modification.
- **Hors ligne et privé** : aucune requête réseau, aucune télémétrie et aucun accès aux jetons, sessions ou identifiants.
- **Pluriels et direction d'écriture** : les textes comportant une quantité utilisent les formes plurielles CLDR via `Intl.PluralRules` ; les langues de droite à gauche peuvent déclarer leur direction.

---

## Utilisation

Node.js 16 ou version ultérieure est requis.

### Démarrage rapide

```bash
# Appliquer le français
npx antigravity-i18n apply --locale fr

# 简体中文: npx antigravity-i18n apply --locale zh-CN
# 繁體中文: npx antigravity-i18n apply --locale zh-Hant
# 日本語: npx antigravity-i18n apply --locale ja
# 한국어: npx antigravity-i18n apply --locale ko
# Español: npx antigravity-i18n apply --locale es
# Deutsch: npx antigravity-i18n apply --locale de
# Português do Brasil: npx antigravity-i18n apply --locale pt-BR
# Русский: npx antigravity-i18n apply --locale ru

# Restaurer la version officielle
npx antigravity-i18n restore

# Vérifier la langue active et les sauvegardes
npx antigravity-i18n status

# Afficher les paquets de langue inclus
npx antigravity-i18n locales
```

`zh` est un raccourci pour `apply --locale zh-CN` et `en` pour `restore`.

### Exécuter depuis le code source

```bash
git clone https://github.com/wade19990814-hue/antigravity-i18n.git
cd antigravity-i18n
npm install
node bin/cli.js apply --locale fr
```

### Options

```text
Commands:
  apply             Installer un paquet de langue (sélection avec --locale)
  restore           Restaurer la version officielle non traduite
  status            Afficher la langue active et le chemin de l'application
  locales           Lister les paquets de langue inclus

Options:
  --app-dir <path>  Chemin d'installation d'Antigravity
  --locale <code>   Paquet de langue à installer (par défaut : zh-CN)
  --no-restart      Ne pas redémarrer Antigravity après l'application du correctif
  --no-kill         Exiger qu'Antigravity soit déjà fermé sans jamais arrêter le processus
  --force           Arrêter immédiatement l'application sans attendre sa fermeture normale
  -h, --help        Afficher l'aide
  -v, --version     Afficher la version
```

---

## Remarques

1. **Enregistrez votre travail** : l'outil attend jusqu'à 20 secondes afin que l'application puisse sauvegarder son état et se fermer normalement. Enregistrez tout travail en cours avant de l'exécuter.
2. **Mises à jour officielles** : une mise à jour d'Antigravity remplace `app.asar`. Exécutez de nouveau `apply` après la mise à jour.
3. **Fichiers de sauvegarde** : la première exécution crée `app.asar.clean-backup` dans le dossier `resources`. Après une mise à jour officielle, ce fichier est automatiquement renouvelé depuis l'archive actuelle non modifiée. Ne le supprimez pas manuellement.
4. **Changement de langue** : l'application d'un autre paquet remplace directement le paquet actif ; il n'est pas nécessaire d'exécuter `restore` au préalable.

---

## Contribuer

Les corrections de traduction et les nouveaux paquets de langue sont les bienvenus. Les clés du dictionnaire correspondent aux textes anglais non traduits de l'interface ; le paquet de référence sert donc également d'inventaire pour toute nouvelle langue.

```bash
# Afficher tous les textes anglais à traduire
node scripts/locale-report.js --keys

# Afficher la couverture, les éléments manquants et les valeurs non traduites
node scripts/locale-report.js
```

Consultez [CONTRIBUTING.md](./CONTRIBUTING.md) pour le format des paquets et exécutez `npm test` avant d'envoyer une Pull Request.

---

## Avertissement

1. Utilisez ce projet uniquement dans la mesure permise par les lois, les contrats et les conditions d'utilisation d'Antigravity applicables. Il vous appartient de vérifier la conformité de votre utilisation.
2. Cet outil indépendant et open source n'est ni affilié, ni approuvé, ni autorisé par Google. Antigravity et les marques associées appartiennent à leurs détenteurs respectifs.
3. Toute modification du client s'effectue à vos propres risques. Les auteurs déclinent toute responsabilité en cas de problème inattendu, de perte de données ou d'autres conséquences.

---

## Licence

[MIT](./LICENSE)
