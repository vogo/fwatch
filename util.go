package fwatch

import (
	"os"
	"path/filepath"

	"github.com/vogo/fsnotify"
)

func isDir(name string) bool {
	stat, err := os.Stat(name)

	return err == nil && stat != nil && stat.IsDir()
}

func unlink(path string, info os.FileInfo) (unlinkPath string, dir bool, fileErr error) {
	if info.IsDir() {
		return path, true, nil
	}

	var err error

	for info.Mode()&os.ModeSymlink != 0 {
		path, err = filepath.EvalSymlinks(path)

		if err != nil {
			return "", false, err
		}

		info, err = os.Lstat(path)

		if err != nil {
			return "", false, err
		}
	}

	return path, info.IsDir(), nil
}

func opMatch(op fsnotify.Op, target ...fsnotify.Op) bool {
	for _, t := range target {
		if op&t != t {
			return false
		}
	}

	return true
}
