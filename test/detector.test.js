const assert = require('assert');
const fs = require('fs');
const path = require('path');
const vm = require('vm');

const source = fs.readFileSync(path.join(__dirname, '../src/detector.js'), 'utf8');
function harness(platform, execute) {
    let now = 0;
    const commands = [];
    const context = {
        module: { exports: {} },
        Date: { now: () => now },
        Atomics: { wait: (_array, _index, _value, ms) => { now += ms; } },
        require(id) {
            if (id === 'os') return { platform: () => platform };
            if (id === 'child_process') return {
                execFileSync(file, args) {
                    commands.push({ file, args, time: now });
                    return execute(file, args, commands);
                }
            };
            return require(id);
        }
    };
    vm.runInNewContext(source, context);
    return { api: context.module.exports, commands, time: () => now };
}
let failures = 0;
function check(name, fn) {
    try { fn(); console.log('  ok   ' + name); }
    catch (err) { failures++; console.error('  FAIL ' + name + ': ' + err.message); }
}

console.log('Process discovery and shutdown:');
check('default shutdown allows 30 seconds before forced termination', () => {
    let killed = false;
    const h = harness('win32', (file, args) => {
        if (file === 'tasklist.exe') return killed ? '' : 'Antigravity.exe 1234';
        if (args.includes('/F')) killed = true;
    });
    assert.strictEqual(h.api.stopAntigravityProcesses().forced, true);
    assert.strictEqual(h.commands.find(c => c.args.includes('/F')).time, 30000);
});
check('a normal exit does not force-close the application', () => {
    let closed = false;
    const h = harness('win32', (file) => {
        if (file === 'tasklist.exe') return closed ? '' : 'Antigravity.exe 1234';
        closed = true;
    });
    assert.strictEqual(h.api.stopAntigravityProcesses().forced, false);
    assert.ok(!h.commands.some(c => c.args.includes('/F')));
});
check('--force still bypasses the waiting period', () => {
    let killed = false;
    const h = harness('win32', (file, args) => {
        if (file === 'tasklist.exe') return killed ? '' : 'Antigravity.exe 1234';
        killed = args.includes('/F');
    });
    assert.strictEqual(h.api.stopAntigravityProcesses({ force: true }).stopped, true);
    assert.strictEqual(h.time(), 0);
    assert.ok(!h.commands.some(c => c.file === 'taskkill.exe' && !c.args.includes('/F')));
});
check('Windows discovery errors stop before any termination attempt', () => {
    const h = harness('win32', () => { throw Object.assign(new Error('denied'), { code: 'EACCES' }); });
    assert.throws(() => h.api.stopAntigravityProcesses(), /Could not query/);
    assert.strictEqual(h.commands.length, 1);
});
check('discovery errors while waiting do not mean the app exited', () => {
    let probes = 0;
    const h = harness('win32', (file) => {
        if (file !== 'tasklist.exe') return '';
        if (++probes === 1) return 'Antigravity.exe 1234';
        throw new Error('probe failed');
    });
    assert.throws(() => h.api.stopAntigravityProcesses(), /Could not query/);
    assert.ok(!h.commands.some(c => c.args.includes('/F')));
});
for (const platform of ['linux', 'darwin']) {
    check(platform + ': pgrep status 1 means no matching process', () => {
        const h = harness(platform, () => { throw Object.assign(new Error('no matches'), { status: 1 }); });
        assert.strictEqual(h.api.isAntigravityRunning(), false);
        assert.strictEqual(h.commands.length, 2);
    });
    check(platform + ': pgrep failures are not treated as no matches', () => {
        for (const fields of [{ status: 2 }, { code: 'ENOENT' }, { status: null, signal: 'SIGTERM' }]) {
            const h = harness(platform, () => { throw Object.assign(new Error('query failed'), fields); });
            assert.throws(() => h.api.isAntigravityRunning(), /Could not query/);
        }
    });
}
if (failures) process.exit(1);
console.log('All process checks passed.');
