package fwatch_test

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLinkFile(t *testing.T) {
	t.Parallel()

	linkDir := filepath.Join(os.TempDir(), "fwatch-util-link")
	tempDir := filepath.Join(os.TempDir(), "fwatch-util")
	_ = os.Mkdir(linkDir, os.ModePerm)
	_ = os.Mkdir(tempDir, os.ModePerm)

	defer func() {
		_ = os.RemoveAll(tempDir)
		_ = os.RemoveAll(linkDir)
	}()

	linkDirInTemp := filepath.Join(tempDir, "link-dir")
	_ = os.Symlink(linkDir, linkDirInTemp)

	info, err := os.Lstat(linkDirInTemp)
	if err != nil {
		t.Error(err)
	}

	t.Log("linkDir name:", info.Name())
	t.Log("linkDir is dir:", info.IsDir())
	t.Log("linkDir file mode:", info.Mode())
	t.Log("linkDir is link:", info.Mode()&os.ModeSymlink != 0)
	t.Log(filepath.EvalSymlinks(linkDirInTemp))
}
