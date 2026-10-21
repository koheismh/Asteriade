//go:build !windows

package privatefs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EnsureDirectory creates a private directory or rejects an existing unsafe one.
func EnsureDirectory(path string) error {
	if path == "" {
		return fmt.Errorf("privatefs: empty directory path")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("privatefs: resolve directory path: %w", err)
	}
	component := string(filepath.Separator)
	for _, part := range strings.Split(strings.TrimPrefix(abs, component), component) {
		if part == "" {
			continue
		}
		component = filepath.Join(component, part)
		parent, e := os.Lstat(component)
		if errors.Is(e, os.ErrNotExist) {
			if e = os.Mkdir(component, 0700); e != nil {
				return e
			}
			parent, e = os.Lstat(component)
		}
		if e != nil {
			return e
		}
		if parent.Mode()&os.ModeSymlink != 0 || !parent.IsDir() {
			return fmt.Errorf("privatefs: directory component is a link or non-directory")
		}
	}
	info, err := os.Lstat(abs)
	if err != nil {
		return fmt.Errorf("privatefs: inspect directory: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("privatefs: path is not a regular directory")
	}
	if info.Mode().Perm()&0077 != 0 {
		return fmt.Errorf("privatefs: directory grants group/other permissions")
	}
	return nil
}

// CheckFile accepts only an existing regular file without group/other access.
func CheckFile(path string) error {
	if path == "" {
		return fmt.Errorf("privatefs: empty file path")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("privatefs: inspect file: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("privatefs: path is not a regular file")
	}
	if info.Mode().Perm()&0077 != 0 {
		return fmt.Errorf("privatefs: file grants group/other permissions")
	}
	return nil
}
