package fs

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

func copyFile(from, to string) error {
	fromFile, err := os.Open(from)
	if err != nil {
		return err
	}
	defer fromFile.Close()

	toFile, err := os.Create(to)
	if err != nil {
		return err
	}
	defer toFile.Close()

	if _, err := io.Copy(toFile, fromFile); err != nil {
		return err
	}

	return nil
}

func moveDir(oldDir, newDir string, copy bool) error {
	return fs.WalkDir(os.DirFS(oldDir), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		newPath := filepath.Join(newDir, path)
		if d.IsDir() {
			return os.MkdirAll(newPath, 0755)
		}

		oldPath := filepath.Join(oldDir, path)
		if !copy {
			// Rename mode.
			// Try to rename(2) first. If oldDir and newDir are on different devices, this should fail.
			// If it fails, we'll fall back to copy-and-remove.
			if err := os.Rename(oldPath, newPath); err == nil {
				return nil
			}
		}

		if err := copyFile(oldPath, newPath); err != nil {
			return err
		}

		if !copy {
			// Try to remove the moved file.
			// We'll ignore error because it's not critical.
			os.Remove(oldPath)
		}

		return nil
	})
}

func CopyAll(oldDir, newDir string) error {
	return moveDir(oldDir, newDir, true)
}

func MoveAll(oldDir, newDir string) error {
	return moveDir(oldDir, newDir, false)
}
