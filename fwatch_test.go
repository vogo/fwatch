package fwatch_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vogo/fwatch"
	"github.com/vogo/logger"
)

func TestFileWatcher(t *testing.T) {
	t.Parallel()

	otherDir := filepath.Join(os.TempDir(), "fwatch-other")
	linkDir := filepath.Join(os.TempDir(), "fwatch-link")
	tempDir := filepath.Join(os.TempDir(), "fwatch")
	_ = os.Mkdir(otherDir, os.ModePerm)
	_ = os.Mkdir(linkDir, os.ModePerm)
	_ = os.Mkdir(tempDir, os.ModePerm)

	_ = ioutil.WriteFile(filepath.Join(otherDir, "test1.txt"), []byte("test"), filePerm)
	_ = ioutil.WriteFile(filepath.Join(otherDir, "test2.txt"), []byte("test"), filePerm)
	_ = ioutil.WriteFile(filepath.Join(linkDir, "test-link-dir.txt"), []byte("test"), filePerm)

	_ = os.Link(filepath.Join(otherDir, "test1.txt"), filepath.Join(tempDir, "link-test1.txt"))
	_ = os.Symlink(filepath.Join(otherDir, "test2.txt"), filepath.Join(tempDir, "link-test2.txt"))
	_ = os.Symlink(linkDir, filepath.Join(tempDir, "link-dir"))

	defer removeFile(otherDir)
	defer removeFile(linkDir)
	defer removeFile(tempDir)

	fileWatcher, err := fwatch.NewFileWatcher(tempDir, true, time.Second*5, func(s string) bool {
		return true
	})
	if err != nil {
		t.Error(err)

		return
	}

	go func() {
		for {
			select {
			case <-fileWatcher.Done:
				return
			case f := <-fileWatcher.ActiveChan:
				logger.Infof("--> active file: %s", f.Name)
			case f := <-fileWatcher.InactiveChan:
				logger.Infof("--> inactive file: %s", f.Name)
			case name := <-fileWatcher.RemoveChan:
				logger.Infof("--> remove file: %s", name)
			}
		}
	}()

	if err := fileWatcher.Start(); err != nil {
		logger.Fatalf("create file watcher error: %v", err)
	}

	time.Sleep(time.Second)

	startFileUpdater(tempDir, otherDir)

	time.Sleep(time.Second)

	removeFile(tempDir)

	time.Sleep(time.Second)

	_ = fileWatcher.Stop()
}

const filePerm = 0o600

func startFileUpdater(dir, otherDir string) {
	// 1. create file
	f := filepath.Join(dir, "test.txt")
	_ = ioutil.WriteFile(f, []byte("test"), filePerm)

	time.Sleep(time.Second)

	// 2. update file
	_ = ioutil.WriteFile(f, []byte("update1"), filePerm)

	time.Sleep(time.Second)

	// 3. update file again
	_ = ioutil.WriteFile(f, []byte("update2"), filePerm)

	time.Sleep(time.Second)

	// 4. rename file
	_ = os.Rename(f, filepath.Join(dir, "test-1.txt"))

	time.Sleep(time.Second)

	// 5. rename file to other dir
	_ = os.Rename(filepath.Join(dir, "test-1.txt"), filepath.Join(otherDir, "test-1.txt"))

	// 6. create sub dir
	subDir := filepath.Join(dir, "sub")
	_ = os.Mkdir(subDir, os.ModePerm)

	time.Sleep(time.Second)

	// 7. create file in sub dir
	subFile := filepath.Join(subDir, "sub.txt")
	_ = ioutil.WriteFile(subFile, []byte("test"), filePerm)

	time.Sleep(time.Second)

	// 8. update file in sub dir
	_ = ioutil.WriteFile(subFile, []byte("update 1"), filePerm)

	time.Sleep(time.Second * 10)

	// 9. update file again in sub dir after a long time
	_ = ioutil.WriteFile(subFile, []byte("update 2"), filePerm)
}

func removeFile(f string) {
	logger.Infof("remove file: %s", f)
	_ = os.RemoveAll(f)
}
