/**
 * Security and failure-safety regression checks.
 */

const assert = require('assert');
const fs = require('fs');
const os = require('os');
const path = require('path');
const { spawnSync } = require('child_process');
const { parseArgs } = require('../bin/cli');
const { resolveAppPaths } = require('../src/detector');
const {
    atomicReplaceFile,
    createUniqueBackupPath,
    ensureCleanBackupForPatch,
    ensureRestorableCleanBackup,
    restoreOfficial,
    STAGE_DIR_PREFIX,
    readArchiveIdentity,
} = require('../src/index');
const { loadLocale, validateLocale } = require('../src/locale');

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

function withTempDir(fn) {
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'antigravity-i18n-security-'));
    try {
        return fn(dir);
    } finally {
        fs.rmSync(dir, { recursive: true, force: true });
    }
}

function cloneLocale() {
    return JSON.parse(JSON.stringify(loadLocale('zh-CN')));
}

console.log('Security and failure safety:');

check('CLI rejects missing option values and unknown flags', () => {
    assert.throws(() => parseArgs(['zh', '--app-dir']), /requires a value/);
    assert.throws(() => parseArgs(['zh', '--locale', '--force']), /requires a value/);
    assert.throws(() => parseArgs(['zh', '--no-restar']), /Unknown option/);
    assert.throws(() => parseArgs(['zh', 'extra']), /Unexpected arguments/);
});

check('CLI rejects contradictory process-control options', () => {
    assert.throws(() => parseArgs(['zh', '--force', '--no-kill']), /cannot be combined/);
    assert.deepStrictEqual(parseArgs(['en', '--no-restart']), {
        command: 'en', appDir: null, restart: false, noKill: false, force: false, locale: 'zh-CN'
    });
});

check('CLI reports argument errors without an unhandled stack trace', () => {
    const result = spawnSync(process.execPath, [path.join(__dirname, '..', 'bin', 'cli.js'), 'zh', '--unknown'], {
        encoding: 'utf8'
    });
    assert.strictEqual(result.status, 1);
    assert.match(result.stderr, /\[Error\] Unknown option/);
    assert.ok(!result.stderr.includes('\n    at '), result.stderr);
});

check('locale loading cannot escape the bundled locale directory', () => {
    for (const code of ['../package', '..\\package', 'zh-CN/../../package', '', null]) {
        assert.throws(() => loadLocale(code), /Unknown locale/);
    }
});

check('dynamic locale rules must be anchored, sampled and non-stateful', () => {
    const unanchored = cloneLocale();
    unanchored.patterns[0].pattern = 'Thought for (\\d+)s';
    assert.throws(() => validateLocale('zh-CN', unanchored), /anchored/);

    const global = cloneLocale();
    global.patterns[0].flags = 'gi';
    assert.throws(() => validateLocale('zh-CN', global), /stateful/);

    const unsampled = cloneLocale();
    delete unsampled.patterns[0].sample;
    assert.throws(() => validateLocale('zh-CN', unsampled), /sample/);
});

check('nested value rules and dictionaries are validated before injection', () => {
    const badRule = cloneLocale();
    badRule.valueRules.timeParts[0].pattern = '(';
    assert.throws(() => validateLocale('zh-CN', badRule), /invalid regex/);

    const badText = cloneLocale();
    badText.text.Settings = { unsafe: true };
    assert.throws(() => validateLocale('zh-CN', badText), /non-empty string/);
});

check('app path resolution fails closed when app.asar is absent', () => withTempDir((dir) => {
    assert.throws(() => resolveAppPaths(dir), /app\.asar was not found/);
    fs.mkdirSync(path.join(dir, 'resources'));
    fs.writeFileSync(path.join(dir, 'resources', 'app.asar'), 'archive');
    assert.strictEqual(resolveAppPaths(dir).asarPath, path.join(dir, 'resources', 'app.asar'));
}));

check('atomic replacement preserves the destination if staging fails', () => withTempDir((dir) => {
    const destination = path.join(dir, 'app.asar');
    fs.writeFileSync(destination, 'original');
    assert.throws(() => atomicReplaceFile(path.join(dir, 'missing.asar'), destination));
    assert.strictEqual(fs.readFileSync(destination, 'utf8'), 'original');
    assert.ok(!fs.readdirSync(dir).some((name) => name.startsWith(STAGE_DIR_PREFIX)));
}));

check('atomic replacement swaps a fully staged file', () => withTempDir((dir) => {
    const source = path.join(dir, 'new.asar');
    const destination = path.join(dir, 'app.asar');
    fs.writeFileSync(source, 'new archive');
    fs.writeFileSync(destination, 'old archive');
    atomicReplaceFile(source, destination);
    assert.strictEqual(fs.readFileSync(destination, 'utf8'), 'new archive');
    assert.ok(!fs.readdirSync(dir).some((name) => name.startsWith(STAGE_DIR_PREFIX)));
}));

check('official updates refresh a stale clean backup', () => withTempDir((dir) => {
    const asar = path.join(dir, 'app.asar');
    const clean = path.join(dir, 'app.asar.clean-backup');
    fs.writeFileSync(asar, 'new official archive');
    fs.writeFileSync(clean, 'old official archive');
    ensureCleanBackupForPatch(dir, asar, clean, false);
    assert.strictEqual(fs.readFileSync(clean, 'utf8'), 'new official archive');
}));

