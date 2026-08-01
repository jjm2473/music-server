package serve

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"music-server/server/internal/config"
)

func TestCleanHrefEscapesNonASCII(t *testing.T) {
	got := cleanHref("/data", "中文目录/你好 world.txt", false)
	want := "/data/%E4%B8%AD%E6%96%87%E7%9B%AE%E5%BD%95/%E4%BD%A0%E5%A5%BD%20world.txt"
	if got != want {
		t.Fatalf("want %s, got %s", want, got)
	}
}

func TestCleanHrefDirectoryTrailingSlash(t *testing.T) {
	got := cleanHref("/data", "中文目录", true)
	want := "/data/%E4%B8%AD%E6%96%87%E7%9B%AE%E5%BD%95/"
	if got != want {
		t.Fatalf("want %s, got %s", want, got)
	}
}

func TestWebUIFallbackAndDataPriority(t *testing.T) {
	dataDir := t.TempDir()
	webDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dataDir, "song.txt"), []byte("data-song"), 0o644); err != nil {
		t.Fatalf("write data file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(webDir, "index.html"), []byte("<h1>webui</h1>"), 0o644); err != nil {
		t.Fatalf("write web file: %v", err)
	}

	h := NewHandler(config.Config{
		Common: config.CommonConfig{Root: dataDir, Path: "/data"},
		Serve:  config.ServeConfig{WebUI: webDir},
	})

	// Requests under common.path must stay in data mapping.
	reqData := httptest.NewRequest(http.MethodGet, "/data/song.txt", nil)
	rrData := httptest.NewRecorder()
	h.ServeHTTP(rrData, reqData)
	if rrData.Code != http.StatusOK {
		t.Fatalf("data status: want %d, got %d", http.StatusOK, rrData.Code)
	}
	if body := rrData.Body.String(); body != "data-song" {
		t.Fatalf("data body: want %q, got %q", "data-song", body)
	}

	// Requests outside common.path should fallback to webui.
	reqWeb := httptest.NewRequest(http.MethodGet, "/", nil)
	rrWeb := httptest.NewRecorder()
	h.ServeHTTP(rrWeb, reqWeb)
	if rrWeb.Code != http.StatusOK {
		t.Fatalf("webui status: want %d, got %d", http.StatusOK, rrWeb.Code)
	}
	if body := rrWeb.Body.String(); body != "<h1>webui</h1>" {
		t.Fatalf("webui body: want %q, got %q", "<h1>webui</h1>", body)
	}

	// Even if webui exists, matched data path does not fallback.
	reqMiss := httptest.NewRequest(http.MethodGet, "/data/index.html", nil)
	rrMiss := httptest.NewRecorder()
	h.ServeHTTP(rrMiss, reqMiss)
	if rrMiss.Code != http.StatusNotFound {
		t.Fatalf("data miss status: want %d, got %d", http.StatusNotFound, rrMiss.Code)
	}
}

func TestWebUIOnlyAllowsReadMethods(t *testing.T) {
	dataDir := t.TempDir()
	webDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(webDir, "index.html"), []byte("ok"), 0o644); err != nil {
		t.Fatalf("write web file: %v", err)
	}

	h := NewHandler(config.Config{
		Common: config.CommonConfig{Root: dataDir, Path: "/data"},
		Serve:  config.ServeConfig{WebUI: webDir},
	})

	req := httptest.NewRequest("PROPFIND", "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status: want %d, got %d", http.StatusMethodNotAllowed, rr.Code)
	}
	if got := rr.Header().Get("Allow"); got != "GET, HEAD" {
		t.Fatalf("allow: want %q, got %q", "GET, HEAD", got)
	}

	reqHead := httptest.NewRequest(http.MethodHead, "/", nil)
	rrHead := httptest.NewRecorder()
	h.ServeHTTP(rrHead, reqHead)
	if rrHead.Code != http.StatusOK {
		t.Fatalf("head status: want %d, got %d", http.StatusOK, rrHead.Code)
	}
	data, err := io.ReadAll(rrHead.Body)
	if err != nil {
		t.Fatalf("read head body: %v", err)
	}
	if len(data) != 0 {
		t.Fatalf("head body should be empty, got %q", string(data))
	}
}

