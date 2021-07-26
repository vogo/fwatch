package fwatch

import (
	"container/list"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/vogo/fsnotify"
	"github.com/vogo/logger"
)

const (
	defaultMapSize          = 32
	minimalInactiveDeadline = 5 * time.Second
)

type WatchMethod string

const (
	// WatchMethodOS using os file system api to watch file events.
	WatchMethodOS WatchMethod = "os"

	// WatchMethodTimer interval schedule check stat of files and trigger file change events.
	WatchMethodTimer WatchMethod = "timer"
)

// FileMatcher whether a file name matches.
type FileMatcher func(string) bool

// WatchFile watch file info.
type WatchFile struct {
	Name string
	Time time.Time
}

// FileWatcher a file watcher, watch change event in directory/sub-directories.
// Note: the change event may be duplicated.
type FileWatcher struct {
	mu sync.Mutex

	// root directory to watch
	dir string

	// watch directories, used to avoid duplicated watching.
	dirs map[string]struct{}

	// whether to include sub-directories
	includeSub bool

	// watch method, os or timer.
	method WatchMethod

	// a file list to schedule checkFiles whether not being updated for a long time,
	// if yes then stop active and add it to inactive file watch list.
	activeFiles *list.List

	// a file map to avoid duplicated file event.
	activeFilesMap map[string]*WatchFile

	// a duration if a file not being updated in, then move it to watch file list.
	inactiveDeadline time.Duration

	// a channel to notify active files.
	ActiveChan chan *WatchFile

	// a channel to notify inactive files.
	InactiveChan chan *WatchFile

	// a channel to notify removed files.
	RemoveChan chan string

	// fsnotify file watcher to watch inactive files being updated.
	updateWatcher FsNotifyWatcher

	// chan to control watching goroutines
	Done chan struct{}

	// the file matcher to checkFiles whether to watch a file.
	fileMatcher FileMatcher
}

var errFileMatcherNil = errors.New("fileMatcher nil")

// NewFileWatcher create a new file watcher.
func NewFileWatcher(dir string, includeSub bool, watchMethod WatchMethod,
	inactiveDeadline time.Duration, fileMatcher func(string) bool) (*FileWatcher, error) {
	if !IsDir(dir) {
		return nil, fmt.Errorf("invalid dir %s", dir)
	}

	if inactiveDeadline < minimalInactiveDeadline {
		return nil, fmt.Errorf("inactiveDeadline %s is less than the minimal %s", inactiveDeadline, minimalInactiveDeadline)
	}

	if fileMatcher == nil {
		return nil, errFileMatcherNil
	}

	return &FileWatcher{
		mu:               sync.Mutex{},
		dir:              dir,
		dirs:             make(map[string]struct{}, defaultMapSize),
		includeSub:       includeSub,
		method:           watchMethod,
		inactiveDeadline: inactiveDeadline,
		activeFilesMap:   make(map[string]*WatchFile),
		activeFiles:      list.New(),
		Done:             make(chan struct{}),
		ActiveChan:       make(chan *WatchFile, defaultMapSize),
		InactiveChan:     make(chan *WatchFile, defaultMapSize),
		RemoveChan:       make(chan string, defaultMapSize),
		fileMatcher:      fileMatcher,
	}, nil
}

func (fw *FileWatcher) addActive(name string) {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if _, ok := fw.activeFilesMap[name]; ok {
		logger.Debugf("duplicated add active file: %s", name)

		return
	}

	f := &WatchFile{
		Name: name,
		Time: time.Now(),
	}

	fw.activeFiles.PushBack(f)
	fw.activeFilesMap[f.Name] = f

	fw.ActiveChan <- f
}

func (fw *FileWatcher) addInactive(wf *WatchFile) {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	delete(fw.activeFilesMap, wf.Name)

	logger.Debugf("add inactive file: %s", wf.Name)

	if err := fw.updateWatcher.AddWatch(wf.Name, FileWriteDeleteEvents); err != nil {
		logger.Warnf("watch file write event error: %s, %v", wf.Name, err)
	}

	fw.InactiveChan <- wf
}

func (fw *FileWatcher) remove(name string) {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	delete(fw.activeFilesMap, name)

	fw.RemoveChan <- name
}

func (fw *FileWatcher) Start() error {
	var err error
	fw.updateWatcher, err = NewFsNotifyWatcher(fw.method, 0, fw.fileMatcher)

	if err != nil {
		return err
	}

	dirWatcher, err := NewFsNotifyWatcher(fw.method, fw.inactiveDeadline, fw.fileMatcher)
	if err != nil {
		return err
	}

	go fw.watchInactiveFiles()

	go fw.loopCheckActive()

	go fw.watchDir(dirWatcher)

	return fw.watchDirRecursively(dirWatcher, fw.dir)
}

func (fw *FileWatcher) Stop() error {
	close(fw.Done)

	return nil
}

func (fw *FileWatcher) watchInactiveFiles() {
	defer func() {
		logger.Warn("stop inactive files watch")

		_ = fw.updateWatcher.Close()
	}()

	for {
		select {
		case <-fw.Done:
			return
		case event, ok := <-fw.updateWatcher.Events():
			if !ok {
				logger.Warn("failed to watch inactive files")

				return
			}

			logger.Debugf("inactive file event: %v", event)

			if event.Op&fsnotify.Remove == fsnotify.Remove {
				fw.remove(event.Name)
				_ = fw.updateWatcher.Remove(event.Name)

				continue
			}

			if event.Op&fsnotify.Write == fsnotify.Write {
				_ = fw.updateWatcher.Remove(event.Name)

				fw.addActive(event.Name)
			}
		case err, ok := <-fw.updateWatcher.Errors():
			if !ok {
				logger.Warnf("failed to get error event for inactive files")

				return
			}

			logger.Errorf("error: %v", err)
		}
	}
}

