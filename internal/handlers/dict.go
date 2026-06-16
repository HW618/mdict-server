package handlers

import (
	"io"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/HW618/mdict-server/internal/dict"
	"github.com/HW618/mdict-server/internal/models"
	"github.com/HW618/mdict-server/internal/store"
	"github.com/rs/zerolog/log"
)

// DictHandler handles dictionary endpoints
type DictHandler struct {
	engine         *dict.Engine
	dictStore      *store.DictStore
	dictDir        string
	maxUploadBytes int64
}

// NewDictHandler creates a new dict handler
func NewDictHandler(engine *dict.Engine, dictStore *store.DictStore, dictDir string, maxUploadBytes int64) *DictHandler {
	return &DictHandler{
		engine:         engine,
		dictStore:      dictStore,
		dictDir:        dictDir,
		maxUploadBytes: maxUploadBytes,
	}
}

// List returns all dictionaries
func (h *DictHandler) List(c *gin.Context) {
	dicts, err := h.dictStore.List()
	if err != nil {
		log.Error().Err(err).Msg("Failed to list dictionaries")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50002,
			"message": "Failed to list dictionaries",
			"data":    nil,
		})
		return
	}

	// Convert to response
	responses := make([]models.DictResponse, len(dicts))
	for i, d := range dicts {
		responses[i] = d.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    responses,
	})
}

// UpdateStatus updates dictionary enabled status
func (h *DictHandler) UpdateStatus(c *gin.Context) {
	dictID := c.Param("id")

	var req models.DictStatusUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": "Invalid request body",
			"data":    nil,
		})
		return
	}

	// Check if dictionary exists
	_, err := h.dictStore.GetByID(dictID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    40401,
			"message": "Dictionary not found",
			"data":    nil,
		})
		return
	}

	// Update status in database
	if err := h.dictStore.UpdateStatus(dictID, req.IsEnabled); err != nil {
		log.Error().Err(err).Msg("Failed to update dictionary status")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50002,
			"message": "Failed to update dictionary status",
			"data":    nil,
		})
		return
	}

	// Update in engine
	if req.IsEnabled {
		// Reload dictionary
		if err := h.engine.Reload(dictID); err != nil {
			log.Error().Err(err).Msg("Failed to reload dictionary")
		}
	} else {
		// Unload dictionary
		h.engine.Unload(dictID)
	}

	log.Info().
		Str("audit", "true").
		Str("action", "dict_status_changed").
		Str("dict_id", dictID).
		Bool("is_enabled", req.IsEnabled).
		Str("operator_id", c.GetString("userID")).
		Msg("Dictionary status updated")

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Dictionary status updated",
	})
}

// Upload handles dictionary file upload
func (h *DictHandler) Upload(c *gin.Context) {
	// Parse multipart form with memory buffer limit
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": "Failed to parse upload: " + err.Error(),
			"data":    nil,
		})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": "File is required",
			"data":    nil,
		})
		return
	}
	defer file.Close()

	// Sanitize filename to prevent path traversal
	safeFilename := filepath.Base(header.Filename)
	if safeFilename == "." || safeFilename == "/" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": "Invalid filename",
			"data":    nil,
		})
		return
	}

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(safeFilename))
	if ext != ".mdx" && ext != ".mdd" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": "Only .mdx and .mdd files are allowed",
			"data":    nil,
		})
		return
	}

	// Check if dictionary already exists (for .mdx files)
	if ext == ".mdx" {
		exists, err := h.dictStore.ExistsByFilename(safeFilename)
		if err != nil {
			log.Error().Err(err).Msg("Failed to check dictionary existence")
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    50002,
				"message": "Failed to check dictionary existence",
				"data":    nil,
			})
			return
		}
		if exists {
			c.JSON(http.StatusConflict, gin.H{
				"code":    40901,
				"message": "Dictionary already exists",
				"data":    nil,
			})
			return
		}
	}

	// Stream file to disk via temp file, then rename atomically
	dst := filepath.Join(h.dictDir, safeFilename)
	tmpDst := dst + ".tmp"

	tmpFile, err := os.Create(tmpDst)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create temp file")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50001,
			"message": "Failed to save file",
			"data":    nil,
		})
		return
	}
	defer os.Remove(tmpDst)

	written, err := io.Copy(tmpFile, file)
	tmpFile.Close()
	if err != nil {
		log.Error().Err(err).Msg("Failed to write uploaded file")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50001,
			"message": "Failed to save file",
			"data":    nil,
		})
		return
	}

	// Check file size limit after streaming
	if h.maxUploadBytes > 0 && written > h.maxUploadBytes {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"code":    41301,
			"message": fmt.Sprintf("File too large (max %d MB)", h.maxUploadBytes/(1024*1024)),
			"data":    nil,
		})
		return
	}

	// Atomically move temp file to final destination
	if err := os.Rename(tmpDst, dst); err != nil {
		log.Error().Err(err).Msg("Failed to rename temp file")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50001,
			"message": "Failed to save file",
			"data":    nil,
		})
		return
	}

	// If it's an .mdx file, load it into the engine
	if ext == ".mdx" {
		if err := h.engine.LoadAll(); err != nil {
			log.Error().Err(err).Msg("Failed to reload dictionaries")
		}
	}

	log.Info().
		Str("audit", "true").
		Str("action", "dict_uploaded").
		Str("filename", safeFilename).
		Int64("file_size", written).
		Str("operator_id", c.GetString("userID")).
		Msg("Dictionary file uploaded")

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "File uploaded successfully",
		"data": gin.H{
			"filename":  safeFilename,
			"file_size": written,
		},
	})
}

