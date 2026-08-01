package scan

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"music-server/server/internal/config"
	"music-server/server/internal/pathmap"
	"music-server/server/internal/security"
)

func Run(cfg config.Config) error {
	extAllow := make(map[string]struct{}, len(cfg.Scan.Ext))
	for _, e := range cfg.Scan.Ext {
		extAllow[strings.ToLower(strings.TrimPrefix(e, "."))] = struct{}{}
	}

	items := make([]MusicItem, 0, 256)
	walkErr := filepath.WalkDir(cfg.Common.Root, func(current string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if current != cfg.Common.Root && security.IsHiddenBase(d.Name()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}

		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(d.Name()), "."))
		if _, ok := extAllow[ext]; !ok {
			return nil
		}

		item, err := buildItem(cfg, current)
		if err != nil {
			// Keep scan robust: skip unreadable or malformed files.
			return nil
		}
		items = append(items, item)
		return nil
	})
	if walkErr != nil {
		return walkErr
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].URL < items[j].URL
	})

	if err := writeIndexAtomic(cfg.Common.Root, items); err != nil {
		return fmt.Errorf("write music.json: %w", err)
	}
	return nil
}

func buildItem(cfg config.Config, audioPath string) (MusicItem, error) {
	url, err := pathmap.FSToURL(cfg.Common.Root, cfg.Common.Path, audioPath)
	if err != nil {
		return MusicItem{}, err
	}

	md, err := readMetadata(audioPath)
	if err != nil {
		return MusicItem{}, err
	}

	item := MusicItem{URL: url}
	item.Title = md.Title
	item.Artist = md.Artist
	item.Album = md.Album
	item.Length = md.LengthSec

	if item.Title != "" {
		item.Name = item.Title
	} else {
		base := filepath.Base(url)
		item.Name = strings.TrimSuffix(base, filepath.Ext(base))
	}

	lrcURL, err := findLRC(cfg, audioPath)
	if err != nil {
		return MusicItem{}, err
	}
	item.LRC = lrcURL

	coverURL, err := persistCover(cfg.Common.Root, cfg.Common.Path, md)
	if err != nil {
		return MusicItem{}, err
	}
	item.Cover = coverURL

	if item.Name == "" {
		return MusicItem{}, errors.New("empty name")
	}
	return item, nil
}

func findLRC(cfg config.Config, audioPath string) (string, error) {
	lrcPath := strings.TrimSuffix(audioPath, filepath.Ext(audioPath)) + ".lrc"
	st, err := os.Stat(lrcPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	if st.IsDir() {
		return "", nil
	}
	return pathmap.FSToURL(cfg.Common.Root, cfg.Common.Path, lrcPath)
}
