package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/HW618/mdict-server/internal/store"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	dictStore  *store.DictStore
	userStore  *store.UserStore
	startTime  time.Time
	version    string
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(dictStore *store.DictStore, userStore *store.UserStore, version string) *HealthHandler {
	return &HealthHandler{
		dictStore: dictStore,
		userStore: userStore,
		startTime: time.Now(),
		version:   version,
	}
}

// Health returns service health status
func (h *HealthHandler) Health(c *gin.Context) {
	// Get dictionary count
	dictCount, err := h.dictStore.Count()
	if err != nil {
		dictCount = 0
	}

	// Calculate uptime
	uptime := time.Since(h.startTime).Seconds()

	c.JSON(http.StatusOK, gin.H{
		"status":        "healthy",
		"version":       h.version,
		"uptime":        int64(uptime),
		"dicts_loaded":  dictCount,
	})
}

// Stats returns system statistics (admin only)
func (h *HealthHandler) Stats(c *gin.Context) {
	// Get dictionary stats
	totalDicts, enabledDicts, totalEntries, err := h.dictStore.GetStats()
	if err != nil {
		totalDicts = 0
		enabledDicts = 0
		totalEntries = 0
	}

	// Get user count
	totalUsers, err := h.userStore.Count()
	if err != nil {
		totalUsers = 0
	}

	// Calculate uptime
	uptime := time.Since(h.startTime).Seconds()

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"total_dicts":    totalDicts,
			"enabled_dicts":  enabledDicts,
			"total_users":    totalUsers,
			"total_entries":  totalEntries,
			"uptime_seconds": int64(uptime),
		},
	})
}
