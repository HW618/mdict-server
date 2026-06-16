package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/HW618/mdict-server/internal/dict"
	"github.com/HW618/mdict-server/internal/models"
	"github.com/HW618/mdict-server/internal/store"
	"github.com/rs/zerolog/log"
)

// safePath ensures the resolved path stays within baseDir, preventing path traversal.
func safePath(baseDir, userPath string) (string, error) {
	cleanBase := filepath.Clean(baseDir)
	resolved := filepath.Clean(filepath.Join(cleanBase, userPath))
	if !strings.HasPrefix(resolved, cleanBase+string(os.PathSeparator)) && resolved != cleanBase {
		return "", fmt.Errorf("path traversal detected: %s escapes %s", userPath, cleanBase)
	}
	return resolved, nil
}

// DictHandler handles dictionary endpoints
type DictHandler struct {
	engine         *dict.Engine
	dictStore      *store.DictStore
	dictDir        string
	maxUploadBytes int64
	uploads        sync.Map // upload_id -> *chunkedUpload
}

// chunkedUpload tracks a multipart upload session
type chunkedUpload struct {
	Filename     string
	RelativePath string // preserved subdirectory path for folder uploads
	TotalSize    int64
	ChunkSize    int64
	TotalParts   int
	CreatedAt    time.Time
	chunks       map[int]string // chunk_index -> temp file path
	mu           sync.Mutex
}

// allowedUploadExts defines file extensions permitted for upload.
// Includes dictionary files (.mdx, .mdd) and resource files (.css, .jpg, etc.)
// used by dictionary layouts.
var allowedUploadExts = map[string]bool{
	".mdx":  true,
	".mdd":  true,
	".css":  true,
	".js":   true,
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".svg":  true,
	".webp": true,
	".mp3":  true,
	".wav":  true,
	".ogg":  true,
	".woff": true,
	".woff2":true,
	".ttf":  true,
	".eot":  true,
	".ico":  true,
	".pdf":  true,
}

// isAllowedUploadExt checks if a file extension is permitted for upload
func isAllowedUploadExt(ext string) bool {
	return allowedUploadExts[strings.ToLower(ext)]
}

// findDictIDByMdd returns the dict ID for the .mdx file matching a .mdd filename.
// e.g. "Oxford.mdd" -> looks up "Oxford.mdx" in the dict store.
func (h *DictHandler) findDictIDByMdd(mddFilename string) string {
	mdxFilename := strings.TrimSuffix(mddFilename, ".mdd") + ".mdx"
	dict, err := h.dictStore.GetByFilename(mdxFilename)
	if err != nil {
		return ""
	}
	return dict.ID
}

