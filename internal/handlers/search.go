package handlers

import (
	"net/http"
	"strconv"
	"strings"

	markdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/gin-gonic/gin"
	"github.com/HW618/mdict-server/internal/dict"
)

// SearchHandler handles search endpoints
type SearchHandler struct {
	engine *dict.Engine
}

// NewSearchHandler creates a new search handler
func NewSearchHandler(engine *dict.Engine) *SearchHandler {
	return &SearchHandler{
		engine: engine,
	}
}

// Search handles exact word search
func (h *SearchHandler) Search(c *gin.Context) {
	word := c.Query("word")
	if word == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": "word parameter is required",
			"data":    nil,
		})
		return
	}

	dictID := c.Query("dict_id")

	result, err := h.engine.Search(word, dictID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50001,
			"message": "Search failed",
			"data":    nil,
		})
		return
	}

	if len(result.Results) == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    40401,
			"message": "No results found",
			"data":    nil,
		})
		return
	}

	// If markdown=true, convert HTML to Markdown and omit the HTML field
	if strings.ToLower(c.Query("markdown")) == "true" {
		for i := range result.Results {
			md, err := markdown.ConvertString(result.Results[i].HTML)
			if err == nil {
				result.Results[i].Markdown = strings.TrimSpace(md)
			}
			result.Results[i].HTML = "" // omit from JSON via omitempty
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}

// FuzzySearch handles fuzzy word search
func (h *SearchHandler) FuzzySearch(c *gin.Context) {
	keyword := c.Query("keyword")
	if len(keyword) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": "keyword must be at least 2 characters",
			"data":    nil,
		})
		return
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	result, err := h.engine.FuzzySearch(keyword, "", page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50001,
			"message": "Search failed",
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}