func TestHandleReadFileCacheControlAndETag(t *testing.T) {
	dataDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dataDir, "music.json"), []byte("[]"), 0o644); err != nil {
		t.Fatalf("write json file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "song.mp3"), []byte("fake-mp3"), 0o644); err != nil {
		t.Fatalf("write media file: %v", err)
	}

	h := NewHandler(config.Config{
		Common: config.CommonConfig{Root: dataDir, Path: "/data"},
	})

	reqJSON := httptest.NewRequest(http.MethodGet, "/data/music.json", nil)
	rrJSON := httptest.NewRecorder()
	h.ServeHTTP(rrJSON, reqJSON)
	if rrJSON.Code != http.StatusOK {
		t.Fatalf("json status: want %d, got %d", http.StatusOK, rrJSON.Code)
	}
	if got := rrJSON.Header().Get("Cache-Control"); got != "public, max-age=300" {
		t.Fatalf("json cache-control: want %q, got %q", "public, max-age=300", got)
	}
	if got := rrJSON.Header().Get("ETag"); got == "" {
		t.Fatalf("json etag should not be empty")
	}

	reqMedia := httptest.NewRequest(http.MethodGet, "/data/song.mp3", nil)
	rrMedia := httptest.NewRecorder()
	h.ServeHTTP(rrMedia, reqMedia)
	if rrMedia.Code != http.StatusOK {
		t.Fatalf("media status: want %d, got %d", http.StatusOK, rrMedia.Code)
	}
	if got := rrMedia.Header().Get("Cache-Control"); got != "public, max-age=31536000" {
		t.Fatalf("media cache-control: want %q, got %q", "public, max-age=31536000", got)
	}
	if got := rrMedia.Header().Get("ETag"); got == "" {
		t.Fatalf("media etag should not be empty")
	}
}

func TestHandleReadFileSupportsIfNoneMatch(t *testing.T) {
	dataDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dataDir, "music.json"), []byte("[]"), 0o644); err != nil {
		t.Fatalf("write json file: %v", err)
	}

	h := NewHandler(config.Config{
		Common: config.CommonConfig{Root: dataDir, Path: "/data"},
	})

	firstReq := httptest.NewRequest(http.MethodGet, "/data/music.json", nil)
	firstRR := httptest.NewRecorder()
	h.ServeHTTP(firstRR, firstReq)
	if firstRR.Code != http.StatusOK {
		t.Fatalf("first status: want %d, got %d", http.StatusOK, firstRR.Code)
	}
	etag := firstRR.Header().Get("ETag")
	if etag == "" {
		t.Fatalf("etag should not be empty")
	}

	secondReq := httptest.NewRequest(http.MethodGet, "/data/music.json", nil)
	secondReq.Header.Set("If-None-Match", etag)
	secondRR := httptest.NewRecorder()
	h.ServeHTTP(secondRR, secondReq)
	if secondRR.Code != http.StatusNotModified {
		t.Fatalf("second status: want %d, got %d", http.StatusNotModified, secondRR.Code)
	}
	if secondRR.Body.Len() != 0 {
		t.Fatalf("not-modified body should be empty")
	}
}

func TestHandleWebUIAddsETagOnly(t *testing.T) {
	dataDir := t.TempDir()
	webDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(webDir, "index.html"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("write web file: %v", err)
	}

	h := NewHandler(config.Config{
		Common: config.CommonConfig{Root: dataDir, Path: "/data"},
		Serve:  config.ServeConfig{WebUI: webDir},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: want %d, got %d", http.StatusOK, rr.Code)
	}
	if got := rr.Header().Get("ETag"); got == "" {
		t.Fatalf("etag should not be empty")
	}
	if got := rr.Header().Get("Cache-Control"); got != "" {
		t.Fatalf("webui cache-control should stay empty, got %q", got)
	}
}

func TestHandleWebUISupportsIfNoneMatch(t *testing.T) {
	dataDir := t.TempDir()
	webDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(webDir, "index.html"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("write web file: %v", err)
	}

	h := NewHandler(config.Config{
		Common: config.CommonConfig{Root: dataDir, Path: "/data"},
		Serve:  config.ServeConfig{WebUI: webDir},
	})

	firstReq := httptest.NewRequest(http.MethodGet, "/", nil)
	firstRR := httptest.NewRecorder()
	h.ServeHTTP(firstRR, firstReq)
	if firstRR.Code != http.StatusOK {
		t.Fatalf("first status: want %d, got %d", http.StatusOK, firstRR.Code)
	}
	etag := firstRR.Header().Get("ETag")
	if etag == "" {
		t.Fatalf("etag should not be empty")
	}

	secondReq := httptest.NewRequest(http.MethodGet, "/", nil)
	secondReq.Header.Set("If-None-Match", etag)
	secondRR := httptest.NewRecorder()
	h.ServeHTTP(secondRR, secondReq)
	if secondRR.Code != http.StatusNotModified {
		t.Fatalf("second status: want %d, got %d", http.StatusNotModified, secondRR.Code)
	}
	if secondRR.Body.Len() != 0 {
		t.Fatalf("not-modified body should be empty")
	}
}

func TestServeHTTPRejectsInvalidDotDotPath(t *testing.T) {
	dataDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dataDir, "song.txt"), []byte("ok"), 0o644); err != nil {
		t.Fatalf("write data file: %v", err)
	}

	h := NewHandler(config.Config{
		Common: config.CommonConfig{Root: dataDir, Path: "/data"},
	})

	cases := []string{
		"/../../etc/passwd",
		"/../../../etc/passwd",
		"/%2e%2e/%2e%2e/etc/passwd",
		"/../%2e%2e/../etc/passwd",
		"/data/../song.txt",
		"/data/%2e%2e/song.txt",
	}

	for _, u := range cases {
		req := httptest.NewRequest(http.MethodGet, u, nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("url %q status: want %d, got %d", u, http.StatusBadRequest, rr.Code)
		}
	}
}
