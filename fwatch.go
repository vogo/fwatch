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

package fwatch

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/vogo/grunner"
	"github.com/vogo/logger"
)

const (
	defaultMapSize          = 32
	minimalInactiveDeadline = 5 * time.Second
)

type WatchMethod string

const (
	// WatchMethodFS using os file system api to watch file events.
	WatchMethodFS WatchMethod = "fs"

	// WatchMethodTimer interval schedule check stat of files and trigger file change events.
	WatchMethodTimer WatchMethod = "timer"
)

// FileMatcher whether a file name matches.
type FileMatcher func(string) bool

// Event describes a set of file event.
type Event uint32

// These are file events that can trigger a notification.
const (
	Create Event = 1 << iota
	Write
	Remove
	Inactive
	Silence
)

// String event desc.
func (e Event) String() string {
	switch e {
	case Create:
		return "Create"
	case Write:
		return "Write"
	case Remove:
		return "Remove"
	case Inactive:
		return "Inactive"
	case Silence:
		return "Silence"
	}

	return ""
}

// WatchEvent file watch event.
type WatchEvent struct {
	Name  string
	Event Event
}

// FileStat file stat.
type FileStat struct {
	modTime time.Time
	active  bool
}

// DirStat dir stat.
type DirStat struct {
	modTime    time.Time
	includeSub bool
	matcher    FileMatcher
}

// FileWatcher a file watcher, watch change event in directory/sub-directories.
// Note: the change event may be duplicated.
type FileWatcher struct {
	mu sync.Mutex

	// Runner to control watching goroutines.
	Runner *grunner.Runner

	// watch method, fs or timer.
	method WatchMethod

	// a duration if a file not being updated in, then it's inactive.
	inactiveDuration time.Duration

	// a duration if a file not being updated in, then it's silence and remove it from watch list.
	silenceDuration time.Duration

	// watch directories.
	dirs map[string]*DirStat

	// temp add watch directories.
	newDirs map[string]*DirStat

	// a file map to watch.
	files map[string]*FileStat

	// temp add file map.
	newFiles map[string]*FileStat

	// a channel to notify active files.
	Events chan *WatchEvent

	// a channel to notify errors.
	Errors chan error

	// func to call for a new dir.
	newDirWatchInit func(dir string)

	// func to check dir.
	timerDirsChecker func(silenceDeadline time.Time)
}

var errFileMatcherNil = errors.New("fileMatcher nil")

// New create a new file watcher.
func New(watchMethod WatchMethod, inactiveDeadline, silenceDeadline time.Duration) (*FileWatcher, error) {
	if inactiveDeadline < minimalInactiveDeadline {
		return nil, fmt.Errorf("inactiveDuration %s is less than the minimal %s", inactiveDeadline, minimalInactiveDeadline)
	}

	fileWatcher := &FileWatcher{
		mu:               sync.Mutex{},
		Runner:           grunner.New(),
		method:           watchMethod,
		inactiveDuration: inactiveDeadline,
		silenceDuration:  silenceDeadline,
		dirs:             make(map[string]*DirStat, defaultMapSize),
		files:            make(map[string]*FileStat, defaultMapSize),
		newDirs:          make(map[string]*DirStat, defaultMapSize),
		newFiles:         make(map[string]*FileStat, defaultMapSize),
		Events:           make(chan *WatchEvent, defaultMapSize),
		Errors:           make(chan error, defaultMapSize),
		newDirWatchInit:  func(dir string) {},
		timerDirsChecker: func(silenceDeadline time.Time) {},
	}

	if fileWatcher.method != WatchMethodFS {
		fileWatcher.timerDirsChecker = fileWatcher.checkDirs
	}

	if err := fileWatcher.start(); err != nil {
		return nil, err
	}

	return fileWatcher, nil
}

func (fw *FileWatcher) WatchDir(dir string, includeSub bool, fileMatcher FileMatcher) error {
	if fileMatcher == nil {
		return errFileMatcherNil
	}

	dirInfo, err := os.Stat(dir)
	if err != nil {
		return err
	}

	if !dirInfo.IsDir() {
		return fmt.Errorf("invalid dir %s", dir)
	}

	// lock to make sure the timer is not executing check.
	fw.mu.Lock()
	defer fw.mu.Unlock()

	dirStat := &DirStat{
		modTime:    dirInfo.ModTime().Add(-time.Second),
		includeSub: includeSub,
		matcher:    fileMatcher,
	}
	fw.dirs[dir] = dirStat
	fw.checkDirInfo(dir, dirInfo, dirStat, time.Now().Add(-fw.silenceDuration))
	fw.newDirWatchInit(dir)

	return nil
}

func (fw *FileWatcher) Stop() error {
	fw.Runner.Stop()

	return nil
}

func (fw *FileWatcher) tryAddNewSubDir(info os.FileInfo, dir string, parentDirStat *DirStat, silenceDeadline time.Time) {
	if !parentDirStat.includeSub {
		return
	}

	if _, ok := fw.dirs[dir]; ok {
		return
	}

	if _, ok := fw.newDirs[dir]; ok {
		return
	}

	logger.Infof("add new dir: %s", dir)

	newDirStat := &DirStat{
		modTime:    info.ModTime().Add(-time.Second),
		includeSub: parentDirStat.includeSub,
		matcher:    parentDirStat.matcher,
	}

	fw.newDirs[dir] = newDirStat

	// check files and directories in new dir first.
	fw.checkDirInfo(dir, info, newDirStat, silenceDeadline)
}

func (fw *FileWatcher) tryAddNewFile(path string, fileInfo os.FileInfo, silenceDeadline time.Time) {
	if _, ok := fw.files[path]; ok {
		return
	}

	if !fileInfo.ModTime().After(silenceDeadline) {
		return
	}

	logger.Tracef("add new file: %s", path)

	fw.newFiles[path] = &FileStat{
		active:  true,
		modTime: fileInfo.ModTime(),
	}

	fw.Events <- &WatchEvent{
		Name:  path,
		Event: Create,
	}
}

func (fw *FileWatcher) tryRemoveFile(path string, _ *DirStat) {
	if _, ok := fw.files[path]; ok {
		return
	}

	delete(fw.files, path)

	fw.Events <- &WatchEvent{
		Name:  path,
		Event: Remove,
	}
}
