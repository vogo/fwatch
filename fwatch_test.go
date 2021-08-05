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
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vogo/fwatch"
	"github.com/vogo/logger"
)

const (
	inactiveSeconds  = 5
	inactiveDuration = time.Second * inactiveSeconds
	silenceSeconds   = 10
	silenceDuration  = time.Second * silenceSeconds
)

func TestFileWatcher(t *testing.T) {
	t.Parallel()

	doTestTypedFileWatcher(t, fwatch.WatchMethodTimer)

	doTestTypedFileWatcher(t, fwatch.WatchMethodFS)
}

func doTestTypedFileWatcher(t *testing.T, method fwatch.WatchMethod) {
	t.Helper()

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

	fileWatcher, err := fwatch.New(method, inactiveDuration, silenceDuration)
	if err != nil {
		t.Error(err)

		return
	}

	go func() {
		for {
			select {
			case <-fileWatcher.Stopper.C:
				return
			case f := <-fileWatcher.Events:
				logger.Infof("--> events : %s, %v", f.Name, f.Event)
			}
		}
	}()

	if err = fileWatcher.WatchDir(tempDir, true, func(s string) bool {
		return true
	}); err != nil {
		t.Error(err)

		return
	}

	time.Sleep(inactiveSeconds)

	startFileUpdater(tempDir, otherDir)

	time.Sleep(inactiveSeconds)

	removeFile(tempDir)

	time.Sleep(inactiveSeconds)

	_ = fileWatcher.Stop()
}

const filePerm = 0o600

func startFileUpdater(dir, otherDir string) {
	logger.Info("-------- 1. create test.txt")

	f := filepath.Join(dir, "test.txt")
	_ = ioutil.WriteFile(f, []byte("test"), filePerm)

	sleep(inactiveSeconds+2, "")

	logger.Info("-------- 2. update test.txt")

	_ = ioutil.WriteFile(f, []byte("update1"), filePerm)

	sleep(silenceSeconds+2, "test.txt should be consider being silence")

	logger.Info("-------- 3. update test.txt again")

	_ = ioutil.WriteFile(f, []byte("update2"), filePerm)

	sleep(inactiveSeconds+2, "")

	logger.Info("--------  4. rename test.txt to test-1.txt")

	_ = os.Rename(f, filepath.Join(dir, "test-1.txt"))

	sleep(inactiveSeconds+2, "")

	fromPath := filepath.Join(dir, "test-1.txt")
	toPath := filepath.Join(otherDir, "test-1.txt")
	logger.Infof("--------  5. rename %s to other dir %s\n", fromPath, toPath)
	_ = os.Rename(fromPath, toPath)

	logger.Info("--------  6. create sub dir")

	subDir := filepath.Join(dir, "sub")
	_ = os.Mkdir(subDir, os.ModePerm)

	sleep(inactiveSeconds+2, "")

	logger.Info("--------  7. create sub.txt in sub dir")

	subFile := filepath.Join(subDir, "sub.txt")
	_ = ioutil.WriteFile(subFile, []byte("test"), filePerm)

	sleep(inactiveSeconds+2, "")

	logger.Info("--------  8. update sub.txt in sub dir")

	_ = ioutil.WriteFile(subFile, []byte("update 1"), filePerm)

	sleep(silenceSeconds+2, "all files should be consider being silence")

	logger.Info("--------  9. update sub.txt again in sub dir after a long time")

	_ = ioutil.WriteFile(subFile, []byte("update 2"), filePerm)

	sleep(silenceSeconds+2, "sub.txt should be consider being silence")
}

func sleep(seconds int64, message string) {
	logger.Infof("\n  ==== sleep %ds --> %s\n", seconds, message)
	time.Sleep(time.Second * time.Duration(seconds))
}

func removeFile(f string) {
	logger.Infof("remove file: %s", f)
	_ = os.RemoveAll(f)
}
