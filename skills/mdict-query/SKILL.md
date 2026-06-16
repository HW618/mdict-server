---
name: mdict-query
description: |-
  Look up word definitions, translations, and examples from the mdict-server dictionary service using curl.
  TRIGGER whenever: the user asks about a word's meaning, definition, translation, spelling, usage, or example;
  the user wants to look up or search for any word or phrase in a dictionary;
  the user asks "what does X mean", "define X", "translate X", "how to spell X", "synonyms for X";
  the user wants to find similar or related words via fuzzy/prefix search;
  the user asks to list available dictionaries on the server.
  SKIP only when: the user is asking about code in this project (not word lookups);
  the user explicitly asks to use a different dictionary tool or website.
---

# Mdict Dictionary Query

Query word definitions from the mdict-server dictionary service via its REST API using curl.

## Environment Variables

Before making any API calls, verify these environment variables are set. If either is missing, tell the user to set them.

| Variable | Required | Description |
|----------|----------|-------------|
| `MDICT_SERVER_URL` | Yes | Base URL of the mdict-server (e.g., `http://localhost:8080`) |
| `MDICT_API_TOKEN` | Yes | API token for authentication (starts with `mdtk_`) |

Check with:
```bash
echo "URL: ${MDICT_SERVER_URL:-NOT SET}"
echo "TOKEN: ${MDICT_API_TOKEN:+SET}"
```

## API Endpoints

All endpoints use the base URL from `MDICT_SERVER_URL` and require `Authorization: Bearer ${MDICT_API_TOKEN}` header (except health check).

### Exact Search — look up a word

**IMPORTANT:** Always add `markdown=true` to return Markdown instead of HTML, which significantly reduces token consumption.

```bash
curl -s "${MDICT_SERVER_URL}/api/v1/search?word=<WORD>&markdown=true" \
  -H "Authorization: Bearer ${MDICT_API_TOKEN}"
```

- `word` (required): the word to search
- `dict_id` (optional): search in a specific dictionary only
- `markdown` (required, must be `true`): returns definition in Markdown format (no HTML field) to minimize token usage

**Response** (`code == 0` means success, `markdown=true`):
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "word": "hello",
    "results": [
      {
        "dict_id": "abc12345",
        "dict_name": "Oxford Dictionary",
        "markdown": "# hello\n\n1. CONVENTION — You say **hello** to someone when you meet them.",
        "has_audio": true,
        "audio_url": "/api/v1/assets/abc12345/hello.mp3"
      }
    ]
  }
}
```

> **Note:** With `markdown=true`, the `html` field is omitted from the response. Only `markdown` is returned, which is far more token-efficient for LLM consumption.

### Fuzzy Search — find similar words

```bash
curl -s "${MDICT_SERVER_URL}/api/v1/search/fuzzy?keyword=<KEYWORD>&page=1&page_size=10" \
  -H "Authorization: Bearer ${MDICT_API_TOKEN}"
```

- `keyword` (required, min 2 chars): search prefix/keyword
- `dict_id` (optional): limit to one dictionary
- `page` (optional, default 1): page number
- `page_size` (optional, default 20, max 100): results per page

**Response** (`code == 0` means success):
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {"word": "hello", "dict_id": "abc12345", "dict_name": "Oxford Dictionary"},
      {"word": "help", "dict_id": "abc12345", "dict_name": "Oxford Dictionary"}
    ],
    "total": 150,
    "page": 1,
    "page_size": 10,
    "total_pages": 15
  }
}
```

### List Dictionaries

```bash
curl -s "${MDICT_SERVER_URL}/api/v1/dicts" \
  -H "Authorization: Bearer ${MDICT_API_TOKEN}"
```

Returns all dictionaries with metadata: `id`, `filename`, `title`, `description`, `entry_count`, `is_enabled`.

### Health Check (no auth needed)

```bash
curl -s "${MDICT_SERVER_URL}/api/v1/health"
```

## Workflow

When the user asks about a word:

1. **Verify env vars** — check `MDICT_SERVER_URL` and `MDICT_API_TOKEN` are set. If not, instruct the user to set them.

2. **Exact search first** — run the exact search curl command with the word, **always with `markdown=true`**.

3. **Handle results:**
   - If `code == 0` and `data.results` is non-empty, read the `markdown` field from each result and present it to the user. Mention which dictionary each definition comes from. If `has_audio` is true, note the audio URL.
   - If `code == 40401` (not found), fall back to fuzzy search with the word as keyword.
   - If fuzzy search also returns no results, tell the user the word was not found.

4. **When the user asks for similar/spelled words**, use fuzzy search directly.

5. **When presenting results**, use the Markdown content directly. Group by dictionary if multiple dictionaries return results.

## Error Codes

| Code | Meaning | Action |
|------|---------|--------|
| `0` | Success | Parse and present results |
| `40001` | Bad request (missing word/keyword) | Check the query parameter |
| `40101` | Not authenticated | Check `MDICT_API_TOKEN` is set correctly |
| `40301` | Permission denied | Token may lack `can_use_api` permission |
| `40401` | Not found | Try fuzzy search instead |
| `42901` | Rate limited | Wait a moment and retry |
| `50001` | Server error | Tell user the server has an issue |

## Example

User: "What does ephemeral mean?"

```bash
# Step 1: Exact search (always use markdown=true)
curl -s "${MDICT_SERVER_URL}/api/v1/search?word=ephemeral&markdown=true" \
  -H "Authorization: Bearer ${MDICT_API_TOKEN}"

# Step 2: If not found, fuzzy search
curl -s "${MDICT_SERVER_URL}/api/v1/search/fuzzy?keyword=ephemeral&page=1&page_size=10" \
  -H "Authorization: Bearer ${MDICT_API_TOKEN}"
```
