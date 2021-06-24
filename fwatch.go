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
	fileChangeChanSize      = 32
	minimalInactiveDeadline = 5 * time.Second
)

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

	// whether to include sub-directories
	includeSub bool

	// a file slice to schedule check whether not updated for a long time,
	// if yes then stop active and add it to watching file slice.
	activeFiles *list.List

	// a duration if a file not being updated in, then move it to watch file list.
	inactiveDeadline time.Duration

	// a channel to notify active files.
	ActiveChan chan *WatchFile

	// a channel to notify inactive files.
	InactiveChan chan *WatchFile

	// a channel to notify removed files.
	RemoveChan chan string

	// fsnotify file watcher to watch inactive files being updated.
	updateWatcher *fsnotify.Watcher

	// chan to control watching goroutines
	Done chan struct{}

	// the file matcher to check whether to watch a file.
	fileMatcher func(string) bool
}

var errFileMatcherNil = errors.New("fileMatcher nil")

// NewFileWatcher create a new file watcher.
func NewFileWatcher(dir string, includeSub bool, inactiveDeadline time.Duration, fileMatcher func(string) bool) (*FileWatcher, error) {
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
		includeSub:       includeSub,
		inactiveDeadline: inactiveDeadline,
		activeFiles:      list.New(),
		Done:             make(chan struct{}),
		ActiveChan:       make(chan *WatchFile, fileChangeChanSize),
		InactiveChan:     make(chan *WatchFile, fileChangeChanSize),
		RemoveChan:       make(chan string, fileChangeChanSize),
		fileMatcher:      fileMatcher,
	}, nil
}

func (fw *FileWatcher) addActive(name string) {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	f := &WatchFile{
		Name: name,
		Time: time.Now(),
	}
	fw.activeFiles.PushBack(f)

	fw.ActiveChan <- f
}

func (fw *FileWatcher) addInactive(wf *WatchFile) {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	logger.Debugf("add inactive file: %s", wf.Name)

	if err := fw.updateWatcher.AddWatch(wf.Name, fileWriteDeleteEvents); err != nil {
		logger.Warnf("watch file write event error: %s, %v", wf.Name, err)

		return
	}

	fw.InactiveChan <- wf
}

func (fw *FileWatcher) remove(name string) {
	fw.RemoveChan <- name
}

func (fw *FileWatcher) Start() error {
	var err error
	fw.updateWatcher, err = fsnotify.NewWatcher()

	if err != nil {
		return err
	}

	dirWatcher, err := fsnotify.NewWatcher()
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
		case event, ok := <-fw.updateWatcher.Events:
			if !ok {
				logger.Warn("failed to watch inactive files")

				return
			}

			logger.Debugf("file event: %v", event)

			if !fw.fileMatcher(event.Name) {
				continue
			}

			if event.Op&fsnotify.Remove == fsnotify.Remove {
				fw.remove(event.Name)
				_ = fw.updateWatcher.Remove(event.Name)

				continue
			}

			if event.Op&fsnotify.Write == fsnotify.Write {
				fw.addActive(event.Name)
			}
		case err, ok := <-fw.updateWatcher.Errors:
			if !ok {
				logger.Warnf("failed to get error event for inactive files")

				return
			}

			logger.Errorf("error: %v", err)
		}
	}
}

func (fw *FileWatcher) watchDirRecursively(fsnotifyWatcher *fsnotify.Watcher, dir string) error {
	err := fsnotifyWatcher.AddWatch(dir, fileCreateDeleteEvents)
	if err != nil {
		logger.Fatal(err)
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

	var (
		isDirPath bool
		filePath  string
	)

	for _, info := range fileInfos {
		filePath, isDirPath, err = unlink(filepath.Join(dir, info.Name()), info)
		if err != nil {
			logger.Debugf("read file error：%v", err)

			continue
		}

		if isDirPath {
			if fw.includeSub {
				go func() {
					_ = fw.watchDirRecursively(fsnotifyWatcher, filePath)
				}()
			}

			continue
		}

		if !fw.fileMatcher(filePath) {
			continue
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

	return nil
}

func (fw *FileWatcher) watchDir(dirWatcher *fsnotify.Watcher) {
	defer func() {
		logger.Warnf("stop watch directory")

		_ = dirWatcher.Close()
	}()

	for {
		select {
		case <-fw.Done:
			return
		case event, ok := <-dirWatcher.Events:
			if !ok {
				logger.Warnf("failed to listen watch event")

				return
			}

			logger.Debugf("dir event: %v", event)

			// ignore root dir events
			if event.Name == "" || event.Name == "." {
				continue
			}

			if opMatch(event.Op, fsnotify.Create) {
				if IsDir(event.Name) {
					_ = fw.watchDirRecursively(dirWatcher, event.Name)

					continue
				}

				fw.addActive(event.Name)

				continue
			}

			if opMatch(event.Op, fsnotify.Remove, fsnotify.Rename) {
				if IsDir(event.Name) {
					_ = dirWatcher.Remove(event.Name)
				} else {
					fw.remove(event.Name)
				}

				continue
			}

			if opMatch(event.Op, fsnotify.Write) {
				fw.addActive(event.Name)
			}
		case err, ok := <-dirWatcher.Errors:
			if !ok {
				logger.Warnf("failed to listen error event")

				return
			}

			logger.Errorf("watch dir error: %v", err)
		}
	}
}

// loopCheckActive check all active files, check whether it's already inactive.
func (fw *FileWatcher) loopCheckActive() {
	ticker := time.NewTicker(fw.inactiveDeadline)

	for {
		select {
		case <-fw.Done:
			return
		case <-ticker.C:
			for e := fw.activeFiles.Front(); e != nil; {
				watchFile := e.Value.(*WatchFile)

				if time.Since(watchFile.Time) < fw.inactiveDeadline {
					e = e.Next()

					continue
				}

				stat, err := os.Stat(watchFile.Name)
				if err != nil {
					logger.Debugf("check active file stat error: %v", err)

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
