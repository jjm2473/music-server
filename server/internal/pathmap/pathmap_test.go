package pathmap

import (
	"path/filepath"
	"testing"
)

func TestFSToURL(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "A", "song.mp3")
	got, err := FSToURL(root, "/data", target)
	if err != nil {
		t.Fatalf("FSToURL error: %v", err)
	}
	want := "/data/A/song.mp3"
	if got != want {
		t.Fatalf("want %s, got %s", want, got)
	}
}

func TestURLToRelative(t *testing.T) {
	rel, err := URLToRelative("/data", "/data/one/two.mp3")
	if err != nil {
		t.Fatalf("URLToRelative error: %v", err)
	}
	if rel != "one/two.mp3" {
		t.Fatalf("unexpected rel: %s", rel)
	}
}

func TestURLToRelativeRejectOutsideBase(t *testing.T) {
	_, err := URLToRelative("/data", "/other/a.mp3")
	if err == nil {
		t.Fatal("expected error for path outside base")
	}
}

func TestURLToRelative_DecodeEscapedPath(t *testing.T) {
	rel, err := URLToRelative("/data", "/data/%E4%B8%AD%E6%96%87/%E4%BD%A0%E5%A5%BD.mp3")
	if err != nil {
		t.Fatalf("URLToRelative error: %v", err)
	}
	if rel != "中文/你好.mp3" {
		t.Fatalf("unexpected rel: %s", rel)
	}
}
