/**
 * Translation-correctness gate for language features the reference pack does
 * not exercise: dollar-sign safe substitution, CLDR plural selection, and
 * right-to-left declaration.
 *
 * The engine core is evaluated directly, without a DOM, because these are
 * string-level behaviours.
 */

const assert = require('assert');
const { buildPreloadFragment, loadLocale, validateLocale } = require('../src/locale');

let failures = 0;

function check(name, fn) {
    try {
        fn();
        console.log('  ok   ' + name);
    } catch (err) {
        failures += 1;
        console.error('  FAIL ' + name + ': ' + err.message);
    }
}

/**
 * Evaluate the non-DOM half of a generated fragment.
 */
function loadCore(locale) {
    const fragment = buildPreloadFragment(locale);
    const marker = fragment.indexOf('// DOM traversal layer');
    const core = marker > 0 ? fragment.slice(0, fragment.lastIndexOf('// ---', marker)) : fragment;
    const module_ = { exports: {} };
    new Function('module', core + '\nmodule.exports = { translateString, hasTranslation };')(module_);
    return module_.exports;
}

function pack(overrides) {
    return Object.assign({
        language: 'de',
        name: 'Deutsch',
        text: { Settings: 'Einstellungen' },
        patterns: []
    }, overrides);
}

console.log('Dollar-sign safe substitution:');

// String.prototype.replace interprets $&, $` , $' and $n inside the
// replacement even when the pattern is a plain string. A translation containing
// a dollar sign would otherwise splice the English source back in.
const dollarCases = [
    ['Cost', '$5 per month', '$5 per month'],
    ['Total', 'Gesamt $& Summe', 'Gesamt $& Summe'],
    ['Price', "Preis $' Ende", "Preis $' Ende"],
    ['Amount', 'Montant $1', 'Montant $1'],
    ['Sum', 'Somme $$ fin', 'Somme $$ fin']
];

for (const [key, translation, expected] of dollarCases) {
    check('a translation containing ' + JSON.stringify(translation) + ' is inserted literally', () => {
        const { translateString } = loadCore(pack({ text: { [key]: translation } }));
        assert.strictEqual(translateString(key), expected);
    });
}

check('surrounding whitespace is preserved around a dollar translation', () => {
    const { translateString } = loadCore(pack({ text: { Cost: '$& total' } }));
    assert.strictEqual(translateString('  Cost  '), '  $& total  ');
});

check('a dynamic rule with a dollar template is inserted literally', () => {
    const { translateString } = loadCore(pack({
        text: { Settings: 'Einstellungen' },
        patterns: [{ id: 'price', pattern: '^Costs (\\d+)$', template: '$& kostet {1}', sample: 'Costs 5' }]
    }));
    assert.strictEqual(translateString('Costs 5'), '$& kostet 5');
});

check('a captured value containing a dollar sign survives expansion', () => {
    const { translateString } = loadCore(pack({
        text: { Settings: 'Einstellungen' },
        patterns: [{ id: 'model', pattern: '^Model: (.+)$', template: 'Modell: {1}', sample: 'Model: x' }]
    }));
    assert.strictEqual(translateString('Model: $& v$1'), 'Modell: $& v$1');
});

check('punctuation localization is also dollar safe', () => {
    const { translateString } = loadCore(pack({
        text: { Loading: 'Wird geladen $&' },
        punctuation: { ':': '：' }
    }));
    assert.strictEqual(translateString('Loading:'), 'Wird geladen $&：');
});

console.log('\nCLDR plural selection:');

const russian = pack({
    language: 'ru',
    name: 'Русский',
    text: { Settings: 'Настройки' },
    patterns: [{
        id: 'tasksRunning',
        pattern: '^(\\d+) tasks? running$',
        flags: 'i',
        templates: {
            one: '{1} задача выполняется',
            few: '{1} задачи выполняются',
            many: '{1} задач выполняется',
            other: '{1} задачи выполняются'
        },
        sample: '7 tasks running'
    }]
});

check('Russian selects one / few / many by count', () => {
    const { translateString } = loadCore(russian);
    assert.strictEqual(translateString('1 task running'), '1 задача выполняется');
    assert.strictEqual(translateString('2 tasks running'), '2 задачи выполняются');
    assert.strictEqual(translateString('5 tasks running'), '5 задач выполняется');
});

check('an exact-count form outranks its plural category', () => {
    const { translateString } = loadCore(pack({
        language: 'en',
        name: 'English',
        patterns: [{
            id: 'tasksRunning',
            pattern: '^(\\d+) tasks? running$',
            templates: { '=0': 'nothing running', one: '{1} task running', other: '{1} tasks running' },
            sample: '7 tasks running'
        }]
    }));
    assert.strictEqual(translateString('0 tasks running'), 'nothing running');
    assert.strictEqual(translateString('1 task running'), '1 task running');
    assert.strictEqual(translateString('7 tasks running'), '7 tasks running');
});

check('a grouped thousands count still selects the right form', () => {
    const { translateString } = loadCore(pack({
        language: 'ru',
        name: 'Русский',
        patterns: [{
            id: 'conversations',
            pattern: '^([\\d,]+) active conversations$',
            templates: { one: '{1} диалог', few: '{1} диалога', many: '{1} диалогов', other: '{1} диалога' },
            sample: '70 active conversations'
        }]
    }));
    assert.strictEqual(translateString('1,000 active conversations'), '1,000 диалогов');
});

