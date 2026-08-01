package pathmap

import (
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"strings"
)

func FSToURL(root, baseURL, absPath string) (string, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve root: %w", err)
	}
	targetAbs, err := filepath.Abs(absPath)
	if err != nil {
		return "", fmt.Errorf("resolve target: %w", err)
	}

	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil {
		return "", fmt.Errorf("build relative path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %s escapes root %s", targetAbs, rootAbs)
	}

	relSlash := filepath.ToSlash(rel)
	if relSlash == "." {
		return baseURL, nil
	}
	if strings.HasSuffix(baseURL, "/") {
		return baseURL + relSlash, nil
	}
	return baseURL + "/" + relSlash, nil
}

func URLToRelative(baseURL, requestPath string) (string, error) {
	decodedPath, err := url.PathUnescape(requestPath)
	if err != nil {
		return "", fmt.Errorf("invalid request path encoding: %w", err)
	}
	requestPath = decodedPath

	if baseURL == "/" {
		cleaned := path.Clean("/" + strings.TrimPrefix(requestPath, "/"))
		return strings.TrimPrefix(cleaned, "/"), nil
	}

	if requestPath == baseURL {
		return "", nil
	}
	prefix := baseURL + "/"
	if !strings.HasPrefix(requestPath, prefix) {
		return "", fmt.Errorf("path %s not under base %s", requestPath, baseURL)
	}
	part := strings.TrimPrefix(requestPath, prefix)
	cleaned := path.Clean("/" + part)
	if cleaned == "/" {
		return "", nil
	}
	return strings.TrimPrefix(cleaned, "/"), nil
}

func RelativeToFS(root, rel string) (string, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve root: %w", err)
	}
	joined := filepath.Join(rootAbs, filepath.FromSlash(rel))
	full, err := filepath.Abs(joined)
	if err != nil {
		return "", fmt.Errorf("resolve full path: %w", err)
	}
	return full, nil
}
