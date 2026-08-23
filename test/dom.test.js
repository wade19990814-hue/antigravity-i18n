/**
 * DOM traversal layer tests for the injected translation engine.
 *
 * locale.test.js deliberately evaluates only the non-DOM half of the
 * generated fragment. This suite runs the whole fragment (including the DOM
 * traversal layer) against a minimal DOM stub and exercises translateNode,
 * translateAttributes and the MutationObserver wiring.
 */

const assert = require('assert');
const { buildPreloadFragment, loadLocale } = require('../src/locale');

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

// ---- Minimal DOM stubs ---------------------------------------------------

const Node = { ELEMENT_NODE: 1, TEXT_NODE: 3 };
const NodeFilter = {
    SHOW_ELEMENT: 0x1,
    SHOW_TEXT: 0x4,
    FILTER_ACCEPT: 1,
    FILTER_REJECT: 2,
    FILTER_SKIP: 3,
};

class FakeText {
    constructor(value) {
        this.nodeType = Node.TEXT_NODE;
        this.nodeValue = value;
        this.parentNode = null;
        this.parentElement = null;
    }
}

class FakeElement {
    constructor(tagName, attrs = {}) {
        this.nodeType = Node.ELEMENT_NODE;
        this.tagName = String(tagName).toUpperCase();
        this.attributes = Object.assign({}, attrs);
        this.childNodes = [];
        this.parentNode = null;
        this.parentElement = null;
    }

    append(...children) {
        for (const child of children) {
            child.parentNode = this;
            child.parentElement = this;
            this.childNodes.push(child);
        }
    }

    getAttribute(name) {
        return Object.prototype.hasOwnProperty.call(this.attributes, name)
            ? this.attributes[name]
            : null;
    }

    hasAttribute(name) {
        return Object.prototype.hasOwnProperty.call(this.attributes, name);
    }

    setAttribute(name, value) {
        this.attributes[name] = value;
    }

    get isContentEditable() {
        let current = this;
        while (current) {
            if (Object.prototype.hasOwnProperty.call(current.attributes, 'contenteditable')) {
                return current.attributes.contenteditable !== 'false';
            }
            current = current.parentElement;
        }
        return false;
    }

    // Understands only the selectors the engine uses: tag names, .classes
    // and [attr="value"].
    matches(selector) {
        return selector.split(',').some((part) => {
            const sel = part.trim();
            const attr = sel.match(/^\[([a-z-]+)="([^"]*)"\]$/);
            if (attr) {
                return this.getAttribute(attr[1]) === attr[2];
            }
            const cls = sel.match(/^\.([\w-]+)$/);
            if (cls) {
                return (this.attributes.class || '').split(/\s+/).includes(cls[1]);
            }
            return this.tagName === sel.toUpperCase();
        });
    }

    closest(selector) {
        let current = this;
        while (current) {
            if (current.matches(selector)) {
                return current;
            }
            current = current.parentElement;
        }
        return null;
    }
}

class FakeTreeWalker {
    constructor(root, whatToShow, filter) {
        this._nodes = [];
        const visit = (node) => {
            for (const child of node.childNodes) {
                let accepted = true;
                if (child.nodeType === Node.ELEMENT_NODE) {
                    const decision = filter.acceptNode(child);
                    if (decision === NodeFilter.FILTER_REJECT) {
                        // Skip the whole subtree.
                        continue;
                    }
                    accepted = decision === NodeFilter.FILTER_ACCEPT;
                }
                if (accepted) {
                    const shown = (child.nodeType === Node.ELEMENT_NODE && (whatToShow & NodeFilter.SHOW_ELEMENT))
                        || (child.nodeType === Node.TEXT_NODE && (whatToShow & NodeFilter.SHOW_TEXT));
                    if (shown) {
                        this._nodes.push(child);
                    }
                }
                if (child.nodeType === Node.ELEMENT_NODE) {
                    visit(child);
                }
            }
        };
        visit(root);
    }

    nextNode() {
        return this._nodes.length ? this._nodes.shift() : null;
    }
}

class FakeMutationObserver {
    constructor(callback) {
        this.callback = callback;
        this.target = null;
        this.options = null;
        FakeMutationObserver.instances.push(this);
    }

    observe(target, options) {
        this.target = target;
        this.options = options;
    }

    disconnect() {}

    trigger(mutations) {
        this.callback(mutations, this);
    }
}
FakeMutationObserver.instances = [];

function makeDocument(root) {
    return {
        readyState: 'complete',
        documentElement: root,
        addEventListener() {},
        createTreeWalker: (node, whatToShow, filter) => new FakeTreeWalker(node, whatToShow, filter),
    };
}

function el(tag, attrs = {}, ...children) {
    const element = new FakeElement(tag, attrs);
    for (const child of children) {
        element.append(child);
    }
    return element;
}

function txt(value) {
    return new FakeText(value);
}

// ---- Engine loading ------------------------------------------------------

const locale = loadLocale();
const sample = Object.entries(locale.text)[0];
const sampleKey = sample[0];
const sampleValue = sample[1];
const unknown = '__definitely not a ui string__';

function loadEngine(root) {
    const code = buildPreloadFragment(locale);
    const document = makeDocument(root);
    const module_ = { exports: {} };
    const factory = new Function(
        'module', 'document', 'Node', 'NodeFilter', 'MutationObserver',
        code + '\nmodule.exports = { translateNode, translateAttributes };'
    );
    factory(module_, document, Node, NodeFilter, FakeMutationObserver);
    const observer = FakeMutationObserver.instances[FakeMutationObserver.instances.length - 1];
    return {
        translateNode: module_.exports.translateNode,
        translateAttributes: module_.exports.translateAttributes,
        observer,
    };
}

