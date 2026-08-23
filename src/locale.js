const fs = require('fs');
const path = require('path');

const LOCALES_DIR = path.join(__dirname, 'locales');
const PATCHES_DIR = path.join(__dirname, 'patches');
const DEFAULT_LOCALE = 'zh-CN';
const LOCALE_CODE_RE = /^[A-Za-z]{2,3}(?:-[A-Za-z0-9]{2,8})*$/;

/**
 * List the locale codes shipped with this package.
 *
 * @returns {string[]}
 */
function listLocales() {
    try {
        return fs.readdirSync(LOCALES_DIR)
            .filter((name) => LOCALE_CODE_RE.test(name.replace(/\.json$/, '')) && name.endsWith('.json'))
            .map((name) => name.replace(/\.json$/, ''))
            .sort();
    } catch {
        return [];
    }
}

/**
 * Load and validate a locale definition.
 *
 * @param {string} [code=DEFAULT_LOCALE]
 * @returns {object}
 * @throws {Error} When the locale is missing or structurally invalid.
 */
function loadLocale(code = DEFAULT_LOCALE) {
    const availableLocales = listLocales();
    if (typeof code !== 'string' || !LOCALE_CODE_RE.test(code) || !availableLocales.includes(code)) {
        const available = availableLocales.join(', ') || 'none';
        throw new Error(`Unknown locale '${String(code)}'. Available locales: ${available}`);
    }
    const file = path.join(LOCALES_DIR, `${code}.json`);

    let locale;
    try {
        locale = JSON.parse(fs.readFileSync(file, 'utf8'));
    } catch (err) {
        throw new Error(`Locale '${code}' is not valid JSON: ${err.message}`);
    }

    validateLocale(code, locale);
    return locale;
}

/**
 * Reject malformed locale data before it is injected into the app.
 *
 * A bad regex or a template referencing a missing capture group would only
 * surface as a broken UI after app.asar had already been rewritten, so every
 * pattern is compiled and checked here instead.
 *
 * @param {string} code
 * @param {object} locale
 */
function isPlainObject(value) {
    return Boolean(value) && typeof value === 'object' && !Array.isArray(value);
}

function validateStringMap(code, label, value, options = {}) {
    if (!isPlainObject(value)) {
        throw new Error(`Locale '${code}': '${label}' must be an object.`);
    }
    for (const [key, translated] of Object.entries(value)) {
        if (!options.allowEmptyKey && !key) {
            throw new Error(`Locale '${code}': '${label}' contains an empty key.`);
        }
        if (key !== key.trim()) {
            throw new Error(`Locale '${code}': '${label}' key '${key}' has surrounding whitespace.`);
        }
        if (typeof translated !== 'string' || (!options.allowEmptyValue && translated.length === 0)) {
            throw new Error(`Locale '${code}': '${label}' value for '${key}' must be a non-empty string.`);
        }
    }
}

function countCaptureGroups(regex) {
    const flags = regex.flags.replace(/[gy]/g, '');
    return new RegExp(`${regex.source}|`, flags).exec('').length - 1;
}

function validateTemplateGroups(code, label, regex, template) {
    const groupCount = countCaptureGroups(regex);
    for (const match of template.matchAll(/\{(\d+)\}/g)) {
        const index = Number(match[1]);
        if (index < 1 || index > groupCount) {
            throw new Error(
                `Locale '${code}': ${label} template references {${index}} but the pattern has ${groupCount} capture group(s).`
            );
        }
    }
    return groupCount;
}