// sanitizeRelativePath cleans a relative path and rejects path traversal attempts.
// It returns the cleaned path or an error if the path is unsafe.
func sanitizeRelativePath(p string) (string, error) {
	cleaned := filepath.Clean(p)
	if strings.Contains(cleaned, "..") || strings.HasPrefix(cleaned, "/") {
		return "", fmt.Errorf("unsafe path: %s", p)
	}
	return cleaned, nil
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

// Upload handles dictionary and resource file upload (supports single and batch)
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

	form := c.Request.MultipartForm
	files := form.File["files"]
	if len(files) == 0 {
		// Fallback: single file field "file"
		singleFile, hasFile := form.File["file"]
		if !hasFile || len(singleFile) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    40001,
				"message": "No files provided",
				"data":    nil,
			})
			return
		}
		files = singleFile
	}

	type uploadResult struct {
		Filename string `json:"filename"`
		FileSize int64  `json:"file_size"`
		Error    string `json:"error,omitempty"`
	}

	// Parse relative_path fields for folder uploads (one per file, same order)
	relativePaths := form.Value["relative_path"]

	var results []uploadResult
	var needReloadMdx []string // .mdx filenames that need loading
	var needReloadMdd []string // .mdd filenames whose .mdx needs reloading

	for i, header := range files {
		safeFilename := filepath.Base(header.Filename)
		if safeFilename == "." || safeFilename == "/" {
			results = append(results, uploadResult{Filename: header.Filename, Error: "Invalid filename"})
			continue
		}

		ext := strings.ToLower(filepath.Ext(safeFilename))
		if !isAllowedUploadExt(ext) {
			results = append(results, uploadResult{Filename: safeFilename, Error: "File type not allowed"})
			continue
		}

		// Check if .mdx already exists (use full relative path for subdirectory uploads)
		if ext == ".mdx" {
			checkFilename := safeFilename
			if i < len(relativePaths) && relativePaths[i] != "" {
				rp, rpErr := sanitizeRelativePath(relativePaths[i])
				if rpErr == nil {
					checkFilename = rp
				}
			}
			exists, err := h.dictStore.ExistsByFilename(checkFilename)
			if err != nil {
				results = append(results, uploadResult{Filename: safeFilename, Error: "Failed to check existence"})
				continue
			}
			if exists {
				results = append(results, uploadResult{Filename: safeFilename, Error: "Dictionary already exists"})
				continue
			}
		}

		// Open uploaded file
		file, err := header.Open()
		if err != nil {
			results = append(results, uploadResult{Filename: safeFilename, Error: "Failed to read upload"})
			continue
		}

		// Determine destination: use relative_path for folder uploads if available
		var dst string
		if i < len(relativePaths) && relativePaths[i] != "" {
			relPath, err := sanitizeRelativePath(relativePaths[i])
			if err != nil {
				results = append(results, uploadResult{Filename: header.Filename, Error: "Unsafe path"})
				continue
			}
			dst = filepath.Join(h.dictDir, relPath)
		} else {
			dst = filepath.Join(h.dictDir, safeFilename)
		}

		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			results = append(results, uploadResult{Filename: safeFilename, Error: "Failed to create directory"})
			continue
		}

		// Stream to temp file then rename atomically
		tmpDst := dst + ".tmp"

		tmpFile, err := os.Create(tmpDst)
		if err != nil {
			_ = file.Close()
			results = append(results, uploadResult{Filename: safeFilename, Error: "Failed to save file"})
			continue
		}

		written, copyErr := io.Copy(tmpFile, file)
		_ = tmpFile.Close()
		_ = file.Close()

		if copyErr != nil {
			_ = os.Remove(tmpDst)
			results = append(results, uploadResult{Filename: safeFilename, Error: "Failed to write file"})
			continue
		}

		// Check file size
		if h.maxUploadBytes > 0 && written > h.maxUploadBytes {
			_ = os.Remove(tmpDst)
			results = append(results, uploadResult{Filename: safeFilename, Error: fmt.Sprintf("File too large (max %d MB)", h.maxUploadBytes/(1024*1024))})
			continue
		}

		if err := os.Rename(tmpDst, dst); err != nil {
			_ = os.Remove(tmpDst)
			results = append(results, uploadResult{Filename: safeFilename, Error: "Failed to save file"})
			continue
		}

		results = append(results, uploadResult{Filename: safeFilename, FileSize: written})

		// Use the full relative path for reload lookup (needed for subdirectory uploads)
		reloadPath := safeFilename
		if i < len(relativePaths) && relativePaths[i] != "" {
			rp, rpErr := sanitizeRelativePath(relativePaths[i])
			if rpErr == nil {
				reloadPath = rp
			}
		}
		switch ext {
		case ".mdx":
			needReloadMdx = append(needReloadMdx, reloadPath)
		case ".mdd":
			needReloadMdd = append(needReloadMdd, reloadPath)
		}

		log.Info().
			Str("audit", "true").
			Str("action", "dict_uploaded").
			Str("filename", safeFilename).
			Int64("file_size", written).
			Str("operator_id", c.GetString("userID")).
			Msg("File uploaded")
	}

	// Load new .mdx dictionaries
	if len(needReloadMdx) > 0 {
		if err := h.engine.LoadAll(); err != nil {
			log.Error().Err(err).Msg("Failed to reload dictionaries after upload")
		}
	}

	// For uploaded .mdd files, reload the corresponding .mdx dictionary
	// so it picks up the new resource file
	for _, mddFilename := range needReloadMdd {
		dictID := h.findDictIDByMdd(mddFilename)
		if dictID != "" {
			if err := h.engine.Reload(dictID); err != nil {
				log.Error().Err(err).Str("dict_id", dictID).Msg("Failed to reload dictionary after .mdd upload")
			} else {
				log.Info().Str("dict_id", dictID).Str("mdd", mddFilename).Msg("Reloaded dictionary after .mdd upload")
			}
		}
	}

	// Check if any file succeeded
	successCount := 0
	for _, r := range results {
		if r.Error == "" {
			successCount++
		}
	}

	message := fmt.Sprintf("%d/%d files uploaded successfully", successCount, len(files))
	if successCount == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": message,
			"data":    results,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": message,
		"data":    results,
	})
}

