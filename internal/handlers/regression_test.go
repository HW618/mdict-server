package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/HW618/mdict-server/internal/auth"
	"github.com/HW618/mdict-server/internal/dict"
	"github.com/HW618/mdict-server/internal/models"
	"github.com/HW618/mdict-server/internal/store"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func TestMCPConnectionAuthorization(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := store.NewSQLiteStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	us := store.NewUserStore(db)
	jwt := auth.NewJWTManager("secret", time.Hour, time.Hour)
	admin, user, other := models.NewUser("admin", "x"), models.NewUser("user", "x"), models.NewUser("other", "x")
	admin.IsUserAdmin = true
	for _, u := range []*models.User{admin, user, other} {
		if err := us.Create(u); err != nil {
			t.Fatal(err)
		}
	}
	router := gin.New()
	h := NewUserHandler(us, jwt)
	router.GET("/users/:id/mcp", auth.NewAuthMiddleware(jwt, us).RequireAuth(), h.GetMCPConnection)
	for _, tc := range []struct {
		actor, target *models.User
		want          int
	}{{user, user, 200}, {user, other, 403}, {admin, other, 200}} {
		token, _ := jwt.GenerateAccessToken(tc.actor)
		req := httptest.NewRequest("GET", "/users/"+tc.target.ID+"/mcp", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != tc.want {
			t.Errorf("%s -> %s: %d %s", tc.actor.Username, tc.target.Username, rec.Code, rec.Body.String())
		}
		if tc.want == 200 && !bytes.Contains(rec.Body.Bytes(), []byte(tc.target.APIToken)) {
			t.Errorf("missing target credential: %s", rec.Body.String())
		}
	}
}

func TestUploadRejectsInvalidDictionary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := store.NewSQLiteStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	dir := t.TempDir()
	ds := store.NewDictStore(db)
	h := NewDictHandler(dict.NewEngine(dir, ds), ds, dir, 0)
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	f, _ := form.CreateFormFile("files", "invalid.mdx")
	f.Write([]byte("not a dictionary"))
	form.Close()
	req := httptest.NewRequest(http.MethodPost, "/", &body)
	req.Header.Set("Content-Type", form.FormDataContentType())
	rec := httptest.NewRecorder()
	router := gin.New()
	router.POST("/", h.Upload)
	router.ServeHTTP(rec, req)
	var response struct {
		Data []struct {
			Error string `json:"error"`
		} `json:"data"`
	}
	json.Unmarshal(rec.Body.Bytes(), &response)
	if len(response.Data) != 1 || response.Data[0].Error == "" {
		t.Fatalf("invalid dictionary reported success: %s", rec.Body.String())
	}
}

func TestPasswordChangeInvalidatesExistingAccessToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := store.NewSQLiteStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	us := store.NewUserStore(db)
	jwt := auth.NewJWTManager("secret", time.Hour, time.Hour)
	hashed, err := bcrypt.GenerateFromPassword([]byte("old-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	user := models.NewUser("user", string(hashed))
	if err = us.Create(user); err != nil {
		t.Fatal(err)
	}
	token, _ := jwt.GenerateAccessToken(user)
	router := gin.New()
	mw := auth.NewAuthMiddleware(jwt, us)
	h := NewUserHandler(us, jwt)
	router.PUT("/password", mw.RequireAuth(), h.ChangePassword)
	router.GET("/private", mw.RequireAuth(), func(c *gin.Context) { c.Status(204) })
	req := httptest.NewRequest("PUT", "/password", bytes.NewBufferString(`{"old_password":"old-password","new_password":"new-password"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatal(rec.Body.String())
	}
	req = httptest.NewRequest("GET", "/private", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != 401 {
		t.Fatalf("old access token accepted: %d", rec.Code)
	}
}

func TestChunkedUploadLoadsDictionaryAndRejectsWrongSize(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := store.NewSQLiteStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	dir := t.TempDir()
	ds := store.NewDictStore(db)
	h := NewDictHandler(dict.NewEngine(dir, ds), ds, dir, 0)
	fixture, err := os.ReadFile("../dict/testdata/primary.mdx")
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.POST("/init", h.UploadInit)
	router.PUT("/chunk", h.UploadChunk)
	router.POST("/complete", h.UploadComplete)
	initBody := fmt.Sprintf(`{"filename":"sample.mdx","file_size":%d,"chunk_size":%d}`, len(fixture), len(fixture))
	req := httptest.NewRequest("POST", "/init", strings.NewReader(initBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	var initResp struct {
		Data struct {
			UploadID string `json:"upload_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &initResp); err != nil || initResp.Data.UploadID == "" {
		t.Fatalf("init: %s", rec.Body.String())
	}
	send := func(data []byte) int {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		w.WriteField("upload_id", initResp.Data.UploadID)
		w.WriteField("chunk_index", "0")
		f, _ := w.CreateFormFile("chunk", "part")
		f.Write(data)
		w.Close()
		req := httptest.NewRequest("PUT", "/chunk", &b)
		req.Header.Set("Content-Type", w.FormDataContentType())
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec.Code
	}
	if got := send(fixture[:len(fixture)-1]); got != 400 {
		t.Fatalf("short chunk status = %d", got)
	}
	if got := send(fixture); got != 200 {
		t.Fatalf("complete chunk status = %d", got)
	}
	req = httptest.NewRequest("POST", "/complete", strings.NewReader(fmt.Sprintf(`{"upload_id":%q}`, initResp.Data.UploadID)))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("completion: %s", rec.Body.String())
	}
	if _, err := ds.GetByFilename("sample.mdx"); err != nil {
		t.Fatal(err)
	}
}
