package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/HW618/mdict-server/internal/store"
)

func TestMCPUnauthorizedChallenge(t *testing.T) {
	database, err := store.NewSQLiteStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })

	gin.SetMode(gin.TestMode)
	router := gin.New()
	middleware := NewAuthMiddleware(NewJWTManager("test-secret", time.Hour, time.Hour), store.NewUserStore(database))
	router.POST("/mcp", middleware.RequireAPIAccess(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	router.GET("/api/v1/search", middleware.RequireAPIAccess(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	mcpResponse := httptest.NewRecorder()
	router.ServeHTTP(mcpResponse, httptest.NewRequest(http.MethodPost, "/mcp", nil))
	if mcpResponse.Code != http.StatusUnauthorized {
		t.Fatalf("mcp status %d", mcpResponse.Code)
	}
	if got := mcpResponse.Header().Get("WWW-Authenticate"); got != `Bearer realm="mdict"` {
		t.Fatalf("challenge %q", got)
	}

	apiResponse := httptest.NewRecorder()
	router.ServeHTTP(apiResponse, httptest.NewRequest(http.MethodGet, "/api/v1/search", nil))
	if apiResponse.Code != http.StatusUnauthorized {
		t.Fatalf("api status %d", apiResponse.Code)
	}
	if got := apiResponse.Header().Get("WWW-Authenticate"); got != "" {
		t.Fatalf("api challenge %q", got)
	}
}