// UpdateTitle updates dictionary title
// UploadInit starts a chunked upload session
type uploadInitRequest struct {
	Filename     string `json:"filename" binding:"required"`
	RelativePath string `json:"relative_path"`
	FileSize     int64  `json:"file_size" binding:"required"`
	ChunkSize    int64  `json:"chunk_size" binding:"required"`
}

func (h *DictHandler) UploadInit(c *gin.Context) {
	var req uploadInitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "message": "Invalid request body", "data": nil})
		return
	}

	safeFilename := filepath.Base(req.Filename)
	if safeFilename == "." || safeFilename == "/" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "message": "Invalid filename", "data": nil})
		return
	}

	// Sanitize relative_path for folder uploads
	var relativePath string
	if req.RelativePath != "" {
		rp, err := sanitizeRelativePath(req.RelativePath)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "message": "Unsafe relative path", "data": nil})
			return
		}
		relativePath = rp
	}

	ext := strings.ToLower(filepath.Ext(safeFilename))
	if !isAllowedUploadExt(ext) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "message": "File type not allowed", "data": nil})
		return
	}

	if h.maxUploadBytes > 0 && req.FileSize > h.maxUploadBytes {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"code": 41301, "message": fmt.Sprintf("File too large (max %d MB)", h.maxUploadBytes/(1024*1024)), "data": nil})
		return
	}

	// Check if .mdx already exists
	if ext == ".mdx" {
		exists, err := h.dictStore.ExistsByFilename(safeFilename)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 50002, "message": "Failed to check dictionary existence", "data": nil})
			return
		}
		if exists {
			c.JSON(http.StatusConflict, gin.H{"code": 40901, "message": "Dictionary already exists", "data": nil})
			return
		}
	}

	totalParts := int((req.FileSize + req.ChunkSize - 1) / req.ChunkSize)

	// Generate upload ID
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s-%d-%d", safeFilename, req.FileSize, time.Now().UnixNano())))
	uploadID := hex.EncodeToString(hash[:16])

	upload := &chunkedUpload{
		Filename:     safeFilename,
		RelativePath: relativePath,
		TotalSize:    req.FileSize,
		ChunkSize:    req.ChunkSize,
		TotalParts:   totalParts,
		CreatedAt:    time.Now(),
		chunks:       make(map[int]string),
	}
	h.uploads.Store(uploadID, upload)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Upload initialized",
		"data": gin.H{
			"upload_id":   uploadID,
			"total_parts": totalParts,
		},
	})
}

