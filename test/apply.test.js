/**
 * End-to-end apply/restore gate against a synthetic Antigravity archive.
 *
 * Runs without Antigravity installed: the fixture carries a fake archive whose
 * files hold the same anchors the real app exposes, and the process probe is
 * stubbed so the test never touches a real installation or process.
 */

const assert = require('assert');
const fs = require('fs');
const os = require('os');
const path = require('path');
const asar = require('@electron/asar');
const detector = require('../src/detector');
const { applyLocale, restoreOfficial, getStatus, readArchiveIdentity } = require('../src/index');

// Fixture sources carrying the same anchors the real app exposes.
const MAIN_JS = [
    "const electron_1 = require('electron');",
    "if (!electron_1.app.commandLine.hasSwitch('remote-debugging-port')) {",
    "    electron_1.app.commandLine.appendSwitch('remote-debugging-port', '0');",
    '}',
    'electron_1.app.whenReady();',
    ''
].join('\n');

const MENU_JS = [
    'function buildApplicationMenu() {',
    '    const menu = createMenu();',
    '    // Re-apply the menu so the change takes effect.',
    '    return menu;',
    '}',
    ''
].join('\n');

const PRELOAD_JS = [
    "const { contextBridge } = require('electron');",
    'const updaterAPI = {',
    '    check: () => null',
    '};',
    'contextBridge.exposeInMainWorld("updaterAPI", updaterAPI);',
    ''
].join('\n');

const APP_IDENTITY = { name: 'antigravity', version: '9.9.9' };
const ENGINE_BLOCK = '/* antigravity-i18n:begin */';

let failures = 0;
let reportedFailure = false;

async function step(name, fn) {
    try {
        await fn();
        console.log('  ok   ' + name);
    } catch (err) {
        // Later steps depend on earlier ones, so bail on the first failure.
        failures += 1;
        reportedFailure = true;
        console.error('  FAIL ' + name + ': ' + err.message);
        throw err;
    }
}

async function buildFixture(appDir, identity = APP_IDENTITY) {
    const resourcesDir = path.join(appDir, 'resources');
    fs.mkdirSync(resourcesDir, { recursive: true });

    const staging = path.join(appDir, '.fixture-src');
    fs.mkdirSync(path.join(staging, 'dist'), { recursive: true });
    // A module matching the real app's unpackDir so the packing options are
    // exercised the same way.
    fs.mkdirSync(path.join(staging, 'node_modules', 'chrome-devtools-mcp'), { recursive: true });
    fs.writeFileSync(path.join(staging, 'package.json'), JSON.stringify(identity));
    fs.writeFileSync(path.join(staging, 'dist', 'main.js'), MAIN_JS);
    fs.writeFileSync(path.join(staging, 'dist', 'menu.js'), MENU_JS);
    fs.writeFileSync(path.join(staging, 'dist', 'preload.js'), PRELOAD_JS);
    fs.writeFileSync(
        path.join(staging, 'node_modules', 'chrome-devtools-mcp', 'index.js'),
        'module.exports = {};\n'
    );
    await asar.createPackageWithOptions(staging, path.join(resourcesDir, 'app.asar'), {
        unpackDir: 'node_modules/chrome-devtools-mcp'
    });
    fs.rmSync(staging, { recursive: true, force: true });
}

function readPacked(appDir, relativePath) {
    return asar.extractFile(path.join(appDir, 'resources', 'app.asar'), relativePath).toString('utf8');
}

function countBlocks(source) {
    return source.split(ENGINE_BLOCK).length - 1;
}