check('poisoned clean backups are replaced only by verified pristine archives', () => withTempDir((dir) => {
    const asar = path.join(dir, 'app.asar');
    const clean = path.join(dir, 'app.asar.clean-backup');
    const pristine = path.join(dir, 'app.asar.bak-20260101-000000');
    fs.writeFileSync(asar, 'installLocalePatch');
    fs.writeFileSync(clean, 'installLocalePatch');
    fs.writeFileSync(pristine, 'official archive');
    ensureCleanBackupForPatch(dir, asar, clean, true);
    assert.strictEqual(fs.readFileSync(clean, 'utf8'), 'official archive');

    fs.writeFileSync(clean, 'installLocalePatch');
    ensureRestorableCleanBackup(dir, clean);
    assert.strictEqual(fs.readFileSync(clean, 'utf8'), 'official archive');
}));

check('restore refuses an unrelated archive before refreshing the clean backup', () => withTempDir((dir) => {
    const resources = path.join(dir, 'resources');
    const asar = path.join(resources, 'app.asar');
    fs.mkdirSync(resources);
    const clean = path.join(resources, 'app.asar.clean-backup');
    fs.writeFileSync(asar, 'not an Antigravity archive');
    fs.writeFileSync(clean, 'known-good backup');
    assert.throws(
        () => restoreOfficial({ appDir: dir, restart: false, noKill: true }),
        /Could not verify Antigravity archive identity/
    );
    assert.strictEqual(fs.readFileSync(clean, 'utf8'), 'known-good backup');
}));

check('backup names never overwrite an existing same-second backup', () => withTempDir((dir) => {
    const name = 'app.asar.bak-20260101-000000';
    fs.writeFileSync(path.join(dir, name), 'first');
    assert.strictEqual(createUniqueBackupPath(dir, name), path.join(dir, name + '-1'));
}));

check('process control remains shell-free and never targets generic language servers', () => {
    const source = fs.readFileSync(path.join(__dirname, '..', 'src', 'detector.js'), 'utf8');
    assert.ok(!/\bexecSync\b/.test(source));
    assert.ok(!/language_server(?:\.exe)?/.test(source));
});

/**
 * Packing an archive is asynchronous, so this check runs after the synchronous
 * ones rather than inside withTempDir, whose cleanup is synchronous.
 */
async function checkAlreadyEnglishRestore() {
    const name = 'restoring an already-English app refreshes backup without rewriting app.asar';
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'antigravity-i18n-restore-'));
    try {
        const resources = path.join(dir, 'resources');
        const source = path.join(dir, 'source');
        const archive = path.join(resources, 'app.asar');
        const clean = path.join(resources, 'app.asar.clean-backup');
        fs.mkdirSync(resources);
        fs.mkdirSync(source);
        fs.writeFileSync(
            path.join(source, 'package.json'),
            JSON.stringify({ name: 'antigravity', version: '2.0.0' })
        );
        await require('@electron/asar').createPackageWithOptions(source, archive, {});
        const original = fs.readFileSync(archive);
        fs.writeFileSync(clean, 'stale official archive');

        restoreOfficial({ appDir: dir, restart: false, noKill: true });

        assert.deepStrictEqual(fs.readFileSync(archive), original, 'live archive was rewritten');
        assert.deepStrictEqual(fs.readFileSync(clean), original, 'clean backup was not refreshed');
        console.log('  ok   ' + name);
    } catch (err) {
        failures += 1;
        console.error('  FAIL ' + name + ': ' + err.message);
    } finally {
        fs.rmSync(dir, { recursive: true, force: true });
    }
}

async function checkArchiveCacheInvalidation() {
    const name = 'replacing an archive invalidates the memoised asar header';
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'antigravity-i18n-cache-'));
    try {
        // @electron/asar memoises archive headers by path. Without invalidation a
        // read after replacement returns the previous archive, which made a
        // second apply fail its identity check against an archive it had just
        // written correctly.
        const asar = require('@electron/asar');
        const build = (version) => {
            const staging = fs.mkdtempSync(path.join(dir, 'src-'));
            fs.writeFileSync(path.join(staging, 'package.json'), JSON.stringify({ name: 'antigravity', version }));
            return staging;
        };
        const first = path.join(dir, 'first.asar');
        const second = path.join(dir, 'second.asar');
        const destination = path.join(dir, 'app.asar');
        await asar.createPackageWithOptions(build('1.0.0'), first, {});
        await asar.createPackageWithOptions(build('2.0.0'), second, {});

        atomicReplaceFile(first, destination);
        assert.strictEqual(readArchiveIdentity(destination).version, '1.0.0');
        atomicReplaceFile(second, destination);
        assert.strictEqual(readArchiveIdentity(destination).version, '2.0.0', 'stale asar header was reused');
        console.log('  ok   ' + name);
    } catch (err) {
        failures += 1;
        console.error('  FAIL ' + name + ': ' + err.message);
    } finally {
        fs.rmSync(dir, { recursive: true, force: true });
    }
}

checkAlreadyEnglishRestore().then(checkArchiveCacheInvalidation).then(() => {
    if (failures > 0) {
        console.error('\n' + failures + ' security check(s) failed.');
        process.exit(1);
    }
    console.log('\nAll security checks passed.');
});
