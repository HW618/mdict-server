const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const path = require('node:path');
const source = fs.readFileSync(path.join(__dirname, '../../templates/assets/app.js'), 'utf8');

function createApp() {
    const saved = new Map();
    const sandbox = {
        AbortController, DOMException, URLSearchParams, setTimeout, clearTimeout, Map, Set, JSON, Promise,
        fetch: async () => { throw new Error('unexpected fetch'); },
        URL: { createObjectURL: () => 'blob:asset', revokeObjectURL() {} },
        localStorage: { getItem: key => saved.get(key) ?? null, setItem: (key, value) => saved.set(key, value), removeItem: key => saved.delete(key) },
    };
    const context = vm.createContext(sandbox);
    vm.runInContext(source, context);
    const app = context.app();
    app.token = 'test-token';
    app.$nextTick = () => {};
    app.storage = saved;
    app.sandbox = sandbox;
    return app;
}
const deferred = () => { let resolve; const promise = new Promise(done => { resolve = done; }); return { promise, resolve }; };
const suggestion = word => ({ code: 0, data: { items: [{ word }] } });

test('late suggestions never replace the latest keyword results', async () => {
    const app = createApp(), old = deferred(), current = deferred();
    let oldSignal;
    app.fetchWithAuth = (_, { signal }) => { oldSignal = signal; return old.promise; };
    app.searchWord = 'ap';
    const pending = app.fuzzySearch();
    app.cancelSuggestions();
    assert.equal(oldSignal.aborted, true);
    app.searchWord = 'ba';
    app.fetchWithAuth = () => current.promise;
    const latest = app.fuzzySearch();
    current.resolve(suggestion('banana')); await latest;
    old.resolve(suggestion('apple')); await pending;
    assert.equal(app.suggestions[0].word, 'banana');
    assert.equal(app.suggestionsLoading, false);
});

test('clearing input or dismissing suggestions rejects in-flight results', async () => {
    const app = createApp(), response = deferred();
    app.searchWord = 'ap'; app.fetchWithAuth = () => response.promise;
    const pending = app.fuzzySearch();
    app.searchWord = ''; app.debouncedFuzzySearch();
    response.resolve(suggestion('apple')); await pending;
    assert.equal(app.suggestions.length, 0);
    assert.equal(app.suggestionsOpen, false);
});

test('dictionary changes invalidate suggestions and pass the new filter', async () => {
    const app = createApp(), response = deferred();
    app.searchWord = 'ap'; app.selectedDict = 'first'; app.fetchWithAuth = () => response.promise;
    const pending = app.fuzzySearch();
    app.chooseDict('second');
    clearTimeout(app.fuzzySearchTimeout);
    response.resolve(suggestion('apricot')); await pending;
    assert.equal(app.suggestions.length, 0);
    let requested;
    app.fetchWithAuth = async url => { requested = url; return suggestion('apple'); };
    await app.fuzzySearch();
    assert.equal(new URL(requested, 'http://localhost').searchParams.get('dict_id'), 'second');
});

test('late exact searches do not overwrite results or search history', async () => {
    const app = createApp(), old = deferred();
    app.searchWord = 'apple'; app.fetchWithAuth = () => old.promise;
    const pending = app.search();
    app.searchWord = 'banana';
    app.fetchWithAuth = async () => ({ code: 0, data: { results: [{ dict_id: 'second', html: 'banana' }] } });
    await app.search();
    old.resolve({ code: 0, data: { results: [{ dict_id: 'first', html: 'apple' }] } }); await pending;
    assert.equal(app.resultWord, 'banana');
    assert.equal(app.results[0].html, 'banana');
    assert.equal(app.searchHistory.join(','), 'banana');
});

test('composition and one Unicode character do not issue suggestions', () => {
    const app = createApp();
    app.fuzzySearchTimeout = null;
    app.searchWord = '中'; app.debouncedFuzzySearch();
    assert.equal(app.fuzzySearchTimeout, null);
    app.searchWord = '中文'; app.composing = true; app.debouncedFuzzySearch();
    assert.equal(app.fuzzySearchTimeout, null);
});

