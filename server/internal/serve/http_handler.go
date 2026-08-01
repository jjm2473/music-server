package serve

import (
	"errors"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"music-server/server/internal/config"
	"music-server/server/internal/pathmap"
	"music-server/server/internal/security"
)

var readMethods = map[string]struct{}{
	"GET":      {},
	"HEAD":     {},
	"OPTIONS":  {},
	"PROPFIND": {},
}

type Handler struct {
	cfg          config.Config
	webUIHandler http.Handler
}

func NewHandler(cfg config.Config) http.Handler {
	h := &Handler{cfg: cfg}
	if cfg.Serve.WebUI != "" {
		h.webUIHandler = http.FileServer(http.Dir(cfg.Serve.WebUI))
	}
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if hasInvalidPath(r) {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if matchesBasePath(h.cfg.Common.Path, r.URL.Path) {
		h.handleData(w, r)
		return
	}
	h.handleWebUI(w, r)
}

func hasInvalidPath(r *http.Request) bool {
	if containsDotDotToken(r.RequestURI) {
		return true
	}
	if containsDotDotToken(r.URL.RawPath) {
		return true
	}
	if containsDotDotToken(r.URL.EscapedPath()) {
		return true
	}
	return containsDotDotToken(r.URL.Path)
}

func containsDotDotToken(p string) bool {
	if p == "" {
		return false
	}
	v := strings.ToLower(p)
	if i := strings.IndexAny(v, "?#"); i >= 0 {
		v = v[:i]
	}
	for _, seg := range strings.Split(v, "/") {
		if seg == ".." {
			return true
		}
	}
	return strings.Contains(v, "%2e%2e")
}

func (h *Handler) handleData(w http.ResponseWriter, r *http.Request) {
	if _, ok := readMethods[r.Method]; !ok {
		methodNotAllowedData(w)
		return
	}

	switch r.Method {
	case "OPTIONS":
		w.Header().Set("Allow", "OPTIONS, GET, HEAD, PROPFIND")
		w.Header().Set("DAV", "1")
		w.WriteHeader(http.StatusNoContent)
		return
	case "PROPFIND":
		h.handlePROPFIND(w, r)
		return
	case "GET", "HEAD":
		h.handleReadFile(w, r)
		return
	default:
		methodNotAllowedData(w)
		return
	}
}

func (h *Handler) handleWebUI(w http.ResponseWriter, r *http.Request) {
	if h.webUIHandler == nil {
		http.NotFound(w, r)
		return
	}
	if r.Method != "GET" && r.Method != "HEAD" {
		methodNotAllowedWebUI(w)
		return
	}
	if st, ok := resolveWebUIETagTarget(h.cfg.Serve.WebUI, r.URL.Path); ok {
		w.Header().Set("ETag", makeETag(st))
	}
	h.webUIHandler.ServeHTTP(w, r)
}

func (h *Handler) handleReadFile(w http.ResponseWriter, r *http.Request) {
	_, safePath, notFound, err := h.resolveSafeRequestPath(r.URL.Path)
	if err != nil {
		if notFound {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	f, err := os.Open(safePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if st.IsDir() {
		http.NotFound(w, r)
		return
	}

	if ctype := mime.TypeByExtension(strings.ToLower(path.Ext(st.Name()))); ctype != "" {
		w.Header().Set("Content-Type", ctype)
	}
	w.Header().Set("Cache-Control", cacheControlForData(st.Name()))
	w.Header().Set("ETag", makeETag(st))
	w.Header().Set("Content-Length", strconv.FormatInt(st.Size(), 10))
	w.Header().Set("Last-Modified", st.ModTime().UTC().Format(http.TimeFormat))
	http.ServeContent(w, r, st.Name(), st.ModTime(), f)
}

func (h *Handler) resolveSafeRequestPath(urlPath string) (rel string, safePath string, notFound bool, err error) {
	rel, err = pathmap.URLToRelative(h.cfg.Common.Path, urlPath)
	if err != nil {
		return "", "", false, err
	}

	fullPath, err := pathmap.RelativeToFS(h.cfg.Common.Root, rel)
	if err != nil {
		return "", "", false, err
	}

	safePath, err = security.ResolveSafeReadPath(h.cfg.Common.Root, fullPath)
	if err != nil {
		return "", "", errors.Is(err, os.ErrNotExist), err
	}

	return rel, safePath, false, nil
}

func makeETag(st os.FileInfo) string {
	return fmt.Sprintf(`"%x-%x"`, st.Size(), st.ModTime().UTC().UnixNano())
}

func cacheControlForData(name string) string {
	if strings.EqualFold(path.Ext(name), ".json") {
		return "public, max-age=300"
	}
	return "public, max-age=31536000"
}

func resolveWebUIETagTarget(root, reqPath string) (os.FileInfo, bool) {
	cleanURLPath := path.Clean("/" + reqPath)
	rel := strings.TrimPrefix(cleanURLPath, "/")
	fullPath := filepath.Clean(filepath.Join(root, filepath.FromSlash(rel)))
	if !isSubPath(root, fullPath) {
		return nil, false
	}

	st, err := os.Stat(fullPath)
	if err != nil {
		return nil, false
	}

	if st.IsDir() {
		idxPath := filepath.Join(fullPath, "index.html")
		idxStat, err := os.Stat(idxPath)
		if err != nil || idxStat.IsDir() {
			return nil, false
		}
		return idxStat, true
	}

	if !st.Mode().IsRegular() {
		return nil, false
	}
	return st, true
}

func isSubPath(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	if rel == ".." {
		return false
	}
	prefix := ".." + string(os.PathSeparator)
	return !strings.HasPrefix(rel, prefix)
}

func methodNotAllowedData(w http.ResponseWriter) {
	w.Header().Set("Allow", "OPTIONS, GET, HEAD, PROPFIND")
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func methodNotAllowedWebUI(w http.ResponseWriter) {
	w.Header().Set("Allow", "GET, HEAD")
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func matchesBasePath(basePath, reqPath string) bool {
	if basePath == "/" {
		return true
	}
	return reqPath == basePath || strings.HasPrefix(reqPath, basePath+"/")
}

func formatHTTPDate(t time.Time) string {
	return t.UTC().Format(http.TimeFormat)
}

func cleanHref(baseURL, rel string, isDir bool) string {
	if rel == "" {
		if isDir && !strings.HasSuffix(baseURL, "/") {
			return baseURL + "/"
		}
		return baseURL
	}
	href := baseURL
	if !strings.HasSuffix(href, "/") {
		href += "/"
	}
	href += escapeDAVRelPath(path.Clean(rel))
	if isDir && !strings.HasSuffix(href, "/") {
		href += "/"
	}
	return href
}

func escapeDAVRelPath(rel string) string {
	if rel == "" || rel == "." {
		return ""
	}
	parts := strings.Split(rel, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return strings.Join(parts, "/")
}

func guessContentType(name string, isDir bool) string {
	if isDir {
		return ""
	}
	ext := strings.ToLower(path.Ext(name))
	if ext == "" {
		return "application/octet-stream"
	}
	if t := mime.TypeByExtension(ext); t != "" {
		return t
	}
	return "application/octet-stream"
}

func parseDepth(header string) int {
	v := strings.TrimSpace(strings.ToLower(header))
	switch v {
	case "0":
		return 0
	case "1", "", "infinity":
		return 1
	default:
		return 1
	}
}

func splitRel(parentRel, childName string) string {
	if parentRel == "" {
		return childName
	}
	return fmt.Sprintf("%s/%s", parentRel, childName)
}
