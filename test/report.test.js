const assert = require('assert');
const fs = require('fs');
const path = require('path');
const vm = require('vm');
const { spawnSync } = require('child_process');

const reportPath = path.join(__dirname, '../scripts/locale-report.js');
const source = fs.readFileSync(reportPath, 'utf8');
const run = (...args) => spawnSync(process.execPath, [reportPath, ...args], { encoding: 'utf8' });

assert.strictEqual(run('zh-CN').status, 0);
assert.strictEqual(run('xx').status, 1, 'unknown locale must fail');
assert.strictEqual(run('--typo').status, 1, 'unknown flag must fail');
assert.strictEqual(run('--keys').status, 0);

for (const kind of ['text', 'menu', 'dialogs', 'patterns', 'untranslated']) {
    const report = {
        missing: { text: [], menu: [], dialogs: [], patterns: [] },
        extra: { text: [] }, untranslated: [], coverage: 1
    };
    if (kind === 'untranslated') report.untranslated.push('Missing');
    else report.missing[kind].push('Missing');
    const fakeProcess = { argv: ['node', reportPath, 'ja'], exitCode: 0 };
    vm.runInNewContext(source, {
        process: fakeProcess,
        console: { log() {}, error() {} },
        require: () => ({
            DEFAULT_LOCALE: 'zh-CN', listLocales: () => ['ja'],
            loadLocale: () => ({ name: '日本語', text: {} }), diffLocale: () => report
        })
    });
    assert.strictEqual(fakeProcess.exitCode, 1, `missing ${kind} must fail`);
}
console.log('All report exit-code checks passed.');
