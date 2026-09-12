const assert = require('assert');
const vm = require('vm');
const { buildDialogFragment, loadLocale, listLocales } = require('../src/locale');

const makeOptions = () => ({
    type: 'question', title: 'Confirm Quit', message: 'Are you sure you want to quit?',
    detail: 'There may be agents or background tasks running.',
    buttons: ['Cancel', 'Quit'], defaultId: 1, cancelId: 0, noLink: true
});
const response = Promise.resolve({ response: 0, checkboxChecked: false });
let failures = 0;
function check(name, fn) {
    try { fn(); console.log('  ok   ' + name); }
    catch (err) { failures++; console.error('  FAIL ' + name + ': ' + err.message); }
}

function harness(code) {
    const calls = [];
    const dialog = {
        showMessageBox(...args) { calls.push({ method: 'async', args, receiver: this }); return response; },
        showMessageBoxSync(...args) { calls.push({ method: 'sync', args, receiver: this }); return 0; }
    };
    vm.runInNewContext(buildDialogFragment(loadLocale(code)), { electron_1: { dialog } });
    return { dialog, calls };
}
console.log('Native quit confirmation:');
for (const code of listLocales()) {
    check(code + ': translates both overloads without changing button behavior', () => {
        const { dialog, calls } = harness(code);
        const options = makeOptions();
        const window = { id: 42 };
        assert.strictEqual(dialog.showMessageBox(window, options), response);
        assert.strictEqual(dialog.showMessageBoxSync(options), 0);
        const pack = loadLocale(code);
        for (const [call, index] of [[calls[0], 1], [calls[1], 0]]) {
            const translated = call.args[index];
            for (const field of ['title', 'message', 'detail']) {
                assert.strictEqual(translated[field], pack.dialogs[options[field]]);
                assert.notStrictEqual(translated[field], options[field]);
            }
            assert.deepStrictEqual(Array.from(translated.buttons), options.buttons.map(b => pack.dialogs[b]));
            for (const field of ['defaultId', 'cancelId', 'type', 'noLink']) {
                assert.strictEqual(translated[field], options[field]);
            }
            assert.strictEqual(call.receiver, dialog);
        }
        assert.strictEqual(calls[0].args[0], window);
        assert.deepStrictEqual(options, makeOptions(), 'caller options were mutated');
    });
}
check('unrelated messages are passed through without cloning or translation', () => {
    const { dialog, calls } = harness('zh-CN');
    const options = { title: 'Settings', message: 'Settings', buttons: ['Cancel'] };
    dialog.showMessageBox(null, options);
    assert.strictEqual(calls[0].args[1], options);
});
if (failures) process.exit(1);
console.log('All native dialog checks passed.');