// fsnotifyWatchDir file system watch directory, return true if success.
func (fw *FileWatcher) fsnotifyWatchDir(fsnotifyWatcher FsNotifyWatcher, dir string) bool {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	// ignore duplicated directory
	if _, ok := fw.dirs[dir]; ok {
		return false
	}

	err := fsnotifyWatcher.AddWatch(dir, FileCreateDeleteEvents)
	if err != nil {
		logger.Errorf("fs watch dire error: %v", err)

		return false
	}

	fw.dirs[dir] = struct{}{}

	return true
}

func (fw *FileWatcher) watchDirRecursively(fsnotifyWatcher FsNotifyWatcher, dir string) error {
	if !fw.fsnotifyWatchDir(fsnotifyWatcher, dir) {
		return nil
	}

	logger.Infof("start watch directory: %s", dir)

	f, err := os.Open(dir)
	if err != nil {
		logger.Warnf("open directory error: %v", err)

		return err
	}

	fileInfos, err := f.Readdir(-1)
	_ = f.Close()

	if err != nil {
		logger.Warnf("read directory error: %v", err)

		return err
	}

	for _, info := range fileInfos {
		fw.watchSubFile(fsnotifyWatcher, dir, info)
	}

	return nil
}

func (fw *FileWatcher) watchSubFile(fsnotifyWatcher FsNotifyWatcher, dir string, info os.FileInfo) {
	filePath, isDirPath, pathErr := unlink(filepath.Join(dir, info.Name()), info)
	if pathErr != nil {
		logger.Debugf("read file error：%v", pathErr)

		return
	}

	if isDirPath {
		if fw.includeSub {
			go func() {
				_ = fw.watchDirRecursively(fsnotifyWatcher, filePath)
			}()
		}

		return
	}

	if !fw.fileMatcher(filePath) {
		return
	}

	if time.Since(info.ModTime()) < fw.inactiveDeadline {
		fw.addActive(filePath)
	} else {
		fw.addInactive(&WatchFile{
			Name: filePath,
			Time: info.ModTime(),
		})
	}
}

func (fw *FileWatcher) watchDir(dirWatcher FsNotifyWatcher) {
	defer func() {
		logger.Warnf("stop watch directory")

		_ = dirWatcher.Close()
	}()

	for {
		select {
		case <-fw.Done:
			return
		case event, ok := <-dirWatcher.Events():
			if !ok {
				logger.Warnf("failed to listen watch event")

				return
			}

			fw.handleDirEvent(dirWatcher, event)
		case err, ok := <-dirWatcher.Errors():
			if !ok {
				logger.Warnf("failed to listen error event")

				return
			}

			logger.Errorf("watch dir error: %v", err)
		}
	}
}

func (fw *FileWatcher) handleDirEvent(dirWatcher FsNotifyWatcher, event fsnotify.Event) {
	logger.Debugf("dir event: %v", event)

	// ignore root dir events
	if event.Name == "" || event.Name == "." {
		return
	}

	if !fw.fileMatcher(event.Name) {
		return
	}

	if opMatch(event.Op, fsnotify.Create) {
		if IsDir(event.Name) {
			_ = fw.watchDirRecursively(dirWatcher, event.Name)

			return
		}

		fw.addActive(event.Name)

		return
	}

	if opMatch(event.Op, fsnotify.Remove, fsnotify.Rename) {
		if IsDir(event.Name) {
			_ = dirWatcher.Remove(event.Name)
		} else {
			fw.remove(event.Name)
		}

		return
	}

	if opMatch(event.Op, fsnotify.Write) {
		fw.addActive(event.Name)
	}
}

const maxActiveCheckInterval = time.Minute * 5

// loopCheckActive checkFiles all active files, checkFiles whether it's already inactive.
func (fw *FileWatcher) loopCheckActive() {
	checkInterval := fw.inactiveDeadline
	if checkInterval > maxActiveCheckInterval {
		checkInterval = maxActiveCheckInterval
	}

	ticker := time.NewTicker(checkInterval)

	for {
		select {
		case <-fw.Done:
			return
		case <-ticker.C:
			for e := fw.activeFiles.Front(); e != nil; {
				watchFile, _ := e.Value.(*WatchFile)

				if time.Since(watchFile.Time) < fw.inactiveDeadline {
					e = e.Next()

					continue
				}

				stat, err := os.Stat(watchFile.Name)
				if err != nil {
					if !os.IsNotExist(err) {
						logger.Warnf("check active file stat error: %v", err)
					}

					next := e.Next()
					fw.activeFiles.Remove(e)
					e = next

					fw.remove(watchFile.Name)

					continue
				}

				watchFile.Time = stat.ModTime()
				if time.Since(watchFile.Time) < fw.inactiveDeadline {
					e = e.Next()

					continue
				}

				next := e.Next()
				fw.activeFiles.Remove(e)
				e = next

				fw.addInactive(watchFile)
			}
		}
	}
}