// UploadChunk receives a single chunk of a chunked upload
func (h *DictHandler) UploadChunk(c *gin.Context) {
	uploadID := c.PostForm("upload_id")
	if uploadID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "message": "upload_id is required", "data": nil})
		return
	}

	chunkIndexStr := c.PostForm("chunk_index")
	if chunkIndexStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "message": "chunk_index is required", "data": nil})
		return
	}
	var chunkIndex int
	if _, err := fmt.Sscanf(chunkIndexStr, "%d", &chunkIndex); err != nil || chunkIndex < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "message": "Invalid chunk_index", "data": nil})
		return
	}

	val, ok := h.uploads.Load(uploadID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"code": 40401, "message": "Upload session not found or expired", "data": nil})
		return
	}
	upload := val.(*chunkedUpload)

	if chunkIndex >= upload.TotalParts {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "message": "chunk_index out of range", "data": nil})
		return
	}

	file, _, err := c.Request.FormFile("chunk")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "message": "Chunk data is required", "data": nil})
		return
	}
	defer func() { _ = file.Close() }()

	// Write chunk to temp file
	tmpDir := filepath.Join(h.dictDir, ".uploads")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": "Failed to create upload directory", "data": nil})
		return
	}

	tmpPath := filepath.Join(tmpDir, fmt.Sprintf("%s_%d.tmp", uploadID, chunkIndex))
	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": "Failed to save chunk", "data": nil})
		return
	}
	written, err := io.Copy(tmpFile, file)
	_ = tmpFile.Close()
	if err != nil {
		_ = os.Remove(tmpPath)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": "Failed to save chunk", "data": nil})
		return
	}

	upload.mu.Lock()
	upload.chunks[chunkIndex] = tmpPath
	received := len(upload.chunks)
	upload.mu.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Chunk received",
		"data": gin.H{
			"chunk_index":  chunkIndex,
			"chunk_size":   written,
			"received":     received,
			"total_parts":  upload.TotalParts,
		},
	})
}

// UploadComplete finalizes a chunked upload by assembling all chunks
type uploadCompleteRequest struct {
	UploadID string `json:"upload_id" binding:"required"`
}

func (h *DictHandler) UploadComplete(c *gin.Context) {
	var req uploadCompleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "message": "Invalid request body", "data": nil})
		return
	}

	val, ok := h.uploads.LoadAndDelete(req.UploadID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"code": 40401, "message": "Upload session not found", "data": nil})
		return
	}
	upload := val.(*chunkedUpload)

	// Verify all chunks received
	if len(upload.chunks) != upload.TotalParts {
		h.cleanupChunks(upload)
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": fmt.Sprintf("Missing chunks: got %d, expected %d", len(upload.chunks), upload.TotalParts),
			"data":    nil,
		})
		return
	}

	// Determine destination: use RelativePath for folder uploads
	var dst string
	if upload.RelativePath != "" {
		dst = filepath.Join(h.dictDir, upload.RelativePath)
	} else {
		dst = filepath.Join(h.dictDir, upload.Filename)
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		h.cleanupChunks(upload)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": "Failed to create directory", "data": nil})
		return
	}

	// Assemble chunks in order
	tmpDst := dst + ".tmp"

	outFile, err := os.Create(tmpDst)
	if err != nil {
		h.cleanupChunks(upload)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": "Failed to create file", "data": nil})
		return
	}

	var totalWritten int64
	indices := make([]int, 0, len(upload.chunks))
	for idx := range upload.chunks {
		indices = append(indices, idx)
	}
	sort.Ints(indices)

	for _, idx := range indices {
		chunkPath := upload.chunks[idx]
		cf, err := os.Open(chunkPath)
		if err != nil {
			_ = outFile.Close()
			_ = os.Remove(tmpDst)
			h.cleanupChunks(upload)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": "Failed to read chunk", "data": nil})
			return
		}
		n, err := io.Copy(outFile, cf)
		_ = cf.Close()
		_ = os.Remove(chunkPath)
		if err != nil {
			_ = outFile.Close()
			_ = os.Remove(tmpDst)
			h.cleanupChunks(upload)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": "Failed to assemble chunk", "data": nil})
			return
		}
		totalWritten += n
	}
	_ = outFile.Close()

	// Clean up any remaining chunk temp files and .uploads directory
	h.cleanupChunks(upload)

	if err := os.Rename(tmpDst, dst); err != nil {
		_ = os.Remove(tmpDst)
		// tmpDst is already cleaned; no chunk files remain
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": "Failed to finalize file", "data": nil})
		return
	}

	// Reload engine with new file
	ext := strings.ToLower(filepath.Ext(upload.Filename))
	switch ext {
	case ".mdx":
		if err := h.engine.LoadAll(); err != nil {
			log.Error().Err(err).Msg("Failed to reload dictionaries")
		}
	case ".mdd":
		dictID := h.findDictIDByMdd(upload.Filename)
		if dictID != "" {
			if err := h.engine.Reload(dictID); err != nil {
				log.Error().Err(err).Str("dict_id", dictID).Msg("Failed to reload dictionary after .mdd upload")
			}
		}
	}

	log.Info().
		Str("audit", "true").
		Str("action", "dict_uploaded").
		Str("filename", upload.Filename).
		Int64("file_size", totalWritten).
		Str("operator_id", c.GetString("userID")).
		Msg("Dictionary file uploaded (chunked)")

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "File uploaded successfully",
		"data": gin.H{
			"filename":  upload.Filename,
			"file_size": totalWritten,
		},
	})
}

