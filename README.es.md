# antigravity-i18n

[English](./README.md) | [简体中文](./README.zh-CN.md) | [日本語](./README.ja.md) | [한국어](./README.ko.md) | Español | [Deutsch](./README.de.md) | [Français](./README.fr.md) | [Português do Brasil](./README.pt-BR.md) | [Русский](./README.ru.md)

Instala un paquete de idioma para la interfaz de la aplicación de escritorio [Google Antigravity](https://antigravity.google/) con un solo comando y restaura la versión oficial byte por byte cuando quieras.

![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20Linux%20%7C%20macOS-blue?style=flat-square)
![License](https://img.shields.io/badge/license-MIT-green?style=flat-square)
![Node](https://img.shields.io/badge/node-%3E%3D16-brightgreen?style=flat-square)

---

## Características

- **Sin instalación permanente**: ejecútalo directamente con un único comando `npx`. Detecta la ruta de instalación y reinicia la aplicación automáticamente.
- **Multilingüe**: los datos de cada idioma viven en paquetes JSON bajo `src/locales/`. El motor de traducción no contiene datos específicos de ningún idioma. Incluye chino simplificado, japonés, coreano, español, alemán, francés, portugués de Brasil y ruso.
- **No invasivo**: traduce únicamente la interfaz general y los menús nativos. No modifica el editor de código (Monaco), la terminal (xterm) ni las áreas de conversación.
- **Restauración exacta**: guarda una copia del `app.asar` original durante la primera ejecución. `restore` recupera el archivo oficial byte por byte.
- **Privado y sin conexión**: no realiza solicitudes de red, no incluye telemetría y no accede a tokens, sesiones ni credenciales.
- **Compatible con plurales y dirección de escritura**: las cadenas con cantidades seleccionan las formas plurales CLDR mediante `Intl.PluralRules`; los idiomas de derecha a izquierda pueden declarar su dirección de escritura.

---

## Uso

Requiere Node.js 16 o posterior.

### Inicio rápido

```bash
# Aplicar español
npx antigravity-i18n apply --locale es

# 简体中文: npx antigravity-i18n apply --locale zh-CN
# 日本語: npx antigravity-i18n apply --locale ja
# 한국어: npx antigravity-i18n apply --locale ko
# Deutsch: npx antigravity-i18n apply --locale de
# Français: npx antigravity-i18n apply --locale fr
# Português do Brasil: npx antigravity-i18n apply --locale pt-BR
# Русский: npx antigravity-i18n apply --locale ru

# Restaurar la versión oficial
npx antigravity-i18n restore

# Consultar el idioma activo y las copias de seguridad
npx antigravity-i18n status

# Ver los paquetes de idioma incluidos
npx antigravity-i18n locales
```

`zh` es una abreviatura de `apply --locale zh-CN` y `en` es una abreviatura de `restore`.

### Ejecutar desde el código fuente

```bash
git clone https://github.com/wade19990814-hue/antigravity-i18n.git
cd antigravity-i18n
npm install
node bin/cli.js apply --locale es
```

### Opciones

```text
Commands:
  apply             Instala un paquete de idioma (selecciónalo con --locale)
  restore           Restaura la versión oficial sin traducir
  status            Muestra el idioma activo y la ruta de la aplicación
  locales           Enumera los paquetes de idioma incluidos

Options:
  --app-dir <path>  Ruta de instalación de Antigravity
  --locale <code>   Paquete de idioma que se instalará (predeterminado: zh-CN)
  --no-restart      No reinicia Antigravity después de aplicar el parche
  --no-kill         Exige que Antigravity ya esté cerrado y nunca finaliza el proceso
  --force           Finaliza la aplicación inmediatamente, sin esperar un cierre normal
  -h, --help        Muestra la ayuda
  -v, --version     Muestra la versión
```

---

## Notas

1. **Guarda tu trabajo**: la herramienta espera hasta 20 segundos para que la aplicación se cierre normalmente y guarde su estado. Guarda cualquier trabajo pendiente antes de ejecutarla.
2. **Actualizaciones oficiales**: una actualización de Antigravity sobrescribe `app.asar`. Vuelve a ejecutar `apply` después de actualizar.
3. **Copias de seguridad**: la primera ejecución crea `app.asar.clean-backup` en el directorio `resources`. Después de una actualización oficial se renueva automáticamente desde el archivo actual sin modificar. No la elimines manualmente.
4. **Cambio de idioma**: al aplicar otro paquete se reemplaza directamente el anterior; no hace falta ejecutar `restore` primero.

---

## Contribuir

Se agradecen tanto las correcciones de traducción como los nuevos paquetes de idioma. Las claves del diccionario son las cadenas originales en inglés, por lo que el paquete incluido también sirve como inventario para crear otro idioma.

```bash
# Mostrar todas las cadenas en inglés que deben traducirse
node scripts/locale-report.js --keys

# Mostrar cobertura, ausencias y marcadores todavía sin traducir
node scripts/locale-report.js
```

Consulta el formato en [CONTRIBUTING.md](./CONTRIBUTING.md) y ejecuta `npm test` antes de enviar un Pull Request.

---

## Aviso legal

1. Usa este proyecto únicamente cuando lo permitan las leyes, los contratos y los términos de Antigravity aplicables. Eres responsable de comprobar que tu uso sea conforme.
2. Este es un proyecto independiente de código abierto y no está afiliado, respaldado ni autorizado por Google. Antigravity y las marcas relacionadas pertenecen a sus respectivos titulares.
3. Modificar el cliente corre por tu cuenta y riesgo. Los autores no asumen responsabilidad por problemas inesperados, pérdida de datos ni otras consecuencias.

---

## Licencia

[MIT](./LICENSE)