check('the shipped Russian pack selects real plural forms and inflects refresh durations', () => {
    const { translateString } = loadCore(loadLocale('ru'));
    assert.strictEqual(translateString('1 task running'), 'Выполняется 1 задача');
    assert.strictEqual(translateString('2 tasks running'), 'Выполняются 2 задачи');
    assert.strictEqual(translateString('5 tasks running'), 'Выполняются 5 задач');
    assert.strictEqual(translateString('21 tasks running'), 'Выполняется 21 задача');
    assert.strictEqual(translateString('22 tasks running'), 'Выполняются 22 задачи');
    assert.strictEqual(translateString('25 tasks running'), 'Выполняются 25 задач');
    assert.strictEqual(
        translateString('You have used some of your weekly limit, it will fully refresh in 1 day, 2 hours, 5 minutes.'),
        'Вы использовали часть недельного лимита; он полностью восстановится через 1 день, 2 часа, 5 минут.'
    );
});

check('a language with no plural agreement still works with a single template', () => {
    const { translateString } = loadCore(pack({
        language: 'zh-CN',
        name: '简体中文',
        patterns: [{
            id: 'tasksRunning',
            pattern: '^(\\d+) tasks? running$',
            template: '{1} 个任务正在运行',
            sample: '7 tasks running'
        }]
    }));
    assert.strictEqual(translateString('1 task running'), '1 个任务正在运行');
    assert.strictEqual(translateString('9 tasks running'), '9 个任务正在运行');
});

console.log('\nPlural and direction validation:');

check('template and templates are mutually exclusive', () => {
    assert.throws(() => validateLocale('de', pack({
        patterns: [{ id: 'x', pattern: '^(\\d+) x$', template: 'a {1}', templates: { other: 'b {1}' }, sample: '1 x' }]
    })), /exactly one of/);
    assert.throws(() => validateLocale('de', pack({
        patterns: [{ id: 'x', pattern: '^(\\d+) x$', sample: '1 x' }]
    })), /exactly one of/);
});

check('a missing other form is rejected as having no fallback', () => {
    assert.throws(() => validateLocale('ru', pack({
        language: 'ru',
        patterns: [{ id: 'x', pattern: '^(\\d+) x$', templates: { one: 'a {1}' }, sample: '1 x' }]
    })), /required fallback/);
});

check('a misspelled plural category is rejected instead of silently ignored', () => {
    assert.throws(() => validateLocale('ru', pack({
        language: 'ru',
        patterns: [{ id: 'x', pattern: '^(\\d+) x$', templates: { mnay: 'a {1}', other: 'b {1}' }, sample: '1 x' }]
    })), /unknown plural form/);
});

check('a plural template referencing a missing capture group is rejected', () => {
    assert.throws(() => validateLocale('ru', pack({
        language: 'ru',
        patterns: [{ id: 'x', pattern: '^(\\d+) x$', templates: { other: 'a {2}' }, sample: '1 x' }]
    })), /capture group/);
});

check('an out-of-range pluralGroup is rejected', () => {
    assert.throws(() => validateLocale('ru', pack({
        language: 'ru',
        patterns: [{ id: 'x', pattern: '^(\\d+) x$', templates: { other: 'a {1}' }, pluralGroup: 3, sample: '1 x' }]
    })), /pluralGroup/);
});

check('a right-to-left language must declare its direction', () => {
    // A translated RTL pack without dir would render correctly but lay out
    // left to right, which no other check can detect.
    assert.throws(() => validateLocale('ar', pack({ language: 'ar', name: 'العربية' })), /right to left/);
    assert.doesNotThrow(() => validateLocale('ar', pack({ language: 'ar', name: 'العربية', dir: 'rtl' })));
    assert.doesNotThrow(() => validateLocale('he', pack({ language: 'he', name: 'עברית', dir: 'rtl' })));
});

check('an unknown writing direction is rejected', () => {
    assert.throws(() => validateLocale('de', pack({ dir: 'sideways' })), /must be one of/);
});

check('the engine emits <html dir> only when the pack declares one', () => {
    assert.match(buildPreloadFragment(pack({ language: 'ar', name: 'ع', dir: 'rtl' })), /"dir":"rtl"/);
    assert.ok(!/"dir":/.test(buildPreloadFragment(pack({}))), 'dir leaked into an LTR pack');
});

console.log('\nMetadata and re-translation cycles:');

check('a translation chain ending at a fixed point is accepted', () => {
    // French translates 'App' to 'Application', and 'Application' is itself a
    // source entry whose translation equals itself, so the rewrite stops there.
    assert.doesNotThrow(() => validateLocale('fr', pack({
        language: 'fr',
        name: 'Français',
        text: { App: 'Application', Application: 'Application' },
        allowSourceEqual: ['Application']
    })));
});

check('a translation cycle is rejected before injection', () => {
    assert.throws(() => validateLocale('de', pack({
        text: { Settings: 'Preferences', Preferences: 'Settings' }
    })), /cycle/);
});

check('an unusable htmlLang or chromiumLang is rejected at load time', () => {
    assert.throws(() => validateLocale('de', pack({ htmlLang: '"><script>' })), /htmlLang/);
    assert.throws(() => validateLocale('de', pack({ chromiumLang: "a'); evil(); ('" })), /chromiumLang/);
});

if (failures > 0) {
    console.error('\n' + failures + ' translation check(s) failed.');
    process.exit(1);
}
console.log('\nAll translation checks passed.');
