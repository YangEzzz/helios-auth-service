package v1

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"helios-auth-service/internal/middleware"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestUploadImageRequiresAuthentication(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/upload/image", middleware.AuthMiddleware("test-secret"), UploadImage)
	r.POST("/api/v1/upload/image/check", middleware.AuthMiddleware("test-secret"), CheckImage)
	for _, path := range []string{"/api/v1/upload/image", "/api/v1/upload/image/check"} {
		for _, token := range []string{"", "Bearer invalid"} {
			req := httptest.NewRequest(http.MethodPost, path, nil)
			req.Header.Set("Authorization", token)
			response := httptest.NewRecorder()
			r.ServeHTTP(response, req)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("unauthenticated upload returned %d", response.Code)
			}
		}
	}
}

func TestImageMD5Deduplication(t *testing.T) {
	t.Chdir(t.TempDir())
	var picture bytes.Buffer
	if err := png.Encode(&picture, image.NewRGBA(image.Rect(0, 0, 3, 3))); err != nil {
		t.Fatal(err)
	}
	digest := fmt.Sprintf("%x", md5.Sum(picture.Bytes()))
	r := gin.New()
	r.POST("/upload", UploadImage)
	r.POST("/check", CheckImage)
	check := func(value string, status int, exists bool) {
		t.Helper()
		request := httptest.NewRequest("POST", "/check", strings.NewReader(fmt.Sprintf(`{"md5":%q}`, value)))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		r.ServeHTTP(response, request)
		var result struct {
			Data struct {
				Exists bool
				URL    string
			}
		}
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if response.Code != status || result.Data.Exists != exists {
			t.Fatalf("check: %d %s", response.Code, response.Body.String())
		}
		if exists && result.Data.URL != "/uploads/images/"+digest+".png" {
			t.Fatalf("unexpected URL: %s", result.Data.URL)
		}
	}
	check(digest, 200, false)
	check("../invalid", 400, false)
	check("", 400, false)
	upload := func(filename, suppliedMD5 string) *httptest.ResponseRecorder {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		part, _ := writer.CreateFormFile("file", filename)
		_, _ = part.Write(picture.Bytes())
		_ = writer.WriteField("md5", suppliedMD5)
		_ = writer.Close()
		request := httptest.NewRequest("POST", "/upload", &body)
		request.Header.Set("Content-Type", writer.FormDataContentType())
		response := httptest.NewRecorder()
		r.ServeHTTP(response, request)
		return response
	}
	if response := upload("wrong.png", strings.Repeat("0", 32)); response.Code != 400 {
		t.Fatalf("wrong MD5 accepted: %s", response.Body.String())
	}
	check(digest, 200, false)
	const count = 8
	responses := make(chan *httptest.ResponseRecorder, count)
	var workers sync.WaitGroup
	for i := 0; i < count; i++ {
		workers.Add(1)
		go func(i int) {
			defer workers.Done()
			responses <- upload(fmt.Sprintf("image-%d.png", i), strings.ToUpper(digest))
		}(i)
	}
	workers.Wait()
	close(responses)
	created := 0
	for response := range responses {
		var result struct {
			Data struct {
				URL, MD5 string
				Reused   bool
			}
		}
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if response.Code != 200 || result.Data.MD5 != digest || result.Data.URL != "/uploads/images/"+digest+".png" {
			t.Fatalf("upload: %d %s", response.Code, response.Body.String())
		}
		if !result.Data.Reused {
			created++
		}
	}
	if created != 1 {
		t.Fatalf("created %d files, want one", created)
	}
	check(strings.ToUpper(digest), 200, true)
	if response := upload("renamed.PNG", ""); response.Code != 200 || !strings.Contains(response.Body.String(), `"reused":true`) {
		t.Fatalf("repeat: %s", response.Body.String())
	}
	files, err := os.ReadDir("uploads/images")
	if err != nil || len(files) != 1 {
		t.Fatalf("unexpected files: %v %v", files, err)
	}
	path := filepath.Join("uploads/images", digest+".png")
	if data, err := os.ReadFile(path); err != nil || !bytes.Equal(data, picture.Bytes()) {
		t.Fatalf("stored image changed: %v", err)
	}
	if _, err := saveSharedImage(path, []byte("different contents")); err == nil {
		t.Fatal("digest collision overwritten")
	}
}

func TestSaveSharedImage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "image.png")
	if reused, err := saveSharedImage(path, []byte("contents")); err != nil || reused {
		t.Fatalf("first save reused=%v error=%v", reused, err)
	}
	if reused, err := saveSharedImage(path, []byte("contents")); err != nil || !reused {
		t.Fatalf("repeat save reused=%v error=%v", reused, err)
	}
}

func TestUploadImage(t *testing.T) {
	t.Chdir(t.TempDir())
	var picture bytes.Buffer
	if err := png.Encode(&picture, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	r := gin.New()
	r.POST("/upload", UploadImage)
	r.Static("/uploads", "./uploads")
	for _, tc := range []struct {
		name, filename, field string
		data                  []byte
		want                  int
	}{
		{"png", "picture.png", "file", picture.Bytes(), 200},
		{"uppercase extension", "picture.PNG", "file", picture.Bytes(), 200},
		{"wrong extension", "picture.jpg", "file", picture.Bytes(), 400},
		{"fake image", "picture.png", "file", []byte("<script>alert(1)</script>"), 400},
		{"svg", "picture.svg", "file", []byte("<svg xmlns=\"http://www.w3.org/2000/svg\"/>"), 400},
		{"empty", "picture.png", "file", nil, 400},
		{"missing file", "picture.png", "other", picture.Bytes(), 400},
		{"oversize file", "picture.png", "file", make([]byte, 10*1024*1024+1), 400},
		{"oversize request", "picture.png", "file", make([]byte, 11*1024*1024), 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			part, err := writer.CreateFormFile(tc.field, tc.filename)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := part.Write(tc.data); err != nil {
				t.Fatal(err)
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest(http.MethodPost, "/upload", &body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			response := httptest.NewRecorder()
			r.ServeHTTP(response, req)
			if response.Code != tc.want {
				t.Fatalf("status %d: %s", response.Code, response.Body.String())
			}
			if tc.want != 200 {
				return
			}
			var result struct {
				Code int `json:"code"`
				Data struct {
					URL      string `json:"url"`
					Size     int    `json:"size"`
					MimeType string `json:"mimeType"`
				} `json:"data"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result.Code != 200 || !strings.HasPrefix(result.Data.URL, "/uploads/images/") || result.Data.Size != len(tc.data) || result.Data.MimeType != "image/png" {
				t.Fatalf("unexpected response: %s", response.Body.String())
			}
			saved, err := os.ReadFile(filepath.Join(".", result.Data.URL))
			if err != nil || !bytes.Equal(saved, tc.data) {
				t.Fatalf("stored bytes differ: %v", err)
			}
			get := httptest.NewRecorder()
			r.ServeHTTP(get, httptest.NewRequest(http.MethodGet, result.Data.URL, nil))
			if get.Code != 200 || !bytes.Equal(get.Body.Bytes(), tc.data) {
				t.Fatalf("public image unavailable: %d", get.Code)
			}
		})
	}
	response := httptest.NewRecorder()
	r.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader("invalid")))
	if response.Code != 400 {
		t.Fatalf("non-multipart accepted: %d", response.Code)
	}
}
