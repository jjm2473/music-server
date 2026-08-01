package security

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ResolveSafeReadPath(root, fullPath string) (string, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve root: %w", err)
	}

	targetAbs, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("resolve target: %w", err)
	}
	if err := ensureUnderRoot(rootAbs, targetAbs); err != nil {
		return "", err
	}

	// File must exist for read operations and symlink target checks.
	if _, err := os.Stat(targetAbs); err != nil {
		return "", err
	}

	targetReal, err := filepath.EvalSymlinks(targetAbs)
	if err != nil {
		return "", fmt.Errorf("resolve target symlink: %w", err)
	}

	// Request path must stay under root, but resolved symlink target may be outside root.
	return targetReal, nil
}

func ensureUnderRoot(root, target string) error {
	root = filepath.Clean(root)
	target = filepath.Clean(target)
	if root == target {
		return nil
	}
	sep := string(filepath.Separator)
	if strings.HasPrefix(target, root+sep) {
		return nil
	}
	return fmt.Errorf("path escapes root")
}

func IsHiddenBase(name string) bool {
	return strings.HasPrefix(name, ".") && name != "." && name != ".."
}
