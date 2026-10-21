//go:build !windows

package privatefs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureDirectoryCreatesAndAcceptsPrivateDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "private")
	if err := EnsureDirectory(dir); err != nil {
		t.Fatalf("EnsureDirectory() error = %v", err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got&0077 != 0 {
		t.Fatalf("permissions = %04o, group/other permissions remain", got)
	}
	if err := EnsureDirectory(dir); err != nil {
		t.Fatalf("EnsureDirectory(existing) error = %v", err)
	}
}

func TestEnsureDirectoryRejectsLoosePermissionsAndSymlink(t *testing.T) {
	loose := filepath.Join(t.TempDir(), "loose")
	if err := os.Mkdir(loose, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(loose, 0755); err != nil {
		t.Fatal(err)
	}
	if err := EnsureDirectory(loose); err == nil {
		t.Fatal("EnsureDirectory accepted group/other permissions")
	}
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(t.TempDir(), link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if err := EnsureDirectory(link); err == nil {
		t.Fatal("EnsureDirectory accepted symlink")
	}
}

func TestCheckFileAcceptsPrivateRegularFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private")
	if err := os.WriteFile(path, []byte("private"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := CheckFile(path); err != nil {
		t.Fatalf("CheckFile() error = %v", err)
	}
}

func TestCheckFileRejectsLoosePermissionsSymlinkAndDirectory(t *testing.T) {
	loose := filepath.Join(t.TempDir(), "loose")
	if err := os.WriteFile(loose, []byte("private"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(loose, 0644); err != nil {
		t.Fatal(err)
	}
	if err := CheckFile(loose); err == nil {
		t.Fatal("CheckFile accepted group/other permissions")
	}
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(loose, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if err := CheckFile(link); err == nil {
		t.Fatal("CheckFile accepted symlink")
	}
	if err := CheckFile(t.TempDir()); err == nil {
		t.Fatal("CheckFile accepted directory")
	}
}

func TestAncestorSymlinkDoesNotCreateOutsideDirectory(t *testing.T) {
	parent := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(parent, "link")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if err := EnsureDirectory(filepath.Join(link, "must-not-exist")); err == nil {
		t.Fatal("accepted ancestor link")
	}
	if _, err := os.Stat(filepath.Join(outside, "must-not-exist")); !os.IsNotExist(err) {
		t.Fatal("created directory through ancestor link", err)
	}
}
