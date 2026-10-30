package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/HW618/mdict-server/internal/dict"
	"github.com/gin-gonic/gin"
)

func TestGetAssetRouteReturnsNotFoundForMissingDictionary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewDictHandler(dict.NewEngine(t.TempDir(), nil), nil, t.TempDir(), 0)
	router.GET("/assets/:id/*path", handler.GetAsset)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/assets/missing/images/hello.png", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["code"] != float64(40401) {
		t.Fatalf("code = %v, body = %s", body["code"], response.Body.String())
	}
}
