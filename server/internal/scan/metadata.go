package scan

import (
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/dhowden/tag"
	"github.com/hajimehoshi/go-mp3"
	"github.com/jfreymuth/oggvorbis"
)

type extractedMetadata struct {
	Title     string
	Artist    string
	Album     string
	LengthSec int
	CoverData []byte
	CoverMIME string
	CoverExt  string
}

func readMetadata(audioPath string) (extractedMetadata, error) {
	f, err := os.Open(audioPath)
	if err != nil {
		return extractedMetadata{}, err
	}
	defer f.Close()

	m, err := tag.ReadFrom(f)
	if err != nil && !errors.Is(err, io.EOF) {
		return extractedMetadata{}, err
	}

	var out extractedMetadata
	if m != nil {
		out.Title = strings.TrimSpace(m.Title())
		out.Artist = strings.TrimSpace(m.Artist())
		out.Album = strings.TrimSpace(m.Album())
		if pic := m.Picture(); pic != nil && len(pic.Data) > 0 {
			out.CoverData = pic.Data
			out.CoverMIME = strings.TrimSpace(pic.MIMEType)
		}
	}

	length, err := readDurationSeconds(audioPath)
	if err == nil && length > 0 {
		out.LengthSec = length
	}
	return out, nil
}

func readDurationSeconds(audioPath string) (int, error) {
	ext := strings.ToLower(filepath.Ext(audioPath))
	switch ext {
	case ".mp3":
		return readMP3Duration(audioPath)
	case ".ogg":
		return readOGGDuration(audioPath)
	default:
		return 0, errors.New("unsupported duration codec")
	}
}

func readMP3Duration(audioPath string) (int, error) {
	f, err := os.Open(audioPath)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	dec, err := mp3.NewDecoder(f)
	if err != nil {
		return 0, err
	}
	if dec.SampleRate() <= 0 {
		return 0, errors.New("invalid sample rate")
	}
	seconds := float64(dec.Length()) / 4.0 / float64(dec.SampleRate())
	if seconds <= 0 {
		return 0, errors.New("invalid duration")
	}
	return int(math.Round(seconds)), nil
}

func readOGGDuration(audioPath string) (int, error) {
	f, err := os.Open(audioPath)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	dec, err := oggvorbis.NewReader(f)
	if err != nil {
		return 0, err
	}
	if dec.SampleRate() <= 0 {
		return 0, errors.New("invalid sample rate")
	}
	seconds := float64(dec.Length()) / float64(dec.SampleRate())
	if seconds <= 0 {
		return 0, errors.New("invalid duration")
	}
	return int(math.Round(seconds)), nil
}
