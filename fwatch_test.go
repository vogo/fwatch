package fwatch_test

import (
	"fmt"
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
	_ = ioutil.WriteFile(filepath.Join(linkDir, "test-link-file.txt"), []byte("test"), filePerm)

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
	fmt.Println("-------- 1. create test.txt")
	f := filepath.Join(dir, "test.txt")
	_ = ioutil.WriteFile(f, []byte("test"), filePerm)

	sleep(1, "")

	fmt.Println("-------- 2. update test.txt")
	_ = ioutil.WriteFile(f, []byte("update1"), filePerm)

	sleep(10, "text.txt should be consider being inactive")

	fmt.Println("-------- 3. update test.txt again")
	_ = ioutil.WriteFile(f, []byte("update2"), filePerm)

	sleep(1, "")

	fmt.Println("--------  4. rename text.text to text-1.txt")
	_ = os.Rename(f, filepath.Join(dir, "test-1.txt"))

	sleep(1, "")

	fromPath := filepath.Join(dir, "test-1.txt")
	toPath := filepath.Join(otherDir, "test-1.txt")
	fmt.Printf("--------  5. rename %s to other dir %s\n", fromPath, toPath)
	_ = os.Rename(fromPath, toPath)

	fmt.Println("--------  6. create sub dir")
	subDir := filepath.Join(dir, "sub")
	_ = os.Mkdir(subDir, os.ModePerm)

	sleep(1, "")

	fmt.Println("--------  7. create sub.txt in sub dir")
	subFile := filepath.Join(subDir, "sub.txt")
	_ = ioutil.WriteFile(subFile, []byte("test"), filePerm)

	sleep(1, "")

	fmt.Println("--------  8. update sub.txt in sub dir")
	_ = ioutil.WriteFile(subFile, []byte("update 1"), filePerm)

	sleep(10, "all files should be consider being inactive")

	fmt.Println("--------  9. update sub.txt again in sub dir after a long time")
	_ = ioutil.WriteFile(subFile, []byte("update 2"), filePerm)

	sleep(10, "sub.txt should be consider being inactive")
}

func sleep(seconds int64, message string) {
	fmt.Printf("\n  ==== sleep %ds --> %s\n", seconds, message)
	time.Sleep(time.Second * time.Duration(seconds))
}

func removeFile(f string) {
	logger.Infof("remove file: %s", f)
	_ = os.RemoveAll(f)
}
