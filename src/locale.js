const fs = require('fs');
const path = require('path');

const LOCALES_DIR = path.join(__dirname, 'locales');
const PATCHES_DIR = path.join(__dirname, 'patches');
const DEFAULT_LOCALE = 'zh-CN';
const LOCALE_CODE_RE = /^[A-Za-z]{2,3}(?:-[A-Za-z0-9]{2,8})*$/;

// Writing directions the injected engine knows how to apply to <html dir>.
const WRITING_DIRECTIONS = ['ltr', 'rtl'];

// CLDR plural categories. A locale supplies only the ones its language uses,
// but 'other' is mandatory because it is the fallback that keeps a missing
// category from rendering as an empty string in the UI.
const PLURAL_CATEGORIES = ['zero', 'one', 'two', 'few', 'many', 'other'];
const EXACT_PLURAL_KEY_RE = /^=\d+$/;

/**
 * Languages written right to left, keyed by primary subtag.
 *
 * Used to reject a locale that would be fully translated yet still laid out
 * left to right, which is a silent visual defect rather than an error.
 */
const RTL_LANGUAGES = new Set(['ar', 'arc', 'ckb', 'dv', 'fa', 'he', 'ku', 'nqo', 'ps', 'sd', 'ug', 'ur', 'yi']);

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

/**
 * Validate a plural rule and return every template it can render.
 *
 * Categories are checked against CLDR's fixed set so a typo such as 'mnay'
 * fails here instead of silently falling through to 'other' at runtime, and
 * 'other' is required because it is the engine's only fallback.
 *
 * @param {string} code
 * @param {string} label
 * @param {object} rule
 * @param {number} groupCount
 * @returns {string[]}
 */
