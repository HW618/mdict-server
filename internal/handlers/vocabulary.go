package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/HW618/mdict-server/internal/store"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type VocabularyHandler struct{ store *store.VocabularyStore }

func NewVocabularyHandler(s *store.VocabularyStore) *VocabularyHandler {
	return &VocabularyHandler{store: s}
}

func vocabularyWord(raw string) (string, bool) {
	word := strings.TrimSpace(raw)
	if word == "" || !utf8.ValidString(word) || utf8.RuneCountInString(word) > 200 {
		return "", false
	}
	for _, r := range raw {
		if unicode.IsControl(r) {
			return "", false
		}
	}
	return word, true
}

func vocabularyInvalid(c *gin.Context) {
	c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "message": "单词需为 1–200 个字符，级别需为 1–5。", "data": nil})
}

func vocabularyFailure(c *gin.Context, err error) {
	log.Error().Err(err).Msg("Vocabulary operation failed")
	c.JSON(http.StatusInternalServerError, gin.H{"code": 50002, "message": "生词本暂时不可用，请重试。", "data": nil})
}

func vocabularyOK(c *gin.Context, data any) {
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": data})
}

func (h *VocabularyHandler) Get(c *gin.Context) {
	word, ok := vocabularyWord(c.Query("word"))
	if !ok {
		vocabularyInvalid(c)
		return
	}
	entry, err := h.store.Get(c.GetString("userID"), word)
	if err != nil {
		vocabularyFailure(c, err)
		return
	}
	vocabularyOK(c, entry)
}

func (h *VocabularyHandler) Set(c *gin.Context) {
	var req struct {
		Word  string `json:"word"`
		Level int    `json:"level"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		vocabularyInvalid(c)
		return
	}
	word, ok := vocabularyWord(req.Word)
	if !ok || req.Level < 1 || req.Level > 5 {
		vocabularyInvalid(c)
		return
	}
	entry, err := h.store.Set(c.GetString("userID"), word, req.Level)
	if err != nil {
		vocabularyFailure(c, err)
		return
	}
	vocabularyOK(c, entry)
}

func (h *VocabularyHandler) Delete(c *gin.Context) {
	word, ok := vocabularyWord(c.Query("word"))
	if !ok {
		vocabularyInvalid(c)
		return
	}
	if err := h.store.Delete(c.GetString("userID"), word); err != nil {
		vocabularyFailure(c, err)
		return
	}
	vocabularyOK(c, gin.H{"word": word, "level": 0})
}

func (h *VocabularyHandler) List(c *gin.Context) {
	parse := func(key string, fallback, min, max int) (int, bool) {
		raw := c.Query(key)
		if raw == "" {
			return fallback, true
		}
		n, err := strconv.Atoi(raw)
		return n, err == nil && n >= min && n <= max
	}
	level, okLevel := parse("level", 0, 1, 5)
	page, okPage := parse("page", 1, 1, 1000000)
	pageSize, okSize := parse("page_size", 30, 1, 100)
	keyword := strings.TrimSpace(c.Query("keyword"))
	if !okLevel || !okPage || !okSize || utf8.RuneCountInString(keyword) > 200 {
		vocabularyInvalid(c)
		return
	}
	items, total, err := h.store.List(c.GetString("userID"), level, keyword, page, pageSize)
	if err != nil {
		vocabularyFailure(c, err)
		return
	}
	vocabularyOK(c, gin.H{"items": items, "total": total})
}