async function main() {
    console.log('End-to-end apply and restore:');
    const appDir = fs.mkdtempSync(path.join(os.tmpdir(), 'antigravity-i18n-e2e-'));
    const resourcesDir = path.join(appDir, 'resources');
    // The fixture is not a running app; stubbing the probe keeps the test
    // independent of whatever is running on the machine.
    const originalProbe = detector.isAntigravityRunning;
    const originalStop = detector.stopAntigravityProcesses;
    const originalPack = asar.createPackageWithOptions;
    detector.isAntigravityRunning = () => false;

    try {
        await buildFixture(appDir);
        const archivePath = path.join(resourcesDir, 'app.asar');
        const pristineBytes = fs.readFileSync(archivePath);

        await step('apply installs the pack and reports it', async () => {
            await applyLocale({ appDir, locale: 'ja', restart: false, noKill: true });
            const status = getStatus({ appDir });
            assert.strictEqual(status.currentLanguage, 'ja');
            assert.strictEqual(status.localeName, '日本語');
            assert.strictEqual(status.hasCleanBackup, true);
            assert.strictEqual(status.stateIsStale, false);
            assert.strictEqual(readArchiveIdentity(status.asarPath).version, APP_IDENTITY.version);

            assert.match(readPacked(appDir, 'dist/main.js'), /appendSwitch\('lang', 'ja'\)/);
            assert.match(readPacked(appDir, 'dist/main.js'), /installLocaleDialogs/);
            assert.strictEqual(countBlocks(readPacked(appDir, 'dist/main.js')), 1);
            assert.strictEqual(countBlocks(readPacked(appDir, 'dist/preload.js')), 1);
            assert.strictEqual(countBlocks(readPacked(appDir, 'dist/menu.js')), 1);
            assert.match(readPacked(appDir, 'dist/menu.js'), /translateMenu\(menu\);/);
            assert.ok(
                fs.readdirSync(resourcesDir).some((f) => /^app\.asar\.bak-\d{8}-\d{6}$/.test(f)),
                'timestamped backup was not created'
            );
        });

        await step('re-applying the same locale keeps exactly one block', async () => {
            await applyLocale({ appDir, locale: 'ja', restart: false, noKill: true });
            assert.strictEqual(countBlocks(readPacked(appDir, 'dist/main.js')), 1);
            assert.strictEqual(countBlocks(readPacked(appDir, 'dist/preload.js')), 1);
            assert.strictEqual(countBlocks(readPacked(appDir, 'dist/menu.js')), 1);
            assert.strictEqual(
                (readPacked(appDir, 'dist/menu.js').match(/translateMenu\(menu\);/g) || []).length,
                1
            );
            assert.strictEqual(getStatus({ appDir }).currentLanguage, 'ja');
        });

        await step('switching locale replaces the pack in place', async () => {
            await applyLocale({ appDir, locale: 'ru', restart: false, noKill: true });
            const status = getStatus({ appDir });
            assert.strictEqual(status.currentLanguage, 'ru');
            const main = readPacked(appDir, 'dist/main.js');
            assert.match(main, /appendSwitch\('lang', 'ru'\)/);
            assert.ok(!main.includes("'ja'"), 'stale language switch survived');
            assert.match(main, /Подтверждение выхода/);
            assert.ok(!main.includes('終了の確認'), 'stale native dialog language survived');
            assert.strictEqual(countBlocks(readPacked(appDir, 'dist/preload.js')), 1);
        });

        await step('restore returns the official archive byte for byte', async () => {
            restoreOfficial({ appDir, restart: false, noKill: true });
            const status = getStatus({ appDir });
            assert.strictEqual(status.currentLanguage, 'en');
            assert.strictEqual(status.hasCleanBackup, true);
            assert.deepStrictEqual(fs.readFileSync(archivePath), pristineBytes, 'full archive differs from the original');
            assert.strictEqual(readPacked(appDir, 'dist/preload.js'), PRELOAD_JS, 'restored preload differs');
            assert.strictEqual(readPacked(appDir, 'dist/main.js'), MAIN_JS, 'restored main differs');
            assert.strictEqual(readPacked(appDir, 'dist/menu.js'), MENU_JS, 'restored menu differs');
            assert.ok(
                fs.readdirSync(resourcesDir).some((f) => f.startsWith('app.asar.bak-before-restore-')),
                'pre-restore safety backup was not created'
            );
        });

        await step('a restored install can be patched again', async () => {
            await applyLocale({ appDir, locale: 'zh-CN', restart: false, noKill: true });
            const status = getStatus({ appDir });
            assert.strictEqual(status.currentLanguage, 'zh-CN');
            assert.strictEqual(countBlocks(readPacked(appDir, 'dist/preload.js')), 1);
        });

        const clean = path.join(resourcesDir, 'app.asar.clean-backup');
        await step('status rejects corrupt, truncated, patched and wrong-version backups', async () => {
            const valid = fs.readFileSync(clean);
            const before = fs.readFileSync(archivePath);
            const setBackup = (bytes) => { fs.writeFileSync(clean, bytes); asar.uncache(clean); };
            try {
                setBackup(Buffer.from('broken backup'));
                assert.strictEqual(getStatus({ appDir }).cleanBackupStatus, 'invalid');
                assert.strictEqual(getStatus({ appDir }).hasCleanBackup, false);
                setBackup(valid.subarray(0, valid.length - 1));
                assert.strictEqual(getStatus({ appDir }).cleanBackupStatus, 'invalid');
                const tampered = Buffer.from(valid);
                // Alter a packed file in the valid archive, keeping its manifest readable.
                setBackup(valid);
                const entry = asar.statFile(clean, 'dist/main.js');
                tampered[8 + asar.getRawHeader(clean).headerSize + Number(entry.offset)] ^= 1;
                setBackup(tampered);
                assert.strictEqual(getStatus({ appDir }).cleanBackupStatus, 'invalid');
                // Restore must recover from a valid snapshot instead of copying corruption.
                restoreOfficial({ appDir, restart: false, noKill: true });
                assert.deepStrictEqual(fs.readFileSync(archivePath), pristineBytes);
                fs.writeFileSync(archivePath, before);
                asar.uncache(archivePath);
                setBackup(before);
                assert.strictEqual(getStatus({ appDir }).cleanBackupStatus, 'patched');
                const differentApp = path.join(appDir, 'different-version');
                await buildFixture(differentApp, { name: 'antigravity', version: '9.9.8' });
                setBackup(fs.readFileSync(path.join(differentApp, 'resources/app.asar')));
                assert.strictEqual(getStatus({ appDir }).cleanBackupStatus, 'version-mismatch');
                fs.unlinkSync(clean);
                assert.strictEqual(getStatus({ appDir }).cleanBackupStatus, 'missing');
                assert.deepStrictEqual(fs.readFileSync(archivePath), before, 'status modified the live archive');
            } finally { setBackup(valid); }
        });

        await step('process discovery failure leaves the archive and backup untouched', async () => {
            const before = fs.readFileSync(archivePath);
            const cleanBefore = fs.readFileSync(clean);
            detector.isAntigravityRunning = () => { throw new Error('Could not query Antigravity processes'); };
            try {
                await assert.rejects(applyLocale({ appDir, locale: 'ja', noKill: true, restart: false }), /Could not query/);
                assert.throws(() => restoreOfficial({ appDir, noKill: true, restart: false }), /Could not query/);
                assert.deepStrictEqual(fs.readFileSync(archivePath), before);
                assert.deepStrictEqual(fs.readFileSync(clean), cleanBefore);
            } finally { detector.isAntigravityRunning = () => false; }
        });

        const newApp = path.join(appDir, 'update');
        await buildFixture(newApp, { name: 'antigravity', version: '9.9.10' });
        const updateBytes = fs.readFileSync(path.join(newApp, 'resources/app.asar'));
        const replaceDuringUpdate = () => {
            // Deliberately leave the asar library's cached header stale, just as an external updater would.
            fs.writeFileSync(archivePath, updateBytes);
        };
        await step('restore aborts when the app updates while closing', async () => {
            const cleanBefore = fs.readFileSync(clean);
            detector.stopAntigravityProcesses = () => {
                replaceDuringUpdate();
                return { wasRunning: true, stopped: true, forced: false };
            };
            assert.throws(() => restoreOfficial({ appDir, restart: false }), /archive changed/);
            assert.deepStrictEqual(fs.readFileSync(archivePath), updateBytes);
            assert.deepStrictEqual(fs.readFileSync(clean), cleanBefore);
        });
        await step('apply aborts before refreshing a backup if shutdown installed an update', async () => {
            await buildFixture(appDir);
            asar.uncache(archivePath);
            const cleanBefore = fs.readFileSync(clean);
            await assert.rejects(applyLocale({ appDir, locale: 'ja', restart: false }), /archive changed/);
            assert.deepStrictEqual(fs.readFileSync(archivePath), updateBytes);
            assert.deepStrictEqual(fs.readFileSync(clean), cleanBefore);
        });
        await step('an update during repacking is preserved before final replacement', async () => {
            await buildFixture(appDir);
            asar.uncache(archivePath);
            asar.createPackageWithOptions = async (...args) => {
                await originalPack(...args);
                replaceDuringUpdate();
            };
            try {
                await assert.rejects(applyLocale({ appDir, locale: 'ja', noKill: true, restart: false }), /archive changed/);
                assert.deepStrictEqual(fs.readFileSync(archivePath), updateBytes);
            } finally { asar.createPackageWithOptions = originalPack; }
        });
    } finally {
        detector.isAntigravityRunning = originalProbe;
        detector.stopAntigravityProcesses = originalStop;
        asar.createPackageWithOptions = originalPack;
        fs.rmSync(appDir, { recursive: true, force: true });
    }

    if (failures > 0) {
        console.error('\n' + failures + ' end-to-end check(s) failed.');
        process.exit(1);
    }
    console.log('\nAll end-to-end checks passed.');
}

main().catch((err) => {
    if (!reportedFailure) {
        console.error('  FAIL fixture setup: ' + err.message);
    }
    process.exit(1);
});
