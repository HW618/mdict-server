package mcp

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *Service) registerPrompts() {
	s.server.AddPrompt(&sdkmcp.Prompt{
		Name:        "mdict_define_word",
		Title:       "Define a word",
		Description: "Fetch one headword from the installed dictionaries and ask the model to explain only those senses.",
		Arguments: []*sdkmcp.PromptArgument{
			{Name: "word", Title: "Headword", Description: "Word or phrase to define, for example ephemeral.", Required: true},
			{Name: "dict_id", Title: "Dictionary", Description: "Optional dictionary id from mdict_list_dictionaries."},
		},
	}, s.defineWord)

	s.server.AddPrompt(&sdkmcp.Prompt{
		Name:        "mdict_explore_prefix",
		Title:       "Explore a prefix",
		Description: "Fetch headwords that start with a prefix and ask the model to help the user pick one to define.",
		Arguments: []*sdkmcp.PromptArgument{
			{Name: "keyword", Title: "Prefix", Description: "At least 2 characters, for example phil.", Required: true},
			{Name: "dict_id", Title: "Dictionary", Description: "Optional dictionary id from mdict_list_dictionaries."},
			{Name: "limit", Title: "Limit", Description: "How many headwords to include, from 1 to 20. Default 10."},
		},
	}, s.explorePrefix)
}

func (s *Service) defineWord(ctx context.Context, req *sdkmcp.GetPromptRequest) (*sdkmcp.GetPromptResult, error) {
	word := strings.TrimSpace(req.Params.Arguments["word"])
	if word == "" {
		return nil, fmt.Errorf("argument word is required. Call this prompt with the headword to explain, for example word=hello")
	}
	dictID := strings.TrimSpace(req.Params.Arguments["dict_id"])
	result, _, err := s.searchWord(ctx, nil, searchWordInput{Word: word, DictID: dictID, ResponseFormat: formatMarkdown})
	body := ""
	if err != nil {
		body = fmt.Sprintf("The dictionary lookup for %q failed: %s\nTell the user, then call mdict_fuzzy_search if a shorter prefix might match.", word, err.Error())
	} else if result != nil && len(result.Content) > 0 {
		if block, ok := result.Content[0].(*sdkmcp.TextContent); ok {
			body = block.Text
		}
	}
	if body == "" {
		body = fmt.Sprintf("No dictionary text was returned for %q.", word)
	}
	instruction := fmt.Sprintf("Explain %q to the user using only the dictionary text below. Do not add senses that are not written there. Mention an audio path when one is present. If the lookup failed, say so and suggest the next lookup.", word)
	return promptResult("Define "+word, instruction+"\n\n"+body), nil
}

func (s *Service) explorePrefix(ctx context.Context, req *sdkmcp.GetPromptRequest) (*sdkmcp.GetPromptResult, error) {
	keyword := strings.TrimSpace(req.Params.Arguments["keyword"])
	if keyword == "" {
		return nil, fmt.Errorf("argument keyword is required. Call this prompt with a prefix of at least 2 characters, for example keyword=app")
	}
	limit := 10
	if raw := strings.TrimSpace(req.Params.Arguments["limit"]); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 20 {
			return nil, fmt.Errorf("argument limit must be a whole number from 1 to 20, not %q", raw)
		}
		limit = parsed
	}
	result, _, err := s.fuzzySearch(ctx, nil, fuzzySearchInput{
		Keyword:        keyword,
		DictID:         req.Params.Arguments["dict_id"],
		Limit:          limit,
		ResponseFormat: formatMarkdown,
	})
	body := ""
	if err != nil {
		body = "The prefix search failed: " + err.Error()
	} else if result != nil && len(result.Content) > 0 {
		if block, ok := result.Content[0].(*sdkmcp.TextContent); ok {
			body = block.Text
		}
	}
	instruction := fmt.Sprintf("The user is looking for a headword starting with %q. Show the matches below, ask which one they want defined, and do not invent headwords that are not in the list.", keyword)
	return promptResult("Explore "+keyword, instruction+"\n\n"+body), nil
}

func promptResult(description, text string) *sdkmcp.GetPromptResult {
	return &sdkmcp.GetPromptResult{
		Description: description,
		Messages: []*sdkmcp.PromptMessage{{
			Role:    "user",
			Content: &sdkmcp.TextContent{Text: text},
		}},
	}
}
