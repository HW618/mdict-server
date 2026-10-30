function app() {
    return {
        dicts: [], selectedDict: '', dictsError: '', searchWord: '', resultWord: '',
        results: [], suggestions: [], suggestionsOpen: false, suggestionsLoading: false,
        suggestionsError: '', activeSuggestion: -1, composing: false,
        loading: false, searched: false, error: null, user: null, token: null,
        showLogin: false, loginLoading: false, loginForm: { username: '', password: '' }, loginError: null,
        fuzzySearchTimeout: null, fuzzyController: null, fuzzyRequest: 0,
        searchController: null, searchRequest: 0, refreshPromise: null, dictRequest: 0,
        searchHistory: [], maxHistorySize: 50, assetBlobs: new Map(),

        async init() {
            try {
                this.token = localStorage.getItem('token');
                this.user = this.token ? JSON.parse(localStorage.getItem('user') || 'null') : null;
            } catch (_) {
                this.token = null;
                this.user = null;
            }
            this.loadHistory();
            if (this.token) await this.loadDicts();
        },

        async loadDicts() {
            this.dictsError = '';
            const request = ++this.dictRequest;
            const token = this.token;
            try {
                const response = await this.fetchWithAuth('/api/v1/dicts');
                if (request !== this.dictRequest || token !== this.token) return;
                if (response.code === 0) {
                    this.dicts = (response.data || []).filter(dict => dict.is_enabled);
                    if (!this.dicts.some(dict => dict.id === this.selectedDict)) this.selectedDict = '';
                } else {
                    this.dictsError = '词典列表加载失败，请重新登录或刷新页面。';
                }
            } catch (_) {
                this.dictsError = '暂时无法连接服务，请稍后刷新页面。';
            }
        },

        chooseDict(id) {
            this.selectedDict = id;
            this.cancelSuggestions();
            this.cancelSearch();
            if (this.searched && this.searchWord.trim()) this.search();
            else this.debouncedFuzzySearch();
        },

        onSearchInput(event) {
            this.cancelSearch();
            if (event.isComposing || this.composing) return;
            this.debouncedFuzzySearch();
        },

        submitSearch() {
            if (this.composing) return;
            if (this.suggestionsOpen && this.activeSuggestion >= 0) {
                this.selectSuggestion(this.suggestions[this.activeSuggestion].word);
            } else {
                this.search();
            }
        },

        cancelSearch() {
            this.searchRequest++;
            this.searchController?.abort();
            this.searchController = null;
            this.loading = false;
        },

        async search() {
            const word = this.searchWord.trim();
            if (!word) return;
            this.cancelSuggestions();
            this.cancelSearch();
            if (!this.token) { this.openLogin(); return; }
            const request = this.searchRequest;
            const dict = this.selectedDict;
            const controller = new AbortController();
            this.searchController = controller;
            this.loading = true;
            this.error = null;
            try {
                const params = new URLSearchParams({ word });
                if (dict) params.set('dict_id', dict);
                const response = await this.fetchWithAuth(`/api/v1/search?${params}`, { signal: controller.signal });
                if (request !== this.searchRequest) return;
                this.searched = true;
                this.resultWord = word;
                if (response.code === 0) {
                    this.results = response.data.results || [];
                    if (this.results.length) this.addToHistory(word);
                } else if (response.code === 40401) {
                    this.results = [];
                } else {
                    this.results = [];
                    this.error = response.code === 42901 ? '查询太频繁，请稍后再试。' : '查询失败，请检查登录状态后重试。';
                }
            } catch (error) {
                if (error.name !== 'AbortError' && request === this.searchRequest) {
                    this.searched = true;
                    this.resultWord = word;
                    this.results = [];
                    this.error = '网络连接失败，请重试。';
                }
            } finally {
                if (request === this.searchRequest) { this.loading = false; this.searchController = null; }
            }
        },

        cancelSuggestions() {
            clearTimeout(this.fuzzySearchTimeout);
            this.fuzzyController?.abort();
            this.fuzzyController = null;
            this.fuzzyRequest++;
            this.suggestions = [];
            this.suggestionsOpen = false;
            this.suggestionsLoading = false;
            this.suggestionsError = '';
            this.activeSuggestion = -1;
        },

        debouncedFuzzySearch() {
            this.cancelSuggestions();
            if (!this.token || this.composing || Array.from(this.searchWord.trim()).length < 2) return;
            this.fuzzySearchTimeout = setTimeout(() => this.fuzzySearch(), 150);
        },

        async fuzzySearch() {
            const keyword = this.searchWord.trim();
            if (!this.token || Array.from(keyword).length < 2) return;
            const dict = this.selectedDict;
            const request = ++this.fuzzyRequest;
            const controller = new AbortController();
            this.fuzzyController?.abort();
            this.fuzzyController = controller;
            this.suggestionsOpen = true;
            this.suggestionsLoading = true;
            this.suggestionsError = '';
            try {
                const params = new URLSearchParams({ keyword, page_size: '10' });
                if (dict) params.set('dict_id', dict);
                const response = await this.fetchWithAuth(`/api/v1/search/fuzzy?${params}`, { signal: controller.signal });
                if (request !== this.fuzzyRequest || keyword !== this.searchWord.trim() || dict !== this.selectedDict) return;
                if (response.code === 0) this.suggestions = response.data.items || [];
                else this.suggestionsError = '联想暂不可用，可按 Enter 直接查询。';
            } catch (error) {
                if (error.name !== 'AbortError' && request === this.fuzzyRequest) this.suggestionsError = '联想加载失败，可按 Enter 直接查询。';
            } finally {
                if (request === this.fuzzyRequest) { this.suggestionsLoading = false; this.fuzzyController = null; }
            }
        },

        moveSuggestion(direction) {
            if (!this.suggestions.length || !this.suggestionsOpen) return;
            this.activeSuggestion = this.activeSuggestion < 0
                ? (direction > 0 ? 0 : this.suggestions.length - 1)
                : (this.activeSuggestion + direction + this.suggestions.length) % this.suggestions.length;
            this.$nextTick(() => document.getElementById(`suggestion-${this.activeSuggestion}`)?.scrollIntoView({ block: 'nearest' }));
        },

        selectSuggestion(word) { this.searchWord = word; this.search(); },
        handleShortcut(event) {
            if (event.key !== '/' || event.ctrlKey || event.metaKey || event.altKey || this.showLogin) return;
            if (event.target.closest('input, textarea, select, [contenteditable="true"]')) return;
            event.preventDefault();
            this.$refs.searchInput.focus();
        },

        historyStorageKey() {
            return this.user?.id ? `search_history:${this.user.id}` : '';
        },
        loadHistory() {
            try { localStorage.removeItem('search_history'); } catch (_) { /* Drop the old shared history key. */ }
            const key = this.historyStorageKey();
            if (!key) { this.searchHistory = []; return; }
            try {
                const history = JSON.parse(localStorage.getItem(key) || '[]');
                this.searchHistory = Array.isArray(history)
                    ? [...new Set(history.filter(word => typeof word === 'string' && word.trim()))].slice(0, this.maxHistorySize) : [];
            } catch (_) { this.searchHistory = []; }
        },
        persistHistory() {
            const key = this.historyStorageKey();
            if (!key) return;
            try { localStorage.setItem(key, JSON.stringify(this.searchHistory)); } catch (_) { /* Private storage may be unavailable. */ }
        },
        addToHistory(word) {
            this.searchHistory = [word, ...this.searchHistory.filter(item => item.toLowerCase() !== word.toLowerCase())].slice(0, this.maxHistorySize);
            this.persistHistory();
        },
        removeFromHistory(word) { this.searchHistory = this.searchHistory.filter(item => item !== word); this.persistHistory(); },
        clearHistory() { this.searchHistory = []; this.persistHistory(); },

        sanitizeHtml(html) {
            return DOMPurify.sanitize(html || '', {
                ALLOWED_TAGS: ['div', 'span', 'p', 'b', 'i', 'u', 'a', 'img', 'audio', 'source', 'br', 'hr', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'ul', 'ol', 'li', 'table', 'tr', 'td', 'th', 'thead', 'tbody', 'font', 'sup', 'sub', 'blockquote', 'style', 'dl', 'dt', 'dd', 'pre', 'code', 'em', 'strong', 'small', 'mark', 'abbr', 'ruby', 'rt', 'rp', 'bdo'],
                ALLOWED_ATTR: ['href', 'src', 'alt', 'title', 'class', 'style', 'width', 'height', 'controls', 'type', 'color', 'size', 'face', 'cellpadding', 'cellspacing', 'border', 'align', 'valign', 'bgcolor', 'nowrap', 'colspan', 'rowspan', 'scope', 'id', 'name'],
                FORCE_BODY: true, ALLOW_DATA_ATTR: true
            });
        },
        renderDefinition(element, html) {
            // Keep dictionary CSS from changing the surrounding app or other definitions.
            const root = element.shadowRoot || element.attachShadow({ mode: 'open' });
            const generation = (element._renderGeneration || 0) + 1;
            element._renderGeneration = generation;
            root.innerHTML = `<style>:host{display:block;font-size:16px;line-height:1.8;overflow-wrap:anywhere}img{max-width:100%;height:auto}table{max-width:100%;border-collapse:collapse}a{color:#22685a}p{margin:.6em 0}</style>${this.sanitizeHtml(html)}`;
            this.hydrateAssets(root, generation, element);
        },
        normalizeAssetURL(url) {
            if (typeof url !== 'string') return '';
            const normalized = url.replace(/\\/g, '/').replace(/\/{2,}/g, '/');
            return normalized.startsWith('/api/v1/assets/') ? normalized : '';
        },
        async authorizedBlob(url) {
            const assetURL = this.normalizeAssetURL(url);
            if (!assetURL) return url;
            const cached = this.assetBlobs.get(assetURL);
            if (cached) return cached;
            const response = await this.fetchAsset(assetURL);
            if (!response.ok) throw new Error('asset');
            const blobURL = URL.createObjectURL(await response.blob());
            this.assetBlobs.set(assetURL, blobURL);
            return blobURL;
        },
        async fetchAsset(url) {
            const headers = {};
            if (this.token) headers.Authorization = `Bearer ${this.token}`;
            let response = await fetch(url, { headers });
            if (response.status === 401 && this.token) {
                const refreshed = await this.refreshToken();
                if (refreshed) {
                    headers.Authorization = `Bearer ${this.token}`;
                    response = await fetch(url, { headers });
                }
            }
            return response;
        },
        revokeAssetBlobs() {
            for (const blobURL of this.assetBlobs.values()) URL.revokeObjectURL(blobURL);
            this.assetBlobs.clear();
        },
        async hydrateAssets(root, generation, element) {
            const current = () => element._renderGeneration === generation;
            const nodes = [...root.querySelectorAll('[src], [href]')];
            await Promise.all(nodes.map(async node => {
                for (const attr of ['src', 'href']) {
                    const value = node.getAttribute(attr);
                    if (!current() || !this.normalizeAssetURL(value)) continue;
                    try {
                        const next = await this.authorizedBlob(value);
                        if (current() && next !== value) node.setAttribute(attr, next);
                    } catch (_) { /* Leave the original URL when the asset cannot be loaded. */ }
                }
            }));
            for (const style of root.querySelectorAll('style')) {
                let css = style.textContent || '';
                for (const assetURL of [...new Set(css.match(/\/api\/v1\/assets\/[^"')\s]+/g) || [])]) {
                    if (!current()) return;
                    try { css = css.split(assetURL).join(await this.authorizedBlob(assetURL)); }
                    catch (_) { /* Keep the original CSS reference. */ }
                }
                if (current()) style.textContent = css;
            }
        },
        async playAudio(url) {
            if (!url) return;
            try {
                this.$refs.audioPlayer.src = this.normalizeAssetURL(url) ? await this.authorizedBlob(url) : url;
                await this.$refs.audioPlayer.play();
            } catch (_) { this.error = '发音播放失败，请稍后重试。'; }
        },

        openLogin() { this.showLogin = true; this.$nextTick(() => this.$refs.loginUsername.focus()); },
        closeLogin() { this.showLogin = false; this.$refs.searchInput.focus(); },
        trapLoginFocus(event) {
            const items = Array.from(event.currentTarget.querySelectorAll('button:not(:disabled), input'));
            const first = items[0], last = items[items.length - 1];
            if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last.focus(); }
            else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first.focus(); }
        },
        async login() {
            if (this.loginLoading) return;
            this.loginLoading = true;
            this.loginError = null;
            try {
                const response = await fetch('/api/v1/auth/login', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(this.loginForm) });
                const data = await response.json();
                if (data.code === 0) {
                    this.token = data.data.access_token;
                    this.user = data.data.user;
                    localStorage.setItem('token', this.token);
                    localStorage.setItem('refresh_token', data.data.refresh_token);
                    localStorage.setItem('user', JSON.stringify(this.user));
                    this.closeLogin();
                    this.loginForm = { username: '', password: '' };
                    this.loadHistory();
                    await this.loadDicts();
                    if (this.searchWord.trim()) await this.search();
                } else this.loginError = '登录失败，请检查用户名和密码。';
            } catch (_) { this.loginError = '网络连接失败，请重试。'; }
            finally { this.loginLoading = false; }
        },
        logout() {
            this.cancelSuggestions();
            this.cancelSearch();
            this.dictRequest++;
            this.revokeAssetBlobs();
            this.token = null;
            this.user = null;
            this.dicts = [];
            this.selectedDict = '';
            this.results = [];
            this.searched = false;
            this.error = null;
            this.searchHistory = [];
            localStorage.removeItem('token');
            localStorage.removeItem('refresh_token');
            localStorage.removeItem('user');
            try { localStorage.removeItem('search_history'); } catch (_) { /* The shared key is unused after logout. */ }
        },
        async fetchWithAuth(url, options = {}) {
            const headers = { 'Content-Type': 'application/json', ...options.headers };
            if (this.token) headers.Authorization = `Bearer ${this.token}`;
            const response = await fetch(url, { ...options, headers });
            if (response.status === 401 && this.token) {
                const refreshed = await this.refreshToken();
                if (options.signal?.aborted) throw new DOMException('Aborted', 'AbortError');
                if (refreshed) {
                    headers.Authorization = `Bearer ${this.token}`;
                    return fetch(url, { ...options, headers }).then(result => result.json());
                }
                this.logout();
                this.openLogin();
            }
            return response.json();
        },
        async refreshToken() {
            if (!this.refreshPromise) {
                this.refreshPromise = this.performTokenRefresh().finally(() => { this.refreshPromise = null; });
            }
            return this.refreshPromise;
        },
        async performTokenRefresh() {
            const token = this.token;
            try {
                const refresh = localStorage.getItem('refresh_token');
                if (!refresh) return false;
                const response = await fetch('/api/v1/auth/refresh', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ refresh_token: refresh }) });
                const data = await response.json();
                if (data.code === 0 && token === this.token) {
                    this.token = data.data.access_token;
                    localStorage.setItem('token', this.token);
                    return true;
                }
            } catch (_) { /* The caller returns to login if the session cannot be refreshed. */ }
            return false;
        }
    };
}
