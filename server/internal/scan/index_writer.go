package scan

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func writeIndexAtomic(root string, items []MusicItem) error {
	finalPath := filepath.Join(root, "music.json")
	tmpPath := finalPath + ".tmp"

	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, finalPath)
}
