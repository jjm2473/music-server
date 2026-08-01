package scan

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"music-server/server/internal/pathmap"
)

func persistCover(root, baseURL string, md extractedMetadata) (string, error) {
	if len(md.CoverData) == 0 {
		return "", nil
	}

	ext := coverExt(md.CoverData, md.CoverMIME)
	sum := md5.Sum(md.CoverData)
	name := fmt.Sprintf("%x%s", sum, ext)

	coverDir := filepath.Join(root, ".cover")
	if err := os.MkdirAll(coverDir, 0o755); err != nil {
		return "", err
	}
	coverPath := filepath.Join(coverDir, name)

	if _, err := os.Stat(coverPath); err != nil {
		if !os.IsNotExist(err) {
			return "", err
		}
		if err := os.WriteFile(coverPath, md.CoverData, 0o644); err != nil {
			return "", err
		}
	}
	url, err := pathmap.FSToURL(root, baseURL, coverPath)
	if err != nil {
		return "", err
	}
	return url, nil
}

func coverExt(data []byte, mimeType string) string {
	if len(data) >= 8 && bytes.Equal(data[:8], []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}) {
		return ".png"
	}
	if len(data) >= 3 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff {
		return ".jpg"
	}
	if len(data) >= 6 && (bytes.Equal(data[:6], []byte("GIF87a")) || bytes.Equal(data[:6], []byte("GIF89a"))) {
		return ".gif"
	}
	if len(data) >= 12 && bytes.Equal(data[:4], []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WEBP")) {
		return ".webp"
	}

	m := strings.ToLower(strings.TrimSpace(mimeType))
	switch m {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		return ".bin"
	}
}
