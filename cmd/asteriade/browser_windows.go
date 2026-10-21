package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
)

func openBrowser(target string) error {
	root := os.Getenv("SystemRoot")
	if !filepath.IsAbs(root) {
		return errors.New("Windows system directory unavailable")
	}
	command := exec.Command(filepath.Join(root, "System32", "rundll32.exe"), "url.dll,FileProtocolHandler", target)
	if err := command.Start(); err != nil {
		return err
	}
	go command.Wait()
	return nil
}