test('signing out cancels searches and clears dictionary content', async () => {
    const app = createApp(), response = deferred();
    app.searchWord = 'apple'; app.fetchWithAuth = () => response.promise;
    const pending = app.search();
    app.logout();
    response.resolve({ code: 0, data: { results: [{ html: 'apple' }] } }); await pending;
    assert.equal(app.results.length, 0);
    assert.equal(app.token, null);
    assert.equal(app.loading, false);
});

test('search history stays with the account and is hidden after logout', () => {
    const app = createApp();
    app.storage.set('search_history', JSON.stringify(['shared']));
    app.user = { id: 'user-a' };
    app.addToHistory('secret');
    app.loadHistory();
    assert.equal(app.storage.has('search_history'), false);
    assert.equal(app.searchHistory.join(','), 'secret');
    app.logout();
    assert.equal(app.searchHistory.length, 0);
    app.user = { id: 'user-b' };
    app.loadHistory();
    assert.equal(app.searchHistory.length, 0);
    app.user = { id: 'user-a' };
    app.loadHistory();
    assert.equal(app.searchHistory.join(','), 'secret');
});

test('an older dictionary list cannot overwrite a newer one', async () => {
    const app = createApp(), first = deferred(), second = deferred();
    let calls = 0;
    app.fetchWithAuth = () => ++calls === 1 ? first.promise : second.promise;
    const older = app.loadDicts();
    const newer = app.loadDicts();
    second.resolve({ code: 0, data: [{ id: 'new', is_enabled: true }] });
    await newer;
    first.resolve({ code: 0, data: [{ id: 'old', is_enabled: true }] });
    await older;
    assert.equal(app.dicts[0].id, 'new');
});

test('definition media is fetched with the session token', async () => {
    const app = createApp();
    app.token = 'session-token';
    const seen = [];
    app.sandbox.fetch = async (url, options) => {
        seen.push({ url, authorization: options.headers.Authorization });
        return { ok: true, status: 200, blob: async () => ({}) };
    };
    const attrs = new Map();
    let style = 'p{background:url("/api/v1/assets/dict/bg.png")}';
    const element = {
        _renderGeneration: 1,
        querySelectorAll(selector) {
            if (selector === 'style') {
                return [{ get textContent() { return style; }, set textContent(value) { style = value; } }];
            }
            return [{
                getAttribute: name => name === 'src' ? '/api/v1/assets/dict/\\a.png' : null,
                setAttribute: (name, value) => attrs.set(name, value),
            }];
        },
    };
    await app.hydrateAssets(element, 1, element);
    app.$refs = { audioPlayer: { src: '', async play() {} } };
    await app.playAudio('/api/v1/assets/dict/\\hello.mp3');
    assert.equal(attrs.get('src'), 'blob:asset');
    assert.match(style, /blob:asset/);
    assert.equal(app.$refs.audioPlayer.src, 'blob:asset');
    assert.deepEqual(seen.map(item => item.url), [
        '/api/v1/assets/dict/a.png',
        '/api/v1/assets/dict/bg.png',
        '/api/v1/assets/dict/hello.mp3',
    ]);
    assert.ok(seen.every(item => item.authorization === 'Bearer session-token'));
});

test('arrow keys select the first or last suggestion and wrap', () => {
    const app = createApp();
    app.suggestions = [{ word: 'apple' }, { word: 'application' }, { word: 'apricot' }];
    app.suggestionsOpen = true;
    app.moveSuggestion(-1); assert.equal(app.activeSuggestion, 2);
    app.moveSuggestion(1); assert.equal(app.activeSuggestion, 0);
    app.activeSuggestion = -1;
    app.moveSuggestion(1); assert.equal(app.activeSuggestion, 0);
});