function validatePluralTemplates(code, label, rule, groupCount) {
    if (!isPlainObject(rule.templates)) {
        throw new Error(`Locale '${code}': ${label} 'templates' must be an object of plural forms.`);
    }
    const entries = Object.entries(rule.templates);
    if (entries.length === 0) {
        throw new Error(`Locale '${code}': ${label} 'templates' is empty.`);
    }
    for (const [category, template] of entries) {
        if (!PLURAL_CATEGORIES.includes(category) && !EXACT_PLURAL_KEY_RE.test(category)) {
            throw new Error(
                `Locale '${code}': ${label} has unknown plural form '${category}'; use ${PLURAL_CATEGORIES.join(', ')} or =N.`
            );
        }
        if (typeof template !== 'string' || !template) {
            throw new Error(`Locale '${code}': ${label} plural form '${category}' must be a non-empty string.`);
        }
    }
    if (typeof rule.templates.other !== 'string') {
        throw new Error(
            `Locale '${code}': ${label} must define the 'other' plural form as the required fallback.`
        );
    }
    // The counted group is what selects the plural form, so an out-of-range
    // index would make every string fall back to 'other'.
    const pluralGroup = rule.pluralGroup === undefined ? 1 : rule.pluralGroup;
    if (!Number.isInteger(pluralGroup) || pluralGroup < 1 || pluralGroup > groupCount) {
        throw new Error(
            `Locale '${code}': ${label} has an invalid 'pluralGroup' ${JSON.stringify(rule.pluralGroup)}; the pattern has ${groupCount} capture group(s).`
        );
    }
    return entries.map(([, template]) => template);
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

    // A right-to-left language that does not declare its direction would be
    // fully translated yet still laid out left to right, which is a silent
    // visual defect the injected engine cannot infer on its own.
    if (locale.dir !== undefined && !WRITING_DIRECTIONS.includes(locale.dir)) {
        throw new Error(`Locale '${code}': 'dir' must be one of ${WRITING_DIRECTIONS.join(', ')}.`);
    }
    const primarySubtag = code.split('-')[0].toLowerCase();
    if (RTL_LANGUAGES.has(primarySubtag) && locale.dir !== 'rtl') {
        throw new Error(
            `Locale '${code}': '${primarySubtag}' is written right to left, so 'dir' must be set to 'rtl'.`
        );
    }

    // pluralLocale lets a regional file borrow another language's plural rules
    // when its own code is not a language Intl.PluralRules recognises.
    if (locale.pluralLocale !== undefined
        && (typeof locale.pluralLocale !== 'string' || !LOCALE_CODE_RE.test(locale.pluralLocale))) {
        throw new Error(`Locale '${code}': 'pluralLocale' must be a BCP 47 code.`);
    }

    validateStringMap(code, 'text', locale.text);
    if (locale.allowSourceEqual !== undefined) {
        if (!Array.isArray(locale.allowSourceEqual)
            || locale.allowSourceEqual.some((key) => typeof key !== 'string' || !key)) {
            throw new Error(`Locale '${code}': 'allowSourceEqual' must be an array of non-empty strings.`);
        }
        if (new Set(locale.allowSourceEqual).size !== locale.allowSourceEqual.length) {
            throw new Error(`Locale '${code}': 'allowSourceEqual' cannot contain duplicates.`);
        }
        for (const key of locale.allowSourceEqual) {
            if (!Object.prototype.hasOwnProperty.call(locale.text, key)) {
                throw new Error(`Locale '${code}': 'allowSourceEqual' references unknown text key '${key}'.`);
            }
            if (locale.text[key] !== key) {
                throw new Error(
                    `Locale '${code}': 'allowSourceEqual' key '${key}' must have a value exactly equal to its source.`
                );
            }
        }
    }
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
        if (!rule || typeof rule.pattern !== 'string') {
            throw new Error(`Locale '${code}': ${label} needs a 'pattern' string.`);
        }
        // A rule carries either one template or a plural category map, never
        // both: two sources of output would silently disagree.
        const hasTemplate = typeof rule.template === 'string';
        const hasTemplates = rule.templates !== undefined;
        if (hasTemplate === hasTemplates) {
            throw new Error(
                `Locale '${code}': ${label} needs exactly one of 'template' or 'templates'.`
            );
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
        const groupCount = countCaptureGroups(compiled);
        const templateList = hasTemplate ? [rule.template] : validatePluralTemplates(code, label, rule, groupCount);
        for (const template of templateList) {
            validateTemplateGroups(code, label, compiled, template);
        }
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
    // The engine receives runtime locale data only. The menu map belongs to the
    // main process, while allowSourceEqual is metadata used only by reports.
    const { menu, allowSourceEqual, ...rendererLocale } = locale;
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

/**
 * Look up the display name of a bundled locale without throwing.
 *
 * Used by the CLI to label a language it did not just install, where an
 * unrecognised code should read as unknown rather than abort the command.
 *
 * @param {string} code
 * @returns {string|null}
 */
function describeLocale(code) {
    try {
        return loadLocale(code).name || null;
    } catch {
        return null;
    }
}

/**
 * Collect the English source strings every locale must cover.
 *
 * The dictionary keys are the untranslated UI strings, so the default locale
 * doubles as the source-of-truth inventory. A new language is authored against
 * this list instead of having to be reverse engineered out of another
 * language's file.
 *
 * @param {string} [reference=DEFAULT_LOCALE]
 * @returns {{ text: string[], menu: string[], patterns: string[] }}
 */
function collectSourceStrings(reference = DEFAULT_LOCALE) {
    const locale = loadLocale(reference);
    return {
        text: Object.keys(locale.text).sort(),
        menu: Object.keys(locale.menu || {}).sort(),
        patterns: (locale.patterns || []).map((rule) => rule.id).filter(Boolean).sort()
    };
}

/**
 * Compare a locale against the reference inventory.
 *
 * @param {string} code
 * @param {string} [reference=DEFAULT_LOCALE]
 * @returns {{ missing: object, extra: object, untranslated: string[], coverage: number }}
 */
function diffLocale(code, reference = DEFAULT_LOCALE) {
    const source = collectSourceStrings(reference);
    const locale = loadLocale(code);
    const textKeys = new Set(Object.keys(locale.text));
    const menuKeys = new Set(Object.keys(locale.menu || {}));
    const patternIds = new Set((locale.patterns || []).map((rule) => rule.id).filter(Boolean));

    // An entry whose value equals its key is normally a copied placeholder.
    // Packs must explicitly allowlist legitimate cognates and technical terms.
    const allowedSourceEqual = new Set(locale.allowSourceEqual || []);
    const untranslated = Object.entries(locale.text)
        .filter(([key, value]) => key === value && !allowedSourceEqual.has(key))
        .map(([key]) => key)
        .sort();

    const missing = {
        text: source.text.filter((key) => !textKeys.has(key)),
        menu: source.menu.filter((key) => !menuKeys.has(key)),
        patterns: source.patterns.filter((id) => !patternIds.has(id))
    };
    const extra = {
        text: [...textKeys].filter((key) => !source.text.includes(key)).sort(),
        menu: [...menuKeys].filter((key) => !source.menu.includes(key)).sort(),
        patterns: [...patternIds].filter((id) => !source.patterns.includes(id)).sort()
    };
    const covered = source.text.length - missing.text.length;
    const coverage = source.text.length === 0 ? 1 : covered / source.text.length;
    return { missing, extra, untranslated, coverage };
}

module.exports = {
    DEFAULT_LOCALE,
    LOCALES_DIR,
    PLURAL_CATEGORIES,
    WRITING_DIRECTIONS,
    buildMenuFragment,
    buildPreloadFragment,
    collectSourceStrings,
    describeLocale,
    diffLocale,
    listLocales,
    loadLocale,
    validateLocale,
};
