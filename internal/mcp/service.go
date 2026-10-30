// Package mcp exposes the dictionary engine over the Model Context Protocol.
// The HTTP endpoint is a stateless Streamable HTTP server mounted at /mcp.
package mcp

import (
	"net/http"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/HW618/mdict-server/internal/dict"
	"github.com/HW618/mdict-server/internal/store"
)

const instructions = `Mdict looks up headwords in the dictionaries installed on this server.
Call mdict_list_dictionaries before using dict_id. Use mdict_search_word for an exact entry and mdict_fuzzy_search when the spelling is incomplete.
Definitions are Markdown. An audio_url is a path on this same server and still needs the bearer token used for this MCP session.
Read mdict://dictionaries for the catalog and mdict://entries/{dict_id}/{word} for one entry.`

// Service is the MCP server backed by the dictionary engine.
type Service struct {
	engine *dict.Engine
	dicts  *store.DictStore
	server *sdkmcp.Server
	http   http.Handler
}

// New builds the MCP server and its Streamable HTTP handler.
// version is the mdict-server version reported during initialization.
func New(engine *dict.Engine, dicts *store.DictStore, version string) *Service {
	if version == "" {
		version = "dev"
	}
	server := sdkmcp.NewServer(&sdkmcp.Implementation{
		Name:        "mdict",
		Title:       "Mdict Dictionary",
		Version:     version,
		Description: "Look up headwords in the MDX dictionaries loaded by this mdict-server.",
	}, &sdkmcp.ServerOptions{
		Instructions: instructions,
		// Empty capability objects advertise tools, resources, and prompts
		// without list-changed notifications. Stateless HTTP cannot deliver those.
		Capabilities: &sdkmcp.ServerCapabilities{
			Tools:     &sdkmcp.ToolCapabilities{},
			Resources: &sdkmcp.ResourceCapabilities{},
			Prompts:   &sdkmcp.PromptCapabilities{},
		},
	})

	svc := &Service{
		engine: engine,
		dicts:  dicts,
		server: server,
	}
	svc.registerTools()
	svc.registerResources()
	svc.registerPrompts()
	svc.http = sdkmcp.NewStreamableHTTPHandler(func(*http.Request) *sdkmcp.Server {
		return server
	}, &sdkmcp.StreamableHTTPOptions{
		Stateless:    true,
		JSONResponse: true,
	})
	return svc
}

// Handler is the Streamable HTTP endpoint. Mount it at /mcp.
func (s *Service) Handler() http.Handler {
	return s.http
}

func readOnlyAnnotations(title string) *sdkmcp.ToolAnnotations {
	closedWorld := false
	notDestructive := false
	return &sdkmcp.ToolAnnotations{
		Title:           title,
		ReadOnlyHint:    true,
		IdempotentHint:  true,
		DestructiveHint: &notDestructive,
		OpenWorldHint:   &closedWorld,
	}
}