// cleanupChunks removes all temp chunk files for an upload.
// Must be called with upload.mu held to avoid data race on upload.chunks.
func (h *DictHandler) cleanupChunks(upload *chunkedUpload) {
	upload.mu.Lock()
	defer upload.mu.Unlock()
	for _, path := range upload.chunks {
		_ = os.Remove(path)
	}
	upload.chunks = make(map[int]string)
	// Try removing the .uploads directory if empty
	_ = os.Remove(filepath.Join(h.dictDir, ".uploads"))
}

// CleanupExpiredUploads removes chunked upload sessions older than maxAge.
// Called periodically from the server to prevent temp file leaks from abandoned uploads.
func (h *DictHandler) CleanupExpiredUploads(maxAge time.Duration) {
	now := time.Now()
	h.uploads.Range(func(key, value interface{}) bool {
		upload := value.(*chunkedUpload)
		if now.Sub(upload.CreatedAt) > maxAge {
			h.uploads.Delete(key)
			h.cleanupChunks(upload)
			log.Warn().
				Str("upload_id", key.(string)).
				Str("filename", upload.Filename).
				Msg("Cleaned up expired upload session")
		}
		return true
	})
}

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

	// Build file path (sanitize against path traversal)
	filePath, err := safePath(h.dictDir, dictInfo.Filename)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "message": "Invalid file path", "data": nil})
		return
	}

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
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", strings.ReplaceAll(filepath.Base(dictInfo.Filename), `"`, `\"`)))
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

	// Delete file (sanitize against path traversal)
	mdxPath, pathErr := safePath(h.dictDir, dictInfo.Filename)
	if pathErr != nil {
		log.Error().Err(pathErr).Msg("Invalid dictionary file path")
	} else if err := os.Remove(mdxPath); err != nil && !os.IsNotExist(err) {
		log.Error().Err(err).Msg("Failed to delete dictionary file")
	}

	// Delete .mdd file if exists
	if dictInfo.HasMdd {
		mddFilename := strings.TrimSuffix(dictInfo.Filename, ".mdx") + ".mdd"
		mddPath, mddPathErr := safePath(h.dictDir, mddFilename)
		if mddPathErr != nil {
			log.Error().Err(mddPathErr).Msg("Invalid mdd file path")
		} else if err := os.Remove(mddPath); err != nil && !os.IsNotExist(err) {
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

	// Strip leading slash from wildcard path param (Gin includes it)
	assetPath = strings.TrimPrefix(assetPath, "/")

	// Get asset from engine
	data, mimeType, err := h.engine.GetAsset(dictID, assetPath)
	if err != nil {
		h.engine.DebugLogAssetMiss(dictID, assetPath)
		c.JSON(http.StatusNotFound, gin.H{
			"code":    40401,
			"message": "Asset not found",
			"data":    nil,
		})
		return
	}

	c.Data(http.StatusOK, mimeType, data)
}
