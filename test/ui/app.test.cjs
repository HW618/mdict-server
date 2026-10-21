const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const path = require('node:path');
const source = fs.readFileSync(path.join(__dirname, '../../templates/assets/app.js'), 'utf8');

function createApp() {
    const saved = new Map();
    const context = vm.createContext({
        AbortController, DOMException, URLSearchParams, setTimeout, clearTimeout,
        localStorage: { getItem: key => saved.get(key) ?? null, setItem: (key, value) => saved.set(key, value), removeItem: key => saved.delete(key) },
    });
    vm.runInContext(source, context);
    const app = context.app();
    app.token = 'test-token';
    app.$nextTick = () => {};
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

test('arrow keys select the first or last suggestion and wrap', () => {
    const app = createApp();
    app.suggestions = [{ word: 'apple' }, { word: 'application' }, { word: 'apricot' }];
    app.suggestionsOpen = true;
    app.moveSuggestion(-1); assert.equal(app.activeSuggestion, 2);
    app.moveSuggestion(1); assert.equal(app.activeSuggestion, 0);
    app.activeSuggestion = -1;
    app.moveSuggestion(1); assert.equal(app.activeSuggestion, 0);
});
