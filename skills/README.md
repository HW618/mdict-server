# Claude Code Skills for mdict-server

This directory contains Claude Code skills that enable AI agents to interact with the mdict-server dictionary service.

## Available Skills

### mdict-query

Look up word definitions, translations, and examples from the mdict-server using curl.

**Trigger:** When a user asks about a word's meaning, definition, translation, spelling, usage, or wants to search for words in the dictionary.

**Setup:**

Set the required environment variables:

```bash
export MDICT_SERVER_URL="http://localhost:8080"  # Your mdict-server URL
export MDICT_API_TOKEN="mdtk_your_token_here"     # Your API token
```

**Usage:**

Just ask about any word:
- "What does 'ephemeral' mean?"
- "Define 'serendipity'"
- "Look up the word 'ubiquitous'"
- "Find words starting with 'phil'"

Or invoke explicitly: `/mdict-query hello`

**API Endpoints Used:**

| Endpoint | Auth | Description |
|----------|------|-------------|
| `GET /api/v1/search?word=<word>` | API Token | Exact word lookup |
| `GET /api/v1/search/fuzzy?keyword=<keyword>` | API Token | Fuzzy/prefix search |
| `GET /api/v1/dicts` | API Token | List available dictionaries |
| `GET /api/v1/health` | None | Health check |