// UpdateTitle updates dictionary title
func (h *DictHandler) UpdateTitle(c *gin.Context) {
	dictID := c.Param("id")

	var req models.DictTitleUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": "Invalid request body",
			"data":    nil,
		})
		return
	}

	// Check if dictionary exists
	dictInfo, err := h.dictStore.GetByID(dictID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    40401,
			"message": "Dictionary not found",
			"data":    nil,
		})
		return
	}

	// Update title in database
	if err := h.dictStore.UpdateTitle(dictID, req.Title); err != nil {
		log.Error().Err(err).Msg("Failed to update dictionary title")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50002,
			"message": "Failed to update dictionary title",
			"data":    nil,
		})
		return
	}

	// Update in-memory loaded dict title
	if loaded, ok := h.engine.GetDict(dictID); ok {
		loaded.Info.Title = req.Title
	}

	log.Info().
		Str("audit", "true").
		Str("action", "dict_title_updated").
		Str("dict_id", dictID).
		Str("filename", dictInfo.Filename).
		Str("new_title", req.Title).
		Str("operator_id", c.GetString("userID")).
		Msg("Dictionary title updated")

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Dictionary title updated",
	})
}

// Download handles dictionary file download
func (h *DictHandler) Download(c *gin.Context) {
	dictID := c.Param("id")

	// Get dictionary info
	dictInfo, err := h.dictStore.GetByID(dictID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    40401,
			"message": "Dictionary not found",
			"data":    nil,
		})
		return
	}

	// Build file path
	filePath := filepath.Join(h.dictDir, dictInfo.Filename)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    40401,
			"message": "Dictionary file not found",
			"data":    nil,
		})
		return
	}

	// Set headers for file download
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", dictInfo.Filename))
	c.Header("Content-Type", "application/octet-stream")

	c.File(filePath)
}

// Delete deletes a dictionary
func (h *DictHandler) Delete(c *gin.Context) {
	dictID := c.Param("id")

	// Get dictionary info
	dictInfo, err := h.dictStore.GetByID(dictID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    40401,
			"message": "Dictionary not found",
			"data":    nil,
		})
		return
	}

	// Remove from engine
	h.engine.Unload(dictID)

	// Delete file
	mdxPath := filepath.Join(h.dictDir, dictInfo.Filename)
	if err := os.Remove(mdxPath); err != nil && !os.IsNotExist(err) {
		log.Error().Err(err).Msg("Failed to delete dictionary file")
	}

	// Delete .mdd file if exists
	if dictInfo.HasMdd {
		mddFilename := strings.TrimSuffix(dictInfo.Filename, ".mdx") + ".mdd"
		mddPath := filepath.Join(h.dictDir, mddFilename)
		if err := os.Remove(mddPath); err != nil && !os.IsNotExist(err) {
			log.Error().Err(err).Msg("Failed to delete mdd file")
		}
	}

	// Delete from database
	if err := h.dictStore.Delete(dictID); err != nil {
		log.Error().Err(err).Msg("Failed to delete dictionary from database")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50002,
			"message": "Failed to delete dictionary",
			"data":    nil,
		})
		return
	}

	log.Info().
		Str("audit", "true").
		Str("action", "dict_deleted").
		Str("dict_id", dictID).
		Str("filename", dictInfo.Filename).
		Str("operator_id", c.GetString("userID")).
		Msg("Dictionary deleted")

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Dictionary deleted",
	})
}

// GetAsset returns a dictionary asset
func (h *DictHandler) GetAsset(c *gin.Context) {
	dictID := c.Param("id")
	assetPath := c.Param("path")

	// Get asset from engine
	data, mimeType, err := h.engine.GetAsset(dictID, assetPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    40401,
			"message": "Asset not found",
			"data":    nil,
		})
		return
	}

	c.Data(http.StatusOK, mimeType, data)
}
