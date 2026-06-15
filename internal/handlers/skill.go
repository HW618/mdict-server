package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SkillHandler handles agent skill endpoints
type SkillHandler struct {
	serverURL string
}

// NewSkillHandler creates a new skill handler
func NewSkillHandler(serverURL string) *SkillHandler {
	return &SkillHandler{
		serverURL: serverURL,
	}
}

// GetSkill returns the OpenAPI 3.0 skill configuration
func (h *SkillHandler) GetSkill(c *gin.Context) {
	skill := map[string]interface{}{
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title":       "Mdict Dictionary Service",
			"description": "Query word definitions from Mdx dictionaries. Server URL: " + h.serverURL + ". Use Authorization: Bearer <YOUR_TOKEN> header.",
			"version":     "1.0.0",
		},
		"servers": []map[string]string{
			{"url": h.serverURL},
		},
		"paths": map[string]interface{}{
			"/api/v1/search": map[string]interface{}{
				"get": map[string]interface{}{
					"operationId": "searchWord",
					"summary":     "Search word definition",
					"description": "Query the exact definition of a word from enabled dictionaries",
					"parameters": []map[string]interface{}{
						{
							"name":        "word",
							"in":          "query",
							"required":    true,
							"schema":      map[string]string{"type": "string"},
							"description": "The word to search",
						},
						{
							"name":        "dict_id",
							"in":          "query",
							"required":    false,
							"schema":      map[string]string{"type": "string"},
							"description": "Optional dictionary ID to search in",
						},
					},
					"security": []map[string][]string{
						{"bearerAuth": {}},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Word definition found",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]string{"$ref": "#/components/schemas/SearchResult"},
								},
							},
						},
					},
				},
			},
		},
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"bearerAuth": map[string]interface{}{
					"type":         "http",
					"scheme":       "bearer",
					"bearerFormat": "JWT",
				},
			},
			"schemas": map[string]interface{}{
				"SearchResult": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"code":    map[string]string{"type": "integer"},
						"message": map[string]string{"type": "string"},
						"data": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"word": map[string]string{"type": "string"},
								"results": map[string]interface{}{
									"type": "array",
									"items": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"dict_id":   map[string]string{"type": "string"},
											"dict_name": map[string]string{"type": "string"},
											"html":      map[string]string{"type": "string"},
											"has_audio": map[string]string{"type": "boolean"},
											"audio_url": map[string]string{"type": "string"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	c.JSON(http.StatusOK, skill)
}