function validateLocale(code, locale) {
    if (!isPlainObject(locale)) {
        throw new Error(`Locale '${code}' must be a JSON object.`);
    }
    if (!isPlainObject(locale.text)) {
        throw new Error(`Locale '${code}' is missing a 'text' dictionary.`);
    }
    // The 'language' field becomes <html lang> and the CLI's reported code, so a
    // copy-pasted locale file whose field still names the source language would
    // silently mislabel the app.
    if (locale.language !== code) {
        throw new Error(
            `Locale '${code}': 'language' is '${locale.language}' but the file is named '${code}.json'; they must match.`
        );
    }
    if (typeof locale.name !== 'string' || !locale.name.trim()) {
        throw new Error(`Locale '${code}': 'name' must be a non-empty string.`);
    }

    validateStringMap(code, 'text', locale.text);
    if (locale.menu !== undefined) validateStringMap(code, 'menu', locale.menu);
    if (locale.punctuation !== undefined) {
        validateStringMap(code, 'punctuation', locale.punctuation, { allowEmptyKey: true, allowEmptyValue: true });
    }
    if (locale.punctuationSkipSuffixes !== undefined
        && (!Array.isArray(locale.punctuationSkipSuffixes)
            || locale.punctuationSkipSuffixes.some((item) => typeof item !== 'string'))) {
        throw new Error(`Locale '${code}': 'punctuationSkipSuffixes' must be an array of strings.`);
    }

    if (locale.valueMaps !== undefined) {
        if (!isPlainObject(locale.valueMaps)) {
            throw new Error(`Locale '${code}': 'valueMaps' must be an object.`);
        }
        for (const [name, map] of Object.entries(locale.valueMaps)) {
            validateStringMap(code, `valueMaps.${name}`, map, { allowEmptyKey: true, allowEmptyValue: true });
        }
    }

    if (locale.valueRules !== undefined) {
        if (!isPlainObject(locale.valueRules)) {
            throw new Error(`Locale '${code}': 'valueRules' must be an object.`);
        }
        for (const [name, rules] of Object.entries(locale.valueRules)) {
            if (!Array.isArray(rules)) {
                throw new Error(`Locale '${code}': 'valueRules.${name}' must be an array.`);
            }
            rules.forEach((rule, index) => {
                const label = `value rule '${name}' #${index + 1}`;
                if (!isPlainObject(rule) || typeof rule.pattern !== 'string' || typeof rule.template !== 'string') {
                    throw new Error(`Locale '${code}': ${label} needs both 'pattern' and 'template' strings.`);
                }
                let compiled;
                try {
                    compiled = new RegExp(rule.pattern, rule.flags || 'g');
                } catch (err) {
                    throw new Error(`Locale '${code}': ${label} has an invalid regex: ${err.message}`);
                }
                validateTemplateGroups(code, label, compiled, rule.template);
            });
        }
    }

    const patterns = locale.patterns || [];
    if (!Array.isArray(patterns)) {
        throw new Error(`Locale '${code}': 'patterns' must be an array.`);
    }

    const seen = new Set();
    patterns.forEach((rule, index) => {
        const label = rule && rule.id ? `pattern '${rule.id}'` : `pattern #${index + 1}`;
        if (!rule || typeof rule.pattern !== 'string' || typeof rule.template !== 'string') {
            throw new Error(`Locale '${code}': ${label} needs both 'pattern' and 'template' strings.`);
        }
        if (typeof rule.sample !== 'string' || !rule.sample) {
            throw new Error(`Locale '${code}': ${label} needs a non-empty 'sample' string.`);
        }
        if (!rule.pattern.startsWith('^') || !rule.pattern.endsWith('$')) {
            throw new Error(`Locale '${code}': ${label} must be anchored with ^ and $.`);
        }
        if (typeof rule.flags !== 'undefined' && typeof rule.flags !== 'string') {
            throw new Error(`Locale '${code}': ${label} flags must be a string.`);
        }
        if (/[gy]/.test(rule.flags || '')) {
            throw new Error(`Locale '${code}': ${label} cannot use stateful 'g' or 'y' flags.`);
        }
        if (rule.id) {
            if (seen.has(rule.id)) {
                throw new Error(`Locale '${code}': duplicate pattern id '${rule.id}'.`);
            }
            seen.add(rule.id);
        }

        let compiled;
        try {
            compiled = new RegExp(rule.pattern, rule.flags || '');
        } catch (err) {
            throw new Error(`Locale '${code}': ${label} has an invalid regex: ${err.message}`);
        }

        // A template placeholder beyond the pattern's capture-group count would
        // silently render as an empty string in the UI.
        const groupCount = validateTemplateGroups(code, label, compiled, rule.template);
        if (!compiled.test(rule.sample)) {
            throw new Error(`Locale '${code}': ${label} does not match its own sample.`);
        }

        if (rule.replace !== undefined && !isPlainObject(rule.replace)) {
            throw new Error(`Locale '${code}': ${label} 'replace' must be an object.`);
        }
        for (const [capture, name] of Object.entries(rule.replace || {})) {
            const captureIndex = Number(capture);
            if (!Number.isInteger(captureIndex) || captureIndex < 1 || captureIndex > groupCount) {
                throw new Error(`Locale '${code}': ${label} has invalid replace capture '${capture}'.`);
            }
            const known = (isPlainObject(locale.valueMaps)
                && Object.prototype.hasOwnProperty.call(locale.valueMaps, name))
                || (isPlainObject(locale.valueRules)
                    && Object.prototype.hasOwnProperty.call(locale.valueRules, name));
            if (!known) {
                throw new Error(`Locale '${code}': ${label} references unknown value map '${name}'.`);
            }
        }
        if (rule.trimGroups !== undefined
            && (!Array.isArray(rule.trimGroups)
                || rule.trimGroups.some((group) => !Number.isInteger(group) || group < 1 || group > groupCount))) {
            throw new Error(`Locale '${code}': ${label} has an invalid 'trimGroups' capture index.`);
        }
    });
}

/**
 * Build the preload fragment for a locale by injecting its data into the
 * locale-neutral engine template.
 *
 * @param {object} locale
 * @returns {string}
 */
function buildPreloadFragment(locale) {
    const template = fs.readFileSync(path.join(PATCHES_DIR, 'engine.jsfrag'), 'utf8');
    // The engine reads everything except the menu map, which is applied in the
    // main process rather than the renderer.
    const { menu, ...rendererLocale } = locale;
    return template.replace('LOCALE_DATA_PLACEHOLDER', () => JSON.stringify(rendererLocale));
}

/**
 * Build the native menu fragment for a locale.
 *
 * @param {object} locale
 * @returns {string}
 */
function buildMenuFragment(locale) {
    const template = fs.readFileSync(path.join(PATCHES_DIR, 'menu.jsfrag'), 'utf8');
    return template.replace('MENU_DATA_PLACEHOLDER', () => JSON.stringify(locale.menu || {}));
}

module.exports = {
    DEFAULT_LOCALE,
    LOCALES_DIR,
    buildMenuFragment,
    buildPreloadFragment,
    listLocales,
    loadLocale,
    validateLocale,
};
