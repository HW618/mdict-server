package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *Service) registerResources() {
	s.server.AddResource(&sdkmcp.Resource{
		URI:         "mdict://dictionaries",
		Name:        "dictionaries",
		Title:       "Installed dictionaries",
		Description: "JSON catalog of dictionaries installed on this server. Each id can be passed as dict_id to the search tools.",
		MIMEType:    "application/json",
	}, s.readDictionaries)

	s.server.AddResourceTemplate(&sdkmcp.ResourceTemplate{
		URITemplate: "mdict://dictionaries/{dict_id}",
		Name:        "dictionary",
		Title:       "Dictionary",
		Description: "JSON metadata for one dictionary. dict_id comes from mdict://dictionaries.",
		MIMEType:    "application/json",
	}, s.readDictionary)

	s.server.AddResourceTemplate(&sdkmcp.ResourceTemplate{
		URITemplate: "mdict://entries/{dict_id}/{word}",
		Name:        "entry",
		Title:       "Dictionary entry",
		Description: "Markdown definition of one headword in one dictionary. Percent-encode the headword. Example: mdict://entries/abc12345/hello",
		MIMEType:    "text/markdown",
	}, s.readEntry)
}

func (s *Service) readDictionaries(ctx context.Context, req *sdkmcp.ReadResourceRequest) (*sdkmcp.ReadResourceResult, error) {
	catalog, err := s.dictionaryCatalog(false)
	if err != nil {
		return nil, err
	}
	payload, err := json.MarshalIndent(listDictionariesOutput{
		Dictionaries: catalog,
		Total:        len(catalog),
		EnabledCount: enabledIn(catalog),
	}, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("could not encode the dictionary catalog. Retry the read")
	}
	return textResource(req.Params.URI, "application/json", string(payload)), nil
}

func (s *Service) readDictionary(_ context.Context, req *sdkmcp.ReadResourceRequest) (*sdkmcp.ReadResourceResult, error) {
	dictID, _, ok := resourcePath(req.Params.URI, "dictionaries", 1)
	if !ok {
		return nil, sdkmcp.ResourceNotFoundError(req.Params.URI)
	}
	catalog, err := s.dictionaryCatalog(false)
	if err != nil {
		return nil, err
	}
	for _, dict := range catalog {
		if dict.ID != dictID {
			continue
		}
		payload, err := json.MarshalIndent(dict, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("could not encode dictionary %q. Retry the read", dictID)
		}
		return textResource(req.Params.URI, "application/json", string(payload)), nil
	}
	return nil, sdkmcp.ResourceNotFoundError(req.Params.URI)
}

func (s *Service) readEntry(ctx context.Context, req *sdkmcp.ReadResourceRequest) (*sdkmcp.ReadResourceResult, error) {
	dictID, word, ok := resourcePath(req.Params.URI, "entries", 2)
	if !ok {
		return nil, sdkmcp.ResourceNotFoundError(req.Params.URI)
	}
	result, _, err := s.searchWord(ctx, nil, searchWordInput{Word: word, DictID: dictID, ResponseFormat: formatMarkdown})
	if err != nil {
		return nil, sdkmcp.ResourceNotFoundError(req.Params.URI)
	}
	text := ""
	if result != nil && len(result.Content) > 0 {
		if block, ok := result.Content[0].(*sdkmcp.TextContent); ok {
			text = block.Text
		}
	}
	if text == "" {
		return nil, sdkmcp.ResourceNotFoundError(req.Params.URI)
	}
	return textResource(req.Params.URI, "text/markdown", text), nil
}

func resourcePath(rawURI, host string, parts int) (string, string, bool) {
	parsed, err := url.Parse(rawURI)
	if err != nil || parsed.Scheme != "mdict" || parsed.Host != host {
		return "", "", false
	}
	segments := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(segments) != parts {
		return "", "", false
	}
	for i, segment := range segments {
		decoded, err := url.PathUnescape(segment)
		if err != nil || decoded == "" {
			return "", "", false
		}
		segments[i] = decoded
	}
	if parts == 1 {
		return segments[0], "", true
	}
	return segments[0], segments[1], true
}

func textResource(uri, mimeType, text string) *sdkmcp.ReadResourceResult {
	return &sdkmcp.ReadResourceResult{
		Contents: []*sdkmcp.ResourceContents{{
			URI:      uri,
			MIMEType: mimeType,
			Text:     text,
		}},
	}
}

func enabledIn(catalog []dictionaryInfo) int {
	count := 0
	for _, dict := range catalog {
		if dict.IsEnabled {
			count++
		}
	}
	return count
}