// ---- Tests ---------------------------------------------------------------

console.log('DOM translation engine:');

check('installs <html lang> and translates the initial tree', () => {
    const root = el('html');
    const p = el('p');
    p.append(txt(sampleKey));
    root.append(p);
    loadEngine(root);
    assert.strictEqual(root.getAttribute('lang'), locale.htmlLang || locale.language);
    assert.strictEqual(p.childNodes[0].nodeValue, sampleValue);
});

check('leaves skippable containers untouched', () => {
    const cases = [
        ['code', {}],
        ['pre', {}],
        ['kbd', {}],
        ['samp', {}],
        ['select', {}],
        ['div', { role: 'textbox' }],
        ['div', { role: 'log' }],
        ['div', { role: 'terminal' }],
        ['div', { contenteditable: 'true' }],
        ['div', { contenteditable: '' }],
        ['div', { contenteditable: 'plaintext-only' }],
        ['div', { class: 'monaco-editor' }],
        ['div', { class: 'xterm' }],
        ['script', {}],
        ['style', {}],
    ];
    const root = el('html');
    for (const [tag, attrs] of cases) {
        const container = el(tag, attrs);
        container.append(txt(sampleKey));
        root.append(container);
    }
    loadEngine(root);
    cases.forEach(([tag], index) => {
        assert.strictEqual(
            root.childNodes[index].childNodes[0].nodeValue,
            sampleKey,
            'text inside <' + tag + '> was modified'
        );
    });
});

check('contenteditable=false does not suppress ordinary UI translation', () => {
    const root = el('html');
    const panel = el('div', { contenteditable: 'false' });
    panel.append(txt(sampleKey));
    root.append(panel);
    loadEngine(root);
    assert.strictEqual(panel.childNodes[0].nodeValue, sampleValue);
});

check('translates attributes but not text of skippable elements', () => {
    const root = el('html');
    const log = el('div', { role: 'log', 'aria-label': sampleKey, title: sampleKey, 'data-tooltip': sampleKey });
    log.append(txt(sampleKey));
    root.append(log);
    loadEngine(root);
    assert.strictEqual(log.getAttribute('aria-label'), sampleValue);
    assert.strictEqual(log.getAttribute('title'), sampleValue);
    assert.strictEqual(log.getAttribute('data-tooltip'), sampleValue);
    assert.strictEqual(log.childNodes[0].nodeValue, sampleKey);
});

check('translates input placeholders and labels', () => {
    const root = el('html');
    const input = el('input', { placeholder: sampleKey, 'aria-label': sampleKey });
    root.append(input);
    loadEngine(root);
    assert.strictEqual(input.getAttribute('placeholder'), sampleValue);
    assert.strictEqual(input.getAttribute('aria-label'), sampleValue);
});

check('leaves unknown attribute values unchanged', () => {
    const root = el('html');
    const button = el('button', { 'aria-label': unknown });
    root.append(button);
    loadEngine(root);
    assert.strictEqual(button.getAttribute('aria-label'), unknown);
});

check('translateNode translates an added subtree', () => {
    const root = el('html');
    const engine = loadEngine(root);
    const p = el('p');
    p.append(txt(sampleKey));
    engine.translateNode(p);
    assert.strictEqual(p.childNodes[0].nodeValue, sampleValue);
});

check('observer watches the engine attribute list', () => {
    const root = el('html');
    const engine = loadEngine(root);
    assert.deepStrictEqual(engine.observer.options.attributeFilter, ['aria-label', 'title', 'placeholder', 'data-tooltip', 'alt']);
});

check('characterData mutations are translated', () => {
    const root = el('html');
    const p = el('p');
    p.append(txt(unknown));
    root.append(p);
    const engine = loadEngine(root);
    const textNode = p.childNodes[0];
    textNode.nodeValue = sampleKey;
    engine.observer.trigger([{ type: 'characterData', target: textNode }]);
    assert.strictEqual(textNode.nodeValue, sampleValue);
});

check('attribute mutations are translated', () => {
    const root = el('html');
    const button = el('button', { 'aria-label': unknown });
    root.append(button);
    const engine = loadEngine(root);
    button.setAttribute('aria-label', sampleKey);
    engine.observer.trigger([{ type: 'attributes', target: button }]);
    assert.strictEqual(button.getAttribute('aria-label'), sampleValue);
});

check('added nodes are translated', () => {
    const root = el('html');
    const engine = loadEngine(root);
    const p = el('p');
    p.append(txt(sampleKey));
    engine.observer.trigger([{ type: 'childList', addedNodes: [p] }]);
    assert.strictEqual(p.childNodes[0].nodeValue, sampleValue);
});

check('added skippable nodes keep text but translate labels', () => {
    const root = el('html');
    const engine = loadEngine(root);
    const editor = el('div', { class: 'monaco-editor', 'aria-label': sampleKey });
    editor.append(txt(sampleKey));
    engine.observer.trigger([{ type: 'childList', addedNodes: [editor] }]);
    assert.strictEqual(editor.childNodes[0].nodeValue, sampleKey);
    assert.strictEqual(editor.getAttribute('aria-label'), sampleValue);
});

check('translateAttributes only touches listed attributes', () => {
    const root = el('html');
    const engine = loadEngine(root);
    const button = el('button', { 'data-custom': sampleKey });
    engine.translateAttributes(button);
    assert.strictEqual(button.getAttribute('data-custom'), sampleKey);
});

if (failures > 0) {
    console.error('\n' + failures + ' DOM check(s) failed.');
    process.exit(1);
}
console.log('\nAll DOM checks passed.');
