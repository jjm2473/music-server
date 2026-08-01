package security

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveSafeReadPath_AllowRegularFile(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "a.mp3")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := ResolveSafeReadPath(root, p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == "" {
		t.Fatal("empty resolved path")
	}
}

func TestResolveSafeReadPath_RejectOutsideRoot(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "x.mp3")
	if err := os.WriteFile(outside, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveSafeReadPath(root, outside); err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveSafeReadPath_AllowSymlinkOutsideRoot(t *testing.T) {
	root := t.TempDir()
	outsideDir := t.TempDir()
	outside := filepath.Join(outsideDir, "x.mp3")
	if err := os.WriteFile(outside, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(root, "link.mp3")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	got, err := ResolveSafeReadPath(root, link)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want, err := filepath.EvalSymlinks(outside)
	if err != nil {
		t.Fatalf("eval outside symlink: %v", err)
	}
	if got != want {
		t.Fatalf("want resolved path %s, got %s", want, got)
	}
}

func TestIsHiddenBase(t *testing.T) {
	if !IsHiddenBase(".abc") {
		t.Fatal(".abc should be hidden")
	}
	if IsHiddenBase("abc") {
		t.Fatal("abc should not be hidden")
	}
}
