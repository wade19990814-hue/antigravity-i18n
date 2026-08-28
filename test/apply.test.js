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

async function buildFixture(appDir) {
    const resourcesDir = path.join(appDir, 'resources');
    fs.mkdirSync(resourcesDir, { recursive: true });

    const staging = path.join(appDir, '.fixture-src');
    fs.mkdirSync(path.join(staging, 'dist'), { recursive: true });
    // A module matching the real app's unpackDir so the packing options are
    // exercised the same way.
    fs.mkdirSync(path.join(staging, 'node_modules', 'chrome-devtools-mcp'), { recursive: true });
    fs.writeFileSync(path.join(staging, 'package.json'), JSON.stringify(APP_IDENTITY));
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
    detector.isAntigravityRunning = () => false;

    try {
        await buildFixture(appDir);

        await step('apply installs the pack and reports it', async () => {
            await applyLocale({ appDir, locale: 'ja', restart: false, noKill: true });
            const status = getStatus({ appDir });
            assert.strictEqual(status.currentLanguage, 'ja');
            assert.strictEqual(status.localeName, '日本語');
            assert.strictEqual(status.hasCleanBackup, true);
            assert.strictEqual(status.stateIsStale, false);
            assert.strictEqual(readArchiveIdentity(status.asarPath).version, APP_IDENTITY.version);

            assert.match(readPacked(appDir, 'dist/main.js'), /appendSwitch\('lang', 'ja'\)/);
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
            assert.strictEqual(countBlocks(readPacked(appDir, 'dist/preload.js')), 1);
        });

        await step('restore returns the official archive byte for byte', async () => {
            restoreOfficial({ appDir, restart: false, noKill: true });
            const status = getStatus({ appDir });
            assert.strictEqual(status.currentLanguage, 'en');
            assert.strictEqual(status.hasCleanBackup, true);
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
    } finally {
        detector.isAntigravityRunning = originalProbe;
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
