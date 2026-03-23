/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package fwatch_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vogo/fwatch"
	"github.com/vogo/vogo/vlog"
)

const (
	inactiveSeconds  = 2
	inactiveDuration = time.Second * inactiveSeconds
	silenceSeconds   = 4
	silenceDuration  = time.Second * silenceSeconds

	filePerm = 0o600
)

func TestFileWatcher(t *testing.T) {
	t.Parallel()

	vlog.SetLevel(vlog.LevelDebug)

	t.Run("Timer", func(t *testing.T) {
		t.Parallel()
		doTestTypedFileWatcher(t, fwatch.WatchMethodTimer)
	})

	t.Run("FS", func(t *testing.T) {
		t.Parallel()
		doTestTypedFileWatcher(t, fwatch.WatchMethodFS)
	})
}

func doTestTypedFileWatcher(t *testing.T, method fwatch.WatchMethod) {
	t.Helper()

	// --- setup phase: prepare temp dirs, link files ---
	t.Logf("[setup] watch method=%s, inactive=%v, silence=%v", method, inactiveDuration, silenceDuration)

	tempDir := t.TempDir()
	otherDir := t.TempDir()
	linkDir := t.TempDir()

	t.Logf("[setup] tempDir=%s", tempDir)
	t.Logf("[setup] otherDir=%s", otherDir)
	t.Logf("[setup] linkDir=%s", linkDir)

	_ = os.WriteFile(filepath.Join(otherDir, "test1.txt"), []byte("test"), filePerm)
	_ = os.WriteFile(filepath.Join(otherDir, "test2.txt"), []byte("test"), filePerm)
	_ = os.WriteFile(filepath.Join(linkDir, "test-link-file.txt"), []byte("test"), filePerm)

	_ = os.Link(filepath.Join(otherDir, "test1.txt"), filepath.Join(tempDir, "link-test1.txt"))
	_ = os.Symlink(filepath.Join(otherDir, "test2.txt"), filepath.Join(tempDir, "link-test2.txt"))
	_ = os.Symlink(linkDir, filepath.Join(tempDir, "link-dir"))
	t.Log("[setup] hard link, symlink, and link-dir created")

	// --- create watcher ---
	t.Logf("[watcher] creating FileWatcher (method=%s)", method)

	fileWatcher, err := fwatch.New(method, inactiveDuration, silenceDuration)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("[watcher] FileWatcher created, starting event consumer")

	go func() {
		for {
			select {
			case <-fileWatcher.Runner.C:
				return
			case ev := <-fileWatcher.Events:
				t.Logf("[event] %s | %v", ev.Name, ev.Event)
			case watchErr := <-fileWatcher.Errors:
				t.Logf("[error] %v", watchErr)
			}
		}
	}()

	// --- start watching ---
	t.Logf("[watch] WatchDir(%s, includeSub=true)", tempDir)

	if err = fileWatcher.WatchDir(tempDir, true, func(s string) bool {
		return true
	}); err != nil {
		t.Fatal(err)
	}

	t.Logf("[watch] waiting %v for initial file detection", inactiveDuration)
	time.Sleep(inactiveDuration)

	// --- file update phase ---
	t.Log("[phase] starting file operations")
	startFileUpdater(t, tempDir, otherDir)

	// --- wait for remaining events ---
	t.Logf("[phase] file operations done, waiting %v for remaining events", inactiveDuration)
	time.Sleep(inactiveDuration)

	// --- cleanup phase ---
	t.Log("[cleanup] removing watched directory")
	_ = os.RemoveAll(tempDir)

	t.Logf("[cleanup] waiting %v for removal events", inactiveDuration)
	time.Sleep(inactiveDuration)

	// --- stop watcher ---
	t.Log("[stop] stopping FileWatcher")
	_ = fileWatcher.Stop()
	t.Log("[stop] FileWatcher stopped, test complete")
}

func startFileUpdater(t *testing.T, dir, otherDir string) {
	t.Helper()

	// step 1: create a new file
	filePath := filepath.Join(dir, "test.txt")
	t.Logf("[step 1/9] create file: %s", filePath)
	_ = os.WriteFile(filePath, []byte("test"), filePerm)
	sleep(t, inactiveSeconds+1, "wait for inactive detection")

	// step 2: update file content
	t.Logf("[step 2/9] update file: %s", filePath)
	_ = os.WriteFile(filePath, []byte("update1"), filePerm)
	sleep(t, silenceSeconds+1, "wait for silence detection")

	// step 3: update file again after silence
	t.Logf("[step 3/9] update file again after silence: %s", filePath)
	_ = os.WriteFile(filePath, []byte("update2"), filePerm)
	sleep(t, inactiveSeconds+1, "wait for inactive detection")

	// step 4: rename file within same directory
	newPath := filepath.Join(dir, "test-1.txt")
	t.Logf("[step 4/9] rename %s -> %s", filePath, newPath)
	_ = os.Rename(filePath, newPath)
	sleep(t, inactiveSeconds+1, "wait for rename detection")

	// step 5: move file to another directory
	toPath := filepath.Join(otherDir, "test-1.txt")
	t.Logf("[step 5/9] move %s -> %s (cross-dir)", newPath, toPath)
	_ = os.Rename(newPath, toPath)

	// step 6: create sub directory
	subDir := filepath.Join(dir, "sub")
	t.Logf("[step 6/9] create sub dir: %s", subDir)
	_ = os.Mkdir(subDir, os.ModePerm)
	sleep(t, inactiveSeconds+1, "wait for sub dir detection")

	// step 7: create file in sub directory
	subFile := filepath.Join(subDir, "sub.txt")
	t.Logf("[step 7/9] create file in sub dir: %s", subFile)
	_ = os.WriteFile(subFile, []byte("test"), filePerm)
	sleep(t, inactiveSeconds+1, "wait for inactive detection")

	// step 8: update file in sub directory
	t.Logf("[step 8/9] update file in sub dir: %s", subFile)
	_ = os.WriteFile(subFile, []byte("update 1"), filePerm)
	sleep(t, silenceSeconds+1, "wait for silence detection")

	// step 9: update file in sub directory after long silence
	t.Logf("[step 9/9] update file in sub dir after silence: %s", subFile)
	_ = os.WriteFile(subFile, []byte("update 2"), filePerm)
	sleep(t, silenceSeconds+1, "wait for silence detection")
}

func sleep(t *testing.T, seconds int64, reason string) {
	t.Helper()

	t.Logf("         sleep %ds (%s)", seconds, reason)
	time.Sleep(time.Second * time.Duration(seconds))
}
