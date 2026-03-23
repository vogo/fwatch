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
	"fmt"
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

func TestMain(m *testing.M) {
	vlog.SetLevel(vlog.LevelDebug)
	os.Exit(m.Run())
}

func TestFileWatcher(t *testing.T) {
	t.Parallel()

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

	fileWatcher, err := fwatch.New(
		fwatch.WithMethod(method),
		fwatch.WithInactiveDuration(inactiveDuration),
		fwatch.WithSilenceDuration(silenceDuration),
	)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("[watcher] FileWatcher created, starting event consumer")

	go func() {
		for {
			select {
			case <-fileWatcher.Done():
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

func TestNewOptionValidation(t *testing.T) {
	t.Parallel()

	// invalid inactive duration
	_, err := fwatch.New(
		fwatch.WithInactiveDuration(time.Millisecond),
	)
	if err == nil {
		t.Fatal("expected error for too small inactive duration")
	}

	t.Logf("expected error: %v", err)

	// invalid dir file count limit
	_, err = fwatch.New(
		fwatch.WithInactiveDuration(2*time.Second),
		fwatch.WithDirFileCountLimit(10),
	)
	if err == nil {
		t.Fatal("expected error for invalid dir file count limit")
	}

	t.Logf("expected error: %v", err)

	// valid options
	w, err := fwatch.New(
		fwatch.WithMethod(fwatch.WatchMethodFS),
		fwatch.WithInactiveDuration(2*time.Second),
		fwatch.WithSilenceDuration(5*time.Second),
		fwatch.WithDirFileCountLimit(64),
	)
	if err != nil {
		t.Fatal(err)
	}

	_ = w.Stop()
}

func TestStatsAndUnwatchDir(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	_ = os.WriteFile(filepath.Join(tempDir, "a.txt"), []byte("aaa"), filePerm)
	_ = os.WriteFile(filepath.Join(tempDir, "b.txt"), []byte("bbb"), filePerm)

	w, err := fwatch.New(
		fwatch.WithMethod(fwatch.WatchMethodTimer),
		fwatch.WithInactiveDuration(2*time.Second),
		fwatch.WithSilenceDuration(10*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = w.Stop() }()

	// drain events
	go func() {
		for {
			select {
			case <-w.Done():
				return
			case <-w.Events:
			case <-w.Errors:
			}
		}
	}()

	if err = w.WatchDir(tempDir, false, func(s string) bool { return true }); err != nil {
		t.Fatal(err)
	}

	// wait for initial scan
	time.Sleep(2 * time.Second)

	stats := w.Stats()
	t.Logf("[stats] dirs=%d, files=%d, active=%d", stats.Dirs, stats.Files, stats.ActiveFiles)

	if stats.Dirs < 1 {
		t.Errorf("expected at least 1 dir, got %d", stats.Dirs)
	}

	if stats.Files < 2 {
		t.Errorf("expected at least 2 files, got %d", stats.Files)
	}

	// unwatch
	w.UnwatchDir(tempDir)

	stats = w.Stats()
	t.Logf("[stats after unwatch] dirs=%d, files=%d", stats.Dirs, stats.Files)

	if stats.Dirs != 0 {
		t.Errorf("expected 0 dirs after unwatch, got %d", stats.Dirs)
	}
}

func TestEventString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		event fwatch.Event
		want  string
	}{
		{fwatch.Create, "Create"},
		{fwatch.Write, "Write"},
		{fwatch.Remove, "Remove"},
		{fwatch.Inactive, "Inactive"},
		{fwatch.Silence, "Silence"},
		{fwatch.Event(0), ""},
	}

	for _, tt := range tests {
		if got := tt.event.String(); got != tt.want {
			t.Errorf("Event(%d).String() = %q, want %q", tt.event, got, tt.want)
		}
	}
}

func TestIsDir(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	if !fwatch.IsDir(tempDir) {
		t.Errorf("IsDir(%s) = false, want true", tempDir)
	}

	filePath := filepath.Join(tempDir, "file.txt")
	_ = os.WriteFile(filePath, []byte("x"), filePerm)

	if fwatch.IsDir(filePath) {
		t.Errorf("IsDir(%s) = true, want false", filePath)
	}

	if fwatch.IsDir(filepath.Join(tempDir, "nonexistent")) {
		t.Error("IsDir(nonexistent) = true, want false")
	}
}

func TestWatchDirErrors(t *testing.T) {
	t.Parallel()

	w, err := fwatch.New(
		fwatch.WithInactiveDuration(2 * time.Second),
		fwatch.WithSilenceDuration(5 * time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = w.Stop() }()

	// nil matcher
	if err = w.WatchDir(t.TempDir(), false, nil); err == nil {
		t.Error("expected error for nil matcher")
	}

	// nonexistent dir
	if err = w.WatchDir("/nonexistent/path", false, func(string) bool { return true }); err == nil {
		t.Error("expected error for nonexistent dir")
	}

	// file instead of dir
	f := filepath.Join(t.TempDir(), "file.txt")
	_ = os.WriteFile(f, []byte("x"), filePerm)

	if err = w.WatchDir(f, false, func(string) bool { return true }); err == nil {
		t.Error("expected error for file path")
	}
}

func TestDirFileCountLimit(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	// create 40 files to exceed limit of 32
	for i := range 40 {
		_ = os.WriteFile(filepath.Join(tempDir, fmt.Sprintf("file%d.txt", i)), []byte("x"), filePerm)
	}

	w, err := fwatch.New(
		fwatch.WithMethod(fwatch.WatchMethodTimer),
		fwatch.WithInactiveDuration(2*time.Second),
		fwatch.WithSilenceDuration(5*time.Second),
		fwatch.WithDirFileCountLimit(32),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = w.Stop() }()

	errCh := make(chan error, 32)

	go func() {
		for {
			select {
			case <-w.Done():
				return
			case <-w.Events:
			case watchErr := <-w.Errors:
				errCh <- watchErr
			}
		}
	}()

	err = w.WatchDir(tempDir, false, func(string) bool { return true })
	t.Logf("WatchDir returned: %v", err)

	// wait for error from timer check
	select {
	case e := <-errCh:
		t.Logf("got expected error: %v", e)
	case <-time.After(5 * time.Second):
		t.Log("no error received (dir was already rejected in WatchDir)")
	}
}

func TestFileRemoveDetection(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "removeme.txt")
	_ = os.WriteFile(filePath, []byte("data"), filePerm)

	w, err := fwatch.New(
		fwatch.WithMethod(fwatch.WatchMethodFS),
		fwatch.WithInactiveDuration(2*time.Second),
		fwatch.WithSilenceDuration(10*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = w.Stop() }()

	createCh := make(chan struct{}, 1)
	removeCh := make(chan struct{}, 1)

	go func() {
		for {
			select {
			case <-w.Done():
				return
			case ev := <-w.Events:
				t.Logf("[event] %s | %v", ev.Name, ev.Event)

				switch ev.Event {
				case fwatch.Create:
					select {
					case createCh <- struct{}{}:
					default:
					}
				case fwatch.Remove:
					select {
					case removeCh <- struct{}{}:
					default:
					}
				}
			case watchErr := <-w.Errors:
				t.Logf("[error] %v", watchErr)
			}
		}
	}()

	if err = w.WatchDir(tempDir, false, func(string) bool { return true }); err != nil {
		t.Fatal(err)
	}

	// wait for Create event
	select {
	case <-createCh:
		t.Log("got Create event")
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for Create event")
	}

	// remove the file
	t.Logf("[action] removing %s", filePath)
	_ = os.Remove(filePath)

	// wait for Remove event
	select {
	case <-removeCh:
		t.Log("got Remove event")
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for Remove event")
	}
}

func TestDoneChannel(t *testing.T) {
	t.Parallel()

	w, err := fwatch.New(
		fwatch.WithInactiveDuration(2 * time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}

	done := w.Done()

	select {
	case <-done:
		t.Fatal("Done() should not be closed before Stop()")
	default:
	}

	_ = w.Stop()

	select {
	case <-done:
		// expected
	case <-time.After(time.Second):
		t.Fatal("Done() should be closed after Stop()")
	}
}
